# Service Catalog — CodeShop

## 1. API Gateway

- **Purpose**: Single public entry point untuk seluruh client request.
- **Responsibilities**: routing, TLS termination (di belakang Cloudflare), authentication token pass-through/validation, rate limiting, request ID/correlation ID injection, request/response transformation (REST publik ↔ gRPC internal).
- **Data ownership**: tidak memiliki database sendiri (stateless), kecuali cache rate-limit counter di Redis (shared infra, bukan data domain).
- **Database**: tidak ada.
- **APIs**: seluruh REST public API (lihat `API_DESIGN.md`).
- **Dependencies**: Auth, Catalog, Order, Payment, Download Service (gRPC internal).
- **Inbound communication**: HTTPS dari client (via Cloudflare).
- **Outbound communication**: gRPC ke seluruh backend service.
- **Events produced**: tidak ada.
- **Events consumed**: tidak ada.
- **Security requirements**: rate limiting per-IP dan per-user, validasi header/token sebelum diteruskan, WAF di layer Cloudflare, request size limit.
- **Scaling strategy**: horizontal, stateless, scale berdasarkan request rate.
- **Failure modes**: jika satu backend service down, Gateway mengembalikan error terisolasi (bukan cascading failure) menggunakan timeout dan circuit breaker per upstream.
- **Observability requirements**: request rate, latency per route, error rate per upstream, status code distribution.

## 2. Auth Service

- **Purpose**: Mengelola identitas, autentikasi, dan otorisasi dasar.
- **Responsibilities**: register, login, token issuance (JWT), refresh token, password hashing & verification, role management dasar.
- **Data ownership**: `User`, `Role`, refresh token session.
- **Database**: Auth DB (PostgreSQL), Redis untuk session/refresh token & login rate limiting.
- **APIs**: gRPC internal (`ValidateToken`, `GetUser`) + REST publik via Gateway (`/auth/*`).
- **Dependencies**: tidak bergantung pada service lain.
- **Inbound communication**: gRPC dari Gateway, Order, Download Service (validasi token/identitas).
- **Outbound communication**: tidak ada gRPC keluar ke service domain lain.
- **Events produced**: `UserRegistered`.
- **Events consumed**: tidak ada.
- **Security requirements**: password hashing modern (Argon2id), JWT dengan short TTL access token, refresh token rotation, rate limiting login untuk mencegah brute force, audit log untuk login/register.
- **Scaling strategy**: horizontal, stateless untuk validasi JWT (signature-based), Redis untuk session yang perlu revocation.
- **Failure modes**: jika Auth Service down sepenuhnya, request baru tidak dapat login/register; namun token JWT yang belum expired tetap dapat divalidasi secara stateless oleh service lain via public key, mengurangi single point of failure untuk operasi baca.
- **Observability requirements**: login success/fail rate, token issuance rate, refresh token rotation rate, brute-force attempt detection.

## 3. Catalog Service

- **Purpose**: Mengelola data produk digital yang dijual.
- **Responsibilities**: CRUD produk (oleh admin), manajemen versi produk, kategori, pricing, pencarian/listing katalog untuk publik.
- **Data ownership**: `Product`, `Category`, `ProductVersion`.
- **Database**: Catalog DB (PostgreSQL), Redis untuk cache listing/detail produk (read-heavy).
- **APIs**: gRPC internal (`GetProduct`, `ValidateProducts`, `GetPrice`) + REST publik (`/products/*`, `/admin/products/*`).
- **Dependencies**: Object Storage (untuk preview/metadata file, bukan file utama produk yang dilindungi).
- **Inbound communication**: gRPC dari Order Service (validasi produk saat checkout), Download Service (metadata produk untuk generate signed URL — file path).
- **Outbound communication**: tidak memanggil service domain lain.
- **Events produced**: `ProductPublished` *(opsional, Phase 2 — tidak wajib MVP)*.
- **Events consumed**: tidak ada.
- **Security requirements**: hanya admin (role-based) yang dapat mengubah katalog; endpoint publik read-only dan di-cache.
- **Scaling strategy**: horizontal, read-heavy, agresif menggunakan cache (Redis) dan CDN untuk asset publik non-sensitif (thumbnail, deskripsi).
- **Failure modes**: jika Catalog Service down, checkout baru tidak dapat divalidasi (produk tidak dapat dikonfirmasi), namun browsing dapat tetap sebagian berjalan dari cache/CDN untuk mengurangi dampak ke user.
- **Observability requirements**: cache hit rate, query latency, product view count (business metric).

## 4. Order Service

- **Purpose**: Mengelola siklus hidup order, termasuk **sub-domain Cart**.
- **Responsibilities**: manajemen cart (sebagai order berstatus `DRAFT`), checkout (konversi cart → order `PENDING_PAYMENT`), snapshot harga saat checkout, pelacakan status order, penyediaan data entitlement (order `PAID` = entitlement download).
- **Data ownership**: `Cart` (opsional sebagai representasi `Order` status `DRAFT`), `CartItem`, `Order`, `OrderItem`.
- **Database**: Order DB (PostgreSQL).
- **APIs**: gRPC internal (`AddCartItem`, `CreateOrder`, `GetOrder`, `UpdateOrderStatus`, `CheckEntitlement`) + REST publik (`/cart/*`, `/orders/*`).
- **Dependencies**: Catalog Service (validasi produk & harga), Auth Service (validasi identitas).
- **Inbound communication**: gRPC dari Gateway, Payment Service (update status), Download Service (cek entitlement).
- **Outbound communication**: gRPC ke Catalog & Auth.
- **Events produced**: `OrderCreated`.
- **Events consumed**: `PaymentSuccess`/`PaymentFailed` *(untuk update status; lihat catatan di `EVENT_DESIGN.md` — pada MVP, update status dilakukan secara synchronous oleh Payment Service via gRPC, event dipublish setelahnya untuk consumer lain)*.
- **Security requirements**: hanya pemilik order (ownership check) yang dapat melihat/mengubah order miliknya; price snapshot mencegah manipulasi harga oleh client.
- **Scaling strategy**: horizontal; write path (checkout) memerlukan konsistensi transaksional dalam database sendiri.
- **Failure modes**: jika Catalog Service tidak dapat dihubungi saat checkout, checkout ditolak dengan error jelas (fail-fast) — tidak membuat order dengan data produk yang tidak tervalidasi.
- **Observability requirements**: cart abandonment rate, checkout success/fail rate, order status distribution, average order value (business metric).

## 5. Payment Service

- **Purpose**: Mengelola integrasi dengan payment provider dan verifikasi pembayaran.
- **Responsibilities**: membuat sesi pembayaran ke payment provider, menerima & memverifikasi webhook, memperbarui status order menjadi `PAID`/`FAILED`, mencatat riwayat transaksi pembayaran.
- **Data ownership**: `Payment` (riwayat transaksi, status, referensi provider).
- **Database**: Payment DB (PostgreSQL).
- **APIs**: gRPC internal (`CreatePayment`, `GetPaymentStatus`) + REST publik (`/payments/*`) + webhook endpoint (`/payments/webhook`).
- **Dependencies**: Order Service (gRPC, untuk get/update order), Payment Provider (eksternal, REST + webhook).
- **Inbound communication**: gRPC dari Gateway; webhook HTTPS dari Payment Provider.
- **Outbound communication**: gRPC ke Order Service; REST ke Payment Provider.
- **Events produced**: `PaymentPending`, `PaymentSuccess`, `PaymentFailed`, `OrderPaid`.
- **Events consumed**: tidak ada.
- **Security requirements**: verifikasi signature webhook wajib sebelum diproses; idempotency terhadap webhook duplikat (payment provider dapat mengirim webhook lebih dari sekali); data pembayaran sensitif (jika ada) tidak disimpan penuh — hanya referensi/token dari provider (tidak menyimpan data kartu).
- **Scaling strategy**: horizontal; webhook endpoint harus idempotent sehingga aman menerima beban tinggi dan retry.
- **Failure modes**: jika Order Service tidak dapat dihubungi saat update status, Payment Service melakukan retry dengan backoff dan tetap menyimpan status pembayaran di Payment DB sebagai source of truth sementara (reconciliation).
- **Observability requirements**: payment success/fail rate, webhook processing latency, webhook duplicate rate, provider error rate.

## 6. Download Service

- **Purpose**: Mengelola otorisasi dan penerbitan akses download produk digital.
- **Responsibilities**: verifikasi entitlement (ownership + payment status), generate temporary signed URL ke object storage, pencatatan riwayat download.
- **Data ownership**: `DownloadEntitlement`, `DownloadRecord`.
- **Database**: Download DB (PostgreSQL).
- **APIs**: gRPC internal (`RequestDownload`, `GetEntitlement`) + REST publik (`/downloads/*`).
- **Dependencies**: Auth Service (validasi user), Order Service (cek entitlement/order PAID), Catalog Service (metadata file), Object Storage (generate signed URL).
- **Inbound communication**: gRPC dari Gateway.
- **Outbound communication**: gRPC ke Auth, Order, Catalog; API ke Object Storage.
- **Events produced**: `DownloadGranted`, `DownloadCompleted` *(opsional, jika object storage dapat memberi callback/log akses)*.
- **Events consumed**: `OrderPaid` (untuk membuat `DownloadEntitlement` secara asynchronous sebagai denormalized cache, mempercepat pengecekan tanpa selalu memanggil Order Service — opsional optimasi Phase 2; MVP dapat langsung gRPC call synchronous ke Order Service).
- **Security requirements**: signed URL dengan TTL singkat (mis. beberapa menit), limit jumlah request signed URL per periode waktu untuk mencegah abuse, audit log setiap penerbitan signed URL dan akses.
- **Scaling strategy**: horizontal; beban utama adalah request penerbitan signed URL (ringan), bukan transfer file (file ditransfer langsung dari object storage ke client).
- **Failure modes**: jika Object Storage tidak dapat dihubungi, request download gagal dengan error jelas; jika Order Service tidak dapat dihubungi, Download Service menolak request (fail-closed — tidak mengizinkan download tanpa verifikasi entitlement yang berhasil).
- **Observability requirements**: signed URL issuance rate, download success/fail rate, abuse pattern (permintaan berulang dari IP/user yang sama), signed URL expiry hit rate.

## 7. Notification Service

- **Purpose**: Mengirim notifikasi (email) untuk event penting dalam sistem.
- **Responsibilities**: konsumsi event dari RabbitMQ, render template notifikasi, mengirim melalui email/notification provider eksternal, mencatat riwayat notifikasi.
- **Data ownership**: `Notification` (log pengiriman, status).
- **Database**: Notification DB (PostgreSQL).
- **APIs**: tidak ada API publik/gRPC yang dipanggil service lain secara synchronous (murni event-driven); dapat menyediakan gRPC internal read-only untuk keperluan admin/audit (opsional).
- **Dependencies**: Email/Notification Provider (eksternal).
- **Inbound communication**: event dari RabbitMQ.
- **Outbound communication**: REST/SMTP ke provider eksternal.
- **Events produced**: tidak ada (consumer murni pada MVP).
- **Events consumed**: `UserRegistered`, `OrderCreated`, `OrderPaid`, `PaymentFailed`, `DownloadGranted`.
- **Security requirements**: tidak menyimpan data sensitif berlebih, hanya data minimal untuk keperluan pengiriman (email, template context); rate limiting pengiriman untuk mencegah spam akibat event loop/bug.
- **Scaling strategy**: horizontal, consumer group RabbitMQ untuk distribusi beban.
- **Failure modes**: kegagalan pengiriman notifikasi tidak boleh menggagalkan proses bisnis utama; pesan gagal masuk retry queue lalu dead-letter queue jika terus gagal.
- **Observability requirements**: notification delivery success/fail rate, queue lag, dead-letter queue size.

---

## Dependency Matrix

| Service | Bergantung pada (synchronous/gRPC) | Bergantung pada (asynchronous/event) |
|---|---|---|
| API Gateway | Auth, Catalog, Order, Payment, Download | — |
| Auth Service | — | — |
| Catalog Service | — | — |
| Order Service | Catalog, Auth | (opsional consume `PaymentSuccess`) |
| Payment Service | Order, Payment Provider (eksternal) | — |
| Download Service | Auth, Order, Catalog, Object Storage (eksternal) | `OrderPaid` (opsional, optimasi) |
| Notification Service | Email Provider (eksternal) | `UserRegistered`, `OrderCreated`, `OrderPaid`, `PaymentFailed`, `DownloadGranted` |

**Catatan arah dependency**: tidak ada circular dependency synchronous. Auth dan Catalog adalah *leaf service* (tidak bergantung pada service domain lain), sehingga dapat tetap tersedia meskipun service lain terganggu — ini sengaja didesain demikian untuk mengurangi blast radius kegagalan.

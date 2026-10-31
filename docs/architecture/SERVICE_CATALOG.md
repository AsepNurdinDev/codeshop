# Service Catalog — CodeShop

## 1. API Gateway

- **Purpose**: Single public entry point untuk seluruh client request.
- **Responsibilities**: routing, TLS termination (di belakang Cloudflare), **coarse** authentication enforcement (validasi struktural/signature JWT, bukan business authorization), rate limiting, request ID/correlation ID injection, request/response transformation (REST publik ↔ gRPC internal). **API Gateway secara eksplisit BUKAN otoritas otorisasi bisnis** — lihat `ARCHITECTURE.md` §11a untuk penjelasan lengkap dan alasan desain.
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

- **Purpose**: Mengelola siklus hidup order, termasuk **sub-domain Cart**. Pemilik tunggal `OrderLifecycle`.
- **Ownership correction (Phase 1.1)**: Order Service adalah **satu-satunya** pemilik `Order`, `OrderItem`, `OrderStatus`, dan `OrderLifecycle` (state machine di `ARCHITECTURE.md` §11b). Transisi `PENDING_PAYMENT → PAID` **hanya** dipicu oleh Order Service sendiri saat mengonsumsi event `PaymentSucceeded` dari Payment Service — tidak ada service lain, termasuk Payment Service, yang diberi izin menulis `OrderStatus` secara langsung (gRPC `UpdateOrderStatus` yang dipanggil pihak eksternal dihapus dari desain; lihat `EVENT_DESIGN.md` §1a untuk riwayat koreksi ini).
- **Responsibilities**: manajemen cart (sebagai order berstatus `DRAFT`), checkout (konversi cart → order `PENDING_PAYMENT`), snapshot harga saat checkout, menjalankan state machine order sepenuhnya (termasuk transisi ke `PAID`, `CANCELLED`, `COMPLETED`), penyediaan data entitlement (order `PAID`/`COMPLETED` = entitlement download berhak diberikan), expiry order `PENDING_PAYMENT` yang tidak kunjung dibayar.
- **Data ownership**: `Cart` (representasi `Order` status `DRAFT`), `CartItem`, `Order`, `OrderItem`.
- **Database**: Order DB (PostgreSQL).
- **APIs**: gRPC internal (`AddCartItem`, `CreateOrder`, `GetOrder` [read-only, dipanggil Payment & Download Service], `CheckEntitlement`) + REST publik (`/cart/*`, `/orders/*`). **Tidak ada** `UpdateOrderStatus` yang dapat dipanggil service lain — perubahan status hanya terjadi secara internal sebagai reaksi terhadap event.
- **Dependencies**: Catalog Service (validasi produk & harga), Auth Service (validasi identitas).
- **Inbound communication**: gRPC read-only dari Gateway, Payment Service (`GetOrder`), Download Service (`CheckEntitlement`); event `PaymentSucceeded`/`PaymentFailed` dari RabbitMQ.
- **Outbound communication**: gRPC ke Catalog & Auth; event `OrderCreated`/`OrderPaid` ke RabbitMQ.
- **Events produced**: `OrderCreated`, `OrderPaid`.
- **Events consumed**: `PaymentSucceeded`, `PaymentFailed` — konsumsi ini **idempotent** (dedup by `event_id`, lihat `EVENT_DESIGN.md` §8) dan merupakan **satu-satunya** mekanisme yang boleh mengubah `OrderStatus` menjadi `PAID`.
- **Security requirements**: hanya pemilik order (ownership check) yang dapat melihat/mengubah order miliknya; price snapshot mencegah manipulasi harga oleh client; lihat Authorization Boundary di `ARCHITECTURE.md` §11a — Order Service tidak mempercayai klaim status apa pun dari Gateway atau service lain tanpa validasi internal.
- **Scaling strategy**: horizontal; write path (checkout, status transition) memerlukan konsistensi transaksional dalam database sendiri.
- **Failure modes**: jika Catalog Service tidak dapat dihubungi saat checkout, checkout ditolak dengan error jelas (fail-fast) — tidak membuat order dengan data produk yang tidak tervalidasi. Jika event `PaymentSucceeded` gagal dikonsumsi berulang kali, pesan masuk DLQ dan order tetap `PENDING_PAYMENT` hingga direkonsiliasi secara manual/operasional (lihat `RELIABILITY.md`).
- **Observability requirements**: cart abandonment rate, checkout success/fail rate, order status distribution, order expiry rate, average order value (business metric), lag antara `PaymentSucceeded` diterima dan `OrderPaid` dipublish (consistency-window metric).

## 5. Payment Service

- **Purpose**: Mengelola integrasi dengan payment provider dan siklus hidup **pembayaran** (bukan siklus hidup order).
- **Ownership correction (Phase 1.1)**: Payment Service **tidak memiliki dan tidak mengubah** `Order`/`OrderStatus`. Payment Service adalah pemilik tunggal atas domain pembayaran: `Payment`, `PaymentTransaction`, `ProviderReference`, `PaymentStatus`, `PaymentMetadata`, `WebhookProcessing`. Order Service tetap pemilik tunggal `Order`/`OrderItem`/`OrderStatus`/`OrderLifecycle` (lihat entri Order Service di bawah). Lihat `ARCHITECTURE.md` §9 dan §11b untuk alur dan state machine lengkap.
- **Responsibilities**: membuat sesi pembayaran ke payment provider, menerima & memverifikasi webhook (lihat `THREAT_MODEL.md` untuk ancaman webhook), menjalankan state machine `Payment` (`INITIATED → PENDING → SUCCEEDED/FAILED/EXPIRED/CANCELLED`), mencatat riwayat transaksi pembayaran, mempublikasikan hasil verifikasi sebagai event (`PaymentSucceeded`/`PaymentFailed`) — **bukan** memanggil Order Service untuk mengubah status order.
- **Data ownership**: `Payment`, `PaymentTransaction`, `ProviderReference`, `PaymentMetadata`, `WebhookProcessing` log.
- **Database**: Payment DB (PostgreSQL).
- **APIs**: gRPC internal (`CreatePayment`, `GetPaymentStatus` — read-only untuk service lain) + REST publik (`/payments/*`) + webhook endpoint (`/payments/webhook`).
- **Dependencies**: Order Service (gRPC, **read-only** — `GetOrder` untuk validasi amount/currency saat membuat sesi pembayaran), Payment Provider (eksternal, REST + webhook).
- **Inbound communication**: gRPC dari Gateway; webhook HTTPS dari Payment Provider (autentikasi via signature, **bukan** JWT user — lihat §Security requirements).
- **Outbound communication**: gRPC read-only ke Order Service; REST ke Payment Provider; event ke RabbitMQ.
- **Events produced**: `PaymentPending` *(opsional/analytics)*, `PaymentSucceeded`, `PaymentFailed`.
- **Events consumed**: tidak ada.
- **Security requirements**: webhook adalah **security boundary terpisah** dari autentikasi user — payload diverifikasi melalui signature/HMAC dari provider, bukan JWT (lihat `THREAT_MODEL.md` §Payment dan ADR pembayaran terkait); idempotency wajib terhadap webhook duplikat via unique constraint pada `ProviderReference`; validasi amount/currency terhadap order sebelum menandai `SUCCEEDED`; data pembayaran sensitif tidak disimpan penuh — hanya referensi/token dari provider (tidak menyimpan data kartu, lihat `DATA_CLASSIFICATION.md`).
- **Scaling strategy**: horizontal; webhook endpoint harus idempotent sehingga aman menerima beban tinggi dan retry.
- **Failure modes**: jika publish event ke RabbitMQ gagal setelah `Payment` berhasil ditransisikan ke `SUCCEEDED`, status pembayaran tetap benar di Payment DB (source of truth); event dipublish ulang via outbox/retry pattern (lihat `RELIABILITY.md`) tanpa mengulang efek samping pada Order Service (idempotent consumer).
- **Observability requirements**: payment success/fail rate, webhook processing latency, webhook duplicate/rejected-signature rate, provider error rate, reconciliation job lag (lihat `OBSERVABILITY.md`).

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
- **Security requirements**: signed URL dengan TTL singkat (mis. beberapa menit), limit jumlah request signed URL per periode waktu untuk mencegah abuse, audit log setiap penerbitan signed URL dan akses, pengecekan `DownloadEntitlement.status != REVOKED` sebelum penerbitan (lihat `ARCHITECTURE.md` §10 Entitlement revocation). Download Service adalah pemegang otoritas otorisasi akhir untuk download — bukan Gateway (lihat `ARCHITECTURE.md` §11a).
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
| Order Service | Catalog, Auth | consume `PaymentSucceeded`, `PaymentFailed` (satu-satunya trigger transisi ke `PAID`) |
| Payment Service | Order (read-only, `GetOrder`), Payment Provider (eksternal) | — |
| Download Service | Auth, Order, Catalog, Object Storage (eksternal) | consume `OrderPaid` (opsional, optimasi entitlement) |
| Notification Service | Email Provider (eksternal) | `UserRegistered`, `OrderCreated`, `OrderPaid`, `PaymentFailed`, `DownloadGranted` |

**Catatan arah dependency**: tidak ada circular dependency synchronous. Auth dan Catalog adalah *leaf service* (tidak bergantung pada service domain lain), sehingga dapat tetap tersedia meskipun service lain terganggu — ini sengaja didesain demikian untuk mengurangi blast radius kegagalan.

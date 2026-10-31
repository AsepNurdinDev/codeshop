# Database Design — CodeShop

Prinsip utama: **Database per Service**. Tidak ada service yang mengakses database milik service lain secara langsung; komunikasi data lintas service hanya melalui gRPC API atau event.

## 1. Entity Ownership Table

| Entity | Service Owner |
|---|---|
| User | Auth Service |
| Role | Auth Service |
| Product | Catalog Service |
| Category | Catalog Service |
| ProductVersion | Catalog Service |
| Cart | Order Service |
| CartItem | Order Service |
| Order | Order Service |
| OrderItem | Order Service |
| Payment | Payment Service |
| PaymentTransaction | Payment Service |
| ProviderReference | Payment Service |
| PaymentMetadata | Payment Service |
| WebhookProcessing (log) | Payment Service |
| DownloadEntitlement | Download Service |
| DownloadRecord | Download Service |
| Notification | Notification Service |
| AuditLog | Terdistribusi — setiap service mencatat audit log miliknya sendiri (lihat catatan §7) |

> Tidak ada SQL/DDL pada dokumen ini sesuai batasan Phase 1. Struktur di bawah bersifat konseptual.

## 2. Auth DB (Auth Service)

### Entities

- **User**: id, email (unique), password_hash, full_name, status (active/suspended), created_at, updated_at.
- **Role**: id, name (`buyer`, `admin`), assigned to User (many-to-many atau simple enum field pada MVP karena hanya 2 role).

### Relationship
`User` 1—N `Role` assignment (atau simplifikasi: field `role` langsung di `User` untuk MVP, karena kompleksitas RBAC penuh belum diperlukan).

### Important Fields
- `email` — unique, digunakan untuk login.
- `password_hash` — hasil Argon2id, tidak pernah di-expose.
- `status` — untuk mendukung suspend user tanpa hapus data.

### Indexes
- Unique index pada `email`.
- Index pada `status` untuk query admin.

### Constraints
- `email` unique dan not null.
- `password_hash` not null.

### Consistency
Strong consistency (single database, transactional).

### Retention
Data user dipertahankan selama akun aktif; kebijakan penghapusan (right to be forgotten) menjadi topik Phase 2 (perlu koordinasi dengan Order/Download untuk data historis).

### Sensitive Data
`email`, `password_hash`, `full_name` — PII, wajib dienkripsi saat transit (TLS) dan dibatasi akses melalui service boundary (tidak ada akses langsung dari service lain).

### Audit Requirements
Login attempt (berhasil/gagal), perubahan password, perubahan role dicatat sebagai audit log di Auth Service.

## 3. Catalog DB (Catalog Service)

### Entities
- **Product**: id, name, slug, description, category_id, base_price, currency, status (draft/published/deactivated), storage_object_key (referensi ke file di object storage), created_at, updated_at.
- **Category**: id, name, slug.
- **ProductVersion**: id, product_id, version_label, changelog, storage_object_key, released_at.

### Relationship
`Category` 1—N `Product`; `Product` 1—N `ProductVersion`.

### Important Fields
- `storage_object_key` pada `Product`/`ProductVersion` — referensi ke lokasi file di private bucket, **tidak pernah** berupa URL publik langsung.
- `status` — mengontrol visibilitas di katalog publik.

### Indexes
- Index pada `slug` (unique per entity untuk keperluan URL).
- Index pada `category_id`, `status` untuk listing/filtering.

### Constraints
- `slug` unique.
- `base_price` >= 0.

### Consistency
Strong consistency dalam Catalog DB. Harga yang digunakan saat checkout adalah **snapshot** yang disalin ke `OrderItem` di Order DB (bukan referensi live), sehingga perubahan harga di Catalog tidak memengaruhi order yang sudah dibuat — ini adalah keputusan eventual-consistency-safe by design.

### Retention
Produk yang deactivated tidak dihapus (soft delete) agar riwayat order/entitlement tetap valid secara referensial (by ID, bukan foreign key lintas database).

### Sensitive Data
Tidak ada data sensitif tingkat tinggi; `storage_object_key` dianggap semi-sensitif karena dapat memberi petunjuk struktur storage internal — akses dibatasi hanya untuk Catalog & Download Service.

### Audit Requirements
Perubahan produk oleh admin (create/update/deactivate) dicatat sebagai audit log.

## 4. Order DB (Order Service)

### Entities
- **Cart**: id, user_id, status (`DRAFT`), created_at, updated_at. *(Cart dimodelkan sebagai Order berstatus DRAFT untuk menghindari duplikasi model — lihat catatan di `ARCHITECTURE.md` §3.)*
- **CartItem**: id, cart_id (order_id), product_id (referensi by ID ke Catalog), product_name_snapshot, price_snapshot, quantity.
- **Order**: id, user_id, status (`DRAFT`/`PENDING_PAYMENT`/`PAID`/`CANCELLED`/`EXPIRED`), total_amount, currency, created_at, updated_at.
- **OrderItem**: id, order_id, product_id, product_name_snapshot, price_snapshot, quantity.

### Relationship
`Order` 1—N `OrderItem`. `CartItem`/`OrderItem` mereferensikan `Product` **by ID saja** (tidak ada foreign key lintas database — cross-service reference by ID adalah pola standar di seluruh sistem ini).

### Important Fields
- `status` — state machine order (lihat alur di `ARCHITECTURE.md`).
- `price_snapshot` — mencegah manipulasi/perubahan harga setelah checkout.

### Indexes
- Index pada `user_id` (untuk listing order milik user).
- Index pada `status` (untuk query operasional/reconciliation).

### Constraints
- `status` hanya dapat bertransisi sesuai state machine yang valid (mis. tidak dapat langsung dari `DRAFT` ke `PAID`).
- `total_amount` = jumlah dari `price_snapshot * quantity` seluruh `OrderItem`.

### Consistency
Strong consistency dalam Order DB untuk transisi status order (transactional). Update status dari `PENDING_PAYMENT` → `PAID` dipicu oleh Payment Service melalui gRPC synchronous call (bukan hanya event), agar status order langsung konsisten begitu payment terverifikasi.

### Retention
Order disimpan permanen sebagai riwayat transaksi (kebutuhan bisnis & kemungkinan kebutuhan pajak/audit).

### Sensitive Data
Tidak menyimpan data pembayaran; hanya referensi transaksi (jika diperlukan, `payment_reference_id`).

### Audit Requirements
Setiap perubahan status order dicatat (audit trail status transition) untuk keperluan dispute/refund.

## 5. Payment DB (Payment Service)

> **Ownership correction (Phase 1.1)**: entity di bawah ini (`Payment`, `PaymentTransaction`, `ProviderReference`, `WebhookProcessing` log) adalah **satu-satunya** domain yang dimiliki Payment Service. Payment Service **tidak** memiliki tabel atau kolom apa pun terkait `Order`/`OrderStatus` — referensi ke order dilakukan murni via `order_id` sebagai foreign-key-by-convention (bukan foreign key fisik lintas database), divalidasi lewat `GetOrder` (read-only gRPC call) saat pembuatan sesi pembayaran.

### Entities
- **Payment**: id, order_id (referensi, read-only), user_id, provider, amount, currency, status (`INITIATED`/`PENDING`/`SUCCEEDED`/`FAILED`/`EXPIRED`/`CANCELLED` — lihat state machine di `ARCHITECTURE.md` §11b), created_at, updated_at. Merepresentasikan satu *intent* pembayaran untuk satu order.
- **PaymentTransaction**: id, payment_id, provider_reference_id, event_type (mis. `charge.succeeded`, `charge.failed`), amount, currency, raw_status_from_provider, received_at. Merepresentasikan **setiap** notifikasi mentah yang diterima dari provider untuk satu `Payment` (bisa lebih dari satu baris per `Payment` jika provider mengirim beberapa event/retry) — dipisahkan dari `Payment` agar riwayat webhook mentah tidak hilang saat status `Payment` disimpulkan.
- **ProviderReference**: id, payment_id, provider_reference_id (unique), provider_session_id. Menyimpan identitas kanonik dari sisi provider, dipisahkan agar idempotency check dapat dilakukan cepat tanpa memindai seluruh `PaymentTransaction`.
- **PaymentMetadata**: id, payment_id, key, value — pasangan metadata non-sensitif tambahan dari provider (mis. metode pembayaran yang dipakai: `bank_transfer`, `card`, dsb.) untuk keperluan analytics, **bukan** untuk menyimpan data instrumen pembayaran mentah.
- **WebhookProcessing** (log): id, provider_reference_id, signature_valid (bool), schema_valid (bool), processed (bool), rejection_reason (nullable), received_at. Mencatat **setiap** webhook masuk termasuk yang ditolak (signature invalid, duplikat, dsb.) — wajib untuk audit dan investigasi (`THREAT_MODEL.md` §Payment).

### Relationship
`Payment` 1—N `PaymentTransaction`; `Payment` 1—1 `ProviderReference` (kanonik); `Payment` 1—N `PaymentMetadata`. `Payment` N—1 `Order` (by ID reference, lintas service, read-only).

### Important Fields
- `ProviderReference.provider_reference_id` — untuk idempotency & rekonsiliasi dengan payment provider.
- `Payment.status` — hasil akhir verifikasi webhook, mengikuti state machine di `ARCHITECTURE.md` §11b.

### Indexes
- Unique index pada `ProviderReference.provider_reference_id` (mencegah duplikasi pemrosesan webhook — dasar idempotency).
- Index pada `Payment.order_id`.
- Index pada `WebhookProcessing.provider_reference_id` dan `received_at` (untuk investigasi replay/duplikat).

### Constraints
- `ProviderReference.provider_reference_id` unique — dasar mekanisme idempotency webhook.
- `Payment.status` hanya dapat bertransisi sesuai state machine yang valid (lihat `ARCHITECTURE.md` §11b) — transisi tidak valid ditolak di application layer dan dicatat sebagai anomali, bukan diterapkan.

### Consistency
Strong consistency di Payment DB. Payment adalah **source of truth untuk status pembayaran saja**. Order Service tetap satu-satunya source of truth untuk status order (lihat §4 di atas) — status order **tidak lagi diupdate secara synchronous oleh Payment Service**; Payment Service hanya mempublikasikan event `PaymentSucceeded`/`PaymentFailed`, dan Order Service yang memutuskan transisi order-nya sendiri (lihat `ARCHITECTURE.md` §9 untuk koreksi desain ini).

### Retention
Disimpan permanen untuk kebutuhan audit finansial dan potensi refund/dispute.

### Sensitive Data
**Tidak menyimpan data kartu/instrumen pembayaran mentah.** Hanya referensi/token dari payment provider. Ini adalah keputusan desain wajib untuk mengurangi scope kepatuhan (mis. PCI-DSS).

### Audit Requirements
Seluruh perubahan status payment dan setiap webhook yang diterima (termasuk yang ditolak karena signature invalid) dicatat.

## 6. Download DB (Download Service)

### Entities
- **DownloadEntitlement**: id, user_id, product_id, order_id (referensi), granted_at, status (`ACTIVE`/`REVOKED`).
- **DownloadRecord**: id, entitlement_id, user_id, product_id, ip_address, requested_at, signed_url_expires_at.

### Relationship
`DownloadEntitlement` 1—N `DownloadRecord`.

### Important Fields
- `status` pada `DownloadEntitlement` — memungkinkan revoke akses (mis. jika terjadi refund/chargeback) tanpa menghapus data historis.
- `signed_url_expires_at` — untuk audit & investigasi abuse.

### Indexes
- Index pada `(user_id, product_id)` — pengecekan entitlement cepat.
- Index pada `entitlement_id` di `DownloadRecord`.

### Constraints
- Kombinasi `(user_id, product_id, order_id)` unique per entitlement untuk mencegah duplikasi.

### Consistency
Strong consistency dalam Download DB. Entitlement dibuat berdasarkan konfirmasi dari Order Service (order status `PAID`), baik secara synchronous (MVP) maupun melalui event `OrderPaid` (opsional optimasi).

### Retention
`DownloadRecord` disimpan untuk keperluan audit abuse; kebijakan retensi jangka panjang (mis. anonymize setelah N bulan) dapat menjadi topik Phase 2.

### Sensitive Data
`ip_address` bersifat semi-sensitif (PII) — retensi dan akses dibatasi.

### Audit Requirements
Setiap penerbitan signed URL dan setiap perubahan status entitlement (grant/revoke) dicatat sebagai audit trail wajib (bagian dari Download Security).

## 7. Notification DB (Notification Service)

### Entities
- **Notification**: id, user_id, type (`EMAIL`), event_source (`UserRegistered`, dll.), recipient, status (`SENT`/`FAILED`/`RETRYING`), sent_at, created_at.

### Important Fields
- `event_source` — untuk traceability terhadap event pemicu.
- `status` — untuk retry logic dan monitoring.

### Indexes
- Index pada `user_id`, `status`.

### Constraints
- Tidak ada constraint lintas service (murni log pengiriman).

### Consistency
Eventual consistency terhadap event sumber (Notification Service adalah consumer asynchronous).

### Retention
Log notifikasi disimpan untuk troubleshooting; dapat di-archive setelah periode tertentu (Phase 2).

### Sensitive Data
`recipient` (email) — PII, retensi dibatasi sesuai kebijakan privasi.

### Audit Requirements
Kegagalan pengiriman berulang (masuk dead-letter) dicatat untuk investigasi operasional.

---

## 10. Cross-Service Reference Pattern

Karena tidak ada foreign key lintas database, seluruh referensi antar service (mis. `Order.user_id` merujuk `User` di Auth DB) menggunakan **reference by ID** yang divalidasi pada saat penulisan data (melalui gRPC call), bukan dijamin oleh database constraint. Ini adalah trade-off standar pada arsitektur database-per-service, didokumentasikan sebagai keputusan sadar, bukan kelalaian desain.

## 11. Database Reliability

Bagian ini melengkapi §1–§9 dengan aspek operasional database yang menjadi kebutuhan wajib untuk produksi (detail implementasi/skrip tetap Phase implementasi, bukan Phase 1.1).

### Backup & Restore

- Setiap database PostgreSQL per service memiliki jadwal backup independen (lihat `BACKUP_STRATEGY.md` untuk frekuensi dan retensi per service — Payment DB dan Order DB mendapat prioritas RPO lebih ketat dibanding Notification DB).
- Restore diuji secara berkala, bukan diasumsikan berhasil (lihat prinsip di `BACKUP_STRATEGY.md`: "backup belum valid sampai restore teruji").

### Replication

- Production baseline: satu primary + minimal satu read replica untuk database dengan beban baca tinggi (Catalog DB) untuk memisahkan beban baca dari beban tulis dan sebagai langkah awal menuju high availability.
- Database lain (Auth, Order, Payment, Download, Notification) dapat memulai tanpa read replica pada baseline produksi awal, dengan replica ditambahkan berdasarkan data beban nyata (lihat `TRADE_OFFS.md` — menghindari over-provisioning sejak awal).
- Replication tidak menggantikan backup — replica melindungi dari kegagalan node/availability, bukan dari kesalahan logis (mis. penghapusan data keliru yang ikut ter-replikasi).

### Connection Pooling

- Setiap service menggunakan connection pool ke database miliknya dengan batas maksimum koneksi yang wajar, mencegah satu service menghabiskan seluruh slot koneksi database saat lonjakan trafik.
- Pertimbangan pooler eksternal (mis. PgBouncer) sebagai lapisan tambahan didokumentasikan sebagai opsi Phase implementasi jika jumlah replica pod per service bertumbuh besar dan mendekati batas `max_connections` PostgreSQL.

### Migrations

- Skema database dikelola melalui migration tool per service (versioned, forward-only sebagai default, dengan kemampuan rollback terbatas didokumentasikan per migration) — **tidak ada file migration SQL yang dibuat pada Phase 1.1 ini**, sesuai batasan dokumentasi-only.
- Migration dijalankan sebagai langkah terpisah dari deployment aplikasi (bukan otomatis saat pod start) untuk menghindari race condition saat multiple replica pod start bersamaan — detail mekanisme (init job, dsb.) adalah Phase implementasi.

### Schema Ownership & Transaction Boundaries

- Skema database dimiliki sepenuhnya oleh satu service (selaras ADR-002) — perubahan skema hanya dilakukan oleh tim/pipeline yang memiliki service tersebut.
- Transaksi database tidak pernah melintasi batas service (tidak ada distributed transaction/2PC lintas database) — konsistensi lintas service dicapai melalui pola saga sederhana berbasis event (lihat `EVENT_DESIGN.md`, `ARCHITECTURE.md` §9), bukan transaksi terdistribusi yang menambah kompleksitas signifikan.

### Consistency

- Strong consistency dijaga **di dalam** boundary satu database/service (ACID standar PostgreSQL).
- Konsistensi **lintas** service bersifat eventual, dengan window yang dipantau (lihat `RELIABILITY.md` §12, `OBSERVABILITY.md` §3).

### Disaster Recovery & Retention

- Lihat `DISASTER_RECOVERY.md` untuk RPO/RTO per database dan skenario kegagalan.
- Retensi data per entity mengikuti klasifikasi di `DATA_CLASSIFICATION.md` dan catatan retensi per database di §2–§7 dokumen ini.

## 12. Entity Relationship Overview (Cross-Service)

```mermaid
erDiagram
    USER ||--o{ ORDER : places
    USER ||--o{ DOWNLOAD_ENTITLEMENT : owns
    CATEGORY ||--o{ PRODUCT : contains
    PRODUCT ||--o{ PRODUCT_VERSION : has
    PRODUCT ||--o{ ORDER_ITEM : "referenced by"
    ORDER ||--o{ ORDER_ITEM : contains
    ORDER ||--o| PAYMENT : "paid via"
    ORDER ||--o{ DOWNLOAD_ENTITLEMENT : grants
    DOWNLOAD_ENTITLEMENT ||--o{ DOWNLOAD_RECORD : "accessed via"
```

> Diagram di atas menggambarkan relasi **logis lintas service** (bukan foreign key fisik). Garis penghubung merepresentasikan reference by ID yang divalidasi via API, bukan constraint database.

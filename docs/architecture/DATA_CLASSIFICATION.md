# Data Classification — CodeShop

## 1. Purpose

Dokumen ini mengklasifikasikan data yang dikelola CodeShop agar setiap service, log, dan proses backup/observability menerapkan kontrol yang proporsional terhadap sensitivitas data — tidak under-protecting data sensitif, dan tidak over-engineering kontrol untuk data publik.

## 2. Classification Levels

| Level | Definisi |
|---|---|
| **Public** | Boleh diakses siapa saja tanpa autentikasi; kebocoran tidak menimbulkan dampak. |
| **Internal** | Tidak dipublikasikan, tetapi kebocoran berdampak rendah–menengah (operasional/reputasi ringan). |
| **Sensitive** | Data pribadi atau data bisnis yang kebocorannya berdampak signifikan pada user atau bisnis. |
| **Highly Sensitive** | Kebocoran berdampak kritis: kerugian finansial langsung, pelanggaran hukum/privasi berat, atau kompromi sistem lebih lanjut. |

## 3. Data Inventory & Classification

| Data | Klasifikasi | Owner Service | Storage | Encryption | Access Control | Retention | Logging Restriction |
|---|---|---|---|---|---|---|---|
| Product metadata (nama, deskripsi, harga, kategori) | Public | Catalog Service | Catalog DB | TLS in transit; at-rest sesuai kebijakan infrastruktur | Read: publik; Write: admin only | Selama produk aktif + soft-delete permanen | Boleh muncul penuh di log/analytics. |
| Product source file | Highly Sensitive | Catalog/Download Service (referensi), Object Storage (fisik) | Private object storage bucket | TLS in transit; at-rest encryption pada object storage | Hanya via signed URL tervalidasi entitlement | Selama produk listing aktif; retensi setelah takedown mengikuti kebijakan hukum/kontrak | `storage_object_key` tidak pernah dicatat di log yang dapat diakses luas; access log dicatat terpisah sebagai audit trail (`DownloadRecord`). |
| User profile (nama, email) | Sensitive | Auth Service | Auth DB | TLS in transit; at-rest encryption direkomendasikan | Read: pemilik akun + admin terbatas; Write: pemilik akun | Selama akun aktif; kebijakan penghapusan Phase 2 | Email di-mask sebagian di log aplikasi umum; boleh penuh di Auth DB dan audit log akses terbatas. |
| Password hash | Highly Sensitive | Auth Service | Auth DB | Hashing Argon2id (bukan reversible encryption) | Tidak pernah di-expose via API; tidak pernah dicatat di log | Selama akun aktif | **Tidak boleh muncul di log dalam bentuk apa pun**, termasuk log debug. |
| Access/refresh token | Highly Sensitive | Auth Service (session), client (in-memory) | Redis (refresh session) | TLS in transit; Redis akses cluster-internal only | Hanya pemilik sesi; revocable | TTL pendek (access), TTL menengah dengan rotation (refresh) | Token value tidak pernah dicatat penuh di log — gunakan hash/prefix jika perlu debugging. |
| Order data (item, harga snapshot) | Internal–Sensitive | Order Service | Order DB | TLS in transit | Read: pemilik order + admin; Write: Order Service internal only | Permanen (riwayat transaksi/audit bisnis) | Boleh dicatat di log operasional tanpa data pembayaran. |
| Payment reference (`ProviderReference`, status) | Highly Sensitive | Payment Service | Payment DB | TLS in transit; at-rest encryption direkomendasikan | Read: pemilik + admin finansial; Write: Payment Service internal only (via webhook terverifikasi) | Permanen (audit finansial) | Referensi transaksi boleh dicatat; **tidak pernah** menyimpan/mencatat data instrumen pembayaran mentah (nomor kartu, dsb.) — CodeShop secara desain tidak pernah menerima data tersebut. |
| Download entitlement & record | Sensitive | Download Service | Download DB | TLS in transit | Read: pemilik + admin; Write: Download Service internal only | Entitlement permanen; `DownloadRecord` (termasuk IP) dengan kebijakan retensi lebih pendek (Phase 2: anonymize setelah N bulan) | IP address diperlakukan sebagai PII — dibatasi pada audit trail, tidak dipublikasikan ke dashboard umum. |
| Audit log (login, checkout, payment verification, download) | Sensitive | Masing-masing service pemilik domain | DB/log store masing-masing service | TLS in transit | Read: admin/security terbatas | Retensi lebih panjang dari data operasional biasa (kebutuhan investigasi) | Audit log itu sendiri adalah kontrol keamanan — akses ke audit log dibatasi lebih ketat daripada data yang dicatatnya. |
| Application secrets (DB credential, JWT signing key, webhook secret, object storage key) | Highly Sensitive | Infrastruktur (bukan domain bisnis) | Kubernetes Secrets / secret manager (lihat ADR-013) | Encryption at-rest wajib | Least privilege per service, tidak pernah di Git dalam bentuk plaintext | Rotasi berkala | **Tidak pernah** dicatat di log aplikasi maupun log CI/CD. |
| Infrastructure credentials (cluster access, registry credential, cloud provider key) | Highly Sensitive | Platform/DevOps | Secret manager, CI/CD secret store | Encryption at-rest wajib | Terbatas ke pipeline CI/CD dan operator terotorisasi | Rotasi berkala | Tidak pernah dicatat di log CI/CD job output. |

## 4. Cross-Cutting Rules

- Data **Highly Sensitive** tidak boleh direplikasi ke lingkungan non-produksi (staging/dev) dalam bentuk apa adanya — lihat `ENVIRONMENT_STRATEGY.md` untuk kebijakan data masking/seeding di staging.
- Data **Sensitive/Highly Sensitive** tidak boleh muncul dalam trace/span attribute pada distributed tracing (lihat `OBSERVABILITY.md` §Log/trace correlation) — hanya ID referensi yang boleh muncul.
- Setiap penambahan field data baru oleh tim wajib diklasifikasikan mengikuti tabel di atas sebelum diimplementasikan (bagian dari definition of done, lihat `PRODUCTION_READINESS.md`).

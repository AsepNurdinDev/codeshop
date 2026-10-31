# Disaster Recovery — CodeShop

## 1. Scope & Realism

Target di dokumen ini disusun untuk **deployment produksi awal skala indie/startup kecil** — bukan sistem hyperscale multi-region. Target RPO/RTO dipilih agar realistis dicapai dengan tim kecil dan biaya proporsional, dengan catatan eksplisit bahwa target ini dapat diperketat setelah bisnis tumbuh (lihat `SCALABILITY` roadmap di `TRADE_OFFS.md`).

## 2. RPO / RTO Targets

| Komponen | RPO (maksimum data hilang) | RTO (maksimum downtime pemulihan) | Justifikasi |
|---|---|---|---|
| Payment DB | ≤ 15 menit | ≤ 2 jam | Data finansial paling kritikal — prioritas RPO tertinggi. |
| Order DB | ≤ 15 menit | ≤ 2 jam | Terhubung erat dengan integritas transaksi. |
| Auth DB | ≤ 1 jam | ≤ 2 jam | Kehilangan sesi/registrasi terbaru dapat diterima dalam batas wajar; dampak utama adalah user perlu login ulang, bukan kehilangan data finansial. |
| Download DB (entitlement) | ≤ 1 jam | ≤ 2 jam | Entitlement dapat direkonsiliasi dari Order DB (order PAID) jika perlu, sebagai lapisan pemulihan tambahan. |
| Catalog DB | ≤ 24 jam | ≤ 4 jam | Perubahan katalog relatif jarang dan dapat diinput ulang oleh admin bila perlu; dampak downtime lebih rendah karena ada cache. |
| Notification DB | ≤ 24 jam | ≤ 4 jam | Data log pengiriman, dampak kehilangan rendah. |
| Object Storage (produk digital) | ≤ 24 jam (mengandalkan versioning/replication provider) | ≤ 4 jam | File produk jarang berubah setelah upload; prioritas utama adalah *tidak pernah kehilangan* file yang sudah terjual, dicapai melalui retensi/versioning bawaan object storage, bukan RPO ketat gaya database transaksional. |
| RabbitMQ (definitions & in-flight message) | Best-effort (durable queue) | ≤ 1 jam | Pesan yang hilang dapat dimitigasi melalui reconciliation (lihat `ARCHITECTURE.md` §9 failure table); definitions (exchange/queue/binding) di-backup sebagai konfigurasi, idealnya melalui GitOps/IaC di Phase implementasi. |
| Kubernetes cluster (workload state) | N/A (stateless, dideklarasikan di Git) | ≤ 2 jam untuk membangun ulang cluster dari GitOps repo | Karena GitOps (ADR-011), state workload dapat direkonstruksi penuh dari Git — cluster baru cukup di-bootstrap dan Argo CD melakukan sync. |

## 3. Backup Frequency & Retention

Lihat `BACKUP_STRATEGY.md` untuk detail teknis per komponen. Ringkasan kebijakan retensi:

- Database kritikal (Payment, Order): backup penuh harian + point-in-time recovery (WAL/transaction log) untuk mencapai RPO ≤ 15 menit; retensi 30 hari.
- Database non-kritikal (Catalog, Notification, Auth): backup penuh harian; retensi 14 hari.
- Object storage: versioning diaktifkan; retensi mengikuti kebijakan bisnis/kontrak produk digital, bukan kebijakan teknis semata.

## 4. Restore Testing

- Restore **wajib diuji secara berkala** (bukan hanya diasumsikan berhasil) — lihat prinsip di `BACKUP_STRATEGY.md`. Jadwal uji restore minimal setiap kuartal untuk database kritikal (Payment, Order), dan setelah setiap perubahan signifikan pada skema/infrastruktur backup.
- Hasil uji restore didokumentasikan (waktu yang dibutuhkan, apakah RTO tercapai) sebagai bagian dari evidence kesiapan produksi (`PRODUCTION_READINESS.md`).

## 5. Failure Scenarios

### Database Failure (single service DB down/corrupt)

- Dampak terisolasi ke domain service tersebut (selaras ADR-002) — service lain tetap berjalan, meskipun fitur yang bergantung pada data service yang down akan terganggu (mis. Payment DB down → checkout baru tidak dapat memulai pembayaran, tetapi browsing katalog tetap normal).
- Pemulihan: restore dari backup terbaru + replay WAL untuk meminimalkan data loss (mencapai RPO di §2).

### Object Storage Failure

- Dampak: download baru gagal (Download Service mengembalikan error jelas, lihat `ARCHITECTURE.md` §12); upload produk baru oleh admin tertunda.
- Pemulihan: bergantung pada mekanisme durability/replication bawaan provider object storage (S3-compatible umumnya memiliki durability tinggi secara native); jika self-hosted (MinIO), diperlukan strategi replikasi/backup terpisah yang didokumentasikan sebagai kebutuhan wajib sebelum go-live dengan MinIO self-hosted.

### Kubernetes Node Failure

- Node tunggal gagal: Kubernetes scheduler memindahkan pod ke node lain secara otomatis (self-healing bawaan); PodDisruptionBudget dan anti-affinity (`KUBERNETES_DESIGN.md` §6) memastikan tidak seluruh replica satu service hilang bersamaan.
- Kegagalan skala cluster (mis. control plane down jika self-managed): mitigasi utama adalah menggunakan managed Kubernetes (control plane dikelola provider) untuk mengurangi permukaan kegagalan ini pada tim kecil — didokumentasikan sebagai preferensi arah implementasi, bukan keputusan final teknologi provider.

### RabbitMQ Failure

- Broker down sementara: producer melakukan retry dengan backoff (`RELIABILITY.md` §3); consumer akan menerima backlog pesan setelah broker pulih (durable queue).
- Broker down berkepanjangan: fitur asynchronous (notifikasi, pembuatan entitlement via event opsional) tertunda; jalur kritikal (status pembayaran, status order) tetap konsisten karena masing-masing tersimpan durable di database pemiliknya (lihat `ARCHITECTURE.md` §9) — RabbitMQ bukan source of truth untuk data apa pun, hanya jalur pengiriman event.

### Region/Site Failure

- **Di luar scope MVP/Production Baseline** (lihat `REQUIREMENTS.md` Non-Goals, `ENVIRONMENT_STRATEGY.md`). CodeShop pada tahap ini berjalan single-region. Dampak site failure penuh = downtime penuh hingga infrastruktur dipulihkan secara manual di region yang sama atau region baru menggunakan GitOps repo (workload) + restore backup (data) — proses ini **tidak** dirancang untuk otomatis/near-zero-downtime pada tahap ini, dan itu adalah trade-off yang diterima secara sadar, bukan kelalaian (lihat `TRADE_OFFS.md`).

## 6. Explicit Non-Goals

- Multi-region active-active atau active-passive automatic failover.
- RPO mendekati nol (near-zero data loss) untuk seluruh database — hanya diterapkan pada domain paling kritikal (Payment, Order).
- Recovery otomatis tanpa intervensi manusia untuk skenario region failure — masih memerlukan operator manusia menjalankan prosedur pemulihan pada tahap kematangan ini.

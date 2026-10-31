# Backup Strategy — CodeShop

> **Prinsip utama**: *Backup is not considered valid until restore has been tested.* Sebuah backup yang belum pernah diverifikasi melalui proses restore nyata dianggap **tidak dapat diandalkan** sampai dibuktikan sebaliknya — ini adalah kebijakan, bukan slogan, dan tercermin di jadwal uji restore pada `DISASTER_RECOVERY.md` §4.

## 1. PostgreSQL (per service)

| Aspek | Kebijakan |
|---|---|
| Metode | Backup penuh terjadwal + continuous WAL archiving (untuk point-in-time recovery) pada database kritikal (Payment, Order). |
| Frekuensi (Payment, Order DB) | Full backup harian + WAL kontinu → mendukung RPO ≤ 15 menit (`DISASTER_RECOVERY.md` §2). |
| Frekuensi (Auth, Catalog, Download, Notification DB) | Full backup harian. |
| Retensi | 30 hari untuk Payment/Order DB; 14 hari untuk database lain. |
| Lokasi penyimpanan | Terpisah secara fisik/logis dari cluster produksi (mis. object storage terpisah/region terpisah dari database asal) — backup yang disimpan di tempat yang sama dengan sumbernya bukan strategi DR yang valid. |
| Enkripsi | Backup terenkripsi at-rest (selaras `DATA_CLASSIFICATION.md` untuk data Sensitive/Highly Sensitive). |
| Uji restore | Kuartalan untuk Payment/Order DB; semesteran untuk database lain; wajib setelah perubahan skema besar. |

## 2. Redis

- Redis di CodeShop **tidak menyimpan data yang menjadi source of truth** (cache katalog, session/refresh token, rate-limit counter) — lihat ADR-007.
- Backup Redis **tidak diperlukan sebagai jalur pemulihan data kritikal**. Kehilangan Redis berdampak pada: cache miss (Catalog Service fallback ke database, lihat `RELIABILITY.md` §8) dan user perlu login ulang (refresh token session hilang) — bukan kehilangan data bisnis permanen.
- Snapshot Redis (RDB) opsional diaktifkan semata untuk mempercepat pemulihan performa cache setelah restart terjadwal, bukan sebagai kebutuhan disaster recovery wajib.

## 3. RabbitMQ

- **Message di dalam queue**: tidak di-backup secara tradisional — durability dicapai melalui replikasi broker (mirrored queue/quorum queue, detail Phase implementasi) dan durable queue configuration, bukan snapshot berkala.
- **Definitions** (exchange, queue, binding, user/permission): idealnya didefinisikan sebagai kode/konfigurasi versioned di GitOps repo (Infrastructure as Config), sehingga "backup"-nya secara efektif adalah Git history — dicatat sebagai arah target Phase implementasi.
- Prinsip: RabbitMQ bukan source of truth data bisnis (lihat `DISASTER_RECOVERY.md` §5 RabbitMQ Failure) — sehingga strategi backupnya berfokus pada **ketersediaan cepat**, bukan **durabilitas data jangka panjang**.

## 4. Object Storage

| Aspek | Kebijakan |
|---|---|
| Metode | Versioning diaktifkan pada bucket produk digital (mencegah overwrite/delete tidak sengaja kehilangan versi sebelumnya); replikasi lintas region/provider dipertimbangkan sebagai peningkatan Future jika volume/nilai bisnis produk bertumbuh signifikan. |
| Retensi | Mengikuti kebijakan bisnis produk digital (file yang sudah pernah terjual idealnya tidak pernah dihapus permanen selama entitlement pembeli masih berlaku) — bukan kebijakan retensi teknis semata. |
| Uji restore | Verifikasi berkala bahwa file dapat diambil kembali dari versi backup/snapshot storage. |

## 5. Configuration & Secrets Recovery

- Konfigurasi aplikasi (ConfigMap) dan struktur deployment sepenuhnya berada di GitOps repository — pemulihannya adalah `git clone` + Argo CD sync, tidak memerlukan backup terpisah (selaras ADR-011).
- **Secrets** memerlukan strategi pemulihan terpisah dari kode (karena secret tidak boleh berada di Git plaintext, lihat ADR-013) — backup secret dilakukan melalui mekanisme secret manager yang dipilih (lihat ADR-013), dengan akses pemulihan dibatasi ke operator yang berwenang dan dicatat sebagai audit trail.

## 6. Kubernetes Configuration

- Seluruh manifest/Helm values berada di GitOps repository (`ADR-011`) — cluster baru dapat dibangun ulang dari Git tanpa "backup Kubernetes" tradisional, selama GitOps repo itu sendiri terjaga (lihat §7).
- Resource yang dibuat di luar GitOps (jika ada, idealnya tidak ada — lihat prinsip GitOps) tidak tercakup oleh strategi ini secara desain, yang merupakan alasan tambahan mengapa perubahan manual ke cluster harus dihindari.

## 7. GitOps Repository

- GitOps repository (berisi seluruh definisi deployment) di-host di penyedia Git terkelola (mis. GitHub) yang sudah memiliki durability/replikasi bawaan; **tidak memerlukan proses backup tambahan di luar praktik Git standar** (mis. tidak melakukan force-push destruktif ke `main`, branch protection).
- Sebagai lapisan tambahan, clone lokal/mirror periodik dapat dipertimbangkan sebagai mitigasi terhadap skenario ekstrem (akun penyedia Git terkunci) — dicatat sebagai opsi, bukan kebutuhan wajib pada tahap ini.

## 8. Backup Validation Checklist (ringkasan operasional)

- [ ] Backup terjadwal berjalan sesuai frekuensi yang ditentukan (dimonitor, bukan diasumsikan — lihat `OBSERVABILITY.md`).
- [ ] Backup tersimpan di lokasi terpisah dari sumber data.
- [ ] Backup terenkripsi sesuai klasifikasi data.
- [ ] Restore telah diuji dalam periode yang ditentukan dan RTO tercapai.
- [ ] Hasil uji restore didokumentasikan (bagian dari `PRODUCTION_READINESS.md`).

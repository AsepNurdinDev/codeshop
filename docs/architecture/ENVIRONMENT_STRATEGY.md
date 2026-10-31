# Environment Strategy — CodeShop

## 1. Pragmatic Principle

CodeShop tidak memerlukan empat lingkungan fisik penuh (Local, Development, Staging, Production) yang identik sejak hari pertama. Untuk skala tim indie/kecil, dokumen ini mendefinisikan **tiga lingkungan bermakna** secara operasional, dengan "Development" digabung ke dalam alur Local + Staging agar tidak menambah biaya/kompleksitas infrastruktur tanpa manfaat proporsional.

```text
Local (developer machine)
   ↓
Staging (shared, production-like)
   ↓
Production
```

> "Development" sebagai cluster terpisah **tidak dibuat** pada tahap ini — kebutuhan iterasi cepat developer sudah terpenuhi oleh Local, dan kebutuhan validasi terhadap infrastruktur mendekati produksi terpenuhi oleh Staging. Menambah cluster Development terpisah pada skala tim ini hanya menambah biaya operasional tanpa manfaat yang jelas (lihat `TRADE_OFFS.md`).

## 2. Local

- **Purpose**: iterasi cepat developer, debugging, menjalankan satu atau beberapa service sekaligus dengan dependency (PostgreSQL, Redis, RabbitMQ, MinIO) berjalan sebagai container lokal.
- **Data**: data sintetis/seed, tidak pernah menyalin data produksi.
- **Configuration**: environment variable lokal (`.env` per service, tidak di-commit), nilai default aman untuk pengembangan.
- **Secrets**: dummy/placeholder secret untuk integrasi eksternal (mis. payment provider sandbox key), tidak pernah secret produksi.
- **Deployment strategy**: Docker Compose (bukan Kubernetes) — proporsional untuk kebutuhan lokal, cepat untuk startup/iterasi, tidak memerlukan cluster. *(Catatan: file Docker Compose adalah scope Phase 2 implementasi, tidak dibuat pada Phase 1.1 ini.)*
- **Observability**: opsional/minimal — log ke stdout cukup untuk debugging lokal, tidak memerlukan stack Prometheus/Grafana/Loki penuh berjalan lokal (dapat dijalankan opsional jika developer ingin menguji instrumentasi).
- **Access control**: tidak relevan (single-user, mesin developer sendiri).

## 3. Staging

- **Purpose**: validasi end-to-end pada infrastruktur yang mendekati produksi (Kubernetes, observability stack, GitOps) sebelum promosi ke production; tempat automated verification pada release pipeline (`CI_CD_DESIGN.md` §4) dijalankan.
- **Data**: data sintetis yang realistis secara volume/bentuk, **atau** subset data produksi yang sudah di-mask/anonymize untuk field Sensitive/Highly Sensitive (lihat `DATA_CLASSIFICATION.md`) — data produksi mentah tidak pernah disalin langsung ke staging.
- **Configuration**: menyerupai production (namespace/Helm values terpisah di GitOps repo), tetapi menunjuk ke payment provider **sandbox/test mode**, bukan live payment gateway.
- **Secrets**: secret staging terpisah sepenuhnya dari secret production (kredensial database berbeda, signing key berbeda) — kompromi staging tidak boleh berdampak ke production.
- **Deployment strategy**: continuous deployment otomatis dari main branch pipeline (lihat `CI_CD_DESIGN.md` §4) via Argo CD.
- **Observability**: stack penuh (Prometheus/Grafana/Loki/Tempo) berjalan, digunakan untuk memvalidasi dashboard/alert sebelum dipercaya di production.
- **Access control**: dibatasi ke tim internal (bukan publik); tidak diindeks mesin pencari.

## 4. Production

- **Purpose**: melayani user nyata dan transaksi finansial nyata.
- **Data**: data user dan transaksi nyata — tunduk penuh pada `DATA_CLASSIFICATION.md`.
- **Configuration**: dikelola sepenuhnya via GitOps repo (production overlay), tidak ada perubahan manual (`kubectl edit`) yang diperbolehkan tanpa tercermin di Git (lihat ADR-011).
- **Secrets**: secret production paling ketat — akses terbatas ke jumlah orang/sistem minimal, rotasi berkala (lihat ADR-013).
- **Deployment strategy**: memerlukan approval eksplisit (promotion gate, lihat `CI_CD_DESIGN.md` §4 dan §6), rolling deployment sebagai strategi awal (lihat `PRODUCTION_READINESS.md`/`CI_CD_DESIGN.md` untuk perbandingan strategi deployment).
- **Observability**: stack penuh dengan alerting aktif ke on-call (lihat `OBSERVABILITY.md`, `INCIDENT_RESPONSE.md`).
- **Access control**: paling ketat — akses cluster dibatasi RBAC Kubernetes per-role, akses database langsung dibatasi hanya untuk kebutuhan operasional darurat dengan audit trail.

## 5. Environment Parity

Untuk menghindari "works in staging, breaks in production", prinsip yang dipegang:

- Staging dan Production menjalankan **image container yang identik** (digest sama, lihat `CI_CD_DESIGN.md` §7) — perbedaan hanya pada konfigurasi (Helm values), bukan kode.
- Topologi service (jumlah service, dependency graph) identik antara Staging dan Production; yang berbeda adalah **skala** (jumlah replica, ukuran resource request/limit) — Staging boleh berjalan dengan replica minimal untuk efisiensi biaya.
- Payment provider di Staging selalu **sandbox mode** — ini adalah satu-satunya perbedaan integrasi eksternal yang disengaja dan didokumentasikan secara eksplisit di sini agar tidak mengejutkan tim.

## 6. What Is Explicitly Not Required at This Stage

- Cluster Kubernetes terpisah untuk "Development" — ditolak, lihat §1.
- Multi-region Staging/Production — di luar scope (lihat `REQUIREMENTS.md` Non-Goals dan `DISASTER_RECOVERY.md`).
- Ephemeral per-PR preview environment — dipertimbangkan sebagai peningkatan Future, bukan kebutuhan MVP/Production Baseline (lihat `TRADE_OFFS.md`).

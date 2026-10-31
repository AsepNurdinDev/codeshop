# Architecture Decision Records — CodeShop

Direktori ini berisi Architecture Decision Records (ADR) untuk project CodeShop. Setiap ADR mendokumentasikan satu keputusan arsitektur penting beserta konteks, alternatif yang dipertimbangkan, dan konsekuensinya.

**Untuk engineer baru**: mulai dari `../architecture/REQUIREMENTS.md` dan `../architecture/ARCHITECTURE.md` untuk gambaran besar, lalu `../architecture/SERVICE_CATALOG.md` untuk detail per service, sebelum membaca ADR di bawah untuk memahami *mengapa* keputusan tertentu diambil. Lihat juga `Documentation Index` di bagian bawah file ini.

## Index

| ADR | Judul | Status |
|---|---|---|
| [ADR-001](./ADR-001-microservices-architecture.md) | Microservices Architecture | Accepted |
| [ADR-002](./ADR-002-database-per-service.md) | Database per Service | Accepted |
| [ADR-003](./ADR-003-grpc-internal-communication.md) | gRPC untuk Komunikasi Internal | Accepted |
| [ADR-004](./ADR-004-rest-public-api.md) | REST untuk Public API | Accepted |
| [ADR-005](./ADR-005-rabbitmq-messaging.md) | RabbitMQ untuk Asynchronous Messaging | Accepted |
| [ADR-006](./ADR-006-postgresql-database.md) | PostgreSQL sebagai Database Utama | Accepted |
| [ADR-007](./ADR-007-redis.md) | Redis untuk Cache & Session | Accepted |
| [ADR-008](./ADR-008-object-storage.md) | Object Storage untuk File Produk Digital | Accepted |
| [ADR-009](./ADR-009-signed-download-url.md) | Temporary Signed URL untuk Download | Accepted |
| [ADR-010](./ADR-010-kubernetes.md) | Kubernetes sebagai Target Deployment Platform | Accepted |
| [ADR-011](./ADR-011-gitops-argo-cd.md) | GitOps dengan Argo CD | Accepted |
| [ADR-012](./ADR-012-opentelemetry.md) | OpenTelemetry untuk Observability (Instrumentasi) | Accepted |
| [ADR-013](./ADR-013-secrets-management.md) | Secrets Management | Accepted |
| [ADR-014](./ADR-014-container-supply-chain-security.md) | Container & Supply Chain Security | Accepted |
| [ADR-015](./ADR-015-threat-modeling.md) | Threat Modeling Practice | Accepted |
| [ADR-016](./ADR-016-production-deployment-strategy.md) | Production Deployment Strategy (Rolling) | Accepted |
| [ADR-017](./ADR-017-observability-architecture.md) | Observability Architecture (SLI/SLO & Alerting) | Accepted |
| [ADR-018](./ADR-018-disaster-recovery-strategy.md) | Disaster Recovery Strategy | Accepted |

> **Catatan Phase 1.1**: ADR-012 (OpenTelemetry) mencakup keputusan *instrumentasi*; ADR-017 melengkapi dengan keputusan *arsitektur observability* (SLI/SLO, dashboard, alerting) di atasnya. Keduanya saling melengkapi, tidak tumpang tindih — ADR-012 menjawab "standar instrumentasi apa", ADR-017 menjawab "bagaimana data itu digunakan untuk menjaga sistem tetap sehat".

## Ownership Correction Recorded Outside a Dedicated ADR

Koreksi ownership Payment↔Order (Payment Service tidak lagi memiliki/mengubah `OrderStatus`) yang dilakukan pada Phase 1.1 **tidak dibuat sebagai ADR baru**, karena ini bukan keputusan arsitektur baru melainkan **perbaikan konsistensi** terhadap prinsip yang sudah diputuskan di ADR-001 dan ADR-002 (single ownership per aggregate/service). Riwayat dan alasan koreksi ini didokumentasikan secara rinci di `../architecture/EVENT_DESIGN.md` §1a dan tercermin di `../architecture/ARCHITECTURE.md` §9, §11b, `../architecture/SERVICE_CATALOG.md` §4–§5, dan `../architecture/DATABASE_DESIGN.md` §5.

## ADR yang Tidak Dibuat

Sesuai instruksi untuk tidak membuat keputusan palsu, berikut kandidat keputusan yang **dievaluasi tetapi tidak menghasilkan ADR terpisah**, beserta alasannya:

- **Cart sebagai service terpisah** — tidak menjadi ADR tersendiri karena keputusan untuk *tidak* membuat service terpisah bukan keputusan arsitektur besar yang berdiri sendiri; ini adalah bagian dari evaluasi service boundary yang sudah didokumentasikan secara memadai di `../architecture/ARCHITECTURE.md` §3 dan `../architecture/SERVICE_CATALOG.md`.
- **Pemilihan Payment Provider spesifik (mis. Midtrans/Xendit/Stripe)** — bukan keputusan arsitektur tetapi keputusan vendor/bisnis yang dapat berubah tanpa memengaruhi arsitektur (Payment Service didesain agnostik terhadap provider tertentu). Tidak dibuat ADR karena akan menjadi keputusan palsu jika dipaksakan pada Phase 1 tanpa evaluasi vendor yang sebenarnya.
- **Multi-region / multi-cluster deployment** — di luar scope MVP/Production Baseline (lihat `../architecture/REQUIREMENTS.md` Non-Goals dan `../architecture/DISASTER_RECOVERY.md`), dicatat sebagai Future di `../architecture/TRADE_OFFS.md`. ADR akan dibuat pada fase ketika kebutuhan itu benar-benar muncul, bukan diputuskan secara spekulatif sekarang.
- **Search engine terpisah (Elasticsearch/Meilisearch, dst.)** — dievaluasi dan ditolak untuk MVP (lihat `../architecture/ARCHITECTURE.md` §3). Karena hasil evaluasi adalah "tidak dipakai", ini tidak memenuhi kriteria ADR; alasan penolakan cukup didokumentasikan di `ARCHITECTURE.md`.
- **Service mesh penuh (mTLS sidecar, dsb.)** — dievaluasi dan ditolak untuk tahap ini karena kompleksitas operasional tidak proporsional (lihat `../architecture/NETWORK_DESIGN.md` §2 dan `../architecture/TRADE_OFFS.md`); dicatat sebagai kandidat Future, bukan keputusan yang perlu diformalkan sebagai ADR sekarang.
- **HashiCorp Vault** — dievaluasi secara eksplisit di ADR-013 dan **ditolak sebagai default** (bukan diabaikan) — alasan penolakannya didokumentasikan di dalam ADR-013 itu sendiri, bukan sebagai ADR terpisah.

## Documentation Index

Peta lengkap dokumentasi arsitektur CodeShop (Phase 1.1):

| Dokumen | Isi |
|---|---|
| `../architecture/REQUIREMENTS.md` | Kebutuhan bisnis & non-fungsional, MVP vs Phase 2 vs Future. |
| `../architecture/ARCHITECTURE.md` | Peta arsitektur utama: system context, service architecture, trust boundary, alur checkout/payment/download, state machine Payment & Order, authorization boundary. |
| `../architecture/SERVICE_CATALOG.md` | Detail tiap service: ownership, API, dependency, event, security, observability. |
| `../architecture/DATABASE_DESIGN.md` | Ownership entity per service, database reliability (backup/replication/migration), ER diagram lintas service. |
| `../architecture/API_DESIGN.md` | Konvensi REST/gRPC publik & internal, endpoint kunci. |
| `../architecture/EVENT_DESIGN.md` | Desain event asynchronous, termasuk koreksi ownership Payment↔Order (§1a). |
| `../architecture/THREAT_MODEL.md` | Threat model ringan berbasis aset (identity, API, payment, download, infrastructure, supply chain). |
| `../architecture/DATA_CLASSIFICATION.md` | Klasifikasi data & kebijakan penanganan per level sensitivitas. |
| `../architecture/CI_CD_DESIGN.md` | Arsitektur pipeline CI/CD, security gate, strategi rollback & deployment. |
| `../architecture/ENVIRONMENT_STRATEGY.md` | Strategi Local/Staging/Production yang pragmatis. |
| `../architecture/KUBERNETES_DESIGN.md` | Target arsitektur Kubernetes produksi. |
| `../architecture/NETWORK_DESIGN.md` | Arsitektur jaringan, trust zone, prinsip least-trust. |
| `../architecture/OBSERVABILITY.md` | Golden signals, SLI/SLO, dashboard, alerting. |
| `../architecture/RELIABILITY.md` | Timeout, retry, circuit breaker, idempotency, graceful degradation. |
| `../architecture/DISASTER_RECOVERY.md` | RPO/RTO per komponen, skenario kegagalan. |
| `../architecture/BACKUP_STRATEGY.md` | Strategi backup per komponen & kebijakan uji restore. |
| `../architecture/INCIDENT_RESPONSE.md` | Severity, lifecycle insiden, contoh skenario. |
| `../architecture/PRODUCTION_READINESS.md` | Checklist kesiapan produksi lintas domain. |
| `../architecture/TRADE_OFFS.md` | Klasifikasi Required/Recommended/Optional/Future per komponen, roadmap skalabilitas. |
| `./README.md` (dokumen ini) | Index ADR. |
| `../diagrams/README.md` | Index seluruh diagram Mermaid dan lokasinya. |

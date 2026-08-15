# Architecture Decision Records — CodeShop

Direktori ini berisi Architecture Decision Records (ADR) untuk project CodeShop. Setiap ADR mendokumentasikan satu keputusan arsitektur penting beserta konteks, alternatif yang dipertimbangkan, dan konsekuensinya.

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
| [ADR-012](./ADR-012-opentelemetry.md) | OpenTelemetry untuk Observability | Accepted |

## ADR yang Tidak Dibuat

Sesuai instruksi untuk tidak membuat keputusan palsu, berikut kandidat keputusan yang **dievaluasi tetapi tidak menghasilkan ADR terpisah**, beserta alasannya:

- **Cart sebagai service terpisah** — tidak menjadi ADR tersendiri karena keputusan untuk *tidak* membuat service terpisah bukan keputusan arsitektur besar yang berdiri sendiri; ini adalah bagian dari evaluasi service boundary yang sudah didokumentasikan secara memadai di `../architecture/ARCHITECTURE.md` §3 dan `../architecture/SERVICE_CATALOG.md`.
- **Pemilihan Payment Provider spesifik (mis. Midtrans/Xendit/Stripe)** — bukan keputusan arsitektur tetapi keputusan vendor/bisnis yang dapat berubah tanpa memengaruhi arsitektur (Payment Service didesain agnostik terhadap provider tertentu). Tidak dibuat ADR karena akan menjadi keputusan palsu jika dipaksakan pada Phase 1 tanpa evaluasi vendor yang sebenarnya.
- **Multi-region / multi-cluster deployment** — di luar scope MVP (lihat `REQUIREMENTS.md` — Non-Goals & Assumptions), sehingga ADR terkait strategi multi-region akan dibuat pada fase ketika kebutuhan itu benar-benar muncul, bukan diputuskan secara spekulatif sekarang.
- **Search engine terpisah (Elasticsearch/Meilisearch, dst.)** — dievaluasi dan ditolak untuk MVP (lihat `ARCHITECTURE.md` §3). Karena hasil evaluasi adalah "tidak dipakai", ini tidak memenuhi kriteria ADR (ADR mendokumentasikan keputusan yang diambil dan berdampak signifikan, bukan setiap opsi yang ditolak); alasan penolakan cukup didokumentasikan di `ARCHITECTURE.md`.

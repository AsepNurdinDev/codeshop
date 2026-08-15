# ADR-001: Microservices Architecture

## Status

Accepted

## Context

CodeShop memiliki beberapa domain bisnis yang berbeda karakteristik: katalog (read-heavy), order/checkout (transactional), payment (integrasi eksternal, kepatuhan keamanan tinggi), download (keamanan akses file), dan notifikasi (asynchronous, non-kritikal). Domain-domain ini memiliki kebutuhan scaling, tingkat kekritisan, dan siklus perubahan yang berbeda. Tim menargetkan teknologi cloud-native (Kubernetes, gRPC, event-driven) dan arah karier/eksplorasi DevOps yang relevan dengan pola microservices.

## Decision

CodeShop dibangun sebagai **microservices**, dengan boundary awal: API Gateway, Auth Service, Catalog Service, Order Service, Payment Service, Download Service, Notification Service — sebagaimana didokumentasikan di `../architecture/SERVICE_CATALOG.md`.

## Alternatives Considered

1. **Monolith modular (modular monolith)** — satu deployable unit dengan modul terpisah secara internal. Lebih sederhana untuk dioperasikan pada tahap awal, mengurangi overhead network call. Ditolak karena target teknologi dan arah tim secara eksplisit condong ke microservices/Kubernetes, dan domain payment/download memiliki kebutuhan isolasi keamanan yang lebih natural dipisah sebagai service dengan boundary jelas.
2. **Microservices granular berlebih** (memecah setiap entity menjadi service, mis. Cart Service, ProductVersion Service terpisah) — ditolak karena over-engineering untuk skala tim kecil (lihat evaluasi di `ARCHITECTURE.md` §3).
3. **Microservices dengan boundary yang diusulkan (baseline 7 service)** — **dipilih**, seimbang antara isolasi domain dan kompleksitas operasional yang masih dapat dikelola tim kecil.

## Consequences

**Positif**: isolasi kegagalan per domain, scaling independen, kemungkinan pengembangan paralel, selaras dengan target teknologi.

**Negatif**: kompleksitas operasional lebih tinggi dibanding monolith (perlu service discovery, observability lintas service, deployment orchestration); latensi tambahan dari network call antar service; kebutuhan disiplin tim untuk menjaga service boundary agar tidak terjadi tight coupling implisit.

## Security Implications

Isolasi service memungkinkan penerapan least-privilege per service (mis. hanya Download Service yang punya akses ke object storage credential), memperkecil blast radius jika satu service disusupi.

## Operational Implications

Memerlukan observability lintas service (distributed tracing, correlation ID) sejak awal — lihat ADR-012. Memerlukan orkestrasi deployment (Kubernetes) — lihat ADR-010, dan strategi GitOps — lihat ADR-011.

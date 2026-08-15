# ADR-010: Kubernetes sebagai Target Deployment Platform

## Status

Accepted

## Context

Dengan arsitektur microservices (ADR-001), dibutuhkan platform orkestrasi yang dapat mengelola banyak service, scaling independen, health check, dan rolling deployment secara konsisten. Arah teknologi target juga mencakup Helm, Argo CD, dan observability stack (Prometheus, Grafana, Loki, Tempo) yang secara ekosistem terintegrasi erat dengan Kubernetes.

## Decision

CodeShop menargetkan **Kubernetes** sebagai platform deployment untuk seluruh service (lihat gambaran arsitektur target di `../architecture/ARCHITECTURE.md` §18).

## Alternatives Considered

1. **Docker Compose / single VM deployment** — lebih sederhana untuk operasional awal dan cocok untuk MVP tahap sangat dini, tetapi ditolak sebagai target arsitektur karena tidak menyediakan self-healing, auto-scaling, dan rolling update yang konsisten dengan kebutuhan microservices jangka menengah–panjang. Dicatat bahwa Docker Compose tetap relevan sebagai lingkungan pengembangan lokal (development environment), bukan target produksi.
2. **PaaS terkelola (mis. Cloud Run, App Platform, Heroku-like)** — mengurangi beban operasional, tetapi ditolak sebagai target utama karena kurang fleksibel untuk kebutuhan observability stack custom (Prometheus/Grafana/Loki/Tempo) dan tujuan eksplorasi/pembelajaran Kubernetes yang menjadi arah tim.
3. **Kubernetes (self-managed atau managed, mis. GKE/EKS-equivalent)** — **dipilih**, selaras dengan seluruh target teknologi lain (Helm, Argo CD, observability stack) dan kebutuhan orkestrasi microservices jangka panjang.

## Consequences

**Positif**: orkestrasi matang (self-healing, rolling update, horizontal pod autoscaling), ekosistem tooling luas, selaras dengan seluruh target teknologi lain.

**Negatif**: kompleksitas operasional signifikan, terutama untuk tim kecil — memerlukan investasi belajar/waktu operasional yang besar; risiko over-provisioning infrastruktur relatif terhadap skala trafik MVP awal jika tidak dikelola hati-hati (mitigasi: mulai dengan cluster kecil/managed Kubernetes, bukan self-managed multi-node dari hari pertama).

## Security Implications

Kubernetes menyediakan primitif keamanan native yang relevan dengan trust boundary di `../architecture/ARCHITECTURE.md` §4–§6: NetworkPolicy (isolasi jaringan antar namespace/service), RBAC Kubernetes (kontrol akses ke resource cluster), Pod Security (pembatasan privilege container), dan Secrets management (kredensial database, JWT signing key, dsb.) — detail konfigurasi merupakan Phase implementasi.

## Operational Implications

Memerlukan Helm chart per service dan strategi GitOps (lihat ADR-011) untuk deployment yang konsisten dan dapat diaudit; memerlukan observability stack terintegrasi (lihat ADR-012).

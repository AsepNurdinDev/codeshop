# Kubernetes Production Architecture — CodeShop

Dokumen ini mendeskripsikan target desain Kubernetes secara konseptual. **Tidak ada file YAML/Helm chart yang dibuat pada Phase 1.1** — ini adalah spesifikasi untuk Phase 6 (Kubernetes) dan Phase 10 (Production) implementasi.

## 1. Namespaces

| Namespace | Isi |
|---|---|
| `ingress` | Ingress controller. |
| `app` | Seluruh service aplikasi (API Gateway, Auth, Catalog, Order, Payment, Download, Notification). |
| `data` | PostgreSQL (jika self-hosted di cluster), Redis, RabbitMQ. |
| `observability` | Prometheus, Grafana, Loki, Tempo, Alertmanager. |
| `argocd` | Argo CD control plane. |

Pemisahan namespace memungkinkan penerapan NetworkPolicy dan RBAC granular per domain fungsional (lihat §7, §8).

## 2. Ingress

- Satu Ingress controller (namespace `ingress`) menerima seluruh trafik dari Cloudflare/Load Balancer eksternal.
- Rule Ingress hanya mengarahkan ke **API Gateway** — tidak ada service backend lain yang diekspos langsung via Ingress (konsisten dengan trust boundary di `ARCHITECTURE.md` §4).
- TLS termination dapat terjadi di Cloudflare (edge) dan/atau di Ingress (defense in depth) — detail konfigurasi Phase implementasi.

## 3. Services & Deployments

- Setiap microservice memiliki satu `Deployment` + satu `Service` (ClusterIP, internal) di namespace `app`.
- Tidak ada service yang menggunakan `NodePort`/`LoadBalancer` individual — hanya API Gateway yang menjadi entry point via Ingress.

## 4. Resource Requests & Limits

- Setiap container mendeklarasikan `requests` (jaminan minimum) dan `limits` (batas maksimum) untuk CPU dan memory.
- Nilai awal ditentukan berdasarkan baseline load testing per service (Phase implementasi), bukan angka sembarangan — didokumentasikan sebagai kebutuhan wajib, bukan angka final di Phase 1.1.
- Prinsip: `requests` cukup untuk operasi normal, `limits` mencegah satu pod "noisy neighbor" menghabiskan resource node.

## 5. Autoscaling — HPA

- **Horizontal Pod Autoscaler** diterapkan pada service dengan trafik variabel: API Gateway, Catalog Service, Download Service (baca `ARCHITECTURE.md` §14 Scalability).
- Auth Service dan Order Service di-scale berdasarkan CPU/latency threshold moderat mengingat sifatnya yang lebih write-heavy dan transactional.
- Notification Service di-scale berdasarkan queue consumer lag (custom metric via KEDA atau Prometheus adapter — dicatat sebagai opsi Phase implementasi, bukan keputusan final).

## 6. Availability & Resilience Primitives

- **Readiness probe**: menandakan pod siap menerima trafik (mis. koneksi database berhasil) — dipisahkan dari liveness agar pod yang sedang startup tidak menerima trafik prematur.
- **Liveness probe**: mendeteksi pod yang deadlock/hang dan perlu di-restart.
- **Startup probe**: untuk service dengan waktu inisialisasi lebih lama (migrasi/warming cache), mencegah liveness probe membunuh pod yang masih startup normal.
- **PodDisruptionBudget (PDB)**: memastikan minimum jumlah replica tetap berjalan selama voluntary disruption (node drain, upgrade cluster).
- **Anti-affinity**: replica dari service yang sama disebar ke node berbeda (`podAntiAffinity`) agar kegagalan satu node tidak menjatuhkan seluruh replica satu service sekaligus.
- **Topology considerations**: jika cluster multi-zone (Phase implementasi cloud), replica disebar lintas zone (topology spread constraints) untuk ketahanan terhadap kegagalan satu zone.
- **Graceful shutdown**: setiap service menangani `SIGTERM` untuk menyelesaikan request in-flight dan menutup koneksi (database, RabbitMQ) dengan bersih sebelum pod dihentikan, dengan `terminationGracePeriodSeconds` yang memadai — penting khususnya untuk Payment Service (webhook in-flight) dan Download Service (signed URL generation in-flight).

## 7. Networking — NetworkPolicy

- Default-deny sebagai baseline: pod tidak dapat berkomunikasi dengan pod lain kecuali diizinkan eksplisit.
- Contoh kebijakan (konseptual):
  - `app` namespace hanya dapat menerima trafik masuk dari `ingress` namespace (untuk API Gateway) dan dari pod lain di `app` namespace sesuai dependency matrix (`SERVICE_CATALOG.md`).
  - `data` namespace hanya menerima trafik dari service `app` yang menjadi pemilik data tersebut (mis. hanya Auth Service yang dapat menghubungi Auth DB).
  - `observability` namespace dapat melakukan scrape (masuk) dari seluruh namespace `app`/`data` untuk metrics, tetapi tidak sebaliknya.

## 8. RBAC & ServiceAccount

- Setiap service memiliki `ServiceAccount` sendiri dengan permission minimal yang dibutuhkan (least privilege) — sebagian besar service aplikasi tidak memerlukan permission terhadap Kubernetes API sama sekali.
- Argo CD memiliki `ServiceAccount`/RBAC khusus dengan permission deploy ke namespace `app`/`data`/`observability`, tetapi **tidak** memiliki akses ke Kubernetes Secrets di luar yang didefinisikan di GitOps repo.
- Operator manusia mengakses cluster melalui RBAC berbasis identitas (bukan shared credential), dibatasi sesuai kebutuhan (mis. read-only untuk sebagian besar engineer, write terbatas untuk on-call).

## 9. Pod Security

- Non-root container (`runAsNonRoot: true`).
- Read-only root filesystem di mana memungkinkan (`readOnlyRootFilesystem: true`), dengan `emptyDir` untuk direktori yang benar-benar butuh tulis (mis. temp file).
- Dropped Linux capabilities (`drop: ["ALL"]`, tambahkan hanya yang benar-benar diperlukan).
- Pod Security Standards (`restricted` profile) diterapkan sebagai baseline di namespace `app`.

## 10. Secrets & Config

- Kubernetes Secrets digunakan sebagai mekanisme penyimpanan runtime, tetapi **sumber kebenaran** secret bukan Git plaintext — lihat ADR-013 untuk strategi pengelolaan penuh.
- ConfigMap untuk konfigurasi non-sensitif (feature flag, nilai timeout, dsb.), terpisah dari Secret.

## 11. Persistent Storage

- Database (jika self-hosted PostgreSQL di cluster) menggunakan `PersistentVolumeClaim` dengan storage class yang mendukung backup snapshot (lihat `BACKUP_STRATEGY.md`).
- Object storage (MinIO, jika self-hosted) juga menggunakan persistent storage terpisah dari database.
- Alternatif: PostgreSQL/object storage sebagai **managed service** di luar cluster (mengurangi kompleksitas operasional stateful-in-Kubernetes) — dipertimbangkan sebagai opsi yang setara, dipilih berdasarkan biaya/kapabilitas tim saat implementasi (lihat `TRADE_OFFS.md`).

## 12. Node Pools & Autoscaling (Cluster-Level)

- Baseline awal: satu node pool umum (general-purpose) — tidak perlu node pool terpisah untuk setiap jenis beban di tahap awal (lihat `SCALABILITY` roadmap terkait di `TRADE_OFFS.md`).
- Cluster autoscaler dipertimbangkan setelah pola beban trafik nyata teramati, bukan diaktifkan spekulatif sejak hari pertama produksi.

## 13. What This Phase Does Not Cover

Nilai konkret (jumlah replica minimum/maksimum, angka resource request/limit final, nama node pool aktual) ditentukan pada Phase implementasi berdasarkan data nyata (load test, biaya infrastruktur), bukan diputuskan spekulatif di sini.

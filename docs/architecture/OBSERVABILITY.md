# Observability Architecture — CodeShop

## 1. Pillars

| Pillar | Tooling |
|---|---|
| Metrics | Prometheus |
| Logs | Loki |
| Traces | OpenTelemetry (instrumentasi) + Tempo (backend) |
| Visualization | Grafana |
| Alerting | Alertmanager |

Instrumentasi aplikasi menggunakan standar **OpenTelemetry** (ADR-012) agar tidak terkunci pada backend tertentu — Prometheus/Loki/Tempo adalah pilihan backend saat ini, dapat diganti tanpa mengubah instrumentasi.

## 2. Golden Signals

Setiap service (khususnya yang menerima trafik sinkron: API Gateway, Auth, Catalog, Order, Payment, Download) mengukur:

1. **Latency** — p50/p95/p99 per endpoint/method.
2. **Traffic** — request rate per endpoint/method.
3. **Errors** — error rate per endpoint/method, dipecah per kode error.
4. **Saturation** — pemakaian resource (CPU/memory) relatif terhadap limit, panjang antrian (RabbitMQ) untuk service asynchronous.

## 3. Service-Level Indicators (SLI) & Objectives (SLO)

Target awal yang **realistis untuk baseline produksi awal** (bukan hyperscale), akan direvisi berdasarkan data nyata setelah beberapa bulan operasi:

| Service/Alur | SLI | SLO Awal |
|---|---|---|
| API Gateway (keseluruhan) | Availability (request sukses / total request non-4xx) | 99.5% per bulan |
| Catalog Service (browsing) | p95 latency | < 300ms |
| Checkout flow | p95 latency | < 800ms (di luar latensi payment provider eksternal) |
| Download Service (signed URL issuance) | p95 latency | < 500ms |
| Payment webhook processing | p95 latency dari terima → publish event | < 2s |
| `PaymentSucceeded` → `OrderPaid` consistency window | p95 lag | < 5s |
| RabbitMQ consumer lag (Notification, Download) | p95 lag | < 10s |

SLO ini digunakan untuk menentukan ambang alert (§6), bukan sekadar dokumentasi pasif.

## 4. Business Metrics

Selain metrics teknis, didefinisikan metrics bisnis (lihat juga masing-masing entri `SERVICE_CATALOG.md`):

- Order created rate, checkout success/fail rate, cart abandonment rate.
- Payment success/fail rate, webhook duplicate/rejected rate.
- Download issuance rate, download abuse signal (rate limit hit rate).
- Notification delivery success/fail rate.

Metrics bisnis ditampilkan di dashboard terpisah dari dashboard teknis agar dapat dikonsumsi oleh pemilik produk/bisnis, bukan hanya engineer.

## 5. Dashboards (Target)

- **Service Health Overview** — golden signals seluruh service dalam satu pandangan.
- **Checkout & Payment Funnel** — visualisasi alur cart → checkout → payment → paid → entitlement, dengan drop-off rate di setiap tahap.
- **Download Security** — signed URL issuance rate, abuse signal, entitlement revocation rate.
- **Infrastructure** — Kubernetes pod health, resource utilization, RabbitMQ queue depth/DLQ size, database connection pool utilization.

## 6. Alerting

Alert dikategorikan berdasarkan urgensi (selaras dengan `INCIDENT_RESPONSE.md` §Severity):

| Kondisi | Severity | Contoh Alert |
|---|---|---|
| Error rate API Gateway melampaui ambang SLO | Critical | `error_rate > 5% selama 5 menit` |
| Payment webhook processing latency melampaui SLO | High | `p95 webhook latency > 2s selama 10 menit` |
| `PaymentSucceeded → OrderPaid` lag abnormal | High | `p95 lag > 30s` (indikasi Order Service atau RabbitMQ bermasalah) |
| DLQ size bertambah terus-menerus | High | `dlq_size meningkat > N pesan dalam 15 menit` |
| Signed URL issuance rate melonjak tajam per user | Medium | Indikasi abuse, lihat `THREAT_MODEL.md` §Download |
| Webhook dengan signature invalid melonjak | High | Indikasi serangan fake webhook, lihat `THREAT_MODEL.md` §Payment |
| Database connection pool saturasi | High | `pool utilization > 90%` |
| Pod restart berulang (CrashLoopBackOff) | Critical | Restart count meningkat dalam window singkat |

Alert Critical/High diarahkan ke on-call (mekanisme notifikasi konkret — misal integrasi ke messaging tool — adalah detail Phase implementasi).

## 7. Log Correlation

- Setiap log terstruktur menyertakan `request_id` dan `correlation_id` (lihat `API_DESIGN.md` §10–11), memungkinkan pencarian seluruh log lintas service untuk satu alur bisnis di Loki.
- Log tidak pernah menyertakan data **Highly Sensitive** (password hash, token mentah, data pembayaran mentah) — lihat `DATA_CLASSIFICATION.md` §4.

## 8. Trace Correlation

- `correlation_id` dipropagasikan melalui gRPC metadata dan event envelope (`EVENT_DESIGN.md` §6), dan diselaraskan dengan `trace_id` OpenTelemetry sehingga satu alur bisnis (checkout → payment → entitlement → download) dapat ditelusuri end-to-end di Tempo, termasuk melintasi batas asynchronous (message broker) — trace context disertakan dalam event envelope, bukan hanya pada call gRPC synchronous.

## 9. Request ID vs Correlation ID

- `request_id`: unik per HTTP/gRPC request tunggal, untuk debugging satu panggilan spesifik.
- `correlation_id`: konsisten di seluruh rangkaian request/event yang termasuk dalam satu alur bisnis (satu checkout, satu proses download), untuk debugging end-to-end.

## 10. What This Phase Does Not Cover

Query PromQL/LogQL konkret, definisi dashboard JSON, dan konfigurasi Alertmanager route tidak dibuat pada Phase 1.1 — ini adalah spesifikasi kebutuhan observability, implementasi teknis adalah Phase 8.

# ADR-012: OpenTelemetry untuk Observability

## Status

Accepted

## Context

Arsitektur microservices (ADR-001) menyebarkan logika bisnis ke banyak service yang saling berkomunikasi secara synchronous (gRPC) dan asynchronous (RabbitMQ). Tanpa observability yang terkorelasi lintas service, troubleshooting alur seperti checkout → payment → entitlement → download akan sangat sulit. Target teknologi mencakup Prometheus, Grafana, Loki, Tempo.

## Decision

CodeShop mengadopsi **OpenTelemetry (OTel)** sebagai standar instrumentasi untuk metrics, logs, dan traces di seluruh service, dengan backend: Prometheus (metrics), Loki (logs), Tempo (traces), divisualisasikan melalui Grafana, dan alerting melalui Alertmanager.

## Alternatives Considered

1. **Instrumentasi custom per service tanpa standar terpadu** (masing-masing service memilih library sendiri) — ditolak karena akan menyulitkan korelasi data observability lintas service (format log berbeda, tidak ada trace context propagation yang konsisten).
2. **Vendor-specific APM (mis. Datadog, New Relic)** — menawarkan kemudahan setup dan UI matang, tetapi ditolak sebagai target utama karena menimbulkan vendor lock-in dan biaya berulang yang tidak proporsional untuk skala tim kecil/indie; juga tidak selaras dengan preferensi self-hosted/open-source pada target teknologi (Prometheus/Grafana/Loki/Tempo).
3. **OpenTelemetry + Prometheus/Grafana/Loki/Tempo (self-hosted, open-source)** — **dipilih**, standar vendor-neutral untuk instrumentasi, dapat diekspor ke berbagai backend, selaras dengan target teknologi dan menghindari lock-in.

## Consequences

**Positif**: satu standar instrumentasi untuk metrics/logs/traces di seluruh service; correlation ID/trace ID dapat dipropagasikan secara konsisten lintas gRPC dan event RabbitMQ (lihat `../architecture/EVENT_DESIGN.md` §6); fleksibel mengganti backend observability di masa depan tanpa mengubah instrumentasi aplikasi.

**Negatif**: menambah beban operasional mengelola stack observability sendiri (Prometheus, Grafana, Loki, Tempo, Alertmanager) dibanding solusi SaaS terkelola; memerlukan disiplin instrumentasi konsisten di setiap service baru yang dibuat.

## Security Implications

Log dan trace berpotensi memuat data sensitif jika tidak hati-hati (mis. payload request yang mengandung PII) — memerlukan kebijakan redaction/masking data sensitif dalam instrumentasi (mis. tidak logging `password`, `password_hash`, data pembayaran mentah) sebagai prinsip wajib, detail teknis di Phase implementasi.

## Operational Implications

Setiap service wajib mengekspos metrics dasar (request rate, error rate, latency), structured logs dengan correlation ID, dan trace span untuk operasi lintas service — kebutuhan ini didokumentasikan per service di `../architecture/SERVICE_CATALOG.md` dan menjadi bagian dari definition of done pada Phase implementasi.

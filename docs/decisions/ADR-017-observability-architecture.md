# ADR-017: Observability Architecture

## Status

Accepted

## Context

Arsitektur microservices menyebarkan logika bisnis lintas banyak service yang berkomunikasi synchronous (gRPC) dan asynchronous (RabbitMQ). Koreksi ownership Phase 1.1 (`EVENT_DESIGN.md` §1a) juga memperkenalkan window eventual consistency (`PaymentSucceeded → OrderPaid`) yang perlu dapat diukur dan dipantau, bukan menjadi "black box". ADR-012 sudah menetapkan OpenTelemetry sebagai standar instrumentasi; ADR ini menetapkan **arsitektur observability secara menyeluruh** (SLI/SLO, dashboard, alerting) sebagai lapisan di atasnya.

## Decision

CodeShop mengadopsi observability berbasis **tiga pilar (metrics/logs/traces) + SLI/SLO eksplisit + alerting berbasis severity**, didokumentasikan penuh di `OBSERVABILITY.md`, dengan prinsip: setiap SLO memiliki alert yang berkorespondensi (bukan SLO yang hanya didokumentasikan tanpa penegakan).

## Alternatives Considered

1. **Monitoring reaktif tanpa SLO eksplisit (hanya dashboard, alert ad-hoc)** — ditolak, tidak memberi kriteria objektif untuk menentukan kapan sebuah kondisi benar-benar butuh tindakan vs. noise, berisiko menyebabkan alert fatigue atau sebaliknya insiden nyata tidak terdeteksi tepat waktu.
2. **SLO sangat ketat sejak awal (mendekati 99.99%)** — ditolak, tidak realistis untuk baseline produksi awal dengan infrastruktur single-region tanpa redundansi penuh (lihat `DISASTER_RECOVERY.md` §1 Scope & Realism); SLO yang tidak realistis akan terus-menerus dilanggar dan kehilangan makna sebagai sinyal.
3. **SLI/SLO moderat berbasis golden signals, direvisi berkala berdasarkan data nyata** — **dipilih**, memberi target yang dapat dicapai sekaligus bermakna sebagai sinyal kesehatan sistem (`OBSERVABILITY.md` §3).

## Consequences

**Positif**: tim memiliki kriteria objektif untuk "apakah sistem sehat" yang selaras dengan kebutuhan bisnis (checkout, payment, download sebagai alur prioritas); insiden dapat dideteksi dan dieskalasi secara konsisten (`INCIDENT_RESPONSE.md`).

**Negatif**: memerlukan investasi instrumentasi di setiap service (menambah kode boilerplate observability) dan disiplin untuk menjaga SLO tetap relevan (direvisi, bukan dibiarkan usang) — dicatat sebagai tanggung jawab berkelanjutan, bukan kerja satu kali.

## Security Implications

Observability, jika tidak dijaga, dapat menjadi jalur kebocoran data sensitif melalui log/trace (lihat `DATA_CLASSIFICATION.md` §4) — instrumentasi wajib memastikan data Sensitive/Highly Sensitive tidak masuk ke span attribute atau log message secara sembarangan.

## Operational Implications

Dashboard dan alert yang didefinisikan di `OBSERVABILITY.md` §5–§6 menjadi bagian dari `PRODUCTION_READINESS.md` checklist — sistem tidak dianggap siap produksi tanpa observability dasar aktif dan alerting yang menjangkau on-call.

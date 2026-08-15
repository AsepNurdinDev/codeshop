# ADR-005: RabbitMQ untuk Asynchronous Messaging

## Status

Accepted

## Context

Beberapa alur bisnis bersifat asynchronous dan tidak memerlukan response langsung dari consumer (mis. pengiriman notifikasi setelah order dibuat). Dibutuhkan message broker yang reliable, mendukung retry dan dead-letter queue, dan proporsional dengan skala tim/proyek (bukan platform streaming skala besar).

## Decision

CodeShop menggunakan **RabbitMQ** sebagai message broker untuk komunikasi asynchronous antar service, dengan pola event sebagaimana didokumentasikan di `../architecture/EVENT_DESIGN.md`.

## Alternatives Considered

1. **Apache Kafka** — unggul untuk throughput sangat tinggi dan event streaming/replay jangka panjang, tetapi ditolak untuk MVP karena kompleksitas operasional (Zookeeper/KRaft, partitioning, tuning) tidak proporsional dengan volume event CodeShop di tahap awal — berpotensi over-engineering.
2. **Redis Pub/Sub** — ditolak karena tidak menjamin delivery (tidak ada persistence/DLQ built-in yang memadai untuk kebutuhan reliability event bisnis seperti `OrderPaid`).
3. **RabbitMQ** — **dipilih**, mendukung reliable delivery, routing fleksibel (exchange/queue), dead-letter queue native, dan operasional lebih ringan dibanding Kafka untuk skala tim kecil–menengah.

## Consequences

**Positif**: decoupling antar service, kemampuan retry & DLQ native, cukup matang dan banyak didukung tooling/observability.

**Negatif**: tidak sekuat Kafka untuk kebutuhan event replay jangka panjang atau throughput sangat tinggi — jika kebutuhan itu muncul di masa depan (Future Scope), migrasi/adopsi tambahan perlu dievaluasi ulang sebagai ADR baru saat itu terjadi.

## Security Implications

Akses ke RabbitMQ dibatasi hanya untuk service internal (tidak ada expose publik), menggunakan kredensial per service (least privilege terhadap exchange/queue tertentu) — detail konfigurasi di Phase implementasi.

## Operational Implications

Memerlukan monitoring queue depth, consumer lag, dan DLQ size (lihat `../architecture/ARCHITECTURE.md` §16 dan `EVENT_DESIGN.md` §10) sebagai bagian dari observability wajib.

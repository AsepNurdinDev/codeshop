# ADR-003: gRPC untuk Komunikasi Internal

## Status

Accepted

## Context

Service internal (Order ↔ Catalog, Payment ↔ Order, Download ↔ Auth/Order) memerlukan komunikasi synchronous berlatensi rendah dengan kontrak yang jelas dan strongly-typed, mengingat implementasi menggunakan Go di seluruh service.

## Decision

Komunikasi internal (service-to-service) menggunakan **gRPC**. Komunikasi publik (Client ↔ API Gateway) tetap menggunakan REST/HTTP (lihat ADR-004).

## Alternatives Considered

1. **REST/HTTP+JSON untuk komunikasi internal** — lebih sederhana untuk debugging manual, tetapi ditolak karena overhead serialisasi JSON lebih tinggi, kontrak API kurang strict (tidak ada schema enforcement built-in) dibanding Protocol Buffers.
2. **gRPC** — **dipilih**: strongly-typed contract via `.proto`, performa lebih baik (HTTP/2, binary serialization), dukungan streaming bila dibutuhkan di masa depan, cocok untuk komunikasi internal antar service Go.
3. **GraphQL internal** — ditolak, tidak memberikan manfaat signifikan untuk pola request-response service-to-service yang relatif sederhana, dan menambah kompleksitas yang tidak diperlukan.

## Consequences

**Positif**: kontrak API service jelas dan dapat divalidasi saat compile-time (via generated code, Phase implementasi); performa komunikasi internal lebih baik dibanding REST/JSON.

**Negatif**: kurva belajar tambahan (protobuf, gRPC tooling); debugging manual (mis. via curl) lebih sulit dibanding REST — memerlukan tooling tambahan (mis. grpcurl) pada Phase implementasi.

## Security Implications

gRPC internal direncanakan berjalan di atas mTLS dalam cluster (Phase implementasi/infrastruktur) untuk memastikan komunikasi service-to-service terenkripsi dan saling terautentikasi, selaras dengan trust boundary di `../architecture/ARCHITECTURE.md` §4.

## Operational Implications

Memerlukan service mesh atau konfigurasi mTLS manual di Kubernetes (keputusan detail infrastruktur ada di Phase implementasi, bukan Phase 1). Observability gRPC (tracing, metrics) perlu diinstrumentasi melalui OpenTelemetry (ADR-012).

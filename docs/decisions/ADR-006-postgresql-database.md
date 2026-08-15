# ADR-006: PostgreSQL sebagai Database Utama

## Status

Accepted

## Context

Sebagian besar data CodeShop (User, Product, Order, Payment, dsb.) bersifat relasional dengan kebutuhan konsistensi transaksional yang kuat (mis. status order, referensi pembayaran) sebagaimana didokumentasikan di `../architecture/DATABASE_DESIGN.md`.

## Decision

Setiap service (kecuali yang murni stateless) menggunakan **PostgreSQL** sebagai database utamanya, mengikuti prinsip database-per-service (ADR-002).

## Alternatives Considered

1. **NoSQL document store (mis. MongoDB)** — fleksibel untuk skema yang berubah cepat, tetapi ditolak sebagai database utama karena mayoritas data CodeShop bersifat relasional dengan kebutuhan constraint dan transaksi ACID yang kuat (mis. status order, unique constraint pada payment reference).
2. **MySQL** — alternatif relasional yang valid, tetapi PostgreSQL dipilih karena fitur yang lebih kaya untuk kebutuhan potensial (JSONB untuk data semi-terstruktur seperti metadata produk, indexing lebih fleksibel) dan familiaritas ekosistem Go yang kuat.
3. **PostgreSQL** — **dipilih**, konsistensi ACID kuat, fitur matang, dukungan ekosistem Go yang baik.

## Consequences

**Positif**: konsistensi transaksional kuat untuk data kritikal (order, payment), fleksibilitas tambahan via JSONB bila diperlukan untuk data semi-terstruktur, ekosistem tooling matang.

**Negatif**: scaling horizontal PostgreSQL secara native lebih kompleks dibanding beberapa NoSQL store (memerlukan strategi read replica/partitioning bila volume tumbuh besar — dipertimbangkan sebagai Phase 2/Future, bukan kebutuhan MVP).

## Security Implications

Data sensitif (kredensial, dsb.) mengikuti kebutuhan enkripsi at-rest sesuai kebijakan infrastruktur (Phase implementasi); akses database dibatasi per service credential (least privilege), selaras dengan ADR-002.

## Operational Implications

Memerlukan strategi backup terjadwal dan pengujian restore (lihat `../architecture/ARCHITECTURE.md` §17 Disaster Recovery Concept) untuk setiap instance PostgreSQL per service.

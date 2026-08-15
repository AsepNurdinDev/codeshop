# ADR-007: Redis untuk Cache & Session

## Status

Accepted

## Context

Beberapa kebutuhan bersifat latency-sensitive dan tidak memerlukan durability penuh layaknya data transaksional: cache katalog produk (read-heavy), session/refresh token, dan counter rate limiting (lihat `../architecture/ARCHITECTURE.md` §13 dan `SERVICE_CATALOG.md`).

## Decision

CodeShop menggunakan **Redis** sebagai in-memory data store untuk: (1) cache listing/detail produk di Catalog Service, (2) refresh token session di Auth Service, (3) rate limiting counter di API Gateway/Auth Service.

## Alternatives Considered

1. **Menggunakan PostgreSQL saja untuk session/cache** — ditolak karena latensi lebih tinggi untuk operasi read-heavy berulang (cache) dan operasi counter frekuensi tinggi (rate limiting) dibanding in-memory store.
2. **Memcached** — alternatif in-memory yang valid untuk cache murni, tetapi ditolak karena tidak mendukung struktur data yang dibutuhkan untuk rate limiting (mis. sliding window counter) dan session TTL selengkap Redis.
3. **Redis** — **dipilih**, mendukung TTL native, struktur data fleksibel (string, hash, sorted set untuk rate limiting), dan matang digunakan lintas kebutuhan (cache, session, counter) tanpa menambah lebih dari satu jenis infrastruktur baru.

## Consequences

**Positif**: latensi rendah untuk cache dan session; mengurangi beban baca ke PostgreSQL untuk data yang sering diakses (katalog); mendukung revocation refresh token secara langsung.

**Negatif**: data di Redis bersifat non-durable secara default (tergantung konfigurasi persistence) — Redis **tidak** digunakan sebagai source of truth untuk data kritikal (semua data kritikal tetap di PostgreSQL); kehilangan data Redis (mis. restart tanpa persistence) hanya berdampak pada cache miss atau perlu login ulang, bukan kehilangan data bisnis.

## Security Implications

Refresh token yang disimpan di Redis harus dapat di-revoke sewaktu-waktu (mis. saat logout/reset password); akses Redis dibatasi hanya untuk service yang membutuhkan (Auth Service, Catalog Service, API Gateway) melalui network policy.

## Operational Implications

Memerlukan monitoring cache hit rate dan memory usage (lihat `../architecture/ARCHITECTURE.md` §16); kebijakan eviction (mis. LRU) perlu dikonfigurasi sesuai kapasitas pada Phase implementasi.

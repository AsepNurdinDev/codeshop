# ADR-004: REST untuk Public API

## Status

Accepted

## Context

API publik CodeShop dikonsumsi oleh web client (browser) dan berpotensi konsumen pihak ketiga di masa depan (mis. integrasi partner). Dibutuhkan API yang mudah dikonsumsi secara luas, mudah di-cache oleh CDN/browser, dan familiar bagi kebanyakan frontend developer.

## Decision

API publik (Client ↔ API Gateway) menggunakan **REST/HTTP dengan JSON**, mengikuti konvensi yang didokumentasikan di `../architecture/API_DESIGN.md`.

## Alternatives Considered

1. **GraphQL** — memberi fleksibilitas query bagi client, tetapi ditolak untuk MVP karena menambah kompleksitas operasional (resolver, N+1 query concern) yang tidak proporsional dengan kebutuhan MVP yang relatif terprediksi (halaman katalog, cart, checkout, download).
2. **gRPC-Web untuk publik** — ditolak karena kurang familiar bagi ekosistem frontend umum, tooling debugging lebih terbatas dibanding REST, dan caching HTTP standar (CDN) lebih sulit diterapkan.
3. **REST/HTTP+JSON** — **dipilih**, paling matang untuk kebutuhan public API, mudah di-cache, mudah didokumentasikan (OpenAPI di Phase implementasi), dan familiar bagi tim.

## Consequences

**Positif**: mudah dikonsumsi oleh berbagai jenis client, memanfaatkan caching HTTP/CDN standar (Cloudflare), tooling debugging luas.

**Negatif**: over-fetching/under-fetching data mungkin terjadi dibanding GraphQL (mitigasi: desain endpoint yang cukup granular sesuai kebutuhan UI, lihat `API_DESIGN.md`).

## Security Implications

REST publik menjadi permukaan serangan utama (public attack surface) — memerlukan rate limiting, WAF (Cloudflare), dan validasi input ketat di API Gateway serta di masing-masing service (defense in depth), lihat `../architecture/ARCHITECTURE.md` §5.

## Operational Implications

API Gateway bertanggung jawab menerjemahkan REST publik ke gRPC internal (lihat ADR-003), menambah satu lapisan translasi yang perlu diobservasi (latency overhead, error mapping) sebagaimana didokumentasikan di `SERVICE_CATALOG.md` §1.

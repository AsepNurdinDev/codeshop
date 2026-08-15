# Diagrams — CodeShop

Seluruh diagram arsitektur CodeShop dibuat menggunakan **Mermaid** dan disematkan langsung di dalam dokumen Markdown terkait (tidak ada file gambar biner terpisah), sesuai instruksi Phase 1.

## Index Diagram

| # | Diagram | Lokasi |
|---|---|---|
| 1 | System Context | `../architecture/ARCHITECTURE.md` §1 |
| 2 | High-Level Architecture | `../architecture/ARCHITECTURE.md` §2 |
| 3 | Service Communication (Trust Boundaries) | `../architecture/ARCHITECTURE.md` §4 |
| 4 | Checkout Flow | `../architecture/ARCHITECTURE.md` §8 |
| 5 | Payment Flow | `../architecture/ARCHITECTURE.md` §9 |
| 6 | Download Authorization Flow | `../architecture/ARCHITECTURE.md` §10 |
| 7 | Event Flow | `../architecture/EVENT_DESIGN.md` §13 |
| 8 | Database Ownership (Entity Relationship Overview) | `../architecture/DATABASE_DESIGN.md` §9 |
| 9 | Security / Trust Boundaries | `../architecture/ARCHITECTURE.md` §4 |
| 10 | Target Kubernetes Architecture | `../architecture/ARCHITECTURE.md` §18 |

> Catatan: Diagram #3 (Service Communication) dan #9 (Security / Trust Boundaries) merujuk ke diagram trust boundary yang sama di `ARCHITECTURE.md` §4, karena keduanya secara substansi menggambarkan hal yang sama (batas komunikasi antar service = batas keamanan/trust). Duplikasi diagram dengan isi sama di dua tempat berbeda dihindari agar dokumentasi tetap konsisten dan mudah dipelihara — mengikuti prinsip "single source of truth" per diagram. Autentikasi flow (bukan bagian dari 10 diagram wajib namun relevan) juga tersedia di `../architecture/ARCHITECTURE.md` §11.

## Konvensi Diagram

- Semua diagram menggunakan sintaks Mermaid standar (`graph`, `sequenceDiagram`, `erDiagram`, `C4Context`) yang didukung oleh kebanyakan renderer Markdown modern (GitHub, GitLab, dsb.).
- Warna/style spesifik sengaja tidak digunakan agar diagram tetap portable dan mudah dibaca di berbagai renderer.
- Penamaan node/service pada seluruh diagram konsisten dengan nama service di `../architecture/SERVICE_CATALOG.md` untuk menghindari ambiguitas.

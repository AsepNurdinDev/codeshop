# Diagrams — CodeShop

Seluruh diagram arsitektur CodeShop dibuat menggunakan **Mermaid** dan disematkan langsung di dalam dokumen Markdown terkait (tidak ada file gambar biner terpisah), sesuai instruksi Phase 1/1.1.

## Index Diagram

| # | Diagram | Lokasi |
|---|---|---|
| 1 | System Context | `../architecture/ARCHITECTURE.md` §1 |
| 2 | Service Architecture / High-Level Architecture | `../architecture/ARCHITECTURE.md` §2 |
| 3 | Trust Boundaries (aplikasi) | `../architecture/ARCHITECTURE.md` §4 |
| 3a | Trust Boundaries (lengkap, termasuk CI/CD, Registry, Control Plane) | `../architecture/NETWORK_DESIGN.md` §3 |
| 4 | Network Architecture (traffic path) | `../architecture/NETWORK_DESIGN.md` §1 |
| 5 | Checkout Flow | `../architecture/ARCHITECTURE.md` §8 |
| 6 | Payment Flow | `../architecture/ARCHITECTURE.md` §9 |
| 7 | Payment State Machine | `../architecture/ARCHITECTURE.md` §11b |
| 8 | Order State Machine | `../architecture/ARCHITECTURE.md` §11b |
| 9 | Download Security / Authorization Flow | `../architecture/ARCHITECTURE.md` §10, §11a |
| 10 | Event Architecture / Event Flow | `../architecture/EVENT_DESIGN.md` §13 |
| 11 | Database Ownership (Entity Relationship Overview) | `../architecture/DATABASE_DESIGN.md` §12 |
| 12 | CI/CD Pipeline | `../architecture/CI_CD_DESIGN.md` §1 |
| 13 | GitOps Flow | `../architecture/CI_CD_DESIGN.md` §1 (bagian GitOps repo → Argo CD → Kubernetes) |
| 14 | Kubernetes Production Architecture | `../architecture/ARCHITECTURE.md` §18 |
| 15 | Observability Architecture | Direpresentasikan secara tabular (golden signals, SLI/SLO, dashboard) di `../architecture/OBSERVABILITY.md` — tidak dibuat sebagai diagram Mermaid terpisah karena hubungan antar pilar (metrics/logs/traces → Grafana/Alertmanager) lebih jelas disampaikan sebagai tabel daripada graph node, mengikuti prinsip menghindari diagram yang dipaksakan tanpa menambah kejelasan (lihat `TRADE_OFFS.md`). |
| 16 | Disaster Recovery | Direpresentasikan sebagai tabel RPO/RTO dan skenario kegagalan di `../architecture/DISASTER_RECOVERY.md` §2, §5 — dipilih tabular karena informasi utamanya adalah nilai target per komponen, bukan alur/relasi yang butuh divisualisasikan sebagai graph. |
| 17 | Threat Model | Direpresentasikan sebagai tabel risiko per kategori aset di `../architecture/THREAT_MODEL.md` §2–§7 — format tabel (Asset/Threat/Vector/Impact/Likelihood/Risk/Mitigation) lebih dapat ditelusuri dan diperbarui dibanding diagram. |

> **Catatan konsistensi**: Diagram #3 (Trust Boundaries — aplikasi) di `ARCHITECTURE.md` §4 dan #3a (Trust Boundaries — lengkap) di `NETWORK_DESIGN.md` §3 sengaja tidak digabung menjadi satu diagram tunggal: #3 berfokus pada trust boundary di level komunikasi aplikasi (Client → Gateway → Service → Storage), sementara #3a memperluas cakupan ke zona infrastruktur (CI/CD, Container Registry, Kubernetes Control Plane) yang relevan untuk `THREAT_MODEL.md` §6–§7. Keduanya konsisten satu sama lain (tidak ada kontradiksi penamaan zona), hanya berbeda level detail sesuai kebutuhan dokumen masing-masing.

## Konvensi Diagram

- Semua diagram menggunakan sintaks Mermaid standar (`graph`, `sequenceDiagram`, `erDiagram`, `stateDiagram-v2`, `C4Context`) yang didukung oleh kebanyakan renderer Markdown modern (GitHub, GitLab, dsb.).
- Warna/style spesifik sengaja tidak digunakan agar diagram tetap portable dan mudah dibaca di berbagai renderer.
- Penamaan node/service pada seluruh diagram konsisten dengan nama service di `../architecture/SERVICE_CATALOG.md` untuk menghindari ambiguitas.
- Diagram baru pada Phase 1.1 (State Machine Payment/Order, Trust Zone lengkap, CI/CD, Kubernetes production) mengikuti konvensi yang sama dan telah diperiksa silang terhadap tabel/dependency matrix di dokumen terkait agar tidak ada kontradiksi (lihat §29 Document Consistency Rule pada instruksi Phase 1.1, ditindaklanjuti di seluruh dokumen `architecture/*.md`).

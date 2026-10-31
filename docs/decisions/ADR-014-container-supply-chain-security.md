# ADR-014: Container & Supply Chain Security

## Status

Accepted

## Context

CodeShop menjual produk digital dan memproses pembayaran — kompromi pada rantai build/deploy (supply chain) dapat berujung pada backdoor produksi yang berdampak langsung ke data finansial dan source code pelanggan (lihat `THREAT_MODEL.md` §7 Supply Chain Threats). Target teknologi mencakup Trivy, Syft, dan Cosign.

## Decision

CodeShop menerapkan target pipeline supply-chain berikut (konseptual, lihat `CI_CD_DESIGN.md` untuk alur lengkap):

```text
Source → Dependency → Build → Container Image → Scan (Trivy) → SBOM (Syft) → Sign (Cosign) → Registry → Deploy
```

Dengan prinsip tambahan:

- **Minimal base image** (mis. distroless atau Alpine minimal) untuk mengurangi permukaan serangan.
- **Non-root container** secara default.
- **Read-only filesystem** di mana memungkinkan.
- **Dropped Linux capabilities** (`drop: ["ALL"]`, tambahkan hanya yang benar-benar dibutuhkan).
- **Resource limits** wajib pada setiap container.
- **Immutable tags & digest pinning** — deployment selalu merujuk digest image spesifik, tidak pernah tag mengambang.

## Alternatives Considered

1. **Tanpa scanning otomatis, hanya review manual dependency** — ditolak, tidak scalable dan rawan human error, terutama untuk transitive dependency yang jarang direview manual.
2. **Scanning tanpa signing (Trivy + Syft saja, tanpa Cosign)** — memberi visibilitas kerentanan tetapi tidak memberi jaminan provenance (bahwa image yang di-deploy benar-benar berasal dari pipeline CI yang sah, bukan disisipkan di tengah jalan) — ditolak sebagai solusi akhir, meskipun dapat menjadi langkah awal bertahap saat implementasi.
3. **Full pipeline (scan + SBOM + sign)** — **dipilih**, memberi pertahanan berlapis: kerentanan diketahui (Trivy), komposisi dependency transparan (Syft), dan integritas/asal image terverifikasi (Cosign).

## Consequences

**Positif**: kerentanan pada dependency/base image terdeteksi sebelum produksi; SBOM memberi visibilitas cepat saat CVE baru diumumkan ("apakah kita terpengaruh?" dapat dijawab dari SBOM tanpa rebuild); image signing mencegah deployment image yang tidak berasal dari pipeline resmi.

**Negatif**: menambah waktu build pipeline (scanning & signing tidak instan); memerlukan kebijakan yang jelas untuk menangani temuan CVE severity rendah/menengah agar tidak memblokir pipeline secara berlebihan (hanya severity tinggi/kritis yang blocking, lihat `CI_CD_DESIGN.md` §8) — kebijakan yang terlalu ketat pada severity rendah akan menciptakan alert fatigue dan menghambat kecepatan tim tanpa manfaat keamanan proporsional.

## Security Implications

- Admission policy di Kubernetes (Phase implementasi) dapat dikonfigurasi untuk menolak deploy image yang tidak memiliki signature Cosign valid — menjadikan signing bukan sekadar rekomendasi tetapi kontrol teknis yang ditegakkan.
- Non-root + read-only filesystem + dropped capabilities secara signifikan mengurangi dampak jika satu container berhasil dieksploitasi (mengurangi kemampuan attacker melakukan lateral movement, selaras `THREAT_MODEL.md` §6).
- Digest pinning mencegah "supply chain drift" di mana tag yang sama secara diam-diam menunjuk ke image berbeda di registry.

## Operational Implications

Tim perlu menetapkan kebijakan triase CVE (severity apa yang blocking vs. yang dicatat sebagai technical debt dengan batas waktu perbaikan) sebagai bagian dari Phase implementasi CI (Phase 5). SBOM dan hasil scan disimpan sebagai artifact CI untuk keperluan audit/investigasi insiden (`INCIDENT_RESPONSE.md` §4 — Container vulnerability).

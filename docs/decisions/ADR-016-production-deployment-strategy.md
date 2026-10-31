# ADR-016: Production Deployment Strategy

## Status

Accepted

## Context

CodeShop memerlukan strategi deployment produksi yang memberi keseimbangan antara kecepatan rilis, keamanan (mengurangi risiko downtime/regresi pada alur kritikal seperti checkout dan payment), dan kompleksitas operasional yang proporsional dengan skala tim (lihat `TRADE_OFFS.md`).

## Decision

CodeShop mengadopsi **rolling deployment** sebagai strategi deployment produksi awal, dengan promotion gate manual dari Staging ke Production (lihat `CI_CD_DESIGN.md` §4, §10), dan rollback melalui Git revert pada GitOps repository.

## Alternatives Considered

1. **Blue/green deployment** — memberi rollback instan dan isolasi penuh antar versi, tetapi memerlukan dua kali resource cluster selama masa transisi dan kompleksitas routing tambahan (dua environment penuh berjalan paralel) — ditolak sebagai strategi awal karena biaya infrastruktur dan operasional lebih tinggi tanpa manfaat proporsional pada volume trafik saat ini. Dicatat sebagai peningkatan **Future** untuk Payment Service secara spesifik jika risiko deployment pada domain itu dianggap perlu isolasi lebih tinggi di masa depan.
2. **Canary deployment** — memberi deteksi dini masalah dengan blast radius terbatas, tetapi memerlukan traffic-splitting capability (Ingress/service mesh canary) dan volume trafik yang cukup besar agar canary bermakna secara statistik — ditolak sebagai strategi awal, dicatat sebagai **Future** setelah observability dan volume trafik matang (`TRADE_OFFS.md`).
3. **Rolling deployment** — **dipilih**, native di Kubernetes tanpa tooling tambahan, sudah memberi zero-downtime deployment (dengan readiness probe yang benar) untuk sebagian besar skenario, dan proporsional dengan kompleksitas operasional tim saat ini.

## Consequences

**Positif**: tidak memerlukan tooling/infrastruktur tambahan di luar kapabilitas native Kubernetes; deployment zero-downtime untuk kasus umum; kompleksitas operasional rendah, cocok untuk tim kecil.

**Negatif**: rollback tidak seinstan blue/green (masih memerlukan waktu bagi Kubernetes untuk mengganti pod kembali ke versi lama); tidak ada isolasi bertahap seperti canary — bug yang lolos staging berpotensi terlihat oleh 100% trafik begitu rollout selesai, bukan hanya sebagian kecil. Risiko ini dimitigasi oleh gate staging + automated verification sebelum promotion (`CI_CD_DESIGN.md` §4), bukan dihilangkan sepenuhnya.

## Security Implications

Rolling deployment yang dikombinasikan dengan artifact immutability (ADR-014, digest pinning) memastikan setiap rollout dapat ditelusuri ke image spesifik yang sudah melalui seluruh security gate (`CI_CD_DESIGN.md` §8) — tidak ada "hot patch" langsung ke pod yang berjalan.

## Operational Implications

Readiness probe yang akurat (`RELIABILITY.md` §10, `KUBERNETES_DESIGN.md` §6) menjadi prasyarat kritikal agar rolling deployment benar-benar zero-downtime — pod baru yang belum sepenuhnya siap tidak boleh menerima trafik. Strategi ini akan dievaluasi ulang (berpotensi menjadi ADR baru) begitu volume trafik atau risiko bisnis membenarkan investasi ke blue/green atau canary.

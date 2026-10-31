# ADR-018: Disaster Recovery Strategy

## Status

Accepted

## Context

CodeShop menyimpan data finansial (Payment DB), data transaksi (Order DB), dan aset digital bernilai (Object Storage) yang tidak boleh hilang tanpa rencana pemulihan yang jelas. Namun CodeShop juga merupakan proyek dengan tim dan anggaran kecil, sehingga strategi DR harus realistis, bukan meniru pola hyperscale yang tidak proporsional (lihat `TRADE_OFFS.md`).

## Decision

CodeShop mengadopsi strategi DR **single-region dengan RPO/RTO terdiferensiasi per komponen** (`DISASTER_RECOVERY.md` §2) — komponen finansial (Payment, Order) mendapat target RPO/RTO paling ketat yang dapat dicapai tim saat ini (RPO ≤15 menit, RTO ≤2 jam), sementara komponen non-kritikal (Catalog, Notification) mendapat target yang lebih longgar. Backup dianggap tidak valid sampai proses restore diuji secara berkala (`BACKUP_STRATEGY.md`).

## Alternatives Considered

1. **Multi-region active-active** — memberi RTO mendekati nol dan ketahanan terhadap kegagalan region penuh, tetapi ditolak karena kompleksitas (data replication lintas region, konsistensi, biaya infrastruktur ganda) sangat tidak proporsional untuk skala bisnis dan tim saat ini — eksplisit dicatat sebagai **Future**, bukan kebutuhan sekarang (`REQUIREMENTS.md` Non-Goals).
2. **Single-region tanpa strategi backup terstruktur (mengandalkan durability provider semata)** — ditolak, terlalu berisiko untuk data finansial; durability provider (mis. replikasi disk) tidak melindungi dari kesalahan logis (data terhapus/dikorupsi oleh bug aplikasi), yang memerlukan backup point-in-time terpisah.
3. **Single-region dengan backup terstruktur + RPO/RTO terdiferensiasi per kekritisan data** — **dipilih**, memberi perlindungan proporsional: investasi terbesar pada data yang paling mahal untuk hilang (finansial), tanpa memaksakan biaya redundansi penuh pada seluruh sistem.

## Consequences

**Positif**: risiko kehilangan data finansial diminimalkan melalui RPO ketat + backup teruji; biaya infrastruktur tetap proporsional karena tidak seluruh komponen diperlakukan setara.

**Negatif**: kegagalan region penuh tetap berarti downtime signifikan (tidak ada failover otomatis lintas region) — ini adalah risiko yang **diterima secara sadar** pada tahap kematangan bisnis saat ini, didokumentasikan secara eksplisit alih-alih diabaikan diam-diam (`DISASTER_RECOVERY.md` §6 Explicit Non-Goals).

## Security Implications

Backup terenkripsi at-rest dan disimpan terpisah dari sumber data (`BACKUP_STRATEGY.md` §1) — backup yang tidak terenkripsi atau disimpan di lokasi yang sama dengan sumbernya akan menjadi vektor kebocoran/kegagalan tambahan, bukan mitigasi.

## Operational Implications

Uji restore terjadwal (kuartalan untuk Payment/Order DB) menjadi item wajib berulang, bukan aktivitas satu kali — dilacak sebagai bagian dari `PRODUCTION_READINESS.md` dan menjadi bahan evaluasi apakah target RPO/RTO di ADR ini masih realistis seiring pertumbuhan sistem, berpotensi memicu ADR baru jika strategi perlu diperketat (mis. menuju replikasi lintas region) di masa depan.

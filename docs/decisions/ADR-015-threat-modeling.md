# ADR-015: Threat Modeling Practice

## Status

Accepted

## Context

Sebagai marketplace yang menangani pembayaran dan aset digital bernilai (source code), CodeShop memiliki permukaan risiko yang tidak trivial (lihat kategori ancaman di `THREAT_MODEL.md`). Tanpa proses threat modeling yang eksplisit, keputusan keamanan cenderung ad-hoc dan reaktif (baru dipikirkan setelah insiden terjadi).

## Decision

CodeShop mengadopsi **lightweight, asset-centric threat modeling** sebagai praktik berkelanjutan (bukan aktivitas satu kali di Phase 1.1), didokumentasikan di `THREAT_MODEL.md`, dengan aturan:

- Threat model direview ulang setiap kali ada perubahan arsitektur signifikan (service baru, alur data baru, integrasi eksternal baru) — bukan dokumen statis.
- Setiap insiden keamanan nyata (lihat `INCIDENT_RESPONSE.md` §5) wajib memicu update terhadap `THREAT_MODEL.md` jika ancaman yang terjadi belum terdaftar.
- Risiko diklasifikasikan kualitatif (`Impact × Likelihood`) — bukan kuantitatif formal (mis. FAIR model) yang memerlukan data historis yang belum dimiliki tim di tahap ini.

## Alternatives Considered

1. **Formal STRIDE per komponen untuk seluruh sistem** — lebih menyeluruh secara metodologis, tetapi ditolak sebagai pendekatan wajib di Phase 1.1 karena volume dokumentasi yang dihasilkan tidak proporsional dengan jumlah engineer yang akan memeliharanya; berisiko menjadi dokumen yang dibuat sekali lalu tidak pernah diperbarui (stale documentation risk lebih besar dari manfaatnya).
2. **Tidak ada threat modeling terdokumentasi (hanya code review ad-hoc)** — ditolak, terlalu reaktif dan tidak memberi visibilitas prioritas risiko ke seluruh tim/stakeholder non-teknis.
3. **Lightweight asset-centric model dengan risk table** — **dipilih**, cukup terstruktur untuk memandu keputusan mitigasi (lihat prioritisasi Critical risk di `THREAT_MODEL.md` §8) tanpa overhead pemeliharaan berlebihan.

## Consequences

**Positif**: prioritas mitigasi keamanan (`THREAT_MODEL.md` §8) menjadi eksplisit dan dapat dirujuk saat membuat keputusan desain baru; memberi bahasa bersama antara tim teknis dan pemilik produk untuk mendiskusikan risiko.

**Negatif**: lightweight model dapat melewatkan ancaman yang lebih halus yang baru terungkap melalui analisis STRIDE penuh per komponen — mitigasi: threat model bersifat hidup dan diperbarui, bukan dianggap lengkap selamanya.

## Security Implications

Threat model secara langsung menjadi input bagi `PRODUCTION_READINESS.md` (item "Threat model telah direview dan risiko Critical memiliki mitigasi terverifikasi") — menjadikannya bagian dari gate kesiapan produksi, bukan dokumen dekoratif.

## Operational Implications

Pemilik dokumen `THREAT_MODEL.md` adalah tim/person yang juga bertanggung jawab atas `INCIDENT_RESPONSE.md`, memastikan siklus deteksi-mitigasi-dokumentasi tetap tertutup (closed loop).

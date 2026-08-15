# ADR-008: Object Storage untuk File Produk Digital

## Status

Accepted

## Context

Produk digital (arsip source code/template) perlu disimpan secara aman, tidak dapat diakses langsung oleh publik, dan dapat diserahkan ke pembeli yang sah melalui mekanisme akses terbatas waktu (lihat ADR-009).

## Decision

CodeShop menyimpan seluruh file produk digital di **object storage S3-compatible (MinIO atau layanan setara)**, dalam **private bucket** yang tidak dapat diakses langsung tanpa signed URL.

## Alternatives Considered

1. **Menyimpan file di filesystem lokal/volume Kubernetes** — ditolak karena tidak scalable secara horizontal (file harus tersedia di semua replica/pod Download Service), sulit untuk backup terpusat, dan tidak menyediakan mekanisme signed URL native.
2. **Menyimpan file sebagai BLOB di PostgreSQL** — ditolak karena tidak efisien untuk file berukuran menengah, membebani database yang seharusnya fokus pada data transaksional terstruktur (melanggar prinsip di ADR-002/ADR-006).
3. **Object storage S3-compatible (MinIO/S3)** — **dipilih**, mendukung signed URL native, scalable, dapat dijalankan self-hosted (MinIO) maupun managed (S3 atau kompatibel), dan menjadi standar de facto untuk penyimpanan file di arsitektur cloud-native.

## Consequences

**Positif**: transfer file besar tidak membebani service aplikasi (client mengunduh langsung dari object storage via signed URL); scalable dan dapat diganti provider (S3-compatible) tanpa mengubah arsitektur aplikasi secara signifikan.

**Negatif**: menambah satu komponen infrastruktur yang perlu dikelola (jika self-hosted MinIO) termasuk backup dan high availability-nya sendiri (lihat Disaster Recovery Concept di `../architecture/ARCHITECTURE.md` §17).

## Security Implications

Bucket bersifat **private** secara default — tidak ada akses publik langsung. Akses hanya melalui signed URL berumur pendek yang diterbitkan oleh Download Service (lihat ADR-009). Credential akses object storage hanya dimiliki oleh service yang membutuhkan (Download Service untuk generate signed URL, Catalog Service untuk upload metadata/preview) — least privilege.

## Operational Implications

Memerlukan strategi backup/versioning object storage dan monitoring kapasitas storage sebagai bagian dari Disaster Recovery Concept.

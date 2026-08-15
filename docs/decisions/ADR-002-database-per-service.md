# ADR-002: Database per Service

## Status

Accepted

## Context

Mengikuti keputusan microservices (ADR-001), diperlukan strategi data ownership yang jelas agar service benar-benar independen, tidak saling terikat pada skema database bersama yang dapat menimbulkan coupling tersembunyi dan kesulitan evolusi skema.

## Decision

Setiap service memiliki **database sendiri** (database-per-service). Tidak ada service yang boleh mengakses database milik service lain secara langsung. Komunikasi data lintas service hanya melalui gRPC API atau event (lihat `../architecture/DATABASE_DESIGN.md` §8 untuk pola cross-service reference).

## Alternatives Considered

1. **Shared database** — satu database untuk seluruh service. Lebih sederhana untuk query lintas domain (JOIN langsung), tetapi ditolak karena menghilangkan manfaat utama microservices (independent deployability, isolasi kegagalan) dan menciptakan coupling implisit melalui skema bersama.
2. **Database per service dengan schema-per-service dalam satu instance PostgreSQL** — dipertimbangkan sebagai varian implementasi (bukan alternatif keputusan, melainkan detail teknis Phase implementasi) untuk efisiensi biaya di tahap awal, selama isolasi akses tetap dijaga di level user/permission.
3. **Database per service, instance terpisah penuh** — **dipilih sebagai arah target**, dengan catatan bahwa implementasi awal (Phase implementasi) dapat memulai dari opsi 2 sebagai stepping stone tanpa melanggar prinsip isolasi logis.

## Consequences

**Positif**: service benar-benar dapat dikembangkan dan dideploy independen; skema dapat berevolusi tanpa memengaruhi service lain; kegagalan database satu service tidak langsung menjatuhkan service lain.

**Negatif**: tidak ada JOIN lintas domain — data teragregasi harus digabung di application layer (via gRPC call atau denormalisasi terbatas); potensi data inconsistency sementara (eventual consistency) untuk data yang direplikasi/direferensikan lintas service; kompleksitas operasional bertambah (lebih banyak database untuk di-manage, backup, monitor).

## Security Implications

Membatasi credential database hanya untuk service pemiliknya memperkecil blast radius kebocoran data jika satu service disusupi — service lain tidak otomatis punya akses ke data domain lain.

## Operational Implications

Memerlukan strategi backup/restore per database (lihat `../architecture/ARCHITECTURE.md` §17 Disaster Recovery Concept) dan observability per database (lihat ADR-012).

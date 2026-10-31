# ADR-013: Secrets Management

## Status

Accepted

## Context

CodeShop menangani sejumlah secret bernilai tinggi: kredensial database, JWT signing key, secret webhook payment provider, kredensial object storage, dan kredensial infrastruktur (registry, cluster). Kebocoran salah satu dari secret ini termasuk kategori risiko **Critical** di `THREAT_MODEL.md` §6 (Infrastructure — Secret leakage). Dibutuhkan strategi yang jelas per environment (Development, Staging, Production, CI/CD, Kubernetes) tanpa memilih tooling yang berlebihan untuk skala tim saat ini.

## Decision

CodeShop mengadopsi pendekatan **bertingkat dan pragmatis**, bukan langsung mengadopsi HashiCorp Vault:

| Environment | Mekanisme |
|---|---|
| Local (developer) | File `.env` lokal (tidak di-commit, di-`.gitignore`), berisi nilai dummy/sandbox — bukan secret produksi. |
| CI/CD | Secret store bawaan platform CI (mis. GitHub Actions Secrets), diakses hanya oleh job yang membutuhkan, tidak pernah dicetak ke log job. |
| Kubernetes (Staging & Production) | **Kubernetes Secrets**, dikelola melalui **External Secrets Operator** yang mensinkronkan dari cloud secret manager (mis. secret manager bawaan penyedia cloud yang dipakai) — bukan Kubernetes Secrets mentah yang di-manage manual/di-commit ke GitOps repo dalam bentuk plaintext. |
| GitOps repository | Tidak pernah menyimpan secret plaintext. Jika diperlukan referensi terenkripsi di Git (mis. untuk nilai yang jarang berubah), menggunakan **SOPS** atau **Sealed Secrets** sebagai lapisan enkripsi sebelum commit — dipilih salah satu berdasarkan preferensi tooling tim saat implementasi, bukan diwajibkan keduanya. |

## Alternatives Considered

1. **HashiCorp Vault** — solusi paling matang dan fleksibel untuk secret management enterprise (dynamic secret, lease, audit granular), tetapi ditolak sebagai pilihan default karena overhead operasional (menjalankan & mengamankan Vault itu sendiri, unsealing, HA setup) tidak proporsional untuk tim kecil di tahap ini. **Tidak dipilih hanya karena populer** — dicatat secara eksplisit sebagai kandidat re-evaluasi di masa depan jika kebutuhan dynamic secret/lease jangka pendek menjadi kritikal.
2. **Kubernetes Secrets murni (tanpa External Secrets/sinkronisasi cloud)** — sederhana, tetapi Kubernetes Secrets native hanya base64-encoded (bukan terenkripsi secara default tanpa encryption-at-rest tambahan pada etcd) dan rawan menjadi sumber kebenaran yang tidak tersinkronisasi dengan baik — ditolak sebagai satu-satunya mekanisme, digunakan hanya sebagai **runtime delivery**, bukan source of truth.
3. **Environment variable langsung di manifest/Helm values (di-commit ke Git)** — ditolak keras, melanggar prinsip dasar bahwa secret tidak boleh berada di Git plaintext.
4. **External Secrets Operator + cloud secret manager** — **dipilih**, memberi source of truth terpusat di layanan yang sudah dikelola (mengurangi beban operasional dibanding Vault self-hosted) sambil tetap mengintegrasikan secara native dengan Kubernetes.

## Consequences

**Positif**: tidak ada secret plaintext di Git; source of truth secret terpusat dan dapat diaudit; overhead operasional jauh lebih rendah dibanding menjalankan Vault sendiri.

**Negatif**: bergantung pada secret manager penyedia cloud tertentu (sedikit vendor lock-in dibanding Vault yang lebih portable) — dianggap trade-off yang dapat diterima mengingat manfaat operasional; developer lokal tetap memerlukan disiplin untuk tidak pernah menyalin secret produksi ke `.env` lokal.

## Security Implications

- **Rotasi secret**: kredensial database, JWT signing key, dan webhook secret dirotasi secara berkala (jadwal ditentukan Phase implementasi); rotasi tidak boleh menyebabkan downtime (mendukung dual-key/grace period untuk JWT signing key khususnya).
- **Least privilege**: setiap service hanya memiliki akses ke secret miliknya sendiri (selaras `KUBERNETES_DESIGN.md` §8 ServiceAccount per service) — tidak ada secret bersama lintas service.
- **Prevention**: secret scanning di CI (mendeteksi kebocoran tidak sengaja sebelum merge) sebagai lapisan pertahanan tambahan, terlepas dari mekanisme penyimpanan.
- **Developer secrets vs Production secrets**: developer tidak pernah memerlukan akses ke secret production untuk bekerja secara lokal — nilai sandbox/dummy sudah cukup untuk seluruh alur pengembangan lokal (`ENVIRONMENT_STRATEGY.md` §2).

## Operational Implications

Memerlukan setup awal: External Secrets Operator di cluster, koneksi ke cloud secret manager, dan kebijakan RBAC yang membatasi siapa yang dapat membaca/menulis secret di secret manager itu sendiri (lapisan otorisasi di luar Kubernetes). Detail konfigurasi teknis adalah Phase implementasi (Phase 6/9), bukan Phase 1.1.

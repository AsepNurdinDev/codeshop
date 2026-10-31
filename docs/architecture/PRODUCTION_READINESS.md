# Production Readiness Checklist — CodeShop

Checklist ini adalah kriteria kelulusan sebelum CodeShop dianggap siap menerima trafik produksi nyata dan transaksi finansial nyata. Setiap item mengacu ke dokumen arsitektur yang mendefinisikannya secara detail — checklist ini adalah ringkasan yang dapat diverifikasi, bukan pengganti dokumen sumber.

## Application

- [ ] Authentication berfungsi sesuai `ARCHITECTURE.md` §11 (Auth flow) dan `API_DESIGN.md` §2.
- [ ] Authorization business-level diterapkan di setiap service pemilik data, bukan hanya di Gateway (`ARCHITECTURE.md` §11a).
- [ ] Input validation diterapkan di seluruh endpoint publik (`REQUIREMENTS.md` §8).
- [ ] Error handling konsisten mengikuti format di `API_DESIGN.md` §6.
- [ ] Idempotency diterapkan pada endpoint dengan efek samping finansial dan pada webhook (`API_DESIGN.md` §9, `RELIABILITY.md` §6).
- [ ] Graceful shutdown diimplementasikan di seluruh service, khususnya Payment & Download Service (`RELIABILITY.md` §9).
- [ ] Health checks (liveness/readiness/startup) berfungsi sesuai `RELIABILITY.md` §10 dan `KUBERNETES_DESIGN.md` §6.

## Security

- [ ] TLS end-to-end (Cloudflare → Ingress → internal, target mTLS internal) sesuai `NETWORK_DESIGN.md`.
- [ ] Secrets management sesuai ADR-013, tidak ada secret plaintext di Git.
- [ ] RBAC Kubernetes dikonfigurasi least-privilege sesuai `KUBERNETES_DESIGN.md` §8.
- [ ] NetworkPolicy default-deny diterapkan sesuai `KUBERNETES_DESIGN.md` §7 dan `NETWORK_DESIGN.md`.
- [ ] Image scanning (Trivy) dan dependency scanning menjadi gate wajib di CI (`CI_CD_DESIGN.md` §8).
- [ ] SBOM dihasilkan untuk setiap image produksi (ADR-014).
- [ ] Audit logging aktif untuk aksi sensitif (login, checkout, payment verification, download) sesuai `REQUIREMENTS.md` §8 dan `DATA_CLASSIFICATION.md`.
- [ ] Threat model (`THREAT_MODEL.md`) telah direview dan risiko Critical memiliki mitigasi terverifikasi.

## Infrastructure

- [ ] Resource requests/limits ditetapkan untuk seluruh deployment (`KUBERNETES_DESIGN.md` §4).
- [ ] HPA dikonfigurasi untuk service dengan trafik variabel (`KUBERNETES_DESIGN.md` §5).
- [ ] PodDisruptionBudget dikonfigurasi untuk seluruh service kritikal (`KUBERNETES_DESIGN.md` §6).
- [ ] Backup terjadwal aktif untuk seluruh database & object storage (`BACKUP_STRATEGY.md`).
- [ ] Restore test telah dilakukan dan RTO tercapai untuk database kritikal (Payment, Order) (`DISASTER_RECOVERY.md` §4).
- [ ] Monitoring (Prometheus/Grafana) aktif dengan dashboard golden signals (`OBSERVABILITY.md` §5).
- [ ] Alerting (Alertmanager) aktif untuk kondisi Critical/High (`OBSERVABILITY.md` §6).

## CI/CD

- [ ] Automated tests (unit, integration) berjalan sebagai gate wajib PR (`CI_CD_DESIGN.md` §2).
- [ ] Security gates (SAST, dependency scan, container scan) memblokir merge/deploy pada temuan severity tinggi (`CI_CD_DESIGN.md` §8).
- [ ] Artifact immutable (digest-based, bukan tag mengambang) (`CI_CD_DESIGN.md` §7).
- [ ] GitOps (Argo CD) mengelola deployment ke seluruh environment (`ADR-011`, `CI_CD_DESIGN.md`).
- [ ] Rollback path terverifikasi berfungsi (Git revert → Argo CD sync) (`CI_CD_DESIGN.md` §5).

## Operations

- [ ] Runbook operasional tersedia untuk skenario di `INCIDENT_RESPONSE.md` §4 (setidaknya payment outage, database outage, unauthorized download).
- [ ] Proses incident response (`INCIDENT_RESPONSE.md`) dipahami oleh seluruh anggota tim yang berpotensi menjadi on-call.
- [ ] Disaster recovery plan (`DISASTER_RECOVERY.md`) telah direview dan RPO/RTO disepakati sebagai target yang realistis untuk tim saat ini.
- [ ] SLO awal (`OBSERVABILITY.md` §3) disepakati dan dashboard untuk memantaunya sudah aktif.

## Business/Product Sign-off (di luar scope teknis murni, tetapi bagian dari kesiapan produksi)

- [ ] Payment provider sudah dalam mode live (bukan sandbox) dengan kontrak/verifikasi bisnis selesai.
- [ ] Kebijakan refund, dispute, dan support user telah ditetapkan (walau implementasinya Phase 2, kebijakannya perlu ada sebelum go-live).
- [ ] Kebijakan privasi & syarat layanan yang relevan dengan `DATA_CLASSIFICATION.md` telah dipublikasikan kepada user.

## Explicit Notes

- Checklist ini **bukan** gerbang all-or-nothing yang kaku untuk fitur non-kritikal — item di bawah "Business/Product Sign-off" dan sebagian item Future-leaning dapat dinegosiasikan berdasarkan risiko, tetapi seluruh item di bawah **Security** dan item RPO/RTO Payment/Order di **Infrastructure** dianggap non-negotiable sebelum menerima transaksi finansial nyata.
- Checklist ini direview ulang setiap kali ada perubahan arsitektur signifikan, bukan dokumen statis yang diisi sekali.

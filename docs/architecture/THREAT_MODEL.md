# Threat Model — CodeShop

Ini adalah lightweight threat model (bukan full STRIDE per-komponen formal), difokuskan pada aset yang paling bernilai bagi CodeShop: identitas user, integritas pembayaran, dan kerahasiaan produk digital (source code).

## 1. Scope & Approach

Threat model ini mengikuti pendekatan berbasis aset (asset-centric): untuk setiap kategori aset, kami mendaftar ancaman realistis, vektor serangan, dampak, kemungkinan, level risiko, dan mitigasi yang **sudah didesain** (Phase 1/1.1) — bukan mitigasi hipotetis yang belum ada rencananya.

Level risiko dihitung kualitatif: `Risk ≈ Impact × Likelihood`, dengan skala `Low / Medium / High / Critical`.

## 2. Identity Threats

| Asset | Threat | Attack Vector | Impact | Likelihood | Risk | Mitigation |
|---|---|---|---|---|---|---|
| User account | Credential stuffing | Attacker mencoba kredensial bocor dari breach lain | Account takeover | Medium | High | Rate limiting login (`SERVICE_CATALOG.md` §2), password hashing Argon2id, deteksi anomali login (Phase 2: alerting berbasis pola). |
| User account | Brute force login | Percobaan password berulang | Account takeover | Medium | High | Rate limiting per-IP & per-account, lockout sementara setelah N percobaan gagal. |
| Session | Access token theft (XSS/leak di client) | Token dicuri dari browser/localStorage | Impersonasi user | Low–Medium | Medium | Access token TTL pendek, refresh token revocable di Redis, rekomendasi client menyimpan token secara aman (httpOnly cookie dipertimbangkan di Phase implementasi). |
| Session | Refresh token replay | Refresh token dicuri dan dipakai ulang | Perpanjangan akses ilegal | Low | Medium | Refresh token rotation (token lama di-invalidate setelah dipakai), deteksi reuse sebagai sinyal kompromi. |
| Role/permission | Privilege escalation | User biasa mengeksploitasi bug otorisasi untuk mendapat akses admin | Kontrol penuh atas katalog/data | Low | High | Role check di level service (bukan Gateway), lihat `ARCHITECTURE.md` §11a; audit log perubahan role. |

## 3. API Threats

| Asset | Threat | Attack Vector | Impact | Likelihood | Risk | Mitigation |
|---|---|---|---|---|---|---|
| Any write endpoint | Injection (SQLi dsb.) | Input tidak divalidasi diteruskan ke query | Kebocoran/kerusakan data | Low | High | Parameterized query/ORM (Phase implementasi), input validation di setiap service (`REQUIREMENTS.md` §8). |
| Order/Payment/Download endpoint | Broken authorization / IDOR | Mengganti `order_id`/`product_id` di request untuk mengakses resource user lain | Kebocoran data order/pembayaran/entitlement user lain | Medium | High | Ownership check wajib di setiap service (`ARCHITECTURE.md` §11a), bukan hanya cek keberadaan ID. |
| Checkout/payment endpoint | Replay request | Request checkout/payment dikirim ulang | Order/payment duplikat | Medium | Medium | `Idempotency-Key` wajib untuk endpoint dengan efek samping finansial (`API_DESIGN.md` §9). |
| Public endpoint | Abuse/scraping katalog | Bot melakukan scraping besar-besaran | Beban server, kebocoran data kompetitif | Medium | Low–Medium | Rate limiting, Cloudflare bot management (Phase implementasi/infrastruktur), cache agresif di Catalog Service. |
| Any public endpoint | Denial of Service (DoS) | Flood request ke endpoint publik | Downtime layanan | Medium | High | Cloudflare WAF/DDoS protection di edge, rate limiting di Gateway, resource limit per pod (Kubernetes). |

## 4. Payment Threats

| Asset | Threat | Attack Vector | Impact | Likelihood | Risk | Mitigation |
|---|---|---|---|---|---|---|
| Webhook endpoint | Fake webhook | Attacker mengirim payload webhook palsu tanpa signature valid | Order dianggap PAID secara ilegal | Medium | Critical | Signature verification wajib sebelum diproses (`API_DESIGN.md` §Webhook, koreksi ownership §1a), request tanpa signature valid ditolak dan dicatat di `WebhookProcessing`. |
| Webhook endpoint | Replayed webhook | Payload valid lama dikirim ulang oleh attacker yang mengintersep | Duplikasi efek samping (double entitlement) | Low–Medium | Medium | Idempotency via `ProviderReference` unique constraint, timestamp validation (webhook terlalu lama ditolak). |
| Payment amount | Manipulated amount | Attacker mencoba memengaruhi amount yang diverifikasi | Pembayaran kurang bayar dianggap lunas | Low | Critical | Amount/currency divalidasi terhadap data order (`GetOrder` read-only) sebelum `Payment` ditandai `SUCCEEDED`, bukan dipercaya mentah dari payload webhook. |
| Payment↔Order integrity | Duplicate payment | User membayar dua kali untuk order yang sama (double-click, retry) | Kebingungan akuntansi, potensi refund manual | Medium | Medium | Idempotency key di `POST /payments`, `Order` yang sudah `PAID` menolak sesi pembayaran baru. |
| Payment↔Order integrity | Payment/order mismatch | Race condition antara dua order dari user yang sama | Entitlement salah diberikan | Low | High | `order_id` selalu menjadi bagian dari `ProviderReference`/metadata sesi pembayaran, divalidasi ulang saat webhook diproses. |

## 5. Download Threats

| Asset | Threat | Attack Vector | Impact | Likelihood | Risk | Mitigation |
|---|---|---|---|---|---|---|
| Product file | Unauthorized download | Request download tanpa entitlement valid | Kebocoran source code | Medium | Critical | Entitlement + order/payment check wajib sebelum signed URL diterbitkan (`ARCHITECTURE.md` §10). |
| Product file | IDOR pada endpoint download | Mengganti `product_id` di request | Download produk yang tidak dibeli | Medium | Critical | Entitlement check berbasis `(user_id, product_id)` yang tervalidasi, bukan hanya keberadaan produk. |
| Signed URL | URL sharing | User sah membagikan signed URL yang masih aktif ke pihak lain | Distribusi ilegal terbatas window waktu | Medium | Medium | **Diterima secara sadar sebagai limitasi teknis** (lihat ADR-009) — dimitigasi dengan TTL pendek dan limit penerbitan URL, bukan diklaim tercegah sepenuhnya. |
| Product catalog/file | Scraping/bulk download abuse | Script otomatis mencoba mengunduh berulang | Beban object storage, potensi kebocoran skala besar | Low–Medium | Medium | Rate limiting penerbitan signed URL per user/produk, audit log & alert pada pola abnormal. |
| Product file | Excessive downloads dari satu entitlement | User berulang kali generate signed URL baru | Beban biaya bandwidth, indikasi sharing | Medium | Low–Medium | Download limit/policy per entitlement per periode waktu (`ARCHITECTURE.md` §10 download policy check). |

## 6. Infrastructure Threats

| Asset | Threat | Attack Vector | Impact | Likelihood | Risk | Mitigation |
|---|---|---|---|---|---|---|
| Container/pod | Compromised container | Vulnerable dependency dieksploitasi di runtime | Lateral movement, akses data | Low–Medium | High | Image scanning (Trivy), minimal base image, non-root container, read-only filesystem, dropped capabilities (`KUBERNETES_DESIGN.md`, ADR-014). |
| Dependency | Vulnerable dependency | Library dengan CVE diketahui dipakai tanpa patch | Eksploitasi remote | Medium | High | Dependency scanning otomatis di CI (`CI_CD_DESIGN.md`), SBOM untuk visibilitas (ADR-014). |
| Secrets | Secret leakage | Secret ter-commit ke Git atau ter-log | Kompromi database/service lain | Medium | Critical | Secrets management terpisah dari kode (ADR-013), scanning secret di CI, `.gitignore`/pre-commit hook (Phase implementasi). |
| Cluster network | Lateral movement | Pod yang dikompromikan mencoba mengakses service lain | Perluasan blast radius | Low–Medium | High | NetworkPolicy default-deny antar namespace/service (`KUBERNETES_DESIGN.md`, `NETWORK_DESIGN.md`). |
| Database | Exposed database | Database ter-expose ke publik akibat misconfiguration | Kebocoran data massal | Low | Critical | Database hanya cluster-internal (tidak ada LoadBalancer/NodePort publik), NetworkPolicy membatasi akses hanya dari service pemilik. |
| Message broker | Exposed message broker | RabbitMQ management UI/port ter-expose | Manipulasi/pembacaan event | Low | High | RabbitMQ cluster-internal only, kredensial per service, tidak ada expose publik. |

## 7. Supply Chain Threats

| Asset | Threat | Attack Vector | Impact | Likelihood | Risk | Mitigation |
|---|---|---|---|---|---|---|
| Dependency tree | Malicious dependency | Package pihak ketiga disusupi (typosquatting/compromised maintainer) | Kode berbahaya masuk ke build | Low–Medium | High | Dependency scanning, pinning versi, review dependency baru (kebijakan tim, Phase implementasi). |
| Base image | Compromised base image | Base image publik mengandung malware/vuln | Kompromi seluruh container turunan | Low | High | Minimal & trusted base image, scanning rutin (Trivy), digest pinning (bukan `latest`). |
| Build artifact | Vulnerable image | Image hasil build mengandung CVE yang belum dipatch | Eksploitasi runtime | Medium | High | Container scanning wajib sebagai gate CI sebelum push ke registry (`CI_CD_DESIGN.md`). |
| Build pipeline | Poisoned build artifact | CI/CD disusupi, artifact dimodifikasi sebelum deploy | Backdoor di produksi | Low | Critical | SBOM (Syft) + image signing (Cosign) + provenance, immutable tags & digest pinning saat deploy (ADR-014). |

## 8. Risk Summary & Prioritization

Ancaman dengan risiko **Critical** yang mendapat prioritas mitigasi tertinggi pada arah desain saat ini:

1. Fake/manipulated webhook menyebabkan order dianggap PAID tanpa pembayaran nyata.
2. Unauthorized download / IDOR pada endpoint download → kebocoran source code.
3. Secret leakage → kompromi lintas service.
4. Poisoned build artifact → backdoor produksi.

Ancaman dengan risiko **Medium/Low** tetap didokumentasikan agar tidak diabaikan, tetapi tidak memerlukan kontrol tambahan di luar apa yang sudah didesain pada Phase 1.1 — menghindari over-engineering kontrol keamanan untuk skenario berdampak rendah (lihat `TRADE_OFFS.md`).

## 9. Explicit Non-Goals of This Threat Model

- Ini bukan penetration test atau audit keamanan formal.
- Tidak mencakup ancaman fisik/hardware.
- Tidak mencakup ancaman terhadap payment provider itu sendiri (di luar kendali CodeShop) — hanya interaksi CodeShop dengannya.
- Model ini akan direvisi ulang setiap kali ada perubahan arsitektur signifikan (bagian dari proses ADR, bukan dokumen statis).

# Requirements — CodeShop

## 1. Project Overview

CodeShop adalah marketplace digital untuk menjual **source code dan software template** (boilerplate, starter kit, UI kit, script, template aplikasi, dsb). Penjual mengunggah produk digital ke platform, pembeli melakukan pembayaran, dan setelah pembayaran terverifikasi pembeli mendapatkan akses download produk melalui tautan yang bersifat sementara (temporary signed URL) ke private object storage.

Fokus Phase 1 adalah **arsitektur dan desain sistem**, bukan implementasi.

## 2. Problem Statement

Penjual source code/template saat ini umumnya menjual melalui kanal umum (marketplace generik, repo pribadi, atau transfer manual) yang tidak memiliki:

- Kontrol akses download yang aman dan dapat diaudit
- Alur checkout dan verifikasi pembayaran yang terstruktur
- Pemisahan hak akses berbasis kepemilikan order yang valid
- Observability terhadap penyalahgunaan (download berulang, sharing link, dsb)

CodeShop menyediakan platform khusus untuk transaksi produk digital dengan kontrol akses yang lebih ketat dan dapat diaudit.

## 3. Goals

- Menyediakan marketplace untuk browsing, pembelian, dan download source code/template secara aman.
- Memberikan hak download hanya kepada user dengan order yang **PAID**.
- Menggunakan temporary signed URL untuk mencegah akses langsung ke object storage.
- Arsitektur berbasis microservices yang dapat berkembang secara independen per domain.
- Observability penuh (metrics, logs, traces) sejak awal desain.
- Desain yang aman secara default (secure by default) untuk data pembayaran dan file produk.

## 4. Non-Goals (Phase 1 & MVP)

- Tidak membangun fitur affiliate/referral.
- Tidak membangun sistem review/rating produk (dipertimbangkan di Future Scope).
- Tidak membangun multi-currency/multi-region tax engine.
- Tidak membangun custom payment gateway sendiri (integrasi ke payment provider pihak ketiga).
- Tidak membangun mobile native app (hanya web).
- Tidak melakukan implementasi kode, infrastruktur, atau deployment pada Phase 1 ini.

## 5. Actors

| Actor | Deskripsi |
|---|---|
| Guest | Pengunjung belum login, dapat browsing katalog dan melihat detail produk. |
| Registered User / Buyer | User yang sudah register/login, dapat checkout, membayar, dan download produk yang dimiliki. |
| Seller/Admin (internal, MVP: single-seller/platform-owned) | Pihak yang mengunggah dan mengelola produk di katalog. |
| Payment Provider (external) | Pihak ketiga yang memproses pembayaran dan mengirim notifikasi status. |
| System/Internal Services | Service-to-service actor (Notification, Download, dsb). |

> Catatan: MVP mengasumsikan model **single-seller/platform sebagai penjual** (CodeShop sendiri yang menjual produk digital). Model multi-seller marketplace didokumentasikan sebagai **Future Scope** karena menambah kompleksitas signifikan pada payment split, KYC, dan payout — berpotensi over-engineering untuk MVP.

## 6. Functional Requirements

### MVP

1. Guest dapat browsing katalog produk dan melihat detail produk (deskripsi, harga, preview, versi).
2. User dapat register dan login (email + password).
3. User dapat menambahkan produk ke cart.
4. User dapat melakukan checkout dari cart menjadi order.
5. User dapat melakukan pembayaran melalui payment provider pihak ketiga.
6. Sistem menerima notifikasi status pembayaran (webhook) dan memverifikasinya.
7. Order berubah status menjadi **PAID** setelah pembayaran terverifikasi.
8. User dengan order **PAID** mendapatkan entitlement download atas produk terkait.
9. User dapat melakukan download melalui temporary signed URL.
10. Sistem mencatat setiap permintaan/aktivitas download (audit).
11. Sistem mengirim notifikasi (mis. email) untuk event penting: registrasi, order dibuat, pembayaran berhasil/gagal, entitlement diberikan.
12. Admin (internal) dapat mengelola katalog produk (create/update/deactivate).

### Phase 2

13. Refund request & refund processing.
14. Multiple product versions dengan changelog per versi, dan user dapat memilih versi mana yang ingin di-download (masih dalam entitlement yang sama).
15. Diskon/kupon/promo code.
16. Rate limiting adaptif berbasis behavior (bukan hanya fixed window).

### Future

17. Multi-seller marketplace dengan payout terpisah per seller.
18. Review & rating produk.
19. Wishlist dan rekomendasi produk.
20. Subscription/bundle produk.

## 7. Non-Functional Requirements

- **Availability**: target awal 99.5% untuk MVP (single-region), tanpa multi-region failover otomatis di Phase 1.
- **Latency**: p95 API publik (catalog browsing) < 300ms; p95 endpoint checkout/payment < 800ms (di luar latensi payment provider eksternal).
- **Consistency**: strong consistency di dalam boundary satu service/database; eventual consistency antar service melalui event.
- **Maintainability**: setiap service dapat dikembangkan, diuji, dan dideploy secara independen.
- **Portability**: seluruh service berjalan sebagai container, tidak bergantung pada vendor tertentu di level aplikasi.

## 8. Security Requirements

- Autentikasi berbasis JWT dengan access token berumur pendek dan refresh token.
- Password disimpan menggunakan algoritma hashing modern (mis. Argon2id) — detail di `ADR` terkait bila diperlukan pada Phase implementasi.
- Otorisasi berbasis kepemilikan resource (ownership-based) untuk order, payment, dan download — bukan hanya role-based.
- Download hanya dapat dilakukan melalui signed URL yang memiliki masa berlaku terbatas.
- Seluruh komunikasi eksternal menggunakan TLS.
- Audit log untuk aksi sensitif: login, checkout, payment verification, download.
- Rate limiting untuk endpoint publik dan endpoint autentikasi guna mencegah brute force/abuse.
- Webhook dari payment provider harus diverifikasi signature-nya sebelum diproses.

## 9. Reliability Requirements

- Setiap service memiliki liveness dan readiness check konseptual.
- Komunikasi asynchronous antar service menggunakan message broker dengan dead-letter queue untuk pesan yang gagal diproses berulang kali.
- Operasi yang memicu efek samping (mis. pemberian entitlement, pengiriman notifikasi) harus idempotent terhadap retry.
- Kegagalan pada service non-kritikal (mis. Notification Service) tidak boleh menggagalkan alur kritikal (mis. proses payment/order).

## 10. Scalability Requirements

- Service yang menerima trafik tinggi (Catalog, Download) harus dapat di-scale secara horizontal secara independen dari service lain.
- Desain database per service memungkinkan scaling storage secara independen.
- Object storage produk digital harus dapat diakses secara scalable tanpa membebani service aplikasi (akses langsung client → object storage via signed URL).

## 11. Observability Requirements

- Setiap service mengekspos metrics (Prometheus-compatible) untuk request rate, error rate, dan latency minimal.
- Setiap service menghasilkan structured logs yang dapat dikorelasikan lintas service menggunakan correlation ID.
- Distributed tracing untuk alur lintas service (checkout → payment → entitlement → download).
- Metrics bisnis (jumlah order paid, download berhasil, payment gagal) harus terukur, bukan hanya metrics teknis.

## 12. Business Rules

- Order hanya dapat masuk status **PAID** setelah pembayaran diverifikasi oleh payment provider (bukan berdasarkan klaim dari client).
- Entitlement download hanya diberikan untuk order dengan status **PAID**.
- Satu order dapat berisi lebih dari satu produk (multi-item order), masing-masing menghasilkan entitlement terpisah.
- Signed URL download memiliki waktu kedaluwarsa singkat dan tidak dapat diperpanjang tanpa request ulang yang melalui otorisasi.
- Produk yang dinonaktifkan (deactivated) oleh admin tidak dapat dibeli lagi, tetapi user yang sudah memiliki entitlement tetap dapat mengunduhnya (kecuali dinyatakan lain oleh kebijakan bisnis Phase 2).

## 13. Constraints

- Phase 1 hanya menghasilkan dokumentasi arsitektur; tidak ada kode, konfigurasi, atau infrastruktur yang diimplementasikan.
- Teknologi target sudah ditentukan (Go, gRPC, REST, PostgreSQL, Redis, RabbitMQ, S3-compatible storage, Kubernetes, dsb) sebagai arah, bukan sebagai keputusan final yang tidak dapat dievaluasi ulang.
- Tim pengembang kemungkinan kecil (indie/startup-scale), sehingga desain harus menghindari over-engineering yang tidak proporsional dengan skala tim.

## 14. Assumptions

- MVP beroperasi single-region.
- Payment diproses oleh payment provider pihak ketiga yang mendukung webhook notification.
- Volume awal transaksi tidak memerlukan sharding database sejak hari pertama.
- Produk digital berupa file (arsip source code) berukuran relatif kecil–menengah (bukan file berukuran puluhan GB).

## 15. MVP Scope

Lihat daftar Functional Requirements bagian **MVP** di atas — mencakup end-to-end flow: browse → register/login → cart → checkout → payment → verifikasi → entitlement → download, dengan observability dan security dasar.

## 16. Future Scope

Lihat daftar Functional Requirements bagian **Phase 2** dan **Future** di atas.

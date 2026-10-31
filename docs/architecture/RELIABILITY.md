# Reliability Engineering — CodeShop

## 1. Principle

Reliability mechanisms diterapkan **selektif**, berdasarkan karakteristik risiko masing-masing komunikasi — bukan diterapkan seragam ("retry everywhere") yang dapat memperburuk masalah (mis. retry pada operasi non-idempotent, atau retry storm yang memperparah outage).

## 2. Timeout

- Setiap panggilan sinkron (gRPC, REST ke provider eksternal) memiliki timeout eksplisit, lebih pendek dari timeout caller-nya (mis. timeout Order→Catalog lebih pendek dari timeout Gateway→Order) untuk mencegah efek "timeout menumpuk".
- Timeout ke Payment Provider eksternal diberi margin lebih longgar (provider pihak ketiga dapat lebih lambat), tetapi tetap dibatasi — kegagalan di sini menghasilkan error jelas ke user, bukan hang tanpa batas.

## 3. Retry & Backoff

Diterapkan **hanya** pada operasi yang:

1. Idempotent (aman diulang), atau
2. Memiliki mekanisme idempotency eksplisit (idempotency key, dedup by event_id).

| Operasi | Retry? | Alasan |
|---|---|---|
| gRPC read-only (`GetProduct`, `GetOrder`) | Ya, dengan exponential backoff, jumlah percobaan terbatas | Read-only, aman diulang. |
| gRPC write tanpa idempotency key | **Tidak** secara otomatis di level transport | Berisiko duplikasi efek samping; jika perlu retry, harus di level aplikasi dengan idempotency check eksplisit. |
| Event consumption (RabbitMQ) | Ya, dengan backoff, hingga batas maksimum lalu ke DLQ | Consumer didesain idempotent (`EVENT_DESIGN.md` §8), sehingga retry aman. |
| Webhook Payment Provider → Payment Service | Retry dikendalikan oleh **provider**, bukan oleh CodeShop | CodeShop hanya perlu memastikan endpoint idempotent terhadap retry pihak provider. |
| Publish event ke RabbitMQ dari service | Ya, dengan backoff, disertai outbox pattern untuk mencegah event hilang jika publish gagal setelah commit database | Mencegah kehilangan event akibat kegagalan sesaat broker. |

Backoff menggunakan strategi **exponential dengan jitter** (mis. 1s, 2s, 4s, ... + random jitter) untuk menghindari efek thundering herd saat dependency pulih dari outage.

## 4. Circuit Breaker

Diterapkan pada panggilan ke dependency yang **berpotensi mengalami outage berkepanjangan** dan di mana kegagalan cepat lebih baik daripada menunggu timeout berulang:

- API Gateway → setiap backend service (mencegah satu service down membuat Gateway ikut lambat/habis resource menunggu).
- Payment Service → Payment Provider (mencegah retry berulang ke provider yang sedang down memperparah beban).

Tidak diterapkan pada komunikasi antar service internal yang sudah dilindungi oleh timeout ketat dan volume rendah (mis. Download Service → Catalog Service) — circuit breaker di sini menambah kompleksitas tanpa manfaat proporsional pada skala trafik MVP.

## 5. Bulkhead

- Resource (connection pool, goroutine worker pool) untuk memanggil dependency berbeda dipisahkan secara logis (mis. pool koneksi ke Catalog Service terpisah dari pool ke Auth Service) sehingga kelambatan satu dependency tidak menghabiskan resource yang dibutuhkan untuk memanggil dependency lain.
- Diterapkan terutama di API Gateway dan Order Service (yang memanggil beberapa service sekaligus saat checkout).

## 6. Idempotency

- **Public API** dengan efek samping finansial: `Idempotency-Key` header wajib (`API_DESIGN.md` §9).
- **Webhook**: idempotency via `ProviderReference` unique constraint (`DATABASE_DESIGN.md` §5, `THREAT_MODEL.md` §Payment).
- **Event consumer**: dedup by `event_id` (`EVENT_DESIGN.md` §8), terutama krusial untuk transisi `PENDING_PAYMENT → PAID` di Order Service dan pembuatan `DownloadEntitlement` di Download Service.

## 7. Queue Retry & Dead-Letter Queue

- Setiap queue konsumen memiliki retry dengan backoff (lihat §3) dan DLQ pasangan (`EVENT_DESIGN.md` §10).
- Pesan di DLQ **tidak di-drop** — dipertahankan untuk investigasi dan kemungkinan replay manual setelah root cause diperbaiki.
- Metric `dlq_size` dipantau sebagai alert (`OBSERVABILITY.md` §6).

## 8. Graceful Degradation

| Dependency Gagal | Degradasi yang Diterima |
|---|---|
| Notification Service down | Checkout/payment/download tetap berjalan normal; notifikasi tertunda hingga service pulih (event tetap di queue). |
| Catalog cache (Redis) tidak tersedia | Catalog Service fallback ke query langsung ke database (lebih lambat, tetapi tetap berfungsi), bukan gagal total. |
| Object storage lambat/tidak tersedia | Download Service mengembalikan error jelas ke user (tidak ada fallback fungsional — file memang tidak dapat diambil), tetapi tidak menjatuhkan service lain. |
| Auth Service down | Validasi JWT baru gagal (login/refresh tidak dapat dilakukan), namun token yang masih valid tetap dapat diverifikasi secara stateless oleh service lain via public key tanpa memanggil Auth Service — mengurangi dampak terhadap operasi baca yang sudah terautentikasi. |

## 9. Graceful Shutdown

- Setiap service menangani sinyal terminasi (`SIGTERM`) dengan: berhenti menerima request baru, menyelesaikan request in-flight dalam batas waktu tertentu, menutup koneksi database/broker dengan bersih, baru kemudian keluar.
- Khusus penting untuk Payment Service (webhook sedang diproses) dan Download Service (signed URL generation in-flight) — pemutusan paksa di tengah proses ini berisiko menciptakan state tidak konsisten. Lihat `KUBERNETES_DESIGN.md` §6.

## 10. Health Checks

- **Liveness**: proses masih hidup dan responsif (bukan deadlock).
- **Readiness**: service siap menerima trafik (dependency kritikal — database — dapat dihubungi).
- Health check tidak memeriksa dependency non-kritikal (mis. Notification Service tidak menjadikan readiness-nya bergantung pada email provider eksternal — kegagalan sesaat email provider tidak boleh membuat pod dianggap unhealthy dan di-restart berulang).

## 11. Dependency Failure & Partial Failure

- Kegagalan satu dependency non-kritikal tidak boleh menjadi kegagalan total (lihat §8 Graceful Degradation).
- Kegagalan dependency kritikal (mis. Order Service tidak dapat dihubungi saat checkout) menghasilkan **fail-fast dengan error jelas**, bukan mencoba melanjutkan dengan data tidak lengkap/tidak tervalidasi (lihat `ARCHITECTURE.md` §12 Failure Scenarios).

## 12. Eventual Consistency

- Window eventual consistency yang disengaja (`PaymentSucceeded → OrderPaid`, lihat `EVENT_DESIGN.md` §12) dipantau sebagai metric SLO (`OBSERVABILITY.md` §3), bukan diabaikan sebagai detail implementasi.
- UI/UX (di luar scope dokumen ini) diharapkan mengomunikasikan status "processing" kepada user selama window ini, bukan mengasumsikan konsistensi instan.

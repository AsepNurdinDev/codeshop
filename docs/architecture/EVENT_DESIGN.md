# Event Design — CodeShop

## 1a. Ownership Correction (Phase 1.1)

Phase 1 dokumentasi awal menggunakan `PaymentSuccess`/`OrderPaid` sebagai satu event yang digabung, dipublish oleh Payment Service, dengan status order diupdate **synchronous** oleh Payment Service via gRPC `UpdateOrderStatus` sebelum publish.

Setelah review arsitektur (Phase 1.1), keputusan ini **dikoreksi**:

- Payment Service **tidak lagi** memanggil `UpdateOrderStatus` pada Order Service. Payment Service hanya boleh menulis ke domainnya sendiri (`Payment`, `PaymentTransaction`, dst. — lihat `DATABASE_DESIGN.md` §5).
- Event pembayaran dipisah menjadi `PaymentSucceeded` dan `PaymentFailed`, dipublish oleh Payment Service sebagai pernyataan otoritatif tentang **pembayaran**, bukan tentang **order**.
- `OrderPaid` tetap ada sebagai event terpisah, tetapi sekarang dipublish oleh **Order Service** — bukan Payment Service — setelah Order Service sendiri memutuskan untuk mentransisikan order ke `PAID` sebagai reaksi terhadap `PaymentSucceeded`.

Alasan koreksi: pada desain awal, Payment Service secara implisit menjadi penulis (writer) untuk `OrderStatus`, yang melanggar prinsip *single writer per aggregate* dan bertentangan dengan prinsip database-per-service/ownership yang sudah dianut di seluruh sistem (ADR-002). Desain baru memastikan setiap service hanya pernah menulis state miliknya sendiri, dengan konsekuensi berupa satu hop asynchronous tambahan (lihat `ARCHITECTURE.md` §9 untuk trade-off latensi vs. ownership ini) yang dianggap dapat diterima secara bisnis.

## 1. Evaluasi Kebutuhan Event

Daftar kandidat event dari brief awal dievaluasi sebagai berikut:

| Event Kandidat | Keputusan | Alasan |
|---|---|---|
| `UserRegistered` | **Dipakai** | Memicu email welcome via Notification Service. |
| `OrderCreated` | **Dipakai** | Memicu notifikasi "order dibuat, menunggu pembayaran". |
| `PaymentPending` | **Dipakai (opsional trigger notifikasi)** | Berguna untuk observability/analytics, notifikasi ke user bersifat opsional (tidak wajib email). |
| `PaymentSucceeded` | **Dipakai (revisi Phase 1.1)** | Sebelumnya diberi nama `PaymentSuccess` dan digabung konsepnya dengan `OrderPaid`; setelah koreksi ownership (§1a), `PaymentSucceeded` kini menjadi event **terpisah** yang murni menyatakan fakta pembayaran, dipublish oleh Payment Service, dikonsumsi oleh Order Service (bukan oleh Download/Notification Service secara langsung). |
| `PaymentFailed` | **Dipakai** | Memicu notifikasi kegagalan ke user; dikonsumsi Order Service (untuk kemungkinan transisi/analytics) dan Notification Service. |
| `OrderPaid` | **Dipakai — sekarang dipublish oleh Order Service** (koreksi §1a) | Trigger utama untuk Download Service (buat entitlement) dan Notification Service. Tidak lagi dipublish oleh Payment Service. |
| `DownloadRequested` | **Tidak dipakai sebagai event** | Bersifat synchronous request-response (user menunggu signed URL langsung), tidak ada consumer asynchronous yang membutuhkan notifikasi "permintaan download dibuat" — akan menjadi over-engineering. |
| `DownloadGranted` | **Dipakai** | Untuk audit trail dan opsional notifikasi "produk Anda siap diunduh". |
| `DownloadCompleted` | **Tidak dipakai pada MVP** | Memerlukan callback dari object storage yang menambah kompleksitas signifikan (event dari S3-compatible storage) tanpa kebutuhan bisnis mendesak di MVP. Dicatat sebagai kandidat Phase 2 jika dibutuhkan analytics penyelesaian download. |
| `RefundRequested` | **Tidak dipakai pada MVP** | Fitur refund adalah Phase 2 (lihat `REQUIREMENTS.md`); event ini didesain saat fitur refund direalisasikan. |
| `RefundCompleted` | **Tidak dipakai pada MVP** | Sama seperti di atas. |

### Event Final MVP

```text
UserRegistered
OrderCreated
PaymentPending      (opsional/analytics)
PaymentSucceeded
PaymentFailed
OrderPaid
DownloadGranted
```

## 2. Event Naming

- Format: `PascalCase`, past-tense (menyatakan fakta yang sudah terjadi), tanpa versi di nama event — versi ada di envelope (`event_version`).
- Contoh: `OrderCreated`, `OrderPaid`, `PaymentFailed`.

## 3. Event Envelope

Struktur konseptual (bukan skema final/kode):

```text
{
  event_id: string (UUID),
  event_name: string,          // "OrderPaid"
  event_version: int,          // 1
  occurred_at: timestamp,
  correlation_id: string,      // konsisten sepanjang alur bisnis terkait
  producer: string,            // "payment-service"
  payload: { ... }             // data spesifik event
}
```

## 4. Producer & Consumer

| Event | Producer | Consumer |
|---|---|---|
| `UserRegistered` | Auth Service | Notification Service |
| `OrderCreated` | Order Service | Notification Service |
| `PaymentPending` | Payment Service | (analytics/observability, tanpa consumer bisnis wajib di MVP) |
| `PaymentSucceeded` | Payment Service | **Order Service** (trigger transisi ke `PAID`) |
| `PaymentFailed` | Payment Service | Order Service (analytics/logging), Notification Service |
| `OrderPaid` | **Order Service** (koreksi §1a) | Download Service, Notification Service |
| `DownloadGranted` | Download Service | Notification Service |

## 5. Event Version

- Setiap event memiliki `event_version` (integer, dimulai dari 1).
- Perubahan payload yang breaking (menghapus/mengubah tipe field) memerlukan increment versi dan consumer harus mendukung kedua versi selama masa transisi.

## 6. Correlation ID

- `correlation_id` dibuat di titik awal alur bisnis (mis. saat `OrderCreated`) dan diteruskan ke seluruh event turunan (`OrderPaid`, `DownloadGranted`) agar seluruh rangkaian dapat ditelusuri sebagai satu alur end-to-end, konsisten dengan `X-Correlation-ID` di REST/gRPC (lihat `API_DESIGN.md`).

## 7. Delivery Semantics

- RabbitMQ dikonfigurasi untuk **at-least-once delivery**.
- Konsekuensi: consumer **wajib idempotent** (lihat §9) karena pesan yang sama berpotensi diterima lebih dari sekali.

## 8. Idempotency

- Setiap consumer menyimpan/mengecek `event_id` yang sudah diproses (dedup table atau mekanisme setara) sebelum menjalankan efek samping, khususnya untuk:
  - **Order Service** yang mengonsumsi `PaymentSucceeded` — transisi `PENDING_PAYMENT → PAID` tidak boleh dieksekusi dua kali untuk order yang sama (idempotency di sini krusial karena ini adalah satu-satunya jalur order menjadi `PAID`).
  - **Download Service** yang mengonsumsi `OrderPaid` — pembuatan `DownloadEntitlement` tidak boleh duplikat.
  - **Notification Service** — email tidak boleh terkirim dobel.

## 9. Retry

- Consumer yang gagal memproses pesan melakukan retry dengan **exponential backoff** (mis. 1s, 2s, 4s, ... hingga batas maksimum percobaan).
- Setelah batas maksimum tercapai, pesan dipindahkan ke Dead-Letter Queue (lihat §10), bukan di-drop begitu saja.

## 10. Dead-Letter Queue (DLQ)

- Setiap queue utama memiliki DLQ pasangan.
- Pesan di DLQ dipertahankan untuk investigasi manual/replay, disertai metrics `dlq_size` yang dipantau (lihat `ARCHITECTURE.md` §16 Observability).

## 11. Ordering

- Ordering **tidak dijamin secara global** antar event berbeda jenis.
- Untuk event dalam satu alur order yang sama (mis. `OrderCreated` → `OrderPaid`), ordering secara alami terjaga karena keduanya dipicu oleh state transition sekuensial dan konsumen (Notification Service) tidak bergantung pada urutan strict antar event berbeda entity — masing-masing notifikasi independen secara konten.
- Jika di masa depan dibutuhkan strict ordering per entity (mis. menggunakan RabbitMQ dengan single active consumer per queue atau partition key = `order_id`), ini didokumentasikan sebagai pertimbangan Phase 2, bukan kebutuhan MVP.

## 12. Eventual Consistency

- **Revisi Phase 1.1**: status order **tidak lagi** diupdate synchronous oleh Payment Service. Transisi `PENDING_PAYMENT → PAID` terjadi secara **eventually consistent**, dipicu oleh konsumsi event `PaymentSucceeded` di Order Service. Window eventual consistency ini kecil dalam praktik (dalam orde detik, tergantung throughput RabbitMQ) dan dipantau sebagai metric (`payment-to-order-paid lag`, lihat `OBSERVABILITY.md`).
- Efek turunan lebih lanjut — pembuatan `DownloadEntitlement` dan pengiriman notifikasi — tetap bersifat **eventually consistent** di atas eventual consistency order itu sendiri (dua lapis), yang secara bisnis dapat diterima: user melihat status "menunggu konfirmasi" sesaat sebelum entitlement aktif, bukan penolakan permanen.
- Trade-off ini didokumentasikan secara eksplisit di `ARCHITECTURE.md` §9 sebagai keputusan sadar untuk menjaga ownership boundary yang bersih, bukan sebagai kompromi yang tidak disengaja.

## 13. Event Flow Diagram

```mermaid
sequenceDiagram
    participant Order as Order Service
    participant Payment as Payment Service
    participant MQ as RabbitMQ
    participant Download as Download Service
    participant Notif as Notification Service

    Order->>MQ: publish OrderCreated
    MQ->>Notif: consume OrderCreated

    Payment->>Payment: verify webhook, transition Payment to SUCCEEDED
    Payment->>MQ: publish PaymentSucceeded

    MQ->>Order: consume PaymentSucceeded
    Order->>Order: validate PENDING_PAYMENT, transition to PAID (idempotent by event_id)
    Order->>MQ: publish OrderPaid

    MQ->>Download: consume OrderPaid
    Download->>Download: create DownloadEntitlement (idempotent by event_id)
    Download->>MQ: publish DownloadGranted

    MQ->>Notif: consume OrderPaid
    MQ->>Notif: consume DownloadGranted
    Notif->>Notif: send email(s)

    Payment->>MQ: publish PaymentFailed (alternate path)
    MQ->>Order: consume PaymentFailed (analytics/logging, no order-owning side effect required)
    MQ->>Notif: consume PaymentFailed
```

# Architecture — CodeShop

## 1. System Context

CodeShop adalah sistem marketplace digital yang berinteraksi dengan:

- **Buyer/Guest** melalui web client.
- **Payment Provider** pihak ketiga (menerima pembayaran, mengirim webhook status).
- **Email/Notification Provider** pihak ketiga (SMTP/email API) untuk mengirim notifikasi.
- **Object Storage** (MinIO/S3-compatible) sebagai tempat penyimpanan file produk digital.

```mermaid
C4Context
  title System Context - CodeShop
  Person(guest, "Guest", "Belum login, browsing katalog")
  Person(buyer, "Registered Buyer", "User yang membeli & download produk")
  System(codeshop, "CodeShop Platform", "Marketplace source code & template")
  System_Ext(payment, "Payment Provider", "Memproses pembayaran, mengirim webhook")
  System_Ext(email, "Email/Notification Provider", "Mengirim email transaksional")
  System_Ext(storage, "Object Storage (S3-compatible)", "Menyimpan file produk digital")

  Rel(guest, codeshop, "Browse catalog")
  Rel(buyer, codeshop, "Register, checkout, pay, download")
  Rel(codeshop, payment, "Create payment, receive webhook")
  Rel(codeshop, email, "Send transactional email")
  Rel(codeshop, storage, "Store/retrieve product files, generate signed URL")
```

## 2. High-Level Architecture

CodeShop dibangun sebagai **microservices** dengan API Gateway sebagai satu-satunya entry point publik. Setiap service memiliki database sendiri (database-per-service). Komunikasi internal menggunakan gRPC untuk synchronous call dan RabbitMQ untuk asynchronous event.

```mermaid
graph TB
    Client[Web Client] --> GW[API Gateway]

    GW --> AUTH[Auth Service]
    GW --> CATALOG[Catalog Service]
    GW --> ORDER[Order Service]
    GW --> PAYMENT[Payment Service]
    GW --> DOWNLOAD[Download Service]

    ORDER -.grpc.-> CATALOG
    ORDER -.grpc.-> AUTH
    PAYMENT -.grpc.-> ORDER
    DOWNLOAD -.grpc.-> AUTH
    DOWNLOAD -.grpc.-> ORDER

    ORDER -->|event| MQ[(RabbitMQ)]
    PAYMENT -->|event| MQ
    DOWNLOAD -->|event| MQ
    MQ --> NOTIF[Notification Service]

    DOWNLOAD --> STORAGE[(Object Storage)]
    CATALOG --> STORAGE

    AUTH --> AUTHDB[(Auth DB)]
    CATALOG --> CATDB[(Catalog DB)]
    ORDER --> ORDDB[(Order DB)]
    PAYMENT --> PAYDB[(Payment DB)]
    DOWNLOAD --> DLDB[(Download DB)]
    NOTIF --> NOTIFDB[(Notification DB)]

    AUTH --> REDIS[(Redis)]
    GW --> REDIS
```

## 3. Service Architecture

Baseline service yang dievaluasi:

```text
API Gateway
Auth Service
Catalog Service
Order Service
Payment Service
Download Service
Notification Service
```

**Evaluasi**: baseline ini dipertahankan untuk MVP. Pertimbangan yang dievaluasi dan ditolak:

- **Cart sebagai service terpisah** — ditolak. Cart bersifat state sederhana dan berumur pendek (ephemeral), lebih tepat disimpan sebagai bagian dari Order Service (status `DRAFT`) atau di client-side/Redis, bukan service gRPC/database tersendiri. Membuat Cart Service terpisah menambah satu network hop dan satu database tanpa manfaat isolasi domain yang signifikan pada skala MVP. Cart didokumentasikan sebagai **sub-domain di dalam Order Service** (lihat `SERVICE_CATALOG.md`).
- **Product Version sebagai service terpisah** — ditolak, tetap menjadi bagian dari Catalog Service karena siklus hidup dan ownership data-nya melekat pada produk.
- **Search Service terpisah (mis. Elasticsearch-backed)** — ditolak untuk MVP, akan dipertimbangkan di Phase 2 jika volume katalog besar. Untuk MVP, pencarian dilakukan melalui query PostgreSQL di Catalog Service.

Detail tiap service ada di `SERVICE_CATALOG.md`.

## 4. Trust Boundaries

```mermaid
graph TB
    subgraph Untrusted["Untrusted Zone"]
        Client[Web Client / Browser]
    end

    subgraph DMZ["Edge / DMZ"]
        CDN[Cloudflare CDN + WAF]
        GW[API Gateway]
    end

    subgraph Trusted["Trusted Internal Zone (Cluster-internal)"]
        AUTH[Auth Service]
        CATALOG[Catalog Service]
        ORDER[Order Service]
        PAYMENT[Payment Service]
        DOWNLOAD[Download Service]
        NOTIF[Notification Service]
        MQ[(RabbitMQ)]
        DBs[(Databases)]
    end

    subgraph External["External Trusted Third Parties"]
        PP[Payment Provider]
        STORAGE[(Object Storage)]
    end

    Client --> CDN --> GW
    GW --> AUTH
    GW --> CATALOG
    GW --> ORDER
    GW --> PAYMENT
    GW --> DOWNLOAD
    PAYMENT <-->|webhook, TLS + signature verification| PP
    DOWNLOAD -->|signed URL| STORAGE
    Client -->|GET via signed URL only| STORAGE
```

Prinsip: hanya API Gateway yang dapat diakses dari luar cluster. Seluruh service internal tidak memiliki exposure publik. Object storage tidak pernah diakses langsung menggunakan credential permanen oleh client — hanya melalui signed URL berumur pendek.

## 5. Security Boundaries

- **Public boundary**: Client ↔ API Gateway (TLS, rate limiting, WAF via Cloudflare).
- **Service boundary**: antar service internal menggunakan mTLS/NetworkPolicy dalam cluster (konsep, detail implementasi di Phase 2/3).
- **Data boundary**: setiap service hanya boleh mengakses database miliknya sendiri; tidak ada shared database.
- **Storage boundary**: object storage private, hanya dapat diakses via signed URL yang diterbitkan Download Service.

## 6. Network Boundaries

- Public-facing: API Gateway (di belakang Cloudflare).
- Cluster-internal only: seluruh backend service, database, Redis, RabbitMQ.
- Object storage: private bucket, network policy membatasi akses hanya dari Download/Catalog Service untuk operasi manajemen; akses publik hanya melalui signed URL.

## 7. Communication Patterns

### Synchronous (gRPC / REST)

Digunakan ketika caller membutuhkan response langsung untuk melanjutkan alur (mis. validasi ownership sebelum generate signed URL).

| Caller | Callee | Protokol | Alasan |
|---|---|---|---|
| API Gateway | Semua service | REST (public) → gRPC (internal) | Entry point publik |
| Order Service | Catalog Service | gRPC | Validasi produk & harga saat checkout |
| Order Service | Auth Service | gRPC | Validasi identitas user |
| Payment Service | Order Service | gRPC | Update status order setelah verifikasi |
| Download Service | Auth Service | gRPC | Validasi token/identitas |
| Download Service | Order Service | gRPC | Cek kepemilikan order/entitlement |

### Asynchronous (RabbitMQ Event)

Digunakan untuk efek samping yang tidak perlu diketahui hasilnya secara langsung oleh pengirim (fire-and-forget dengan guarantee delivery), dan untuk decoupling antar domain.

| Event Producer | Event | Consumer |
|---|---|---|
| Order Service | `OrderCreated` | Notification Service |
| Payment Service | `OrderPaid` | Download Service, Notification Service |
| Payment Service | `PaymentFailed` | Notification Service |
| Download Service | `DownloadGranted` | Notification Service |

Detail lengkap di `EVENT_DESIGN.md`.

## 8. Checkout Flow

```mermaid
sequenceDiagram
    participant U as User
    participant GW as API Gateway
    participant O as Order Service
    participant C as Catalog Service
    participant A as Auth Service

    U->>GW: POST /cart/items
    GW->>O: AddCartItem (gRPC)
    O->>C: GetProduct(productId) (gRPC)
    C-->>O: product + price
    O-->>GW: cart updated
    GW-->>U: 200 OK

    U->>GW: POST /orders/checkout
    GW->>A: ValidateToken (gRPC)
    A-->>GW: userId
    GW->>O: CreateOrder(cart) (gRPC)
    O->>C: ValidateProducts & price snapshot
    C-->>O: OK
    O-->>GW: order (status=PENDING_PAYMENT)
    GW-->>U: order created
```

## 9. Payment Flow

```mermaid
sequenceDiagram
    participant U as User
    participant GW as API Gateway
    participant P as Payment Service
    participant PP as Payment Provider
    participant O as Order Service
    participant MQ as RabbitMQ

    U->>GW: POST /payments (orderId)
    GW->>P: CreatePayment (gRPC)
    P->>O: GetOrder(orderId) (gRPC)
    O-->>P: order detail
    P->>PP: Create payment session (REST)
    PP-->>P: payment URL/token
    P-->>GW: payment URL
    GW-->>U: redirect to payment page

    PP->>P: Webhook: payment status (signed)
    P->>P: Verify signature
    P->>O: UpdateOrderStatus(PAID) (gRPC)
    P->>MQ: publish OrderPaid
```

## 10. Download Flow

```mermaid
sequenceDiagram
    participant U as User
    participant GW as API Gateway
    participant D as Download Service
    participant A as Auth Service
    participant O as Order Service
    participant S as Object Storage

    U->>GW: GET /downloads/{productId}
    GW->>A: ValidateToken
    A-->>GW: userId
    GW->>D: RequestDownload(userId, productId)
    D->>O: CheckEntitlement(userId, productId) (gRPC)
    O-->>D: entitlement valid (order PAID)
    D->>S: Generate temporary signed URL
    S-->>D: signed URL (short TTL)
    D-->>GW: signed URL
    GW-->>U: 200 signed URL
    U->>S: GET file via signed URL
```

## 11. Authentication Flow

```mermaid
sequenceDiagram
    participant U as User
    participant GW as API Gateway
    participant A as Auth Service
    participant R as Redis

    U->>GW: POST /auth/login
    GW->>A: Login(email, password) (gRPC)
    A->>A: Verify password hash (Argon2id)
    A->>A: Issue access token (JWT, short TTL) + refresh token
    A->>R: Store refresh token session
    A-->>GW: access token + refresh token
    GW-->>U: tokens

    U->>GW: Request with expired access token
    GW-->>U: 401
    U->>GW: POST /auth/refresh (refresh token)
    GW->>A: Refresh(refreshToken)
    A->>R: Validate & rotate refresh token
    A-->>GW: new access token
```

## 12. Failure Scenarios

| Skenario | Dampak | Mitigasi |
|---|---|---|
| Payment webhook gagal diterima | Order tetap `PENDING_PAYMENT` walau user sudah bayar | Payment provider melakukan retry webhook; Payment Service juga menyediakan reconciliation job (polling status ke provider) sebagai fallback — didokumentasikan sebagai kebutuhan, bukan implementasi. |
| RabbitMQ down saat `OrderPaid` dipublish | Download entitlement tertunda | Payment Service tetap mengubah status order (source of truth di DB), event dipublish ulang dengan retry/outbox pattern. |
| Object Storage tidak dapat diakses | Signed URL gagal dibuat/gagal diakses | Download Service mengembalikan error jelas; retry di sisi client; observability alert. |
| Notification Service down | Email tidak terkirim | Tidak menggagalkan alur utama (checkout/payment/download tetap berjalan); event tetap di queue untuk diproses saat service pulih. |
| Auth Service down | Seluruh service yang butuh validasi token terganggu | Auth adalah dependency kritikal; didesain untuk high availability (multiple replica) dan token JWT tetap dapat divalidasi secara stateless (signature) tanpa selalu memanggil Auth Service — lihat `API_DESIGN.md`. |

## 13. Storage Architecture

- **Relational data**: PostgreSQL, satu instance/skema logis per service (database-per-service).
- **Cache/session**: Redis — digunakan untuk cache katalog (read-heavy), session/refresh token, dan rate limiting counter.
- **Message broker**: RabbitMQ — untuk event asynchronous antar service.
- **Object storage**: MinIO/S3-compatible — menyimpan file produk digital di private bucket, tidak pernah diakses langsung oleh client kecuali melalui signed URL.

## 14. Scalability

- API Gateway dan Catalog Service (read-heavy) di-scale horizontal terlebih dahulu.
- Download Service di-scale berdasarkan concurrency permintaan signed URL, bukan berdasarkan bandwidth file (karena transfer file terjadi langsung antara client dan object storage, bukan melalui service).
- Database per service memungkinkan scaling storage/index secara independen sesuai beban masing-masing domain.

## 15. Reliability

Lihat detail di `RELIABILITY` section pada dokumen ini level project — didokumentasikan lebih lanjut sebagai bagian dari operational readiness, mencakup retry, timeout, circuit breaker, dan idempotency di level design (bukan kode).

## 16. Observability

Setiap service didesain untuk mengekspos metrics, logs terstruktur, dan traces yang dapat dikorelasikan menggunakan correlation ID/trace ID lintas service — lihat kebutuhan detail per service di `SERVICE_CATALOG.md`.

## 17. Disaster Recovery Concept

Lihat pembahasan konsep RPO/RTO dan strategi backup di bagian akhir dokumen ini (ringkasan) — detail penuh sengaja tidak diberikan sebagai file terpisah pada Phase 1 karena termasuk dalam scope `ARCHITECTURE.md` sebagai bagian dari reliability design, mencakup: backup database terjadwal, backup/versioning object storage, dan prosedur restore konseptual untuk kegagalan node Kubernetes, database, object storage, dan RabbitMQ.

## 18. Target Kubernetes Architecture

```mermaid
graph TB
    subgraph Internet
        User[User Browser]
    end

    subgraph Cloudflare
        CF[Cloudflare CDN/WAF/DNS]
    end

    subgraph K8sCluster["Kubernetes Cluster"]
        subgraph IngressNS["ingress namespace"]
            ING[Ingress Controller]
        end

        subgraph AppNS["app namespace"]
            GWPOD[API Gateway Pods]
            AUTHPOD[Auth Service Pods]
            CATPOD[Catalog Service Pods]
            ORDPOD[Order Service Pods]
            PAYPOD[Payment Service Pods]
            DLPOD[Download Service Pods]
            NOTIFPOD[Notification Service Pods]
        end

        subgraph DataNS["data namespace"]
            PG[(PostgreSQL - per service)]
            REDIS[(Redis)]
            MQ[(RabbitMQ)]
        end

        subgraph ObsNS["observability namespace"]
            PROM[Prometheus]
            GRAF[Grafana]
            LOKI[Loki]
            TEMPO[Tempo]
        end
    end

    subgraph ExternalCloud["External Managed / Third-Party"]
        S3[(Object Storage)]
        PP[Payment Provider]
    end

    User --> CF --> ING --> GWPOD
    GWPOD --> AUTHPOD & CATPOD & ORDPOD & PAYPOD & DLPOD
    ORDPOD & PAYPOD & DLPOD & NOTIFPOD --> MQ
    AUTHPOD & CATPOD & ORDPOD & PAYPOD & DLPOD & NOTIFPOD --> PG
    AUTHPOD --> REDIS
    DLPOD --> S3
    PAYPOD --> PP

    AppNS -.metrics/logs/traces.-> ObsNS
```

Argo CD (GitOps) mengelola deployment ke cluster ini pada Phase implementasi; Helm chart per service. Detail implementasi tidak dibahas pada Phase 1.

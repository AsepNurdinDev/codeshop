# API Design — CodeShop

Dokumen ini mendeskripsikan kontrak API secara **konseptual**. Tidak ada `.proto` atau kode yang dihasilkan pada Phase 1.

## 1. API Versioning

- REST publik menggunakan path-based versioning: `/api/v1/...`.
- gRPC internal menggunakan package versioning pada level `.proto` (mis. `codeshop.order.v1`) — dirancang saat Phase implementasi.
- Breaking change memerlukan versi baru (`v2`); versi lama tetap didukung selama periode deprecation yang diumumkan.

## 2. Authentication

- Autentikasi menggunakan **JWT Bearer Token** pada header `Authorization: Bearer <token>`.
- Access token berumur pendek (mis. 15 menit, nilai final ditentukan saat implementasi).
- Refresh token berumur lebih panjang, disimpan sebagai session di Redis dan dapat di-revoke.
- Endpoint publik (browsing katalog) tidak memerlukan autentikasi.

## 3. Authorization

- Otorisasi berbasis **role** (`buyer`, `admin`) untuk operasi administratif.
- Otorisasi berbasis **ownership** untuk resource milik user (order, payment, download) — user hanya dapat mengakses resource miliknya sendiri, divalidasi di level service (bukan hanya di Gateway).

## 4. REST Conventions

- Menggunakan HTTP method sesuai semantik: `GET` (read), `POST` (create/action), `PUT/PATCH` (update), `DELETE` (hapus/soft-delete).
- Resource dinamai dengan plural noun (`/products`, `/orders`).
- Response menggunakan JSON dengan struktur konsisten: `{ "data": ..., "meta": ... }` untuk sukses, `{ "error": { "code", "message" } }` untuk gagal.

## 5. gRPC Conventions

- Service dan method dinamai dengan pola `VerbNoun` (mis. `GetProduct`, `CreateOrder`).
- Setiap request/response message memiliki nama eksplisit (`GetProductRequest`, `GetProductResponse`) — bukan primitive langsung.
- Menggunakan status code gRPC standar (`NOT_FOUND`, `PERMISSION_DENIED`, `INVALID_ARGUMENT`, dll.) dipetakan ke HTTP status code yang sesuai di API Gateway.

## 6. Error Response

Format error REST konsisten:

```text
{
  "error": {
    "code": "ORDER_NOT_FOUND",
    "message": "Order dengan ID tersebut tidak ditemukan",
    "request_id": "..."
  }
}
```

- `code` bersifat mesin-terbaca (machine-readable), stabil antar versi API.
- `message` bersifat human-readable, dapat berubah tanpa breaking change.
- `request_id` untuk korelasi dengan log/tracing.

## 7. Pagination

- Menggunakan **cursor-based pagination** untuk endpoint dengan potensi data besar (`/products`), agar konsisten meski data berubah selama scrolling.
- Parameter: `?cursor=<opaque>&limit=<n>`.
- Response menyertakan `next_cursor` pada `meta`.

## 8. Filtering & Sorting

- Filtering melalui query parameter eksplisit (mis. `/products?category=ui-kit&min_price=10`).
- Sorting melalui parameter `sort` (mis. `?sort=price_asc`, `?sort=newest`).
- Daftar field yang dapat difilter/disortir dibatasi eksplisit per endpoint (whitelist) untuk mencegah query yang tidak efisien.

## 9. Idempotency

- Endpoint yang memicu efek samping finansial (`POST /orders/checkout`, `POST /payments`) mendukung header `Idempotency-Key` yang dikirim client.
- Server menyimpan hasil request pertama untuk key tersebut dan mengembalikan hasil yang sama jika request diulang dengan key sama (mencegah duplikasi order/payment akibat retry client atau double-click).
- Webhook dari Payment Provider di-deduplikasi menggunakan `provider_reference_id` (lihat `DATABASE_DESIGN.md`).

## 10. Request ID

- Setiap request masuk diberi `X-Request-ID` oleh API Gateway (atau diteruskan bila sudah ada dari client/CDN).
- Digunakan untuk pelacakan single request pada log.

## 11. Correlation ID

- Untuk alur lintas service (mis. checkout → payment → entitlement), digunakan `X-Correlation-ID` yang tetap sama di seluruh service yang terlibat dalam satu alur bisnis, memungkinkan tracing end-to-end.
- Diteruskan melalui metadata gRPC dan event envelope (lihat `EVENT_DESIGN.md`).

---

## 12. Endpoint Groups

```text
/auth
/products
/cart
/orders
/payments
/downloads
/users
/admin
```

## 13. Key Endpoints

### POST /api/v1/auth/register

- **Purpose**: Registrasi user baru.
- **Authentication**: tidak diperlukan.
- **Authorization**: publik.
- **Request**: `email`, `password`, `full_name`.
- **Response**: `user_id`, `email` (tanpa password).
- **Error cases**: `EMAIL_ALREADY_EXISTS`, `INVALID_INPUT` (password lemah, email tidak valid), rate limited (`TOO_MANY_REQUESTS`).

### POST /api/v1/auth/login

- **Purpose**: Login dan penerbitan token.
- **Authentication**: tidak diperlukan.
- **Authorization**: publik.
- **Request**: `email`, `password`.
- **Response**: `access_token`, `refresh_token`, `expires_in`.
- **Error cases**: `INVALID_CREDENTIALS`, `ACCOUNT_SUSPENDED`, rate limited.

### GET /api/v1/products

- **Purpose**: Listing katalog produk.
- **Authentication**: tidak diperlukan.
- **Authorization**: publik.
- **Request**: query params `category`, `sort`, `cursor`, `limit`.
- **Response**: array `Product` (ringkas) + `meta.next_cursor`.
- **Error cases**: `INVALID_QUERY_PARAM`.

### GET /api/v1/products/{id}

- **Purpose**: Detail produk.
- **Authentication**: tidak diperlukan.
- **Authorization**: publik (hanya produk berstatus `published` yang terlihat oleh non-admin).
- **Response**: detail produk termasuk daftar `ProductVersion` (metadata, tanpa link download langsung).
- **Error cases**: `PRODUCT_NOT_FOUND`.

### POST /api/v1/cart/items

- **Purpose**: Menambahkan item ke cart.
- **Authentication**: wajib.
- **Authorization**: user hanya dapat mengubah cart miliknya.
- **Request**: `product_id`, `quantity`.
- **Response**: cart terbaru.
- **Error cases**: `PRODUCT_NOT_FOUND`, `PRODUCT_UNAVAILABLE`, `UNAUTHORIZED`.

### POST /api/v1/orders/checkout

- **Purpose**: Mengubah cart menjadi order (`PENDING_PAYMENT`).
- **Authentication**: wajib.
- **Authorization**: user hanya dapat checkout cart miliknya.
- **Request**: header `Idempotency-Key`; body opsional (mis. catatan).
- **Response**: `Order` (dengan `price_snapshot` tiap item, `total_amount`).
- **Error cases**: `CART_EMPTY`, `PRODUCT_UNAVAILABLE` (produk dinonaktifkan setelah masuk cart), `PRODUCT_PRICE_CHANGED` *(opsional notifikasi, bukan blocking)*.

### POST /api/v1/payments

- **Purpose**: Membuat sesi pembayaran untuk sebuah order.
- **Authentication**: wajib.
- **Authorization**: user hanya dapat membayar order miliknya, dan order harus berstatus `PENDING_PAYMENT`.
- **Request**: `order_id`, header `Idempotency-Key`.
- **Response**: `payment_url` atau `payment_token` dari provider.
- **Error cases**: `ORDER_NOT_FOUND`, `ORDER_NOT_PAYABLE` (status salah), `PAYMENT_PROVIDER_ERROR`.

### POST /api/v1/payments/webhook

- **Purpose**: Endpoint internal untuk menerima notifikasi status dari Payment Provider.
- **Authentication**: signature-based (bukan JWT user) — verifikasi menggunakan secret/signature dari provider.
- **Authorization**: hanya request dengan signature valid yang diproses.
- **Request**: payload sesuai format Payment Provider.
- **Response**: `200 OK` (acknowledgement).
- **Error cases**: `INVALID_SIGNATURE` (ditolak & dicatat sebagai security event), `DUPLICATE_EVENT` (diabaikan secara idempotent, tetap `200 OK`).

### GET /api/v1/downloads/{product_id}

- **Purpose**: Memperoleh temporary signed URL untuk mengunduh produk yang telah dibeli.
- **Authentication**: wajib.
- **Authorization**: user harus memiliki `DownloadEntitlement` aktif untuk produk tersebut.
- **Response**: `signed_url`, `expires_at`.
- **Error cases**: `ENTITLEMENT_NOT_FOUND` (belum membeli/order belum PAID), `PRODUCT_NOT_FOUND`, rate limited (`DOWNLOAD_LIMIT_EXCEEDED`).

### GET /api/v1/users/me

- **Purpose**: Mendapatkan profil user yang sedang login.
- **Authentication**: wajib.
- **Authorization**: hanya data milik diri sendiri.
- **Response**: `user_id`, `email`, `full_name`.
- **Error cases**: `UNAUTHORIZED`.

### POST /api/v1/admin/products

- **Purpose**: Membuat produk baru di katalog (admin only).
- **Authentication**: wajib.
- **Authorization**: role `admin` wajib.
- **Request**: `name`, `description`, `category_id`, `base_price`, `storage_object_key`.
- **Response**: `Product` yang dibuat (status `draft`).
- **Error cases**: `FORBIDDEN` (bukan admin), `INVALID_INPUT`.

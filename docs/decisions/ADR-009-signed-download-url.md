# ADR-009: Temporary Signed URL untuk Download

## Status

Accepted

## Context

Akses download produk digital harus dibatasi hanya kepada user dengan order **PAID**, dan tidak boleh memungkinkan akses langsung permanen ke file di object storage (lihat kebutuhan Download Security di `REQUIREMENTS.md` §8 dan alur di `ARCHITECTURE.md` §10).

## Decision

Download Service menerbitkan **temporary signed URL** berumur pendek setiap kali user yang berhak meminta akses download, alih-alih memberi URL publik permanen atau proxy seluruh transfer file melalui service aplikasi.

## Alternatives Considered

1. **Proxy download melalui Download Service** (service membaca file dari storage lalu stream ke client) — ditolak sebagai pendekatan utama karena membebani service aplikasi dengan bandwidth transfer file yang seharusnya bisa ditangani langsung oleh object storage, mengurangi scalability. Tetap dicatat sebagai fallback opsional jika suatu saat dibutuhkan kontrol ekstra (mis. watermarking dinamis) — dievaluasi ulang di Phase 2 jika diperlukan.
2. **URL publik permanen per file** — ditolak keras karena tidak ada kontrol akses sama sekali setelah URL diketahui; siapa pun yang memiliki link dapat mengunduh tanpa batas waktu.
3. **Temporary signed URL dengan TTL singkat** — **dipilih**, menyeimbangkan keamanan (akses terbatas waktu, tervalidasi entitlement sebelum penerbitan) dengan performa (transfer langsung client↔storage).

## Consequences

**Positif**: mengurangi beban Download Service (tidak perlu proxy file besar); memberikan lapisan otorisasi eksplisit sebelum setiap sesi download; audit trail penerbitan URL dapat dicatat (`DownloadRecord`).

**Negatif**: signed URL yang **masih aktif** (belum expired) secara teknis tetap dapat dibagikan/disalin ke pihak lain selama TTL berlaku — signed URL bukan jaminan mutlak bahwa hanya pembeli sah yang mengunduh dalam window waktu tersebut. Ini adalah batasan teknis yang diterima secara sadar (lihat Security section di `ARCHITECTURE.md`), dimitigasi dengan TTL yang singkat dan limit jumlah penerbitan URL per user/produk per periode waktu.

## Security Implications

- **URL expiration**: TTL singkat (nilai final ditentukan di Phase implementasi, indikasi awal beberapa menit).
- **Download limits**: rate limiting jumlah permintaan signed URL baru per user/produk per periode waktu untuk mengurangi window abuse.
- **Abuse prevention & hotlinking**: signed URL divalidasi terhadap entitlement setiap kali diterbitkan (bukan cache permanen); tidak ada endpoint yang mengekspos signed URL tanpa autentikasi.
- **URL sharing limitations**: seperti dijelaskan di atas, ini adalah batasan yang **diterima secara realistis**, bukan diklaim sebagai tidak mungkin terjadi.
- **File integrity**: object storage mendukung checksum/ETag bawaan untuk verifikasi integritas file (detail implementasi Phase berikutnya).
- **Storage isolation**: bucket file produk terpisah dari bucket lain (mis. asset publik/preview), lihat ADR-008.

## Operational Implications

Memerlukan observability khusus: signed URL issuance rate, download success/fail rate, dan pola permintaan abnormal (lihat `SERVICE_CATALOG.md` §6 Observability requirements Download Service).

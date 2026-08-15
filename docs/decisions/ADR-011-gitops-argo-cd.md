# ADR-011: GitOps dengan Argo CD

## Status

Accepted

## Context

Dengan target deployment ke Kubernetes (ADR-010) dan banyak service yang perlu dideploy secara konsisten dan dapat diaudit, dibutuhkan pendekatan deployment yang deklaratif, dapat direview (melalui Git), dan dapat di-rollback dengan mudah.

## Decision

CodeShop menargetkan pendekatan **GitOps menggunakan Argo CD**: state deployment Kubernetes (manifest/Helm values) didefinisikan di Git repository, dan Argo CD melakukan sinkronisasi otomatis antara state di Git dengan state aktual di cluster.

## Alternatives Considered

1. **Manual `kubectl apply` / imperative deployment** — ditolak sebagai target jangka panjang karena tidak memberikan audit trail yang jelas, rawan configuration drift antara apa yang seharusnya berjalan (menurut dokumentasi/niat tim) dan apa yang benar-benar berjalan di cluster.
2. **CI-driven push deployment** (GitHub Actions langsung menjalankan `helm upgrade` ke cluster) — merupakan pola valid dan lebih sederhana, tetapi ditolak sebagai pendekatan utama karena kredensial akses cluster harus dipegang oleh CI (memperluas attack surface) dan tidak ada rekonsiliasi otomatis berkelanjutan (drift detection) seperti pada model pull-based GitOps.
3. **GitOps dengan Argo CD (pull-based)** — **dipilih**, cluster secara aktif menyesuaikan diri dengan state yang dideklarasikan di Git, kredensial cluster tidak perlu dibagikan ke CI eksternal, drift terdeteksi otomatis.

## Consequences

**Positif**: seluruh perubahan deployment terlacak melalui Git history (siapa mengubah apa dan kapan), rollback menjadi semudah revert commit, drift antara state Git dan cluster terdeteksi otomatis oleh Argo CD.

**Negatif**: menambah komponen infrastruktur (Argo CD itu sendiri) yang perlu dikelola; memerlukan disiplin tim untuk tidak melakukan perubahan manual langsung ke cluster (`kubectl edit` dsb.) karena akan dianggap drift dan berpotensi di-revert otomatis oleh Argo CD.

## Security Implications

Model pull-based mengurangi kebutuhan menyimpan kredensial cluster produksi di sistem CI eksternal (GitHub Actions), karena Argo CD berjalan di dalam cluster dan yang perlu diamankan adalah akses ke Git repository (mis. branch protection, review wajib untuk perubahan manifest produksi).

## Operational Implications

CI (GitHub Actions) bertanggung jawab untuk build, test, container scanning (lihat Supply Chain security di `../architecture/ARCHITECTURE.md`), dan publish image serta update manifest/Helm values di Git; Argo CD bertanggung jawab murni untuk sinkronisasi ke cluster — pemisahan tanggung jawab ini didokumentasikan sebagai prinsip, detail pipeline merupakan Phase implementasi.

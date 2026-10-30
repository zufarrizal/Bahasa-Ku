# Analisis & Catatan Masalah Pengembangan Aplikasi Desktop Bahasa Ku

Dokumen ini mencatat evaluasi teknis, kendala arsitektur, dan temuan masalah yang dialami saat membangun aplikasi desktop CRUD Pengguna, Autentikasi, dan Role menggunakan Bahasa Ku, beserta **solusi tuntas yang telah diimplementasikan ke dalam core Bahasa Ku**.

---

## 1. Masalah Ketergantungan Server HTTP Lokal untuk SQLite

### Status: ✅ SELESAI DIPERBAIKI (Fitur IPC Native Binding)

### Solusi yang Diimplementasikan
1. Bahasa Ku kini menyediakan fungsi **`jendela.daftarkanFungsi(nama, fungsi)`** atau **`jendela.ikat(nama, fungsi)`**.
2. Di dalam jendela antarmuka WebView (HTML/JS), runtime Bahasa Ku secara otomatis menyuntikkan objek global:
   ```javascript
   const data = await window.ku.panggil("namaFungsi", ...argumen);
   ```
3. Pengembang **tidak perlu lagi membuat server HTTP REST API manual** hanya untuk melakukan query SQLite. Antarmuka UI dapat langsung memanggil query SQLite di runtime Bahasa Ku via `window.ku.panggil(...)`.

---

## 2. Masalah Perintah Kompilasi ke Biner Mandiri (`ku kompilasi`)

### Status: ✅ SELESAI DIPERBAIKI (Compiler Bundler & Standalone Exe)

### Solusi yang Diimplementasikan
1. Perintah `ku kompilasi berkas.ku -o Aplikasi.exe` kini menggunakan mekanisme **Binary Code Bundling & Runner Compiler Go**.
2. Kompilasi secara otomatis mengaitkan modul runtime Bahasa Ku (`github.com/zufarrizal/bahasa-ku`) dan menjalankan `go mod tidy` di folder sementara.
3. Seluruh modul bawaan (`basisdata`, `kripto`, `json`, `berkas`, `jendela`, `sistem`, dll.) kini **100% didukung saat dikompilasi menjadi biner `.exe` mandiri**.

---

## 3. Masalah Routing Server Web Bahasa Ku

### Status: ✅ SELESAI DIPERBAIKI (Route Precedence Scoring & Catch-All)

### Solusi yang Diimplementasikan
1. Evaluator server web (`pkg/evaluator/evaluator.go`) kini menggunakan sistem **Skor Prioritas Rute (*Route Precedence Scoring*)**:
   - Rute spesifik statis (misal: `/api/pengguna/aktif`) mendapat skor tertinggi (100 per segmen).
   - Rute parameter dinamis (misal: `/api/pengguna/:id`) mendapat skor parameter (10 per segmen).
   - Rute wildcard (misal: `/api/*`) mendapat skor dasar (5).
   - Rute root (`/`) ditangani secara presisi tanpa menimpa rute sub-path.
2. Urutan pemanggilan fungsi rute tidak lagi menyebabkan konflik atau tertimpa.

---

*Seluruh kendala teknis di atas telah diselesaikan dan diverifikasi dengan tes unit serta biner rilis.*

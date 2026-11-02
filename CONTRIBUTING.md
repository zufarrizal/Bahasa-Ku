# Panduan Kontribusi (Contributing Guide)

Terima kasih atas ketertarikan Anda untuk berkontribusi pada pengembangan **Bahasa Ku**! Proyek ini terbuka bagi siapa saja yang ingin membantu memajukan ekosistem bahasa pemrograman berbahasa Indonesia.

---

## 🛠️ Cara Memulai

### 1. Prasyarat Sistem
- **Go**: Versi 1.22 atau yang lebih baru.
- **Git**

### 2. Menjalankan Kode Secara Lokal
1. Lakukan *fork* dan *clone* repositori ini:
   ```bash
   git clone https://github.com/zufarrizal/Bahasa-Ku.git
   cd "Bahasa Ku"
   ```
2. Jalankan seluruh unit test untuk memastikan lingkungan siap:
   ```bash
   go test -count=1 ./pkg/... ./cmd/ku
   ```
3. Kompilasi biner `ku.exe`:
   ```bash
   go build -o ku.exe ./cmd/ku
   ```
4. Jalankan contoh program:
   ```bash
   .\ku.exe jalankan contoh\halo_dunia.ku
   ```

---

## 📋 Alur Kontribusi

1. Buat branch baru dari `main`:
   ```bash
   git checkout -b fitur/nama-fitur-anda
   ```
2. Tulis kode yang rapi, ringkas, dan backward-compatible.
3. Selalu sertakan **Unit Test** di folder `pkg/...` yang relevan jika menambahkan fitur baru atau memperbaiki bug.
4. Pastikan semua tes lulus:
   ```bash
   go test -count=1 ./pkg/... ./cmd/ku
   ```
5. Buat commit yang deskriptif dan buat Pull Request (PR) ke branch `main`.

---

## 💡 Konvensi Kode & Filosofi Bahasa Ku

- **Nuansa Go**: Selalu utamakan kesederhanaan, performa, konkurensi native, dan nol dependensi eksternal tak perlu (*zero runtime dependency*).
- **Istilah Bahasa Indonesia**: Gunakan istilah bahasa Indonesia yang baku, mudah dipahami, dan konsisten (misal: `fungsi`, `kembalikan`, `antarmuka`, `kanal`, `pilih`, `tunda`, `panik`, `pulih`).
- **Dokumentasi**: Jika menambahkan fitur atau modul baru, perbarui juga file `dokumentasi.html` dan `contoh/`.

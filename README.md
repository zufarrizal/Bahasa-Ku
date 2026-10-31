# Bahasa Ku

<p align="center">
  <img src="aset/logo.svg" alt="Logo Bahasa Ku" width="180" height="180" />
</p>

<p align="center">
  <b>Bahasa pemrograman edukatif bernuansa Go dengan sintaks murni berbahasa Indonesia.</b><br>
  <i>Cepat, Modern, Multiplatform, dan Ramah bagi Pemula.</i>
</p>

<p align="center">
  <a href="https://zufarrizal.github.io/Bahasa-Ku/" target="_blank">🌐 <b>Situs Resmi & Playground</b></a> &nbsp;|&nbsp;
  <a href="https://zufarrizal.github.io/Bahasa-Ku/dokumentasi.html" target="_blank">📖 <b>Dokumentasi Lengkap</b></a> &nbsp;|&nbsp;
  <a href="https://github.com/zufarrizal/Bahasa-Ku/releases/tag/v0.3.6">📦 <b>Unduh Biner v0.3.6</b></a>
</p>

---

## ⚡ Ringkasan Cepat

Bahasa Ku dirancang untuk mempermudah siapa saja memahami logika dan arsitektur komputasi modern tanpa hambatan bahasa Inggris.

```ku
paket utama

pakai "cetak"

fungsi utama() {
    cetak.tulis("Halo Dunia dari Bahasa Ku!")
}
```

> 📖 **Dokumentasi Kode Lengkap & Referensi API:**  
> Untuk panduan sintaksis, seluruh contoh kode, dan referensi pustaka standar (`cetak`, `waktu`, `arsip`, `pola`, `larik`, `peta`, `kompresi`, `jalur`, `enkoding`, `bita`, `acak`, `teks`, `konversi`, `kripto`, `berkas`, `sistem`, `web`, `jendela`, `desktop`, `basisdata`, `soket`, `jadwal`, `gambar`, `cli`, `pekerja`, `singgahan`, `peristiwa`, `konfigurasi`, `debug`, `uji`), silakan baca langsung di:  
> 👉 **[Halaman Dokumentasi Web Resmi](https://zufarrizal.github.io/Bahasa-Ku/dokumentasi.html)** atau buka file lokal [`dokumentasi.html`](dokumentasi.html).

---

## 📦 Pemasangan Instan (1 Baris Terminal)

### 🪟 Windows (PowerShell):
```powershell
irm https://raw.githubusercontent.com/zufarrizal/Bahasa-Ku/main/pasang.ps1 | iex
```

### 🐧 Linux & 🍎 macOS (Terminal):
```bash
curl -fsSL https://raw.githubusercontent.com/zufarrizal/Bahasa-Ku/main/pasang.sh | bash
```

### Alternatif Unduh Manual:
- Biner siap pakai tersedia di halaman **[Rilis Resmi GitHub](https://github.com/zufarrizal/Bahasa-Ku/releases/tag/v0.3.6)** (`ku-windows-amd64.exe`, `ku-linux-amd64`, `ku-darwin-arm64`).
- Atau pasang via Go: `go install github.com/zufarrizal/bahasa-ku/cmd/ku@latest`.

---

## 🛠️ Perintah Dasar CLI `ku`

```bash
# Terminal Interaktif (REPL)
ku

# Menjalankan berkas kode (otomatis mendeteksi 'utama.ku' jika tanpa nama file)
ku jalankan contoh/halo.ku
ku jalankan              # Compact run: otomatis jalankan utama.ku di folder aktif
ku j                     # Pintasan super cepat 1 huruf (alias 'ku jalankan')
ku j -d                  # Menjalankan dengan mode debug & profil memori/waktu

# Menjalankan aplikasi dengan Hot Reload (otomatis restart saat kode disimpan)
ku awasi contoh/halo.ku
ku awasi                 # Compact watch: awasi seluruh folder aktif
ku a                     # Pintasan 1 huruf (alias 'ku awasi')

# Menjalankan suite pengujian unit test
ku uji
ku u                     # Pintasan 1 huruf (alias 'ku uji')

# Menjalankan tolok ukur performa (Benchmark dengan metrik ns/op, B/op, allocs/op)
ku tolakUkur contoh/matematika_tolakukur.ku

# Merapikan format kode secara otomatis (mirip gofmt)
ku rapi -w contoh/halo.ku

# Memeriksa kerapian format kode (untuk CI/CD)
ku rapi -c .

# Kompilasi mandiri ke biner executable (.exe / Linux ELF / Darwin Mach-O)
ku kompilasi contoh/halo.ku -o HaloApp.exe
ku kompilasi             # Compact compile: otomatis kompilasi utama.ku
ku k                     # Pintasan 1 huruf (alias 'ku kompilasi')
ku kompilasi -gui contoh/aplikasi_desktop.ku -o DesktopApp.exe

# Mengimpor modul langsung dari URL Remote (HTTPS dengan caching lokal otomatis)
# Di dalam kode: pakai "https://raw.githubusercontent.com/.../modul.ku"

# Membuat kerangka kerja proyek baru (scaffolding rapi dengan ku.mod, .env, .gitignore)
ku inisiasi web portal-berita
ku inisiasi desktop aplikasi-kasir
ku inisiasi cli perkakas-data

# Informasi versi & periksa pembaruan rilis
ku versi
ku versi --cek

# Memeriksa dan memperbarui ke biner Bahasa Ku terbaru
ku perbarui
```

---

## 🎨 Dukungan Visual Studio Code

Ekstensi resmi Visual Studio Code menyediakan pewarnaan sintaks (*syntax highlighting*) dan ikon berkas `.ku`.
- Unduh paket **`bahasa-ku-vscode-0.2.0.vsix`** dari [Rilis GitHub](https://github.com/zufarrizal/Bahasa-Ku/releases/tag/v0.3.6).
- Buka VS Code > Menu Extensions (`Ctrl+Shift+X`) > Klik `...` > **Install from VSIX...**.
- Panduan lengkap ekstensi tersedia di [ekstensi-vscode/README.md](ekstensi-vscode/README.md).

---

## 📖 Ringkasan Padanan Kata Kunci (Go vs Ku)

| Go | Ku | Deskripsi |
| :--- | :--- | :--- |
| `package main` | `paket utama` | Deklarasi paket aplikasi |
| `import "fmt"` | `pakai "cetak"` | Mengimpor pustaka modul bawaan/lokal |
| `func main()` | `fungsi utama()` | Deklarasi fungsi utama |
| `type User struct` | `tipe Pengguna struktur` | Deklarasi data terstruktur (*struct*) |
| `var x = 10` | `var x = 10` | Deklarasi variabel |
| `x := 10` | `x := 10` | Inisialisasi cepat |
| `const PI = 3.14` | `tetap PI = 3.14` | Deklarasi konstanta |
| `if` / `else` | `jika` / `lainnya` | Percabangan kondisi |
| `switch` / `case` / `default` | `pilih` / `kasus` / `bawaan` | Percabangan seleksi |
| `for i, v := range arr` | `untuk i, v dalam arr` | Perulangan koleksi (range loop) |
| `break` / `continue` | `henti` / `lanjut` | Kontrol perulangan |
| `defer` | `tunda` | Eksekusi LIFO di akhir fungsi |
| `go fn()` | `ku fn()` | Konkurensi Ku-routine di latar belakang |
| `return` | `kembalikan` | Nilai balik fungsi (mendukung multi-return) |
| `len(arr)` | `panjang(arr)` | Menghitung panjang array/string/map |
| `append(arr, item)` | `tambah(arr, item)` | Menambah elemen ke larik |
| `delete(map, key)` | `hapus(map, key)` | Menghapus kunci dari map |
| `true` / `false` | `benar` / `salah` | Nilai boolean |
| `nil` | `kosong` / `nihil` | Nilai kosong |
| `obj?.prop` | `obj?.prop` | Akses properti aman (*Optional Chaining*) |
| `val ?? default` | `val ?? bawaan` | Nilai bawaan jika kosong (*Nullish Coalescing*) |

---

## 📂 Struktur Proyek

```
Bahasa Ku/
├── cmd/
│   ├── ku/                   # CLI Native Bahasa Ku (Go Native)
│   └── wasm/                 # Engine WebAssembly untuk browser playground
├── pkg/
│   ├── ast/                  # Abstract Syntax Tree
│   ├── evaluator/            # Runtime evaluator & Pustaka bawaan lengkap
│   ├── formatter/            # Pengatur format kode otomatis (ku rapi)
│   ├── lexer/                # Tokenizer scanner
│   ├── parser/               # Parser tata bahasa Ku
│   ├── repl/                 # Read-Eval-Print Loop terminal
│   ├── token/                # Definisi kata kunci & token bahasa
│   └── transpiler/           # Generator kode Go murni (.go)
├── contoh/                   # Koleksi contoh program mandiri (.ku)
├── ekstensi-vscode/          # Source kode ekstensi resmi VS Code
├── index.html                # Web beranda & WebAssembly playground
├── dokumentasi.html          # Dokumentasi web lengkap & referensi API
├── go.mod                    # Konfigurasi modul Go resmi
└── README.md                 # Informasi ringkas repositori
```

---

## 🧪 Pengujian Unit Test

```bash
go test ./...
```

---

## 📄 Lisensi & Pernyataan Etika (Ethical Notice)

Didistribusikan di bawah lisensi terbuka [MIT License](LICENSE).

### Pengakuan & Penghormatan Hak Cipta
- **Sumber Inspirasi:** Bahasa Ku dirancang untuk tujuan edukasi dan aksesibilitas, yang **terinspirasi oleh arsitektur dan kesederhanaan bahasa Go (The Go Authors / Google LLC)**.
- **Batasan Afiliasi:** Bahasa Ku adalah proyek independen dan tidak terafiliasi secara resmi dengan Google LLC atau The Go Authors.
- **Tujuan Pengembangan:** Sebagai sarana edukasi untuk membantu penutur Bahasa Indonesia mempelajari konsep komputasi dan arsitektur sistem modern.

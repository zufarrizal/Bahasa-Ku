# Bahasa Ku

<p align="center">
  <img src="aset/logo.svg" alt="Logo Bahasa Ku" width="180" height="180" />
</p>

<p align="center">
  <b>Bahasa pemrograman bernuansa Go dengan sintaks murni berbahasa Indonesia.</b><br>
  <i>Cepat, Modern, Multiplatform, dan Mudah Dipahami.</i>
</p>

---

## 🚀 Fitur Utama

- **Native Binary Mandiri:** Dijalankan langsung melalui file biner tanpa perlu runtime tambahan.
- **Kompilasi Langsung ke Biner (`ku kompilasi`):** Mengubah file `.ku` langsung menjadi biner `.exe` mandiri.
- **Generator Proyek Otomatis (`ku inisiasi`):** Menyiapkan kerangka kerja proyek baru (`web`, `desktop`, `cli`) secara instan.
- **Sintaks Berbahasa Indonesia:** Kata kunci natural seperti `paket`, `fungsi`, `var`, `jika`, `lainnya`, `untuk`, `pilih`, `kembalikan`, dan `tunda`.
- **Dukungan Server Web HTTP & REST API (`web`):** Siap digunakan untuk membuat web server dan API JSON.
- **Dukungan Klien Jaringan HTTP (`jaringan`):** Melakukan panggilan HTTP GET / POST JSON ke API eksternal secara native.
- **Dukungan Aplikasi Desktop GUI (`jendela`):** Membuka dialog native dan jendela aplikasi desktop modern tanpa Electron.
- **Dukungan I/O Berkas & Sistem (`berkas` & `sistem`):** Membaca/menulis file lokal dan mengakses environment OS.
- **Dukungan Database MySQL (`basisdata`):** Terhubung ke database MySQL secara native dengan operasi `kueri` dan `eksekusi`.
- **Transpiler ke Go Murni:** Menerjemahkan file `.ku` menjadi kode `.go` standar untuk dikompilasi ke file biner native.

---

## 📦 Cara Memasang & Menggunakan

### ⚡ Cara Instan 1 Baris (Sangat Direkomendasikan)
Pengguna **tidak perlu mengunduh file satu per satu, tidak perlu mengatur PATH secara manual, dan tidak perlu memasang ekstensi VS Code secara manual**. Cukup jalankan perintah 1 baris berikut di terminal:

#### 🪟 Windows (PowerShell):
Buka **PowerShell**, salin dan tempel perintah ini lalu tekan Enter:
```powershell
irm https://raw.githubusercontent.com/zufarrizal/Bahasa-Ku/main/pasang.ps1 | iex
```
> **Apa yang dilakukan skrip ini secara otomatis?**
> 1. Mengunduh `ku.exe` versi resmi terbaru.
> 2. Mendaftarkan `ku` ke variabel lingkungan `PATH` Windows Anda secara otomatis (langsung bisa dipanggil di folder mana pun).
> 3. Otomatis memasang ekstensi VS Code Bahasa Ku (*syntax highlighting* & ikon).

#### 🐧 Linux & 🍎 macOS (Terminal):
```bash
curl -fsSL https://raw.githubusercontent.com/zufarrizal/Bahasa-Ku/main/pasang.sh | bash
```

---

### Alternatif Pemasangan Manual:

#### 1. Unduh Binary Siap Pakai (Windows, Linux, macOS)
Kunjungi halaman **[Rilis Resmi GitHub](https://github.com/zufarrizal/Bahasa-Ku/releases/tag/v0.2.0)**:
- **Windows:** `ku-windows-amd64.exe`
- **Linux:** `ku-linux-amd64`
- **macOS:** `ku-darwin-arm64`
- **Ekstensi VS Code:** `bahasa-ku-vscode-0.1.0.vsix`

#### 2. Atau Pasang Menggunakan Go (Jika Memiliki Go di Komputer)
```bash
go install github.com/zufarrizal/bahasa-ku/cmd/ku@latest
```

---

## 🛠️ Menjalankan Program dengan CLI `ku`

```bash
# Membuka REPL Terminal Interaktif
ku

# Menjalankan program berkas .ku
ku jalankan contoh/halo.ku
ku jalankan contoh/server_web.ku
ku jalankan contoh/jaringan.ku
ku jalankan contoh/aplikasi_desktop.ku
ku jalankan contoh/berkas_sistem.ku

# Mengompilasi langsung berkas .ku menjadi biner mandiri (.exe)
ku kompilasi contoh/halo.ku -o ProgramHalo.exe

# Membuat kerangka kerja proyek baru secara instan
ku inisiasi web my-api
ku inisiasi desktop my-gui
ku inisiasi cli my-tool

# Mengonversi berkas .ku ke kode Go murni (.go)
ku buat contoh/tunda.ku hasil.go

# Memeriksa versi
ku versi
```

---

## 🎨 Dukungan Editor VS Code (Syntax Highlighting)

Agar penulisan kode `.ku` memiliki warna sintaksis (*syntax highlighting*) dan ikon resmi di Visual Studio Code:
1. Unduh file paket ekstensi **`bahasa-ku-vscode-0.1.0.vsix`** dari halaman **[Rilis GitHub](https://github.com/zufarrizal/Bahasa-Ku/releases/tag/v0.2.0)**.
2. Buka Visual Studio Code.
3. Masuk ke menu **Extensions** (`Ctrl+Shift+X`), klik tanda titik tiga (`...`) di sudut kanan atas panel ekstensi, lalu pilih **"Install from VSIX..."**.
4. Pilih file `bahasa-ku-vscode-0.1.0.vsix`. Selesai! Seluruh kode `.ku` akan langsung berwarna rapi dan memiliki ikon resmi.

---

## 📖 Padanan Kata Kunci (Go vs Ku)

| Go | Ku | Deskripsi |
| :--- | :--- | :--- |
| `package main` | `paket utama` | Deklarasi paket aplikasi |
| `import "fmt"` | `pakai "cetak"` | Mengimpor modul pustaka |
| `func main()` | `fungsi utama()` | Deklarasi fungsi utama / fungsi kustom |
| `type User struct` | `tipe Pengguna struktur` | Deklarasi data terstruktur (*struct*) |
| `var x = 10` | `var x = 10` | Deklarasi variabel |
| `x := 10` | `x := 10` | Deklarasi & inisialisasi ringkas |
| `const PI = 3.14` | `tetap PI = 3.14` | Deklarasi konstanta |
| `if` / `else` | `jika` / `lainnya` | Percabangan kondisi |
| `switch` / `case` / `default` | `pilih` / `kasus` / `bawaan` | Percabangan seleksi |
| `for i := 0; i < 5; i++` | `untuk i := 0; i < 5; i++` | Perulangan |
| `defer` | `tunda` | Eksekusi LIFO di akhir fungsi |
| `return` | `kembalikan` | Mengembalikan nilai fungsi |
| `len(arr)` | `panjang(arr)` | Menghitung panjang array/string/map |
| `append(arr, item)` | `tambah(arr, item)` | Menambah elemen ke array/slice |
| `true` / `false` | `benar` / `salah` | Nilai boolean |
| `nil` | `kosong` / `nihil` | Nilai null/kosong |

---

## 📝 Contoh-Contoh Program

### 1. Halo Dunia (`contoh/halo.ku`)
```ku
paket utama

pakai "cetak"

fungsi utama() {
    cetak.tulis("Halo Dunia dari Bahasa Ku!")
    cetak.tulis("Selamat datang di era pemrograman berbahasa Indonesia.")
}
```

### 2. Logika Percabangan & Rekursi Faktorial (`contoh/logika.ku`)
```ku
paket utama

pakai "cetak"

fungsi hitungFaktorial(n) {
    jika n <= 1 {
        kembalikan 1
    }
    kembalikan n * hitungFaktorial(n - 1)
}

fungsi periksaGenapGanjil(angka) {
    jika angka % 2 == 0 {
        kembalikan "Genap"
    } lainnya {
        kembalikan "Ganjil"
    }
}

fungsi utama() {
    cetak.tulis("=== Uji Coba Logika Bahasa Ku ===")

    untuk i := 1; i <= 5; i++ {
        cetak.tulis("Angka:", i, "->", periksaGenapGanjil(i))
    }

    var hasil = hitungFaktorial(5)
    cetak.tulis("Faktorial dari 5 adalah:", hasil)
}
```

### 3. Server Web HTTP & REST API JSON (`contoh/server_web.ku`)
```ku
paket utama

pakai "cetak"
pakai "web"

fungsi sapaHalaman(permintaan) {
    kembalikan "<h1>Selamat Datang di Web Bahasa Ku!</h1>"
}

fungsi ambilDataJson(permintaan) {
    kembalikan {
        "status": "sukses",
        "pesan": "API REST Bahasa Ku Berfungsi!"
    }
}

fungsi utama() {
    web.tangani("/", sapaHalaman)
    web.tangani("/api", ambilDataJson)

    cetak.tulis("Server web aktif di http://localhost:8080")
    web.dengarkan(":8080")
}
```

### 4. Aplikasi Desktop GUI Windows (`contoh/aplikasi_desktop.ku`)
```ku
paket utama

pakai "cetak"
pakai "jendela"

fungsi utama() {
    cetak.tulis("Membuka aplikasi desktop Bahasa Ku...")

    jendela.buka({
        "judul": "Aplikasi Desktop Bahasa Ku",
        "konten": "<h1>Halo dari Desktop Bahasa Ku! 🖥️</h1><p>Aplikasi GUI Windows tanpa Electron.</p>"
    })
}
```

### 5. Fitur Tunda / Defer Khas Go (`contoh/tunda.ku`)
```ku
paket utama

pakai "cetak"

fungsi ujiTunda() {
    cetak.tulis("1. Fungsi ujiTunda dimulai")
    tunda cetak.tulis("4. Pesan tunda ke-1 (dieksekusi terakhir)")
    tunda cetak.tulis("3. Pesan tunda ke-2 (LIFO: dieksekusi sebelum tunda ke-1)")
    cetak.tulis("2. Fungsi ujiTunda selesai dieksekusi")
}

fungsi utama() {
    ujiTunda()
}
```

---

## 📂 Struktur Proyek

```
Bahasa Ku/
├── cmd/
│   └── ku/
│       └── main.go           # CLI Native Ku
├── pkg/
│   ├── ast/
│   │   └── ast.go            # Abstract Syntax Tree
│   ├── evaluator/
│   │   ├── evaluator.go      # Runtime & Pustaka Bawaan (cetak, web, jendela, basisdata)
│   │   ├── evaluator_test.go # Unit Test Go
│   │   ├── jendela_windows.go# Native GUI Windows
│   │   └── jendela_other.go  # Native GUI Non-Windows
│   ├── lexer/
│   │   └── lexer.go          # Tokenizer Scanner
│   ├── parser/
│   │   └── parser.go         # Parser Bahasa Ku
│   ├── token/
│   │   └── token.go          # Definisi Kata Kunci & Token
│   └── transpiler/
│       └── transpiler.go     # Generator Kode Go Murni (.go)
├── contoh/
│   ├── aplikasi_desktop.ku   # Contoh GUI Desktop Native
│   ├── server_web.ku         # Contoh Server HTTP & REST API
│   ├── fitur_lanjutan.ku     # Contoh Array, Map, Struct, Switch
│   ├── halo.ku               # Contoh Halo Dunia
│   ├── logika.ku             # Contoh Rekursi Faktorial
│   └── tunda.ku              # Contoh Defer Khas Go
├── go.mod                    # Konfigurasi Modul Go Resmi
└── README.md                 # Dokumentasi Lengkap
```

---

## 🧪 Pengujian Unit Test

```bash
go test ./...
```

---

## 📄 Lisensi

Didistribusikan di bawah lisensi [MIT](LICENSE).

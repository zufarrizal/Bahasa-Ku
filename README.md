# Bahasa Ku

Bahasa pemrograman bernuansa Go (*Go-flavored*) dengan sintaks murni berbahasa Indonesia, dirancang agar ramah dan mudah dipelajari oleh penutur bahasa Indonesia.

Compiler dan runtime **Bahasa Ku** dibangun secara native menggunakan **Go Murni** sehingga menghasilkan file executable mandiri (`ku.exe` di Windows, `ku` di Linux/macOS) tanpa dependensi eksternal.

---

## 🚀 Fitur Utama

- **Native Binary Mandiri:** Dijalankan langsung melalui file biner tanpa perlu runtime tambahan.
- **Sintaks Berbahasa Indonesia:** Kata kunci natural seperti `paket`, `fungsi`, `var`, `jika`, `lainnya`, `untuk`, `pilih`, `kembalikan`, dan `tunda`.
- **Dukungan Server Web HTTP & REST API (`web`):** Siap digunakan untuk membuat web server dan API JSON.
- **Dukungan Aplikasi Desktop GUI (`jendela`):** Membuka dialog native dan jendela aplikasi desktop modern tanpa Electron.
- **Dukungan Database MySQL (`basisdata`):** Terhubung ke database MySQL secara native dengan operasi `kueri` dan `eksekusi`.
- **Transpiler ke Go Murni:** Menerjemahkan file `.ku` menjadi kode `.go` standar untuk dikompilasi ke file biner native.

---

## 📦 Cara Memasang & Menggunakan

Pengguna **TIDAK PERLU** memasang Go jika hanya ingin menjalankan Bahasa Ku. Anda cukup mengunduh file biner siap pakai:

### 1. Unduh Binary Siap Pakai (Windows, Linux, macOS)
Kunjungi halaman **[Rilis Resmi GitHub](https://github.com/zufarrizal/Bahasa-Ku/releases/tag/v0.2.0)**, lalu unduh sesuai sistem operasi Anda:
- **Windows (x64):** Unduh `ku-windows-amd64.exe` (dapat diganti nama menjadi `ku.exe`).
- **Linux (x64):** Unduh `ku-linux-amd64` (jalankan `chmod +x ku-linux-amd64`).
- **macOS (Apple Silicon M1/M2/M3):** Unduh `ku-darwin-arm64`.

### 2. Atau Pasang Menggunakan Go (Jika Memiliki Go di Komputer)
```bash
go install github.com/zufarrizal/bahasa-ku/cmd/ku@latest
```

---

## 🛠️ Menjalankan Program dengan CLI `ku`

```bash
# Menjalankan program berkas .ku
ku jalankan contoh/halo.ku
ku jalankan contoh/logika.ku
ku jalankan contoh/server_web.ku
ku jalankan contoh/aplikasi_desktop.ku

# Mengonversi berkas .ku ke kode Go murni (.go)
ku buat contoh/tunda.ku hasil.go

# Memeriksa versi
ku versi
```

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

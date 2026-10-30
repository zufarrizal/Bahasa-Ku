# Bahasa Ku

Bahasa pemrograman bernuansa Go (*Go-flavored*) dengan sintaks murni berbahasa Indonesia, dirancang agar ramah dan mudah dipelajari oleh penutur bahasa Indonesia.

Engine compiler & runtime **Ku** dibangun langsung menggunakan **Go Murni** (menghasilkan binary mandiri `ku.exe`), serta tetap menyediakan dukungan dual engine (Go & Node.js).

---

## 🚀 Fitur Utama

- **Native Binary (`ku.exe`):** Berdiri sendiri tanpa dependensi runtime eksternal.
- **Sintaks Berbahasa Indonesia:** Mengganti kata kunci bahasa Inggris dengan kosakata alami bahasa Indonesia.
- **Filosofi Mirip Go:** Deklarasi fungsi `fungsi utama()`, deklarasi variabel `var` atau `:=`, modul `cetak.tulis(...)` (*fmt.Println*), data struktur `tipe ... struktur` (*struct*), serta kontrol `tunda` (*defer* LIFO).
- **Transpiler ke Go Murni:** Dapat langsung mengonversi file `.ku` ke `.go` asli.

---

## 🛠️ Cara Menjalankan

### Menggunakan Native Binary (`ku.exe`)

1. **Build compiler:**
```bash
go build -o ku.exe ./cmd/ku
```

2. **Cek Versi:**
```bash
.\ku.exe versi
```

3. **Menjalankan Program:**
```bash
.\ku.exe jalankan contoh/halo.ku
.\ku.exe jalankan contoh/logika.ku
.\ku.exe jalankan contoh/fitur_lanjutan.ku
.\ku.exe jalankan contoh/tunda.ku
```

4. **Kompilasi / Transpilasi ke Go:**
```bash
.\ku.exe buat contoh/tunda.ku hasil.go
```

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
| `switch` / `case` / `default` | `pilih` / `kasus` / `bawaan` | Percabangan banyak kondisi |
| `for i := 0; i < 5; i++` | `untuk i := 0; i < 5; i++` | Perulangan |
| `return` | `kembalikan` | Mengembalikan nilai fungsi |
| `len(arr)` | `panjang(arr)` | Menghitung panjang array/string/map |
| `append(arr, item)` | `tambah(arr, item)` | Menambah elemen ke array/slice |
| `true` / `false` | `benar` / `salah` | Nilai boolean |
| `nil` | `kosong` / `nihil` | Nilai null/kosong |

---

## 🛠️ Cara Menjalankan

### 1. Cek Versi
```bash
node bin/ku.js versi
```

### 2. Menjalankan Berkas Kode Ku (Interpreter)
```bash
node bin/ku.js jalankan contoh/halo.ku
node bin/ku.js jalankan contoh/logika.ku
node bin/ku.js jalankan contoh/fitur_lanjutan.ku
node bin/ku.js jalankan contoh/tunda.ku
```

### 3. Mengompilasi Kode Ku ke Go Murni (.go)
```bash
# Menghasilkan berkas .go
node bin/ku.js buat contoh/tunda.ku tunda.go
```

### 4. Menjalankan Pengujian Otomatis
```bash
node test/run_tests.js
```

---

## 📝 Contoh Program

### 1. Halo Dunia (`contoh/halo.ku`)
```ku
paket utama

pakai "cetak"

fungsi utama() {
    cetak.tulis("Halo Dunia dari Bahasa Ku!")
    cetak.tulis("Selamat datang di era pemrograman berbahasa Indonesia.")
}
```

### 2. Logika & Faktorial (`contoh/logika.ku`)
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

---

## 📂 Struktur Proyek

```
Bahasa Ku/
├── bin/
│   └── ku.js            # Antarmuka CLI 'ku jalankan'
├── src/
│   ├── ast.js           # Struktur pohon sintaks (AST)
│   ├── evaluator.js     # Interpreter & modul bawaan (cetak)
│   ├── index.js         # Entry point pustaka compiler
│   ├── lexer.js         # Pemindai teks & token
│   ├── lingkungan.js    # Pengelola variabel (Environment / Scope)
│   └── token.js         # Daftar tipe token & kata kunci
├── contoh/
│   ├── halo.ku          # Contoh Halo Dunia
│   └── logika.ku        # Contoh logika percabangan, fungsi, perulangan
├── test/
│   └── run_tests.js     # Test suite otomatis
├── README.md            # Dokumentasi lengkap
└── package.json
```

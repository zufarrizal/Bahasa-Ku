# Bahasa Ku

<p align="center">
  <img src="aset/logo.svg" alt="Logo Bahasa Ku" width="180" height="180" />
</p>

<p align="center">
  <b>Bahasa pemrograman edukatif bernuansa Go dengan sintaks murni berbahasa Indonesia.</b><br>
  <i>Cepat, Modern, Multiplatform, dan Ramah bagi Pemula.</i>
</p>

<p align="center">
  <a href="https://zufarrizal.github.io/Bahasa-Ku/" target="_blank">🌐 <b>Coba Online Playground</b></a> &nbsp;|&nbsp;
  <a href="https://github.com/zufarrizal/Bahasa-Ku/releases/tag/v0.2.0">📦 <b>Unduh Biner v0.2.0</b></a>
</p>

---

## 🚀 Fitur Utama

- **Native Binary Mandiri:** Dijalankan langsung melalui file biner tanpa perlu runtime tambahan.
- **Kompilasi Langsung ke Biner (`ku kompilasi`):** Mengubah file `.ku` langsung menjadi biner `.exe` mandiri.
- **Formatter Kode Otomatis (`ku rapi`):** Merapikan format spasi, tab, dan kurung kurawal secara otomatis (mirip `gofmt`).
- **Generator Proyek Otomatis (`ku inisiasi`):** Menyiapkan kerangka kerja proyek baru (`web`, `desktop`, `cli`) secara instan.
- **Sintaks Berbahasa Indonesia:** Kata kunci natural seperti `paket`, `fungsi`, `var`, `jika`, `lainnya`, `untuk`, `pilih`, `kembalikan`, dan `tunda`.
- **Kontrol Perulangan Lengkap:** Didukung kata kunci `henti` (*break*) dan `lanjut` (*continue*).
- **Interaksi Masukan Pengguna (`masukan`):** Mengambil input keyboard dari terminal melalui `masukan.tanya()` dan `masukan.baca()`.
- **Manipulasi Koleksi Lengkap:** Fungsi bawaan `panjang()`, `tambah()`, dan `hapus()` untuk Array dan Map.
- **Dukungan Server Web HTTP & REST API (`web`):** Siap digunakan untuk membuat web server dan API JSON.
- **Dukungan Klien Jaringan HTTP (`jaringan`):** Melakukan panggilan HTTP GET / POST JSON ke API eksternal secara native.
- **Dukungan Aplikasi Desktop GUI (`jendela`):** Membuka dialog native dan jendela aplikasi desktop modern tanpa Electron.
- **Perulangan Modern (`untuk ... dalam`):** Iterasi array dan map secara alami (ekivalen `for ... range` di Go).
- **Dukungan Manipulasi Teks (`teks`):** Pemisahan (`pisah`), penggabungan (`gabung`), kapitalisasi, pemangkasan spasi (`pangkas`), dan pencarian string (`mengandung`).
- **Dukungan Konversi Data (`konversi`):** Mengubah teks ke angka numerik (`keAngka`), pecahan (`kePecahan`), teks (`keTeks`), dan inspeksi tipe (`tipe`).
- **Dukungan Keamanan & Enkripsi (`kripto`):** SHA-256, MD5, Base64 encode/decode, dan pembuatan token acak aman.
- **Dukungan Kalkulasi Numerik (`matematika`):** Akar kuadrat, pemangkatan, pembulatan, angka acak, dan nilai PI.
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
ku jalankan contoh/kripto_matematika.ku
ku jalankan contoh/aplikasi_desktop.ku
ku jalankan contoh/berkas_sistem.ku

# Merapikan format berkas kode secara otomatis
ku rapi -w contoh/halo.ku

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
| `for i, v := range arr` | `untuk i, v dalam arr` | Perulangan koleksi (range loop) |
| `break` | `henti` | Keluar dari perulangan |
| `continue` | `lanjut` | Lewati ke iterasi berikutnya |
| `defer` | `tunda` | Eksekusi LIFO di akhir fungsi |
| `go fn()` | `ku fn()` | Konkurensi Ku-routine di latar belakang |
| `return` | `kembalikan` | Mengembalikan nilai fungsi |
| `len(arr)` | `panjang(arr)` | Menghitung panjang array/string/map |
| `append(arr, item)` | `tambah(arr, item)` | Menambah elemen ke array/slice |
| `delete(map, key)` | `hapus(map, key)` | Menghapus kunci dari map |
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

### 6. Pengembalian Multi-Nilai, Blank Identifier, Bitwise, & Operator Gabungan (`contoh/dasar_multi_return.ku`)
```ku
paket utama

pakai "cetak"

// Mengembalikan dua nilai: hasil dan galat (idiom error Go)
fungsi bagi(a, b) {
    jika b == 0 {
        kembalikan 0, "tidak dapat membagi dengan nol!"
    }
    kembalikan a / b, nihil
}

fungsi utama() {
    // 1. Konstanta
    tetap PI = 3.14159

    // 2. Destrukturisasi pengembalian multi-nilai
    var hasil, galat = bagi(20, 4)
    jika galat == nihil {
        cetak.tulis("Hasil bagi:", hasil)
    }

    // 3. Mengabaikan nilai dengan blank identifier (_)
    _, err := bagi(10, 0)
    jika err != nihil {
        cetak.tulis("Galat tertangkap:", err)
    }

    // 4. Operator Gabungan & Slicing
    daftar := [10, 20, 30, 40, 50]
    potongan := daftar[1:4] // [20, 30, 40]

    // 5. Operator Bitwise (&, |, ^, <<, >>)
    cetak.tulis("Bitwise 12 & 10 =", 12 & 10) // 8
    cetak.tulis("Geser Kiri 12 << 2 =", 12 << 2) // 48
}
```

### 7. Method Receiver Struct, Fungsi Anonim, & Argumen Variadik (`contoh/dasar_method_variadic.ku`)
```ku
paket utama

pakai "cetak"

tipe Persegi struktur {
    panjang angka
    lebar   angka
}

// Method Receiver khas Go terikat pada tipe struct
fungsi (p Persegi) hitungLuas() {
    kembalikan p.panjang * p.lebar
}

// Argumen variadik (...args) menerima jumlah argumen dinamis
fungsi jumlahkanSemua(...daftarAngka) {
    var total = 0
    untuk _, x dalam daftarAngka {
        total += x
    }
    kembalikan total
}

fungsi utama() {
    // 1. Inisialisasi struct & pemanggilan method
    var kotak = Persegi({ panjang: 10, lebar: 5 })
    cetak.tulis("Luas Persegi:", kotak.hitungLuas()) // 50

    // 2. Argumen variadik tak terhingga
    cetak.tulis("Total:", jumlahkanSemua(1, 2, 3, 4, 5)) // 15

    // 3. Fungsi Anonim (Closure / First-Class Citizen)
    var pengali = fungsi(x) {
        kembalikan x * 3
    }
    cetak.tulis("Hasil Pengali:", pengali(7)) // 21
}
```

### 8. Panik & Pulih (panic & recover) serta Pencacah Otomatis (`contoh/dasar_panik_cacah.ku`)
```ku
paket utama

pakai "cetak"

// 1. Pencacah Konstanta Otomatis (cacah / iota)
tetap STATUS_MENUNGGU = cacah // 0
tetap STATUS_DIPROSES = cacah // 1
tetap STATUS_SELESAI  = cacah // 2

// 2. Proteksi Panik & Pulih
fungsi prosesTransaksi(jumlah) {
    tunda fungsi() {
        var err = pulih()
        jika err != nihil {
            cetak.tulis("🛡️ [PULIH] Program terselamatkan dari crash:", err)
        }
    }()

    jika jumlah <= 0 {
        panik("jumlah transaksi tidak valid!")
    }
    cetak.tulis("Transaksi berhasil diproses!")
}

fungsi utama() {
    // Pengujian otomatis internal
    uji.sama(STATUS_MENUNGGU, 0)
    uji.sama(STATUS_SELESAI, 2)

    prosesTransaksi(10000)
    prosesTransaksi(-500) // Memantik panik & tertangkap oleh pulih()
}
```

### 9. Konkurensi Ku-Routine (Kata Kunci `ku`) (`contoh/dasar_konkurensi.ku`)
```ku
paket utama

pakai "cetak"
pakai "sistem"

fungsi unduhAset(nama) {
    cetak.tulis("Mulai mengunduh:", nama)
    sistem.tidur(1)
    cetak.tulis("Selesai unduh:", nama)
}

fungsi utama() {
    // Menjalankan tugas di latar belakang (paralel) menggunakan kata kunci 'ku'
    ku unduhAset("gambar.png")
    ku unduhAset("audio.mp3")

    // Fungsi anonim di latar belakang
    ku fungsi() {
        cetak.tulis("Ku-routine anonim aktif!")
    }()

    sistem.tidur(2)
}
```

### 10. Kanal Konkurensi (`kanal`), Null Coalescing (`??`), & Berkas Lanjutan (`contoh/dasar_kanal_coalesce.ku`)
```ku
paket utama

pakai "cetak"
pakai "kanal"
pakai "sistem"
pakai "berkas"

fungsi utama() {
    // 1. Saluran komunikasi konkurensi (channel Go)
    var pipa = kanal.buat(1)
    ku fungsi() {
        sistem.tidur(1)
        pipa.kirim("Pesan rahasia dari Ku-routine!")
    }()

    var pesan = pipa.terima()
    cetak.tulis("Diterima:", pesan)

    // 2. Operator Fallback Null Coalescing (??)
    var port = sistem.lingkungan("PORT") ?? "3000"
    cetak.tulis("Port aktif:", port)

    // 3. Operasi berkas lanjutan
    berkas.buatDirektori("arsip/laporan")
    var isiFolder = berkas.daftar("arsip")
    cetak.tulis("Daftar arsip:", isiFolder)
}
```

### 11. Impor Berkas Lokal, Modul JSON, & Modul Larik Fungsional (`contoh/dasar_multiberkas_json.ku`)
```ku
paket utama

pakai "cetak"
pakai "json"
pakai "larik"
pakai "./contoh/pembantu.ku"

fungsi utama() {
    // 1. Memanggil fungsi dari berkas .ku lokal yang diimpor
    var hasil = kalikanTiga(10) // dari pembantu.ku -> 30

    // 2. Serialisasi & Deserialisasi JSON
    var data = {"nama": "Ahmad", "skor": 100}
    var jsonTeks = json.kodekan(data) // '{"nama":"Ahmad","skor":100}'
    var obj = json.urai(jsonTeks)

    // 3. Manipulasi Array Fungsional (Filter & Map)
    var angka = [1, 2, 3, 4, 5, 6]
    var genap = larik.saring(angka, fungsi(x) { kembalikan x % 2 == 0 }) // [2, 4, 6]
    var kaliDua = larik.petakan(genap, fungsi(x) { kembalikan x * 2 })   // [4, 8, 12]
}
```

### 12. Sinkronisasi WaitGroup (`kelompokTunggu`) & Utilitas Teks Lanjutan (`contoh/dasar_waitgroup_teks.ku`)
```ku
paket utama

pakai "cetak"
pakai "sistem"
pakai "teks"
pakai "kanal"

fungsi pekerja(id, kt) {
    tunda kt.selesai()
    cetak.tulis(teks.format("Pekerja #{0} selesai tugas!", id))
}

fungsi utama() {
    // 1. Sinkronisasi paralel dengan WaitGroup
    var kt = sistem.kelompokTunggu()
    untuk i := 1; i <= 3; i++ {
        kt.tambah(1)
        ku pekerja(i, kt)
    }
    kt.tunggu() // Menunggu hingga ketiga pekerja selesai

    // 2. Utilitas teks lengkap
    cetak.tulis(teks.ulang("=", 20))
    cetak.tulis("Awalan Bahasa:", teks.awalan("Bahasa Ku", "Bahasa"))
    cetak.tulis("Indeks 'Ku':", teks.indeks("Bahasa Ku", "Ku"))

    // 3. Kanal Non-blocking check
    var pipa = kanal.buat(1)
    var nilai, ada = pipa.cobaTerima()
    jika !ada {
        cetak.tulis("Kanal kosong tanpa blocking!")
    }
}
```

### 13. Waktu, Ekspresi Reguler (Regex), Pengurutan Larik & Trigonometri (`contoh/dasar_waktu_pola.ku`)
```ku
paket utama

pakai "cetak"

fungsi utama() {
    // 1. Modul Waktu & Durasi
    var sekarang = waktu.sekarang()
    cetak.tulis("Waktu saat ini:", sekarang["teks"])
    cetak.tulis("Unix timestamp:", waktu.detikUnix())
    cetak.tulis("Waktu 1 jam ke depan:", waktu.tambahDetik(sekarang["detikUnix"], 3600))

    // 2. Modul Pola (Ekspresi Reguler / Regex)
    var email = "pengguna@bahasaku.id"
    jika pola.cocok("^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$", email) {
        cetak.tulis("Surel valid!")
    }
    var daftarID = pola.temukanSemua("ID-[0-9]+", "ID-100, ID-250, ID-999")
    var sensor = pola.ganti("[0-9]", "Pin: 1234", "*") // Pin: ****

    // 3. Pengurutan & Manipulasi Koleksi
    var angka = [45, 12, 89, 3, 27]
    var terurut = larik.urut(angka)   // [3, 12, 27, 45, 89]
    var terbalik = larik.balik(terurut) // [89, 45, 27, 12, 3]

    // 4. Statistik & Trigonometri
    cetak.tulis("Rata-rata:", matematika.rataRata(angka)) // 35.2
    cetak.tulis("Sin(0):", matematika.sin(0))
    cetak.tulis("Cos(0):", matematika.cos(0))
}
```

### 14. Jalur Berkas, Enkoding CSV & URL, UUID & HMAC (`contoh/dasar_jalur_enkoding.ku`)
```ku
paket utama

pakai "cetak"
pakai "jalur"
pakai "enkoding"
pakai "kripto"
pakai "konversi"

fungsi utama() {
    // 1. Modul Jalur (filepath)
    var path = jalur.gabung("proyek", "sumber", "app.ku")
    cetak.tulis("Nama berkas:", jalur.nama(path))   // app.ku
    cetak.tulis("Folder induk:", jalur.induk(path)) // proyek/sumber
    cetak.tulis("Ekstensi:", jalur.ekstensi(path))  // .ku

    // 2. Modul Enkoding CSV & URL Query
    var tabel = enkoding.uraiCSV("nama,skor\nBudi,95")
    var urlEnk = enkoding.kodekanURL("bahasa ku & golang") // bahasa+ku+%26+golang

    // 3. Kripto UUID & HMAC-SHA256
    var idUnik = kripto.uuid() // misal: 63107efa-1d39-4669-93b5-4dc560bc08d1
    var signature = kripto.hmacSha256("rahasia", "data payload")

    // 4. Konversi Format Rupiah & Desimal
    cetak.tulis(konversi.rupiah(2500000))        // Rp 2.500.000
    cetak.tulis(konversi.desimal(3.14159265, 3)) // 3.142
}
```

### 15. Ekstraksi Kamus (`peta`), Kompresi Gzip, Himpunan Larik & Benchmark (`contoh/dasar_peta_kompresi.ku`)
```ku
paket utama

pakai "cetak"
pakai "peta"
pakai "kompresi"
pakai "larik"
pakai "waktu"
pakai "sistem"

fungsi utama() {
    // 1. Ekstraksi dan Penggabungan Map (Peta)
    var profil = {"nama": "Bahasa Ku", "status": "aktif"}
    var kunci = peta.kunci(profil)  // ["nama", "status"]
    var ada = peta.ada(profil, "nama") // benar

    // 2. Kompresi & Dekompresi Gzip
    var dataGz = kompresi.gzip("Teks rahasia yang panjang...")
    var teksAsli = kompresi.ungzip(dataGz)

    // 3. Himpunan & Dedup Larik
    var angka = [1, 2, 2, 3, 4, 4, 5]
    var unik = larik.unik(angka)                 // [1, 2, 3, 4, 5]
    var irisan = larik.irisan(unik, [4, 5, 6])   // [4, 5]
    var gabung = larik.gabungan(unik, [5, 6, 7]) // [1, 2, 3, 4, 5, 6, 7]

    // 4. Pengukur Waktu Eksekusi (Benchmark)
    var tMulai = waktu.mulai()
    // Operasi...
    cetak.tulis("Waktu eksekusi:", waktu.lewat(tMulai), "ms")

    // 5. Eksekusi Perintah Subprocess OS
    var res = sistem.perintah("cmd", "/c", "echo Halo Ku!")
    cetak.tulis(res["keluaran"])
}
```

### 16. I/O Berkas Lanjutan, Logging Terstruktur (`catatan`), Padding Teks, & Enkoding Heksa (`contoh/dasar_berkas_catatan.ku`)
```ku
paket utama

pakai "cetak"
pakai "berkas"
pakai "catatan"
pakai "teks"
pakai "enkoding"
pakai "larik"

fungsi utama() {
    // 1. Logging Terstruktur dengan Timestamp Otomatis
    catatan.info("Memulai pemrosesan transaksi...")
    catatan.peringatan("Beban sistem mendekati ambang batas!")

    // 2. Manipulasi File: Tambah (Append), Salin, dan Pindah
    berkas.tambah("riwayat.txt", "Pencatatan baris baru\n")
    berkas.salin("riwayat.txt", "cadangan_riwayat.txt")
    berkas.pindah("cadangan_riwayat.txt", "arsip.txt")

    // 3. Padding Teks & Karakter
    var idOrder = teks.padKiri("42", 5, "0") // "00042"
    var kolom = teks.padKanan("Barang", 10, " ") // "Barang    "
    var hurufPertama = teks.karakter("Bahasa Ku", 0) // "B"

    // 4. Enkoding Biner Heksadesimal
    var heksa = enkoding.kodekanHeksa("Ku") // "4b75"
    var teksAsli = enkoding.dekodekanHeksa(heksa) // "Ku"

    // 5. Pencarian Elemen Larik
    var bahasa = ["Go", "Ku", "Python"]
    var adaKu = larik.mengandung(bahasa, "Ku") // benar
    var posisiKu = larik.indeks(bahasa, "Ku") // 1
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

## 📄 Lisensi & Pernyataan Etika (Ethical Notice)

Proyek ini didistribusikan di bawah lisensi terbuka [MIT License](LICENSE).

### Pengakuan & Penghormatan Hak Cipta
- **Sumber Inspirasi:** Bahasa Ku dirancang sebagai bahasa tingkat tinggi untuk tujuan edukasi dan kemudahan akses (*accessibility*), yang **terinspirasi oleh arsitektur dan kesederhanaan bahasa Go (The Go Authors / Google LLC)**.
- **Batasan Afiliasi:** Bahasa Ku adalah proyek independen dan tidak terafiliasi, disponsori, maupun didukung secara resmi oleh Google LLC atau The Go Authors. Seluruh merek dagang dan hak kekayaan intelektual dari bahasa Go tetap sepenuhnya milik pemegang hak cipta aslinya.
- **Tujuan Pengembangan:** Proyek ini lahir sebagai sarana edukasi untuk membantu penutur Bahasa Indonesia mempelajari konsep komputasi, algoritma, dan pemrograman sistem dengan hambatan bahasa yang seminimal mungkin.

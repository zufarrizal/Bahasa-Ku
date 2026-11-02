# 📖 Katalog Lengkap Bahasa Ku (Go Native Edition)

Dokumen ini berisi inventarisasi menyeluruh seluruh sintaksis, kata kunci, fungsi bawaan global (*built-in*), alias kompak, operator, modul pustaka standar, serta perkakas CLI yang ada di dalam **Bahasa Ku v0.3.7**.

---

## 📑 Daftar Isi
1. [Kata Kunci Bahasa (Keywords)](#1-kata-kunci-bahasa-keywords)
2. [Fungsi Bawaan Global Universal (Built-in)](#2-fungsi-bawaan-global-universal-built-in)
3. [Operator Bahasa & Karakter Khusus](#3-operator-bahasa--karakter-khusus)
4. [Tipe Data Primitif & Komposit](#4-tipe-data-primitif--komposit)
5. [Modul Pustaka Standar & Alias Kompak](#5-modul-pustaka-standar--alias-kompak)
6. [Daftar Perintah Baris Terminal CLI (`ku`)](#6-daftar-perintah-baris-terminal-cli-ku)

---

## 1. Kata Kunci Bahasa (Keywords)

| Kata Kunci | Padanan di Go / Lainnya | Penjelasan & Fungsi |
| :--- | :--- | :--- |
| `paket` | `package` | Mendeklarasikan paket berkas sumber (contoh: `paket utama`). |
| `pakai` | `import` | Mengimpor pustaka standar atau berkas lokal `.ku` (contoh: `pakai "mat"`). |
| `fungsi` | `func` | Mendefinisikan fungsi atau metode receiver struct. |
| `var` / `variabel` | `var` | Mendeklarasikan variabel dengan tipe dinamis atau eksplisit. |
| `tetap` / `konst` | `const` | Mendeklarasikan konstanta yang nilainya tidak dapat diubah. |
| `tipe` | `type` | Mendeklarasikan tipe kustom baru (struktur atau antarmuka). |
| `struktur` | `struct` | Mendefinisikan skema tipe data komposit terstruktur ala Go. |
| `antarmuka` | `interface` | Mendefinisikan kontrak interface dengan polimorfisme struktural ala Go. |
| `jika` | `if` | Percabangan kondisional (mendukung short statement: `jika err := ...; err != kosong`). |
| `lainnya` | `else` | Blok percabangan alternatif jika kondisi tidak terpenuhi. |
| `untuk` | `for` | Perulangan tunggal serbaguna: standar for, while-style, iterasi atas larik/peta, atau channel drain loop. |
| `dalam` / `rentang` | `range` | Klausa iterasi perulangan atas elemen larik atau peta (`untuk i, v dalam koleksi`). |
| `pilih` | `switch` / `select` | Pernyataan seleksi nilai kasus (switch) atau multiplexer kanal konkurensi (select). |
| `kasus` | `case` | Cabang pengujian nilai atau kesiapan kanal di dalam blok `pilih`. |
| `bawaan` | `default` | Klausa default jika tidak ada kondisi kasus yang cocok atau kanal belum siap. |
| `teruskan` | `fallthrough` | Melanjutkan eksekusi ke blok kasus berikutnya secara eksplisit. |
| `kembalikan` | `return` | Mengembalikan nilai dari fungsi (mendukung multiple return values). |
| `tunda` | `defer` | Menunda eksekusi kode hingga blok fungsi pembungkus selesai (antrean LIFO). |
| `ku` | `go` | Meluncurkan pekerjaan asinkron/paralel dalam thread ringan Ku-routine (Goroutine). |
| `panik` | `panic` | Memicu kondisi fatal atau penghentian runtime darurat. |
| `pulih` | `recover` | Menangkap dan memulihkan eksekusi program dari kondisi panik. |
| `henti` / `berhenti` | `break` | Menghentikan paksa perulangan atau keluar dari blok `pilih`. |
| `lanjut` | `continue` | Melompat ke iterasi perulangan berikutnya. |
| `coba` | `try` | Memulai blok penanganan pengecualian terstruktur. |
| `tangkap` | `catch` | Menangkap galat dari blok coba. |
| `akhirnya` | `finally` | Menjalankan blok pembersihan yang selalu dieksekusi setelah coba-tangkap. |

---

## 2. Fungsi Bawaan Global Universal (Built-in)
Dapat langsung dipanggil di mana saja tanpa memerlukan awalan nama modul atau impor:

| Fungsi Bawaan | Parameter | Deskripsi & Contoh |
| :--- | :--- | :--- |
| `tulis(...)` | `...data` | Mencetak teks/data langsung ke terminal diikuti baris baru: `tulis("Halo!", 42)` |
| `tanya(...)` | `prompt, [default]` | Meminta masukan teks dari pengguna di terminal: `var nama = tanya("Nama: ")` |
| `angka(...)` | `nilai` | Konversi string/desimal ke integer: `angka("123")` (sekaligus modul `angka`) |
| `teks(...)` | `nilai` | Konversi apa pun ke teks string: `teks(42)` (sekaligus modul `teks`) |
| `baca(...)` | `jalurBerkas` | Membaca seluruh isi file teks lokal secara instan: `var data = baca("log.txt")` |
| `ambil(...)` | `url` | Melakukan HTTP GET request mentah ke web / REST API: `var res = ambil("https://api.com")` |
| `rentang(...)` | `start, end, [step]` | Menghasilkan larik angka berurutan: `rentang(1, 5)` ➔ `[1, 2, 3, 4]` |
| `kunci(...)` | `peta` | Mengambil seluruh nama kunci properti dari peta: `kunci({"a": 1, "b": 2})` ➔ `["a", "b"]` |
| `nilai(...)` | `peta` | Mengambil seluruh nilai elemen dari peta: `nilai({"a": 1, "b": 2})` ➔ `[1, 2]` |
| `sekarang()` | `-` | Mengembalikan objek tanggal/waktu saat ini (`tahun`, `bulan`, `jam`, `teks`, `detikUnix`) |
| `format(...)` | `pola, ...arg` | Memformat teks dengan pola printf (`%s`, `%d`) atau placeholder (`{0}`): `format("ID: %04d", 5)` |
| `tidur(...)` | `milidetik` | Menghentikan sementara alur rutin selama N milidetik: `tidur(500)` |
| `urai(...)` | `jsonTeks` | Mem-parse string JSON langsung menjadi Peta atau Larik: `var d = urai(jsonStr)` |
| `kodekan(...)` | `objek` | Mengonversi Peta, Larik, atau Struct menjadi format string JSON: `kodekan(d)` |
| `keluar(...)` | `kodeStatus` | Mengakhiri program dengan kode keluar mesin: `keluar(0)` |
| `perintah(...)` | `cmd, ...args` | Mengeksekusi perintah shell OS: `var res = perintah("git", "status")` |
| `panjang(...)` | `koleksi` | Mengembalikan panjang larik, string, atau peta (mirip `len()` di Go): `panjang([1, 2])` |
| `kapasitas(...)`| `larik/kanal` | Mengembalikan kapasitas buffer memori (mirip `cap()` di Go): `kapasitas(ch)` |
| `buat(...)` | `tipe, [panjang]` | Mengalokasikan larik, kanal, atau peta (mirip `make()` di Go): `buat("kanal", 5)` |
| `salin(...)` | `tujuan, sumber` | Menyalin elemen antar larik (mirip `copy()` di Go) |
| `tutup(...)` | `kanal` | Menutup kanal komunikasi setelah transmisi selesai (mirip `close()` di Go) |
| `tambah(...)` | `larik, item` | Menambahkan elemen ke larik (append): `tambah(daftar, 99)` |
| `hapus(...)` | `peta, kunci` | Menghapus kunci dari peta atau menghapus berkas (mirip `delete()` di Go) |
| `cacah` / `iota`| `-` | Generator nilai integer berurutan otomatis dalam blok deklarasi konstanta. |

---

## 3. Operator Bahasa & Karakter Khusus

| Operator | Kategori | Penjelasan & Contoh |
| :--- | :--- | :--- |
| `+`, `-`, `*`, `/`, `%` | Aritmatika | Operasi penambahan, pengurangan, perkalian, pembagian, dan sisa bagi (modulus). Operator `+` otomatis mengonversi tipe angka/boolean ke teks jika digabung dengan string. |
| `+=`, `-=`, `*=`, `/=` | Penugasan | Operator compound assignment: `x += 5` |
| `++`, `--` | Increment / Decrement | Penaikan dan penurunan nilai integer: `i++`, `i--` |
| `==`, `!=` | Kesetaraan | Memeriksa nilai sama atau tidak sama. |
| `<`, `<=`, `>`, `>=` | Relasional | Pembandingan angka numerik dan string. |
| `&&`, `\|\|`, `!` | Logika Boolean | Operator logika DAN, ATAU, dan BUKAN. |
| `??` | Null Coalescing | Fallback default jika nilai `kosong` (*nil*): `var port = input ?? 8080` |
| `<-` | Komunikasi Kanal | Operator kirim/terima pesan kanal: `ch <- pesan` atau `var pesan = <-ch` |
| `?.` | Optional Chaining | Akses properti aman tanpa panik jika objek bernilai kosong: `pengguna?.profil?.alamat` |
| `.` | Dot Operator & Method Chaining | Akses properti/metode pada struct, peta, modul, serta metode bawaan langsung pada: <br>• **Teks (String):** `.panjang()`, `.pangkas()`, `.kecil()`, `.kapital()`, `.pisah()`, `.mengandung()`, `.ganti()`, `.awalan()`, `.akhiran()`<br>• **Larik (Array):** `.panjang()`, `.saring()`, `.petakan()`, `.gabung()`, `.tambah()`, `.mengandung()`<br>• **Peta:** `.panjang()`, `.kunci()`, `.nilai()`, `.ada()`, `.ambil()`, `.hapus()` |
| `:=` | Deklarasi Instan | Deklarasi variabel baru dan penugasan sekaligus ala Go: `nama := "Budi"` atau `val, ok := peta[kunci]` |
| `_` | Blank Identifier | Pengabaian nilai yang tidak digunakan ala Go: `_, ok := kamus["kunci"]` atau `untuk _, item dalam daftar` |
| `//`, `/* ... */` | Komentar | Komentar satu baris dan komentar multi-baris. |
| `${ekspresi}` | Interpolasi String | Template literal teks di dalam string: `"Halo ${nama}, umurmu ${usia}"` |

---

## 4. Tipe Data Primitif & Komposit

| Tipe Data | Representasi | Contoh Nilai |
| :--- | :--- | :--- |
| **Integer** | Angka bulat (int64) | `42`, `-10`, `0xFF` (heksa), `0b1010` (biner), `1_000_000` |
| **Float** | Angka desimal (float64) | `3.14`, `-0.005`, `2.5e3` |
| **String** | Teks berkarakter UTF-8 | `"Halo Dunia"`, ``raw string multiline`` |
| **Boolean** | Nilai kebenaran | `benar` (*true*), `salah` (*false*) |
| **Kosong / Nil**| Ketiadaan nilai | `kosong`, `nihil` (*nil / null*) |
| **Larik (Array)**| Kumpulan elemen dinamis | `[1, 2, "tiga", benar]` |
| **Peta** | Pasangan kunci dan nilai (key-value) | `{"nama": "Ahmad", "usia": 28}` |
| **Struktur** | Definisi struct ala Go | `tipe Orang struktur { nama teks, umur angka }` |
| **Kanal** | Komunikasi rutin Ku-routine | `var ch = buat("kanal", 10)` |

---

## 5. Modul Pustaka Standar & Alias Kompak

Bahasa Ku menyediakan sejumlah modul bawaan resmi beserta alias pendek untuk penulisan cepat:

### 🌟 Tabel Pustaka Standar & Alias Ringkas

| Modul Utama | Alias Kompak | Kategori | Deskripsi Singkat & Metode Utama |
| :--- | :--- | :--- | :--- |
| `basisdata` | **`bd`** | Database | SQLite & MySQL query builder: `bd.bukaBerkas()`, `bd.tabel("produk").pilih("id, nama")`, `.masukkan(...)` (otomatis batch insert jika diberi larik data), `.dimana()`, `.ambilSemua()`, `.hitung()`, `.halaman(nomor, perHalaman)`, `.lewat(offset)`, `.batas(limit)`, `bd.transaksi(fungsi(tx) { .. })`. |
| `konfigurasi` | **`konf`** | Konfigurasi | Pengelolaan file `.env` dan JSON: `konf.muatBerkas()`, `konf.ambilAngka("PORT", 8080)`. |
| `strukturData`| **`sd`** | Algoritma | Struktur data lanjutan: `sd.tumpukan()` (Stack LIFO) & `sd.antrean()` (Queue FIFO). |
| `singgahan` | **`ingat`** | Cache Memori | In-memory key-value cache dengan TTL & memoize: `ingat.taruh()`, `ingat.ambil()`, `ingat.ingat()`. |
| `peristiwa` | **`sinyal`** | Event Bus | Sistem pub-sub internal: `sinyal.pada("daftar", fn)`, `sinyal.pancarkan()`, `sinyal.lepas()`. |
| `matematika` | **`mat`** | Numerik | Kalkulasi numerik, statistik, & trigonometri: `mat.sin()`, `mat.akar()`, `mat.jumlah()`, `mat.pangkat()`. |
| `kelompokTunggu`| **`tunggu`** *(pada `sistem`)* | Konkurensi | WaitGroup sinkronisasi goroutine: `var kt = sistem.tunggu(); kt.tambah(1); kt.tunggu()`. |
| `cetak` | - | I/O Terminal | Pencetakan teks format: `cetak.tulis()`, `cetak.tipe()`, `cetak.dump()`. |
| `teks` | - | Manipulasi String | Pemrosesan string: `teks.kapital()`, `teks.kecil()`, `teks.pisah()`, `teks.gabung()`, `teks.slug()`. |
| `berkas` | - | File I/O | Operasi filesystem: `berkas.baca()`, `berkas.tulis()`, `berkas.ada()`, `berkas.buatDirektori()`. |
| `jalur` | - | Filepath | Manipulasi path: `jalur.gabung()`, `jalur.ekstensi()`, `jalur.direktori()`, `jalur.namaBerkas()`. |
| `sistem` | - | Mesin OS | Introspeksi OS & Proses: `sistem.sistemOperasi()`, `sistem.arsitektur()`, `sistem.memori()`, `sistem.pid()`. |
| `waktu` | - | Penanggalan | Pengelolaan durasi & stopwatch: `waktu.sekarang()`, `waktu.tidur()`, `waktu.setiap()`, `waktu.sejak()`. |
| `kripto` | - | Keamanan | Enkripsi & Hashing: `kripto.sha256()`, `kripto.sha512()`, `kripto.md5()`, `kripto.buatJWT()`, `kripto.hashSandi()`. |
| `jaringan` | - | HTTP Client | Request HTTP web: `jaringan.ambil()`, `jaringan.ambilJSON()`, `jaringan.kirimJSON()`. |
| `web` | - | Web Server & REST API | Router HTTP REST API bawaan ala Go Gin/Echo: `web.dapatkan()`, `web.pos()`, `web.taruh()`, `web.hapus()`, `web.grup()`, `web.gunakan()` (middleware global dan scoped grup), `web.cors()`, `web.catatAkses()`, `web.pulihkan()` (pemulihan panik), `web.berkasStatis()`, `web.putar()` / `web.jalankan(port)`. Mendukung handler fleksibel: `fungsi(konteks)` atau `fungsi(req, resp)` dengan metode konteks ringkas idiomatis Indonesia: `konteks.param()`, `konteks.kueri()`, `konteks.json()`, `konteks.ikat()`, `konteks.kepala()` (header), `konteks.badan` (body), `konteks.kaki()` (trailer), `konteks.kue()` (cookie), `konteks.taruh()`, `konteks.ambil()`, `konteks.sukses()`, `konteks.gagal()`, `konteks.konflik()`, dll. |
| `soket` | - | TCP Network | Jaringan soket TCP murni tingkat rendah: `soket.dengar("tcp", ":9000")`, `soket.hubungkan()`. |
| `desktop` / `jendela` | - | GUI Desktop | Aplikasi desktop mandiri (Microsoft Edge WebView2 native tanpa CGO) & dialog pesan OS. |
| `json` | - | Format JSON | Serialisasi format JSON: `json.urai(str)`, `json.kodekan(obj)`, `json.format(obj)`. |
| `csv` | - | Format Tabular | Baca & tulis data tabular: `csv.urai()`, `csv.kodekan()`. |
| `arsip` / `kompresi`| - | Arsip & ZIP | Kompresi arsip berkas: `arsip.buatZip()`, `arsip.ekstrakZip()`, `kompresi.gzipKompres()`. |
| `validasi` | - | Validasi Input | Validator format data: `validasi.apakahSurel()`, `validasi.apakahURL()`, `validasi.apakahAngka()`. |
| `terminal` | - | CLI Warna | Penataan warna konsol ANSI: `terminal.hijau()`, `terminal.merah()`, `terminal.kuning()`, `terminal.bersihkan()`. |
| `masukan` | - | Prompt Input | Pembaca input terminal: `masukan.tanya()`, `masukan.baca()`, `masukan.konfirmasi()`. |
| `jadwal` | - | Cron Scheduler | Penjadwalan tugas latar belakang: `jadwal.setiap()`, `jadwal.cron()`. |
| `pekerja` | - | Worker Pool | Eksekusi kumpulan Ku-routine pekerja terkendali: `pekerja.buat()`. |
| `uji` | - | Unit Testing | Framework pengujian unit internal: `uji.sama()`, `uji.benar()`, `uji.tidakKosong()`. |
| `debug` | - | Profiling | Pelacakan alokasi heap memori dan stack trace runtime. |

---

## 6. Daftar Perintah Baris Terminal CLI (`ku`)

| Perintah Terminal | Kegunaan & Contoh |
| :--- | :--- |
| `ku` | Membuka sesi REPL interaktif langsung di terminal. |
| `ku <berkas.ku>` | **Eksekusi langsung** tanpa subperintah: `ku aplikasi.ku`. |
| `ku .` | Otomatis mencari dan menjalankan `utama.ku` atau `main.ku` di direktori saat ini. |
| `ku -e '<kode>'` | Mengevaluasi ekspresi satu baris: `ku -e 'tulis(1 + 2)'`. |
| `ku jalankan <berkas.ku>` | Mengeksekusi berkas sumber `.ku` (opsi: `--debug` / `-d`). |
| `ku awasi <berkas.ku>` | Menjalankan berkas dengan **Hot Reload** (restart otomatis saat berkas diubah/disimpan). |
| `ku uji [berkas_uji.ku]` | Menjalankan suite unit test otomatis (`*_uji.ku`). |
| `ku tolakUkur [berkas]` | Menjalankan tolok ukur benchmark performa komputasi. |
| `ku rapi [-w] <berkas\|folder>` | Merapikan format indentasi kode (`-w`: timpa simpan berkas, `-c`: validasi CI). |
| `ku kompilasi <berkas.ku>` | Mengompilasi kode `.ku` menjadi berkas biner mandiri `.exe` (`-gui`, `-ikon`, `-o`). |
| `ku inisialisasi <folder> [web\|desktop\|cli]` | Membuat template proyek baru beserta manifes `ku.mod`. |
| `ku buat <berkas.ku>` | Mengonversi kode Bahasa Ku menjadi berkas bahasa Go murni (`.go`). |
| `ku versi [--cek]` | Menampilkan versi runtime dan memeriksa ketersediaan rilis baru. |
| `ku perbarui` | Mengunduh dan memperbarui biner CLI Bahasa Ku ke versi rilis terbaru secara otomatis. |

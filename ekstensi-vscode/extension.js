const vscode = require('vscode');

// Metadata Autocomplete Modul dan Fungsi Bawaan Bahasa Ku
const MODULES_DATA = {
  arsip: {
    doc: "Modul kompresi arsip ZIP dan ekstraksi berkas lokal.",
    methods: {
      buatZip: { sig: "arsip.buatZip(namaBerkasZip, daftarBerkas)", doc: "Mengompresi berkas atau folder ke format .zip." },
      ekstrakZip: { sig: "arsip.ekstrakZip(namaBerkasZip, tujuanFolder)", doc: "Mengekstrak seluruh isi berkas .zip ke folder target." }
    }
  },
  jendela: {
    doc: "Modul GUI Desktop Native WebView2 (Arsitektur Wails-Style) berkinerja tinggi.",
    methods: {
      buka: { sig: "jendela.buka(konfigurasi)", doc: "Membuka jendela GUI aplikasi desktop native (judul, lebar, tinggi, konten/html)." },
      ikat: { sig: "jendela.ikat(nama, fungsi)", doc: "Mendaftarkan fungsi Bahasa Ku ke runtime IPC JavaScript (window.ku.panggil)." },
      pesan: { sig: "jendela.pesan(judul, isi, [tipe])", doc: "Menampilkan kotak dialog pesan native OS." },
      konfirmasi: { sig: "jendela.konfirmasi(judul, isi)", doc: "Menampilkan kotak dialog konfirmasi Ya/Tidak native OS." },
      pilihBerkas: { sig: "jendela.pilihBerkas([judul], [filter])", doc: "Membuka dialog pemilih berkas native OS." },
      pilihFolder: { sig: "jendela.pilihFolder([judul])", doc: "Membuka dialog pemilih direktori/folder native OS." },
      salinTeks: { sig: "jendela.salinTeks(teks)", doc: "Menyalin teks ke papan klip (clipboard) sistem." },
      bacaKlip: { sig: "jendela.bacaKlip()", doc: "Membaca teks dari papan klip (clipboard) sistem." },
      tutup: { sig: "jendela.tutup()", doc: "Menutup jendela aplikasi desktop dan membersihkan koneksi basis data." }
    }
  },
  desktop: {
    doc: "Alias modul jendela untuk GUI Desktop Native WebView2.",
    methods: {
      buka: { sig: "desktop.buka(konfigurasi)", doc: "Membuka jendela GUI aplikasi desktop native (judul, lebar, tinggi, konten/html)." },
      ikat: { sig: "desktop.ikat(nama, fungsi)", doc: "Mendaftarkan fungsi Bahasa Ku ke runtime IPC JavaScript (window.ku.panggil)." },
      pesan: { sig: "desktop.pesan(judul, isi, [tipe])", doc: "Menampilkan kotak dialog pesan native OS." },
      konfirmasi: { sig: "desktop.konfirmasi(judul, isi)", doc: "Menampilkan kotak dialog konfirmasi Ya/Tidak native OS." },
      pilihBerkas: { sig: "desktop.pilihBerkas([judul], [filter])", doc: "Membuka dialog pemilih berkas native OS." },
      pilihFolder: { sig: "desktop.pilihFolder([judul])", doc: "Membuka dialog pemilih direktori/folder native OS." },
      salinTeks: { sig: "desktop.salinTeks(teks)", doc: "Menyalin teks ke papan klip (clipboard) sistem." },
      bacaKlip: { sig: "desktop.bacaKlip()", doc: "Membaca teks dari papan klip (clipboard) sistem." },
      tutup: { sig: "desktop.tutup()", doc: "Menutup jendela aplikasi desktop dan membersihkan koneksi basis data." }
    }
  },
  waktu: {
    doc: "Modul penanganan tanggal, jam, pewaktu, dan durasi.",
    methods: {
      sekarang: { sig: "waktu.sekarang()", doc: "Mengambil objek waktu saat ini (Unix timestamp & tanggal)." },
      tidur: { sig: "waktu.tidur(milidetik)", doc: "Menghentikan rutin sementara selama N milidetik." },
      setiap: { sig: "waktu.setiap(milidetik, fungsi)", doc: "Menjalankan fungsi berulang secara asinkron dengan interval milidetik." },
      setelah: { sig: "waktu.setelah(milidetik, fungsi)", doc: "Menjalankan fungsi sekali setelah jeda milidetik terlewati (timeout)." },
      format: { sig: "waktu.format(wkt, formatPola)", doc: "Memformat tanggal/waktu sesuai pola string." },
      sejak: { sig: "waktu.sejak(wktAwal)", doc: "Menghitung selisih durasi yang telah berlalu sejak waktu awal." }
    }
  },
  cetak: {
    doc: "Modul I/O terminal untuk menampilkan informasi dan format teks.",
    methods: {
      tulis: { sig: "cetak.tulis(...args)", doc: "Mencetak argumen ke konsol baris terminal." },
      tulisBaris: { sig: "cetak.tulisBaris(...args)", doc: "Mencetak argumen diikuti dengan baris baru." },
      format: { sig: "cetak.format(formatStr, ...args)", doc: "Memformat string ala sprintf." },
      tipe: { sig: "cetak.tipe(obj)", doc: "Mendapatkan nama tipe data dari sebuah objek saat runtime." },
      dump: { sig: "cetak.dump(obj)", doc: "Melakukan inspeksi mendalam (dumping) struktur objek data." }
    }
  },
  kanal: {
    doc: "Modul komunikasi dan sinkronisasi kanal data (Go Channel).",
    methods: {
      buat: { sig: "kanal.buat(kapasitas)", doc: "Membuat instance kanal baru dengan kapasitas buffer tertentu." },
      pilih: { sig: "kanal.pilih(...kasus)", doc: "Mengevaluasi kesiapan kanal secara dinamis." },
      tutup: { sig: "tutup(ch)", doc: "Menutup kanal setelah pengiriman selesai." }
    }
  },
  basisdata: {
    doc: "Modul interaksi basis data SQL relasional (MySQL & SQLite).",
    methods: {
      bukaBerkas: { sig: "basisdata.bukaBerkas(namaBerkas)", doc: "Membuka atau membuat berkas database SQLite lokal." },
      buka: { sig: "basisdata.buka(tipe, dsn)", doc: "Membuka koneksi jaringan database (misal: MySQL)." },
      sambung: { sig: "basisdata.sambung(tipe, dsn)", doc: "Menyambungkan koneksi database (MySQL / SQLite)." },
      kueri: { sig: "db.kueri(sql, ...params)", doc: "Menjalankan kueri SQL SELECT dan mengembalikan larik peta baris." },
      eksekusi: { sig: "db.eksekusi(sql, ...params)", doc: "Menjalankan kueri SQL INSERT/UPDATE/DELETE/DDL." },
      transaksi: { sig: "db.transaksi(fungsi(tx) { ... })", doc: "Menjalankan blok transaksi atomik dengan commit/rollback otomatis." },
      tabel: { sig: "db.tabel(namaTabel)", doc: "Query builder tabel database terstruktur (pilih, dimana, urutkan, batas, lewat, halaman, hitung, ambilSemua, ambilSatu, masukkan, perbarui, hapus)." },
      tutup: { sig: "db.tutup()", doc: "Menutup koneksi database." }
    }
  },
  bd: {
    doc: "Alias ringkas untuk modul basisdata (SQLite & MySQL).",
    methods: {
      bukaBerkas: { sig: "bd.bukaBerkas(namaBerkas)", doc: "Membuka atau membuat berkas database SQLite lokal." },
      buka: { sig: "bd.buka(tipe, dsn)", doc: "Membuka koneksi jaringan database (misal: MySQL) atau berkas lokal." },
      sambung: { sig: "bd.sambung(tipe, dsn)", doc: "Menyambungkan koneksi database (MySQL / SQLite)." },
      kueri: { sig: "db.kueri(sql, ...params)", doc: "Menjalankan kueri SQL SELECT dan mengembalikan larik peta baris." },
      eksekusi: { sig: "db.eksekusi(sql, ...params)", doc: "Menjalankan kueri SQL INSERT/UPDATE/DELETE/DDL." },
      transaksi: { sig: "db.transaksi(fungsi(tx) { ... })", doc: "Menjalankan blok transaksi atomik dengan commit/rollback otomatis." },
      tabel: { sig: "db.tabel(namaTabel)", doc: "Query builder tabel database terstruktur (pilih, dimana, urutkan, batas, lewat, halaman, hitung, ambilSemua, ambilSatu, masukkan, perbarui, hapus)." },
      tutup: { sig: "db.tutup()", doc: "Menutup koneksi database." }
    }
  },
  strukturData: {
    doc: "Modul struktur data lanjutan (Tumpukan / Stack & Antrean / Queue).",
    methods: {
      tumpukan: { sig: "strukturData.tumpukan()", doc: "Membuat struktur data tumpukan LIFO (dorong, tarik, intip, panjang, apakahKosong, semua)." },
      antrean: { sig: "strukturData.antrean()", doc: "Membuat struktur data antrean FIFO (masuk, keluar, depan, panjang, apakahKosong, semua)." }
    }
  },
  sd: {
    doc: "Alias ringkas untuk modul strukturData (Tumpukan & Antrean).",
    methods: {
      tumpukan: { sig: "sd.tumpukan()", doc: "Membuat struktur data tumpukan LIFO (dorong, tarik, intip, panjang, apakahKosong, semua)." },
      antrean: { sig: "sd.antrean()", doc: "Membuat struktur data antrean FIFO (masuk, keluar, depan, panjang, apakahKosong, semua)." }
    }
  },
  konfigurasi: {
    doc: "Modul manajemen konfigurasi aplikasi (.json, .env, key-value).",
    methods: {
      atur: { sig: "konfigurasi.atur(kunci, nilai)", doc: "Menyetel nilai konfigurasi secara manual." },
      ambil: { sig: "konfigurasi.ambil(kunci, bawaan)", doc: "Mengambil nilai konfigurasi apa saja." },
      ambilTeks: { sig: "konfigurasi.ambilTeks(kunci, bawaan)", doc: "Mengambil nilai konfigurasi tipe string." },
      ambilAngka: { sig: "konfigurasi.ambilAngka(kunci, bawaan)", doc: "Mengambil nilai konfigurasi tipe float/integer." },
      ambilBenarSalah: { sig: "konfigurasi.ambilBenarSalah(kunci, bawaan)", doc: "Mengambil nilai konfigurasi tipe boolean." },
      muatBerkas: { sig: "konfigurasi.muatBerkas(jalurBerkas)", doc: "Memuat variabel konfigurasi dari berkas JSON atau .env." }
    }
  },
  konf: {
    doc: "Alias ringkas untuk modul konfigurasi (.json, .env, key-value).",
    methods: {
      atur: { sig: "konf.atur(kunci, nilai)", doc: "Menyetel nilai konfigurasi secara manual." },
      ambil: { sig: "konf.ambil(kunci, bawaan)", doc: "Mengambil nilai konfigurasi apa saja." },
      ambilTeks: { sig: "konf.ambilTeks(kunci, bawaan)", doc: "Mengambil nilai konfigurasi tipe string." },
      ambilAngka: { sig: "konf.ambilAngka(kunci, bawaan)", doc: "Mengambil nilai konfigurasi tipe float/integer." },
      ambilBenarSalah: { sig: "konf.ambilBenarSalah(kunci, bawaan)", doc: "Mengambil nilai konfigurasi tipe boolean." },
      muatBerkas: { sig: "konf.muatBerkas(jalurBerkas)", doc: "Memuat variabel konfigurasi dari berkas JSON atau .env." }
    }
  },
  singgahan: {
    doc: "Modul cache in-memory dengan masa berlaku TTL dan pola memoize.",
    methods: {
      taruh: { sig: "singgahan.taruh(kunci, nilai, [ttl])", doc: "Menyimpan nilai ke singgahan dengan opsi masa berlaku TTL (misal '5m')." },
      ambil: { sig: "singgahan.ambil(kunci, bawaan)", doc: "Mengambil nilai dari singgahan." },
      ada: { sig: "singgahan.ada(kunci)", doc: "Memeriksa apakah kunci tersedia di singgahan dan belum kadaluwarsa." },
      hapus: { sig: "singgahan.hapus(kunci)", doc: "Menghapus entri singgahan berdasarkan kunci." },
      bersihkan: { sig: "singgahan.bersihkan()", doc: "Mengosongkan seluruh entri di dalam singgahan." },
      ingat: { sig: "singgahan.ingat(kunci, ttl, fn)", doc: "Cache-aside memoize: ambil jika ada, jika tidak jalankan fungsi fn lalu simpan." }
    }
  },
  ingat: {
    doc: "Alias ringkas untuk modul singgahan (Cache In-Memory & TTL).",
    methods: {
      taruh: { sig: "ingat.taruh(kunci, nilai, [ttl])", doc: "Menyimpan nilai ke memori dengan opsi masa berlaku TTL (misal '5m')." },
      ambil: { sig: "ingat.ambil(kunci, bawaan)", doc: "Mengambil nilai dari cache memori." },
      ada: { sig: "ingat.ada(kunci)", doc: "Memeriksa apakah kunci tersedia di memori dan belum kadaluwarsa." },
      hapus: { sig: "ingat.hapus(kunci)", doc: "Menghapus entri cache berdasarkan kunci." },
      bersihkan: { sig: "ingat.bersihkan()", doc: "Mengosongkan seluruh isi cache memori." },
      ingat: { sig: "ingat.ingat(kunci, ttl, fn)", doc: "Cache-aside memoize: ambil jika ada, jika tidak jalankan fungsi fn lalu simpan." }
    }
  },
  peristiwa: {
    doc: "Modul event emitter / pub-sub bus internal antarkomponen.",
    methods: {
      pada: { sig: "peristiwa.pada(namaPeristiwa, fungsi)", doc: "Mendaftarkan pendengar peristiwa (mengembalikan id langganan)." },
      sekali: { sig: "peristiwa.sekali(namaPeristiwa, fungsi)", doc: "Mendaftarkan pendengar yang hanya berjalan satu kali saat dipicu." },
      pancarkan: { sig: "peristiwa.pancarkan(namaPeristiwa, payload)", doc: "Memicu seluruh pendengar peristiwa dengan data payload." },
      lepas: { sig: "peristiwa.lepas(namaPeristiwa, idLangganan)", doc: "Mencabut pendaftaran pendengar peristiwa." },
      jumlahPendengar: { sig: "peristiwa.jumlahPendengar(namaPeristiwa)", doc: "Menghitung total pendengar aktif untuk peristiwa." }
    }
  },
  sinyal: {
    doc: "Alias ringkas untuk modul peristiwa (Event Bus / Pub-Sub).",
    methods: {
      pada: { sig: "sinyal.pada(namaPeristiwa, fungsi)", doc: "Mendaftarkan pendengar sinyal/peristiwa (mengembalikan id langganan)." },
      sekali: { sig: "sinyal.sekali(namaPeristiwa, fungsi)", doc: "Mendaftarkan pendengar yang hanya berjalan satu kali saat dipicu." },
      pancarkan: { sig: "sinyal.pancarkan(namaPeristiwa, payload)", doc: "Memicu seluruh pendengar sinyal dengan data payload." },
      lepas: { sig: "sinyal.lepas(namaPeristiwa, idLangganan)", doc: "Mencabut pendaftaran pendengar sinyal." },
      jumlahPendengar: { sig: "sinyal.jumlahPendengar(namaPeristiwa)", doc: "Menghitung total pendengar aktif untuk sinyal." }
    }
  },
  web: {
    doc: "Modul server web HTTP REST API bawaan ala Go Gin/Echo.",
    methods: {
      dapatkan: { sig: "web.dapatkan(rute, penangan)", doc: "Mendaftarkan endpoint HTTP GET." },
      pos: { sig: "web.pos(rute, penangan)", doc: "Mendaftarkan endpoint HTTP POST." },
      taruh: { sig: "web.taruh(rute, penangan)", doc: "Mendaftarkan endpoint HTTP PUT." },
      tambal: { sig: "web.tambal(rute, penangan)", doc: "Mendaftarkan endpoint HTTP PATCH." },
      hapus: { sig: "web.hapus(rute, penangan)", doc: "Mendaftarkan endpoint HTTP DELETE." },
      opsi: { sig: "web.opsi(rute, penangan)", doc: "Mendaftarkan endpoint HTTP OPTIONS." },
      kepala: { sig: "web.kepala(rute, penangan)", doc: "Mendaftarkan endpoint HTTP HEAD." },
      rute: { sig: "web.rute(metode, rute, penangan)", doc: "Mendaftarkan endpoint dengan metode HTTP dinamis." },
      tangani: { sig: "web.tangani(rute, penangan)", doc: "Mendaftarkan endpoint universal untuk seluruh metode HTTP." },
      grup: { sig: "web.grup(prefix, [fn])", doc: "Membuat grup rute berawalan prefix tertentu dengan dukungan scoped middleware grup.gunakan()." },
      gunakan: { sig: "web.gunakan(middleware)", doc: "Memasang middleware pemrosesan rantai permintaan HTTP (mendukung fungsi(konteks, lanjut) atau fungsi(req, res, lanjut))." },
      cors: { sig: "web.cors()", doc: "Mengaktifkan CORS (Cross-Origin Resource Sharing) publik." },
      aktifkanCORS: { sig: "web.aktifkanCORS()", doc: "Mengaktifkan kepala CORS untuk API publik." },
      catatAkses: { sig: "web.catatAkses()", doc: "Memasang middleware pencatat riwayat akses HTTP (Logger)." },
      pulihkan: { sig: "web.pulihkan()", doc: "Memasang middleware pemulihan panic runtime (Panic Recovery)." },
      berkasStatis: { sig: "web.berkasStatis(prefix, direktori)", doc: "Menyajikan berkas statis publik (HTML, CSS, JS, aset)." },
      dengarkan: { sig: "web.dengarkan(port)", doc: "Menjalankan server web HTTP pada port tertentu." },
      jalankan: { sig: "web.jalankan(port)", doc: "Menjalankan server web HTTP pada port tertentu." },
      putar: { sig: "web.putar(port)", doc: "Alias ringkas untuk menjalankan server web HTTP." }
    }
  },
  sistem: {
    doc: "Modul operasi sistem, runtime goroutine, dan introspeksi lingkungan OS.",
    methods: {
      arsitektur: { sig: "sistem.arsitektur()", doc: "Mengembalikan arsitektur CPU target (amd64, arm64, wasm)." },
      sistemOperasi: { sig: "sistem.sistemOperasi()", doc: "Mengembalikan nama OS (windows, linux, darwin)." },
      namaHost: { sig: "sistem.namaHost()", doc: "Mengembalikan nama host / hostname komputer." },
      pid: { sig: "sistem.pid()", doc: "Mengembalikan Process ID dari proses yang sedang berjalan." },
      keluar: { sig: "sistem.keluar(kode)", doc: "Mengakhiri eksekusi program dengan kode keluar (exit code)." },
      kelompokTunggu: { sig: "sistem.kelompokTunggu()", doc: "Membuat instance WaitGroup untuk menunggu sekumpulan Ku-routine." },
      tunggu: { sig: "sistem.tunggu()", doc: "Alias ringkas untuk sistem.kelompokTunggu() membuat instance WaitGroup." },
      kunci: { sig: "sistem.kunci()", doc: "Membuat kunci Mutex eksklusif." },
      kunciBacaTulis: { sig: "sistem.kunciBacaTulis()", doc: "Membuat kunci RWMutex bersama." },
      tidur: { sig: "sistem.tidur(detik)", doc: "Menghentikan sementara eksekusi rutin selama N detik." },
      argumen: { sig: "sistem.argumen()", doc: "Mengambil parameter argumen baris perintah CLI." }
    }
  },
  mat: {
    doc: "Modul kalkulasi angka, trigonometri, dan fungsi matematika statistik.",
    methods: {
      sin: { sig: "mat.sin(x)", doc: "Menghitung sinus sudut radian." },
      cos: { sig: "mat.cos(x)", doc: "Menghitung kosinus sudut radian." },
      akar: { sig: "mat.akar(x)", doc: "Menghitung akar kuadrat sqrt(x)." },
      pangkat: { sig: "mat.pangkat(x, y)", doc: "Menghitung perpangkatan x^y." },
      bulat: { sig: "mat.bulat(x)", doc: "Membulatkan angka ke integer terdekat." },
      lantai: { sig: "mat.lantai(x)", doc: "Pembulatan ke bawah (floor)." },
      langit: { sig: "mat.langit(x)", doc: "Pembulatan ke atas (ceil)." },
      jumlah: { sig: "mat.jumlah(larik)", doc: "Menjumlahkan seluruh angka di dalam larik (sum)." }
    }
  },
  matematika: {
    doc: "Alias modul mat untuk kalkulasi angka dan trigonometri.",
    methods: {
      sin: { sig: "matematika.sin(x)", doc: "Menghitung sinus sudut radian." },
      cos: { sig: "matematika.cos(x)", doc: "Menghitung kosinus sudut radian." },
      akar: { sig: "matematika.akar(x)", doc: "Menghitung akar kuadrat sqrt(x)." },
      pangkat: { sig: "matematika.pangkat(x, y)", doc: "Menghitung perpangkatan x^y." },
      bulat: { sig: "matematika.bulat(x)", doc: "Membulatkan angka ke integer terdekat." },
      lantai: { sig: "matematika.lantai(x)", doc: "Pembulatan ke bawah (floor)." },
      langit: { sig: "matematika.langit(x)", doc: "Pembulatan ke atas (ceil)." },
      jumlah: { sig: "matematika.jumlah(larik)", doc: "Menjumlahkan seluruh angka di dalam larik (sum)." }
    }
  },
  kripto: {
    doc: "Modul enkripsi hash, JWT, sandi PBKDF2, dan pengacakan kriptografi aman.",
    methods: {
      sha256: { sig: "kripto.sha256(teks)", doc: "Menghasilkan heksadesimal hash SHA-256." },
      sha256Berkas: { sig: "kripto.sha256Berkas(jalur)", doc: "Menghitung hash SHA-256 dari sebuah berkas secara hemat memori." },
      sha512: { sig: "kripto.sha512(teks)", doc: "Menghasilkan heksadesimal hash SHA-512 (128 karakter)." },
      sha512Berkas: { sig: "kripto.sha512Berkas(jalur)", doc: "Menghitung hash SHA-512 dari sebuah berkas secara hemat memori." },
      md5: { sig: "kripto.md5(teks)", doc: "Menghasilkan heksadesimal hash MD5." },
      md5Berkas: { sig: "kripto.md5Berkas(jalur)", doc: "Menghitung hash MD5 dari sebuah berkas." },
      hmacSha256: { sig: "kripto.hmacSha256(kunci, pesan)", doc: "Menghasilkan tanda tangan HMAC berbasis SHA-256." },
      kodekanBase64: { sig: "kripto.kodekanBase64(teks)", doc: "Mengonversi teks ke format Base64." },
      dekodekanBase64: { sig: "kripto.dekodekanBase64(str)", doc: "Mendekode string Base64 kembali ke teks asli." },
      buatJWT: { sig: "kripto.buatJWT(payload, rahasia, [expDetik])", doc: "Membuat token JWT bertanda tangan HMAC-SHA256." },
      verifikasiJWT: { sig: "kripto.verifikasiJWT(token, rahasia)", doc: "Memverifikasi token JWT dan mengembalikan peta payload." },
      hashSandi: { sig: "kripto.hashSandi(sandi)", doc: "Menghasilkan hash sandi PBKDF2-SHA512 dengan salt acak." },
      cocokkanSandi: { sig: "kripto.cocokkanSandi(sandi, hash)", doc: "Memverifikasi kecocokan sandi terhadap hash PBKDF2-SHA512." },
      tokenAcak: { sig: "kripto.tokenAcak(panjang)", doc: "Menghasilkan string heksadesimal acak kriptografis." },
      acakBita: { sig: "kripto.acakBita(panjang)", doc: "Alias ringkas untuk menghasilkan token bita acak kriptografis." },
      uuid: { sig: "kripto.uuid()", doc: "Membuat string UUID v4 acak murni." }
    }
  },
  berkas: {
    doc: "Modul manipulasi file dan direktori pada filesystem lokal.",
    methods: {
      baca: { sig: "berkas.baca(jalur)", doc: "Membaca seluruh isi berkas menjadi teks string." },
      tulis: { sig: "berkas.tulis(jalur, isi)", doc: "Menulis atau menimpa isi ke dalam berkas." },
      tambah: { sig: "berkas.tambah(jalur, isi)", doc: "Menambahkan teks ke baris akhir berkas (append)." },
      ada: { sig: "berkas.ada(jalur)", doc: "Memeriksa apakah berkas atau folder tertentu eksis." },
      hapus: { sig: "berkas.hapus(jalur)", doc: "Menghapus berkas dari sistem." }
    }
  },
  jaringan: {
    doc: "Modul HTTP Client dan konektivitas jaringan web.",
    methods: {
      ambil: { sig: "jaringan.ambil(url)", doc: "Melakukan HTTP GET request mentah." },
      ambilJSON: { sig: "jaringan.ambilJSON(url)", doc: "Melakukan HTTP GET dan langsung mengurai payload JSON." },
      kirimJSON: { sig: "jaringan.kirimJSON(url, data)", doc: "Melakukan HTTP POST payload JSON ke server tujuan." }
    }
  },
  enkoding: {
    doc: "Modul konversi format data CSV, URL Escape, dan Heksadesimal.",
    methods: {
      uraiCSV: { sig: "enkoding.uraiCSV(teks)", doc: "Mem-parse teks CSV menjadi larik 2-dimensi." },
      kodekanCSV: { sig: "enkoding.kodekanCSV(larik2D)", doc: "Mengonversi matriks 2D menjadi teks CSV." },
      kodekanURL: { sig: "enkoding.kodekanURL(teks)", doc: "Meng-encode karakter string menjadi format URL Escape." },
      kodekanHeksa: { sig: "enkoding.kodekanHeksa(teks)", doc: "Mengonversi teks menjadi string heksadesimal biner." }
    }
  },
  bita: {
    doc: "Modul penanganan karakter Unicode rune dan pengindeksan bita string.",
    methods: {
      karakter: { sig: "bita.karakter(str)", doc: "Mengonversi string ke larik integer kode rune Unicode." },
      dariKarakter: { sig: "bita.dariKarakter(runes)", doc: "Mengubah larik kode rune kembali ke teks string asli." },
      jumlahKarakter: { sig: "bita.jumlahKarakter(str)", doc: "Menghitung jumlah karakter visual asli Unicode." }
    }
  },
  atomik: {
    doc: "Modul operasi konkurensi atomic berkinerja tinggi tanpa kunci mutex.",
    methods: {
      tambah: { sig: "atomik.tambah(namaVar, delta)", doc: "Menambahkan nilai integer secara atomik." },
      baca: { sig: "atomik.baca(namaVar)", doc: "Membaca nilai integer secara atomik." },
      ambil: { sig: "atomik.ambil(namaVar)", doc: "Alias untuk membaca nilai integer secara atomik." },
      simpan: { sig: "atomik.simpan(namaVar, nilai)", doc: "Menyimpan nilai integer secara atomik." },
      tukar: { sig: "atomik.tukar(namaVar, lama, baru)", doc: "Operasi Compare-And-Swap (CAS) secara atomik." }
    }
  },
  konteks: {
    doc: "Modul manajemen batas waktu dan pembatalan konkurensi (ala Go context) serta objek konteks HTTP.",
    methods: {
      latarBelakang: { sig: "konteks.latarBelakang()", doc: "Membuat konteks dasar kosong (context.Background)." },
      denganBatal: { sig: "konteks.denganBatal()", doc: "Membuat konteks yang dapat dibatalkan secara manual (mengembalikan ktk, batal)." },
      denganBatasWaktu: { sig: "konteks.denganBatasWaktu(milidetik)", doc: "Membuat konteks dengan batas waktu otomatis dalam milidetik (mengembalikan ktk, batal)." },
      kepala: { sig: "konteks.kepala(kunci, [nilai])", doc: "Membaca kepala HTTP permintaan atau mengatur kepala HTTP respons." },
      kaki: { sig: "konteks.kaki(kunci, [nilai])", doc: "Membaca atau mengatur kaki (trailer) HTTP." },
      kue: { sig: "konteks.kue(nama, [nilai], [opsi])", doc: "Membaca kue HTTP permintaan atau mengatur kue HTTP respons." },
      kueri: { sig: "konteks.kueri(kunci, [bawaan])", doc: "Mengambil parameter query URL dengan nilai bawaan opsional." },
      param: { sig: "konteks.param(kunci, [bawaan])", doc: "Mengambil parameter rute dinamis (:id)." },
      ikat: { sig: "konteks.ikat()", doc: "Mengikat (bind) badan JSON, form, atau kueri permintaan secara otomatis ke peta." },
      taruh: { sig: "konteks.taruh(kunci, nilai)", doc: "Menyimpan data state per-permintaan di dalam konteks." },
      ambil: { sig: "konteks.ambil(kunci, [bawaan])", doc: "Mengambil data state per-permintaan dari konteks." },
      json: { sig: "konteks.json(data, [kodeStatus])", doc: "Mengirimkan respons berformat JSON." },
      teks: { sig: "konteks.teks(isi, [kodeStatus])", doc: "Mengirimkan respons teks polos." },
      html: { sig: "konteks.html(isi, [kodeStatus])", doc: "Mengirimkan respons dokumen HTML." },
      status: { sig: "konteks.status(kode)", doc: "Mengatur kode status HTTP respons." }
    }
  }
};

const KEYWORDS_DOCS = {
  paket: "Mendeklarasikan nama paket berkas sumber (contoh: `paket utama`).",
  pakai: "Mengimpor modul pustaka standar bawaan atau berkas modul lokal (contoh: `pakai \"cetak\"`).",
  fungsi: "Mendefinisikan fungsi atau metode receiver struct.",
  var: "Mendeklarasikan variabel baru.",
  tetap: "Mendeklarasikan konstanta tetap.",
  tipe: "Mendeklarasikan tipe baru (struktur atau antarmuka).",
  struktur: "Mendefinisikan bentuk data struct ala Go.",
  antarmuka: "Mendefinisikan antarmuka (interface) dengan polimorfisme struktural implisit ala Go.",
  jika: "Pernyataan kondisional jika (if statement).",
  lainnya: "Cabang alternatif kondisional (else).",
  untuk: "Pernyataan perulangan (standard for, while-style, range loop, atau channel drain loop).",
  pilih: "Pernyataan seleksi nilai (switch) atau channel multiplexer (select).",
  kasus: "Klausa kondisi kasus dalam blok `pilih`.",
  bawaan: "Klausa default jika tidak ada kasus yang cocok atau kanal belum siap.",
  teruskan: "Melanjutkan eksekusi langsung ke kasus berikutnya (*fallthrough*).",
  kembalikan: "Mengembalikan nilai dari suatu fungsi.",
  tunda: "Menunda eksekusi fungsi hingga fungsi pembungkus selesai (LIFO defer ala Go).",
  ku: "Meluncurkan pekerjaan secara paralel dalam thread ringan Ku-routine (Goroutine).",
  panik: "Memicu kondisi fatal atau panik runtime (panic ala Go).",
  pulih: "Menangkap dan memulihkan eksekusi dari kondisi panik (recover ala Go).",
  buat: "Fungsi bawaan universal untuk mengalokasikan larik, kanal, atau peta (make ala Go).",
  tutup: "Menutup kanal komunikasi setelah transmisi data selesai (close ala Go).",
  tulis: "Fungsi bawaan langsung untuk mencetak data/teks ke baris perintah terminal.",
  tanya: "Fungsi bawaan untuk membaca masukan/prompt teks pengguna dari baris perintah terminal.",
  sekarang: "Fungsi bawaan mengambil objek waktu dan tanggal saat ini.",
  perintah: "Fungsi bawaan mengeksekusi perintah shell / subprocess sistem operasi.",
  baca: "Fungsi bawaan untuk membaca seluruh isi berkas teks lokal.",
  ambil: "Fungsi bawaan untuk melakukan HTTP GET request.",
  rentang: "Fungsi bawaan menghasilkan deret larik integer berurutan (start, end, [step]).",
  kunci: "Fungsi bawaan mengambil seluruh kunci nama properti peta.",
  nilai: "Fungsi bawaan mengambil seluruh nilai elemen peta.",
  tidur: "Fungsi bawaan untuk menghentikan rutin sementara selama N milidetik.",
  format: "Fungsi bawaan pemformat string teks pola sprintf / placeholder.",
  urai: "Fungsi bawaan mem-parse teks string JSON menjadi Peta atau Larik.",
  kodekan: "Fungsi bawaan mengonversi Peta, Larik, atau Struct ke format teks JSON string.",
  keluar: "Fungsi bawaan mengakhiri eksekusi program dengan kode keluar (exit code).",
  panjang: "Mengembalikan panjang larik, string, atau peta (len ala Go).",
  kapasitas: "Mengembalikan kapasitas alokasi memori larik atau buffer kanal (cap ala Go)."
};

function activate(context) {
  // 1. Completion Provider (Autocomplete)
  const completionProvider = vscode.languages.registerCompletionItemProvider('ku', {
    provideCompletionItems(document, position) {
      const linePrefix = document.lineAt(position).text.substr(0, position.character);
      const items = [];

      // Cek apakah ada notasi dot: misal `cetak.` atau `basisdata.`
      const dotMatch = linePrefix.match(/([a-zA-Z0-9_]+)\.$/);
      if (dotMatch) {
        const modName = dotMatch[1];
        if (MODULES_DATA[modName]) {
          const mod = MODULES_DATA[modName];
          for (const [mName, mInfo] of Object.entries(mod.methods)) {
            const item = new vscode.CompletionItem(mName, vscode.CompletionItemKind.Method);
            item.detail = mInfo.sig;
            item.documentation = new vscode.MarkdownString(mInfo.doc);
            item.insertText = new vscode.SnippetString(mName + '(${1})');
            items.push(item);
          }
          return items;
        }
      }

      // Tampilkan modul bawaan
      for (const [mName, mInfo] of Object.entries(MODULES_DATA)) {
        const item = new vscode.CompletionItem(mName, vscode.CompletionItemKind.Module);
        item.detail = "Modul Bawaan: " + mName;
        item.documentation = new vscode.MarkdownString(mInfo.doc);
        items.push(item);
      }

      // Tampilkan kata kunci
      for (const [kw, doc] of Object.entries(KEYWORDS_DOCS)) {
        const item = new vscode.CompletionItem(kw, vscode.CompletionItemKind.Keyword);
        item.detail = "Kata Kunci Bahasa Ku";
        item.documentation = new vscode.MarkdownString(doc);
        items.push(item);
      }

      return items;
    }
  }, '.');

  // 2. Hover Provider (Dokumentasi kursor melayang)
  const hoverProvider = vscode.languages.registerHoverProvider('ku', {
    provideHover(document, position) {
      const range = document.getWordRangeAtPosition(position);
      if (!range) return;
      const word = document.getText(range);

      // Cek apakah kata adalah kata kunci
      if (KEYWORDS_DOCS[word]) {
        return new vscode.Hover(new vscode.MarkdownString(`**Bahasa Ku Keyword: \`${word}\`**\n\n${KEYWORDS_DOCS[word]}`));
      }

      // Cek apakah kata adalah nama modul
      if (MODULES_DATA[word]) {
        return new vscode.Hover(new vscode.MarkdownString(`**Modul Bawaan: \`${word}\`**\n\n${MODULES_DATA[word].doc}`));
      }

      // Cek apakah kursor berada di method modul: modul.metode
      const lineText = document.lineAt(position).text;
      const beforeWord = lineText.substring(0, range.start.character);
      const dotModMatch = beforeWord.match(/([a-zA-Z0-9_]+)\.$/);
      if (dotModMatch && MODULES_DATA[dotModMatch[1]]) {
        const mod = MODULES_DATA[dotModMatch[1]];
        if (mod.methods[word]) {
          const m = mod.methods[word];
          return new vscode.Hover(new vscode.MarkdownString(`**Metode: \`${m.sig}\`**\n\n${m.doc}`));
        }
      }
    }
  });

  // 3. Command: Jalankan Berkas Bahasa Ku Saat Ini
  const runCommand = vscode.commands.registerCommand('bahasa-ku.jalankanBerkas', () => {
    const editor = vscode.window.activeTextEditor;
    if (!editor) {
      vscode.window.showInformationMessage('Tidak ada berkas .ku yang sedang aktif.');
      return;
    }
    const filePath = editor.document.fileName;
    if (!filePath.endsWith('.ku')) {
      vscode.window.showWarningMessage('Hanya berkas berekstensi .ku yang dapat dijalankan.');
      return;
    }

    const terminal = vscode.window.activeTerminal || vscode.window.createTerminal('Bahasa Ku');
    terminal.show();
    terminal.sendText(`ku jalankan "${filePath}"`);
  });

  context.subscriptions.push(completionProvider, hoverProvider, runCommand);
}

function deactivate() {}

module.exports = {
  activate,
  deactivate
};

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
      tutup: { sig: "jendela.tutup()", doc: "Menutup jendela aplikasi desktop dan membersihkan koneksi basis data." }
    }
  },
  waktu: {
    doc: "Modul penanganan tanggal, jam, pewaktu, dan durasi.",
    methods: {
      sekarang: { sig: "waktu.sekarang()", doc: "Mengambil objek waktu saat ini (Unix timestamp & tanggal)." },
      tidur: { sig: "waktu.tidur(nanodetik)", doc: "Menghentikan rutin sementara selama durasi nanodetik tertentu." },
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
      kueri: { sig: "db.kueri(sql, ...params)", doc: "Menjalankan kueri SQL SELECT dan mengembalikan larik kamus baris." },
      eksekusi: { sig: "db.eksekusi(sql, ...params)", doc: "Menjalankan kueri SQL INSERT/UPDATE/DELETE/DDL." },
      tutup: { sig: "db.tutup()", doc: "Menutup koneksi database." }
    }
  },
  web: {
    doc: "Modul server web HTTP REST API bawaan.",
    methods: {
      dapatkan: { sig: "web.dapatkan(rute, penangan)", doc: "Mendaftarkan endpoint HTTP GET." },
      pos: { sig: "web.pos(rute, penangan)", doc: "Mendaftarkan endpoint HTTP POST." },
      put: { sig: "web.put(rute, penangan)", doc: "Mendaftarkan endpoint HTTP PUT." },
      hapus: { sig: "web.hapus(rute, penangan)", doc: "Mendaftarkan endpoint HTTP DELETE." },
      aktifkanCORS: { sig: "web.aktifkanCORS()", doc: "Mengaktifkan header CORS untuk API publik." },
      jalankan: { sig: "web.jalankan(port)", doc: "Menjalankan server web HTTP pada port tertentu." }
    }
  },
  sistem: {
    doc: "Modul operasi sistem, runtime goroutine, dan sinkronisasi memori.",
    methods: {
      kelompokTunggu: { sig: "sistem.kelompokTunggu()", doc: "Membuat instance WaitGroup untuk menunggu sekumpulan Ku-routine." },
      kunci: { sig: "sistem.kunci()", doc: "Membuat kunci Mutex eksklusif." },
      kunciBacaTulis: { sig: "sistem.kunciBacaTulis()", doc: "Membuat kunci RWMutex bersama." },
      tidur: { sig: "sistem.tidur(milidetik)", doc: "Menghentikan sementara eksekusi rutin selama N milidetik." },
      argumen: { sig: "sistem.argumen()", doc: "Mengambil parameter argumen baris perintah CLI." }
    }
  },
  matematika: {
    doc: "Modul kalkulasi angka, trigonometri, dan fungsi matematika statistik.",
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
    doc: "Modul enkripsi hash dan pengacakan kriptografi aman.",
    methods: {
      sha256: { sig: "kripto.sha256(teks)", doc: "Menghasilkan heksadesimal hash SHA-256." },
      md5: { sig: "kripto.md5(teks)", doc: "Menghasilkan heksadesimal hash MD5." },
      acakBita: { sig: "kripto.acakBita(panjang)", doc: "Menghasilkan string bita acak berkekuatan kriptografi." }
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
      tambah: { sig: "atomik.tambah(ptr, val)", doc: "Menambahkan nilai integer secara atomik." },
      ambil: { sig: "atomik.ambil(ptr)", doc: "Membaca nilai integer secara atomik." },
      simpan: { sig: "atomik.simpan(ptr, val)", doc: "Menyimpan nilai integer secara atomik." }
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

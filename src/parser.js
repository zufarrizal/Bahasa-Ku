// src/parser.js
import { TokenType } from './token.js';
import * as AST from './ast.js';

const TingkatPrioritas = {
  TERENDAH: 1,
  LOGIKA: 2,        // ||, &&
  PERSAMAAN: 3,     // ==, !=
  PERBANDINGAN: 4,  // <, >, <=, >=
  JUMLAH: 5,        // +, -
  KALI: 6,          // *, /, %
  AWALAN: 7,        // -X, !X
  PANGGIL: 8,       // fungsi(X)
  INDEKS: 9         // obj.prop, arr[x]
};

const PRIORITAS_OPERATOR = {
  [TokenType.DAN]: TingkatPrioritas.LOGIKA,
  [TokenType.ATAU]: TingkatPrioritas.LOGIKA,
  [TokenType.SAMA_PERSIS]: TingkatPrioritas.PERSAMAAN,
  [TokenType.TIDAK_SAMA]: TingkatPrioritas.PERSAMAAN,
  [TokenType.KURANG_DARI]: TingkatPrioritas.PERBANDINGAN,
  [TokenType.LEBIH_DARI]: TingkatPrioritas.PERBANDINGAN,
  [TokenType.KURANG_SAMA]: TingkatPrioritas.PERBANDINGAN,
  [TokenType.LEBIH_SAMA]: TingkatPrioritas.PERBANDINGAN,
  [TokenType.TAMBAH]: TingkatPrioritas.JUMLAH,
  [TokenType.KURANG]: TingkatPrioritas.JUMLAH,
  [TokenType.BAGI]: TingkatPrioritas.KALI,
  [TokenType.KALI]: TingkatPrioritas.KALI,
  [TokenType.SISA_BAGI]: TingkatPrioritas.KALI,
  [TokenType.KURUNG_BUKA]: TingkatPrioritas.PANGGIL,
  [TokenType.TITIK]: TingkatPrioritas.INDEKS,
  [TokenType.SIKU_BUKA]: TingkatPrioritas.INDEKS
};

export class Parser {
  constructor(lexer) {
    this.lexer = lexer;
    this.tokenSaatIni = null;
    this.tokenBerikutnya = null;
    this.daftarError = [];

    // Baca dua token awal untuk mengisi tokenSaatIni dan tokenBerikutnya
    this.lanjutkanToken();
    this.lanjutkanToken();
  }

  lanjutkanToken() {
    this.tokenSaatIni = this.tokenBerikutnya;
    this.tokenBerikutnya = this.lexer.tokenBerikutnya();
  }

  apakahTokenSaatIni(tipe) {
    return this.tokenSaatIni && this.tokenSaatIni.tipe === tipe;
  }

  apakahTokenBerikutnya(tipe) {
    return this.tokenBerikutnya && this.tokenBerikutnya.tipe === tipe;
  }

  harusTokenBerikutnya(tipe) {
    if (this.apakahTokenBerikutnya(tipe)) {
      this.lanjutkanToken();
      return true;
    }
    this.catatError(`Diharapkan token '${tipe}', tetapi ditemukan '${this.tokenBerikutnya.tipe}' (${this.tokenBerikutnya.literal}) pada baris ${this.tokenBerikutnya.baris}:${this.tokenBerikutnya.kolom}`);
    return false;
  }

  prioritasBerikutnya() {
    return PRIORITAS_OPERATOR[this.tokenBerikutnya?.tipe] || TingkatPrioritas.TERENDAH;
  }

  prioritasSaatIni() {
    return PRIORITAS_OPERATOR[this.tokenSaatIni?.tipe] || TingkatPrioritas.TERENDAH;
  }

  catatError(pesan) {
    this.daftarError.push(pesan);
  }

  parseProgram() {
    const program = new AST.Program();

    while (!this.apakahTokenSaatIni(TokenType.EOF)) {
      const stmt = this.parsePernyataan();
      if (stmt) {
        program.pernyataan.push(stmt);
      }
      this.lanjutkanToken();
    }

    return program;
  }

  parsePernyataan() {
    // Lewati titik koma berlebih jika ada
    if (this.apakahTokenSaatIni(TokenType.TITIK_KOMA)) {
      return null;
    }

    switch (this.tokenSaatIni.tipe) {
      case TokenType.PAKET:
        return this.parsePaketStatement();
      case TokenType.PAKAI:
        return this.parsePakaiStatement();
      case TokenType.TIPE:
        return this.parseTipeStatement();
      case TokenType.PILIH:
        return this.parsePilihStatement();
      case TokenType.VAR:
        return this.parseVarStatement(false);
      case TokenType.TETAP:
        return this.parseVarStatement(true);
      case TokenType.FUNGSI:
        return this.parseFungsiStatement();
      case TokenType.JIKA:
        return this.parseJikaStatement();
      case TokenType.UNTUK:
      case TokenType.SELAMA:
        return this.parseUntukStatement();
      case TokenType.KEMBALIKAN:
        return this.parseKembalikanStatement();
      case TokenType.TUNDA:
        return this.parseTundaStatement();
      default:
        return this.parseEkspresiAtauPenetapanStatement();
    }
  }

  parseTipeStatement() {
    const token = this.tokenSaatIni;
    if (!this.harusTokenBerikutnya(TokenType.PENGIDENTIFIKASI)) {
      return null;
    }
    const nama = new AST.Pengidentifikasi(this.tokenSaatIni, this.tokenSaatIni.literal);

    if (!this.harusTokenBerikutnya(TokenType.STRUKTUR)) {
      return null;
    }

    if (!this.harusTokenBerikutnya(TokenType.KURAWAL_BUKA)) {
      return null;
    }

    const bidang = [];
    while (!this.apakahTokenBerikutnya(TokenType.KURAWAL_TUTUP) && !this.apakahTokenBerikutnya(TokenType.EOF)) {
      this.lanjutkanToken();
      if (this.apakahTokenSaatIni(TokenType.PENGIDENTIFIKASI)) {
        const namaBidang = this.tokenSaatIni.literal;
        let tipeBidang = 'apapun';
        if (this.apakahTokenBerikutnya(TokenType.PENGIDENTIFIKASI)) {
          this.lanjutkanToken();
          tipeBidang = this.tokenSaatIni.literal;
        }
        bidang.push({ nama: namaBidang, tipe: tipeBidang });
      }
    }

    if (!this.harusTokenBerikutnya(TokenType.KURAWAL_TUTUP)) {
      return null;
    }

    return new AST.TipeStrukturStatement(token, nama, bidang);
  }

  parsePilihStatement() {
    const token = this.tokenSaatIni;
    let ekspresi = null;

    if (!this.apakahTokenBerikutnya(TokenType.KURAWAL_BUKA)) {
      this.lanjutkanToken();
      ekspresi = this.parseEkspresi(TingkatPrioritas.TERENDAH);
    }

    if (!this.harusTokenBerikutnya(TokenType.KURAWAL_BUKA)) {
      return null;
    }

    const daftarKasus = [];
    let bawaan = null;

    while (!this.apakahTokenBerikutnya(TokenType.KURAWAL_TUTUP) && !this.apakahTokenBerikutnya(TokenType.EOF)) {
      this.lanjutkanToken();
      if (this.apakahTokenSaatIni(TokenType.KASUS)) {
        this.lanjutkanToken();
        const nilaiKasus = this.parseEkspresi(TingkatPrioritas.TERENDAH);
        this.harusTokenBerikutnya(TokenType.TITIK_DUA);
        
        const badanKasus = new AST.BlokStatement(this.tokenSaatIni);
        while (
          !this.apakahTokenBerikutnya(TokenType.KASUS) &&
          !this.apakahTokenBerikutnya(TokenType.BAWAAN) &&
          !this.apakahTokenBerikutnya(TokenType.KURAWAL_TUTUP) &&
          !this.apakahTokenBerikutnya(TokenType.EOF)
        ) {
          this.lanjutkanToken();
          const stmt = this.parsePernyataan();
          if (stmt) badanKasus.pernyataan.push(stmt);
        }
        daftarKasus.push({ nilai: nilaiKasus, badan: badanKasus });
      } else if (this.apakahTokenSaatIni(TokenType.BAWAAN)) {
        this.harusTokenBerikutnya(TokenType.TITIK_DUA);
        const badanBawaan = new AST.BlokStatement(this.tokenSaatIni);
        while (
          !this.apakahTokenBerikutnya(TokenType.KASUS) &&
          !this.apakahTokenBerikutnya(TokenType.BAWAAN) &&
          !this.apakahTokenBerikutnya(TokenType.KURAWAL_TUTUP) &&
          !this.apakahTokenBerikutnya(TokenType.EOF)
        ) {
          this.lanjutkanToken();
          const stmt = this.parsePernyataan();
          if (stmt) badanBawaan.pernyataan.push(stmt);
        }
        bawaan = badanBawaan;
      }
    }

    if (!this.harusTokenBerikutnya(TokenType.KURAWAL_TUTUP)) {
      return null;
    }

    return new AST.PilihStatement(token, ekspresi, daftarKasus, bawaan);
  }

  parsePaketStatement() {
    const token = this.tokenSaatIni;
    if (!this.harusTokenBerikutnya(TokenType.PENGIDENTIFIKASI)) {
      return null;
    }
    const nama = this.tokenSaatIni.literal;
    return new AST.PaketStatement(token, nama);
  }

  parsePakaiStatement() {
    const token = this.tokenSaatIni;
    if (!this.harusTokenBerikutnya(TokenType.TEKS)) {
      return null;
    }
    const jalur = this.tokenSaatIni.literal;
    return new AST.PakaiStatement(token, jalur);
  }

  parseVarStatement(isKonstanta) {
    const token = this.tokenSaatIni;
    if (!this.harusTokenBerikutnya(TokenType.PENGIDENTIFIKASI)) {
      return null;
    }
    const nama = new AST.Pengidentifikasi(this.tokenSaatIni, this.tokenSaatIni.literal);

    let nilai = null;
    // Boleh opsional tipe lalu sama dengan atau langsung sama dengan
    if (this.apakahTokenBerikutnya(TokenType.PENGIDENTIFIKASI)) {
      // Ada tipe eksplisit seperti: var x angka = 10
      this.lanjutkanToken();
    }

    if (this.apakahTokenBerikutnya(TokenType.SAMA_DENGAN)) {
      this.lanjutkanToken(); // ke '='
      this.lanjutkanToken(); // ke ekspresi awal
      nilai = this.parseEkspresi(TingkatPrioritas.TERENDAH);
    }

    if (this.apakahTokenBerikutnya(TokenType.TITIK_KOMA)) {
      this.lanjutkanToken();
    }

    return new AST.VarStatement(token, nama, nilai, isKonstanta);
  }

  parseFungsiStatement() {
    const token = this.tokenSaatIni;
    if (!this.harusTokenBerikutnya(TokenType.PENGIDENTIFIKASI)) {
      return null;
    }
    const nama = new AST.Pengidentifikasi(this.tokenSaatIni, this.tokenSaatIni.literal);

    if (!this.harusTokenBerikutnya(TokenType.KURUNG_BUKA)) {
      return null;
    }

    const parameter = this.parseParameterFungsi();

    // Jika ada tipe return (misal: fungsi tambah(a, b) angka { ... })
    if (this.apakahTokenBerikutnya(TokenType.PENGIDENTIFIKASI)) {
      this.lanjutkanToken();
    }

    if (!this.harusTokenBerikutnya(TokenType.KURAWAL_BUKA)) {
      return null;
    }

    const badan = this.parseBlokStatement();
    return new AST.FungsiStatement(token, nama, parameter, badan);
  }

  parseParameterFungsi() {
    const parameter = [];
    if (this.apakahTokenBerikutnya(TokenType.KURUNG_TUTUP)) {
      this.lanjutkanToken();
      return parameter;
    }

    this.lanjutkanToken();
    parameter.push(new AST.Pengidentifikasi(this.tokenSaatIni, this.tokenSaatIni.literal));

    // Jika parameter memiliki tipe (misal x angka)
    if (this.apakahTokenBerikutnya(TokenType.PENGIDENTIFIKASI)) {
      this.lanjutkanToken();
    }

    while (this.apakahTokenBerikutnya(TokenType.KOMA)) {
      this.lanjutkanToken(); // lewati koma
      this.lanjutkanToken(); // baca param berikutnya
      parameter.push(new AST.Pengidentifikasi(this.tokenSaatIni, this.tokenSaatIni.literal));
      if (this.apakahTokenBerikutnya(TokenType.PENGIDENTIFIKASI)) {
        this.lanjutkanToken();
      }
    }

    if (!this.harusTokenBerikutnya(TokenType.KURUNG_TUTUP)) {
      return null;
    }

    return parameter;
  }

  parseBlokStatement() {
    const blok = new AST.BlokStatement(this.tokenSaatIni);
    this.lanjutkanToken();

    while (!this.apakahTokenSaatIni(TokenType.KURAWAL_TUTUP) && !this.apakahTokenSaatIni(TokenType.EOF)) {
      const stmt = this.parsePernyataan();
      if (stmt) {
        blok.pernyataan.push(stmt);
      }
      this.lanjutkanToken();
    }

    return blok;
  }

  parseJikaStatement() {
    const token = this.tokenSaatIni;
    this.lanjutkanToken(); // pindah ke awal ekspresi kondisi
    const kondisi = this.parseEkspresi(TingkatPrioritas.TERENDAH);

    if (!this.harusTokenBerikutnya(TokenType.KURAWAL_BUKA)) {
      return null;
    }

    const konsekuensi = this.parseBlokStatement();
    let alternatif = null;

    if (this.apakahTokenBerikutnya(TokenType.LAINNYA)) {
      this.lanjutkanToken(); // ke LAINNYA
      if (this.apakahTokenBerikutnya(TokenType.JIKA)) {
        this.lanjutkanToken(); // ke JIKA
        alternatif = this.parseJikaStatement();
      } else if (this.harusTokenBerikutnya(TokenType.KURAWAL_BUKA)) {
        alternatif = this.parseBlokStatement();
      }
    }

    return new AST.JikaStatement(token, kondisi, konsekuensi, alternatif);
  }

  parseUntukStatement() {
    const token = this.tokenSaatIni;
    this.lanjutkanToken(); // pindah ke ekspresi pertama setelah 'untuk'

    // Kasus 1: Perulangan sederhana (seperti while): untuk kondisi { ... }
    // Kasus 2: Perulangan 3 klausa Go: untuk i := 0; i < 10; i++ { ... }
    let inisialisasi = null;
    let kondisi = null;
    let langkah = null;

    if (this.apakahTokenSaatIni(TokenType.KURAWAL_BUKA)) {
      // Loop tak terbatas: untuk { ... }
      const badan = this.parseBlokStatement();
      return new AST.UntukStatement(token, null, null, null, badan);
    }

    // Coba deteksi apakah ada ';' di depan
    const ekspresiPertama = this.parseEkspresi(TingkatPrioritas.TERENDAH);

    if (this.apakahTokenBerikutnya(TokenType.TITIK_KOMA)) {
      // Pola 3 klausa: inisialisasi; kondisi; langkah
      inisialisasi = new AST.EkspresiStatement(this.tokenSaatIni, ekspresiPertama);
      this.lanjutkanToken(); // lewati ';'
      this.lanjutkanToken(); // mulai kondisi
      kondisi = this.parseEkspresi(TingkatPrioritas.TERENDAH);
      this.harusTokenBerikutnya(TokenType.TITIK_KOMA);
      this.lanjutkanToken(); // mulai langkah
      langkah = this.parseEkspresiAtauPenetapanStatement();
      if (!this.harusTokenBerikutnya(TokenType.KURAWAL_BUKA)) {
        return null;
      }
    } else if (this.apakahTokenBerikutnya(TokenType.TITIK_DUA_SAMA) || this.apakahTokenBerikutnya(TokenType.SAMA_DENGAN)) {
      // Pola: untuk i := 0; i < 10; i++
      const opToken = this.tokenBerikutnya;
      this.lanjutkanToken(); // ke ':='
      this.lanjutkanToken(); // ke nilai
      const nilai = this.parseEkspresi(TingkatPrioritas.TERENDAH);
      inisialisasi = new AST.PenetapanStatement(opToken, ekspresiPertama, nilai);
      
      this.harusTokenBerikutnya(TokenType.TITIK_KOMA);
      this.lanjutkanToken(); // ke kondisi
      kondisi = this.parseEkspresi(TingkatPrioritas.TERENDAH);

      this.harusTokenBerikutnya(TokenType.TITIK_KOMA);
      this.lanjutkanToken(); // ke langkah
      langkah = this.parseEkspresiAtauPenetapanStatement();

      if (!this.harusTokenBerikutnya(TokenType.KURAWAL_BUKA)) {
        return null;
      }
    } else {
      // Pola while: untuk x < 10 { ... }
      kondisi = ekspresiPertama;
      if (!this.harusTokenBerikutnya(TokenType.KURAWAL_BUKA)) {
        return null;
      }
    }

    const badan = this.parseBlokStatement();
    return new AST.UntukStatement(token, inisialisasi, kondisi, langkah, badan);
  }

  parseTundaStatement() {
    const token = this.tokenSaatIni;
    this.lanjutkanToken(); // ke ekspresi pemanggilan fungsi
    const ekspresi = this.parseEkspresi(TingkatPrioritas.TERENDAH);
    if (this.apakahTokenBerikutnya(TokenType.TITIK_KOMA)) {
      this.lanjutkanToken();
    }
    return new AST.TundaStatement(token, ekspresi);
  }

  parseKembalikanStatement() {
    const token = this.tokenSaatIni;
    this.lanjutkanToken(); // pindah ke ekspresi yang dikembalikan (jika ada)

    let nilaiKembalian = null;
    if (!this.apakahTokenSaatIni(TokenType.TITIK_KOMA) && !this.apakahTokenSaatIni(TokenType.KURAWAL_TUTUP)) {
      nilaiKembalian = this.parseEkspresi(TingkatPrioritas.TERENDAH);
    }

    if (this.apakahTokenBerikutnya(TokenType.TITIK_KOMA)) {
      this.lanjutkanToken();
    }

    return new AST.KembalikanStatement(token, nilaiKembalian);
  }

  parseEkspresiAtauPenetapanStatement() {
    const ekspresi = this.parseEkspresi(TingkatPrioritas.TERENDAH);

    // Cek apakah ini penetapan := atau =
    if (this.apakahTokenBerikutnya(TokenType.TITIK_DUA_SAMA) || this.apakahTokenBerikutnya(TokenType.SAMA_DENGAN)) {
      this.lanjutkanToken();
      const opToken = this.tokenSaatIni;
      this.lanjutkanToken();
      const nilai = this.parseEkspresi(TingkatPrioritas.TERENDAH);
      return new AST.PenetapanStatement(opToken, ekspresi, nilai);
    }

    // Cek postfix ++ atau -- (seperti i++)
    if (this.apakahTokenBerikutnya(TokenType.TAMBAH_SATU)) {
      this.lanjutkanToken();
      const tok = this.tokenSaatIni;
      const nilai = new AST.OperasiBinerEkspresi(tok, ekspresi, '+', new AST.AngkaLiteral(tok, 1));
      return new AST.PenetapanStatement(tok, ekspresi, nilai);
    } else if (this.apakahTokenBerikutnya(TokenType.KURANG_SATU)) {
      this.lanjutkanToken();
      const tok = this.tokenSaatIni;
      const nilai = new AST.OperasiBinerEkspresi(tok, ekspresi, '-', new AST.AngkaLiteral(tok, 1));
      return new AST.PenetapanStatement(tok, ekspresi, nilai);
    }

    if (this.apakahTokenBerikutnya(TokenType.TITIK_KOMA)) {
      this.lanjutkanToken();
    }

    return new AST.EkspresiStatement(this.tokenSaatIni, ekspresi);
  }

  parseEkspresi(prioritas) {
    let kiri = this.parseEkspresiAwalan();

    while (!this.apakahTokenBerikutnya(TokenType.TITIK_KOMA) && prioritas < this.prioritasBerikutnya()) {
      switch (this.tokenBerikutnya.tipe) {
        case TokenType.TAMBAH:
        case TokenType.KURANG:
        case TokenType.BAGI:
        case TokenType.KALI:
        case TokenType.SISA_BAGI:
        case TokenType.SAMA_PERSIS:
        case TokenType.TIDAK_SAMA:
        case TokenType.KURANG_DARI:
        case TokenType.LEBIH_DARI:
        case TokenType.KURANG_SAMA:
        case TokenType.LEBIH_SAMA:
        case TokenType.DAN:
        case TokenType.ATAU:
          this.lanjutkanToken();
          kiri = this.parseOperasiBiner(kiri);
          break;
        case TokenType.KURUNG_BUKA:
          this.lanjutkanToken();
          kiri = this.parsePemanggilanEkspresi(kiri);
          break;
        case TokenType.TITIK:
          this.lanjutkanToken();
          kiri = this.parseAksesTitik(kiri);
          break;
        case TokenType.SIKU_BUKA:
          this.lanjutkanToken();
          kiri = this.parseAksesIndeks(kiri);
          break;
        default:
          return kiri;
      }
    }

    return kiri;
  }

  parseEkspresiAwalan() {
    switch (this.tokenSaatIni.tipe) {
      case TokenType.PENGIDENTIFIKASI:
        return new AST.Pengidentifikasi(this.tokenSaatIni, this.tokenSaatIni.literal);
      case TokenType.ANGKA:
        return new AST.AngkaLiteral(this.tokenSaatIni, this.tokenSaatIni.literal);
      case TokenType.TEKS:
        return new AST.TeksLiteral(this.tokenSaatIni, this.tokenSaatIni.literal);
      case TokenType.BENAR:
        return new AST.BooleanLiteral(this.tokenSaatIni, true);
      case TokenType.SALAH:
        return new AST.BooleanLiteral(this.tokenSaatIni, false);
      case TokenType.KOSONG:
        return new AST.KosongLiteral(this.tokenSaatIni);
      case TokenType.SIKU_BUKA:
        return this.parseArrayLiteral();
      case TokenType.KURAWAL_BUKA:
        return this.parsePetaLiteral();
      case TokenType.BUKAN:
      case TokenType.KURANG: {
        const token = this.tokenSaatIni;
        const operator = token.literal;
        this.lanjutkanToken();
        const kanan = this.parseEkspresi(TingkatPrioritas.AWALAN);
        return new AST.OperasiAwalEkspresi(token, operator, kanan);
      }
      case TokenType.KURUNG_BUKA: {
        this.lanjutkanToken();
        const ekspresi = this.parseEkspresi(TingkatPrioritas.TERENDAH);
        if (!this.harusTokenBerikutnya(TokenType.KURUNG_TUTUP)) {
          return null;
        }
        return ekspresi;
      }
      default:
        this.catatError(`Tidak ada penanganan awalan untuk token '${this.tokenSaatIni.literal}'`);
        return null;
    }
  }

  parseOperasiBiner(kiri) {
    const token = this.tokenSaatIni;
    const operator = token.literal;
    const prioritas = this.prioritasSaatIni();
    this.lanjutkanToken();
    const kanan = this.parseEkspresi(prioritas);
    return new AST.OperasiBinerEkspresi(token, kiri, operator, kanan);
  }

  parsePemanggilanEkspresi(fungsi) {
    const token = this.tokenSaatIni; // '('
    const argumen = [];

    if (!this.apakahTokenBerikutnya(TokenType.KURUNG_TUTUP)) {
      this.lanjutkanToken();
      argumen.push(this.parseEkspresi(TingkatPrioritas.TERENDAH));

      while (this.apakahTokenBerikutnya(TokenType.KOMA)) {
        this.lanjutkanToken(); // lewati ','
        this.lanjutkanToken(); // ke ekspresi argumen
        argumen.push(this.parseEkspresi(TingkatPrioritas.TERENDAH));
      }
    }

    if (!this.harusTokenBerikutnya(TokenType.KURUNG_TUTUP)) {
      return null;
    }

    return new AST.PemanggilanEkspresi(token, fungsi, argumen);
  }

  parseArrayLiteral() {
    const token = this.tokenSaatIni; // '['
    const elemen = [];

    if (this.apakahTokenBerikutnya(TokenType.SIKU_TUTUP)) {
      this.lanjutkanToken();
      return new AST.ArrayLiteral(token, elemen);
    }

    this.lanjutkanToken();
    elemen.push(this.parseEkspresi(TingkatPrioritas.TERENDAH));

    while (this.apakahTokenBerikutnya(TokenType.KOMA)) {
      this.lanjutkanToken(); // lewati ','
      if (this.apakahTokenBerikutnya(TokenType.SIKU_TUTUP)) break;
      this.lanjutkanToken();
      elemen.push(this.parseEkspresi(TingkatPrioritas.TERENDAH));
    }

    if (!this.harusTokenBerikutnya(TokenType.SIKU_TUTUP)) {
      return null;
    }

    return new AST.ArrayLiteral(token, elemen);
  }

  parsePetaLiteral() {
    const token = this.tokenSaatIni; // '{'
    const pasangan = [];

    if (this.apakahTokenBerikutnya(TokenType.KURAWAL_TUTUP)) {
      this.lanjutkanToken();
      return new AST.PetaLiteral(token, pasangan);
    }

    while (!this.apakahTokenBerikutnya(TokenType.KURAWAL_TUTUP) && !this.apakahTokenBerikutnya(TokenType.EOF)) {
      this.lanjutkanToken();
      let kunci;
      if (this.apakahTokenSaatIni(TokenType.TEKS)) {
        kunci = new AST.TeksLiteral(this.tokenSaatIni, this.tokenSaatIni.literal);
      } else if (this.apakahTokenSaatIni(TokenType.PENGIDENTIFIKASI)) {
        kunci = new AST.TeksLiteral(this.tokenSaatIni, this.tokenSaatIni.literal);
      } else {
        kunci = this.parseEkspresi(TingkatPrioritas.TERENDAH);
      }

      this.harusTokenBerikutnya(TokenType.TITIK_DUA);
      this.lanjutkanToken();
      const nilai = this.parseEkspresi(TingkatPrioritas.TERENDAH);
      pasangan.push({ kunci, nilai });

      if (this.apakahTokenBerikutnya(TokenType.KOMA)) {
        this.lanjutkanToken();
      }
    }

    if (!this.harusTokenBerikutnya(TokenType.KURAWAL_TUTUP)) {
      return null;
    }

    return new AST.PetaLiteral(token, pasangan);
  }

  parseAksesIndeks(kiri) {
    const token = this.tokenSaatIni; // '['
    this.lanjutkanToken();
    const indeks = this.parseEkspresi(TingkatPrioritas.TERENDAH);
    if (!this.harusTokenBerikutnya(TokenType.SIKU_TUTUP)) {
      return null;
    }
    return new AST.IndeksEkspresi(token, kiri, indeks);
  }

  parseAksesTitik(objek) {
    const token = this.tokenSaatIni; // '.'
    if (!this.harusTokenBerikutnya(TokenType.PENGIDENTIFIKASI)) {
      return null;
    }
    const properti = new AST.Pengidentifikasi(this.tokenSaatIni, this.tokenSaatIni.literal);
    return new AST.AksesTitikEkspresi(token, objek, properti);
  }
}

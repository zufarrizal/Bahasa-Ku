// src/ast.js

export class Node {
  tipeNode() {
    return this.constructor.name;
  }
}

export class TipeStrukturStatement extends Node {
  constructor(token, nama, bidang) {
    super();
    this.token = token;   // TokenType.TIPE
    this.nama = nama;     // Identifier nama tipe
    this.bidang = bidang; // Array { nama, tipe }
  }
}

export class PilihStatement extends Node {
  constructor(token, ekspresi, daftarKasus, bawaan) {
    super();
    this.token = token;             // TokenType.PILIH
    this.ekspresi = ekspresi;       // Ekspresi yang diuji (atau null jika boolean switch)
    this.daftarKasus = daftarKasus; // Array { nilai: Ekspresi, badan: BlokStatement }
    this.bawaan = bawaan;           // BlokStatement atau null
  }
}

export class ArrayLiteral extends Node {
  constructor(token, elemen) {
    super();
    this.token = token;   // TokenType.SIKU_BUKA
    this.elemen = elemen; // Array of Ekspresi
  }
}

export class PetaLiteral extends Node {
  constructor(token, pasangan) {
    super();
    this.token = token;     // TokenType.KURAWAL_BUKA
    this.pasangan = pasangan; // Array of { kunci: Ekspresi, nilai: Ekspresi }
  }
}

export class IndeksEkspresi extends Node {
  constructor(token, kiri, indeks) {
    super();
    this.token = token;   // TokenType.SIKU_BUKA
    this.kiri = kiri;     // Ekspresi
    this.indeks = indeks; // Ekspresi
  }
}

export class Program extends Node {
  constructor() {
    super();
    this.pernyataan = []; // daftar statement
  }
}

export class PaketStatement extends Node {
  constructor(token, nama) {
    super();
    this.token = token; // token TokenType.PAKET
    this.nama = nama;   // pengidentifikasi paket (misal "utama")
  }
}

export class PakaiStatement extends Node {
  constructor(token, jalur) {
    super();
    this.token = token; // token TokenType.PAKAI
    this.jalur = jalur; // teks path / modul (misal "cetak")
  }
}

export class VarStatement extends Node {
  constructor(token, nama, nilai, konstanta = false) {
    super();
    this.token = token;       // TokenType.VAR / TETAP
    this.nama = nama;         // Identifier nama variabel
    this.nilai = nilai;       // Ekspresi nilai
    this.konstanta = konstanta;
  }
}

export class PenetapanStatement extends Node {
  constructor(token, target, nilai) {
    super();
    this.token = token;   // TokenType.SAMA_DENGAN atau TITIK_DUA_SAMA
    this.target = target; // Identifier / Akses
    this.nilai = nilai;   // Ekspresi nilai
    this.deklarasiBaru = token.tipe === ':=';
  }
}

export class BlokStatement extends Node {
  constructor(token) {
    super();
    this.token = token; // TokenType.KURAWAL_BUKA
    this.pernyataan = [];
  }
}

export class FungsiStatement extends Node {
  constructor(token, nama, parameter, badan) {
    super();
    this.token = token;       // TokenType.FUNGSI
    this.nama = nama;         // Identifier nama fungsi
    this.parameter = parameter; // array of Identifier
    this.badan = badan;       // BlokStatement
  }
}

export class JikaStatement extends Node {
  constructor(token, kondisi, konsekuensi, alternatif = null) {
    super();
    this.token = token;           // TokenType.JIKA
    this.kondisi = kondisi;       // Ekspresi kondisi
    this.konsekuensi = konsekuensi; // BlokStatement
    this.alternatif = alternatif;   // BlokStatement atau JikaStatement
  }
}

export class UntukStatement extends Node {
  constructor(token, inisialisasi, kondisi, langkah, badan) {
    super();
    this.token = token;               // TokenType.UNTUK / SELAMA
    this.inisialisasi = inisialisasi; // Statement atau null
    this.kondisi = kondisi;           // Ekspresi atau null
    this.langkah = langkah;           // Statement atau null
    this.badan = badan;               // BlokStatement
  }
}

export class TundaStatement extends Node {
  constructor(token, ekspresi) {
    super();
    this.token = token;       // TokenType.TUNDA
    this.ekspresi = ekspresi; // Ekspresi pemanggilan fungsi yang ditunda
  }
}

export class KembalikanStatement extends Node {
  constructor(token, nilaiKembalian) {
    super();
    this.token = token; // TokenType.KEMBALIKAN
    this.nilaiKembalian = nilaiKembalian;
  }
}

export class EkspresiStatement extends Node {
  constructor(token, ekspresi) {
    super();
    this.token = token;
    this.ekspresi = ekspresi;
  }
}

export class PemanggilanEkspresi extends Node {
  constructor(token, fungsi, argumen) {
    super();
    this.token = token;     // TokenType.KURUNG_BUKA
    this.fungsi = fungsi;   // Identifier atau AksesTitikEkspresi
    this.argumen = argumen; // array of Ekspresi
  }
}

export class OperasiBinerEkspresi extends Node {
  constructor(token, kiri, operator, kanan) {
    super();
    this.token = token;
    this.kiri = kiri;
    this.operator = operator;
    this.kanan = kanan;
  }
}

export class OperasiAwalEkspresi extends Node {
  constructor(token, operator, kanan) {
    super();
    this.token = token;
    this.operator = operator;
    this.kanan = kanan;
  }
}

export class AksesTitikEkspresi extends Node {
  constructor(token, objek, properti) {
    super();
    this.token = token;       // TokenType.TITIK
    this.objek = objek;       // Identifier
    this.properti = properti; // Identifier
  }
}

export class Pengidentifikasi extends Node {
  constructor(token, nilai) {
    super();
    this.token = token;
    this.nilai = nilai;
  }
}

export class AngkaLiteral extends Node {
  constructor(token, nilai) {
    super();
    this.token = token;
    this.nilai = Number(nilai);
  }
}

export class TeksLiteral extends Node {
  constructor(token, nilai) {
    super();
    this.token = token;
    this.nilai = nilai;
  }
}

export class BooleanLiteral extends Node {
  constructor(token, nilai) {
    super();
    this.token = token;
    this.nilai = nilai; // true atau false
  }
}

export class KosongLiteral extends Node {
  constructor(token) {
    super();
    this.token = token;
    this.nilai = null;
  }
}

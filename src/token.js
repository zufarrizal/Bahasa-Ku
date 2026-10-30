// src/token.js
export const TokenType = {
  // Kata Kunci (Keywords)
  PAKET: 'PAKET',         // paket
  PAKAI: 'PAKAI',         // pakai
  FUNGSI: 'FUNGSI',       // fungsi
  VAR: 'VAR',             // var / variabel
  TETAP: 'TETAP',         // tetap / konst
  TIPE: 'TIPE',           // tipe
  STRUKTUR: 'STRUKTUR',   // struktur
  PILIH: 'PILIH',         // pilih
  KASUS: 'KASUS',         // kasus
  BAWAAN: 'BAWAAN',       // bawaan
  JIKA: 'JIKA',           // jika
  LAINNYA: 'LAINNYA',     // lainnya
  UNTUK: 'UNTUK',         // untuk
  SELAMA: 'SELAMA',       // selama
  KEMBALIKAN: 'KEMBALIKAN', // kembalikan
  BENAR: 'BENAR',         // benar
  SALAH: 'SALAH',         // salah
  KOSONG: 'KOSONG',       // kosong / nihil
  HENTI: 'HENTI',         // henti
  LANJUT: 'LANJUT',       // lanjut
  TUNDA: 'TUNDA',         // tunda (defer)

  // Identifier & Literal
  PENGIDENTIFIKASI: 'PENGIDENTIFIKASI', // nama variabel/fungsi
  ANGKA: 'ANGKA',                       // 123, 3.14
  TEKS: 'TEKS',                         // "halo"

  // Operator
  SAMA_DENGAN: '=',
  TITIK_DUA_SAMA: ':=',
  TAMBAH: '+',
  KURANG: '-',
  KALI: '*',
  BAGI: '/',
  SISA_BAGI: '%',
  TAMBAH_SATU: '++',
  KURANG_SATU: '--',

  // Pembanding & Logika
  SAMA_PERSIS: '==',
  TIDAK_SAMA: '!=',
  KURANG_DARI: '<',
  LEBIH_DARI: '>',
  KURANG_SAMA: '<=',
  LEBIH_SAMA: '>=',
  DAN: '&&',
  ATAU: '||',
  BUKAN: '!',

  // Delimiter
  TITIK: '.',
  KOMA: ',',
  TITIK_KOMA: ';',
  TITIK_DUA: ':',
  KURUNG_BUKA: '(',
  KURUNG_TUTUP: ')',
  KURAWAL_BUKA: '{',
  KURAWAL_TUTUP: '}',
  SIKU_BUKA: '[',
  SIKU_TUTUP: ']',

  // Status
  EOF: 'AKHIR_BERKAS',
  ILLEGAL: 'TIDAK_DIKENAL'
};

export const KATA_KUNCI = {
  'paket': TokenType.PAKET,
  'pakai': TokenType.PAKAI,
  'impor': TokenType.PAKAI,
  'fungsi': TokenType.FUNGSI,
  'var': TokenType.VAR,
  'variabel': TokenType.VAR,
  'tetap': TokenType.TETAP,
  'konst': TokenType.TETAP,
  'tipe': TokenType.TIPE,
  'struktur': TokenType.STRUKTUR,
  'pilih': TokenType.PILIH,
  'kasus': TokenType.KASUS,
  'bawaan': TokenType.BAWAAN,
  'jika': TokenType.JIKA,
  'lainnya': TokenType.LAINNYA,
  'untuk': TokenType.UNTUK,
  'selama': TokenType.SELAMA,
  'kembalikan': TokenType.KEMBALIKAN,
  'benar': TokenType.BENAR,
  'salah': TokenType.SALAH,
  'kosong': TokenType.KOSONG,
  'nihil': TokenType.KOSONG,
  'henti': TokenType.HENTI,
  'lanjut': TokenType.LANJUT,
  'tunda': TokenType.TUNDA
};

export class Token {
  constructor(tipe, literal, baris, kolom) {
    this.tipe = tipe;
    this.literal = literal;
    this.baris = baris;
    this.kolom = kolom;
  }
}

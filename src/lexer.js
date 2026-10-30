// src/lexer.js
import { TokenType, Token, KATA_KUNCI } from './token.js';

export class Lexer {
  constructor(input) {
    this.input = input;
    this.posisi = 0;       // posisi karakter saat ini
    this.posisiBaca = 0;   // posisi pembacaan berikutnya
    this.karakter = '';    // karakter saat ini
    this.baris = 1;
    this.kolom = 0;

    this.bacaKarakter();
  }

  bacaKarakter() {
    if (this.posisiBaca >= this.input.length) {
      this.karakter = '\0';
    } else {
      this.karakter = this.input[this.posisiBaca];
    }
    this.posisi = this.posisiBaca;
    this.posisiBaca++;
    this.kolom++;
  }

  intipKarakter() {
    if (this.posisiBaca >= this.input.length) {
      return '\0';
    }
    return this.input[this.posisiBaca];
  }

  lewatiSpasi() {
    while (
      this.karakter === ' ' ||
      this.karakter === '\t' ||
      this.karakter === '\n' ||
      this.karakter === '\r'
    ) {
      if (this.karakter === '\n') {
        this.baris++;
        this.kolom = 0;
      }
      this.bacaKarakter();
    }
  }

  lewatiKomentar() {
    if (this.karakter === '/' && this.intipKarakter() === '/') {
      while (this.karakter !== '\n' && this.karakter !== '\0') {
        this.bacaKarakter();
      }
      this.lewatiSpasi();
      this.lewatiKomentar();
    } else if (this.karakter === '/' && this.intipKarakter() === '*') {
      this.bacaKarakter(); // lewati /
      this.bacaKarakter(); // lewati *
      while (!(this.karakter === '*' && this.intipKarakter() === '/') && this.karakter !== '\0') {
        if (this.karakter === '\n') {
          this.baris++;
          this.kolom = 0;
        }
        this.bacaKarakter();
      }
      this.bacaKarakter(); // lewati *
      this.bacaKarakter(); // lewati /
      this.lewatiSpasi();
      this.lewatiKomentar();
    }
  }

  tokenBerikutnya() {
    this.lewatiSpasi();
    this.lewatiKomentar();

    let tok;
    const barisSaatIni = this.baris;
    const kolomSaatIni = this.kolom;

    switch (this.karakter) {
      case '=':
        if (this.intipKarakter() === '=') {
          this.bacaKarakter();
          tok = new Token(TokenType.SAMA_PERSIS, '==', barisSaatIni, kolomSaatIni);
        } else {
          tok = new Token(TokenType.SAMA_DENGAN, '=', barisSaatIni, kolomSaatIni);
        }
        break;
      case ':':
        if (this.intipKarakter() === '=') {
          this.bacaKarakter();
          tok = new Token(TokenType.TITIK_DUA_SAMA, ':=', barisSaatIni, kolomSaatIni);
        } else {
          tok = new Token(TokenType.TITIK_DUA, ':', barisSaatIni, kolomSaatIni);
        }
        break;
      case '+':
        if (this.intipKarakter() === '+') {
          this.bacaKarakter();
          tok = new Token(TokenType.TAMBAH_SATU, '++', barisSaatIni, kolomSaatIni);
        } else {
          tok = new Token(TokenType.TAMBAH, '+', barisSaatIni, kolomSaatIni);
        }
        break;
      case '-':
        if (this.intipKarakter() === '-') {
          this.bacaKarakter();
          tok = new Token(TokenType.KURANG_SATU, '--', barisSaatIni, kolomSaatIni);
        } else {
          tok = new Token(TokenType.KURANG, '-', barisSaatIni, kolomSaatIni);
        }
        break;
      case '!':
        if (this.intipKarakter() === '=') {
          this.bacaKarakter();
          tok = new Token(TokenType.TIDAK_SAMA, '!=', barisSaatIni, kolomSaatIni);
        } else {
          tok = new Token(TokenType.BUKAN, '!', barisSaatIni, kolomSaatIni);
        }
        break;
      case '*':
        tok = new Token(TokenType.KALI, '*', barisSaatIni, kolomSaatIni);
        break;
      case '/':
        tok = new Token(TokenType.BAGI, '/', barisSaatIni, kolomSaatIni);
        break;
      case '%':
        tok = new Token(TokenType.SISA_BAGI, '%', barisSaatIni, kolomSaatIni);
        break;
      case '<':
        if (this.intipKarakter() === '=') {
          this.bacaKarakter();
          tok = new Token(TokenType.KURANG_SAMA, '<=', barisSaatIni, kolomSaatIni);
        } else {
          tok = new Token(TokenType.KURANG_DARI, '<', barisSaatIni, kolomSaatIni);
        }
        break;
      case '>':
        if (this.intipKarakter() === '=') {
          this.bacaKarakter();
          tok = new Token(TokenType.LEBIH_SAMA, '>=', barisSaatIni, kolomSaatIni);
        } else {
          tok = new Token(TokenType.LEBIH_DARI, '>', barisSaatIni, kolomSaatIni);
        }
        break;
      case '&':
        if (this.intipKarakter() === '&') {
          this.bacaKarakter();
          tok = new Token(TokenType.DAN, '&&', barisSaatIni, kolomSaatIni);
        } else {
          tok = new Token(TokenType.ILLEGAL, '&', barisSaatIni, kolomSaatIni);
        }
        break;
      case '|':
        if (this.intipKarakter() === '|') {
          this.bacaKarakter();
          tok = new Token(TokenType.ATAU, '||', barisSaatIni, kolomSaatIni);
        } else {
          tok = new Token(TokenType.ILLEGAL, '|', barisSaatIni, kolomSaatIni);
        }
        break;
      case ';':
        tok = new Token(TokenType.TITIK_KOMA, ';', barisSaatIni, kolomSaatIni);
        break;
      case ',':
        tok = new Token(TokenType.KOMA, ',', barisSaatIni, kolomSaatIni);
        break;
      case '.':
        tok = new Token(TokenType.TITIK, '.', barisSaatIni, kolomSaatIni);
        break;
      case '(':
        tok = new Token(TokenType.KURUNG_BUKA, '(', barisSaatIni, kolomSaatIni);
        break;
      case ')':
        tok = new Token(TokenType.KURUNG_TUTUP, ')', barisSaatIni, kolomSaatIni);
        break;
      case '{':
        tok = new Token(TokenType.KURAWAL_BUKA, '{', barisSaatIni, kolomSaatIni);
        break;
      case '}':
        tok = new Token(TokenType.KURAWAL_TUTUP, '}', barisSaatIni, kolomSaatIni);
        break;
      case '[':
        tok = new Token(TokenType.SIKU_BUKA, '[', barisSaatIni, kolomSaatIni);
        break;
      case ']':
        tok = new Token(TokenType.SIKU_TUTUP, ']', barisSaatIni, kolomSaatIni);
        break;
      case '"':
      case '`':
        return this.bacaTeks(this.karakter);
      case '\0':
        tok = new Token(TokenType.EOF, '', barisSaatIni, kolomSaatIni);
        break;
      default:
        if (this.apakahHuruf(this.karakter)) {
          const kata = this.bacaPengenal();
          const tipe = KATA_KUNCI[kata] || TokenType.PENGIDENTIFIKASI;
          return new Token(tipe, kata, barisSaatIni, kolomSaatIni);
        } else if (this.apakahAngka(this.karakter)) {
          const angka = this.bacaAngka();
          return new Token(TokenType.ANGKA, angka, barisSaatIni, kolomSaatIni);
        } else {
          tok = new Token(TokenType.ILLEGAL, this.karakter, barisSaatIni, kolomSaatIni);
        }
    }

    this.bacaKarakter();
    return tok;
  }

  bacaPengenal() {
    const posisiAwal = this.posisi;
    while (this.apakahHuruf(this.karakter) || this.apakahAngka(this.karakter)) {
      this.bacaKarakter();
    }
    return this.input.substring(posisiAwal, this.posisi);
  }

  bacaAngka() {
    const posisiAwal = this.posisi;
    while (this.apakahAngka(this.karakter) || (this.karakter === '.' && this.apakahAngka(this.intipKarakter()))) {
      this.bacaKarakter();
    }
    return this.input.substring(posisiAwal, this.posisi);
  }

  bacaTeks(kutip) {
    const barisSaatIni = this.baris;
    const kolomSaatIni = this.kolom;
    this.bacaKarakter(); // lewati tanda kutip buka
    const posisiAwal = this.posisi;

    let nilai = '';
    while (this.karakter !== kutip && this.karakter !== '\0') {
      if (this.karakter === '\\') {
        this.bacaKarakter();
        if (this.karakter === 'n') nilai += '\n';
        else if (this.karakter === 't') nilai += '\t';
        else if (this.karakter === 'r') nilai += '\r';
        else if (this.karakter === '"') nilai += '"';
        else if (this.karakter === '\\') nilai += '\\';
        else nilai += this.karakter;
      } else {
        if (this.karakter === '\n') {
          this.baris++;
          this.kolom = 0;
        }
        nilai += this.karakter;
      }
      this.bacaKarakter();
    }

    this.bacaKarakter(); // lewati tanda kutip tutup
    return new Token(TokenType.TEKS, nilai, barisSaatIni, kolomSaatIni);
  }

  apakahHuruf(ch) {
    return (
      ('a' <= ch && ch <= 'z') ||
      ('A' <= ch && ch <= 'Z') ||
      ch === '_'
    );
  }

  apakahAngka(ch) {
    return '0' <= ch && ch <= '9';
  }
}

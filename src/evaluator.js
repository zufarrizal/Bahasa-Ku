// src/evaluator.js
import * as AST from './ast.js';
import { Lingkungan } from './lingkungan.js';

class NilaiKembalian {
  constructor(nilai) {
    this.nilai = nilai;
  }
}

class FungsiPengguna {
  constructor(nodeFungsi, lingkunganPenutup) {
    this.node = nodeFungsi;
    this.lingkungan = lingkunganPenutup;
  }
}

export class Evaluator {
  constructor() {
    this.lingkunganGlobal = new Lingkungan();
    this.inisialisasiPustakaStandar();
  }

  inisialisasiPustakaStandar() {
    // Paket standar: cetak (mirip fmt Go)
    const modulCetak = {
      tulis: (...argumen) => {
        const pesan = argumen.map(arg => {
          if (arg === null) return 'kosong';
          if (arg === true) return 'benar';
          if (arg === false) return 'salah';
          if (Array.isArray(arg)) return JSON.stringify(arg);
          if (typeof arg === 'object') return JSON.stringify(arg);
          return String(arg);
        }).join(' ');
        console.log(pesan);
        return null;
      },
      tulisBaris: (...argumen) => {
        return modulCetak.tulis(...argumen);
      }
    };

    this.lingkunganGlobal.tetapkan('cetak', modulCetak);

    // Fungsi bawaan Go: panjang (len) & tambah (append)
    this.lingkunganGlobal.tetapkan('panjang', (target) => {
      if (Array.isArray(target) || typeof target === 'string') {
        return target.length;
      }
      if (target && typeof target === 'object') {
        return Object.keys(target).length;
      }
      return 0;
    });

    this.lingkunganGlobal.tetapkan('tambah', (daftar, ...elemen) => {
      if (Array.isArray(daftar)) {
        return [...daftar, ...elemen];
      }
      return daftar;
    });
  }

  evaluasi(node, env = this.lingkunganGlobal) {
    if (!node) return null;

    if (node instanceof AST.Program) {
      let hasil = null;
      for (const stmt of node.pernyataan) {
        hasil = this.evaluasi(stmt, env);
        if (hasil instanceof NilaiKembalian) {
          return hasil.nilai;
        }
      }

      // Cari dan panggil fungsi utama() jika ada (mirip Go)
      const fungsiUtama = env.dapatkan('utama');
      if (fungsiUtama && fungsiUtama instanceof FungsiPengguna) {
        return this.terapkanFungsi(fungsiUtama, [], env);
      }

      return hasil;
    }

    if (node instanceof AST.PaketStatement) {
      // Menyimpan nama paket jika diperlukan
      return null;
    }

    if (node instanceof AST.PakaiStatement) {
      // Modul import (misal "cetak" sudah otomatis diinisialisasi)
      return null;
    }

    if (node instanceof AST.TipeStrukturStatement) {
      // Deklarasi konstruktor struktur
      const pembuatStruktur = (nilaiAwal = {}) => {
        const objek = {};
        for (const f of node.bidang) {
          objek[f.nama] = nilaiAwal[f.nama] !== undefined ? nilaiAwal[f.nama] : null;
        }
        return objek;
      };
      env.tetapkan(node.nama.nilai, pembuatStruktur);
      return pembuatStruktur;
    }

    if (node instanceof AST.PilihStatement) {
      const targetNilai = node.ekspresi ? this.evaluasi(node.ekspresi, env) : true;
      let kecocokanDitemukan = false;

      for (const k of node.daftarKasus) {
        const nilaiKasus = this.evaluasi(k.nilai, env);
        if (targetNilai === nilaiKasus) {
          kecocokanDitemukan = true;
          return this.evaluasi(k.badan, env);
        }
      }

      if (!kecocokanDitemukan && node.bawaan) {
        return this.evaluasi(node.bawaan, env);
      }
      return null;
    }

    if (node instanceof AST.VarStatement) {
      const nilai = node.nilai ? this.evaluasi(node.nilai, env) : null;
      env.tetapkan(node.nama.nilai, nilai);
      return nilai;
    }

    if (node instanceof AST.PenetapanStatement) {
      const nilai = this.evaluasi(node.nilai, env);
      if (node.target instanceof AST.Pengidentifikasi) {
        const nama = node.target.nilai;
        if (node.deklarasiBaru) {
          env.tetapkan(nama, nilai);
        } else {
          const berhasil = env.perbarui(nama, nilai);
          if (!berhasil) {
            env.tetapkan(nama, nilai);
          }
        }
        return nilai;
      } else if (node.target instanceof AST.IndeksEkspresi) {
        const wadah = this.evaluasi(node.target.kiri, env);
        const indeks = this.evaluasi(node.target.indeks, env);
        wadah[indeks] = nilai;
        return nilai;
      } else if (node.target instanceof AST.AksesTitikEkspresi) {
        const wadah = this.evaluasi(node.target.objek, env);
        const properti = node.target.properti.nilai;
        wadah[properti] = nilai;
        return nilai;
      }
      throw new Error(`Target penetapan tidak valid pada baris ${node.token?.baris}`);
    }

    if (node instanceof AST.FungsiStatement) {
      const fn = new FungsiPengguna(node, env);
      env.tetapkan(node.nama.nilai, fn);
      return fn;
    }

    if (node instanceof AST.BlokStatement) {
      let hasil = null;
      for (const stmt of node.pernyataan) {
        hasil = this.evaluasi(stmt, env);
        if (hasil instanceof NilaiKembalian) {
          return hasil;
        }
      }
      return hasil;
    }

    if (node instanceof AST.JikaStatement) {
      const kondisi = this.evaluasi(node.kondisi, env);
      if (this.apakahBenar(kondisi)) {
        return this.evaluasi(node.konsekuensi, env);
      } else if (node.alternatif) {
        return this.evaluasi(node.alternatif, env);
      }
      return null;
    }

    if (node instanceof AST.UntukStatement) {
      const envLoop = new Lingkungan(env);
      if (node.inisialisasi) {
        this.evaluasi(node.inisialisasi, envLoop);
      }

      let hasil = null;
      while (true) {
        if (node.kondisi) {
          const cek = this.evaluasi(node.kondisi, envLoop);
          if (!this.apakahBenar(cek)) break;
        }

        hasil = this.evaluasi(node.badan, envLoop);
        if (hasil instanceof NilaiKembalian) {
          return hasil;
        }

        if (node.langkah) {
          this.evaluasi(node.langkah, envLoop);
        }
      }
      return hasil;
    }

    if (node instanceof AST.KembalikanStatement) {
      const nilai = node.nilaiKembalian ? this.evaluasi(node.nilaiKembalian, env) : null;
      return new NilaiKembalian(nilai);
    }

    if (node instanceof AST.TundaStatement) {
      if (env.tumpukanTunda) {
        env.tumpukanTunda.push({ node: node.ekspresi, env });
      } else {
        // Jika di luar fungsi, simpan di global
        this.tumpukanTundaGlobal = this.tumpukanTundaGlobal || [];
        this.tumpukanTundaGlobal.push({ node: node.ekspresi, env });
      }
      return null;
    }

    if (node instanceof AST.EkspresiStatement) {
      return this.evaluasi(node.ekspresi, env);
    }

    if (node instanceof AST.Pengidentifikasi) {
      const val = env.dapatkan(node.nilai);
      if (val === undefined) {
        throw new Error(`Variabel atau fungsi '${node.nilai}' tidak ditemukan pada baris ${node.token.baris}:${node.token.kolom}`);
      }
      return val;
    }

    if (node instanceof AST.AngkaLiteral || node instanceof AST.TeksLiteral || node instanceof AST.BooleanLiteral) {
      return node.nilai;
    }

    if (node instanceof AST.KosongLiteral) {
      return null;
    }

    if (node instanceof AST.ArrayLiteral) {
      return node.elemen.map(e => this.evaluasi(e, env));
    }

    if (node instanceof AST.PetaLiteral) {
      const hasilObj = {};
      for (const p of node.pasangan) {
        const kunci = this.evaluasi(p.kunci, env);
        const nilai = this.evaluasi(p.nilai, env);
        hasilObj[kunci] = nilai;
      }
      return hasilObj;
    }

    if (node instanceof AST.IndeksEkspresi) {
      const wadah = this.evaluasi(node.kiri, env);
      const indeks = this.evaluasi(node.indeks, env);
      if (wadah !== undefined && wadah !== null) {
        return wadah[indeks];
      }
      return null;
    }

    if (node instanceof AST.OperasiAwalEkspresi) {
      const kanan = this.evaluasi(node.kanan, env);
      if (node.operator === '-') return -kanan;
      if (node.operator === '!') return !this.apakahBenar(kanan);
      throw new Error(`Operator awalan tidak dikenal '${node.operator}'`);
    }

    if (node instanceof AST.OperasiBinerEkspresi) {
      const kiri = this.evaluasi(node.kiri, env);
      const kanan = this.evaluasi(node.kanan, env);

      switch (node.operator) {
        case '+': return kiri + kanan;
        case '-': return kiri - kanan;
        case '*': return kiri * kanan;
        case '/': return kiri / kanan;
        case '%': return kiri % kanan;
        case '==': return kiri === kanan;
        case '!=': return kiri !== kanan;
        case '<': return kiri < kanan;
        case '<=': return kiri <= kanan;
        case '>': return kiri > kanan;
        case '>=': return kiri >= kanan;
        case '&&': return this.apakahBenar(kiri) && this.apakahBenar(kanan);
        case '||': return this.apakahBenar(kiri) || this.apakahBenar(kanan);
        default:
          throw new Error(`Operator biner tidak dikenal '${node.operator}'`);
      }
    }

    if (node instanceof AST.AksesTitikEkspresi) {
      const target = this.evaluasi(node.objek, env);
      const properti = node.properti.nilai;
      if (target && properti in target) {
        return target[properti];
      }
      throw new Error(`Properti '${properti}' tidak ditemukan pada objek pada baris ${node.token.baris}`);
    }

    if (node instanceof AST.PemanggilanEkspresi) {
      const fungsi = this.evaluasi(node.fungsi, env);
      const argumen = node.argumen.map(arg => this.evaluasi(arg, env));
      return this.terapkanFungsi(fungsi, argumen, env);
    }

    throw new Error(`Tipe node AST tidak didukung: ${node.constructor.name}`);
  }

  terapkanFungsi(fn, argumen, envInduk) {
    if (typeof fn === 'function') {
      return fn(...argumen);
    }

    if (fn instanceof FungsiPengguna) {
      const envFungsi = new Lingkungan(fn.lingkungan);
      envFungsi.tumpukanTunda = [];

      fn.node.parameter.forEach((param, indeks) => {
        envFungsi.tetapkan(param.nilai, argumen[indeks] !== undefined ? argumen[indeks] : null);
      });

      let hasil = null;
      try {
        hasil = this.evaluasi(fn.node.badan, envFungsi);
      } finally {
        // Eksekusi tunda LIFO (Last-In First-Out) khas Go defer
        while (envFungsi.tumpukanTunda.length > 0) {
          const itemTunda = envFungsi.tumpukanTunda.pop();
          this.evaluasi(itemTunda.node, itemTunda.env);
        }
      }

      if (hasil instanceof NilaiKembalian) {
        return hasil.nilai;
      }
      return null;
    }

    throw new Error(`Target pemanggilan bukan merupakan fungsi`);
  }

  apakahBenar(obj) {
    if (obj === false || obj === null || obj === undefined || obj === 0 || obj === '') {
      return false;
    }
    return true;
  }
}

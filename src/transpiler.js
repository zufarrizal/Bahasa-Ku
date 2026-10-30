// src/transpiler.js
import * as AST from './ast.js';

export class TranspilerGo {
  constructor() {
    this.indentasi = 0;
  }

  spasi() {
    return '  '.repeat(this.indentasi);
  }

  transpilasi(node) {
    if (!node) return '';

    if (node instanceof AST.Program) {
      let hasil = '';
      for (const stmt of node.pernyataan) {
        hasil += this.transpilasi(stmt) + '\n';
      }
      return hasil.trim() + '\n';
    }

    if (node instanceof AST.PaketStatement) {
      const nama = node.nama === 'utama' ? 'main' : node.nama;
      return `package ${nama}\n`;
    }

    if (node instanceof AST.PakaiStatement) {
      let jalur = node.jalur;
      if (jalur === 'cetak') jalur = 'fmt';
      return `import "${jalur}"\n`;
    }

    if (node instanceof AST.TipeStrukturStatement) {
      let out = `type ${node.nama.nilai} struct {\n`;
      this.indentasi++;
      for (const b of node.bidang) {
        let t = b.tipe;
        if (t === 'angka') t = 'int';
        if (t === 'teks') t = 'string';
        out += `${this.spasi()}${b.nama} ${t}\n`;
      }
      this.indentasi--;
      out += `}\n`;
      return out;
    }

    if (node instanceof AST.FungsiStatement) {
      const nama = node.nama.nilai === 'utama' ? 'main' : node.nama.nilai;
      const param = node.parameter.map(p => `${p.nilai} interface{}`).join(', ');
      let out = `func ${nama}(${param}) interface{} `;
      out += this.transpilasi(node.badan);
      return out + '\n';
    }

    if (node instanceof AST.BlokStatement) {
      let out = '{\n';
      this.indentasi++;
      for (const stmt of node.pernyataan) {
        out += `${this.spasi()}${this.transpilasi(stmt)}\n`;
      }
      this.indentasi--;
      out += `${this.spasi()}}`;
      return out;
    }

    if (node instanceof AST.VarStatement) {
      const nama = node.nama.nilai;
      if (node.nilai) {
        return `var ${nama} = ${this.transpilasi(node.nilai)}`;
      }
      return `var ${nama} interface{}`;
    }

    if (node instanceof AST.PenetapanStatement) {
      const target = this.transpilasi(node.target);
      const nilai = this.transpilasi(node.nilai);
      const op = node.deklarasiBaru ? ':=' : '=';
      return `${target} ${op} ${nilai}`;
    }

    if (node instanceof AST.JikaStatement) {
      let out = `if ${this.transpilasi(node.kondisi)} `;
      out += this.transpilasi(node.konsekuensi);
      if (node.alternatif) {
        out += ` else `;
        out += this.transpilasi(node.alternatif);
      }
      return out;
    }

    if (node instanceof AST.UntukStatement) {
      if (!node.inisialisasi && !node.kondisi && !node.langkah) {
        return `for ${this.transpilasi(node.badan)}`;
      }
      if (!node.inisialisasi && !node.langkah) {
        return `for ${this.transpilasi(node.kondisi)} ${this.transpilasi(node.badan)}`;
      }
      const init = this.transpilasi(node.inisialisasi);
      const cond = this.transpilasi(node.kondisi);
      const step = this.transpilasi(node.langkah);
      return `for ${init}; ${cond}; ${step} ${this.transpilasi(node.badan)}`;
    }

    if (node instanceof AST.PilihStatement) {
      let out = `switch ${node.ekspresi ? this.transpilasi(node.ekspresi) + ' ' : ''}{\n`;
      this.indentasi++;
      for (const k of node.daftarKasus) {
        out += `${this.spasi()}case ${this.transpilasi(k.nilai)}:\n`;
        this.indentasi++;
        for (const stmt of k.badan.pernyataan) {
          out += `${this.spasi()}${this.transpilasi(stmt)}\n`;
        }
        this.indentasi--;
      }
      if (node.bawaan) {
        out += `${this.spasi()}default:\n`;
        this.indentasi++;
        for (const stmt of node.bawaan.pernyataan) {
          out += `${this.spasi()}${this.transpilasi(stmt)}\n`;
        }
        this.indentasi--;
      }
      this.indentasi--;
      out += `${this.spasi()}}`;
      return out;
    }

    if (node instanceof AST.TundaStatement) {
      return `defer ${this.transpilasi(node.ekspresi)}`;
    }

    if (node instanceof AST.KembalikanStatement) {
      if (node.nilaiKembalian) {
        return `return ${this.transpilasi(node.nilaiKembalian)}`;
      }
      return `return`;
    }

    if (node instanceof AST.EkspresiStatement) {
      return this.transpilasi(node.ekspresi);
    }

    if (node instanceof AST.PemanggilanEkspresi) {
      let fn = this.transpilasi(node.fungsi);
      if (fn === 'cetak.tulis' || fn === 'cetak.tulisBaris') {
        fn = 'fmt.Println';
      } else if (fn === 'panjang') {
        fn = 'len';
      } else if (fn === 'tambah') {
        fn = 'append';
      }
      const args = node.argumen.map(a => this.transpilasi(a)).join(', ');
      return `${fn}(${args})`;
    }

    if (node instanceof AST.OperasiBinerEkspresi) {
      return `${this.transpilasi(node.kiri)} ${node.operator} ${this.transpilasi(node.kanan)}`;
    }

    if (node instanceof AST.OperasiAwalEkspresi) {
      return `${node.operator}${this.transpilasi(node.kanan)}`;
    }

    if (node instanceof AST.AksesTitikEkspresi) {
      return `${this.transpilasi(node.objek)}.${node.properti.nilai}`;
    }

    if (node instanceof AST.IndeksEkspresi) {
      return `${this.transpilasi(node.kiri)}[${this.transpilasi(node.indeks)}]`;
    }

    if (node instanceof AST.Pengidentifikasi) {
      return node.nilai;
    }

    if (node instanceof AST.AngkaLiteral) {
      return String(node.nilai);
    }

    if (node instanceof AST.TeksLiteral) {
      return JSON.stringify(node.nilai);
    }

    if (node instanceof AST.BooleanLiteral) {
      return node.nilai ? 'true' : 'false';
    }

    if (node instanceof AST.KosongLiteral) {
      return 'nil';
    }

    if (node instanceof AST.ArrayLiteral) {
      const el = node.elemen.map(e => this.transpilasi(e)).join(', ');
      return `[]interface{}{${el}}`;
    }

    if (node instanceof AST.PetaLiteral) {
      const pas = node.pasangan.map(p => `${this.transpilasi(p.kunci)}: ${this.transpilasi(p.nilai)}`).join(', ');
      return `map[string]interface{}{${pas}}`;
    }

    return '';
  }
}

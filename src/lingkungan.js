// src/lingkungan.js
export class Lingkungan {
  constructor(induk = null) {
    this.toko = new Map();
    this.induk = induk;
  }

  dapatkan(nama) {
    if (this.toko.has(nama)) {
      return this.toko.get(nama);
    }
    if (this.induk !== null) {
      return this.induk.dapatkan(nama);
    }
    return undefined;
  }

  tetapkan(nama, nilai) {
    this.toko.set(nama, nilai);
    return nilai;
  }

  perbarui(nama, nilai) {
    if (this.toko.has(nama)) {
      this.toko.set(nama, nilai);
      return true;
    }
    if (this.induk !== null) {
      return this.induk.perbarui(nama, nilai);
    }
    return false;
  }
}

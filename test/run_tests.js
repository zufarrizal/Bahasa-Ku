// test/run_tests.js
import assert from 'node:assert';
import { jalankanKode } from '../src/index.js';

console.log('Menjalankan pengujian otomatis Bahasa Ku...');

// Uji 1: Operasi Aritmatika & Variabel
{
  const kode = `
    paket utama
    fungsi utama() {
      var a = 10
      var b = 20
      kembalikan a + b
    }
  `;
  const hasil = jalankanKode(kode);
  assert.strictEqual(hasil, 30, 'Penjumlahan harus bernilai 30');
  console.log('✓ Uji 1 Lulus: Operasi Aritmatika & Variabel');
}

// Uji 2: Percabangan jika - lainnya
{
  const kode = `
    paket utama
    fungsi cek(x) {
      jika x > 5 {
        kembalikan "besar"
      } lainnya {
        kembalikan "kecil"
      }
    }
    fungsi utama() {
      kembalikan cek(3)
    }
  `;
  const hasil = jalankanKode(kode);
  assert.strictEqual(hasil, 'kecil', 'Hasil percabangan harus "kecil"');
  console.log('✓ Uji 2 Lulus: Percabangan jika - lainnya');
}

// Uji 3: Perulangan untuk (for loop)
{
  const kode = `
    paket utama
    fungsi utama() {
      var total = 0
      untuk i := 1; i <= 4; i++ {
        total = total + i
      }
      kembalikan total
    }
  `;
  const hasil = jalankanKode(kode);
  assert.strictEqual(hasil, 10, 'Total perulangan 1+2+3+4 harus bernilai 10');
  console.log('✓ Uji 3 Lulus: Perulangan untuk');
}

// Uji 4: Rekursi
{
  const kode = `
    paket utama
    fungsi fib(n) {
      jika n <= 1 {
        kembalikan n
      }
      kembalikan fib(n - 1) + fib(n - 2)
    }
    fungsi utama() {
      kembalikan fib(6)
    }
  `;
  const hasil = jalankanKode(kode);
  assert.strictEqual(hasil, 8, 'Fibonacci ke-6 harus bernilai 8');
  console.log('✓ Uji 4 Lulus: Rekursi');
}

// Uji 5: Array & fungsi panjang, tambah, indeks
{
  const kode = `
    paket utama
    fungsi utama() {
      var arr = [10, 20]
      arr = tambah(arr, 30)
      kembalikan arr[0] + arr[2] + panjang(arr)
    }
  `;
  const hasil = jalankanKode(kode);
  assert.strictEqual(hasil, 43, 'Array [10, 20, 30] -> 10 + 30 + 3 = 43');
  console.log('✓ Uji 5 Lulus: Array, Slice & Bawaan (panjang, tambah)');
}

// Uji 6: Map & Struktur & Pilih (Switch)
{
  const kode = `
    paket utama
    tipe Akun struktur {
      nama string
      saldo angka
    }
    fungsi ujiPilih(status) {
      pilih status {
        kasus "aktif":
          kembalikan 1
        bawaan:
          kembalikan 0
      }
    }
    fungsi utama() {
      var user = Akun({ nama: "Andi", saldo: 5000 })
      var peta = { "x": 100 }
      kembalikan user.saldo + peta["x"] + ujiPilih("aktif")
    }
  `;
  const hasil = jalankanKode(kode);
  assert.strictEqual(hasil, 5101, '5000 + 100 + 1 = 5101');
  console.log('✓ Uji 6 Lulus: Map, Struct, dan Pilih-Kasus');
}

// Uji 7: Tunda (defer - LIFO)
{
  const kode = `
    paket utama
    var jejak = ""
    fungsi coba() {
      tunda {
        jejak = jejak + "T3"
      }
      tunda {
        jejak = jejak + "T2"
      }
      jejak = jejak + "T1"
    }
    fungsi utama() {
      coba()
      kembalikan jejak
    }
  `;
  // uji tunda pemanggilan fungsi sederhana
  const kode2 = `
    paket utama
    var urutan = ""
    fungsi rekam(x) {
      urutan = urutan + x
    }
    fungsi tes() {
      tunda rekam("3")
      tunda rekam("2")
      rekam("1")
    }
    fungsi utama() {
      tes()
      kembalikan urutan
    }
  `;
  const hasil = jalankanKode(kode2);
  assert.strictEqual(hasil, "123", 'Tunda harus dieksekusi secara LIFO (1 -> 2 -> 3)');
  console.log('✓ Uji 7 Lulus: Tunda (defer LIFO)');
}

console.log('\nSemua uji otomatis berhasil lulus 100%! 🎉');

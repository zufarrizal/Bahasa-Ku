#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import { jalankanKode, transpilasiKodeKeGo } from '../src/index.js';

const argumen = process.argv.slice(2);

if (argumen.length === 0) {
  console.log(`Penggunaan CLI Bahasa Ku:
  node bin/ku.js jalankan <nama-berkas.ku>
  node bin/ku.js buat <nama-berkas.ku> [nama-tujuan.go]
  node bin/ku.js versi

Contoh:
  node bin/ku.js jalankan contoh/halo.ku
  node bin/ku.js buat contoh/halo.ku hasil.go
`);
  process.exit(0);
}

const perintah = argumen[0];

if (perintah === 'versi' || perintah === '-v' || perintah === '--version') {
  console.log('Bahasa Ku v0.1.0 (Edisi Bahasa Indonesia)');
  process.exit(0);
}

if (perintah === 'jalankan') {
  const targetBerkas = argumen[1];
  if (!targetBerkas) {
    console.error('Galat: Harap tentukan nama berkas .ku yang ingin dijalankan.');
    process.exit(1);
  }

  const jalurAbsolut = path.resolve(process.cwd(), targetBerkas);
  if (!fs.existsSync(jalurAbsolut)) {
    console.error(`Galat: Berkas '${targetBerkas}' tidak ditemukan.`);
    process.exit(1);
  }

  try {
    const kodeSumber = fs.readFileSync(jalurAbsolut, 'utf-8');
    jalankanKode(kodeSumber);
  } catch (err) {
    console.error(err.message);
    process.exit(1);
  }
} else if (perintah === 'buat' || perintah === 'kompilasi') {
  const targetBerkas = argumen[1];
  if (!targetBerkas) {
    console.error('Galat: Harap tentukan nama berkas .ku yang ingin ditranspilasi ke Go.');
    process.exit(1);
  }

  const jalurAbsolut = path.resolve(process.cwd(), targetBerkas);
  if (!fs.existsSync(jalurAbsolut)) {
    console.error(`Galat: Berkas '${targetBerkas}' tidak ditemukan.`);
    process.exit(1);
  }

  try {
    const kodeSumber = fs.readFileSync(jalurAbsolut, 'utf-8');
    const kodeGo = transpilasiKodeKeGo(kodeSumber);

    const berkasKeluar = argumen[2] || targetBerkas.replace(/\.ku$/, '.go');
    const jalurKeluar = path.resolve(process.cwd(), berkasKeluar);

    fs.writeFileSync(jalurKeluar, kodeGo, 'utf-8');
    console.log(`Berhasil mentranspilasi '${targetBerkas}' -> '${berkasKeluar}'! ✨`);
  } catch (err) {
    console.error(err.message);
    process.exit(1);
  }
} else {
  console.error(`Perintah '${perintah}' tidak dikenal. Gunakan 'node bin/ku.js jalankan <berkas.ku>' atau 'node bin/ku.js buat <berkas.ku>'`);
  process.exit(1);
}

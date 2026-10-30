# Ekstensi Bahasa Ku untuk Visual Studio Code

Ekstensi resmi Visual Studio Code untuk bahasa pemrograman **Bahasa Ku** (`.ku`). Menyediakan penyorotan sintaksis (*syntax highlighting*) cerdas, konfigurasi bahasa, auto-closing kurung & tanda petik, serta ikon berkas resmi.

<p align="center">
  <img src="icon.png" alt="Logo Ekstensi Bahasa Ku" width="128" height="128" />
</p>

---

## ✨ Fitur

- 🎨 **Syntax Highlighting Lengkap:** Mewarnai seluruh kata kunci kontrol (`jika`, `lainnya`, `untuk`, `pilih`, `kembalikan`, `tunda`, `henti`, `berhenti`, `lanjut`, `coba`, `tangkap`, `akhirnya`), deklarasi (`fungsi`, `paket`, `pakai`, `var`, `tetap`, `tipe`), modul bawaan (`cetak`, `waktu`, `pola`, `larik`, `peta`, `kompresi`, `jalur`, `enkoding`, `bita`, `acak`, `teks`, `konversi`, `kripto`, `berkas`, `sistem`, `web`, `jendela`, `basisdata`, `uji`, `json`, `validasi`), konstanta, string, dan komentar.
- 📁 **Ikon Berkas Resmi:** Menampilkan ikon resmi Bahasa Ku di samping setiap file berekstensi `.ku`.
- ⌨️ **Konfigurasi Bahasa Pintar:**
  - Auto-closing tanda kurung `()`, kurung kurawal `{}`, kurung siku `[]`, dan tanda kutip `""` / ` `` `.
  - Komentar baris otomatis dengan tombol pintas (`Ctrl+/`).
  - Indentasi otomatis pada blok kode baru.

---

## 📦 Cara Memasang

### Metode 1: Menggunakan Skrip Otomatis
Jika Anda memasang Bahasa Ku menggunakan skrip instalasi terminal:
```powershell
irm https://raw.githubusercontent.com/zufarrizal/Bahasa-Ku/main/pasang.ps1 | iex
```
Ekstensi ini akan otomatis terpasang ke VS Code di komputer Anda.

### Metode 2: Pemasangan Manual dari VSIX
1. Unduh paket ekstensi **`bahasa-ku-vscode-0.1.0.vsix`** dari [Halaman Rilis GitHub Bahasa Ku](https://github.com/zufarrizal/Bahasa-Ku/releases/tag/v0.2.0).
2. Buka Visual Studio Code.
3. Buka menu **Extensions** (`Ctrl+Shift+X`).
4. Klik tanda titik tiga (`...`) di pojok kanan atas bilah ekstensi.
5. Pilih **"Install from VSIX..."**.
6. Pilih berkas `bahasa-ku-vscode-0.1.0.vsix` yang telah Anda unduh.
7. Selesai! Buka berkas `.ku` apa pun untuk menikmati penyorotan sintaksis berwarna.

---

## 🛠️ Pengembangan & Pengemasan Ulang

Untuk mengemas ulang berkas `.vsix` dari kode sumber:
```bash
# Pastikan Node.js terpasang
npm install -g @vscode/vsce

# Masuk ke direktori ekstensi
cd ekstensi-vscode

# Kemas menjadi file .vsix
vsce package --no-dependencies --allow-missing-repository
```

---

## 📄 Lisensi

Ekstensi ini didistribusikan di bawah lisensi terbuka [MIT License](https://github.com/zufarrizal/Bahasa-Ku/blob/main/LICENSE).

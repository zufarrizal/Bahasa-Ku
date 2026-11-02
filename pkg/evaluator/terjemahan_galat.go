package evaluator

import (
	"strings"
)

// TerjemahkanGalat mengonversi pesan galat sistem tingkat rendah (SQLite, OS, IO, Network)
// menjadi kalimat Bahasa Indonesia yang ramah, jelas, dan edukatif sesuai standar Bahasa Ku.
func TerjemahkanGalat(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	msgLower := strings.ToLower(msg)

	// 1. Galat Basis Data (SQLite / Driver)
	if strings.Contains(msgLower, "unique constraint failed") {
		return "Data sudah terdaftar di basis data (duplikasi nilai unik)"
	}
	if strings.Contains(msgLower, "no such table") {
		return "Tabel basis data tidak ditemukan"
	}
	if strings.Contains(msgLower, "no such column") {
		return "Kolom basis data tidak ditemukan"
	}
	if strings.Contains(msgLower, "database is locked") {
		return "Basis data sedang digunakan atau dikunci oleh proses lain"
	}
	if strings.Contains(msgLower, "datatype mismatch") {
		return "Tipe data nilai tidak sesuai dengan skema kolom tabel"
	}
	if strings.Contains(msgLower, "not null constraint failed") {
		return "Kolom wajib diisi dan tidak boleh dibiarkan kosong"
	}
	if strings.Contains(msgLower, "foreign key constraint failed") {
		return "Relasi data rujukan tidak ditemukan"
	}
	if strings.Contains(msgLower, "disk i/o error") {
		return "Galat pembacaan atau penulisan berkas pada disk fisik"
	}

	// 2. Galat Jaringan & IPC (Network / Fetch / Socket)
	if strings.Contains(msgLower, "connection refused") || strings.Contains(msgLower, "connectex") {
		return "Koneksi ke port IPC internal ditolak atau belum siap"
	}
	if strings.Contains(msgLower, "failed to fetch") {
		return "Gagal terhubung ke layanan jembatan IPC aplikasi"
	}
	if strings.Contains(msgLower, "target closed") {
		return "Jendela aplikasi ditutup sebelum proses selesai"
	}

	// 3. Galat Berkas & Sistem Operasi
	if strings.Contains(msgLower, "the system cannot find the file specified") || strings.Contains(msgLower, "no such file or directory") {
		return "Berkas tidak ditemukan: periksa lokasi dan nama berkas"
	}
	if strings.Contains(msgLower, "access is denied") || strings.Contains(msgLower, "permission denied") {
		return "Akses ditolak: izin tidak mencukupi untuk membuka berkas atau port"
	}
	if strings.Contains(msgLower, "file already exists") {
		return "Berkas dengan nama tersebut sudah ada di direktori"
	}
	if strings.Contains(msgLower, "being used by another process") {
		return "Berkas sedang digunakan atau dikunci oleh aplikasi lain"
	}
	if strings.Contains(msgLower, "timeout") {
		return "Batas waktu eksekusi operasi telah habis (timeout)"
	}

	return msg
}

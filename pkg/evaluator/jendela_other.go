//go:build !windows

package evaluator

import (
	"fmt"
	"os/exec"
)

func tampilkanPesanNative(judul, isi, tipe string) {
	fmt.Printf("[%s][%s] %s\n", tipe, judul, isi)
}

func tampilkanKonfirmasiNative(judul, isi string) bool {
	fmt.Printf("[KONFIRMASI][%s] %s (y/n)\n", judul, isi)
	return true
}

func dialogPilihBerkasNative(judul, filter string) string {
	return ""
}

func dialogPilihFolderNative(judul string) string {
	return ""
}

func salinPapanKlipNative(teks string) bool {
	return false
}

func bacaPapanKlipNative() string {
	return ""
}

func bukaJendelaNative(judul, konten, targetUrl string, lebar, tinggi int, startIPC func() int, invokeIPC func(fungsi string, argumen []interface{}) (interface{}, error), onCleanup func()) {
	// Fallback untuk non-windows (Linux / macOS)
	// Buka URL via browser default atau server IPC
	url := targetUrl
	if url == "" {
		port := startIPC()
		url = fmt.Sprintf("http://127.0.0.1:%d", port)
	}
	_ = exec.Command("xdg-open", url).Start()
}

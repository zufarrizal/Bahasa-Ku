//go:build !windows

package evaluator

import "fmt"

func tampilkanPesanNative(judul, isi, tipe string) {
	fmt.Printf("[%s][%s] %s\n", tipe, judul, isi)
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

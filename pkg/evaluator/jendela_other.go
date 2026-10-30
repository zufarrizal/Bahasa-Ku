//go:build !windows

package evaluator

import "fmt"

func tampilkanPesanNative(judul, isi string) {
	fmt.Printf("[%s] %s\n", judul, isi)
}

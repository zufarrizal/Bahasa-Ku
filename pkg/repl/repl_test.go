package repl

import (
	"bytes"
	"strings"
	"testing"
)

func TestREPLMultiLine(t *testing.T) {
	input := `fungsi tambah(a, b) {
    kembalikan a + b
}
tambah(10, 20)
keluar
`
	in := strings.NewReader(input)
	out := &bytes.Buffer{}

	Start(in, out, "0.3.5-test")

	result := out.String()
	if !strings.Contains(result, "=> 30") {
		t.Fatalf("REPL multi-line gagal menghitung fungsi tambah. Output:\n%s", result)
	}
}

func TestREPLUserExample(t *testing.T) {
	input := `paket utama
pakai "cetak"

fungsi utama() {
    cetak.tulis("Halo Dunia dari Bahasa Ku!")
}
utama()
keluar
`
	in := strings.NewReader(input)
	out := &bytes.Buffer{}

	Start(in, out, "0.3.5-test")

	result := out.String()
	if strings.Contains(result, "Galat sintaks") {
		t.Fatalf("Ditemukan galat sintaks pada eksekusi REPL yang sah:\n%s", result)
	}
	if !strings.Contains(result, "Sampai jumpa di Bahasa Ku!") {
		t.Fatalf("REPL tidak keluar dengan benar:\n%s", result)
	}
}

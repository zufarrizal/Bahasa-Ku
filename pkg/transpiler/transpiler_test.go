package transpiler

import (
	"strings"
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestTranspilerNumericLiterals(t *testing.T) {
	input := `paket utama

var heksa = 0xFF
var biner = 0b1010
var juta = 1_000_000

fungsi utama() {
}
`
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	tr := New()
	goCode := tr.Transpile(prog)

	if !strings.Contains(goCode, "0xFF") {
		t.Errorf("transpiler gagal mempertahankan 0xFF: %s", goCode)
	}
	if !strings.Contains(goCode, "0b1010") {
		t.Errorf("transpiler gagal mempertahankan 0b1010: %s", goCode)
	}
	if !strings.Contains(goCode, "1_000_000") {
		t.Errorf("transpiler gagal mempertahankan 1_000_000: %s", goCode)
	}
	if !strings.Contains(goCode, "package main") {
		t.Errorf("transpiler gagal menerjemahkan paket utama -> package main: %s", goCode)
	}
}

func TestTranspilerInterpolatedString(t *testing.T) {
	input := `paket utama
fungsi utama() {
	nama := "Budi"
	sapa := "Halo ${nama}"
}
`
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	tr := New()
	goCode := tr.Transpile(prog)

	if !strings.Contains(goCode, "fmt.Sprint(nama)") {
		t.Errorf("transpiler gagal mengubah interpolasi string ke fmt.Sprint: %s", goCode)
	}
}

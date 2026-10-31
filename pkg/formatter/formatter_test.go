package formatter

import (
	"strings"
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestFormatterHexAndFloat(t *testing.T) {
	input := `paket utama

var angkaHeksa = 0xFF
var angkaBiner = 0b1010
var angkaJuta = 1_000_000
var floatPresisi = 3.14
var desimalKoma = 1_234.56
`
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	f := New()
	formatted := f.Format(prog)

	if !strings.Contains(formatted, "0xFF") {
		t.Errorf("formatter menghilangkan heksadesimal 0xFF: %s", formatted)
	}
	if !strings.Contains(formatted, "0b1010") {
		t.Errorf("formatter menghilangkan biner 0b1010: %s", formatted)
	}
	if !strings.Contains(formatted, "1_000_000") {
		t.Errorf("formatter menghilangkan underscore 1_000_000: %s", formatted)
	}
	if !strings.Contains(formatted, "3.14") {
		t.Errorf("formatter merusak float 3.14: %s", formatted)
	}
	if strings.Contains(formatted, "3.140000") {
		t.Errorf("formatter menambahkan nol berlebih 3.140000: %s", formatted)
	}
}

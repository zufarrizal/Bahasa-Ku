package evaluator

import (
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestInterpolatedStringLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input: `
			nama := "Budi"
			usia := 25
			teks := "Halo, nama saya ${nama} dan usia saya ${usia} tahun."
			teks
			`,
			expected: "Halo, nama saya Budi dan usia saya 25 tahun.",
		},
		{
			input: `
			a := 10
			b := 20
			hasil := "Hasil: ${a + b}"
			hasil
			`,
			expected: "Hasil: 30",
		},
		{
			input: `
			nama := "Dunia"
			teks := ` + "`Halo ${nama}!`" + `
			teks
			`,
			expected: "Halo Dunia!",
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		env := NewEnvironment()
		evaluated := Eval(program, env)
		if evaluated == nil {
			t.Fatalf("hasil evaluasi nil")
		}

		str, ok := evaluated.(*String)
		if !ok {
			t.Fatalf("objek bukan *String, didapat=%T (%+v)", evaluated, evaluated)
		}

		if str.Value != tt.expected {
			t.Errorf("nilai tidak sesuai. diharapkan=%q, didapat=%q", tt.expected, str.Value)
		}
	}
}

package evaluator

import (
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestOptionalChainingAndCoalesce(t *testing.T) {
	input := `
	pengguna := {
		"profil": {
			"nama": "Zufar"
		}
	}
	
	namaAda := pengguna?.profil?.nama ?? "Tamu"
	namaKosong := pengguna?.alamat?.kota ?? "Jakarta"
	
	var dataNihil = kosong
	kotaNihil := dataNihil?.detail?.kota ?? "Bandung"
	
	hasil := [namaAda, namaKosong, kotaNihil]
	hasil
	`

	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := InitGlobalEnvironment()
	evalRes := Eval(prog, env)
	if evalRes == nil {
		t.Fatalf("evaluator returned nil")
	}

	arr, ok := evalRes.(*Array)
	if !ok || len(arr.Elements) < 3 {
		t.Fatalf("hasil bukan Array valid: %+v", evalRes)
	}

	res1 := arr.Elements[0].(*String).Value
	res2 := arr.Elements[1].(*String).Value
	res3 := arr.Elements[2].(*String).Value

	if res1 != "Zufar" {
		t.Errorf("diharapkan Zufar, didapat %s", res1)
	}
	if res2 != "Jakarta" {
		t.Errorf("diharapkan Jakarta, didapat %s", res2)
	}
	if res3 != "Bandung" {
		t.Errorf("diharapkan Bandung, didapat %s", res3)
	}
}

package evaluator

import (
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestKriptoPasswordHashing(t *testing.T) {
	input := `
	sandiRahasia := "RahasiaSuperKu_2026!"
	hash := kripto.hashSandi(sandiRahasia)
	
	cocokBenar := kripto.cocokkanSandi("RahasiaSuperKu_2026!", hash)
	cocokSalah := kripto.cocokkanSandi("SandiSalah!", hash)
	
	hasil := [cocokBenar, cocokSalah]
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
	if !ok || len(arr.Elements) < 2 {
		t.Fatalf("hasil bukan Array valid: %+v", evalRes)
	}

	b1 := arr.Elements[0].(*Boolean).Value
	b2 := arr.Elements[1].(*Boolean).Value

	if !b1 {
		t.Errorf("sandi benar harus cocok (TRUE)")
	}
	if b2 {
		t.Errorf("sandi salah tidak boleh cocok (FALSE)")
	}
}

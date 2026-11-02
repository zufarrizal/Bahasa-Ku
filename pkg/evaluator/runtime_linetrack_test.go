package evaluator

import (
	"strings"
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestDivisionByZeroLineTracking(t *testing.T) {
	input := `fungsi bagiNol() {
    var x = 10 / 0
    kembalikan x
}
bagiNol()
`
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := InitGlobalEnvironment()
	res := Eval(prog, env)

	pObj, ok := res.(*PanicObject)
	if !ok {
		t.Fatalf("diharapkan *PanicObject, diperoleh %T (%v)", res, res)
	}

	inspect := pObj.Inspect()
	if !strings.Contains(inspect, "pembagian atau modulus dengan angka nol") {
		t.Fatalf("pesan panik tidak sesuai: %s", inspect)
	}
	if !strings.Contains(inspect, "baris 2:16") {
		t.Fatalf("nomor baris tidak akurat: %s", inspect)
	}
}

func TestArrayIndexOutOfRangeLineTracking(t *testing.T) {
	input := `fungsi aksesLuar() {
    var daftar = [1, 2, 3]
    var item = daftar[99]
    kembalikan item
}
aksesLuar()
`
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := InitGlobalEnvironment()
	res := Eval(prog, env)

	pObj, ok := res.(*PanicObject)
	if !ok {
		t.Fatalf("diharapkan *PanicObject, diperoleh %T (%v)", res, res)
	}

	inspect := pObj.Inspect()
	if !strings.Contains(inspect, "di luar batas") {
		t.Fatalf("pesan panik tidak sesuai: %s", inspect)
	}
	if !strings.Contains(inspect, "baris 3:22") {
		t.Fatalf("nomor baris pengindeksan tidak akurat: %s", inspect)
	}
}

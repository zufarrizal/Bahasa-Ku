package evaluator

import (
	"strings"
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestStackTraceOnPanic(t *testing.T) {
	input := `fungsi tingkatTiga() {
    panik("terjadi galat fatal di tingkat 3!")
}

fungsi tingkatDua() {
    tingkatTiga()
}

fungsi tingkatSatu() {
    tingkatDua()
}

tingkatSatu()
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

	trace := pObj.Inspect()
	if !strings.Contains(trace, "Jejak Pemanggilan (Stack Trace):") {
		t.Fatalf("output panik tidak mengandung Stack Trace:\n%s", trace)
	}
	if !strings.Contains(trace, "tingkatTiga()") || !strings.Contains(trace, "tingkatDua()") || !strings.Contains(trace, "tingkatSatu()") {
		t.Fatalf("Stack trace tidak melacak nama fungsi pemanggil secara akurat:\n%s", trace)
	}
}

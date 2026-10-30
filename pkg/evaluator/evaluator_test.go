package evaluator_test

import (
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func runKuCode(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	env := evaluator.InitGlobalEnvironment()
	return evaluator.Eval(prog, env)
}

func TestEvaluasiDasar(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{
			input: `
paket utama
fungsi utama() {
    var a = 10
    var b = 20
    kembalikan a + b
}
`,
			expected: 30,
		},
		{
			input: `
paket utama
fungsi faktorial(n) {
    jika n <= 1 {
        kembalikan 1
    }
    kembalikan n * faktorial(n - 1)
}
fungsi utama() {
    kembalikan faktorial(5)
}
`,
			expected: 120,
		},
		{
			input: `
paket utama
fungsi utama() {
    var arr = [10, 20]
    arr = tambah(arr, 30)
    kembalikan arr[0] + arr[2] + panjang(arr)
}
`,
			expected: 43,
		},
	}

	for _, tt := range tests {
		val := runKuCode(tt.input)
		intVal, ok := val.(*evaluator.Integer)
		if !ok {
			t.Fatalf("Nilai kembalian bukan Integer: %v", val)
		}
		if intVal.Value != tt.expected {
			t.Errorf("Diharapkan %d, tetapi didapat %d", tt.expected, intVal.Value)
		}
	}
}

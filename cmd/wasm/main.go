package main

import (
	"strings"
	"syscall/js"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func jalankanKodeKu(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return "Galat: Kode sumber kosong"
	}
	sumber := args[0].String()

	l := lexer.New(sumber)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return "Galat Sintaks:\n" + strings.Join(p.Errors(), "\n")
	}

	var outputLogs []string

	// Buat environment mandiri untuk browser
	env := evaluator.InitGlobalEnvironment()

	// Override modul cetak untuk menangkap output ke buffer browser
	cetakMap := &evaluator.Map{Pairs: make(map[string]evaluator.Object)}
	cetakMap.Pairs["tulis"] = &evaluator.Builtin{
		Fn: func(fnArgs ...evaluator.Object) evaluator.Object {
			parts := []string{}
			for _, a := range fnArgs {
				parts = append(parts, a.Inspect())
			}
			outputLogs = append(outputLogs, strings.Join(parts, " "))
			return evaluator.NIL
		},
	}
	cetakMap.Pairs["tulisBaris"] = cetakMap.Pairs["tulis"]
	env.Set("cetak", cetakMap)

	evaluator.Eval(prog, env)

	if len(outputLogs) == 0 {
		return "(Program berhasil dieksekusi tanpa output)"
	}
	return strings.Join(outputLogs, "\n")
}

func main() {
	c := make(chan struct{}, 0)
	js.Global().Set("jalankanEngineKuAsli", js.FuncOf(jalankanKodeKu))
	<-c
}

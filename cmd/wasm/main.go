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
	env := evaluator.NewEnvironment()

	// Modul cetak yang menangkap output ke buffer browser
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

	// Bawaan: panjang & tambah
	env.Set("panjang", &evaluator.Builtin{
		Fn: func(fnArgs ...evaluator.Object) evaluator.Object {
			if len(fnArgs) != 1 {
				return &evaluator.Integer{Value: 0}
			}
			switch arg := fnArgs[0].(type) {
			case *evaluator.Array:
				return &evaluator.Integer{Value: int64(len(arg.Elements))}
			case *evaluator.String:
				return &evaluator.Integer{Value: int64(len(arg.Value))}
			case *evaluator.Map:
				return &evaluator.Integer{Value: int64(len(arg.Pairs))}
			}
			return &evaluator.Integer{Value: 0}
		},
	})

	env.Set("tambah", &evaluator.Builtin{
		Fn: func(fnArgs ...evaluator.Object) evaluator.Object {
			if len(fnArgs) < 2 {
				return evaluator.NIL
			}
			if arr, ok := fnArgs[0].(*evaluator.Array); ok {
				newElems := append([]evaluator.Object{}, arr.Elements...)
				newElems = append(newElems, fnArgs[1:]...)
				return &evaluator.Array{Elements: newElems}
			}
			return evaluator.NIL
		},
	})

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

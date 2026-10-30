package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
	"github.com/zufarrizal/bahasa-ku/pkg/repl"
	"github.com/zufarrizal/bahasa-ku/pkg/transpiler"
)

const version = "0.2.0 (Go Native Edition)"

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		repl.Start(os.Stdin, os.Stdout, version)
		return
	}

	command := args[0]

	switch command {
	case "repl":
		repl.Start(os.Stdin, os.Stdout, version)
	case "versi", "-v", "--version":
		fmt.Println("Bahasa Ku versi", version)

	case "jalankan":
		if len(args) < 2 {
			fmt.Println("Galat: Harap tentukan nama berkas .ku yang ingin dijalankan.")
			os.Exit(1)
		}
		runFile(args[1])

	case "buat", "kompilasi":
		if len(args) < 2 {
			fmt.Println("Galat: Harap tentukan nama berkas .ku yang ingin ditranspilasi.")
			os.Exit(1)
		}
		target := ""
		if len(args) >= 3 {
			target = args[2]
		}
		transpileFile(args[1], target)

	default:
		fmt.Printf("Perintah '%s' tidak dikenal. Ketik 'ku' untuk bantuan.\n", command)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`Penggunaan CLI Bahasa Ku (Go Native):
  ku                       Membuka REPL (terminal interaktif)
  ku jalankan <berkas.ku>  Menjalankan berkas kode .ku
  ku buat <berkas.ku>      Mengonversi berkas .ku ke .go
  ku versi                 Menampilkan versi

Contoh:
  ku
  ku jalankan contoh/halo.ku
  ku buat contoh/tunda.ku hasil.go
`)
}

func runFile(filename string) {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Galat membaca berkas: %v\n", err)
		os.Exit(1)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Println("Galat Sintaks:")
		for _, e := range p.Errors() {
			fmt.Println(" -", e)
		}
		os.Exit(1)
	}

	env := evaluator.InitGlobalEnvironment()
	evaluator.Eval(prog, env)
}

func transpileFile(source, target string) {
	content, err := os.ReadFile(source)
	if err != nil {
		fmt.Printf("Galat membaca berkas: %v\n", err)
		os.Exit(1)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Println("Galat Sintaks:")
		for _, e := range p.Errors() {
			fmt.Println(" -", e)
		}
		os.Exit(1)
	}

	tr := transpiler.New()
	goCode := tr.Transpile(prog)

	if target == "" {
		ext := filepath.Ext(source)
		target = strings.TrimSuffix(source, ext) + ".go"
	}

	err = os.WriteFile(target, []byte(goCode), 0644)
	if err != nil {
		fmt.Printf("Galat menyimpan berkas .go: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Berhasil mentranspilasi '%s' -> '%s'! ✨\n", source, target)
}

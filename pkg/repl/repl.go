package repl

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

const PROMPT = "ku> "

func Start(in io.Reader, out io.Writer, version string) {
	scanner := bufio.NewScanner(in)
	env := evaluator.InitGlobalEnvironment()

	fmt.Fprintf(out, "Selamat datang di REPL Bahasa Ku %s!\n", version)
	fmt.Fprintf(out, "Ketik kode langsung di sini atau ketik 'keluar' / 'exit' untuk mengakhiri.\n\n")

	for {
		fmt.Fprintf(out, PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if line == "keluar" || line == "exit" || line == "quit" {
			fmt.Fprintf(out, "Sampai jumpa di Bahasa Ku! 👋\n")
			return
		}

		l := lexer.New(line)
		p := parser.New(l)
		prog := p.ParseProgram()

		if len(p.Errors()) > 0 {
			for _, err := range p.Errors() {
				fmt.Fprintf(out, "Galat sintaks: %s\n", err)
			}
			continue
		}

		evaluated := evaluator.Eval(prog, env)
		if evaluated != nil && evaluated != evaluator.NIL {
			fmt.Fprintf(out, "=> %s\n", evaluated.Inspect())
		}
	}
}

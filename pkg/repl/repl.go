package repl

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
	"github.com/zufarrizal/bahasa-ku/pkg/token"
)

const (
	PROMPT      = "ku> "
	PROMPT_MORE = "... "
)

// countBracketBalance menghitung selisih kurung buka dan tutup di luar string literal
func countBracketBalance(code string) int {
	l := lexer.New(code)
	balance := 0
	for {
		tok := l.NextToken()
		if tok.Type == token.EOF {
			break
		}
		switch tok.Type {
		case token.LBRACE, token.LPAREN, token.LBRACKET:
			balance++
		case token.RBRACE, token.RPAREN, token.RBRACKET:
			balance--
		}
	}
	return balance
}

func Start(in io.Reader, out io.Writer, version string) {
	scanner := bufio.NewScanner(in)
	env := evaluator.InitGlobalEnvironment()

	fmt.Fprintf(out, "Selamat datang di REPL Bahasa Ku %s!\n", version)
	fmt.Fprintf(out, "Ketik kode langsung di sini atau ketik 'keluar' / 'exit' untuk mengakhiri.\n\n")

	var multiLineBuffer []string
	bracketDepth := 0

	for {
		if bracketDepth > 0 {
			fmt.Fprintf(out, PROMPT_MORE)
		} else {
			fmt.Fprintf(out, PROMPT)
		}

		scanned := scanner.Scan()
		if !scanned {
			return
		}

		rawLine := scanner.Text()
		trimmedLine := strings.TrimSpace(rawLine)

		// Perintah keluar saat berada di prompt utama
		if bracketDepth == 0 && (trimmedLine == "keluar" || trimmedLine == "exit" || trimmedLine == "quit") {
			fmt.Fprintf(out, "Sampai jumpa di Bahasa Ku! 👋\n")
			return
		}

		// Jika baris kosong dan tidak sedang dalam blok multi-line, lewati
		if trimmedLine == "" && bracketDepth == 0 {
			continue
		}

		multiLineBuffer = append(multiLineBuffer, rawLine)
		combined := strings.Join(multiLineBuffer, "\n")
		bracketDepth = countBracketBalance(combined)

		// Jika masih ada kurung yang belum ditutup, teruskan membaca baris berikutnya
		if bracketDepth > 0 {
			continue
		}

		// Reset buffer untuk baris selanjutnya
		codeToEval := combined
		multiLineBuffer = nil
		bracketDepth = 0

		if strings.TrimSpace(codeToEval) == "" {
			continue
		}

		l := lexer.New(codeToEval)
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

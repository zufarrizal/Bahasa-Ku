package evaluator

import (
	"strings"
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestModulCLI(t *testing.T) {
	input := `
	cli.nama("alat-ku")
	cli.deskripsi("Alat CLI serbaguna Bahasa Ku")

	var hasilAksi = ""
	var portDidapat = ""

	cli.perintah("layani", "Menjalankan server", fungsi(cmd) {
		cmd.opsi("port", "p", "Nomor port server", "8080")
		cmd.aksi(fungsi(konteks) {
			hasilAksi = konteks["perintah"]
			portDidapat = konteks["opsi"]["port"]
		})
	})

	// Simulasi pemanggilan: alat-ku layani -p 9090
	cli.jalankan(["layani", "-p", "9090"])
	`
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	env := InitGlobalEnvironment()

	result := Eval(prog, env)
	if result != nil && result.Type() == PANIC_OBJ {
		t.Fatalf("evaluasi panik: %s", result.Inspect())
	}

	valAksi, _ := env.Get("hasilAksi")
	if str, ok := valAksi.(*String); !ok || str.Value != "layani" {
		t.Fatalf("hasilAksi salah, dapat: %v", valAksi)
	}

	valPort, _ := env.Get("portDidapat")
	if str, ok := valPort.(*String); !ok || str.Value != "9090" {
		t.Fatalf("portDidapat salah, dapat: %v", valPort)
	}

	// Test bantuan
	inputBantuan := `cli.bantuan()`
	lBantuan := lexer.New(inputBantuan)
	pBantuan := parser.New(lBantuan)
	progBantuan := pBantuan.ParseProgram()
	resBantuan := Eval(progBantuan, env)
	if str, ok := resBantuan.(*String); !ok || !strings.Contains(str.Value, "alat-ku") {
		t.Fatalf("teks bantuan salah: %v", resBantuan)
	}
}

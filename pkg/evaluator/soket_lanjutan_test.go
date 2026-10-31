package evaluator

import (
	"testing"
	"time"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestSoketLayaniDanBaris(t *testing.T) {
	input := `
	var pesanDiterimaServer = ""
	var server = soket.dengar("tcp", "127.0.0.1:19876")

	server.layani(fungsi(koneksi) {
		pesanDiterimaServer = koneksi.bacaBaris()
		koneksi.tulisBaris("BALASAN_DARI_SERVER")
		koneksi.tutup()
	})

	// Hubungkan client
	var client = soket.hubungkan("tcp", "127.0.0.1:19876")
	client.tulisBaris("HALO_SERVER")
	var balasan = client.bacaBaris()
	client.tutup()
	server.tutup()
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

	time.Sleep(100 * time.Millisecond)

	valServer, ok1 := env.Get("pesanDiterimaServer")
	if !ok1 || valServer.Inspect() != "HALO_SERVER" {
		t.Fatalf("pesanDiterimaServer salah, dapat: %v", valServer)
	}

	valBalasan, ok2 := env.Get("balasan")
	if !ok2 || valBalasan.Inspect() != "BALASAN_DARI_SERVER" {
		t.Fatalf("balasan client salah, dapat: %v", valBalasan)
	}
}

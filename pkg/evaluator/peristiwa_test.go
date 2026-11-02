package evaluator

import (
	"testing"
	"time"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestModulPeristiwa(t *testing.T) {
	input := `
	var dataDiterima = ""
	var hitungSekali = 0

	// 1. Langganan biasa (peristiwa.pada)
	var id1 = peristiwa.pada("pengguna.daftar", fungsi(data) {
		dataDiterima = data["nama"]
	})

	// 2. Langganan sekali pakai (peristiwa.sekali)
	peristiwa.sekali("sistem.mulai", fungsi(data) {
		hitungSekali = hitungSekali + 1
	})

	// Pancarkan pertama
	peristiwa.pancarkan("pengguna.daftar", {"nama": "Ahmad"})
	peristiwa.pancarkan("sistem.mulai", {})

	// Pancarkan kedua (sekali seharusnya tidak bertambah)
	peristiwa.pancarkan("sistem.mulai", {})

	// Hitung pendengar
	var pendengarDaftar = peristiwa.jumlahPendengar("pengguna.daftar")

	// Lepas langganan
	peristiwa.lepas("pengguna.daftar", id1)
	var pendengarSetelahLepas = peristiwa.jumlahPendengar("pengguna.daftar")
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

	valNama, _ := env.Get("dataDiterima")
	if s, ok := valNama.(*String); !ok || s.Value != "Ahmad" {
		t.Fatalf("dataDiterima salah, dapat: %v", valNama)
	}

	valSekali, _ := env.Get("hitungSekali")
	if i, ok := valSekali.(*Integer); !ok || i.Value != 1 {
		t.Fatalf("hitungSekali diharapkan 1, dapat: %v", valSekali)
	}

	valPendengar, _ := env.Get("pendengarDaftar")
	if i, ok := valPendengar.(*Integer); !ok || i.Value != 1 {
		t.Fatalf("pendengarDaftar diharapkan 1, dapat: %v", valPendengar)
	}

	valLepas, _ := env.Get("pendengarSetelahLepas")
	if i, ok := valLepas.(*Integer); !ok || i.Value != 0 {
		t.Fatalf("pendengarSetelahLepas diharapkan 0, dapat: %v", valLepas)
	}

	// 3. Uji pancarkanAsinkron
	inputAsync := `
	var asyncDiterima = salah
	peristiwa.pada("tugas.selesai", fungsi(data) {
		asyncDiterima = benar
	})
	peristiwa.pancarkanAsinkron("tugas.selesai", {})
	`
	lAsync := lexer.New(inputAsync)
	pAsync := parser.New(lAsync)
	Eval(pAsync.ParseProgram(), env)

	time.Sleep(50 * time.Millisecond)
	valAsync, _ := env.Get("asyncDiterima")
	if b, ok := valAsync.(*Boolean); !ok || !b.Value {
		t.Fatalf("asyncDiterima diharapkan benar")
	}
}

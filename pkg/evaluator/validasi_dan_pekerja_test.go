package evaluator

import (
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestValidasiPeriksa(t *testing.T) {
	input := `
	var dataValid = {
		"nama": "Zufar",
		"email": "zufar@example.com",
		"umur": "25"
	}

	var aturan = {
		"nama": "wajib|min:3",
		"email": "wajib|email",
		"umur": "wajib|numerik"
	}

	var hasilValid = validasi.periksa(dataValid, aturan)
	var apakahLolos = hasilValid["valid"]

	var dataTidakValid = {
		"nama": "Al",
		"email": "bukan-email",
		"umur": "abc"
	}

	var hasilGagal = validasi.periksa(dataTidakValid, aturan)
	var apakahGagal = hasilGagal["valid"]
	var galatNama = hasilGagal["galat"]["nama"]
	var galatEmail = hasilGagal["galat"]["email"]
	var galatUmur = hasilGagal["galat"]["umur"]
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

	valLolos, _ := env.Get("apakahLolos")
	if b, ok := valLolos.(*Boolean); !ok || !b.Value {
		t.Fatalf("diharapkan apakahLolos bernilai benar, dapat: %v", valLolos)
	}

	valGagal, _ := env.Get("apakahGagal")
	if b, ok := valGagal.(*Boolean); !ok || b.Value {
		t.Fatalf("diharapkan apakahGagal bernilai salah, dapat: %v", valGagal)
	}

	valGalatNama, _ := env.Get("galatNama")
	if s, ok := valGalatNama.(*String); !ok || s.Value == "" {
		t.Fatalf("galat nama tidak boleh kosong")
	}
}

func TestPekerjaKolam(t *testing.T) {
	input := `
	var totalDiproses = 0
	var kunci = sistem.kunci()

	var pool = pekerja.kolam(3, fungsi(tugas) {
		kunci.kunci()
		totalDiproses = totalDiproses + tugas
		kunci.buka()
	})

	pool.kirim(10)
	pool.kirim(20)
	pool.kirim(30)
	pool.tunggu()
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

	valTotal, _ := env.Get("totalDiproses")
	if i, ok := valTotal.(*Integer); !ok || i.Value != 60 {
		t.Fatalf("totalDiproses salah, diharapkan 60, dapat: %v", valTotal)
	}
}

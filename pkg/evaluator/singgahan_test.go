package evaluator

import (
	"testing"
	"time"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestModulSinggahan(t *testing.T) {
	input := `
	// 1. Taruh & Ambil dasar
	singgahan.taruh("pengguna:1", "Budi")
	var budi = singgahan.ambil("pengguna:1")
	var adaBudi = singgahan.ada("pengguna:1")

	// 2. Default value jika tidak ada
	var tamu = singgahan.ambil("pengguna:999", "Tamu Anonim")

	// 3. TTL kadaluwarsa
	singgahan.taruh("tokenTmp", "rahasia", "50ms")
	var tokenSebelum = singgahan.ambil("tokenTmp")

	// 4. Pola Ingat (Cache-aside memoize)
	var hitungHit = 0
	var ambilDataMahal = fungsi() {
		hitungHit = hitungHit + 1
		kembalikan "DATA_BERAT"
	}

	var d1 = singgahan.ingat("kunci_mahal", "200ms", ambilDataMahal)
	var d2 = singgahan.ingat("kunci_mahal", "200ms", ambilDataMahal)
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

	valBudi, _ := env.Get("budi")
	if s, ok := valBudi.(*String); !ok || s.Value != "Budi" {
		t.Fatalf("budi salah, dapat: %v", valBudi)
	}

	valAda, _ := env.Get("adaBudi")
	if b, ok := valAda.(*Boolean); !ok || !b.Value {
		t.Fatalf("adaBudi salah, dapat: %v", valAda)
	}

	valTamu, _ := env.Get("tamu")
	if s, ok := valTamu.(*String); !ok || s.Value != "Tamu Anonim" {
		t.Fatalf("tamu salah, dapat: %v", valTamu)
	}

	valToken, _ := env.Get("tokenSebelum")
	if s, ok := valToken.(*String); !ok || s.Value != "rahasia" {
		t.Fatalf("tokenSebelum salah: %v", valToken)
	}

	valD1, _ := env.Get("d1")
	valD2, _ := env.Get("d2")
	if valD1.Inspect() != "DATA_BERAT" || valD2.Inspect() != "DATA_BERAT" {
		t.Fatalf("hasil d1/d2 salah: %v / %v", valD1, valD2)
	}

	valHit, _ := env.Get("hitungHit")
	if i, ok := valHit.(*Integer); !ok || i.Value != 1 {
		t.Fatalf("hitungHit diharapkan 1 (hanya dieksekusi sekali karena di-cache), dapat: %v", valHit)
	}

	// Tunggu kadaluwarsa tokenTmp
	time.Sleep(70 * time.Millisecond)
	inputExpired := `singgahan.ambil("tokenTmp", "KOSONG")`
	lExp := lexer.New(inputExpired)
	pExp := parser.New(lExp)
	resExp := Eval(pExp.ParseProgram(), env)
	if resExp.Inspect() != "KOSONG" {
		t.Fatalf("tokenTmp seharusnya kadaluwarsa, dapat: %s", resExp.Inspect())
	}
}

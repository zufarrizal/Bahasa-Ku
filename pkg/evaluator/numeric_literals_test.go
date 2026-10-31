package evaluator

import (
	"testing"
)

func TestEnhancedNumericLiterals(t *testing.T) {
	env := InitGlobalEnvironment()
	src := `
		var juta = 1_000_000
		var heksa = 0xFF
		var biner = 0b1010
		var oktal = 0o77
		var desimalKoma = 1_234.56_78
		var hitung = 0xFF + 0b1
	`
	res := evalTestProgram(src, env)
	if p, ok := res.(*PanicObject); ok {
		t.Fatalf("Panik saat evaluasi angka lanjutan: %s", p.Inspect())
	}

	jutaVal, ok := env.Get("juta")
	if !ok || jutaVal.Inspect() != "1000000" {
		t.Errorf("juta salah: %v (ekspektasi 1000000)", jutaVal)
	}

	heksaVal, ok := env.Get("heksa")
	if !ok || heksaVal.Inspect() != "255" {
		t.Errorf("heksa salah: %v (ekspektasi 255)", heksaVal)
	}

	binerVal, ok := env.Get("biner")
	if !ok || binerVal.Inspect() != "10" {
		t.Errorf("biner salah: %v (ekspektasi 10)", binerVal)
	}

	oktalVal, ok := env.Get("oktal")
	if !ok || oktalVal.Inspect() != "63" {
		t.Errorf("oktal salah: %v (ekspektasi 63)", oktalVal)
	}

	hitungVal, ok := env.Get("hitung")
	if !ok || hitungVal.Inspect() != "256" {
		t.Errorf("hitung salah: %v (ekspektasi 256)", hitungVal)
	}

	desVal, ok := env.Get("desimalKoma")
	if !ok {
		t.Errorf("desimalKoma tidak ditemukan")
	} else if f, isFloat := desVal.(*Float); !isFloat || f.Value < 1234.5 || f.Value > 1234.6 {
		t.Errorf("desimalKoma salah: %v", desVal)
	}
}

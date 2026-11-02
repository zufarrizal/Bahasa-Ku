package evaluator_test

import (
	"os"
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
)

func TestChannelCloseAndDrainLoop(t *testing.T) {
	code := `
paket utama
fungsi utama() {
    var ch = buat("kanal", 3)
    
    ch <- 10
    ch <- 20
    ch <- 30
    tutup(ch)

    var total = 0
    untuk v dalam ch {
        total = total + v
    }

    // Uji channel receive koma ok (v, ok := <-ch)
    var nilai, masihBuka = <-ch
    jika masihBuka {
        kembalikan -1
    }

    kembalikan total
}
`
	res := runKuCode(code)
	intVal, ok := res.(*evaluator.Integer)
	if !ok || intVal.Value != 60 {
		t.Fatalf("Diharapkan total 60 dari drain channel loop, didapat: %v", res)
	}
}

func TestExportVisibilityRule(t *testing.T) {
	// Buat modul sub-file sementara
	subModuleContent := `
paket pembantu

var Rahasia = "Terekspor Ke Publik"
var rahasiaLokal = "Privat Internal"

fungsi HitungBonus(gaji) {
    kembalikan gaji * 2
}

fungsi rumusInternal(x) {
    kembalikan x + 1
}
`
	tempFile := "modul_uji_visibilitas.ku"
	err := os.WriteFile(tempFile, []byte(subModuleContent), 0644)
	if err != nil {
		t.Fatalf("Gagal membuat file sementara: %v", err)
	}
	defer os.Remove(tempFile)

	callerCode := `
paket utama

pakai "./modul_uji_visibilitas.ku"

fungsi utama() {
    // 1. Simbol huruf kapital harus dapat diakses
    jika Rahasia != "Terekspor Ke Publik" {
        kembalikan 1
    }
    jika HitungBonus(500) != 1000 {
        kembalikan 2
    }

    // 2. Simbol huruf kecil tidak boleh bocor/terekspor
    jika rahasiaLokal != kosong {
        kembalikan 3
    }
    jika rumusInternal != kosong {
        kembalikan 4
    }

    kembalikan 999
}
`
	res := runKuCode(callerCode)
	intVal, ok := res.(*evaluator.Integer)
	if !ok || intVal.Value != 999 {
		t.Fatalf("Diharapkan 999 (visibilitas ekspor kapital berhasil), didapat: %v", res)
	}
}

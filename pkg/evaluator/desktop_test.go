package evaluator_test

import (
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
)

func TestDesktopDanJendelaMap(t *testing.T) {
	code := `
paket utama
fungsi utama() {
    var apakahAdaPesan = desktop["pesan"] != kosong
    var apakahAdaPilihBerkas = desktop["pilihBerkas"] != kosong
    var apakahAdaPilihFolder = desktop["pilihFolder"] != kosong
    var apakahAdaBuka = desktop["buka"] != kosong
    var apakahSamaDenganJendela = desktop == jendela

    jika !apakahAdaPesan || !apakahAdaPilihBerkas || !apakahAdaPilihFolder || !apakahAdaBuka || !apakahSamaDenganJendela {
        kembalikan 1
    }
    kembalikan 100
}
`
	val := runKuCode(code)
	intVal, ok := val.(*evaluator.Integer)
	if !ok {
		t.Fatalf("Nilai bukan Integer: %v", val)
	}
	if intVal.Value != 100 {
		t.Errorf("Diharapkan 100, didapat %d", intVal.Value)
	}
}

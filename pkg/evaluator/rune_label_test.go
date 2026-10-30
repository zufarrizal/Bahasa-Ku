package evaluator_test

import (
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
)

func TestRuneDanLabelledLoop(t *testing.T) {
	code := `
paket utama

fungsi utama() {
    // 1. Pengindeksan byte string dan fungsi bita.karakter (rune)
    var s = "Halo"
    var b0 = s[0] // byte mentah 'H' = 72
    jika b0 != 72 {
        kembalikan 1
    }

    var unicodeStr = "Go世界"
    var jk = bita.jumlahKarakter(unicodeStr) // 4 karakter asli
    jika jk != 4 {
        kembalikan 2
    }

    var runes = bita.karakter(unicodeStr) // [71, 111, 19990, 30028]
    jika panjang(runes) != 4 {
        kembalikan 3
    }
    jika runes[0] != 71 || runes[1] != 111 {
        kembalikan 4
    }

    var balikKeStr = bita.dariKarakter(runes)
    jika balikKeStr != unicodeStr {
        kembalikan 5
    }

    // 2. Iterasi untuk ... dalam string menghasilkan (byteOffset, runeInt)
    var totalOffset = 0
    var jumlahLoop = 0
    var teks = "Ku"
    untuk idx, r dalam teks {
        totalOffset = totalOffset + idx
        jumlahLoop = jumlahLoop + 1
    }
    jika jumlahLoop != 2 || totalOffset != 1 {
        kembalikan 6
    }

    // 3. Labelled Loop dengan 'henti Label' (Break outer loop)
    var hitungLuar = 0
    var hitungDalam = 0
Luar:
    untuk var i = 0; i < 5; i = i + 1 {
        hitungLuar = hitungLuar + 1
        untuk var j = 0; j < 5; j = j + 1 {
            hitungDalam = hitungDalam + 1
            jika i == 2 && j == 1 {
                henti Luar
            }
        }
    }

    // Loop terluar harus berhenti saat i = 2, j = 1
    // i=0 -> j: 0..4 (5 kali)
    // i=1 -> j: 0..4 (5 kali)
    // i=2 -> j: 0, 1 (2 kali) => total hitungDalam = 12, hitungLuar = 3
    jika hitungLuar != 3 || hitungDalam != 12 {
        kembalikan 7
    }

    kembalikan 999
}
`
	val := runKuCode(code)
	intVal, ok := val.(*evaluator.Integer)
	if !ok {
		t.Fatalf("Nilai bukan Integer: %v", val)
	}
	if intVal.Value != 999 {
		t.Errorf("Diharapkan 999, didapat %d", intVal.Value)
	}
}

package evaluator_test

import (
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
)

func TestBuiltinsGoLanjutan(t *testing.T) {
	code := `
paket utama

fungsi gantiNilai(ptr, nilaiBaru) {
    alamat.ubah(ptr, nilaiBaru)
}

fungsi utama() {
    // 1. Uji buat() (make) untuk daftar
    var s = buat("daftar", 3, 5)
    jika panjang(s) != 3 || kapasitas(s) != 5 {
        kembalikan 1
    }

    // 2. Uji salin() (copy)
    var sumber = [10, 20, 30]
    var tujuan = [0, 0, 0]
    var n = salin(tujuan, sumber)
    jika n != 3 || tujuan[0] != 10 || tujuan[2] != 30 {
        kembalikan 2
    }

    // 3. Uji buat() untuk peta
    var p = buat("peta", 10)
    p["kunci"] = "nilai"
    jika p["kunci"] != "nilai" {
        kembalikan 3
    }

    // 4. Uji alamat & pointer
    var angkaAsli = 42
    var ptr = alamat.pointer(angkaAsli)
    jika alamat.baca(ptr) != 42 {
        kembalikan 4
    }
    gantiNilai(ptr, 999)
    jika alamat.baca(ptr) != 999 {
        kembalikan 5
    }

    // 5. Uji teruskan (fallthrough) pada pilih (switch)
    var langkah = 0
    var mode = "langkah1"
    pilih mode {
    kasus "langkah1":
        langkah = 1
        teruskan
    kasus "langkah2":
        langkah = langkah + 10
    kasus "langkah3":
        langkah = 999
    }
    // Jika fallthrough bekerja: 1 + 10 = 11 (tidak mengeksekusi langkah3)
    jika langkah != 11 {
        kembalikan 6
    }

    kembalikan 777
}
`
	val := runKuCode(code)
	intVal, ok := val.(*evaluator.Integer)
	if !ok {
		t.Fatalf("Nilai bukan Integer: %v", val)
	}
	if intVal.Value != 777 {
		t.Errorf("Diharapkan 777, didapat %d", intVal.Value)
	}
}

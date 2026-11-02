package evaluator_test

import (
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
)

func TestIdiomGoLanjutan(t *testing.T) {
	code := `
paket utama

fungsi ambilNilaiDanStatus() {
    kembalikan 100, benar
}

fungsi utama() {
    // 1. Uji statement inisialisasi pada percabangan jika (If with short statement)
    var lolos = salah
    jika var nilai, status = ambilNilaiDanStatus(); status == benar && nilai == 100 {
        lolos = benar
    }
    jika lolos != benar {
        kembalikan 1
    }

    // 2. Uji multi-value pada kasus pilih (Multi-case switch)
    var kategori = ""
    var ekstensi = ".png"
    pilih ekstensi {
    kasus ".jpg", ".jpeg", ".png", ".webp":
        kategori = "gambar"
    kasus ".mp4", ".mkv":
        kategori = "video"
    bawaan:
        kategori = "lainnya"
    }
    jika kategori != "gambar" {
        kembalikan 2
    }

    var katVideo = ""
    pilih ".mkv" {
    kasus ".jpg", ".png":
        katVideo = "gambar"
    kasus ".mp4", ".mkv", ".avi":
        katVideo = "video"
    }
    jika katVideo != "video" {
        kembalikan 3
    }

    // 3. Uji fungsi anonim self-invoking langsung (IIFE)
    var hasilIIFE = fungsi(x, y) {
        kembalikan x * y
    }(6, 7)

    jika hasilIIFE != 42 {
        kembalikan 4
    }

    kembalikan 888
}
`
	val := runKuCode(code)
	intVal, ok := val.(*evaluator.Integer)
	if !ok {
		t.Fatalf("Nilai bukan Integer: %v", val)
	}
	if intVal.Value != 888 {
		t.Errorf("Diharapkan 888, didapat %d", intVal.Value)
	}
}

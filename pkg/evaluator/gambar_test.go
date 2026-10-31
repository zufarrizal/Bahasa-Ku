package evaluator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestModulGambar(t *testing.T) {
	tempDir := t.TempDir()
	outPng := filepath.ToSlash(filepath.Join(tempDir, "test.png"))
	outJpg := filepath.ToSlash(filepath.Join(tempDir, "test.jpg"))

	input := `
	var img = gambar.buat(100, 50, "#FF0000")
	var l = img.lebar
	var tg = img.tinggi

	// Simpan PNG
	var simpanPngSukses = img.simpan("` + outPng + `")

	// Info gambar
	var info = gambar.info("` + outPng + `")
	var infoFormat = info.format
	var infoLebar = info.lebar
	var infoTinggi = info.tinggi

	// Potong & Ubah Ukuran
	var imgKecil = img.ubahUkuran(50, 25)
	var kecilL = imgKecil.lebar
	var kecilTg = imgKecil.tinggi

	// Simpan JPEG
	var simpanJpgSukses = imgKecil.simpan("` + outJpg + `", 90)

	// Buka kembali
	var imgBuka = gambar.buka("` + outPng + `")
	var bukaL = imgBuka.lebar
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

	// Verifikasi variabel
	assertInt := func(varName string, expected int64) {
		val, ok := env.Get(varName)
		if !ok {
			t.Fatalf("variabel %s tidak ditemukan", varName)
		}
		num, ok := val.(*Integer)
		if !ok || num.Value != expected {
			t.Fatalf("variabel %s = %v, diharapkan %d", varName, val, expected)
		}
	}

	assertBool := func(varName string, expected bool) {
		val, ok := env.Get(varName)
		if !ok {
			t.Fatalf("variabel %s tidak ditemukan", varName)
		}
		b, ok := val.(*Boolean)
		if !ok || b.Value != expected {
			t.Fatalf("variabel %s = %v, diharapkan %v", varName, val, expected)
		}
	}

	assertInt("l", 100)
	assertInt("tg", 50)
	assertBool("simpanPngSukses", true)
	assertInt("infoLebar", 100)
	assertInt("infoTinggi", 50)
	assertInt("kecilL", 50)
	assertInt("kecilTg", 25)
	assertBool("simpanJpgSukses", true)
	assertInt("bukaL", 100)

	if _, err := os.Stat(outPng); err != nil {
		t.Fatalf("file png tidak terbuat di disk")
	}
	if _, err := os.Stat(outJpg); err != nil {
		t.Fatalf("file jpg tidak terbuat di disk")
	}
}

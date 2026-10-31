package evaluator

import (
	"path/filepath"
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestBasisdataSkemaDeklaratif(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.ToSlash(filepath.Join(tempDir, "test_skema.db"))

	input := `
	var db = basisdata.bukaBerkas("` + dbPath + `")

	// 1. Deklarasi skema tabel otomatis
	db.skema("barang", {
		"id": "kunci_utama",
		"nama": "teks",
		"harga": "angka",
		"stok": "angka",
		"tersedia": "saklar"
	})

	// 2. Masukkan data via QueryBuilder
	db.tabel("barang").masukkan({
		"nama": "Buku Catatan",
		"harga": 15000,
		"stok": 50,
		"tersedia": benar
	})

	// 3. Ambil data
	var hasil = db.tabel("barang").dimana("nama", "Buku Catatan").ambilSatu()
	var namaBarang = hasil["nama"]
	var hargaBarang = hasil["harga"]
	db.tutup()
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

	valNama, _ := env.Get("namaBarang")
	if valNama.Inspect() != "Buku Catatan" {
		t.Fatalf("namaBarang salah, dapat: %v", valNama)
	}

	valHarga, _ := env.Get("hargaBarang")
	if valHarga.Inspect() != "15000" {
		t.Fatalf("hargaBarang salah, dapat: %v", valHarga)
	}
}

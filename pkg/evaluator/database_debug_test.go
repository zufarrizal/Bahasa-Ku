package evaluator

import (
	"path/filepath"
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestDatabaseDebugSQL(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.ToSlash(filepath.Join(tempDir, "test_debug.db"))

	input := `
	var db = basisdata.bukaBerkas("` + dbPath + `")
	db.debug(benar)
	db.eksekusi("CREATE TABLE pengguna (id INTEGER PRIMARY KEY, nama TEXT);")
	db.eksekusi("INSERT INTO pengguna (nama) VALUES (?);", "Ahmad")
	var hasil = db.kueri("SELECT * FROM pengguna WHERE nama = ?;", "Ahmad")
	var nama = hasil[0]["nama"]
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

	valNama, _ := env.Get("nama")
	if valNama.Inspect() != "Ahmad" {
		t.Fatalf("nama salah, dapat: %v", valNama)
	}
}

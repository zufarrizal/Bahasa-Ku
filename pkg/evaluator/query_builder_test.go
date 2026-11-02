package evaluator

import (
	"os"
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestQueryBuilder(t *testing.T) {
	dbFile := "test_qb.db"
	_ = os.Remove(dbFile)
	defer os.Remove(dbFile)

	input := `
	db := basisdata.bukaBerkas("test_qb.db")
	db.eksekusi("CREATE TABLE pengguna (id INTEGER PRIMARY KEY, nama TEXT, peran TEXT)")
	
	// Masukkan data
	db.tabel("pengguna").masukkan({"nama": "Ahmad", "peran": "admin"})
	db.tabel("pengguna").masukkan({"nama": "Budi", "peran": "staf"})
	
	// Ambil data satu
	admin := db.tabel("pengguna").dimana("peran", "admin").ambilSatu()
	
	// Perbarui
	db.tabel("pengguna").dimana("nama", "Budi").perbarui({"peran": "manajer"})
	budiBaru := db.tabel("pengguna").dimana("nama", "Budi").ambilSatu()
	
	// Ambil semua
	semua := db.tabel("pengguna").ambilSemua()
	
	hasil := [admin["nama"], budiBaru["peran"]]
	hasil
	`

	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := InitGlobalEnvironment()
	evalRes := Eval(prog, env)
	if evalRes == nil {
		t.Fatalf("evaluator returned nil")
	}

	arr, ok := evalRes.(*Array)
	if !ok || len(arr.Elements) < 2 {
		t.Fatalf("hasil bukan Array valid: %+v", evalRes)
	}

	t.Logf("Elements: %+v, %+v", arr.Elements[0], arr.Elements[1])

	adminNama := arr.Elements[0].(*String).Value
	budiPeran := arr.Elements[1].(*String).Value

	if adminNama != "Ahmad" {
		t.Errorf("diharapkan Ahmad, didapat %s", adminNama)
	}
	if budiPeran != "manajer" {
		t.Errorf("diharapkan manajer, didapat %s", budiPeran)
	}
}

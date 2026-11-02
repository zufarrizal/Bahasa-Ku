package evaluator_test

import (
	"os"
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
)

func TestSqliteDanClientJson(t *testing.T) {
	dbFile := "test_sqlite.db"
	_ = os.Remove(dbFile)
	defer os.Remove(dbFile)

	code := `
paket utama
fungsi utama() {
    // 1. Uji SQLite lokal
    var db = basisdata.bukaBerkas("test_sqlite.db")
    jika db["terhubung"] != benar {
        kembalikan 1
    }

    db.eksekusi("CREATE TABLE produk (id INTEGER PRIMARY KEY, nama TEXT, harga INTEGER);")
    db.eksekusi("INSERT INTO produk (id, nama, harga) VALUES (1, 'Kopi Ku', 15000);")
    db.eksekusi("INSERT INTO produk (id, nama, harga) VALUES (2, 'Teh Ku', 8000);")

    var baris = db.kueri("SELECT id, nama, harga FROM produk ORDER BY id ASC")
    jika panjang(baris) != 2 {
        kembalikan 2
    }
    jika baris[0]["nama"] != "Kopi Ku" || baris[1]["harga"] != "8000" {
        kembalikan 3
    }
    db.tutup()

    // 2. Uji ketersediaan metode desktop.salinTeks & desktop.bacaKlip
    jika desktop["salinTeks"] == kosong || desktop["bacaKlip"] == kosong {
        kembalikan 4
    }

    // 3. Uji ketersediaan metode jaringan.ambilJSON & jaringan.kirimJSON
    jika jaringan["ambilJSON"] == kosong || jaringan["kirimJSON"] == kosong {
        kembalikan 5
    }

    kembalikan 600
}
`
	val := runKuCode(code)
	intVal, ok := val.(*evaluator.Integer)
	if !ok {
		t.Fatalf("Nilai bukan Integer: %v", val)
	}
	if intVal.Value != 600 {
		t.Errorf("Diharapkan 600, didapat %d", intVal.Value)
	}
}

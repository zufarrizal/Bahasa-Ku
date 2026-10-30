package evaluator_test

import (
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestEmbeddingTypeSwitchNamedReturn(t *testing.T) {
	code := `
tipe Orang struktur {
    nama
    umur
}

fungsi (o Orang) sapa() {
    kembalikan "Halo, saya " + o.nama
}

tipe Karyawan struktur {
    Orang
    jabatan
}

fungsi periksaTipe(x) {
    pilih v := x.(tipe) {
    kasus teks:
        kembalikan "String: " + v
    kasus angka:
        kembalikan v * 2
    kasus Orang:
        kembalikan "Orang bernama " + v.nama
    kasus Karyawan:
        kembalikan "Karyawan jabatan " + v.jabatan
    bawaan:
        kembalikan "Lainnya"
    }
}

fungsi hitungKalkulasi(x, y) (hasil, info) {
    hasil = x * y
    info = "sukses"
    kembalikan
}

var xTeks = "contoh"

fungsi utama() {
    var k = Karyawan({
        "Orang": Orang({"nama": "Budi", "umur": 30}),
        "jabatan": "Insinyur"
    })
    
    // Test Struct Embedding & Method Promotion
    var pesanSapa = k.sapa()
    var namaK = k.nama
    
    // Test Type Assertion
    var s = xTeks.(teks)
    
    // Test Type Switch
    var hasilTipe1 = periksaTipe("halo")
    var hasilTipe2 = periksaTipe(10)
    var hasilTipe3 = periksaTipe(k)
    
    // Test Named Return Values & Naked Return
    var h, msg = hitungKalkulasi(6, 7)

    kembalikan [pesanSapa, namaK, s, hasilTipe1, hasilTipe2, hasilTipe3, h, msg]
}

utama()
`
	l := lexer.New(code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	env := evaluator.InitGlobalEnvironment()
	val := evaluator.Eval(prog, env)
	if panicObj, ok := val.(*evaluator.PanicObject); ok {
		t.Fatalf("Evaluator panic: %s", panicObj.Inspect())
	}

	arr, ok := val.(*evaluator.Array)
	if !ok {
		t.Fatalf("Expected Array result, got %T: %v", val, val)
	}

	if len(arr.Elements) != 8 {
		t.Fatalf("Expected 8 elements, got %d", len(arr.Elements))
	}

	// 1. pesanSapa == "Halo, saya Budi"
	if arr.Elements[0].Inspect() != "Halo, saya Budi" {
		t.Errorf("Element 0 mismatch: %s", arr.Elements[0].Inspect())
	}
	// 2. namaK == "Budi"
	if arr.Elements[1].Inspect() != "Budi" {
		t.Errorf("Element 1 mismatch: %s", arr.Elements[1].Inspect())
	}
	// 3. s == "contoh"
	if arr.Elements[2].Inspect() != "contoh" {
		t.Errorf("Element 2 mismatch: %s", arr.Elements[2].Inspect())
	}
	// 4. hasilTipe1 == "String: halo"
	if arr.Elements[3].Inspect() != "String: halo" {
		t.Errorf("Element 3 mismatch: %s", arr.Elements[3].Inspect())
	}
	// 5. hasilTipe2 == 20
	if arr.Elements[4].Inspect() != "20" {
		t.Errorf("Element 4 mismatch: %s", arr.Elements[4].Inspect())
	}
	// 6. hasilTipe3 == "Karyawan jabatan Insinyur"
	if arr.Elements[5].Inspect() != "Karyawan jabatan Insinyur" {
		t.Errorf("Element 5 mismatch: %s", arr.Elements[5].Inspect())
	}
	// 7. h == 42
	if arr.Elements[6].Inspect() != "42" {
		t.Errorf("Element 6 mismatch: %s", arr.Elements[6].Inspect())
	}
	// 8. msg == "sukses"
	if arr.Elements[7].Inspect() != "sukses" {
		t.Errorf("Element 7 mismatch: %s", arr.Elements[7].Inspect())
	}
}

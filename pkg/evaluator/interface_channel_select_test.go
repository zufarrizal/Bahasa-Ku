package evaluator_test

import (
	"testing"
	"time"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestChannelOperatorAndNativeSelect(t *testing.T) {
	code := `
paket utama

fungsi utama() {
    var ch = kanal.buat(2)

    // 1. Operator kirim (ch <- x)
    ch <- "pesan-1"
    ch <- "pesan-2"

    // 2. Operator terima (<-ch)
    var m1 = <-ch
    var m2 = <-ch

    // 3. Native channel select non-blocking (dengan bawaan)
    var statusSelect = ""
    var chEmpty = kanal.buat(1)

    pilih {
    kasus data := <-chEmpty:
        statusSelect = "menerima: " + data
    bawaan:
        statusSelect = "kanal-kosong-nonblocking"
    }

    // 4. Native channel select dengan data siap
    var chReady = kanal.buat(1)
    chReady <- "data-siap"

    var hasilReady = ""
    pilih {
    kasus pesan := <-chReady:
        hasilReady = pesan
    bawaan:
        hasilReady = "gagal"
    }

    kembalikan [m1, m2, statusSelect, hasilReady]
}
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
		t.Fatalf("Panic: %s", panicObj.Inspect())
	}

	arr, ok := val.(*evaluator.Array)
	if !ok {
		t.Fatalf("Expected array, got %T: %v", val, val)
	}

	if len(arr.Elements) != 4 {
		t.Fatalf("Expected 4 elements, got %d", len(arr.Elements))
	}

	if arr.Elements[0].Inspect() != "pesan-1" {
		t.Errorf("m1 mismatch: %s", arr.Elements[0].Inspect())
	}
	if arr.Elements[1].Inspect() != "pesan-2" {
		t.Errorf("m2 mismatch: %s", arr.Elements[1].Inspect())
	}
	if arr.Elements[2].Inspect() != "kanal-kosong-nonblocking" {
		t.Errorf("statusSelect mismatch: %s", arr.Elements[2].Inspect())
	}
	if arr.Elements[3].Inspect() != "data-siap" {
		t.Errorf("hasilReady mismatch: %s", arr.Elements[3].Inspect())
	}
}

func TestInterfaceDefinitionAndAssertion(t *testing.T) {
	code := `
paket utama

// Definisi antarmuka
tipe Pembicara antarmuka {
    Suara()
}

tipe Kucing struktur {
    nama
}

fungsi (k Kucing) Suara() {
    kembalikan "Meong dari " + k.nama
}

tipe Meja struktur {
    kaki
}

fungsi periksaBicara(x) {
    pilih v := x.(tipe) {
    kasus Pembicara:
        kembalikan v.Suara()
    bawaan:
        kembalikan "Bukan pembicara"
    }
}

fungsi utama() {
    var k = Kucing({"nama": "Kitty"})
    var m = Meja({"kaki": 4})

    // Type Assertion antarmuka
    var asersiKucing = k.(Pembicara)
    var asersiMeja = m.(Pembicara) // harus kosong / nil

    // Type Switch antarmuka
    var switchKucing = periksaBicara(k)
    var switchMeja = periksaBicara(m)

    kembalikan [asersiKucing.Suara(), asersiMeja, switchKucing, switchMeja]
}
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
		t.Fatalf("Panic: %s", panicObj.Inspect())
	}

	arr, ok := val.(*evaluator.Array)
	if !ok {
		t.Fatalf("Expected array, got %T: %v", val, val)
	}

	if len(arr.Elements) != 4 {
		t.Fatalf("Expected 4 elements, got %d", len(arr.Elements))
	}

	if arr.Elements[0].Inspect() != "Meong dari Kitty" {
		t.Errorf("asersiKucing mismatch: %s", arr.Elements[0].Inspect())
	}
	if arr.Elements[1] != evaluator.NIL {
		t.Errorf("asersiMeja must be nil, got: %s", arr.Elements[1].Inspect())
	}
	if arr.Elements[2].Inspect() != "Meong dari Kitty" {
		t.Errorf("switchKucing mismatch: %s", arr.Elements[2].Inspect())
	}
	if arr.Elements[3].Inspect() != "Bukan pembicara" {
		t.Errorf("switchMeja mismatch: %s", arr.Elements[3].Inspect())
	}
	_ = time.Second
}

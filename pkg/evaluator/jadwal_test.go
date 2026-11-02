package evaluator

import (
	"testing"
	"time"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestJadwalSetelah(t *testing.T) {
	input := `
	var dipanggil = salah
	jadwal.setelah("50ms", fungsi() {
		dipanggil = benar
	})
	`
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	env := InitGlobalEnvironment()

	Eval(prog, env)

	time.Sleep(100 * time.Millisecond)

	val, ok := env.Get("dipanggil")
	if !ok {
		t.Fatalf("variabel dipanggil tidak ditemukan")
	}
	b, ok := val.(*Boolean)
	if !ok || !b.Value {
		t.Fatalf("diharapkan dipanggil bernilai benar, dapat: %v", val)
	}
}

func TestJadwalSetelahBatal(t *testing.T) {
	input := `
	var dipanggil = salah
	var t = jadwal.setelah("100ms", fungsi() {
		dipanggil = benar
	})
	t.batal()
	`
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	env := InitGlobalEnvironment()

	Eval(prog, env)

	time.Sleep(150 * time.Millisecond)

	val, ok := env.Get("dipanggil")
	if !ok {
		t.Fatalf("variabel dipanggil tidak ditemukan")
	}
	b, ok := val.(*Boolean)
	if !ok || b.Value {
		t.Fatalf("diharapkan dipanggil tetap bernilai salah karena dibatalkan, dapat: %v", val)
	}
}

func TestJadwalSetiapHenti(t *testing.T) {
	input := `
	var hitung = 0
	var t = jadwal.setiap("30ms", fungsi() {
		hitung = hitung + 1
	})
	`
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	env := InitGlobalEnvironment()

	Eval(prog, env)

	time.Sleep(100 * time.Millisecond)

	// Hentikan jadwal
	tVal, _ := env.Get("t")
	tMap := tVal.(*Map)
	stopFn := tMap.Pairs["henti"].(*Builtin)
	stopFn.Fn()

	val1, _ := env.Get("hitung")
	count1 := val1.(*Integer).Value
	if count1 == 0 {
		t.Fatalf("diharapkan hitung > 0, dapat: %d", count1)
	}

	time.Sleep(60 * time.Millisecond)
	val2, _ := env.Get("hitung")
	count2 := val2.(*Integer).Value

	if count2 != count1 {
		t.Fatalf("diharapkan hitung tidak bertambah setelah dihentikan, val1=%d, val2=%d", count1, count2)
	}
}

func TestJadwalParseDurasi(t *testing.T) {
	input := `
	var t1 = jadwal.setelah(10, fungsi() {})
	var t2 = jadwal.setiap("20ms", fungsi() {})
	t1.batal()
	t2.henti()
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
		t.Fatalf("evaluasi gagal: %s", result.Inspect())
	}
}

package evaluator

import (
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestWaktuSetiapDanHenti(t *testing.T) {
	input := `paket utama
pakai "waktu"

var counter = 0
var pewaktu = waktu.setiap(20, fungsi() {
    counter = counter + 1
})

waktu.tidur(65)
pewaktu.hentikan()
var hasilSebelum = counter
waktu.tidur(50)
var hasilSesudah = counter
`
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := InitGlobalEnvironment()
	Eval(prog, env)

	cObj, ok := env.Get("hasilSebelum")
	if !ok {
		t.Fatalf("variabel hasilSebelum tidak ditemukan")
	}
	valBefore := cObj.(*Integer).Value
	if valBefore < 1 {
		t.Fatalf("waktu.setiap tidak menjalankan fungsi berkala, nilai: %d", valBefore)
	}

	cObjAfter, ok := env.Get("hasilSesudah")
	if !ok {
		t.Fatalf("variabel hasilSesudah tidak ditemukan")
	}
	valAfter := cObjAfter.(*Integer).Value
	if valAfter != valBefore {
		t.Fatalf("pewaktu.hentikan() gagal menghentikan ticker, nilai bertambah dari %d menjadi %d", valBefore, valAfter)
	}
}

func TestWaktuSetelah(t *testing.T) {
	input := `paket utama
pakai "waktu"

var status = "menunggu"
waktu.setelah(30, fungsi() {
    status = "selesai"
})

waktu.tidur(10)
var statusAwal = status
waktu.tidur(50)
var statusAkhir = status
`
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := InitGlobalEnvironment()
	Eval(prog, env)

	sAwal, _ := env.Get("statusAwal")
	if sAwal.(*String).Value != "menunggu" {
		t.Fatalf("statusAwal seharusnya 'menunggu', diperoleh: %s", sAwal.Inspect())
	}

	sAkhir, _ := env.Get("statusAkhir")
	if sAkhir.(*String).Value != "selesai" {
		t.Fatalf("statusAkhir seharusnya 'selesai', diperoleh: %s", sAkhir.Inspect())
	}
}

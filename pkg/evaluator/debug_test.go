package evaluator

import (
	"strings"
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestModulDebug(t *testing.T) {
	input := `
	// 1. Uji debug.dump
	var data = {"nama": "Bahasa Ku", "versi": 2, "fitur": ["web", "cli", "debug"]}
	var teksDump = debug.dump(data)

	// 2. Uji debug.profil
	var profilRes = debug.profil("penjumlahan_loop", fungsi() {
		var total = 0
		untuk var i = 0; i < 1000; i = i + 1 {
			total = total + i
		}
		kembalikan total
	})
	var namaProfil = profilRes["nama"]
	var totalProfil = profilRes["hasil"]

	// 3. Uji debug.memori
	var mem = debug.memori()
	var alokasi = mem["alokasiBita"]
	var goroutine = mem["jumlahGoroutine"]

	// 4. Uji debug.kondisi (assertion sukses)
	debug.kondisi(10 > 5, "10 harus lebih besar dari 5")

	// 5. Uji debug.jejak di dalam fungsi
	var jejakStr = ""
	var fTingkat1 = fungsi() {
		jejakStr = debug.jejak("cek pemanggilan")
	}
	fTingkat1()
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

	valDump, _ := env.Get("teksDump")
	if str, ok := valDump.(*String); !ok || !strings.Contains(str.Value, "peta(3)") {
		t.Fatalf("dump teks salah: %v", valDump)
	}

	valNamaProfil, _ := env.Get("namaProfil")
	if valNamaProfil.Inspect() != "penjumlahan_loop" {
		t.Fatalf("namaProfil salah: %v", valNamaProfil)
	}

	valTotalProfil, _ := env.Get("totalProfil")
	if i, ok := valTotalProfil.(*Integer); !ok || i.Value != 499500 {
		t.Fatalf("totalProfil salah, dapat: %v", valTotalProfil)
	}

	valAlokasi, _ := env.Get("alokasi")
	if i, ok := valAlokasi.(*Integer); !ok || i.Value <= 0 {
		t.Fatalf("alokasi memori salah: %v", valAlokasi)
	}

	valJejak, _ := env.Get("jejakStr")
	if str, ok := valJejak.(*String); !ok || !strings.Contains(str.Value, "Jejak Eksekusi") {
		t.Fatalf("jejak string salah: %v", valJejak)
	}

	// 6. Uji debug.kondisi gagal (harus memicu panik)
	inputGagal := `debug.kondisi(2 == 5, "2 tidak sama dengan 5!")`
	lG := lexer.New(inputGagal)
	pG := parser.New(lG)
	resG := Eval(pG.ParseProgram(), env)
	if resG == nil || resG.Type() != PANIC_OBJ {
		t.Fatalf("debug.kondisi seharusnya memicu PANIC_OBJ saat gagal, dapat: %v", resG)
	}
}

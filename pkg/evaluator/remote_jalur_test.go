package evaluator

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func evalTestProgram(src string, env *Environment) Object {
	l := lexer.New(src)
	p := parser.New(l)
	prog := p.ParseProgram()
	return Eval(prog, env)
}

func TestRemoteAndJalurModules(t *testing.T) {
	// 1. Uji modul jalur
	env := InitGlobalEnvironment()
	res := evalTestProgram(`
		pakai "jalur"
		var gab = jalur.gabung("folder", "sub", "berkas.txt")
		var nama = jalur.nama(gab)
		var ext = jalur.ekstensi(gab)
		var par = jalur.induk(gab)
		var abs = jalur.apakahAbsolut("c:/test") || jalur.apakahAbsolut("/test")
	`, env)

	if p, ok := res.(*PanicObject); ok {
		t.Fatalf("Panik saat modul jalur: %s", p.Inspect())
	}

	gabVal, _ := env.Get("gab")
	if gabVal == nil || (gabVal.Inspect() != "folder/sub/berkas.txt" && gabVal.Inspect() != "folder\\sub\\berkas.txt") {
		t.Errorf("jalur.gabung salah: %v", gabVal)
	}

	namaVal, _ := env.Get("nama")
	if namaVal == nil || namaVal.Inspect() != "berkas.txt" {
		t.Errorf("jalur.nama salah: %v", namaVal)
	}

	extVal, _ := env.Get("ext")
	if extVal == nil || extVal.Inspect() != ".txt" {
		t.Errorf("jalur.ekstensi salah: %v", extVal)
	}

	// 2. Uji Remote HTTP Module Import
	remoteKuCode := `
		var PesanRemote = "Halo dari Remote Server!"
		fungsi TambahRemote(a, b) {
			kembalikan a + b
		}
	`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, remoteKuCode)
	}))
	defer ts.Close()

	remoteURL := ts.URL + "/modul_remote.ku"

	env2 := InitGlobalEnvironment()
	testKu := fmt.Sprintf(`
		pakai %q
		var hasil = TambahRemote(10, 25)
		var salam = PesanRemote
	`, remoteURL)

	res2 := evalTestProgram(testKu, env2)
	if p, ok := res2.(*PanicObject); ok {
		t.Fatalf("Panik saat remote import: %s", p.Inspect())
	}

	hasilVal, ok := env2.Get("hasil")
	if !ok || hasilVal.Inspect() != "35" {
		t.Errorf("Ekspektasi hasil = 35 dari remote module, dapat: %v", hasilVal)
	}

	salamVal, ok := env2.Get("salam")
	if !ok || salamVal.Inspect() != "Halo dari Remote Server!" {
		t.Errorf("Ekspektasi salam = 'Halo dari Remote Server!', dapat: %v", salamVal)
	}

	// Bersihkan cache jika dibuat di temp
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".bahasa-ku", "cache")
	_ = os.RemoveAll(cacheDir)
}

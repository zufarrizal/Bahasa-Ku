package evaluator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestModulArsipZip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ku_test_arsip_*")
	if err != nil {
		t.Fatalf("gagal membuat temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	file1 := filepath.Join(tempDir, "berkas1.txt")
	file2 := filepath.Join(tempDir, "berkas2.txt")
	zipPath := filepath.Join(tempDir, "hasil.zip")
	outDir := filepath.Join(tempDir, "ekstrak")

	_ = os.WriteFile(file1, []byte("Halo dari Berkas 1"), 0644)
	_ = os.WriteFile(file2, []byte("Halo dari Berkas 2"), 0644)

	input := `paket utama
pakai "arsip"

var suksesBuat = arsip.buatZip("` + filepath.ToSlash(zipPath) + `", ["` + filepath.ToSlash(file1) + `", "` + filepath.ToSlash(file2) + `"])
var suksesEkstrak = arsip.ekstrakZip("` + filepath.ToSlash(zipPath) + `", "` + filepath.ToSlash(outDir) + `")
`
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := InitGlobalEnvironment()
	Eval(prog, env)

	sBuat, _ := env.Get("suksesBuat")
	if sBuat != TRUE {
		t.Fatalf("arsip.buatZip gagal")
	}

	sEkstrak, _ := env.Get("suksesEkstrak")
	if sEkstrak != TRUE {
		t.Fatalf("arsip.ekstrakZip gagal")
	}

	content1, err := os.ReadFile(filepath.Join(outDir, "berkas1.txt"))
	if err != nil || string(content1) != "Halo dari Berkas 1" {
		t.Fatalf("ekstraksi berkas1.txt tidak sesuai, err: %v, isi: %s", err, string(content1))
	}

	content2, err := os.ReadFile(filepath.Join(outDir, "berkas2.txt"))
	if err != nil || string(content2) != "Halo dari Berkas 2" {
		t.Fatalf("ekstraksi berkas2.txt tidak sesuai, err: %v, isi: %s", err, string(content2))
	}
}

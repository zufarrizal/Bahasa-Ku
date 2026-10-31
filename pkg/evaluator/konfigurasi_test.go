package evaluator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestModulKonfigurasi(t *testing.T) {
	tempDir := t.TempDir()
	jsonConfig := filepath.ToSlash(filepath.Join(tempDir, "config.json"))
	envConfig := filepath.ToSlash(filepath.Join(tempDir, "app.env"))

	_ = os.WriteFile(jsonConfig, []byte(`{"app_name": "Bahasa Ku Web", "port": 3000, "debug": true}`), 0644)
	_ = os.WriteFile(envConfig, []byte("DB_HOST=127.0.0.1\nDB_PORT=5432\nDB_USER=admin"), 0644)

	input := `
	// 1. Set & Ambil manual
	konfigurasi.atur("versi", "2.0")
	var v = konfigurasi.ambil("versi")

	// 2. Default value jika tidak ada
	var timeout = konfigurasi.ambilAngka("timeout_detik", 30)

	// 3. Muat berkas JSON
	konfigurasi.muatBerkas("` + jsonConfig + `")
	var appName = konfigurasi.ambilTeks("app_name")
	var appPort = konfigurasi.ambilAngka("port")
	var appDebug = konfigurasi.ambilBenarSalah("debug")

	// 4. Muat berkas .env
	konfigurasi.muatBerkas("` + envConfig + `")
	var dbHost = konfigurasi.ambilTeks("DB_HOST")
	var dbPort = konfigurasi.ambilAngka("DB_PORT")
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

	valV, _ := env.Get("v")
	if valV.Inspect() != "2.0" {
		t.Fatalf("versi salah, dapat: %v", valV)
	}

	valTimeout, _ := env.Get("timeout")
	if i, ok := valTimeout.(*Integer); !ok || i.Value != 30 {
		t.Fatalf("timeout salah, dapat: %v", valTimeout)
	}

	valAppName, _ := env.Get("appName")
	if valAppName.Inspect() != "Bahasa Ku Web" {
		t.Fatalf("appName salah, dapat: %v", valAppName)
	}

	valPort, _ := env.Get("appPort")
	if i, ok := valPort.(*Integer); !ok || i.Value != 3000 {
		t.Fatalf("appPort salah, dapat: %v", valPort)
	}

	valDebug, _ := env.Get("appDebug")
	if b, ok := valDebug.(*Boolean); !ok || !b.Value {
		t.Fatalf("appDebug salah, dapat: %v", valDebug)
	}

	valDbHost, _ := env.Get("dbHost")
	if valDbHost.Inspect() != "127.0.0.1" {
		t.Fatalf("dbHost salah, dapat: %v", valDbHost)
	}

	valDbPort, _ := env.Get("dbPort")
	if i, ok := valDbPort.(*Integer); !ok || i.Value != 5432 {
		t.Fatalf("dbPort salah, dapat: %v", valDbPort)
	}
}

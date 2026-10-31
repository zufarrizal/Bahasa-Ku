package evaluator

import (
	"crypto/sha512"
	"encoding/hex"
	"os"
	"runtime"
	"testing"
)

func TestSistemInfoAndSha512(t *testing.T) {
	env := InitGlobalEnvironment()
	res := evalTestProgram(`
		pakai "sistem"
		pakai "kripto"

		var arch = sistem.arsitektur()
		var osName = sistem.sistemOperasi()
		var host = sistem.namaHost()
		var pid = sistem.pid()

		var hash512 = kripto.sha512("Bahasa Ku Berdaulat")
	`, env)

	if p, ok := res.(*PanicObject); ok {
		t.Fatalf("Panik saat pengujian sistem dan kripto: %s", p.Inspect())
	}

	archVal, ok := env.Get("arch")
	if !ok || archVal.Inspect() != runtime.GOARCH {
		t.Errorf("sistem.arsitektur() = %v, ekspektasi: %s", archVal, runtime.GOARCH)
	}

	osVal, ok := env.Get("osName")
	if !ok || osVal.Inspect() != runtime.GOOS {
		t.Errorf("sistem.sistemOperasi() = %v, ekspektasi: %s", osVal, runtime.GOOS)
	}

	hostVal, ok := env.Get("host")
	if !ok || hostVal.Inspect() == "" {
		t.Errorf("sistem.namaHost() kosong: %v", hostVal)
	}

	pidVal, ok := env.Get("pid")
	if !ok {
		t.Errorf("sistem.pid() gagal diambil")
	} else if i, isInt := pidVal.(*Integer); !isInt || i.Value <= 0 {
		t.Errorf("sistem.pid() bukan int positif: %v", pidVal)
	}

	expectedHash := sha512.Sum512([]byte("Bahasa Ku Berdaulat"))
	expectedHashStr := hex.EncodeToString(expectedHash[:])

	hashVal, ok := env.Get("hash512")
	if !ok || hashVal.Inspect() != expectedHashStr {
		t.Errorf("kripto.sha512() = %v, ekspektasi: %s", hashVal, expectedHashStr)
	}

	// Uji sha512Berkas
	tmpFile, err := os.CreateTemp("", "test_sha512_*.txt")
	if err != nil {
		t.Fatalf("Gagal membuat berkas sementara: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	_, _ = tmpFile.WriteString("Konten berkas untuk verifikasi sha512")
	_ = tmpFile.Close()

	env2 := InitGlobalEnvironment()
	res2 := evalTestProgram(`
		pakai "kripto"
		var berkasHash = kripto.sha512Berkas("`+filepathToSlash(tmpFile.Name())+`")
	`, env2)

	if p, ok := res2.(*PanicObject); ok {
		t.Fatalf("Panik sha512Berkas: %s", p.Inspect())
	}

	bHashVal, ok := env2.Get("berkasHash")
	if !ok || bHashVal.Inspect() == "" {
		t.Errorf("kripto.sha512Berkas() kosong: %v", bHashVal)
	}
}

func filepathToSlash(p string) string {
	res := ""
	for _, r := range p {
		if r == '\\' {
			res += "/"
		} else {
			res += string(r)
		}
	}
	return res
}

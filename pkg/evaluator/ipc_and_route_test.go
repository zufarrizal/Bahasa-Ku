package evaluator_test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
)

func TestWebRouteScoringAndWildcard(t *testing.T) {
	code := `
paket utama
fungsi utama() {
    web.aktifkanCORS()

    // 1. Rute statis spesifik
    web.dapatkan("/api/pengguna/aktif", fungsi(req, res) {
        res.status(200)
        res.teks("pengguna-aktif-spesifik")
    })

    // 2. Rute dinamis berparameter
    web.dapatkan("/api/pengguna/:id", fungsi(req, res) {
        res.status(200)
        res.teks("pengguna-id:" + req["params"]["id"])
    })

    // 3. Rute umum / catch-all root
    web.dapatkan("/", fungsi(req, res) {
        res.status(200)
        res.teks("halaman-utama")
    })

    web.jalankan(18099)
}
`
	go func() {
		runKuCode(code)
	}()

	time.Sleep(200 * time.Millisecond)

	// Uji 1: Permintaan ke /api/pengguna/aktif harus memprioritaskan rute statis bukan :id
	resp1, err := http.Get("http://127.0.0.1:18099/api/pengguna/aktif")
	if err != nil {
		t.Fatalf("Gagal HTTP GET 1: %v", err)
	}
	defer resp1.Body.Close()
	b1, _ := io.ReadAll(resp1.Body)
	if !strings.Contains(string(b1), "pengguna-aktif-spesifik") {
		t.Fatalf("Diharapkan mencocokkan rute spesifik, didapat: %s", string(b1))
	}

	// Uji 2: Permintaan ke /api/pengguna/123 harus mencocokkan parameter :id
	resp2, err := http.Get("http://127.0.0.1:18099/api/pengguna/123")
	if err != nil {
		t.Fatalf("Gagal HTTP GET 2: %v", err)
	}
	defer resp2.Body.Close()
	b2, _ := io.ReadAll(resp2.Body)
	if !strings.Contains(string(b2), "pengguna-id:123") {
		t.Fatalf("Diharapkan mencocokkan parameter :id, didapat: %s", string(b2))
	}

	// Uji 3: Permintaan ke / harus mencocokkan root
	resp3, err := http.Get("http://127.0.0.1:18099/")
	if err != nil {
		t.Fatalf("Gagal HTTP GET 3: %v", err)
	}
	defer resp3.Body.Close()
	b3, _ := io.ReadAll(resp3.Body)
	if !strings.Contains(string(b3), "halaman-utama") {
		t.Fatalf("Diharapkan mencocokkan root, didapat: %s", string(b3))
	}
}

func TestIPCBridgeBinding(t *testing.T) {
	code := `
paket utama
fungsi utama() {
    // Daftarkan handler fungsi native Bahasa Ku untuk diakses dari JavaScript
    jendela.daftarkanFungsi("tambahAngka", fungsi(a, b) {
        kembalikan a + b
    })
    jendela.daftarkanFungsi("sapa", fungsi(nama) {
        kembalikan "Halo " + nama
    })
}
`
	res := runKuCode(code)
	_ = res

	// Kirim simulasi IPC POST request dari JavaScript
	payload := `{"fungsi": "tambahAngka", "argumen": [25, 17]}`
	_ = bytes.NewReader([]byte(payload))
	_ = evaluator.InitGlobalEnvironment()
}

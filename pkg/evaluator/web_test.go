package evaluator_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestWebServerRestApi(t *testing.T) {
	serverCode := `
paket utama
fungsi utama() {
    web.aktifkanCORS()

    // Endpoint GET dengan URL params :id
    web.dapatkan("/api/pengguna/:id", fungsi(permintaan, respons) {
        var id = permintaan["params"]["id"]
        var filter = permintaan["kueri"]["status"]
        respons.status(200)
        respons.json({"id": id, "status": filter, "aktif": benar})
    })

    // Endpoint POST
    web.pos("/api/pesan", fungsi(permintaan, respons) {
        var badan = permintaan["badan"]
        respons.json({"diterima": badan})
    })

    // Jalankan server di port 18080 secara lokal
    web.jalankan(18080)
}
`
	go func() {
		runKuCode(serverCode)
	}()

	// Tunggu server mulai
	time.Sleep(200 * time.Millisecond)

	// Uji GET request
	resp, err := http.Get("http://127.0.0.1:18080/api/pengguna/42?status=vip")
	if err != nil {
		t.Fatalf("Gagal menghubungi server web: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Status bukan 200: %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("Gagal decode JSON: %v", err)
	}

	if data["id"] != "42" {
		t.Errorf("Diharapkan id '42', didapat '%v'", data["id"])
	}
	if data["status"] != "vip" {
		t.Errorf("Diharapkan status 'vip', didapat '%v'", data["status"])
	}
	if data["aktif"] != true {
		t.Errorf("Diharapkan aktif true, didapat '%v'", data["aktif"])
	}

	// Periksa header CORS
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("CORS header belum aktif")
	}
}

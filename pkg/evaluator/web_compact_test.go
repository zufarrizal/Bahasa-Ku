package evaluator

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestWebCompactResponseHelpers(t *testing.T) {
	input := `
	web.dapatkan("/tes/sukses", fungsi(minta, respon) {
		respon.sukses({"id": 1, "nama": "Buku"}, "Berhasil diambil")
	})

	web.dapatkan("/tes/gagal", fungsi(minta, respon) {
		respon.gagal("Saldo tidak mencukupi", 402)
	})

	web.dapatkan("/tes/404", fungsi(minta, respon) {
		respon.tidakDitemukan("Barang tidak ada")
	})

	web.dapatkan("/tes/403", fungsi(minta, respon) {
		respon.dilarang("Anda bukan admin")
	})
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

	webMap := env.GetStore()["web"].(*Map)
	handlerBuiltin := webMap.Pairs["_penangan"].(*Builtin)
	httpHandler := handlerBuiltin.Fn().(*NativeHandlerObject).Handler

	// 1. Tes respon.sukses
	w1 := httptest.NewRecorder()
	httpHandler.ServeHTTP(w1, httptest.NewRequest("GET", "/tes/sukses", nil))
	if w1.Code != 200 || !strings.Contains(w1.Body.String(), `"sukses":true`) || !strings.Contains(w1.Body.String(), "Berhasil diambil") {
		t.Fatalf("respon.sukses tidak sesuai: %d %s", w1.Code, w1.Body.String())
	}

	// 2. Tes respon.gagal
	w2 := httptest.NewRecorder()
	httpHandler.ServeHTTP(w2, httptest.NewRequest("GET", "/tes/gagal", nil))
	if w2.Code != 402 || !strings.Contains(w2.Body.String(), `"sukses":false`) || !strings.Contains(w2.Body.String(), "Saldo tidak mencukupi") {
		t.Fatalf("respon.gagal tidak sesuai: %d %s", w2.Code, w2.Body.String())
	}

	// 3. Tes respon.tidakDitemukan
	w3 := httptest.NewRecorder()
	httpHandler.ServeHTTP(w3, httptest.NewRequest("GET", "/tes/404", nil))
	if w3.Code != 404 || !strings.Contains(w3.Body.String(), "Barang tidak ada") {
		t.Fatalf("respon.tidakDitemukan tidak sesuai: %d %s", w3.Code, w3.Body.String())
	}

	// 4. Tes respon.dilarang
	w4 := httptest.NewRecorder()
	httpHandler.ServeHTTP(w4, httptest.NewRequest("GET", "/tes/403", nil))
	if w4.Code != 403 || !strings.Contains(w4.Body.String(), "Anda bukan admin") {
		t.Fatalf("respon.dilarang tidak sesuai: %d %s", w4.Code, w4.Body.String())
	}
}

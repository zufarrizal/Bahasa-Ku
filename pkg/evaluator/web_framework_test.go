package evaluator

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func TestFrameworkWebFeatures(t *testing.T) {
	input := `
	var api = web.grup("/api/v1")

	api.dapatkan("/info", fungsi(minta, respon) {
		respon.status(200).json({
			"status": "aktif",
			"versi": "1.0"
		})
	})

	api.pos("/proses", fungsi(minta, respon) {
		var payload = minta["json"]
		var cookieSesi = minta["cookie"]["sesi"] ?? "tidak-ada"
		respon.setCookie("tokenBaru", "xyz123")
		respon.status(201).json({
			"diterima": payload["nama"],
			"sesi": cookieSesi
		})
	})

	api.tambal("/perbarui", fungsi(minta, respon) {
		respon.status(200).json({"metode": minta["metode"]})
	})

	web.dapatkan("/pindah", fungsi(minta, respon) {
		respon.alihkan("/beranda", 301)
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

	webMapObj, ok := env.Get("web")
	if !ok {
		t.Fatalf("modul web tidak ditemukan di env")
	}
	webMap := webMapObj.(*Map)
	handlerBuiltin := webMap.Pairs["_penangan"].(*Builtin)
	handlerObj := handlerBuiltin.Fn().(*NativeHandlerObject)
	httpHandler := handlerObj.Handler

	// Test 1: GET /api/v1/info
	req1 := httptest.NewRequest("GET", "/api/v1/info", nil)
	w1 := httptest.NewRecorder()
	httpHandler.ServeHTTP(w1, req1)

	if w1.Code != 200 {
		t.Fatalf("status code salah, dapat: %d", w1.Code)
	}
	if !strings.Contains(w1.Body.String(), `"status":"aktif"`) {
		t.Fatalf("body respon tidak sesuai: %s", w1.Body.String())
	}

	// Test 2: POST /api/v1/proses dengan JSON & Cookie
	bodyPayload := `{"nama": "Budi Santoso"}`
	req2 := httptest.NewRequest("POST", "/api/v1/proses", bytes.NewBufferString(bodyPayload))
	req2.Header.Set("Content-Type", "application/json")
	req2.AddCookie(&http.Cookie{Name: "sesi", Value: "token-rahasia-999"})
	w2 := httptest.NewRecorder()
	httpHandler.ServeHTTP(w2, req2)

	if w2.Code != 201 {
		t.Fatalf("status code salah, dapat: %d", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), `"diterima":"Budi Santoso"`) || !strings.Contains(w2.Body.String(), `"sesi":"token-rahasia-999"`) {
		t.Fatalf("body json auto-parse salah: %s", w2.Body.String())
	}
	cookies := w2.Result().Cookies()
	var foundCookie bool
	for _, c := range cookies {
		if c.Name == "tokenBaru" && c.Value == "xyz123" {
			foundCookie = true
			break
		}
	}
	if !foundCookie {
		t.Fatalf("cookie setCookie tokenBaru tidak ditemukan di response headers")
	}

	// Test 3: PATCH /api/v1/perbarui
	req3 := httptest.NewRequest("PATCH", "/api/v1/perbarui", nil)
	w3 := httptest.NewRecorder()
	httpHandler.ServeHTTP(w3, req3)

	if w3.Code != 200 {
		t.Fatalf("status patch salah: %d", w3.Code)
	}
	if !strings.Contains(w3.Body.String(), `"metode":"PATCH"`) {
		t.Fatalf("metode patch salah: %s", w3.Body.String())
	}

	// Test 4: Redirect /pindah -> /beranda
	req4 := httptest.NewRequest("GET", "/pindah", nil)
	w4 := httptest.NewRecorder()
	httpHandler.ServeHTTP(w4, req4)

	if w4.Code != 301 {
		t.Fatalf("status redirect salah, dapat %d", w4.Code)
	}
	if loc := w4.Header().Get("Location"); loc != "/beranda" {
		t.Fatalf("location redirect salah, dapat %s", loc)
	}
}

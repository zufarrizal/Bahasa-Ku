package evaluator_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
)

func TestModulKompakBootstrap(t *testing.T) {
	code := `
paket utama

pakai "bd"
pakai "konf"
pakai "sd"
pakai "ingat"
pakai "sinyal"

fungsi utama() {
	// 1. Uji konf (konfigurasi)
	konf.atur("Aplikasi", "Bahasa Ku")
	jika konf.ambil("Aplikasi") != "Bahasa Ku" {
		kembalikan 1
	}

	// 2. Uji ingat (singgahan)
	ingat.taruh("sesi", 12345, 60)
	jika ingat.ambil("sesi") != 12345 {
		kembalikan 2
	}

	// 3. Uji sd (strukturData)
	var tumpuk = sd.tumpukan()
	tumpuk.dorong(100)
	tumpuk.dorong(200)
	jika tumpuk.tarik() != 200 {
		kembalikan 3
	}

	// 4. Uji sinyal (peristiwa)
	var diterima = 0
	sinyal.pada("pesan", fungsi(data) {
		diterima = data
	})
	sinyal.pancarkan("pesan", 999)
	jika diterima != 999 {
		kembalikan 4
	}

	// 5. Uji tunggu (sistem.tunggu / kelompokTunggu)
	var wg = sistem.tunggu()
	wg.tambah(1)
	wg.selesai()
	wg.tunggu()

	// 6. Uji tulis (bawaan global langsung)
	tulis("Uji fungsi tulis bawaan berhasil")

	// 7. Uji fungsi universal ringkas: tidur, format, urai, kodekan
	tidur(10) // tidur 10 milidetik
	var strFmt = format("Nomor: %03d", 7)
	jika strFmt != "Nomor: 007" {
		kembalikan 5
	}

	var dataAsli = {"nama": "Bahasa Ku", "versi": 1}
	var jsonStr = kodekan(dataAsli)
	var dataHasil = urai(jsonStr)
	jika dataHasil["nama"] != "Bahasa Ku" || dataHasil["versi"] != 1 {
		kembalikan 6
	}

	// 8. Uji konversi callable angka(...) dan teks(...)
	var num = angka("12345")
	jika num != 12345 {
		kembalikan 7
	}
	var str = teks(num)
	jika str != "12345" {
		kembalikan 8
	}

	// Uji modul angka dan teks tetap berfungsi normal
	var bin = angka.biner(10)
	jika bin != "1010" {
		kembalikan 9
	}
	var besar = teks.kapital("halo")
	jika besar != "HALO" {
		kembalikan 10
	}

	// 9. Uji sekarang()
	var wkt = sekarang()
	jika wkt["tahun"] < 2026 {
		kembalikan 11
	}

	// 10. Uji rentang(), kunci(), dan nilai()
	var r = rentang(1, 5) // [1, 2, 3, 4] eksklusif end (v < end ala range Go/Python)
	jika panjang(r) != 4 || r[0] != 1 || r[3] != 4 {
		kembalikan 12
	}

	var d = {"a": 10, "b": 20}
	var k = kunci(d)
	var v = nilai(d)
	jika panjang(k) != 2 || panjang(v) != 2 {
		kembalikan 13
	}

	// 11. Uji baca() pada go.mod
	var isiGoMod = baca("go.mod")
	jika !teks.mengandung(isiGoMod, "module github.com/zufarrizal/bahasa-ku") {
		kembalikan 14
	}

	// 12. Uji operator fallback null coalescing (??)
	var x = kosong
	var fallback = x ?? "NilaiCadangan"
	jika fallback != "NilaiCadangan" {
		kembalikan 15
	}

	var adaNilai = "Asli"
	var hasilPasti = adaNilai ?? "Cadangan"
	jika hasilPasti != "Asli" {
		kembalikan 16
	}

	// 13. Uji auto string concatenation (+) dengan tipe non-string
	var strGabung = "Nilai:" + 100 + benar
	jika strGabung != "Nilai:100benar" {
		kembalikan 17
	}

	// 14. Uji method chaining pada Teks (String)
	var teksHasil = "  Halo Dunia  ".pangkas().kecil()
	jika teksHasil != "halo dunia" {
		kembalikan 18
	}
	var larikKata = "apel,jeruk".pisah(",")
	jika panjang(larikKata) != 2 || larikKata[0] != "apel" {
		kembalikan 19
	}

	// 15. Uji method chaining pada Larik (Array)
	var list = [1, 2, 3, 4]
	var saringGenap = list.saring(fungsi(n) { kembalikan n % 2 == 0 })
	jika panjang(saringGenap) != 2 || saringGenap[0] != 2 {
		kembalikan 20
	}
	var dikaliDua = saringGenap.petakan(fungsi(n) { kembalikan n * 2 })
	jika dikaliDua[0] != 4 || dikaliDua[1] != 8 {
		kembalikan 21
	}

	// 16. Uji method chaining pada Peta (Map)
	var user = {"nama": "Ahmad", "kota": "Jakarta"}
	jika !user.ada("nama") || user.panjang() != 2 {
		kembalikan 22
	}
	var umurFallback = user.ambil("umur", 25)
	jika umurFallback != 25 {
		kembalikan 23
	}
	user.hapus("kota")
	jika user.panjang() != 1 {
		kembalikan 24
	}

	// 17. Uji Web Context Helper bootstraping
	web.dapatkan("/uji-konteks", fungsi(req, resp) {
		jika req["token"] != "rahasia123" {
			resp.takTerotorisasi("Token tidak sah")
		} lainnya {
			resp.sukses({"ip": req["ip"], "aman": req["apakahAman"]})
		}
	})

	// 18. Uji Swap Variabel ala Go (a, b = b, a)
	var x1, y1 = 10, 20
	x1, y1 = y1, x1
	jika x1 != 20 || y1 != 10 {
		kembalikan 25
	}

	// 19. Uji Comma-Ok idiom ala Go pada Peta
	var kamus = {"bahasa": "Ku"}
	valAda, okAda := kamus["bahasa"]
	jika !okAda || valAda != "Ku" {
		kembalikan 26
	}
	valNihil, okNihil := kamus["tidak_ada"]
	jika okNihil || valNihil != kosong {
		kembalikan 27
	}

	// 20. Uji Blank Identifier (_) ala Go
	_, okHanya := kamus["bahasa"]
	jika !okHanya {
		kembalikan 28
	}

	// 21. Uji web helper baru: cors(), catatAkses(), pulihkan()
	web.cors()
	web.catatAkses()
	web.pulihkan()

	// 22. Uji Web Context handler Bahasa Indonesia (fungsi(konteks)) & Penataan Data
	web.dapatkan("/uji-konteks-id/:id", fungsi(konteks) {
		id := konteks.param("id")
		q := konteks.kueri("filter", "semua")
		konteks.taruh("pengguna", "budi")
		siapa := konteks.ambil("pengguna")
		// Uji kepala, badan, kaki, kue
		konteks.kepala("X-Kerangka", "Bahasa-Ku")
		konteks.kaki("X-Checksum", "abc123xyz")
		konteks.kue("sesi_id", "token123")
		konteks.sukses({"id": id, "filter": q, "user": siapa})
	})

	// Scoped middleware pada grup rute
	admin := web.grup("/admin")
	admin.gunakan(fungsi(konteks, lanjut) {
		konteks.taruh("peran", "administrator")
		lanjut()
	})
	admin.dapatkan("/dasbor", fungsi(konteks) {
		konteks.sukses({"peran": konteks.ambil("peran")})
	})

	// 23. Uji Basis Data: Transaksi & Masukkan Polimorfik (otomatis batch insert jika larik)
	dbUji := basisdata.bukaBerkas("test_kompak_tx.db")
	dbUji.eksekusi("DROP TABLE IF EXISTS item")
	dbUji.eksekusi("CREATE TABLE item (id INTEGER PRIMARY KEY, nama TEXT, harga INTEGER)")

	// Batch Insert cukup dengan .masukkan([...])
	okBanyak := dbUji.tabel("item").masukkan([
		{"nama": "Buku", "harga": 5000},
		{"nama": "Pena", "harga": 2000}
	])
	jika !okBanyak {
		kembalikan 29
	}

	// Transaction commit
	okTx := dbUji.transaksi(fungsi(tx) {
		tx.eksekusi("INSERT INTO item (nama, harga) VALUES ('Penghapus', 1000)")
		kembalikan benar
	})
	jika !okTx {
		kembalikan 30
	}

	// Cek total data dalam tabel dengan .hitung()
	totalBaris := dbUji.tabel("item").hitung()
	jika totalBaris < 3 {
		kembalikan 31
	}

	// Cek paginasi dengan .halaman() & .lewat()
	hal1 := dbUji.tabel("item").halaman(1, 2).ambilSemua()
	jika panjang(hal1) != 2 {
		kembalikan 32
	}

	hal2 := dbUji.tabel("item").halaman(2, 2).ambilSemua()
	jika panjang(hal2) != 1 {
		kembalikan 33
	}

	// 24. Uji Query Builder Proyeksi Kolom .pilih("nama, harga")
	itemKolom := dbUji.tabel("item").pilih("nama", "harga").dimana("nama", "Buku").ambilSatu()
	jika itemKolom == kosong || itemKolom["nama"] != "Buku" {
		kembalikan 34
	}
	// Pastikan kolom id tidak terambil karena tidak dipilih
	jika itemKolom["id"] != kosong && itemKolom["id"] != "" {
		kembalikan 35
	}

	dbUji.tutup()

	// 25. Uji mat.lantai, mat.langit, kripto.kodekanBase64, kripto.dekodekanBase64, dan konteks.denganBatal
	jika mat.lantai(3.9) != 3 || mat.langit(3.1) != 4 {
		kembalikan 36
	}
	b64 := kripto.kodekanBase64("Bahasa Ku")
	jika kripto.dekodekanBase64(b64) != "Bahasa Ku" {
		kembalikan 37
	}
	ktk, batal := konteks.denganBatal()
	jika ktk.apakahBatal() {
		kembalikan 38
	}
	batal()
	jika !ktk.apakahBatal() {
		kembalikan 39
	}

	kembalikan 0
}
`
	val := runKuCode(code)
	_ = os.Remove("test_kompak_tx.db")
	intVal, ok := val.(*evaluator.Integer)
	if !ok {
		t.Fatalf("Nilai bukan Integer: %v", val)
	}
	if intVal.Value != 0 {
		t.Fatalf("TestModulKompakBootstrap gagal, kode keluar: %d", intVal.Value)
	}
}

func TestWebKonteksKepalaBadanKakiKue(t *testing.T) {
	code := `
paket utama

fungsi utama() {
	web.pos("/api/uji/:id", fungsi(konteks) {
		// Uji akses ganda sebagai peta ["..."] maupun sebagai fungsi ("...")
		id1 := konteks.param["id"]
		id2 := konteks.param("id")
		q1 := konteks.kueri["mode"]
		q2 := konteks.kueri("mode", "bawaan")
		h1 := konteks.kepala["X-Token"]
		h2 := konteks.kepala("x-token")
		k1 := konteks.kue["sesi"]
		k2 := konteks.kue("sesi")

		dataMasuk := konteks.ikat()
		jsonMasuk := konteks.json()

		// Atur kepala, kaki, dan kue respons
		konteks.kepala("X-Server", "Bahasa-Ku")
		konteks.kaki("X-Checksum", "sha256-valid")
		konteks.kue("sesi_baru", "xyz987")

		konteks.json({
			"idSama": id1 == id2 && id1 == "42",
			"kueriSama": q1 == q2 && q1 == "cepat",
			"kepalaSama": h1 == h2 && h1 == "rahasia",
			"kueSama": k1 == k2 && k1 == "aktif123",
			"namaIkat": dataMasuk["nama"],
			"namaJson": jsonMasuk["nama"],
			"badanMentah": konteks.badan
		}, 201)
	})
	kembalikan web._penangan()
}
`
	obj := runKuCode(code)
	handlerObj, ok := obj.(*evaluator.NativeHandlerObject)
	if !ok || handlerObj.Handler == nil {
		t.Fatalf("Gagal mendapatkan NativeHandlerObject: %v", obj)
	}

	reqBody := `{"nama":"Budi"}`
	req := httptest.NewRequest(http.MethodPost, "/api/uji/42?mode=cepat", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Token", "rahasia")
	req.AddCookie(&http.Cookie{Name: "sesi", Value: "aktif123"})

	rec := httptest.NewRecorder()
	handlerObj.Handler.ServeHTTP(rec, req)

	if rec.Code != 201 {
		t.Fatalf("Diharapkan status 201, didapat %d (body: %s)", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Server") != "Bahasa-Ku" {
		t.Fatalf("Kepala X-Server tidak sesuai: %s", rec.Header().Get("X-Server"))
	}
	if rec.Header().Get("X-Checksum") != "sha256-valid" {
		t.Fatalf("Kaki X-Checksum tidak sesuai: %s", rec.Header().Get("X-Checksum"))
	}
	bodyStr := rec.Body.String()
	for _, wajib := range []string{`"idSama":true`, `"kueriSama":true`, `"kepalaSama":true`, `"kueSama":true`, `"namaIkat":"Budi"`, `"namaJson":"Budi"`} {
		if !strings.Contains(bodyStr, wajib) {
			t.Fatalf("Respons JSON tidak mengandung %s: %s", wajib, bodyStr)
		}
	}
}

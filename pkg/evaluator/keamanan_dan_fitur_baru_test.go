package evaluator_test

import (
	"bufio"
	"bytes"
	"fmt"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func runKuCodeString(input string) *evaluator.Environment {
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	env := evaluator.InitGlobalEnvironment()
	evaluator.Eval(prog, env)
	return env
}

func TestSecurityAndAESCrypto(t *testing.T) {
	code := `
	pesan = "Halo Rahasia Indonesia 123!"
	kunci = "kunci-sangat-rahasia-32-byte-ok"

	sandi = kripto.enkripsiAES(pesan, kunci)
	pulih = kripto.dekripsiAES(sandi, kunci)
	kunciSalah = kripto.dekripsiAES(sandi, "kunci-salah-yang-lain")

	bersih = teks.sanitasiHTML("<script>alert('xss');</script>&\"'")
	`
	env := runKuCodeString(code)

	pulihVal, _ := env.Get("pulih")
	if pulihVal == nil || pulihVal.Inspect() != "Halo Rahasia Indonesia 123!" {
		t.Fatalf("AES dekripsi gagal: %v", pulihVal)
	}

	kunciSalahVal, _ := env.Get("kunciSalah")
	if kunciSalahVal != evaluator.NIL {
		t.Fatalf("AES dekripsi dengan kunci salah harus NIL, dapat: %v", kunciSalahVal)
	}

	bersihVal, _ := env.Get("bersih")
	expected := "&lt;script&gt;alert(&#x27;xss&#x27;);&lt;/script&gt;&amp;&quot;&#x27;"
	if bersihVal == nil || bersihVal.Inspect() != expected {
		t.Fatalf("Sanitasi HTML salah, dapat: %v, diharapkan: %v", bersihVal, expected)
	}
}

func TestDatabaseJoinAndMigration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ku_migrasi_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Buat 2 berkas migrasi
	mig1 := filepath.Join(tmpDir, "001_buat_tabel.sql")
	mig2 := filepath.Join(tmpDir, "002_tambah_indeks.sql")
	_ = os.WriteFile(mig1, []byte("CREATE TABLE produk (id INTEGER PRIMARY KEY, nama TEXT, id_kategori INTEGER); CREATE TABLE kategori (id INTEGER PRIMARY KEY, nama TEXT);"), 0644)
	_ = os.WriteFile(mig2, []byte("INSERT INTO kategori VALUES (1, 'Elektronik'); INSERT INTO produk VALUES (10, 'Laptop', 1);"), 0644)

	dbPath := filepath.Join(tmpDir, "test.db")
	dbPathEscaped := strings.ReplaceAll(dbPath, "\\", "/")

	code := `
	db = bd.sambung("sqlite", "` + dbPathEscaped + `")
	jumMigrasi = db.migrasi("` + strings.ReplaceAll(tmpDir, "\\", "/") + `")

	// Query Builder Gabung (JOIN)
	hasil = db.tabel("produk").gabung("kategori", "kategori.id = produk.id_kategori").pilih("produk.nama AS produk_nama, kategori.nama AS kategori_nama").ambil()
	`

	env := runKuCodeString(code)

	jumVal, _ := env.Get("jumMigrasi")
	if intVal, ok := jumVal.(*evaluator.Integer); !ok || intVal.Value != 2 {
		t.Fatalf("Migrasi harus mengeksekusi 2 berkas, dapat: %v", jumVal)
	}

	hasilVal, _ := env.Get("hasil")
	arr, ok := hasilVal.(*evaluator.Array)
	if !ok || len(arr.Elements) != 1 {
		t.Fatalf("Hasil join gagal atau tidak mengembalikan 1 baris: %v", hasilVal)
	}
}

func TestWebSecurityHeadersAndUpload(t *testing.T) {
	code := `
	web.aman()
	web.batasPermintaan(100, "1s")

	web.pos("/uji-unggah", fungsi(k) {
		berkas = k.berkas("foto")
		jika berkas != nil {
			k.teks(berkas.nama + ":" + teks(berkas.ukuran))
		} lainnya {
			k.teks("TIDAK_ADA", 400)
		}
	})

	web.dapatkan("/uji-aman", fungsi(k) {
		k.teks("aman-ok")
	})

	kembalikan web._penangan()
	`
	env := runKuCodeString(code)
	retObj, _ := env.Get("kembalikan")
	_ = retObj

	// Eksekusi langsung melalui penangan HTTP
	l := lexer.New(code)
	p := parser.New(l)
	prog := p.ParseProgram()
	penv := evaluator.InitGlobalEnvironment()
	evalRes := evaluator.Eval(prog, penv)

	handlerObj, ok := evalRes.(*evaluator.NativeHandlerObject)
	if !ok || handlerObj == nil || handlerObj.Handler == nil {
		t.Fatalf("Gagal inisialisasi handler HTTP dari web._penangan(): %v", evalRes)
	}

	// 1. Cek Header Keamanan OWASP via httptest
	reqAman := httptest.NewRequest(http.MethodGet, "/uji-aman", nil)
	recAman := httptest.NewRecorder()
	handlerObj.Handler.ServeHTTP(recAman, reqAman)

	if recAman.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("Header X-Content-Type-Options tidak sesuai: %s", recAman.Header().Get("X-Content-Type-Options"))
	}
	if recAman.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Errorf("Header X-Frame-Options tidak sesuai: %s", recAman.Header().Get("X-Frame-Options"))
	}

	// 2. Cek Upload Berkas Multipart
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("foto", "berkas_profil.png")
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("konten-gambar-palsu")
	part.Write(content)
	writer.Close()

	reqUpload := httptest.NewRequest(http.MethodPost, "/uji-unggah", body)
	reqUpload.Header.Set("Content-Type", writer.FormDataContentType())
	recUpload := httptest.NewRecorder()
	handlerObj.Handler.ServeHTTP(recUpload, reqUpload)

	respStr := recUpload.Body.String()
	if !strings.Contains(respStr, "berkas_profil.png") {
		t.Fatalf("Respons unggah salah: %s", respStr)
	}
}

func TestArgon2idAndStreamingAndPagination(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ku_streaming_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcFile := filepath.Join(tmpDir, "sumber.txt")
	dstFile := filepath.Join(tmpDir, "tujuan.txt")
	_ = os.WriteFile(srcFile, []byte("konten-aliran-sangat-besar-dan-cepat"), 0644)

	dbFile := filepath.Join(tmpDir, "pustaka.db")
	dbPathEscaped := strings.ReplaceAll(dbFile, "\\", "/")
	srcEscaped := strings.ReplaceAll(srcFile, "\\", "/")
	dstEscaped := strings.ReplaceAll(dstFile, "\\", "/")

	code := `
	// 1. Argon2id password hash & verify
	sandiAsli = "kunciKuatArgon2026!"
	hashArgon = kripto.hashArgon2(sandiAsli)
	cocok = kripto.cocokkanArgon2(sandiAsli, hashArgon)
	hasilSalah = kripto.cocokkanArgon2("sandiLain", hashArgon)

	// 2. Berkas alirkan (Zero copy streaming)
	statusAlir = berkas.alirkan("` + srcEscaped + `", "` + dstEscaped + `")

	// 3. Database pagination helper
	db = bd.sambung("sqlite", "` + dbPathEscaped + `")
	db.eksekusi("CREATE TABLE buku (id INTEGER PRIMARY KEY, judul TEXT);")
	untuk var i = 1; i <= 25; i = i + 1 {
		db.eksekusi("INSERT INTO buku (judul) VALUES (?);", "Buku Ke-" + teks(i))
	}

	halamanData = db.tabel("buku").urutkan("id ASC").ambilHalaman(2, 10)
	`

	env := runKuCodeString(code)

	cocokVal, _ := env.Get("cocok")
	if cocokVal != evaluator.TRUE {
		t.Fatalf("Argon2id verifikasi sandi asli gagal: %v", cocokVal)
	}

	salahVal, _ := env.Get("hasilSalah")
	if salahVal != evaluator.FALSE {
		t.Fatalf("Argon2id verifikasi sandi salah harus false: %v", salahVal)
	}

	statusAlirVal, _ := env.Get("statusAlir")
	if statusAlirVal != evaluator.TRUE {
		t.Fatalf("berkas.alirkan gagal: %v", statusAlirVal)
	}

	dstData, _ := os.ReadFile(dstFile)
	if string(dstData) != "konten-aliran-sangat-besar-dan-cepat" {
		t.Fatalf("Isi berkas hasil alirkan tidak cocok: %s", string(dstData))
	}

	halVal, _ := env.Get("halamanData")
	hMap, ok := halVal.(*evaluator.Map)
	if !ok {
		t.Fatalf("halamanData bukan Map: %v", halVal)
	}

	totalVal := hMap.Pairs["total"].(*evaluator.Integer).Value
	if totalVal != 25 {
		t.Fatalf("Total data harus 25, didapat: %d", totalVal)
	}

	totalHalVal := hMap.Pairs["totalHalaman"].(*evaluator.Integer).Value
	if totalHalVal != 3 {
		t.Fatalf("Total halaman harus 3 (25 / 10 = 3), didapat: %d", totalHalVal)
	}

	dataArr := hMap.Pairs["data"].(*evaluator.Array).Elements
	if len(dataArr) != 10 {
		t.Fatalf("Data per halaman harus 10, didapat: %d", len(dataArr))
	}
}

func TestVirtualMemoryFSAndStrictSandbox(t *testing.T) {
	code := `
	// 1. Uji Virtual Memory File System (berkas.memori)
	berkas.memori.taruh("aset/gaya.css", "body { background: #000; }")
	berkas.memori.taruh("aset/skrip.js", "console.log('halo')")

	cssAda = berkas.memori.ada("aset/gaya.css")
	isiCss = berkas.memori.baca("aset/gaya.css")
	daftarAset = berkas.memori.daftar("aset")

	// 2. Uji Multi-Channel Broadcasting (web.saluran)
	saluranPesanan = web.saluran("pesanan")
	pesanMasuk = ""
	unsubscribe = saluranPesanan.langgan(fungsi(p) {
		pesanMasuk = p
	})
	saluranPesanan.kirimSemua("pesanan-baru-123")
	`

	env := runKuCodeString(code)

	cssAdaVal, _ := env.Get("cssAda")
	if cssAdaVal != evaluator.TRUE {
		t.Fatalf("berkas.memori.ada gagal: %v", cssAdaVal)
	}

	isiCssVal, _ := env.Get("isiCss")
	if isiCssVal == nil || isiCssVal.Inspect() != "body { background: #000; }" {
		t.Fatalf("berkas.memori.baca gagal: %v", isiCssVal)
	}

	daftarVal, _ := env.Get("daftarAset")
	if arr, ok := daftarVal.(*evaluator.Array); !ok || len(arr.Elements) < 2 {
		t.Fatalf("berkas.memori.daftar gagal, dapat: %v", daftarVal)
	}

	// 3. Uji Strict Sandboxing
	evaluator.SetStrictSandboxMode(true)
	defer evaluator.SetStrictSandboxMode(false)

	codeSandbox := `
	hasilCmd = sistem.perintah("dir")
	`
	envSandbox := runKuCodeString(codeSandbox)
	resCmd, _ := envSandbox.Get("hasilCmd")
	cMap, ok := resCmd.(*evaluator.Map)
	if !ok {
		t.Fatalf("hasilCmd bukan Map: %v", resCmd)
	}
	if cMap.Pairs["sukses"] != evaluator.FALSE {
		t.Fatalf("Mode sandboxing ketat harus memblokir sistem.perintah!")
	}
}

func TestAsyncQueueJWTRevocationAndVersionedRoutes(t *testing.T) {
	code := `
	// 1. Uji Antrean Tugas Asinkron (antrean.dorong & antrean.proses)
	terproses = ""
	antrean.proses("kirim_notif", fungsi(data) {
		terproses = data["tujuan"]
	})
	antrean.dorong("kirim_notif", {"tujuan": "pengguna@domain.id"})

	// 2. Uji JWT Revocation (kripto.cabutJWT & verifikasiJWT)
	kunciJWT = "rahasia-jwt-2026-super"
	tokenA = kripto.buatJWT({"user": "ahmad"}, kunciJWT)
	cekSebelumCabut = kripto.verifikasiJWT(tokenA, kunciJWT)
	kripto.cabutJWT(tokenA)
	cekSetelahCabut = kripto.verifikasiJWT(tokenA, kunciJWT)

	// 3. Uji Prefiks Versi Rute (web.versi)
	apiV1 = web.versi("v1")
	apiV1.dapatkan("/profil", fungsi(k) {
		k.teks("profil v1")
	})
	`

	env := runKuCodeString(code)

	// Beri jeda sangat singkat agar goroutine antrean selesai memproses
	time.Sleep(50 * time.Millisecond)

	tpVal, _ := env.Get("terproses")
	if tpVal == nil || tpVal.Inspect() != "pengguna@domain.id" {
		t.Fatalf("Antrean tugas asinkron gagal memproses payload: %v", tpVal)
	}

	cekSebelum, _ := env.Get("cekSebelumCabut")
	if cekSebelum == evaluator.NIL {
		t.Fatalf("Verifikasi JWT sebelum dicabut harus valid!")
	}

	cekSetelah, _ := env.Get("cekSetelahCabut")
	if cekSetelah != evaluator.NIL {
		t.Fatalf("Verifikasi JWT setelah dicabut harus NIL (ditolak), dapat: %v", cekSetelah)
	}
}

func TestDesktopReactiveState(t *testing.T) {
	code := `
	perubahanTerdeteksi = ""
	jendela.padaPerubahanState(fungsi(kunci, nilai) {
		perubahanTerdeteksi = kunci + ":" + teks(nilai)
	})

	jendela.ikatState("skor", 999)
	skorSekarang = jendela.ambilState("skor")
	`
	env := runKuCodeString(code)

	skorVal, _ := env.Get("skorSekarang")
	if intVal, ok := skorVal.(*evaluator.Integer); !ok || intVal.Value != 999 {
		t.Fatalf("jendela.ambilState gagal, dapat: %v", skorVal)
	}

	ptVal, _ := env.Get("perubahanTerdeteksi")
	if ptVal == nil || ptVal.Inspect() != "skor:999" {
		t.Fatalf("Listener padaPerubahanState gagal menangkap event: %v", ptVal)
	}
}

func TestHTTPClientAndHealthCheck(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"pesan":"berhasil"}`))
	}))
	defer ts.Close()

	code := `
	klien = jaringan.buatKlien({"batasWaktu": "5s", "ikutiPengalihan": benar})
	respons = klien.dapatkan("` + ts.URL + `")

	web.kesehatan("/status-sistem")
	`
	env := runKuCodeString(code)

	respVal, _ := env.Get("respons")
	rMap, ok := respVal.(*evaluator.Map)
	if !ok {
		t.Fatalf("Respons klien bukan Map: %v", respVal)
	}
	if rMap.Pairs["sukses"] != evaluator.TRUE {
		t.Fatalf("Panggilan klien.dapatkan gagal: %v", rMap.Pairs["badan"])
	}
}

func TestSwaggerDocAndTarGz(t *testing.T) {
	// 1. Uji OpenAPI & Swagger UI
	codeWeb := `
	web.dokumentasi("/api-docs", {
		"judul": "API Toko Indonesia",
		"versi": "2.0.0",
		"deskripsi": "Layanan Penjualan Online"
	})
	kembalikan web._penangan()
	`
	l := lexer.New(codeWeb)
	p := parser.New(l)
	prog := p.ParseProgram()
	envWeb := evaluator.InitGlobalEnvironment()
	evalRes := evaluator.Eval(prog, envWeb)

	handlerObj, ok := evalRes.(*evaluator.NativeHandlerObject)
	if !ok || handlerObj == nil || handlerObj.Handler == nil {
		t.Fatalf("Gagal inisialisasi handler dokumentasi: %v", evalRes)
	}

	// Request HTML Swagger UI
	reqHTML := httptest.NewRequest(http.MethodGet, "/api-docs", nil)
	recHTML := httptest.NewRecorder()
	handlerObj.Handler.ServeHTTP(recHTML, reqHTML)
	if !strings.Contains(recHTML.Body.String(), "SwaggerUIBundle") {
		t.Fatalf("Halaman Swagger UI tidak sesuai: %s", recHTML.Body.String())
	}

	// Request JSON OpenAPI Spec
	reqJSON := httptest.NewRequest(http.MethodGet, "/api-docs/openapi.json", nil)
	recJSON := httptest.NewRecorder()
	handlerObj.Handler.ServeHTTP(recJSON, reqJSON)
	if !strings.Contains(recJSON.Body.String(), "API Toko Indonesia") {
		t.Fatalf("OpenAPI JSON tidak sesuai: %s", recJSON.Body.String())
	}

	// 2. Uji Kompresi dan Ekstraksi TarGz
	tmpDir, err := os.MkdirTemp("", "ku_targz_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcFolder := filepath.Join(tmpDir, "sumber")
	destFolder := filepath.Join(tmpDir, "tujuan")
	_ = os.MkdirAll(srcFolder, 0755)
	_ = os.WriteFile(filepath.Join(srcFolder, "catatan.txt"), []byte("Halo dari Bahasa Ku TarGz!"), 0644)

	tarGzPath := filepath.Join(tmpDir, "arsip.tar.gz")
	srcEscaped := strings.ReplaceAll(srcFolder, "\\", "/")
	destEscaped := strings.ReplaceAll(destFolder, "\\", "/")
	tarEscaped := strings.ReplaceAll(tarGzPath, "\\", "/")

	codeTar := `
	buatSukses = arsip.buatTarGz("` + tarEscaped + `", "` + srcEscaped + `")
	ekstrakSukses = arsip.ekstrakTarGz("` + tarEscaped + `", "` + destEscaped + `")
	`
	envTar := runKuCodeString(codeTar)

	bVal, _ := envTar.Get("buatSukses")
	if bVal != evaluator.TRUE {
		t.Fatalf("arsip.buatTarGz gagal: %v", bVal)
	}

	eVal, _ := envTar.Get("ekstrakSukses")
	if eVal != evaluator.TRUE {
		t.Fatalf("arsip.ekstrakTarGz gagal: %v", eVal)
	}

	extractedFile := filepath.Join(destFolder, "catatan.txt")
	content, err := os.ReadFile(extractedFile)
	if err != nil || string(content) != "Halo dari Bahasa Ku TarGz!" {
		t.Fatalf("Hasil ekstraksi berkas tar.gz tidak sesuai: %s, galat: %v", string(content), err)
	}
}

func TestRedisMockProtocol(t *testing.T) {
	// Buat TCP Mock Server sederhana untuk protokol RESP Redis
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("Gagal membuka socket mock redis")
	}
	defer ln.Close()

	addr := ln.Addr().String()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				reader := bufio.NewReader(c)
				for {
					line, _, err := reader.ReadLine()
					if err != nil {
						return
					}
					lineStr := string(line)
					if strings.HasPrefix(lineStr, "*") {
						// RESP array command, read until command name
						argCount, _ := strconv.Atoi(strings.TrimPrefix(lineStr, "*"))
						var parts []string
						for i := 0; i < argCount; i++ {
							_, _, _ = reader.ReadLine() // $<len>
							argLine, _, _ := reader.ReadLine()
							parts = append(parts, string(argLine))
						}
						if len(parts) > 0 {
							cmd := strings.ToUpper(parts[0])
							if cmd == "SET" {
								c.Write([]byte("+OK\r\n"))
							} else if cmd == "GET" {
								val := "nilai_redis_uji"
								c.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)))
							} else {
								c.Write([]byte("+OK\r\n"))
							}
						}
					}
				}
			}(conn)
		}
	}()

	code := `
	klien = singgahan.redis("` + addr + `")
	simpanOk = klien.taruh("kunci_uji", "nilai_redis_uji", "10s")
	nilaiAmbil = klien.ambil("kunci_uji")
	`
	env := runKuCodeString(code)

	simpanVal, _ := env.Get("simpanOk")
	if simpanVal != evaluator.TRUE {
		t.Fatalf("klien.taruh redis gagal: %v", simpanVal)
	}

	ambilVal, _ := env.Get("nilaiAmbil")
	if ambilVal == nil || ambilVal.Inspect() != "nilai_redis_uji" {
		t.Fatalf("klien.ambil redis salah: %v", ambilVal)
	}
}

func TestBinaryRPCAndTelemetry(t *testing.T) {
	// 1. Uji RPC Binary Server & Client
	codeRPC := `
	server = jaringan.buatRPCServer("127.0.0.1:9095")
	server.daftarkan("Matematika", {
		"tambah": fungsi(a, b) {
			kembalikan a + b
		},
		"kali": fungsi(a, b) {
			kembalikan a * b
		}
	})
	server.dengarkan()

	klien = jaringan.buatRPCKlien("127.0.0.1:9095")
	hasilTambah = klien.panggil("Matematika.tambah", 15, 35)
	hasilKali = klien.panggil("Matematika.kali", 6, 7)

	// 2. Uji Distributed Tracing Telemetri
	rentangInduk = telemetri.mulaiRentang("proses_pesanan")
	rentangInduk.tambahAtribut("id_pengguna", "user_123")

	rentangAnak = telemetri.mulaiRentang("kueri_db", rentangInduk)
	rentangAnak.tambahAtribut("tabel", "pesanan")
	rentangAnak.selesai()

	rentangInduk.selesai()

	attrInduk = rentangInduk.ambilAtribut("id_pengguna")
	attrAnak = rentangAnak.ambilAtribut("tabel")
	samaTraceID = rentangInduk["traceID"] == rentangAnak["traceID"]
	`

	env := runKuCodeString(codeRPC)

	// Beri jeda sangat singkat agar listener TCP aktif
	time.Sleep(50 * time.Millisecond)

	htVal, _ := env.Get("hasilTambah")
	if intVal, ok := htVal.(*evaluator.Integer); !ok || intVal.Value != 50 {
		t.Fatalf("RPC Matematika.tambah gagal: %v", htVal)
	}

	hkVal, _ := env.Get("hasilKali")
	if intVal, ok := hkVal.(*evaluator.Integer); !ok || intVal.Value != 42 {
		t.Fatalf("RPC Matematika.kali gagal: %v", hkVal)
	}

	attrIVal, _ := env.Get("attrInduk")
	if attrIVal == nil || attrIVal.Inspect() != "user_123" {
		t.Fatalf("telemetri atribut induk salah: %v", attrIVal)
	}

	attrAVal, _ := env.Get("attrAnak")
	if attrAVal == nil || attrAVal.Inspect() != "pesanan" {
		t.Fatalf("telemetri atribut anak salah: %v", attrAVal)
	}

	samaVal, _ := env.Get("samaTraceID")
	if samaVal != evaluator.TRUE {
		t.Fatalf("telemetri traceID harus diwarisi oleh rentang anak!")
	}
}

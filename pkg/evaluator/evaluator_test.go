package evaluator_test

import (
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

func runKuCode(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	env := evaluator.InitGlobalEnvironment()
	return evaluator.Eval(prog, env)
}

func TestEvaluasiDasar(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{
			input: `
paket utama
fungsi utama() {
    var a = 10
    var b = 20
    kembalikan a + b
}
`,
			expected: 30,
		},
		{
			input: `
paket utama
fungsi faktorial(n) {
    jika n <= 1 {
        kembalikan 1
    }
    kembalikan n * faktorial(n - 1)
}
fungsi utama() {
    kembalikan faktorial(5)
}
`,
			expected: 120,
		},
		{
			input: `
paket utama
fungsi utama() {
    var arr = [10, 20]
    arr = tambah(arr, 30)
    kembalikan arr[0] + arr[2] + panjang(arr)
}
`,
			expected: 43,
		},
	}

	for _, tt := range tests {
		val := runKuCode(tt.input)
		intVal, ok := val.(*evaluator.Integer)
		if !ok {
			t.Fatalf("Nilai kembalian bukan Integer: %v", val)
		}
		if intVal.Value != tt.expected {
			t.Errorf("Diharapkan %d, tetapi didapat %d", tt.expected, intVal.Value)
		}
	}
}

func TestTryCatchFinally(t *testing.T) {
	input := `
paket utama
fungsi utama() {
    var jejak = 0
    coba {
        jejak = jejak + 10
        panik("terjadi galat")
        jejak = jejak + 999
    } tangkap (e) {
        jejak = jejak + 20
    } akhirnya {
        jejak = jejak + 5
    }
    kembalikan jejak
}
`
	val := runKuCode(input)
	intVal, ok := val.(*evaluator.Integer)
	if !ok {
		t.Fatalf("Nilai bukan Integer: %v", val)
	}
	if intVal.Value != 35 {
		t.Errorf("Diharapkan 35 (10+20+5), didapat %d", intVal.Value)
	}
}

func TestLoopBreakLanjut(t *testing.T) {
	input := `
paket utama
fungsi utama() {
    var total = 0
    untuk var i = 0; i < 10; i = i + 1 {
        jika i == 2 {
            lanjut
        }
        jika i == 5 {
            berhenti
        }
        total = total + i
    }
    kembalikan total
}
`
	val := runKuCode(input)
	intVal, ok := val.(*evaluator.Integer)
	if !ok {
		t.Fatalf("Nilai bukan Integer: %v", val)
	}
	// 0 + 1 + 3 + 4 = 8 (2 di-skip dengan lanjut, 5 berhenti)
	if intVal.Value != 8 {
		t.Errorf("Diharapkan 8, didapat %d", intVal.Value)
	}
}

func TestJsonDanValidasi(t *testing.T) {
	input := `
paket utama
fungsi utama() {
    var teks = "{\"nama\": \"Budi\", \"umur\": 25}"
    jika !json.valid(teks) {
        kembalikan 0
    }
    var obj = json.urai(teks)
    jika obj["nama"] != "Budi" {
        kembalikan 1
    }
    jika !validasi.apakahEmail("budi@example.com") {
        kembalikan 2
    }
    jika validasi.apakahEmail("bukan-email") {
        kembalikan 3
    }
    jika !validasi.apakahNumerik("123456") {
        kembalikan 4
    }
    jika !validasi.apakahAlfabet("HaloSemua") {
        kembalikan 5
    }
    kembalikan 100
}
`
	val := runKuCode(input)
	intVal, ok := val.(*evaluator.Integer)
	if !ok {
		t.Fatalf("Nilai bukan Integer: %v", val)
	}
	if intVal.Value != 100 {
		t.Errorf("Diharapkan 100, didapat %d", intVal.Value)
	}
}

func TestSistemKunciDanAngka(t *testing.T) {
	input := `
paket utama
fungsi utama() {
    var k = sistem.kunci()
    k.kunci()
    var bisa = k.cobaKunci()
    jika bisa {
        kembalikan 1
    }
    k.buka()

    // Uji modul angka bitwise & konversi
    var binerStr = angka.biner(10) // "1010"
    jika binerStr != "1010" {
        kembalikan 2
    }
    var heksaStr = angka.heksa(255) // "ff"
    jika heksaStr != "ff" {
        kembalikan 3
    }
    var geser = angka.geserKiri(1, 4) // 16
    jika geser != 16 {
        kembalikan 4
    }
    var dan = angka.dan(7, 3) // 3
    jika dan != 3 {
        kembalikan 5
    }
    var warna = terminal.merah("Bahasa Ku")
    jika warna == "" {
        kembalikan 6
    }

    kembalikan 200
}
`
	val := runKuCode(input)
	intVal, ok := val.(*evaluator.Integer)
	if !ok {
		t.Fatalf("Nilai bukan Integer: %v", val)
	}
	if intVal.Value != 200 {
		t.Errorf("Diharapkan 200, didapat %d", intVal.Value)
	}
}

func TestStrukturDataDanSoket(t *testing.T) {
	input := `
paket utama
fungsi utama() {
    // 1. Uji Tumpukan (Stack)
    var s = strukturData.tumpukan()
    s.dorong("A")
    s.dorong("B")
    s.dorong("C")
    jika s.panjang() != 3 {
        kembalikan 1
    }
    var atas = s.tarik() // "C"
    jika atas != "C" {
        kembalikan 2
    }
    jika s.intip() != "B" {
        kembalikan 3
    }

    // 2. Uji Antrean (Queue)
    var q = strukturData.antrean()
    q.masuk(10)
    q.masuk(20)
    q.masuk(30)
    jika q.depan() != 10 {
        kembalikan 4
    }
    var keluar = q.keluar() // 10
    jika keluar != 10 {
        kembalikan 5
    }
    jika q.panjang() != 2 {
        kembalikan 6
    }

    // 3. Uji Sistem Flags / OS Helper
    jika sistem.apakahWindows() != benar && sistem.apakahWindows() != salah {
        kembalikan 7
    }

    kembalikan 300
}
`
	val := runKuCode(input)
	intVal, ok := val.(*evaluator.Integer)
	if !ok {
		t.Fatalf("Nilai bukan Integer: %v", val)
	}
	if intVal.Value != 300 {
		t.Errorf("Diharapkan 300, didapat %d", intVal.Value)
	}
}

func TestJWTDanMuatEnv(t *testing.T) {
	input := `
paket utama
fungsi utama() {
    // 1. Uji JWT
    var payload = {"sub": "user123", "peran": "admin"}
    var secret = "kunci_rahasia_super_aman"
    var token = kripto.buatJWT(payload, secret, 3600)
    jika token == "" {
        kembalikan 1
    }

    var terverifikasi = kripto.verifikasiJWT(token, secret)
    jika terverifikasi == kosong {
        kembalikan 2
    }
    jika terverifikasi["sub"] != "user123" {
        kembalikan 3
    }

    // Uji salah secret
    var palsu = kripto.verifikasiJWT(token, "secret_salah")
    jika palsu != kosong {
        kembalikan 4
    }

    // 2. Uji muatEnv & bacaBaris
    berkas.tulis("test_env.tmp", "APP_MODE=development\nAPP_PORT=3000\n")
    sistem.muatEnv("test_env.tmp")
    jika sistem.lingkungan("APP_MODE") != "development" {
        kembalikan 5
    }

    var baris = berkas.bacaBaris("test_env.tmp")
    jika panjang(baris) != 2 {
        kembalikan 6
    }
    berkas.hapus("test_env.tmp")

    kembalikan 400
}
`
	val := runKuCode(input)
	intVal, ok := val.(*evaluator.Integer)
	if !ok {
		t.Fatalf("Nilai bukan Integer: %v", val)
	}
	if intVal.Value != 400 {
		t.Errorf("Diharapkan 400, didapat %d", intVal.Value)
	}
}

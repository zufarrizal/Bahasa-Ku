package evaluator_test

import (
	"testing"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
)

func TestFiturLanjutanGo(t *testing.T) {
	code := `
paket utama

tipe Makhluk struktur {
    kategori teks
}

fungsi (m Makhluk) Suara() {
    kembalikan "..."
}

tipe Kucing struktur {
    identitas Makhluk
    nama teks
}

fungsi (k Kucing) Meong() {
    kembalikan "Meong dari " + k.nama
}

fungsi utama() {
    // 1. Uji Struct Embedding & Method/Field Promotion
    var k = Kucing({
        "nama": "Milo",
        "identitas": Makhluk({"kategori": "Mamalia"})
    })
    jika k.kategori != "Mamalia" {
        kembalikan 1
    }
    jika k.Suara() != "..." {
        kembalikan 2
    }
    jika k.Meong() != "Meong dari Milo" {
        kembalikan 3
    }

    // 2. Uji kanal.pilih (select statement multiplexing)
    var ch1 = kanal.buat(1)
    var ch2 = kanal.buat(1)
    ch1.kirim("pesan 1")

    var hasilPilih = ""
    kanal.pilih({
        "kanal": ch1,
        "aksi": fungsi(data) {
            hasilPilih = data
        }
    }, {
        "kanal": ch2,
        "aksi": fungsi(data) {
            hasilPilih = "dari ch2"
        }
    })
    jika hasilPilih != "pesan 1" {
        kembalikan 4
    }

    // 3. Uji Non-blocking Default pada kanal.pilih
    var defaultTerpanggil = salah
    kanal.pilih({
        "kanal": ch2,
        "aksi": fungsi(data) {
            defaultTerpanggil = salah
        }
    }, {
        "bawaan": fungsi() {
            defaultTerpanggil = benar
        }
    })
    jika defaultTerpanggil != benar {
        kembalikan 5
    }

    // 4. Uji atomik lock-free concurrency
    var hitungan = 10
    atomik.tambah(hitungan, 5) // +5
    jika hitungan != 15 {
        kembalikan 6
    }
    var berhasilTukar = atomik.tukar(hitungan, 15, 100)
    jika berhasilTukar != benar || hitungan != 100 {
        kembalikan 7
    }

    // 5. Uji konteks batas waktu (context with cancellation & timeout)
    var ctx, batal = konteks.denganBatasWaktu(50)
    jika ctx.apakahBatal() != salah {
        kembalikan 8
    }
    batal()
    jika ctx.apakahBatal() != benar {
        kembalikan 9
    }

    // 6. Uji sistem.kunciBacaTulis (RWMutex)
    var rw = sistem.kunciBacaTulis()
    rw.kunciBaca()
    rw.bukaBaca()
    rw.kunci()
    rw.buka()

    // 7. Uji konversi.nilaiAwal (Zero values)
    jika konversi.nilaiAwal("angka") != 0 || konversi.nilaiAwal("teks") != "" || konversi.nilaiAwal("boolean") != salah {
        kembalikan 10
    }

    // 8. Uji konversi.apakahStruktur & konversi.apakahTipe
    jika konversi.apakahStruktur(k) != benar || konversi.apakahTipe(k, "Kucing") != benar {
        kembalikan 11
    }

    kembalikan 600
}
`
	val := runKuCode(code)
	intVal, ok := val.(*evaluator.Integer)
	if !ok {
		t.Fatalf("Nilai bukan Integer: %v", val)
	}
	if intVal.Value != 600 {
		t.Errorf("Diharapkan 600, didapat %d", intVal.Value)
	}
}

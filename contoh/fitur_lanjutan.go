package main

import "fmt"

type Pengguna struct {
  nama string
  umur int
}

func sapaHari(hari interface{}) interface{} {
  switch hari {
    case "Senin":
      return "Semangat awal pekan!"
    case "Jumat":
      return "Hampir akhir pekan!"
    case "Minggu":
      return "Waktu istirahat."
    default:
      return "Hari kerja biasa."
  }
}

func main() interface{} {
  fmt.Println("=== Uji Coba Fitur Lanjutan Bahasa Ku ===")
  var buah = []interface{}{"Apel", "Mangga", "Jeruk"}
  fmt.Println("Daftar buah awal:", buah)
  fmt.Println("Jumlah buah:", len(buah))
  fmt.Println("Buah pertama:", buah[0])
  buah = append(buah, "Pisang")
  fmt.Println("Setelah ditambah:", buah)
  var ibukota = map[string]interface{}{"Indonesia": "Nusantara", "Jepang": "Tokyo"}
  fmt.Println("Ibukota Indonesia:", ibukota["Indonesia"])
  var programmer = Pengguna(map[string]interface{}{"nama": "Budi Santoso", "umur": 25})
  fmt.Println("Data Pengguna:", programmer.nama, "- Umur:", programmer.umur)
  fmt.Println("Pesan hari Jumat:", sapaHari("Jumat"))
  fmt.Println("Pesan hari Rabu:", sapaHari("Rabu"))
}

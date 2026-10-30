package main

import "fmt"

func ujiTunda() interface{} {
  fmt.Println("1. Fungsi ujiTunda dimulai")
  defer fmt.Println("4. Pesan tunda ke-1 (dieksekusi terakhir)")
  defer fmt.Println("3. Pesan tunda ke-2 (LIFO: dieksekusi sebelum tunda ke-1)")
  fmt.Println("2. Fungsi ujiTunda selesai dieksekusi")
}

func main() interface{} {
  fmt.Println("=== Uji Fitur Tunda (Defer) ===")
  ujiTunda()
}

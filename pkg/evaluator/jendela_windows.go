//go:build windows

package evaluator

import (
	"syscall"
	"unsafe"
)

func tampilkanPesanNative(judul, isi string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	msgBox := user32.NewProc("MessageBoxW")
	tPtr, _ := syscall.UTF16PtrFromString(judul)
	cPtr, _ := syscall.UTF16PtrFromString(isi)
	msgBox.Call(0, uintptr(unsafe.Pointer(cPtr)), uintptr(unsafe.Pointer(tPtr)), 0)
}

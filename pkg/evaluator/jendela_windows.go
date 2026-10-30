//go:build windows

package evaluator

import (
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

func tampilkanPesanNative(judul, isi, tipe string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	msgBox := user32.NewProc("MessageBoxW")
	tPtr, _ := syscall.UTF16PtrFromString(judul)
	cPtr, _ := syscall.UTF16PtrFromString(isi)

	var uType uint32 = 0x00000000 // MB_OK
	switch strings.ToLower(tipe) {
	case "peringatan":
		uType |= 0x00000030 // MB_ICONWARNING
	case "galat":
		uType |= 0x00000010 // MB_ICONERROR
	case "tanya":
		uType |= 0x00000020 // MB_ICONQUESTION
	default:
		uType |= 0x00000040 // MB_ICONINFORMATION
	}

	msgBox.Call(0, uintptr(unsafe.Pointer(cPtr)), uintptr(unsafe.Pointer(tPtr)), uintptr(uType))
}

func dialogPilihBerkasNative(judul, filter string) string {
	psScript := `Add-Type -AssemblyName System.Windows.Forms; $d = New-Object System.Windows.Forms.OpenFileDialog; `
	if judul != "" {
		psScript += `$d.Title = '` + strings.ReplaceAll(judul, "'", "''") + `'; `
	}
	if filter != "" {
		psScript += `$d.Filter = '` + strings.ReplaceAll(filter, "'", "''") + `'; `
	}
	psScript += `if ($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { Write-Output $d.FileName }`

	cmd := exec.Command("powershell", "-NoProfile", "-Command", psScript)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func dialogPilihFolderNative(judul string) string {
	psScript := `Add-Type -AssemblyName System.Windows.Forms; $d = New-Object System.Windows.Forms.FolderBrowserDialog; `
	if judul != "" {
		psScript += `$d.Description = '` + strings.ReplaceAll(judul, "'", "''") + `'; `
	}
	psScript += `if ($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { Write-Output $d.SelectedPath }`

	cmd := exec.Command("powershell", "-NoProfile", "-Command", psScript)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func salinPapanKlipNative(teks string) bool {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "$input | Set-Clipboard")
	cmd.Stdin = strings.NewReader(teks)
	return cmd.Run() == nil
}

func bacaPapanKlipNative() string {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "Get-Clipboard")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(out), "\r\n")
}

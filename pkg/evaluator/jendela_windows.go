//go:build windows

package evaluator

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2"
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

func tampilkanKonfirmasiNative(judul, isi string) bool {
	user32 := syscall.NewLazyDLL("user32.dll")
	msgBox := user32.NewProc("MessageBoxW")
	tPtr, _ := syscall.UTF16PtrFromString(judul)
	cPtr, _ := syscall.UTF16PtrFromString(isi)

	// MB_YESNO (0x00000004) | MB_ICONQUESTION (0x00000020)
	var uType uint32 = 0x00000004 | 0x00000020
	ret, _, _ := msgBox.Call(0, uintptr(unsafe.Pointer(cPtr)), uintptr(unsafe.Pointer(tPtr)), uintptr(uType))
	// IDYES = 6
	return ret == 6
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

func bukaJendelaNative(judul, konten, targetUrl string, lebar, tinggi int, startIPC func() int, invokeIPC func(fungsi string, argumen []interface{}) (interface{}, error), onCleanup func()) {
	// Direktori profil WebView2 terisolasi agar cache lama tidak mengunci session
	userDataDir := filepath.Join(os.TempDir(), "ku_webview_data")
	_ = os.MkdirAll(userDataDir, 0755)

	// 1. Inisialisasi native Microsoft Edge WebView2 (Win32 Native Window)
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		DataPath:  userDataDir,
		WindowOptions: webview2.WindowOptions{
			Title:  judul,
			Width:  uint(lebar),
			Height: uint(tinggi),
			IconId: 0,
			Center: true,
		},
	})

	if w != nil {
		defer w.Destroy()

		port := startIPC()

		// Binding Native Langsung via webview2.Bind (Zero HTTP Port Dependency)
		// JavaScript memanggil: window.__ku_native_call(fungsi, argumenJSON)
		_ = w.Bind("__ku_native_call", func(fungsi string, argumenJSON string) (string, error) {
			var args []interface{}
			if argumenJSON != "" {
				_ = json.Unmarshal([]byte(argumenJSON), &args)
			}
			res, err := invokeIPC(fungsi, args)
			if err != nil {
				respBytes, _ := json.Marshal(map[string]interface{}{"galat": TerjemahkanGalat(err)})
				return string(respBytes), nil
			}
			respBytes, _ := json.Marshal(map[string]interface{}{"hasil": res})
			return string(respBytes), nil
		})

		// Jembatan IPC Universal: Utamakan Native Binding, Fallback ke HTTP Loopback
		bridgeScript := fmt.Sprintf(`
window.ku = {
  panggil: async function(fungsi, ...argumen) {
    if (window.__ku_native_call) {
      try {
        const raw = await window.__ku_native_call(fungsi, JSON.stringify(argumen));
        const data = typeof raw === 'string' ? JSON.parse(raw) : raw;
        if (data && data.galat) throw new Error(data.galat);
        return data ? data.hasil : null;
      } catch (err) {
        if (!err.message || !err.message.includes("is not a function")) {
          throw err;
        }
      }
    }
    // Fallback via HTTP loopback
    const res = await fetch("http://127.0.0.1:%d/ku_ipc", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ fungsi: fungsi, argumen: argumen })
    });
    const data = await res.json();
    if (data.galat) throw new Error(data.galat);
    return data.hasil;
  },
  tutup: function() {
    try { navigator.sendBeacon("http://127.0.0.1:%d/ku_tutup"); } catch(e) {}
  }
};
window.addEventListener("beforeunload", function() {
  if (window.ku && window.ku.tutup) window.ku.tutup();
});
`, port, port)

		w.Init(bridgeScript)

		if targetUrl != "" {
			w.Navigate(targetUrl)
		} else {
			// Periksa apakah konten sudah merupakan dokumen HTML utuh
			trimKonten := strings.TrimSpace(konten)
			isFullDoc := strings.HasPrefix(strings.ToLower(trimKonten), "<!doctype") ||
				strings.HasPrefix(strings.ToLower(trimKonten), "<html")

			fullHtml := konten
			if !isFullDoc {
				fullHtml = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>%s</title>
<style>
  * { box-sizing: border-box; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    padding: 24px;
    background: #f8fafc;
    color: #1e293b;
    margin: 0;
  }
  h1 { color: #0284c7; }
</style>
</head>
<body>
%s
</body>
</html>`, judul, konten)
			}
			w.SetHtml(fullHtml)
		}

		// Jalankan loop event UI (memblokir hingga jendela ditutup pengguna)
		w.Run()

		// Setelah jendela ditutup, jalankan pembersihan basis data & resource
		if onCleanup != nil {
			onCleanup()
		}
		time.Sleep(50 * time.Millisecond)
		os.Exit(0)
		return
	}

	// 2. Fallback: Browser App Mode jika runtime WebView2 belum terpasang
	browserCandidates := []string{
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("LocalAppData"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(os.Getenv("LocalAppData"), "Google", "Chrome", "Application", "chrome.exe"),
	}

	appUrl := targetUrl
	if appUrl == "" {
		tempDir := os.TempDir()
		htmlFile := filepath.Join(tempDir, "ku_app.html")
		port := startIPC()
		bridge := fmt.Sprintf(`<script>
window.ku = {
  panggil: async function(fungsi, ...argumen) {
    const res = await fetch("http://127.0.0.1:%d/ku_ipc", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ fungsi: fungsi, argumen: argumen })
    });
    const data = await res.json();
    if (data.galat) throw new Error(data.galat);
    return data.hasil;
  },
  tutup: function() {
    try { navigator.sendBeacon("http://127.0.0.1:%d/ku_tutup"); } catch(e) {}
  }
};
window.addEventListener("beforeunload", function() { window.ku.tutup(); });
</script>`, port, port)
		fullHtml := konten
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(konten)), "<!doctype") && !strings.HasPrefix(strings.ToLower(strings.TrimSpace(konten)), "<html") {
			fullHtml = fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>%s</title>%s</head><body>%s</body></html>`, judul, bridge, konten)
		} else {
			// Sisipkan bridge ke dalam head dokumen yang sudah ada
			if idx := strings.Index(strings.ToLower(fullHtml), "<head>"); idx != -1 {
				fullHtml = fullHtml[:idx+6] + bridge + fullHtml[idx+6:]
			} else {
				fullHtml = bridge + fullHtml
			}
		}
		_ = os.WriteFile(htmlFile, []byte(fullHtml), 0644)
		appUrl = htmlFile
	}

	for _, bPath := range browserCandidates {
		if _, err := os.Stat(bPath); err == nil {
			args := []string{
				fmt.Sprintf("--app=%s", appUrl),
				fmt.Sprintf("--window-size=%d,%d", lebar, tinggi),
				fmt.Sprintf("--user-data-dir=%s", userDataDir),
				"--no-first-run",
				"--no-default-browser-check",
			}
			cmd := exec.Command(bPath, args...)
			if err := cmd.Start(); err == nil {
				_ = cmd.Wait()
				os.Exit(0)
				return
			}
		}
	}

	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", appUrl).Start()
}


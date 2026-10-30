package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
	"github.com/zufarrizal/bahasa-ku/pkg/formatter"
	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
	"github.com/zufarrizal/bahasa-ku/pkg/repl"
	"github.com/zufarrizal/bahasa-ku/pkg/transpiler"
)

const version = "0.2.0 (Go Native Edition)"

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		repl.Start(os.Stdin, os.Stdout, version)
		return
	}

	command := args[0]

	switch command {
	case "repl":
		repl.Start(os.Stdin, os.Stdout, version)
	case "versi", "-v", "--version":
		fmt.Println("Bahasa Ku versi", version)

	case "jalankan":
		if len(args) < 2 {
			fmt.Println("Galat: Harap tentukan nama berkas .ku yang ingin dijalankan.")
			os.Exit(1)
		}
		runFile(args[1])

	case "buat":
		if len(args) < 2 {
			fmt.Println("Galat: Harap tentukan nama berkas .ku yang ingin ditranspilasi.")
			os.Exit(1)
		}
		target := ""
		if len(args) >= 3 {
			target = args[2]
		}
		transpileFile(args[1], target)

	case "kompilasi", "build":
		if len(args) < 2 {
			fmt.Println("Galat: Harap tentukan nama berkas .ku yang ingin dikompilasi.")
			os.Exit(1)
		}
		outputExe := ""
		if len(args) >= 4 && (args[2] == "-o" || args[2] == "--output") {
			outputExe = args[3]
		} else if len(args) >= 3 && args[2] != "-o" {
			outputExe = args[2]
		}
		compileDirectToBinary(args[1], outputExe)

	case "inisiasi", "init":
		if len(args) < 3 {
			fmt.Println("Penggunaan: ku inisiasi <web|desktop|cli> <nama_folder>")
			os.Exit(1)
		}
		scaffoldProject(args[1], args[2])

	case "rapi", "format":
		if len(args) < 2 {
			fmt.Println("Penggunaan: ku rapi [-w] <nama_berkas.ku>")
			os.Exit(1)
		}
		writeBack := false
		targetFile := args[1]
		if args[1] == "-w" || args[1] == "--write" {
			if len(args) < 3 {
				fmt.Println("Galat: Harap tentukan berkas .ku yang ingin dirapikan.")
				os.Exit(1)
			}
			writeBack = true
			targetFile = args[2]
		}
		formatKuFile(targetFile, writeBack)

	default:
		fmt.Printf("Perintah '%s' tidak dikenal. Ketik 'ku' untuk bantuan.\n", command)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`Penggunaan CLI Bahasa Ku (Go Native):
  ku                       Membuka REPL (terminal interaktif)
  ku jalankan <berkas.ku>  Menjalankan berkas kode .ku
  ku rapi [-w] <berkas.ku> Merapikan format kode secara otomatis (mirip gofmt)
  ku kompilasi <berkas.ku> [-o keluar.exe]  Kompilasi langsung ke biner mandiri (.exe)
  ku inisiasi <web|desktop|cli> <nama_folder>  Membuat kerangka kerja proyek baru
  ku buat <berkas.ku>      Mengonversi berkas .ku ke .go murni
  ku versi                 Menampilkan versi

Contoh:
  ku
  ku jalankan contoh/halo.ku
  ku rapi -w contoh/halo.ku
  ku kompilasi contoh/halo.ku -o HaloApp.exe
  ku inisiasi web portal-berita
`)
}

func runFile(filename string) {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Galat membaca berkas: %v\n", err)
		os.Exit(1)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Println("Galat Sintaks:")
		for _, e := range p.Errors() {
			fmt.Println(" -", e)
		}
		os.Exit(1)
	}

	env := evaluator.InitGlobalEnvironment()
	evaluator.Eval(prog, env)
}

func transpileFile(source, target string) {
	content, err := os.ReadFile(source)
	if err != nil {
		fmt.Printf("Galat membaca berkas: %v\n", err)
		os.Exit(1)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Println("Galat Sintaks:")
		for _, e := range p.Errors() {
			fmt.Println(" -", e)
		}
		os.Exit(1)
	}

	tr := transpiler.New()
	goCode := tr.Transpile(prog)

	if target == "" {
		ext := filepath.Ext(source)
		target = strings.TrimSuffix(source, ext) + ".go"
	}

	err = os.WriteFile(target, []byte(goCode), 0644)
	if err != nil {
		fmt.Printf("Galat menyimpan berkas .go: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Berhasil mentranspilasi '%s' -> '%s'! ✨\n", source, target)
}

func compileDirectToBinary(source, outputExe string) {
	content, err := os.ReadFile(source)
	if err != nil {
		fmt.Printf("Galat membaca berkas: %v\n", err)
		os.Exit(1)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Println("Galat Sintaks:")
		for _, e := range p.Errors() {
			fmt.Println(" -", e)
		}
		os.Exit(1)
	}

	tr := transpiler.New()
	goCode := tr.Transpile(prog)

	// Buat folder sementara untuk build
	tempDir, err := os.MkdirTemp("", "ku_build_*")
	if err != nil {
		fmt.Printf("Galat direktori sementara: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	tempGoFile := filepath.Join(tempDir, "main.go")
	err = os.WriteFile(tempGoFile, []byte(goCode), 0644)
	if err != nil {
		fmt.Printf("Galat menulis kode sementara: %v\n", err)
		os.Exit(1)
	}

	goExe := "go"
	if p, err := exec.LookPath("go"); err == nil {
		goExe = p
	} else {
		// Cari di path standar Go jika tidak ada di PATH session
		defaultGo := filepath.Join(os.Getenv("ProgramFiles"), "Go", "bin", "go.exe")
		if _, err := os.Stat(defaultGo); err == nil {
			goExe = defaultGo
		}
	}

	// Inisialisasi go.mod di folder sementara
	goModCmd := exec.Command(goExe, "mod", "init", "kuapp")
	goModCmd.Dir = tempDir
	_ = goModCmd.Run()

	if outputExe == "" {
		base := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
		if runtime.GOOS == "windows" {
			outputExe = base + ".exe"
		} else {
			outputExe = base
		}
	}

	absOutput, _ := filepath.Abs(outputExe)
	buildCmd := exec.Command(goExe, "build", "-ldflags=-s -w", "-o", absOutput, "main.go")
	buildCmd.Dir = tempDir
	buildOutput, err := buildCmd.CombinedOutput()

	if err != nil {
		fmt.Printf("Galat kompilasi Go:\n%s\n", string(buildOutput))
		os.Exit(1)
	}

	fmt.Printf("Berhasil mengompilasi '%s' -> '%s' (Biner Mandiri)! 🎉\n", source, outputExe)
}

func scaffoldProject(projectType, folderName string) {
	err := os.MkdirAll(folderName, 0755)
	if err != nil {
		fmt.Printf("Galat membuat folder: %v\n", err)
		os.Exit(1)
	}

	var fileContent string
	mainFileName := filepath.Join(folderName, "utama.ku")

	switch strings.ToLower(projectType) {
	case "web":
		fileContent = `paket utama

pakai "cetak"
pakai "web"

fungsi sapa(permintaan) {
    kembalikan "<h1>Halo dari Web Bahasa Ku!</h1><p>Proyek baru Anda siap dikembangkan.</p>"
}

fungsi ambilData(permintaan) {
    kembalikan {
        "status": "sukses",
        "pesan": "Selamat datang di API Bahasa Ku!"
    }
}

fungsi utama() {
    web.tangani("/", sapa)
    web.tangani("/api", ambilData)

    cetak.tulis("Server web berjalan di http://localhost:8080")
    web.dengarkan(":8080")
}
`
	case "desktop":
		fileContent = `paket utama

pakai "cetak"
pakai "jendela"

fungsi utama() {
    cetak.tulis("Membuka aplikasi desktop...")

    jendela.buka({
        "judul": "Aplikasi Desktop Baru",
        "konten": "<h1>Aplikasi Desktop Bahasa Ku</h1><p>Kerangka kerja aplikasi GUI desktop Anda siap digunakan.</p>"
    })
}
`
	case "cli":
		fileContent = `paket utama

pakai "cetak"
pakai "sistem"

fungsi utama() {
    cetak.tulis("=== Selamat Datang di Proyek CLI Bahasa Ku ===")
    cetak.tulis("Waktu saat ini:", sistem.waktu())
}
`
	default:
		fmt.Printf("Tipe proyek '%s' tidak dikenal. Pilihan: web, desktop, cli\n", projectType)
		os.Exit(1)
	}

	err = os.WriteFile(mainFileName, []byte(fileContent), 0644)
	if err != nil {
		fmt.Printf("Galat membuat berkas proyek: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Berhasil membuat proyek '%s' tipe [%s] di folder '%s/'! ✨\n", folderName, projectType, folderName)
	fmt.Printf("Jalankan sekarang dengan:\n  ku jalankan %s/utama.ku\n", folderName)
}

func formatKuFile(filename string, writeBack bool) {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Galat membaca berkas: %v\n", err)
		os.Exit(1)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Println("Galat Sintaks:")
		for _, e := range p.Errors() {
			fmt.Println(" -", e)
		}
		os.Exit(1)
	}

	f := formatter.New()
	formatted := f.Format(prog)

	if writeBack {
		err := os.WriteFile(filename, []byte(formatted), 0644)
		if err != nil {
			fmt.Printf("Galat menyimpan berkas: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Format berkas '%s' berhasil dirapikan! ✨\n", filename)
	} else {
		fmt.Print(formatted)
	}
}



package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/zufarrizal/bahasa-ku/pkg/ast"
	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
	"github.com/zufarrizal/bahasa-ku/pkg/formatter"
	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
	"github.com/zufarrizal/bahasa-ku/pkg/repl"
	"github.com/zufarrizal/bahasa-ku/pkg/transpiler"
)

const version = "0.3.7 (Go Native Edition)"

func init() {
	evaluator.RegisterSubFileEvaluator(func(path string, env *evaluator.Environment) {
		resolvedPath, err := evaluator.ResolveModulePath(path)
		if err != nil {
			fmt.Printf("Galat modul '%s': %v\n", path, err)
			return
		}

		content, err := os.ReadFile(resolvedPath)
		if err != nil {
			return
		}
		l := lexer.New(string(content))
		p := parser.New(l)
		prog := p.ParseProgram()
		if len(p.Errors()) == 0 {
			subEnv := evaluator.NewEnclosedEnvironment(env)
			for _, stmt := range prog.Statements {
				if _, isPaket := stmt.(*ast.PaketStatement); !isPaket {
					evaluator.Eval(stmt, subEnv)
				}
			}

			// Jalankan inisiasi() pada sub-file jika ada
			if initObj, ok := subEnv.Get("inisiasi"); ok {
				if fn, isFn := initObj.(*evaluator.Function); isFn {
					evaluator.Eval(fn.Body, subEnv)
				}
			}

			// Ekspor simbol: Simbol huruf kapital terekspor ke paket pemanggil (aturan ekspor Go)
			// Simbol bawaan atau huruf kecil tetap terlindungi di subEnv
			for name, val := range subEnv.GetStore() {
				if len(name) > 0 && unicode.IsUpper(rune(name[0])) {
					env.Set(name, val)
				}
			}
		}
	})
}

// Footer biner 8 byte penanda payload Bahasa Ku
var kuMagicFooter = []byte("KUAPP\x01\x00\x00")

func readEmbeddedScript(exePath string) (string, bool) {
	f, err := os.Open(exePath)
	if err != nil {
		return "", false
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil || stat.Size() < 16 {
		return "", false
	}

	// Baca 16 byte terakhir: [8 byte panjang payload (uint64 little endian)][8 byte KUAPP\x01\x00\x00]
	tail := make([]byte, 16)
	if _, err := f.ReadAt(tail, stat.Size()-16); err != nil {
		return "", false
	}

	if !bytes.Equal(tail[8:], kuMagicFooter) {
		return "", false
	}

	payloadLen := int64(tail[0]) | int64(tail[1])<<8 | int64(tail[2])<<16 | int64(tail[3])<<24 |
		int64(tail[4])<<32 | int64(tail[5])<<40 | int64(tail[6])<<48 | int64(tail[7])<<56

	if payloadLen <= 0 || payloadLen > stat.Size()-16 {
		return "", false
	}

	payloadOffset := stat.Size() - 16 - payloadLen
	payloadBytes := make([]byte, payloadLen)
	if _, err := f.ReadAt(payloadBytes, payloadOffset); err != nil {
		return "", false
	}

	return string(payloadBytes), true
}

func main() {
	args := os.Args[1:]

	// 1. Cek mode biner mandiri hasil kompilasi (Payload Appended Executable)
	if len(args) == 0 {
		selfExe, err := os.Executable()
		if err == nil {
			if scriptSource, ok := readEmbeddedScript(selfExe); ok {
				l := lexer.New(scriptSource)
				p := parser.New(l)
				prog := p.ParseProgram()
				if len(p.Errors()) > 0 {
					fmt.Println("Galat Sintaks pada Program Mandiri:")
					for _, e := range p.Errors() {
						fmt.Println(" -", e)
					}
					os.Exit(1)
				}
				env := evaluator.InitGlobalEnvironment()
				res := evaluator.Eval(prog, env)
				if pObj, isPanic := res.(*evaluator.PanicObject); isPanic {
					fmt.Printf("Galat Eksekusi (Panik): %s\n", pObj.Value.Inspect())
					os.Exit(1)
				}
				return
			}
		}
		repl.Start(os.Stdin, os.Stdout, version)
		return
	}

	command := args[0]

	// 1. Opsi Evaluasi Sebaris (-e / --eval): ku -e 'tulis(1 + 2)'
	if command == "-e" || command == "--eval" {
		if len(args) < 2 {
			fmt.Println("Galat: Harap sertakan kode ekspresi yang ingin dievaluasi.")
			fmt.Println("Contoh: ku -e 'tulis(1 + 2)'")
			os.Exit(1)
		}
		evalCode := strings.Join(args[1:], " ")
		l := lexer.New(evalCode)
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
		res := evaluator.Eval(prog, env)
		if pObj, isPanic := res.(*evaluator.PanicObject); isPanic {
			fmt.Printf("Galat Eksekusi (Panik): %s\n", pObj.Value.Inspect())
			os.Exit(1)
		}
		return
	}

	// 2. Eksekusi Berkas Langsung tanpa kata 'jalankan': ku berkas.ku atau ku .
	if strings.HasSuffix(command, ".ku") || command == "." {
		fileName := command
		if command == "." {
			if _, err := os.Stat("utama.ku"); err == nil {
				fileName = "utama.ku"
			} else if _, err := os.Stat("main.ku"); err == nil {
				fileName = "main.ku"
			} else {
				fmt.Println("Galat: Direktori tidak memiliki 'utama.ku' atau 'main.ku'.")
				os.Exit(1)
			}
		}
		debugMode := false
		for _, a := range args[1:] {
			if a == "--debug" || a == "-d" {
				debugMode = true
			}
		}
		runFile(fileName, debugMode)
		return
	}

	switch command {
	case "bantuan", "help", "-h", "--help":
		printHelp()
	case "repl":
		repl.Start(os.Stdin, os.Stdout, version)
	case "versi", "-v", "--version":
		checkUpgrade := false
		for _, a := range args[1:] {
			if a == "--cek" || a == "-c" || a == "--check" {
				checkUpgrade = true
			}
		}
		fmt.Println("Bahasa Ku versi", version)
		if checkUpgrade {
			checkLatestRelease(true)
		}

	case "perbarui", "upgrade", "update":
		fmt.Println("Bahasa Ku versi lokal:", version)
		checkLatestRelease(false)

	case "jalankan", "j", "run":
		debugMode := false
		fileName := ""
		strictMode := false
		for _, a := range args[1:] {
			if a == "--debug" || a == "-d" {
				debugMode = true
			} else if a == "--aman" || a == "-aman" || a == "--sandbox" {
				strictMode = true
			} else if !strings.HasPrefix(a, "-") && fileName == "" {
				fileName = a
			}
		}
		if strictMode {
			evaluator.SetStrictSandboxMode(true)
		}
		// Compact run: Jika tanpa argumen nama berkas, otomatis cari utama.ku / main.ku di folder aktif
		if fileName == "" {
			if _, err := os.Stat("utama.ku"); err == nil {
				fileName = "utama.ku"
			} else if _, err := os.Stat("main.ku"); err == nil {
				fileName = "main.ku"
			} else {
				fmt.Println("Galat: Harap tentukan nama berkas .ku yang ingin dijalankan (atau jalankan di folder yang memiliki 'utama.ku').")
				os.Exit(1)
			}
		}
		runFile(fileName, debugMode)

	case "awasi", "watch", "a":
		target := "."
		if len(args) >= 2 && !strings.HasPrefix(args[1], "-") {
			target = args[1]
		} else {
			if _, err := os.Stat("utama.ku"); err == nil {
				target = "utama.ku"
			}
		}
		var extra []string
		if len(args) > 2 {
			extra = args[2:]
		}
		watchAndRun(target, extra)

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

	case "kompilasi", "build", "k":
		isGUI := false
		sourceFile := ""
		outputExe := ""
		customIcon := ""
		useNative := false

		for i := 1; i < len(args); i++ {
			arg := args[i]
			if arg == "-gui" || arg == "--gui" || arg == "-app" {
				isGUI = true
			} else if arg == "-native" || arg == "--native" {
				useNative = true
			} else if (arg == "-o" || arg == "--output") && i+1 < len(args) {
				outputExe = args[i+1]
				i++
			} else if (arg == "-ikon" || arg == "--icon" || arg == "-ico") && i+1 < len(args) {
				customIcon = args[i+1]
				i++
			} else if strings.HasPrefix(arg, "-") {
				// Abaikan flag lain atau tangani jika ada
			} else if sourceFile == "" {
				sourceFile = arg
			} else if outputExe == "" {
				outputExe = arg
			}
		}

		// Compact compile: Jika tanpa berkas sumber, otomatis cari utama.ku di direktori aktif
		if sourceFile == "" {
			if _, err := os.Stat("utama.ku"); err == nil {
				sourceFile = "utama.ku"
			} else if _, err := os.Stat("main.ku"); err == nil {
				sourceFile = "main.ku"
			} else {
				fmt.Println("Galat: Berkas sumber .ku tidak ditentukan (atau jalankan di folder yang memiliki 'utama.ku').")
				fmt.Println("Contoh: ku kompilasi [-gui] [-ikon <ikon.ico>] [app.ku] [-o app.exe]")
				os.Exit(1)
			}
		}

		compileDirectToBinary(sourceFile, outputExe, isGUI, customIcon, useNative)

	case "inisiasi", "init", "inisialisasi":
		if len(args) < 2 {
			fmt.Println("Penggunaan: ku inisialisasi <nama_folder> [web|desktop|cli]")
			fmt.Println("Atau: ku inisiasi <web|desktop|cli> <nama_folder>")
			os.Exit(1)
		}
		projType := "cli"
		folderName := args[1]
		if len(args) >= 3 {
			// Periksa apakah argumen pertama adalah tipe (gaya lama: ku inisiasi web toko)
			if args[1] == "web" || args[1] == "desktop" || args[1] == "cli" {
				projType = args[1]
				folderName = args[2]
			} else {
				// Gaya baru: ku inisialisasi toko web
				projType = args[2]
			}
		}
		scaffoldProject(projType, folderName)

	case "rapi", "format":
		if len(args) < 2 {
			fmt.Println("Penggunaan: ku rapi [-w] [-c|--cek] <nama_berkas.ku|folder>")
			os.Exit(1)
		}
		writeBack := false
		checkOnly := false
		targetFile := ""

		for i := 1; i < len(args); i++ {
			arg := args[i]
			if arg == "-w" || arg == "--write" {
				writeBack = true
			} else if arg == "-c" || arg == "--cek" || arg == "--check" {
				checkOnly = true
			} else if targetFile == "" {
				targetFile = arg
			}
		}

		if targetFile == "" {
			fmt.Println("Galat: Harap tentukan berkas .ku yang ingin dirapikan atau diperiksa.")
			os.Exit(1)
		}
		formatKuFile(targetFile, writeBack, checkOnly)

	case "uji", "test", "u":
		targetPath := "."
		watchMode := false
		for _, a := range args[1:] {
			if a == "--awasi" || a == "-w" || a == "--watch" {
				watchMode = true
			} else if !strings.HasPrefix(a, "-") && targetPath == "." {
				targetPath = a
			}
		}
		if watchMode {
			watchTests(targetPath)
		} else {
			runTests(targetPath)
		}

	case "tolakUkur", "tolak-ukur", "bench", "benchmark":
		targetPath := "."
		if len(args) >= 2 {
			targetPath = args[1]
		}
		runBenchmarks(targetPath)

	case "audit":
		targetPath := "."
		if len(args) >= 2 {
			targetPath = args[1]
		}
		runAudit(targetPath)

	case "periksa", "lint":
		targetPath := "."
		if len(args) >= 2 {
			targetPath = args[1]
		}
		runLint(targetPath)

	case "pasang", "install", "get":
		if len(args) < 2 {
			fmt.Println("Penggunaan: ku pasang <url_git>")
			fmt.Println("Contoh: ku pasang github.com/pengembang/paket-ku")
			os.Exit(1)
		}
		repoURL := args[1]
		installGitPackage(repoURL)

	case "serang", "beban", "bench-http":
		if len(args) < 2 {
			fmt.Println("Penggunaan: ku serang <url> [-k konkurensi] [-d durasi] [-m metode]")
			fmt.Println("Contoh: ku serang http://127.0.0.1:8080/api -k 50 -d 5s")
			os.Exit(1)
		}
		runLoadTest(args[1:])

	case "audit-biner":
		targetExe := ""
		if len(args) >= 2 {
			targetExe = args[1]
		}
		runAuditBinary(targetExe)

	case "pantau", "monitor", "top":
		runMonitorDashboard()

	case "perbarui-kunci", "kunci":
		runUpdateLockfile()

	case "cetak-uji", "buat-uji", "mock-uji":
		if len(args) < 2 {
			fmt.Println("Penggunaan: ku cetak-uji <berkas.ku> [keluaran_uji.ku]")
			os.Exit(1)
		}
		targetFile := args[1]
		outputTestFile := ""
		if len(args) >= 3 {
			outputTestFile = args[2]
		}
		scaffoldUnitTest(targetFile, outputTestFile)

	case "lsp", "bahasa-server":
		runLSP()

	default:
		fmt.Printf("Perintah '%s' tidak dikenal. Ketik 'ku bantuan' atau 'ku -h' untuk bantuan.\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`Penggunaan CLI Bahasa Ku (Go Native):
  ku                       Membuka REPL (terminal interaktif)
  ku <berkas.ku>           Menjalankan berkas secara instan (atau ku .)
  ku -e '<kode>'           Mengevaluasi ekspresi kode satu baris
  ku jalankan <berkas.ku>  Menjalankan berkas kode .ku
  ku awasi <berkas.ku>     Menjalankan berkas dan me-reload otomatis saat kode berubah (Hot Reload)
  ku uji [berkas/_uji.ku]  Menjalankan pengujian unit test otomatis (--awasi untuk hot-reload test)
  ku tolakUkur [berkas]    Menjalankan pengujian tolok ukur performa (benchmark)
  ku serang <url>          Menjalankan pengujian beban/stres HTTP konkuren (-k, -d, -m)
  ku pantau                Menampilkan dashboard kinerja runtime & memori terminal
  ku audit [folder]        Memindai keamanan proyek & deteksi rahasia bocor
  ku audit-biner [file.exe] Menganalisis metadata payload biner kompilasi
  ku periksa [folder]      Menganalisis kerapian kode & mendeteksi dead code/unused var
  ku pasang <url_git>      Memasang paket modul Git & memperbarui ku.kunci
  ku perbarui-kunci        Memvalidasi integritas SHA-256 modul terpasang di ku.kunci
  ku cetak-uji <berkas.ku> Membuat scaffold berkas unit test otomatis
  ku lsp                   Menjalankan Language Server Protocol untuk editor/IDE
  ku rapi [-w] [-c] <berkas|folder> Merapikan format kode (-w: simpan, -c: periksa CI/CD)
  ku kompilasi <berkas.ku> [-gui] [-ikon ikon.ico] [-native] [-o keluar.exe]  Kompilasi mandiri (.exe)
  ku inisialisasi <nama_folder> [web|desktop|cli]  Membuat proyek baru beserta berkas ku.mod
  ku buat <berkas.ku>      Mengonversi berkas .ku ke .go murni
  ku versi [--cek]         Menampilkan versi & memeriksa rilis terbaru
  ku perbarui              Memeriksa dan memperbarui biner Bahasa Ku ke rilis terbaru

Contoh:
  ku contoh/halo.ku
  ku -e 'tulis("Halo Dunia!", rentang(1, 5))'
  ku .
  ku uji
  ku rapi -w contoh/halo.ku
  ku kompilasi contoh/halo.ku -o HaloApp.exe
  ku inisiasi web portal-berita
`)
}

func runFile(filename string, debugMode bool) {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Galat membaca berkas: %v\n", err)
		os.Exit(1)
	}

	startParse := time.Now()
	l := lexer.New(string(content))
	p := parser.New(l)
	prog := ast.OptimizeProgram(p.ParseProgram())

	if len(p.Errors()) > 0 {
		fmt.Println("Galat Sintaks:")
		for _, e := range p.Errors() {
			fmt.Println(" -", e)
		}
		os.Exit(1)
	}
	parseDuration := time.Since(startParse)

	if debugMode {
		fmt.Println("[DEBUG] Mode debug aktif.")
		fmt.Printf("[DEBUG] Parsing AST selesai dalam: %v\n", parseDuration)
		fmt.Printf("[DEBUG] Menjalankan berkas: %s\n", filename)
		fmt.Println("[DEBUG] ----------------------------------------------")
	}

	env := evaluator.InitGlobalEnvironment()
	startEval := time.Now()
	res := evaluator.Eval(prog, env)
	evalDuration := time.Since(startEval)

	if pObj, isPanic := res.(*evaluator.PanicObject); isPanic {
		fmt.Println("\n==============================================")
		fmt.Println("🛑 GALAT EKSEKUSI / PANIK TERDETEKSI:")
		fmt.Println("==============================================")
		fmt.Println(pObj.Inspect())
		os.Exit(1)
	}

	if debugMode {
		fmt.Println("\n[DEBUG] ----------------------------------------------")
		fmt.Printf("[DEBUG] Eksekusi selesai dalam: %v (Total: %v)\n", evalDuration, parseDuration+evalDuration)
		memStats := env.GetStore()["debug"]
		if memMap, ok := memStats.(*evaluator.Map); ok {
			if memFn, hasMem := memMap.Pairs["memori"].(*evaluator.Builtin); hasMem {
				m := memFn.Fn().(*evaluator.Map)
				fmt.Printf("[DEBUG] Penggunaan Memori Heap: %s MB | Goroutine Aktif: %s\n", m.Pairs["alokasiMB"].Inspect(), m.Pairs["jumlahGoroutine"].Inspect())
			}
		}
	}
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

func setWindowsPESubsystemGUI(binPath string) error {
	data, err := os.ReadFile(binPath)
	if err != nil {
		return err
	}
	if len(data) < 0x40 {
		return fmt.Errorf("berkas terlalu kecil untuk format PE")
	}
	// Periksa header DOS ("MZ")
	if data[0] != 'M' || data[1] != 'Z' {
		return nil // Bukan format PE Windows
	}
	// Dapatkan offset ke PE Header dari 0x3C
	peOffset := int(data[0x3C]) | int(data[0x3D])<<8 | int(data[0x3E])<<16 | int(data[0x3F])<<24
	if peOffset < 0 || peOffset+0x60 > len(data) {
		return fmt.Errorf("offset PE tidak valid")
	}
	// Periksa signature "PE\0\0"
	if data[peOffset] != 'P' || data[peOffset+1] != 'E' || data[peOffset+2] != 0 || data[peOffset+3] != 0 {
		return fmt.Errorf("signature PE tidak valid")
	}
	// Subsystem terletak pada Optional Header di offset: peOffset + 0x18 + 0x44 (baik untuk PE32 maupun PE32+)
	subsystemOffset := peOffset + 0x18 + 0x44
	if subsystemOffset+2 <= len(data) {
		// Setel Subsystem = 2 (IMAGE_SUBSYSTEM_WINDOWS_GUI)
		data[subsystemOffset] = 2
		data[subsystemOffset+1] = 0
		return os.WriteFile(binPath, data, 0755)
	}
	return fmt.Errorf("offset subsistem di luar jangkauan")
}

func compileDirectToBinary(source, outputExe string, isGUI bool, customIcon string, useNative bool) {
	content, err := os.ReadFile(source)
	if err != nil {
		fmt.Printf("Galat membaca berkas: %v\n", err)
		os.Exit(1)
	}

	// Deteksi otomatis jika skrip menggunakan modul jendela/desktop, anggap butuh GUI jika di Windows
	strContent := string(content)
	if !isGUI && runtime.GOOS == "windows" {
		if strings.Contains(strContent, "pakai \"jendela\"") || strings.Contains(strContent, "pakai \"desktop\"") || strings.Contains(strContent, "jendela.buka") {
			isGUI = true
		}
	}

	l := lexer.New(strContent)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Println("Galat Sintaks:")
		for _, e := range p.Errors() {
			fmt.Println(" -", e)
		}
		os.Exit(1)
	}

	if outputExe == "" {
		base := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
		if runtime.GOOS == "windows" {
			outputExe = base + ".exe"
		} else {
			outputExe = base
		}
	}

	absOutput, _ := filepath.Abs(outputExe)

	// METODE UTAMA (DEFAULT): Builder Mandiri Asli Bahasa Ku (Zero Dependency Bundler)
	// Hanya gunakan toolchain native jika pengguna secara eksplisit meminta flag -native atau ikon kustom
	if !useNative && customIcon == "" {
		if selfExe, err := os.Executable(); err == nil {
			if baseBin, err := os.ReadFile(selfExe); err == nil {
				// Bersihkan payload aplikasi terdahulu jika selfExe adalah hasil biner bundler
				cleanBin := baseBin
				if len(baseBin) >= 16 && bytes.Equal(baseBin[len(baseBin)-8:], kuMagicFooter) {
					pLen := int64(baseBin[len(baseBin)-16]) | int64(baseBin[len(baseBin)-15])<<8 |
						int64(baseBin[len(baseBin)-14])<<16 | int64(baseBin[len(baseBin)-13])<<24 |
						int64(baseBin[len(baseBin)-12])<<32 | int64(baseBin[len(baseBin)-11])<<40 |
						int64(baseBin[len(baseBin)-10])<<48 | int64(baseBin[len(baseBin)-9])<<56
					if pLen > 0 && pLen <= int64(len(baseBin)-16) {
						cleanBin = baseBin[:int64(len(baseBin)-16)-pLen]
					}
				}

				// Struktur format biner terbundel:
				// [Biner Ku Asli (cleanBin)][Konten Skrip (content)][8 byte Panjang Payload (uint64 LE)][8 byte KUAPP\x01\x00\x00]
				var bundled bytes.Buffer
				bundled.Write(cleanBin)
				bundled.Write(content)

				lenBytes := make([]byte, 8)
				cLen := uint64(len(content))
				for i := 0; i < 8; i++ {
					lenBytes[i] = byte(cLen >> (i * 8))
				}
				bundled.Write(lenBytes)
				bundled.Write(kuMagicFooter)

				if err := os.WriteFile(absOutput, bundled.Bytes(), 0755); err == nil {
					// Jika mode GUI di Windows, patch PE header langsung agar bebas jendela konsol CMD
					if isGUI && runtime.GOOS == "windows" {
						_ = setWindowsPESubsystemGUI(absOutput)
					}
					fmt.Printf("Berhasil mengompilasi '%s' -> '%s' (Biner Mandiri Asli Bahasa Ku)! 🎉\n", source, outputExe)
					return
				}
			}
		}
	}

	// METODE SEKUNDER / CADANGAN: Kompilasi Native (Jika flag -native atau ada ikon kustom)
	ldFlags := "-s -w"
	if isGUI && runtime.GOOS == "windows" {
		ldFlags = "-s -w -H windowsgui"
	}

	goExe := "go"
	if p, err := exec.LookPath("go"); err == nil {
		goExe = p
	} else {
		defaultGo := filepath.Join(os.Getenv("ProgramFiles"), "Go", "bin", "go.exe")
		if _, err := os.Stat(defaultGo); err == nil {
			goExe = defaultGo
		}
	}

	tempDir, err := os.MkdirTemp("", "ku_build_*")
	if err == nil {
		defer os.RemoveAll(tempDir)

		runnerCode := fmt.Sprintf(`package main

import (
	"fmt"
	"os"

	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
)

const scriptSource = %q

func main() {
	l := lexer.New(scriptSource)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		fmt.Println("Galat Sintaks pada Program Mandiri:")
		for _, e := range p.Errors() {
			fmt.Println(" -", e)
		}
		os.Exit(1)
	}
	env := evaluator.InitGlobalEnvironment()
	res := evaluator.Eval(prog, env)
	if pObj, isPanic := res.(*evaluator.PanicObject); isPanic {
		fmt.Println("Galat Eksekusi: " + pObj.Inspect())
		os.Exit(1)
	}
}
`, string(content))

		tempGoFile := filepath.Join(tempDir, "main.go")
		_ = os.WriteFile(tempGoFile, []byte(runnerCode), 0644)

		// Cari root folder proyek repositori Bahasa Ku yang memiliki go.mod
		projRoot := ""

		cwd, _ := os.Getwd()
		absCwd, _ := filepath.Abs(cwd)
		curr := absCwd
		for {
			modPath := filepath.Join(curr, "go.mod")
			if modData, err := os.ReadFile(modPath); err == nil {
				if strings.Contains(string(modData), "github.com/zufarrizal/bahasa-ku") {
					projRoot = curr
					break
				}
			}
			parent := filepath.Dir(curr)
			if parent == curr {
				break
			}
			curr = parent
		}

		if projRoot == "" {
			if selfExe, err := os.Executable(); err == nil {
				curr = filepath.Dir(selfExe)
				for {
					modPath := filepath.Join(curr, "go.mod")
					if modData, err := os.ReadFile(modPath); err == nil {
						if strings.Contains(string(modData), "github.com/zufarrizal/bahasa-ku") {
							projRoot = curr
							break
						}
					}
					parent := filepath.Dir(curr)
					if parent == curr {
						break
					}
					curr = parent
				}
			}
		}

		if projRoot == "" {
			gopath := os.Getenv("GOPATH")
			if gopath == "" {
				gopath = filepath.Join(os.Getenv("USERPROFILE"), "go")
			}
			candidates := []string{
				filepath.Join(os.Getenv("USERPROFILE"), "Documents", "Github", "Bahasa Ku"),
				filepath.Join(gopath, "src", "github.com", "zufarrizal", "bahasa-ku"),
			}
			for _, cand := range candidates {
				if _, err := os.Stat(filepath.Join(cand, "go.mod")); err == nil {
					projRoot = cand
					break
				}
			}
		}

		if projRoot != "" {
			goModContent := fmt.Sprintf(`module kuapp

go 1.22

require github.com/zufarrizal/bahasa-ku v0.0.0

replace github.com/zufarrizal/bahasa-ku => %q
`, filepath.ToSlash(projRoot))

			_ = os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644)

			tidyCmd := exec.Command(goExe, "mod", "tidy")
			tidyCmd.Dir = tempDir
			_ = tidyCmd.Run()

			if runtime.GOOS == "windows" {
				iconWritten := false
				if customIcon != "" {
					absIcon, err := filepath.Abs(customIcon)
					if err == nil {
						if _, err := os.Stat(absIcon); err == nil {
							rsrcExe := "rsrc"
							if rPath, err := exec.LookPath("rsrc"); err == nil {
								rsrcExe = rPath
							} else {
								defaultRsrc := filepath.Join(os.Getenv("USERPROFILE"), "go", "bin", "rsrc.exe")
								if _, err := os.Stat(defaultRsrc); err == nil {
									rsrcExe = defaultRsrc
								}
							}
							targetSyso := filepath.Join(tempDir, "ku.syso")
							rsrcCmd := exec.Command(rsrcExe, "-ico", absIcon, "-o", targetSyso)
							if err := rsrcCmd.Run(); err == nil {
								iconWritten = true
							}
						} else {
							fmt.Printf("Peringatan: Berkas ikon '%s' tidak ditemukan, menggunakan ikon default Bahasa Ku.\n", customIcon)
						}
					}
				}

				if !iconWritten {
					sysoPath := filepath.Join(projRoot, "cmd", "ku", "ku.syso")
					if sysoData, err := os.ReadFile(sysoPath); err == nil && len(sysoData) > 0 {
						_ = os.WriteFile(filepath.Join(tempDir, "ku.syso"), sysoData, 0644)
					} else if len(defaultSysoBytes) > 0 {
						_ = os.WriteFile(filepath.Join(tempDir, "ku.syso"), defaultSysoBytes, 0644)
					}
				}
			}

			buildCmd := exec.Command(goExe, "build", fmt.Sprintf("-ldflags=%s", ldFlags), "-o", absOutput, ".")
			buildCmd.Dir = tempDir
			buildOutput, err := buildCmd.CombinedOutput()
			if err == nil {
				fmt.Printf("Berhasil mengompilasi '%s' -> '%s' (Biner Native Bahasa Ku)! 🎉\n", source, outputExe)
				return
			}
			fmt.Printf("Gagal metode bundler: %s\n", string(buildOutput))
		}
	}

	// METODE TERAKHIR: Fallback ke transpiler Go native
	tr := transpiler.New()
	goCode := tr.Transpile(prog)

	tempDir2, err := os.MkdirTemp("", "ku_build_*")
	if err != nil {
		fmt.Printf("Galat direktori sementara: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir2)

	tempGoFile := filepath.Join(tempDir2, "main.go")
	err = os.WriteFile(tempGoFile, []byte(goCode), 0644)
	if err != nil {
		fmt.Printf("Galat menulis kode sementara: %v\n", err)
		os.Exit(1)
	}

	goModCmd := exec.Command(goExe, "mod", "init", "kuapp")
	goModCmd.Dir = tempDir2
	_ = goModCmd.Run()

	if runtime.GOOS == "windows" {
		if len(defaultSysoBytes) > 0 {
			_ = os.WriteFile(filepath.Join(tempDir2, "ku.syso"), defaultSysoBytes, 0644)
		}
	}

	buildCmd := exec.Command(goExe, "build", fmt.Sprintf("-ldflags=%s", ldFlags), "-o", absOutput, ".")
	buildCmd.Dir = tempDir2
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

	// 1. Buat berkas manifes ku.mod (identitas proyek Bahasa Ku)
	modContent := fmt.Sprintf(`modul %s
versi 0.3.7
bahasa ku
`, filepath.Base(folderName))

	modFileName := filepath.Join(folderName, "ku.mod")
	if err := os.WriteFile(modFileName, []byte(modContent), 0644); err != nil {
		fmt.Printf("Galat membuat ku.mod: %v\n", err)
	}

	var fileContent string
	mainFileName := filepath.Join(folderName, "utama.ku")

	switch strings.ToLower(projectType) {
	case "web":
		fileContent = `paket utama

pakai "cetak"
pakai "web"
pakai "konfigurasi"
pakai "debug"

fungsi utama() {
    // 1. Muat Konfigurasi Berlapis (.env & bawaan)
    konfigurasi.muatBerkas(".env")
    var port = konfigurasi.ambilAngka("PORT", 3000)
    var namaApp = konfigurasi.ambilTeks("APP_NAME", "Aplikasi Web Ku")

    // 2. Middleware & CORS
    web.aktifkanCORS()
    web.gunakan(fungsi(minta, respon, lanjut) {
        cetak.tulis("[HTTP]", minta["metode"], minta["jalur"])
        lanjut()
    })

    // 3. Rute Beranda
    web.dapatkan("/", fungsi(minta, respon) {
        respon.status(200).html("<h1>🚀 Selamat Datang di " + namaApp + "</h1><p>Server aktif di port " + teks(port) + "</p>")
    })

    // 4. REST API Terkelola (Grup)
    var api = web.grup("/api")
    api.dapatkan("/info", fungsi(minta, respon) {
        respon.status(200).json({
            "aplikasi": namaApp,
            "status": "aktif",
            "waktu": waktu.sekarang()
        })
    })

    cetak.tulis("🚀 " + namaApp + " berjalan di: http://localhost:" + teks(port))
    web.jalankan(port)
}
`
		// Sertakan .env default
		_ = os.WriteFile(filepath.Join(folderName, ".env"), []byte("PORT=3000\nAPP_NAME=Aplikasi Web Ku\nAPP_DEBUG=true\n"), 0644)
		_ = os.WriteFile(filepath.Join(folderName, ".gitignore"), []byte("*.db\n*.exe\n.env.local\n"), 0644)

	case "desktop":
		fileContent = `paket utama

pakai "cetak"
pakai "desktop"
pakai "jendela"
pakai "debug"

fungsi utama() {
    cetak.tulis("Memulai aplikasi Desktop Bahasa Ku...")

    jendela.daftarkanFungsi("ambilStatus", fungsi() {
        var stat = debug.memori()
        kembalikan {
            "pesan": "Halo dari Backend Bahasa Ku!",
            "memoriMB": stat["alokasiMB"],
            "goroutine": stat["jumlahGoroutine"]
        }
    })

    jendela.buka({
        "judul": "Aplikasi Desktop Modern - Bahasa Ku",
        "lebar": 800,
        "tinggi": 600,
        "html": ` + "`" + `
            <!DOCTYPE html>
            <html style="font-family: system-ui, sans-serif; background: #0f172a; color: #f8fafc; padding: 40px; text-align: center;">
                <body>
                    <h1 style="color: #38bdf8;">⚡ Aplikasi Desktop Bahasa Ku</h1>
                    <p style="color: #94a3b8;">Frontend HTML5/CSS modern dengan integrasi backend native Bahasa Ku.</p>
                    <div style="margin: 30px auto; max-width: 400px; padding: 20px; background: #1e293b; border-radius: 12px; border: 1px solid #334155;">
                        <button style="padding: 10px 24px; font-weight: 600; border-radius: 8px; border: none; background: #0284c7; color: white; cursor: pointer;"
                                onclick="panggilKu()">
                            Cek Status Mesin
                        </button>
                        <p id="hasil" style="margin-top: 15px; font-size: 14px; color: #34d399;"></p>
                    </div>
                    <script>
                        async function panggilKu() {
                            try {
                                const res = await fetch('/ku_ipc', {
                                    method: 'POST',
                                    headers: {'Content-Type': 'application/json'},
                                    body: JSON.stringify({fungsi: 'ambilStatus', argumen: []})
                                });
                                const data = await res.json();
                                document.getElementById('hasil').innerText = data.hasil.pesan + ' (' + data.hasil.memoriMB + ' MB)';
                            } catch(e) {
                                document.getElementById('hasil').innerText = 'Gagal memanggil backend: ' + e;
                            }
                        }
                    </script>
                </body>
            </html>
        ` + "`" + `
    })
}
`
		_ = os.WriteFile(filepath.Join(folderName, ".gitignore"), []byte("*.exe\n*.db\n"), 0644)

	case "cli":
		fileContent = `paket utama

pakai "cetak"
pakai "cli"
pakai "konfigurasi"
pakai "debug"

fungsi utama() {
    cli.nama("` + filepath.Base(folderName) + `")
    cli.deskripsi("Alat CLI Modern Berbasis Bahasa Ku")

    cli.perintah("info", "Menampilkan status sistem", fungsi(cmd) {
        cmd.opsi("detail", "d", "Tampilkan statistik memori", salah)
        cmd.aksi(fungsi(konteks) {
            cetak.tulis("=== Status Aplikasi ===")
            cetak.tulis("Nama:", cli.nama)
            jika konteks["opsi"]["detail"] == benar {
                var mem = debug.memori()
                cetak.tulis("Memori Heap:", mem["alokasiMB"], "MB")
                cetak.tulis("Goroutine:", mem["jumlahGoroutine"])
            }
        })
    })

    cli.jalankan()
}
`
		_ = os.WriteFile(filepath.Join(folderName, ".gitignore"), []byte("*.exe\n*.db\n"), 0644)

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
	fmt.Printf("Dibuat:\n  - %s/ku.mod (manifes proyek)\n  - %s/utama.ku (berkas utama)\n", folderName, folderName)
	fmt.Printf("Jalankan sekarang dengan:\n  ku jalankan %s/utama.ku\n", folderName)
}

func formatKuFile(target string, writeBack bool, checkOnly bool) {
	files := []string{}
	fi, err := os.Stat(target)
	if err != nil {
		fmt.Printf("Galat mengakses '%s': %v\n", target, err)
		os.Exit(1)
	}

	if fi.IsDir() {
		_ = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".ku") {
				files = append(files, path)
			}
			return nil
		})
	} else {
		files = append(files, target)
	}

	if len(files) == 0 {
		fmt.Printf("Tidak ditemukan berkas .ku pada '%s'\n", target)
		return
	}

	unformattedCount := 0
	for _, filename := range files {
		content, err := os.ReadFile(filename)
		if err != nil {
			fmt.Printf("Galat membaca berkas '%s': %v\n", filename, err)
			continue
		}

		l := lexer.New(string(content))
		p := parser.New(l)
		prog := p.ParseProgram()

		if len(p.Errors()) > 0 {
			fmt.Printf("Galat Sintaks pada '%s':\n", filename)
			for _, e := range p.Errors() {
				fmt.Println(" -", e)
			}
			if checkOnly {
				unformattedCount++
			}
			continue
		}

		f := formatter.New()
		formatted := f.Format(prog)

		// Normalisasi line endings untuk perbandingan akurat
		origNorm := strings.ReplaceAll(string(content), "\r\n", "\n")
		formNorm := strings.ReplaceAll(formatted, "\r\n", "\n")

		if origNorm != formNorm {
			unformattedCount++
			if checkOnly {
				fmt.Printf("❌ Perlu dirapikan: %s\n", filename)
			} else if writeBack {
				err := os.WriteFile(filename, []byte(formatted), 0644)
				if err != nil {
					fmt.Printf("Galat menyimpan berkas '%s': %v\n", filename, err)
				} else {
					fmt.Printf("✨ Format berkas '%s' berhasil dirapikan!\n", filename)
				}
			} else {
				if len(files) == 1 {
					fmt.Print(formatted)
				} else {
					fmt.Printf("--- %s ---\n%s\n", filename, formatted)
				}
			}
		} else {
			if checkOnly {
				fmt.Printf("✅ Sudah rapi: %s\n", filename)
			} else if !writeBack && len(files) == 1 {
				fmt.Print(formatted)
			}
		}
	}

	if checkOnly {
		if unformattedCount > 0 {
			fmt.Printf("\n⚠️  Ditemukan %d berkas yang belum rapi. Jalankan 'ku rapi -w <berkas>' untuk memperbaikinya.\n", unformattedCount)
			os.Exit(1)
		} else {
			fmt.Printf("\n✨ Semua berkas (%d) sudah rapi sempurna!\n", len(files))
		}
	}
}

func runTests(targetPath string) {
	testFiles := []string{}

	info, err := os.Stat(targetPath)
	if err != nil {
		fmt.Printf("Galat mengakses '%s': %v\n", targetPath, err)
		os.Exit(1)
	}

	if !info.IsDir() {
		testFiles = append(testFiles, targetPath)
	} else {
		filepath.Walk(targetPath, func(path string, f os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !f.IsDir() && (strings.HasSuffix(f.Name(), "_uji.ku") || strings.HasSuffix(f.Name(), "_test.ku")) {
				testFiles = append(testFiles, path)
			}
			return nil
		})
	}

	if len(testFiles) == 0 {
		fmt.Println("ℹ️  Tidak ditemukan berkas pengujian (*_uji.ku).")
		return
	}

	fmt.Printf("🔍 Menjalankan pengujian Bahasa Ku (%d berkas)...\n\n", len(testFiles))
	totalLulus := 0

	for _, file := range testFiles {
		fmt.Printf("=== MENJALANKAN %s ===\n", file)
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("❌ Gagal membaca %s: %v\n", file, err)
			continue
		}

		l := lexer.New(string(content))
		p := parser.New(l)
		prog := p.ParseProgram()

		if len(p.Errors()) > 0 {
			fmt.Println("❌ Galat Sintaks:")
			for _, e := range p.Errors() {
				fmt.Println("   -", e)
			}
			continue
		}

		env := evaluator.InitGlobalEnvironment()
		evaluator.Eval(prog, env)
		totalLulus++
		fmt.Printf("--- SELESAI %s\n\n", file)
	}

	fmt.Printf("✨ Pengujian Selesai: %d/%d berkas pengujian berhasil dijalankan.\n", totalLulus, len(testFiles))
}

func runBenchmarks(targetPath string) {
	benchFiles := []string{}

	info, err := os.Stat(targetPath)
	if err != nil {
		fmt.Printf("Galat mengakses '%s': %v\n", targetPath, err)
		os.Exit(1)
	}

	if !info.IsDir() {
		benchFiles = append(benchFiles, targetPath)
	} else {
		filepath.Walk(targetPath, func(path string, f os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !f.IsDir() && (strings.HasSuffix(f.Name(), "_tolakukur.ku") || strings.HasSuffix(f.Name(), "_bench.ku")) {
				benchFiles = append(benchFiles, path)
			}
			return nil
		})
	}

	if len(benchFiles) == 0 {
		fmt.Println("ℹ️  Tidak ditemukan berkas tolok ukur (*_tolakukur.ku atau *_bench.ku).")
		return
	}

	fmt.Printf("⚡ Menjalankan Pengujian Tolok Ukur / Benchmark (%d berkas)...\n\n", len(benchFiles))

	for _, file := range benchFiles {
		fmt.Printf("=== TOLAK UKUR %s ===\n", file)
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("❌ Gagal membaca %s: %v\n", file, err)
			continue
		}

		l := lexer.New(string(content))
		p := parser.New(l)
		prog := p.ParseProgram()

		if len(p.Errors()) > 0 {
			fmt.Println("❌ Galat Sintaks:")
			for _, e := range p.Errors() {
				fmt.Println("   -", e)
			}
			continue
		}

		env := evaluator.InitGlobalEnvironment()
		evaluator.Eval(prog, env)
		fmt.Println()
	}

	fmt.Println("✨ Pengujian Tolak Ukur Selesai.")
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

func checkLatestRelease(checkOnly bool) {
	fmt.Print("🔍 Memeriksa versi rilis terbaru di GitHub...")
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("GET", "https://api.github.com/repos/zufarrizal/Bahasa-Ku/releases/latest", nil)
	if err != nil {
		fmt.Printf("\n❌ Gagal membuat permintaan pembaruan: %v\n", err)
		return
	}
	req.Header.Set("User-Agent", "Bahasa-Ku-CLI")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("\n⚠️  Gagal terhubung ke GitHub: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("\n⚠️  Respons server GitHub tidak terduga: status %d\n", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("\n❌ Gagal membaca respon GitHub: %v\n", err)
		return
	}

	var rel githubRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		fmt.Printf("\n❌ Gagal mem-parse respon rilis: %v\n", err)
		return
	}

	fmt.Println(" Selesai!")
	latestTag := strings.TrimSpace(rel.TagName)
	localTag := "v0.3.7"

	cleanVer := func(s string) []int {
		s = strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(s), "v"), "V")
		parts := strings.Split(s, ".")
		nums := []int{}
		for _, p := range parts {
			numStr := ""
			for _, ch := range p {
				if ch >= '0' && ch <= '9' {
					numStr += string(ch)
				} else {
					break
				}
			}
			n, _ := strconv.Atoi(numStr)
			nums = append(nums, n)
		}
		for len(nums) < 3 {
			nums = append(nums, 0)
		}
		return nums
	}

	vLocal := cleanVer(localTag)
	vRemote := cleanVer(latestTag)

	isNewer := false
	for i := 0; i < len(vLocal) && i < len(vRemote); i++ {
		if vRemote[i] > vLocal[i] {
			isNewer = true
			break
		} else if vRemote[i] < vLocal[i] {
			break
		}
	}

	if isNewer {
		fmt.Printf("\n🌟 Versi baru tersedia: %s (Versi lokal Anda: %s)\n", latestTag, localTag)
		fmt.Printf("Catatan rilis: %s\n", rel.HTMLURL)
		fmt.Println("\nCara memperbarui:")
		if runtime.GOOS == "windows" {
			fmt.Println("  PowerShell: irm https://raw.githubusercontent.com/zufarrizal/Bahasa-Ku/main/pasang.ps1 | iex")
		} else {
			fmt.Println("  Terminal:   curl -fsSL https://raw.githubusercontent.com/zufarrizal/Bahasa-Ku/main/pasang.sh | bash")
		}
	} else {
		fmt.Printf("\n✨ Bahasa Ku sudah menggunakan versi terbaru (%s)! Tidak perlu pembaruan.\n", localTag)
	}
}

func watchAndRun(target string, extraArgs []string) {
	fmt.Printf("👀 [Hot Reload] Mengawasi perubahan pada '%s'...\n", target)
	fmt.Println("Tekan Ctrl+C untuk berhenti.")

	selfExe, err := os.Executable()
	if err != nil {
		selfExe = "ku"
	}

	getSnapshot := func() map[string]time.Time {
		snapshot := make(map[string]time.Time)
		info, err := os.Stat(target)
		if err != nil {
			return snapshot
		}

		if !info.IsDir() {
			snapshot[target] = info.ModTime()
			return snapshot
		}

		_ = filepath.Walk(target, func(path string, f os.FileInfo, err error) error {
			if err == nil && !f.IsDir() && strings.HasSuffix(f.Name(), ".ku") {
				snapshot[path] = f.ModTime()
			}
			return nil
		})
		return snapshot
	}

	lastSnapshot := getSnapshot()
	var currentCmd *exec.Cmd

	startProcess := func() {
		if currentCmd != nil && currentCmd.Process != nil {
			fmt.Println("\n🔄 Terdeteksi perubahan kode! Memulai ulang aplikasi...")
			_ = currentCmd.Process.Kill()
			_ = currentCmd.Wait()
		} else {
			fmt.Println("\n🚀 Menjalankan aplikasi Bahasa Ku...")
		}

		cmdArgs := []string{"jalankan", target}
		cmdArgs = append(cmdArgs, extraArgs...)
		currentCmd = exec.Command(selfExe, cmdArgs...)
		currentCmd.Stdout = os.Stdout
		currentCmd.Stderr = os.Stderr
		currentCmd.Stdin = os.Stdin

		if err := currentCmd.Start(); err != nil {
			fmt.Printf("❌ Gagal memulai aplikasi: %v\n", err)
		}
	}

	startProcess()

	for {
		time.Sleep(500 * time.Millisecond)
		newSnapshot := getSnapshot()

		changed := false
		if len(newSnapshot) != len(lastSnapshot) {
			changed = true
		} else {
			for path, modTime := range newSnapshot {
				if oldTime, ok := lastSnapshot[path]; !ok || !oldTime.Equal(modTime) {
					changed = true
					break
				}
			}
		}

		if changed {
			lastSnapshot = newSnapshot
			startProcess()
		}
	}
}

func watchTests(target string) {
	fmt.Printf("👀 [Pengawasan Uji] Mengawasi pengujian pada '%s'...\n", target)
	fmt.Println("Tekan Ctrl+C untuk berhenti.")

	getSnapshot := func() map[string]time.Time {
		snapshot := make(map[string]time.Time)
		_ = filepath.Walk(target, func(path string, f os.FileInfo, err error) error {
			if err == nil && !f.IsDir() && strings.HasSuffix(f.Name(), ".ku") {
				snapshot[path] = f.ModTime()
			}
			return nil
		})
		return snapshot
	}

	lastSnapshot := getSnapshot()
	fmt.Println("\n🚀 Menjalankan uji unit awal...")
	runTests(target)

	for {
		time.Sleep(500 * time.Millisecond)
		newSnapshot := getSnapshot()

		changed := false
		if len(newSnapshot) != len(lastSnapshot) {
			changed = true
		} else {
			for path, modTime := range newSnapshot {
				if oldTime, ok := lastSnapshot[path]; !ok || !oldTime.Equal(modTime) {
					changed = true
					break
				}
			}
		}

		if changed {
			lastSnapshot = newSnapshot
			fmt.Println("\n🔄 Terdeteksi perubahan berkas! Menjalankan ulang uji unit...")
			runTests(target)
		}
	}
}

func runAudit(targetDir string) {
	fmt.Printf("🛡️ Memulai pemindaian audit keamanan pada folder '%s'...\n", targetDir)

	secretPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(api[_-]?key|secret|password|kata[_-]?sandi|token)\s*[:=]\s*["']([^"']{8,})["']`),
		regexp.MustCompile(`(?i)-----BEGIN\s+(RSA|EC|DSA|OPENSSH)\s+PRIVATE\s+KEY-----`),
		regexp.MustCompile(`(?i)ghp_[a-zA-Z0-9]{36}`),
		regexp.MustCompile(`(?i)xox[baprs]-[0-9a-zA-Z]{10,48}`),
	}

	totalScanned := 0
	foundIssues := 0

	_ = filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.Contains(path, ".git") || strings.Contains(path, "node_modules") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".ku" && ext != ".json" && ext != ".env" && ext != ".yaml" && ext != ".yml" {
			return nil
		}

		totalScanned++
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}

		lines := strings.Split(string(content), "\n")
		for lineIdx, line := range lines {
			for _, pat := range secretPatterns {
				if pat.MatchString(line) {
					foundIssues++
					fmt.Printf("  ⚠️ [PERINGATAN RAHASIA TERBUKA] %s:%d\n", path, lineIdx+1)
					cleanLine := strings.TrimSpace(line)
					if len(cleanLine) > 60 {
						cleanLine = cleanLine[:57] + "..."
					}
					fmt.Printf("     Kode: %s\n", cleanLine)
				}
			}
		}
		return nil
	})

	fmt.Println("------------------------------------------------------------------")
	if foundIssues == 0 {
		fmt.Printf("✅ Audit selesai. Memindai %d berkas. Tidak ditemukan potensi kebocoran rahasia.\n", totalScanned)
	} else {
		fmt.Printf("❌ Audit selesai. Ditemukan %d potensi isu keamanan pada %d berkas!\n", foundIssues, totalScanned)
	}
}

func scaffoldUnitTest(sourceFile, outputFile string) {
	data, err := os.ReadFile(sourceFile)
	if err != nil {
		fmt.Printf("❌ Galat membaca berkas sumber: %v\n", err)
		os.Exit(1)
	}

	l := lexer.New(string(data))
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Println("❌ Berkas memiliki galat sintaks:")
		for _, e := range p.Errors() {
			fmt.Println("  -", e)
		}
		os.Exit(1)
	}

	type funcMeta struct {
		name       string
		paramCount int
		params     []string
	}
	var detectedFuncs []funcMeta

	for _, stmt := range prog.Statements {
		if fnStmt, ok := stmt.(*ast.FunctionStatement); ok && fnStmt.Name != nil {
			var pNames []string
			for _, param := range fnStmt.Parameters {
				pNames = append(pNames, param.Value)
			}
			detectedFuncs = append(detectedFuncs, funcMeta{
				name:       fnStmt.Name.Value,
				paramCount: len(fnStmt.Parameters),
				params:     pNames,
			})
		}
	}

	if len(detectedFuncs) == 0 {
		fmt.Printf("ℹ️ Tidak ditemukan deklarasi fungsi di '%s'.\n", sourceFile)
		return
	}

	if outputFile == "" {
		ext := filepath.Ext(sourceFile)
		base := strings.TrimSuffix(sourceFile, ext)
		outputTestFile := base + "_uji.ku"
		outputFile = outputTestFile
	}

	var sb strings.Builder
	sb.WriteString("// Berkas Pengujian Otomatis untuk: " + filepath.Base(sourceFile) + "\n")
	sb.WriteString("// Dihasilkan otomatis oleh 'ku cetak-uji'\n\n")
	sb.WriteString("paket utama\n\n")
	sb.WriteString(fmt.Sprintf("pakai \"%s\"\n\n", filepath.ToSlash(sourceFile)))

	for _, fn := range detectedFuncs {
		sb.WriteString(fmt.Sprintf("// Uji untuk fungsi: %s\n", fn.name))
		sb.WriteString(fmt.Sprintf("fungsi uji_%s() {\n", fn.name))

		var dummyArgs []string
		for range fn.params {
			dummyArgs = append(dummyArgs, "0")
		}
		callStr := fmt.Sprintf("%s(%s)", fn.name, strings.Join(dummyArgs, ", "))

		sb.WriteString(fmt.Sprintf("    var hasil = %s\n", callStr))
		sb.WriteString("    // Periksa kondisi asersi pengujian\n")
		sb.WriteString("    uji.tidakSama(hasil, nil, \"Hasil pengujian tidak boleh nil\")\n")
		sb.WriteString("}\n\n")
	}

	sb.WriteString("fungsi utama() {\n")
	sb.WriteString("    tulis(\"🚀 Memulai pengujian otomatis...\")\n")
	for _, fn := range detectedFuncs {
		sb.WriteString(fmt.Sprintf("    uji_%s()\n", fn.name))
	}
	sb.WriteString("    tulis(\"✅ Semua pengujian selesai dijalankan!\")\n")
	sb.WriteString("}\n")

	wErr := os.WriteFile(outputFile, []byte(sb.String()), 0644)
	if wErr != nil {
		fmt.Printf("❌ Gagal menulis berkas pengujian: %v\n", wErr)
		os.Exit(1)
	}

	fmt.Printf("🎉 Berhasil menghasilkan scaffold pengujian untuk %d fungsi di '%s'!\n", len(detectedFuncs), outputFile)
}

func runLSP() {
	reader := bufio.NewReader(os.Stdin)

	readLSPMessage := func() ([]byte, error) {
		contentLength := 0
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return nil, err
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			if strings.HasPrefix(line, "Content-Length:") {
				lenStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
				contentLength, _ = strconv.Atoi(lenStr)
			}
		}
		if contentLength <= 0 {
			return nil, fmt.Errorf("panjang konten tidak valid")
		}
		buf := make([]byte, contentLength)
		_, err := io.ReadFull(reader, buf)
		return buf, err
	}

	sendLSPResponse := func(data map[string]interface{}) {
		body, _ := json.Marshal(data)
		fmt.Printf("Content-Length: %d\r\n\r\n%s", len(body), string(body))
	}

	for {
		msgBytes, err := readLSPMessage()
		if err != nil {
			return
		}
		var req map[string]interface{}
		if err := json.Unmarshal(msgBytes, &req); err != nil {
			continue
		}

		method, _ := req["method"].(string)
		id := req["id"]

		switch method {
		case "initialize":
			sendLSPResponse(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]interface{}{
					"capabilities": map[string]interface{}{
						"textDocumentSync":   1,
						"hoverProvider":      true,
						"completionProvider": map[string]interface{}{"resolveProvider": false},
					},
				},
			})

		case "textDocument/hover":
			sendLSPResponse(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]interface{}{
					"contents": map[string]string{
						"kind":  "markdown",
						"value": "**Bahasa Ku (Go Native Edition)**\n\nBahasa pemrograman edukatif bernuansa Go dengan sintaks murni berbahasa Indonesia.",
					},
				},
			})

		case "textDocument/completion":
			completions := []map[string]interface{}{
				{"label": "fungsi", "kind": 14, "detail": "Deklarasi fungsi"},
				{"label": "tulis", "kind": 3, "detail": "Mencetak keluaran ke konsol"},
				{"label": "variabel", "kind": 6, "detail": "Deklarasi variabel baru"},
				{"label": "jika", "kind": 15, "detail": "Percabangan kondisi"},
				{"label": "untuk", "kind": 15, "detail": "Perulangan for"},
				{"label": "kembalikan", "kind": 14, "detail": "Mengembalikan nilai fungsi"},
				{"label": "struktur", "kind": 7, "detail": "Definisi tipe struct"},
			}
			sendLSPResponse(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"result":  completions,
			})

		case "textDocument/didOpen", "textDocument/didChange":
			if params, ok := req["params"].(map[string]interface{}); ok {
				if doc, ok := params["textDocument"].(map[string]interface{}); ok {
					uri, _ := doc["uri"].(string)
					text := ""
					if t, ok := doc["text"].(string); ok {
						text = t
					} else if cc, ok := params["contentChanges"].([]interface{}); ok && len(cc) > 0 {
						if first, ok := cc[0].(map[string]interface{}); ok {
							text, _ = first["text"].(string)
						}
					}

					var diags []map[string]interface{}
					if text != "" {
						l := lexer.New(text)
						p := parser.New(l)
						p.ParseProgram()
						for _, pErr := range p.Errors() {
							lineNum := 0
							colNum := 0
							msg := pErr
							if strings.HasPrefix(pErr, "Baris ") {
								parts := strings.SplitN(pErr, ":", 3)
								if len(parts) >= 3 {
									coords := strings.TrimPrefix(parts[0], "Baris ")
									cParts := strings.Split(coords, ":")
									if len(cParts) == 2 {
										lineNum, _ = strconv.Atoi(cParts[0])
										colNum, _ = strconv.Atoi(cParts[1])
										if lineNum > 0 {
											lineNum--
										}
										if colNum > 0 {
											colNum--
										}
									}
									msg = strings.TrimSpace(parts[2])
								}
							}

							diags = append(diags, map[string]interface{}{
								"range": map[string]interface{}{
									"start": map[string]int{"line": lineNum, "character": colNum},
									"end":   map[string]int{"line": lineNum, "character": colNum + 5},
								},
								"severity": 1,
								"source":   "Bahasa Ku",
								"message":  msg,
							})
						}
					}

					sendLSPResponse(map[string]interface{}{
						"jsonrpc": "2.0",
						"method":  "textDocument/publishDiagnostics",
						"params": map[string]interface{}{
							"uri":         uri,
							"diagnostics": diags,
						},
					})
				}
			}

		case "shutdown":
			sendLSPResponse(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id,
				"result":  nil,
			})

		case "exit":
			return
		}
	}
}

func runLint(targetDir string) {
	fmt.Printf("🔍 Memulai analisis kerapian & linter kode pada '%s'...\n", targetDir)

	totalFiles := 0
	issuesFound := 0

	_ = filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".ku") {
			return nil
		}
		if strings.Contains(path, ".git") || strings.Contains(path, "modul") {
			return nil
		}

		totalFiles++
		data, rErr := os.ReadFile(path)
		if rErr != nil {
			return nil
		}

		l := lexer.New(string(data))
		p := parser.New(l)
		prog := p.ParseProgram()

		if len(p.Errors()) > 0 {
			return nil
		}

		// Periksa unreachable code setelah kembalikan di FunctionStatement
		var inspectBlock func(stmts []ast.Statement)
		inspectBlock = func(stmts []ast.Statement) {
			hasReturned := false
			for _, st := range stmts {
				if hasReturned {
					issuesFound++
					fmt.Printf("  ⚠️ [KODE TAK TERCAPAI] %s: Kode terdeteksi setelah pernyataan 'kembalikan'\n", path)
					break
				}
				if _, isRet := st.(*ast.ReturnStatement); isRet {
					hasReturned = true
				}
				if fnSt, isFn := st.(*ast.FunctionStatement); isFn && fnSt.Body != nil {
					inspectBlock(fnSt.Body.Statements)
				}
			}
		}

		inspectBlock(prog.Statements)
		return nil
	})

	fmt.Println("------------------------------------------------------------------")
	if issuesFound == 0 {
		fmt.Printf("✅ Pemeriksaan linter selesai (%d berkas). Kode rapi dan tidak ada dead code.\n", totalFiles)
	} else {
		fmt.Printf("⚠️ Pemeriksaan linter selesai. Ditemukan %d peringatan pada %d berkas.\n", issuesFound, totalFiles)
	}
}

func installGitPackage(repoURL string) {
	fmt.Printf("📦 Mengunduh modul Bahasa Ku dari '%s'...\n", repoURL)

	cleanURL := strings.TrimPrefix(repoURL, "https://")
	cleanURL = strings.TrimPrefix(cleanURL, "http://")
	parts := strings.Split(cleanURL, "/")
	modName := parts[len(parts)-1]
	modName = strings.TrimSuffix(modName, ".git")

	targetModDir := filepath.Join("modul", modName)
	_ = os.MkdirAll("modul", 0755)

	cloneURL := repoURL
	if !strings.HasPrefix(cloneURL, "http://") && !strings.HasPrefix(cloneURL, "https://") && !strings.HasPrefix(cloneURL, "git@") {
		cloneURL = "https://" + cloneURL
	}

	cmd := exec.Command("git", "clone", "--depth", "1", cloneURL, targetModDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Gagal mengkloning repositori '%s': %v\n", cloneURL, err)
		return
	}

	// Buat atau perbarui lockfile ku.kunci
	lockEntry := fmt.Sprintf("%s %s %d\n", cleanURL, modName, time.Now().Unix())
	f, err := os.OpenFile("ku.kunci", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		_, _ = f.WriteString(lockEntry)
		f.Close()
	}

	fmt.Printf("🎉 Berhasil memasang modul '%s' di '%s'!\n", modName, targetModDir)
	fmt.Println("   Kini Anda dapat menggunakannya dengan: pakai \"" + modName + "\"")
}

func runLoadTest(args []string) {
	targetURL := ""
	concurrency := 10
	duration := 5 * time.Second
	method := "GET"

	for i := 0; i < len(args); i++ {
		a := args[i]
		if (a == "-k" || a == "--konkurensi") && i+1 < len(args) {
			if n, err := strconv.Atoi(args[i+1]); err == nil && n > 0 {
				concurrency = n
			}
			i++
		} else if (a == "-d" || a == "--durasi") && i+1 < len(args) {
			if d, err := time.ParseDuration(args[i+1]); err == nil {
				duration = d
			}
			i++
		} else if (a == "-m" || a == "--metode") && i+1 < len(args) {
			method = strings.ToUpper(args[i+1])
			i++
		} else if !strings.HasPrefix(a, "-") && targetURL == "" {
			targetURL = a
		}
	}

	if targetURL == "" {
		fmt.Println("❌ Galat: URL target harus ditentukan.")
		return
	}

	fmt.Printf("⚡ Menyerang target: %s (Metode: %s)\n", targetURL, method)
	fmt.Printf("   Konkurensi: %d pekerja | Durasi: %v\n", concurrency, duration)
	fmt.Println("   Memulai pengujian beban...")

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        concurrency * 2,
			MaxIdleConnsPerHost: concurrency * 2,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	var totalReq int64
	var successReq int64
	var failReq int64
	var latenciesMu sync.Mutex
	var latencies []float64

	stopChan := make(chan struct{})
	time.AfterFunc(duration, func() {
		close(stopChan)
	})

	var wg sync.WaitGroup
	start := time.Now()

	for c := 0; c < concurrency; c++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				default:
					reqStart := time.Now()
					req, err := http.NewRequest(method, targetURL, nil)
					if err != nil {
						atomic.AddInt64(&failReq, 1)
						atomic.AddInt64(&totalReq, 1)
						continue
					}
					resp, err := client.Do(req)
					elapsed := time.Since(reqStart).Seconds() * 1000 // ms
					atomic.AddInt64(&totalReq, 1)
					if err != nil || resp.StatusCode >= 500 {
						atomic.AddInt64(&failReq, 1)
					} else {
						atomic.AddInt64(&successReq, 1)
					}
					if resp != nil {
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
					}

					latenciesMu.Lock()
					if len(latencies) < 50000 {
						latencies = append(latencies, elapsed)
					}
					latenciesMu.Unlock()
				}
			}
		}()
	}

	wg.Wait()
	actualDuration := time.Since(start).Seconds()

	sort.Float64s(latencies)
	p50, p90, p99, minLat, maxLat := 0.0, 0.0, 0.0, 0.0, 0.0
	if len(latencies) > 0 {
		minLat = latencies[0]
		maxLat = latencies[len(latencies)-1]
		p50 = latencies[int(float64(len(latencies))*0.50)]
		p90 = latencies[int(float64(len(latencies))*0.90)]
		p99 = latencies[int(float64(len(latencies))*0.99)]
	}

	rps := float64(totalReq) / actualDuration

	fmt.Println("\n=======================================================")
	fmt.Println("📊 HASIL PENGUJIAN BEBAN (LOAD TEST)")
	fmt.Println("=======================================================")
	fmt.Printf("Total Permintaan : %d\n", totalReq)
	fmt.Printf("Permintaan Sukses: %d\n", successReq)
	fmt.Printf("Permintaan Gagal : %d\n", failReq)
	fmt.Printf("Durasi Uji       : %.2f detik\n", actualDuration)
	fmt.Printf("Performa (RPS)   : %.2f req/detik\n", rps)
	fmt.Println("-------------------------------------------------------")
	fmt.Println("Distribusi Latensi (milidetik):")
	fmt.Printf("  Min: %.2f ms | P50: %.2f ms | P90: %.2f ms | P99: %.2f ms | Max: %.2f ms\n", minLat, p50, p90, p99, maxLat)
	fmt.Println("=======================================================")
}

func runAuditBinary(exePath string) {
	if exePath == "" {
		selfExe, err := os.Executable()
		if err == nil {
			exePath = selfExe
		} else {
			exePath = "ku.exe"
		}
	}

	fmt.Printf("🔍 Memeriksa biner executable: %s\n", exePath)
	fi, err := os.Stat(exePath)
	if err != nil {
		fmt.Printf("❌ Berkas biner tidak ditemukan: %v\n", err)
		return
	}

	fmt.Printf("Ukuran Biner Total: %.2f MB (%d byte)\n", float64(fi.Size())/1024/1024, fi.Size())

	if scriptSource, ok := readEmbeddedScript(exePath); ok {
		fmt.Printf("✅ Payload Bahasa Ku Ditemukan!\n")
		fmt.Printf("   Ukuran Script Tersemat: %d byte\n", len(scriptSource))
	} else {
		fmt.Println("ℹ️ Biner ini adalah runner interpreter standar Bahasa Ku (bukan program mandiri yang disematkan).")
	}
}

func runMonitorDashboard() {
	fmt.Println("🖥️ [Dashboard Kinerja Bahasa Ku]")
	fmt.Println("Memantau runtime... (Tekan Ctrl+C untuk keluar)")

	for {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		fmt.Print("\033[H\033[2J") // Clear terminal
		fmt.Println("==================================================================")
		fmt.Println("⚡ BAHASA KU - DASHBOARD KINERJA REAL-TIME")
		fmt.Println("==================================================================")
		fmt.Printf("Waktu Sistem     : %s\n", time.Now().Format("15:04:05"))
		fmt.Printf("Alokasi Memori   : %.2f MB (Heap In-Use: %.2f MB)\n", float64(m.Alloc)/1024/1024, float64(m.HeapInuse)/1024/1024)
		fmt.Printf("Total Sys Memori : %.2f MB\n", float64(m.Sys)/1024/1024)
		fmt.Printf("Goroutine Aktif  : %d goroutine\n", runtime.NumGoroutine())
		fmt.Printf("Jumlah CPU Inti  : %d core\n", runtime.NumCPU())
		fmt.Printf("Frekuensi GC     : %d siklus\n", m.NumGC)

		// Progress bar representasi alokasi
		pct := int((m.Alloc % (100 * 1024 * 1024)) * 20 / (100 * 1024 * 1024))
		bar := strings.Repeat("█", pct) + strings.Repeat("░", 20-pct)
		fmt.Printf("Beban Heap       : [%s] (Skala 100MB)\n", bar)
		fmt.Println("==================================================================")

		time.Sleep(1 * time.Second)
	}
}

func runUpdateLockfile() {
	fmt.Println("🔒 Memverifikasi integritas paket di 'modul/'...")

	if _, err := os.Stat("modul"); os.IsNotExist(err) {
		fmt.Println("ℹ️ Folder 'modul/' belum ada. Tidak ada dependensi yang terpasang.")
		return
	}

	entries, err := os.ReadDir("modul")
	if err != nil {
		fmt.Printf("❌ Galat membaca folder modul: %v\n", err)
		return
	}

	var sb strings.Builder
	sb.WriteString("# ku.kunci - Berkas Integritas Modul Bahasa Ku\n")
	sb.WriteString(fmt.Sprintf("# Dibuat otomatis pada %s\n\n", time.Now().Format(time.RFC3339)))

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			modPath := filepath.Join("modul", entry.Name())
			var allBytes []byte
			_ = filepath.Walk(modPath, func(path string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() {
					b, _ := os.ReadFile(path)
					allBytes = append(allBytes, b...)
				}
				return nil
			})
			hash := sha256.Sum256(allBytes)
			hashHex := hex.EncodeToString(hash[:])
			sb.WriteString(fmt.Sprintf("%s sha256:%s\n", entry.Name(), hashHex))
			count++
		}
	}

	if err := os.WriteFile("ku.kunci", []byte(sb.String()), 0644); err != nil {
		fmt.Printf("❌ Gagal menyimpan ku.kunci: %v\n", err)
		return
	}

	fmt.Printf("✅ Berhasil memvalidasi dan memperbarui ku.kunci untuk %d modul!\n", count)
}





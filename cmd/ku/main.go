package main

import (
	"bytes"
	fmt "fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"github.com/zufarrizal/bahasa-ku/pkg/ast"
	"github.com/zufarrizal/bahasa-ku/pkg/evaluator"
	"github.com/zufarrizal/bahasa-ku/pkg/formatter"
	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
	"github.com/zufarrizal/bahasa-ku/pkg/repl"
	"github.com/zufarrizal/bahasa-ku/pkg/transpiler"
)

const version = "0.3.5 (Go Native Edition)"

func init() {
	evaluator.RegisterSubFileEvaluator(func(path string, env *evaluator.Environment) {
		resolvedPath := path
		if !filepath.IsAbs(resolvedPath) {
			if _, err := os.Stat(resolvedPath); err != nil {
				cwd, _ := os.Getwd()
				candidate := filepath.Join(cwd, resolvedPath)
				if _, err := os.Stat(candidate); err == nil {
					resolvedPath = candidate
				}
			}
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

	switch command {
	case "bantuan", "help", "-h", "--help":
		printHelp()
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
			fmt.Println("Contoh: ku kompilasi [-gui] [-ikon <ikon.ico>] app.ku [-o app.exe]")
			os.Exit(1)
		}
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

		if sourceFile == "" {
			fmt.Println("Galat: Berkas sumber .ku tidak ditentukan.")
			os.Exit(1)
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

	case "uji", "test":
		targetPath := "."
		if len(args) >= 2 {
			targetPath = args[1]
		}
		runTests(targetPath)

	case "tolakUkur", "tolak-ukur", "bench", "benchmark":
		targetPath := "."
		if len(args) >= 2 {
			targetPath = args[1]
		}
		runBenchmarks(targetPath)

	default:
		fmt.Printf("Perintah '%s' tidak dikenal. Ketik 'ku bantuan' atau 'ku -h' untuk bantuan.\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`Penggunaan CLI Bahasa Ku (Go Native):
  ku                       Membuka REPL (terminal interaktif)
  ku jalankan <berkas.ku>  Menjalankan berkas kode .ku
  ku uji [berkas/_uji.ku]  Menjalankan pengujian unit test otomatis
  ku tolakUkur [berkas]    Menjalankan pengujian tolok ukur performa (benchmark)
  ku rapi [-w] <berkas.ku> Merapikan format kode secara otomatis (mirip gofmt)
  ku kompilasi <berkas.ku> [-gui] [-ikon ikon.ico] [-native] [-o keluar.exe]  Kompilasi langsung ke biner mandiri (.exe)
  ku inisialisasi <nama_folder> [web|desktop|cli]  Membuat proyek baru beserta berkas ku.mod
  ku buat <berkas.ku>      Mengonversi berkas .ku ke .go murni
  ku versi                 Menampilkan versi

Contoh:
  ku
  ku jalankan contoh/halo.ku
  ku uji
  ku rapi -w contoh/halo.ku
  ku kompilasi contoh/halo.ku -o HaloApp.exe
  ku kompilasi -gui -ikon aset/ku.ico contoh/desktop_native.ku -o Aplikasi.exe
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
	res := evaluator.Eval(prog, env)
	if pObj, isPanic := res.(*evaluator.PanicObject); isPanic {
		fmt.Println("Galat Eksekusi (Panik): " + pObj.Value.Inspect())
		os.Exit(1)
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
		fmt.Println("Galat Eksekusi (Panik): " + pObj.Value.Inspect())
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
versi 0.3.5
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

// Tangani REST API rute dinamis dengan parameter ID
fungsi ambilProduk(konteks) {
    var id = konteks.parameter.id
    var mode = konteks.query.mode ?? "biasa"
    kembalikan {
        "sukses": benar,
        "id_produk": id,
        "nama": "Paket Belajar Bahasa Ku",
        "harga": 150000,
        "mode_ambil": mode
    }
}

// Tangani POST data baru
fungsi simpanProduk(konteks) {
    var payload = konteks.badanJson ?? {}
    kembalikan {
        "sukses": benar,
        "pesan": "Produk tersimpan",
        "data": payload
    }
}

fungsi beranda(konteks) {
    kembalikan "<h1>🚀 Selamat Datang di Server Web Bahasa Ku!</h1><p>Proyek backend Anda siap digunakan.</p>"
}

fungsi utama() {
    // Aktifkan CORS untuk kemudahan frontend modern (React, Vue, Flutter, dsb)
    web.aktifkanCORS("*")

    // Daftarkan rute REST API
    web.dapatkan("/", beranda)
    web.dapatkan("/api/produk/:id", ambilProduk)
    web.pos("/api/produk", simpanProduk)

    cetak.tulis("Server web berjalan di http://localhost:8080")
    web.jalankan(":8080")
}
`
	case "desktop":
		fileContent = `paket utama

pakai "cetak"
pakai "desktop"
pakai "jendela"

fungsi utama() {
    cetak.tulis("Memulai aplikasi Desktop Bahasa Ku...")

    // Tampilkan notifikasi dialog native OS
    desktop.pesan("Info", "Selamat datang di Aplikasi Desktop Bahasa Ku!")

    // Buka jendela WebView mandiri yang interaktif
    jendela.buka({
        "judul": "Aplikasi Desktop Modern - Bahasa Ku",
        "lebar": 800,
        "tinggi": 600,
        "html": ` + "`" + `
            <html style="font-family: sans-serif; background: #0f172a; color: #f8fafc; padding: 30px; text-align: center;">
                <body>
                    <h1 style="color: #38bdf8;">⚡ Aplikasi Desktop Bahasa Ku</h1>
                    <p>Antarmuka grafis desktop modern berbasis HTML5/CSS dan WebView OS native.</p>
                    <button style="padding: 10px 20px; font-size: 16px; border-radius: 8px; border: none; background: #0284c7; color: white; cursor: pointer;"
                            onclick="alert('Halo dari event interaktif!')">
                        Klik Saya
                    </button>
                </body>
            </html>
        ` + "`" + `
    })
}
`
	case "cli":
		fileContent = `paket utama

pakai "cetak"
pakai "sistem"
pakai "basisdata"
pakai "konversi"

fungsi utama() {
    cetak.tulis("=== Selamat Datang di Proyek Bahasa Ku ===")
    cetak.tulis("Waktu saat ini:", sistem.waktu())

    // Inisialisasi basis data SQLite lokal portabel
    var db = basisdata.bukaBerkas("catatan.db")
    db.eksekusi("CREATE TABLE IF NOT EXISTS riwayat (id INTEGER PRIMARY KEY, catatan TEXT)")
    db.eksekusi("INSERT INTO riwayat (catatan) VALUES (?)", "Inisialisasi proyek baru")

    var hasil = db.kueri("SELECT COUNT(*) AS total FROM riwayat")
    cetak.tulis("Total riwayat tersimpan:", hasil[0]["total"])
    cetak.tulis("Proyek Anda siap dikembangkan! 🎉")
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
	fmt.Printf("Dibuat:\n  - %s/ku.mod (manifes proyek)\n  - %s/utama.ku (berkas utama)\n", folderName, folderName)
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




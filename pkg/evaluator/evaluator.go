package evaluator

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	mrand "math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zufarrizal/bahasa-ku/pkg/ast"
)

type ObjectType string

const (
	INTEGER_OBJ      = "INTEGER"
	FLOAT_OBJ        = "FLOAT"
	BOOLEAN_OBJ      = "BOOLEAN"
	STRING_OBJ       = "STRING"
	NIL_OBJ          = "NIL"
	RETURN_VALUE_OBJ = "RETURN_VALUE"
	FUNCTION_OBJ     = "FUNCTION"
	BUILTIN_OBJ      = "BUILTIN"
	ARRAY_OBJ        = "ARRAY"
	MAP_OBJ          = "MAP"
	STRUCT_DEF_OBJ   = "STRUCT_DEF"
	STRUCT_INST_OBJ  = "STRUCT_INST"
)

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Integer struct{ Value int64 }

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

type Float struct{ Value float64 }

func (f *Float) Type() ObjectType { return FLOAT_OBJ }
func (f *Float) Inspect() string  { return fmt.Sprintf("%f", f.Value) }

type Boolean struct{ Value bool }

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string {
	if b.Value {
		return "benar"
	}
	return "salah"
}

type String struct{ Value string }

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }

type Nil struct{}

func (n *Nil) Type() ObjectType { return NIL_OBJ }
func (n *Nil) Inspect() string  { return "kosong" }

type ReturnValue struct{ Value Object }

func (r *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (r *ReturnValue) Inspect() string  { return r.Value.Inspect() }

type Environment struct {
	store map[string]Object
	outer *Environment
	deferStack []func()
}

func NewEnvironment() *Environment {
	return &Environment{store: make(map[string]Object), deferStack: []func(){}}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Object) Object {
	e.store[name] = val
	return val
}

func (e *Environment) Update(name string, val Object) bool {
	if _, ok := e.store[name]; ok {
		e.store[name] = val
		return true
	}
	if e.outer != nil {
		return e.outer.Update(name, val)
	}
	return false
}

type BuiltinFunction func(args ...Object) Object

type Builtin struct{ Fn BuiltinFunction }

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "fungsi bawaan" }

type Function struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string  { return "fungsi" }

type Array struct{ Elements []Object }

func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string {
	elems := []string{}
	for _, e := range a.Elements {
		elems = append(elems, e.Inspect())
	}
	return "[" + strings.Join(elems, ", ") + "]"
}

type Map struct{ Pairs map[string]Object }

func (m *Map) Type() ObjectType { return MAP_OBJ }
func (m *Map) Inspect() string {
	pairs := []string{}
	for k, v := range m.Pairs {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.Inspect()))
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}

type StructDef struct {
	Name   string
	Fields []ast.StructField
}

func (s *StructDef) Type() ObjectType { return STRUCT_DEF_OBJ }
func (s *StructDef) Inspect() string  { return "definisi struktur " + s.Name }

type StructInstance struct {
	Def    *StructDef
	Fields map[string]Object
}

func (si *StructInstance) Type() ObjectType { return STRUCT_INST_OBJ }
func (si *StructInstance) Inspect() string {
	fields := []string{}
	for k, v := range si.Fields {
		fields = append(fields, fmt.Sprintf("%s: %s", k, v.Inspect()))
	}
	return si.Def.Name + "{" + strings.Join(fields, ", ") + "}"
}

var (
	TRUE  = &Boolean{Value: true}
	FALSE = &Boolean{Value: false}
	NIL   = &Nil{}
)

func InitGlobalEnvironment() *Environment {
	env := NewEnvironment()

	// Modul cetak
	cetakMap := &Map{Pairs: make(map[string]Object)}
	cetakMap.Pairs["tulis"] = &Builtin{
		Fn: func(args ...Object) Object {
			parts := []string{}
			for _, a := range args {
				parts = append(parts, a.Inspect())
			}
			fmt.Println(strings.Join(parts, " "))
			return NIL
		},
	}
	cetakMap.Pairs["tulisBaris"] = cetakMap.Pairs["tulis"]
	env.Set("cetak", cetakMap)

	// Modul web bawaan
	webMap := &Map{Pairs: make(map[string]Object)}
	webMap.Pairs["tangani"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			pathStr := args[0].Inspect()
			fnObj := args[1]

			http.HandleFunc(pathStr, func(w http.ResponseWriter, r *http.Request) {
				reqMap := &Map{Pairs: make(map[string]Object)}
				reqMap.Pairs["metode"] = &String{Value: r.Method}
				reqMap.Pairs["jalur"] = &String{Value: r.URL.Path}

				respObj := applyFunction(fnObj, []Object{reqMap}, env)
				if respObj == nil || respObj == NIL {
					return
				}

				if m, ok := respObj.(*Map); ok {
					// Balas sebagai JSON
					data := make(map[string]interface{})
					for k, v := range m.Pairs {
						data[k] = v.Inspect()
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(data)
					return
				}

				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write([]byte(respObj.Inspect()))
			})
			return NIL
		},
	}
	webMap.Pairs["dengarkan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return NIL
			}
			addr := args[0].Inspect()
			err := http.ListenAndServe(addr, nil)
			if err != nil {
				fmt.Printf("Galat server web: %v\n", err)
			}
			return NIL
		},
	}
	env.Set("web", webMap)

	// Modul desktop: jendela
	jendelaMap := &Map{Pairs: make(map[string]Object)}
	jendelaMap.Pairs["pesan"] = &Builtin{
		Fn: func(args ...Object) Object {
			judul := "Bahasa Ku"
			isi := ""
			if len(args) == 1 {
				isi = args[0].Inspect()
			} else if len(args) >= 2 {
				judul = args[0].Inspect()
				isi = args[1].Inspect()
			}

			tampilkanPesanNative(judul, isi)
			return NIL
		},
	}

	jendelaMap.Pairs["buka"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return NIL
			}

			judul := "Aplikasi Desktop Bahasa Ku"
			konten := "<h1>Bahasa Ku Desktop</h1>"
			url := ""

			if m, ok := args[0].(*Map); ok {
				if j, ok := m.Pairs["judul"]; ok {
					judul = j.Inspect()
				}
				if k, ok := m.Pairs["konten"]; ok {
					konten = k.Inspect()
				}
				if u, ok := m.Pairs["url"]; ok {
					url = u.Inspect()
				}
			}

			if url == "" {
				tempDir := os.TempDir()
				htmlFile := filepath.Join(tempDir, "ku_app.html")
				htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>%s</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; padding: 24px; background: #f8fafc; color: #1e293b; margin: 0; }
  h1 { color: #0284c7; }
</style>
</head>
<body>
%s
</body>
</html>`, judul, konten)
				_ = os.WriteFile(htmlFile, []byte(htmlContent), 0644)
				url = htmlFile
			}

			if runtime.GOOS == "windows" {
				// Membuka jendela webview mandiri menggunakan browser app mode
				edgePath := filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe")
				if _, err := os.Stat(edgePath); err == nil {
					cmd := exec.Command(edgePath, fmt.Sprintf("--app=%s", url), "--window-size=900,650")
					_ = cmd.Start()
					return NIL
				}
				// Fallback standar default browser
				_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
			} else {
				_ = exec.Command("xdg-open", url).Start()
			}

			return NIL
		},
	}
	env.Set("jendela", jendelaMap)

	// Modul basisdata (Database / MySQL)
	basisdataMap := &Map{Pairs: make(map[string]Object)}
	basisdataMap.Pairs["sambung"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			driver := args[0].Inspect() // misal "mysql"
			dsn := args[1].Inspect()    // misal "user:password@tcp(127.0.0.1:3306)/dbname"

			db, err := sql.Open(driver, dsn)
			connMap := &Map{Pairs: make(map[string]Object)}

			if err != nil {
				connMap.Pairs["terhubung"] = FALSE
				connMap.Pairs["pesan"] = &String{Value: err.Error()}
				return connMap
			}

			// Uji ping
			pingErr := db.Ping()
			if pingErr != nil {
				connMap.Pairs["terhubung"] = FALSE
				connMap.Pairs["pesan"] = &String{Value: pingErr.Error()}
			} else {
				connMap.Pairs["terhubung"] = TRUE
				connMap.Pairs["pesan"] = &String{Value: "Koneksi berhasil"}
			}

			// Method eksekusi query
			connMap.Pairs["kueri"] = &Builtin{
				Fn: func(qArgs ...Object) Object {
					if len(qArgs) < 1 {
						return &Array{Elements: []Object{}}
					}
					queryStr := qArgs[0].Inspect()
					rows, qErr := db.Query(queryStr)
					if qErr != nil {
						fmt.Printf("Galat kueri: %v\n", qErr)
						return &Array{Elements: []Object{}}
					}
					defer rows.Close()

					cols, _ := rows.Columns()
					hasilArray := &Array{Elements: []Object{}}

					for rows.Next() {
						vals := make([]interface{}, len(cols))
						valPtrs := make([]interface{}, len(cols))
						for i := range vals {
							valPtrs[i] = &vals[i]
						}
						rows.Scan(valPtrs...)

						barisMap := &Map{Pairs: make(map[string]Object)}
						for i, col := range cols {
							valStr := ""
							if vals[i] != nil {
								if b, ok := vals[i].([]byte); ok {
									valStr = string(b)
								} else {
									valStr = fmt.Sprintf("%v", vals[i])
								}
							}
							barisMap.Pairs[col] = &String{Value: valStr}
						}
						hasilArray.Elements = append(hasilArray.Elements, barisMap)
					}
					return hasilArray
				},
			}

			// Method eksekusi perintah (INSERT, UPDATE, CREATE, DELETE)
			connMap.Pairs["eksekusi"] = &Builtin{
				Fn: func(eArgs ...Object) Object {
					if len(eArgs) < 1 {
						return FALSE
					}
					sqlStr := eArgs[0].Inspect()
					_, exErr := db.Exec(sqlStr)
					if exErr != nil {
						fmt.Printf("Galat eksekusi: %v\n", exErr)
						return FALSE
					}
					return TRUE
				},
			}

			// Tutup koneksi
			connMap.Pairs["tutup"] = &Builtin{
				Fn: func(tArgs ...Object) Object {
					db.Close()
					return NIL
				},
			}

			return connMap
		},
	}
	env.Set("basisdata", basisdataMap)

	// Modul berkas (File I/O)
	berkasMap := &Map{Pairs: make(map[string]Object)}
	berkasMap.Pairs["baca"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return NIL
			}
			pathStr := args[0].Inspect()
			data, err := os.ReadFile(pathStr)
			if err != nil {
				return NIL
			}
			return &String{Value: string(data)}
		},
	}
	berkasMap.Pairs["tulis"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			pathStr := args[0].Inspect()
			content := args[1].Inspect()
			err := os.WriteFile(pathStr, []byte(content), 0644)
			if err != nil {
				return FALSE
			}
			return TRUE
		},
	}
	berkasMap.Pairs["tambah"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			pathStr := args[0].Inspect()
			content := args[1].Inspect()
			f, err := os.OpenFile(pathStr, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return FALSE
			}
			defer f.Close()
			_, err = f.WriteString(content)
			return &Boolean{Value: err == nil}
		},
	}
	berkasMap.Pairs["ada"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			pathStr := args[0].Inspect()
			_, err := os.Stat(pathStr)
			if err == nil {
				return TRUE
			}
			return FALSE
		},
	}
	berkasMap.Pairs["hapus"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			pathStr := args[0].Inspect()
			err := os.Remove(pathStr)
			if err == nil {
				return TRUE
			}
			return FALSE
		},
	}
	env.Set("berkas", berkasMap)

	// Modul sistem (OS, Env, Argumen, Waktu)
	sistemMap := &Map{Pairs: make(map[string]Object)}
	sistemMap.Pairs["lingkungan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			key := args[0].Inspect()
			return &String{Value: os.Getenv(key)}
		},
	}
	sistemMap.Pairs["waktu"] = &Builtin{
		Fn: func(args ...Object) Object {
			return &String{Value: time.Now().Format("2006-01-02 15:04:05")}
		},
	}
	sistemMap.Pairs["argumen"] = &Builtin{
		Fn: func(args ...Object) Object {
			arr := &Array{Elements: []Object{}}
			for _, a := range os.Args {
				arr.Elements = append(arr.Elements, &String{Value: a})
			}
			return arr
		},
	}
	env.Set("sistem", sistemMap)

	// Modul jaringan (HTTP Client / Fetch API)
	jaringanMap := &Map{Pairs: make(map[string]Object)}
	jaringanMap.Pairs["ambil"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return NIL
			}
			urlStr := args[0].Inspect()
			client := &http.Client{Timeout: 15 * time.Second}
			resp, err := client.Get(urlStr)
			if err != nil {
				resMap := &Map{Pairs: make(map[string]Object)}
				resMap.Pairs["sukses"] = FALSE
				resMap.Pairs["status"] = &Integer{Value: 0}
				resMap.Pairs["badan"] = &String{Value: err.Error()}
				return resMap
			}
			defer resp.Body.Close()

			bodyBytes, _ := io.ReadAll(resp.Body)
			bodyStr := string(bodyBytes)

			resMap := &Map{Pairs: make(map[string]Object)}
			resMap.Pairs["sukses"] = &Boolean{Value: resp.StatusCode >= 200 && resp.StatusCode < 300}
			resMap.Pairs["status"] = &Integer{Value: int64(resp.StatusCode)}
			resMap.Pairs["badan"] = &String{Value: bodyStr}

			// Coba parse JSON jika konten berupa JSON
			var jsonObj interface{}
			if err := json.Unmarshal(bodyBytes, &jsonObj); err == nil {
				resMap.Pairs["json"] = convertGoToKuObject(jsonObj)
			} else {
				resMap.Pairs["json"] = NIL
			}

			return resMap
		},
	}

	jaringanMap.Pairs["kirim"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			urlStr := args[0].Inspect()
			var reqBody io.Reader

			if m, ok := args[1].(*Map); ok {
				data := make(map[string]interface{})
				for k, v := range m.Pairs {
					data[k] = v.Inspect()
				}
				jsonBytes, _ := json.Marshal(data)
				reqBody = strings.NewReader(string(jsonBytes))
			} else {
				reqBody = strings.NewReader(args[1].Inspect())
			}

			client := &http.Client{Timeout: 15 * time.Second}
			resp, err := client.Post(urlStr, "application/json", reqBody)
			if err != nil {
				resMap := &Map{Pairs: make(map[string]Object)}
				resMap.Pairs["sukses"] = FALSE
				resMap.Pairs["status"] = &Integer{Value: 0}
				resMap.Pairs["badan"] = &String{Value: err.Error()}
				return resMap
			}
			defer resp.Body.Close()

			bodyBytes, _ := io.ReadAll(resp.Body)
			bodyStr := string(bodyBytes)

			resMap := &Map{Pairs: make(map[string]Object)}
			resMap.Pairs["sukses"] = &Boolean{Value: resp.StatusCode >= 200 && resp.StatusCode < 300}
			resMap.Pairs["status"] = &Integer{Value: int64(resp.StatusCode)}
			resMap.Pairs["badan"] = &String{Value: bodyStr}

			var jsonObj interface{}
			if err := json.Unmarshal(bodyBytes, &jsonObj); err == nil {
				resMap.Pairs["json"] = convertGoToKuObject(jsonObj)
			} else {
				resMap.Pairs["json"] = NIL
			}

			return resMap
		},
	}
	env.Set("jaringan", jaringanMap)

	// Modul kripto (Kriptografi, Hash, Base64, Token)
	kriptoMap := &Map{Pairs: make(map[string]Object)}
	kriptoMap.Pairs["sha256"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			h := sha256.Sum256([]byte(args[0].Inspect()))
			return &String{Value: hex.EncodeToString(h[:])}
		},
	}
	kriptoMap.Pairs["md5"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			h := md5.Sum([]byte(args[0].Inspect()))
			return &String{Value: hex.EncodeToString(h[:])}
		},
	}
	kriptoMap.Pairs["base64Encode"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			return &String{Value: base64.StdEncoding.EncodeToString([]byte(args[0].Inspect()))}
		},
	}
	kriptoMap.Pairs["base64Decode"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			data, err := base64.StdEncoding.DecodeString(args[0].Inspect())
			if err != nil {
				return &String{Value: ""}
			}
			return &String{Value: string(data)}
		},
	}
	kriptoMap.Pairs["tokenAcak"] = &Builtin{
		Fn: func(args ...Object) Object {
			n := 16
			if len(args) >= 1 {
				if i, ok := args[0].(*Integer); ok && i.Value > 0 {
					n = int(i.Value)
				}
			}
			bytes := make([]byte, n)
			_, _ = rand.Read(bytes)
			return &String{Value: hex.EncodeToString(bytes)[:n]}
		},
	}
	env.Set("kripto", kriptoMap)

	// Modul matematika (Kalkulasi Numerik)
	matematikaMap := &Map{Pairs: make(map[string]Object)}
	matematikaMap.Pairs["PI"] = &Float{Value: math.Pi}
	matematikaMap.Pairs["akar"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Float{Value: 0}
			}
			var val float64
			if i, ok := args[0].(*Integer); ok {
				val = float64(i.Value)
			} else if f, ok := args[0].(*Float); ok {
				val = f.Value
			}
			return &Float{Value: math.Sqrt(val)}
		},
	}
	matematikaMap.Pairs["pangkat"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Float{Value: 0}
			}
			var x, y float64
			if i, ok := args[0].(*Integer); ok {
				x = float64(i.Value)
			} else if f, ok := args[0].(*Float); ok {
				x = f.Value
			}
			if i, ok := args[1].(*Integer); ok {
				y = float64(i.Value)
			} else if f, ok := args[1].(*Float); ok {
				y = f.Value
			}
			return &Float{Value: math.Pow(x, y)}
		},
	}
	matematikaMap.Pairs["bulat"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Integer{Value: 0}
			}
			var val float64
			if i, ok := args[0].(*Integer); ok {
				return i
			} else if f, ok := args[0].(*Float); ok {
				val = f.Value
			}
			return &Integer{Value: int64(math.Round(val))}
		},
	}
	matematikaMap.Pairs["acak"] = &Builtin{
		Fn: func(args ...Object) Object {
			minVal := int64(0)
			maxVal := int64(100)
			if len(args) >= 2 {
				if i, ok := args[0].(*Integer); ok {
					minVal = i.Value
				}
				if i, ok := args[1].(*Integer); ok {
					maxVal = i.Value
				}
			}
			if maxVal <= minVal {
				return &Integer{Value: minVal}
			}
			res := minVal + mrand.Int63n(maxVal-minVal+1)
			return &Integer{Value: res}
		},
	}
	matematikaMap.Pairs["maks"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Integer{Value: 0}
			}
			if i1, ok := args[0].(*Integer); ok {
				if i2, ok := args[1].(*Integer); ok {
					if i1.Value > i2.Value {
						return i1
					}
					return i2
				}
			}
			return args[0]
		},
	}
	matematikaMap.Pairs["min"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Integer{Value: 0}
			}
			if i1, ok := args[0].(*Integer); ok {
				if i2, ok := args[1].(*Integer); ok {
					if i1.Value < i2.Value {
						return i1
					}
					return i2
				}
			}
			return args[0]
		},
	}
	env.Set("matematika", matematikaMap)

	// Modul teks (Manipulasi String)
	teksMap := &Map{Pairs: make(map[string]Object)}
	teksMap.Pairs["pisah"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Array{Elements: []Object{}}
			}
			s := args[0].Inspect()
			sep := args[1].Inspect()
			parts := strings.Split(s, sep)
			arr := &Array{Elements: []Object{}}
			for _, p := range parts {
				arr.Elements = append(arr.Elements, &String{Value: p})
			}
			return arr
		},
	}
	teksMap.Pairs["gabung"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &String{Value: ""}
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &String{Value: ""}
			}
			sep := args[1].Inspect()
			items := []string{}
			for _, e := range arr.Elements {
				items = append(items, e.Inspect())
			}
			return &String{Value: strings.Join(items, sep)}
		},
	}
	teksMap.Pairs["kapital"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			return &String{Value: strings.ToUpper(args[0].Inspect())}
		},
	}
	teksMap.Pairs["kecil"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			return &String{Value: strings.ToLower(args[0].Inspect())}
		},
	}
	teksMap.Pairs["pangkas"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			return &String{Value: strings.TrimSpace(args[0].Inspect())}
		},
	}
	teksMap.Pairs["mengandung"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			return &Boolean{Value: strings.Contains(args[0].Inspect(), args[1].Inspect())}
		},
	}
	teksMap.Pairs["ganti"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 3 {
				return &String{Value: ""}
			}
			s := args[0].Inspect()
			lama := args[1].Inspect()
			baru := args[2].Inspect()
			return &String{Value: strings.ReplaceAll(s, lama, baru)}
		},
	}
	env.Set("teks", teksMap)

	// Modul konversi (Konversi Tipe Data)
	konversiMap := &Map{Pairs: make(map[string]Object)}
	konversiMap.Pairs["keAngka"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Integer{Value: 0}
			}
			if i, ok := args[0].(*Integer); ok {
				return i
			}
			v, err := strconv.ParseInt(strings.TrimSpace(args[0].Inspect()), 10, 64)
			if err != nil {
				return &Integer{Value: 0}
			}
			return &Integer{Value: v}
		},
	}
	konversiMap.Pairs["kePecahan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Float{Value: 0}
			}
			if f, ok := args[0].(*Float); ok {
				return f
			}
			v, err := strconv.ParseFloat(strings.TrimSpace(args[0].Inspect()), 64)
			if err != nil {
				return &Float{Value: 0}
			}
			return &Float{Value: v}
		},
	}
	konversiMap.Pairs["keTeks"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			return &String{Value: args[0].Inspect()}
		},
	}
	konversiMap.Pairs["tipe"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: "kosong"}
			}
			switch args[0].Type() {
			case INTEGER_OBJ:
				return &String{Value: "angka"}
			case FLOAT_OBJ:
				return &String{Value: "pecahan"}
			case STRING_OBJ:
				return &String{Value: "teks"}
			case BOOLEAN_OBJ:
				return &String{Value: "boolean"}
			case ARRAY_OBJ:
				return &String{Value: "daftar"}
			case MAP_OBJ:
				return &String{Value: "peta"}
			case STRUCT_INST_OBJ:
				return &String{Value: "struktur"}
			default:
				return &String{Value: "objek"}
			}
		},
	}
	env.Set("konversi", konversiMap)

	// Bawaan: panjang & tambah
	env.Set("panjang", &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return &Integer{Value: 0}
			}
			switch arg := args[0].(type) {
			case *Array:
				return &Integer{Value: int64(len(arg.Elements))}
			case *String:
				return &Integer{Value: int64(len(arg.Value))}
			case *Map:
				return &Integer{Value: int64(len(arg.Pairs))}
			}
			return &Integer{Value: 0}
		},
	})

	env.Set("tambah", &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			if arr, ok := args[0].(*Array); ok {
				newElems := append([]Object{}, arr.Elements...)
				newElems = append(newElems, args[1:]...)
				return &Array{Elements: newElems}
			}
			return NIL
		},
	})

	return env
}

func convertGoToKuObject(val interface{}) Object {
	if val == nil {
		return NIL
	}
	switch v := val.(type) {
	case bool:
		if v {
			return TRUE
		}
		return FALSE
	case float64:
		if v == float64(int64(v)) {
			return &Integer{Value: int64(v)}
		}
		return &Float{Value: v}
	case string:
		return &String{Value: v}
	case []interface{}:
		arr := &Array{Elements: []Object{}}
		for _, item := range v {
			arr.Elements = append(arr.Elements, convertGoToKuObject(item))
		}
		return arr
	case map[string]interface{}:
		m := &Map{Pairs: make(map[string]Object)}
		for k, item := range v {
			m.Pairs[k] = convertGoToKuObject(item)
		}
		return m
	default:
		return &String{Value: fmt.Sprintf("%v", v)}
	}
}

func Eval(node ast.Node, env *Environment) Object {
	if node == nil {
		return NIL
	}

	switch n := node.(type) {
	case *ast.Program:
		var result Object = NIL
		for _, stmt := range n.Statements {
			result = Eval(stmt, env)
			if ret, ok := result.(*ReturnValue); ok {
				return ret.Value
			}
		}
		if mainFn, ok := env.Get("utama"); ok {
			if fn, ok := mainFn.(*Function); ok {
				return applyFunction(fn, []Object{}, env)
			}
		}
		return result

	case *ast.PaketStatement, *ast.PakaiStatement:
		return NIL

	case *ast.TypeStructStatement:
		def := &StructDef{Name: n.Name.Value, Fields: n.Fields}
		env.Set(n.Name.Value, def)
		return def

	case *ast.VarStatement:
		var val Object = NIL
		if n.Value != nil {
			val = Eval(n.Value, env)
		}
		env.Set(n.Name.Value, val)
		return val

	case *ast.AssignStatement:
		val := Eval(n.Value, env)
		if ident, ok := n.Target.(*ast.Identifier); ok {
			if n.IsDefine {
				env.Set(ident.Value, val)
			} else {
				if !env.Update(ident.Value, val) {
					env.Set(ident.Value, val)
				}
			}
			return val
		} else if idxExpr, ok := n.Target.(*ast.IndexExpression); ok {
			left := Eval(idxExpr.Left, env)
			idx := Eval(idxExpr.Index, env)
			if arr, ok := left.(*Array); ok {
				if i, ok := idx.(*Integer); ok && i.Value >= 0 && i.Value < int64(len(arr.Elements)) {
					arr.Elements[i.Value] = val
				}
			} else if m, ok := left.(*Map); ok {
				m.Pairs[idx.Inspect()] = val
			}
			return val
		} else if dotExpr, ok := n.Target.(*ast.DotExpression); ok {
			left := Eval(dotExpr.Left, env)
			if si, ok := left.(*StructInstance); ok {
				si.Fields[dotExpr.Property.Value] = val
			} else if m, ok := left.(*Map); ok {
				m.Pairs[dotExpr.Property.Value] = val
			}
			return val
		}
		return NIL

	case *ast.FunctionStatement:
		fn := &Function{Parameters: n.Parameters, Body: n.Body, Env: env}
		env.Set(n.Name.Value, fn)
		return fn

	case *ast.BlockStatement:
		var result Object = NIL
		for _, stmt := range n.Statements {
			result = Eval(stmt, env)
			if result != nil && result.Type() == RETURN_VALUE_OBJ {
				return result
			}
		}
		return result

	case *ast.IfStatement:
		cond := Eval(n.Condition, env)
		if isTruthy(cond) {
			return Eval(n.Consequence, env)
		} else if n.Alternative != nil {
			return Eval(n.Alternative, env)
		}
		return NIL

	case *ast.ForStatement:
		loopEnv := NewEnclosedEnvironment(env)
		if n.IsRange {
			iterObj := Eval(n.Iterable, env)
			if arr, ok := iterObj.(*Array); ok {
				for i, elem := range arr.Elements {
					if n.Key != nil {
						loopEnv.Set(n.Key.Value, &Integer{Value: int64(i)})
					}
					if n.Value != nil {
						loopEnv.Set(n.Value.Value, elem)
					}
					res := Eval(n.Body, loopEnv)
					if res != nil && res.Type() == RETURN_VALUE_OBJ {
						return res
					}
				}
				return NIL
			} else if m, ok := iterObj.(*Map); ok {
				for k, v := range m.Pairs {
					if n.Key != nil {
						loopEnv.Set(n.Key.Value, &String{Value: k})
					}
					if n.Value != nil {
						loopEnv.Set(n.Value.Value, v)
					}
					res := Eval(n.Body, loopEnv)
					if res != nil && res.Type() == RETURN_VALUE_OBJ {
						return res
					}
				}
				return NIL
			}
			return NIL
		}

		if n.Init != nil {
			Eval(n.Init, loopEnv)
		}
		var result Object = NIL
		for {
			if n.Cond != nil {
				c := Eval(n.Cond, loopEnv)
				if !isTruthy(c) {
					break
				}
			}
			result = Eval(n.Body, loopEnv)
			if result != nil && result.Type() == RETURN_VALUE_OBJ {
				return result
			}
			if n.Post != nil {
				Eval(n.Post, loopEnv)
			}
		}
		return result

	case *ast.SwitchStatement:
		var targetVal Object = TRUE
		if n.Value != nil {
			targetVal = Eval(n.Value, env)
		}
		for _, c := range n.Cases {
			val := Eval(c.Value, env)
			if objectsEqual(targetVal, val) {
				return Eval(c.Body, env)
			}
		}
		if n.Default != nil {
			return Eval(n.Default, env)
		}
		return NIL

	case *ast.ReturnStatement:
		var val Object = NIL
		if n.ReturnValue != nil {
			val = Eval(n.ReturnValue, env)
		}
		return &ReturnValue{Value: val}

	case *ast.DeferStatement:
		deferredCall := func() {
			Eval(n.Expression, env)
		}
		env.deferStack = append(env.deferStack, deferredCall)
		return NIL

	case *ast.ExpressionStatement:
		return Eval(n.Expression, env)

	case *ast.Identifier:
		if val, ok := env.Get(n.Value); ok {
			return val
		}
		return NIL

	case *ast.IntegerLiteral:
		return &Integer{Value: n.Value}

	case *ast.FloatLiteral:
		return &Float{Value: n.Value}

	case *ast.StringLiteral:
		return &String{Value: n.Value}

	case *ast.BooleanLiteral:
		if n.Value {
			return TRUE
		}
		return FALSE

	case *ast.NilLiteral:
		return NIL

	case *ast.ArrayLiteral:
		elems := []Object{}
		for _, e := range n.Elements {
			elems = append(elems, Eval(e, env))
		}
		return &Array{Elements: elems}

	case *ast.MapLiteral:
		m := &Map{Pairs: make(map[string]Object)}
		for _, p := range n.Pairs {
			k := Eval(p.Key, env)
			v := Eval(p.Value, env)
			m.Pairs[k.Inspect()] = v
		}
		return m

	case *ast.IndexExpression:
		left := Eval(n.Left, env)
		idx := Eval(n.Index, env)
		if arr, ok := left.(*Array); ok {
			if i, ok := idx.(*Integer); ok && i.Value >= 0 && i.Value < int64(len(arr.Elements)) {
				return arr.Elements[i.Value]
			}
		} else if m, ok := left.(*Map); ok {
			if val, ok := m.Pairs[idx.Inspect()]; ok {
				return val
			}
		}
		return NIL

	case *ast.DotExpression:
		left := Eval(n.Left, env)
		if m, ok := left.(*Map); ok {
			if val, ok := m.Pairs[n.Property.Value]; ok {
				return val
			}
		} else if si, ok := left.(*StructInstance); ok {
			if val, ok := si.Fields[n.Property.Value]; ok {
				return val
			}
		}
		return NIL

	case *ast.PrefixExpression:
		right := Eval(n.Right, env)
		if n.Operator == "!" {
			if isTruthy(right) {
				return FALSE
			}
			return TRUE
		} else if n.Operator == "-" {
			if i, ok := right.(*Integer); ok {
				return &Integer{Value: -i.Value}
			} else if f, ok := right.(*Float); ok {
				return &Float{Value: -f.Value}
			}
		}
		return NIL

	case *ast.InfixExpression:
		left := Eval(n.Left, env)
		right := Eval(n.Right, env)
		return evalInfixExpression(n.Operator, left, right)

	case *ast.CallExpression:
		fnObj := Eval(n.Function, env)
		args := []Object{}
		for _, a := range n.Arguments {
			args = append(args, Eval(a, env))
		}
		return applyFunction(fnObj, args, env)
	}

	return NIL
}

func applyFunction(fn Object, args []Object, outerEnv *Environment) Object {
	if b, ok := fn.(*Builtin); ok {
		return b.Fn(args...)
	}

	if def, ok := fn.(*StructDef); ok {
		instance := &StructInstance{Def: def, Fields: make(map[string]Object)}
		if len(args) == 1 {
			if m, ok := args[0].(*Map); ok {
				for k, v := range m.Pairs {
					instance.Fields[k] = v
				}
			}
		}
		return instance
	}

	if f, ok := fn.(*Function); ok {
		fnEnv := NewEnclosedEnvironment(f.Env)
		for i, param := range f.Parameters {
			if i < len(args) {
				fnEnv.Set(param.Value, args[i])
			}
		}

		res := Eval(f.Body, fnEnv)

		// Eksekusi tunda (defer) LIFO
		for i := len(fnEnv.deferStack) - 1; i >= 0; i-- {
			fnEnv.deferStack[i]()
		}

		if ret, ok := res.(*ReturnValue); ok {
			return ret.Value
		}
		return NIL
	}

	return NIL
}

func isTruthy(obj Object) bool {
	if obj == nil || obj == NIL || obj == FALSE {
		return false
	}
	if i, ok := obj.(*Integer); ok && i.Value == 0 {
		return false
	}
	if s, ok := obj.(*String); ok && s.Value == "" {
		return false
	}
	return true
}

func objectsEqual(a, b Object) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Type() != b.Type() {
		return false
	}
	switch av := a.(type) {
	case *Integer:
		return av.Value == b.(*Integer).Value
	case *String:
		return av.Value == b.(*String).Value
	case *Boolean:
		return av.Value == b.(*Boolean).Value
	}
	return false
}

func evalInfixExpression(op string, left, right Object) Object {
	if left.Type() == INTEGER_OBJ && right.Type() == INTEGER_OBJ {
		lVal := left.(*Integer).Value
		rVal := right.(*Integer).Value
		switch op {
		case "+":
			return &Integer{Value: lVal + rVal}
		case "-":
			return &Integer{Value: lVal - rVal}
		case "*":
			return &Integer{Value: lVal * rVal}
		case "/":
			if rVal == 0 {
				return &Integer{Value: 0}
			}
			return &Integer{Value: lVal / rVal}
		case "%":
			return &Integer{Value: lVal % rVal}
		case "==":
			if lVal == rVal {
				return TRUE
			}
			return FALSE
		case "!=":
			if lVal != rVal {
				return TRUE
			}
			return FALSE
		case "<":
			if lVal < rVal {
				return TRUE
			}
			return FALSE
		case "<=":
			if lVal <= rVal {
				return TRUE
			}
			return FALSE
		case ">":
			if lVal > rVal {
				return TRUE
			}
			return FALSE
		case ">=":
			if lVal >= rVal {
				return TRUE
			}
			return FALSE
		}
	}

	if left.Type() == STRING_OBJ && right.Type() == STRING_OBJ {
		lVal := left.(*String).Value
		rVal := right.(*String).Value
		if op == "+" {
			return &String{Value: lVal + rVal}
		} else if op == "==" {
			if lVal == rVal {
				return TRUE
			}
			return FALSE
		} else if op == "!=" {
			if lVal != rVal {
				return TRUE
			}
			return FALSE
		}
	}

	if op == "==" {
		if objectsEqual(left, right) {
			return TRUE
		}
		return FALSE
	}
	if op == "!=" {
		if !objectsEqual(left, right) {
			return TRUE
		}
		return FALSE
	}
	if op == "&&" {
		if isTruthy(left) && isTruthy(right) {
			return TRUE
		}
		return FALSE
	}
	if op == "||" {
		if isTruthy(left) || isTruthy(right) {
			return TRUE
		}
		return FALSE
	}

	return NIL
}

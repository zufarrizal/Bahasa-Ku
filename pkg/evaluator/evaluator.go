package evaluator

import (
	"bufio"
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
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	BREAK_OBJ        = "BREAK"
	CONTINUE_OBJ     = "CONTINUE"
	PANIC_OBJ        = "PANIC"
	CHANNEL_OBJ      = "CHANNEL"
)

type ChannelObject struct {
	Ch chan Object
}

func (c *ChannelObject) Type() ObjectType { return CHANNEL_OBJ }
func (c *ChannelObject) Inspect() string  { return "kanal" }

type PanicObject struct {
	Value Object
}

func (p *PanicObject) Type() ObjectType { return PANIC_OBJ }
func (p *PanicObject) Inspect() string  { return fmt.Sprintf("panik: %s", p.Value.Inspect()) }

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

type Break struct{}

func (b *Break) Type() ObjectType { return BREAK_OBJ }
func (b *Break) Inspect() string  { return "henti" }

type Continue struct{}

func (c *Continue) Type() ObjectType { return CONTINUE_OBJ }
func (c *Continue) Inspect() string  { return "lanjut" }

type ReturnValue struct {
	Value  Object
	Values []Object
}

func (r *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (r *ReturnValue) Inspect() string {
	if len(r.Values) > 1 {
		elems := []string{}
		for _, v := range r.Values {
			elems = append(elems, v.Inspect())
		}
		return strings.Join(elems, ", ")
	}
	return r.Value.Inspect()
}

type Environment struct {
	store      map[string]Object
	outer      *Environment
	deferStack []func()
	iotaVal    int64
	panicVal   Object
}

func NewEnvironment() *Environment {
	return &Environment{store: make(map[string]Object), deferStack: []func(){}, iotaVal: 0, panicVal: nil}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	if outer != nil {
		env.panicVal = outer.panicVal
	}
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

var globalPanic Object = nil

type SubFileEvaluator func(path string, env *Environment)

var subFileRunner SubFileEvaluator

func RegisterSubFileEvaluator(sfe SubFileEvaluator) {
	subFileRunner = sfe
}

func SetGlobalPanic(p Object) {
	globalPanic = p
}

func ClearGlobalPanic() Object {
	val := globalPanic
	globalPanic = nil
	return val
}

func GetGlobalPanic() Object {
	return globalPanic
}

type BuiltinFunction func(args ...Object) Object

type Builtin struct{ Fn BuiltinFunction }

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "fungsi bawaan" }

type Function struct {
	Receiver   *ast.FunctionReceiver
	Parameters []*ast.Identifier
	IsVariadic bool
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
	Name    string
	Fields  []ast.StructField
	Methods map[string]*Function
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
	berkasMap.Pairs["buatDirektori"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			err := os.MkdirAll(args[0].Inspect(), 0755)
			return &Boolean{Value: err == nil}
		},
	}
	berkasMap.Pairs["daftar"] = &Builtin{
		Fn: func(args ...Object) Object {
			dir := "."
			if len(args) > 0 {
				dir = args[0].Inspect()
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				return &Array{Elements: []Object{}}
			}
			res := []Object{}
			for _, e := range entries {
				res = append(res, &String{Value: e.Name()})
			}
			return &Array{Elements: res}
		},
	}
	berkasMap.Pairs["ukuran"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Integer{Value: 0}
			}
			fi, err := os.Stat(args[0].Inspect())
			if err != nil {
				return &Integer{Value: 0}
			}
			return &Integer{Value: fi.Size()}
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
	sistemMap.Pairs["tidur"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return NIL
			}
			sec := int64(1)
			if i, ok := args[0].(*Integer); ok {
				sec = i.Value
			} else if f, ok := args[0].(*Float); ok {
				time.Sleep(time.Duration(f.Value * float64(time.Second)))
				return NIL
			}
			time.Sleep(time.Duration(sec) * time.Second)
			return NIL
		},
	}
	sistemMap.Pairs["kelompokTunggu"] = &Builtin{
		Fn: func(args ...Object) Object {
			var wg sync.WaitGroup
			ktMap := &Map{Pairs: make(map[string]Object)}
			ktMap.Pairs["tambah"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					delta := 1
					if len(mArgs) > 0 {
						if i, ok := mArgs[0].(*Integer); ok {
							delta = int(i.Value)
						}
					}
					wg.Add(delta)
					return NIL
				},
			}
			ktMap.Pairs["selesai"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					wg.Done()
					return NIL
				},
			}
			ktMap.Pairs["tunggu"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					wg.Wait()
					return NIL
				},
			}
			return ktMap
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
	matematikaMap.Pairs["sin"] = &Builtin{
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
			return &Float{Value: math.Sin(val)}
		},
	}
	matematikaMap.Pairs["cos"] = &Builtin{
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
			return &Float{Value: math.Cos(val)}
		},
	}
	matematikaMap.Pairs["tan"] = &Builtin{
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
			return &Float{Value: math.Tan(val)}
		},
	}
	matematikaMap.Pairs["rataRata"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Float{Value: 0}
			}
			arr, isArr := args[0].(*Array)
			if !isArr || len(arr.Elements) == 0 {
				return &Float{Value: 0}
			}
			total := float64(0)
			for _, e := range arr.Elements {
				if i, ok := e.(*Integer); ok {
					total += float64(i.Value)
				} else if f, ok := e.(*Float); ok {
					total += f.Value
				}
			}
			return &Float{Value: total / float64(len(arr.Elements))}
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
	teksMap.Pairs["format"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) == 0 {
				return &String{Value: ""}
			}
			template := args[0].Inspect()
			for i := 1; i < len(args); i++ {
				placeholder := fmt.Sprintf("{%d}", i-1)
				template = strings.ReplaceAll(template, placeholder, args[i].Inspect())
			}
			return &String{Value: template}
		},
	}
	teksMap.Pairs["ulang"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &String{Value: ""}
			}
			s := args[0].Inspect()
			count := int(0)
			if i, ok := args[1].(*Integer); ok {
				count = int(i.Value)
			}
			if count < 0 {
				count = 0
			}
			return &String{Value: strings.Repeat(s, count)}
		},
	}
	teksMap.Pairs["awalan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			return &Boolean{Value: strings.HasPrefix(args[0].Inspect(), args[1].Inspect())}
		},
	}
	teksMap.Pairs["akhiran"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			return &Boolean{Value: strings.HasSuffix(args[0].Inspect(), args[1].Inspect())}
		},
	}
	teksMap.Pairs["indeks"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Integer{Value: -1}
			}
			return &Integer{Value: int64(strings.Index(args[0].Inspect(), args[1].Inspect()))}
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

	// Modul masukan (Terminal Input / Scanner)
	masukanMap := &Map{Pairs: make(map[string]Object)}
	masukanMap.Pairs["baca"] = &Builtin{
		Fn: func(args ...Object) Object {
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				return &String{Value: scanner.Text()}
			}
			return &String{Value: ""}
		},
	}
	masukanMap.Pairs["tanya"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) >= 1 {
				fmt.Print(args[0].Inspect())
			}
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				return &String{Value: scanner.Text()}
			}
			return &String{Value: ""}
		},
	}
	env.Set("masukan", masukanMap)

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

	// Bawaan: hapus (Go delete pada Map)
	env.Set("hapus", &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			if m, ok := args[0].(*Map); ok {
				keyStr := args[1].Inspect()
				delete(m.Pairs, keyStr)
				return TRUE
			}
			return FALSE
		},
	})

	// Bawaan: panik (panic ala Go)
	env.Set("panik", &Builtin{
		Fn: func(args ...Object) Object {
			var pVal Object = &String{Value: "panik terjadi"}
			if len(args) > 0 {
				pVal = args[0]
			}
			SetGlobalPanic(pVal)
			return &PanicObject{Value: pVal}
		},
	})

	// Bawaan: pulih (recover ala Go)
	env.Set("pulih", &Builtin{
		Fn: func(args ...Object) Object {
			val := ClearGlobalPanic()
			if val != nil {
				return val
			}
			return NIL
		},
	})

	// Modul kanal (Go Channel)
	kanalMap := &Map{Pairs: make(map[string]Object)}
	kanalMap.Pairs["buat"] = &Builtin{
		Fn: func(args ...Object) Object {
			capVal := int64(0)
			if len(args) > 0 {
				if i, ok := args[0].(*Integer); ok && i.Value > 0 {
					capVal = i.Value
				}
			}
			ch := make(chan Object, capVal)
			chInst := &Map{Pairs: make(map[string]Object)}
			chInst.Pairs["kirim"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					if len(mArgs) > 0 {
						ch <- mArgs[0]
					} else {
						ch <- NIL
					}
					return TRUE
				},
			}
			chInst.Pairs["terima"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					val := <-ch
					if val == nil {
						return NIL
					}
					return val
				},
			}
			chInst.Pairs["terimaDenganWaktu"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					timeoutSec := float64(1)
					if len(mArgs) > 0 {
						if i, ok := mArgs[0].(*Integer); ok {
							timeoutSec = float64(i.Value)
						} else if f, ok := mArgs[0].(*Float); ok {
							timeoutSec = f.Value
						}
					}
					select {
					case val := <-ch:
						if val == nil {
							return NIL
						}
						return val
					case <-time.After(time.Duration(timeoutSec * float64(time.Second))):
						return NIL
					}
				},
			}
			chInst.Pairs["cobaTerima"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					select {
					case val := <-ch:
						if val == nil {
							val = NIL
						}
						return &ReturnValue{Value: val, Values: []Object{val, TRUE}}
					default:
						return &ReturnValue{Value: NIL, Values: []Object{NIL, FALSE}}
					}
				},
			}
			chInst.Pairs["tutup"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					close(ch)
					return TRUE
				},
			}
			return chInst
		},
	}
	env.Set("kanal", kanalMap)

	// Modul json (Serialisasi & Deserialisasi JSON)
	jsonMap := &Map{Pairs: make(map[string]Object)}
	jsonMap.Pairs["kodekan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) == 0 {
				return &String{Value: "null"}
			}
			goObj := convertKuToGoObject(args[0])
			b, err := json.Marshal(goObj)
			if err != nil {
				return &String{Value: "{}"}
			}
			return &String{Value: string(b)}
		},
	}
	jsonMap.Pairs["kodekanRapi"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) == 0 {
				return &String{Value: "null"}
			}
			goObj := convertKuToGoObject(args[0])
			b, err := json.MarshalIndent(goObj, "", "  ")
			if err != nil {
				return &String{Value: "{}"}
			}
			return &String{Value: string(b)}
		},
	}
	jsonMap.Pairs["urai"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) == 0 {
				return NIL
			}
			jsonStr := strings.TrimSpace(args[0].Inspect())
			var raw interface{}
			err := json.Unmarshal([]byte(jsonStr), &raw)
			if err != nil {
				return NIL
			}
			return convertGoToKuObject(raw)
		},
	}
	env.Set("json", jsonMap)

	// Modul larik (Operasi Array Fungsional)
	larikMap := &Map{Pairs: make(map[string]Object)}
	larikMap.Pairs["saring"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Array{Elements: []Object{}}
			}
			arr, isArr := args[0].(*Array)
			fn, isFn := args[1].(*Function)
			if !isArr || !isFn {
				return args[0]
			}
			res := []Object{}
			for _, elem := range arr.Elements {
				if isTruthy(applyFunction(fn, []Object{elem}, env)) {
					res = append(res, elem)
				}
			}
			return &Array{Elements: res}
		},
	}
	larikMap.Pairs["petakan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Array{Elements: []Object{}}
			}
			arr, isArr := args[0].(*Array)
			fn, isFn := args[1].(*Function)
			if !isArr || !isFn {
				return args[0]
			}
			res := []Object{}
			for _, elem := range arr.Elements {
				res = append(res, applyFunction(fn, []Object{elem}, env))
			}
			return &Array{Elements: res}
		},
	}
	larikMap.Pairs["reduksi"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			arr, isArr := args[0].(*Array)
			fn, isFn := args[1].(*Function)
			if !isArr || !isFn {
				return NIL
			}
			var acc Object = NIL
			startIdx := 0
			if len(args) >= 3 {
				acc = args[2]
			} else if len(arr.Elements) > 0 {
				acc = arr.Elements[0]
				startIdx = 1
			}
			for i := startIdx; i < len(arr.Elements); i++ {
				acc = applyFunction(fn, []Object{acc, arr.Elements[i]}, env)
			}
			return acc
		},
	}
	larikMap.Pairs["temukan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			arr, isArr := args[0].(*Array)
			fn, isFn := args[1].(*Function)
			if !isArr || !isFn {
				return NIL
			}
			for _, elem := range arr.Elements {
				if isTruthy(applyFunction(fn, []Object{elem}, env)) {
					return elem
				}
			}
			return NIL
		},
	}
	larikMap.Pairs["urut"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Array{Elements: []Object{}}
			}
			arr, isArr := args[0].(*Array)
			if !isArr {
				return args[0]
			}
			res := append([]Object{}, arr.Elements...)
			sort.Slice(res, func(i, j int) bool {
				if i1, ok1 := res[i].(*Integer); ok1 {
					if i2, ok2 := res[j].(*Integer); ok2 {
						return i1.Value < i2.Value
					}
				}
				if f1, ok1 := res[i].(*Float); ok1 {
					if f2, ok2 := res[j].(*Float); ok2 {
						return f1.Value < f2.Value
					}
				}
				return res[i].Inspect() < res[j].Inspect()
			})
			return &Array{Elements: res}
		},
	}
	larikMap.Pairs["balik"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Array{Elements: []Object{}}
			}
			arr, isArr := args[0].(*Array)
			if !isArr {
				return args[0]
			}
			res := make([]Object, len(arr.Elements))
			for i, e := range arr.Elements {
				res[len(arr.Elements)-1-i] = e
			}
			return &Array{Elements: res}
		},
	}
	larikMap.Pairs["acak"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Array{Elements: []Object{}}
			}
			arr, isArr := args[0].(*Array)
			if !isArr {
				return args[0]
			}
			res := append([]Object{}, arr.Elements...)
			mrand.Shuffle(len(res), func(i, j int) {
				res[i], res[j] = res[j], res[i]
			})
			return &Array{Elements: res}
		},
	}
	env.Set("larik", larikMap)

	// Modul uji (Unit Testing Internal)
	ujiMap := &Map{Pairs: make(map[string]Object)}
	ujiMap.Pairs["sama"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				fmt.Println("❌ [UJI GAGAL] uji.sama membutuhkan minimal 2 argumen")
				return FALSE
			}
			if objectsEqual(args[0], args[1]) {
				fmt.Printf("✅ [UJI LULUS] Nilai sesuai: %s\n", args[0].Inspect())
				return TRUE
			}
			fmt.Printf("❌ [UJI GAGAL] Diharapkan %s, tetapi didapat %s\n", args[1].Inspect(), args[0].Inspect())
			return FALSE
		},
	}
	ujiMap.Pairs["benar"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) > 0 && isTruthy(args[0]) {
				fmt.Println("✅ [UJI LULUS] Kondisi benar")
				return TRUE
			}
			fmt.Println("❌ [UJI GAGAL] Kondisi tidak benar (salah/kosong)")
			return FALSE
		},
	}
	ujiMap.Pairs["gagal"] = &Builtin{
		Fn: func(args ...Object) Object {
			pesan := "uji gagal"
			if len(args) > 0 {
				pesan = args[0].Inspect()
			}
			fmt.Printf("❌ [UJI GAGAL] %s\n", pesan)
			return FALSE
		},
	}
	env.Set("uji", ujiMap)

	// Modul waktu (Pengelolaan Tanggal dan Durasi Waktu)
	waktuMap := &Map{Pairs: make(map[string]Object)}
	waktuMap.Pairs["sekarang"] = &Builtin{
		Fn: func(args ...Object) Object {
			now := time.Now()
			res := &Map{Pairs: make(map[string]Object)}
			res.Pairs["tahun"] = &Integer{Value: int64(now.Year())}
			res.Pairs["bulan"] = &Integer{Value: int64(now.Month())}
			res.Pairs["hari"] = &Integer{Value: int64(now.Day())}
			res.Pairs["jam"] = &Integer{Value: int64(now.Hour())}
			res.Pairs["menit"] = &Integer{Value: int64(now.Minute())}
			res.Pairs["detik"] = &Integer{Value: int64(now.Second())}
			res.Pairs["teks"] = &String{Value: now.Format("2006-01-02 15:04:05")}
			res.Pairs["detikUnix"] = &Integer{Value: now.Unix()}
			return res
		},
	}
	waktuMap.Pairs["detikUnix"] = &Builtin{
		Fn: func(args ...Object) Object {
			return &Integer{Value: time.Now().Unix()}
		},
	}
	waktuMap.Pairs["miliDetikUnix"] = &Builtin{
		Fn: func(args ...Object) Object {
			return &Integer{Value: time.Now().UnixMilli()}
		},
	}
	waktuMap.Pairs["format"] = &Builtin{
		Fn: func(args ...Object) Object {
			// waktu.format(formatStr) atau waktu.format(detikUnix, formatStr)
			layout := "2006-01-02 15:04:05"
			t := time.Now()
			if len(args) == 1 {
				if s, ok := args[0].(*String); ok {
					layout = s.Value
				}
			} else if len(args) >= 2 {
				if sec, ok := args[0].(*Integer); ok {
					t = time.Unix(sec.Value, 0)
				}
				if s, ok := args[1].(*String); ok {
					layout = s.Value
				}
			}
			return &String{Value: t.Format(layout)}
		},
	}
	waktuMap.Pairs["tambahDetik"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			var secVal int64
			var tambah int64
			if s, ok := args[0].(*Integer); ok {
				secVal = s.Value
			}
			if t, ok := args[1].(*Integer); ok {
				tambah = t.Value
			}
			return &Integer{Value: secVal + tambah}
		},
	}
	env.Set("waktu", waktuMap)

	// Modul pola (Ekspresi Reguler / Regex)
	polaMap := &Map{Pairs: make(map[string]Object)}
	polaMap.Pairs["cocok"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			polaStr, ok1 := args[0].(*String)
			teksStr, ok2 := args[1].(*String)
			if !ok1 || !ok2 {
				return FALSE
			}
			matched, err := regexp.MatchString(polaStr.Value, teksStr.Value)
			if err != nil || !matched {
				return FALSE
			}
			return TRUE
		},
	}
	polaMap.Pairs["temukanSemua"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Array{Elements: []Object{}}
			}
			polaStr, ok1 := args[0].(*String)
			teksStr, ok2 := args[1].(*String)
			if !ok1 || !ok2 {
				return &Array{Elements: []Object{}}
			}
			re, err := regexp.Compile(polaStr.Value)
			if err != nil {
				return &Array{Elements: []Object{}}
			}
			matches := re.FindAllString(teksStr.Value, -1)
			res := make([]Object, len(matches))
			for i, m := range matches {
				res[i] = &String{Value: m}
			}
			return &Array{Elements: res}
		},
	}
	polaMap.Pairs["ganti"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 3 {
				return NIL
			}
			polaStr, ok1 := args[0].(*String)
			teksStr, ok2 := args[1].(*String)
			penggantiStr, ok3 := args[2].(*String)
			if !ok1 || !ok2 || !ok3 {
				return NIL
			}
			re, err := regexp.Compile(polaStr.Value)
			if err != nil {
				return args[1]
			}
			res := re.ReplaceAllString(teksStr.Value, penggantiStr.Value)
			return &String{Value: res}
		},
	}
	env.Set("pola", polaMap)

	return env
}

func convertKuToGoObject(obj Object) interface{} {
	if obj == nil || obj == NIL {
		return nil
	}
	switch o := obj.(type) {
	case *Integer:
		return o.Value
	case *Float:
		return o.Value
	case *Boolean:
		return o.Value
	case *String:
		return o.Value
	case *Array:
		res := []interface{}{}
		for _, e := range o.Elements {
			res = append(res, convertKuToGoObject(e))
		}
		return res
	case *Map:
		res := make(map[string]interface{})
		for k, v := range o.Pairs {
			res[k] = convertKuToGoObject(v)
		}
		return res
	case *StructInstance:
		res := make(map[string]interface{})
		for k, v := range o.Fields {
			res[k] = convertKuToGoObject(v)
		}
		return res
	default:
		return o.Inspect()
	}
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
			if p, ok := result.(*PanicObject); ok {
				return p
			}
			if ret, ok := result.(*ReturnValue); ok {
				return ret.Value
			}
		}
		if mainFn, ok := env.Get("utama"); ok {
			if fn, ok := mainFn.(*Function); ok {
				res := applyFunction(fn, []Object{}, env)
				if p, ok := res.(*PanicObject); ok {
					fmt.Println("💥 Galat Panik Tidak Tertangani:", p.Value.Inspect())
				}
				return res
			}
		}
		return result

	case *ast.PaketStatement:
		return NIL

	case *ast.PakaiStatement:
		// Jika mengimpor berkas .ku lokal, misal: pakai "./helper.ku" atau pakai "util.ku"
		if strings.HasSuffix(n.Path, ".ku") && subFileRunner != nil {
			subFileRunner(n.Path, env)
		}
		return NIL

	case *ast.TypeStructStatement:
		def := &StructDef{Name: n.Name.Value, Fields: n.Fields, Methods: make(map[string]*Function)}
		env.Set(n.Name.Value, def)
		return def

	case *ast.VarStatement:
		// Jika deklarasi banyak variabel: var a, b = fn() atau var a, b = 1, 2
		if len(n.Names) > 1 {
			evaluatedVals := []Object{}
			for _, vExpr := range n.Values {
				res := Eval(vExpr, env)
				if ret, ok := res.(*ReturnValue); ok && len(ret.Values) > 0 {
					evaluatedVals = append(evaluatedVals, ret.Values...)
				} else {
					evaluatedVals = append(evaluatedVals, res)
				}
			}

			for i, ident := range n.Names {
				if ident.Value == "_" {
					continue
				}
				var val Object = NIL
				if i < len(evaluatedVals) {
					val = evaluatedVals[i]
				}
				env.Set(ident.Value, val)
			}
			return NIL
		}

		var val Object = NIL
		if n.Value != nil {
			res := Eval(n.Value, env)
			if ret, ok := res.(*ReturnValue); ok {
				val = ret.Value
			} else {
				val = res
			}
		}
		if n.Name.Value != "_" {
			env.Set(n.Name.Value, val)
		}
		return val

	case *ast.AssignStatement:
		// Jika multi-target assignment: a, b := fn() atau a, b = 1, 2
		if len(n.Targets) > 1 {
			evaluatedVals := []Object{}
			for _, vExpr := range n.Values {
				res := Eval(vExpr, env)
				if ret, ok := res.(*ReturnValue); ok && len(ret.Values) > 0 {
					evaluatedVals = append(evaluatedVals, ret.Values...)
				} else {
					evaluatedVals = append(evaluatedVals, res)
				}
			}

			for i, target := range n.Targets {
				var val Object = NIL
				if i < len(evaluatedVals) {
					val = evaluatedVals[i]
				}

				if ident, ok := target.(*ast.Identifier); ok {
					if ident.Value == "_" {
						continue
					}
					if n.IsDefine {
						env.Set(ident.Value, val)
					} else {
						if !env.Update(ident.Value, val) {
							env.Set(ident.Value, val)
						}
					}
				}
			}
			return NIL
		}

		val := Eval(n.Value, env)
		if ret, ok := val.(*ReturnValue); ok {
			val = ret.Value
		}

		if ident, ok := n.Target.(*ast.Identifier); ok {
			if ident.Value == "_" {
				return NIL
			}
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
		fn := &Function{
			Receiver:   n.Receiver,
			Parameters: n.Parameters,
			IsVariadic: n.IsVariadic,
			Body:       n.Body,
			Env:        env,
		}
		// Jika merupakan method receiver: fungsi (p Pengguna) sapa() { ... }
		if n.Receiver != nil {
			if typeObj, ok := env.Get(n.Receiver.Type); ok {
				if def, isStruct := typeObj.(*StructDef); isStruct {
					def.Methods[n.Name.Value] = fn
					return fn
				}
			}
		}
		env.Set(n.Name.Value, fn)
		return fn

	case *ast.FunctionLiteral:
		return &Function{
			Parameters: n.Parameters,
			IsVariadic: n.IsVariadic,
			Body:       n.Body,
			Env:        env,
		}

	case *ast.BreakStatement:
		return &Break{}

	case *ast.ContinueStatement:
		return &Continue{}

	case *ast.BlockStatement:
		var result Object = NIL
		for _, stmt := range n.Statements {
			result = Eval(stmt, env)
			if result != nil {
				rt := result.Type()
				if rt == RETURN_VALUE_OBJ || rt == BREAK_OBJ || rt == CONTINUE_OBJ || rt == PANIC_OBJ {
					return result
				}
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
					if res != nil {
						if res.Type() == RETURN_VALUE_OBJ {
							return res
						}
						if res.Type() == BREAK_OBJ {
							break
						}
						if res.Type() == CONTINUE_OBJ {
							continue
						}
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
					if res != nil {
						if res.Type() == RETURN_VALUE_OBJ {
							return res
						}
						if res.Type() == BREAK_OBJ {
							break
						}
						if res.Type() == CONTINUE_OBJ {
							continue
						}
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
			if result != nil {
				if result.Type() == RETURN_VALUE_OBJ {
					return result
				}
				if result.Type() == BREAK_OBJ {
					break
				}
				if result.Type() == CONTINUE_OBJ {
					if n.Post != nil {
						Eval(n.Post, loopEnv)
					}
					continue
				}
			}
			if n.Post != nil {
				Eval(n.Post, loopEnv)
			}
		}
		return NIL

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
		if len(n.ReturnValues) > 1 {
			vals := []Object{}
			for _, vExpr := range n.ReturnValues {
				vals = append(vals, Eval(vExpr, env))
			}
			return &ReturnValue{Value: vals[0], Values: vals}
		}
		var val Object = NIL
		if n.ReturnValue != nil {
			val = Eval(n.ReturnValue, env)
		}
		return &ReturnValue{Value: val, Values: []Object{val}}

	case *ast.DeferStatement:
		deferredCall := func() {
			Eval(n.Expression, env)
		}
		env.deferStack = append(env.deferStack, deferredCall)
		return NIL

	case *ast.KuStatement:
		go func() {
			kEnv := NewEnclosedEnvironment(env)
			Eval(n.Expression, kEnv)
		}()
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

	case *ast.CacahLiteral:
		val := env.iotaVal
		env.iotaVal++
		return &Integer{Value: val}

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
		if n.IsSlice {
			lowVal := int64(0)
			if n.Index != nil {
				if lObj, ok := Eval(n.Index, env).(*Integer); ok {
					lowVal = lObj.Value
				}
			}

			if arr, ok := left.(*Array); ok {
				highVal := int64(len(arr.Elements))
				if n.High != nil {
					if hObj, ok := Eval(n.High, env).(*Integer); ok {
						highVal = hObj.Value
					}
				}
				if lowVal < 0 {
					lowVal = 0
				}
				if highVal > int64(len(arr.Elements)) {
					highVal = int64(len(arr.Elements))
				}
				if lowVal > highVal {
					return &Array{Elements: []Object{}}
				}
				sub := arr.Elements[lowVal:highVal]
				newSub := append([]Object{}, sub...)
				return &Array{Elements: newSub}
			} else if str, ok := left.(*String); ok {
				highVal := int64(len(str.Value))
				if n.High != nil {
					if hObj, ok := Eval(n.High, env).(*Integer); ok {
						highVal = hObj.Value
					}
				}
				if lowVal < 0 {
					lowVal = 0
				}
				if highVal > int64(len(str.Value)) {
					highVal = int64(len(str.Value))
				}
				if lowVal > highVal {
					return &String{Value: ""}
				}
				return &String{Value: str.Value[lowVal:highVal]}
			}
			return NIL
		}

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
			// Cek apakah ada method terikat pada struct instance
			if si.Def != nil && si.Def.Methods != nil {
				if method, hasMethod := si.Def.Methods[n.Property.Value]; hasMethod {
					// Bungkus method dengan receiver instance ini
					boundFn := &Function{
						Receiver:   method.Receiver,
						Parameters: method.Parameters,
						IsVariadic: method.IsVariadic,
						Body:       method.Body,
						Env:        NewEnclosedEnvironment(method.Env),
					}
					if method.Receiver != nil {
						boundFn.Env.Set(method.Receiver.Name, si)
					}
					return boundFn
				}
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
		if f.Receiver != nil {
			if recObj, exists := f.Env.Get(f.Receiver.Name); exists {
				fnEnv.Set(f.Receiver.Name, recObj)
			}
		}

		if f.IsVariadic && len(f.Parameters) > 0 {
			regularParamCount := len(f.Parameters) - 1
			for i := 0; i < regularParamCount; i++ {
				if i < len(args) {
					fnEnv.Set(f.Parameters[i].Value, args[i])
				}
			}
			var rest []Object
			if len(args) > regularParamCount {
				rest = args[regularParamCount:]
			} else {
				rest = []Object{}
			}
			fnEnv.Set(f.Parameters[regularParamCount].Value, &Array{Elements: rest})
		} else {
			for i, param := range f.Parameters {
				if i < len(args) {
					fnEnv.Set(param.Value, args[i])
				}
			}
		}

		res := Eval(f.Body, fnEnv)

		if pObj, isPanic := res.(*PanicObject); isPanic {
			SetGlobalPanic(pObj.Value)
		}

		// Eksekusi tunda (defer) LIFO
		for i := len(fnEnv.deferStack) - 1; i >= 0; i-- {
			fnEnv.deferStack[i]()
		}

		// Jika panik telah dipulihkan (pulih())
		if GetGlobalPanic() == nil {
			if res != nil && res.Type() == PANIC_OBJ {
				return NIL
			}
		}

		if activePanic := GetGlobalPanic(); activePanic != nil {
			return &PanicObject{Value: activePanic}
		}

		if ret, ok := res.(*ReturnValue); ok {
			if len(ret.Values) > 1 {
				return ret
			}
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
	case *Float:
		return av.Value == b.(*Float).Value
	case *String:
		return av.Value == b.(*String).Value
	case *Boolean:
		return av.Value == b.(*Boolean).Value
	case *Array:
		bv := b.(*Array)
		if len(av.Elements) != len(bv.Elements) {
			return false
		}
		for i := range av.Elements {
			if !objectsEqual(av.Elements[i], bv.Elements[i]) {
				return false
			}
		}
		return true
	case *Map:
		bv := b.(*Map)
		if len(av.Pairs) != len(bv.Pairs) {
			return false
		}
		for k, v := range av.Pairs {
			if bvVal, exists := bv.Pairs[k]; !exists || !objectsEqual(v, bvVal) {
				return false
			}
		}
		return true
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
		case "&":
			return &Integer{Value: lVal & rVal}
		case "|":
			return &Integer{Value: lVal | rVal}
		case "^":
			return &Integer{Value: lVal ^ rVal}
		case "<<":
			return &Integer{Value: lVal << uint64(rVal)}
		case ">>":
			return &Integer{Value: lVal >> uint64(rVal)}
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
	if op == "??" {
		if left != nil && left != NIL {
			if s, isStr := left.(*String); !isStr || s.Value != "" {
				return left
			}
		}
		return right
	}

	return NIL
}

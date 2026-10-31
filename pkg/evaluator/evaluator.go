package evaluator

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	mrand "math/rand"
	"net"
	"net/http"
	"net/url"
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

	_ "github.com/go-sql-driver/mysql"
	"github.com/zufarrizal/bahasa-ku/pkg/ast"
	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/parser"
	"github.com/zufarrizal/bahasa-ku/pkg/token"
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
	FALLTHROUGH_OBJ  = "FALLTHROUGH"
	PANIC_OBJ        = "PANIC"
	CHANNEL_OBJ      = "CHANNEL"
	INTERFACE_DEF_OBJ = "INTERFACE_DEF"
)

type ChannelObject struct {
	Ch chan Object
}

func (c *ChannelObject) Type() ObjectType { return CHANNEL_OBJ }
func (c *ChannelObject) Inspect() string  { return "kanal" }

type CallFrame struct {
	Function string
	File     string
	Line     int
	Column   int
}

type PanicObject struct {
	Value      Object
	StackTrace []CallFrame
}

func (p *PanicObject) Type() ObjectType { return PANIC_OBJ }
func (p *PanicObject) Inspect() string {
	msg := fmt.Sprintf("panik: %s", p.Value.Inspect())
	if len(p.StackTrace) > 0 {
		var sb strings.Builder
		sb.WriteString(msg)
		sb.WriteString("\nJejak Pemanggilan (Stack Trace):")
		for i, frame := range p.StackTrace {
			fnName := frame.Function
			if fnName == "" {
				fnName = "<anonim>"
			}
			fileInfo := frame.File
			if fileInfo == "" {
				fileInfo = "berkas"
			}
			sb.WriteString(fmt.Sprintf("\n  [%d] %s() pada %s:%d:%d", i+1, fnName, fileInfo, frame.Line, frame.Column))
		}
		return sb.String()
	}
	return msg
}

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

type Break struct {
	Label string
}

func (b *Break) Type() ObjectType { return BREAK_OBJ }
func (b *Break) Inspect() string  {
	if b.Label != "" {
		return "henti " + b.Label
	}
	return "henti"
}

type Continue struct {
	Label string
}

func (c *Continue) Type() ObjectType { return CONTINUE_OBJ }
func (c *Continue) Inspect() string  {
	if c.Label != "" {
		return "lanjut " + c.Label
	}
	return "lanjut"
}

type Fallthrough struct{}

func (f *Fallthrough) Type() ObjectType { return FALLTHROUGH_OBJ }
func (f *Fallthrough) Inspect() string  { return "teruskan" }

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

func (e *Environment) GetStore() map[string]Object {
	return e.store
}

var globalPanic Object = nil

type SubFileEvaluator func(path string, env *Environment)

var subFileRunner SubFileEvaluator

func RegisterSubFileEvaluator(sfe SubFileEvaluator) {
	subFileRunner = sfe
}

// ResolveModulePath menyelesaikan path berkas lokal maupun remote URL dengan caching otomatis
func ResolveModulePath(path string) (string, error) {
	// 1. Cek apakah ini Remote URL HTTP/HTTPS
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = os.TempDir()
		}
		cacheDir := filepath.Join(homeDir, ".bahasa-ku", "cache")
		_ = os.MkdirAll(cacheDir, 0755)

		hash := sha256.Sum256([]byte(path))
		hashStr := hex.EncodeToString(hash[:12])

		urlPath := path
		if idx := strings.Index(urlPath, "?"); idx != -1 {
			urlPath = urlPath[:idx]
		}
		baseName := filepath.Base(urlPath)
		if !strings.HasSuffix(baseName, ".ku") {
			baseName += ".ku"
		}

		cachedFile := filepath.Join(cacheDir, fmt.Sprintf("%s_%s", hashStr, baseName))

		// Jika belum ada di cache, unduh berkas
		if _, err := os.Stat(cachedFile); os.IsNotExist(err) {
			client := &http.Client{Timeout: 15 * time.Second}
			resp, err := client.Get(path)
			if err != nil {
				return "", fmt.Errorf("gagal mengunduh modul remote '%s': %v", path, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return "", fmt.Errorf("gagal mengunduh modul remote '%s' (status %d)", path, resp.StatusCode)
			}

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", fmt.Errorf("gagal membaca data modul remote '%s': %v", path, err)
			}

			if err := os.WriteFile(cachedFile, data, 0644); err != nil {
				return "", fmt.Errorf("gagal menyimpan cache modul '%s': %v", cachedFile, err)
			}
		}

		return cachedFile, nil
	}

	// 2. Berkas lokal
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
	return resolvedPath, nil
}

func init() {
	RegisterSubFileEvaluator(func(path string, env *Environment) {
		resolvedPath, err := ResolveModulePath(path)
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
			subEnv := NewEnclosedEnvironment(env)
			for _, stmt := range prog.Statements {
				if _, isPaket := stmt.(*ast.PaketStatement); !isPaket {
					Eval(stmt, subEnv)
				}
			}

			// Jalankan inisiasi() pada sub-file jika ada
			if initObj, ok := subEnv.Get("inisiasi"); ok {
				if fn, isFn := initObj.(*Function); isFn {
					Eval(fn.Body, subEnv)
				}
			}

			// Ekspor simbol: Huruf kapital terekspor ke paket pemanggil (aturan visibilitas Go)
			for name, val := range subEnv.GetStore() {
				if len(name) > 0 && unicode.IsUpper(rune(name[0])) {
					env.Set(name, val)
				}
			}
		}
	})
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
	Name         string
	Token        token.Token
	Receiver     *ast.FunctionReceiver
	Parameters   []*ast.Identifier
	ReturnParams []*ast.Identifier // Named return parameters
	IsVariadic   bool
	Body         *ast.BlockStatement
	Env          *Environment
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

type InterfaceDef struct {
	Name    string
	Methods []string
}

func (id *InterfaceDef) Type() ObjectType { return INTERFACE_DEF_OBJ }
func (id *InterfaceDef) Inspect() string  { return "definisi antarmuka " + id.Name }

func (id *InterfaceDef) ImplementedBy(obj Object) bool {
	if si, ok := obj.(*StructInstance); ok && si.Def != nil {
		for _, m := range id.Methods {
			if _, has := si.Def.Methods[m]; !has {
				return false
			}
		}
		return true
	}
	return false
}

var (
	TRUE  = &Boolean{Value: true}
	FALSE = &Boolean{Value: false}
	NIL   = &Nil{}
)

func getObjectTypeString(obj Object) string {
	if obj == nil {
		return "kosong"
	}
	switch v := obj.(type) {
	case *Integer:
		return "angka"
	case *Float:
		return "desimal"
	case *Boolean:
		return "boolean"
	case *String:
		return "teks"
	case *Array:
		return "larik"
	case *Map:
		return "peta"
	case *Nil:
		return "kosong"
	case *Function, *Builtin:
		return "fungsi"
	case *StructInstance:
		if v.Def != nil {
			return v.Def.Name
		}
		return "struktur"
	case *ChannelObject:
		return "kanal"
	default:
		return strings.ToLower(string(obj.Type()))
	}
}

func formatTipeIndo(obj Object) string {
	if obj == nil {
		return "kosong"
	}
	switch obj.(type) {
	case *Integer:
		return "angka"
	case *Float:
		return "desimal"
	case *Boolean:
		return "boolean"
	case *String:
		return "teks"
	case *Array:
		return "larik"
	case *Map:
		return "peta"
	case *Nil:
		return "kosong"
	case *Function:
		return "fungsi"
	case *Builtin:
		return "fungsi_bawaan"
	case *StructDef:
		return "definisi_struktur"
	case *StructInstance:
		return "struktur"
	case *ChannelObject:
		return "kanal"
	default:
		return string(obj.Type())
	}
}

func formatDumpIndo(obj Object, indent int) string {
	pad := strings.Repeat("  ", indent)
	if obj == nil {
		return pad + "kosong"
	}
	switch v := obj.(type) {
	case *Integer:
		return fmt.Sprintf("%sangka(%d)", pad, v.Value)
	case *Float:
		return fmt.Sprintf("%sdesimal(%f)", pad, v.Value)
	case *Boolean:
		return fmt.Sprintf("%sboolean(%s)", pad, v.Inspect())
	case *String:
		return fmt.Sprintf("%steks(%d) %q", pad, len(v.Value), v.Value)
	case *Nil:
		return pad + "kosong"
	case *Array:
		if len(v.Elements) == 0 {
			return pad + "larik(0) []"
		}
		var lines []string
		lines = append(lines, fmt.Sprintf("%slarik(%d) [", pad, len(v.Elements)))
		for i, el := range v.Elements {
			lines = append(lines, fmt.Sprintf("%s  [%d] =>\n%s", pad, i, formatDumpIndo(el, indent+2)))
		}
		lines = append(lines, pad+"]")
		return strings.Join(lines, "\n")
	case *Map:
		if len(v.Pairs) == 0 {
			return pad + "peta(0) {}"
		}
		var lines []string
		lines = append(lines, fmt.Sprintf("%speta(%d) {", pad, len(v.Pairs)))
		for k, val := range v.Pairs {
			lines = append(lines, fmt.Sprintf("%s  %q =>\n%s", pad, k, formatDumpIndo(val, indent+2)))
		}
		lines = append(lines, pad+"}")
		return strings.Join(lines, "\n")
	case *StructInstance:
		var lines []string
		lines = append(lines, fmt.Sprintf("%sstruktur %s {", pad, v.Def.Name))
		for k, val := range v.Fields {
			lines = append(lines, fmt.Sprintf("%s  %s =>\n%s", pad, k, formatDumpIndo(val, indent+2)))
		}
		lines = append(lines, pad+"}")
		return strings.Join(lines, "\n")
	default:
		return fmt.Sprintf("%s%s: %s", pad, formatTipeIndo(obj), obj.Inspect())
	}
}

func InitGlobalEnvironment() *Environment {
	ClearGlobalPanic()
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
	cetakMap.Pairs["format"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) == 0 {
				return NIL
			}
			formatStr := args[0].Inspect()
			var fmtArgs []interface{}
			for _, a := range args[1:] {
				switch v := a.(type) {
				case *Integer:
					fmtArgs = append(fmtArgs, v.Value)
				case *Float:
					fmtArgs = append(fmtArgs, v.Value)
				case *String:
					fmtArgs = append(fmtArgs, v.Value)
				case *Boolean:
					fmtArgs = append(fmtArgs, v.Value)
				default:
					fmtArgs = append(fmtArgs, a.Inspect())
				}
			}
			fmt.Printf(formatStr, fmtArgs...)
			return NIL
		},
	}
	cetakMap.Pairs["tipe"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: "kosong"}
			}
			return &String{Value: formatTipeIndo(args[0])}
		},
	}
	cetakMap.Pairs["dump"] = &Builtin{
		Fn: func(args ...Object) Object {
			parts := []string{}
			for _, a := range args {
				parts = append(parts, formatDumpIndo(a, 0))
			}
			fmt.Println(strings.Join(parts, "\n"))
			return NIL
		},
	}
	env.Set("cetak", cetakMap)

	// Modul web bawaan (REST API Router & Static Server)
	type webRoute struct {
		method  string
		pattern string
		parts   []string
		handler Object
	}
	var webRoutes []webRoute
	var webStaticPrefix, webStaticDir string
	var corsEnabled bool
	var webMutex sync.Mutex

	registerRoute := func(method, pattern string, fn Object) {
		webMutex.Lock()
		defer webMutex.Unlock()
		cleaned := strings.Trim(pattern, "/")
		var parts []string
		if cleaned != "" {
			parts = strings.Split(cleaned, "/")
		}
		webRoutes = append(webRoutes, webRoute{
			method:  method,
			pattern: pattern,
			parts:   parts,
			handler: fn,
		})
	}

	matchRoute := func(reqMethod, reqPath string) (*webRoute, map[string]string) {
		webMutex.Lock()
		defer webMutex.Unlock()
		cleaned := strings.Trim(reqPath, "/")
		var reqParts []string
		if cleaned != "" {
			reqParts = strings.Split(cleaned, "/")
		}

		type matchCandidate struct {
			route  *webRoute
			params map[string]string
			score  int // Semakin tinggi skor, semakin spesifik rute (exact match > param > wildcard)
		}

		var bestCandidate *matchCandidate

		for i := range webRoutes {
			r := &webRoutes[i]
			if r.method != "ALL" && r.method != reqMethod {
				continue
			}

			// 1. Dukungan Rute Catch-All Root atau Wildcard
			if r.pattern == "/" && len(reqParts) == 0 {
				cand := &matchCandidate{route: r, params: make(map[string]string), score: 100}
				if bestCandidate == nil || cand.score > bestCandidate.score {
					bestCandidate = cand
				}
				continue
			}

			// 2. Pencocokan segmen presisi
			if len(r.parts) == len(reqParts) {
				matched := true
				params := make(map[string]string)
				score := 0
				for j, p := range r.parts {
					if strings.HasPrefix(p, ":") {
						paramName := strings.TrimPrefix(p, ":")
						params[paramName] = reqParts[j]
						score += 10 // Parameter dinamis bernilai 10
					} else if p == reqParts[j] {
						score += 100 // Exact segment bernilai 100
					} else {
						matched = false
						break
					}
				}
				if matched {
					cand := &matchCandidate{route: r, params: params, score: score}
					if bestCandidate == nil || cand.score > bestCandidate.score {
						bestCandidate = cand
					}
				}
			} else if len(r.parts) > 0 && r.parts[len(r.parts)-1] == "*" {
				// Wildcard prefix /api/*
				prefixLen := len(r.parts) - 1
				if len(reqParts) >= prefixLen {
					matched := true
					for j := 0; j < prefixLen; j++ {
						if r.parts[j] != reqParts[j] {
							matched = false
							break
						}
					}
					if matched {
						cand := &matchCandidate{route: r, params: make(map[string]string), score: 5}
						if bestCandidate == nil || cand.score > bestCandidate.score {
							bestCandidate = cand
						}
					}
				}
			}
		}

		if bestCandidate != nil {
			return bestCandidate.route, bestCandidate.params
		}
		return nil, nil
	}

	webMap := &Map{Pairs: make(map[string]Object)}

	webMap.Pairs["aktifkanCORS"] = &Builtin{
		Fn: func(args ...Object) Object {
			corsEnabled = true
			return TRUE
		},
	}

	webMap.Pairs["berkasStatis"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			webStaticPrefix = args[0].Inspect()
			webStaticDir = args[1].Inspect()
			return TRUE
		},
	}

	addRouteMethod := func(httpMethod string) *Builtin {
		return &Builtin{
			Fn: func(args ...Object) Object {
				if len(args) < 2 {
					return NIL
				}
				registerRoute(httpMethod, args[0].Inspect(), args[1])
				return NIL
			},
		}
	}

	webMap.Pairs["tangani"] = addRouteMethod("ALL")
	webMap.Pairs["rute"] = addRouteMethod("ALL")
	webMap.Pairs["dapatkan"] = addRouteMethod("GET")
	webMap.Pairs["pos"] = addRouteMethod("POST")
	webMap.Pairs["taruh"] = addRouteMethod("PUT")
	webMap.Pairs["hapus"] = addRouteMethod("DELETE")

	webHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if corsEnabled {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		// Sajikan berkas statis jika prefix cocok
		if webStaticPrefix != "" && strings.HasPrefix(r.URL.Path, webStaticPrefix) {
			relPath := strings.TrimPrefix(r.URL.Path, webStaticPrefix)
			filePath := filepath.Join(webStaticDir, relPath)
			if fi, err := os.Stat(filePath); err == nil && !fi.IsDir() {
				http.ServeFile(w, r, filePath)
				return
			}
		}

		route, params := matchRoute(r.Method, r.URL.Path)
		if route == nil {
			http.NotFound(w, r)
			return
		}

		// Susun objek permintaan
		reqMap := &Map{Pairs: make(map[string]Object)}
		reqMap.Pairs["metode"] = &String{Value: r.Method}
		reqMap.Pairs["jalur"] = &String{Value: r.URL.Path}

		paramsMap := &Map{Pairs: make(map[string]Object)}
		for k, v := range params {
			paramsMap.Pairs[k] = &String{Value: v}
		}
		reqMap.Pairs["params"] = paramsMap

		kueriMap := &Map{Pairs: make(map[string]Object)}
		for k, v := range r.URL.Query() {
			if len(v) > 0 {
				kueriMap.Pairs[k] = &String{Value: v[0]}
			}
		}
		reqMap.Pairs["kueri"] = kueriMap

		headerMap := &Map{Pairs: make(map[string]Object)}
		for k, v := range r.Header {
			if len(v) > 0 {
				headerMap.Pairs[k] = &String{Value: v[0]}
			}
		}
		reqMap.Pairs["header"] = headerMap

		bodyBytes, _ := io.ReadAll(r.Body)
		reqMap.Pairs["badan"] = &String{Value: string(bodyBytes)}

		// Susun objek respons
		respMap := &Map{Pairs: make(map[string]Object)}
		respMap.Pairs["status"] = &Builtin{
			Fn: func(mArgs ...Object) Object {
				if len(mArgs) > 0 {
					if i, ok := mArgs[0].(*Integer); ok {
						w.WriteHeader(int(i.Value))
					}
				}
				return respMap
			},
		}
		respMap.Pairs["header"] = &Builtin{
			Fn: func(mArgs ...Object) Object {
				if len(mArgs) >= 2 {
					w.Header().Set(mArgs[0].Inspect(), mArgs[1].Inspect())
				}
				return respMap
			},
		}
		respMap.Pairs["teks"] = &Builtin{
			Fn: func(mArgs ...Object) Object {
				if len(mArgs) > 0 {
					w.Header().Set("Content-Type", "text/plain; charset=utf-8")
					w.Write([]byte(mArgs[0].Inspect()))
				}
				return NIL
			},
		}
		respMap.Pairs["html"] = &Builtin{
			Fn: func(mArgs ...Object) Object {
				if len(mArgs) > 0 {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					w.Write([]byte(mArgs[0].Inspect()))
				}
				return NIL
			},
		}
		respMap.Pairs["json"] = &Builtin{
			Fn: func(mArgs ...Object) Object {
				if len(mArgs) > 0 {
					goObj := convertKuToGoObject(mArgs[0])
					b, _ := json.Marshal(goObj)
					w.Header().Set("Content-Type", "application/json")
					w.Write(b)
				}
				return NIL
			},
		}

		// Panggil fungsi handler: bisa fungsi(permintaan, respons) atau fungsi(permintaan) kembalikan nilai
		respObj := applyFunction(route.handler, []Object{reqMap, respMap}, env)
		if respObj != nil && respObj != NIL {
			if m, ok := respObj.(*Map); ok {
				goObj := convertKuToGoObject(m)
				b, _ := json.Marshal(goObj)
				w.Header().Set("Content-Type", "application/json")
				w.Write(b)
			} else if s, ok := respObj.(*String); ok {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write([]byte(s.Value))
			}
		}
	})

	webMap.Pairs["dengarkan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return NIL
			}
			addr := args[0].Inspect()
			if i, ok := args[0].(*Integer); ok {
				addr = fmt.Sprintf("127.0.0.1:%d", i.Value)
			} else if !strings.Contains(addr, ":") {
				addr = "127.0.0.1:" + addr
			} else if strings.HasPrefix(addr, ":") {
				addr = "127.0.0.1" + addr
			}
			err := http.ListenAndServe(addr, webHandler)
			if err != nil {
				fmt.Printf("Galat server web: %v\n", err)
			}
			return NIL
		},
	}
	webMap.Pairs["jalankan"] = webMap.Pairs["dengarkan"]
	env.Set("web", webMap)

	// Registry koneksi basis data aktif untuk graceful shutdown
	var activeDBConnections []*sql.DB
	var activeDBMutex sync.Mutex

	registerDBConnection := func(d *sql.DB) {
		activeDBMutex.Lock()
		defer activeDBMutex.Unlock()
		activeDBConnections = append(activeDBConnections, d)
	}

	closeAllDBConnections := func() {
		activeDBMutex.Lock()
		defer activeDBMutex.Unlock()
		for _, d := range activeDBConnections {
			if d != nil {
				_ = d.Close()
			}
		}
		activeDBConnections = nil
	}

	// Modul desktop: jendela & dialog
	jendelaMap := &Map{Pairs: make(map[string]Object)}

	// IPC Registry untuk jembatan native JS di WebView -> runtime Bahasa Ku
	ipcHandlers := make(map[string]Object)
	invokeIPC := func(fungsi string, argumen []interface{}) (interface{}, error) {
		fnObj, exists := ipcHandlers[fungsi]
		if !exists {
			return nil, fmt.Errorf("Fungsi handler IPC '%s' tidak ditemukan", fungsi)
		}
		callArgs := []Object{}
		for _, a := range argumen {
			callArgs = append(callArgs, convertGoToKuObject(a))
		}
		var res Object = NIL
		if b, ok := fnObj.(*Builtin); ok {
			res = b.Fn(callArgs...)
		} else if f, ok := fnObj.(*Function); ok {
			res = applyFunction(f, callArgs, env)
		}
		return convertKuToGoObject(res), nil
	}
	var ipcServerPort int = 0
	var ipcServerOnce sync.Once

	startIPCServer := func() int {
		ipcServerOnce.Do(func() {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				return
			}
			ipcServerPort = listener.Addr().(*net.TCPAddr).Port

			mux := http.NewServeMux()
			mux.HandleFunc("/ku_tutup", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.WriteHeader(http.StatusOK)
				go func() {
					time.Sleep(100 * time.Millisecond)
					closeAllDBConnections()
					os.Exit(0)
				}()
			})

			mux.HandleFunc("/ku_ipc", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				if r.Method == "OPTIONS" {
					w.WriteHeader(http.StatusOK)
					return
				}
				if r.Method != "POST" {
					http.Error(w, "Metode harus POST", http.StatusMethodNotAllowed)
					return
				}

				body, _ := io.ReadAll(r.Body)
				var reqPayload struct {
					Fungsi   string        `json:"fungsi"`
					Argumen []interface{} `json:"argumen"`
				}
				if err := json.Unmarshal(body, &reqPayload); err != nil {
					http.Error(w, "Payload JSON tidak valid", http.StatusBadRequest)
					return
				}

				fnObj, exists := ipcHandlers[reqPayload.Fungsi]
				if !exists {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"galat": fmt.Sprintf("Fungsi handler IPC '%s' tidak ditemukan", reqPayload.Fungsi),
					})
					return
				}

				// Konversi argumen JSON ke Object Bahasa Ku
				callArgs := []Object{}
				for _, a := range reqPayload.Argumen {
					callArgs = append(callArgs, convertGoToKuObject(a))
				}

				var res Object = NIL
				if b, ok := fnObj.(*Builtin); ok {
					res = b.Fn(callArgs...)
				} else if f, ok := fnObj.(*Function); ok {
					res = applyFunction(f, callArgs, env)
				}

				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"hasil": convertKuToGoObject(res),
				})
			})

			go func() {
				_ = http.Serve(listener, mux)
			}()
		})
		return ipcServerPort
	}

	jendelaMap.Pairs["daftarkanFungsi"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			nama := args[0].Inspect()
			fn := args[1]
			ipcHandlers[nama] = fn
			startIPCServer()
			return TRUE
		},
	}
	jendelaMap.Pairs["ikat"] = jendelaMap.Pairs["daftarkanFungsi"]
	jendelaMap.Pairs["pesan"] = &Builtin{
		Fn: func(args ...Object) Object {
			judul := "Bahasa Ku"
			isi := ""
			tipe := "info"
			if len(args) == 1 {
				isi = args[0].Inspect()
			} else if len(args) == 2 {
				judul = args[0].Inspect()
				isi = args[1].Inspect()
			} else if len(args) >= 3 {
				judul = args[0].Inspect()
				isi = args[1].Inspect()
				tipe = args[2].Inspect()
			}

			tampilkanPesanNative(judul, isi, tipe)
			return NIL
		},
	}

	jendelaMap.Pairs["pilihBerkas"] = &Builtin{
		Fn: func(args ...Object) Object {
			judul := "Pilih Berkas"
			filter := "Semua Berkas (*.*)|*.*"
			if len(args) >= 1 {
				judul = args[0].Inspect()
			}
			if len(args) >= 2 {
				filter = args[1].Inspect()
			}
			res := dialogPilihBerkasNative(judul, filter)
			return &String{Value: res}
		},
	}

	jendelaMap.Pairs["pilihFolder"] = &Builtin{
		Fn: func(args ...Object) Object {
			judul := "Pilih Direktori / Folder"
			if len(args) >= 1 {
				judul = args[0].Inspect()
			}
			res := dialogPilihFolderNative(judul)
			return &String{Value: res}
		},
	}

	jendelaMap.Pairs["salinTeks"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			ok := salinPapanKlipNative(args[0].Inspect())
			return &Boolean{Value: ok}
		},
	}

	jendelaMap.Pairs["bacaKlip"] = &Builtin{
		Fn: func(args ...Object) Object {
			text := bacaPapanKlipNative()
			return &String{Value: text}
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
			lebar := 900
			tinggi := 650

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
				if w, ok := m.Pairs["lebar"]; ok {
					if i, isInt := w.(*Integer); isInt && i.Value > 0 {
						lebar = int(i.Value)
					}
				}
				if h, ok := m.Pairs["tinggi"]; ok {
					if i, isInt := h.(*Integer); isInt && i.Value > 0 {
						tinggi = int(i.Value)
					}
				}
			}

			bukaJendelaNative(judul, konten, url, lebar, tinggi, startIPCServer, invokeIPC, closeAllDBConnections)
			return NIL
		},
	}
	env.Set("jendela", jendelaMap)
	env.Set("desktop", jendelaMap)

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
				connMap.Pairs["pesan"] = &String{Value: TerjemahkanGalat(err)}
				return connMap
			}

			registerDBConnection(db)

			// Uji ping
			pingErr := db.Ping()
			if pingErr != nil {
				connMap.Pairs["terhubung"] = FALSE
				connMap.Pairs["pesan"] = &String{Value: TerjemahkanGalat(pingErr)}
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
					var queryParams []interface{}
					if len(qArgs) > 1 {
						for _, p := range qArgs[1:] {
							switch pv := p.(type) {
							case *Integer:
								queryParams = append(queryParams, pv.Value)
							case *Float:
								queryParams = append(queryParams, pv.Value)
							case *Boolean:
								queryParams = append(queryParams, pv.Value)
							case *String:
								queryParams = append(queryParams, pv.Value)
							case *Nil:
								queryParams = append(queryParams, nil)
							default:
								queryParams = append(queryParams, p.Inspect())
							}
						}
					}
					var rows *sql.Rows
					var qErr error
					if len(queryParams) > 0 {
						rows, qErr = db.Query(queryStr, queryParams...)
					} else {
						rows, qErr = db.Query(queryStr)
					}
					if qErr != nil {
						fmt.Printf("Galat kueri: %s\n", TerjemahkanGalat(qErr))
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
					var queryParams []interface{}
					if len(eArgs) > 1 {
						for _, p := range eArgs[1:] {
							switch pv := p.(type) {
							case *Integer:
								queryParams = append(queryParams, pv.Value)
							case *Float:
								queryParams = append(queryParams, pv.Value)
							case *Boolean:
								queryParams = append(queryParams, pv.Value)
							case *String:
								queryParams = append(queryParams, pv.Value)
							case *Nil:
								queryParams = append(queryParams, nil)
							default:
								queryParams = append(queryParams, p.Inspect())
							}
						}
					}
					var exErr error
					if len(queryParams) > 0 {
						_, exErr = db.Exec(sqlStr, queryParams...)
					} else {
						_, exErr = db.Exec(sqlStr)
					}
					if exErr != nil {
						fmt.Printf("Galat eksekusi: %s\n", TerjemahkanGalat(exErr))
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
	basisdataMap.Pairs["bukaBerkas"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return NIL
			}
			pathStr := args[0].Inspect()
			// Buka basis data lokal SQLite
			return basisdataMap.Pairs["sambung"].(*Builtin).Fn(&String{Value: "sqlite"}, &String{Value: pathStr})
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
	berkasMap.Pairs["cariPola"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Array{Elements: []Object{}}
			}
			pola := args[0].Inspect()
			matches, err := filepath.Glob(pola)
			if err != nil {
				return &Array{Elements: []Object{}}
			}
			res := make([]Object, len(matches))
			for i, m := range matches {
				res[i] = &String{Value: filepath.ToSlash(m)}
			}
			return &Array{Elements: res}
		},
	}
	berkasMap.Pairs["bersihkan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			dir := args[0].Inspect()
			entries, err := os.ReadDir(dir)
			if err != nil {
				return FALSE
			}
			for _, e := range entries {
				itemPath := filepath.Join(dir, e.Name())
				os.RemoveAll(itemPath)
			}
			return TRUE
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
	berkasMap.Pairs["buatFolder"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			err := os.MkdirAll(args[0].Inspect(), 0755)
			return &Boolean{Value: err == nil}
		},
	}
	berkasMap.Pairs["tambah"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			filePath := args[0].Inspect()
			data := args[1].Inspect()
			f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return FALSE
			}
			defer f.Close()
			_, err = f.WriteString(data)
			return &Boolean{Value: err == nil}
		},
	}
	berkasMap.Pairs["salin"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			srcPath := args[0].Inspect()
			dstPath := args[1].Inspect()
			srcFile, err := os.Open(srcPath)
			if err != nil {
				return FALSE
			}
			defer srcFile.Close()
			dstFile, err := os.Create(dstPath)
			if err != nil {
				return FALSE
			}
			defer dstFile.Close()
			_, err = io.Copy(dstFile, srcFile)
			return &Boolean{Value: err == nil}
		},
	}
	berkasMap.Pairs["pindah"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			srcPath := args[0].Inspect()
			dstPath := args[1].Inspect()
			err := os.Rename(srcPath, dstPath)
			return &Boolean{Value: err == nil}
		},
	}
	berkasMap.Pairs["apakahFolder"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			fi, err := os.Stat(args[0].Inspect())
			if err != nil {
				return FALSE
			}
			return &Boolean{Value: fi.IsDir()}
		},
	}
	berkasMap.Pairs["apakahDirektori"] = berkasMap.Pairs["apakahFolder"]
	berkasMap.Pairs["apakahBerkas"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			fi, err := os.Stat(args[0].Inspect())
			if err != nil {
				return FALSE
			}
			return &Boolean{Value: !fi.IsDir()}
		},
	}
	berkasMap.Pairs["salinFolder"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			src := args[0].Inspect()
			dst := args[1].Inspect()

			err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				rel, err := filepath.Rel(src, path)
				if err != nil {
					return err
				}
				targetPath := filepath.Join(dst, rel)
				if info.IsDir() {
					return os.MkdirAll(targetPath, info.Mode())
				}
				sFile, err := os.Open(path)
				if err != nil {
					return err
				}
				defer sFile.Close()
				dFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
				if err != nil {
					return err
				}
				defer dFile.Close()
				_, err = io.Copy(dFile, sFile)
				return err
			})
			return &Boolean{Value: err == nil}
		},
	}
	berkasMap.Pairs["salinDirektori"] = berkasMap.Pairs["salinFolder"]
	berkasMap.Pairs["izin"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			fi, err := os.Stat(args[0].Inspect())
			if err != nil {
				return &String{Value: ""}
			}
			return &String{Value: fi.Mode().String()}
		},
	}
	berkasMap.Pairs["bacaBaris"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Array{Elements: []Object{}}
			}
			f, err := os.Open(args[0].Inspect())
			if err != nil {
				return &Array{Elements: []Object{}}
			}
			defer f.Close()
			scanner := bufio.NewScanner(f)
			res := []Object{}
			for scanner.Scan() {
				res = append(res, &String{Value: scanner.Text()})
			}
			return &Array{Elements: res}
		},
	}
	berkasMap.Pairs["namaBerkas"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			return &String{Value: filepath.Base(args[0].Inspect())}
		},
	}
	berkasMap.Pairs["direktori"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			return &String{Value: filepath.Dir(args[0].Inspect())}
		},
	}
	berkasMap.Pairs["ekstensi"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			return &String{Value: filepath.Ext(args[0].Inspect())}
		},
	}
	berkasMap.Pairs["gabungJalur"] = &Builtin{
		Fn: func(args ...Object) Object {
			parts := []string{}
			for _, a := range args {
				parts = append(parts, a.Inspect())
			}
			return &String{Value: filepath.Join(parts...)}
		},
	}
	env.Set("berkas", berkasMap)

	// Modul jalur (Manipulasi Path Berkas ala filepath)
	jalurMap := &Map{Pairs: make(map[string]Object)}
	jalurMap.Pairs["gabung"] = &Builtin{
		Fn: func(args ...Object) Object {
			parts := []string{}
			for _, a := range args {
				parts = append(parts, a.Inspect())
			}
			return &String{Value: filepath.Join(parts...)}
		},
	}
	jalurMap.Pairs["nama"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			return &String{Value: filepath.Base(args[0].Inspect())}
		},
	}
	jalurMap.Pairs["induk"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			return &String{Value: filepath.Dir(args[0].Inspect())}
		},
	}
	jalurMap.Pairs["ekstensi"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			return &String{Value: filepath.Ext(args[0].Inspect())}
		},
	}
	jalurMap.Pairs["absolut"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			abs, err := filepath.Abs(args[0].Inspect())
			if err != nil {
				return &String{Value: args[0].Inspect()}
			}
			return &String{Value: abs}
		},
	}
	jalurMap.Pairs["relatif"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &String{Value: ""}
			}
			base := args[0].Inspect()
			target := args[1].Inspect()
			rel, err := filepath.Rel(base, target)
			if err != nil {
				return &String{Value: target}
			}
			return &String{Value: rel}
		},
	}
	jalurMap.Pairs["apakahAbsolut"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			return &Boolean{Value: filepath.IsAbs(args[0].Inspect())}
		},
	}
	jalurMap.Pairs["bersihkan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			return &String{Value: filepath.Clean(args[0].Inspect())}
		},
	}
	env.Set("jalur", jalurMap)

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
	sistemMap.Pairs["aturLingkungan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			key := args[0].Inspect()
			val := args[1].Inspect()
			err := os.Setenv(key, val)
			return &Boolean{Value: err == nil}
		},
	}
	sistemMap.Pairs["folderKerja"] = &Builtin{
		Fn: func(args ...Object) Object {
			wd, err := os.Getwd()
			if err != nil {
				return &String{Value: ""}
			}
			return &String{Value: wd}
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
	sistemMap.Pairs["perintah"] = &Builtin{
		Fn: func(args ...Object) Object {
			resMap := &Map{Pairs: make(map[string]Object)}
			if len(args) < 1 {
				resMap.Pairs["sukses"] = FALSE
				resMap.Pairs["kode"] = &Integer{Value: 1}
				resMap.Pairs["keluaran"] = &String{Value: "nama perintah tidak ditentukan"}
				return resMap
			}
			cmdName := args[0].Inspect()
			cmdArgs := []string{}
			for _, a := range args[1:] {
				cmdArgs = append(cmdArgs, a.Inspect())
			}
			cmd := exec.Command(cmdName, cmdArgs...)
			out, err := cmd.CombinedOutput()
			exitCode := int64(0)
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = int64(exitErr.ExitCode())
				} else {
					exitCode = 1
				}
				resMap.Pairs["sukses"] = FALSE
			} else {
				resMap.Pairs["sukses"] = TRUE
			}
			resMap.Pairs["kode"] = &Integer{Value: exitCode}
			resMap.Pairs["keluaran"] = &String{Value: strings.TrimSpace(string(out))}
			return resMap
		},
	}
	sistemMap.Pairs["eksekusi"] = &Builtin{
		Fn: func(args ...Object) Object {
			resMap := &Map{Pairs: make(map[string]Object)}
			if len(args) < 1 {
				resMap.Pairs["sukses"] = FALSE
				resMap.Pairs["kode"] = &Integer{Value: 1}
				resMap.Pairs["stdout"] = &String{Value: ""}
				resMap.Pairs["stderr"] = &String{Value: "nama perintah tidak ditentukan"}
				return resMap
			}
			cmdName := args[0].Inspect()
			cmdArgs := []string{}
			for _, a := range args[1:] {
				cmdArgs = append(cmdArgs, a.Inspect())
			}
			cmd := exec.Command(cmdName, cmdArgs...)
			var stdoutBuf, stderrBuf bytes.Buffer
			cmd.Stdout = &stdoutBuf
			cmd.Stderr = &stderrBuf

			err := cmd.Run()
			exitCode := int64(0)
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = int64(exitErr.ExitCode())
				} else {
					exitCode = 1
				}
				resMap.Pairs["sukses"] = FALSE
			} else {
				resMap.Pairs["sukses"] = TRUE
			}
			resMap.Pairs["kode"] = &Integer{Value: exitCode}
			resMap.Pairs["stdout"] = &String{Value: strings.TrimRight(stdoutBuf.String(), "\r\n")}
			resMap.Pairs["stderr"] = &String{Value: strings.TrimRight(stderrBuf.String(), "\r\n")}
			return resMap
		},
	}
	sistemMap.Pairs["memori"] = &Builtin{
		Fn: func(args ...Object) Object {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			res := &Map{Pairs: make(map[string]Object)}
			res.Pairs["alokasiMB"] = &Float{Value: float64(m.Alloc) / 1024 / 1024}
			res.Pairs["totalAlokasiMB"] = &Float{Value: float64(m.TotalAlloc) / 1024 / 1024}
			res.Pairs["sistemMB"] = &Float{Value: float64(m.Sys) / 1024 / 1024}
			res.Pairs["jumlahGC"] = &Integer{Value: int64(m.NumGC)}
			return res
		},
	}
	sistemMap.Pairs["jumlahCPU"] = &Builtin{
		Fn: func(args ...Object) Object {
			return &Integer{Value: int64(runtime.NumCPU())}
		},
	}
	sistemMap.Pairs["kunci"] = &Builtin{
		Fn: func(args ...Object) Object {
			var mu sync.Mutex
			kunciMap := &Map{Pairs: make(map[string]Object)}
			kunciMap.Pairs["kunci"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					mu.Lock()
					return NIL
				},
			}
			kunciMap.Pairs["buka"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					mu.Unlock()
					return NIL
				},
			}
			kunciMap.Pairs["cobaKunci"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					if mu.TryLock() {
						return TRUE
					}
					return FALSE
				},
			}
			return kunciMap
		},
	}
	sistemMap.Pairs["kunciBacaTulis"] = &Builtin{
		Fn: func(args ...Object) Object {
			var rw sync.RWMutex
			kunciMap := &Map{Pairs: make(map[string]Object)}
			kunciMap.Pairs["kunci"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					rw.Lock()
					return NIL
				},
			}
			kunciMap.Pairs["buka"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					rw.Unlock()
					return NIL
				},
			}
			kunciMap.Pairs["kunciBaca"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					rw.RLock()
					return NIL
				},
			}
			kunciMap.Pairs["bukaBaca"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					rw.RUnlock()
					return NIL
				},
			}
			kunciMap.Pairs["cobaKunci"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					if rw.TryLock() {
						return TRUE
					}
					return FALSE
				},
			}
			kunciMap.Pairs["cobaKunciBaca"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					if rw.TryRLock() {
						return TRUE
					}
					return FALSE
				},
			}
			return kunciMap
		},
	}
	sistemMap.Pairs["apakahWindows"] = &Builtin{
		Fn: func(args ...Object) Object {
			return &Boolean{Value: runtime.GOOS == "windows"}
		},
	}
	sistemMap.Pairs["apakahLinux"] = &Builtin{
		Fn: func(args ...Object) Object {
			return &Boolean{Value: runtime.GOOS == "linux"}
		},
	}
	sistemMap.Pairs["apakahMac"] = &Builtin{
		Fn: func(args ...Object) Object {
			return &Boolean{Value: runtime.GOOS == "darwin"}
		},
	}
	sistemMap.Pairs["bendera"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return NIL
			}
			flagName := args[0].Inspect()
			var defaultVal Object = NIL
			if len(args) > 1 {
				defaultVal = args[1]
			}
			prefix1 := "--" + flagName + "="
			prefix2 := "-" + flagName + "="
			for i, a := range os.Args {
				if strings.HasPrefix(a, prefix1) {
					return &String{Value: strings.TrimPrefix(a, prefix1)}
				}
				if strings.HasPrefix(a, prefix2) {
					return &String{Value: strings.TrimPrefix(a, prefix2)}
				}
				if a == "--"+flagName || a == "-"+flagName {
					if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "-") {
						return &String{Value: os.Args[i+1]}
					}
					return TRUE
				}
			}
			return defaultVal
		},
	}
	sistemMap.Pairs["muatEnv"] = &Builtin{
		Fn: func(args ...Object) Object {
			envPath := ".env"
			if len(args) > 0 {
				envPath = args[0].Inspect()
			}
			data, err := os.ReadFile(envPath)
			if err != nil {
				return FALSE
			}
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				idx := strings.Index(line, "=")
				if idx != -1 {
					k := strings.TrimSpace(line[:idx])
					v := strings.TrimSpace(line[idx+1:])
					// Hapus kutip jika ada
					if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
						v = v[1 : len(v)-1]
					}
					os.Setenv(k, v)
				}
			}
			return TRUE
		},
	}
	sistemMap.Pairs["bandingkanVersi"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Integer{Value: 0}
			}
			cleanVer := func(s string) []int {
				s = strings.TrimPrefix(strings.TrimSpace(s), "v")
				s = strings.TrimPrefix(s, "V")
				parts := strings.Split(s, ".")
				nums := []int{}
				for _, p := range parts {
					// Ambil angka saja jika ada suffix (e.g., -beta)
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
			v1 := cleanVer(args[0].Inspect())
			v2 := cleanVer(args[1].Inspect())

			for i := 0; i < len(v1) && i < len(v2); i++ {
				if v1[i] > v2[i] {
					return &Integer{Value: 1}
				}
				if v1[i] < v2[i] {
					return &Integer{Value: -1}
				}
			}
			return &Integer{Value: 0}
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
	jaringanMap.Pairs["letak"] = &Builtin{
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
			req, err := http.NewRequest(http.MethodPut, urlStr, reqBody)
			if err != nil {
				resMap := &Map{Pairs: make(map[string]Object)}
				resMap.Pairs["sukses"] = FALSE
				resMap.Pairs["status"] = &Integer{Value: 0}
				resMap.Pairs["badan"] = &String{Value: err.Error()}
				return resMap
			}
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{Timeout: 15 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				resMap := &Map{Pairs: make(map[string]Object)}
				resMap.Pairs["sukses"] = FALSE
				resMap.Pairs["status"] = &Integer{Value: 0}
				resMap.Pairs["badan"] = &String{Value: err.Error()}
				return resMap
			}
			defer resp.Body.Close()
			bodyBytes, _ := io.ReadAll(resp.Body)
			resMap := &Map{Pairs: make(map[string]Object)}
			resMap.Pairs["sukses"] = &Boolean{Value: resp.StatusCode >= 200 && resp.StatusCode < 300}
			resMap.Pairs["status"] = &Integer{Value: int64(resp.StatusCode)}
			resMap.Pairs["badan"] = &String{Value: string(bodyBytes)}
			var jsonObj interface{}
			if err := json.Unmarshal(bodyBytes, &jsonObj); err == nil {
				resMap.Pairs["json"] = convertGoToKuObject(jsonObj)
			} else {
				resMap.Pairs["json"] = NIL
			}
			return resMap
		},
	}
	jaringanMap.Pairs["hapus"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return NIL
			}
			urlStr := args[0].Inspect()
			req, err := http.NewRequest(http.MethodDelete, urlStr, nil)
			if err != nil {
				resMap := &Map{Pairs: make(map[string]Object)}
				resMap.Pairs["sukses"] = FALSE
				resMap.Pairs["status"] = &Integer{Value: 0}
				resMap.Pairs["badan"] = &String{Value: err.Error()}
				return resMap
			}
			client := &http.Client{Timeout: 15 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				resMap := &Map{Pairs: make(map[string]Object)}
				resMap.Pairs["sukses"] = FALSE
				resMap.Pairs["status"] = &Integer{Value: 0}
				resMap.Pairs["badan"] = &String{Value: err.Error()}
				return resMap
			}
			defer resp.Body.Close()
			bodyBytes, _ := io.ReadAll(resp.Body)
			resMap := &Map{Pairs: make(map[string]Object)}
			resMap.Pairs["sukses"] = &Boolean{Value: resp.StatusCode >= 200 && resp.StatusCode < 300}
			resMap.Pairs["status"] = &Integer{Value: int64(resp.StatusCode)}
			resMap.Pairs["badan"] = &String{Value: string(bodyBytes)}
			var jsonObj interface{}
			if err := json.Unmarshal(bodyBytes, &jsonObj); err == nil {
				resMap.Pairs["json"] = convertGoToKuObject(jsonObj)
			} else {
				resMap.Pairs["json"] = NIL
			}
			return resMap
		},
	}
	jaringanMap.Pairs["permintaan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return NIL
			}
			cfg, isMap := args[0].(*Map)
			if !isMap {
				return NIL
			}
			urlStr := ""
			method := http.MethodGet
			timeoutSec := 15
			var reqBody io.Reader

			if u, ok := cfg.Pairs["url"]; ok {
				urlStr = u.Inspect()
				if strObj, ok2 := u.(*String); ok2 {
					urlStr = strObj.Value
				}
			}
			if m, ok := cfg.Pairs["metode"]; ok {
				method = strings.ToUpper(m.Inspect())
				if strObj, ok2 := m.(*String); ok2 {
					method = strings.ToUpper(strObj.Value)
				}
			}
			if t, ok := cfg.Pairs["waktuTunggu"]; ok {
				if i, ok2 := t.(*Integer); ok2 {
					timeoutSec = int(i.Value)
				}
			}
			if b, ok := cfg.Pairs["badan"]; ok {
				if m, ok2 := b.(*Map); ok2 {
					data := make(map[string]interface{})
					for k, v := range m.Pairs {
						data[k] = v.Inspect()
					}
					jsonBytes, _ := json.Marshal(data)
					reqBody = strings.NewReader(string(jsonBytes))
				} else {
					reqBody = strings.NewReader(b.Inspect())
				}
			}

			req, err := http.NewRequest(method, urlStr, reqBody)
			if err != nil {
				resMap := &Map{Pairs: make(map[string]Object)}
				resMap.Pairs["sukses"] = FALSE
				resMap.Pairs["status"] = &Integer{Value: 0}
				resMap.Pairs["badan"] = &String{Value: err.Error()}
				return resMap
			}

			if h, ok := cfg.Pairs["tajuk"]; ok {
				if hMap, ok2 := h.(*Map); ok2 {
					for hk, hv := range hMap.Pairs {
						val := hv.Inspect()
						if strObj, ok3 := hv.(*String); ok3 {
							val = strObj.Value
						}
						req.Header.Set(hk, val)
					}
				}
			}

			client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				resMap := &Map{Pairs: make(map[string]Object)}
				resMap.Pairs["sukses"] = FALSE
				resMap.Pairs["status"] = &Integer{Value: 0}
				resMap.Pairs["badan"] = &String{Value: err.Error()}
				return resMap
			}
			defer resp.Body.Close()

			bodyBytes, _ := io.ReadAll(resp.Body)
			resMap := &Map{Pairs: make(map[string]Object)}
			resMap.Pairs["sukses"] = &Boolean{Value: resp.StatusCode >= 200 && resp.StatusCode < 300}
			resMap.Pairs["status"] = &Integer{Value: int64(resp.StatusCode)}
			resMap.Pairs["badan"] = &String{Value: string(bodyBytes)}

			var jsonObj interface{}
			if err := json.Unmarshal(bodyBytes, &jsonObj); err == nil {
				resMap.Pairs["json"] = convertGoToKuObject(jsonObj)
			} else {
				resMap.Pairs["json"] = NIL
			}
			return resMap
		},
	}
	jaringanMap.Pairs["ambilJSON"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return NIL
			}
			cfg := &Map{Pairs: make(map[string]Object)}
			cfg.Pairs["metode"] = &String{Value: "GET"}
			cfg.Pairs["url"] = args[0]
			if len(args) >= 2 {
				cfg.Pairs["tajuk"] = args[1]
			}
			res := jaringanMap.Pairs["permintaan"].(*Builtin).Fn(cfg)
			if rMap, ok := res.(*Map); ok {
				if jVal, exists := rMap.Pairs["json"]; exists && jVal != NIL {
					return jVal
				}
			}
			return NIL
		},
	}
	jaringanMap.Pairs["kirimJSON"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			cfg := &Map{Pairs: make(map[string]Object)}
			cfg.Pairs["metode"] = &String{Value: "POST"}
			cfg.Pairs["url"] = args[0]

			// Marshal JSON body
			goObj := convertKuToGoObject(args[1])
			b, _ := json.Marshal(goObj)
			cfg.Pairs["badan"] = &String{Value: string(b)}

			headers := &Map{Pairs: make(map[string]Object)}
			headers.Pairs["Content-Type"] = &String{Value: "application/json"}
			if len(args) >= 3 {
				if userHeaders, ok := args[2].(*Map); ok {
					for k, v := range userHeaders.Pairs {
						headers.Pairs[k] = v
					}
				}
			}
			cfg.Pairs["tajuk"] = headers

			res := jaringanMap.Pairs["permintaan"].(*Builtin).Fn(cfg)
			if rMap, ok := res.(*Map); ok {
				if jVal, exists := rMap.Pairs["json"]; exists && jVal != NIL {
					return jVal
				}
			}
			return NIL
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
	kriptoMap.Pairs["sha256Berkas"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			f, err := os.Open(args[0].Inspect())
			if err != nil {
				return &String{Value: ""}
			}
			defer f.Close()
			h := sha256.New()
			if _, err := io.Copy(h, f); err != nil {
				return &String{Value: ""}
			}
			return &String{Value: hex.EncodeToString(h.Sum(nil))}
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
	kriptoMap.Pairs["md5Berkas"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			f, err := os.Open(args[0].Inspect())
			if err != nil {
				return &String{Value: ""}
			}
			defer f.Close()
			h := md5.New()
			if _, err := io.Copy(h, f); err != nil {
				return &String{Value: ""}
			}
			return &String{Value: hex.EncodeToString(h.Sum(nil))}
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
	kriptoMap.Pairs["uuid"] = &Builtin{
		Fn: func(args ...Object) Object {
			u := make([]byte, 16)
			_, _ = rand.Read(u)
			u[6] = (u[6] & 0x0f) | 0x40 // Version 4
			u[8] = (u[8] & 0x3f) | 0x80 // Variant RFC 4122
			uuidStr := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:])
			return &String{Value: uuidStr}
		},
	}
	kriptoMap.Pairs["hmacSha256"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &String{Value: ""}
			}
			key := []byte(args[0].Inspect())
			pesan := []byte(args[1].Inspect())
			mac := hmac.New(sha256.New, key)
			mac.Write(pesan)
			return &String{Value: hex.EncodeToString(mac.Sum(nil))}
		},
	}
	kriptoMap.Pairs["buatJWT"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &String{Value: ""}
			}
			secret := []byte(args[1].Inspect())
			headerJSON := `{"alg":"HS256","typ":"JWT"}`
			headerB64 := base64.RawURLEncoding.EncodeToString([]byte(headerJSON))

			var payloadJSON string
			if m, ok := args[0].(*Map); ok {
				data := make(map[string]interface{})
				for k, v := range m.Pairs {
					data[k] = convertKuToGoObject(v)
				}
				if len(args) >= 3 {
					if expSec, ok2 := args[2].(*Integer); ok2 && expSec.Value > 0 {
						data["exp"] = time.Now().Unix() + expSec.Value
					}
				}
				b, _ := json.Marshal(data)
				payloadJSON = string(b)
			} else {
				payloadJSON = args[0].Inspect()
			}
			payloadB64 := base64.RawURLEncoding.EncodeToString([]byte(payloadJSON))
			unsignedToken := headerB64 + "." + payloadB64

			mac := hmac.New(sha256.New, secret)
			mac.Write([]byte(unsignedToken))
			signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

			return &String{Value: unsignedToken + "." + signature}
		},
	}
	kriptoMap.Pairs["verifikasiJWT"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			tokenStr := args[0].Inspect()
			if sObj, ok := args[0].(*String); ok {
				tokenStr = sObj.Value
			}
			secret := []byte(args[1].Inspect())
			parts := strings.Split(tokenStr, ".")
			if len(parts) != 3 {
				return NIL
			}
			unsignedToken := parts[0] + "." + parts[1]
			mac := hmac.New(sha256.New, secret)
			mac.Write([]byte(unsignedToken))
			expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

			if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
				return NIL
			}

			payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
			if err != nil {
				return NIL
			}
			var parsed map[string]interface{}
			if err := json.Unmarshal(payloadBytes, &parsed); err != nil {
				return NIL
			}

			// Cek expiration jika ada
			if expVal, hasExp := parsed["exp"]; hasExp {
				if expFloat, ok := expVal.(float64); ok {
					if int64(expFloat) < time.Now().Unix() {
						return NIL // Token telah kedaluwarsa
					}
				}
			}

			return convertGoToKuObject(parsed)
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
	matematikaMap.Pairs["jumlah"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Integer{Value: 0}
			}
			arr, isArr := args[0].(*Array)
			if !isArr || len(arr.Elements) == 0 {
				return &Integer{Value: 0}
			}
			hasFloat := false
			totalInt := int64(0)
			totalFloat := float64(0)
			for _, e := range arr.Elements {
				if i, ok := e.(*Integer); ok {
					totalInt += i.Value
					totalFloat += float64(i.Value)
				} else if f, ok := e.(*Float); ok {
					hasFloat = true
					totalFloat += f.Value
				}
			}
			if hasFloat {
				return &Float{Value: totalFloat}
			}
			return &Integer{Value: totalInt}
		},
	}
	matematikaMap.Pairs["median"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Float{Value: 0}
			}
			arr, isArr := args[0].(*Array)
			if !isArr || len(arr.Elements) == 0 {
				return &Float{Value: 0}
			}
			nums := []float64{}
			for _, e := range arr.Elements {
				if i, ok := e.(*Integer); ok {
					nums = append(nums, float64(i.Value))
				} else if f, ok := e.(*Float); ok {
					nums = append(nums, f.Value)
				}
			}
			if len(nums) == 0 {
				return &Float{Value: 0}
			}
			sort.Float64s(nums)
			n := len(nums)
			if n%2 == 1 {
				return &Float{Value: nums[n/2]}
			}
			return &Float{Value: (nums[(n/2)-1] + nums[n/2]) / 2.0}
		},
	}
	matematikaMap.Pairs["E"] = &Float{Value: math.E}
	matematikaMap.Pairs["log"] = &Builtin{
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
			return &Float{Value: math.Log(val)}
		},
	}
	matematikaMap.Pairs["log10"] = &Builtin{
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
			return &Float{Value: math.Log10(val)}
		},
	}
	matematikaMap.Pairs["mutlak"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Float{Value: 0}
			}
			if i, ok := args[0].(*Integer); ok {
				if i.Value < 0 {
					return &Integer{Value: -i.Value}
				}
				return i
			} else if f, ok := args[0].(*Float); ok {
				return &Float{Value: math.Abs(f.Value)}
			}
			return args[0]
		},
	}
	matematikaMap.Pairs["hipot"] = &Builtin{
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
			return &Float{Value: math.Hypot(x, y)}
		},
	}
	matematikaMap.Pairs["jepit"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 3 {
				return args[0]
			}
			getVal := func(o Object) float64 {
				if i, ok := o.(*Integer); ok {
					return float64(i.Value)
				} else if f, ok := o.(*Float); ok {
					return f.Value
				}
				return 0
			}
			v := getVal(args[0])
			minVal := getVal(args[1])
			maxVal := getVal(args[2])
			if v < minVal {
				v = minVal
			}
			if v > maxVal {
				v = maxVal
			}
			if _, ok := args[0].(*Integer); ok {
				return &Integer{Value: int64(v)}
			}
			return &Float{Value: v}
		},
	}
	matematikaMap.Pairs["tanda"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Integer{Value: 0}
			}
			var v float64
			if i, ok := args[0].(*Integer); ok {
				v = float64(i.Value)
			} else if f, ok := args[0].(*Float); ok {
				v = f.Value
			}
			if v > 0 {
				return &Integer{Value: 1}
			} else if v < 0 {
				return &Integer{Value: -1}
			}
			return &Integer{Value: 0}
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
	teksMap.Pairs["padKiri"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return args[0]
			}
			s := args[0].Inspect()
			targetLen := 0
			if i, ok := args[1].(*Integer); ok {
				targetLen = int(i.Value)
			}
			padChar := " "
			if len(args) >= 3 && len(args[2].Inspect()) > 0 {
				padChar = string(args[2].Inspect()[0])
			}
			if len(s) >= targetLen {
				return &String{Value: s}
			}
			pad := strings.Repeat(padChar, targetLen-len(s))
			return &String{Value: pad + s}
		},
	}
	teksMap.Pairs["padKanan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return args[0]
			}
			s := args[0].Inspect()
			targetLen := 0
			if i, ok := args[1].(*Integer); ok {
				targetLen = int(i.Value)
			}
			padChar := " "
			if len(args) >= 3 && len(args[2].Inspect()) > 0 {
				padChar = string(args[2].Inspect()[0])
			}
			if len(s) >= targetLen {
				return &String{Value: s}
			}
			pad := strings.Repeat(padChar, targetLen-len(s))
			return &String{Value: s + pad}
		},
	}
	teksMap.Pairs["karakter"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &String{Value: ""}
			}
			s := args[0].Inspect()
			idx := int64(0)
			if i, ok := args[1].(*Integer); ok {
				idx = i.Value
			}
			runes := []rune(s)
			if idx < 0 || idx >= int64(len(runes)) {
				return &String{Value: ""}
			}
			return &String{Value: string(runes[idx])}
		},
	}
	teksMap.Pairs["apakahKosong"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return TRUE
			}
			return &Boolean{Value: len(strings.TrimSpace(args[0].Inspect())) == 0}
		},
	}
	teksMap.Pairs["apakahHuruf"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			s := args[0].Inspect()
			if len(s) == 0 {
				return FALSE
			}
			for _, r := range s {
				if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
					return FALSE
				}
			}
			return TRUE
		},
	}
	teksMap.Pairs["apakahAngka"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			s := strings.TrimSpace(args[0].Inspect())
			if len(s) == 0 {
				return FALSE
			}
			for _, r := range s {
				if !(r >= '0' && r <= '9') {
					return FALSE
				}
			}
			return TRUE
		},
	}
	teksMap.Pairs["indeksTerakhir"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Integer{Value: -1}
			}
			return &Integer{Value: int64(strings.LastIndex(args[0].Inspect(), args[1].Inspect()))}
		},
	}
	teksMap.Pairs["pangkasKiri"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			cutset := " \t\r\n"
			if len(args) >= 2 {
				cutset = args[1].Inspect()
			}
			return &String{Value: strings.TrimLeft(args[0].Inspect(), cutset)}
		},
	}
	teksMap.Pairs["pangkasKanan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			cutset := " \t\r\n"
			if len(args) >= 2 {
				cutset = args[1].Inspect()
			}
			return &String{Value: strings.TrimRight(args[0].Inspect(), cutset)}
		},
	}
	teksMap.Pairs["kapitalkan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			s := args[0].Inspect()
			if len(s) == 0 {
				return &String{Value: ""}
			}
			r := []rune(s)
			r[0] = unicode.ToUpper(r[0])
			return &String{Value: string(r)}
		},
	}
	teksMap.Pairs["slug"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			s := strings.ToLower(args[0].Inspect())
			// Ganti non-alfanumerik dengan tanda hubung
			reg := regexp.MustCompile(`[^a-z0-9]+`)
			slug := reg.ReplaceAllString(s, "-")
			slug = strings.Trim(slug, "-")
			return &String{Value: slug}
		},
	}
	teksMap.Pairs["keJudul"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			words := strings.Fields(args[0].Inspect())
			for i, w := range words {
				r := []rune(w)
				if len(r) > 0 {
					r[0] = unicode.ToUpper(r[0])
					words[i] = string(r)
				}
			}
			return &String{Value: strings.Join(words, " ")}
		},
	}
	teksMap.Pairs["keCamelCase"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			s := args[0].Inspect()
			reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
			parts := reg.Split(s, -1)
			var sb strings.Builder
			count := 0
			for _, p := range parts {
				if p == "" {
					continue
				}
				p = strings.ToLower(p)
				if count == 0 {
					sb.WriteString(p)
				} else {
					r := []rune(p)
					r[0] = unicode.ToUpper(r[0])
					sb.WriteString(string(r))
				}
				count++
			}
			return &String{Value: sb.String()}
		},
	}
	teksMap.Pairs["keSnakeCase"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			s := args[0].Inspect()
			var sb strings.Builder
			for i, r := range s {
				if unicode.IsUpper(r) {
					if i > 0 && s[i-1] != '_' && s[i-1] != '-' && s[i-1] != ' ' {
						sb.WriteByte('_')
					}
					sb.WriteRune(unicode.ToLower(r))
				} else if r == ' ' || r == '-' {
					sb.WriteByte('_')
				} else {
					sb.WriteRune(r)
				}
			}
			reg := regexp.MustCompile(`_+`)
			res := reg.ReplaceAllString(sb.String(), "_")
			return &String{Value: strings.Trim(res, "_")}
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
	konversiMap.Pairs["desimal"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: "0"}
			}
			var val float64
			if i, ok := args[0].(*Integer); ok {
				val = float64(i.Value)
			} else if f, ok := args[0].(*Float); ok {
				val = f.Value
			}
			presisi := 2
			if len(args) >= 2 {
				if p, ok := args[1].(*Integer); ok && p.Value >= 0 {
					presisi = int(p.Value)
				}
			}
			formatStr := fmt.Sprintf("%%.%df", presisi)
			return &String{Value: fmt.Sprintf(formatStr, val)}
		},
	}
	konversiMap.Pairs["rupiah"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: "Rp 0"}
			}
			var val int64
			if i, ok := args[0].(*Integer); ok {
				val = i.Value
			} else if f, ok := args[0].(*Float); ok {
				val = int64(f.Value)
			}
			isNegative := val < 0
			if isNegative {
				val = -val
			}
			s := strconv.FormatInt(val, 10)
			n := len(s)
			if n <= 3 {
				if isNegative {
					return &String{Value: fmt.Sprintf("Rp -%s", s)}
				}
				return &String{Value: fmt.Sprintf("Rp %s", s)}
			}
			var out []string
			rem := n % 3
			if rem > 0 {
				out = append(out, s[:rem])
			}
			for i := rem; i < n; i += 3 {
				out = append(out, s[i:i+3])
			}
			res := strings.Join(out, ".")
			if isNegative {
				res = "-" + res
			}
			return &String{Value: "Rp " + res}
		},
	}
	konversiMap.Pairs["apakahAngka"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			t := args[0].Type()
			return &Boolean{Value: t == INTEGER_OBJ || t == FLOAT_OBJ}
		},
	}
	konversiMap.Pairs["apakahTeks"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			return &Boolean{Value: args[0].Type() == STRING_OBJ}
		},
	}
	konversiMap.Pairs["apakahLarik"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			return &Boolean{Value: args[0].Type() == ARRAY_OBJ}
		},
	}
	konversiMap.Pairs["apakahPeta"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			return &Boolean{Value: args[0].Type() == MAP_OBJ}
		},
	}
	konversiMap.Pairs["apakahStruktur"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			return &Boolean{Value: args[0].Type() == STRUCT_INST_OBJ}
		},
	}
	konversiMap.Pairs["apakahKanal"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			return &Boolean{Value: args[0].Type() == CHANNEL_OBJ}
		},
	}
	konversiMap.Pairs["apakahTipe"] = &Builtin{
		Fn: func(args ...Object) Object {
			// type assertion helper: konversi.apakahTipe(obj, "struktur") atau konversi.apakahTipe(obj, "Orang")
			if len(args) < 2 {
				return FALSE
			}
			expected := strings.ToLower(args[1].Inspect())
			actualTipe := strings.ToLower(formatTipeIndo(args[0]))
			if actualTipe == expected {
				return TRUE
			}
			if si, ok := args[0].(*StructInstance); ok && si.Def != nil {
				if strings.ToLower(si.Def.Name) == expected {
					return TRUE
				}
			}
			return FALSE
		},
	}
	konversiMap.Pairs["apakahKosong"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return TRUE
			}
			return &Boolean{Value: args[0] == nil || args[0].Type() == NIL_OBJ}
		},
	}
	konversiMap.Pairs["nilaiAwal"] = &Builtin{
		Fn: func(args ...Object) Object {
			// Zero value generator: konversi.nilaiAwal("angka") -> 0, "teks" -> "", "boolean" -> salah, "daftar" -> []
			if len(args) < 1 {
				return NIL
			}
			t := strings.ToLower(strings.TrimSpace(args[0].Inspect()))
			switch t {
			case "angka", "int", "integer":
				return &Integer{Value: 0}
			case "pecahan", "desimal", "float":
				return &Float{Value: 0.0}
			case "teks", "string":
				return &String{Value: ""}
			case "boolean", "bool":
				return FALSE
			case "daftar", "larik", "array", "slice":
				return &Array{Elements: []Object{}}
			case "peta", "map":
				return &Map{Pairs: make(map[string]Object)}
			default:
				return NIL
			}
		},
	}
	konversiMap.Pairs["formatBita"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: "0 B"}
			}
			var b float64
			if i, ok := args[0].(*Integer); ok {
				b = float64(i.Value)
			} else if f, ok := args[0].(*Float); ok {
				b = f.Value
			}
			units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
			idx := 0
			for b >= 1024 && idx < len(units)-1 {
				b /= 1024
				idx++
			}
			if idx == 0 {
				return &String{Value: fmt.Sprintf("%.0f B", b)}
			}
			return &String{Value: fmt.Sprintf("%.2f %s", b, units[idx])}
		},
	}
	konversiMap.Pairs["formatAngka"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: "0"}
			}
			var val float64
			if i, ok := args[0].(*Integer); ok {
				val = float64(i.Value)
			} else if f, ok := args[0].(*Float); ok {
				val = f.Value
			}
			des := 2
			if len(args) >= 2 {
				if d, ok := args[1].(*Integer); ok {
					des = int(d.Value)
				}
			}
			formatStr := fmt.Sprintf("%%.%df", des)
			return &String{Value: fmt.Sprintf(formatStr, val)}
		},
	}
	konversiMap.Pairs["formatUang"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: "Rp 0"}
			}
			var n int64
			if i, ok := args[0].(*Integer); ok {
				n = i.Value
			} else if f, ok := args[0].(*Float); ok {
				n = int64(f.Value)
			} else if s, ok := args[0].(*String); ok {
				if parsed, err := strconv.ParseInt(strings.TrimSpace(s.Value), 10, 64); err == nil {
					n = parsed
				} else if parsedF, err := strconv.ParseFloat(strings.TrimSpace(s.Value), 64); err == nil {
					n = int64(parsedF)
				}
			}
			simbol := "Rp "
			if len(args) >= 2 {
				simbol = args[1].Inspect()
			}
			neg := false
			if n < 0 {
				neg = true
				n = -n
			}
			str := strconv.FormatInt(n, 10)
			var sb strings.Builder
			l := len(str)
			for idx, r := range str {
				sb.WriteRune(r)
				if (l-idx-1)%3 == 0 && idx != l-1 {
					sb.WriteByte('.')
				}
			}
			hasil := simbol + sb.String()
			if neg {
				hasil = "-" + hasil
			}
			return &String{Value: hasil}
		},
	}
	env.Set("konversi", konversiMap)

	// Modul bita (Manipulasi Byte Array & Buffer Biner)
	bitaMap := &Map{Pairs: make(map[string]Object)}
	bitaMap.Pairs["dariTeks"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Array{Elements: []Object{}}
			}
			bs := []byte(args[0].Inspect())
			elems := make([]Object, len(bs))
			for i, b := range bs {
				elems[i] = &Integer{Value: int64(b)}
			}
			return &Array{Elements: elems}
		},
	}
	bitaMap.Pairs["keTeks"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &String{Value: ""}
			}
			bs := make([]byte, len(arr.Elements))
			for i, e := range arr.Elements {
				if intObj, isInt := e.(*Integer); isInt {
					bs[i] = byte(intObj.Value)
				}
			}
			return &String{Value: string(bs)}
		},
	}
	bitaMap.Pairs["panjang"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Integer{Value: 0}
			}
			return &Integer{Value: int64(len([]byte(args[0].Inspect())))}
		},
	}
	bitaMap.Pairs["karakter"] = &Builtin{
		Fn: func(args ...Object) Object {
			// Konversi teks ke array rune Unicode (Code points) ala []rune(str) di Go
			if len(args) < 1 {
				return &Array{Elements: []Object{}}
			}
			s := args[0].Inspect()
			runes := []rune(s)
			elems := make([]Object, len(runes))
			for i, r := range runes {
				elems[i] = &Integer{Value: int64(r)}
			}
			return &Array{Elements: elems}
		},
	}
	bitaMap.Pairs["dariKarakter"] = &Builtin{
		Fn: func(args ...Object) Object {
			// Konversi array rune Unicode kembali ke teks string: string([]rune)
			if len(args) < 1 {
				return &String{Value: ""}
			}
			arr, ok := args[0].(*Array)
			if !ok {
				if i, isInt := args[0].(*Integer); isInt {
					return &String{Value: string(rune(i.Value))}
				}
				return &String{Value: ""}
			}
			runes := make([]rune, len(arr.Elements))
			for i, e := range arr.Elements {
				if intObj, isInt := e.(*Integer); isInt {
					runes[i] = rune(intObj.Value)
				}
			}
			return &String{Value: string(runes)}
		},
	}
	bitaMap.Pairs["jumlahKarakter"] = &Builtin{
		Fn: func(args ...Object) Object {
			// Menghitung jumlah karakter Unicode asli (bukan ukuran byte mentah)
			if len(args) < 1 {
				return &Integer{Value: 0}
			}
			return &Integer{Value: int64(len([]rune(args[0].Inspect())))}
		},
	}
	env.Set("bita", bitaMap)

	// Modul acak (Pengacakan Lanjutan & Pemilihan Data)
	acakMap := &Map{Pairs: make(map[string]Object)}
	acakMap.Pairs["pilih"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return NIL
			}
			arr, isArr := args[0].(*Array)
			if !isArr || len(arr.Elements) == 0 {
				return NIL
			}
			idx := mrand.Intn(len(arr.Elements))
			return arr.Elements[idx]
		},
	}
	acakMap.Pairs["pecahan"] = &Builtin{
		Fn: func(args ...Object) Object {
			return &Float{Value: mrand.Float64()}
		},
	}
	acakMap.Pairs["teks"] = &Builtin{
		Fn: func(args ...Object) Object {
			n := 8
			if len(args) > 0 {
				if i, ok := args[0].(*Integer); ok && i.Value > 0 {
					n = int(i.Value)
				}
			}
			const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
			res := make([]byte, n)
			for i := 0; i < n; i++ {
				res[i] = chars[mrand.Intn(len(chars))]
			}
			return &String{Value: string(res)}
		},
	}
	acakMap.Pairs["angka"] = &Builtin{
		Fn: func(args ...Object) Object {
			minVal := int64(0)
			maxVal := int64(100)
			if len(args) == 1 {
				if i, ok := args[0].(*Integer); ok {
					maxVal = i.Value
				}
			} else if len(args) >= 2 {
				if i, ok := args[0].(*Integer); ok {
					minVal = i.Value
				}
				if i, ok := args[1].(*Integer); ok {
					maxVal = i.Value
				}
			}
			if maxVal < minVal {
				minVal, maxVal = maxVal, minVal
			}
			diff := maxVal - minVal + 1
			if diff <= 0 {
				diff = 1
			}
			return &Integer{Value: minVal + mrand.Int63n(diff)}
		},
	}
	acakMap.Pairs["uuid"] = &Builtin{
		Fn: func(args ...Object) Object {
			u := make([]byte, 16)
			rand.Read(u)
			u[6] = (u[6] & 0x0f) | 0x40
			u[8] = (u[8] & 0x3f) | 0x80
			return &String{Value: fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:])}
		},
	}
	env.Set("acak", acakMap)

	// Modul enkoding (CSV & URL Query)
	enkodingMap := &Map{Pairs: make(map[string]Object)}
	enkodingMap.Pairs["uraiCSV"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Array{Elements: []Object{}}
			}
			r := csv.NewReader(strings.NewReader(args[0].Inspect()))
			records, err := r.ReadAll()
			if err != nil {
				return &Array{Elements: []Object{}}
			}
			rows := make([]Object, len(records))
			for i, row := range records {
				cols := make([]Object, len(row))
				for j, col := range row {
					cols[j] = &String{Value: col}
				}
				rows[i] = &Array{Elements: cols}
			}
			return &Array{Elements: rows}
		},
	}
	enkodingMap.Pairs["kodekanCSV"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			arr, isArr := args[0].(*Array)
			if !isArr {
				return &String{Value: ""}
			}
			var buf bytes.Buffer
			w := csv.NewWriter(&buf)
			for _, rowObj := range arr.Elements {
				if rowArr, ok := rowObj.(*Array); ok {
					record := []string{}
					for _, col := range rowArr.Elements {
						record = append(record, col.Inspect())
					}
					_ = w.Write(record)
				}
			}
			w.Flush()
			return &String{Value: buf.String()}
		},
	}
	enkodingMap.Pairs["kodekanURL"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			return &String{Value: url.QueryEscape(args[0].Inspect())}
		},
	}
	enkodingMap.Pairs["dekodekanURL"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			res, err := url.QueryUnescape(args[0].Inspect())
			if err != nil {
				return args[0]
			}
			return &String{Value: res}
		},
	}
	enkodingMap.Pairs["kodekanHeksa"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			return &String{Value: hex.EncodeToString([]byte(args[0].Inspect()))}
		},
	}
	enkodingMap.Pairs["dekodekanHeksa"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			data, err := hex.DecodeString(args[0].Inspect())
			if err != nil {
				return args[0]
			}
			return &String{Value: string(data)}
		},
	}
	enkodingMap.Pairs["bangunQuery"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			m, isMap := args[0].(*Map)
			if !isMap {
				return &String{Value: ""}
			}
			vals := url.Values{}
			for k, v := range m.Pairs {
				vals.Set(k, v.Inspect())
			}
			return &String{Value: vals.Encode()}
		},
	}
	env.Set("enkoding", enkodingMap)

	// Modul url (Parsing & Manipulasi URL / URI)
	urlMap := &Map{Pairs: make(map[string]Object)}
	urlMap.Pairs["urai"] = &Builtin{
		Fn: func(args ...Object) Object {
			resMap := &Map{Pairs: make(map[string]Object)}
			if len(args) < 1 {
				return resMap
			}
			u, err := url.Parse(args[0].Inspect())
			if err != nil {
				return resMap
			}
			resMap.Pairs["skema"] = &String{Value: u.Scheme}
			resMap.Pairs["host"] = &String{Value: u.Host}
			resMap.Pairs["namaHost"] = &String{Value: u.Hostname()}
			resMap.Pairs["port"] = &String{Value: u.Port()}
			resMap.Pairs["jalur"] = &String{Value: u.Path}
			resMap.Pairs["kueri"] = &String{Value: u.RawQuery}
			resMap.Pairs["fragmen"] = &String{Value: u.Fragment}
			return resMap
		},
	}
	urlMap.Pairs["kodekan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			return &String{Value: url.QueryEscape(args[0].Inspect())}
		},
	}
	urlMap.Pairs["dekodekan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			res, err := url.QueryUnescape(args[0].Inspect())
			if err != nil {
				return args[0]
			}
			return &String{Value: res}
		},
	}
	env.Set("url", urlMap)

	// Modul json (Serialisasi & Deserialisasi JSON Standar)
	jsonMap := &Map{Pairs: make(map[string]Object)}
	jsonMap.Pairs["urai"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return NIL
			}
			rawStr := args[0].Inspect()
			var parsed interface{}
			err := json.Unmarshal([]byte(rawStr), &parsed)
			if err != nil {
				return NIL
			}
			return convertGoToKuObject(parsed)
		},
	}
	jsonMap.Pairs["kodekan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: "null"}
			}
			goObj := convertKuToGoObject(args[0])
			b, err := json.Marshal(goObj)
			if err != nil {
				return &String{Value: "null"}
			}
			return &String{Value: string(b)}
		},
	}
	jsonMap.Pairs["format"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: "null"}
			}
			indentStr := "  "
			if len(args) > 1 {
				if intObj, ok := args[1].(*Integer); ok {
					indentStr = strings.Repeat(" ", int(intObj.Value))
				} else {
					indentStr = args[1].Inspect()
				}
			}
			goObj := convertKuToGoObject(args[0])
			b, err := json.MarshalIndent(goObj, "", indentStr)
			if err != nil {
				return &String{Value: "null"}
			}
			return &String{Value: string(b)}
		},
	}
	jsonMap.Pairs["valid"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			var parsed interface{}
			err := json.Unmarshal([]byte(args[0].Inspect()), &parsed)
			return &Boolean{Value: err == nil}
		},
	}
	env.Set("json", jsonMap)

	// Modul csv (Manipulasi dan Serialisasi Data CSV)
	csvMap := &Map{Pairs: make(map[string]Object)}
	csvMap.Pairs["urai"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Array{Elements: []Object{}}
			}
			reader := csv.NewReader(strings.NewReader(args[0].Inspect()))
			records, err := reader.ReadAll()
			if err != nil {
				return &Array{Elements: []Object{}}
			}
			outer := make([]Object, len(records))
			for i, row := range records {
				inner := make([]Object, len(row))
				for j, cell := range row {
					inner[j] = &String{Value: cell}
				}
				outer[i] = &Array{Elements: inner}
			}
			return &Array{Elements: outer}
		},
	}
	csvMap.Pairs["buat"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			arr, isArr := args[0].(*Array)
			if !isArr {
				return &String{Value: ""}
			}
			var buf bytes.Buffer
			writer := csv.NewWriter(&buf)
			for _, rowObj := range arr.Elements {
				if rowArr, ok := rowObj.(*Array); ok {
					record := make([]string, len(rowArr.Elements))
					for j, cell := range rowArr.Elements {
						record[j] = cell.Inspect()
					}
					writer.Write(record)
				}
			}
			writer.Flush()
			return &String{Value: strings.TrimRight(buf.String(), "\r\n")}
		},
	}
	env.Set("csv", csvMap)

	// Modul validasi (Pemeriksaan Format Data Dasar)
	validasiMap := &Map{Pairs: make(map[string]Object)}
	validasiEmailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	validasiMap.Pairs["apakahEmail"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			str := args[0].Inspect()
			if sObj, ok := args[0].(*String); ok {
				str = sObj.Value
			}
			matched := validasiEmailRegex.MatchString(str)
			if matched {
				return TRUE
			}
			return FALSE
		},
	}
	validasiMap.Pairs["apakahURL"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			str := args[0].Inspect()
			if sObj, ok := args[0].(*String); ok {
				str = sObj.Value
			}
			u, err := url.ParseRequestURI(str)
			if err != nil || u.Scheme == "" || u.Host == "" {
				return FALSE
			}
			return TRUE
		},
	}
	validasiMap.Pairs["apakahIP"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			str := args[0].Inspect()
			if sObj, ok := args[0].(*String); ok {
				str = sObj.Value
			}
			if net.ParseIP(str) != nil {
				return TRUE
			}
			return FALSE
		},
	}
	validasiMap.Pairs["apakahAlfabet"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			s := args[0].Inspect()
			if sObj, ok := args[0].(*String); ok {
				s = sObj.Value
			}
			if len(s) == 0 {
				return FALSE
			}
			for _, r := range s {
				if !unicode.IsLetter(r) {
					return FALSE
				}
			}
			return TRUE
		},
	}
	validasiMap.Pairs["apakahNumerik"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			s := args[0].Inspect()
			if sObj, ok := args[0].(*String); ok {
				s = sObj.Value
			}
			if len(s) == 0 {
				return FALSE
			}
			for _, r := range s {
				if !unicode.IsDigit(r) {
					return FALSE
				}
			}
			return TRUE
		},
	}
	validasiMap.Pairs["apakahAlfanumerik"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			s := args[0].Inspect()
			if sObj, ok := args[0].(*String); ok {
				s = sObj.Value
			}
			if len(s) == 0 {
				return FALSE
			}
			for _, r := range s {
				if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
					return FALSE
				}
			}
			return TRUE
		},
	}
	validasiMap.Pairs["apakahPanjang"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 3 {
				return FALSE
			}
			s := args[0].Inspect()
			if sObj, ok := args[0].(*String); ok {
				s = sObj.Value
			}
			minLen := int64(0)
			maxLen := int64(math.MaxInt64)
			if i, ok := args[1].(*Integer); ok {
				minLen = i.Value
			}
			if i, ok := args[2].(*Integer); ok {
				maxLen = i.Value
			}
			l := int64(len(s))
			return &Boolean{Value: l >= minLen && l <= maxLen}
		},
	}
	validasiMap.Pairs["apakahJSON"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			s := args[0].Inspect()
			if sObj, ok := args[0].(*String); ok {
				s = sObj.Value
			}
			s = strings.TrimSpace(s)
			if len(s) < 2 {
				return FALSE
			}
			return &Boolean{Value: json.Valid([]byte(s))}
		},
	}
	validasiMap.Pairs["apakahDiRentang"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 3 {
				return FALSE
			}
			var val, minVal, maxVal float64
			toNum := func(o Object) float64 {
				if i, ok := o.(*Integer); ok {
					return float64(i.Value)
				} else if f, ok := o.(*Float); ok {
					return f.Value
				}
				return 0
			}
			val = toNum(args[0])
			minVal = toNum(args[1])
			maxVal = toNum(args[2])
			return &Boolean{Value: val >= minVal && val <= maxVal}
		},
	}
	validasiMap.Pairs["apakahPositif"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			if i, ok := args[0].(*Integer); ok {
				return &Boolean{Value: i.Value > 0}
			} else if f, ok := args[0].(*Float); ok {
				return &Boolean{Value: f.Value > 0}
			}
			return FALSE
		},
	}
	validasiMap.Pairs["apakahNegatif"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			if i, ok := args[0].(*Integer); ok {
				return &Boolean{Value: i.Value < 0}
			} else if f, ok := args[0].(*Float); ok {
				return &Boolean{Value: f.Value < 0}
			}
			return FALSE
		},
	}
	validasiMap.Pairs["apakahGenap"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			if i, ok := args[0].(*Integer); ok {
				return &Boolean{Value: i.Value%2 == 0}
			}
			return FALSE
		},
	}
	validasiMap.Pairs["apakahGanjil"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			if i, ok := args[0].(*Integer); ok {
				return &Boolean{Value: i.Value%2 != 0}
			}
			return FALSE
		},
	}
	env.Set("validasi", validasiMap)

	// Modul catatan (Logging Terstruktur dengan Timestamp)
	catatanMap := &Map{Pairs: make(map[string]Object)}
	catatanMap.Pairs["info"] = &Builtin{
		Fn: func(args ...Object) Object {
			ts := time.Now().Format("2006-01-02 15:04:05")
			items := []string{}
			for _, a := range args {
				items = append(items, a.Inspect())
			}
			fmt.Printf("[INFO] %s: %s\n", ts, strings.Join(items, " "))
			return NIL
		},
	}
	catatanMap.Pairs["peringatan"] = &Builtin{
		Fn: func(args ...Object) Object {
			ts := time.Now().Format("2006-01-02 15:04:05")
			items := []string{}
			for _, a := range args {
				items = append(items, a.Inspect())
			}
			fmt.Printf("[PERINGATAN] %s: %s\n", ts, strings.Join(items, " "))
			return NIL
		},
	}
	catatanMap.Pairs["galat"] = &Builtin{
		Fn: func(args ...Object) Object {
			ts := time.Now().Format("2006-01-02 15:04:05")
			items := []string{}
			for _, a := range args {
				items = append(items, a.Inspect())
			}
			fmt.Printf("[GALAT] %s: %s\n", ts, strings.Join(items, " "))
			return NIL
		},
	}
	env.Set("catatan", catatanMap)

	// Modul kompresi (Gzip)
	kompresiMap := &Map{Pairs: make(map[string]Object)}
	kompresiMap.Pairs["gzip"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			var buf bytes.Buffer
			gw := gzip.NewWriter(&buf)
			_, err := gw.Write([]byte(args[0].Inspect()))
			if err != nil {
				return &String{Value: ""}
			}
			_ = gw.Close()
			return &String{Value: base64.StdEncoding.EncodeToString(buf.Bytes())}
		},
	}
	kompresiMap.Pairs["ungzip"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: ""}
			}
			data, err := base64.StdEncoding.DecodeString(args[0].Inspect())
			if err != nil {
				return &String{Value: ""}
			}
			gr, err := gzip.NewReader(bytes.NewReader(data))
			if err != nil {
				return &String{Value: ""}
			}
			defer gr.Close()
			res, err := io.ReadAll(gr)
			if err != nil {
				return &String{Value: ""}
			}
			return &String{Value: string(res)}
		},
	}
	kompresiMap.Pairs["buatZip"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			sourcePath := args[0].Inspect()
			zipFilePath := args[1].Inspect()

			zipFile, err := os.Create(zipFilePath)
			if err != nil {
				return FALSE
			}
			defer zipFile.Close()

			zw := zip.NewWriter(zipFile)
			defer zw.Close()

			info, err := os.Stat(sourcePath)
			if err != nil {
				return FALSE
			}

			if !info.IsDir() {
				fileToZip, err := os.Open(sourcePath)
				if err != nil {
					return FALSE
				}
				defer fileToZip.Close()
				w, err := zw.Create(filepath.Base(sourcePath))
				if err != nil {
					return FALSE
				}
				_, err = io.Copy(w, fileToZip)
				return &Boolean{Value: err == nil}
			}

			err = filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if path == sourcePath {
					return nil
				}
				relPath, err := filepath.Rel(sourcePath, path)
				if err != nil {
					return err
				}
				if info.IsDir() {
					_, err := zw.Create(relPath + "/")
					return err
				}
				w, err := zw.Create(relPath)
				if err != nil {
					return err
				}
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()
				_, err = io.Copy(w, f)
				return err
			})
			return &Boolean{Value: err == nil}
		},
	}
	kompresiMap.Pairs["ekstrakZip"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			zipFilePath := args[0].Inspect()
			destDir := args[1].Inspect()

			r, err := zip.OpenReader(zipFilePath)
			if err != nil {
				return FALSE
			}
			defer r.Close()

			_ = os.MkdirAll(destDir, 0755)

			for _, f := range r.File {
				fpath := filepath.Join(destDir, f.Name)
				if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
					continue
				}
				if f.FileInfo().IsDir() {
					_ = os.MkdirAll(fpath, os.ModePerm)
					continue
				}
				if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
					return FALSE
				}
				outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
				if err != nil {
					return FALSE
				}
				rc, err := f.Open()
				if err != nil {
					outFile.Close()
					return FALSE
				}
				_, err = io.Copy(outFile, rc)
				outFile.Close()
				rc.Close()
				if err != nil {
					return FALSE
				}
			}
			return TRUE
		},
	}
	env.Set("kompresi", kompresiMap)

	// Modul peta (Utilitas Kamus / Map)
	petaMap := &Map{Pairs: make(map[string]Object)}
	petaMap.Pairs["kunci"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Array{Elements: []Object{}}
			}
			m, isMap := args[0].(*Map)
			if !isMap {
				return &Array{Elements: []Object{}}
			}
			keys := make([]string, 0, len(m.Pairs))
			for k := range m.Pairs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			res := make([]Object, len(keys))
			for i, k := range keys {
				res[i] = &String{Value: k}
			}
			return &Array{Elements: res}
		},
	}
	petaMap.Pairs["nilai"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Array{Elements: []Object{}}
			}
			m, isMap := args[0].(*Map)
			if !isMap {
				return &Array{Elements: []Object{}}
			}
			keys := make([]string, 0, len(m.Pairs))
			for k := range m.Pairs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			res := make([]Object, len(keys))
			for i, k := range keys {
				res[i] = m.Pairs[k]
			}
			return &Array{Elements: res}
		},
	}
	petaMap.Pairs["ada"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			m, isMap := args[0].(*Map)
			if !isMap {
				return FALSE
			}
			_, exists := m.Pairs[args[1].Inspect()]
			return &Boolean{Value: exists}
		},
	}
	petaMap.Pairs["gabung"] = &Builtin{
		Fn: func(args ...Object) Object {
			newMap := &Map{Pairs: make(map[string]Object)}
			for _, a := range args {
				if m, ok := a.(*Map); ok {
					for k, v := range m.Pairs {
						newMap.Pairs[k] = v
					}
				}
			}
			return newMap
		},
	}
	petaMap.Pairs["salin"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Map{Pairs: make(map[string]Object)}
			}
			m, isMap := args[0].(*Map)
			if !isMap {
				return args[0]
			}
			newMap := &Map{Pairs: make(map[string]Object)}
			for k, v := range m.Pairs {
				newMap.Pairs[k] = v
			}
			return newMap
		},
	}
	petaMap.Pairs["ambil"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			m, isMap := args[0].(*Map)
			if !isMap {
				return NIL
			}
			key := args[1].Inspect()
			if val, exists := m.Pairs[key]; exists {
				return val
			}
			if len(args) >= 3 {
				return args[2]
			}
			return NIL
		},
	}
	petaMap.Pairs["balik"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Map{Pairs: make(map[string]Object)}
			}
			m, isMap := args[0].(*Map)
			if !isMap {
				return args[0]
			}
			newMap := &Map{Pairs: make(map[string]Object)}
			for k, v := range m.Pairs {
				newMap.Pairs[v.Inspect()] = &String{Value: k}
			}
			return newMap
		},
	}
	petaMap.Pairs["saring"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Map{Pairs: make(map[string]Object)}
			}
			m, isMap := args[0].(*Map)
			if !isMap {
				return &Map{Pairs: make(map[string]Object)}
			}
			fn := args[1]
			newMap := &Map{Pairs: make(map[string]Object)}
			for k, v := range m.Pairs {
				res := applyFunction(fn, []Object{&String{Value: k}, v}, env)
				if isTruthy(res) {
					newMap.Pairs[k] = v
				}
			}
			return newMap
		},
	}
	petaMap.Pairs["petakan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Map{Pairs: make(map[string]Object)}
			}
			m, isMap := args[0].(*Map)
			if !isMap {
				return &Map{Pairs: make(map[string]Object)}
			}
			fn := args[1]
			newMap := &Map{Pairs: make(map[string]Object)}
			for k, v := range m.Pairs {
				res := applyFunction(fn, []Object{&String{Value: k}, v}, env)
				newMap.Pairs[k] = res
			}
			return newMap
		},
	}
	env.Set("peta", petaMap)

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
				prompt := args[0].Inspect()
				if len(args) >= 2 {
					prompt = fmt.Sprintf("%s [%s]: ", strings.TrimRight(prompt, " :"), args[1].Inspect())
				}
				fmt.Print(prompt)
			}
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				input := strings.TrimRight(scanner.Text(), "\r\n")
				if input == "" && len(args) >= 2 {
					return &String{Value: args[1].Inspect()}
				}
				return &String{Value: input}
			}
			if len(args) >= 2 {
				return &String{Value: args[1].Inspect()}
			}
			return &String{Value: ""}
		},
	}
	masukanMap.Pairs["konfirmasi"] = &Builtin{
		Fn: func(args ...Object) Object {
			prompt := "Apakah Anda yakin? (y/n): "
			if len(args) >= 1 {
				prompt = fmt.Sprintf("%s (y/n): ", strings.TrimRight(args[0].Inspect(), " :?"))
			}
			fmt.Print(prompt)
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				ans := strings.ToLower(strings.TrimSpace(scanner.Text()))
				if ans == "y" || ans == "ya" || ans == "yes" || ans == "benar" || ans == "1" {
					return TRUE
				}
			}
			return FALSE
		},
	}
	masukanMap.Pairs["rahasia"] = &Builtin{
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

	// Modul terminal (Warna ANSI & Utilitas Terminal CLI)
	terminalMap := &Map{Pairs: make(map[string]Object)}
	wrapColor := func(prefix, suffix string) *Builtin {
		return &Builtin{
			Fn: func(args ...Object) Object {
				parts := []string{}
				for _, a := range args {
					parts = append(parts, a.Inspect())
				}
				return &String{Value: prefix + strings.Join(parts, " ") + suffix}
			},
		}
	}
	terminalMap.Pairs["hitam"] = wrapColor("\033[30m", "\033[0m")
	terminalMap.Pairs["merah"] = wrapColor("\033[31m", "\033[0m")
	terminalMap.Pairs["hijau"] = wrapColor("\033[32m", "\033[0m")
	terminalMap.Pairs["kuning"] = wrapColor("\033[33m", "\033[0m")
	terminalMap.Pairs["biru"] = wrapColor("\033[34m", "\033[0m")
	terminalMap.Pairs["ungu"] = wrapColor("\033[35m", "\033[0m")
	terminalMap.Pairs["sian"] = wrapColor("\033[36m", "\033[0m")
	terminalMap.Pairs["putih"] = wrapColor("\033[37m", "\033[0m")
	terminalMap.Pairs["tebal"] = wrapColor("\033[1m", "\033[0m")
	terminalMap.Pairs["garisBawah"] = wrapColor("\033[4m", "\033[0m")
	terminalMap.Pairs["bersihkan"] = &Builtin{
		Fn: func(args ...Object) Object {
			fmt.Print("\033[2J\033[H")
			return NIL
		},
	}
	terminalMap.Pairs["sembunyikanKursor"] = &Builtin{
		Fn: func(args ...Object) Object {
			fmt.Print("\033[?25l")
			return NIL
		},
	}
	terminalMap.Pairs["tampilkanKursor"] = &Builtin{
		Fn: func(args ...Object) Object {
			fmt.Print("\033[?25h")
			return NIL
		},
	}
	terminalMap.Pairs["progres"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			var cur, total float64
			if i, ok := args[0].(*Integer); ok {
				cur = float64(i.Value)
			} else if f, ok := args[0].(*Float); ok {
				cur = f.Value
			}
			if i, ok := args[1].(*Integer); ok {
				total = float64(i.Value)
			} else if f, ok := args[1].(*Float); ok {
				total = f.Value
			}
			width := 20
			if len(args) >= 3 {
				if i, ok := args[2].(*Integer); ok && i.Value > 0 {
					width = int(i.Value)
				}
			}
			if total <= 0 {
				total = 1
			}
			ratio := cur / total
			if ratio > 1 {
				ratio = 1
			}
			if ratio < 0 {
				ratio = 0
			}
			filled := int(ratio * float64(width))
			empty := width - filled
			bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
			persen := int(ratio * 100)
			fmt.Printf("\r[%s] %d%% (%.0f/%.0f)", bar, persen, cur, total)
			if cur >= total {
				fmt.Println()
			}
			return NIL
		},
	}
	terminalMap.Pairs["tabel"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			headersArr, ok1 := args[0].(*Array)
			rowsArr, ok2 := args[1].(*Array)
			if !ok1 || !ok2 {
				return NIL
			}
			headers := []string{}
			for _, h := range headersArr.Elements {
				headers = append(headers, h.Inspect())
			}
			rows := [][]string{}
			for _, rObj := range rowsArr.Elements {
				if rArr, ok := rObj.(*Array); ok {
					row := []string{}
					for _, c := range rArr.Elements {
						row = append(row, c.Inspect())
					}
					rows = append(rows, row)
				}
			}
			colWidths := make([]int, len(headers))
			for i, h := range headers {
				colWidths[i] = len(h)
			}
			for _, row := range rows {
				for i, cell := range row {
					if i < len(colWidths) && len(cell) > colWidths[i] {
						colWidths[i] = len(cell)
					}
				}
			}
			sepParts := []string{}
			for _, w := range colWidths {
				sepParts = append(sepParts, strings.Repeat("-", w+2))
			}
			sep := "+" + strings.Join(sepParts, "+") + "+"
			fmt.Println(sep)
			headerParts := []string{}
			for i, h := range headers {
				headerParts = append(headerParts, fmt.Sprintf(" %-*s ", colWidths[i], h))
			}
			fmt.Println("|" + strings.Join(headerParts, "|") + "|")
			fmt.Println(sep)
			for _, row := range rows {
				cellParts := []string{}
				for i := 0; i < len(colWidths); i++ {
					val := ""
					if i < len(row) {
						val = row[i]
					}
					cellParts = append(cellParts, fmt.Sprintf(" %-*s ", colWidths[i], val))
				}
				fmt.Println("|" + strings.Join(cellParts, "|") + "|")
			}
			fmt.Println(sep)
			return NIL
		},
	}
	env.Set("terminal", terminalMap)

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

	// Bawaan: tutup (Go close() pada Kanal)
	env.Set("tutup", &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return FALSE
			}
			if chObj, ok := args[0].(*ChannelObject); ok {
				close(chObj.Ch)
				return TRUE
			} else if m, ok := args[0].(*Map); ok {
				if tutupFn, ok := m.Pairs["tutup"].(*Builtin); ok {
					return tutupFn.Fn()
				}
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

	// Bawaan: kapasitas (cap() ala Go)
	env.Set("kapasitas", &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return &Integer{Value: 0}
			}
			switch arg := args[0].(type) {
			case *Array:
				return &Integer{Value: int64(cap(arg.Elements))}
			case *Map:
				if capObj, ok := arg.Pairs["_kapasitas"]; ok {
					if i, isInt := capObj.(*Integer); isInt {
						return i
					}
				}
				return &Integer{Value: int64(len(arg.Pairs))}
			}
			return &Integer{Value: 0}
		},
	})

	// Bawaan: salin (copy(dst, src) ala Go)
	env.Set("salin", &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Integer{Value: 0}
			}
			dst, okDst := args[0].(*Array)
			src, okSrc := args[1].(*Array)
			if !okDst || !okSrc {
				return &Integer{Value: 0}
			}
			n := len(dst.Elements)
			if len(src.Elements) < n {
				n = len(src.Elements)
			}
			for i := 0; i < n; i++ {
				dst.Elements[i] = src.Elements[i]
			}
			return &Integer{Value: int64(n)}
		},
	})

	// Bawaan: buat (make() ala Go)
	env.Set("buat", &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return NIL
			}
			tipe := strings.ToLower(strings.TrimSpace(args[0].Inspect()))
			switch tipe {
			case "daftar", "larik", "array", "slice":
				panjang := int64(0)
				kapasitas := int64(0)
				if len(args) >= 2 {
					if p, ok := args[1].(*Integer); ok {
						panjang = p.Value
					}
				}
				kapasitas = panjang
				if len(args) >= 3 {
					if k, ok := args[2].(*Integer); ok && k.Value > panjang {
						kapasitas = k.Value
					}
				}
				elems := make([]Object, panjang, kapasitas)
				for i := int64(0); i < panjang; i++ {
					elems[i] = &Integer{Value: 0}
				}
				return &Array{Elements: elems}
			case "peta", "map":
				m := &Map{Pairs: make(map[string]Object)}
				if len(args) >= 2 {
					if capVal, ok := args[1].(*Integer); ok {
						m.Pairs["_kapasitas"] = capVal
					}
				}
				return m
			case "kanal", "channel":
				capVal := int64(0)
				if len(args) >= 2 {
					if c, ok := args[1].(*Integer); ok {
						capVal = c.Value
					}
				}
				ch := make(chan Object, capVal)
				chObj := &ChannelObject{Ch: ch}
				return chObj
			default:
				return NIL
			}
		},
	})

	// Modul alamat (Pointer Aman ala Go)
	alamatMap := &Map{Pairs: make(map[string]Object)}
	alamatMap.Pairs["pointer"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return NIL
			}
			ptrMap := &Map{Pairs: make(map[string]Object)}
			ptrMap.Pairs["_isPointer"] = TRUE
			ptrMap.Pairs["nilai"] = args[0]
			ptrMap.Pairs["ambil"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					return ptrMap.Pairs["nilai"]
				},
			}
			ptrMap.Pairs["atur"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					if len(mArgs) > 0 {
						ptrMap.Pairs["nilai"] = mArgs[0]
					}
					return NIL
				},
			}
			return ptrMap
		},
	}
	alamatMap.Pairs["dari"] = alamatMap.Pairs["pointer"]
	alamatMap.Pairs["baca"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return NIL
			}
			if m, ok := args[0].(*Map); ok {
				if val, exists := m.Pairs["nilai"]; exists {
					return val
				}
			}
			return args[0]
		},
	}
	alamatMap.Pairs["ubah"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			if m, ok := args[0].(*Map); ok {
				m.Pairs["nilai"] = args[1]
				return TRUE
			}
			return FALSE
		},
	}
	env.Set("alamat", alamatMap)

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
			chInst.Pairs["_rawChan"] = &ChannelObject{Ch: ch}
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
					val, ok := <-ch
					if !ok {
						return &ReturnValue{Value: NIL, Values: []Object{NIL, FALSE}}
					}
					if val == nil {
						val = NIL
					}
					return &ReturnValue{Value: val, Values: []Object{val, TRUE}}
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
	kanalMap.Pairs["pilih"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) == 0 {
				return NIL
			}
			var chList []*Map
			var defaultFn *Function
			timeoutSec := float64(-1)
			var timeoutFn *Function

			for _, a := range args {
				if m, ok := a.(*Map); ok {
					// Apakah ini opsi kasus kanal?
					if _, hasKanal := m.Pairs["kanal"]; hasKanal {
						chList = append(chList, m)
					} else if df, hasBawaan := m.Pairs["bawaan"]; hasBawaan {
						if fn, isFn := df.(*Function); isFn {
							defaultFn = fn
						}
					} else if _, hasWaktu := m.Pairs["batasWaktu"]; hasWaktu {
						if tVal, ok := m.Pairs["batasWaktu"].(*Integer); ok {
							timeoutSec = float64(tVal.Value) / 1000.0
						} else if tValF, ok := m.Pairs["batasWaktu"].(*Float); ok {
							timeoutSec = tValF.Value / 1000.0
						}
						if fn, isFn := m.Pairs["aksi"].(*Function); isFn {
							timeoutFn = fn
						}
					}
				}
			}

			// Jika ada non-blocking default, coba cek sekali
			if defaultFn != nil && timeoutSec < 0 {
				for _, chM := range chList {
					if chRaw, ok := chM.Pairs["kanal"].(*Map); ok {
						if terimaFn, ok := chRaw.Pairs["cobaTerima"].(*Builtin); ok {
							res := terimaFn.Fn()
							if ret, ok := res.(*ReturnValue); ok && len(ret.Values) > 1 {
								if ret.Values[1] == TRUE {
									val := ret.Values[0]
									if aksi, hasAksi := chM.Pairs["aksi"].(*Function); hasAksi {
										return applyFunction(aksi, []Object{val}, env)
									}
									return val
								}
							}
						}
					}
				}
				return applyFunction(defaultFn, []Object{}, env)
			}

			// Polling wait dengan batas waktu jika ada
			start := time.Now()
			for {
				for _, chM := range chList {
					if chRaw, ok := chM.Pairs["kanal"].(*Map); ok {
						if terimaFn, ok := chRaw.Pairs["cobaTerima"].(*Builtin); ok {
							res := terimaFn.Fn()
							if ret, ok := res.(*ReturnValue); ok && len(ret.Values) > 1 {
								if ret.Values[1] == TRUE {
									val := ret.Values[0]
									if aksi, hasAksi := chM.Pairs["aksi"].(*Function); hasAksi {
										return applyFunction(aksi, []Object{val}, env)
									}
									return val
								}
							}
						}
					}
				}
				if timeoutSec >= 0 && time.Since(start).Seconds() >= timeoutSec {
					if timeoutFn != nil {
						return applyFunction(timeoutFn, []Object{}, env)
					}
					return NIL
				}
				time.Sleep(2 * time.Millisecond)
			}
			return NIL
		},
	}
	env.Set("kanal", kanalMap)

	// Modul atomik (Operasi Lock-free Atomic Concurrency ala sync/atomic Go)
	atomikMap := &Map{Pairs: make(map[string]Object)}
	atomikMap.Pairs["tambah"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			delta := int64(1)
			if d, ok := args[1].(*Integer); ok {
				delta = d.Value
			}
			if ptr, ok := args[0].(*Integer); ok {
				newVal := atomic.AddInt64(&ptr.Value, delta)
				return &Integer{Value: newVal}
			}
			return NIL
		},
	}
	atomikMap.Pairs["baca"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Integer{Value: 0}
			}
			if ptr, ok := args[0].(*Integer); ok {
				val := atomic.LoadInt64(&ptr.Value)
				return &Integer{Value: val}
			}
			return &Integer{Value: 0}
		},
	}
	atomikMap.Pairs["simpan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			val := int64(0)
			if v, ok := args[1].(*Integer); ok {
				val = v.Value
			}
			if ptr, ok := args[0].(*Integer); ok {
				atomic.StoreInt64(&ptr.Value, val)
				return &Integer{Value: val}
			}
			return NIL
		},
	}
	atomikMap.Pairs["tukar"] = &Builtin{
		Fn: func(args ...Object) Object {
			// CompareAndSwap: atomik.tukar(ptr, nilaiLama, nilaiBaru)
			if len(args) < 3 {
				return FALSE
			}
			oldVal := int64(0)
			newVal := int64(0)
			if o, ok := args[1].(*Integer); ok {
				oldVal = o.Value
			}
			if n, ok := args[2].(*Integer); ok {
				newVal = n.Value
			}
			if ptr, ok := args[0].(*Integer); ok {
				swapped := atomic.CompareAndSwapInt64(&ptr.Value, oldVal, newVal)
				if swapped {
					return TRUE
				}
				return FALSE
			}
			return FALSE
		},
	}
	env.Set("atomik", atomikMap)

	// Modul konteks (Manajemen Batas Waktu dan Pembatalan ala context.Context Go)
	konteksMap := &Map{Pairs: make(map[string]Object)}
	konteksMap.Pairs["latarBelakang"] = &Builtin{
		Fn: func(args ...Object) Object {
			ctxInst := &Map{Pairs: make(map[string]Object)}
			ctxInst.Pairs["selesai"] = &Boolean{Value: false}
			ctxInst.Pairs["apakahBatal"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					return ctxInst.Pairs["selesai"]
				},
			}
			return ctxInst
		},
	}
	konteksMap.Pairs["denganBatasWaktu"] = &Builtin{
		Fn: func(args ...Object) Object {
			ms := int64(1000)
			if len(args) > 0 {
				if i, ok := args[0].(*Integer); ok {
					ms = i.Value
				} else if f, ok := args[0].(*Float); ok {
					ms = int64(f.Value)
				}
			}
			ctxInst := &Map{Pairs: make(map[string]Object)}
			ctxInst.Pairs["selesai"] = &Boolean{Value: false}
			ctxInst.Pairs["alasan"] = &String{Value: ""}

			batalFn := &Builtin{
				Fn: func(mArgs ...Object) Object {
					ctxInst.Pairs["selesai"] = TRUE
					ctxInst.Pairs["alasan"] = &String{Value: "dibatalkan secara manual"}
					return NIL
				},
			}

			go func() {
				time.Sleep(time.Duration(ms) * time.Millisecond)
				if ctxInst.Pairs["selesai"] == FALSE {
					ctxInst.Pairs["selesai"] = TRUE
					ctxInst.Pairs["alasan"] = &String{Value: "batas waktu terlampaui"}
				}
			}()

			ctxInst.Pairs["batal"] = batalFn
			ctxInst.Pairs["apakahBatal"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					return ctxInst.Pairs["selesai"]
				},
			}
			ctxInst.Pairs["galat"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					return ctxInst.Pairs["alasan"]
				},
			}

			// Mengembalikan pasangan [ctx, batal]
			return &ReturnValue{Value: ctxInst, Values: []Object{ctxInst, batalFn}}
		},
	}
	env.Set("konteks", konteksMap)

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
	larikMap.Pairs["unik"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Array{Elements: []Object{}}
			}
			arr, isArr := args[0].(*Array)
			if !isArr {
				return args[0]
			}
			seen := make(map[string]bool)
			res := []Object{}
			for _, elem := range arr.Elements {
				key := elem.Inspect()
				if !seen[key] {
					seen[key] = true
					res = append(res, elem)
				}
			}
			return &Array{Elements: res}
		},
	}
	larikMap.Pairs["irisan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Array{Elements: []Object{}}
			}
			arr1, isArr1 := args[0].(*Array)
			arr2, isArr2 := args[1].(*Array)
			if !isArr1 || !isArr2 {
				return &Array{Elements: []Object{}}
			}
			map2 := make(map[string]bool)
			for _, e := range arr2.Elements {
				map2[e.Inspect()] = true
			}
			res := []Object{}
			seen := make(map[string]bool)
			for _, e := range arr1.Elements {
				k := e.Inspect()
				if map2[k] && !seen[k] {
					seen[k] = true
					res = append(res, e)
				}
			}
			return &Array{Elements: res}
		},
	}
	larikMap.Pairs["selisih"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Array{Elements: []Object{}}
			}
			arr1, isArr1 := args[0].(*Array)
			arr2, isArr2 := args[1].(*Array)
			if !isArr1 || !isArr2 {
				return &Array{Elements: []Object{}}
			}
			map2 := make(map[string]bool)
			for _, e := range arr2.Elements {
				map2[e.Inspect()] = true
			}
			res := []Object{}
			seen := make(map[string]bool)
			for _, e := range arr1.Elements {
				k := e.Inspect()
				if !map2[k] && !seen[k] {
					seen[k] = true
					res = append(res, e)
				}
			}
			return &Array{Elements: res}
		},
	}
	larikMap.Pairs["apakahSama"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			arr1, isArr1 := args[0].(*Array)
			arr2, isArr2 := args[1].(*Array)
			if !isArr1 || !isArr2 {
				return FALSE
			}
			if len(arr1.Elements) != len(arr2.Elements) {
				return FALSE
			}
			for i := range arr1.Elements {
				if !objectsEqual(arr1.Elements[i], arr2.Elements[i]) {
					return FALSE
				}
			}
			return TRUE
		},
	}
	larikMap.Pairs["gabungan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Array{Elements: []Object{}}
			}
			arr1, isArr1 := args[0].(*Array)
			arr2, isArr2 := args[1].(*Array)
			if !isArr1 || !isArr2 {
				return &Array{Elements: []Object{}}
			}
			seen := make(map[string]bool)
			res := []Object{}
			for _, e := range arr1.Elements {
				k := e.Inspect()
				if !seen[k] {
					seen[k] = true
					res = append(res, e)
				}
			}
			for _, e := range arr2.Elements {
				k := e.Inspect()
				if !seen[k] {
					seen[k] = true
					res = append(res, e)
				}
			}
			return &Array{Elements: res}
		},
	}
	larikMap.Pairs["iris"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Array{Elements: []Object{}}
			}
			arr, isArr := args[0].(*Array)
			if !isArr {
				return args[0]
			}
			start := int64(0)
			end := int64(len(arr.Elements))
			if len(args) >= 2 {
				if s, ok := args[1].(*Integer); ok {
					start = s.Value
				}
			}
			if len(args) >= 3 {
				if e, ok := args[2].(*Integer); ok {
					end = e.Value
				}
			}
			if start < 0 {
				start = 0
			}
			if end > int64(len(arr.Elements)) {
				end = int64(len(arr.Elements))
			}
			if start > end {
				return &Array{Elements: []Object{}}
			}
			return &Array{Elements: arr.Elements[start:end]}
		},
	}
	larikMap.Pairs["mengandung"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			arr, isArr := args[0].(*Array)
			if !isArr {
				return FALSE
			}
			target := args[1]
			for _, e := range arr.Elements {
				if objectsEqual(e, target) {
					return TRUE
				}
			}
			return FALSE
		},
	}
	larikMap.Pairs["indeks"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Integer{Value: -1}
			}
			arr, isArr := args[0].(*Array)
			if !isArr {
				return &Integer{Value: -1}
			}
			target := args[1]
			for i, e := range arr.Elements {
				if objectsEqual(e, target) {
					return &Integer{Value: int64(i)}
				}
			}
			return &Integer{Value: -1}
		},
	}
	larikMap.Pairs["rentang"] = &Builtin{
		Fn: func(args ...Object) Object {
			start := int64(0)
			end := int64(0)
			step := int64(1)
			if len(args) == 1 {
				if i, ok := args[0].(*Integer); ok {
					end = i.Value
				}
			} else if len(args) >= 2 {
				if i, ok := args[0].(*Integer); ok {
					start = i.Value
				}
				if i, ok := args[1].(*Integer); ok {
					end = i.Value
				}
				if len(args) >= 3 {
					if i, ok := args[2].(*Integer); ok && i.Value != 0 {
						step = i.Value
					}
				}
			}
			res := []Object{}
			if step > 0 {
				for v := start; v < end; v += step {
					res = append(res, &Integer{Value: v})
				}
			} else {
				for v := start; v > end; v += step {
					res = append(res, &Integer{Value: v})
				}
			}
			return &Array{Elements: res}
		},
	}
	larikMap.Pairs["gabungSemua"] = &Builtin{
		Fn: func(args ...Object) Object {
			res := []Object{}
			for _, a := range args {
				if arr, ok := a.(*Array); ok {
					res = append(res, arr.Elements...)
				}
			}
			return &Array{Elements: res}
		},
	}
	larikMap.Pairs["rata"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Array{Elements: []Object{}}
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return args[0]
			}
			res := []Object{}
			for _, item := range arr.Elements {
				if subArr, isSub := item.(*Array); isSub {
					res = append(res, subArr.Elements...)
				} else {
					res = append(res, item)
				}
			}
			return &Array{Elements: res}
		},
	}
	larikMap.Pairs["ratakan"] = larikMap.Pairs["rata"]
	larikMap.Pairs["ambilAcak"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Array{Elements: []Object{}}
			}
			arr, ok := args[0].(*Array)
			if !ok || len(arr.Elements) == 0 {
				return &Array{Elements: []Object{}}
			}
			count := 1
			if len(args) >= 2 {
				if i, isInt := args[1].(*Integer); isInt && i.Value > 0 {
					count = int(i.Value)
				}
			}
			if count > len(arr.Elements) {
				count = len(arr.Elements)
			}
			indices := mrand.Perm(len(arr.Elements))[:count]
			res := make([]Object, count)
			for i, idx := range indices {
				res[i] = arr.Elements[idx]
			}
			return &Array{Elements: res}
		},
	}
	larikMap.Pairs["pecah"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return args[0]
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return args[0]
			}
			size := int64(1)
			if i, isInt := args[1].(*Integer); isInt && i.Value > 0 {
				size = i.Value
			}
			chunks := []Object{}
			n := int64(len(arr.Elements))
			for i := int64(0); i < n; i += size {
				end := i + size
				if end > n {
					end = n
				}
				chunks = append(chunks, &Array{Elements: arr.Elements[i:end]})
			}
			return &Array{Elements: chunks}
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
	ujiMap.Pairs["tolakUkur"] = &Builtin{
		Fn: func(args ...Object) Object {
			// Benchmark: uji.tolakUkur("Nama Tolak Ukur", fungsi(b) { ... })
			if len(args) < 2 {
				return NIL
			}
			benchName := args[0].Inspect()
			fn, isFn := args[1].(*Function)
			if !isFn {
				return NIL
			}

			// Iterasi penyesuaian durasi adaptif ala testing.B Go (100, 1000, 10000, ...)
			bObj := &Map{Pairs: make(map[string]Object)}
			var iterCount int64 = 1000
			bObj.Pairs["N"] = &Integer{Value: iterCount}

			runtime.GC()
			var memStart, memEnd runtime.MemStats
			runtime.ReadMemStats(&memStart)

			start := time.Now()
			applyFunction(fn, []Object{bObj}, env)
			elapsed := time.Since(start)
			runtime.ReadMemStats(&memEnd)

			// Sesuaikan jika terlalu cepat (< 100ms)
			if elapsed < 100*time.Millisecond {
				iterCount = 100000
				bObj.Pairs["N"] = &Integer{Value: iterCount}
				runtime.GC()
				runtime.ReadMemStats(&memStart)
				start = time.Now()
				applyFunction(fn, []Object{bObj}, env)
				elapsed = time.Since(start)
				runtime.ReadMemStats(&memEnd)
			}

			nsPerOp := float64(elapsed.Nanoseconds()) / float64(iterCount)
			bytesAllocated := int64(0)
			if memEnd.TotalAlloc > memStart.TotalAlloc {
				bytesAllocated = int64(memEnd.TotalAlloc - memStart.TotalAlloc)
			}
			allocsCount := int64(0)
			if memEnd.Mallocs > memStart.Mallocs {
				allocsCount = int64(memEnd.Mallocs - memStart.Mallocs)
			}
			bytesPerOp := float64(bytesAllocated) / float64(iterCount)
			allocsPerOp := float64(allocsCount) / float64(iterCount)

			fmt.Printf("⏱️  [TOLAK UKUR] %-30s %10d iterasi\t%10.2f ns/op\t%8.0f B/op\t%6.0f allocs/op\n",
				benchName, iterCount, nsPerOp, bytesPerOp, allocsPerOp)

			resMap := &Map{Pairs: make(map[string]Object)}
			resMap.Pairs["nama"] = &String{Value: benchName}
			resMap.Pairs["iterasi"] = &Integer{Value: iterCount}
			resMap.Pairs["nsPerOp"] = &Float{Value: nsPerOp}
			resMap.Pairs["bytesPerOp"] = &Float{Value: bytesPerOp}
			resMap.Pairs["allocsPerOp"] = &Float{Value: allocsPerOp}
			return resMap
		},
	}
	env.Set("uji", ujiMap)

	// Modul waktu (Pengelolaan Tanggal dan Durasi Waktu)
	waktuMap := &Map{Pairs: make(map[string]Object)}
	waktuMap.Pairs["tidur"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return NIL
			}
			// Mendukung milidetik (integer) atau detik desimal (float)
			if f, ok := args[0].(*Float); ok {
				time.Sleep(time.Duration(f.Value * float64(time.Millisecond)))
				return NIL
			}
			if i, ok := args[0].(*Integer); ok {
				time.Sleep(time.Duration(i.Value) * time.Millisecond)
				return NIL
			}
			return NIL
		},
	}
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
	waktuMap.Pairs["tambahMenit"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			var secVal int64
			var menit int64
			if s, ok := args[0].(*Integer); ok {
				secVal = s.Value
			}
			if m, ok := args[1].(*Integer); ok {
				menit = m.Value
			}
			return &Integer{Value: secVal + menit*60}
		},
	}
	waktuMap.Pairs["tambahJam"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			var secVal int64
			var jam int64
			if s, ok := args[0].(*Integer); ok {
				secVal = s.Value
			}
			if j, ok := args[1].(*Integer); ok {
				jam = j.Value
			}
			return &Integer{Value: secVal + jam*3600}
		},
	}
	waktuMap.Pairs["selisihDetik"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Integer{Value: 0}
			}
			var s1, s2 int64
			if i1, ok := args[0].(*Integer); ok {
				s1 = i1.Value
			}
			if i2, ok := args[1].(*Integer); ok {
				s2 = i2.Value
			}
			diff := s1 - s2
			if diff < 0 {
				diff = -diff
			}
			return &Integer{Value: diff}
		},
	}
	waktuMap.Pairs["mulai"] = &Builtin{
		Fn: func(args ...Object) Object {
			return &Integer{Value: time.Now().UnixNano()}
		},
	}
	waktuMap.Pairs["lewat"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Float{Value: 0}
			}
			if i, ok := args[0].(*Integer); ok {
				elapsedNano := time.Now().UnixNano() - i.Value
				elapsedMs := float64(elapsedNano) / 1e6
				return &Float{Value: elapsedMs}
			}
			return &Float{Value: 0}
		},
	}
	waktuMap.Pairs["setelah"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			var ms int64 = 1000
			if i, ok := args[0].(*Integer); ok {
				ms = i.Value
			}
			fnObj, isFn := args[1].(*Function)
			if !isFn {
				return NIL
			}
			go func() {
				time.Sleep(time.Duration(ms) * time.Millisecond)
				applyFunction(fnObj, []Object{}, env)
			}()
			return NIL
		},
	}
	waktuMap.Pairs["setiap"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			var ms int64 = 1000
			if i, ok := args[0].(*Integer); ok {
				ms = i.Value
			}
			fnObj, isFn := args[1].(*Function)
			if !isFn {
				return NIL
			}
			stopCh := make(chan struct{})
			go func() {
				ticker := time.NewTicker(time.Duration(ms) * time.Millisecond)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						applyFunction(fnObj, []Object{}, env)
					case <-stopCh:
						return
					}
				}
			}()
			timerMap := &Map{Pairs: make(map[string]Object)}
			stopFn := &Builtin{
				Fn: func(mArgs ...Object) Object {
					select {
					case <-stopCh:
					default:
						close(stopCh)
					}
					return NIL
				},
			}
			timerMap.Pairs["hentikan"] = stopFn
			timerMap.Pairs["henti"] = stopFn
			timerMap.Pairs["stop"] = stopFn
			return timerMap
		},
	}
	waktuMap.Pairs["formatDurasi"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: "0ms"}
			}
			var ms int64
			if i, ok := args[0].(*Integer); ok {
				ms = i.Value
			} else if f, ok := args[0].(*Float); ok {
				ms = int64(f.Value)
			}
			if ms < 0 {
				ms = -ms
			}
			if ms < 1000 {
				return &String{Value: fmt.Sprintf("%dms", ms)}
			}
			totalSec := ms / 1000
			remMs := ms % 1000
			if totalSec < 60 {
				if remMs > 0 {
					return &String{Value: fmt.Sprintf("%.1fd", float64(ms)/1000.0)}
				}
				return &String{Value: fmt.Sprintf("%dd", totalSec)}
			}
			menit := totalSec / 60
			detik := totalSec % 60
			if menit < 60 {
				if detik > 0 {
					return &String{Value: fmt.Sprintf("%dm %dd", menit, detik)}
				}
				return &String{Value: fmt.Sprintf("%dm", menit)}
			}
			jam := menit / 60
			remMenit := menit % 60
			if remMenit > 0 {
				return &String{Value: fmt.Sprintf("%dj %dm", jam, remMenit)}
			}
			return &String{Value: fmt.Sprintf("%dj", jam)}
		},
	}
	waktuMap.Pairs["sejak"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Integer{Value: 0}
			}
			var startMili int64
			if i, ok := args[0].(*Integer); ok {
				startMili = i.Value
			}
			diff := time.Now().UnixMilli() - startMili
			if diff < 0 {
				diff = 0
			}
			return &Integer{Value: diff}
		},
	}
	env.Set("waktu", waktuMap)

	// Modul arsip (Kompresi & Dekompresi ZIP)
	arsipMap := &Map{Pairs: make(map[string]Object)}
	arsipMap.Pairs["buatZip"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			zipName, ok := args[0].(*String)
			if !ok {
				return FALSE
			}

			filesToZip := []string{}
			if arr, ok := args[1].(*Array); ok {
				for _, el := range arr.Elements {
					if s, ok := el.(*String); ok {
						filesToZip = append(filesToZip, s.Value)
					}
				}
			} else if s, ok := args[1].(*String); ok {
				filesToZip = append(filesToZip, s.Value)
			}

			zipFile, err := os.Create(zipName.Value)
			if err != nil {
				return FALSE
			}
			defer zipFile.Close()

			zipWriter := zip.NewWriter(zipFile)
			defer zipWriter.Close()

			for _, path := range filesToZip {
				info, err := os.Stat(path)
				if err != nil {
					continue
				}

				if info.IsDir() {
					_ = filepath.Walk(path, func(subPath string, subInfo os.FileInfo, walkErr error) error {
						if walkErr != nil || subInfo.IsDir() {
							return nil
						}
						relPath, _ := filepath.Rel(filepath.Dir(path), subPath)
						w, err := zipWriter.Create(filepath.ToSlash(relPath))
						if err != nil {
							return nil
						}
						f, err := os.Open(subPath)
						if err == nil {
							_, _ = io.Copy(w, f)
							f.Close()
						}
						return nil
					})
				} else {
					w, err := zipWriter.Create(filepath.Base(path))
					if err == nil {
						f, err := os.Open(path)
						if err == nil {
							_, _ = io.Copy(w, f)
							f.Close()
						}
					}
				}
			}
			return TRUE
		},
	}
	arsipMap.Pairs["ekstrakZip"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			zipPath, ok := args[0].(*String)
			if !ok {
				return FALSE
			}
			destFolder, ok := args[1].(*String)
			if !ok {
				return FALSE
			}

			r, err := zip.OpenReader(zipPath.Value)
			if err != nil {
				return FALSE
			}
			defer r.Close()

			_ = os.MkdirAll(destFolder.Value, 0755)

			for _, f := range r.File {
				fpath := filepath.Join(destFolder.Value, f.Name)
				// Cegah Zip Slip Vulnerability
				if !strings.HasPrefix(filepath.Clean(fpath), filepath.Clean(destFolder.Value)+string(os.PathSeparator)) && filepath.Clean(fpath) != filepath.Clean(destFolder.Value) {
					continue
				}

				if f.FileInfo().IsDir() {
					_ = os.MkdirAll(fpath, os.ModePerm)
					continue
				}

				_ = os.MkdirAll(filepath.Dir(fpath), os.ModePerm)
				outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
				if err != nil {
					continue
				}

				rc, err := f.Open()
				if err == nil {
					_, _ = io.Copy(outFile, rc)
					rc.Close()
				}
				outFile.Close()
			}
			return TRUE
		},
	}
	env.Set("arsip", arsipMap)

	// Modul angka (Operasi Bitwise & Representasi Basis Angka)
	angkaMap := &Map{Pairs: make(map[string]Object)}
	angkaMap.Pairs["biner"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: "0"}
			}
			var val int64
			if i, ok := args[0].(*Integer); ok {
				val = i.Value
			}
			return &String{Value: strconv.FormatInt(val, 2)}
		},
	}
	angkaMap.Pairs["heksa"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: "0"}
			}
			var val int64
			if i, ok := args[0].(*Integer); ok {
				val = i.Value
			}
			return &String{Value: strconv.FormatInt(val, 16)}
		},
	}
	angkaMap.Pairs["oktal"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &String{Value: "0"}
			}
			var val int64
			if i, ok := args[0].(*Integer); ok {
				val = i.Value
			}
			return &String{Value: strconv.FormatInt(val, 8)}
		},
	}
	angkaMap.Pairs["geserKiri"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Integer{Value: 0}
			}
			var val, shift int64
			if i, ok := args[0].(*Integer); ok {
				val = i.Value
			}
			if s, ok := args[1].(*Integer); ok {
				shift = s.Value
			}
			return &Integer{Value: val << shift}
		},
	}
	angkaMap.Pairs["geserKanan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Integer{Value: 0}
			}
			var val, shift int64
			if i, ok := args[0].(*Integer); ok {
				val = i.Value
			}
			if s, ok := args[1].(*Integer); ok {
				shift = s.Value
			}
			return &Integer{Value: val >> shift}
		},
	}
	angkaMap.Pairs["dan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Integer{Value: 0}
			}
			var a, b int64
			if i, ok := args[0].(*Integer); ok {
				a = i.Value
			}
			if i, ok := args[1].(*Integer); ok {
				b = i.Value
			}
			return &Integer{Value: a & b}
		},
	}
	angkaMap.Pairs["atau"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Integer{Value: 0}
			}
			var a, b int64
			if i, ok := args[0].(*Integer); ok {
				a = i.Value
			}
			if i, ok := args[1].(*Integer); ok {
				b = i.Value
			}
			return &Integer{Value: a | b}
		},
	}
	angkaMap.Pairs["xor"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return &Integer{Value: 0}
			}
			var a, b int64
			if i, ok := args[0].(*Integer); ok {
				a = i.Value
			}
			if i, ok := args[1].(*Integer); ok {
				b = i.Value
			}
			return &Integer{Value: a ^ b}
		},
	}
	angkaMap.Pairs["tidak"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 1 {
				return &Integer{Value: 0}
			}
			var a int64
			if i, ok := args[0].(*Integer); ok {
				a = i.Value
			}
			return &Integer{Value: ^a}
		},
	}
	env.Set("angka", angkaMap)

	// Modul strukturData (Tumpukan / Stack & Antrean / Queue)
	strukturDataMap := &Map{Pairs: make(map[string]Object)}
	strukturDataMap.Pairs["tumpukan"] = &Builtin{
		Fn: func(args ...Object) Object {
			var stackItems []Object
			var mu sync.Mutex
			sInst := &Map{Pairs: make(map[string]Object)}
			sInst.Pairs["dorong"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					mu.Lock()
					defer mu.Unlock()
					for _, item := range mArgs {
						stackItems = append(stackItems, item)
					}
					return &Integer{Value: int64(len(stackItems))}
				},
			}
			sInst.Pairs["tarik"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					mu.Lock()
					defer mu.Unlock()
					if len(stackItems) == 0 {
						return NIL
					}
					top := stackItems[len(stackItems)-1]
					stackItems = stackItems[:len(stackItems)-1]
					return top
				},
			}
			sInst.Pairs["intip"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					mu.Lock()
					defer mu.Unlock()
					if len(stackItems) == 0 {
						return NIL
					}
					return stackItems[len(stackItems)-1]
				},
			}
			sInst.Pairs["panjang"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					mu.Lock()
					defer mu.Unlock()
					return &Integer{Value: int64(len(stackItems))}
				},
			}
			sInst.Pairs["apakahKosong"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					mu.Lock()
					defer mu.Unlock()
					return &Boolean{Value: len(stackItems) == 0}
				},
			}
			sInst.Pairs["semua"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					mu.Lock()
					defer mu.Unlock()
					res := append([]Object{}, stackItems...)
					return &Array{Elements: res}
				},
			}
			return sInst
		},
	}
	strukturDataMap.Pairs["antrean"] = &Builtin{
		Fn: func(args ...Object) Object {
			var queueItems []Object
			var mu sync.Mutex
			qInst := &Map{Pairs: make(map[string]Object)}
			qInst.Pairs["masuk"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					mu.Lock()
					defer mu.Unlock()
					for _, item := range mArgs {
						queueItems = append(queueItems, item)
					}
					return &Integer{Value: int64(len(queueItems))}
				},
			}
			qInst.Pairs["keluar"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					mu.Lock()
					defer mu.Unlock()
					if len(queueItems) == 0 {
						return NIL
					}
					front := queueItems[0]
					queueItems = queueItems[1:]
					return front
				},
			}
			qInst.Pairs["depan"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					mu.Lock()
					defer mu.Unlock()
					if len(queueItems) == 0 {
						return NIL
					}
					return queueItems[0]
				},
			}
			qInst.Pairs["panjang"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					mu.Lock()
					defer mu.Unlock()
					return &Integer{Value: int64(len(queueItems))}
				},
			}
			qInst.Pairs["apakahKosong"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					mu.Lock()
					defer mu.Unlock()
					return &Boolean{Value: len(queueItems) == 0}
				},
			}
			qInst.Pairs["semua"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					mu.Lock()
					defer mu.Unlock()
					res := append([]Object{}, queueItems...)
					return &Array{Elements: res}
				},
			}
			return qInst
		},
	}
	env.Set("strukturData", strukturDataMap)

	// Modul soket (TCP Client & Server Sederhana)
	soketMap := &Map{Pairs: make(map[string]Object)}
	soketMap.Pairs["hubungkan"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			jaringanTipe := args[0].Inspect()
			alamat := args[1].Inspect()
			conn, err := net.DialTimeout(jaringanTipe, alamat, 10*time.Second)
			if err != nil {
				return NIL
			}
			cMap := &Map{Pairs: make(map[string]Object)}
			cMap.Pairs["tulis"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					if len(mArgs) < 1 {
						return &Integer{Value: 0}
					}
					n, err := conn.Write([]byte(mArgs[0].Inspect()))
					if err != nil {
						return &Integer{Value: 0}
					}
					return &Integer{Value: int64(n)}
				},
			}
			cMap.Pairs["baca"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					bufSize := 4096
					if len(mArgs) > 0 {
						if i, ok := mArgs[0].(*Integer); ok && i.Value > 0 {
							bufSize = int(i.Value)
						}
					}
					buf := make([]byte, bufSize)
					n, err := conn.Read(buf)
					if err != nil {
						return &String{Value: ""}
					}
					return &String{Value: string(buf[:n])}
				},
			}
			cMap.Pairs["tutup"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					_ = conn.Close()
					return TRUE
				},
			}
			return cMap
		},
	}
	soketMap.Pairs["dengar"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return NIL
			}
			jaringanTipe := args[0].Inspect()
			alamat := args[1].Inspect()
			ln, err := net.Listen(jaringanTipe, alamat)
			if err != nil {
				return NIL
			}
			sMap := &Map{Pairs: make(map[string]Object)}
			sMap.Pairs["terima"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					conn, err := ln.Accept()
					if err != nil {
						return NIL
					}
					cMap := &Map{Pairs: make(map[string]Object)}
					cMap.Pairs["tulis"] = &Builtin{
						Fn: func(subArgs ...Object) Object {
							if len(subArgs) < 1 {
								return &Integer{Value: 0}
							}
							n, err := conn.Write([]byte(subArgs[0].Inspect()))
							if err != nil {
								return &Integer{Value: 0}
							}
							return &Integer{Value: int64(n)}
						},
					}
					cMap.Pairs["baca"] = &Builtin{
						Fn: func(subArgs ...Object) Object {
							bufSize := 4096
							if len(subArgs) > 0 {
								if i, ok := subArgs[0].(*Integer); ok && i.Value > 0 {
									bufSize = int(i.Value)
								}
							}
							buf := make([]byte, bufSize)
							n, err := conn.Read(buf)
							if err != nil {
								return &String{Value: ""}
							}
							return &String{Value: string(buf[:n])}
						},
					}
					cMap.Pairs["tutup"] = &Builtin{
						Fn: func(subArgs ...Object) Object {
							_ = conn.Close()
							return TRUE
						},
					}
					return cMap
				},
			}
			sMap.Pairs["tutup"] = &Builtin{
				Fn: func(mArgs ...Object) Object {
					_ = ln.Close()
					return TRUE
				},
			}
			return sMap
		},
	}
	env.Set("soket", soketMap)

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
		// Jalankan fungsi inisiasi / init otomatis jika ada (filosofi func init() Go)
		if initFn, ok := env.Get("inisiasi"); ok {
			if fn, ok := initFn.(*Function); ok {
				res := applyFunction(fn, []Object{}, env)
				if p, ok := res.(*PanicObject); ok {
					fmt.Println("💥 Galat Panik pada inisiasi():", p.Value.Inspect())
					return p
				}
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

	case *ast.TypeInterfaceStatement:
		methodNames := []string{}
		for _, m := range n.Methods {
			methodNames = append(methodNames, m.Name)
		}
		intf := &InterfaceDef{Name: n.Name.Value, Methods: methodNames}
		env.Set(n.Name.Value, intf)
		return intf

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
				if _, hasField := si.Fields[dotExpr.Property.Value]; hasField {
					si.Fields[dotExpr.Property.Value] = val
					return val
				}
				// Cari apakah field berada pada embedded struct
				for _, fieldVal := range si.Fields {
					if embeddedSI, isEmbed := fieldVal.(*StructInstance); isEmbed {
						if _, hasEmbField := embeddedSI.Fields[dotExpr.Property.Value]; hasEmbField {
							embeddedSI.Fields[dotExpr.Property.Value] = val
							return val
						}
					}
				}
				si.Fields[dotExpr.Property.Value] = val
			} else if m, ok := left.(*Map); ok {
				m.Pairs[dotExpr.Property.Value] = val
			}
			return val
		}
		return NIL

	case *ast.ChannelSendStatement:
		chObj := Eval(n.Channel, env)
		val := Eval(n.Value, env)
		if ret, ok := val.(*ReturnValue); ok {
			val = ret.Value
		}
		if c, ok := chObj.(*ChannelObject); ok {
			c.Ch <- val
			return TRUE
		} else if m, ok := chObj.(*Map); ok {
			if kirimFn, ok := m.Pairs["kirim"].(*Builtin); ok {
				return kirimFn.Fn(val)
			}
		}
		return NIL

	case *ast.FunctionStatement:
		fnName := ""
		if n.Name != nil {
			fnName = n.Name.Value
		}
		fn := &Function{
			Name:         fnName,
			Token:        n.Token,
			Receiver:     n.Receiver,
			Parameters:   n.Parameters,
			ReturnParams: n.ReturnParams,
			IsVariadic:   n.IsVariadic,
			Body:         n.Body,
			Env:          env,
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
			Name:         "<anonim>",
			Token:        n.Token,
			Parameters:   n.Parameters,
			ReturnParams: n.ReturnParams,
			IsVariadic:   n.IsVariadic,
			Body:         n.Body,
			Env:          env,
		}

	case *ast.BreakStatement:
		return &Break{Label: n.Label}

	case *ast.ContinueStatement:
		return &Continue{Label: n.Label}

	case *ast.FallthroughStatement:
		return &Fallthrough{}

	case *ast.TryCatchStatement:
		// Simpan panic sebelumnya jika ada
		prevPanic := GetGlobalPanic()
		SetGlobalPanic(nil)

		// Evaluasi blok coba
		tryEnv := NewEnclosedEnvironment(env)
		res := Eval(n.TryBlock, tryEnv)

		var caughtPanic Object = nil
		if pObj, isPanic := res.(*PanicObject); isPanic {
			caughtPanic = pObj.Value
			SetGlobalPanic(nil)
		} else if activeP := GetGlobalPanic(); activeP != nil {
			caughtPanic = activeP
			SetGlobalPanic(nil)
		}

		var catchResult Object = nil
		if caughtPanic != nil && n.CatchBlock != nil {
			catchEnv := NewEnclosedEnvironment(env)
			if n.CatchIdent != nil {
				catchEnv.Set(n.CatchIdent.Value, caughtPanic)
			}
			catchResult = Eval(n.CatchBlock, catchEnv)
		}

		// Jalankan blok akhirnya jika ada
		if n.FinallyBlock != nil {
			finEnv := NewEnclosedEnvironment(env)
			finRes := Eval(n.FinallyBlock, finEnv)
			if finRes != nil {
				if finRes.Type() == RETURN_VALUE_OBJ || finRes.Type() == BREAK_OBJ || finRes.Type() == CONTINUE_OBJ || finRes.Type() == PANIC_OBJ {
					return finRes
				}
			}
		}

		if catchResult != nil {
			if catchResult.Type() == RETURN_VALUE_OBJ || catchResult.Type() == BREAK_OBJ || catchResult.Type() == CONTINUE_OBJ || catchResult.Type() == PANIC_OBJ {
				return catchResult
			}
			return catchResult
		}

		if caughtPanic != nil && n.CatchBlock == nil {
			// Jika tidak ada catch handler, lempar kembali
			SetGlobalPanic(caughtPanic)
			return &PanicObject{Value: caughtPanic}
		}

		// Pulihkan status panic sebelumnya jika tidak ada panic baru
		if prevPanic != nil && caughtPanic == nil {
			SetGlobalPanic(prevPanic)
		}

		if res != nil && (res.Type() == RETURN_VALUE_OBJ || res.Type() == BREAK_OBJ || res.Type() == CONTINUE_OBJ) {
			return res
		}
		return NIL

	case *ast.BlockStatement:
		var result Object = NIL
		for _, stmt := range n.Statements {
			result = Eval(stmt, env)
			if result != nil {
				rt := result.Type()
				if rt == RETURN_VALUE_OBJ || rt == BREAK_OBJ || rt == CONTINUE_OBJ || rt == FALLTHROUGH_OBJ || rt == PANIC_OBJ {
					return result
				}
			}
		}
		return result

	case *ast.LabeledStatement:
		// Jika statement adalah perulangan, jalankan dengan penanganan continue bertarget label ini
		if forStmt, ok := n.Statement.(*ast.ForStatement); ok {
			for {
				res := Eval(forStmt, env)
				if res != nil {
					if b, isBreak := res.(*Break); isBreak {
						if b.Label == n.Label {
							return NIL
						}
						return b // bubble up ke outer label
					}
					if c, isCont := res.(*Continue); isCont {
						if c.Label == n.Label {
							// Continue pada loop terluar berlabel ini
							continue
						}
						return c // bubble up ke outer label
					}
					return res
				}
				return NIL
			}
		}

		res := Eval(n.Statement, env)
		if res != nil {
			if b, ok := res.(*Break); ok && b.Label == n.Label {
				return NIL
			}
			if c, ok := res.(*Continue); ok && c.Label == n.Label {
				return NIL
			}
		}
		return res

	case *ast.IfStatement:
		ifEnv := env
		if n.Init != nil {
			ifEnv = NewEnclosedEnvironment(env)
			Eval(n.Init, ifEnv)
		}
		cond := Eval(n.Condition, ifEnv)
		if isTruthy(cond) {
			return Eval(n.Consequence, ifEnv)
		} else if n.Alternative != nil {
			return Eval(n.Alternative, ifEnv)
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
							if b, ok := res.(*Break); ok && b.Label != "" {
								return res
							}
							break
						}
						if res.Type() == CONTINUE_OBJ {
							if c, ok := res.(*Continue); ok && c.Label != "" {
								return res
							}
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
							if b, ok := res.(*Break); ok && b.Label != "" {
								return res
							}
							break
						}
						if res.Type() == CONTINUE_OBJ {
							if c, ok := res.(*Continue); ok && c.Label != "" {
								return res
							}
							continue
						}
					}
				}
				return NIL
			} else if str, ok := iterObj.(*String); ok {
				// Iterasi rentang string: indeks integer & rune karakter Unicode
				runes := []rune(str.Value)
				byteOffset := int64(0)
				for i, r := range runes {
					if n.Key != nil {
						loopEnv.Set(n.Key.Value, &Integer{Value: byteOffset})
					}
					if n.Value != nil {
						loopEnv.Set(n.Value.Value, &Integer{Value: int64(r)})
					}
					res := Eval(n.Body, loopEnv)
					if res != nil {
						if res.Type() == RETURN_VALUE_OBJ {
							return res
						}
						if res.Type() == BREAK_OBJ {
							if b, ok := res.(*Break); ok && b.Label != "" {
								return res
							}
							break
						}
						if res.Type() == CONTINUE_OBJ {
							if c, ok := res.(*Continue); ok && c.Label != "" {
								return res
							}
							byteOffset += int64(len(string(r)))
							continue
						}
					}
					byteOffset += int64(len(string(r)))
					_ = i
				}
				return NIL
			} else if chObj, ok := iterObj.(*ChannelObject); ok {
				// Iterasi drain kanal: untuk val dalam ch { ... } (ala for val := range ch di Go)
				for val := range chObj.Ch {
					if n.Key != nil && n.Value != nil {
						loopEnv.Set(n.Key.Value, val)
					} else if n.Key != nil {
						loopEnv.Set(n.Key.Value, val)
					} else if n.Value != nil {
						loopEnv.Set(n.Value.Value, val)
					}
					res := Eval(n.Body, loopEnv)
					if res != nil {
						if res.Type() == RETURN_VALUE_OBJ {
							return res
						}
						if res.Type() == BREAK_OBJ {
							if b, ok := res.(*Break); ok && b.Label != "" {
								return res
							}
							break
						}
						if res.Type() == CONTINUE_OBJ {
							if c, ok := res.(*Continue); ok && c.Label != "" {
								return res
							}
							continue
						}
					}
				}
				return NIL
			} else if m, ok := iterObj.(*Map); ok {
				if raw, okRaw := m.Pairs["_rawChan"].(*ChannelObject); okRaw {
					for val := range raw.Ch {
						if n.Key != nil && n.Value != nil {
							loopEnv.Set(n.Key.Value, val)
						} else if n.Key != nil {
							loopEnv.Set(n.Key.Value, val)
						} else if n.Value != nil {
							loopEnv.Set(n.Value.Value, val)
						}
						res := Eval(n.Body, loopEnv)
						if res != nil {
							if res.Type() == RETURN_VALUE_OBJ {
								return res
							}
							if res.Type() == BREAK_OBJ {
								if b, ok := res.(*Break); ok && b.Label != "" {
									return res
								}
								break
							}
							if res.Type() == CONTINUE_OBJ {
								if c, ok := res.(*Continue); ok && c.Label != "" {
									return res
								}
								continue
							}
						}
					}
					return NIL
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
					if b, ok := result.(*Break); ok && b.Label != "" {
						return result
					}
					break
				}
				if result.Type() == CONTINUE_OBJ {
					if c, ok := result.(*Continue); ok && c.Label != "" {
						return result
					}
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
		// Cek apakah ini native channel select: pilih { kasus msg := <-ch: ... kasus ch <- val: ... }
		isChannelSelect := false
		if n.Value == nil && n.Init == nil {
			for _, c := range n.Cases {
				if c.Send != nil || c.Assign != nil {
					isChannelSelect = true
					break
				}
				if c.Value != nil {
					if pref, isPref := c.Value.(*ast.PrefixExpression); isPref && pref.Operator == "<-" {
						isChannelSelect = true
						break
					}
				}
			}
		}

		if isChannelSelect {
			// Evaluasi native channel select
			// 1. Jika ada default block, lakukan non-blocking check
			if n.Default != nil {
				for _, c := range n.Cases {
					if c.Assign != nil {
						// kasus msg := <-ch:
						if pref, isPref := c.Assign.Value.(*ast.PrefixExpression); isPref && pref.Operator == "<-" {
							chObj := Eval(pref.Right, env)
							val, ok := tryReceiveChannel(chObj)
							if ok {
								caseEnv := NewEnclosedEnvironment(env)
								if targetIdent, isId := c.Assign.Target.(*ast.Identifier); isId {
									caseEnv.Set(targetIdent.Value, val)
								}
								return Eval(c.Body, caseEnv)
							}
						}
					} else if c.Send != nil {
						// kasus ch <- val:
						chObj := Eval(c.Send.Channel, env)
						valObj := Eval(c.Send.Value, env)
						if trySendChannel(chObj, valObj) {
							return Eval(c.Body, env)
						}
					} else if c.Value != nil {
						if pref, isPref := c.Value.(*ast.PrefixExpression); isPref && pref.Operator == "<-" {
							chObj := Eval(pref.Right, env)
							_, ok := tryReceiveChannel(chObj)
							if ok {
								return Eval(c.Body, env)
							}
						}
					}
				}
				// Tidak ada kanal yang siap, jalankan default
				return Eval(n.Default, env)
			}

			// Blocking multiplexing polling
			for {
				for _, c := range n.Cases {
					if c.Assign != nil {
						if pref, isPref := c.Assign.Value.(*ast.PrefixExpression); isPref && pref.Operator == "<-" {
							chObj := Eval(pref.Right, env)
							val, ok := tryReceiveChannel(chObj)
							if ok {
								caseEnv := NewEnclosedEnvironment(env)
								if targetIdent, isId := c.Assign.Target.(*ast.Identifier); isId {
									caseEnv.Set(targetIdent.Value, val)
								}
								return Eval(c.Body, caseEnv)
							}
						}
					} else if c.Send != nil {
						chObj := Eval(c.Send.Channel, env)
						valObj := Eval(c.Send.Value, env)
						if trySendChannel(chObj, valObj) {
							return Eval(c.Body, env)
						}
					} else if c.Value != nil {
						if pref, isPref := c.Value.(*ast.PrefixExpression); isPref && pref.Operator == "<-" {
							chObj := Eval(pref.Right, env)
							_, ok := tryReceiveChannel(chObj)
							if ok {
								return Eval(c.Body, env)
							}
						}
					}
				}
				time.Sleep(2 * time.Millisecond)
			}
		}

		switchEnv := env
		var rawSubject Object = nil
		isTypeSwitch := false

		// Cek apakah ada statement assignment pembuat variabel, misal: pilih v := x.(tipe) { ... }
		if n.Init != nil {
			switchEnv = NewEnclosedEnvironment(env)
			if assign, ok := n.Init.(*ast.AssignStatement); ok {
				if typeAssert, isTA := assign.Value.(*ast.TypeAssertionExpression); isTA && typeAssert.IsType {
					isTypeSwitch = true
					rawSubject = Eval(typeAssert.Left, env)
					if ident, isIdent := assign.Target.(*ast.Identifier); isIdent {
						switchEnv.Set(ident.Value, rawSubject)
					}
				} else {
					Eval(n.Init, switchEnv)
				}
			} else {
				Eval(n.Init, switchEnv)
			}
		}

		var targetVal Object = TRUE
		if n.Value != nil {
			if typeAssert, isTA := n.Value.(*ast.TypeAssertionExpression); isTA && typeAssert.IsType {
				isTypeSwitch = true
				rawSubject = Eval(typeAssert.Left, env)
			} else if !isTypeSwitch {
				targetVal = Eval(n.Value, switchEnv)
			}
		}

		fallthroughNext := false
		for idx, c := range n.Cases {
			matched := fallthroughNext
			if !matched {
				if isTypeSwitch {
					subjectType := strings.ToLower(getObjectTypeString(rawSubject))
					for _, vExpr := range c.Values {
						typeStr := ""
						if ident, ok := vExpr.(*ast.Identifier); ok {
							typeStr = strings.ToLower(ident.Value)
						} else {
							typeStr = strings.ToLower(Eval(vExpr, switchEnv).Inspect())
						}

						if typeStr == subjectType ||
							(subjectType == "angka" && (typeStr == "int" || typeStr == "angka")) ||
							(subjectType == "teks" && (typeStr == "string" || typeStr == "teks")) ||
							(subjectType == "desimal" && (typeStr == "float" || typeStr == "desimal")) ||
							(subjectType == "boolean" && (typeStr == "bool" || typeStr == "boolean")) {
							matched = true
							break
						}
						// Cek apakah typeStr adalah nama interface yang diimplementasikan oleh rawSubject
						if origIdent, isID := vExpr.(*ast.Identifier); isID {
							if typeObj, hasType := env.Get(origIdent.Value); hasType {
								if intfDef, isIntf := typeObj.(*InterfaceDef); isIntf {
									if intfDef.ImplementedBy(rawSubject) {
										matched = true
										break
									}
								}
							}
						}
					}
				} else {
					if len(c.Values) > 0 {
						for _, vExpr := range c.Values {
							val := Eval(vExpr, switchEnv)
							if objectsEqual(targetVal, val) {
								matched = true
								break
							}
						}
					} else if c.Value != nil {
						val := Eval(c.Value, switchEnv)
						if objectsEqual(targetVal, val) {
							matched = true
						}
					}
				}
			}

			if matched {
				res := Eval(c.Body, switchEnv)
				if res != nil && res.Type() == FALLTHROUGH_OBJ {
					fallthroughNext = true
					continue
				}
				return res
			}
			fallthroughNext = false
			_ = idx
		}
		if n.Default != nil {
			return Eval(n.Default, switchEnv)
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
		if str, ok := left.(*String); ok {
			// Pengindeksan string str[i] mengembalikan byte mentah (uint8/int64) ala Go
			if i, ok := idx.(*Integer); ok {
				if i.Value < 0 || i.Value >= int64(len(str.Value)) {
					panicMsg := fmt.Sprintf("indeks teks [%d] di luar batas panjang %d pada baris %d:%d", i.Value, len(str.Value), n.Token.Line, n.Token.Column)
					SetGlobalPanic(&String{Value: panicMsg})
					return &PanicObject{
						Value: &String{Value: panicMsg},
						StackTrace: []CallFrame{{
							Function: "<indeks>",
							Line:     n.Token.Line,
							Column:   n.Token.Column,
						}},
					}
				}
				return &Integer{Value: int64(str.Value[i.Value])}
			}
		} else if arr, ok := left.(*Array); ok {
			if i, ok := idx.(*Integer); ok {
				if i.Value < 0 || i.Value >= int64(len(arr.Elements)) {
					panicMsg := fmt.Sprintf("indeks larik [%d] di luar batas panjang %d pada baris %d:%d", i.Value, len(arr.Elements), n.Token.Line, n.Token.Column)
					SetGlobalPanic(&String{Value: panicMsg})
					return &PanicObject{
						Value: &String{Value: panicMsg},
						StackTrace: []CallFrame{{
							Function: "<indeks>",
							Line:     n.Token.Line,
							Column:   n.Token.Column,
						}},
					}
				}
				return arr.Elements[i.Value]
			}
		} else if m, ok := left.(*Map); ok {
			if val, ok := m.Pairs[idx.Inspect()]; ok {
				return val
			}
		}
		return NIL

	case *ast.TypeAssertionExpression:
		val := Eval(n.Left, env)
		if n.IsType {
			// Type switch helper x.(tipe) / x.(type) -> kembalikan representasi tipe
			return &String{Value: getObjectTypeString(val)}
		}

		// x.(Tipe) Type assertion:
		targetType := strings.ToLower(n.Type)
		actualType := getObjectTypeString(val)
		if strings.EqualFold(targetType, actualType) || (actualType == "angka" && targetType == "int") || (actualType == "teks" && targetType == "string") {
			return val
		}
		// Cek apakah targetType adalah definisi Antarmuka (Interface)
		if typeObj, hasType := env.Get(n.Type); hasType {
			if intfDef, isIntf := typeObj.(*InterfaceDef); isIntf {
				if intfDef.ImplementedBy(val) {
					return val
				}
				return NIL
			}
		}
		// Jika tipe struct custom:
		if si, ok := val.(*StructInstance); ok && si.Def != nil {
			if strings.EqualFold(si.Def.Name, n.Type) {
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

			// Cek Promoted Field / Method dari Embedded Structs (Komposisi Struct ala Go)
			for _, fieldVal := range si.Fields {
				if embeddedSI, isEmbed := fieldVal.(*StructInstance); isEmbed {
					if pVal, hasP := embeddedSI.Fields[n.Property.Value]; hasP {
						return pVal
					}
					if embeddedSI.Def != nil && embeddedSI.Def.Methods != nil {
						if method, hasMethod := embeddedSI.Def.Methods[n.Property.Value]; hasMethod {
							boundFn := &Function{
								Receiver:   method.Receiver,
								Parameters: method.Parameters,
								IsVariadic: method.IsVariadic,
								Body:       method.Body,
								Env:        NewEnclosedEnvironment(method.Env),
							}
							if method.Receiver != nil {
								boundFn.Env.Set(method.Receiver.Name, embeddedSI)
							}
							return boundFn
						}
					}
				}
			}
		}
		return NIL

	case *ast.PrefixExpression:
		right := Eval(n.Right, env)
		if n.Operator == "<-" {
			// Operasi terima kanal: <-ch
			if chObj, ok := right.(*ChannelObject); ok {
				val, okChan := <-chObj.Ch
				if !okChan {
					return &ReturnValue{Value: NIL, Values: []Object{NIL, FALSE}}
				}
				if val == nil {
					val = NIL
				}
				return &ReturnValue{Value: val, Values: []Object{val, TRUE}}
			} else if m, ok := right.(*Map); ok {
				if terimaFn, ok := m.Pairs["terima"].(*Builtin); ok {
					return terimaFn.Fn()
				}
			}
			return NIL
		} else if n.Operator == "!" {
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
		if n.Operator == "/" || n.Operator == "%" {
			if rInt, ok := right.(*Integer); ok && rInt.Value == 0 {
				panicMsg := fmt.Sprintf("pembagian atau modulus dengan angka nol pada baris %d:%d", n.Token.Line, n.Token.Column)
				SetGlobalPanic(&String{Value: panicMsg})
				return &PanicObject{
					Value: &String{Value: panicMsg},
					StackTrace: []CallFrame{{
						Function: "<operasi>",
						Line:     n.Token.Line,
						Column:   n.Token.Column,
					}},
				}
			} else if rFloat, ok := right.(*Float); ok && rFloat.Value == 0 {
				panicMsg := fmt.Sprintf("pembagian dengan angka nol pada baris %d:%d", n.Token.Line, n.Token.Column)
				SetGlobalPanic(&String{Value: panicMsg})
				return &PanicObject{
					Value: &String{Value: panicMsg},
					StackTrace: []CallFrame{{
						Function: "<operasi>",
						Line:     n.Token.Line,
						Column:   n.Token.Column,
					}},
				}
			}
		}
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

func tryReceiveChannel(chObj Object) (Object, bool) {
	if c, ok := chObj.(*ChannelObject); ok {
		select {
		case v, okChan := <-c.Ch:
			if !okChan {
				return NIL, false
			}
			if v == nil {
				v = NIL
			}
			return v, true
		default:
			return NIL, false
		}
	} else if m, ok := chObj.(*Map); ok {
		if raw, okRaw := m.Pairs["_rawChan"].(*ChannelObject); okRaw {
			select {
			case v, okChan := <-raw.Ch:
				if !okChan {
					return NIL, false
				}
				if v == nil {
					v = NIL
				}
				return v, true
			default:
				return NIL, false
			}
		}
		if cobaTerimaFn, ok := m.Pairs["cobaTerima"].(*Builtin); ok {
			res := cobaTerimaFn.Fn()
			if ret, ok := res.(*ReturnValue); ok && len(ret.Values) > 1 {
				if ret.Values[1] == TRUE {
					return ret.Values[0], true
				}
			}
		}
	}
	return NIL, false
}

func trySendChannel(chObj Object, val Object) bool {
	if c, ok := chObj.(*ChannelObject); ok {
		select {
		case c.Ch <- val:
			return true
		default:
			return false
		}
	} else if m, ok := chObj.(*Map); ok {
		if kirimFn, ok := m.Pairs["kirim"].(*Builtin); ok {
			kirimFn.Fn(val)
			return true
		}
	}
	return false
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

		// Inisialisasi variabel return bernama (Named Return Values ala Go) dengan NIL / Zero value
		if len(f.ReturnParams) > 0 {
			for _, rp := range f.ReturnParams {
				if _, exists := fnEnv.Get(rp.Value); !exists {
					fnEnv.Set(rp.Value, NIL)
				}
			}
		}

		res := Eval(f.Body, fnEnv)

		if pObj, isPanic := res.(*PanicObject); isPanic {
			pObj.StackTrace = append(pObj.StackTrace, CallFrame{
				Function: f.Name,
				Line:     f.Token.Line,
				Column:   f.Token.Column,
			})
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
			if pObj, isPanic := res.(*PanicObject); isPanic {
				return pObj
			}
			return &PanicObject{
				Value: activePanic,
				StackTrace: []CallFrame{{
					Function: f.Name,
					Line:     f.Token.Line,
					Column:   f.Token.Column,
				}},
			}
		}

		if ret, ok := res.(*ReturnValue); ok {
			// Naked return: jika kembalikan dipanggil tanpa nilai dan fungsi memiliki named return parameters
			if len(ret.Values) == 0 || (len(ret.Values) == 1 && ret.Values[0] == NIL && len(f.ReturnParams) > 0) {
				namedVals := []Object{}
				for _, rp := range f.ReturnParams {
					if v, ok := fnEnv.Get(rp.Value); ok {
						namedVals = append(namedVals, v)
					} else {
						namedVals = append(namedVals, NIL)
					}
				}
				if len(namedVals) > 1 {
					return &ReturnValue{Value: namedVals[0], Values: namedVals}
				} else if len(namedVals) == 1 {
					return namedVals[0]
				}
			}
			if len(ret.Values) > 1 {
				return ret
			}
			return ret.Value
		}

		// Jika fungsi selesai tanpa eksplisit kembalikan, tapi ada ReturnParams
		if len(f.ReturnParams) > 0 {
			namedVals := []Object{}
			for _, rp := range f.ReturnParams {
				if v, ok := fnEnv.Get(rp.Value); ok {
					namedVals = append(namedVals, v)
				} else {
					namedVals = append(namedVals, NIL)
				}
			}
			if len(namedVals) > 1 {
				return &ReturnValue{Value: namedVals[0], Values: namedVals}
			} else if len(namedVals) == 1 {
				return namedVals[0]
			}
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
	// Operasi aritmatika & perbandingan jika melibatkan Float atau Integer
	if (left.Type() == FLOAT_OBJ || left.Type() == INTEGER_OBJ) && (right.Type() == FLOAT_OBJ || right.Type() == INTEGER_OBJ) {
		if left.Type() == FLOAT_OBJ || right.Type() == FLOAT_OBJ {
			var lVal, rVal float64
			if left.Type() == FLOAT_OBJ {
				lVal = left.(*Float).Value
			} else {
				lVal = float64(left.(*Integer).Value)
			}
			if right.Type() == FLOAT_OBJ {
				rVal = right.(*Float).Value
			} else {
				rVal = float64(right.(*Integer).Value)
			}

			switch op {
			case "+":
				return &Float{Value: lVal + rVal}
			case "-":
				return &Float{Value: lVal - rVal}
			case "*":
				return &Float{Value: lVal * rVal}
			case "/":
				if rVal == 0 {
					return &Float{Value: 0}
				}
				return &Float{Value: lVal / rVal}
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
	}

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

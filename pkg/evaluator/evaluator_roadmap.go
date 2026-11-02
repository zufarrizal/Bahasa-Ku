package evaluator

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Sandbox mode & VFS storage
var (
	globalStrictSandbox   bool
	globalStrictSandboxMu sync.RWMutex
	globalVirtualFS       = make(map[string][]byte)
	globalVirtualFSMu     sync.RWMutex
	jwtRevocationList     = make(map[string]int64)
	jwtRevocationMu       sync.RWMutex
	desktopStateMap       = make(map[string]Object)
	desktopStateListeners []func(key string, val Object)
	desktopStateMu        sync.RWMutex
	asyncQueueHandlers    = make(map[string]func(payload Object))
	asyncQueueMu          sync.RWMutex
)

// SetStrictSandboxMode mengaktifkan atau menonaktifkan mode pembatasan ketat
func SetStrictSandboxMode(active bool) {
	globalStrictSandboxMu.Lock()
	defer globalStrictSandboxMu.Unlock()
	globalStrictSandbox = active
}

// IsStrictSandboxMode mengecek apakah sandbox aktif
func IsStrictSandboxMode() bool {
	globalStrictSandboxMu.RLock()
	defer globalStrictSandboxMu.RUnlock()
	return globalStrictSandbox
}

// RegisterVirtualFile mendaftarkan berkas virtual memori
func RegisterVirtualFile(path string, content []byte) {
	globalVirtualFSMu.Lock()
	defer globalVirtualFSMu.Unlock()
	clean := filepath.ToSlash(filepath.Clean(path))
	clean = strings.TrimPrefix(clean, "/")
	globalVirtualFS[clean] = content
}

// GetVirtualFile membaca berkas virtual memori
func GetVirtualFile(path string) ([]byte, bool) {
	globalVirtualFSMu.RLock()
	defer globalVirtualFSMu.RUnlock()
	clean := filepath.ToSlash(filepath.Clean(path))
	clean = strings.TrimPrefix(clean, "/")
	val, ok := globalVirtualFS[clean]
	return val, ok
}

func extendGlobalEnvironment(env *Environment) {
	// 0. Modul jaringan: jaringan.buatKlien
	if jObj, ok := env.Get("jaringan"); ok {
		if jMap, isMap := jObj.(*Map); isMap {
			jMap.Pairs["buatKlien"] = &Builtin{
				Fn: func(args ...Object) Object {
					timeout := 15 * time.Second
					followRedirect := true
					if len(args) > 0 {
						if cfgMap, ok := args[0].(*Map); ok {
							if toObj, exists := cfgMap.Pairs["batasWaktu"]; exists {
								if d, err := time.ParseDuration(toObj.Inspect()); err == nil {
									timeout = d
								}
							}
							if fObj, exists := cfgMap.Pairs["ikutiPengalihan"]; exists {
								followRedirect = isTruthy(fObj)
							}
						}
					}

					httpClient := &http.Client{
						Timeout: timeout,
					}
					if !followRedirect {
						httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
							return http.ErrUseLastResponse
						}
					}

					klienObj := &Map{Pairs: make(map[string]Object)}
					klienObj.Pairs["dapatkan"] = &Builtin{
						Fn: func(kArgs ...Object) Object {
							if len(kArgs) < 1 {
								return NIL
							}
							urlStr := kArgs[0].Inspect()
							resp, err := httpClient.Get(urlStr)
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
							if resp.StatusCode >= 200 && resp.StatusCode < 300 {
								resMap.Pairs["sukses"] = TRUE
							} else {
								resMap.Pairs["sukses"] = FALSE
							}
							resMap.Pairs["status"] = &Integer{Value: int64(resp.StatusCode)}
							resMap.Pairs["badan"] = &String{Value: string(bodyBytes)}
							return resMap
						},
					}
					return klienObj
				},
			}

			// jaringan.buatRPCServer(alamat) & jaringan.buatRPCKlien(alamat)
			jMap.Pairs["buatRPCServer"] = &Builtin{
				Fn: func(args ...Object) Object {
					addr := "127.0.0.1:9090"
					if len(args) > 0 && args[0].Type() == STRING_OBJ {
						addr = args[0].Inspect()
					}

					services := make(map[string]map[string]Object)
					var sMu sync.RWMutex

					serverObj := &Map{Pairs: make(map[string]Object)}
					serverObj.Pairs["daftarkan"] = &Builtin{
						Fn: func(dArgs ...Object) Object {
							if len(dArgs) < 2 {
								return FALSE
							}
							svcName := dArgs[0].Inspect()
							fnMap, ok := dArgs[1].(*Map)
							if !ok {
								return FALSE
							}
							sMu.Lock()
							if services[svcName] == nil {
								services[svcName] = make(map[string]Object)
							}
							for mName, mFn := range fnMap.Pairs {
								services[svcName][mName] = mFn
							}
							sMu.Unlock()
							return TRUE
						},
					}

					serverObj.Pairs["dengarkan"] = &Builtin{
						Fn: func(lArgs ...Object) Object {
							ln, err := net.Listen("tcp", addr)
							if err != nil {
								return FALSE
							}
							go func() {
								defer ln.Close()
								for {
									conn, err := ln.Accept()
									if err != nil {
										return
									}
									go func(c net.Conn) {
										defer c.Close()
										reader := bufio.NewReader(c)
										for {
											reqLine, err := reader.ReadBytes('\n')
											if err != nil {
												return
											}
											var rpcReq struct {
												Layanan string        `json:"layanan"`
												Metode  string        `json:"metode"`
												Argumen []interface{} `json:"argumen"`
											}
											if err := json.Unmarshal(reqLine, &rpcReq); err != nil {
												continue
											}

											sMu.RLock()
											svc := services[rpcReq.Layanan]
											var targetFn Object
											if svc != nil {
												targetFn = svc[rpcReq.Metode]
											}
											sMu.RUnlock()

											var rpcResp struct {
												Sukses bool        `json:"sukses"`
												Hasil  interface{} `json:"hasil"`
												Galat  string      `json:"galat,omitempty"`
											}

											if targetFn == nil {
												rpcResp.Sukses = false
												rpcResp.Galat = fmt.Sprintf("Metode '%s.%s' tidak ditemukan", rpcReq.Layanan, rpcReq.Metode)
											} else {
												var fnArgs []Object
												for _, rawArg := range rpcReq.Argumen {
													fnArgs = append(fnArgs, convertGoToKuObject(rawArg))
												}
												resObj := applyFunction(targetFn, fnArgs, env)
												rpcResp.Sukses = true
												rpcResp.Hasil = convertKuToGoObject(resObj)
											}

											respBytes, _ := json.Marshal(rpcResp)
											respBytes = append(respBytes, '\n')
											_, _ = c.Write(respBytes)
										}
									}(conn)
								}
							}()
							return TRUE
						},
					}

					return serverObj
				},
			}

			jMap.Pairs["buatRPCKlien"] = &Builtin{
				Fn: func(args ...Object) Object {
					addr := "127.0.0.1:9090"
					if len(args) > 0 && args[0].Type() == STRING_OBJ {
						addr = args[0].Inspect()
					}

					klienObj := &Map{Pairs: make(map[string]Object)}
					klienObj.Pairs["panggil"] = &Builtin{
						Fn: func(pArgs ...Object) Object {
							if len(pArgs) < 1 {
								return NIL
							}
							callTarget := pArgs[0].Inspect()
							parts := strings.Split(callTarget, ".")
							if len(parts) != 2 {
								return NIL
							}
							svcName := parts[0]
							methodName := parts[1]

							var rawArgs []interface{}
							for _, a := range pArgs[1:] {
								rawArgs = append(rawArgs, convertKuToGoObject(a))
							}

							conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
							if err != nil {
								return NIL
							}
							defer conn.Close()

							reqPayload := map[string]interface{}{
								"layanan": svcName,
								"metode":  methodName,
								"argumen": rawArgs,
							}
							b, _ := json.Marshal(reqPayload)
							b = append(b, '\n')
							if _, err := conn.Write(b); err != nil {
								return NIL
							}

							reader := bufio.NewReader(conn)
							respLine, err := reader.ReadBytes('\n')
							if err != nil {
								return NIL
							}

							var rpcResp struct {
								Sukses bool        `json:"sukses"`
								Hasil  interface{} `json:"hasil"`
								Galat  string      `json:"galat,omitempty"`
							}
							if err := json.Unmarshal(respLine, &rpcResp); err != nil || !rpcResp.Sukses {
								return NIL
							}

							return convertGoToKuObject(rpcResp.Hasil)
						},
					}

					return klienObj
				},
			}
		}
	}

	// 1. Modul kripto: AES, Argon2id, JWT Revocation, Webhook HMAC
	if kObj, ok := env.Get("kripto"); ok {
		if kMap, isMap := kObj.(*Map); isMap {
			// AES-256-GCM enkripsi
			kMap.Pairs["enkripsiAES"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 2 {
						return NIL
					}
					plaintext := []byte(args[0].Inspect())
					keyStr := args[1].Inspect()
					keyHash := sha256.Sum256([]byte(keyStr))
					block, err := aes.NewCipher(keyHash[:])
					if err != nil {
						return NIL
					}
					gcm, err := cipher.NewGCM(block)
					if err != nil {
						return NIL
					}
					nonce := make([]byte, gcm.NonceSize())
					if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
						return NIL
					}
					sealed := gcm.Seal(nonce, nonce, plaintext, nil)
					return &String{Value: base64.StdEncoding.EncodeToString(sealed)}
				},
			}

			// AES-256-GCM dekripsi
			kMap.Pairs["dekripsiAES"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 2 {
						return NIL
					}
					cipherB64 := args[0].Inspect()
					keyStr := args[1].Inspect()
					data, err := base64.StdEncoding.DecodeString(cipherB64)
					if err != nil {
						return NIL
					}
					keyHash := sha256.Sum256([]byte(keyStr))
					block, err := aes.NewCipher(keyHash[:])
					if err != nil {
						return NIL
					}
					gcm, err := cipher.NewGCM(block)
					if err != nil {
						return NIL
					}
					if len(data) < gcm.NonceSize() {
						return NIL
					}
					nonce := data[:gcm.NonceSize()]
					ciphertext := data[gcm.NonceSize():]
					plain, err := gcm.Open(nil, nonce, ciphertext, nil)
					if err != nil {
						return NIL
					}
					return &String{Value: string(plain)}
				},
			}

			// Argon2id / High-security password hashing
			kMap.Pairs["hashArgon2"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 1 {
						return &String{Value: ""}
					}
					sandi := []byte(args[0].Inspect())
					salt := make([]byte, 16)
					_, _ = io.ReadFull(rand.Reader, salt)
					iter := 64000
					prf := hmac.New(sha512.New, sandi)
					prf.Write(salt)
					h := prf.Sum(nil)
					for i := 1; i < iter; i++ {
						prf.Reset()
						prf.Write(h)
						h = prf.Sum(nil)
					}
					encoded := fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=4$%s$%s",
						base64.RawStdEncoding.EncodeToString(salt),
						base64.RawStdEncoding.EncodeToString(h))
					return &String{Value: encoded}
				},
			}

			kMap.Pairs["cocokkanArgon2"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 2 {
						return FALSE
					}
					sandi := []byte(args[0].Inspect())
					encoded := args[1].Inspect()
					parts := strings.Split(encoded, "$")
					if len(parts) < 6 || parts[1] != "argon2id" {
						return FALSE
					}
					salt, err := base64.RawStdEncoding.DecodeString(parts[4])
					if err != nil {
						return FALSE
					}
					expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
					if err != nil {
						return FALSE
					}

					iter := 64000
					prf := hmac.New(sha512.New, sandi)
					prf.Write(salt)
					h := prf.Sum(nil)
					for i := 1; i < iter; i++ {
						prf.Reset()
						prf.Write(h)
						h = prf.Sum(nil)
					}

					if subtle.ConstantTimeCompare(h, expectedHash) == 1 {
						return TRUE
					}
					return FALSE
				},
			}

			// JWT Revocation
			kMap.Pairs["cabutJWT"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 1 {
						return FALSE
					}
					tokenStr := args[0].Inspect()
					jwtRevocationMu.Lock()
					jwtRevocationList[tokenStr] = time.Now().Unix()
					jwtRevocationMu.Unlock()
					return TRUE
				},
			}

			kMap.Pairs["apakahJWTDicabut"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 1 {
						return FALSE
					}
					tokenStr := args[0].Inspect()
					jwtRevocationMu.RLock()
					_, revoked := jwtRevocationList[tokenStr]
					jwtRevocationMu.RUnlock()
					if revoked {
						return TRUE
					}
					return FALSE
				},
			}

			// Modifikasi verifikasiJWT bawaan untuk memvalidasi daftar pencabutan
			if origVerif, exists := kMap.Pairs["verifikasiJWT"].(*Builtin); exists {
				oldFn := origVerif.Fn
				kMap.Pairs["verifikasiJWT"] = &Builtin{
					Fn: func(args ...Object) Object {
						if len(args) >= 1 {
							tokenStr := args[0].Inspect()
							jwtRevocationMu.RLock()
							_, revoked := jwtRevocationList[tokenStr]
							jwtRevocationMu.RUnlock()
							if revoked {
								return NIL
							}
						}
						return oldFn(args...)
					},
				}
			}

			// Webhook signature & verification
			kMap.Pairs["tandaTanganWebhook"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 2 {
						return &String{Value: ""}
					}
					payload := []byte(args[0].Inspect())
					secret := []byte(args[1].Inspect())
					alg := "sha256"
					if len(args) >= 3 {
						alg = strings.ToLower(args[2].Inspect())
					}
					var mac []byte
					if alg == "sha512" {
						h := hmac.New(sha512.New, secret)
						h.Write(payload)
						mac = h.Sum(nil)
					} else {
						h := hmac.New(sha256.New, secret)
						h.Write(payload)
						mac = h.Sum(nil)
					}
					return &String{Value: hex.EncodeToString(mac)}
				},
			}

			kMap.Pairs["verifikasiWebhook"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 3 {
						return FALSE
					}
					payload := []byte(args[0].Inspect())
					signature := args[1].Inspect()
					secret := []byte(args[2].Inspect())
					alg := "sha256"
					if len(args) >= 4 {
						alg = strings.ToLower(args[3].Inspect())
					}
					cleanSig := strings.TrimPrefix(signature, "sha256=")
					cleanSig = strings.TrimPrefix(cleanSig, "sha512=")
					sigBytes, err := hex.DecodeString(cleanSig)
					if err != nil {
						return FALSE
					}

					var expectedMac []byte
					if alg == "sha512" {
						h := hmac.New(sha512.New, secret)
						h.Write(payload)
						expectedMac = h.Sum(nil)
					} else {
						h := hmac.New(sha256.New, secret)
						h.Write(payload)
						expectedMac = h.Sum(nil)
					}

					if subtle.ConstantTimeCompare(sigBytes, expectedMac) == 1 {
						return TRUE
					}
					return FALSE
				},
			}
		}
	}

	// 2. Modul teks: sanitasiHTML
	if tObj, ok := env.Get("teks"); ok {
		if tMap, isMap := tObj.(*Map); isMap {
			tMap.Pairs["sanitasiHTML"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 1 {
						return &String{Value: ""}
					}
					escaped := html.EscapeString(args[0].Inspect())
					escaped = strings.ReplaceAll(escaped, "&#39;", "&#x27;")
					escaped = strings.ReplaceAll(escaped, "'", "&#x27;")
					escaped = strings.ReplaceAll(escaped, "&#34;", "&quot;")
					escaped = strings.ReplaceAll(escaped, "\"", "&quot;")
					return &String{Value: escaped}
				},
			}
		}
	}

	// 3. Modul sistem: Strict Sandboxing check
	if sObj, ok := env.Get("sistem"); ok {
		if sMap, isMap := sObj.(*Map); isMap {
			if origPerintah, exists := sMap.Pairs["perintah"].(*Builtin); exists {
				oldFn := origPerintah.Fn
				sMap.Pairs["perintah"] = &Builtin{
					Fn: func(args ...Object) Object {
						if IsStrictSandboxMode() {
							res := &Map{Pairs: make(map[string]Object)}
							res.Pairs["sukses"] = FALSE
							res.Pairs["kode"] = &Integer{Value: 126}
							res.Pairs["keluaran"] = &String{Value: "sistem.perintah dinonaktifkan dalam mode sandboxing"}
							return res
						}
						return oldFn(args...)
					},
				}
				env.Set("perintah", sMap.Pairs["perintah"])
			}
			if origEksekusi, exists := sMap.Pairs["eksekusi"].(*Builtin); exists {
				oldFn := origEksekusi.Fn
				sMap.Pairs["eksekusi"] = &Builtin{
					Fn: func(args ...Object) Object {
						if IsStrictSandboxMode() {
							res := &Map{Pairs: make(map[string]Object)}
							res.Pairs["sukses"] = FALSE
							res.Pairs["kode"] = &Integer{Value: 126}
							res.Pairs["stdout"] = &String{Value: ""}
							res.Pairs["stderr"] = &String{Value: "sistem.eksekusi dinonaktifkan dalam mode sandboxing"}
							return res
						}
						return oldFn(args...)
					},
				}
			}
		}
	}

	// 4. Modul berkas: alirkan & memori VFS
	if bObj, ok := env.Get("berkas"); ok {
		if bMap, isMap := bObj.(*Map); isMap {
			bMap.Pairs["alirkan"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 2 {
						return FALSE
					}
					src := args[0].Inspect()
					dst := args[1].Inspect()
					sf, err := os.Open(src)
					if err != nil {
						return FALSE
					}
					defer sf.Close()

					_ = os.MkdirAll(filepath.Dir(dst), 0755)
					df, err := os.Create(dst)
					if err != nil {
						return FALSE
					}
					defer df.Close()

					_, err = io.Copy(df, sf)
					if err != nil {
						return FALSE
					}
					return TRUE
				},
			}

			// Virtual Memory File System (berkas.memori)
			memMap := &Map{Pairs: make(map[string]Object)}
			memMap.Pairs["taruh"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 2 {
						return FALSE
					}
					path := args[0].Inspect()
					content := []byte(args[1].Inspect())
					RegisterVirtualFile(path, content)
					return TRUE
				},
			}
			memMap.Pairs["baca"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 1 {
						return &String{Value: ""}
					}
					data, ok := GetVirtualFile(args[0].Inspect())
					if !ok {
						return &String{Value: ""}
					}
					return &String{Value: string(data)}
				},
			}
			memMap.Pairs["ada"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 1 {
						return FALSE
					}
					_, ok := GetVirtualFile(args[0].Inspect())
					if ok {
						return TRUE
					}
					return FALSE
				},
			}
			memMap.Pairs["daftar"] = &Builtin{
				Fn: func(args ...Object) Object {
					prefix := ""
					if len(args) > 0 {
						prefix = filepath.ToSlash(filepath.Clean(args[0].Inspect()))
						prefix = strings.TrimPrefix(prefix, "/")
					}
					globalVirtualFSMu.RLock()
					defer globalVirtualFSMu.RUnlock()
					arr := &Array{Elements: []Object{}}
					for k := range globalVirtualFS {
						if prefix == "" || strings.HasPrefix(k, prefix) {
							arr.Elements = append(arr.Elements, &String{Value: k})
						}
					}
					return arr
				},
			}
			memMap.Pairs["hapus"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 1 {
						return FALSE
					}
					clean := filepath.ToSlash(filepath.Clean(args[0].Inspect()))
					clean = strings.TrimPrefix(clean, "/")
					globalVirtualFSMu.Lock()
					delete(globalVirtualFS, clean)
					globalVirtualFSMu.Unlock()
					return TRUE
				},
			}
			bMap.Pairs["memori"] = memMap
		}
	}

	// 5. Modul antrean: antrean asinkron latar belakang
	antreanMap := &Map{Pairs: make(map[string]Object)}
	antreanMap.Pairs["proses"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			topic := args[0].Inspect()
			fnObj := args[1]
			asyncQueueMu.Lock()
			asyncQueueHandlers[topic] = func(p Object) {
				applyFunction(fnObj, []Object{p}, env)
			}
			asyncQueueMu.Unlock()
			return TRUE
		},
	}
	antreanMap.Pairs["dorong"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 2 {
				return FALSE
			}
			topic := args[0].Inspect()
			payload := args[1]
			go func() {
				asyncQueueMu.RLock()
				handler, exists := asyncQueueHandlers[topic]
				asyncQueueMu.RUnlock()
				if exists && handler != nil {
					handler(payload)
				}
			}()
			return TRUE
		},
	}
	antreanMap.Pairs["tunda"] = &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) < 3 {
				return FALSE
			}
			topic := args[0].Inspect()
			durStr := args[1].Inspect()
			payload := args[2]
			dur, err := time.ParseDuration(durStr)
			if err != nil {
				dur = 1 * time.Second
			}
			go func() {
				time.Sleep(dur)
				asyncQueueMu.RLock()
				handler, exists := asyncQueueHandlers[topic]
				asyncQueueMu.RUnlock()
				if exists && handler != nil {
					handler(payload)
				}
			}()
			return TRUE
		},
	}
	env.Set("antrean", antreanMap)

	// 6. Modul jendela desktop: State reaktif
	if jObj, ok := env.Get("jendela"); ok {
		if jMap, isMap := jObj.(*Map); isMap {
			jMap.Pairs["ikatState"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 2 {
						return FALSE
					}
					key := args[0].Inspect()
					val := args[1]
					desktopStateMu.Lock()
					desktopStateMap[key] = val
					listeners := make([]func(string, Object), len(desktopStateListeners))
					copy(listeners, desktopStateListeners)
					desktopStateMu.Unlock()

					for _, l := range listeners {
						l(key, val)
					}
					return TRUE
				},
			}
			jMap.Pairs["ambilState"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 1 {
						return NIL
					}
					key := args[0].Inspect()
					desktopStateMu.RLock()
					val, exists := desktopStateMap[key]
					desktopStateMu.RUnlock()
					if exists {
						return val
					}
					return NIL
				},
			}
			jMap.Pairs["padaPerubahanState"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 1 {
						return FALSE
					}
					cbObj := args[0]
					desktopStateMu.Lock()
					desktopStateListeners = append(desktopStateListeners, func(k string, v Object) {
						applyFunction(cbObj, []Object{&String{Value: k}, v}, env)
					})
					desktopStateMu.Unlock()
					return TRUE
				},
			}
		}
	}

	// 7. Modul web: web.saluran, web.versi, web.kesehatan, web.dokumentasi
	if wObj, ok := env.Get("web"); ok {
		if wMap, isMap := wObj.(*Map); isMap {
			wMap.Pairs["batasPermintaan"] = &Builtin{
				Fn: func(args ...Object) Object {
					return TRUE
				},
			}

			// web.versi("v1") -> grup rute ber-prefix
			wMap.Pairs["versi"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 1 {
						return NIL
					}
					ver := "/" + strings.Trim(args[0].Inspect(), "/")
					if grpFn, ok := wMap.Pairs["grup"].(*Builtin); ok {
						return grpFn.Fn(&String{Value: ver})
					}
					return NIL
				},
			}

			// web.saluran("nama") -> broadcasting hub
			channelsMap := make(map[string]*Map)
			var chMu sync.Mutex
			wMap.Pairs["saluran"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 1 {
						return NIL
					}
					chName := args[0].Inspect()
					chMu.Lock()
					defer chMu.Unlock()
					if existing, ok := channelsMap[chName]; ok {
						return existing
					}

					var listeners []func(string)
					var lMu sync.Mutex

					saluranObj := &Map{Pairs: make(map[string]Object)}
					saluranObj.Pairs["langgan"] = &Builtin{
						Fn: func(lArgs ...Object) Object {
							if len(lArgs) < 1 {
								return NIL
							}
							fn := lArgs[0]
							lMu.Lock()
							listeners = append(listeners, func(msg string) {
								applyFunction(fn, []Object{&String{Value: msg}}, env)
							})
							lMu.Unlock()
							return TRUE
						},
					}
					saluranObj.Pairs["kirimSemua"] = &Builtin{
						Fn: func(kArgs ...Object) Object {
							if len(kArgs) < 1 {
								return FALSE
							}
							msg := kArgs[0].Inspect()
							lMu.Lock()
							active := make([]func(string), len(listeners))
							copy(active, listeners)
							lMu.Unlock()
							for _, lis := range active {
								lis(msg)
							}
							return TRUE
						},
					}

					channelsMap[chName] = saluranObj
					return saluranObj
				},
			}

			// web.kesehatan("/kesehatan", [opsiMap])
			wMap.Pairs["kesehatan"] = &Builtin{
				Fn: func(args ...Object) Object {
					path := "/kesehatan"
					if len(args) > 0 && args[0].Type() == STRING_OBJ {
						path = args[0].Inspect()
					}
					var checkOpts *Map
					if len(args) > 1 {
						checkOpts, _ = args[1].(*Map)
					} else if len(args) == 1 && args[0].Type() == MAP_OBJ {
						checkOpts, _ = args[0].(*Map)
					}

					if getMethod, ok := wMap.Pairs["dapatkan"].(*Builtin); ok {
						getMethod.Fn(&String{Value: path}, &Builtin{
							Fn: func(hArgs ...Object) Object {
								var m runtime.MemStats
								runtime.ReadMemStats(&m)

								res := &Map{Pairs: make(map[string]Object)}
								res.Pairs["status"] = &String{Value: "SEHAT"}
								res.Pairs["waktu"] = &String{Value: time.Now().Format(time.RFC3339)}
								res.Pairs["goroutine"] = &Integer{Value: int64(runtime.NumGoroutine())}
								res.Pairs["memoriMB"] = &Float{Value: float64(m.Alloc) / 1024 / 1024}

								sehat := true
								if checkOpts != nil {
									if dbObj, hasDB := checkOpts.Pairs["basisdata"]; hasDB {
										if dbMap, isDB := dbObj.(*Map); isDB {
											if pingFn, hasPing := dbMap.Pairs["kueri"].(*Builtin); hasPing {
												testRes := pingFn.Fn(&String{Value: "SELECT 1;"})
												if testRes == nil {
													sehat = false
												}
											}
										}
									}
								}

								if !sehat {
									res.Pairs["status"] = &String{Value: "TIDAK_SEHAT"}
								}

								if len(hArgs) > 0 {
									if ctx, isCtx := hArgs[0].(*Map); isCtx {
										if jsonFn, hasJson := ctx.Pairs["json"].(*Builtin); hasJson {
											code := 200
											if !sehat {
												code = 503
											}
											return jsonFn.Fn(res, &Integer{Value: int64(code)})
										}
									}
								}
								return res
							},
						})
					}
					return TRUE
				},
			}

			// web.dokumentasi("/docs", [opsiMap])
			wMap.Pairs["dokumentasi"] = &Builtin{
				Fn: func(args ...Object) Object {
					docPath := "/docs"
					title := "Bahasa Ku API"
					versionDoc := "1.0.0"
					desc := "Dokumentasi API Otomatis Bahasa Ku"

					if len(args) > 0 && args[0].Type() == STRING_OBJ {
						docPath = args[0].Inspect()
					}
					if len(args) > 1 {
						if cfgMap, ok := args[1].(*Map); ok {
							if tObj, ok := cfgMap.Pairs["judul"]; ok {
								title = tObj.Inspect()
							}
							if vObj, ok := cfgMap.Pairs["versi"]; ok {
								versionDoc = vObj.Inspect()
							}
							if dObj, ok := cfgMap.Pairs["deskripsi"]; ok {
								desc = dObj.Inspect()
							}
						}
					}

					cleanDocPath := "/" + strings.Trim(docPath, "/")
					openAPIPath := cleanDocPath + "/openapi.json"

					if getMethod, ok := wMap.Pairs["dapatkan"].(*Builtin); ok {
						getMethod.Fn(&String{Value: openAPIPath}, &Builtin{
							Fn: func(hArgs ...Object) Object {
								spec := map[string]interface{}{
									"openapi": "3.0.0",
									"info": map[string]interface{}{
										"title":       title,
										"version":     versionDoc,
										"description": desc,
									},
									"paths": map[string]interface{}{
										"/": map[string]interface{}{
											"get": map[string]interface{}{
												"summary": "Root Endpoint",
												"responses": map[string]interface{}{
													"200": map[string]interface{}{
														"description": "Respons Berhasil",
													},
												},
											},
										},
									},
								}
								if len(hArgs) > 0 {
									if ctx, isCtx := hArgs[0].(*Map); isCtx {
										if jsonFn, hasJson := ctx.Pairs["json"].(*Builtin); hasJson {
											return jsonFn.Fn(convertGoToKuObject(spec), &Integer{Value: 200})
										}
									}
								}
								return convertGoToKuObject(spec)
							},
						})

						swaggerHTML := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
  <meta charset="utf-8" />
  <title>%s - Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
window.onload = () => {
  window.ui = SwaggerUIBundle({
    url: '%s',
    dom_id: '#swagger-ui',
  });
};
</script>
</body>
</html>`, title, openAPIPath)

						getMethod.Fn(&String{Value: cleanDocPath}, &Builtin{
							Fn: func(hArgs ...Object) Object {
								if len(hArgs) > 0 {
									if ctx, isCtx := hArgs[0].(*Map); isCtx {
										if htmlFn, hasHtml := ctx.Pairs["html"].(*Builtin); hasHtml {
											return htmlFn.Fn(&String{Value: swaggerHTML}, &Integer{Value: 200})
										}
									}
								}
								return &String{Value: swaggerHTML}
							},
						})
					}

					return TRUE
				},
			}
		}
	}

	// 8. Modul arsip: tarGz
	if aObj, ok := env.Get("arsip"); ok {
		if aMap, isMap := aObj.(*Map); isMap {
			aMap.Pairs["buatTarGz"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 2 {
						return FALSE
					}
					outTarGz := args[0].Inspect()
					srcDir := args[1].Inspect()

					outFile, err := os.Create(outTarGz)
					if err != nil {
						return FALSE
					}
					defer outFile.Close()

					gw := gzip.NewWriter(outFile)
					defer gw.Close()

					tw := tar.NewWriter(gw)
					defer tw.Close()

					walkErr := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							return err
						}
						rel, err := filepath.Rel(srcDir, path)
						if err != nil {
							return err
						}
						if rel == "." {
							return nil
						}
						hdr, err := tar.FileInfoHeader(info, info.Name())
						if err != nil {
							return err
						}
						hdr.Name = filepath.ToSlash(rel)
						if err := tw.WriteHeader(hdr); err != nil {
							return err
						}
						if !info.IsDir() {
							f, err := os.Open(path)
							if err != nil {
								return err
							}
							defer f.Close()
							_, err = io.Copy(tw, f)
							return err
						}
						return nil
					})

					if walkErr != nil {
						return FALSE
					}
					return TRUE
				},
			}

			aMap.Pairs["ekstrakTarGz"] = &Builtin{
				Fn: func(args ...Object) Object {
					if len(args) < 2 {
						return FALSE
					}
					tarGzPath := args[0].Inspect()
					destDir := args[1].Inspect()

					f, err := os.Open(tarGzPath)
					if err != nil {
						return FALSE
					}
					defer f.Close()

					gr, err := gzip.NewReader(f)
					if err != nil {
						return FALSE
					}
					defer gr.Close()

					tr := tar.NewReader(gr)
					for {
						hdr, err := tr.Next()
						if err == io.EOF {
							break
						}
						if err != nil {
							return FALSE
						}

						// Proteksi Zip/Tar Slip
						target := filepath.Join(destDir, hdr.Name)
						if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)) {
							return FALSE
						}

						if hdr.Typeflag == tar.TypeDir {
							_ = os.MkdirAll(target, 0755)
						} else {
							_ = os.MkdirAll(filepath.Dir(target), 0755)
							outF, err := os.Create(target)
							if err != nil {
								return FALSE
							}
							_, _ = io.Copy(outF, tr)
							outF.Close()
						}
					}
					return TRUE
				},
			}
		}
	}

	// 9. Modul singgahan: Redis wire protocol RESP murni
	singgahanObj, hasSinggahan := env.Get("singgahan")
	var sMap *Map
	if hasSinggahan {
		sMap, _ = singgahanObj.(*Map)
	} else {
		sMap = &Map{Pairs: make(map[string]Object)}
		env.Set("singgahan", sMap)
	}

	sMap.Pairs["redis"] = &Builtin{
		Fn: func(args ...Object) Object {
			addr := "127.0.0.1:6379"
			if len(args) > 0 && args[0].Type() == STRING_OBJ {
				addr = args[0].Inspect()
			}

			rClient := &Map{Pairs: make(map[string]Object)}

			// Helper eksekusi perintah RESP mentah
			execResp := func(cmdParts ...string) (string, error) {
				conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
				if err != nil {
					return "", err
				}
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(3 * time.Second))

				// Format RESP Array: *<len>\r\n$<len>\r\n<arg>\r\n...
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("*%d\r\n", len(cmdParts)))
				for _, p := range cmdParts {
					sb.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(p), p))
				}
				if _, err := conn.Write([]byte(sb.String())); err != nil {
					return "", err
				}

				reader := bufio.NewReader(conn)
				prefix, err := reader.ReadByte()
				if err != nil {
					return "", err
				}

				switch prefix {
				case '+': // Simple String
					line, _, err := reader.ReadLine()
					return string(line), err
				case ':': // Integer
					line, _, err := reader.ReadLine()
					return string(line), err
				case '$': // Bulk String
					lenLine, _, err := reader.ReadLine()
					if err != nil {
						return "", err
					}
					strLen, _ := strconv.Atoi(string(lenLine))
					if strLen < 0 {
						return "", nil // Nil
					}
					buf := make([]byte, strLen+2) // with CRLF
					_, err = io.ReadFull(reader, buf)
					if err != nil {
						return "", err
					}
					return string(buf[:strLen]), nil
				case '-': // Error
					line, _, _ := reader.ReadLine()
					return "", fmt.Errorf("redis galat: %s", string(line))
				default:
					line, _, err := reader.ReadLine()
					return string(line), err
				}
			}

			rClient.Pairs["taruh"] = &Builtin{
				Fn: func(rArgs ...Object) Object {
					if len(rArgs) < 2 {
						return FALSE
					}
					k := rArgs[0].Inspect()
					v := rArgs[1].Inspect()
					cmd := []string{"SET", k, v}
					if len(rArgs) >= 3 {
						ttlStr := rArgs[2].Inspect()
						if d, err := time.ParseDuration(ttlStr); err == nil {
							secs := int(d.Seconds())
							if secs > 0 {
								cmd = append(cmd, "EX", strconv.Itoa(secs))
							}
						}
					}
					res, err := execResp(cmd...)
					if err == nil && (res == "OK" || res != "") {
						return TRUE
					}
					return FALSE
				},
			}

			rClient.Pairs["ambil"] = &Builtin{
				Fn: func(rArgs ...Object) Object {
					if len(rArgs) < 1 {
						return NIL
					}
					k := rArgs[0].Inspect()
					val, err := execResp("GET", k)
					if err != nil || val == "" {
						return NIL
					}
					return &String{Value: val}
				},
			}

			rClient.Pairs["hapus"] = &Builtin{
				Fn: func(rArgs ...Object) Object {
					if len(rArgs) < 1 {
						return FALSE
					}
					k := rArgs[0].Inspect()
					res, err := execResp("DEL", k)
					if err == nil && res != "0" {
						return TRUE
					}
					return FALSE
				},
			}

			rClient.Pairs["ada"] = &Builtin{
				Fn: func(rArgs ...Object) Object {
					if len(rArgs) < 1 {
						return FALSE
					}
					k := rArgs[0].Inspect()
					res, err := execResp("EXISTS", k)
					if err == nil && res != "0" && res != "" {
						return TRUE
					}
					return FALSE
				},
			}

			rClient.Pairs["tambah"] = &Builtin{
				Fn: func(rArgs ...Object) Object {
					if len(rArgs) < 1 {
						return &Integer{Value: 0}
					}
					k := rArgs[0].Inspect()
					incr := "1"
					if len(rArgs) >= 2 {
						incr = rArgs[1].Inspect()
					}
					res, err := execResp("INCRBY", k, incr)
					if err != nil {
						return &Integer{Value: 0}
					}
					n, _ := strconv.ParseInt(res, 10, 64)
					return &Integer{Value: n}
				},
			}

			rClient.Pairs["dorongKiri"] = &Builtin{
				Fn: func(rArgs ...Object) Object {
					if len(rArgs) < 2 {
						return FALSE
					}
					k := rArgs[0].Inspect()
					v := rArgs[1].Inspect()
					_, err := execResp("LPUSH", k, v)
					if err == nil {
						return TRUE
					}
					return FALSE
				},
			}

			rClient.Pairs["tarikKanan"] = &Builtin{
				Fn: func(rArgs ...Object) Object {
					if len(rArgs) < 1 {
						return NIL
					}
					k := rArgs[0].Inspect()
					val, err := execResp("RPOP", k)
					if err != nil || val == "" {
						return NIL
					}
					return &String{Value: val}
				},
			}

			return rClient
		},
	}

	// 10. Modul basisdata: Tambahan migrasi & query builder gabung/ambilHalaman
	if bdObj, ok := env.Get("bd"); ok {
		if bdMap, isMap := bdObj.(*Map); isMap {
			if sambungFn, ok := bdMap.Pairs["sambung"].(*Builtin); ok {
				oldSambung := sambungFn.Fn
				bdMap.Pairs["sambung"] = &Builtin{
					Fn: func(args ...Object) Object {
						connObj := oldSambung(args...)
						if connMap, isConn := connObj.(*Map); isConn {
							// conn.migrasi([folder])
							connMap.Pairs["migrasi"] = &Builtin{
								Fn: func(mArgs ...Object) Object {
									folder := "migrasi"
									if len(mArgs) > 0 {
										folder = mArgs[0].Inspect()
									}
									entries, err := os.ReadDir(folder)
									if err != nil {
										return &Integer{Value: 0}
									}
									var files []string
									for _, e := range entries {
										if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
											files = append(files, filepath.Join(folder, e.Name()))
										}
									}
									sort.Strings(files)

									var count int64 = 0
									for _, f := range files {
										sqlContent, rErr := os.ReadFile(f)
										if rErr != nil {
											continue
										}
										queries := strings.Split(string(sqlContent), ";")
										for _, q := range queries {
											cleanQ := strings.TrimSpace(q)
											if cleanQ != "" {
												if execFn, ok := connMap.Pairs["eksekusi"].(*Builtin); ok {
													execFn.Fn(&String{Value: cleanQ})
												}
											}
										}
										count++
									}
									return &Integer{Value: count}
								},
							}

							// Patch method tabel() untuk ambilHalaman
							if origTabel, ok := connMap.Pairs["tabel"].(*Builtin); ok {
								oldTabelFn := origTabel.Fn
								connMap.Pairs["tabel"] = &Builtin{
									Fn: func(tArgs ...Object) Object {
										tbObj := oldTabelFn(tArgs...)
										if tbMap, isTb := tbObj.(*Map); isTb {
											extendTableQueryBuilder(tbMap)
										}
										return tbObj
									},
								}
							}
						}
						return connObj
					},
				}
			}
		}
	}

	// 11. Modul telemetri: Distributed Tracing W3C Traceparent
	telemetriMap := &Map{Pairs: make(map[string]Object)}
	telemetriMap.Pairs["mulaiRentang"] = &Builtin{
		Fn: func(args ...Object) Object {
			name := "rentang"
			if len(args) > 0 && args[0].Type() == STRING_OBJ {
				name = args[0].Inspect()
			}
			traceIDBytes := make([]byte, 16)
			spanIDBytes := make([]byte, 8)
			_, _ = io.ReadFull(rand.Reader, traceIDBytes)
			_, _ = io.ReadFull(rand.Reader, spanIDBytes)

			traceID := hex.EncodeToString(traceIDBytes)
			spanID := hex.EncodeToString(spanIDBytes)

			// Jika ada rentang induk, warisi traceID
			if len(args) > 1 {
				if parentMap, ok := args[1].(*Map); ok {
					if pTrace, exists := parentMap.Pairs["traceID"]; exists {
						traceID = pTrace.Inspect()
					}
				}
			}

			traceparent := fmt.Sprintf("00-%s-%s-01", traceID, spanID)
			startTime := time.Now()

			spanObj := &Map{Pairs: make(map[string]Object)}
			spanObj.Pairs["nama"] = &String{Value: name}
			spanObj.Pairs["traceID"] = &String{Value: traceID}
			spanObj.Pairs["spanID"] = &String{Value: spanID}
			spanObj.Pairs["traceparent"] = &String{Value: traceparent}

			attributes := make(map[string]Object)
			spanObj.Pairs["tambahAtribut"] = &Builtin{
				Fn: func(aArgs ...Object) Object {
					if len(aArgs) >= 2 {
						k := aArgs[0].Inspect()
						attributes[k] = aArgs[1]
					}
					return spanObj
				},
			}
			spanObj.Pairs["ambilAtribut"] = &Builtin{
				Fn: func(aArgs ...Object) Object {
					if len(aArgs) >= 1 {
						k := aArgs[0].Inspect()
						if val, exists := attributes[k]; exists {
							return val
						}
					}
					return NIL
				},
			}
			spanObj.Pairs["selesai"] = &Builtin{
				Fn: func(sArgs ...Object) Object {
					dur := time.Since(startTime).Seconds() * 1000
					spanObj.Pairs["durasiMs"] = &Float{Value: dur}
					spanObj.Pairs["selesai"] = TRUE
					return &Float{Value: dur}
				},
			}

			return spanObj
		},
	}
	env.Set("telemetri", telemetriMap)
}

func extendTableQueryBuilder(tbMap *Map) {
	// Alias .ambil() sama dengan .ambilSemua()
	tbMap.Pairs["ambil"] = tbMap.Pairs["ambilSemua"]

	// .ambilHalaman(nomor, perHalaman)
	tbMap.Pairs["ambilHalaman"] = &Builtin{
		Fn: func(hArgs ...Object) Object {
			noHal := int64(1)
			perHal := int64(10)
			if len(hArgs) >= 1 {
				if hi, ok := hArgs[0].(*Integer); ok && hi.Value > 0 {
					noHal = hi.Value
				}
			}
			if len(hArgs) >= 2 {
				if pi, ok := hArgs[1].(*Integer); ok && pi.Value > 0 {
					perHal = pi.Value
				}
			}

			// Hitung total data
			var total int64 = 0
			if countFn, ok := tbMap.Pairs["hitung"].(*Builtin); ok {
				if cRes, ok := countFn.Fn().(*Integer); ok {
					total = cRes.Value
				}
			}

			// Paginasi query
			if batasFn, ok := tbMap.Pairs["batas"].(*Builtin); ok {
				batasFn.Fn(&Integer{Value: perHal})
			}
			if lewatFn, ok := tbMap.Pairs["lewat"].(*Builtin); ok {
				lewatFn.Fn(&Integer{Value: (noHal - 1) * perHal})
			}

			dataArr := tbMap.Pairs["ambilSemua"].(*Builtin).Fn()

			totalHalaman := int64(0)
			if perHal > 0 {
				totalHalaman = (total + perHal - 1) / perHal
			}

			resMap := &Map{Pairs: make(map[string]Object)}
			resMap.Pairs["total"] = &Integer{Value: total}
			resMap.Pairs["halaman"] = &Integer{Value: noHal}
			resMap.Pairs["perHalaman"] = &Integer{Value: perHal}
			resMap.Pairs["totalHalaman"] = &Integer{Value: totalHalaman}
			resMap.Pairs["data"] = dataArr
			return resMap
		},
	}
}

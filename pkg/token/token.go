package token

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifier & Literals
	IDENT  = "IDENT"
	INT    = "INT"
	FLOAT  = "FLOAT"
	STRING = "STRING"

	// Operators
	ASSIGN   = "="
	DEFINE   = ":="
	PLUS     = "+"
	MINUS    = "-"
	BANG     = "!"
	ASTERISK = "*"
	SLASH    = "/"
	MOD      = "%"
	INC      = "++"
	DEC      = "--"

	// Compound Assignment Operators
	PLUS_ASSIGN  = "+="
	MINUS_ASSIGN = "-="
	MUL_ASSIGN   = "*="
	DIV_ASSIGN   = "/="
	MOD_ASSIGN   = "%="

	EQ     = "=="
	NOT_EQ = "!="
	LT     = "<"
	GT     = ">"
	LTE    = "<="
	GTE    = ">="
	AND    = "&&"
	OR     = "||"

	// Bitwise Operators
	BIT_AND = "&"
	BIT_OR  = "|"
	BIT_XOR = "^"
	SHL     = "<<"
	SHR     = ">>"

	// Delimiters
	COMMA     = ","
	SEMICOLON = ";"
	COLON     = ":"
	DOT       = "."

	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"

	// Keywords Bahasa Ku
	PAKET      = "PAKET"
	PAKAI      = "PAKAI"
	FUNGSI     = "FUNGSI"
	VAR        = "VAR"
	TETAP      = "TETAP"
	TIPE       = "TIPE"
	STRUKTUR   = "STRUKTUR"
	JIKA       = "JIKA"
	LAINNYA    = "LAINNYA"
	UNTUK      = "UNTUK"
	SELAMA     = "SELAMA"
	DALAM      = "DALAM"
	RENTANG    = "RENTANG"
	PILIH      = "PILIH"
	KASUS      = "KASUS"
	BAWAAN     = "BAWAAN"
	KEMBALIKAN = "KEMBALIKAN"
	TUNDA      = "TUNDA"
	HENTI      = "HENTI"
	LANJUT     = "LANJUT"
	BENAR      = "BENAR"
	SALAH      = "SALAH"
	KOSONG     = "KOSONG"
)

var keywords = map[string]TokenType{
	"paket":      PAKET,
	"pakai":      PAKAI,
	"impor":      PAKAI,
	"fungsi":     FUNGSI,
	"var":        VAR,
	"variabel":   VAR,
	"tetap":      TETAP,
	"konst":      TETAP,
	"tipe":       TIPE,
	"struktur":   STRUKTUR,
	"jika":       JIKA,
	"lainnya":    LAINNYA,
	"untuk":      UNTUK,
	"selama":     SELAMA,
	"dalam":      DALAM,
	"rentang":    RENTANG,
	"pilih":      PILIH,
	"kasus":      KASUS,
	"bawaan":     BAWAAN,
	"kembalikan": KEMBALIKAN,
	"tunda":      TUNDA,
	"henti":      HENTI,
	"lanjut":     LANJUT,
	"benar":      BENAR,
	"salah":      SALAH,
	"kosong":     KOSONG,
	"nihil":      KOSONG,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

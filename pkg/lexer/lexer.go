package lexer

import (
	"github.com/zufarrizal/bahasa-ku/pkg/token"
)

type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int
	column       int
}

func New(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.column++
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		l.readChar()
	}
}

func (l *Lexer) skipComment() {
	if l.ch == '/' && l.peekChar() == '/' {
		for l.ch != '\n' && l.ch != 0 {
			l.readChar()
		}
		l.skipWhitespace()
		l.skipComment()
	} else if l.ch == '/' && l.peekChar() == '*' {
		l.readChar()
		l.readChar()
		for !(l.ch == '*' && l.peekChar() == '/') && l.ch != 0 {
			if l.ch == '\n' {
				l.line++
				l.column = 0
			}
			l.readChar()
		}
		l.readChar() // *
		l.readChar() // /
		l.skipWhitespace()
		l.skipComment()
	}
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespace()
	l.skipComment()

	curLine := l.line
	curCol := l.column

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.EQ, Literal: string(ch) + string(l.ch), Line: curLine, Column: curCol}
		} else {
			tok = token.Token{Type: token.ASSIGN, Literal: string(l.ch), Line: curLine, Column: curCol}
		}
	case ':':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.DEFINE, Literal: string(ch) + string(l.ch), Line: curLine, Column: curCol}
		} else {
			tok = token.Token{Type: token.COLON, Literal: string(l.ch), Line: curLine, Column: curCol}
		}
	case '+':
		if l.peekChar() == '+' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.INC, Literal: string(ch) + string(l.ch), Line: curLine, Column: curCol}
		} else if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.PLUS_ASSIGN, Literal: string(ch) + string(l.ch), Line: curLine, Column: curCol}
		} else {
			tok = token.Token{Type: token.PLUS, Literal: string(l.ch), Line: curLine, Column: curCol}
		}
	case '-':
		if l.peekChar() == '-' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.DEC, Literal: string(ch) + string(l.ch), Line: curLine, Column: curCol}
		} else if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.MINUS_ASSIGN, Literal: string(ch) + string(l.ch), Line: curLine, Column: curCol}
		} else {
			tok = token.Token{Type: token.MINUS, Literal: string(l.ch), Line: curLine, Column: curCol}
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.NOT_EQ, Literal: string(ch) + string(l.ch), Line: curLine, Column: curCol}
		} else {
			tok = token.Token{Type: token.BANG, Literal: string(l.ch), Line: curLine, Column: curCol}
		}
	case '*':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.MUL_ASSIGN, Literal: string(ch) + string(l.ch), Line: curLine, Column: curCol}
		} else {
			tok = token.Token{Type: token.ASTERISK, Literal: string(l.ch), Line: curLine, Column: curCol}
		}
	case '/':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.DIV_ASSIGN, Literal: string(ch) + string(l.ch), Line: curLine, Column: curCol}
		} else {
			tok = token.Token{Type: token.SLASH, Literal: string(l.ch), Line: curLine, Column: curCol}
		}
	case '%':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.MOD_ASSIGN, Literal: string(ch) + string(l.ch), Line: curLine, Column: curCol}
		} else {
			tok = token.Token{Type: token.MOD, Literal: string(l.ch), Line: curLine, Column: curCol}
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.LTE, Literal: string(ch) + string(l.ch), Line: curLine, Column: curCol}
		} else if l.peekChar() == '<' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.SHL, Literal: string(ch) + string(l.ch), Line: curLine, Column: curCol}
		} else {
			tok = token.Token{Type: token.LT, Literal: string(l.ch), Line: curLine, Column: curCol}
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.GTE, Literal: string(ch) + string(l.ch), Line: curLine, Column: curCol}
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.SHR, Literal: string(ch) + string(l.ch), Line: curLine, Column: curCol}
		} else {
			tok = token.Token{Type: token.GT, Literal: string(l.ch), Line: curLine, Column: curCol}
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.AND, Literal: string(ch) + string(l.ch), Line: curLine, Column: curCol}
		} else {
			tok = token.Token{Type: token.BIT_AND, Literal: string(l.ch), Line: curLine, Column: curCol}
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.OR, Literal: string(ch) + string(l.ch), Line: curLine, Column: curCol}
		} else {
			tok = token.Token{Type: token.BIT_OR, Literal: string(l.ch), Line: curLine, Column: curCol}
		}
	case '^':
		tok = token.Token{Type: token.BIT_XOR, Literal: string(l.ch), Line: curLine, Column: curCol}
	case ';':
		tok = token.Token{Type: token.SEMICOLON, Literal: string(l.ch), Line: curLine, Column: curCol}
	case ',':
		tok = token.Token{Type: token.COMMA, Literal: string(l.ch), Line: curLine, Column: curCol}
	case '.':
		if l.peekChar() == '.' {
			l.readChar()
			if l.peekChar() == '.' {
				l.readChar()
				tok = token.Token{Type: token.ELLIPSIS, Literal: "...", Line: curLine, Column: curCol}
			} else {
				tok = token.Token{Type: token.DOT, Literal: ".", Line: curLine, Column: curCol}
			}
		} else {
			tok = token.Token{Type: token.DOT, Literal: string(l.ch), Line: curLine, Column: curCol}
		}
	case '(':
		tok = token.Token{Type: token.LPAREN, Literal: string(l.ch), Line: curLine, Column: curCol}
	case ')':
		tok = token.Token{Type: token.RPAREN, Literal: string(l.ch), Line: curLine, Column: curCol}
	case '{':
		tok = token.Token{Type: token.LBRACE, Literal: string(l.ch), Line: curLine, Column: curCol}
	case '}':
		tok = token.Token{Type: token.RBRACE, Literal: string(l.ch), Line: curLine, Column: curCol}
	case '[':
		tok = token.Token{Type: token.LBRACKET, Literal: string(l.ch), Line: curLine, Column: curCol}
	case ']':
		tok = token.Token{Type: token.RBRACKET, Literal: string(l.ch), Line: curLine, Column: curCol}
	case '"', '`':
		quote := l.ch
		tok.Type = token.STRING
		tok.Literal = l.readString(quote)
		tok.Line = curLine
		tok.Column = curCol
		return tok
	case 0:
		tok = token.Token{Type: token.EOF, Literal: "", Line: curLine, Column: curCol}
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			tok.Line = curLine
			tok.Column = curCol
			return tok
		} else if isDigit(l.ch) {
			lit, isFloat := l.readNumber()
			if isFloat {
				tok.Type = token.FLOAT
			} else {
				tok.Type = token.INT
			}
			tok.Literal = lit
			tok.Line = curLine
			tok.Column = curCol
			return tok
		} else {
			tok = token.Token{Type: token.ILLEGAL, Literal: string(l.ch), Line: curLine, Column: curCol}
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) readIdentifier() string {
	pos := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[pos:l.position]
}

func (l *Lexer) readNumber() (string, bool) {
	pos := l.position
	isFloat := false
	for isDigit(l.ch) || (l.ch == '.' && isDigit(l.peekChar())) {
		if l.ch == '.' {
			isFloat = true
		}
		l.readChar()
	}
	return l.input[pos:l.position], isFloat
}

func (l *Lexer) readString(quote byte) string {
	l.readChar()
	pos := l.position
	res := ""
	for l.ch != quote && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				res += "\n"
			case 't':
				res += "\t"
			case 'r':
				res += "\r"
			case '"':
				res += "\""
			case '\\':
				res += "\\"
			default:
				res += string(l.ch)
			}
		} else {
			if l.ch == '\n' {
				l.line++
				l.column = 0
			}
			res += string(l.ch)
		}
		l.readChar()
	}
	l.readChar() // skip closing quote
	_ = pos
	return res
}

func isLetter(ch byte) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

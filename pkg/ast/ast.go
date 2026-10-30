package ast

import (
	"github.com/zufarrizal/bahasa-ku/pkg/token"
)

type Node interface {
	TokenLiteral() string
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

// Statements
type PaketStatement struct {
	Token token.Token // token.PAKET
	Name  string
}

func (ps *PaketStatement) statementNode()       {}
func (ps *PaketStatement) TokenLiteral() string { return ps.Token.Literal }

type PakaiStatement struct {
	Token token.Token // token.PAKAI
	Path  string
}

func (ps *PakaiStatement) statementNode()       {}
func (ps *PakaiStatement) TokenLiteral() string { return ps.Token.Literal }

type VarStatement struct {
	Token    token.Token // token.VAR or token.TETAP
	Name     *Identifier
	Names    []*Identifier // Mendukung multi-ident: var a, b = fn()
	Value    Expression
	Values   []Expression // Mendukung multi-value
	Constant bool
}

func (vs *VarStatement) statementNode()       {}
func (vs *VarStatement) TokenLiteral() string { return vs.Token.Literal }

type BlockStatement struct {
	Token      token.Token // token.LBRACE
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }

type ReturnStatement struct {
	Token        token.Token // token.KEMBALIKAN
	ReturnValue  Expression
	ReturnValues []Expression // multi-value return khas Go
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }

type BreakStatement struct {
	Token token.Token // token.HENTI
}

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }

type ContinueStatement struct {
	Token token.Token // token.LANJUT
}

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }

type DeferStatement struct {
	Token      token.Token // token.TUNDA
	Expression Expression
}

func (ds *DeferStatement) statementNode()       {}
func (ds *DeferStatement) TokenLiteral() string { return ds.Token.Literal }

type KuStatement struct {
	Token      token.Token // token.KU (konkurensi Ku-routine)
	Expression Expression
}

func (ks *KuStatement) statementNode()       {}
func (ks *KuStatement) TokenLiteral() string { return ks.Token.Literal }

type ExpressionStatement struct {
	Token      token.Token
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }

type AssignStatement struct {
	Token    token.Token // = or :=
	Target   Expression
	Targets  []Expression // Mendukung multi-assign: a, b = fn() atau a, b := 1, 2
	Value    Expression
	Values   []Expression // Mendukung multi-value
	IsDefine bool
}

func (as *AssignStatement) statementNode()       {}
func (as *AssignStatement) TokenLiteral() string { return as.Token.Literal }

type StructField struct {
	Name string
	Type string
}

type TypeStructStatement struct {
	Token  token.Token // token.TIPE
	Name   *Identifier
	Fields []StructField
}

func (ts *TypeStructStatement) statementNode()       {}
func (ts *TypeStructStatement) TokenLiteral() string { return ts.Token.Literal }

type FunctionReceiver struct {
	Name string // nama variabel receiver, misal 'l'
	Type string // nama tipe struct, misal 'Lingkaran'
}

type FunctionStatement struct {
	Token      token.Token // token.FUNGSI
	Receiver   *FunctionReceiver // Opsional method receiver: (l Lingkaran)
	Name       *Identifier
	Parameters []*Identifier
	IsVariadic bool // Parameter terakhir variadic (...args)
	Body       *BlockStatement
}

func (fs *FunctionStatement) statementNode()       {}
func (fs *FunctionStatement) TokenLiteral() string { return fs.Token.Literal }

type FunctionLiteral struct {
	Token      token.Token // token.FUNGSI
	Parameters []*Identifier
	IsVariadic bool
	Body       *BlockStatement
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }

type IfStatement struct {
	Token       token.Token // token.JIKA
	Condition   Expression
	Consequence *BlockStatement
	Alternative Statement // BlockStatement or IfStatement
}

func (is *IfStatement) statementNode()       {}
func (is *IfStatement) TokenLiteral() string { return is.Token.Literal }

type ForStatement struct {
	Token token.Token // token.UNTUK
	Init  Statement
	Cond  Expression
	Post  Statement
	Body  *BlockStatement

	// Dukungan untuk .. dalam (for .. range)
	IsRange bool
	Key     *Identifier
	Value   *Identifier
	Iterable Expression
}

func (fs *ForStatement) statementNode()       {}
func (fs *ForStatement) TokenLiteral() string { return fs.Token.Literal }

type SwitchCase struct {
	Value Expression
	Body  *BlockStatement
}

type SwitchStatement struct {
	Token   token.Token // token.PILIH
	Value   Expression
	Cases   []SwitchCase
	Default *BlockStatement
}

func (ss *SwitchStatement) statementNode()       {}
func (ss *SwitchStatement) TokenLiteral() string { return ss.Token.Literal }

// Expressions
type Identifier struct {
	Token token.Token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }

type IntegerLiteral struct {
	Token token.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }

type FloatLiteral struct {
	Token token.Token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }

type StringLiteral struct {
	Token token.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }

type BooleanLiteral struct {
	Token token.Token
	Value bool
}

func (b *BooleanLiteral) expressionNode()      {}
func (b *BooleanLiteral) TokenLiteral() string { return b.Token.Literal }

type NilLiteral struct {
	Token token.Token
}

func (n *NilLiteral) expressionNode()      {}
func (n *NilLiteral) TokenLiteral() string { return n.Token.Literal }

type CacahLiteral struct {
	Token token.Token
}

func (c *CacahLiteral) expressionNode()      {}
func (c *CacahLiteral) TokenLiteral() string { return c.Token.Literal }

type PrefixExpression struct {
	Token    token.Token
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }

type InfixExpression struct {
	Token    token.Token
	Left     Expression
	Operator string
	Right    Expression
}

func (oe *InfixExpression) expressionNode()      {}
func (oe *InfixExpression) TokenLiteral() string { return oe.Token.Literal }

type CallExpression struct {
	Token     token.Token // LPAREN
	Function  Expression
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }

type DotExpression struct {
	Token    token.Token // DOT
	Left     Expression
	Property *Identifier
}

func (de *DotExpression) expressionNode()      {}
func (de *DotExpression) TokenLiteral() string { return de.Token.Literal }

type IndexExpression struct {
	Token token.Token // LBRACKET
	Left  Expression
	Index Expression

	// Slicing support [mulai:selesai]
	IsSlice bool
	High    Expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }

type ArrayLiteral struct {
	Token    token.Token // LBRACKET
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }

type MapPair struct {
	Key   Expression
	Value Expression
}

type MapLiteral struct {
	Token token.Token // LBRACE
	Pairs []MapPair
}

func (ml *MapLiteral) expressionNode()      {}
func (ml *MapLiteral) TokenLiteral() string { return ml.Token.Literal }

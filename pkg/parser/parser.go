package parser

import (
	"fmt"
	"strconv"

	"github.com/zufarrizal/bahasa-ku/pkg/ast"
	"github.com/zufarrizal/bahasa-ku/pkg/lexer"
	"github.com/zufarrizal/bahasa-ku/pkg/token"
)

const (
	_ int = iota
	LOWEST
	COALESCE    // ??
	LOGIC       // ||, &&
	BIT_OR      // |
	BIT_XOR     // ^
	BIT_AND     // &
	EQUALS      // ==, !=
	LESSGREATER // >, <, >=, <=
	SHIFT       // <<, >>
	SUM         // +, -
	PRODUCT     // *, /, %
	PREFIX      // -X, !X
	CALL        // fn(X)
	INDEX       // arr[x], obj.prop
)

var precedences = map[token.TokenType]int{
	token.COALESCE:  COALESCE,
	token.OR:        LOGIC,
	token.AND:       LOGIC,
	token.BIT_OR:    BIT_OR,
	token.BIT_XOR:   BIT_XOR,
	token.BIT_AND:   BIT_AND,
	token.EQ:        EQUALS,
	token.NOT_EQ:    EQUALS,
	token.LT:        LESSGREATER,
	token.GT:        LESSGREATER,
	token.LTE:       LESSGREATER,
	token.GTE:       LESSGREATER,
	token.SHL:       SHIFT,
	token.SHR:       SHIFT,
	token.PLUS:      SUM,
	token.MINUS:     SUM,
	token.SLASH:     PRODUCT,
	token.ASTERISK:  PRODUCT,
	token.MOD:       PRODUCT,
	token.LPAREN:    CALL,
	token.DOT:       INDEX,
	token.LBRACKET:  INDEX,
}

type Parser struct {
	l         *lexer.Lexer
	curToken  token.Token
	peekToken token.Token
	errors    []string
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l, errors: []string{}}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.errors = append(p.errors, fmt.Sprintf("Baris %d:%d: Diharapkan token '%s', tetapi ditemukan '%s'", p.peekToken.Line, p.peekToken.Column, t, p.peekToken.Type))
	return false
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) ParseProgram() *ast.Program {
	prog := &ast.Program{Statements: []ast.Statement{}}
	for !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			prog.Statements = append(prog.Statements, stmt)
		}
		p.nextToken()
	}
	return prog
}

func (p *Parser) parseStatement() ast.Statement {
	if p.curTokenIs(token.SEMICOLON) {
		return nil
	}

	switch p.curToken.Type {
	case token.PAKET:
		return p.parsePaketStatement()
	case token.PAKAI:
		return p.parsePakaiStatement()
	case token.TIPE:
		return p.parseTypeStructStatement()
	case token.VAR, token.TETAP:
		return p.parseVarStatement()
	case token.FUNGSI:
		return p.parseFunctionStatement()
	case token.JIKA:
		return p.parseIfStatement()
	case token.UNTUK, token.SELAMA:
		return p.parseForStatement()
	case token.PILIH:
		return p.parseSwitchStatement()
	case token.KEMBALIKAN:
		return p.parseReturnStatement()
	case token.TUNDA:
		return p.parseDeferStatement()
	case token.KU:
		return p.parseKuStatement()
	case token.HENTI:
		stmt := &ast.BreakStatement{Token: p.curToken}
		if p.peekTokenIs(token.SEMICOLON) {
			p.nextToken()
		}
		return stmt
	case token.LANJUT:
		stmt := &ast.ContinueStatement{Token: p.curToken}
		if p.peekTokenIs(token.SEMICOLON) {
			p.nextToken()
		}
		return stmt
	default:
		return p.parseExpressionOrAssignStatement()
	}
}

func (p *Parser) parsePaketStatement() ast.Statement {
	stmt := &ast.PaketStatement{Token: p.curToken}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal
	return stmt
}

func (p *Parser) parsePakaiStatement() ast.Statement {
	stmt := &ast.PakaiStatement{Token: p.curToken}
	if !p.expectPeek(token.STRING) {
		return nil
	}
	stmt.Path = p.curToken.Literal
	return stmt
}

func (p *Parser) parseTypeStructStatement() ast.Statement {
	stmt := &ast.TypeStructStatement{Token: p.curToken}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.STRUKTUR) {
		return nil
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	for !p.peekTokenIs(token.RBRACE) && !p.peekTokenIs(token.EOF) {
		p.nextToken()
		if p.curTokenIs(token.IDENT) {
			fieldName := p.curToken.Literal
			fieldType := "apapun"
			if p.peekTokenIs(token.IDENT) {
				p.nextToken()
				fieldType = p.curToken.Literal
			}
			stmt.Fields = append(stmt.Fields, ast.StructField{Name: fieldName, Type: fieldType})
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}
	return stmt
}

func (p *Parser) parseVarStatement() ast.Statement {
	stmt := &ast.VarStatement{
		Token:    p.curToken,
		Names:    []*ast.Identifier{},
		Values:   []ast.Expression{},
		Constant: p.curToken.Type == token.TETAP,
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	firstIdent := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	stmt.Name = firstIdent
	stmt.Names = append(stmt.Names, firstIdent)

	for p.peekTokenIs(token.COMMA) {
		p.nextToken() // ke ','
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		stmt.Names = append(stmt.Names, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
	}

	// opsional tipe eksplisit jika ada
	if p.peekTokenIs(token.IDENT) {
		p.nextToken()
	}

	if p.peekTokenIs(token.ASSIGN) {
		p.nextToken() // ke '='
		p.nextToken() // ke ekspresi awal
		firstVal := p.parseExpression(LOWEST)
		stmt.Value = firstVal
		stmt.Values = append(stmt.Values, firstVal)

		for p.peekTokenIs(token.COMMA) {
			p.nextToken() // ke ','
			p.nextToken() // ke ekspresi berikutnya
			stmt.Values = append(stmt.Values, p.parseExpression(LOWEST))
		}
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseFunctionStatement() ast.Statement {
	stmt := &ast.FunctionStatement{Token: p.curToken}

	// Dukungan method receiver khas Go: fungsi (p Pengguna) sapa() { ... }
	if p.peekTokenIs(token.LPAREN) {
		p.nextToken() // ke '('
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		recName := p.curToken.Literal
		recType := ""
		if p.peekTokenIs(token.IDENT) {
			p.nextToken()
			recType = p.curToken.Literal
		}
		if !p.expectPeek(token.RPAREN) {
			return nil
		}
		stmt.Receiver = &ast.FunctionReceiver{Name: recName, Type: recType}
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	params, isVariadic := p.parseFunctionParameters()
	stmt.Parameters = params
	stmt.IsVariadic = isVariadic

	// opsional return type (misal fungsi tes() angka { ... })
	if p.peekTokenIs(token.IDENT) {
		p.nextToken()
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseFunctionParameters() ([]*ast.Identifier, bool) {
	params := []*ast.Identifier{}
	isVariadic := false

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return params, isVariadic
	}

	p.nextToken()
	// Cek apakah parameter pertama variadik (...args)
	if p.curTokenIs(token.ELLIPSIS) {
		isVariadic = true
		p.nextToken()
	}
	params = append(params, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})

	if p.peekTokenIs(token.IDENT) {
		p.nextToken()
	}

	for p.peekTokenIs(token.COMMA) {
		p.nextToken() // ke ','
		p.nextToken() // ke identifier atau '...'
		if p.curTokenIs(token.ELLIPSIS) {
			isVariadic = true
			p.nextToken()
		}
		params = append(params, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
		if p.peekTokenIs(token.IDENT) {
			p.nextToken()
		}
	}

	if !p.expectPeek(token.RPAREN) {
		return nil, false
	}
	return params, isVariadic
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{}}
	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}
	return block
}

func (p *Parser) parseIfStatement() ast.Statement {
	stmt := &ast.IfStatement{Token: p.curToken}
	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.Consequence = p.parseBlockStatement()

	if p.peekTokenIs(token.LAINNYA) {
		p.nextToken()
		if p.peekTokenIs(token.JIKA) {
			p.nextToken()
			stmt.Alternative = p.parseIfStatement()
		} else if p.expectPeek(token.LBRACE) {
			stmt.Alternative = p.parseBlockStatement()
		}
	}
	return stmt
}

func (p *Parser) parseForStatement() ast.Statement {
	stmt := &ast.ForStatement{Token: p.curToken}
	p.nextToken()

	if p.curTokenIs(token.LBRACE) {
		stmt.Body = p.parseBlockStatement()
		return stmt
	}

	// Cek apakah pola for ... dalam koleksi:
	// untuk item dalam list { ... } atau untuk i, item dalam list { ... }
	if p.curTokenIs(token.IDENT) && (p.peekTokenIs(token.COMMA) || p.peekTokenIs(token.DALAM) || p.peekTokenIs(token.RENTANG)) {
		firstIdent := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

		if p.peekTokenIs(token.COMMA) {
			p.nextToken() // ,
			p.nextToken() // second ident
			secondIdent := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

			if p.peekTokenIs(token.DALAM) || p.peekTokenIs(token.RENTANG) {
				p.nextToken() // dalam / rentang
				p.nextToken() // koleksi
				iter := p.parseExpression(LOWEST)
				if !p.expectPeek(token.LBRACE) {
					return nil
				}
				stmt.IsRange = true
				stmt.Key = firstIdent
				stmt.Value = secondIdent
				stmt.Iterable = iter
				stmt.Body = p.parseBlockStatement()
				return stmt
			}
		} else if p.peekTokenIs(token.DALAM) || p.peekTokenIs(token.RENTANG) {
			p.nextToken() // dalam / rentang
			p.nextToken() // koleksi
			iter := p.parseExpression(LOWEST)
			if !p.expectPeek(token.LBRACE) {
				return nil
			}
			stmt.IsRange = true
			stmt.Value = firstIdent
			stmt.Iterable = iter
			stmt.Body = p.parseBlockStatement()
			return stmt
		}
	}

	firstExpr := p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		stmt.Init = &ast.ExpressionStatement{Token: p.curToken, Expression: firstExpr}
		p.nextToken() // ;
		p.nextToken() // cond
		stmt.Cond = p.parseExpression(LOWEST)
		p.expectPeek(token.SEMICOLON)
		p.nextToken() // post
		stmt.Post = p.parseExpressionOrAssignStatement()
		if !p.expectPeek(token.LBRACE) {
			return nil
		}
	} else if p.peekTokenIs(token.DEFINE) || p.peekTokenIs(token.ASSIGN) {
		op := p.peekToken
		p.nextToken()
		p.nextToken()
		val := p.parseExpression(LOWEST)
		stmt.Init = &ast.AssignStatement{Token: op, Target: firstExpr, Value: val, IsDefine: op.Type == token.DEFINE}
		p.expectPeek(token.SEMICOLON)
		p.nextToken()
		stmt.Cond = p.parseExpression(LOWEST)
		p.expectPeek(token.SEMICOLON)
		p.nextToken()
		stmt.Post = p.parseExpressionOrAssignStatement()
		if !p.expectPeek(token.LBRACE) {
			return nil
		}
	} else {
		stmt.Cond = firstExpr
		if !p.expectPeek(token.LBRACE) {
			return nil
		}
	}

	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseSwitchStatement() ast.Statement {
	stmt := &ast.SwitchStatement{Token: p.curToken}
	if !p.peekTokenIs(token.LBRACE) {
		p.nextToken()
		stmt.Value = p.parseExpression(LOWEST)
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	for !p.peekTokenIs(token.RBRACE) && !p.peekTokenIs(token.EOF) {
		p.nextToken()
		if p.curTokenIs(token.KASUS) {
			p.nextToken()
			val := p.parseExpression(LOWEST)
			p.expectPeek(token.COLON)
			caseBody := &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{}}
			for !p.peekTokenIs(token.KASUS) && !p.peekTokenIs(token.BAWAAN) && !p.peekTokenIs(token.RBRACE) && !p.peekTokenIs(token.EOF) {
				p.nextToken()
				s := p.parseStatement()
				if s != nil {
					caseBody.Statements = append(caseBody.Statements, s)
				}
			}
			stmt.Cases = append(stmt.Cases, ast.SwitchCase{Value: val, Body: caseBody})
		} else if p.curTokenIs(token.BAWAAN) {
			p.expectPeek(token.COLON)
			defBody := &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{}}
			for !p.peekTokenIs(token.KASUS) && !p.peekTokenIs(token.BAWAAN) && !p.peekTokenIs(token.RBRACE) && !p.peekTokenIs(token.EOF) {
				p.nextToken()
				s := p.parseStatement()
				if s != nil {
					defBody.Statements = append(defBody.Statements, s)
				}
			}
			stmt.Default = defBody
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}
	return stmt
}

func (p *Parser) parseReturnStatement() ast.Statement {
	stmt := &ast.ReturnStatement{Token: p.curToken, ReturnValues: []ast.Expression{}}
	p.nextToken()
	if !p.curTokenIs(token.SEMICOLON) && !p.curTokenIs(token.RBRACE) {
		val := p.parseExpression(LOWEST)
		stmt.ReturnValue = val
		stmt.ReturnValues = append(stmt.ReturnValues, val)

		for p.peekTokenIs(token.COMMA) {
			p.nextToken() // ke ','
			p.nextToken() // ke ekspresi berikutnya
			nextVal := p.parseExpression(LOWEST)
			stmt.ReturnValues = append(stmt.ReturnValues, nextVal)
		}
	}
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseDeferStatement() ast.Statement {
	stmt := &ast.DeferStatement{Token: p.curToken}
	p.nextToken()
	stmt.Expression = p.parseExpression(LOWEST)
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseKuStatement() ast.Statement {
	stmt := &ast.KuStatement{Token: p.curToken}
	p.nextToken()
	stmt.Expression = p.parseExpression(LOWEST)
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseExpressionOrAssignStatement() ast.Statement {
	expr := p.parseExpression(LOWEST)

	// Cek apakah ada koma untuk multi-assignment (misal: a, b := ... atau a, b = ...)
	if p.peekTokenIs(token.COMMA) {
		targets := []ast.Expression{expr}
		for p.peekTokenIs(token.COMMA) {
			p.nextToken() // ke ','
			p.nextToken() // ke identifier berikutnya
			targets = append(targets, p.parseExpression(LOWEST))
		}

		if p.peekTokenIs(token.DEFINE) || p.peekTokenIs(token.ASSIGN) {
			op := p.peekToken
			p.nextToken() // ke ':=' atau '='
			p.nextToken() // ke ekspresi nilai pertama
			firstVal := p.parseExpression(LOWEST)
			values := []ast.Expression{firstVal}

			for p.peekTokenIs(token.COMMA) {
				p.nextToken() // ke ','
				p.nextToken() // ke ekspresi nilai berikutnya
				values = append(values, p.parseExpression(LOWEST))
			}

			if p.peekTokenIs(token.SEMICOLON) {
				p.nextToken()
			}

			return &ast.AssignStatement{
				Token:    op,
				Target:   targets[0],
				Targets:  targets,
				Value:    values[0],
				Values:   values,
				IsDefine: op.Type == token.DEFINE,
			}
		}
	}

	if p.peekTokenIs(token.DEFINE) || p.peekTokenIs(token.ASSIGN) {
		op := p.peekToken
		p.nextToken()
		p.nextToken()
		val := p.parseExpression(LOWEST)
		return &ast.AssignStatement{
			Token:    op,
			Target:   expr,
			Targets:  []ast.Expression{expr},
			Value:    val,
			Values:   []ast.Expression{val},
			IsDefine: op.Type == token.DEFINE,
		}
	}

	// Compound assignment operators: +=, -=, *=, /=, %=
	if p.peekTokenIs(token.PLUS_ASSIGN) || p.peekTokenIs(token.MINUS_ASSIGN) ||
		p.peekTokenIs(token.MUL_ASSIGN) || p.peekTokenIs(token.DIV_ASSIGN) || p.peekTokenIs(token.MOD_ASSIGN) {
		opTok := p.peekToken
		p.nextToken()
		p.nextToken()
		val := p.parseExpression(LOWEST)

		baseOp := "+"
		switch opTok.Type {
		case token.PLUS_ASSIGN:
			baseOp = "+"
		case token.MINUS_ASSIGN:
			baseOp = "-"
		case token.MUL_ASSIGN:
			baseOp = "*"
		case token.DIV_ASSIGN:
			baseOp = "/"
		case token.MOD_ASSIGN:
			baseOp = "%"
		}

		binaryExpr := &ast.InfixExpression{Token: opTok, Left: expr, Operator: baseOp, Right: val}
		return &ast.AssignStatement{Token: opTok, Target: expr, Value: binaryExpr, IsDefine: false}
	}

	if p.peekTokenIs(token.INC) {
		p.nextToken()
		tok := p.curToken
		one := &ast.IntegerLiteral{Token: tok, Value: 1}
		val := &ast.InfixExpression{Token: tok, Left: expr, Operator: "+", Right: one}
		return &ast.AssignStatement{Token: tok, Target: expr, Value: val, IsDefine: false}
	} else if p.peekTokenIs(token.DEC) {
		p.nextToken()
		tok := p.curToken
		one := &ast.IntegerLiteral{Token: tok, Value: 1}
		val := &ast.InfixExpression{Token: tok, Left: expr, Operator: "-", Right: one}
		return &ast.AssignStatement{Token: tok, Target: expr, Value: val, IsDefine: false}
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return &ast.ExpressionStatement{Token: p.curToken, Expression: expr}
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	left := p.parsePrefix()

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		switch p.peekToken.Type {
		case token.PLUS, token.MINUS, token.SLASH, token.ASTERISK, token.MOD,
			token.EQ, token.NOT_EQ, token.LT, token.GT, token.LTE, token.GTE,
			token.AND, token.OR, token.COALESCE, token.BIT_AND, token.BIT_OR, token.BIT_XOR, token.SHL, token.SHR:
			p.nextToken()
			left = p.parseInfix(left)
		case token.LPAREN:
			p.nextToken()
			left = p.parseCallExpression(left)
		case token.DOT:
			p.nextToken()
			left = p.parseDotExpression(left)
		case token.LBRACKET:
			p.nextToken()
			left = p.parseIndexExpression(left)
		default:
			return left
		}
	}
	return left
}

func (p *Parser) parsePrefix() ast.Expression {
	switch p.curToken.Type {
	case token.IDENT:
		return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	case token.INT:
		v, _ := strconv.ParseInt(p.curToken.Literal, 0, 64)
		return &ast.IntegerLiteral{Token: p.curToken, Value: v}
	case token.FLOAT:
		v, _ := strconv.ParseFloat(p.curToken.Literal, 64)
		return &ast.FloatLiteral{Token: p.curToken, Value: v}
	case token.STRING:
		return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
	case token.BENAR:
		return &ast.BooleanLiteral{Token: p.curToken, Value: true}
	case token.SALAH:
		return &ast.BooleanLiteral{Token: p.curToken, Value: false}
	case token.KOSONG:
		return &ast.NilLiteral{Token: p.curToken}
	case token.CACAH:
		return &ast.CacahLiteral{Token: p.curToken}
	case token.FUNGSI:
		return p.parseFunctionLiteral()
	case token.BANG, token.MINUS:
		tok := p.curToken
		p.nextToken()
		right := p.parseExpression(PREFIX)
		return &ast.PrefixExpression{Token: tok, Operator: tok.Literal, Right: right}
	case token.LPAREN:
		p.nextToken()
		expr := p.parseExpression(LOWEST)
		if !p.expectPeek(token.RPAREN) {
			return nil
		}
		return expr
	case token.LBRACKET:
		return p.parseArrayLiteral()
	case token.LBRACE:
		return p.parseMapLiteral()
	default:
		p.errors = append(p.errors, fmt.Sprintf("Baris %d:%d: Tidak dapat mem-parse awalan token '%s'", p.curToken.Line, p.curToken.Column, p.curToken.Literal))
		return nil
	}
}

func (p *Parser) parseInfix(left ast.Expression) ast.Expression {
	expr := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}
	prec := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(prec)
	return expr
}

func (p *Parser) parseCallExpression(fn ast.Expression) ast.Expression {
	call := &ast.CallExpression{Token: p.curToken, Function: fn, Arguments: []ast.Expression{}}
	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return call
	}

	p.nextToken()
	call.Arguments = append(call.Arguments, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		call.Arguments = append(call.Arguments, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	return call
}

func (p *Parser) parseDotExpression(left ast.Expression) ast.Expression {
	tok := p.curToken
	p.nextToken()
	propName := p.curToken.Literal
	return &ast.DotExpression{Token: tok, Left: left, Property: &ast.Identifier{Token: p.curToken, Value: propName}}
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	tok := p.curToken
	p.nextToken()

	// Kasus slice [:akhir]
	if p.curTokenIs(token.COLON) {
		var high ast.Expression = nil
		if !p.peekTokenIs(token.RBRACKET) {
			p.nextToken()
			high = p.parseExpression(LOWEST)
		}
		if !p.expectPeek(token.RBRACKET) {
			return nil
		}
		return &ast.IndexExpression{Token: tok, Left: left, Index: nil, IsSlice: true, High: high}
	}

	low := p.parseExpression(LOWEST)

	// Kasus slice [awal:akhir] atau [awal:]
	if p.peekTokenIs(token.COLON) {
		p.nextToken() // ke ':'
		var high ast.Expression = nil
		if !p.peekTokenIs(token.RBRACKET) {
			p.nextToken()
			high = p.parseExpression(LOWEST)
		}
		if !p.expectPeek(token.RBRACKET) {
			return nil
		}
		return &ast.IndexExpression{Token: tok, Left: left, Index: low, IsSlice: true, High: high}
	}

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	return &ast.IndexExpression{Token: tok, Left: left, Index: low, IsSlice: false}
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	arr := &ast.ArrayLiteral{Token: p.curToken, Elements: []ast.Expression{}}
	if p.peekTokenIs(token.RBRACKET) {
		p.nextToken()
		return arr
	}

	p.nextToken()
	arr.Elements = append(arr.Elements, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		if p.peekTokenIs(token.RBRACKET) {
			break
		}
		p.nextToken()
		arr.Elements = append(arr.Elements, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	return arr
}

func (p *Parser) parseMapLiteral() ast.Expression {
	m := &ast.MapLiteral{Token: p.curToken, Pairs: []ast.MapPair{}}
	if p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		return m
	}

	for !p.peekTokenIs(token.RBRACE) && !p.peekTokenIs(token.EOF) {
		p.nextToken()
		var key ast.Expression
		if p.curTokenIs(token.STRING) {
			key = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
		} else if p.curTokenIs(token.IDENT) {
			key = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
		} else {
			key = p.parseExpression(LOWEST)
		}

		if !p.expectPeek(token.COLON) {
			return nil
		}
		p.nextToken()
		val := p.parseExpression(LOWEST)
		m.Pairs = append(m.Pairs, ast.MapPair{Key: key, Value: val})

		if p.peekTokenIs(token.COMMA) {
			p.nextToken()
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}
	return m
}

func (p *Parser) parseFunctionLiteral() ast.Expression {
	lit := &ast.FunctionLiteral{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	params, isVariadic := p.parseFunctionParameters()
	lit.Parameters = params
	lit.IsVariadic = isVariadic

	// opsional return type
	if p.peekTokenIs(token.IDENT) {
		p.nextToken()
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	lit.Body = p.parseBlockStatement()
	return lit
}

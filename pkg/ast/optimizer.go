package ast

import (
	"fmt"
	"github.com/zufarrizal/bahasa-ku/pkg/token"
)

// OptimizeProgram melakukan constant folding pada AST Program
func OptimizeProgram(prog *Program) *Program {
	if prog == nil {
		return nil
	}
	for i, stmt := range prog.Statements {
		prog.Statements[i] = optimizeStatement(stmt)
	}
	return prog
}

func optimizeStatement(stmt Statement) Statement {
	if stmt == nil {
		return nil
	}
	switch s := stmt.(type) {
	case *ExpressionStatement:
		s.Expression = optimizeExpression(s.Expression)
		return s
	case *VarStatement:
		s.Value = optimizeExpression(s.Value)
		return s
	case *AssignStatement:
		s.Value = optimizeExpression(s.Value)
		return s
	case *ReturnStatement:
		s.ReturnValue = optimizeExpression(s.ReturnValue)
		return s
	case *BlockStatement:
		for i, st := range s.Statements {
			s.Statements[i] = optimizeStatement(st)
		}
		return s
	case *IfStatement:
		s.Condition = optimizeExpression(s.Condition)
		if s.Consequence != nil {
			s.Consequence = optimizeStatement(s.Consequence).(*BlockStatement)
		}
		if s.Alternative != nil {
			s.Alternative = optimizeStatement(s.Alternative)
		}
		return s
	case *ForStatement:
		s.Cond = optimizeExpression(s.Cond)
		if s.Body != nil {
			s.Body = optimizeStatement(s.Body).(*BlockStatement)
		}
		return s
	case *FunctionStatement:
		if s.Body != nil {
			s.Body = optimizeStatement(s.Body).(*BlockStatement)
		}
		return s
	default:
		return stmt
	}
}

func optimizeExpression(expr Expression) Expression {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *InfixExpression:
		e.Left = optimizeExpression(e.Left)
		e.Right = optimizeExpression(e.Right)

		// Lipat konstanta integer: a OP b
		if leftInt, ok := e.Left.(*IntegerLiteral); ok {
			if rightInt, ok := e.Right.(*IntegerLiteral); ok {
				valL := leftInt.Value
				valR := rightInt.Value
				switch e.Operator {
				case "+":
					return &IntegerLiteral{Token: token.Token{Type: token.INT, Literal: fmt.Sprintf("%d", valL+valR)}, Value: valL + valR}
				case "-":
					return &IntegerLiteral{Token: token.Token{Type: token.INT, Literal: fmt.Sprintf("%d", valL-valR)}, Value: valL - valR}
				case "*":
					return &IntegerLiteral{Token: token.Token{Type: token.INT, Literal: fmt.Sprintf("%d", valL*valR)}, Value: valL * valR}
				case "/":
					if valR != 0 {
						return &IntegerLiteral{Token: token.Token{Type: token.INT, Literal: fmt.Sprintf("%d", valL/valR)}, Value: valL / valR}
					}
				case "%":
					if valR != 0 {
						return &IntegerLiteral{Token: token.Token{Type: token.INT, Literal: fmt.Sprintf("%d", valL%valR)}, Value: valL % valR}
					}
				}
			}
		}

		// Lipat konkatenasi string statis: "a" + "b"
		if leftStr, ok := e.Left.(*StringLiteral); ok {
			if rightStr, ok := e.Right.(*StringLiteral); ok {
				if e.Operator == "+" {
					combined := leftStr.Value + rightStr.Value
					return &StringLiteral{Token: token.Token{Type: token.STRING, Literal: combined}, Value: combined}
				}
			}
		}

		return e

	case *PrefixExpression:
		e.Right = optimizeExpression(e.Right)
		if rightInt, ok := e.Right.(*IntegerLiteral); ok {
			if e.Operator == "-" {
				return &IntegerLiteral{Token: token.Token{Type: token.INT, Literal: fmt.Sprintf("%d", -rightInt.Value)}, Value: -rightInt.Value}
			}
		}
		if rightBool, ok := e.Right.(*BooleanLiteral); ok {
			if e.Operator == "!" {
				tokType := token.TokenType(token.SALAH)
				if !rightBool.Value {
					tokType = token.TokenType(token.BENAR)
				}
				return &BooleanLiteral{Token: token.Token{Type: tokType, Literal: fmt.Sprintf("%v", !rightBool.Value)}, Value: !rightBool.Value}
			}
		}
		return e

	default:
		return expr
	}
}

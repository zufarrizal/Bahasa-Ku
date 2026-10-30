package transpiler

import (
	"fmt"
	"strings"

	"github.com/zufarrizal/bahasa-ku/pkg/ast"
)

type Transpiler struct {
	indent int
}

func New() *Transpiler {
	return &Transpiler{indent: 0}
}

func (t *Transpiler) spasi() string {
	return strings.Repeat("  ", t.indent)
}

func (t *Transpiler) Transpile(node ast.Node) string {
	if node == nil {
		return ""
	}

	switch n := node.(type) {
	case *ast.Program:
		res := ""
		for _, stmt := range n.Statements {
			res += t.Transpile(stmt) + "\n"
		}
		return strings.TrimSpace(res) + "\n"

	case *ast.PaketStatement:
		name := n.Name
		if name == "utama" {
			name = "main"
		}
		return fmt.Sprintf("package %s\n", name)

	case *ast.PakaiStatement:
		path := n.Path
		if path == "cetak" {
			path = "fmt"
		} else if path == "web" || path == "jaringan" {
			path = "net/http"
		}
		return fmt.Sprintf("import %q\n", path)

	case *ast.TypeStructStatement:
		out := fmt.Sprintf("type %s struct {\n", n.Name.Value)
		t.indent++
		for _, f := range n.Fields {
			typ := f.Type
			if typ == "angka" {
				typ = "int"
			} else if typ == "teks" {
				typ = "string"
			}
			out += fmt.Sprintf("%s%s %s\n", t.spasi(), f.Name, typ)
		}
		t.indent--
		out += "}\n"
		return out

	case *ast.FunctionStatement:
		name := n.Name.Value
		if name == "utama" {
			name = "main"
			out := fmt.Sprintf("func main() %s\n", t.Transpile(n.Body))
			return out
		}
		params := []string{}
		for _, p := range n.Parameters {
			params = append(params, p.Value+" interface{}")
		}
		out := fmt.Sprintf("func %s(%s) interface{} %s\n", name, strings.Join(params, ", "), t.Transpile(n.Body))
		return out

	case *ast.BlockStatement:
		out := "{\n"
		t.indent++
		for _, stmt := range n.Statements {
			out += fmt.Sprintf("%s%s\n", t.spasi(), t.Transpile(stmt))
		}
		t.indent--
		out += t.spasi() + "}"
		return out

	case *ast.VarStatement:
		kata := "var"
		if n.Constant {
			kata = "const"
		}
		if len(n.Names) > 1 {
			names := []string{}
			for _, ident := range n.Names {
				names = append(names, ident.Value)
			}
			if len(n.Values) > 0 {
				vals := []string{}
				for _, valExpr := range n.Values {
					vals = append(vals, t.Transpile(valExpr))
				}
				return fmt.Sprintf("%s %s = %s", kata, strings.Join(names, ", "), strings.Join(vals, ", "))
			}
			return fmt.Sprintf("%s %s interface{}", kata, strings.Join(names, ", "))
		}
		if n.Value != nil {
			return fmt.Sprintf("%s %s = %s", kata, n.Name.Value, t.Transpile(n.Value))
		}
		return fmt.Sprintf("%s %s interface{}", kata, n.Name.Value)

	case *ast.AssignStatement:
		op := "="
		if n.IsDefine {
			op = ":="
		}
		if len(n.Targets) > 1 {
			tgts := []string{}
			for _, tg := range n.Targets {
				tgts = append(tgts, t.Transpile(tg))
			}
			vals := []string{}
			for _, v := range n.Values {
				vals = append(vals, t.Transpile(v))
			}
			return fmt.Sprintf("%s %s %s", strings.Join(tgts, ", "), op, strings.Join(vals, ", "))
		}
		return fmt.Sprintf("%s %s %s", t.Transpile(n.Target), op, t.Transpile(n.Value))

	case *ast.IfStatement:
		out := fmt.Sprintf("if %s %s", t.Transpile(n.Condition), t.Transpile(n.Consequence))
		if n.Alternative != nil {
			out += fmt.Sprintf(" else %s", t.Transpile(n.Alternative))
		}
		return out

	case *ast.ForStatement:
		if n.IsRange {
			key := "_"
			if n.Key != nil {
				key = n.Key.Value
			}
			val := "_"
			if n.Value != nil {
				val = n.Value.Value
			}
			return fmt.Sprintf("for %s, %s := range %s %s", key, val, t.Transpile(n.Iterable), t.Transpile(n.Body))
		}
		if n.Init == nil && n.Cond == nil && n.Post == nil {
			return fmt.Sprintf("for %s", t.Transpile(n.Body))
		}
		if n.Init == nil && n.Post == nil {
			return fmt.Sprintf("for %s %s", t.Transpile(n.Cond), t.Transpile(n.Body))
		}
		return fmt.Sprintf("for %s; %s; %s %s", t.Transpile(n.Init), t.Transpile(n.Cond), t.Transpile(n.Post), t.Transpile(n.Body))

	case *ast.SwitchStatement:
		valStr := ""
		if n.Value != nil {
			valStr = " " + t.Transpile(n.Value)
		}
		out := fmt.Sprintf("switch%s {\n", valStr)
		t.indent++
		for _, c := range n.Cases {
			out += fmt.Sprintf("%scase %s:\n", t.spasi(), t.Transpile(c.Value))
			t.indent++
			for _, stmt := range c.Body.Statements {
				out += fmt.Sprintf("%s%s\n", t.spasi(), t.Transpile(stmt))
			}
			t.indent--
		}
		if n.Default != nil {
			out += fmt.Sprintf("%sdefault:\n", t.spasi())
			t.indent++
			for _, stmt := range n.Default.Statements {
				out += fmt.Sprintf("%s%s\n", t.spasi(), t.Transpile(stmt))
			}
			t.indent--
		}
		t.indent--
		out += t.spasi() + "}"
		return out

	case *ast.ReturnStatement:
		if len(n.ReturnValues) > 1 {
			vals := []string{}
			for _, v := range n.ReturnValues {
				vals = append(vals, t.Transpile(v))
			}
			return fmt.Sprintf("return %s", strings.Join(vals, ", "))
		}
		if n.ReturnValue != nil {
			return fmt.Sprintf("return %s", t.Transpile(n.ReturnValue))
		}
		return "return"

	case *ast.BreakStatement:
		return "break"

	case *ast.ContinueStatement:
		return "continue"

	case *ast.DeferStatement:
		return fmt.Sprintf("defer %s", t.Transpile(n.Expression))

	case *ast.ExpressionStatement:
		return t.Transpile(n.Expression)

	case *ast.CallExpression:
		fn := t.Transpile(n.Function)
		if fn == "cetak.tulis" || fn == "cetak.tulisBaris" {
			fn = "fmt.Println"
		} else if fn == "jaringan.ambil" {
			fn = "http.Get"
		} else if fn == "panjang" {
			fn = "len"
		} else if fn == "tambah" {
			fn = "append"
		} else if fn == "hapus" {
			fn = "delete"
		}
		args := []string{}
		for _, a := range n.Arguments {
			args = append(args, t.Transpile(a))
		}
		return fmt.Sprintf("%s(%s)", fn, strings.Join(args, ", "))

	case *ast.InfixExpression:
		return fmt.Sprintf("%s %s %s", t.Transpile(n.Left), n.Operator, t.Transpile(n.Right))

	case *ast.PrefixExpression:
		return fmt.Sprintf("%s%s", n.Operator, t.Transpile(n.Right))

	case *ast.DotExpression:
		return fmt.Sprintf("%s.%s", t.Transpile(n.Left), n.Property.Value)

	case *ast.IndexExpression:
		if n.IsSlice {
			low := ""
			if n.Index != nil {
				low = t.Transpile(n.Index)
			}
			high := ""
			if n.High != nil {
				high = t.Transpile(n.High)
			}
			return fmt.Sprintf("%s[%s:%s]", t.Transpile(n.Left), low, high)
		}
		return fmt.Sprintf("%s[%s]", t.Transpile(n.Left), t.Transpile(n.Index))

	case *ast.Identifier:
		return n.Value

	case *ast.IntegerLiteral:
		return fmt.Sprintf("%d", n.Value)

	case *ast.FloatLiteral:
		return fmt.Sprintf("%f", n.Value)

	case *ast.StringLiteral:
		return fmt.Sprintf("%q", n.Value)

	case *ast.BooleanLiteral:
		if n.Value {
			return "true"
		}
		return "false"

	case *ast.NilLiteral:
		return "nil"

	case *ast.ArrayLiteral:
		elems := []string{}
		for _, e := range n.Elements {
			elems = append(elems, t.Transpile(e))
		}
		return fmt.Sprintf("[]interface{}{%s}", strings.Join(elems, ", "))

	case *ast.MapLiteral:
		pairs := []string{}
		for _, p := range n.Pairs {
			pairs = append(pairs, fmt.Sprintf("%s: %s", t.Transpile(p.Key), t.Transpile(p.Value)))
		}
		return fmt.Sprintf("map[string]interface{}{%s}", strings.Join(pairs, ", "))
	}

	return ""
}

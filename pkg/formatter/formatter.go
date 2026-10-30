package formatter

import (
	"fmt"
	"strings"

	"github.com/zufarrizal/bahasa-ku/pkg/ast"
)

type Formatter struct {
	indent int
}

func New() *Formatter {
	return &Formatter{indent: 0}
}

func (f *Formatter) spasi() string {
	return strings.Repeat("    ", f.indent)
}

func (f *Formatter) Format(node ast.Node) string {
	if node == nil {
		return ""
	}

	switch n := node.(type) {
	case *ast.Program:
		lines := []string{}
		for _, stmt := range n.Statements {
			res := f.Format(stmt)
			if res != "" {
				lines = append(lines, res)
			}
		}
		return strings.Join(lines, "\n") + "\n"

	case *ast.PaketStatement:
		return fmt.Sprintf("paket %s\n", n.Name)

	case *ast.PakaiStatement:
		return fmt.Sprintf("pakai %q", n.Path)

	case *ast.TypeStructStatement:
		out := fmt.Sprintf("tipe %s struktur {\n", n.Name.Value)
		f.indent++
		for _, field := range n.Fields {
			out += fmt.Sprintf("%s%s %s\n", f.spasi(), field.Name, field.Type)
		}
		f.indent--
		out += f.spasi() + "}\n"
		return out

	case *ast.FunctionStatement:
		params := []string{}
		for i, p := range n.Parameters {
			if n.IsVariadic && i == len(n.Parameters)-1 {
				params = append(params, "..."+p.Value)
			} else {
				params = append(params, p.Value)
			}
		}
		recStr := ""
		if n.Receiver != nil {
			recStr = fmt.Sprintf("(%s %s) ", n.Receiver.Name, n.Receiver.Type)
		}
		out := fmt.Sprintf("\nfungsi %s%s(%s) ", recStr, n.Name.Value, strings.Join(params, ", "))
		out += f.Format(n.Body)
		return out

	case *ast.FunctionLiteral:
		params := []string{}
		for i, p := range n.Parameters {
			if n.IsVariadic && i == len(n.Parameters)-1 {
				params = append(params, "..."+p.Value)
			} else {
				params = append(params, p.Value)
			}
		}
		out := fmt.Sprintf("fungsi(%s) ", strings.Join(params, ", "))
		out += f.Format(n.Body)
		return out

	case *ast.BlockStatement:
		out := "{\n"
		f.indent++
		for _, stmt := range n.Statements {
			out += fmt.Sprintf("%s%s\n", f.spasi(), f.Format(stmt))
		}
		f.indent--
		out += f.spasi() + "}"
		return out

	case *ast.VarStatement:
		kata := "var"
		if n.Constant {
			kata = "tetap"
		}
		if len(n.Names) > 1 {
			names := []string{}
			for _, ident := range n.Names {
				names = append(names, ident.Value)
			}
			if len(n.Values) > 0 {
				vals := []string{}
				for _, valExpr := range n.Values {
					vals = append(vals, f.Format(valExpr))
				}
				return fmt.Sprintf("%s %s = %s", kata, strings.Join(names, ", "), strings.Join(vals, ", "))
			}
			return fmt.Sprintf("%s %s", kata, strings.Join(names, ", "))
		}
		if n.Value != nil {
			return fmt.Sprintf("%s %s = %s", kata, n.Name.Value, f.Format(n.Value))
		}
		return fmt.Sprintf("%s %s", kata, n.Name.Value)

	case *ast.AssignStatement:
		op := "="
		if n.IsDefine {
			op = ":="
		}
		if len(n.Targets) > 1 {
			tgts := []string{}
			for _, t := range n.Targets {
				tgts = append(tgts, f.Format(t))
			}
			vals := []string{}
			for _, v := range n.Values {
				vals = append(vals, f.Format(v))
			}
			return fmt.Sprintf("%s %s %s", strings.Join(tgts, ", "), op, strings.Join(vals, ", "))
		}
		return fmt.Sprintf("%s %s %s", f.Format(n.Target), op, f.Format(n.Value))

	case *ast.IfStatement:
		out := fmt.Sprintf("jika %s %s", f.Format(n.Condition), f.Format(n.Consequence))
		if n.Alternative != nil {
			out += fmt.Sprintf(" lainnya %s", f.Format(n.Alternative))
		}
		return out

	case *ast.ForStatement:
		if n.IsRange {
			if n.Key != nil && n.Value != nil {
				return fmt.Sprintf("untuk %s, %s dalam %s %s", n.Key.Value, n.Value.Value, f.Format(n.Iterable), f.Format(n.Body))
			} else if n.Value != nil {
				return fmt.Sprintf("untuk %s dalam %s %s", n.Value.Value, f.Format(n.Iterable), f.Format(n.Body))
			}
		}
		if n.Init == nil && n.Cond == nil && n.Post == nil {
			return fmt.Sprintf("untuk %s", f.Format(n.Body))
		}
		if n.Init == nil && n.Post == nil {
			return fmt.Sprintf("untuk %s %s", f.Format(n.Cond), f.Format(n.Body))
		}
		return fmt.Sprintf("untuk %s; %s; %s %s", f.Format(n.Init), f.Format(n.Cond), f.Format(n.Post), f.Format(n.Body))

	case *ast.SwitchStatement:
		valStr := ""
		if n.Value != nil {
			valStr = " " + f.Format(n.Value)
		}
		out := fmt.Sprintf("pilih%s {\n", valStr)
		f.indent++
		for _, c := range n.Cases {
			out += fmt.Sprintf("%skasus %s:\n", f.spasi(), f.Format(c.Value))
			f.indent++
			for _, s := range c.Body.Statements {
				out += fmt.Sprintf("%s%s\n", f.spasi(), f.Format(s))
			}
			f.indent--
		}
		if n.Default != nil {
			out += fmt.Sprintf("%sbawaan:\n", f.spasi())
			f.indent++
			for _, s := range n.Default.Statements {
				out += fmt.Sprintf("%s%s\n", f.spasi(), f.Format(s))
			}
			f.indent--
		}
		f.indent--
		out += f.spasi() + "}"
		return out

	case *ast.ReturnStatement:
		if len(n.ReturnValues) > 1 {
			vals := []string{}
			for _, v := range n.ReturnValues {
				vals = append(vals, f.Format(v))
			}
			return fmt.Sprintf("kembalikan %s", strings.Join(vals, ", "))
		}
		if n.ReturnValue != nil {
			return fmt.Sprintf("kembalikan %s", f.Format(n.ReturnValue))
		}
		return "kembalikan"

	case *ast.BreakStatement:
		return "henti"

	case *ast.ContinueStatement:
		return "lanjut"

	case *ast.TryCatchStatement:
		res := "coba " + f.Format(n.TryBlock)
		if n.CatchBlock != nil {
			if n.CatchIdent != nil {
				res += fmt.Sprintf(" tangkap (%s) ", n.CatchIdent.Value) + f.Format(n.CatchBlock)
			} else {
				res += " tangkap " + f.Format(n.CatchBlock)
			}
		}
		if n.FinallyBlock != nil {
			res += " akhirnya " + f.Format(n.FinallyBlock)
		}
		return res

	case *ast.DeferStatement:
		return fmt.Sprintf("tunda %s", f.Format(n.Expression))

	case *ast.KuStatement:
		return fmt.Sprintf("ku %s", f.Format(n.Expression))

	case *ast.ExpressionStatement:
		return f.Format(n.Expression)

	case *ast.CallExpression:
		fn := f.Format(n.Function)
		args := []string{}
		for _, a := range n.Arguments {
			args = append(args, f.Format(a))
		}
		return fmt.Sprintf("%s(%s)", fn, strings.Join(args, ", "))

	case *ast.InfixExpression:
		return fmt.Sprintf("%s %s %s", f.Format(n.Left), n.Operator, f.Format(n.Right))

	case *ast.PrefixExpression:
		return fmt.Sprintf("%s%s", n.Operator, f.Format(n.Right))

	case *ast.DotExpression:
		return fmt.Sprintf("%s.%s", f.Format(n.Left), n.Property.Value)

	case *ast.IndexExpression:
		if n.IsSlice {
			low := ""
			if n.Index != nil {
				low = f.Format(n.Index)
			}
			high := ""
			if n.High != nil {
				high = f.Format(n.High)
			}
			return fmt.Sprintf("%s[%s:%s]", f.Format(n.Left), low, high)
		}
		return fmt.Sprintf("%s[%s]", f.Format(n.Left), f.Format(n.Index))

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
			return "benar"
		}
		return "salah"

	case *ast.NilLiteral:
		return "kosong"

	case *ast.CacahLiteral:
		return "cacah"

	case *ast.ArrayLiteral:
		elems := []string{}
		for _, e := range n.Elements {
			elems = append(elems, f.Format(e))
		}
		return fmt.Sprintf("[%s]", strings.Join(elems, ", "))

	case *ast.MapLiteral:
		pairs := []string{}
		for _, p := range n.Pairs {
			pairs = append(pairs, fmt.Sprintf("%s: %s", f.Format(p.Key), f.Format(p.Value)))
		}
		return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
	}

	return ""
}

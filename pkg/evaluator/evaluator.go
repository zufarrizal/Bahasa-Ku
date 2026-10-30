package evaluator

import (
	"fmt"
	"strings"

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
)

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

type ReturnValue struct{ Value Object }

func (r *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (r *ReturnValue) Inspect() string  { return r.Value.Inspect() }

type Environment struct {
	store map[string]Object
	outer *Environment
	deferStack []func()
}

func NewEnvironment() *Environment {
	return &Environment{store: make(map[string]Object), deferStack: []func(){}}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
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

type BuiltinFunction func(args ...Object) Object

type Builtin struct{ Fn BuiltinFunction }

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "fungsi bawaan" }

type Function struct {
	Parameters []*ast.Identifier
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
	Name   string
	Fields []ast.StructField
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

	return env
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
			if ret, ok := result.(*ReturnValue); ok {
				return ret.Value
			}
		}
		if mainFn, ok := env.Get("utama"); ok {
			if fn, ok := mainFn.(*Function); ok {
				return applyFunction(fn, []Object{}, env)
			}
		}
		return result

	case *ast.PaketStatement, *ast.PakaiStatement:
		return NIL

	case *ast.TypeStructStatement:
		def := &StructDef{Name: n.Name.Value, Fields: n.Fields}
		env.Set(n.Name.Value, def)
		return def

	case *ast.VarStatement:
		var val Object = NIL
		if n.Value != nil {
			val = Eval(n.Value, env)
		}
		env.Set(n.Name.Value, val)
		return val

	case *ast.AssignStatement:
		val := Eval(n.Value, env)
		if ident, ok := n.Target.(*ast.Identifier); ok {
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
		fn := &Function{Parameters: n.Parameters, Body: n.Body, Env: env}
		env.Set(n.Name.Value, fn)
		return fn

	case *ast.BlockStatement:
		var result Object = NIL
		for _, stmt := range n.Statements {
			result = Eval(stmt, env)
			if result != nil && result.Type() == RETURN_VALUE_OBJ {
				return result
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
			if result != nil && result.Type() == RETURN_VALUE_OBJ {
				return result
			}
			if n.Post != nil {
				Eval(n.Post, loopEnv)
			}
		}
		return result

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
		var val Object = NIL
		if n.ReturnValue != nil {
			val = Eval(n.ReturnValue, env)
		}
		return &ReturnValue{Value: val}

	case *ast.DeferStatement:
		deferredCall := func() {
			Eval(n.Expression, env)
		}
		env.deferStack = append(env.deferStack, deferredCall)
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
		for i, param := range f.Parameters {
			if i < len(args) {
				fnEnv.Set(param.Value, args[i])
			}
		}

		res := Eval(f.Body, fnEnv)

		// Eksekusi tunda (defer) LIFO
		for i := len(fnEnv.deferStack) - 1; i >= 0; i-- {
			fnEnv.deferStack[i]()
		}

		if ret, ok := res.(*ReturnValue); ok {
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
	case *String:
		return av.Value == b.(*String).Value
	case *Boolean:
		return av.Value == b.(*Boolean).Value
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

	return NIL
}

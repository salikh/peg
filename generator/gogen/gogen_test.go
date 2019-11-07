// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gogen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/salikh/peg/parser/charclass"
)

func TestPackage(t *testing.T) {
	tests := []struct {
		want string
		f    *ast.File
	}{
		{
			want: `package mypackage

func my() string {
}
`,
			f: Package("mypackage", nil,
				Func("my", FuncType(nil, Fields(Field(nil, Ident("string")))))),
		},
		{
			want: `package mypackage

import "fmt"

func my() string {
	fmt.Println("hello")
}
`,
			f: Package("mypackage", []string{"fmt"},
				Func("my", FuncType(nil, Fields(Field(nil, Ident("string")))),
					ExprStmt(Call(Sel(Ident("fmt"), "Println"), String(`"hello"`))),
				)),
		},
		{
			want: `package mypackage

var x = "hello"
`,
			f: Package("mypackage", nil,
				Var("x", nil, String(`"hello"`))),
		},
		{
			want: `package mypackage

var x string = "hello"
`,
			f: Package("mypackage", nil,
				Var("x", Ident("string"), String(`"hello"`))),
		},
		{
			want: `package mypackage

var x = 17
`,
			f: Package("mypackage", nil,
				Var("x", nil, Int("17"))),
		},
		{
			want: `package mypackage

var x = []string{}
`,
			f: Package("mypackage", nil,
				Var("x", nil, Composite(SliceType(Ident("string")), nil))),
		},
		{
			want: `package mypackage

type X struct {
}
`,
			f: Package("mypackage", nil,
				Type("X", Struct())),
		},
		{
			want: `package mypackage

type X struct {
	S string
	i int
}
`,
			f: Package("mypackage", nil,
				Type("X", Struct(AField("S", Ident("string")), AField("i", Ident("int"))))),
		},
		{
			`package mypackage

func LiteralHandler(r *Result, pos int) (int, error) {
	const literal = "abc"
	if len(r.Source)-pos < len(literal) {
		return 0, fmt.Errorf("expecting %q, got %q", literal, r.Source[pos:])
	}
	next := r.Source[pos:]
	if next != literal {
		return 0, fmt.Errorf("expecting %q, got %q", literal, next)
	}
	return len(literal), nil
}
`,
			Package("mypackage", []string{},
				Func("LiteralHandler", FuncType(Fields(AField("r", Star(Ident("Result"))),
					AField("pos", Ident("int"))), Fields(Field(nil, Ident("int")), Field(nil, Ident("error")))),
					DeclStmt(Const("literal", nil, String(`"abc"`))),
					If(nil, Binary(
						Binary(Call(Ident("len"), Sel(Ident("r"), "Source")), token.SUB, Ident("pos")),
						token.LSS, Call(Ident("len"), Ident("literal"))),
						Return(Int("0"), Call(Sel(Ident("fmt"), "Errorf"), String(`"expecting %q, got %q"`),
							Ident("literal"), Slice(Sel(Ident("r"), "Source"), Ident("pos"), nil)))),
					Assign(Ident("next"), Slice(Sel(Ident("r"), "Source"), Ident("pos"), nil)),
					If(nil, Binary(Ident("next"), token.NEQ, Ident("literal")),
						Return(Int("0"), Call(Sel(Ident("fmt"), "Errorf"), String(`"expecting %q, got %q"`),
							Ident("literal"), Ident("next")))),
					Return(Call(Ident("len"), Ident("literal")), Ident("nil")))),
		},
		{
			`package mypackage

func LiteralHandler(r *Result, pos int) (int, error) {
	const literal = "abc"
	if len(r.Source)-pos < len(literal) {
		return 0, fmt.Errorf("expecting %q, got %q", literal, r.Source[pos:])
	}
	next := r.Source[pos : pos+len(literal)]
	if next != literal {
		return 0, fmt.Errorf("expecting %q, got %q", literal, next)
	}
	return len(literal), nil
}
`,
			Package("mypackage", []string{}, LiteralHandler("LiteralHandler", "abc")),
		},
		{
			`package mypackage

func CharClassAlnumHandler(r *Result, pos int) (int, error) {
	c, w := utf8.DecodeRuneInString(r.Source[pos:])
	if w == 0 {
		return 0, fmt.Errorf("expecting char, got EOF")
	}
	if c == utf8.RuneError {
		return w, fmt.Errorf("invalid utf8: %q", r.Source[pos:pos+w])
	}
	if !(unicode.IsLetter(c) || unicode.IsNumber(c)) {
		return 0, fmt.Errorf("character %q does not match class [[:alnum:]]", c)
	}
	return w, nil
}
`,
			Package("mypackage", []string{}, CharClassHandler("CharClassAlnumHandler", &charclass.CharClass{Special: "[:alnum:]"})),
		},
		{
			`package mypackage

func CharClassAlnumHandler(r *Result, pos int) (int, error) {
	c, w := utf8.DecodeRuneInString(r.Source[pos:])
	if w == 0 {
		return 0, fmt.Errorf("expecting char, got EOF")
	}
	if c == utf8.RuneError {
		return w, fmt.Errorf("invalid utf8: %q", r.Source[pos:pos+w])
	}
	if !unicode.IsLetter(c) && !unicode.IsNumber(c) {
		return 0, fmt.Errorf("character %q does not match class [[:alnum:]]", c)
	}
	return w, nil
}
`,
			Package("mypackage", []string{}, Func("CharClassAlnumHandler", FuncType(Fields(AField("r", Star(Ident("Result"))),
				AField("pos", Ident("int"))), Fields(Field(nil, Ident("int")), Field(nil, Ident("error")))),
				AssignMulti(E(Ident("c"), Ident("w")), E(Call(Sel(Ident("utf8"), "DecodeRuneInString"),
					Slice(Sel(Ident("r"), "Source"), Ident("pos"), nil)))),
				If(nil, Binary(Ident("w"), token.EQL, Int("0")),
					Return(Int("0"), Call(Sel(Ident("fmt"), "Errorf"), String(`"expecting char, got EOF"`)))),
				If(nil, Binary(Ident("c"), token.EQL, Sel(Ident("utf8"), "RuneError")),
					Return(Ident("w"), Call(Sel(Ident("fmt"), "Errorf"), String(`"invalid utf8: %q"`),
						Slice(Sel(Ident("r"), "Source"), Ident("pos"), Binary(Ident("pos"), token.ADD, Ident("w")))))),
				If(nil, Binary(Unary(token.NOT, Call(Sel(Ident("unicode"), "IsLetter"), Ident("c"))), token.LAND,
					Unary(token.NOT, Call(Sel(Ident("unicode"), "IsNumber"), Ident("c")))),
					Return(Int("0"), Call(Sel(Ident("fmt"), "Errorf"), String(`"character %q does not match class [[:alnum:]]"`), Ident("c")))),
				Return(Ident("w"), Ident("nil")),
			)),
		},
		{
			`package mypackage

func StarHandler1(r *Result, pos int) (int, error) {
	ww := 0
	for w, err := XHandler(r, pos); err == nil && w > 0; w, err = XHandler(r, pos+ww) {
		ww += w
	}
	return ww, nil
}
`,
			Package("mypackage", []string{}, StarHandler("StarHandler1", "XHandler")),
		},
		{
			`package mypackage

func StarHandler(r *Result, pos int) (int, error) {
	ww := 0
	for w, err := XHandler(r, pos); err == nil && w > 0; w, err = XHandler(r, pos+ww) {
		ww += w
	}
	return ww, nil
}
`,
			Package("mypackage", []string{}, Func("StarHandler", FuncType(Fields(AField("r", Star(Ident("Result"))),
				AField("pos", Ident("int"))), Fields(Field(nil, Ident("int")), Field(nil, Ident("error")))),
				Assign(Ident("ww"), Int("0")),
				For(AssignMulti(E(Ident("w"), Ident("err")), E(Call(Ident("XHandler"),
					Ident("r"), Ident("pos")))),
					Binary(Binary(Ident("err"), token.EQL, Ident("nil")),
						token.LAND, Binary(Ident("w"), token.GTR, Int("0"))),
					AssignMulti(E(Ident("w"), Ident("err")),
						E(Call(Ident("XHandler"), Ident("r"), Binary(Ident("pos"), token.ADD, Ident("ww")))), token.ASSIGN),
					Assign(Ident("ww"), Ident("w"), token.ADD_ASSIGN),
				),
				Return(Ident("ww"), Ident("nil")),
			)),
		},
		{
			`package mypackage

func GroupHandler2(r *Result, pos int) (int, error) {
	ww := 0
	var w int
	var err error
	w, err = YHandler2(r, pos+ww)
	ww += w
	if err != nil {
		return ww, err
	}
	return ww, nil
}
`,
			Package("mypackage", []string{}, GroupHandler("GroupHandler2", []string{"YHandler2"})),
		},
		{
			`package mypackage

func GroupHandler(r *Result, pos int) (int, error) {
	ww := 0
	var w int
	var err error
	w, err = YHandler(r, pos+ww)
	ww += w
	if err != nil {
		return ww, err
	}
	return ww, nil
}
`,
			Package("mypackage", []string{}, Func("GroupHandler", FuncType(Fields(AField("r", Star(Ident("Result"))),
				AField("pos", Ident("int"))), Fields(Field(nil, Ident("int")), Field(nil, Ident("error")))),
				Assign(Ident("ww"), Int("0")),
				DeclStmt(Var("w", Ident("int"), nil)),
				DeclStmt(Var("err", Ident("error"), nil)),
				AssignMulti(E(Ident("w"), Ident("err")), E(
					Call(Ident("YHandler"),
						Ident("r"), Binary(Ident("pos"), token.ADD, Ident("ww")))), token.ASSIGN),
				Assign(Ident("ww"), Ident("w"), token.ADD_ASSIGN),
				If(nil, Binary(Ident("err"), token.NEQ, Ident("nil")),
					Return(Ident("ww"), Ident("err"))),
				Return(Ident("ww"), Ident("nil")),
			)),
		},
		{
			`package mypackage

func PredicateHandler(r *Result, pos int) (int, error) {
	const negative = true
	_, err := ZHandler(r, pos)
	if negative == (err != nil) {
		return 0, nil
	}
	if err == nil {
		return 0, fmt.Errorf("negative predicate matched")
	}
	return 0, err
}
`,
			Package("mypackage", []string{}, PredicateHandler("PredicateHandler", "ZHandler", true)),
		},
		{
			`package mypackage

func PredicateHandler(r *Result, pos int) (int, error) {
	const negative = false
	_, err := ZHandler(r, pos)
	if negative == (err != nil) {
		return 0, nil
	}
	if err == nil {
		return 0, fmt.Errorf("negative predicate matched")
	}
	return 0, err
}
`,
			Package("mypackage", []string{}, Func("PredicateHandler", FuncType(Fields(AField("r", Star(Ident("Result"))),
				AField("pos", Ident("int"))), Fields(Field(nil, Ident("int")), Field(nil, Ident("error")))),
				DeclStmt(Const("negative", nil, Ident("false"))),
				AssignMulti(E(Ident("_"), Ident("err")), E(
					Call(Ident("ZHandler"),
						Ident("r"), Ident("pos")))),
				If(nil, Binary(Ident("negative"), token.EQL, Binary(Ident("err"), token.NEQ, Ident("nil"))),
					Return(Int("0"), Ident("nil"))),
				If(nil, Binary(Ident("err"), token.EQL, Ident("nil")),
					Return(Int("0"), Call(Sel(Ident("fmt"), "Errorf"), String(`"negative predicate matched"`)))),
				Return(Int("0"), Ident("err")),
			)),
		},
		{
			`package mypackage

func ChoiceHandler0(r *Result, pos int) (int, error) {
	w, err := Handler1(r, pos)
	if err != nil {
		w, err = Handler2(r, pos)
	}
	if err != nil {
		w, err = Handler3(r, pos)
	}
	return w, err
}
`,
			Package("mypackage", []string{}, ChoiceHandler("ChoiceHandler0", "Handler1", "Handler2", "Handler3")),
		},
		{
			`package mypackage

func ChoiceHandler(r *Result, pos int) (int, error) {
	w, err := AHandler(r, pos)
	if err != nil {
		w, err = BHandler(r, pos)
	}
	return w, err
}
`,
			Package("mypackage", []string{}, Func("ChoiceHandler", FuncType(Fields(AField("r", Star(Ident("Result"))),
				AField("pos", Ident("int"))), Fields(Field(nil, Ident("int")), Field(nil, Ident("error")))),
				AssignMulti(E(Ident("w"), Ident("err")), E(
					Call(Ident("AHandler"), Ident("r"), Ident("pos")))),
				If(nil, Binary(Ident("err"), token.NEQ, Ident("nil")),
					AssignMulti(E(Ident("w"), Ident("err")), E(
						Call(Ident("BHandler"), Ident("r"), Ident("pos"))), token.ASSIGN),
				),
				Return(Ident("w"), Ident("err")),
			)),
		},
		{
			`package mypackage

func DotHandler1(r *Result, pos int) (int, error) {
	if pos == len(r.Source) {
		return 0, fmt.Errorf("expected character, got EOF")
	}
	c, w := utf8.DecodeRuneInString(r.Source[pos:])
	if c == utf8.RuneError {
		return w, fmt.Errorf("invalid utf8: %q", r.Source[pos:pos+w])
	}
	return w, nil
}
`,
			Package("mypackage", []string{}, DotHandler("DotHandler1")),
		},
		{
			`package mypackage

func DotHandler(r *Result, pos int) (int, error) {
	if pos == len(r.Source) {
		return 0, fmt.Errorf("expected character, got EOF")
	}
	c, w := utf8.DecodeRuneInString(r.Source[pos:])
	if c == utf8.RuneError {
		return w, fmt.Errorf("invalid utf8: %q", r.Source[pos:pos+w])
	}
	return w, nil
}
`,
			Package("mypackage", []string{}, Func("DotHandler", FuncType(Fields(AField("r", Star(Ident("Result"))),
				AField("pos", Ident("int"))), Fields(Field(nil, Ident("int")), Field(nil, Ident("error")))),
				Stmts(`
					if pos == len(r.Source) {
						return 0, fmt.Errorf("expected character, got EOF")
					}
					c, w := utf8.DecodeRuneInString(r.Source[pos:])
					if c == utf8.RuneError {
						return w, fmt.Errorf("invalid utf8: %q", r.Source[pos:pos+w])
					}
					return w, nil
				`)...),
			),
		},
		{
			`package mypackage

func QuestionHandler0(r *Result, pos int) (int, error) {
	w, err := Handler1(r, pos)
	if err != nil {
		return 0, nil
	}
	return w, nil
}
`,
			Package("mypackage", []string{}, QuestionHandler("QuestionHandler0", "Handler1")),
		},
	}

	for _, tt := range tests {
		got, err := Render(tt.f)
		if err != nil {
			t.Errorf("package returns error %s, want success", err)
			return
		}
		if got != tt.want {
			t.Errorf("package generated incorrectly, got:\n---\n%s\n---\nwant\n---\n%s\n---", got, tt.want)
		}
	}
}

func TestStmt(t *testing.T) {
	tests := []struct {
		want string
		stmt ast.Stmt
	}{
		{
			`fmt.Println("hello")`,
			ExprStmt(Call(Sel(Ident("fmt"), "Println"), String(`"hello"`))),
		},
		{
			`x := &X{}`,
			Assign(Ident("x"), Unary(token.AND, Composite(Ident("X"), nil))),
		},
	}

	for _, tt := range tests {
		pkg := Package("mypackage", []string{}, Func("myfunc", FuncType(nil, nil), tt.stmt))
		got, err := Render(pkg)
		if err != nil {
			t.Errorf("package returns error %s, want success", err)
			return
		}
		// Skip the package and function vignette.
		got = got[36 : len(got)-3]
		if got != tt.want {
			t.Errorf("package generated incorrectly, got:\n---\n%s\n---\nwant\n---\n%s\n---", got, tt.want)
		}
	}
}

func TestExpr(t *testing.T) {
	tests := []struct {
		expr string
		node ast.Expr
	}{
		{
			`&X{}`,
			Unary(token.AND, Composite(Ident("X"), nil)),
		},
		{
			`&X{}`,
			StructLiteral("&X"),
		},
		{
			`X{}`,
			StructLiteral("X"),
		},
		{
			`a.X{}`,
			StructLiteral("a.X"),
		},
		{
			`&a.X{}`,
			StructLiteral("&a.X"),
		},
		{
			`X{1}`,
			StructLiteral("X", Int("1")),
		},
	}

	for _, tt := range tests {
		node, err := parser.ParseExpr(tt.expr)
		if err != nil {
			t.Errorf("error in test: expr %q does not parse: %s", tt.expr, err)
			continue
		}
		got, err := Render(node)
		if err != nil {
			t.Errorf("error in test: could not render node %s: %s", node, err)
			continue
		}
		if got != tt.expr {
			t.Errorf("Render(ParseExpr(%q)) returns %q, want %q", tt.expr, got, tt.expr)
			continue
		}
		got, err = Render(tt.node)
		if err != nil {
			t.Errorf("error in test: could not render node %s: %s", tt.node, err)
			continue
		}
		if got != tt.expr {
			t.Errorf("Render(%v) returns %q, want %q", tt.node, got, tt.expr)
		}
	}
}

func TestParseStmt(t *testing.T) {
	tests := []struct {
		stmt string
		node ast.Stmt
	}{
		{
			`x := 1`,
			Assign(Ident("x"), Int("1")),
		},
		{
			`for x := range e {
}`,
			Range(Ident("x"), nil, Ident("e")),
		},
		{
			`switch x {
case 1:
	return true
default:
	return false
}`,
			Switch(nil, Ident("x"),
				Case(E(Int("1")), Return(Ident("true"))),
				Case(nil, Return(Ident("false"))),
			),
		},
	}
	for _, tt := range tests {
		node, err := ParseStmt(tt.stmt)
		if err != nil {
			t.Errorf("ParseStmt(%q) returned error %s, want sucess", tt.stmt, err)
			continue
		}
		got, err := Render(node)
		if err != nil {
			t.Errorf("error in test: could not render node %s: %s", node, err)
			continue
		}
		if got != tt.stmt {
			t.Errorf("Render(ParseStmt(%q)) returns %s, want %s", tt.stmt, got, tt.stmt)
			continue
		}
		got, err = Render(tt.node)
		if err != nil {
			t.Errorf("error in test: could not render node %s: %s", tt.node, err)
			continue
		}
		if got != tt.stmt {
			t.Errorf("Render(%v) returns %s, want %s", tt.node, got, tt.stmt)
		}
	}
}

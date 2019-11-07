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

// Package gogen contains a few functions for ad-hoc generation of Go source
// snippets.
package gogen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"unicode"

	"github.com/salikh/peg/parser/charclass"
)

func Render(node interface{}) (string, error) {
	var buf bytes.Buffer
	fset := token.NewFileSet()
	err := format.Node(&buf, fset, node)
	if err != nil {
		return "", fmt.Errorf("error rendering node %v: %s", node, err)
	}
	return buf.String(), nil
}

func String(val string) *ast.BasicLit {
	return &ast.BasicLit{
		Kind:  token.STRING,
		Value: val,
	}
}

func Char(val string) *ast.BasicLit {
	return &ast.BasicLit{
		Kind:  token.CHAR,
		Value: val,
	}
}

func Int(val string) *ast.BasicLit {
	return &ast.BasicLit{
		Kind:  token.INT,
		Value: val,
	}
}

func E(elts ...ast.Expr) []ast.Expr {
	return elts
}

func S(elts ...ast.Stmt) []ast.Stmt {
	return elts
}

func Composite(ty ast.Expr, elts []ast.Expr) *ast.CompositeLit {
	return &ast.CompositeLit{
		Type: ty,
		Elts: elts,
	}
}

func Package(name string, imports []string, decls ...ast.Decl) *ast.File {
	var imps []*ast.ImportSpec
	var specs []ast.Spec
	for _, importedName := range imports {
		impSpec := &ast.ImportSpec{
			Path: String(`"` + importedName + `"`),
		}
		imps = append(imps, impSpec)
		specs = append(specs, impSpec)
	}
	var importDecls []ast.Decl
	if len(specs) > 0 {
		importDecl := &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: specs,
		}
		importDecls = []ast.Decl{importDecl}
	}
	//log.Infof("len(specs) = %d, specs = %#v", len(specs), specs)
	return &ast.File{
		Name: &ast.Ident{
			Name: name,
		},
		Imports: imps,
		Decls:   append(importDecls, decls...),
	}
}

func Ident(name string) *ast.Ident {
	return &ast.Ident{
		Name: name,
	}
}

func Field(names []*ast.Ident, typeExpr ast.Expr) *ast.Field {
	return &ast.Field{
		Names: names,
		Type:  typeExpr,
	}
}

func AField(name string, typeExpr ast.Expr) *ast.Field {
	return &ast.Field{
		Names: []*ast.Ident{Ident(name)},
		Type:  typeExpr,
	}
}

func Fields(args ...*ast.Field) []*ast.Field {
	return args
}

func FuncType(args, returns []*ast.Field) *ast.FuncType {
	return &ast.FuncType{
		Params:  &ast.FieldList{List: args},
		Results: &ast.FieldList{List: returns},
	}
}

func SliceType(x ast.Expr) *ast.ArrayType {
	return &ast.ArrayType{
		Len: nil, // Slice.
		Elt: x,
	}
}

func MapType(key, value ast.Expr) *ast.MapType {
	return &ast.MapType{
		Key:   key,
		Value: value,
	}
}

func Sel(x ast.Expr, sel string) *ast.SelectorExpr {
	return &ast.SelectorExpr{
		X:   x,
		Sel: Ident(sel),
	}
}

func Slice(x, low, high ast.Expr, other ...ast.Expr) *ast.SliceExpr {
	r := &ast.SliceExpr{
		X:    x,
		Low:  low,
		High: high,
	}
	if len(other) == 1 {
		r.Max = other[1]
	} else if len(other) > 1 {
		panic("unexpected number of arguments to gogen.Slice")
	}
	return r
}

func Index(x, index ast.Expr) *ast.IndexExpr {
	return &ast.IndexExpr{
		X:     x,
		Index: index,
	}
}

func KeyValue(key, value ast.Expr) *ast.KeyValueExpr {
	return &ast.KeyValueExpr{
		Key:   key,
		Value: value,
	}
}

func Call(fun ast.Expr, args ...ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun:  fun,
		Args: args,
	}
}

func ExprStmt(x ast.Expr) *ast.ExprStmt {
	return &ast.ExprStmt{
		X: x,
	}
}

func Func(name string, funcType *ast.FuncType, stmt ...ast.Stmt) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: Ident(name),
		Type: funcType,
		Body: &ast.BlockStmt{
			List: stmt,
		},
	}
}

func Return(vals ...ast.Expr) *ast.ReturnStmt {
	return &ast.ReturnStmt{
		Results: vals,
	}
}

func Type(name string, ty ast.Expr) *ast.GenDecl {
	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{&ast.TypeSpec{
			Name: Ident(name),
			Type: ty,
		}},
	}
}

func Star(x ast.Expr) *ast.StarExpr {
	return &ast.StarExpr{
		X: x,
	}
}

func Struct(fields ...*ast.Field) ast.Expr {
	return &ast.StructType{
		Fields: &ast.FieldList{
			List: fields,
		},
	}
}

func Var(name string, ty, val ast.Expr) *ast.GenDecl {
	var values []ast.Expr
	if val != nil {
		values = []ast.Expr{val}
	}
	return &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names:  []*ast.Ident{Ident(name)},
			Type:   ty,
			Values: values,
		}},
	}
}

func Const(name string, ty, val ast.Expr) *ast.GenDecl {
	return &ast.GenDecl{
		Tok: token.CONST,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names:  []*ast.Ident{Ident(name)},
			Type:   ty,
			Values: []ast.Expr{val},
		}},
	}
}

func AppendDecl(f *ast.File, decl ast.Decl) {
	f.Decls = append(f.Decls, decl)
}

func DeclStmt(decl ast.Decl) *ast.DeclStmt {
	return &ast.DeclStmt{
		Decl: decl,
	}
}

func If(init ast.Stmt, cond ast.Expr, statements ...ast.Stmt) *ast.IfStmt {
	return &ast.IfStmt{
		Init: init,
		Cond: cond,
		Body: Block(statements...),
		Else: nil,
	}
}

// Else modifies ast.Ifstmt in-place to add an else clause.
func Else(ifstmt *ast.IfStmt, els ast.Stmt) *ast.IfStmt {
	ifstmt.Else = els
	return ifstmt
}

func For(init ast.Stmt, cond ast.Expr, post ast.Stmt, statements ...ast.Stmt) *ast.ForStmt {
	return &ast.ForStmt{
		Init: init,
		Cond: cond,
		Post: post,
		Body: Block(statements...),
	}
}

func Block(statements ...ast.Stmt) *ast.BlockStmt {
	return &ast.BlockStmt{
		List: statements,
	}
}

func Binary(x ast.Expr, op token.Token, y ast.Expr) *ast.BinaryExpr {
	return &ast.BinaryExpr{
		X:  x,
		Op: op,
		Y:  y,
	}
}

func Unary(op token.Token, x ast.Expr) *ast.UnaryExpr {
	return &ast.UnaryExpr{
		X:  x,
		Op: op,
	}
}

func Assign(lhs, rhs ast.Expr, tok ...token.Token) *ast.AssignStmt {
	if len(tok) == 0 {
		tok = []token.Token{token.DEFINE}
	}
	return &ast.AssignStmt{
		Lhs: []ast.Expr{lhs},
		Tok: tok[0],
		Rhs: []ast.Expr{rhs},
	}
}

func AssignMulti(lhs, rhs []ast.Expr, tok ...token.Token) *ast.AssignStmt {
	if len(tok) == 0 {
		tok = []token.Token{token.DEFINE}
	}
	return &ast.AssignStmt{
		Lhs: lhs,
		Tok: tok[0],
		Rhs: rhs,
	}
}

func Range(key, value, x ast.Expr, stmt ...ast.Stmt) *ast.RangeStmt {
	return &ast.RangeStmt{
		Key:   key,
		Value: value,
		X:     x,
		Tok:   token.DEFINE,
		Body: &ast.BlockStmt{
			List: stmt,
		},
	}
}

func Case(expr []ast.Expr, stmt ...ast.Stmt) *ast.CaseClause {
	return &ast.CaseClause{
		List: expr,
		Body: stmt,
	}
}

// cases should include *ast.CaseClause only.
func Switch(init ast.Stmt, tag ast.Expr, cases ...ast.Stmt) *ast.SwitchStmt {
	return &ast.SwitchStmt{
		Init: init,
		Tag:  tag,
		Body: &ast.BlockStmt{
			List: cases,
		},
	}
}

// LiteralHandler generates Go AST for the literal handler.
func LiteralHandler(name, literal string) *ast.FuncDecl {
	return Func(name, FuncType(Fields(AField("r", Star(Ident("Result"))),
		AField("pos", Ident("int"))), Fields(Field(nil, Ident("int")), Field(nil, Ident("error")))),
		DeclStmt(Const("literal", nil, String(strconv.Quote(literal)))),
		If(nil, Binary(
			Binary(Call(Ident("len"), Sel(Ident("r"), "Source")), token.SUB, Ident("pos")),
			token.LSS, Call(Ident("len"), Ident("literal"))),
			Return(Int("0"), Call(Sel(Ident("fmt"), "Errorf"), String(`"expecting %q, got %q"`),
				Ident("literal"), Slice(Sel(Ident("r"), "Source"), Ident("pos"), nil)))),
		Assign(Ident("next"), Slice(Sel(Ident("r"), "Source"), Ident("pos"), Binary(Ident("pos"), token.ADD, Call(Ident("len"), Ident("literal"))))),
		If(nil, Binary(Ident("next"), token.NEQ, Ident("literal")),
			Return(Int("0"), Call(Sel(Ident("fmt"), "Errorf"), String(`"expecting %q, got %q"`),
				Ident("literal"), Ident("next")))),
		Return(Call(Ident("len"), Ident("literal")), Ident("nil")))
}

func makeCharClassMap(name string, m map[rune]bool) *ast.DeclStmt {
	var vals []ast.Expr
	for c := range m {
		vals = append(vals, KeyValue(Char(strconv.QuoteRune(c)), Ident("true")))
	}
	return DeclStmt(Var(name, nil, Composite(MapType(Ident("rune"), Ident("bool")), vals)))
}

func SelIdent(name string) ast.Expr {
	pos := strings.Index(name, ".")
	if pos == -1 {
		return Ident(name)
	}
	return Sel(Ident(name[:pos]), name[pos+1:])
}

func StructLiteral(name string, elts ...ast.Expr) ast.Expr {
	pointerType := false
	if name[0] == '&' {
		name = name[1:]
		pointerType = true
	}
	typeExpr := SelIdent(name)
	var ret ast.Expr = Composite(typeExpr, elts)
	if pointerType {
		// TODO(salikh): Check whether this is a correct way to create a struct literal.
		ret = Unary(token.AND, ret)
	}
	return ret
}

func makeRange(typeName string, lo, hi, stride int64) ast.Expr {
	return StructLiteral(typeName,
		KeyValue(Ident("Lo"), Int("0x"+strconv.FormatInt(lo, 16))),
		KeyValue(Ident("Hi"), Int("0x"+strconv.FormatInt(hi, 16))),
		KeyValue(Ident("Stride"), Int(strconv.FormatInt(stride, 10))))
}

func makeRangeTable(name string, t *unicode.RangeTable) *ast.DeclStmt {
	var keyvals []ast.Expr
	if t.R16 != nil {
		var vals []ast.Expr
		for _, rg := range t.R16 {
			vals = append(vals, makeRange("unicode.Range16", int64(rg.Lo), int64(rg.Hi), int64(rg.Stride)))
		}
		keyvals = append(keyvals, KeyValue(Ident("R16"), Composite(SliceType(SelIdent("unicode.Range16")), vals)))
	}
	if t.R32 != nil {
		var vals []ast.Expr
		for _, rg := range t.R32 {
			vals = append(vals, makeRange("unicode.Range16", int64(rg.Lo), int64(rg.Hi), int64(rg.Stride)))
		}
		keyvals = append(keyvals, KeyValue(Ident("R32"), Composite(Star(SelIdent("unicode.Range32")), vals)))
	}
	return DeclStmt(Var(name, nil, StructLiteral("&unicode.RangeTable", keyvals...)))
}

// CharClassHandler generates Go AST for the character class handler.
// Note: it panics on invalid arg string, because checking validity is a
// responsibility of the PEG parser.
func CharClassHandler(name string, cc *charclass.CharClass) *ast.FuncDecl {
	var stmt []ast.Stmt
	var cond ast.Expr
	switch {
	case cc.Special == "[:alnum:]":
		cond = Binary(Call(Sel(Ident("unicode"), "IsLetter"), Ident("c")), token.LOR,
			Call(Sel(Ident("unicode"), "IsNumber"), Ident("c")))
	case cc.Special != "":
		cond = Call(Sel(Ident("unicode"), cc.Special), Ident("c"))
	default:
		if cc.Map != nil {
			stmt = append(stmt, makeCharClassMap("charClassMap", cc.Map))
			cond = Index(Ident("charClassMap"), Ident("c"))
		}
		if cc.RangeTable != nil {
			stmt = append(stmt, makeRangeTable("rangeTable", cc.RangeTable))
			newcond := Call(Sel(Ident("unicode"), "Is"), Ident("rangeTable"), Ident("c"))
			if cond != nil {
				cond = Binary(cond, token.LOR, newcond)
			} else {
				cond = newcond
			}
		}
	}
	if !cc.Negated {
		cond = Unary(token.NOT, cond)
	}
	stmt = append(stmt,
		AssignMulti(E(Ident("c"), Ident("w")), E(Call(Sel(Ident("utf8"), "DecodeRuneInString"),
			Slice(Sel(Ident("r"), "Source"), Ident("pos"), nil)))),
		If(nil, Binary(Ident("w"), token.EQL, Int("0")), Return(Int("0"), Call(Sel(Ident("fmt"), "Errorf"),
			String(`"expecting char, got EOF"`)))),
		If(nil, Binary(Ident("c"), token.EQL, Sel(Ident("utf8"), "RuneError")),
			Return(Ident("w"), Call(Sel(Ident("fmt"), "Errorf"), String(`"invalid utf8: %q"`),
				Slice(Sel(Ident("r"), "Source"), Ident("pos"), Binary(Ident("pos"), token.ADD, Ident("w")))))),
		If(nil, cond,
			Return(Int("0"), Call(Sel(Ident("fmt"), "Errorf"), String(fmt.Sprintf(`"character %%q does not match class [%s]"`, cc)), Ident("c")))),
		Return(Ident("w"), Ident("nil")),
	)
	return Func(name, FuncType(
		Fields(AField("r", Star(Ident("Result"))), AField("pos", Ident("int"))),
		Fields(Field(nil, Ident("int")), Field(nil, Ident("error")))),
		stmt...)
}

func StarHandler(name, subhandler string) *ast.FuncDecl {
	return Func(name, FuncType(Fields(AField("r", Star(Ident("Result"))),
		AField("pos", Ident("int"))), Fields(Field(nil, Ident("int")), Field(nil, Ident("error")))),
		Assign(Ident("ww"), Int("0")),
		For(AssignMulti(E(Ident("w"), Ident("err")), E(Call(Ident(subhandler),
			Ident("r"), Ident("pos")))),
			Binary(Binary(Ident("err"), token.EQL, Ident("nil")),
				token.LAND, Binary(Ident("w"), token.GTR, Int("0"))),
			AssignMulti(E(Ident("w"), Ident("err")),
				E(Call(Ident(subhandler), Ident("r"), Binary(Ident("pos"), token.ADD, Ident("ww")))), token.ASSIGN),
			Assign(Ident("ww"), Ident("w"), token.ADD_ASSIGN),
		),
		Return(Ident("ww"), Ident("nil")),
	)
}

func GroupHandler(name string, subhandlers []string) *ast.FuncDecl {
	st := []ast.Stmt{
		Assign(Ident("ww"), Int("0")),
		DeclStmt(Var("w", Ident("int"), nil)),
		DeclStmt(Var("err", Ident("error"), nil)),
	}
	for _, subhandler := range subhandlers {
		st = append(st,
			AssignMulti(E(Ident("w"), Ident("err")), E(
				Call(Ident(subhandler),
					Ident("r"), Binary(Ident("pos"), token.ADD, Ident("ww")))), token.ASSIGN),
			Assign(Ident("ww"), Ident("w"), token.ADD_ASSIGN),
			If(nil, Binary(Ident("err"), token.NEQ, Ident("nil")),
				Return(Ident("ww"), Ident("err"))))
	}
	st = append(st, Return(Ident("ww"), Ident("nil")))
	return Func(name, FuncType(Fields(AField("r", Star(Ident("Result"))),
		AField("pos", Ident("int"))), Fields(Field(nil, Ident("int")), Field(nil, Ident("error")))), st...)
}

func PredicateHandler(name, subhandler string, negative bool) *ast.FuncDecl {
	negStr := "false"
	if negative {
		negStr = "true"
	}
	return Func(name, FuncType(Fields(AField("r", Star(Ident("Result"))),
		AField("pos", Ident("int"))), Fields(Field(nil, Ident("int")), Field(nil, Ident("error")))),
		DeclStmt(Const("negative", nil, Ident(negStr))),
		AssignMulti(E(Ident("_"), Ident("err")), E(
			Call(Ident(subhandler),
				Ident("r"), Ident("pos")))),
		If(nil, Binary(Ident("negative"), token.EQL, Binary(Ident("err"), token.NEQ, Ident("nil"))),
			Return(Int("0"), Ident("nil"))),
		If(nil, Binary(Ident("err"), token.EQL, Ident("nil")),
			Return(Int("0"), Call(Sel(Ident("fmt"), "Errorf"), String(`"negative predicate matched"`)))),
		Return(Int("0"), Ident("err")),
	)
}

func ChoiceHandler(name, subhandler string, subhandlers ...string) *ast.FuncDecl {
	stmts := []ast.Stmt{
		AssignMulti(E(Ident("w"), Ident("err")), E(
			Call(Ident(subhandler), Ident("r"), Ident("pos")))),
	}
	for _, subhandler := range subhandlers {
		stmts = append(stmts,
			If(nil, Binary(Ident("err"), token.NEQ, Ident("nil")),
				AssignMulti(E(Ident("w"), Ident("err")), E(
					Call(Ident(subhandler), Ident("r"), Ident("pos"))), token.ASSIGN),
			))
	}
	stmts = append(stmts, Return(Ident("w"), Ident("err")))
	return Func(name, FuncType(Fields(AField("r", Star(Ident("Result"))),
		AField("pos", Ident("int"))), Fields(Field(nil, Ident("int")), Field(nil, Ident("error")))), stmts...)
}

func CaptureHandler(name, subhandler string) *ast.FuncDecl {
	return Func(name, FuncType(Fields(AField("r", Star(Ident("Result"))),
		AField("pos", Ident("int"))), Fields(Field(nil, Ident("int")), Field(nil, Ident("error")))),
		Stmts(fmt.Sprintf(`
			w, err := %s(r, pos)
			if err != nil {
				return w, err
			}
			r.TopNode().Start = pos
			r.TopNode().Text = r.Source[pos:pos+w]
			return w, nil
		`, subhandler))...)
}

func DotHandler(name string) *ast.FuncDecl {
	return Func(name, FuncType(Fields(AField("r", Star(Ident("Result"))),
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
				`)...)
}

func QuestionHandler(name, subhandler string) *ast.FuncDecl {
	stmts := []ast.Stmt{
		AssignMulti(E(Ident("w"), Ident("err")), E(
			Call(Ident(subhandler), Ident("r"), Ident("pos")))),
	}
	stmts = append(stmts,
		Stmts(`
			if err != nil {
				return 0, nil
			}
			return w, nil
		`)...)
	return Func(name, FuncType(Fields(AField("r", Star(Ident("Result"))),
		AField("pos", Ident("int"))), Fields(Field(nil, Ident("int")), Field(nil, Ident("error")))), stmts...)
}

func PlusHandler(name, subhandler string) *ast.FuncDecl {
	stmts := Stmts(fmt.Sprintf(`
			w, err := %s(r, pos)
			if err != nil {
				return 0, err
			}
			ww := w
			for w, err = %s(r, pos+ww); err == nil && w > 0 && pos+ww < len(r.Source); w, err = %s(r, pos+ww) {
				ww += w
			}
			return ww, nil
		`, subhandler, subhandler, subhandler))
	return Func(name, FuncType(Fields(AField("r", Star(Ident("Result"))),
		AField("pos", Ident("int"))), Fields(Field(nil, Ident("int")), Field(nil, Ident("error")))), stmts...)
}

func ParseStmt(stmt string) (ast.Stmt, error) {
	source := `package my

func myfunc() {
	` + stmt + `
}
`
	fset := token.NewFileSet()
	goFile, err := parser.ParseFile(fset, "my.go", source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("could not parse go statement %q: %s", stmt, err)
	}
	return goFile.Decls[0].(*ast.FuncDecl).Body.List[0], nil
}

func ParseStmts(stmts string) ([]ast.Stmt, error) {
	source := `package my

func myfunc() {
	` + stmts + `
}
`
	fset := token.NewFileSet()
	goFile, err := parser.ParseFile(fset, "my.go", source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("could not parse go statements %q: %s", stmts, err)
	}
	return goFile.Decls[0].(*ast.FuncDecl).Body.List, nil
}

// Expr returns the ast.Expr parsed from a string. It panics
// if the string cannot be parsed as Go expression.
func Expr(expr string) ast.Expr {
	node, err := parser.ParseExpr(expr)
	if err != nil {
		panic(fmt.Sprintf("could not parse Go expression %q: %s", expr, err))
	}
	return node
}

// Stmt returns the ast.Stmt parsed from a string. It panics
// if the string cannot be parsed as Go statement.
func Stmt(stmt string) ast.Stmt {
	node, err := ParseStmt(stmt)
	if err != nil {
		panic(fmt.Sprintf("could not parse Go statement %q: %s", stmt, err))
	}
	return node
}

func Stmts(stmts string) []ast.Stmt {
	nodes, err := ParseStmts(stmts)
	if err != nil {
		panic(fmt.Sprintf("could not parse Go statements %q: %s", stmts, err))
	}
	return nodes
}

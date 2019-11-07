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

package astutil

import (
	"fmt"
	"go/ast"
	"reflect"
	"strings"

	log "github.com/golang/glog"
)

func Dup(n ast.Node) ast.Node {
	if n == nil {
		return nil
	}
	switch t := n.(type) {
	case *ast.File:
		return DupFile(t)
	default:
		log.Exitf("NYI: Dup(%s)", reflect.TypeOf(n).String())
	}
	return nil
}

func DupFile(f *ast.File) *ast.File {
	if f == nil {
		return nil
	}
	return &ast.File{
		Name:    DupIdent(f.Name),
		Decls:   DupDeclList(f.Decls),
		Scope:   f.Scope, // NOTE: not copied
		Imports: DupImportSpecs(f.Imports),
	}
}

func DupImportSpecs(imps []*ast.ImportSpec) []*ast.ImportSpec {
	if imps == nil {
		return nil
	}
	r := make([]*ast.ImportSpec, len(imps))
	for i, imp := range imps {
		r[i] = &ast.ImportSpec{
			Name: DupIdent(imp.Name),
			Path: DupBasicLit(imp.Path),
		}
	}
	return r
}

func DupDeclList(ll []ast.Decl) []ast.Decl {
	if ll == nil {
		return nil
	}
	r := make([]ast.Decl, len(ll))
	for i, d := range ll {
		r[i] = DupDecl(d)
	}
	return r
}

func DupDecl(d ast.Decl) ast.Decl {
	if d == nil {
		return nil
	}
	switch t := d.(type) {
	case *ast.FuncDecl:
		return DupFuncDecl(t)
	case *ast.GenDecl:
		return DupGenDecl(t)
	default:
		log.Exitf("NYI: DupDecl(%s)", reflect.TypeOf(d).String())
	}
	return nil
}

func DupBlockStmt(b *ast.BlockStmt) *ast.BlockStmt {
	if b == nil {
		return nil
	}
	return &ast.BlockStmt{
		List: DupStmtList(b.List),
	}
}

func DupFuncDecl(f *ast.FuncDecl) *ast.FuncDecl {
	if f == nil {
		return nil
	}
	return &ast.FuncDecl{
		Recv: f.Recv, // NOTE: we are not rewriting args
		Name: DupIdent(f.Name),
		Type: f.Type, // NOTE: we are not rewriting types.
		Body: DupBlockStmt(f.Body),
	}
}

func DupIdentList(l []*ast.Ident) []*ast.Ident {
	if l == nil {
		return nil
	}
	r := make([]*ast.Ident, len(l))
	for i, id := range l {
		r[i] = DupIdent(id)
	}
	return r
}

func DupSpec(s ast.Spec) ast.Spec {
	if s == nil {
		return nil
	}
	switch t := s.(type) {
	case *ast.ValueSpec:
		return &ast.ValueSpec{
			Names:  DupIdentList(t.Names),
			Type:   t.Type, // NOTE: we are not rewriting types.
			Values: DupExprList(t.Values),
		}
	case *ast.ImportSpec:
		return &ast.ImportSpec{
			Name: DupIdent(t.Name),
			Path: DupBasicLit(t.Path),
		}
	case *ast.TypeSpec:
		return &ast.TypeSpec{
			Name: DupIdent(t.Name),
			Type: DupExpr(t.Type),
		}
	default:
		log.Exitf("NYI: DupSpec(%s)", reflect.TypeOf(s).String())
	}
	return nil
}

func DupGenDecl(g *ast.GenDecl) *ast.GenDecl {
	if g == nil {
		return nil
	}
	var r ast.GenDecl = *g
	r.Specs = make([]ast.Spec, len(g.Specs))
	for i, spec := range g.Specs {
		r.Specs[i] = DupSpec(spec)
	}
	return &r
}

func DupCallExpr(t *ast.CallExpr) *ast.CallExpr {
	if t == nil {
		return nil
	}
	return &ast.CallExpr{
		Fun:  DupExpr(t.Fun),
		Args: DupExprList(t.Args),
	}
}

func DupIdent(id *ast.Ident) *ast.Ident {
	if id == nil {
		return nil
	}
	return &ast.Ident{Name: id.Name}
}

func DupFieldList(t *ast.FieldList) *ast.FieldList {
	if t == nil {
		return nil
	}
	return &ast.FieldList{List: DupFieldSlice(t.List)}
}

func DupFieldSlice(ff []*ast.Field) []*ast.Field {
	if ff == nil {
		return nil
	}
	r := make([]*ast.Field, len(ff))
	for i, f := range ff {
		r[i] = DupField(f)
	}
	return r
}

func DupFuncType(t *ast.FuncType) *ast.FuncType {
	if t == nil {
		return nil
	}
	return &ast.FuncType{
		Params:  DupFieldList(t.Params),
		Results: DupFieldList(t.Results),
	}
}

func DupExpr(l ast.Expr) ast.Expr {
	if l == nil {
		return nil
	}
	switch t := l.(type) {
	case *ast.BasicLit:
		return DupBasicLit(t)
	case *ast.CompositeLit:
		return DupCompositeLit(t)
	case *ast.FuncLit:
		return &ast.FuncLit{
			Type: DupFuncType(t.Type),
			Body: DupBlockStmt(t.Body),
		}
	case *ast.Ident:
		return DupIdent(t)
	case *ast.BinaryExpr:
		return &ast.BinaryExpr{
			X:  DupExpr(t.X),
			Op: t.Op,
			Y:  DupExpr(t.Y),
		}
	case *ast.CallExpr:
		return DupCallExpr(t)
	case *ast.IndexExpr:
		return &ast.IndexExpr{
			X:     DupExpr(t.X),
			Index: DupExpr(t.Index),
		}
	case *ast.KeyValueExpr:
		return &ast.KeyValueExpr{
			Key:   DupExpr(t.Key),
			Value: DupExpr(t.Value),
		}
	case *ast.ParenExpr:
		return &ast.ParenExpr{
			X: DupExpr(t.X),
		}
	case *ast.SelectorExpr:
		return &ast.SelectorExpr{X: DupExpr(t.X), Sel: DupIdent(t.Sel)}
	case *ast.SliceExpr:
		return &ast.SliceExpr{
			X:      DupExpr(t.X),
			Low:    DupExpr(t.Low),
			High:   DupExpr(t.High),
			Max:    DupExpr(t.Max),
			Slice3: t.Slice3,
		}
	case *ast.StarExpr:
		return &ast.StarExpr{
			Star: t.Star,
			X:    DupExpr(t.X),
		}
	case *ast.UnaryExpr:
		return &ast.UnaryExpr{
			Op: t.Op,
			X:  DupExpr(t.X),
		}
	case *ast.ArrayType:
		return &ast.ArrayType{
			Len: DupExpr(t.Len),
			Elt: DupExpr(t.Elt),
		}
	case *ast.FuncType:
		return DupFuncType(t)
	case *ast.MapType:
		return &ast.MapType{
			Key:   DupExpr(t.Key),
			Value: DupExpr(t.Value),
		}
	case *ast.StructType:
		return &ast.StructType{
			Fields:     DupFieldList(t.Fields),
			Incomplete: t.Incomplete,
		}
	default:
		log.Exitf("NYI: DupExpr(%s)", reflect.TypeOf(l).String())
	}
	return l
}

func DupExprList(ee []ast.Expr) []ast.Expr {
	if ee == nil {
		return nil
	}
	r := make([]ast.Expr, len(ee))
	for i, e := range ee {
		r[i] = DupExpr(e)
	}
	return r
}

func DupField(f *ast.Field) *ast.Field {
	if f == nil {
		return nil
	}
	return &ast.Field{
		Names: DupIdentList(f.Names),
		Type:  DupExpr(f.Type),
		Tag:   DupBasicLit(f.Tag),
	}
}

func DupStmt(l ast.Stmt) ast.Stmt {
	if l == nil {
		return nil
	}
	switch t := l.(type) {
	case *ast.AssignStmt:
		return &ast.AssignStmt{
			Lhs:    DupExprList(t.Lhs),
			TokPos: t.TokPos,
			Tok:    t.Tok,
			Rhs:    DupExprList(t.Rhs),
		}
	case *ast.DeclStmt:
		return &ast.DeclStmt{
			Decl: DupDecl(t.Decl),
		}
	case *ast.DeferStmt:
		return &ast.DeferStmt{
			Call: DupCallExpr(t.Call),
		}
	case *ast.ExprStmt:
		return &ast.ExprStmt{
			X: DupExpr(t.X),
		}
	case *ast.ForStmt:
		return &ast.ForStmt{
			Init: DupStmt(t.Init),
			Cond: DupExpr(t.Cond),
			Post: DupStmt(t.Post),
			Body: DupBlockStmt(t.Body),
		}
	case *ast.IfStmt:
		return &ast.IfStmt{
			Init: DupStmt(t.Init),
			Cond: DupExpr(t.Cond),
			Body: DupBlockStmt(t.Body),
			Else: DupStmt(t.Else),
		}
	case *ast.IncDecStmt:
		return &ast.IncDecStmt{
			X:   DupExpr(t.X),
			Tok: t.Tok,
		}
	case *ast.RangeStmt:
		return &ast.RangeStmt{
			Key:   DupExpr(t.Key),
			Value: DupExpr(t.Value),
			Tok:   t.Tok,
			X:     DupExpr(t.X),
			Body:  DupBlockStmt(t.Body),
		}
	case *ast.ReturnStmt:
		return &ast.ReturnStmt{
			Results: DupExprList(t.Results),
		}
	default:
		log.Exitf("NYI: DupStmt(%s)", reflect.TypeOf(l).String())
	}
	return nil
}

func DupStmtList(l []ast.Stmt) []ast.Stmt {
	if l == nil {
		return nil
	}
	r := make([]ast.Stmt, len(l))
	for i := range l {
		r[i] = DupStmt(l[i])
	}
	return r
}

func DupBasicLit(l *ast.BasicLit) *ast.BasicLit {
	if l == nil {
		return nil
	}
	return &ast.BasicLit{
		Kind:  l.Kind,
		Value: l.Value,
	}
}

func DupCompositeLit(l *ast.CompositeLit) *ast.CompositeLit {
	if l == nil {
		return nil
	}
	return &ast.CompositeLit{
		Type: DupExpr(l.Type),
		Elts: DupExprList(l.Elts),
	}
}

func StringSlice(s interface{}) string {
	var r []string
	switch t := s.(type) {
	case []ast.Decl:
		for _, x := range t {
			r = append(r, String(x))
		}
	case []ast.Spec:
		for _, x := range t {
			r = append(r, String(x))
		}
	default:
		log.Exitf("NYI: StringSlice(%s)", reflect.TypeOf(s))
	}
	return strings.Join(r, " ")
}

func String(n ast.Node) string {
	ty := reflect.TypeOf(n)
	switch t := n.(type) {
	case *ast.File:
		return "(File " + String(t.Name) + " " + StringSlice(t.Decls) + ")"
	case *ast.SelectorExpr:
		return "(SelectorExpr " + String(t.X) + " . " + t.Sel.Name + ")"
	case *ast.Ident:
		return "(Ident " + t.Name + ")"
	case *ast.GenDecl:
		return fmt.Sprintf("(%v %s)", t.Tok, StringSlice(t.Specs))
	default:
		return "(" + ty.String() + ")"
	}
}

func Wrap(s string) string {
	var r []string
	const shift = " "
	indent := ""
	pos := 0
	base := 0
	for i := range s {
		pos++
		switch s[i] {
		case '(':
			indent = indent + shift
		case ')':
			indent = indent[0 : len(indent)-len(shift)]
		case ' ':
			if pos > 40 {
				r = append(r, s[base:i], "\n", indent)
				pos = len(indent)
				base = i + 1
			}
		}
	}
	r = append(r, s[base:len(s)])
	return strings.Join(r, "")
}

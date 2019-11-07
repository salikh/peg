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

// Binary gentests is used to generated the test sources from the tests package.
//
//go:generate go run gentests.go
package main

import (
	"bytes"
	"flag"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	"github.com/salikh/peg/compat/runfiles"
	"github.com/salikh/peg/generator"
	peg "github.com/salikh/peg/parser"
	"github.com/salikh/peg/tests"
)

var (
	outputDir = flag.String("output_dir", "generated", "The output directory to write files to. It is deleted and recreated.")
)

// extractFirstIdent is a quick and hacky solution to extract a human-readable name
// from a grammar.
func extractFirstIdent(s string) string {
	parts := strings.SplitN(s, " ", 20)
	for _, part := range parts {
		if part != "" {
			return part
		}
	}
	return "grammar"
}

// genFinder implements ast.Visitor interface to find a GenDecl
// that defines a given constant.
type genFinder struct {
	name string
	token.Token
	*ast.GenDecl
}

func (v *genFinder) Visit(node ast.Node) ast.Visitor {
	if v.GenDecl != nil {
		return nil
	}
	switch t := node.(type) {
	case *ast.GenDecl:
		if t.Tok != v.Token || len(t.Specs) != 1 {
			break
		}
		switch s := t.Specs[0].(type) {
		case *ast.ValueSpec:
			if len(s.Names) == 1 && s.Names[0].Name == v.name {
				v.GenDecl = t
				return nil
			}
		}
	}
	return v
}

func main() {
	flag.Parse()
	log.Info("Generating parser tests...")
	err := os.RemoveAll(*outputDir)
	if err != nil {
		log.Exitf("Error cleaning up %s/: %s", *outputDir, err)
	}
	err = os.MkdirAll(*outputDir, 0775)
	if err != nil {
		log.Exitf("Error trying to mkdir %s/: %s", *outputDir, err)
	}
	// Parse a template of the test.
	fset := token.NewFileSet()
	gentestTemplate := runfiles.Path("github.com/salikh/peg/generator/testing/test_template.golang")
	testTree, err := parser.ParseFile(fset, gentestTemplate, nil, parser.ParseComments)
	if err != nil {
		log.Exitf("Could not parse %s: %s", gentestTemplate, err)
	}
	// Find the testNum constant.
	v := &genFinder{name: "testNum", Token: token.CONST}
	ast.Walk(v, testTree)
	if v.GenDecl == nil {
		log.Exitf("Could not find testNum in %s", gentestTemplate)
	}
	testNumValue := v.GenDecl.Specs[0].(*ast.ValueSpec).Values[0].(*ast.BasicLit)
	// A counter of the directories with the same name.
	dirs := make(map[string]int)
	config := printer.Config{Mode: printer.UseSpaces, Tabwidth: 2}
	for i, test := range tests.Positive {
		_, err := peg.New(test.Grammar)
		if err != nil {
			log.Exitf("Failed to parse the grammar [%s]: %s", test.Grammar, err)
		}
		name := extractFirstIdent(test.Grammar)
		g, err := generator.New(test.Grammar)
		if err != nil {
			log.Infof("Failed to parse PEG [%s]: %s", test.Grammar, err)
			continue
		}
		goSource, err := g.Generate("gen")
		if err != nil {
			log.Infof("Failed to generate go source for [%s]: %s", test.Grammar, err)
			continue
		}
		count := dirs[name] + 1
		dirs[name] = count
		if count > 1 {
			name = name + strconv.FormatInt(int64(count), 10)
		}
		log.Infof("%d: %s", i, name)
		dir := filepath.Join(*outputDir, name)
		err = os.Mkdir(dir, 0775)
		if err != nil {
			log.Exitf("Failed to mkdir %s: %s", dir, err)
		}
		filename := filepath.Join(dir, "gen.go")
		err = ioutil.WriteFile(filename, []byte(goSource), 0664)
		if err != nil {
			log.Exitf("Failed to write %s: %s", filename, err)
		}
		// Overwrite the testNum value in place.
		testNumValue.Value = strconv.FormatInt(int64(i), 10)
		var buf bytes.Buffer
		err = config.Fprint(&buf, fset, testTree)
		filename = filepath.Join(dir, "gen_test.go")
		err = ioutil.WriteFile(filename, buf.Bytes(), 0664)
		if err != nil {
			log.Exitf("Failed to write %s: %s", filename, err)
		}
	}
}

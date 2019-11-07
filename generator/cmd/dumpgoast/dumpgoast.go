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

package main

import (
	"context"
	"flag"
	"fmt"
	"go/parser"
	"go/token"

	log "github.com/golang/glog"
	"github.com/salikh/peg/compat/file"
)

var input = flag.String("input", "", "The name of the go source file to parse.")
var printDecls = flag.Bool("print_decls", false, "If true, print decls.")

func main() {
	flag.Parse()
	goSource, err := file.ReadFile(context.Background(), *input)
	if err != nil {
		log.Exitf("Could not read go source %s: %s", *input, err)
	}
	fset := token.NewFileSet()
	goFile, err := parser.ParseFile(fset, *input, goSource, parser.ParseComments)
	if err != nil {
		log.Exitf("Could not parse go source %s: %s", *input, err)
	}
	fmt.Printf("ast.File: %#v\n", goFile)
	if *printDecls {
		for i, decl := range goFile.Decls {
			fmt.Printf("%d: %#v\n", i, decl)
		}
	}
}

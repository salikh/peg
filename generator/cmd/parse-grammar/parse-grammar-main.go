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

// Binary parse-grammar-main is a utility binary to read parse and print PEG grammars.
// It is mostly useful for ad-hoc testing.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"

	log "github.com/golang/glog"
	"github.com/salikh/peg/generator"
	"github.com/salikh/peg/parser"
)

var (
	grammarFlag       = flag.String("grammar", "", "The path to the grammar file.")
	printSyntaxTree   = flag.Bool("print_syntax_tree", false, "If true, print the raw syntax tree of the PEG grammar.")
	printSemanticTree = flag.Bool("print_semantic_tree", true, "If true, print the semantic tree of the PEG grammar.")
)

func main() {
	flag.Parse()
	if *grammarFlag == "" {
		log.Exitf("--grammar must not be empty.")
	}
	grammar, err := ioutil.ReadFile(*grammarFlag)
	if err != nil {
		log.Exitf("Cannot read the grammar from %q: %s", *grammarFlag, err)
	}
	_, err = parser.New(string(grammar))
	if err != nil {
		log.Exitf("Error parsing the grammar file %q: %s", *grammarFlag, err)
	}
	g, err := generator.New(string(grammar))
	if err != nil {
		log.Exitf("Error parsing the PEG: %s", err)
	}
	if *printSyntaxTree {
		fmt.Println(g.ParseTree())
	}
	if *printSemanticTree {
		fmt.Println(g.Grammar)
	}
}

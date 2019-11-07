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

// Binary example provides an example to parse an arbitrary input with an arbitrary
// grammar using the automatically generated (bootstrapped) parser2 implementation.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"

	log "github.com/golang/glog"
	"github.com/salikh/peg/parser2"
)

var (
	grammarFile = flag.String("grammar", "experimental/users/salikh/tools/go/peg/tests/testdata/io.g",
		"The path to the file with the grammar sources.")
	rule      = flag.String("rule", "", "The top rule to use. If empty, use the first rule.")
	inputFile = flag.String("input_file", "experimental/users/salikh/tools/go/peg/tests/testdata/io.14",
		"The name of the input file to parse.")
	input = flag.String("input", "",
		"The input to feed to the parser. Takes precedence over inputFile")
	ignoreUnconsumedTail = flag.Bool("ignore_unconsumed_tail", false, "ParserOptions.IgnoreUnconsumedTail")
	skipEmptyNodes       = flag.Bool("skip_empty_nodes", false, "ParserOptions.SkipEmptyNodes")
)

var grammar *parser2.Grammar

func main() {
	flag.Parse()
	b, err := ioutil.ReadFile(*grammarFile)
	if err != nil {
		log.Exitf("Error loading grammar: %s", err)
	}
	grammarSource := string(b)
	options := &parser2.ParserOptions{
		IgnoreUnconsumedTail: *ignoreUnconsumedTail,
		SkipEmptyNodes:       *skipEmptyNodes,
	}
	grammar, err = parser2.New(grammarSource, options)
	if err != nil {
		log.Exitf("Error parsing grammar %q: %s", *grammarFile, err)
	}
	source := *input
	if source == "" {
		b, err := ioutil.ReadFile(*inputFile)
		if err != nil {
			log.Exitf("Error reading file %q: %s", *inputFile, err)
		}
		source = string(b)
	}
	log.Infof("source:\n---[%s]---\n", source)
	var result *parser2.Result
	if *rule != "" {
		result, err = grammar.ParseRule(source, *rule)
	} else {
		result, err = grammar.Parse(source)
	}
	if err != nil {
		log.Exitf("Error parsing file %q: %s", *inputFile, err)
	}
	fmt.Printf("Parse tree:\n%s\nOK\n", result.Tree)
}

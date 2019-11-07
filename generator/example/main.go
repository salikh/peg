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

// Binary main shows an example of using the parser that was generated from
// parser expression grammar (PEG).
package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	"github.com/salikh/peg/generator"
	"github.com/salikh/peg/generator/example/simple"
	"github.com/salikh/peg/parser"
)

//go:generate mkdir -p simple
//go:generate go run ../cmd/generator/generator-main.go --grammar=simple.peg --output=simple/simple.go --package=simple

var (
	input = flag.String("input", "a  aa   aaa   b   bb  bbb", "The input string to parse with simple.g grammar.")
)

func main() {
	flag.Parse()
	result, err := simple.Parse(*input)
	if err != nil {
		log.Exitf("Could not parse %q: %s", *input, err)
	}
	fmt.Printf("Parse OK\nTree: %v\n", result.Tree)
	si, err := Convert(result.Tree)
	if err != nil {
		log.Exitf("Semantic conversion failed: %s", err)
	}
	fmt.Printf("Semantic tree:\n%s\n", si)
}

// Simple is a semanic type representing our language.
type Simple struct {
	A []string
	B string
}

// Convert shows how to request the conversion from the parse syntax tree
// to the semantic tree.
func Convert(n *parser.Node) (*Simple, error) {
	val, err := generator.Construct(n, callback, &generator.AccessorOptions{ErrorOnUnusedChild: true})
	if err != nil {
		return nil, err
	}
	return val.(*Simple), nil
}

// callback is a an example of a conversion from the parse syntax tree (generically typed)
// to the semantic typed tree.
func callback(label string, ac generator.Accessor) (interface{}, error) {
	switch label {
	case "Simple":
		return &Simple{
			A: ac.Get("A", []string{}).([]string),
			B: strings.Join(ac.Get("B", []string{}).([]string), " "),
		}, nil
	case "A":
		return ac.Node().Text, nil
	case "B":
		return ac.Node().Text, nil
	case "_":
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected label requested: %q", label)
}

// String provides the serialization format for the semantic tree
// that is compatible with parse tree serialization format.
func (si *Simple) String() string {
	r := []string{"(Simple"}
	if si.A != nil {
		for _, val := range si.A {
			r = append(r, " (A ", strconv.Quote(val), ")")
		}
	}
	if si.B != "" {
		r = append(r, " (B ", strconv.Quote(si.B), ")")
	}
	r = append(r, ")")
	return strings.Join(r, "")
}

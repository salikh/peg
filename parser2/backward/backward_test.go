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

// This test gives an example of a simple test suite to develop, debug and
// tests a PEG grammar.
package backward

import (
	"regexp"
	"testing"

	log "github.com/golang/glog"
	"github.com/salikh/peg/parser2"
	"github.com/salikh/peg/tree"
)

var grammarSource = `
Top <- _ A* B*
A <- <"a"*> _
B <- <"b"*> _
_ <- [ \t\n\r]*
`
var grammar2 *parser2.Grammar

func init() {
	var err error
	grammar2, err = parser2.New(grammarSource, &parser2.ParserOptions{
		IgnoreUnconsumedTail: false,
		SkipEmptyNodes:       true,
	})
	if err != nil {
		log.Exitf("Error creating grammar [%s]: %s", grammarSource, err)
	}
}

type grammarTest struct {
	input string
	expr  string
	want  string
	err   string
}

var grammarTests = []grammarTest{
	{input: "aabb", expr: "A", want: "aa"},
	{input: "aabb", expr: "B", want: "bb"},
	{input: "aabb", expr: "B row", want: "1"},
	{input: "aabb", expr: "B col", want: "2"},
	{input: "aa\nbb", expr: "B row", want: "2"},
	{input: "aa\nbb", expr: "B col", want: "0"},
	{input: "aabbb", expr: "B pos", want: "2"},
	{input: "aabbb", expr: "B len", want: "3"},
	{input: "aabb", expr: "B", want: "bb"},
	{input: "aabb", expr: "B text", want: "bb"},
	{input: "aa abb b", expr: "B[0]", want: "bb"},
	{input: "aa abb b", expr: "B[1]", want: "b"},
	{input: "aa abb b", expr: "[1]", want: "a"},
	{input: "aa abb b", expr: "[2]", want: "bb"},
	{input: "aa abb b", expr: "[3]", want: "b"},
	{input: "aa a bb b", expr: "[3]", want: "b"},
	{input: "aa a bb b", expr: "A[1]", want: "a"},
	{input: "aa a bb b", expr: "A[0]", want: "aa"},
	{input: "aabbaa", err: `"a`},
	{input: "ccc", err: `"c`},
}

func TestForward(t *testing.T) {
	for _, tt := range grammarTests {
		t.Run(tt.input+"/"+tt.expr+tt.err, func(t *testing.T) {
			t.Logf("Grammar:\n%s\n---\n", grammarSource)
			t.Logf("Input:\n%s\n---\n", tt.input)
			result, err := grammar2.Parse(tt.input)
			if err != nil {
				t.Logf("Parse error: %s", err)
				if tt.err == "" {
					t.Errorf("Parse(%q) returns error %s, want success", tt.input, err)
					return
				}
				if ok, _ := regexp.MatchString(tt.err, err.Error()); !ok {
					t.Errorf("Parse(%q) returns error %s, want /%s/", tt.input, err, tt.err)
					return
				}
				return
			}
			result.ComputeContent()
			// Non-error case.
			t.Logf("Parse tree:\n%s\n---\n", result.Tree)
			val, err := tree.Extract(result.Tree, tt.expr)
			t.Logf("Extracted %s: %q", tt.expr, val)
			if err != nil {
				t.Errorf("Error while extracting %s: %s", tt.expr, err)
				return
			}
			if val != tt.want {
				t.Errorf("Parse(%q) %s returns %s, want %s", tt.input, tt.expr, val, tt.want)
				return
			}
		})
	}
}

func TestBackward(t *testing.T) {
	for _, tt := range grammarTests {
		t.Run(tt.input+"/"+tt.expr+tt.err, func(t *testing.T) {
			t.Logf("Grammar:\n%s\n---\n", grammarSource)
			t.Logf("Input:\n%s\n---\n", tt.input)
			result, err := grammar2.ParseBackward(tt.input)
			if err != nil {
				t.Logf("Parse error: %s", err)
				if tt.err == "" {
					t.Errorf("Parse(%q) returns error %s, want success", tt.input, err)
					return
				}
				if ok, _ := regexp.MatchString(tt.err, err.Error()); !ok {
					t.Errorf("Parse(%q) returns error %s, want /%s/", tt.input, err, tt.err)
					return
				}
				return
			}
			result.ComputeContent()
			// Non-error case.
			t.Logf("Parse tree:\n%s\n---\n", result.Tree)
			val, err := tree.Extract(result.Tree, tt.expr)
			t.Logf("Extracted %s: %q", tt.expr, val)
			if err != nil {
				t.Errorf("Error while extracting %s: %s", tt.expr, err)
				return
			}
			if val != tt.want {
				t.Errorf("Parse(%q) %s returns %s, want %s", tt.input, tt.expr, val, tt.want)
				return
			}
		})
	}
}

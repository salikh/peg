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

package template

import (
	"regexp"
	"testing"
)

type test struct {
	input string
	pos   int
	w     int
	err   string
}

func testOneHandler(t *testing.T, h handler, name string, tests []test) {
	for _, tt := range tests {
		r := &Result{
			Source: tt.input,
			Memo:   make(map[int]map[int]*Node),
		}
		r.NodeStack.Push(&Node{Label: "top"})
		w, terr := h(r, tt.pos)
		if terr == nil && tt.err != "" {
			t.Errorf("%s(%q,%d) returns success, want error %q",
				name, tt.input, tt.pos, tt.err)
			continue
		}
		if terr != nil && tt.err == "" {
			t.Errorf("%s(%q,%d) returns error %q, want success",
				name, tt.input, tt.pos, terr)
			continue
		}
		if tt.err != "" {
			re, err := regexp.Compile(tt.err)
			if err != nil {
				t.Errorf("Error in regexp %q: %s", tt.err, err)
				continue
			}
			if terr != nil && !re.Match([]byte(terr.Error())) {
				t.Errorf("%s(%q,%d) returns error %q, want %q",
					name, tt.input, tt.pos, terr, tt.err)
				continue
			}
		}
		if w != tt.w {
			t.Errorf("%s(%q,%d) returns w=%d, want %d",
				name, tt.input, tt.pos, w, tt.w)
		}
	}
}

var literalTests = []test{
	{"abc", 0, 3, ""},
	{"", 0, 0, "expecting.*got"},
	{"abc", 1, 0, "expecting.*got"},
	{"abd", 0, 0, "expecting.*got"},
	{"abc ", 0, 3, ""},
	{"abcd", 0, 3, ""},
	{"xabcyz", 1, 3, ""},
}

func TestLiteralHandler(t *testing.T) {
	testOneHandler(t, LiteralHandler, "LiteralHandler", literalTests)
}

func TestCharClassHandler(t *testing.T) {
	var tests = []test{
		{"", 0, 0, "expecting.*got EOF"},
		{"abc", 3, 0, "expecting.*got EOF"},
		{"x", 0, 0, "character.*does not match class"},
		{"abc", 1, 0, "character.*does not match class"},
		{" bc", 0, 1, ""},
		{"a d", 1, 1, ""},
		{"a\nd", 1, 1, ""},
		{"a\td", 1, 1, ""},
		{"ю\td", 2, 1, ""},
		{"日\td", 3, 1, ""},
	}
	testOneHandler(t, CharClassHandler, "CharClassHandler", tests)
}

func TestCharClassAlnumHandler(t *testing.T) {
	var tests = []test{
		{"", 0, 0, "expecting.*got EOF"},
		{"abc", 3, 0, "expecting.*got EOF"},
		{"x", 0, 1, ""},
		{"abc", 1, 1, ""},
		{" bc", 0, 0, "character.*does not match class"},
		{" bc", 1, 1, ""},
		{"a\nd", 1, 0, "character.*does not match class"},
		{"a\td", 1, 0, "character.*does not match class"},
		{"ю\td", 0, 2, ""},
		{"日\td", 0, 3, ""},
	}
	testOneHandler(t, CharClassAlnumHandler, "CharClassHandler", tests)
}

func TestStarHandler(t *testing.T) {
	var tests = []test{
		{"abc", 0, 0, ""},
		{"", 0, 0, ""},
		{" abc ", 0, 1, ""},
		{"  abc ", 0, 2, ""},
		{"   abc ", 0, 3, ""},
		{"\n\n\n\nabc ", 0, 4, ""},
		{"ab\n \tcd", 2, 3, ""},
		{"xabcyz", 3, 0, ""},
	}
	testOneHandler(t, StarHandler, "StarHandler", tests)
}

func TestGroupHandler(t *testing.T) {
	var tests = []test{
		{"abc", 0, 3, ""},
		{"abc   ", 0, 6, ""},
		{"", 0, 0, "expecting.*got"},
		{"abd", 0, 0, "expecting.*got"},
		{" abd", 1, 0, "expecting.*got"},
		{" abc ", 0, 5, ""},
		{"abcd", 0, 3, ""},
		{"   abc   def", 0, 9, ""},
		{"   abc   def", 3, 6, ""},
		{"xabcyz", 1, 3, ""},
	}
	testOneHandler(t, GroupHandler, "GroupHandler", tests)
}

func TestRuleHandler(t *testing.T) {
	testOneHandler(t, RuleHandler, "RuleHandler", literalTests)
	testOneHandler(t, RuleHandler, "RuleHandler", literalTests)
}

func TestPredicateHandler(t *testing.T) {
	//predicateNegative = false
	var posTests = []test{
		{"abc", 0, 0, ""},
		{"abc   ", 0, 0, ""},
		{"", 0, 0, "expecting.*got"},
		{"abd", 0, 0, "expecting.*got"},
		{" abd", 1, 0, "expecting.*got"},
		{" abc ", 0, 0, ""},
		{"abcd", 0, 0, ""},
		{"   abc   def", 0, 0, ""},
		{"   abc   def", 3, 0, ""},
		{"xabcyz", 1, 0, ""},
	}
	testOneHandler(t, PredicateHandler, "PredicateHandler(pos)", posTests)
	/*
		//TODO(salikh): test with negative predicate.
		predicateNegative = true
		var negTests = []test{
			{"abc", 0, 0, "negative predicate matched"},
			{"abc   ", 0, 0, "negative predicate matched"},
			{"", 0, 0, ""},
			{"abd", 0, 0, ""},
			{" abd", 1, 0, ""},
			{" abc ", 0, 0, "negative predicate matched"},
			{"abcd", 0, 0, "negative predicate matched"},
			{"   abc   def", 0, 0, "negative predicate matched"},
			{"   abc   def", 3, 0, "negative predicate matched"},
			{"xabcyz", 1, 0, "negative predicate matched"},
		}
		testOneHandler(t, PredicateHandler, "PredicateHandler(neg)", negTests)
	*/
}

func TestChoiceHandler(t *testing.T) {
	var tests = []test{
		{"abc", 0, 3, ""},
		{"abc   ", 0, 3, ""},
		{"", 0, 0, ""},
		{"abd", 0, 0, ""},
		{" abd", 1, 0, ""},
		{" abc ", 0, 1, ""},
		{"abcd", 0, 3, ""},
		{"   abc   def", 0, 3, ""},
		{"   abc   def", 3, 3, ""},
		{"xabcyz", 1, 3, ""},
	}
	testOneHandler(t, ChoiceHandler, "ChoiceHandler", tests)
}

func TestDotHandler(t *testing.T) {
	var tests = []test{
		{"a", 0, 1, ""},
		{" ", 0, 1, ""},
		{"", 0, 0, "expected char.*got EOF"},
		{"日", 0, 3, ""},
		{"日", 1, 1, "invalid"},
		{"日", 2, 1, "invalid"},
		{"ю", 0, 2, ""},
		{"ю", 1, 1, "invalid"},
		{"a", 1, 0, "expected char.*got EOF"},
		{"abc", 1, 1, ""},
	}
	testOneHandler(t, DotHandler, "DotHandler", tests)
}

func TestPlusHandler(t *testing.T) {
	var tests = []test{
		{" ", 0, 1, ""},
		{"  ", 0, 2, ""},
		{"", 0, 0, "expect.*char.*got EOF"},
		{"日", 0, 0, "char.*does not match"},
		{"日", 1, 1, "invalid"},
		{"日", 2, 1, "invalid"},
		{" \n\r\t", 0, 4, ""},
		{"  abc", 1, 1, ""},
	}
	testOneHandler(t, PlusHandler, "PlusHandler", tests)
}

func TestQuestionHandler(t *testing.T) {
	var tests = []test{
		{" ", 0, 1, ""},
		{"  ", 0, 1, ""},
		{"", 0, 0, ""},
		{"日", 0, 0, ""},
		// QuestionHandler does not detect invalid utf8
		{"日", 1, 0, ""},
		{"日", 2, 0, ""},
		{" \n\r\t", 0, 1, ""},
		{" \n\r\t", 2, 1, ""},
		{"  abc", 1, 1, ""},
	}
	testOneHandler(t, QuestionHandler, "QuestionHandler", tests)
}

func TestCaptureHandlers(t *testing.T) {
	var tests = []struct {
		input   string
		pos     int
		capture string
		err     string
	}{
		{"abc", 0, "abc", ""},
		{" abc", 0, "abc", ""},
		{"abc ", 0, "abc", ""},
		{" abc ", 0, "abc", ""},
		{"\n \tabc\t \n", 0, "abc", ""},
		{"\n \tabc\t \n", 3, "abc", ""},
		{"abc", 1, "", "expect.*got"},
	}
	for _, tt := range tests {
		r := &Result{
			Source: tt.input,
			Memo:   make(map[int]map[int]*Node),
		}
		node := &Node{Label: "top"}
		r.NodeStack.Push(node)
		_, terr := GroupHandler(r, tt.pos)
		if terr == nil && tt.err != "" {
			t.Errorf("GroupHandler(%q,%d) returns success, want error %q",
				tt.input, tt.pos, tt.err)
			continue
		}
		if terr != nil && tt.err == "" {
			t.Errorf("GroupHandler(%q,%d) returns error %q, want success",
				tt.input, tt.pos, terr)
			continue
		}
		if tt.err != "" {
			re, err := regexp.Compile(tt.err)
			if err != nil {
				t.Errorf("Error in regexp %q: %s", tt.err, err)
				continue
			}
			if terr != nil && !re.Match([]byte(terr.Error())) {
				t.Errorf("GroupHandler(%q,%d) returns error %q, want %q",
					tt.input, tt.pos, terr, tt.err)
				continue
			}
		}
		if node.Text != tt.capture {
			t.Errorf("GroupHandler(%q,%d) captures %q, want %q",
				tt.input, tt.pos, node.Text, tt.capture)
		}
	}
}

func TestParse(t *testing.T) {
	testHandler = GroupHandler
	tests := []struct {
		input   string
		capture string
		err     string
	}{
		{"abc", "abc", ""},
		{"  abc", "abc", ""},
		{"abc  ", "abc", ""},
		{"  abc  ", "abc", ""},
		{"  abd  ", "", "expecting.*got"},
	}
	for _, tt := range tests {
		r, terr := Parse(tt.input)
		if terr == nil && tt.err != "" {
			t.Errorf("Parse(%q) returns success, want error %q",
				tt.input, tt.err)
			continue
		}
		if terr != nil && tt.err == "" {
			t.Errorf("Parse(%q) returns error %q, want success",
				tt.input, terr)
			continue
		}
		if tt.err != "" {
			re, err := regexp.Compile(tt.err)
			if err != nil {
				t.Errorf("Error in regexp %q: %s", tt.err, err)
				continue
			}
			if terr != nil && !re.Match([]byte(terr.Error())) {
				t.Errorf("Parse(%q) returns error %q, want %q",
					tt.input, terr, tt.err)
			}
			continue
		}
		if r.Tree == nil {
			t.Errorf("Parse(%q) returns nil r.Tree, want non-nil",
				tt.input)
			continue
		}
		if r.Tree.Text != tt.capture {
			t.Errorf("GroupHandler(%q) captures %q, want %q",
				tt.input, r.Tree.Text, tt.capture)
		}
	}
}

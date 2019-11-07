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

// Package template provides the code templates that are emitted to generate
// the parser.
//
// Each template handler is tested independently in isolation.
//
// This package as given implements the following grammar:
//
//   Literal <- "abc"
//   CharClass <- [ \n\r\t]
//   Star <- CharClass *
//   Group <- Star < Literal > Star
//   Predicate <- & Group
//   ... etc.
//
package template

import (
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/salikh/peg/parser"
)

type Node = parser.Node

type NodeStack []*Node

// Result encapsulates one parse result.
type Result struct {
	Source string
	Memo   map[int]map[int]*Node
	Level  int
	// Final AST.
	Tree *parser.Node
	NodeStack
}

func (s *NodeStack) Push(n *Node) {
	*s = append(*s, n)
}

func (s *NodeStack) Pop() *Node {
	last := len(*s) - 1
	n := (*s)[last]
	*s = (*s)[:last]
	return n
}

func (r *Result) TopNode() *Node {
	last := len(r.NodeStack) - 1
	if last < 0 {
		panic("Internal error: no top node")
	}
	return r.NodeStack[last]
}

func (r *Result) Attach(n *Node) {
	if n.Text == "" && n.Start == 0 && len(n.Children) == 0 && len(n.Annotations) == 0 &&
		len(r.NodeStack) > 0 {
		// Heuristic: do not attach the nodes without any useful annotations,
		// text or children. Note, that captured text may be empty, but n.Start is
		// non-zero in that case.
		return
	}
	last := len(r.NodeStack) - 1
	if last < 0 {
		//log.Infof("Attaching root node %v", n)
		if r.Tree != nil {
			panic("Internal error: Attempting to attach root node twice")
		}
		// attaching the top node
		r.Tree = (*parser.Node)(n)
		return
	}
	//log.Infof("Attaching node %v to %v", n, r.NodeStack[last])
	//children := &r.NodeStack[last].Children
	//*children = append(*children, n)
	r.NodeStack[last].Children = append(r.NodeStack[last].Children, (*parser.Node)(n))
	//log.Infof("node = %s, attached = %s", r.NodeStack[last], n)
}

//------------------------------------------------------------------------------
// The following handlers are not copied to the generated parsers verbatim.
// Instead, they are used as templates, with the variables replaced with the
// data from the actual grammar. The below code is used for testing.

var labels = []string{"<nil>", "AbcLiteral", "SpaceChar", "Space", "Abc"}

// literal defines the literal string expected by LiteralHandler. It is replaced
// by the actual values from a grammar by the parser generator.
const literal = "abc"

// LiteralHandler is a template code for literal handlers in the generated parser.
func LiteralHandler(r *Result, pos int) (int, error) {
	// LiteralHandler
	if len(r.Source)-pos < len(literal) {
		return 0, fmt.Errorf("expecting %q, got %q",
			literal, r.Source[pos:])
	}
	next := r.Source[pos : pos+len(literal)]
	if next != literal {
		return 0, fmt.Errorf("expecting %q, got %q", literal, next)
	}
	return len(literal), nil
}

// charClassMap defines the char class definition for the CharClassHandler. It is replaced
// by the actual definition from a grammar by the parser generator.
var charClassMap = map[rune]bool{' ': true, '\n': true, '\t': true, '\r': true}

const charClassSource = " \n\t\r"
const charClassNegated = false

func CharClassHandler(r *Result, pos int) (int, error) {
	// CharClassHandler
	c, w := utf8.DecodeRuneInString(r.Source[pos:])
	if w == 0 {
		return 0, fmt.Errorf("expecting char, got EOF")
	}
	if c == utf8.RuneError {
		return w, fmt.Errorf("invalid utf8: %q", r.Source[pos:pos+w])
	}
	if charClassNegated == charClassMap[c] {
		return 0, fmt.Errorf("character %q does not match class [%s]",
			c, charClassSource)
	}
	return w, nil
}

func CharClassAlnumHandler(r *Result, pos int) (int, error) {
	c, w := utf8.DecodeRuneInString(r.Source[pos:])
	if w == 0 {
		return 0, fmt.Errorf("expecting char, got EOF")
	}
	if c == utf8.RuneError {
		return w, fmt.Errorf("invalid utf8: %q", r.Source[pos:pos+w])
	}
	if !unicode.IsLetter(c) && !unicode.IsNumber(c) {
		return 0, fmt.Errorf("character %q does not match class [:alnum]]", c)
	}
	return w, nil
}

func StarHandler(r *Result, pos int) (int, error) {
	// StarHandler
	ww := 0
	for w, err := CharClassHandler(r, pos); err == nil && w > 0; w, err = CharClassHandler(r, pos+ww) {
		ww += w
	}
	return ww, nil
}

func GroupHandler(r *Result, pos int) (int, error) {
	// GroupHandler
	// COPY
	ww := 0
	var w int
	var err error
	// IGNORE
	w, err = StarHandler(r, pos+ww)
	ww += w
	if err != nil {
		return ww, err
	}
	w, err = CaptureStartHandler(r, pos+ww)
	ww += w
	if err != nil {
		return ww, err
	}
	w, err = CaptureStartHandler(r, pos+ww)
	ww += w
	if err != nil {
		return ww, err
	}
	// START BLOCK
	w, err = LiteralHandler(r, pos+ww)
	ww += w
	if err != nil {
		return ww, err
	}
	// END BLOCK
	// IGNORE
	w, err = CaptureEndHandler(r, pos+ww)
	ww += w
	if err != nil {
		return ww, err
	}
	w, err = StarHandler(r, pos+ww)
	ww += w
	if err != nil {
		return ww, err
	}
	// COPY
	return ww, nil
}

func RuleHandler(r *Result, pos int) (int, error) {
	// RuleHandler
	return apply(r, pos, LiteralHandler, 1)
}

const predicateNegative = false

func PredicateHandler(r *Result, pos int) (int, error) {
	// PredicateHandler
	_, err := GroupHandler(r, pos)
	if predicateNegative == (err != nil) {
		return 0, nil
	}
	if err == nil {
		return 0, fmt.Errorf("negative predicate matched")
	}
	return 0, err
}

func CaptureStartHandler(r *Result, pos int) (int, error) {
	// CaptureStartHandler
	if r.TopNode() == nil {
		return 0, fmt.Errorf("internal error, cannot start capture without a node")
	}
	r.TopNode().Start = pos
	return 0, nil
}

func CaptureEndHandler(r *Result, pos int) (int, error) {
	// CaptureEndHandler
	if r.TopNode() == nil {
		return 0, fmt.Errorf("internal error, cannot end capture without a node")
	}
	r.TopNode().Text = r.Source[r.TopNode().Start:pos]
	return 0, nil
}

func ChoiceHandler(r *Result, pos int) (int, error) {
	// ChoiceHandler
	w, err := LiteralHandler(r, pos)
	if err != nil {
		w, err = StarHandler(r, pos)
	}
	return w, err
}

func DotHandler(r *Result, pos int) (int, error) {
	// DotHandler
	if pos == len(r.Source) {
		return 0, fmt.Errorf("expected character, got EOF")
	}
	c, w := utf8.DecodeRuneInString(r.Source[pos:])
	if c == utf8.RuneError {
		return w, fmt.Errorf("invalid utf8: %q", r.Source[pos:pos+w])
	}
	return w, nil
}

func PlusHandler(r *Result, pos int) (int, error) {
	// PlusHandler
	ww, err := CharClassHandler(r, pos)
	if err != nil {
		return ww, err
	}
	for w, err := CharClassHandler(r, pos+ww); err == nil && w > 0; w, err = CharClassHandler(r, pos+ww) {
		ww += w
	}
	return ww, nil
}

func QuestionHandler(r *Result, pos int) (int, error) {
	// QuestionHandler
	w, err := CharClassHandler(r, pos)
	if err != nil {
		return 0, nil
	}
	return w, nil
}

type handler func(r *Result, pos int) (int, error)

func apply(r *Result, pos int, h handler, hi int) (int, error) {
	r.Level++
	defer func() { r.Level-- }()
	//log.Infof("%d> applying rule %q at pos %d", r.Level, ru.rhs, pos)
	memo, ok := r.Memo[pos]
	if !ok {
		// This is indexed by handler index (hi)
		memo = make(map[int]*Node)
		r.Memo[pos] = memo
	}
	n := memo[hi]
	if n != nil && n.Err == nil {
		//log.Infof("%d> cached success w%d", r.Level, w)
		r.Attach(n)
		return n.Len, nil
	}
	if n != nil && n.Err != nil {
		//log.Infof("%d> cached fail", r.Level)
		return n.Len, n.Err
	}
	n = &Node{Label: labels[hi]}
	r.NodeStack.Push(n)
	w, err := h(r, pos)
	if err != nil {
		//log.Infof("%d> fail w%d", r.Level, w+w1)
		n := r.NodeStack.Pop()
		n.Len = w
		n.Err = err
		memo[hi] = n
		return n.Len, err
	}
	n = r.NodeStack.Pop()
	n.Len = w
	n.Pos = pos
	memo[hi] = n
	//log.Infof("%d> success w%d", r.Level, w)
	r.Attach(n)
	return w, nil
}

// testHandler can be overridden by tests to facilitate testing of Parse.
var testHandler = LiteralHandler

func Parse(source string) (*Result, error) {
	r := &Result{
		Source:    source,
		Memo:      make(map[int]map[int]*Node),
		NodeStack: make([]*Node, 0, 10),
	}
	w, err := apply(r, 0, testHandler, 4)
	if err != nil {
		return r, err
	}
	if w == 0 && len(source) > 0 {
		return r, fmt.Errorf("grammar matched 0 characters")
	}
	if w != len(source) {
		return r, fmt.Errorf("some characters remain unconsumed: %q", source[w:])
	}
	return r, nil
}

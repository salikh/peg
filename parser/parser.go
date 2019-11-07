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

package parser

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	log "github.com/golang/glog"
	"github.com/salikh/peg/parser/charclass"
)

type Grammar interface {
	// Parse parses the source text according to the grammar.
	Parse(source string) (*Result, error)
	// Source returns the text source of the grammar description.
	Source() string
	// Generate composes the Go source for the parser.
	// TODO(salikh): Move generator out to a separate package.
	//Generate() (string, error)
}

type handler func(r *Result, pos int) (int, error)

type rule struct {
	label    string
	rhs      string
	handlers []handler
	isTop    bool
}

type grammar struct {
	// persistent part
	gsource   string
	topRule   string
	rules     map[string]*rule
	lateRules []string
	utf8Used  bool
}

type NodeStack []*Node

// Result encapsulates one parse result.
type Result struct {
	G      *grammar
	Source string
	Memo   map[int]map[*rule]*Node
	Level  int
	// Final AST.
	Tree *Node
	NodeStack
}

// TODO: rename Child to First
func (n *Node) Child(label string) *Node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if c.Label == label {
			return c
		}
	}
	return nil
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
		log.Exitf("Internal error: no top node")
	}
	return r.NodeStack[last]
}

func (r *Result) Attach(n *Node) {
	if n.Text == "" && n.Start == 0 && len(n.Children) == 0 && len(n.Annotations) == 0 {
		// Heuristic: do not attach the nodes without any useful annotations either
		// text or children. Note, that captured text may be empty, but n.Start is
		// non-zero in that case.
		return
	}
	last := len(r.NodeStack) - 1
	if last < 0 {
		//log.Infof("Attaching root node %v", n)
		if r.Tree != nil {
			log.Exitf("Internal error: Attempting to attach root node twice")
		}
		// attaching the top node
		r.Tree = n
		return
	}
	//log.Infof("Attaching node %v to %v", n, r.NodeStack[last])
	//children := &r.NodeStack[last].Children
	//*children = append(*children, n)
	r.NodeStack[last].Children = append(r.NodeStack[last].Children, n)
	//log.Infof("node = %s, attached = %s", r.NodeStack[last], n)
}

// New creates a new grammar.
func New(gsource string) (Grammar, error) {
	scanner := bufio.NewScanner(bytes.NewReader([]byte(gsource)))
	g := &grammar{
		gsource: gsource,
		rules:   make(map[string]*rule),
	}
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " \t\n")
		if line == "" || line[0] == '#' {
			// ignore comments and empty lines
			continue
		}
		err := g.addRule(line)
		if err != nil {
			return nil, err
		}
	}
	if g.topRule == "" {
		return nil, fmt.Errorf("grammar must have at least one rule")
	}
	g.rules[g.topRule].isTop = true
	for _, rule := range g.lateRules {
		if _, ok := g.rules[rule]; !ok {
			return nil, fmt.Errorf("grammar rule %s undefined", rule)
		}
	}
	return g, nil
}

func (g *grammar) Source() string {
	return g.gsource
}

func (g *grammar) addRule(line string) error {
	parts := strings.Split(line, "<-")
	if len(parts) != 2 {
		return fmt.Errorf("expecting to have exactly 1 '<-' per grammar line")
	}
	name := strings.Trim(parts[0], " ")
	rhs := strings.Trim(parts[1], " ")
	if g.topRule == "" {
		g.topRule = name
	}
	ru, err := g.createRule(name, rhs)
	if err != nil {
		return err
	}
	g.rules[name] = ru
	return nil
}

// ParseEscape parses one C-style escape character from the beginning of string s.
// It returnes the parsed rune, the number of consumed bytes and error.
func ParseEscape(s string) (rune, int, error) {
	c, w := utf8.DecodeRuneInString(s)
	switch c {
	case 'n':
		c = '\n'
	case 'r':
		c = '\r'
	case 't':
		c = '\t'
	case '\'', '"', '\\':
		// Do nothing, c has the appropriate value already.
	default:
		return c, w, fmt.Errorf("invalid escape sequence: \"\\%c\"", c)
	}
	return c, w, nil
}

func parseSingleQuoted(s string, q rune) (string, int, error) {
	var i int
	for i = 0; i < len(s) && s[i] != byte(q); i++ {
	}
	if i == len(s) {
		return "", 0, fmt.Errorf("unclosed quote %c", q)
	}
	val := s[0:i]
	return val, i + 1, nil
}

func parseDoubleQuoted(s string, q rune) (string, int, error) {
	//log.Infof("parseDoubleQuoted(%q,%q)", s, q)
	var r []rune
	for i, w := 0, 0; i < len(s); i += w {
		var c rune
		c, w = utf8.DecodeRuneInString(s[i:])
		switch c {
		case q:
			//log.Infof("returning %q, %d, nil", string(r), i+w)
			return string(r), i + w, nil
		case '\\':
			i += w
			var err error
			c, w, err = ParseEscape(s[i:])
			if err != nil {
				return string(r), i + w, err
			}
			//log.Infof("parsed escape %q", c)
		}
		r = append(r, c)
		if w == 0 {
			return "", 0, fmt.Errorf("internal error in parseQuoted, w = 0")
		}
	}
	return "", 0, fmt.Errorf("unterminated quoted literal: %c%s", q, s)
}

func parseBrackets(s string, qo, qc rune) (string, int, error) {
	//log.Infof("parseBrackets(%q,%q,%q)", s, qo, qc)
	var r []rune
	depth := 0
	for i, w := 0, 0; i < len(s); i += w {
		var c rune
		c, w = utf8.DecodeRuneInString(s[i:])
		//log.Infof("in brackets: %q", c)
		switch c {
		case qo:
			depth++
		case qc:
			if depth == 0 {
				//log.Infof("returning %q, %d, nil", string(r), i+w)
				return string(r), i + w, nil
			}
			depth--
		case '\\':
			i += w
			var err error
			c, w, err = ParseEscape(s[i:])
			if err != nil {
				return string(r), i + w, err
			}
			//log.Infof("parsed escape:%q", c)
		}
		r = append(r, c)
		if w == 0 {
			return "", 0, fmt.Errorf("internal error in parseBrackets, w = 0")
		}
	}
	return "", 0, fmt.Errorf("unterminated bracket, expecting %q, got '%s'", qc, s)
}

func parseParens(s string) (string, int, error) {
	level := 1
	for i, w := 0, 0; i < len(s); i += w {
		var c rune
		c, w = utf8.DecodeRuneInString(s[i:])
		switch c {
		case '(':
			level++
		case ')':
			level--
			if level == 0 {
				return s[0:i], i + w, nil
			}
		case '\'':
			_, wq, err := parseSingleQuoted(s[i+w:], c)
			if err != nil {
				return s[0 : i+w+wq], i + w + wq, err
			}
			w += wq
		case '"':
			_, wq, err := parseDoubleQuoted(s[i+w:], c)
			if err != nil {
				return s[0 : i+w+wq], i + w + wq, err
			}
			//log.Infof("parsed literal %q", val)
			w += wq
		}
	}
	return s, len(s), fmt.Errorf("reached end of line while expecting ')'")
}

func parseIdent(s string) (string, int, error) {
	for i, w := 0, 0; i < len(s); i += w {
		var c rune
		c, w = utf8.DecodeRuneInString(s[i:])
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' {
			if i == 0 {
				return "", 0, fmt.Errorf("identifier expected in %q", s)
			}
			// Found the end of the identifier.
			return s[0:i], i, nil
		}
	}
	if len(s) == 0 {
		return "", 0, fmt.Errorf("identifier expected in %q", s)
	}
	// The whole string was consumed as an identifier
	return s, len(s), nil
}

func inc(label string) string {
	return label + "I"
}

func (g *grammar) createRule(label, rhs string) (*rule, error) {
	//log.Infof("createRule(%q)", rhs)
	var handlers []handler
	captureCount := 0
	var predChar rune
	for i, w := 0, 0; i < len(rhs); i += w {
		var c rune
		c, w = utf8.DecodeRuneInString(rhs[i:])
		//log.Infof("c = %q", c)
		switch c {
		case '\'':
			val, wq, err := parseSingleQuoted(rhs[i+w:], c)
			if err != nil {
				return nil, err
			}
			handlers = append(handlers, g.newLiteralHandler(val))
			w += wq
		case '"':
			val, wq, err := parseDoubleQuoted(rhs[i+w:], c)
			if err != nil {
				return nil, err
			}
			//log.Infof("parsed literal %q", val)
			handlers = append(handlers, g.newLiteralHandler(val))
			w += wq
		case '<':
			switch captureCount {
			case 1:
				return nil, fmt.Errorf("capture already started")
			case 2:
				return nil, fmt.Errorf("only one capture group allowed")
			}
			captureCount++
			handlers = append(handlers, g.newCaptureHandler(true))
		case '>':
			switch captureCount {
			case 0:
				return nil, fmt.Errorf("capture not started")
			case 2:
				return nil, fmt.Errorf("capture already ended")
			}
			captureCount++
			handlers = append(handlers, g.newCaptureHandler(false))
		case ')':
			return nil, fmt.Errorf("unexpected ')'")
		case '(':
			val, wp, err := parseParens(rhs[i+w:])
			if err != nil {
				return nil, err
			}
			t, err := g.createRule(label+"_group", val)
			if err != nil {
				return nil, err
			}
			handlers = append(handlers, g.newGroupHandler(t.handlers))
			w += wp
		case '[':
			val, wp, err := parseBrackets(rhs[i+w:], '[', ']')
			if err != nil {
				return nil, err
			}
			h, err := g.newCharClassHandler(val)
			if err != nil {
				return nil, err
			}
			handlers = append(handlers, h)
			w += wp
		case ' ', '\t':
			continue
			// ignore
		case '.':
			handlers = append(handlers, g.newDotHandler())
		case '*':
			if len(handlers) == 0 {
				return nil, fmt.Errorf("'*' needs an expr before it, none found")
			}
			ii := len(handlers) - 1
			handlers[ii] = g.newStarHandler(handlers[ii])
		case '+':
			if len(handlers) == 0 {
				return nil, fmt.Errorf("'+' needs an expr before it, none found")
			}
			ii := len(handlers) - 1
			handlers[ii] = g.newPlusHandler(handlers[ii])
		case '?':
			if len(handlers) == 0 {
				return nil, fmt.Errorf("'?' needs an expr before it, none found")
			}
			ii := len(handlers) - 1
			handlers[ii] = g.newQuestionHandler(handlers[ii])
		case '/':
			//log.Infof("remaining rhs %q", rhs[i+w:])
			ru, err := g.createRule(inc(label), rhs[i+w:])
			if err != nil {
				return nil, err
			}
			handlers = []handler{g.newChoiceHandler(
				g.newGroupHandler(handlers),
				g.newGroupHandler(ru.handlers))}
			//log.Infof("now handlers = %v", handlers)
			w = len(rhs) - i
		case '!', '&':
			predChar = c
			continue
		default:
			if unicode.IsLetter(c) || c == '_' {
				val, wi, err := parseIdent(rhs[i:])
				if err != nil {
					return nil, err
				}
				handlers = append(handlers, g.newRuleHandler(val))
				g.lateRules = append(g.lateRules, val)
				w = wi
				break
			}
			return nil, fmt.Errorf("unrecognized character in grammar: %q", c)
		}
		if predChar == 0 {
			continue
		}
		//log.Infof("handlers = %v", handlers)
		l := len(handlers) - 1
		if l < 0 {
			return nil, fmt.Errorf("no handler to predicate on")
		}
		handlers[l] = g.newPredicateHandler(handlers[l], predChar == '!')
		predChar = 0
	}
	if predChar != 0 {
		return nil, fmt.Errorf("no handler to predicate on")
	}
	return &rule{
		label:    label,
		rhs:      rhs,
		handlers: handlers,
	}, nil
}

var unsuccess = fmt.Errorf("parse unsuccessful")

func (g *grammar) newRuleHandler(name string) handler {
	var ru *rule
	cached := false
	return func(r *Result, pos int) (int, error) {
		if !cached {
			var ok bool
			if ru, ok = g.rules[name]; !ok {
				return 0, fmt.Errorf("unknown rule: %s", name)
			}
			cached = true
		}
		return r.apply(ru, pos)
	}
}

func (g *grammar) newPredicateHandler(h handler, negative bool) handler {
	return func(r *Result, pos int) (int, error) {
		_, err := h(r, pos)
		if negative == (err != nil) {
			return 0, nil
		}
		if err == nil {
			return 0, fmt.Errorf("negative predicate matched")
		}
		return 0, err
	}
}

func (g *grammar) newCaptureHandler(start bool) handler {
	if start {
		return func(r *Result, pos int) (int, error) {
			r.TopNode().Start = pos
			return 0, nil
		}
	} else {
		return func(r *Result, pos int) (int, error) {
			n := r.TopNode()
			n.Text = r.Source[n.Start:pos]
			// Indicate that the capture was done.
			n.Start = 0
			return 0, nil
		}
	}
}

func (g *grammar) newChoiceHandler(h1, h2 handler) handler {
	return func(r *Result, pos int) (int, error) {
		save := r.TopNode().Children
		w, err := h1(r, pos)
		//log.Infof("applying first handler got w%d,%s", w, err)
		if err != nil {
			// reset the children
			r.TopNode().Children = save
			w, err = h2(r, pos)
			//log.Infof("applying second handler got w%d,%s", w, err)
		}
		return w, err
	}
}

func (g *grammar) newLiteralHandler(literal string) handler {
	return func(r *Result, pos int) (int, error) {
		//log.Infof("literal(%q)\n", literal)
		if len(r.Source)-pos < len(literal) {
			return 0, fmt.Errorf("expecting %q, got %q",
				literal, r.Source[pos:])
		}
		next := r.Source[pos : pos+len(literal)]
		if next != literal {
			return 0, fmt.Errorf("Expecting literal %q, got %q", literal, next)
		}
		// parse successful
		return len(literal), nil
	}
}

func (g *grammar) newDotHandler() handler {
	return func(r *Result, pos int) (int, error) {
		if pos == len(r.Source) {
			return 0, fmt.Errorf("expecting a char, got EOF at %d", pos)
		}
		_, w := utf8.DecodeRuneInString(r.Source[pos:])
		return w, nil
	}
}

func (g *grammar) newCharClassHandler(val string) (handler, error) {
	cc, err := charclass.Parse(val)
	if err != nil {
		return nil, fmt.Errorf("error parsing char class: %s", err)
	}
	if cc.Special != "" {
		special := cc.Special
		return func(r *Result, pos int) (int, error) {
			c, w := utf8.DecodeRuneInString(r.Source[pos:])
			if w == 0 {
				return 0, fmt.Errorf("expecting char, got EOF")
			}
			var match bool
			switch special {
			case "[:alnum:]":
				match = unicode.IsLetter(c) || unicode.IsDigit(c)
			case "IsLetter":
				match = unicode.IsLetter(c)
			case "IsNumber":
				match = unicode.IsNumber(c)
			case "IsSpace":
				match = unicode.IsSpace(c)
			case "IsLower":
				match = unicode.IsLower(c)
			case "IsUpper":
				match = unicode.IsUpper(c)
			case "IsPunct":
				match = unicode.IsPunct(c)
			case "IsPrint":
				match = unicode.IsPrint(c)
			case "IsGraphic":
				match = unicode.IsGraphic(c)
			case "IsControl":
				match = unicode.IsControl(c)
			}
			if cc.Negated {
				match = !match
			}
			if !match {
				return 0, fmt.Errorf("character %q does not match class %q", c, val)
			}
			return w, nil
		}, nil
	}
	return func(r *Result, pos int) (int, error) {
		c, w := utf8.DecodeRuneInString(r.Source[pos:])
		if w == 0 {
			return 0, fmt.Errorf("expecting char, got EOF")
		}
		match := false
		if cc.Map != nil {
			match = cc.Map[c]
		}
		if cc.RangeTable != nil {
			match = match || unicode.Is(cc.RangeTable, c)
		}
		if cc.Negated {
			match = !match
		}
		if !match {
			return 0, fmt.Errorf("character %q does not match class %q", c, val)
		}
		return w, nil
	}, nil
}

func (g *grammar) newGroupHandler(hh []handler) handler {
	return func(r *Result, pos int) (int, error) {
		//log.Infof("group(%v)\n", hh)
		ww := 0
		for _, h := range hh {
			w, err := h(r, pos+ww)
			if err != nil {
				return ww, err
			}
			ww += w
		}
		return ww, nil
	}
}

func (g *grammar) newStarHandler(h handler) handler {
	return func(r *Result, pos int) (int, error) {
		//log.Infof("star(%v) @ pos %d\n", h, pos)
		ww := 0
		// We want to get the longest match
		save := r.TopNode().Children
		for w, err := h(r, pos); err == nil && w > 0; w, err = h(r, pos+ww) {
			ww += w
			// Update the saved nodes in case of success
			save = r.TopNode().Children
		}
		// Reset the nodes appended by the last unsuccessful match.
		r.TopNode().Children = save
		// Star repetition always matches, in worst case it's zero length
		return ww, nil
	}
}

func (g *grammar) newPlusHandler(h handler) handler {
	return func(r *Result, pos int) (int, error) {
		//log.Infof("plus(%v) @ pos %d\n", h, pos)
		ww, err := h(r, pos)
		if err != nil {
			return ww, err
		}
		// We want to get the longest match
		for w, err := h(r, pos+ww); err == nil && w > 0; w, err = h(r, pos+ww) {
			ww += w
		}
		return ww, nil
	}
}

func (g *grammar) newQuestionHandler(h handler) handler {
	return func(r *Result, pos int) (int, error) {
		//log.Infof("question(%v)\n", h)
		w, err := h(r, pos)
		if err != nil {
			// Question option always matches, in worst case it's zero length
			return 0, nil
		}
		return w, nil
	}
}

func (g *grammar) Parse(source string) (*Result, error) {
	r := &Result{
		G:         g,
		Source:    source,
		Memo:      make(map[int]map[*rule]*Node),
		NodeStack: make([]*Node, 0, 10),
	}
	top := g.rules[g.topRule]
	w, err := r.apply(top, 0)
	if err != nil {
		return r, err
	}
	if w == 0 && len(source) > 0 {
		return r, errors.New("grammar matched 0 characters")
	}
	if w != len(source) {
		return r, fmt.Errorf("some characters remain unconsumed: %q",
			source[w:])
	}
	return r, nil
}

func (r *Result) apply(ru *rule, pos int) (int, error) {
	r.Level++
	defer func() { r.Level-- }()
	//log.Infof("%d> applying rule %q at pos %d", r.Level, ru.rhs, pos)
	memo, ok := r.Memo[pos]
	if !ok {
		memo = make(map[*rule]*Node)
		r.Memo[pos] = memo
	}
	n := memo[ru]
	if n != nil && n.Err == nil {
		//log.Infof("%d> cached success w%d", r.Level, w)
		r.Attach(n)
		return n.Len, nil
	}
	if n != nil && n.Err != nil {
		//log.Infof("%d> cached fail", r.Level)
		return n.Len, n.Err
	}
	n = &Node{Label: ru.label, Pos: pos}
	if ru.isTop {
		n.Annotations = map[string]string{"top": "1"}
	}
	r.NodeStack.Push(n)
	w := 0
	for _, h := range ru.handlers {
		w1, err := h(r, pos)
		if err != nil {
			//log.Infof("%d> fail w%d", r.Level, w+w1)
			n := r.NodeStack.Pop()
			n.Len = w + w1
			n.Err = err
			memo[ru] = n
			return n.Len, err
		}
		w += w1
		pos += w1
	}
	n = r.NodeStack.Pop()
	n.Len = w
	memo[ru] = n
	//log.Infof("%d> success w%d", r.Level, w)
	n.Len = pos - n.Pos
	r.Attach(n)
	return w, nil
}

// ComputeContent annotates the parse tree with pieces of original content
// and line/column positions in the original parser input.
func (r *Result) ComputeContent() {
	r.computeContent(r.Tree, 1, 0)
}

// Counts newlines in the string s and returns updated
// row and col numbers. Note: the col is counted in bytes, not runes.
func countRowCol(s string, row, col int) (int, int) {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			row++
			col = 0
		} else {
			col++
		}
	}
	return row, col
}

// Computes the content, row, col for the node and returns the update
// (row, col) pair.
func (r *Result) computeContent(n *Node, row, col int) (int, int) {
	pos := n.Pos
	n.Row = row
	n.Col = col
	for _, ch := range n.Children {
		piece := r.Source[pos:ch.Pos]
		row, col = countRowCol(piece, row, col)
		n.Content = append(n.Content, piece)
		row, col = r.computeContent(ch, row, col)
		pos = ch.Pos + ch.Len
	}
	piece := r.Source[pos : n.Pos+n.Len]
	row, col = countRowCol(piece, row, col)
	n.Content = append(n.Content, piece)
	return row, col
}

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

// Package parser2 is a clean reimplementation of the parser
// that uses a generated parser instead of hand-coded one.
// Feature-wise, parser and parser2 are supposed to be equivalent,
// but parser2 may be a bit ahead.
//
// TODO(salikh): Remove the bootstrap parser and rename this package
// to parser (without "2" suffix).
package parser2

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	log "github.com/golang/glog"
	"github.com/salikh/peg/parser"
	"github.com/salikh/peg/parser/charclass"
)

type ParserOptions struct {
	// IgnoreUnconsumedTail specifies whether unconsumed tail content should
	// be treated as an error or not. Default false value instructs the parser
	// to report unconsumed content as an error.
	IgnoreUnconsumedTail bool
	// SkipEmptyNodes specifies whether empty trivial nodes should be attached
	// to the syntax tree. Default false value instructs the parser to attach
	// all nodes. A node is considered trivially empty if it has no Node
	// children and its text capture is empty.
	SkipEmptyNodes bool
	// LongErrorMessage specifies whether to include the full content
	// into error messages. By default just a few first characters are included.
	LongErrorMessage bool
}

// New parses a PEG grammar source into a Grammar object.
// Optional ParserOptions specify the parser options.
func New(source string, options *ParserOptions) (*Grammar, error) {
	result, err := parse(source)
	if err != nil {
		return nil, fmt.Errorf("could not parse grammar source: %s", err)
	}
	grammar, err := convert(result.Tree)
	if err != nil {
		return nil, fmt.Errorf("internal error constructing semantic tree: %s", err)
	}
	if options != nil {
		grammar.ParserOptions = *options
	}
	for _, rule := range grammar.Rules {
		rule.handler, err = grammar.makeRHSHandler(rule.RHS)
		if err != nil {
			return nil, err
		}
		rule.backwardHandler, err = grammar.makeBackwardRHSHandler(rule.RHS)
		if err != nil {
			return nil, err
		}
	}
	return grammar, nil
}

// Grammar is parsing expression grammar (PEG).
type Grammar struct {
	// Rule is a dictionary of rules.
	Rules map[string]*Rule
	// RuleNames is the list of rule names in their original order.
	RuleNames []string
	// Source is the source text of the PEG grammar.
	Source string
	// ParserOptions specify the parser options.
	ParserOptions
}

// handler is the basic parse handler.
type handler func(r *Result, pos int) (int, error)

// Rule represent one PEG rule (Rule <- RHS).
type Rule struct {
	// Ident is the name of the rule, defined in LHS.
	Ident string
	// RHS is the rule's right-hand side.
	*RHS
	// handler is the parse handler of this rule.
	handler
	// backwardHandler is the backward parse handler of this rule.
	backwardHandler handler
}

// RHS is the right-hand side of one rule or the contents of parenthesized expression.
type RHS struct {
	// Terms is the RHS, top iteration over choices, and inside
	// iteration over concatenation of terms.
	Terms [][]*Term
}

// Term is one term. Note: Special characters in the grammar are handled
// specially and are not mapped to Term one-to-one. Instead, *?+ combine with
// the previous Term, and . is converted to a special CharClass.
type Term struct {
	Parens  *RHS
	NegPred *Term
	Pred    *Term
	*Special
	Capture *RHS
	*charclass.CharClass
	Literal string
	Ident   string
}

// Special is a term with a option or repeat special modifer (*?+).
type Special struct {
	*Term
	// Rune is one of '*' '?' '+'.
	Rune rune
}

func (g *Grammar) String() string {
	if g == nil {
		return "(nil)"
	}
	r := []string{"(Grammar "}
	for _, name := range g.RuleNames {
		rule := g.Rules[name]
		r = append(r, rule.String())
	}
	r = append(r, ")")
	return strings.Join(r, "")
}

func (rule *Rule) String() string {
	if rule == nil {
		return "(nil)"
	}
	r := []string{`(Rule text("`, rule.Ident, `") `}
	r = append(r, rule.RHS.String())
	r = append(r, ")")
	return strings.Join(r, "")
}

func (rhs *RHS) String() string {
	if rhs == nil {
		return "(nil)"
	}
	r := []string{"(RHS "}
	for _, terms := range rhs.Terms {
		r = append(r, "(Choice ")
		for _, term := range terms {
			r = append(r, term.String())
		}
		r = append(r, ")")
	}
	r = append(r, ")")
	return strings.Join(r, "")
}

func (t *Term) String() string {
	r := []string{"(Term"}
	if t.Parens != nil {
		r = append(r, " :Parens", t.Parens.String())
	}
	if t.NegPred != nil {
		r = append(r, " :NegPred", t.NegPred.String())
	}
	if t.Pred != nil {
		r = append(r, " :Pred", t.Pred.String())
	}
	if t.Capture != nil {
		r = append(r, " :Capture", t.Capture.String())
	}
	if t.CharClass != nil {
		r = append(r, ` :CharClass(`, strconv.Quote(t.CharClass.String()), `)`)
	}
	if t.Literal != "" {
		r = append(r, ` :Literal(`, strconv.Quote(t.Literal), `)`)
	}
	if t.Ident != "" {
		r = append(r, ` :Ident(`, strconv.Quote(t.Ident), `)`)
	}
	if t.Special != nil {
		r = append(r, ` :Special`, t.Special.String())
	}
	r = append(r, ")")
	return strings.Join(r, "")
}

func (s *Special) String() string {
	r := []string{"(Special"}
	q := strconv.QuoteRune(s.Rune)
	if s.Term != nil {
		r = append(r, s.Term.String())
	}
	r = append(r, ` :Rune("`, q[1:len(q)-1], `")`)
	r = append(r, ")")
	return strings.Join(r, "")
}

// convert converts the syntax tree of a grammar into the semantic
// grammar tree that is directly usable for parsing.
func convert(n *parser.Node) (*Grammar, error) {
	val, err := Construct(n, callback, &AccessorOptions{
		ErrorOnUnusedChild: true,
	})
	if err != nil {
		return nil, err
	}
	g, ok := val.(*Grammar)
	if !ok {
		return nil, fmt.Errorf("internal error: could convert type %s to *Grammar",
			reflect.TypeOf(val))
	}
	return g, nil
}

// The callback that is used to convert the syntax parse tree into
// the semantic tree.
func callback(label string, ca Accessor) (interface{}, error) {
	switch label {
	case "Grammar":
		rules := make(map[string]*Rule)
		var ruleNames []string
		var topRule string
		for _, rule := range ca.Get("Rule", []*Rule{}).([]*Rule) {
			if topRule == "" {
				topRule = rule.Ident
			}
			_, ok := rules[rule.Ident]
			if ok {
				return nil, fmt.Errorf("rule %s is duplicated", rule.Ident)
			}
			rules[rule.Ident] = rule
			ruleNames = append(ruleNames, rule.Ident)
		}
		return &Grammar{
			Rules:     rules,
			RuleNames: ruleNames,
		}, nil
	case "Rule":
		return &Rule{
			Ident: ca.String("Ident"),
			RHS:   ca.Get("RHS", &RHS{}).(*RHS),
		}, nil
	case "RHS":
		return &RHS{ca.Get("Terms", [][]*Term{}).([][]*Term)}, nil
	case "Terms":
		terms := ca.Get("Term", []*Term{}).([]*Term)
		// Convert the postfix Specials *?+ into subtrees.
		for i := 0; i < len(terms); i++ {
			if terms[i].Special == nil {
				// Anything but Special does not require special handling.
				continue
			}
			// Handle Special.
			if i == 0 {
				return nil, fmt.Errorf("Special character %q cannot be first in the rule",
					terms[i].Special.Rune)
			}
			// Move the previous term under Special.
			terms[i].Special.Term = terms[i-1]
			terms = append(terms[0:i-1], terms[i:]...)
			i--
		}
		return terms, nil
	case "Term":
		term := &Term{}
		switch ca.Child(0) {
		case "Parens":
			term.Parens = ca.Get("Parens", &RHS{}).(*RHS)
		case "NegPred":
			term.NegPred = ca.Get("NegPred", &Term{}).(*Term)
		case "Pred":
			term.Pred = ca.Get("Pred", &Term{}).(*Term)
		case "Capture":
			term.Capture = ca.Get("Capture", &RHS{}).(*RHS)
		case "CharClass":
			term.CharClass = ca.Get("CharClass", &charclass.CharClass{}).(*charclass.CharClass)
		case "Literal":
			raw := ca.String("Literal")
			var unquoted string
			if raw[0] == '"' {
				var err error
				unquoted, err = strconv.Unquote(raw)
				if err != nil {
					return nil, fmt.Errorf("error in strconv.Unquote(%q): %s", raw, err)
				}
			} else {
				unquoted = raw[1 : len(raw)-1]
			}
			term.Literal = unquoted
		case "Ident":
			term.Ident = ca.String("Ident")
		case "Special":
			special := ca.Get("Special", &Special{}).(*Special)
			if special.Rune == '.' {
				term.CharClass = &charclass.CharClass{
					Special: "[:any:]",
				}
			} else {
				term.Special = special
			}
		}
		return term, nil
	case "Special":
		c, _ := utf8.DecodeRuneInString(ca.Node().Text)
		return &Special{Rune: c}, nil
	case "Parens":
		return ca.GetTyped("RHS", &RHS{})
	case "NegPred":
		return ca.GetTyped("Term", &Term{})
	case "Pred":
		return ca.GetTyped("Term", &Term{})
	case "Capture":
		return ca.GetTyped("RHS", &RHS{})
	case "Literal":
		return ca.Node().Text, nil
	case "Ident":
		return ca.Node().Text, nil
	case "CharClass":

		return charclass.Parse(ca.Node().Text)
	case "EndOfLine":
		return nil, nil
	case "_":
		return nil, nil
	}
	return nil, fmt.Errorf("Unexpected label: %s", label)
}

type RowCol struct {
	Row int
	Col int
}

// Result is the object that parser uses to keep track of its state
// and to return results
type Result struct {
	// Grammar points to the grammar.
	*Grammar
	// Source is the input string to parse.
	Source string
	// Tree is the parsed syntax tree.
	Tree *parser.Node
	// Internal nodes.
	memo      map[int]map[*Rule]*parser.Node
	nodeStack NodeStack
	// fyiError helps to identify the issues with grammar
	fyiError error
	// rowCol helps to avoid recomputing row/col information for the same
	// locations. Maps position to row/col pair.
	rowCol map[int]RowCol
}

// Parse parses the input string accoring to the PEG grammar.
func (g *Grammar) Parse(input string) (*Result, error) {
	if g == nil {
		return nil, fmt.Errorf("nil grammar")
	}
	result := &Result{
		Grammar:   g,
		Source:    input,
		memo:      make(map[int]map[*Rule]*parser.Node),
		nodeStack: make(NodeStack, 0, 10),
		rowCol:    make(map[int]RowCol),
	}
	if len(g.RuleNames) == 0 {
		return nil, fmt.Errorf("invalid grammar without rules")
	}
	topRule := g.RuleNames[0]
	top, ok := g.Rules[topRule]
	if !ok {
		return nil, fmt.Errorf("invalid grammar with missing top rule %s", topRule)
	}
	w, err := result.apply(top, 0)
	if err != nil {
		return result, err
	}
	if w == 0 && len(input) > 0 {
		return result, fmt.Errorf("Grammar matched 0 characters: %s", result.fyiError)
	}
	if w != len(input) && !g.ParserOptions.IgnoreUnconsumedTail {
		errContent := input[w:]
		if !g.ParserOptions.LongErrorMessage && len(errContent) > 13 {
			errContent = errContent[:10] + "..."
		}
		return result, fmt.Errorf("some characers remain unconsumed: %q"+
			"\nPrevious error: %s",
			errContent, result.fyiError)
	}
	if result.Tree == nil {
		return nil, fmt.Errorf("internal error: no syntax tree. len(nodeStack) = %d",
			len(result.nodeStack))
	}
	return result, nil
}

func (r *Result) apply(ru *Rule, pos int) (int, error) {
	//log.Infof("%d> applying rule %q at pos %d", r.Level, ru.rhs, pos)
	memo, ok := r.memo[pos]
	if !ok {
		memo = make(map[*Rule]*parser.Node)
		r.memo[pos] = memo
	}
	n := memo[ru]
	if n != nil {
		if n.Err != nil {
			return n.Len, n.Err
		}
		r.Attach(n)
		return n.Len, nil
	}
	n = &parser.Node{Label: ru.Ident, Pos: pos}
	r.nodeStack.Push(n)
	w, hErr := ru.handler(r, pos)
	n = r.nodeStack.Pop()
	n.Len = w
	n.Err = hErr
	memo[ru] = n
	n.Len = w
	log.V(6).Infof("attaching %s", n.Label)
	r.Attach(n)
	return w, hErr
}

func (r *Result) TopNode() *parser.Node {
	last := len(r.nodeStack) - 1
	if last < 0 {
		log.Exitf("Internal error: no top node")
	}
	return r.nodeStack[last]
}

func (r *Result) Attach(n *parser.Node) {
	if r.Grammar.ParserOptions.SkipEmptyNodes &&
		(n.Text == "" && len(n.Children) == 0 && len(n.Annotations) == 0 &&
			len(n.TreeAnnotations) == 0 && len(r.nodeStack) > 0) {
		// Heuristic: do not attach the nodes without any useful annotations,
		// text or children.
		log.V(6).Infof("not attaching %s", n.Label)
		return
	}
	last := len(r.nodeStack) - 1
	if last < 0 {
		if r.Tree != nil {
			log.Exitf("Internal error: Attempting to attach root node twice")
		}
		r.Tree = n
		return
	}
	r.nodeStack[last].Children = append(r.nodeStack[last].Children, n)
}

type rhsError struct {
	rhs      *RHS
	details  map[string]error
	fyiError error
}

func (e rhsError) Error() string {
	var errs []string
	for name, err := range e.details {
		errs = append(errs, fmt.Sprintf("%s: %s", name, err))
	}
	var fyi string
	if e.fyiError != nil {
		fyi = e.fyiError.Error()
	}
	return fmt.Sprintf("rhs %s did not apply:{\n%s\nPrevious fyi error: %s\n}",
		e.rhs.ShortString(), strings.Join(errs, "\n"),
		strings.Replace(fyi, "\n", "\n  ", -1))
}

// TODO(salikh): rename ShortString->String, String->TreeString
func (rhs *RHS) ShortString() string {
	r := make([]string, 0, len(rhs.Terms))
	for _, terms := range rhs.Terms {
		r = append(r, groupToString(terms))
	}
	return strings.Join(r, " / ")
}

func (term *Term) ShortString() string {
	if term.Parens != nil {
		return "(" + term.Parens.ShortString() + ")"
	} else if term.NegPred != nil {
		return "!" + term.NegPred.ShortString()
	} else if term.Pred != nil {
		return "&" + term.Pred.ShortString()
	} else if term.Special != nil {
		return term.Special.ShortString()
	} else if term.Capture != nil {
		return "<" + term.Capture.ShortString() + ">"
	} else if term.CharClass != nil {
		return "[" + term.CharClass.String() + "]"
	} else if term.Literal != "" {
		return strconv.Quote(term.Literal)
	} else if term.Ident != "" {
		return term.Ident
	}
	return "<nil term>"
}

func (special *Special) ShortString() string {
	return fmt.Sprintf("%s%c", special.Term.ShortString(), special.Rune)
}

func groupToString(terms []*Term) string {
	r := make([]string, 0, len(terms))
	for _, term := range terms {
		r = append(r, term.ShortString())
	}
	return strings.Join(r, " ")
}

func (g *Grammar) makeRHSHandler(rhs *RHS) (handler, error) {
	choices := rhs.Terms
	if len(choices) == 1 {
		return g.makeGroupHandler(choices[0])
	}
	var hh []handler
	for _, terms := range choices {
		h, err := g.makeGroupHandler(terms)
		if err != nil {
			return nil, err
		}
		hh = append(hh, h)
	}
	return func(r *Result, pos int) (int, error) {
		save := r.TopNode().Children
		w, err := hh[0](r, pos)
		if err == nil {
			return w, nil
		}
		errMap := map[string]error{groupToString(choices[0]): err}
		for i := 1; err != nil && i < len(hh); i++ {
			r.TopNode().Children = save
			w, err = hh[i](r, pos)
			if err != nil {
				errMap[groupToString(choices[i])] = err
			}
		}
		if err != nil {
			return w,
				&rhsError{rhs: rhs, details: errMap, fyiError: r.fyiError}
		}
		return w, nil
	}, nil
}

func (g *Grammar) makeGroupHandler(terms []*Term) (handler, error) {
	if len(terms) == 1 {
		return g.makeTermHandler(terms[0])
	}
	var hh []handler
	for _, term := range terms {
		h, err := g.makeTermHandler(term)
		if err != nil {
			return nil, err
		}
		hh = append(hh, h)
	}
	return func(r *Result, pos int) (int, error) {
		ww := 0
		for _, h := range hh {
			w, err := h(r, pos+ww)
			if err != nil {
				return ww, err
			}
			ww += w
		}
		return ww, nil
	}, nil
}

func (g *Grammar) makeTermHandler(term *Term) (handler, error) {
	switch {
	case term.Parens != nil:
		return g.makeRHSHandler(term.Parens)
	case term.NegPred != nil:
		return g.makePredicateHandler(term.NegPred, false)
	case term.Pred != nil:
		return g.makePredicateHandler(term.Pred, true)
	case term.Special != nil:
		return g.makeSpecialHandler(term.Special)
	case term.Capture != nil:
		return g.makeCaptureHandler(term.Capture)
	case term.CharClass != nil:
		return g.makeCharClassHandler(term.CharClass)
	case term.Literal != "":
		return g.makeLiteralHandler(term.Literal)
	case term.Ident != "":
		return g.makeRuleHandler(term.Ident)
	default:
		log.Exitf("makeTermHandler NYI: %v", term)
	}
	panic("Should not be reached.")
}

func (r *Result) computeRowCol(pos int) (row, col int) {
	rowcol, ok := r.rowCol[pos]
	if ok {
		return rowcol.Row, rowcol.Col
	}
	row, col = countRowCol(r.Source, 1, 0)
	r.rowCol[pos] = RowCol{row, col}
	return row, col
}

func (r *Result) parserErrorf(pos int, format string, args ...interface{}) error {
	row, col := r.computeRowCol(pos)
	message := fmt.Sprintf(format, args...)
	return fmt.Errorf("%d:%d:%s", row, col, message)
}

func (g *Grammar) makeLiteralHandler(literal string) (handler, error) {
	return func(r *Result, pos int) (int, error) {
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
	}, nil
}

func (g *Grammar) makeCharClassHandler(cc *charclass.CharClass) (handler, error) {
	if cc.Special != "" {
		return func(r *Result, pos int) (int, error) {
			c, w := utf8.DecodeRuneInString(r.Source[pos:])
			if w == 0 {
				return 0, fmt.Errorf("expecting char, got EOF")
			}
			var match bool
			switch cc.Special {
			case "[:any:]":
				match = true
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
				return 0, fmt.Errorf("character %q does not match class %q", c, cc)
			}
			return w, nil
		}, nil
	}
	// Regular map case.
	return func(r *Result, pos int) (int, error) {
		c, w := utf8.DecodeRuneInString(r.Source[pos:])
		if w == 0 {
			return 0, fmt.Errorf("expecting char, got EOF")
		}
		if c == utf8.RuneError {
			return 0, fmt.Errorf("expecting utf-8 char, got RuneError")
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
			return 0, fmt.Errorf("character %q does not match class %q", c, cc)
		}
		return w, nil
	}, nil
}

func (g *Grammar) makeCaptureHandler(rhs *RHS) (handler, error) {
	h, err := g.makeRHSHandler(rhs)
	if err != nil {
		return nil, err
	}
	return func(r *Result, pos int) (int, error) {
		w, err := h(r, pos)
		if err != nil {
			return w, err
		}
		n := r.TopNode()
		if n == nil {
			return 0, fmt.Errorf("internal error, " +
				"cannot handle capture without a top node")
		}
		n.Text = r.Source[pos : pos+w]
		return w, nil
	}, nil
}

func (g *Grammar) makeSpecialHandler(special *Special) (handler, error) {
	h, err := g.makeTermHandler(special.Term)
	if err != nil {
		return nil, err
	}
	switch special.Rune {
	case '*':
		return g.makeStarHandler(h)
	case '?':
		return g.makeQuestionHandler(h)
	case '+':
		return g.makePlusHandler(h)
	default:
		log.Exitf("invalid special: %q", special.Rune)
	}
	panic("Should not be reached.")
}

func (g *Grammar) makeStarHandler(h handler) (handler, error) {
	return func(r *Result, pos int) (int, error) {
		ww := 0
		// We want to get the longest match
		save := r.TopNode().Children
		var w int
		var err error
		for w, err = h(r, pos); err == nil && w > 0; w, err = h(r, pos+ww) {
			ww += w
			// Update the saved nodes in case of success
			save = r.TopNode().Children
		}
		// Reset the nodes appended by the last unsuccessful match.
		r.TopNode().Children = save
		// Store the error just as FYI.
		if ww == 0 && err != nil {
			log.V(4).Infof("StarHandler error: %s", err)
			r.fyiError = err
		}
		// Star repetition always matches, in worst case it's zero length
		return ww, nil
	}, nil
}

func (g *Grammar) makePlusHandler(h handler) (handler, error) {
	return func(r *Result, pos int) (int, error) {
		ww, err := h(r, pos)
		if err != nil {
			return ww, err
		}
		// We want to get the longest match
		save := r.TopNode().Children
		for w, err := h(r, pos+ww); err == nil && w > 0; w, err = h(r, pos+ww) {
			ww += w
			// Update the saved nodes in case of success
			save = r.TopNode().Children
		}
		// Reset the nodes appended by the last unsuccessful match.
		r.TopNode().Children = save
		return ww, nil
	}, nil
}

func (g *Grammar) makeQuestionHandler(h handler) (handler, error) {
	return func(r *Result, pos int) (int, error) {
		w, err := h(r, pos)
		if err != nil {
			// Question option always matches, in worst case it's zero length
			return 0, nil
		}
		return w, nil
	}, nil
}

func (g *Grammar) makePredicateHandler(term *Term, positive bool) (handler, error) {
	h, err := g.makeTermHandler(term)
	if err != nil {
		return nil, err
	}
	return func(r *Result, pos int) (int, error) {
		_, err := h(r, pos)
		if positive == (err == nil) {
			return 0, nil
		}
		if err == nil {
			return 0, fmt.Errorf("negative predicate matched")
		}
		return 0, err
	}, nil
}

func (g *Grammar) makeRuleHandler(name string) (handler, error) {
	ru, ok := g.Rules[name]
	if !ok {
		return nil, fmt.Errorf("unknown rule: %s", name)
	}
	return func(r *Result, pos int) (int, error) {
		return r.apply(ru, pos)
	}, nil
}

// ComputeContent annotates the parse tree with pieces of original content
// and line/column positions in the original parser input.
func (r *Result) ComputeContent() {
	r.computeContent(r.Tree, 0 /*pos*/, 1 /*row*/, 0 /*col*/)
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
func (r *Result) computeContent(n *parser.Node, pos, row, col int) (int, int) {
	row, col = countRowCol(r.Source[pos:n.Pos], row, col)
	pos = n.Pos
	n.Row = row
	n.Col = col
	for _, ch := range n.Children {
		//log.Infof("child %s: pos=%d, ch.Pos=%d, ch.Len=%d", ch.Label, pos, ch.Pos, ch.Len)
		piece := r.Source[pos:ch.Pos]
		row, col = countRowCol(piece, row, col)
		n.Content = append(n.Content, piece)
		row, col = r.computeContent(ch, ch.Pos, row, col)
		pos = ch.Pos + ch.Len
	}
	//log.Infof("%s source: [0, %d), n.Pos=%d, pos=%d, n.Pos+n.Len=%d", n.Label, len(r.Source), n.Pos, pos, n.Pos+n.Len)
	piece := r.Source[pos : n.Pos+n.Len]
	row, col = countRowCol(piece, row, col)
	n.Content = append(n.Content, piece)
	return row, col
}

func (g *Grammar) ParseBackward(input string) (*Result, error) {
	result := &Result{
		Grammar:   g,
		Source:    input,
		memo:      make(map[int]map[*Rule]*parser.Node),
		nodeStack: make(NodeStack, 0, 10),
		rowCol:    make(map[int]RowCol),
	}
	if len(g.RuleNames) == 0 {
		return nil, fmt.Errorf("invalid grammar without rules")
	}
	topRule := g.RuleNames[0]
	top, ok := g.Rules[topRule]
	if !ok {
		return nil, fmt.Errorf("invalid grammar with missing top rule %s", topRule)
	}
	// TODO(salikh): check whether backwardApply requires anything special, or if apply() can be shared.
	w, err := result.backwardApply(top, len(input))
	if err != nil {
		return result, err
	}
	if w != len(input) && !g.ParserOptions.IgnoreUnconsumedTail {
		return result, fmt.Errorf("some characers remain unconsumed: %q"+
			"\nPrevious error: %s",
			input[0:len(input)-w], result.fyiError)
	}
	if result.Tree == nil {
		return nil, fmt.Errorf("internal error: no syntax tree. len(nodeStack) = %d",
			len(result.nodeStack))
	}
	reverse(result.Tree)
	return result, nil
}

func reverse(n *parser.Node) {
	log.V(5).Infof("reversing %s %#v", n.Label, n)
	// Make pos to point at the start of the matched content.
	n.Pos -= n.Len
	if len(n.Children) == 0 {
		return
	}
	l := len(n.Children)
	for i := range n.Children {
		reverse(n.Children[i])
	}
	for i := 0; i < l/2; i++ {
		log.V(5).Infof("len %d, (%d,%d)", len(n.Children), i, l-1-i)
		n.Children[i], n.Children[l-1-i] = n.Children[l-1-i], n.Children[i]
	}
}

func (g *Grammar) makeBackwardRHSHandler(rhs *RHS) (handler, error) {
	choices := rhs.Terms
	if len(choices) == 1 {
		return g.makeBackwardGroupHandler(choices[0])
	}
	var hh []handler
	for _, terms := range choices {
		h, err := g.makeBackwardGroupHandler(terms)
		if err != nil {
			return nil, err
		}
		hh = append(hh, h)
	}
	return func(r *Result, pos int) (int, error) {
		save := r.TopNode().Children
		w, err := hh[0](r, pos)
		for i := 1; err != nil && i < len(hh); i++ {
			r.TopNode().Children = save
			w, err = hh[i](r, pos)
		}
		// TODO(salikh): Collect errors from all branches to make
		// the error message more user-friendly.
		return w, err
	}, nil
}

func (g *Grammar) makeBackwardGroupHandler(terms []*Term) (handler, error) {
	if len(terms) == 1 {
		return g.makeBackwardTermHandler(terms[0])
	}
	var hh []handler
	for i := len(terms) - 1; i >= 0; i-- {
		term := terms[i]
		h, err := g.makeBackwardTermHandler(term)
		if err != nil {
			return nil, err
		}
		hh = append(hh, h)
	}
	return func(r *Result, pos int) (int, error) {
		ww := 0
		for _, h := range hh {
			w, err := h(r, pos-ww)
			if err != nil {
				return ww, err
			}
			ww += w
		}
		return ww, nil
	}, nil
}

func (g *Grammar) makeBackwardTermHandler(term *Term) (handler, error) {
	switch {
	case term.Parens != nil:
		return g.makeBackwardRHSHandler(term.Parens)
	case term.NegPred != nil:
		return g.makeBackwardPredicateHandler(term.NegPred, false)
	case term.Pred != nil:
		return g.makeBackwardPredicateHandler(term.Pred, true)
	case term.Special != nil:
		return g.makeBackwardSpecialHandler(term.Special)
	case term.Capture != nil:
		return g.makeBackwardCaptureHandler(term.Capture)
	case term.CharClass != nil:
		return g.makeBackwardCharClassHandler(term.CharClass)
	case term.Literal != "":
		return g.makeBackwardLiteralHandler(term.Literal)
	case term.Ident != "":
		return g.makeBackwardRuleHandler(term.Ident)
	default:
		log.Exitf("makeBackwardTermHandler NYI: %v", term)
	}
	panic("Should not be reached.")
}

func (g *Grammar) makeBackwardLiteralHandler(literal string) (handler, error) {
	return func(r *Result, pos int) (int, error) {
		log.V(5).Infof("trying backward literal %q at %d{%s}", literal, pos, r.Source[0:pos])
		if pos < len(literal) {
			log.V(5).Infof("too few characters available: %d", pos)
			return 0, fmt.Errorf("expecting %q, got %q",
				literal, r.Source[0:pos])
		}
		next := r.Source[pos-len(literal) : pos]
		if next != literal {
			log.V(5).Infof("does not match: got %q", next)
			return 0, fmt.Errorf("Expecting literal %q, got %q", literal, next)
		}
		// parse successful
		return len(literal), nil
	}, nil
}

func (g *Grammar) makeBackwardCharClassHandler(cc *charclass.CharClass) (handler, error) {
	if cc.Special != "" {
		return func(r *Result, pos int) (int, error) {
			c, w := utf8.DecodeLastRuneInString(r.Source[:pos])
			if w == 0 {
				return 0, fmt.Errorf("expecting char, got EOF")
			}
			if c == utf8.RuneError {
				return 0, fmt.Errorf("expecting utf-8 char, got RuneError")
			}
			var match bool
			switch cc.Special {
			case "[:any:]":
				match = true
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
				return 0, fmt.Errorf("character %q does not match class %q", c, cc)
			}
			return w, nil
		}, nil
	}
	// Regular map case.
	return func(r *Result, pos int) (int, error) {
		c, w := utf8.DecodeLastRuneInString(r.Source[:pos])
		if w == 0 {
			return 0, fmt.Errorf("expecting char, got EOF")
		}
		if c == utf8.RuneError {
			return 0, fmt.Errorf("expecting utf-8 char, got RuneError")
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
			return 0, fmt.Errorf("character %q does not match class %q", c, cc)
		}
		return w, nil
	}, nil
}

func (g *Grammar) makeBackwardCaptureHandler(rhs *RHS) (handler, error) {
	h, err := g.makeBackwardRHSHandler(rhs)
	if err != nil {
		return nil, err
	}
	return func(r *Result, pos int) (int, error) {
		w, err := h(r, pos)
		if err != nil {
			return w, err
		}
		n := r.TopNode()
		if n == nil {
			return 0, fmt.Errorf("internal error, " +
				"cannot handle capture without a top node")
		}
		n.Text = r.Source[pos-w : pos]
		return w, nil
	}, nil
}

func (g *Grammar) makeBackwardSpecialHandler(special *Special) (handler, error) {
	h, err := g.makeBackwardTermHandler(special.Term)
	if err != nil {
		return nil, err
	}
	switch special.Rune {
	case '*':
		return g.makeBackwardStarHandler(h)
	case '?':
		return g.makeBackwardQuestionHandler(h)
	case '+':
		return g.makeBackwardPlusHandler(h)
	default:
		log.Exitf("invalid special: %q", special.Rune)
	}
	panic("Should not be reached.")
}

func (g *Grammar) makeBackwardStarHandler(h handler) (handler, error) {
	return func(r *Result, pos int) (int, error) {
		ww := 0
		// We want to get the longest match
		save := r.TopNode().Children
		for w, err := h(r, pos); err == nil && w > 0; w, err = h(r, pos-ww) {
			ww += w
			// Update the saved nodes in case of success
			save = r.TopNode().Children
		}
		// Reset the nodes appended by the last unsuccessful match.
		r.TopNode().Children = save
		// Star repetition always matches, in worst case it's zero length
		return ww, nil
	}, nil
}

func (g *Grammar) makeBackwardPlusHandler(h handler) (handler, error) {
	return func(r *Result, pos int) (int, error) {
		ww, err := h(r, pos)
		if err != nil {
			return ww, err
		}
		// We want to get the longest match
		save := r.TopNode().Children
		for w, err := h(r, pos-ww); err == nil && w > 0; w, err = h(r, pos-ww) {
			ww += w
			// Update the saved nodes in case of success
			save = r.TopNode().Children
		}
		// Reset the nodes appended by the last unsuccessful match.
		r.TopNode().Children = save
		return ww, nil
	}, nil
}

func (g *Grammar) makeBackwardQuestionHandler(h handler) (handler, error) {
	return func(r *Result, pos int) (int, error) {
		w, err := h(r, pos)
		if err != nil {
			// Question option always matches, in worst case it's zero length
			return 0, nil
		}
		return w, nil
	}, nil
}

func (g *Grammar) makeBackwardPredicateHandler(term *Term, positive bool) (handler, error) {
	// Note: predicates are checked backward!
	h, err := g.makeBackwardTermHandler(term)
	if err != nil {
		return nil, err
	}
	return func(r *Result, pos int) (int, error) {
		_, err := h(r, pos)
		if positive == (err == nil) {
			return 0, nil
		}
		if err == nil {
			return 0, fmt.Errorf("negative predicate matched")
		}
		return 0, err
	}, nil
}

func (g *Grammar) makeBackwardRuleHandler(name string) (handler, error) {
	ru, ok := g.Rules[name]
	if !ok {
		return nil, fmt.Errorf("unknown rule: %s", name)
	}
	return func(r *Result, pos int) (int, error) {
		return r.backwardApply(ru, pos)
	}, nil
}

func (r *Result) backwardApply(ru *Rule, pos int) (int, error) {
	log.V(5).Infof("backwardApply(%s, %d)  {%s}", ru.Ident, pos, r.Source[0:pos])
	memo, ok := r.memo[pos]
	if !ok {
		memo = make(map[*Rule]*parser.Node)
		r.memo[pos] = memo
	}
	n := memo[ru]
	if n != nil {
		if n.Err != nil {
			return n.Len, n.Err
		}
		// Since nodes are attached in backward direction, the trees will be reversed.
		r.Attach(n)
		return n.Len, nil
	}
	n = &parser.Node{Label: ru.Ident, Pos: pos}
	r.nodeStack.Push(n)
	w, hErr := ru.backwardHandler(r, pos)
	n = r.nodeStack.Pop()
	n.Len = w
	n.Err = hErr
	memo[ru] = n
	n.Len = w
	log.V(6).Infof("attaching %s", n.Label)
	r.Attach(n)
	return w, hErr
}

// ParseRule parses the input starting with a specified rule.
// If the ruleName is empty, uses the top rule.
func (g *Grammar) ParseRule(input, ruleName string) (*Result, error) {
	if g == nil {
		return nil, fmt.Errorf("nil grammar")
	}
	result := &Result{
		Grammar:   g,
		Source:    input,
		memo:      make(map[int]map[*Rule]*parser.Node),
		nodeStack: make(NodeStack, 0, 10),
		rowCol:    make(map[int]RowCol),
	}
	if len(g.RuleNames) == 0 {
		return nil, fmt.Errorf("invalid grammar without rules")
	}
	if ruleName == "" {
		ruleName = g.RuleNames[0]
	}
	rule, ok := g.Rules[ruleName]
	if !ok {
		return nil, fmt.Errorf("missing rule %s", ruleName)
	}
	w, err := result.apply(rule, 0)
	if err != nil {
		return result, err
	}
	if w == 0 && len(input) > 0 {
		return result, fmt.Errorf("grammar matched 0 characters: %s", result.fyiError)
	}
	if w != len(input) && !g.ParserOptions.IgnoreUnconsumedTail {
		errContent := input[w:]
		if !g.ParserOptions.LongErrorMessage {
			num := 10
			if len(errContent) > num {
				errContent = errContent[:num] + "..."
			}
		}
		return result, fmt.Errorf("some characers remain unconsumed: %q"+
			"\nPrevious error: %s", errContent, result.fyiError)
	}
	if result.Tree == nil {
		return nil, fmt.Errorf("internal error: no syntax tree. len(nodeStack) = %d",
			len(result.nodeStack))
	}
	return result, nil
}

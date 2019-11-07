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

// This package implements a static parser generator for
// parsing expression grammars (PEG) that is compatible
// with dynamic parser.
//
// go:generate go run cmd/generator/generator-main.go --grammar=peg.peg --output=peg.peg.go --package=generator
package generator

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/salikh/peg/parser"
	"github.com/salikh/peg/parser/charclass"
)

// Grammar represents the complete PEG grammar.
type Grammar struct {
	// Rule is a dictionary of rules.
	Rules map[string]*Rule
	// RuleNames keeps the list of rule name in the original definition order.
	RuleNames []string
	Source    string
}

// Rule represent one PEG rule (Rule <- RHS).
type Rule struct {
	// Ident is the name of the rule, defined in LHS.
	Ident string
	// RHS is the rule's right-hand side.
	*RHS
}

// RHS is the right-hand side of one rule or the contents of parenthesized expression.
type RHS struct {
	// Terms is the RHS, top iteration over choices, and inside
	// iteration over concatenation of terms.
	Terms [][]*Term
}

// Term is one term. Note: Special characters in the grammar are handled specially
// and are not mapped to Term one-to-one. Instead, *?+ combine with the previous
// Term, and . is converted to a special CharClass.
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

// ConvertGrammar2 is a  reimplementation of the semantic tree reconstruction
// using callbacks.
func ConvertGrammar2(n *parser.Node) (*Grammar, error) {
	val, err := Construct(n, callback, &AccessorOptions{
		ErrorOnUnusedChild: true,
	})
	if err != nil {
		return nil, err
	}
	g, ok := val.(*Grammar)
	if !ok {
		return nil, fmt.Errorf("Could convert type %s to *Grammar",
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
		for _, rule := range ca.Get("Rule", []*Rule{}).([]*Rule) {
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

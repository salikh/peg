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

// Package tree provides a few utility functions for working with syntax tree
// generated by the PEG parser, including serialization, deserialization,
// canonicalization and diffing.
// It also provides a handy utility for extracting parts of the tree that
// is useful for testing.
package tree

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	"github.com/salikh/peg/parser"
)

var (
	treeGrammar parser.Grammar
)

// TODO(salikh): Port tree parser to generated parser.
var treeGrammarSource = `
Node <- _ "(" Label (Node / Annotation / String)* _ ")" _

Label <- _ < [A-Za-z_][A-Za-z0-9_]* >
Annotation <- _ < ":"? > Label ( _ "(" String _ ")" / Node )
String <- _ < '"' ('\"' / !'"' .)* '"' >

_ <- [ \t\n]*
`

func init() {
	var err error
	treeGrammar, err = parser.New(treeGrammarSource)
	if err != nil {
		log.Exitf("Error in tree grammar:\n%s.", err)
	}
}

// Parse parses the tree serialization format into regular
// parser tree format.
func Parse(input string) (*parser.Node, error) {
	ast, err := treeGrammar.Parse(input)
	if err != nil {
		return nil, err
	}
	log.V(5).Infof("Tree parse tree:\n%s\n---", ast.Tree)
	return rewriteNode(ast.Tree)
}

func rewriteNode(ast *parser.Node) (*parser.Node, error) {
	if ast.Label != "Node" {
		return nil, fmt.Errorf("expecting Node, got %s", ast.Label)
	}
	label, err := rewriteLabel(ast)
	if err != nil {
		return nil, fmt.Errorf("error parsing node label:\n%s", err)
	}
	var children []*parser.Node
	var text string
	ann := make(map[string]string)
	treeAnn := make(map[string]*parser.Node)
	astChildren := ast.Children
	for i := 1; i < len(astChildren); i++ {
		n := astChildren[i]
		if n.Label == "String" {
			text, err = unEscape(n.Text)
			if err != nil {
				return nil, fmt.Errorf("error unescaping string %q: %s", n.Text, err)
			}
			continue
		} else if n.Label == "Annotation" {
			key, err := rewriteLabel(n)
			if err != nil {
				return nil, fmt.Errorf("error parsing annot label:\n%s", err)
			}
			val, subtree, err := rewriteValue(n)
			if err != nil {
				return nil, fmt.Errorf("error parsing annot value:\n%s", err)
			}
			if n.Text == "" {
				// Fixed annotations, e.g. Text.
				switch key {
				case "text", "Text":
					text = val
				default:
					return nil, fmt.Errorf("unknown fixed annotation %q", key)
				}
			} else if n.Text == ":" {
				// Flexible annotations, e.g. :something.
				if subtree == nil {
					// Text annotation.
					ann[key] = val
				} else {
					treeAnn[key] = subtree
				}
			} else {
				// Should not happen.
				return nil, fmt.Errorf("annotation %s has unexpected character %q", key, n.Text)
			}
			continue
		} else if n.Label == "Label" {
			text = n.Text
			continue
		}
		// Not an annotation, just rewrite the node and append to children
		r, err := rewriteNode(n)
		if err != nil {
			return nil, fmt.Errorf("error rewriting children of %s:\n%s", label, err)
		}
		children = append(children, r)
	}
	return &parser.Node{
		Label:           label,
		Text:            text,
		Annotations:     ann,
		TreeAnnotations: treeAnn,
		Children:        children,
	}, nil
}

// TODO: fix this to expect (Label)
func rewriteLabel(ast *parser.Node) (string, error) {
	astChildren := ast.Children
	if len(astChildren) < 1 {
		return "", fmt.Errorf("expecting at least 1 child (Label), got 0")
	}
	n := astChildren[0]
	if n.Label != "Label" {
		return "", fmt.Errorf("expecting Label, got %s", n.Label)
	}
	return n.Text, nil
}

// unEscape replaces the standard C escape sequences with their unespaced values.
// \n\r\t
func unEscape(s string) (string, error) {
	if s[0] == '"' {
		return strconv.Unquote(s)
	}
	// TODO(salikh): Remove the custom escaping code.
	in := []rune(s)
	out := make([]rune, 0, len(in))
	for i := 0; i < len(in); i++ {
		c := in[i]
		if c == '\\' {
			i++
			c = in[i]
			switch c {
			case 'n':
				c = '\n'
			case 'r':
				c = '\r'
			case 't':
				c = '\t'
			}
		}
		out = append(out, c)
	}
	return string(out), nil
}

func rewriteValue(ast *parser.Node) (string, *parser.Node, error) {
	astChildren := ast.Children
	if len(astChildren) != 2 {
		return "", nil, fmt.Errorf("expecting 2 children in Annotation, got %d", len(astChildren))
	}
	n := astChildren[1]
	if n.Label == "Node" {
		subtree, err := rewriteNode(n)
		return "", subtree, err
	} else if n.Label == "String" {
		unesc, err := unEscape(n.Text)
		if err != nil {
			return "", nil, fmt.Errorf("error unescaping string %q: %s", n.Text, err)
		}
		return unesc, nil, nil
	}
	return "", nil, fmt.Errorf("expecting Node or String, got %s", n.Label)
}

func Pretty(input string) (string, error) {
	tree, err := Parse(input)
	if err != nil {
		return input, err
	}
	return tree.String(), nil
}

func PrettyNoErr(input string) string {
	ret, err := Pretty(input)
	if err != nil {
		return ret + fmt.Sprintf("(error %s)", err)
	}
	return ret
}

// Extract extracts a single piece of information from a tree and returns it as
// a string.
// The accepted expressions are space-separated chained instructions:
// * 'Label' selects a first child with a matching label.
// * '[3]' selects 3rd (0-based) child.
// * 'Label[3]' selects 3rd (0-based) child with a matching label.
// * 'Label[-1]' selects the last child with a matching label.
// At the end of evaluation, the captured text of the current
// node is returned. Instead of the captured text, the following may be
// used as the last instruction:
// * 'num' extracts the number of child nodes of the current node, if a
//   specific node was selected (the top node or via indexed accessor),
//   or returns the number of nodes matching the last mulitple selector
//   (e.g. the child label selector without index).
// * 'pos' extracts the byte position of the current node match.
// * 'row' extracts the row (1-based) of the current node match.
// * 'col' extracts the column (0-based byte position in the line) of the
//   current node match.
// * 'len' extracts the length of the full node match.
// * 'text' extracts the captured text of the current node match (same as
//   default action).
// Note: to use row, col, one must have called ComputeContent on the parse
// Result.
// If the accessor instructions does not match anything, this method
// returns a human-readable error description.
func Extract(n *parser.Node, expr string) (string, error) {
	parts := strings.Split(expr, " ")
	// The state of the evaluator is either single node (cur) or multiple nodes
	// (list).
	cur := n
	var list []*parser.Node
	for termIndex, term := range parts {
		termIsLast := termIndex == len(parts)-1
		pos := strings.Index(term, "[")
		if pos == 0 {
			// The numbered child accessor.
			last := strings.Index(term[pos+1:], "]")
			if last < 0 {
				return "", fmt.Errorf("unterminated '[' in term %s", term)
			}
			val, err := strconv.ParseInt(term[pos+1:pos+1+last], 10, 32)
			if err != nil {
				return "", fmt.Errorf("could not parse term term %s: %s)", term, err)
			}
			index := int(val)
			if index < 0 || index >= len(cur.Children) {
				return "", fmt.Errorf("index %d out of bounds of %s children (%d)",
					index, cur.Label, len(cur.Children))
			}
			cur = cur.Children[index]
			list = nil
			continue
		}
		if pos > 0 {
			label := term[0:pos]
			last := strings.Index(term[pos+1:], "]")
			if last < 0 {
				return "", fmt.Errorf("unterminated '[' in term %s", term)
			}
			val, err := strconv.ParseInt(term[pos+1:pos+1+last], 10, 32)
			if err != nil {
				return "", fmt.Errorf("could not parse term term %s: %s)", term, err)
			}
			index := int(val)
			count := 0
			found := false
			for _, ch := range cur.Children {
				if ch.Label == label {
					if count == index {
						found = true
						cur = ch
						list = nil
						break
					}
					count++
					continue
				}
			}
			if !found {
				return "", fmt.Errorf("could not find %s[%d] in %s", label, index, cur.Label)
			}
			continue
		}
		if term == "text" || term == "row" || term == "col" || term == "pos" ||
			term == "len" || term == "num" {
			if !termIsLast {
				return "", fmt.Errorf("term %s must the last", term)
			}
			switch term {
			case "text":
				return cur.Text, nil
			case "row":
				return strconv.FormatInt(int64(cur.Row), 10), nil
			case "col":
				return strconv.FormatInt(int64(cur.Col), 10), nil
			case "pos":
				return strconv.FormatInt(int64(cur.Pos), 10), nil
			case "len":
				return strconv.FormatInt(int64(cur.Len), 10), nil
			case "num":
				if len(list) > 0 {
					return strconv.FormatInt(int64(len(list)), 10), nil
				} else {
					return strconv.FormatInt(int64(len(cur.Children)), 10), nil
				}
			}
			// Should not be reached.
			return "", fmt.Errorf("unknown term %s", term)
		}
		// Find the children with matching label.
		list = nil
		for _, ch := range cur.Children {
			if ch.Label == term {
				list = append(list, ch)
			}
		}
		if len(list) == 0 {
			return "", fmt.Errorf("could not find %s in %s", term, cur.Label)
		}
		// For node-based commands, default to the first element of the list.
		cur = list[0]
	}
	return cur.Text, nil
}

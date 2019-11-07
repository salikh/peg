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
	"bytes"
	"fmt"
	"sort"
	"strings"
)

type Node struct {
	// Label determines the type of the node, usually corresponding
	// to the rule name (LHS of the parser rule).
	Label string
	// Text is a captured text, if a rule defines a capture region.
	Text string
	// Content is a complete text that was consumed during parsing of this
	// element.  Maybe nil if the content was not recorded during parsing, or for
	// nodes that were programmatically generated.  If not nil, len(Content)
	// should be len(Children) + 1 When present, concatenating node Content in
	// order with children content results in exactly the parser input.
	// NOTE: this is not computed during parsing, but at a later stage.
	Content []string
	// The byte posittion of the first character consumed by this node rule
	// during parsing in the buffer that was passed to parser.
	Pos int
	// The number of bytes consumed by this node rule.
	Len int
	// The line number of the first character consumed by this node. 1-based.
	Row int
	// The column number of the first character consumed by this node. 0-based.
	Col int
	// The children of this node.
	Children []*Node
	// Err caches the error that resulted from application of some parsing rule at some position.
	Err error
	// Annotations stores some string-form annotations.
	Annotations map[string]string
	// TreeAnnotations stores some tree-form annotations (aka labelled children).
	TreeAnnotations map[string]*Node
	// Private fields.
	// Start is used by the bootstrap parser to implement captures. Parser generator or main parser
	// do not need it, but they also set it for compatibility with node drop/keep heuristics.
	// TODO(salikh): Remove after deprecating the bootstrap parser.
	Start int
}

// toString converts a node to string, taking the current indent level
// as a parameter. If full is true, it prints all node annotations,
// otherwise only the parse result is shown.
func (n *Node) toString(indent string, full bool) string {
	var r []string
	r = append(r, "(", n.Label)
	keys := make([]string, 0, len(n.Annotations))
	for k := range n.Annotations {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := n.Annotations[k]
		r = append(r, fmt.Sprintf(" :%s(%q)", k, v))
	}
	keys = make([]string, 0, len(n.TreeAnnotations))
	for k := range n.TreeAnnotations {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := n.TreeAnnotations[k]
		ns := v.toString(indent+"  ", full)
		r = append(r, fmt.Sprintf(" :%s%s", k, ns))
	}
	if n.Text != "" {
		r = append(r, fmt.Sprintf(" %q", n.Text))
	}
	if full {
		r = append(r, fmt.Sprintf(" pos(%d,%d)", n.Pos, n.Len))
		if n.Row != 0 {
			r = append(r, fmt.Sprintf(" rowcol(%d,%d)", n.Row, n.Col))
		}
	}
	nl := false
	//r = append(r, fmt.Sprintf(" children:%d", len(n.Children)))
	for _, child := range n.Children {
		ss := child.toString(indent+"  ", full)
		if len(ss) > 40 {
			nl = true
		}
		if nl {
			r = append(r, "\n", indent)
		}
		r = append(r, " ", ss)
	}
	r = append(r, ")")
	return strings.Join(r, "")
}

// Dup makes a shallow copy of a node.
func (n *Node) Dup() *Node {
	ann := make(map[string]string)
	for k, v := range n.Annotations {
		ann[k] = v
	}
	treeAnn := make(map[string]*Node)
	for k, v := range n.TreeAnnotations {
		treeAnn[k] = v
	}
	newChildren := make([]*Node, len(n.Children))
	copy(newChildren, n.Children)
	return &Node{
		Label:           n.Label,
		Text:            n.Text,
		Annotations:     ann,
		TreeAnnotations: treeAnn,
		Children:        newChildren,
	}
}

// DeepDup makes a deep copy of a node.
func (n *Node) DeepDup() *Node {
	// Start with a shallow copy.
	n = n.Dup()
	// And then recursively copy children.
	for i, ch := range n.Children {
		n.Children[i] = ch.DeepDup()
	}
	return n
}

func (n *Node) String() string {
	if n == nil {
		return "(nil)"
	}
	return n.toString("", false)
}

func (n *Node) Dump() string {
	if n == nil {
		return "(nil)"
	}
	return n.toString("", true)
}

func (n *Node) reconstructContent(buf *bytes.Buffer) error {
	if len(n.Content) == 0 {
		return fmt.Errorf("empty content: Node %v", n)
	}
	if len(n.Content) != len(n.Children)+1 {
		return fmt.Errorf("broken content: Node %s", n.Dump())
	}
	for i, ch := range n.Children {
		buf.WriteString(n.Content[i])
		ch.reconstructContent(buf)
	}
	buf.WriteString(n.Content[len(n.Children)])
	return nil
}

func (n *Node) ReconstructContent() (string, error) {
	buf := new(bytes.Buffer)
	err := n.reconstructContent(buf)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// First returns the first child with index at least n that has the specified label.
// If ther is no matching child node, it returns nil.
func (n *Node) First(label string, start int) *Node {
	for i := start; i < len(n.Children); i++ {
		ch := n.Children[i]
		if ch.Label == label {
			return ch
		}
	}
	return nil
}

// All returns the slice of node children with the specified label.
func (n *Node) All(label string) []*Node {
	var r []*Node
	for _, ch := range n.Children {
		if ch.Label == label {
			r = append(r, ch)
		}
	}
	return r
}

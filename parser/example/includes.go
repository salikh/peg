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

package includes

import (
	"fmt"
	"unicode/utf8"

	"github.com/salikh/peg/parser"
)

type Source struct {
	Includes []IncludeBlock
	Using    []Using
}

type IncludeBlock []Include

type Include struct {
	QuoteOpen rune
	Text      string
	Lineno    int
}

type Using struct {
	Text   string
	Lineno int
}

func Convert(n *parser.Node) (*Source, error) {
	return convertSource(n)
}

func convertSource(n *parser.Node) (*Source, error) {
	if n == nil {
		return nil, fmt.Errorf("expected Source, got nil")
	}
	if n.Label != "Source" {
		return nil, fmt.Errorf("expected Source, got %s", n.Label)
	}
	r := &Source{}
	for _, ch := range n.Children {
		switch ch.Label {
		case "Includes":
			includes, err := convertIncludes(ch)
			if err != nil {
				return nil, fmt.Errorf("error while converting Includes: %s", err)
			}
			r.Includes = includes
		case "Using":
			using, err := convertUsing(ch)
			if err != nil {
				return nil, fmt.Errorf("error while converting Using: %s", err)
			}
			r.Using = append(r.Using, using)
		}
	}
	return r, nil
}

func convertUsing(n *parser.Node) (Using, error) {
	if n == nil {
		return Using{}, fmt.Errorf("expected Using, got nil")
	}
	if n.Label != "Using" {
		return Using{}, fmt.Errorf("expected Using, got %s", n.Label)
	}
	return Using{
		Text:   n.Text,
		Lineno: n.Row,
	}, nil
}

func convertIncludes(n *parser.Node) ([]IncludeBlock, error) {
	if n == nil {
		return nil, fmt.Errorf("expected Includes, got nil")
	}
	if n.Label != "Includes" {
		return nil, fmt.Errorf("expected Includes, got %s", n.Label)
	}
	r := []IncludeBlock{}
	for _, ch := range n.Children {
		switch ch.Label {
		case "IncludeBlock":
			includeblock, err := convertIncludeBlock(ch)
			if err != nil {
				return nil, fmt.Errorf("error converting IncludeBlock: %s", err)
			}
			r = append(r, includeblock)
		}
	}
	return r, nil
}

func convertIncludeBlock(n *parser.Node) (IncludeBlock, error) {
	if n == nil {
		return nil, fmt.Errorf("expected IncludeBlock, got nil")
	}
	if n.Label != "IncludeBlock" {
		return nil, fmt.Errorf("expected IncludeBlock, got %s", n.Label)
	}
	r := IncludeBlock{}
	for _, ch := range n.Children {
		switch ch.Label {
		case "Include":
			include, err := convertInclude(ch)
			if err != nil {
				return nil, fmt.Errorf("error converting Include: %s", err)
			}
			r = append(r, include)
		}
	}
	return r, nil
}

func convertInclude(n *parser.Node) (Include, error) {
	if n == nil {
		return Include{}, fmt.Errorf("expected Include, got nil")
	}
	if n.Label != "Include" {
		return Include{}, fmt.Errorf("expected Include, got %s", n.Label)
	}
	r := &Include{
		Lineno: n.Row,
	}
	for _, ch := range n.Children {
		switch ch.Label {
		case "QuoteOpen":
			ru, n := utf8.DecodeRuneInString(ch.Text)
			if n == 0 {
				return Include{}, fmt.Errorf("invalid rune in %q", ch.Text)
			}
			if n < len(ch.Text) {
				return Include{}, fmt.Errorf("rune did not consume characters: %q", ch.Text[n:])
			}
			r.QuoteOpen = ru
		}
	}
	r.Text = n.Text
	return *r, nil
}

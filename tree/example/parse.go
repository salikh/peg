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

package example

import (
	"fmt"

	"github.com/salikh/peg/parser"
	"github.com/salikh/peg/tree"
)

func ParseTree(source string) (*Tree, error) {
	if source == "" {
		// Special nil case.
		return nil, nil
	}
	parsed, err := tree.Parse(source)
	if err != nil {
		return nil, err
	}
	return convertTree(parsed)
}

func convertTree(n *parser.Node) (*Tree, error) {
	if n.Label != "Tree" {
		return nil, fmt.Errorf("Expecting Tree, got %s", n.Label)
	}
	r := &Tree{
		Text: n.Text,
	}
	countTree := 0
	for _, ch := range n.Children {
		switch ch.Label {
		case "Tree":
			tree, err := convertTree(ch)
			if err != nil {
				return nil, err
			}
			label := ch.Annotations["label"]
			if (countTree == 0 && label == "") || label == "Left" {
				r.Left = tree
			} else if (countTree == 1 && label == "") || label == "Right" {
				r.Right = tree
			} else if label != "" {
				return nil, fmt.Errorf("Tree has unexpected label %s", label)
			}
			countTree++
		default:
			return nil, fmt.Errorf("Unexpected Tree child %s", ch.Label)
		}
	}
	if countTree > 2 {
		return nil, fmt.Errorf("Tree has too many Tree children (%d), expected up to 2", countTree)
	}
	return r, nil
}

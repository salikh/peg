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
	"strconv"
	"strings"
)

func SerializeTree(tree *Tree) string {
	return serializeTree(tree, "")
}

func serializeTree(tree *Tree, label string) string {
	if tree == nil {
		return ""
	}
	parts := []string{"(Tree"}
	if label != "" {
		parts = append(parts, " :label(", strconv.Quote(label), ")")
	}
	if tree.Text != "" {
		parts = append(parts, " ", "Text(", strconv.Quote(tree.Text), ")")
	}
	if tree.Left != nil {
		parts = append(parts, " ", serializeTree(tree.Left, ""))
	}
	if tree.Right != nil {
		label := ""
		if tree.Left == nil {
			// This node is in non-default position so needs a label.
			label = "Right"
		}
		parts = append(parts, " ", serializeTree(tree.Right, label))
	}
	parts = append(parts, ")")
	return strings.Join(parts, "")
}

func (t *Tree) String() string {
	return SerializeTree(t)
}

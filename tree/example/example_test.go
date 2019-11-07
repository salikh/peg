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
	"reflect"
	"testing"
)

type TreeTest struct {
	*Tree
	Serialized string
}

var treeTests = []TreeTest{
	{
		Tree:       nil,
		Serialized: "",
	},
	{
		Tree:       &Tree{},
		Serialized: "(Tree)",
	},
	{
		Tree: &Tree{
			Text: "abc",
		},
		Serialized: `(Tree Text("abc"))`,
	},
	{
		Tree: &Tree{
			Text: `"abc"`,
		},
		Serialized: `(Tree Text("\"abc\""))`,
	},
	{
		Tree: &Tree{
			Left: &Tree{},
		},
		Serialized: `(Tree (Tree))`,
	},
	{
		Tree: &Tree{
			Right: &Tree{},
		},
		Serialized: `(Tree (Tree :label("Right")))`,
	},
	{
		Tree: &Tree{
			Left:  &Tree{},
			Right: &Tree{},
		},
		Serialized: `(Tree (Tree) (Tree))`,
	},
	{
		Tree: &Tree{
			Left: &Tree{
				Left: &Tree{
					Left: &Tree{},
				},
			},
			Right: &Tree{},
		},
		Serialized: `(Tree (Tree (Tree (Tree))) (Tree))`,
	},
}

func TestSerialize(t *testing.T) {
	for _, tt := range treeTests {
		got := SerializeTree(tt.Tree)
		if got != tt.Serialized {
			t.Errorf("Serialize(%#v) = %q, want %q", tt.Tree, got, tt.Serialized)
		}
	}
}

func TestParse(t *testing.T) {
	for _, tt := range treeTests {
		got, err := ParseTree(tt.Serialized)
		if err != nil {
			t.Errorf("ParseTree(`%s`) returns error %s, want success", tt.Serialized, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.Tree) {
			t.Errorf("ParseTree(`%s`) = %s, want %s", tt.Serialized, got, tt.Tree)
		}
	}
}

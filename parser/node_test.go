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
	"strings"
	"testing"
)

func TestNodeToString(t *testing.T) {
	tests := []struct {
		node *Node
		want string
	}{
		{
			node: &Node{
				Label: "Label",
			},
			want: "(Label)",
		},
		{
			node: &Node{
				Label: "Label",
				Text:  "txt",
			},
			want: `(Label "txt")`,
		},
		{
			node: &Node{
				Label: "Label",
				Children: []*Node{
					&Node{
						Label: "Child1",
					},
					&Node{
						Label: "Child2",
					},
				},
			},
			want: `(Label (Child1) (Child2))`,
		},
	}
	for _, tt := range tests {
		got := tt.node.String()
		if got != tt.want {
			t.Errorf("%#v.String() = %q, want %q", tt.node, got, tt.want)
		}
	}
}

func TestNodeReconstruct(t *testing.T) {
	tests := []struct {
		node *Node
		want string
	}{
		{
			node: &Node{
				Label:   "Label",
				Content: []string{"content"},
			},
			want: "content",
		},
		{
			node: &Node{
				Label:   "Label",
				Text:    "txt",
				Content: []string{"content"},
			},
			want: "content",
		},
		{
			node: &Node{
				Label:   "Label",
				Content: []string{"0-", "-1-", "-2"},
				Children: []*Node{
					&Node{
						Label:   "Child1",
						Content: []string{"child1"},
					},
					&Node{
						Label:   "Child2",
						Content: []string{"child2"},
					},
				},
			},
			want: "0-child1-1-child2-2",
		},
	}
	for _, tt := range tests {
		got, err := tt.node.ReconstructContent()
		if err != nil {
			t.Errorf("Node(%s).ReconstructContent() returns error %s, want success", tt.node, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%#v.ReconstructContent() = %q, want %q", tt.node, got, tt.want)
		}
	}
}

func TestNodeReconstructError(t *testing.T) {
	tests := []struct {
		node *Node
		want string
	}{
		{
			node: &Node{
				Label:   "Label",
				Content: []string{"content", "abc"},
			},
			want: "broken content",
		},
		{
			node: &Node{
				Label: "Label",
				Text:  "txt",
			},
			want: "empty content",
		},
	}
	for _, tt := range tests {
		_, err := tt.node.ReconstructContent()
		if err == nil {
			t.Errorf("Node(%s).ReconstructContent() returns success, want error %q", tt.node, tt.want)
			continue
		}
		got := err.Error()
		if strings.Index(got, tt.want) == -1 {
			t.Errorf("%#v.ReconstructContent() returns error %q, want %q", tt.node, got, tt.want)
		}
	}
}

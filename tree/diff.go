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

package tree

import (
	"fmt"

	"github.com/salikh/peg/parser"
)

func Diff(got, want *parser.Node) (diff []string) {
	if got == nil && want == nil {
		return nil
	}
	if got == nil {
		diff = append(diff, fmt.Sprintf("Expected (%s), got nil", want.Label))
		return
	}
	if want == nil {
		diff = append(diff, fmt.Sprintf("Expected nil, got (%s)", got.Label))
		return
	}
	if got.Label != want.Label {
		diff = append(diff, fmt.Sprintf("Expected (%s), got (%q)", want.Label, got.Label))
	}
	checked := make(map[string]bool)
	for k, v := range want.Annotations {
		vv, ok := got.Annotations[k]
		if !ok {
			diff = append(diff, fmt.Sprintf("Expected annotation :%s(%q), not found", k, v))
			continue
		}
		if vv != v {
			diff = append(diff, fmt.Sprintf("Expected annotation :%s(%q), got %q", k, v, vv))
		}
		checked[k] = true
	}
	for k, v := range got.Annotations {
		if checked[k] {
			continue
		}
		diff = append(diff, fmt.Sprintf("Extra annotation :%s(%q), not expected", k, v))
	}
	treeChecked := make(map[string]bool)
	for k, v := range want.TreeAnnotations {
		vv, ok := got.TreeAnnotations[k]
		if !ok {
			diff = append(diff, fmt.Sprintf("Expected annotation :%s%s, not found", k, v))
			continue
		}
		subdiffs := Diff(vv, v)
		if len(subdiffs) > 0 {
			diff = append(diff, fmt.Sprintf("Diffs in tree annotation :%s():", k))
			diff = append(diff, subdiffs...)
		}
		treeChecked[k] = true
	}
	for k, v := range got.TreeAnnotations {
		if treeChecked[k] {
			continue
		}
		diff = append(diff, fmt.Sprintf("Extra annotation :%s%s, not expected", k, v))
	}
	if got.Text != want.Text {
		diff = append(diff, fmt.Sprintf("Expected text %q, got %q", want.Text, got.Text))
	}
	if len(got.Children) != len(want.Children) {
		diff = append(diff, fmt.Sprintf("Expected %d children got %d", len(want.Children), len(got.Children)))
	}
	n := len(got.Children)
	if len(want.Children) < n {
		n = len(want.Children)
	}
	for i := 0; i < n; i++ {
		diff = append(diff, Diff(got.Children[i], want.Children[i])...)
	}
	return
}

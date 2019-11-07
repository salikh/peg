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
	"strings"
	"testing"
)

func TestDiff(t *testing.T) {
	tests := []struct {
		a, b  string
		equal bool
	}{
		{"(x)", "(x)", true},
		{`(x text(""))`, `(x)`, true},
		{`(x text("a"))`, `(x)`, false},
		{`(x :attr("a"))`, `(x)`, false},
		{`(x :attr("a"))`, `(x :attr("a"))`, true},
		{`(x :attr1("a"))`, `(x :attr2("a"))`, false},
		{`(x )`, `(x text("a"))`, false},
		{`(x text("a"))`, `(x)`, false},
		{`(x text("a"))`, `(x text("a"))`, true},
		{`(x text("a"))`, `(x text("b"))`, false},
		{"(x)", "(y)", false},
		{"(x (y))", "(x)", false},
		{"(x )", "(x (y))", false},
		{"(x (y))", "(x (y))", true},
		{"(x (y) (z))", "(x (y) (z))", true},
		{"(x (z) (y))", "(x (y) (z))", false},
		{"(x (y (z)))", "(x (y (z)))", true},
		{"(x (z (y)))", "(x (y (z)))", false},
		{`(x (y text("a") (z)))`, `(x (y (z)))`, false},
		{`(x (y text("a") (z)))`, `(x (y text("a") (z)))`, true},
	}
	for _, tt := range tests {
		a, err := Parse(tt.a)
		if err != nil {
			t.Errorf("could not parse tree %s: %s", tt.a, err)
			continue
		}
		b, err := Parse(tt.b)
		if err != nil {
			t.Errorf("could not parse tree %s: %s", tt.b, err)
			continue
		}
		diffs := Diff(a, b)
		if tt.equal && len(diffs) > 0 {
			t.Errorf("Diff(%s, %s) returned %v, want {}", tt.a, tt.b, strings.Join(diffs, "\n"))
			continue
		}
		if !tt.equal && len(diffs) == 0 {
			t.Errorf("Diff(%s, %s) returned {}, want diff", tt.a, tt.b)
		}
	}
}

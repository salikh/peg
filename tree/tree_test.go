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

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		tree string
	}{
		{"(X)"},
		{`(X "xxx")`},
		{`(X :text("xxx"))`},
		{`(X :text("\""))`},
		{`(X :abc("xyz"))`},
		{`(X :abc("xyz") :cde("uvw"))`},
		{`(X (A) (B) (C))`},
		{`(X (A (B (C))))`},
		{`(X "x" (A "a" (B "b" (C "c"))))`},
		{`(X :Left(A) :Right(B))`},
		{`(X :Left(A (C)) :Right(B))`},
		{`(X :Left(A (C)) :Right(B (C) (D) (E) (F (G) (H (I) (J (K))))))`},
	}
	for _, tt := range tests {
		tree, err := Parse(tt.tree)
		if err != nil {
			t.Errorf("Tree %s deserialization returned error %s, want success", tt.tree, err)
			continue
		}
		got := tree.String()
		if got != tt.tree {
			t.Errorf("Tree %s deserialized differently: %s", tt.tree, got)
		}
	}
}

// TODO(salikh): Switch the canonical tree representation to a shorter one.
func TestNonCanonical(t *testing.T) {
	tests := []struct {
		tree string
		want string
	}{
		{tree: `(X text("xxx"))`, want: `(X "xxx")`},
	}
	for _, tt := range tests {
		tree, err := Parse(tt.tree)
		if err != nil {
			t.Errorf("Tree %s deserialization returned error %s, want success", tt.tree, err)
			continue
		}
		got := tree.String()
		if got != tt.want {
			t.Errorf("Tree %s deserialized differently: %s, want: %s", tt.tree, got, tt.want)
		}
	}
}

func TestExtract(t *testing.T) {
	tests := []struct {
		name string
		tree string
		expr string
		want string
		// TODO(salikh): Test errors too.
		//err string
	}{
		{tree: "(X)",
			expr: "text",
			want: ""},
		{tree: `(X text("xxx"))`,
			expr: "text", want: "xxx"},
		{tree: `(X text("xxx"))`,
			expr: "num", want: "0"},
		{tree: `(X text("\""))`,
			expr: "text", want: "\""},
		{tree: `(X (Y text("yy")))`,
			expr: "Y text", want: "yy"},
		{tree: `(X (Y text("yy")) (Y text("yyy")))`,
			expr: "Y text", want: "yy"},
		{tree: `(X (Y text("yy")) (Y text("yyy")))`,
			expr: "Y[0] text", want: "yy"},
		{tree: `(X (Y text("yy")) (Y text("yyy")))`,
			expr: "Y[1] text", want: "yyy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, err := Parse(tt.tree)
			if err != nil {
				t.Errorf("Tree %s deserialization returned error %s, want success", tt.tree, err)
				return
			}
			t.Logf("Tree:\n%s", tree)
			got, err := Extract(tree, tt.expr)
			if err != nil {
				t.Errorf("Extract(%s) returns error %s, want success", tt.expr, err)
				return
			}
			if got != tt.want {
				t.Errorf("Extract(%s) returned %s, want %s", tt.expr, got, tt.want)
			}
		})
	}
}

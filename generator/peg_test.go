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

package generator

import (
	"testing"

	"github.com/salikh/peg/tree"
)

func TestString(t *testing.T) {
	tests := []struct {
		*Grammar
		want string
	}{
		{nil, "(nil)"},
		{&Grammar{}, "(Grammar )"},
		{&Grammar{
			RuleNames: []string{"abc"},
			Rules: map[string]*Rule{"abc": &Rule{
				Ident: "abc",
			}},
		}, `(Grammar (Rule text("abc") (nil)))`},
		{&Grammar{
			RuleNames: []string{"abc"},
			Rules: map[string]*Rule{"abc": &Rule{
				Ident: "abc",
				RHS: &RHS{
					[][]*Term{
						[]*Term{
							&Term{
								NegPred: &Term{
									Literal: "abc",
								},
							},
						},
						[]*Term{
							&Term{
								Pred: &Term{
									Ident: "abc",
								},
							},
							&Term{
								Pred: &Term{
									Special: &Special{Rune: '.'},
								},
							},
						},
					},
				},
			}},
		}, `
		(Grammar (Rule text("abc") (RHS
		 (Choice (Term :NegPred(Term :Literal("abc"))))
		  (Choice
			 (Term :Pred(Term :Ident("abc")))
			 (Term :Pred(Term :Special(Special :Rune(".")))))
	  )))`},
		{&Grammar{
			RuleNames: []string{"abc"},
			Rules: map[string]*Rule{"abc": &Rule{
				Ident: "abc",
				RHS: &RHS{
					[][]*Term{
						[]*Term{
							&Term{
								Parens: &RHS{
									[][]*Term{
										[]*Term{
											&Term{
												Pred: &Term{
													Special: &Special{Rune: '.'},
												},
											},
										},
									},
								},
								Capture: &RHS{
									[][]*Term{
										[]*Term{
											&Term{
												Pred: &Term{
													Special: &Special{Rune: '.'},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}},
		}, `
		(Grammar (Rule text("abc") (RHS
		 (Choice (Term
		   :Capture(RHS (Choice (Term :Pred(Term :Special(Special :Rune("."))))))
			 :Parens(RHS (Choice (Term :Pred(Term :Special(Special :Rune("."))))))))
	  )))`},
	}
	for _, tt := range tests {
		got := tt.Grammar.String()
		gotPretty, err := tree.Pretty(got)
		if err != nil {
			t.Errorf("Grammar %#v produced invalid tree serialization string %q: %s",
				tt.Grammar, got, err)
			continue
		}
		wantPretty, err := tree.Pretty(tt.want)
		if err != nil {
			t.Errorf("Error in test, want invalid tree %q: %s", tt.want, err)
			continue
		}
		if gotPretty != wantPretty {
			t.Errorf("Grammar %#v.String() returns \n%s,\nwant\n%s", tt.Grammar, gotPretty, wantPretty)
		}
	}
}

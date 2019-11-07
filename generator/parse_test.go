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
	"regexp"
	"testing"

	"github.com/salikh/peg/tree"
)

type pegTest struct {
	// source is the PEG grammar source.
	source string
	// tree is the serialized expected raw parse tree.
	tree string
}

var pegTests = []pegTest{
	{`A <- .`, `
		(Grammar :top("1")
	   (Rule
		  (Ident text("A")) (RHS (Terms (Term (Special text(".")))))))`},
}

func TestParseTree(t *testing.T) {
	for _, tt := range pegTests {
		t.Logf("grammar source:\n%s\n---", tt.source)
		result, err := pegG.Parse(tt.source)
		if err != nil {
			t.Errorf("grammar.Parse(%s) returned error %s, want success",
				tt.source, err)
			continue
		}
		t.Logf("grammar tree:\n%s\n---", result.Tree)
		got := result.Tree.String()
		wantTree, err := tree.Parse(tt.tree)
		if err != nil {
			t.Errorf("error in test, parse tree unparseable: %s", err)
			continue
		}
		want := wantTree.String()
		if got != want {
			// TODO: compute a string diff.
			t.Errorf("Parse(%s) returned tree \n%s\n---, want \n%s\n---",
				tt.source, got, want)
		}
	}
}

type parseTest struct {
	// source is the PEG grammar source.
	source string
	// tree is the serialized expected semantic tree.
	tree string
}

var parseTests = []parseTest{
	{`A <- .`,
		`(Grammar
	    (Rule text("A") (RHS (Choice (Term :CharClass("[:any:]"))))))`},
	{`A <- .?`,
		`(Grammar
	    (Rule text("A") (RHS (Choice
			 (Term :Special(Special (Term :CharClass("[:any:]")) :Rune("?")))))))`},
	{`A <- .*`,
		`(Grammar
	    (Rule text("A") (RHS (Choice
			 (Term :Special(Special (Term :CharClass("[:any:]")) :Rune("*")))))))`},
	{`A <-.**`, // weird, but valid.
		`(Grammar
	    (Rule text("A") (RHS (Choice
			 (Term :Special(Special
			  (Term :Special(Special
				 (Term :CharClass("[:any:]")) :Rune("*"))) :Rune("*")))))))`},
	{`A <- .+`,
		`(Grammar
	    (Rule text("A") (RHS (Choice
			 (Term :Special(Special (Term :CharClass("[:any:]")) :Rune("+")))))))`},
	{`A <- !.`,
		`(Grammar
	    (Rule text("A") (RHS (Choice
			 (Term :NegPred(Term :CharClass("[:any:]")))))))`},
	{`A <- &.`,
		`(Grammar
	    (Rule text("A") (RHS (Choice
			 (Term :Pred(Term :CharClass("[:any:]")))))))`},
	{`A <- [x]`,
		`(Grammar
	    (Rule text("A") (RHS (Choice (Term :CharClass("x"))))))`},
	{`A <- [[:alpha:]]`,
		`(Grammar
	    (Rule text("A") (RHS (Choice (Term :CharClass("[:alpha:]"))))))`},
	{`A <- [[:any:]]`,
		`(Grammar
	    (Rule text("A") (RHS (Choice (Term :CharClass("[:any:]"))))))`},
	{`A <- "abc"`,
		`(Grammar
	    (Rule text("A") (RHS (Choice (Term :Literal("abc"))))))`},
	{`A <- 'abc'`,
		`(Grammar
	    (Rule text("A") (RHS (Choice (Term :Literal("abc"))))))`},
	{`A <-
	'abc'`,
		`(Grammar
	    (Rule text("A") (RHS (Choice (Term :Literal("abc"))))))`},
	{`A <- < "abc" > `,
		`(Grammar
	    (Rule text("A") (RHS (Choice
			 (Term :Capture(RHS (Choice (Term :Literal("abc")))))))))`},
	{`A <- <"abc"> `,
		`(Grammar
	    (Rule text("A") (RHS (Choice
			 (Term :Capture(RHS (Choice (Term :Literal("abc")))))))))`},
	{`A<-<"abc">`,
		`(Grammar
	    (Rule text("A") (RHS (Choice
			 (Term :Capture(RHS (Choice (Term :Literal("abc")))))))))`},
	{`A<-<"abc">.*`,
		`(Grammar
	    (Rule text("A") (RHS (Choice
			 (Term :Capture(RHS (Choice (Term :Literal("abc")))))
			 (Term :Special(Special (Term :CharClass("[:any:]")) :Rune("*")))
		 ))))`},
	{`A <- ("abc")`,
		`(Grammar
	    (Rule text("A") (RHS (Choice (Term :Parens
				(RHS (Choice (Term :Literal("abc"))))
			)))))`},
	/*
				FIXEM
		{`A <- ("abc\n\r")`,
			`(Grammar
		    (Rule text("A") (RHS (Choice (Term :Parens
					(RHS (Choice (Term :Literal("abc\n\r"))))
				)))))`},
	*/
	{`A <- ("abc" *)`,
		`(Grammar
	    (Rule text("A") (RHS (Choice (Term :Parens
			 (RHS (Choice
				(Term :Special(Special (Term :Literal("abc")) :Rune("*")))))
			)))))`},
	{`A <- ("abc" / . / .)`,
		`(Grammar
	    (Rule text("A") (RHS (Choice (Term :Parens
				(RHS (Choice (Term :Literal("abc")))
				     (Choice (Term :CharClass("[:any:]")))
						 (Choice (Term :CharClass("[:any:]"))))
			)))))`},
	{`A <- ! .`,
		`(Grammar
	    (Rule text("A") (RHS (Choice
			 (Term :NegPred(Term :CharClass("[:any:]")))))))`},
	{`A <- . [x]`,
		`(Grammar
	    (Rule text("A") (RHS (Choice
				(Term :CharClass("[:any:]")) (Term :CharClass("x"))))))`},
	{`A <- . / [x]`,
		`(Grammar
	    (Rule text("A") (RHS
			 (Choice (Term :CharClass("[:any:]")))
			 (Choice (Term :CharClass("x"))))))`},
	{`A <- . /
	[x]`,
		`(Grammar
	    (Rule text("A") (RHS
			 (Choice (Term :CharClass("[:any:]")))
			 (Choice (Term :CharClass("x"))))))`},
	{`A <- .
	/ [x]`,
		`(Grammar
	    (Rule text("A") (RHS
			 (Choice (Term :CharClass("[:any:]")))
			 (Choice (Term :CharClass("x"))))))`},
	{`A <- .
	/ [x]
	B <- .`,
		`(Grammar
	    (Rule text("A") (RHS
			 (Choice (Term :CharClass("[:any:]")))
			 (Choice (Term :CharClass("x")))))
			(Rule text("B") (RHS
			 (Choice (Term :CharClass("[:any:]"))))))`},
	{`A <- .

	/ [x]

	B <- .`,
		`(Grammar
	    (Rule text("A") (RHS
			 (Choice (Term :CharClass("[:any:]")))
			 (Choice (Term :CharClass("x")))))
			(Rule text("B") (RHS
			 (Choice (Term :CharClass("[:any:]"))))))`},
	{`A <- . * / [x] [y]`,
		`(Grammar
	    (Rule text("A") (RHS
			 (Choice (Term :Special(Special (Term :CharClass("[:any:]")) :Rune("*"))))
			 (Choice (Term :CharClass("x")) (Term :CharClass("y"))))))`},
	{`A <- .
	#B <- .`,
		`(Grammar
	    (Rule text("A") (RHS (Choice (Term :CharClass("[:any:]"))))))`},
	{`A <- .
	B <- .`,
		`(Grammar
	    (Rule text("A") (RHS (Choice (Term :CharClass("[:any:]")))))
	    (Rule text("B") (RHS (Choice (Term :CharClass("[:any:]"))))))`},
	{`A <- .

	B <- .`,
		`(Grammar
	    (Rule text("A") (RHS (Choice (Term :CharClass("[:any:]")))))
	    (Rule text("B") (RHS (Choice (Term :CharClass("[:any:]"))))))`},
	{`A <- . # sdfsdfadfadfas asdf <- asdfadf -> asdf

	B <- .`,
		`(Grammar
	    (Rule text("A") (RHS (Choice (Term :CharClass("[:any:]")))))
	    (Rule text("B") (RHS (Choice (Term :CharClass("[:any:]"))))))`},
	{`A <- .
   # asdfasdfasdf dsf <- asdf asdf -> asdf
	B <- .`,
		`(Grammar
	    (Rule text("A") (RHS (Choice (Term :CharClass("[:any:]")))))
	    (Rule text("B") (RHS (Choice (Term :CharClass("[:any:]"))))))`},
}

type invalidParseTest struct {
	source      string
	syntaxErr   string
	semanticErr string
}

var invalidParseTests = []invalidParseTest{
	{source: `A <- *`,
		semanticErr: `Special character '\*' cannot be first in the rule`},
	{source: `A <- +`,
		semanticErr: `Special character '\+' cannot be first in the rule`},
	{source: `A <- ?`,
		semanticErr: `Special character '\?' cannot be first in the rule`},
	{source: `A <- .`}, // No error.
	{source: `A <- (`,
		syntaxErr: `'\('`},
	{source: `A <- .  B <- .`,
		syntaxErr: `"<-`},
	{source: `A <- .

	/ [x] B <- .`, syntaxErr: `"<-`},
}

func TestParse2(t *testing.T) {
	for _, tt := range parseTests {
		t.Run(tt.source, func(t *testing.T) {
			t.Logf("grammar source:\n%s\n---", tt.source)
			result, err := pegG.Parse(tt.source)
			if err != nil {
				t.Errorf("grammar.Parse(%s) returned error %s, want success",
					tt.source, err)
				return
			}
			grammar, err := ConvertGrammar2(result.Tree)
			if err != nil {
				t.Errorf("Parse(%s) returned error %s, want success", tt.source, err)
				return
			}
			t.Logf("grammar:\n%s\n---", grammar)
			got := grammar.String()
			gotPretty, err := tree.Pretty(got)
			if err != nil {
				t.Errorf("grammar.String() returned invalid tree %s: %s", got, err)
			}
			wantPretty, err := tree.Pretty(tt.tree)
			if err != nil {
				t.Errorf("error in test, invalid wanted tree %s: %s", tt.tree, err)
				return
			}
			if gotPretty != wantPretty {
				// TODO: compute a string diff.
				t.Errorf("Parse(%s) returned tree \n%s\n---, want \n%s\n---",
					tt.source, gotPretty, wantPretty)
			}
		})
	}
}

func TestParseError(t *testing.T) {
	for _, tt := range invalidParseTests {
		t.Run(tt.source, func(t *testing.T) {
			t.Logf("grammar source:\n%s\n---", tt.source)
			result, sErr := pegG.Parse(tt.source)
			if sErr == nil && tt.syntaxErr != "" {
				t.Errorf("grammar.Parse(%s) returned success, want error [%s]",
					tt.source, tt.syntaxErr)
				return
			}
			if sErr != nil {
				if tt.syntaxErr == "" {
					t.Errorf("grammar.Parse(%s) returned error %s, want success",
						tt.source, sErr)
					return
				}
				re, err := regexp.Compile(tt.syntaxErr)
				if err != nil {
					t.Errorf("error in test, regexp /%s/ error: %s", tt.syntaxErr, err)
					return
				}
				if !re.MatchString(sErr.Error()) {
					t.Errorf("grammar.Parse(%s) returned error [%s], but want error [%s]",
						tt.source, sErr.Error(), tt.syntaxErr)
				}
				return
			}
			_, sErr = ConvertGrammar2(result.Tree)
			// Expect the error during
			if sErr == nil && tt.semanticErr != "" {
				t.Errorf("ConvertGrammar2(%s) returned success, want error [%s]",
					tt.source, tt.semanticErr)
				return
			}
			if sErr != nil {
				if tt.semanticErr == "" {
					t.Errorf("grammar.Parse(%s) returned error %s, want success",
						tt.source, sErr)
					return
				}
				re, err := regexp.Compile(tt.semanticErr)
				if err != nil {
					t.Errorf("error in test, regexp /%s/ error: %s", tt.semanticErr, err)
					return
				}
				if !re.MatchString(sErr.Error()) {
					t.Errorf("ConvertGrammar2(%s) returned error [%s], but want error [%s]",
						tt.source, sErr.Error(), tt.semanticErr)
				}
				return
			}
		})
	}
}

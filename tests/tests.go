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

// Package tests is a holder for various parser tests.
package tests

// InvalidGrammarTest defines a negative test for parser generator.
// The parser generator should return an error when generating a grammar.
type InvalidGrammarTest struct {
	Grammar string
}

// Outcome provides one test input for a constructed parser.
type Outcome struct {
	// Input is a string given to the parser as input.
	Input string
	// Ok indicates whether the constructed parser should accept
	// the input.
	Ok bool
}

// PositiveTest defines a positive test for parser generator.
// The grammar should successfully construct a parser, which should
// then accept and reject the provided inputs.
type PositiveTest struct {
	// Grammar contains a grammar to be tested. It must be a correct grammar.
	Grammar  string
	Outcomes []Outcome
}

// CaptureOutcome defines one test case with a capture.
type CaptureOutcome struct {
	// Input string.
	Input string
	// Ok is expected success value. If true, the input
	// must be parsed successfully.  If false, the input
	// must trigger parser error.
	Ok bool
	// Result is the expected value of captured text on
	// the root AST node. If parsed successfully, the top
	// tree node captured text must be equal to this
	// result.
	Result string
}

// CaptureTest defines one test for captures.
type CaptureTest struct {
	Grammar  string
	Outcomes []CaptureOutcome
}

// Invalid is an array of negative tests with invalid grammars.
var Invalid = []InvalidGrammarTest{
	{"Ident <- abc <- xyz"},
	{"#abc"},
	{"abc <- '"},
	{"abc <- \""},
	{"I <- ?"},
	{"I <- *"},
	{"I <- ("},
	{"I <- )"},
	{"I <- )("},
	{"I <- ('abc'"},
	{"I <- ( 'abc' ()"},
	{"I <- ( 'abc' ('x')"},
	{"A <- B"},
	{"A <- B \n B <- C"},
	{"I <- \\x"},
	//NYI:{"I <- [:xyz:]"},
	//{"I <- [^-z]"}, // no longer invalid.
	{"I <- [z-a]"},
	{"I <- &"},
	{"I <- !"},
}

// Positive is an array of positive tests.
var Positive = []PositiveTest{
	{
		Grammar: "Space1 <- ' '",
		Outcomes: []Outcome{
			{" ", true},
			{"", false},
			{"  ", false},
			{"x", false},
		},
	},
	{
		Grammar: "Space2 <- ' '",
		Outcomes: []Outcome{
			{" ", true},
			{"  ", false},
			{"", false},
			{"x", false},
		},
	},
	{
		Grammar: "Space3 <- '  '",
		Outcomes: []Outcome{
			{" ", false},
			{"  ", true},
			{"   ", false},
			{"", false},
			{"x", false},
		},
	},
	{
		Grammar: "Space4 <- . +",
		Outcomes: []Outcome{
			{"", false},
			{" ", true},
			{"  ", true},
			{"   ", true},
			{"x", true},
			{"xyz\n abc \n efg\n", true},
		},
	},
	{
		Grammar: "Space5 <- . *",
		Outcomes: []Outcome{
			{"", true},
			{" ", true},
			{"  ", true},
			{"   ", true},
			{"x", true},
			{"xyz\n abc \n efg\n", true},
		},
	},
	{
		Grammar: `Newline1 <- "\n"`,
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"  ", false},
			{"\n", true},
			{"\n\n", false},
			{"xyz\n abc \n efg\n", false},
		},
	},
	{
		Grammar: `Newline2 <- [\n]`,
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"  ", false},
			{"\n", true},
			{"\n\n", false},
			{"xyz\n abc \n efg\n", false},
		},
	},
	{
		Grammar: `Newline3 <- '\n'`,
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"  ", false},
			{"\n", false},
			{"\n\n", false},
			{"xyz\n abc \n efg\n", false},
			{"\\n", true},
		},
	},
	{
		Grammar: `Tab1 <- "\t"`,
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"\n\n", false},
			{"\t", true},
			{"\t\t", false},
			{"\txyz\n abc \n efg\n", false},
		},
	},
	{
		Grammar: `Tab2 <- "	"`, // Literal tab (0x9)
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"\n\n", false},
			{"\t", true},
			{"\t\t", false},
			{"\txyz\n abc \n efg\n", false},
		},
	},
	{
		Grammar: "Letter <- [a-z]",
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"ab", false},
			{"a", true},
			{"b", true},
			{"1", false},
			{"z", true},
			{"\txyz\n abc \n efg\n", false},
		},
	},
	{
		Grammar: "Space6 <- [\\n\\t ]",
		Outcomes: []Outcome{
			{"", false},
			{" ", true},
			{"\t", true},
			{"\n", true},
			{"  ", false},
			{"\txyz\n abc \n efg\n", false},
		},
	},
	{
		Grammar: "Caret1 <- [v^]",
		Outcomes: []Outcome{
			{"", false},
			{"^", true},
			{"v", true},
			{"^^", false},
			{"vv", false},
			{"\txyz\n abc \n efg\n", false},
		},
	},
	{
		Grammar: `String <- '"' ( '\"' / !'"' . )* '"'`,
		Outcomes: []Outcome{
			{``, false},
			{`"`, false},
			{`""`, true},
			{`" "`, true},
			{`"x"`, true},
			{`"xxxxx"`, true},
			{`"xx\"xxx"`, true},
			{`"xx\"x\"xx"`, true},
			{`"xx"x\"xx"`, false},
			{`"xx"x"xx"`, false},
			{`"xx"x"`, false},
			{`"xx\"x"`, true},
		},
	},
	{
		Grammar: "Caret2 <- [v-]",
		Outcomes: []Outcome{
			{"", false},
			{"^", false},
			{"v", true},
			{"-", true},
			{"^^", false},
			{"vv", false},
			{"--", false},
			{"\txyz\n abc \n efg\n", false},
		},
	},
	{
		Grammar: "Char <- [^a-x]",
		Outcomes: []Outcome{
			{"", false},
			{" ", true},
			{"a", false},
			{"x", false},
			{"z", true},
			{"aa", false},
			{"zz", false},
			{"\n\n", false},
			{"\t", true},
			{"\t\t", false},
			{"\txyz\n abc \n efg\n", false},
		},
	},
	{
		Grammar: "Ident1 <- [a-zA-Z_][a-zA-Z0-9_]*",
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"a", true},
			{"x", true},
			{"z", true},
			{"aa", true},
			{"Aa", true},
			{"A1", true},
			{"A1", true},
			{"A_1", true},
			{"A1_", true},
			{"_1_", true},
			{"1", false},
			{"1_", false},
			{"_1", true},
			{"zz", true},
			{"\n\n", false},
			{"\t", false},
			{"\t\t", false},
			{"\txyz\n abc \n efg\n", false},
		},
	},
	{
		Grammar: "Space7 <- \"  \"",
		Outcomes: []Outcome{
			{" ", false},
			{"  ", true},
			{"   ", false},
			{"", false},
			{"x", false},
		},
	},
	{
		Grammar: "Space8 <- 'xyz'",
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"x", false},
			{"xy", false},
			{"xyz", true},
			{"xyzt", false},
		},
	},
	{
		Grammar: "Space9 <- 'xy' 'z'",
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"x", false},
			{"xy", false},
			{"xyz", true},
			{"xyzt", false},
		},
	},
	{
		Grammar: "Space10 <- 'x' 'y' 'z'",
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"x", false},
			{"xy", false},
			{"xyz", true},
			{"xyzt", false},
		},
	},
	{
		Grammar: "Space11 <- 'x' 'y' '*' 'z'",
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"x", false},
			{"xy", false},
			{"xz", false},
			{"xyz", false},
			{"xy*z", true},
			{"xyzt", false},
			{"xyyzt", false},
		},
	},
	{
		Grammar: "Space12 <- 'x' 'y' + 'z'",
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"x", false},
			{"xy", false},
			{"xz", false},
			{"xyz", true},
			{"xyyz", true},
			{"xyzt", false},
			{"xyyzt", false},
			{"xyyyyz", true},
		},
	},
	{
		Grammar: "Space13 <- 'x' 'y' * 'z'",
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"x", false},
			{"xy", false},
			{"xz", true},
			{"xyz", true},
			{"xyyz", true},
			{"xyzt", false},
			{"xyyzt", false},
		},
	},
	{
		Grammar: "Space14 <- 'x' 'y' ? 'z'",
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"x", false},
			{"xy", false},
			{"xz", true},
			{"xyz", true},
			{"xyyz", false},
			{"xyzt", false},
		},
	},
	{
		Grammar: "Space15 <- 'x' ( 'y' ) 'z'",
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"x", false},
			{"xy", false},
			{"xz", false},
			{"xyz", true},
			{"xyyz", false},
			{"xyzt", false},
		},
	},
	{
		Grammar: "Space16 <- 'x' ( 'y' ) * 'z'",
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"x", false},
			{"xy", false},
			{"xz", true},
			{"yz", false},
			{"xyz", true},
			{"xyyz", true},
			{"xyzt", false},
		},
	},
	{
		Grammar: "Space17 <- 'x' ( 'y' 'z' ) * 't' ",
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"x", false},
			{"xy", false},
			{"xz", false},
			{"xt", true},
			{"yz", false},
			{"xyz", false},
			{"xyyz", false},
			{"xyzt", true},
			{"xyzyzt", true},
			{"xyzyt", false},
			{"xzyzt", false},
			{"xyzyzyzt", true},
		},
	},
	{
		Grammar: "Space18 <- 'x' ( ('y')* ('z')* ) * 't' ",
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"x", false},
			{"xy", false},
			{"xz", false},
			{"xt", true},
			{"yz", false},
			{"xyz", false},
			{"xyyz", false},
			{"xt", true},
			{"xyt", true},
			{"xzt", true},
			{"xyzt", true},
			{"xyzyzt", true},
			{"xyzyt", true},
			{"xzyzt", true},
			{"xyzyzyzt", true},
			{"xyyyzzzt", true},
		},
	},
	{
		Grammar: "Ident2 <- Space 'a'+ \n Space <- ' '*",
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"a", true},
			{"aa", true},
			{"xa", false},
			{"ax", false},
			{"  a", true},
			{"  aaa", true},
			{"  aaa ", false},
			{"    a a", false},
			{"     aa", true},
		},
	},
	{
		Grammar: "Ident3 <- Space 'a'+ / Space 'b'+ \n Space <- ' '*",
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"a", true},
			{"b", true},
			{"ab", false},
			{"aa", true},
			{"bb", true},
			{"ab", false},
			{"xa", false},
			{"ax", false},
			{"  a", true},
			{"  b", true},
			{"  aaa", true},
			{"  bbb", true},
			{"  aaa ", false},
			{"  aab ", false},
			{"  bbb ", false},
			{"    a a", false},
			{"    a b", false},
			{"    b a", false},
			{"     aa", true},
			{"     bb", true},
		},
	},
	{
		Grammar: `Quoted1 <- "'" ( ! "'" . )* "'"`,
		Outcomes: []Outcome{
			{"", false},
			{" ", false},
			{"a", false},
			{"''", true},
			{"' '", true},
			{"'a'", true},
			{"'abc'", true},
			{"'''", false},
			{" ''", false},
			{"'' ", false},
			{" 'abc' ", false},
		},
	},
	{
		Grammar: "Quoted2 <- 'a' ! 'b' .* ",
		Outcomes: []Outcome{
			{"", false},
			{"a", true},
			{"ab", false},
			{"aa", true},
			{"acb", true},
			{"abcd", false},
		},
	},
	{
		Grammar: `ABString <-  A* B* _
								A <- _ 'a'*
								B <- _ 'b'*
								_ <- (' ' ' '* / "\n" "\n"*)*`,
		Outcomes: []Outcome{
			{"", true},
			{"a", true},
			{" a", true},
			{"a ", true},
			{" a ", true},
			{"b", true},
			{" b", true},
			{"b ", true},
			{" b ", true},
			{"ab", true},
			{" a b ", true},
			{" aaa bbbb ", true},
			{"c", false},
			{"\n", true},
			{"\n\n\n", true},
			{"   \n\n\n", true},
			{"   \n  \n     \n   \n   \n", true},
			{"   \n  \naa     \nb   \n   \n", true},
			{"   \n  \na a     \nb   \n   \n", true},
			{"   \n  \naa     \nb   \n b  \n", true},
			{"   \n  \naa  x   \nb   \n   \n", false},
		},
	},
	{
		Grammar: "Ident4 <- [[:alpha:]][[:alnum:]][[:digit:]]",
		Outcomes: []Outcome{
			{"", false},
			{"abc", false},
			{"ab1", true},
			{"123", false},
			{"a23", true},
			{"__3", false},
			{"a_3", false},
			{"ab1\n", false},
		},
	},
}

// Capture is an array of capture tests.
var Capture = []CaptureTest{
	{
		Grammar: "X <- 'x' < 'y'* > 'z' ",
		Outcomes: []CaptureOutcome{
			{"", false, ""},
			{" ", false, ""},
			{"x", false, ""},
			{"xy", false, ""},
			{"xz", true, ""},
			{"xt", false, ""},
			{"yz", false, ""},
			{"xyz", true, "y"},
			{"xyyz", true, "yy"},
			{"xyyytyyyyz", false, ""},
			{"xyyyzt", false, ""},
		},
	},
	{
		Grammar: "X <- Space < Ident > Space \n Space <- ' '* \n Ident <- ('x' / 'y' / 'z')+",
		Outcomes: []CaptureOutcome{
			{"", false, ""},
			{" ", false, ""},
			{"x", true, "x"},
			{" x", true, "x"},
			{"x ", true, "x"},
			{" x ", true, "x"},
			{"xy", true, "xy"},
			{"xz", true, "xz"},
			{"xt", false, ""},
			{"yz", true, "yz"},
			{"xyz", true, "xyz"},
			{"xyyz", true, "xyyz"},
			{"xyyyyyyyz", true, "xyyyyyyyz"},
			{"xyyyzt", false, ""},
		},
	},
	{
		Grammar: "X <- _ A (_ A)* _\nA <- 'a'+\n_ <- ' '*",
		Outcomes: []CaptureOutcome{
			{"", false, ""},
			{" ", false, ""},
			{"a", true, ""},
			{"aaa", true, ""},
			{" a", true, ""},
			{" a ", true, ""},
			{"aaa ", true, ""},
			{" aaa ", true, ""},
			{"a a", true, ""},
			{"a  a", true, ""},
			{"a a  ", true, ""},
			{"   a    a    a   ", true, ""},
			{"a       a", true, ""},
			{"a    aa", true, ""},
			{"a   a   a   aa", true, ""},
		},
	},
}

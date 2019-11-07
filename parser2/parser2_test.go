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

package parser2

import (
	"flag"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"testing"

	"github.com/salikh/peg/compat/runfiles"
	"github.com/salikh/peg/tests"
)

func TestInvalidGrammars(t *testing.T) {
	for _, tt := range tests.Invalid {
		_, err := New(tt.Grammar, nil)
		if err == nil {
			t.Errorf("New(%q) returns success, want error", tt.Grammar)
		} else {
			//t.Logf("New(%q) returns error {%s}", tt.Grammar, err)
		}
	}
}

func testParser(t *testing.T, test tests.PositiveTest) {
	g, err := New(test.Grammar, nil)
	if err != nil {
		t.Errorf("New(%q) returns error %q, want success", test.Grammar, err)
		return
	}
	for _, tt := range test.Outcomes {
		t.Run(tt.Input, func(t *testing.T) {
			//t.Logf("Grammar:\n%s", test.Grammar)
			//t.Logf("Input: [%s]", tt.Input)
			result, err := g.Parse(tt.Input)
			//t.Logf("Result: err{%s}, Tree: %s", err, result.Tree)
			if tt.Ok && err != nil {
				t.Errorf("New(%q).Parse(%q) returns error %s, want success",
					test.Grammar, tt.Input, err)
				return
			}
			if !tt.Ok && err == nil {
				t.Errorf("New(%q).Parse(%q) returns success, want error",
					test.Grammar, tt.Input)
				return
			}
			if err == nil {
				if result.Tree == nil {
					t.Errorf("New(%q).Parse(%q) returns nil Tree, want non nil: %#v", test.Grammar, tt.Input, result)
					return
				}
				result.ComputeContent()
				got, err := result.Tree.ReconstructContent()
				if err != nil {
					t.Errorf("New(%q).Parse(%q).ReconstructContent() returns error %q, want success",
						test.Grammar, tt.Input, err)
					return
				}
				if got != tt.Input {
					t.Errorf("New(%q).Parse(%q).ReconstructContent() returns %q, not equal to input",
						test.Grammar, tt.Input, got)
				}
			}
		})
	}
}

func TestPositiveGrammars(t *testing.T) {
	for _, test := range tests.Positive {
		testParser(t, test)
	}
}

var dataTest = flag.String("data_test", "", "The names of the tests in datatest/ to run.")

func TestData(t *testing.T) {
	dirname := runfiles.Path("github.com/salikh/peg/tests/testdata")
	dir, err := os.Open(dirname)
	if err != nil {
		t.Errorf("Cannot find testdata: %s", err)
		return
	}
	defer dir.Close()
	names, err := dir.Readdirnames(0)
	if err != nil {
		t.Errorf("Cannot list testdata: %s", err)
		return
	}
	re, err := regexp.Compile("\\.g$")
	if err != nil {
		t.Errorf("Error in regexp: %s", err)
		return
	}
	gg := make(map[string]*Grammar)
	for _, name := range names {
		if re.Match([]byte(name)) {
			source, err := ioutil.ReadFile(path.Join(dirname, name))
			if err != nil {
				t.Errorf("Error reading %q: %s", name, err)
				continue
			}
			g, err := New(string(source), nil)
			if err != nil {
				t.Errorf("New(%q) returns error %s, want success", source, err)
				continue
			}
			gg[name[0:len(name)-2]] = g
		}
	}
	re2, err := regexp.Compile("^[a-zA-Z0-9_-]*")
	if err != nil {
		t.Errorf("error in re2: %s", err)
		return
	}
	for _, name := range names {
		base := re2.FindString(name)
		if base == "" {
			continue
		}
		if ok, _ := regexp.MatchString("\\.g$", name); ok {
			// skip grammar files
			continue
		}
		if g, ok := gg[base]; ok {
			if *dataTest != "" && name != *dataTest {
				continue
			}
			contents, err := ioutil.ReadFile(path.Join(dirname, name))
			t.Logf("Testing %s\n", name)
			neg, err := regexp.MatchString("[nN]eg", name)
			if err != nil {
				t.Errorf("1")
				continue
			}
			_, err = g.Parse(string(contents))
			if neg && err == nil {
				t.Errorf("%s: got success, want error", name)
				continue
			} else if !neg && err != nil {
				t.Errorf("New(%q).Parse(%q)\n%s: got error %s, want success",
					g.Source, contents, name, err)
			}
			//log.Infof("%s: ok", name)

		} else {
			t.Errorf("File %q does not have a grammar %q", name, base)
		}
	}
}

func testParserCapture(t *testing.T, test tests.CaptureTest) {
	g, err := New(test.Grammar, nil)
	if err != nil {
		t.Errorf("New(%q) returns error %q, want success", test.Grammar, err)
		return
	}
	for _, tt := range test.Outcomes {
		r, err := g.Parse(tt.Input)
		if tt.Ok && err != nil {
			t.Errorf("New(%q).Parse(%q) returns error %s, want success",
				test.Grammar, tt.Input, err)
			continue
		}
		if !tt.Ok && err == nil {
			t.Errorf("New(%q).Parse(%q) returns success, want error",
				test.Grammar, tt.Input)
			continue
		}
		if err != nil {
			continue
		}
		if r.Tree == nil {
			if tt.Result != "" {
				t.Errorf("New(%q).Parse(%q) returns empty tree, want %q",
					test.Grammar, tt.Input, tt.Result)
			}
			continue
		}
		if r.Tree.Text != tt.Result {
			t.Errorf("New(%q).Parse(%q) returns %q, want %q",
				test.Grammar, tt.Input, r.Tree.Text, tt.Result)
		}
	}
}

func TestParserCapture(t *testing.T) {
	for _, test := range tests.Capture {
		//t.Logf("Grammar:\n%s", test.Grammar)
		testParserCapture(t, test)
	}
}

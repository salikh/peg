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

package charclass

import (
	"reflect"
	"testing"
	"unicode"
)

func TestParseCanonical(t *testing.T) {
	tests := []struct {
		input string
		want  *CharClass
	}{
		{"abc", &CharClass{Map: map[rune]bool{'a': true, 'b': true, 'c': true}}},
		{`\t `, &CharClass{Map: map[rune]bool{' ': true, '\t': true}}},
		{"-", &CharClass{Map: map[rune]bool{'-': true}}},
		{"a-", &CharClass{Map: map[rune]bool{'-': true, 'a': true}}},
		{`a-c`, &CharClass{
			RangeTable: &unicode.RangeTable{R16: []unicode.Range16{unicode.Range16{'a', 'c', 1}}}}},
		{`a-c-`, &CharClass{
			Map:        map[rune]bool{'-': true},
			RangeTable: &unicode.RangeTable{R16: []unicode.Range16{unicode.Range16{'a', 'c', 1}}}}},
		{`c-a`, nil},
		{`a-a`, nil},
		{`a-ct-tx-z`, nil},
		{`a-co-px-z`, &CharClass{
			RangeTable: &unicode.RangeTable{R16: []unicode.Range16{
				unicode.Range16{'a', 'c', 1},
				unicode.Range16{'o', 'p', 1},
				unicode.Range16{'x', 'z', 1}}}}},
		{`a-ce-f`, &CharClass{
			RangeTable: &unicode.RangeTable{R16: []unicode.Range16{unicode.Range16{'a', 'c', 1}, unicode.Range16{'e', 'f', 1}}}}},
		{`A-Za-z`, &CharClass{
			RangeTable: &unicode.RangeTable{R16: []unicode.Range16{unicode.Range16{'A', 'Z', 1}, unicode.Range16{'a', 'z', 1}}}}},
		{`_A-Za-z`, &CharClass{
			Map:        map[rune]bool{'_': true},
			RangeTable: &unicode.RangeTable{R16: []unicode.Range16{unicode.Range16{'A', 'Z', 1}, unicode.Range16{'a', 'z', 1}}}}},
		{`_0-9A-Za-z`, &CharClass{
			Map: map[rune]bool{'_': true},
			RangeTable: &unicode.RangeTable{R16: []unicode.Range16{
				unicode.Range16{'0', '9', 1},
				unicode.Range16{'A', 'Z', 1},
				unicode.Range16{'a', 'z', 1},
			}}}},
		{`^a-ce-f`, &CharClass{
			Negated:    true,
			RangeTable: &unicode.RangeTable{R16: []unicode.Range16{unicode.Range16{'a', 'c', 1}, unicode.Range16{'e', 'f', 1}}}}},
		{`a-ce-f^`, &CharClass{
			Map:        map[rune]bool{'^': true},
			RangeTable: &unicode.RangeTable{R16: []unicode.Range16{unicode.Range16{'a', 'c', 1}, unicode.Range16{'e', 'f', 1}}}}},
		{`\b\t\n\r`, &CharClass{Map: map[rune]bool{'\n': true, '\t': true, '\b': true, '\r': true}}},
		{`\n-\r`, &CharClass{
			RangeTable: &unicode.RangeTable{R16: []unicode.Range16{
				unicode.Range16{10, 13, 1}}}}},
		{`\n-`, &CharClass{
			Map: map[rune]bool{'-': true, '\x0a': true},
		}},
		{`^\n-\r`, &CharClass{
			Negated: true,
			RangeTable: &unicode.RangeTable{R16: []unicode.Range16{
				unicode.Range16{10, 13, 1},
			},
			}}},
		{`\x0-\x0d`, nil},
		{"А-Я", &CharClass{RangeTable: &unicode.RangeTable{
			R16: []unicode.Range16{
				unicode.Range16{0x410, 0x42f, 1},
			},
		}}},
		{"А-Я-", &CharClass{
			Map: map[rune]bool{'-': true},
			RangeTable: &unicode.RangeTable{R16: []unicode.Range16{
				unicode.Range16{0x410, 0x42f, 1},
			},
			}}},
		{"А-Я-", &CharClass{
			Map: map[rune]bool{'-': true},
			RangeTable: &unicode.RangeTable{R16: []unicode.Range16{
				unicode.Range16{0x410, 0x42f, 1},
			},
			}}},
		{"\u0410-\u042f", &CharClass{RangeTable: &unicode.RangeTable{
			R16: []unicode.Range16{
				unicode.Range16{0x410, 0x42f, 1},
			},
		}}},
		{`\U00101410-\U0010142f`, &CharClass{RangeTable: &unicode.RangeTable{
			R32: []unicode.Range32{
				unicode.Range32{0x101410, 0x10142f, 1},
			},
		}}},
		{`日-本`, &CharClass{RangeTable: &unicode.RangeTable{
			R16: []unicode.Range16{
				unicode.Range16{0x65E5, 0x672C, 1},
			},
		}}},
		{"日-本", &CharClass{RangeTable: &unicode.RangeTable{
			R16: []unicode.Range16{
				unicode.Range16{0x65E5, 0x672C, 1},
			},
		}}},
		{"^日-本", &CharClass{
			Negated: true,
			RangeTable: &unicode.RangeTable{
				R16: []unicode.Range16{
					unicode.Range16{0x65E5, 0x672C, 1},
				},
			}}},
		{"[:alnum:]", &CharClass{Special: "[:alnum:]"}},
		{"[:alpha:]", &CharClass{Special: "IsLetter"}},
		{"[:digit:]", &CharClass{Special: "IsDigit"}},
		{"[:cntrl:]", &CharClass{Special: "IsControl"}},
		{"[:punct:]", &CharClass{Special: "IsPunct"}},
		{"[:graph:]", &CharClass{Special: "IsGraphic"}},
		{"[:upper:]", &CharClass{Special: "IsUpper"}},
		{"[:lower:]", &CharClass{Special: "IsLower"}},
		{"[:space:]", &CharClass{Special: "IsSpace"}},
		{"[:any:]", &CharClass{Special: "[:any:]"}}, // Non-standard
		{"[:xxx:]", nil},
	}
	for _, tt := range tests {
		got, err := Parse(tt.input)
		if tt.want != nil && err != nil {
			t.Errorf("Parse(%q) returned error %s, want success", tt.input, err)
			continue
		}
		if tt.want == nil && err == nil {
			t.Errorf("Parse(%q) returned success, want error", tt.input)
			continue
		}
		if !reflect.DeepEqual(tt.want, got) {
			if !reflect.DeepEqual(tt.want.Map, got.Map) {
				t.Errorf("Parse(%q) got map %#v, want %#v", tt.input, got.Map, tt.want.Map)
				continue
			}
			if !reflect.DeepEqual(tt.want.RangeTable, got.RangeTable) {
				t.Errorf("Parse(%q) got range table %#v, want %#v", tt.input, got.RangeTable, tt.want.RangeTable)
				continue
			}
		}
		if tt.want == nil {
			continue
		}
		back := got.String()
		if tt.input != back {
			t.Errorf("Parse(%q).String() returned %q, want %q", tt.input, back, tt.input)
		}
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		input     string
		canonical string
	}{
		{"abc", "abc"},
		{"cab", "abc"},
		{`x-zo-pa-c`, `a-co-px-z`},
		{" \t", `\t `},
		{`\t `, `\t `},
		{"-a", `a-`},
		{"-a-c", `a-c-`},
		{"a-zA-Z", "A-Za-z"},
		{"a-zA-Z_", "_A-Za-z"},
		{"a-zA-Z-", `A-Za-z-`},
		{"-a-zA-Z", `A-Za-z-`},
		{`\-a-zA-Z`, `A-Za-z-`},
		{`a-zA-Z\-`, `A-Za-z-`},
		{`a-z\-A-Z`, `A-Za-z-`},
		{`a-zA-Z0-9_`, `_0-9A-Za-z`},
		{`a-c^e-f`, `a-ce-f^`},
		{`\^a-ce-f`, `a-ce-f^`},
		{`\^a-ce-f-`, `a-ce-f^-`},
		{`-^a-ce-f`, `a-ce-f^-`},
		{`\n\b\t\r`, `\b\t\n\r`},
		{`\x0a-\x0d`, `\n-\r`},
		{"\x0a-\x0d", `\n-\r`},
		{`\x0a-`, `\n-`},
		{`-\x0a`, `\n-`},
		{`\^\x0a-\x0d`, `\n-\r^`},
		{`\x0a-\x0d^`, `\n-\r^`},
		{`^\x0a-\x0d`, `^\n-\r`},
		{`^^\x0a-\x0d`, `^\n-\r^`},
		{`^^\x0a-\x0d`, `^\n-\r^`},
		{`^^^\x0a-\x0d`, `^\n-\r^`},
		{`^^^^\x0a-\x0d`, `^\n-\r^`},
		{"-А-Я", "А-Я-"},
		{`\-А-Я`, "А-Я-"},
		{`А-Я\-`, "А-Я-"},
		{`\u0410-\u042f`, "А-Я"},
		{`^\u0410-\u042f`, "^А-Я"},
		{`^^\u0410-\u042f`, "^А-Я^"},
		{`\u0410-\u042f^`, "А-Я^"},
		{`^\u0410-\u042f^`, "^А-Я^"},
		{"\U00101410-\U0010142f", `\U00101410-\U0010142f`},
	}
	for _, tt := range tests {
		got, err := Parse(tt.input)
		if err != nil {
			t.Errorf("Parse(%q) returned error %s, want success", tt.input, err)
			continue
		}
		back := got.String()
		if tt.canonical != back {
			t.Errorf("Parse(%q).String() returned %q, want %q", tt.input, back, tt.canonical)
		}
	}
}

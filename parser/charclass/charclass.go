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

// Package charclass provide functions for handling character class regular expressions.
package charclass

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// CharClass represents a char class
type CharClass struct {
	// Map represents individual elements
	Map map[rune]bool
	// RangeTable represents ranges
	*unicode.RangeTable
	// Negated indicates that the char class expression is negated.
	Negated bool
	// Special is empty or otherwise indicates a special class, with expectation that
	// method unicode.${Special} exists. The regex notation taken from GNU Grep
	// is translated into unicode package name conventions (e.g. "[:alpha:]" -> "IsLetter").
	// "[:alnum:]" is a special special case and is represented by the value "[:alnum:]" without translation.
	Special string
}

var specialClasses = map[string]string{
	"[:alpha:]": "IsLetter",
	"[:digit:]": "IsNumber",
	"[:space:]": "IsSpace",
	"[:lower:]": "IsLower",
	"[:upper:]": "IsUpper",
	"[:punct:]": "IsPunct",
	"[:print:]": "IsPrint",
	"[:graph:]": "IsGraphic",
	"[:cntrl:]": "IsControl",
	"[:alnum:]": "[:alnum:]",
	"[:any:]":   "[:any:]",
}

// Parse parses a charclass string.
// TODO(salikh): Support mixed special classes (both specials and regular
// chars).
func Parse(arg string) (*CharClass, error) {
	if len(arg) == 0 {
		return nil, errors.New("empty char class")
	}
	if arg[0] == '^' {
		if len(arg) == 1 {
			return &CharClass{
				Map: map[rune]bool{
					'^': true,
				},
			}, nil
		}
		skip := 1
		for skip < len(arg) && arg[skip] == '^' {
			skip++
		}
		r, err := Parse(arg[1:])
		if err != nil {
			return nil, err
		}
		// Parse returned a fresh CharClass instance, so it is fine to mutate it in place.
		r.Negated = true
		if skip > 1 {
			if r.Map == nil {
				r.Map = make(map[rune]bool)
			}
			r.Map['^'] = true
		}
		return r, nil
	}
	if arg[0] == '[' && arg[len(arg)-1] == ']' {
		special, ok := specialClasses[arg]
		if !ok {
			return nil, fmt.Errorf("unknown char class: %q", arg)
		}
		return &CharClass{
			Special: special,
		}, nil
	}
	var last rune = 0
	var start rune = 0
	ret := &CharClass{}
	for pos := 0; pos < len(arg); {
		r, w := utf8.DecodeRuneInString(arg[pos:])
		if r == utf8.RuneError {
			return nil, fmt.Errorf("error parsing utf8 rune at pos %d: %q", pos, arg)
		}
		if r == '-' {
			// Treat the leading or tail '-' as a plain rune.
			if pos != 0 && pos+w != len(arg) {
				// Mark the start of the range.
				start = last
				last = 0
				pos += w
				continue
			}
		}
		if r == '\\' {
			if pos+1 < len(arg) {
				switch arg[pos+1] {
				case '^', '-', '[', ']':
					// Special charclass-specific non-standard escapes.
					r = rune(arg[pos+1])
					w = 2
				default:
					val, _, tail, err := strconv.UnquoteChar(arg[pos:], 0)
					if err != nil {
						return nil, fmt.Errorf("error parsing escape at pos %d in %q: %s", pos, arg, err)
					}
					r = val
					w = len(arg) - pos - len(tail)
				}
				// Fallthrough and use r as in normal case.
			}
		}
		if start != 0 {
			// Close the range.
			if r <= start {
				return nil, fmt.Errorf("invalid interval in %c-%c in %q", start, r, arg)
			}
			if ret.RangeTable == nil {
				ret.RangeTable = &unicode.RangeTable{}
			}
			if start >= 1<<16 {
				ret.RangeTable.R32 = append(ret.RangeTable.R32, unicode.Range32{uint32(start), uint32(r), 1})
			} else if r < 1<<16 {
				ret.RangeTable.R16 = append(ret.RangeTable.R16, unicode.Range16{uint16(start), uint16(r), 1})
			} else {
				return nil, fmt.Errorf("%q: invalid char range across 16-bit and 32-bit boundary: %d to %d", arg, start, r)
			}
			pos += w
			start = 0
			last = 0
			continue
		}
		if last != 0 {
			if ret.Map == nil {
				ret.Map = make(map[rune]bool)
			}
			ret.Map[last] = true
		}
		last = r
		pos += w
	}
	if last != 0 {
		if ret.Map == nil {
			ret.Map = make(map[rune]bool)
		}
		ret.Map[last] = true
	}
	if ret.RangeTable != nil && len(ret.RangeTable.R16) > 0 {
		sort.Slice(ret.RangeTable.R16, func(i, j int) bool {
			return ret.RangeTable.R16[i].Lo < ret.RangeTable.R16[j].Lo
		})
	}
	if ret.RangeTable != nil && len(ret.RangeTable.R32) > 0 {
		sort.Slice(ret.RangeTable.R32, func(i, j int) bool {
			return ret.RangeTable.R32[i].Lo < ret.RangeTable.R32[j].Lo
		})
	}
	return ret, nil
}

func runeToString(c rune) string {
	q := strconv.QuoteRune(c)
	return q[1 : len(q)-1]
}

// String converts the CharClass instance into a canonical string
// representation that can be parsed back to the CharClass.
func (cc *CharClass) String() string {
	if cc == nil {
		return "nil"
	}
	var ret []string
	if cc.Negated {
		ret = append(ret, "^")
	}
	if cc.Special != "" {
		for k, v := range specialClasses {
			if cc.Special == v {
				return k
			}
		}
	}
	var runes []int
	for c := range cc.Map {
		if c == '-' || c == '^' {
			continue
		}
		runes = append(runes, int(c))
	}
	sort.Ints(runes)
	for _, c := range runes {
		if c == ']' {
			ret = append(ret, "\\]")
			continue
		}
		ret = append(ret, runeToString(rune(c)))
	}
	if cc.RangeTable != nil {
		// Assume stride is always 1.
		for _, r := range cc.RangeTable.R16 {
			ret = append(ret, runeToString(rune(r.Lo)),
				"-", runeToString(rune(r.Hi)))
		}
		for _, r := range cc.RangeTable.R32 {
			ret = append(ret, runeToString(rune(r.Lo)),
				"-", runeToString(rune(r.Hi)))
		}
	}
	if cc.Map['^'] {
		ret = append(ret, "^")
	}
	if cc.Map['-'] {
		ret = append(ret, "-")
	}
	return strings.Join(ret, "")
}

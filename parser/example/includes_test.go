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

package includes

import (
	"reflect"
	"testing"

	"github.com/salikh/peg/tree"
)

func TestSemantic(t *testing.T) {
	input := `
		(Source
			(Lines (Line text("// Arbitrary line")))
			(Includes
				(IncludeBlock
			    (Include text("stdio.h") (QuoteOpen text("<")) (QuoteClose text(">"))))
			  (IncludeBlock
					(Include text("my.h") (QuoteOpen text("\"")) (QuoteClose text("\"")))
					(Include text("my/flags.h") (QuoteOpen text("\"")) (QuoteClose text("\"")))
					(Include text("my/init.h") (QuoteOpen text("\"")) (QuoteClose text("\"")))))
			(Lines (Line text("// More lines")))
			(Using text("std::string"))
			(Lines (Line text("// Yet more lines")))
			(Using text("std::vector"))
		)`
	tr, err := tree.Parse(input)
	if err != nil {
		t.Errorf("error in test: %s", err)
		return
	}
	t.Logf("tree input:\n%s", tr)
	got, err := Convert(tr)
	if err != nil {
		t.Errorf("Convert(%s) returned error %s, want success", input, err)
		return
	}
	want := Source{
		Includes: []IncludeBlock{
			[]Include{
				Include{
					QuoteOpen: '<',
					Text:      "stdio.h",
				},
			},
			[]Include{
				Include{
					QuoteOpen: '"',
					Text:      "my.h",
				},
				Include{
					QuoteOpen: '"',
					Text:      "my/flags.h",
				},
				Include{
					QuoteOpen: '"',
					Text:      "my/init.h",
				},
			},
		},
		Using: []Using{
			Using{
				Text: "std::string",
			},
			Using{
				Text: "std::vector",
			},
		},
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("Convert(%s) returned %v, want %v", input, got, want)
	}
}

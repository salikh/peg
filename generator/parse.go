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

// Parse parses the PEG grammar source into a grammar.
func Parse2(input string) (*Grammar, error) {
	result, err := pegG.Parse(input)
	if err != nil {
		return nil, err
	}
	return ConvertGrammar2(result.Tree)
}

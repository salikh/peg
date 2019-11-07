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

package main

import (
	"flag"
	"io/ioutil"

	log "github.com/golang/glog"
	"github.com/salikh/peg/generator"
	"github.com/salikh/peg/parser"
)

var (
	grammarFlag = flag.String("grammar", "", "The path to the grammar file.")
	userSource  = flag.String("user_source", "", "The path to the go source file with data types. Optional.")
	outputFlag  = flag.String("output", "", "The path to write the parser Go source.")
	packageName = flag.String("package", "gen", "The name of the package to generate.")
)

func main() {
	flag.Parse()
	if *grammarFlag == "" {
		log.Exitf("--grammar must not be empty.")
	}
	if *outputFlag == "" {
		log.Exitf("--output must not be empty.")
	}
	grammar, err := ioutil.ReadFile(*grammarFlag)
	if err != nil {
		log.Exitf("Cannot read the grammar from %q: %s", *grammarFlag, err)
	}
	_, err = parser.New(string(grammar))
	if err != nil {
		log.Exitf("Error parsing the grammar file %q: %s", *grammarFlag, err)
	}
	g, err := generator.New(string(grammar))
	if err != nil {
		log.Exitf("Error parsing the PEG: %s", err)
	}
	if *userSource != "" {

		// ZZZ: For the time being, do not run the generator when having userSource.
		return
	}
	output, err := g.Generate(*packageName)
	if err != nil {
		log.Exitf("Error generating the parser: %s", err)
	}
	err = ioutil.WriteFile(*outputFlag, []byte(output), 0755)
	if err != nil {
		log.Exitf("Error writing the output to %q: %s", *outputFlag, err)
	}
}

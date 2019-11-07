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

// Package runfiles provides access to files containted in the source tree.
package runfiles

import (
	"os"
	"path"
	"regexp"
)

// Path returns an absolute path using a relative path from $GOPATH/src
func Path(p string) string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return p
		}
		gopath = path.Join(regexp.MustCompile("/src/.*$").ReplaceAllString(cwd, ""), p)
	}
	return path.Join(gopath, "src", p)
}

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

package workspace

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestFindDir(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "workspace_test")
	if err != nil {
		t.Errorf("error creating temp dir: %s", err)
		return
	}
	// We assume no .git or github.com in tmpDir.
	a := filepath.Join(tmpDir, "a")
	a_b := filepath.Join(a, "b")
	a_b_c := filepath.Join(a_b, "c")
	a_git := filepath.Join(tmpDir, "a", ".git")
	a_b_github := filepath.Join(tmpDir, "a", "b", "github.com")
	a_b_c_git := filepath.Join(tmpDir, "a", "b", "c", ".git")
	dirs := []string{a, a_b, a_b_c, a_git, a_b_github, a_b_c_git}
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			t.Errorf("error creating dir %q: %s", dir, err)
			return
		}
	}
	tests := []struct {
		start string
		what  string
		err   bool
		want  string
	}{
		{start: a_b_c, what: ".git", want: a_b_c},
		{start: a_b, what: ".git", want: a},
		{start: a, what: ".git", want: a},
		{start: tmpDir, what: ".git", err: true},
		{start: a_b_c, what: "github.com", want: a_b},
		{start: a_b, what: "github.com", want: a_b},
		{start: a, what: "github.com", err: true},
		{start: tmpDir, what: "github.com", err: true},
	}
	for _, tt := range tests {
		got, err := findDir(tt.start, tt.what)
		if err != nil {
			if tt.err {
				continue
			}
			t.Errorf("findDir(%q, %q) returned error %s, want success",
				tt.start, tt.what, err)
			continue
		}
		if tt.err {
			t.Errorf("findDir(%q, %q) returned %q, want error", tt.start, tt.what, got)
			continue
		}
		if got != tt.want {
			t.Errorf("findDir(%q, %q) returned %q, want %q", tt.start, tt.what, got, tt.want)
		}
	}
}

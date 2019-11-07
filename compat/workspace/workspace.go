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
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Root takes an arbitrary path inside a workspace returns the path to the
// logical root of the workspace from which workspace-absolute paths are
// resolved. In the case of github repos, it is the part of the input path
// up to and including "/github.com/" (without trailing slash).
// If no well-known workspace identification method works, it returns a
// directory part of the input path.
// TODO(salikh): Implement looking up for $DIR/.git.
func Root(p string) (string, error) {
	abs := filepath.Clean(p)
	if pos := strings.Index(abs, "/github.com/"); pos != -1 {
		return abs[:pos], nil
	}
	return filepath.Dir(p), nil
}

// findDir finds either github.com/ or .git above the current
// directory or returns an error.
func findDir(start, what string) (string, error) {
	dir := filepath.Clean(start)
	for dir != "." {
		whatDir := filepath.Join(dir, what)
		fs, err := os.Stat(whatDir)
		if err == nil && fs.IsDir() {
			return dir, nil
		}
		up := filepath.Dir(dir)
		if up == dir {
			break
		}
		dir = up
	}
	return "", fmt.Errorf("could not find %q starting from %q", what, start)
}

// GetGitDir finds a directory containing .git/ starting
// from the current directory and going up.
func GetGitDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return findDir(cwd, ".git")
}

func GetGithubDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return findDir(cwd, "github.com")
}

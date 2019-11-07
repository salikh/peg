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

package file

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/golang/leveldb/db"
	"github.com/golang/leveldb/memfs"
)

var (
	once  sync.Once
	memFS db.FileSystem
)

const bufSize = 1048576 - 32

// ReadFile reads the contents of the file into memory.
func ReadFile(ctx context.Context, filename string) ([]byte, error) {
	// Handle "/memfs" prefix by hijacking the call.
	if strings.HasPrefix(filename, "/memfs/") {
		once.Do(func() {
			memFS = memfs.New()
		})
		fi, err := memFS.Stat(filename)
		if err != nil {
			return nil, err
		}
		f, err := memFS.Open(filename)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		buf := make([]byte, int(fi.Size()))
		n, err := f.Read(buf)
		if err != nil {
			return nil, err
		}
		return buf[:n], nil
	}
	return ioutil.ReadFile(filename)
}

// WriteFile writes the given contents into a file.
func WriteFile(ctx context.Context, filename string, contents []byte) error {
	// Handle "/memfs" prefix by hijacking the call.
	if strings.HasPrefix(filename, "/memfs/") {
		once.Do(func() {
			memFS = memfs.New()
		})
		err := memFS.MkdirAll(path.Dir(filename), 0770)
		if err != nil {
			return err
		}
		f, err := memFS.Create(filename)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(contents)
		if err != nil {
			return err
		}
		return nil
	}
	return ioutil.WriteFile(filename, contents, 0775)
}

type File struct {
	*os.File
}

func (f *File) IO(ctx context.Context) *os.File {
	return f.File
}

// Options may contain some options later.
type Options struct {
}

// Open opens a file.
func Open(ctx context.Context, filename, modestr string, options *Options) (db.File, error) {
	if strings.HasPrefix(filename, "/memfs/") {
		once.Do(func() {
			memFS = memfs.New()
		})
		switch modestr {
		case "r", "rt", "rb":
			return memFS.Open(filename)
		case "w", "wt", "wb":
			return memFS.Create(filename)
		case "a", "at", "ab":
			return nil, errors.New("append not supported on /memfs/")
		default:
			return nil, fmt.Errorf("unrecognized format %q", modestr)
		}
	}
	var mode int
	switch modestr {
	case "r", "rt", "rb":
		mode = os.O_RDONLY
	case "w", "wt", "wb":
		mode = os.O_RDWR | os.O_CREATE | os.O_TRUNC
	case "a", "at", "ab":
		mode = os.O_RDWR | os.O_APPEND
	default:
		return nil, fmt.Errorf("unrecognized format %q", modestr)
	}
	f, err := os.OpenFile(filename, mode, 0775)
	return &File{f}, err
}

func Stat(ctx context.Context, filename string) (os.FileInfo, error) {
	if strings.HasPrefix(filename, "/memfs/") {
		once.Do(func() {
			memFS = memfs.New()
		})
		return memFS.Stat(filename)
	}
	return os.Stat(filename)
}

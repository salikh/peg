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
	"fmt"
	"reflect"
	"strings"

	"github.com/salikh/peg/parser"
)

// Accessor is the interface given to the callback during
// the semantic tree construction.
type Accessor interface {
	// Node returns the parse tree node that is currently being converted
	// (so technically this is not about child access).
	Node() *parser.Node
	// String returns the string value associated with a child node.
	// In case of error it records the error and returns empty string.
	String(name string) string
	// GetString returns the string value associated with a child node.
	// It returns an error if the requested child node does not exist
	// or has different type.
	GetString(name string) (string, error)
	// Get returns the interface type matching the given type instance.
	// In case of error it records the error and returns nil.
	Get(name string, ty interface{}) interface{}
	// GetTyped returns the interface type matching the given type instance.
	// If the requested child does not exist or has different type, it returns
	// an error. If the requested type is a slice, and only one child instance
	// exists, it is returned as an 1-element slice.
	GetTyped(name string, ty interface{}) (interface{}, error)
	// Returns the node label of the ith child in the parse tree. If the index
	// is out of bounds, it records an error and
	Child(i int) string
	// Returns the first child node with the specified label.
	// If not found, returns nil.
	GetChild(label string) *parser.Node
}

// AccessorOptions configures the behavior of error checking of Accessor.
type AccessorOptions struct {
	// If true, the Check() method will return error if it detects that some
	// of children node were converted to non-trivial semantic subtrees,
	// but were not used during semantic tree construction of this node.
	ErrorOnUnusedChild bool
}

type accessor struct {
	node     *parser.Node
	children map[string]interface{}
	accessed map[string]bool
	errs     []error
	options  *AccessorOptions
}

func (ca *accessor) Node() *parser.Node {
	return ca.node
}

func (ca *accessor) GetString(name string) (string, error) {
	ca.accessed[name] = true
	val, ok := ca.children[name]
	if !ok {
		return "", fmt.Errorf("in %s expected %s as string, got none", ca.node, name)
	}
	s, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("in %s expected %s as string, got %s", ca.node,
			name, reflect.TypeOf(val))
	}
	return s, nil
}

func (ca *accessor) String(name string) string {
	val, err := ca.GetString(name)
	if err != nil {
		ca.errs = append(ca.errs, err)
	}
	return val
}

func (ca *accessor) Get(name string, ty interface{}) interface{} {
	val, err := ca.GetTyped(name, ty)
	if err != nil {
		ca.errs = append(ca.errs, err)
	}
	return val
}

func (ca *accessor) GetTyped(name string, ty interface{}) (interface{}, error) {
	ca.accessed[name] = true
	val, ok := ca.children[name]
	if !ok {
		if ty == nil {
			return nil, nil
			//return nil, fmt.Errorf("%s not found", name)
		}
		if reflect.TypeOf(ty).Kind() == reflect.Slice {
			// If the expected type is slice, tolerate empty slice and return the passed
			// in type instance instead (which is expected to be an empty slice typically).
			return ty, nil
		}
		return ty, fmt.Errorf("expected %s as %s, got none", name, reflect.TypeOf(ty))
	}
	if ty == nil {
		return val, nil
	}
	if reflect.TypeOf(val) == reflect.TypeOf(ty) {
		// Expecting exactly this type.
		return val, nil
	}
	sliceTy := reflect.SliceOf(reflect.TypeOf(val))
	if reflect.TypeOf(ty) == sliceTy {
		// Expecting a slice, but got a single element.
		s := reflect.MakeSlice(sliceTy, 0, 1)
		s = reflect.Append(s, reflect.ValueOf(val))
		return s.Interface(), nil
	}
	// Slice type is mismatched, reconstruct.
	if reflect.TypeOf(ty).Kind() == reflect.Slice && reflect.TypeOf(val).Kind() == reflect.Slice {
		s := reflect.MakeSlice(reflect.TypeOf(ty), 0, reflect.ValueOf(val).Len())
		for i := 0; i < reflect.ValueOf(val).Len(); i++ {
			s = reflect.Append(s, reflect.ValueOf(val).Index(i))
		}
		return s.Interface(), nil
	}
	// Exactly one element returned, try to fit it into a slice.
	if reflect.TypeOf(ty).Kind() == reflect.Slice {
		s := reflect.MakeSlice(reflect.TypeOf(ty), 0, 1)
		s = reflect.Append(s, reflect.ValueOf(val))
		return s.Interface(), nil
	}
	return ty, fmt.Errorf("expected %s, got %s", reflect.TypeOf(ty), reflect.TypeOf(val))
}

func (ca *accessor) Check() error {
	if ca.options != nil && ca.options.ErrorOnUnusedChild {
		for k := range ca.children {
			if !ca.accessed[k] {
				ca.errs = append(ca.errs, fmt.Errorf("child %s was not used during conversion", k))
			}
		}
	}
	if len(ca.errs) == 0 {
		return nil
	}
	var r []string
	for _, err := range ca.errs {
		r = append(r, err.Error())
	}
	return fmt.Errorf("multiple errors:\n%s\n", strings.Join(r, "\n"))
}

func (ca *accessor) Child(i int) string {
	if i < 0 || len(ca.node.Children) <= i {
		ca.errs = append(ca.errs,
			fmt.Errorf("child access out of bounds: want [%d], got %d children",
				i, len(ca.node.Children)))
		return ""
	}
	return ca.node.Children[i].Label
}

func (ca *accessor) GetChild(label string) *parser.Node {
	for _, ch := range ca.node.Children {
		if ch.Label == label {
			return ch
		}
	}
	return nil
}

func Construct(n *parser.Node, callback func(string, Accessor) (interface{}, error), options *AccessorOptions) (interface{}, error) {
	ca := &accessor{
		node:     n,
		children: make(map[string]interface{}),
		accessed: make(map[string]bool),
		options:  options,
	}
	for _, ch := range n.Children {
		val, err := Construct(ch, callback, options)
		if err != nil {
			return nil, err
		}
		if val == nil {
			// No value to store.
			continue
		}
		have, ok := ca.children[ch.Label]
		if !ok {
			// Not seen yet: just store a single value.
			ca.children[ch.Label] = val
		} else {
			// Have seen already.
			if reflect.TypeOf(have) == reflect.TypeOf(val) {
				// Second value, create a slice and store it.
				s := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(have)), 0, 2)
				s = reflect.Append(s, reflect.ValueOf(have), reflect.ValueOf(val))
				ca.children[ch.Label] = s.Interface()
			} else if reflect.TypeOf(have) == reflect.SliceOf(reflect.TypeOf(val)) {
				// Additional value, append to the slice and store it.
				s := reflect.Append(reflect.ValueOf(have), reflect.ValueOf(val))
				ca.children[ch.Label] = s.Interface()
			} else {
				return nil, fmt.Errorf("incompatible types, have %s and got %s\na: %s",
					reflect.TypeOf(have), reflect.TypeOf(val), ch)
			}
		}
	}
	val, err := callback(n.Label, ca)
	if err != nil {
		return nil, fmt.Errorf("error constructing %s: %s\nTree: %s", n.Label, err, n)
	}
	err = ca.Check()
	if err != nil {
		return nil, fmt.Errorf("error constructing %s: %s\nTree: %s", n.Label, err, n)
	}
	return val, nil
}

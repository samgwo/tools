// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package godoc

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"code.google.com/p/go.tools/godoc/vfs/mapfs"
)

func newCorpus(t *testing.T) *Corpus {
	c := NewCorpus(mapfs.New(map[string]string{
		"src/pkg/foo/foo.go": `// Package foo is an example.
package foo

import "bar"

// Foo is stuff.
type Foo struct{}

func New() *Foo {
   return new(Foo)
}
`,
		"src/pkg/bar/bar.go": `// Package bar is another example to test races.
package bar
`,
		"src/pkg/other/bar/bar.go": `// Package bar is another bar package.
package bar
func X() {}
`,
		"src/pkg/skip/skip.go": `// Package skip should be skipped.
package skip
func Skip() {}
`,
	}))
	c.IndexEnabled = true
	c.IndexDirectory = func(dir string) bool {
		return !strings.Contains(dir, "skip")
	}

	if err := c.Init(); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestIndex(t *testing.T) {
	c := newCorpus(t)
	c.UpdateIndex()
	ix, _ := c.CurrentIndex()
	if ix == nil {
		t.Fatal("no index")
	}
	t.Logf("Got: %#v", ix)
	testIndex(t, ix)
}

func TestIndexWriteRead(t *testing.T) {
	c := newCorpus(t)
	c.UpdateIndex()
	ix, _ := c.CurrentIndex()
	if ix == nil {
		t.Fatal("no index")
	}

	var buf bytes.Buffer
	nw, err := ix.WriteTo(&buf)
	if err != nil {
		t.Fatalf("Index.WriteTo: %v", err)
	}

	ix2 := new(Index)
	nr, err := ix2.ReadFrom(&buf)
	if err != nil {
		t.Fatalf("Index.ReadFrom: %v", err)
	}
	if nr != nw {
		t.Errorf("Wrote %d bytes to index but read %d", nw, nr)
	}
	testIndex(t, ix2)
}

func testIndex(t *testing.T, ix *Index) {
	wantStats := Statistics{Bytes: 256, Files: 3, Lines: 16, Words: 6, Spots: 9}
	if !reflect.DeepEqual(ix.Stats(), wantStats) {
		t.Errorf("Stats = %#v; want %#v", ix.Stats(), wantStats)
	}

	if _, ok := ix.words["Skip"]; ok {
		t.Errorf("the word Skip was found; expected it to be skipped")
	}

	if got, want := ix.ImportCount(), map[string]int{
		"bar": 1,
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("ImportCount = %v; want %v", got, want)
	}

	if got, want := ix.PackagePath(), map[string]map[string]bool{
		"foo": map[string]bool{
			"foo": true,
		},
		"bar": map[string]bool{
			"bar":       true,
			"other/bar": true,
		},
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("PackagePath = %v; want %v", got, want)
	}

	if got, want := ix.Exports(), map[string]map[string]SpotKind{
		"foo": map[string]SpotKind{
			"Foo": TypeDecl,
			"New": FuncDecl,
		},
		"other/bar": map[string]SpotKind{
			"X": FuncDecl,
		},
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("Exports = %v; want %v", got, want)
	}
}

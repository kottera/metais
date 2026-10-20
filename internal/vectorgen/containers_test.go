// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package vectorgen

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// The expected containers of W6-4's vectors. aut/test-containers' examples
// each break the comparison by entries in one way, and "equal" holds the
// render's entries in zips of other bytes, every method the render does
// not fix chosen the other way; aut/test-containers-packed is its source
// directory packed. aut/pack-containers is where W6-4 meets pack (W6-1):
// its package's one example is "equal", so it packs, and its failing
// source's is "order", so pack refuses it.
const (
	containersVector = "aut/test-containers"
	containersPacked = "aut/test-containers-packed"
	packContainers   = "aut/pack-containers"
)

// containerSources are the source directories whose examples/ the
// generator writes, each with the examples it holds.
var containerSources = []struct {
	dir      string
	examples []string
}{
	{containersVector + "/package", []string{"content", "equal", "method", "missing", "order", "unexpected", "unreadable"}},
	{packContainers + "/package", []string{"equal"}},
	{packContainers + "/failing", []string{"order"}},
}

// The rendered entries: the package's templates.
var (
	ctA        = []byte("a line of a\n")
	ctB        = []byte("a line of b\n")
	ctC        = []byte("a line of c\n")
	ctContent  = []byte("<office:document-content/>\n")
	ctManifest = []byte("<manifest:manifest/>\n")
	ctMimetype = []byte("application/vnd.oasis.opendocument.text")
)

// containerFiles is every file of each named example of aut/test-containers'
// package, by its path within the package.
func containerFiles(names []string) map[string][]byte {
	inner := func(es ...zentry) zentry { return zentry{name: "inner.zip", content: zipOf(es)} }
	c := zentry{name: "c.txt", content: ctC}
	box := func(es ...zentry) []byte { return zipOf(es) }
	a, b, in := zentry{name: "a.txt", content: ctA}, zentry{name: "b.txt", content: ctB}, inner(c)
	odt := func(mimetype, manifest, content zentry) []byte { return zipOf([]zentry{mimetype, manifest, content}) }
	mt, mf, ct := zentry{name: "mimetype", content: ctMimetype}, zentry{name: "META-INF/manifest.xml", content: ctManifest}, zentry{name: "content.xml", content: ctContent}
	deflate := func(e zentry) zentry { e.deflate = true; return e }
	rendered := odt(mt, mf, ct)
	examples := map[string]struct {
		title    string
		box, odt []byte
	}{
		"equal": {"Equal by entries, in zips of other bytes",
			box(deflate(a), b, zentry{name: "inner.zip", content: zipOf([]zentry{deflate(c)})}), odt(mt, deflate(mf), deflate(ct))},
		"missing":    {"An entry expected and not written", box(a, b, in, zentry{name: "z.txt", content: []byte("z\n")}), rendered},
		"unexpected": {"An entry written and not expected", box(b, in), rendered},
		"order":      {"Entries in another order", box(b, a, in), rendered},
		"content": {"An entry's content, and a container entry's entry's",
			box(zentry{name: "a.txt", content: []byte("another line of a\n")}, b, inner(zentry{name: "c.txt", content: []byte("a line of c, changed\n")})), rendered},
		"method":     {"A stored entry, and a profile's mimetype, deflated", box(a, deflate(b), in), odt(deflate(mt), mf, ct)},
		"unreadable": {"An expected container that is not a zip", []byte("not a zip archive\n"), rendered},
	}
	out := map[string][]byte{}
	for _, name := range names {
		e := examples[name]
		dir := "examples/" + name + "/"
		out[dir+"example.yaml"] = []byte("title: " + e.title + "\n")
		out[dir+"expected/box.zip"] = e.box
		out[dir+"expected/doc.odt"] = e.odt
	}
	return out
}

// TestContainerVectors is the program and its reproduction test (ADR-011
// D16 item 5): without -update, every file containerFiles names is the
// committed one, the examples hold nothing else, and packed.metais is the
// source directory packed; with it, each is written.
func TestContainerVectors(t *testing.T) {
	testenv.Home(t)
	for _, src := range containerSources {
		root := filepath.Join(vectorsDir, filepath.FromSlash(src.dir))
		files := containerFiles(src.examples)
		for name, want := range files {
			path := filepath.Join(root, filepath.FromSlash(name))
			if *update {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, want, 0o644); err != nil {
					t.Fatal(err)
				}
				continue
			}
			if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, want) {
				t.Errorf("%s/%s is not what its case builds (run go test ./internal/vectorgen -run TestContainerVectors -args -update): %v", src.dir, name, err)
			}
		}
		var committed []string
		err := filepath.WalkDir(filepath.Join(root, "examples"), func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				rel, _ := filepath.Rel(root, p)
				committed = append(committed, filepath.ToSlash(rel))
			}
			return err
		})
		if err != nil && !(*update && errors.Is(err, fs.ErrNotExist)) {
			t.Fatal(err)
		}
		for _, n := range committed {
			if _, ok := files[n]; !ok {
				t.Errorf("%s/%s: a committed file that no case builds", src.dir, n)
			}
		}
	}
	packed := pack(t, sourceFiles(t, containersVector))
	path := filepath.Join(vectorsDir, filepath.FromSlash(containersPacked), "packed.metais")
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, packed, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, packed) {
		t.Errorf("%s/packed.metais is not the source directory packed: %v", containersPacked, err)
	}
}

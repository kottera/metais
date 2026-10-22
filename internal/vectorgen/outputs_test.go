// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package vectorgen

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
)

// packOutputs are the packages pack's vectors expect it to write: each is
// a source directory of the vector packed, file by file as pack writes
// it, so a change to the README or to the compiled files is regenerated
// rather than packed again by hand.
var packOutputs = []struct {
	vector, source string
	outputs        []string // from the vector's directory
}{
	{"aut/pack", "package", []string{"expected/out/pack-1.0.0.metais", "expected/named/named.metais"}},
	{"aut/pack-replace", "package", []string{"expected/out/replace-1.0.0.metais"}},
	{"gen/worked-package", "package", []string{"expected/out/release-notes-1.0.0.metais"}},
	{"pkg/pack-round-trip", "fixture", []string{"expected/fixture/out/pack-1.0.0.metais"}},
	{"pkg/pack-round-trip", "scaffold", []string{"expected/scaffold/out/acme-report-0.1.0.metais"}},
}

// layoutTrees are the top-level directories of the package layout
// (ADR-001 D5): pack writes metais.yaml and what is beneath these, and
// leaves every other top-level name of a source directory out.
var layoutTrees = []string{"assets", "examples", "inputs", "references", "templates"}

// packedFiles is files without the names pack leaves out of a package.
func packedFiles(files map[string][]byte) map[string][]byte {
	out := map[string][]byte{}
	for name, b := range files {
		top, _, nested := strings.Cut(name, "/")
		for _, tree := range layoutTrees {
			if nested && top == tree {
				out[name] = b
			}
		}
		if name == "metais.yaml" {
			out[name] = b
		}
	}
	return out
}

// TestPackOutputs is the program and its reproduction test for
// packOutputs: each output is rebuilt from its source directory and
// compared with the committed bytes, or, with -update, written.
func TestPackOutputs(t *testing.T) {
	testenv.Home(t)
	for _, o := range packOutputs {
		got := pack(t, packedFiles(treeFiles(t, o.vector, o.source))).bytes
		for _, out := range o.outputs {
			path := filepath.Join(vectorsDir, filepath.FromSlash(o.vector), filepath.FromSlash(out))
			if *update {
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatal(err)
				}
				continue
			}
			if want, err := os.ReadFile(path); err != nil || !bytes.Equal(got, want) {
				t.Errorf("%s/%s is not %s packed (run go test ./internal/vectorgen -run TestPackOutputs -args -update): %v", o.vector, out, o.source, err)
			}
		}
	}
}

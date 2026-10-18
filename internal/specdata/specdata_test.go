// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package specdata

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// mismatches names every file of names whose copy in copies is not byte
// for byte the file below specRoot.
func mismatches(specRoot string, copies map[string][]byte, names []string) []string {
	var bad []string
	for _, name := range names {
		want, err := os.ReadFile(filepath.Join(specRoot, filepath.FromSlash(name)))
		if err != nil {
			bad = append(bad, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		got, ok := copies[name]
		switch {
		case !ok:
			bad = append(bad, name+": not embedded")
		case !bytes.Equal(got, want):
			bad = append(bad, name+": the embedded copy differs from spec/; run go generate ./internal/specdata")
		}
	}
	return bad
}

// TestSpecDataMatchesSpec fails on any difference between an embedded
// copy and its spec file (ADR-010 D2 item 4, ADR-011 D7 item 6).
func TestSpecDataMatchesSpec(t *testing.T) {
	for _, m := range mismatches(specRoot, files, specFiles) {
		t.Error(m)
	}
	embedded := slices.Sorted(maps.Keys(files))
	if listed := slices.Sorted(slices.Values(specFiles)); !slices.Equal(embedded, listed) {
		t.Errorf("embedded %q, but specFiles lists %q", embedded, listed)
	}
}

// TestSpecDataMatchesSpecCatchesOneByte is TestSpecDataMatchesSpec's
// control: a copy one byte away from its spec file, anywhere in it, is
// caught.
func TestSpecDataMatchesSpecCatchesOneByte(t *testing.T) {
	for _, name := range specFiles {
		orig := files[name]
		for _, at := range []int{0, len(orig) / 2, len(orig) - 1} {
			changed := bytes.Clone(orig)
			changed[at] ^= 1
			copies := maps.Clone(files)
			copies[name] = changed
			if got := mismatches(specRoot, copies, specFiles); len(got) != 1 {
				t.Errorf("%s with byte %d changed: %d mismatches reported, want 1", name, at, len(got))
			}
		}
		copies := maps.Clone(files)
		copies[name] = append(bytes.Clone(orig), '\n')
		if got := mismatches(specRoot, copies, specFiles); len(got) != 1 {
			t.Errorf("%s with a byte added: %d mismatches reported, want 1", name, len(got))
		}
	}
	// And a spec file changed under an unchanged copy.
	root := testenv.Dir(t)
	for _, name := range specFiles {
		b := bytes.Clone(files[name])
		if name == specFiles[0] {
			b[len(b)/3] ^= 0x20
		}
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got := mismatches(root, files, specFiles); len(got) != 1 {
		t.Errorf("a changed spec file: %d mismatches reported, want 1", len(got))
	}
}

// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/kottera/metais/internal/inputs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/testenv"
)

// TestInputsOverTheLoader is W3-1's meeting with W3-2: a package opened
// by the loader, packed and as a source directory, gives the inputs layer
// what ADR-003 D22 step 1 produced — its manifest, every schema file
// compiled, and the combined schema bundled and compiled — and the layer
// runs steps 2 to 5 over a data directory against those schemas: a valid
// supply gives no finding, and an invalid one every finding of its value.
func TestInputsOverTheLoader(t *testing.T) {
	testenv.Home(t)
	for _, form := range []string{"pkg/opens-source/package", "pkg/opens-packed/package.metais"} {
		p, fs := Open(filepath.Join(vectorsDir, filepath.FromSlash(form)), defaults, 0)
		if p == nil {
			t.Fatalf("%s: %v", form, codes(fs))
		}
		l, err := inputs.New(inputs.Package{Manifest: p.Manifest, Compiler: p.Schemas, Combined: p.CombinedValue, Schema: p.Combined})
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range []struct {
			files map[string][]byte
			want  []string
		}{
			{map[string][]byte{"project.json": []byte(`{"name": "Example", "owner": "owner@example.org"}`)}, nil},
			{map[string][]byte{
				"project.json":    []byte(`{"name": "", "owner": "not an email", "extra": 1}`),
				"items/first.md":  []byte("---\ntitle: First\n---\nBody\n"),
				"items/second.md": []byte("---\nbody: x\n---\n"),
			}, []string{"input.additional_property", "input.format", "input.min_length", "input.validation.partial", "markdown.frontmatter_body_field"}},
			{map[string][]byte{}, []string{"input.missing"}},
		} {
			dir := writeSource(t, c.files)
			s := l.NewSupply(limits.Defaults())
			if err := s.ReadDataDir(dir); err != nil {
				t.Fatal(err)
			}
			res, err := s.Validate()
			if err != nil {
				t.Fatal(err)
			}
			got := codes(res.Findings)
			slices.Sort(got)
			if !slices.Equal(got, c.want) {
				t.Errorf("%s: %v gives %q, want %q", form, keys(c.files), got, c.want)
			}
		}
		p.Close()
	}
}

func keys(m map[string][]byte) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

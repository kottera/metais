// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package cli

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/testenv"
)

// plant makes each link of links, by path beneath root, pointing at its
// target, and the files of files.
func plant(t *testing.T, root string, links, files map[string]string) {
	t.Helper()
	for name, body := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for name, target := range links {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, p); err != nil {
			t.Fatal(err)
		}
	}
}

// TestHostileUnpackPlantedLink is ADR-009 D17 item 11's unpack into a
// directory holding a planted link (ADR-005 D15 item 4, MTS-AUT-028): a
// link at an entry's path pointing out of the target, one at an entry's
// directory, one with the name of a packed package, and a link loop are
// each refused as an entry the target may not hold, and nothing anywhere
// is written, through the link or beside it. The package is the suite's
// own aut/unpack, which a vector cannot plant a link beside without its
// own copy of those bytes.
func TestHostileUnpackPlantedLink(t *testing.T) {
	testenv.Home(t)
	pkg := filepath.Join("..", "..", "spec", "conformance", "vectors", "aut", "unpack", "package.metais")
	for _, c := range []struct {
		name  string
		links map[string]string
		want  string
	}{
		{"a file pointing out", map[string]string{"src/metais.yaml": "../victim.txt"}, "metais.yaml"},
		{"a directory pointing out", map[string]string{"src/templates": "../elsewhere"}, "templates"},
		{"a packed package's name", map[string]string{"src/old.metais": "../victim.txt"}, "old.metais"},
		{"a loop", map[string]string{"src/inputs": "inputs"}, "inputs"},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := testenv.Dir(t)
			plant(t, root, c.links, map[string]string{"victim.txt": "theirs\n", "elsewhere/keep.txt": "theirs\n"})
			before := snapshot(t, root)
			r := runTool(t, "unpack", "--json", pkg, "--out", filepath.Join(root, "src"))
			want := []string{`output.destination.not_empty {"entries":["` + c.want + `"],"more":false}`}
			if got := findingParams(r); r.status != finding.ExitDestination || !slices.Equal(got, want) {
				t.Errorf("exit %d, findings %q, want %q", r.status, got, want)
			}
			if after := snapshot(t, root); !slices.Equal(after, before) {
				t.Errorf("the refusal wrote: %q, was %q", after, before)
			}
		})
	}
}

// TestHostileExtractPlantedLink is ADR-009 D17 item 11's extract into a
// directory holding a planted link (ADR-005 D15 item 4), through the
// command line: a link at a file's path and one at a directory above a
// file, each pointing out of the destination, are refused unfollowed,
// without --overwrite as an entry the destination may not hold and with
// it as a link at each path it would write, and nothing anywhere is
// written.
func TestHostileExtractPlantedLink(t *testing.T) {
	testenv.Home(t)
	pkg := browsePackage(t)
	for _, c := range []struct {
		name  string
		links map[string]string
		// want is the refusal without --overwrite, which refuses a
		// destination holding anything, and wantOverwrite with it.
		want, wantOverwrite string
	}{
		{"a file pointing out", map[string]string{"to/metais.yaml": "../victim.txt"},
			`output.destination.not_empty {"entries":["metais.yaml"],"more":false}`, `output.destination.link {"path":"metais.yaml"}`},
		{"a directory pointing out", map[string]string{"to/references": "../elsewhere"},
			`output.destination.not_empty {"entries":["references"],"more":false}`, `output.destination.link {"path":"references/crlf.txt"}`},
	} {
		for _, overwrite := range []bool{false, true} {
			root := testenv.Dir(t)
			plant(t, root, c.links, map[string]string{"victim.txt": "theirs\n", "elsewhere/keep.txt": "theirs\n"})
			before := snapshot(t, root)
			argv := []string{"extract", "--json", pkg, "metais.yaml", "references/*.txt", "--out", filepath.Join(root, "to")}
			if overwrite {
				argv = append(argv, "--overwrite")
			}
			want := c.want
			if overwrite {
				want = c.wantOverwrite
			}
			r := runTool(t, argv...)
			got := findingParams(r)
			if r.status != finding.ExitDestination || len(got) == 0 || got[0] != want {
				t.Errorf("%s, --overwrite %v: exit %d, findings %q, want %q first", c.name, overwrite, r.status, got, want)
			}
			if after := snapshot(t, root); !slices.Equal(after, before) {
				t.Errorf("%s, --overwrite %v: the refusal wrote: %q, was %q", c.name, overwrite, after, before)
			}
		}
	}
}

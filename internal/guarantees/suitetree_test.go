// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/testenv"
	"github.com/kottera/metais/internal/unitext"
)

// suiteRoot is the conformance tree, relative to the repository root.
const suiteRoot = "spec/conformance"

// maxSuitePath is the longest a path under the conformance tree may be,
// in bytes from the repository root: Windows' MAX_PATH of 260 characters,
// less room for the clone's own location (ADR-011 D9 item 8).
const maxSuitePath = 160

// suiteTextLine is the .gitattributes line that stops line-ending
// conversion under the conformance tree (ADR-011 D9 item 8).
var suiteTextLine = []string{suiteRoot + "/**", "-text"}

// suiteFile is one non-directory under the conformance tree: its path from
// the repository root, with / separators, and whether it is a link.
type suiteFile struct {
	path string
	link bool
}

// walkSuite lists every non-directory under root's conformance tree, links
// included and not followed. A tree that does not exist yet has none.
func walkSuite(root string) ([]suiteFile, error) {
	var out []suiteFile
	err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(suiteRoot)), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		out = append(out, suiteFile{path: filepath.ToSlash(rel), link: d.Type()&fs.ModeSymlink != 0})
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return out, err
}

// suiteTreeProblems applies ADR-011 D9 item 8 to the conformance tree's
// files and to the repository's .gitattributes: every path obeys ADR-005
// D12's portable-path predicate, and no two collide; none is longer than
// 160 bytes; none is a link; and .gitattributes holds the -text line.
func suiteTreeProblems(files []suiteFile, gitattributes string) []string {
	var bad []string
	lim := limits.Defaults()
	var paths []string
	for _, f := range files {
		paths = append(paths, f.path)
		if f.link {
			bad = append(bad, fmt.Sprintf("%q is a link", f.path))
		}
		if len(f.path) > maxSuitePath {
			bad = append(bad, fmt.Sprintf("%q is %d bytes from the repository root, more than %d", f.path, len(f.path), maxSuitePath))
		}
		for _, pr := range unitext.CheckPath(f.path, lim) {
			bad = append(bad, fmt.Sprintf("%q breaks ADR-005 D12: %+v", f.path, pr))
		}
	}
	for _, c := range unitext.Collisions(paths) {
		bad = append(bad, fmt.Sprintf("%q collides with %q (ADR-005 D12 items 6 and 7)", c.Path, c.Other))
	}
	found := false
	for line := range strings.SplitSeq(gitattributes, "\n") {
		found = found || slices.Equal(strings.Fields(line), suiteTextLine)
	}
	if !found {
		bad = append(bad, fmt.Sprintf(".gitattributes lacks the line %q", strings.Join(suiteTextLine, " ")))
	}
	return bad
}

// TestSuiteTreePortable is ADR-011 D14's tripwire for D9 item 8: the
// conformance tree checks out the same on every platform.
func TestSuiteTreePortable(t *testing.T) {
	files, err := walkSuite(moduleRoot)
	if err != nil {
		t.Fatal(err)
	}
	ga, err := os.ReadFile(filepath.Join(moduleRoot, ".gitattributes"))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range suiteTreeProblems(files, string(ga)) {
		t.Error(p)
	}
}

// TestSuiteTreePortableControl holds the tripwire red on each way a tree
// can break D9 item 8, and green on a tree that does not (W2-4.9).
func TestSuiteTreePortableControl(t *testing.T) {
	const ga = "spec/conformance/** -text\n"
	ok := []suiteFile{
		{path: "spec/conformance/README.md"},
		{path: "spec/conformance/vectors/out/for-each/vector.json"},
		{path: "spec/conformance/vectors/out/for-each/expected/out/caf\u00e9.md"},
	}
	if got := suiteTreeProblems(ok, ga); len(got) > 0 {
		t.Errorf("a portable tree is flagged: %q", got)
	}
	for name, c := range map[string]struct {
		files []suiteFile
		ga    string
	}{
		"a reserved name":        {append(slices.Clone(ok), suiteFile{path: "spec/conformance/vectors/a/b/CON.txt"}), ga},
		"a reserved character":   {append(slices.Clone(ok), suiteFile{path: "spec/conformance/vectors/a/b/what?.txt"}), ga},
		"not NFC":                {append(slices.Clone(ok), suiteFile{path: "spec/conformance/vectors/a/b/cafe\u0301.md"}), ga},
		"a case-only difference": {append(slices.Clone(ok), suiteFile{path: "spec/conformance/readme.md"}), ga},
		"too long":               {append(slices.Clone(ok), suiteFile{path: "spec/conformance/vectors/" + strings.Repeat("a", 140)}), ga},
		"a link":                 {append(slices.Clone(ok), suiteFile{path: "spec/conformance/vectors/a/link", link: true}), ga},
		"no -text line":          {ok, "THIRD-PARTY-NOTICES.txt -text\n"},
		"text set, not unset":    {ok, "spec/conformance/** text\n"},
	} {
		if got := suiteTreeProblems(c.files, c.ga); len(got) != 1 {
			t.Errorf("%s: flagged %q, want one problem", name, got)
		}
	}
}

// TestWalkSuiteSeesLinks is the walk's control: a link under the tree is
// listed as one, not followed, and an absent tree has no files.
func TestWalkSuiteSeesLinks(t *testing.T) {
	root := testenv.Dir(t)
	if files, err := walkSuite(root); err != nil || len(files) != 0 {
		t.Fatalf("an absent tree: %v, %v", files, err)
	}
	dir := filepath.Join(root, "spec", "conformance", "vectors")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.json"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("a.json", filepath.Join(dir, "b.json")); err != nil {
		t.Skipf("this file system cannot hold a symbolic link: %v", err)
	}
	files, err := walkSuite(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []suiteFile{{path: "spec/conformance/vectors/a.json"}, {path: "spec/conformance/vectors/b.json", link: true}}
	if !slices.Equal(files, want) {
		t.Errorf("walkSuite = %+v, want %+v", files, want)
	}
}

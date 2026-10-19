// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/pkgtree/pkgtreetest"
)

// sourceOptions are the defaults with a name rule that admits every name,
// which ADR-001 D2's predicate replaces where the loader composes them.
func sourceOptions(t *testing.T, lower ...string) SourceOptions {
	t.Helper()
	l := limits.Defaults()
	for _, arg := range lower {
		if _, ok := l.Lower(arg); !ok {
			t.Fatalf("Lower(%q) refused", arg)
		}
	}
	return SourceOptions{Limits: l, Names: func([]string) error { return nil }}
}

func wantProblems(t *testing.T, se *SourceError, want []Problem) {
	t.Helper()
	if !slices.Equal(se.Problems, want) {
		t.Fatalf("problems\n got %+v\nwant %+v", se.Problems, want)
	}
}

func openSource(t *testing.T, root string, opt SourceOptions) *Source {
	t.Helper()
	s, err := OpenSource(root, opt)
	if err != nil {
		t.Fatalf("OpenSource: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func refused(t *testing.T, root string, opt SourceOptions) *SourceError {
	t.Helper()
	s, err := OpenSource(root, opt)
	if s != nil {
		s.Close()
		t.Fatal("a Source was returned beside a refusal")
	}
	var se *SourceError
	if !errors.As(err, &se) {
		t.Fatalf("OpenSource = %v, want a *SourceError", err)
	}
	noPath(t, err, root)
	return se
}

// A package's files, as a source directory holds them.
var packageFiles = map[string]string{
	"metais.yaml":                     "metais: '1.0'\nname: report\n",
	"inputs/vendor.schema.json":       `{"type":"object"}`,
	"inputs/vendor.md":                "# Vendor\n",
	"templates/report.md.liquid":      "{{ vendor.name }}\n",
	"templates/partials/_head.liquid": "",
	"assets/.well-known/security.txt": "Contact: x\n",
	"assets/css/site.css":             strings.Repeat("a{}", 3000),
	"references/standard.md":          "text",
	"examples/one/inputs/vendor.json": `{"name":"x"}`,
	"examples/one/expected/report.md": "x\n",
}

// The source-directory tree meets the pkgtree.Tree contract, holds exactly
// the files beneath the layout's roots, and leaves out what ADR-001 D5
// item 5 ignores or leaves unpacked.
func TestSourceTreeChecks(t *testing.T) {
	root := dir(t)
	files(t, root, packageFiles)
	files(t, root, map[string]string{
		".git/HEAD":           "ref: refs/heads/main\n",
		".git/objects/ab/cd":  "x",
		".gitignore":          "*.metais\n",
		".DS_Store":           "litter",
		"report-1.0.0.metais": "PK",
		"README.md":           "# readme\n",
		"build/out.html":      "<p>",
	})
	s := openSource(t, root, sourceOptions(t))
	want := map[string][]byte{}
	for p, b := range packageFiles {
		want[p] = []byte(b)
	}
	if err := pkgtreetest.Check(s, want); err != nil {
		t.Fatal(err)
	}
	if got, w := s.Unknown(), []string{"README.md", "build"}; !slices.Equal(got, w) {
		t.Errorf("Unknown = %q, want %q", got, w)
	}
}

// Nothing outside the layout's roots is listed, let alone opened: the walk
// lists the source directory itself and the directories beneath the roots,
// and nothing else, whatever else the directory holds.
func TestSourceReadsOnlyTheRoots(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{
		"metais.yaml":         "metais: '1.0'\n",
		"templates/a/b.html":  "b",
		"assets/x":            "x",
		".git/config":         "[core]",
		".git/hooks/pre-push": "#!/bin/sh",
		"node_modules/x/y.js": "y",
		"private/key":         "secret",
	})
	outside := testDirOutside(t)
	files(t, outside, map[string]string{"id_ed25519": "KEY"})
	symlink(t, root, outside, "home")
	var listed []string
	opt := sourceOptions(t)
	opt.listed = func(rel string) { listed = append(listed, rel) }
	openSource(t, root, opt)
	slices.Sort(listed)
	if want := []string{".", "assets", "templates", "templates/a"}; !slices.Equal(listed, want) {
		t.Fatalf("listed %q, want only %q", listed, want)
	}
}

// A link anywhere in the layout's roots is refused, even one whose target
// stays inside, and so is a link standing for a root; each is reported, at
// once, beside every other problem.
func TestSourceRefusesLinks(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{
		"metais.yaml":          "metais: '1.0'\n",
		"templates/ok.liquid":  "ok",
		"real-assets/logo.svg": "<svg/>",
	})
	outside := testDirOutside(t)
	files(t, outside, map[string]string{"id_ed25519": "KEY"})
	symlink(t, root, filepath.Join(outside, "id_ed25519"), "templates/x")
	symlink(t, root, "ok.liquid", "templates/inside")
	symlink(t, root, outside, "templates/dir")
	symlink(t, root, "real-assets", "assets")
	se := refused(t, root, sourceOptions(t))
	wantProblems(t, se, []Problem{
		{Kind: ProblemNonRegularFile, Path: "assets"},
		{Kind: ProblemNonRegularFile, Path: "templates/dir"},
		{Kind: ProblemNonRegularFile, Path: "templates/inside"},
		{Kind: ProblemNonRegularFile, Path: "templates/x"},
	})
	noPath(t, se, outside)
}

func TestSourceRefusesAManifestLink(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"real.yaml": "metais: '1.0'\n"})
	symlink(t, root, "real.yaml", "metais.yaml")
	wantProblems(t, refused(t, root, sourceOptions(t)), []Problem{
		{Kind: ProblemNonRegularFile, Path: "metais.yaml"},
	})
}

func TestSourceLayoutRefusals(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{
		"templates/a.liquid":            "a",
		".compiled/manifest.json":       "{}",
		"examples/one/inputs/list/.keep": "",
	})
	for _, d := range []string{"assets/empty", "references", "examples/one/inputs/empty"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(d)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	wantProblems(t, refused(t, root, sourceOptions(t)), []Problem{
		{Kind: ProblemMissingManifest},
		{Kind: ProblemCompiledInSource, Path: ".compiled"},
		{Kind: ProblemEmptyDirectory, Path: "assets/empty"},
		{Kind: ProblemEmptyDirectory, Path: "examples/one/inputs/empty"},
		{Kind: ProblemEmptyDirectory, Path: "references"},
	})
}

// A source directory with no metais.yaml is not a package, even when
// metais.yaml names a directory.
func TestSourceManifestMustBeAFile(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"metais.yaml/x": ""})
	se := refused(t, root, sourceOptions(t))
	wantProblems(t, se, []Problem{{Kind: ProblemMissingManifest}})
}

// The name rule is applied to every entry path at once, in byte order, and
// its refusal refuses the tree, reachable through errors.As.
func TestSourceAppliesTheNameRule(t *testing.T) {
	root := dir(t)
	files(t, root, packageFiles)
	var seen []string
	opt := sourceOptions(t)
	opt.Names = func(paths []string) error {
		seen = paths
		for _, p := range paths {
			if strings.Contains(p, "partials") {
				return &nameRefusal{path: p}
			}
		}
		return nil
	}
	se := refused(t, root, opt)
	want := slices.Sorted(func(yield func(string) bool) {
		for p := range packageFiles {
			if !yield(p) {
				return
			}
		}
	})
	if !slices.Equal(seen, want) {
		t.Errorf("Names saw %q, want %q", seen, want)
	}
	var nr *nameRefusal
	if !errors.As(se, &nr) || nr.path != "templates/partials/_head.liquid" {
		t.Errorf("the name rule's refusal is not reachable: %v", se)
	}
	if len(se.Problems) != 0 {
		t.Errorf("problems %v beside the name rule's refusal", se.Problems)
	}
}

type nameRefusal struct{ path string }

func (e *nameRefusal) Error() string { return "refused name" }

func TestSourceNeedsANameRule(t *testing.T) {
	root := dir(t)
	files(t, root, packageFiles)
	opt := sourceOptions(t)
	opt.Names = nil
	if s, err := OpenSource(root, opt); err == nil {
		s.Close()
		t.Fatal("OpenSource without a name rule opened the tree")
	}
}

// Names no entry path can hold are refused by the walk itself. A host that
// will not make the name skips its case.
func TestSourceRefusesShapelessNames(t *testing.T) {
	for _, name := range []string{`back\slash`, "co:lon", "\uFEFFbom", "bad\xffutf8"} {
		root := dir(t)
		files(t, root, map[string]string{"metais.yaml": "", "templates/ok": ""})
		if err := os.WriteFile(filepath.Join(root, "templates", name), nil, 0o644); err != nil {
			continue
		}
		if es, err := os.ReadDir(filepath.Join(root, "templates")); err != nil || !slices.ContainsFunc(es, func(e os.DirEntry) bool { return e.Name() == name }) {
			continue // the host changed the name
		}
		wantProblems(t, refused(t, root, sourceOptions(t)), []Problem{
			{Kind: ProblemNameShape, Path: "templates/" + name},
		})
	}
}

func TestSourceLimits(t *testing.T) {
	cases := []struct {
		name  string
		lower string
		files map[string]string
		want  []Problem
		stop  bool
	}{
		{"entries", "entries=2", map[string]string{"metais.yaml": "", "templates/a": "", "templates/b": ""},
			[]Problem{{Kind: ProblemLimit, Limit: limits.Entries}}, true},
		{"entry size", "entry-size=3", map[string]string{"metais.yaml": "", "assets/big": "1234", "assets/ok": "123"},
			[]Problem{{Kind: ProblemLimit, Path: "assets/big", Limit: limits.EntrySize}}, false},
		{"package uncompressed", "package-uncompressed=5", map[string]string{"metais.yaml": "12", "assets/a": "123", "assets/b": "1"},
			[]Problem{{Kind: ProblemLimit, Limit: limits.PackageUncompressed}}, true},
		{"path length", "path-length=12", map[string]string{"metais.yaml": "", "assets/abcde": "", "assets/abcdef": ""},
			[]Problem{{Kind: ProblemLimit, Path: "assets/abcdef", Limit: limits.PathLength}}, false},
		{"path depth", "path-depth=3", map[string]string{"metais.yaml": "", "assets/a/b": "", "assets/a/b2/c": ""},
			[]Problem{{Kind: ProblemLimit, Path: "assets/a/b2/c", Limit: limits.PathDepth}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := dir(t)
			files(t, root, c.files)
			se := refused(t, root, sourceOptions(t, c.lower))
			wantProblems(t, se, c.want)
			if se.Incomplete != c.stop {
				t.Errorf("Incomplete = %v, want %v", se.Incomplete, c.stop)
			}
		})
	}
}

// The walk stops once it holds more problems than the findings limit, so
// a directory of endless problems costs a bounded walk. The bound is a
// count of problems, not a time.
func TestSourceStopsAtTheFindingsLimit(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"metais.yaml": ""})
	for i := range 20 {
		if err := os.MkdirAll(filepath.Join(root, "assets", "e"+strings.Repeat("x", i)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	var listed int
	opt := sourceOptions(t, "findings=5")
	opt.listed = func(string) { listed++ }
	se := refused(t, root, opt)
	if len(se.Problems) != 6 || !se.Incomplete {
		t.Errorf("%d problems, Incomplete %v; want 6 and true", len(se.Problems), se.Incomplete)
	}
	if listed > 2+6 {
		t.Errorf("listed %d directories after the bound was reached", listed)
	}
}

// A directory of more names than the entries limit and the findings limit
// together is refused once its listing passes that bound.
func TestSourceBoundsAListing(t *testing.T) {
	root := dir(t)
	m := map[string]string{"metais.yaml": ""}
	for i := range 12 {
		m["assets/f"+strings.Repeat("x", i)] = ""
	}
	files(t, root, m)
	se := refused(t, root, sourceOptions(t, "entries=5", "findings=5"))
	wantProblems(t, se, []Problem{{Kind: ProblemLimit, Path: "assets", Limit: limits.Entries}})
}

// An entry that changes after the tree was opened is an error from Open or
// Read, never different content.
func TestSourceEntryChangedSinceOpening(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{
		"metais.yaml":   "m",
		"assets/grows":  "abc",
		"assets/shrink": "abc",
		"assets/swap":   "abc",
		"assets/gone":   "abc",
		"assets/link":   "abc",
		"assets/other":  "xyz",
		"assets/late":   "abc",
	})
	s := openSource(t, root, sourceOptions(t))
	p := func(n string) string { return filepath.Join(root, "assets", n) }

	late, err := s.Open("assets/late")
	if err != nil {
		t.Fatal(err)
	}
	defer late.Close()

	write := func(name, body string) {
		if err := os.WriteFile(p(name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("grows", "abcd")
	write("shrink", "ab")
	if err := os.Rename(p("swap"), p("swap.old")); err != nil {
		t.Fatal(err)
	}
	write("swap", "xyz")
	if err := os.Remove(p("gone")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(p("link")); err != nil {
		t.Fatal(err)
	}
	symlink(t, root, "other", "assets/link")

	for name, want := range map[string]Reason{
		"grows": Changed, "shrink": Changed, "swap": Changed, "gone": NotFound, "link": Link,
	} {
		r, err := s.Open("assets/" + name)
		if r != nil {
			r.Close()
		}
		t.Run(name, func(t *testing.T) { wantReason(t, err, want, root) })
	}

	// Opened before it changed: the reader ends in an error rather than
	// yield bytes the walk did not find.
	write("late", "abcdef")
	_, err = io.ReadAll(late)
	wantReason(t, err, Changed, root)
}

func TestSourceStatAndOpenRefuseNonEntries(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"metais.yaml": "m", "README.md": "r", ".git/HEAD": "h"})
	s := openSource(t, root, sourceOptions(t))
	for _, p := range []string{"README.md", ".git/HEAD", "../x", "METAIS.YAML", "templates"} {
		if _, err := s.Stat(p); !errors.Is(err, pkgtree.ErrNotExist) {
			t.Errorf("Stat(%q) = %v", p, err)
		}
		if r, err := s.Open(p); r != nil || !errors.Is(err, pkgtree.ErrNotExist) {
			t.Errorf("Open(%q) = %v", p, err)
		}
	}
}

// The caller's own source directory may be a link to one.
func TestSourceFollowsTheCallersLink(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"pkg/metais.yaml": "m"})
	symlink(t, root, "pkg", "named")
	s := openSource(t, filepath.Join(root, "named"), sourceOptions(t))
	if got := s.List(); !slices.Equal(got, []string{"metais.yaml"}) {
		t.Errorf("List = %q", got)
	}
	_, err := OpenSource(filepath.Join(root, "pkg", "metais.yaml"), sourceOptions(t))
	wantReason(t, err, NotDirectory, root)
}

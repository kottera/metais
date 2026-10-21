// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package site holds the tests of the metais.org site, whose source is this
// directory: the pages under public/, and build.sh, which writes the
// published tree with every specification schema and every CLI output
// schema at the path its $id names.
package site

import (
	"bytes"
	"encoding/json"
	"html"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// origin is the site's address. Every schema's $id is beneath it, and the
// published site serves https://metais.org/<p> from <p> in its tree.
const origin = "https://metais.org/"

// schemaSource is one directory of JSON Schemas the site publishes: dir, from
// this directory, whose files the site serves at at followed by their path
// below dir.
type schemaSource struct {
	dir string
	at  string
}

// sources are the schemas the site publishes: the specification's under
// schemas/, and the tool's output schemas, which ADR-011 D7 item 5 keeps
// outside the specification, under cli/schemas/.
var sources = []schemaSource{
	{dir: "../spec/schemas", at: "schemas/"},
	{dir: "../docs/cli/schemas", at: "cli/schemas/"},
}

// helpGolden is the tool's help as one metais.help/1 document, from this
// directory. It is generated from the commands' own declarations, so the
// pages' commands are checked against what the tool accepts.
const helpGolden = "../internal/cli/testdata/help/help.json"

// pages are the site's pages, by their paths in the published tree.
var pages = []string{"index.html", "install/index.html", "getting-started/index.html"}

// schema is one published schema: its path in the repository, its path below
// its source's directory, slash separated, its $id, and its bytes.
type schema struct {
	name string
	rel  string
	id   string
	data []byte
}

// readSchemas reads every file below src's directory.
func readSchemas(t *testing.T, src schemaSource) []schema {
	t.Helper()
	var out []schema
	dir := filepath.FromSlash(src.dir)
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		name := strings.TrimPrefix(src.dir, "../") + "/" + rel
		var doc struct {
			ID string `json:"$id"`
		}
		if err := json.Unmarshal(b, &doc); err != nil {
			t.Errorf("%s: %v", name, err)
		}
		out = append(out, schema{name: name, rel: rel, id: doc.ID, data: b})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatalf("no schema under %s", src.dir)
	}
	return out
}

// allSchemas reads every source's schemas.
func allSchemas(t *testing.T) []schema {
	t.Helper()
	var out []schema
	for _, src := range sources {
		for _, s := range readSchemas(t, src) {
			s.rel = src.at + s.rel
			out = append(out, s)
		}
	}
	return out
}

// sitePath is the path in the published tree that serves id: the part of id
// after the site's origin. It reports false for an $id the site cannot serve:
// one outside the origin, or one with a query, a fragment, or a segment that
// is empty, . or .., which a static host would not map to the same file.
func sitePath(id string) (string, bool) {
	p, ok := strings.CutPrefix(id, origin)
	if !ok || p == "" || strings.ContainsAny(p, "?#\\") {
		return "", false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return "", false
		}
	}
	return p, true
}

// TestSchemaIDPaths requires every published schema's $id to name the path
// build.sh publishes it at: schemas/ and its path below spec/schemas, or
// cli/schemas/ and its path below docs/cli/schemas, so that each $id
// resolves to its own schema.
func TestSchemaIDPaths(t *testing.T) {
	for _, s := range allSchemas(t) {
		p, ok := sitePath(s.id)
		if !ok {
			t.Errorf("%s: $id %q is not a path the site at %s serves", s.name, s.id, origin)
			continue
		}
		if p != s.rel {
			t.Errorf("%s: $id %q resolves to %s on the site, but the site publishes this file at %s", s.name, s.id, p, s.rel)
		}
	}
}

func TestSitePath(t *testing.T) {
	for id, want := range map[string]string{
		"https://metais.org/schemas/1.0/manifest.schema.json": "schemas/1.0/manifest.schema.json",
		"https://metais.org/x":                                "x",
	} {
		if got, ok := sitePath(id); !ok || got != want {
			t.Errorf("sitePath(%q) = %q, %v; want %q", id, got, ok, want)
		}
	}
	for _, id := range []string{
		"", "https://metais.org/", "http://metais.org/x", "https://example.org/x",
		"https://metais.org/a//b", "https://metais.org/a/../b", "https://metais.org/./a",
		"https://metais.org/a/", "https://metais.org/a?b", "https://metais.org/a#b",
	} {
		if got, ok := sitePath(id); ok {
			t.Errorf("sitePath(%q) = %q, true; want it refused", id, got)
		}
	}
}

// build runs build.sh into a new directory and returns it.
func build(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("build.sh is a POSIX shell script; the site is built on Linux")
	}
	out := filepath.Join(testenv.Dir(t), "site")
	cmd := exec.Command("sh", "build.sh", out)
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + testenv.Home(t), "LC_ALL=C"}
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sh build.sh: %v\n%s", err, b)
	}
	return out
}

// TestBuild builds the site and requires the published tree to hold the
// pages, the CNAME, and every specification schema and every CLI output
// schema byte for byte at the path its $id names, and nothing else under
// schemas/ or cli/.
func TestBuild(t *testing.T) {
	out := build(t)

	for _, p := range pages {
		if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(p))); err != nil {
			t.Errorf("page %s: %v", p, err)
		}
	}
	if b, err := os.ReadFile(filepath.Join(out, "CNAME")); err != nil {
		t.Error(err)
	} else if string(b) != "metais.org\n" {
		t.Errorf("CNAME holds %q, want %q", b, "metais.org\n")
	}

	published := map[string]bool{}
	for _, s := range allSchemas(t) {
		p, ok := sitePath(s.id)
		if !ok {
			t.Errorf("%s: $id %q is not a path the site serves", s.name, s.id)
			continue
		}
		published[p] = true
		b, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(p)))
		if err != nil {
			t.Errorf("%s: $id %q: %v", s.name, s.id, err)
			continue
		}
		if !bytes.Equal(b, s.data) {
			t.Errorf("%s: the site's %s, which its $id %q names, differs from %s", s.name, p, s.id, s.name)
		}
	}
	for _, dir := range []string{"schemas", "cli"} {
		err := filepath.WalkDir(filepath.Join(out, dir), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, err := filepath.Rel(out, p)
			if err != nil {
				return err
			}
			if rel = filepath.ToSlash(rel); !published[rel] {
				t.Errorf("the site publishes %s, which no schema's $id names", rel)
			}
			return nil
		})
		if err != nil {
			t.Error(err)
		}
	}
}

// TestBuildRefusesExisting requires build.sh to refuse a directory that
// exists, so a build never mixes with what an earlier one left.
func TestBuildRefusesExisting(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("build.sh is a POSIX shell script; the site is built on Linux")
	}
	out := testenv.Dir(t)
	cmd := exec.Command("sh", "build.sh", out)
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + testenv.Home(t), "LC_ALL=C"}
	if b, err := cmd.CombinedOutput(); err == nil {
		t.Fatalf("sh build.sh %s succeeded over an existing directory:\n%s", out, b)
	}
	if entries, err := os.ReadDir(out); err != nil || len(entries) != 0 {
		t.Fatalf("the refused build wrote into %s: %v, %v", out, entries, err)
	}
}

// commandLine matches a line of a page's code that runs metais.
var commandLine = regexp.MustCompile(`^metais(\s.*)?$`)

// TestPageCommands requires every metais command line in a page's code
// blocks to name a command metais has, with flags that command takes, so the
// pages cannot drift from metais help.
func TestPageCommands(t *testing.T) {
	b, err := os.ReadFile(helpGolden)
	if err != nil {
		t.Fatal(err)
	}
	var help struct {
		Result struct {
			Commands []struct {
				Name  string `json:"name"`
				Flags []struct {
					Name string `json:"name"`
				} `json:"flags"`
			} `json:"commands"`
			GlobalFlags []struct {
				Name string `json:"name"`
			} `json:"global_flags"`
		} `json:"result"`
	}
	if err := json.Unmarshal(b, &help); err != nil {
		t.Fatal(err)
	}
	flags := map[string][]string{}
	for _, c := range help.Result.Commands {
		known := []string{}
		for _, f := range c.Flags {
			known = append(known, f.Name)
		}
		for _, f := range help.Result.GlobalFlags {
			known = append(known, f.Name)
		}
		flags[c.Name] = known
	}
	if len(flags) == 0 {
		t.Fatalf("%s lists no command", helpGolden)
	}

	code := regexp.MustCompile(`(?s)<pre><code>(.*?)</code></pre>`)
	tag := regexp.MustCompile(`<[^>]*>`)
	seen := 0
	for _, p := range pages {
		b, err := os.ReadFile(filepath.Join("public", filepath.FromSlash(p)))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range code.FindAllSubmatch(b, -1) {
			text := html.UnescapeString(tag.ReplaceAllString(string(m[1]), ""))
			for line := range strings.SplitSeq(text, "\n") {
				line = strings.TrimSpace(line)
				if !commandLine.MatchString(line) {
					continue
				}
				seen++
				fields := strings.Fields(line)
				if len(fields) < 2 {
					t.Errorf("%s: %q names no command", p, line)
					continue
				}
				name := fields[1]
				known, ok := flags[name]
				if !ok {
					t.Errorf("%s: %q: metais has no command %s", p, line, name)
					continue
				}
				for _, f := range fields[2:] {
					if !strings.HasPrefix(f, "--") {
						continue
					}
					f, _, _ = strings.Cut(f, "=")
					if !slices.Contains(known, f) {
						t.Errorf("%s: %q: metais %s takes no flag %s", p, line, name, f)
					}
				}
			}
		}
	}
	if seen == 0 {
		t.Fatal("the pages hold no metais command line; the check is blind")
	}
}

// TestWorkflow requires the Pages workflow to build the site with build.sh,
// from a release tag alone, and to run nothing while the repository is
// private: GitHub's free plan publishes Pages only from a public repository.
func TestWorkflow(t *testing.T) {
	b, err := os.ReadFile("../.github/workflows/pages.yml")
	if err != nil {
		t.Fatal(err)
	}
	w := string(b)
	for _, want := range []string{
		"site/build.sh _site",
		"tags: ['v*']",
		"if: ${{ !github.event.repository.private && startsWith(github.ref, 'refs/tags/v') }}",
		"actions/upload-pages-artifact@",
		"actions/deploy-pages@",
	} {
		if !strings.Contains(w, want) {
			t.Errorf(".github/workflows/pages.yml does not contain %q", want)
		}
	}
	// Every job is gated, so a job added without the gate fails here.
	jobs := strings.Count(w, "\n    runs-on:")
	gates := strings.Count(w, "\n    if: ${{ !github.event.repository.private && startsWith(github.ref, 'refs/tags/v') }}")
	if jobs == 0 || jobs != gates {
		t.Errorf(".github/workflows/pages.yml has %d jobs and %d of them gated on a public repository and a v tag", jobs, gates)
	}
}

// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package site holds the tests of the metais.org site, whose source is this
// directory: the METAIS package under package/, which renders its pages;
// the static files under public/; and build.sh, which builds metais from
// the commit, renders the package with it from the inputs
// internal/tools/site exports, and writes the published tree with the
// static files and every specification schema and every CLI output schema
// at the path its $id names (ADR-013 D8).
package site

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"metais.org/metais/internal/testenv"
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
var pages = []string{"index.html", "install/index.html", "getting-started/index.html", "metais/index.html", "metais/cmd/metais/index.html"}

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

// hostEnviron is the environment this test binary started with, taken
// before any test points HOME at a temporary directory: go env reads the
// host's caches from it.
var hostEnviron = os.Environ()

// goHost is the host's toolchain and its caches (testenv.GoHost), read once
// by the go command of the toolchain running this test.
var goHost = sync.OnceValues(func() (testenv.GoHost, error) {
	exe := "go"
	if root := runtime.GOROOT(); root != "" {
		exe = filepath.Join(root, "bin", "go")
	}
	cmd := exec.Command(exe, testenv.GoHostQuery...)
	cmd.Env = append(slices.Clone(hostEnviron), "GOTOOLCHAIN=local")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return testenv.GoHost{}, fmt.Errorf("go %s: %v\n%s", strings.Join(testenv.GoHostQuery, " "), err, stderr.Bytes())
	}
	return testenv.ParseGoHost(hostEnviron, out)
})

// buildCmd is sh build.sh with args, in the whole environment testenv.GoEnv
// gives over home: it takes nothing from the host but its toolchain, its
// caches and PATH, and never reaches the network.
func buildCmd(t testing.TB, home string, args ...string) *exec.Cmd {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("build.sh is a POSIX shell script; the site is built on Linux")
	}
	h, err := goHost()
	if err != nil {
		t.Fatalf("locating the host's toolchain: %v", err)
	}
	cmd := exec.Command("sh", append([]string{"build.sh"}, args...)...)
	cmd.Env = append(testenv.GoEnv(h, home), "LC_ALL=C")
	return cmd
}

// build runs build.sh into a new directory and returns it.
func build(t testing.TB) string {
	t.Helper()
	out := filepath.Join(testenv.Dir(t), "site")
	if b, err := buildCmd(t, testenv.Dir(t), out).CombinedOutput(); err != nil {
		t.Fatalf("sh build.sh: %v\n%s", err, b)
	}
	return out
}

// built is one build of the site, shared by the tests that read it.
var built struct {
	once sync.Once
	dir  string
	err  string
}

// builtSite returns the shared build, building it on first use beneath a
// directory TestMain removes.
func builtSite(t *testing.T) string {
	t.Helper()
	built.once.Do(func() {
		var out bytes.Buffer
		cmd := buildCmd(t, filepath.Join(sharedDir, "home"), filepath.Join(sharedDir, "site"))
		cmd.Stdout, cmd.Stderr = &out, &out
		if err := os.Mkdir(filepath.Join(sharedDir, "home"), 0o755); err != nil {
			built.err = err.Error()
			return
		}
		if err := cmd.Run(); err != nil {
			built.err = fmt.Sprintf("sh build.sh: %v\n%s", err, out.Bytes())
			return
		}
		built.dir = filepath.Join(sharedDir, "site")
	})
	if built.err != "" {
		t.Fatal(built.err)
	}
	return built.dir
}

// sharedDir holds the shared build, made and removed by TestMain.
var sharedDir string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "metais-site-test-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	sharedDir = dir
	code := m.Run()
	if err := os.RemoveAll(dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
	}
	os.Exit(code)
}

// tree reads every file below root, by its slash-separated path.
func tree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !d.Type().IsRegular() {
			t.Errorf("%s is not a regular file", p)
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		files[filepath.ToSlash(rel)] = b
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// TestBuild builds the site and requires the published tree to hold the
// pages, the CNAME, and every specification schema and every CLI output
// schema byte for byte at the path its $id names, and nothing else under
// schemas/ or cli/.
func TestBuild(t *testing.T) {
	out := builtSite(t)

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

// TestStaticFiles requires the built site to serve every file of public/
// at its own path, byte for byte (ADR-013 D1 item 5): install.sh among
// them, once it is there.
func TestStaticFiles(t *testing.T) {
	site := tree(t, builtSite(t))
	static := tree(t, "public")
	if len(static) == 0 {
		t.Fatal("public/ holds no file; the check is blind")
	}
	for p, want := range static {
		if got, ok := site[p]; !ok {
			t.Errorf("the site does not serve public/%s at /%s", p, p)
		} else if !bytes.Equal(got, want) {
			t.Errorf("the site's /%s differs from public/%s", p, p)
		}
	}
}

// TestBuildReproducible builds the site a second time and requires the
// same tree, byte for byte, as the shared build of the same commit.
func TestBuildReproducible(t *testing.T) {
	a := tree(t, builtSite(t))
	b := tree(t, build(t))
	for _, p := range slices.Sorted(maps.Keys(a)) {
		if got, ok := b[p]; !ok {
			t.Errorf("the second build has no %s", p)
		} else if !bytes.Equal(got, a[p]) {
			t.Errorf("the two builds' %s differ", p)
		}
	}
	for p := range b {
		if _, ok := a[p]; !ok {
			t.Errorf("only the second build has %s", p)
		}
	}
}

// footer is ADR-013 D2 item 1's footer: METAIS is made by Kottera, with a
// link to kottera.com.
const footer = `METAIS is made by <a href="https://kottera.com/">Kottera</a>`

// TestFooter requires every page of the built site to end with the footer,
// once.
func TestFooter(t *testing.T) {
	foot := regexp.MustCompile(`(?s)<footer>(.*?)</footer>\s*</body>\s*</html>\s*$`)
	n := 0
	for p, b := range tree(t, builtSite(t)) {
		if !strings.HasSuffix(p, ".html") {
			continue
		}
		n++
		m := foot.FindSubmatch(b)
		if m == nil || !strings.Contains(string(m[1]), footer) || bytes.Count(b, []byte("<footer>")) != 1 {
			t.Errorf("the page %s does not end with one footer saying %s", p, footer)
		}
	}
	if n < len(pages) {
		t.Errorf("the site holds %d pages, fewer than the %d it must; the check is blind", n, len(pages))
	}
}

// TestBuildRefusesExisting requires build.sh to refuse a directory that
// exists, so a build never mixes with what an earlier one left.
func TestBuildRefusesExisting(t *testing.T) {
	out := testenv.Dir(t)
	if b, err := buildCmd(t, testenv.Dir(t), out).CombinedOutput(); err == nil {
		t.Fatalf("sh build.sh %s succeeded over an existing directory:\n%s", out, b)
	}
	if entries, err := os.ReadDir(out); err != nil || len(entries) != 0 {
		t.Fatalf("the refused build wrote into %s: %v, %v", out, entries, err)
	}
}

// TestBuildUsage requires build.sh to refuse a command line it does not
// take, with its usage and status 2, before it builds anything.
func TestBuildUsage(t *testing.T) {
	dir := testenv.Dir(t)
	for _, args := range [][]string{
		nil,
		{"a", "b"},
		{"--release", "1.0.0", filepath.Join(dir, "site")},
		{"--release", "1.0.0", "SHA256SUMS", filepath.Join(dir, "site"), "extra"},
	} {
		b, err := buildCmd(t, testenv.Dir(t), args...).CombinedOutput()
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 2 || !strings.Contains(string(b), "usage: site/build.sh") {
			t.Errorf("sh build.sh %q: %v\n%s\nwant its usage and status 2", args, err, b)
		}
	}
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
		t.Errorf("a refused command line wrote %v, %v", entries, err)
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
	site := builtSite(t)
	for _, p := range pages {
		b, err := os.ReadFile(filepath.Join(site, filepath.FromSlash(p)))
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

// goImport is the content of every go-import page's go-import meta tag
// (ADR-013 D1 item 2): the module's path, its VCS, and the repository that
// holds it.
const goImport = "metais.org/metais git https://github.com/kottera/metais"

// goSource is the content of every go-import page's go-source meta tag: the
// module's path, the repository's home, and the templates pkg.go.dev fills
// in to link a directory and a line of a file to their source on main.
const goSource = "metais.org/metais https://github.com/kottera/metais" +
	" https://github.com/kottera/metais/tree/main{/dir}" +
	" https://github.com/kottera/metais/blob/main{/dir}/{file}#L{line}"

// goImportDir is the directory, in the published tree, of
// the module's go-import pages (ADR-013 D1 item 5): the page for the package
// path metais.org/<p> is <p>/index.html, served at /<p>/.
const goImportDir = "metais"

// goPackages are the package paths a user can name, each of which the site
// serves a go-import page for (ADR-013 D1 item 2), and no other.
var goPackages = []string{"metais.org/metais", "metais.org/metais/cmd/metais"}

// metaTag matches one meta tag with a name and a content, in that order.
var metaTag = regexp.MustCompile(`<meta name="([^"]*)" content="([^"]*)">`)

// goImportPages reads the go-import pages below root and returns each one's
// bytes by the package path it serves. A file below goImportDir that is not
// a directory's index.html is an error: every page there is a package's.
func goImportPages(t *testing.T, root string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	dir := filepath.Join(root, goImportDir)
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		pkg, ok := strings.CutSuffix(rel, "/index.html")
		if !ok {
			t.Errorf("%s is under %s/ but is not a package's index.html", rel, goImportDir)
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		out["metais.org/"+pkg] = b
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// checkGoImportPages requires the pages to serve exactly goPackages, each
// with one go-import tag whose content is goImport and one go-source tag
// whose content is goSource.
func checkGoImportPages(t *testing.T, where string, got map[string][]byte) {
	t.Helper()
	paths := slices.Sorted(maps.Keys(got))
	if !slices.Equal(paths, goPackages) {
		t.Errorf("%s: the go-import pages serve the package paths %q, want exactly %q", where, paths, goPackages)
	}
	for _, pkg := range paths {
		tags := map[string][]string{}
		for _, m := range metaTag.FindAllSubmatch(got[pkg], -1) {
			tags[string(m[1])] = append(tags[string(m[1])], html.UnescapeString(string(m[2])))
		}
		if imp := tags["go-import"]; len(imp) != 1 || imp[0] != goImport {
			t.Errorf("%s: the page for %s has go-import %q, want exactly one, %q", where, pkg, imp, goImport)
		}
		if src := tags["go-source"]; len(src) != 1 || src[0] != goSource {
			t.Errorf("%s: the page for %s has go-source %q, want exactly one, %q", where, pkg, src, goSource)
		}
	}
}

// TestGoImportPages holds the module's go-import pages to ADR-013 D1 item 2,
// in the tree build.sh publishes: one page for each package
// path a user can name and no other, each with the go-import content that
// resolves metais.org/metais to its repository, and the go-source content
// that links each directory and file to its source.
func TestGoImportPages(t *testing.T) {
	checkGoImportPages(t, "the built site", goImportPages(t, builtSite(t)))
}

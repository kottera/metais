// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
	"metais.org/metais/internal/tools/release/buildline"
)

// hostEnviron is the environment this test binary started with.
var hostEnviron = os.Environ()

// gitIn runs git with args in dir, with nothing of the host's git
// configuration, and returns its standard output.
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=Site Test", "-c", "user.email=site@example.org", "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	cmd.Env = testenv.GitEnv(append(slices.Clone(hostEnviron), "HOME="+testenv.Home(t)))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// writeFiles writes each file of files, by its slash-separated path below
// dir.
func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, text := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// fixtureRepo is a repository whose spec/ was released three times, as
// spec-1.0, spec-1.2 and spec-1.10, and then changed: the tags' files and
// the draft's all differ. It also carries tags the site must not read as
// releases.
func fixtureRepo(t *testing.T) string {
	t.Helper()
	dir := testenv.Dir(t)
	gitIn(t, dir, "init", "-q", "-b", "main")
	for _, rev := range []string{"1.0", "1.10", "1.2"} {
		writeFiles(t, dir, map[string]string{
			"spec/README.md":    "# The specification " + rev + "\n\n1. [Intro](01-intro.md)\n",
			"spec/01-intro.md":  "# 1. Intro\n\n## 1.1 Scope " + rev + "\n\nSee [the readme](README.md) and [§1.1](#11-scope-" + strings.ReplaceAll(rev, ".", "") + ").\n",
			"spec/notes.txt":    "not Markdown\n",
			"spec/sub/inner.md": "# Not a chapter\n",
		})
		gitIn(t, dir, "add", "-A")
		gitIn(t, dir, "commit", "-q", "-m", "spec "+rev)
		gitIn(t, dir, "tag", "spec-"+rev)
	}
	gitIn(t, dir, "tag", "spec-1.0-rc.1")
	gitIn(t, dir, "tag", "spec-01.3")
	gitIn(t, dir, "tag", "v1.0.0")
	writeFiles(t, dir, map[string]string{
		"spec/01-intro.md": "# 1. Intro\n\n## 1.1 Scope draft\n\nSee [the appendix](A-more.md#a1-more).\n",
		"spec/A-more.md":   "# Appendix A. More\n\n## A.1 More\n",
	})
	return dir
}

func TestSpecInput(t *testing.T) {
	dir := fixtureRepo(t)
	head := strings.TrimSpace(gitIn(t, dir, "rev-parse", "HEAD"))
	b := &builder{root: dir, environ: hostEnviron}
	got, err := b.specInput()
	if err != nil {
		t.Fatal(err)
	}

	// The draft is the checkout's files, uncommitted ones included, read at
	// its commit.
	if got.Draft.Revision != "draft" || got.Draft.Ref != head || got.Draft.Path != "/spec/draft/" {
		t.Errorf("draft is %q at %q, served at %q; want draft at %s, served at /spec/draft/", got.Draft.Revision, got.Draft.Ref, got.Draft.Path, head)
	}
	files := func(r revision) (out []string) {
		for _, f := range r.Files {
			out = append(out, f.File+" "+f.Path+" "+f.Title)
		}
		return out
	}
	wantDraft := []string{
		"01-intro.md /spec/draft/01-intro/ 1. Intro",
		"A-more.md /spec/draft/A-more/ Appendix A. More",
		"README.md /spec/draft/ The specification 1.2",
	}
	if f := files(got.Draft); !slices.Equal(f, wantDraft) {
		t.Errorf("the draft's files are %q, want %q", f, wantDraft)
	}
	if h := got.Draft.Files[0].HTML; !strings.Contains(h, `<a href="/spec/draft/A-more/#a1-more">`) {
		t.Errorf("the draft's link to its appendix is not the appendix's page:\n%s", h)
	}

	// The releases are the tags spec-<M>.<m>, in numeric order, each read
	// from its tag, not from the checkout.
	var revs []string
	for _, r := range got.Releases {
		revs = append(revs, r.Revision+" "+r.Ref+" "+r.Path)
	}
	wantRevs := []string{"1.0 spec-1.0 /spec/1.0/", "1.2 spec-1.2 /spec/1.2/", "1.10 spec-1.10 /spec/1.10/"}
	if !slices.Equal(revs, wantRevs) {
		t.Fatalf("the releases are %q, want %q", revs, wantRevs)
	}
	for _, r := range got.Releases {
		want := []string{
			"01-intro.md /spec/" + r.Revision + "/01-intro/ 1. Intro",
			"README.md /spec/" + r.Revision + "/ The specification " + r.Revision,
		}
		if f := files(r); !slices.Equal(f, want) {
			t.Errorf("%s's files are %q, want %q", r.Ref, f, want)
		}
		intro := r.Files[0]
		id := "11-scope-" + strings.ReplaceAll(r.Revision, ".", "")
		if len(intro.Headings) != 2 || intro.Headings[1].ID != id {
			t.Errorf("%s's 01-intro.md has headings %v, want its own 1.1 with id %s", r.Ref, intro.Headings, id)
		}
		for _, link := range []string{`href="/spec/` + r.Revision + `/"`, `href="#` + id + `"`} {
			if !strings.Contains(intro.HTML, link) {
				t.Errorf("%s's 01-intro.md lacks %s:\n%s", r.Ref, link, intro.HTML)
			}
		}
	}
}

func TestSpecInputNoRelease(t *testing.T) {
	dir := testenv.Dir(t)
	gitIn(t, dir, "init", "-q", "-b", "main")
	writeFiles(t, dir, map[string]string{"spec/README.md": "# Spec\n"})
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "spec")
	got, err := (&builder{root: dir, environ: hostEnviron}).specInput()
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(got.Releases)
	if err != nil || string(b) != "[]" {
		t.Errorf("with no spec tag the releases are %s, %v; want []", b, err)
	}
}

// TestSpecLinksResolve converts the repository's own spec/ and requires
// every link it makes to one of the draft's pages to name a page the draft
// has, and an id that page carries: a heading's, or a statement's anchor.
func TestSpecLinksResolve(t *testing.T) {
	root := filepath.Join("..", "..", "..", "spec")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			if files[e.Name()], err = os.ReadFile(filepath.Join(root, e.Name())); err != nil {
				t.Fatal(err)
			}
		}
	}
	r, err := convertRevision("draft", "0123456789abcdef0123456789abcdef01234567", files)
	if err != nil {
		t.Fatal(err)
	}
	idAttr := regexp.MustCompile(`\sid="([^"]+)"`)
	href := regexp.MustCompile(`\shref="([^"]+)"`)
	ids := map[string]map[string]bool{}
	for _, f := range r.Files {
		ids[f.Path] = map[string]bool{}
		for _, m := range idAttr.FindAllStringSubmatch(f.HTML, -1) {
			ids[f.Path][m[1]] = true
		}
	}
	checked := 0
	for _, f := range r.Files {
		for _, m := range href.FindAllStringSubmatch(f.HTML, -1) {
			dest := m[1]
			if strings.HasPrefix(dest, "#") {
				dest = f.Path + dest
			}
			if !strings.HasPrefix(dest, "/spec/") {
				if !strings.HasPrefix(dest, "https://") {
					t.Errorf("%s: link %q is neither a page of the site nor an absolute URL", f.File, m[1])
				}
				continue
			}
			checked++
			page, frag, _ := strings.Cut(dest, "#")
			if ids[page] == nil {
				t.Errorf("%s: link %q names no page of the draft", f.File, m[1])
			} else if frag != "" && !ids[page][frag] {
				t.Errorf("%s: link %q names no id of %s", f.File, m[1], page)
			}
		}
	}
	if checked == 0 {
		t.Fatal("the specification links none of its own pages; the check is blind")
	}
}

func TestSpecLink(t *testing.T) {
	files := map[string][]byte{"README.md": nil, "02-container.md": nil}
	link := specLink("/spec/1.0/", "spec-1.0", files)
	for dest, want := range map[string]string{
		"02-container.md":            "/spec/1.0/02-container/",
		"02-container.md#21-x":       "/spec/1.0/02-container/#21-x",
		"./02-container.md":          "/spec/1.0/02-container/",
		"README.md":                  "/spec/1.0/",
		"#local":                     "#local",
		"/schemas/x.json":            "/schemas/x.json",
		"https://example.org/a.md":   "https://example.org/a.md",
		"mailto:a@example.org":       "mailto:a@example.org",
		"03-manifest.md":             repoURL + "/blob/spec-1.0/spec/03-manifest.md",
		"schemas/1.0/m.json":         repoURL + "/blob/spec-1.0/spec/schemas/1.0/m.json",
		"schemas/":                   repoURL + "/tree/spec-1.0/spec/schemas",
		"conformance/README.md#a":    repoURL + "/blob/spec-1.0/spec/conformance/README.md#a",
		"../docs/adr/ADR-001.md":     repoURL + "/blob/spec-1.0/docs/adr/ADR-001.md",
		"sub/../02-container.md#top": "/spec/1.0/02-container/#top",
	} {
		if got := link(dest); got != want {
			t.Errorf("specLink(%q) = %q, want %q", dest, got, want)
		}
	}
}

func TestSkillInput(t *testing.T) {
	text := "---\nname: metais\ndescription: d\n---\n\n# metais\n\nA `.metais` file.\n"
	got, err := skillInput([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != text {
		t.Errorf("Text = %q, want the skill as printed", got.Text)
	}
	if got.HTML != "<h1 id=\"metais\">metais</h1>\n<p>A <code>.metais</code> file.</p>\n" {
		t.Errorf("HTML = %q, want the body after the frontmatter", got.HTML)
	}
	for _, bad := range []string{"# metais\n", "---\nname: metais\n"} {
		if _, err := skillInput([]byte(bad)); !errors.Is(err, errSkill) {
			t.Errorf("skillInput(%q) = %v, want errSkill", bad, err)
		}
	}
}

// sums is a SHA256SUMS of version, one line per shipped target, as the
// release builder writes it.
func sums(version string) string {
	var names []string
	for _, tg := range buildline.Shipped {
		names = append(names, buildline.ArchiveName(version, tg))
	}
	slices.Sort(names)
	var b strings.Builder
	for i, n := range names {
		b.WriteString(strings.Repeat(string("0123456789abcdef"[i]), 64) + "  " + n + "\n")
	}
	return b.String()
}

func TestParseRelease(t *testing.T) {
	r, err := parseRelease("1.2.3", []byte(sums("1.2.3")))
	if err != nil {
		t.Fatal(err)
	}
	if !r.Published || r.Version != "1.2.3" || r.Tag != "v1.2.3" || r.Prerelease ||
		r.SumsURL != "https://github.com/kottera/metais/releases/download/v1.2.3/SHA256SUMS" {
		t.Errorf("parseRelease = %+v", r)
	}
	if len(r.Archives) != len(buildline.Shipped) {
		t.Fatalf("parseRelease lists %d archives, want one per shipped target", len(r.Archives))
	}
	a := r.Archives[0]
	if a.Name != "metais_1.2.3_darwin_amd64.tar.gz" || a.OS != "darwin" || a.Arch != "amd64" ||
		a.SHA256 != strings.Repeat("0", 64) || a.URL != "https://github.com/kottera/metais/releases/download/v1.2.3/metais_1.2.3_darwin_amd64.tar.gz" {
		t.Errorf("the first archive is %+v", a)
	}

	pre, err := parseRelease("1.0.0-rc.1", []byte(sums("1.0.0-rc.1")))
	if err != nil || !pre.Prerelease {
		t.Errorf("parseRelease of a prerelease = %+v, %v; want prerelease true", pre, err)
	}
	if b, _ := json.Marshal(release{Published: false}); string(b) != `{"published":false}` {
		t.Errorf("a build with no release exports %s", b)
	}

	good := sums("1.2.3")
	lines := strings.SplitAfter(strings.TrimSuffix(good, "\n"), "\n")
	for name, c := range map[string]struct {
		version, sums string
		want          error
	}{
		"a leading v":       {"v1.2.3", good, errRelease},
		"build metadata":    {"1.2.3+x", good, errRelease},
		"a short version":   {"1.2", good, errRelease},
		"another version":   {"1.2.4", good, errSums},
		"a missing target":  {"1.2.3", strings.Join(lines[1:], ""), errSums},
		"an extra line":     {"1.2.3", good + strings.Repeat("f", 64) + "  metais_1.2.3_windows_amd64.tar.gz\n", errSums},
		"a repeated line":   {"1.2.3", good + lines[0], errSums},
		"no final newline":  {"1.2.3", strings.TrimSuffix(good, "\n"), errSums},
		"a binary marker":   {"1.2.3", strings.Replace(good, "  metais", " *metais", 1), errSums},
		"upper-case digits": {"1.2.3", strings.Replace(good, strings.Repeat("0", 64), strings.Repeat("A", 64), 1), errSums},
		"out of order":      {"1.2.3", lines[1] + lines[0] + strings.Join(lines[2:], "") + "\n", errSums},
		"empty":             {"1.2.3", "", errSums},
	} {
		if _, err := parseRelease(c.version, []byte(c.sums)); !errors.Is(err, c.want) {
			t.Errorf("%s: parseRelease = %v, want %v", name, err, c.want)
		}
	}
}

// TestRunRefuses holds the refusals run makes before it builds anything.
func TestRunRefuses(t *testing.T) {
	repo := fixtureRepo(t)
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "draft")
	sumsDir := testenv.Dir(t)
	writeFiles(t, sumsDir, map[string]string{"1.0.0": sums("1.0.0"), "2.0.0": sums("2.0.0")})
	sums1, sums2 := filepath.Join(sumsDir, "1.0.0"), filepath.Join(sumsDir, "2.0.0")

	shallow := filepath.Join(testenv.Dir(t), "shallow")
	gitIn(t, repo, "clone", "-q", "--depth", "1", "file://"+repo, shallow)

	withPublic := testenv.Dir(t)
	writeFiles(t, withPublic, map[string]string{"site/public/cli/x.json": "{}"})

	exists := testenv.Dir(t)
	newOut := func() string { return filepath.Join(testenv.Dir(t), "site") }
	for name, c := range map[string]struct {
		b    builder
		want error
	}{
		"an existing out":                 {builder{root: repo, out: exists}, errExists},
		"a static cli/":                   {builder{root: withPublic, out: newOut()}, errPublic},
		"a release from a shallow clone":  {builder{root: shallow, out: newOut(), version: "1.0.0", sums: sums1}, errShallow},
		"another release's sums":          {builder{root: repo, out: newOut(), version: "1.0.1", sums: sums1}, errSums},
		"a release with no tag":           {builder{root: repo, out: newOut(), version: "2.0.0", sums: sums2}, errNotTag},
		"a release whose tag is not HEAD": {builder{root: repo, out: newOut(), version: "1.0.0", sums: sums1}, errNotTag},
		"a release that is no semver":     {builder{root: repo, out: newOut(), version: "1.0", sums: sums1}, errRelease},
	} {
		c.b.environ, c.b.report = hostEnviron, io.Discard
		if err := c.b.run(); !errors.Is(err, c.want) {
			t.Errorf("%s: run = %v, want %v", name, err, c.want)
		}
		if c.want != errExists {
			if _, err := os.Lstat(c.b.out); err == nil {
				t.Errorf("%s: the refused build wrote %s", name, c.b.out)
			}
		}
	}
	if entries, err := os.ReadDir(exists); err != nil || len(entries) != 0 {
		t.Errorf("the build refused over an existing directory wrote into it: %v, %v", entries, err)
	}
}

func TestCopyTree(t *testing.T) {
	src, dst := testenv.Dir(t), testenv.Dir(t)
	writeFiles(t, src, map[string]string{"CNAME": "metais.org\n", "a/b.txt": "b"})
	if err := copyTree(src, dst, "at"); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"at/CNAME": "metais.org\n", "at/a/b.txt": "b"} {
		if b, err := os.ReadFile(filepath.Join(dst, filepath.FromSlash(name))); err != nil || string(b) != want {
			t.Errorf("%s holds %q, %v; want %q", name, b, err, want)
		}
	}
	if err := copyTree(src, dst, "at"); !errors.Is(err, errCollision) {
		t.Errorf("copying over the same paths = %v, want errCollision", err)
	}
	if err := os.Symlink("CNAME", filepath.Join(src, "link")); err != nil {
		t.Fatal(err)
	}
	if err := copyTree(src, testenv.Dir(t), ""); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("copying a symbolic link = %v, want it refused", err)
	}
}

func TestCLIUsage(t *testing.T) {
	abs := testenv.Dir(t)
	for _, args := range [][]string{
		nil,
		{"-root", abs},
		{"-root", "relative", "-out", filepath.Join(abs, "site")},
		{"-root", abs, "-out", "site"},
		{"-root", abs, "-out", filepath.Join(abs, "site"), "-data", "data"},
		{"-root", abs, "-out", filepath.Join(abs, "site"), "-release", "1.0.0"},
		{"-root", abs, "-out", filepath.Join(abs, "site"), "-sums", filepath.Join(abs, "SHA256SUMS")},
		{"-root", abs, "-out", filepath.Join(abs, "site"), "extra"},
	} {
		var stderr bytes.Buffer
		if got := cli(args, io.Discard, &stderr, hostEnviron); got != 2 {
			t.Errorf("cli(%q) = %d, want 2", args, got)
		}
	}
}

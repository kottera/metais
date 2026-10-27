// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package site

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"metais.org/metais/internal/testenv"
)

// The reference pages are generated from metais help --json and metais
// skill --print (ADR-008 A13), and the tests here hold each one to the help
// it came from: the inputs the build step exports from the metais built
// from this commit, which every build of the commit renders from.

// exported is one run of the build step with -data, which keeps the inputs
// it exported, and a metais built from this checkout, which renders the
// site package from inputs a test changes.
var exported struct {
	once   sync.Once
	data   string
	metais string
	err    string
}

// exportedInputs returns the directory of the exported inputs and the
// path of the metais built from this checkout, making both on first use
// beneath the directory TestMain removes.
func exportedInputs(t *testing.T) (data, metais string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the site is built on Linux and macOS")
	}
	exported.once.Do(func() {
		h, err := goHost()
		if err != nil {
			exported.err = err.Error()
			return
		}
		root, err := filepath.Abs("..")
		if err != nil {
			exported.err = err.Error()
			return
		}
		dir := filepath.Join(sharedDir, "exported")
		home := filepath.Join(dir, "home")
		if err := os.MkdirAll(home, 0o755); err != nil {
			exported.err = err.Error()
			return
		}
		data = filepath.Join(dir, "data")
		metais = filepath.Join(dir, "metais")
		for _, args := range [][]string{
			{"run", "./internal/tools/site", "-root", root, "-out", filepath.Join(dir, "site"), "-data", data},
			{"build", "-trimpath", "-o", metais, "./cmd/metais"},
		} {
			cmd := exec.Command(h.Go(), args...)
			cmd.Dir = root
			cmd.Env = append(testenv.GoEnv(h, home), "LC_ALL=C")
			if b, err := cmd.CombinedOutput(); err != nil {
				exported.err = fmt.Sprintf("go %s: %v\n%s", strings.Join(args, " "), err, b)
				return
			}
		}
		exported.data, exported.metais = data, metais
	})
	if exported.err != "" {
		t.Fatal(exported.err)
	}
	return exported.data, exported.metais
}

// helpDoc is the result of metais help --json, metais.help/1, as the
// reference pages show it. It is decoded with every field known, so a field
// the help gains fails here until the pages show it, or until this type
// says why they do not.
type helpDoc struct {
	Description string `json:"description"`
	Usage       string `json:"usage"`
	Paths       []struct {
		Title string `json:"title"`
		Steps string `json:"steps"`
	} `json:"paths"`
	Commands    []helpCommand `json:"commands"`
	GlobalFlags []helpFlag    `json:"global_flags"`
	Refusals    []helpRefusal `json:"refusals"`
	Topics      []helpTopic   `json:"topics"`
	ExitCodes   []struct {
		Code    int    `json:"code"`
		Meaning string `json:"meaning"`
	} `json:"exit_codes"`
}

type helpCommand struct {
	Name        string `json:"name"`
	Group       string `json:"group"`
	Synopsis    string `json:"synopsis"`
	Description string `json:"description"`
	Detail      string `json:"detail"`
	// OpensPackage and Role classify a command for a harness (ADR-008
	// D13), and metais help prints neither, so no page shows them.
	OpensPackage bool   `json:"opens_package"`
	Role         string `json:"role"`
	Operands     []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Required    bool   `json:"required"`
		Repeatable  bool   `json:"repeatable"`
	} `json:"operands"`
	Flags    []helpFlag    `json:"flags"`
	Format   *string       `json:"format"`
	Prints   string        `json:"prints"`
	Refusals []helpRefusal `json:"refusals"`
	Examples []struct {
		Run string `json:"run"`
		Why string `json:"why"`
	} `json:"examples"`
}

type helpFlag struct {
	Name        string   `json:"name"`
	Short       *string  `json:"short"`
	Value       *string  `json:"value"`
	Values      []string `json:"values"`
	Default     *string  `json:"default"`
	Repeatable  bool     `json:"repeatable"`
	Required    bool     `json:"required"`
	Description string   `json:"description"`
}

type helpRefusal struct {
	Code  string `json:"code"`
	Cause string `json:"cause"`
	Next  *struct {
		Run string `json:"run"`
		Why string `json:"why"`
	} `json:"next"`
}

type helpTopic struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// readJSON decodes the file at p into v, refusing a field v does not know.
func readJSON(t *testing.T, p string, v any) {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		t.Fatalf("%s: %v", p, err)
	}
}

var (
	mainElement = regexp.MustCompile(`(?s)<main>(.*)</main>`)
	anyTag      = regexp.MustCompile(`<[^>]*>`)
)

// mainText is the text of a page's main element as a reader sees it: its
// tags dropped, its character references decoded, and every run of
// whitespace one space.
func mainText(b []byte) string {
	m := mainElement.FindSubmatch(b)
	if m == nil {
		return ""
	}
	return words(html.UnescapeString(anyTag.ReplaceAllString(string(m[1]), " ")))
}

// words is parts joined, with every run of whitespace one space.
func words(parts ...string) string {
	return strings.Join(strings.Fields(strings.Join(parts, " ")), " ")
}

// The words the reference pages' layout adds to what the help says: the
// headings of a command's sections, and the labels of a flag's values,
// default and multiplicity, as the text help prints them.
const (
	labelRequired   = "Required."
	labelRepeatable = "May be given more than once."
)

// flagWords is what a page says of each of flags.
func flagWords(flags []helpFlag) []string {
	var out []string
	for _, f := range flags {
		if f.Short != nil {
			out = append(out, *f.Short+",")
		}
		out = append(out, f.Name)
		if f.Value != nil {
			out = append(out, *f.Value)
		}
		out = append(out, f.Description)
		if len(f.Values) > 0 {
			out = append(out, "Values:", strings.Join(f.Values, " , "), ".")
		}
		if f.Default != nil {
			out = append(out, "Default:", *f.Default, ".")
		}
		if f.Required {
			out = append(out, labelRequired)
		}
		if f.Repeatable {
			out = append(out, labelRepeatable)
		}
	}
	return out
}

// refusalWords is what a page says of each of refusals.
func refusalWords(refusals []helpRefusal) []string {
	var out []string
	for _, r := range refusals {
		out = append(out, r.Code, r.Cause)
		if r.Next != nil {
			out = append(out, "Next:", r.Next.Run, ":", r.Next.Why)
		}
	}
	return out
}

// commandText is the whole text of c's page: what metais help <command>
// prints, in its order, with the layout's headings and labels.
func commandText(c helpCommand) string {
	w := []string{"metais " + c.Name, c.Description, c.Group, "Usage", c.Synopsis}
	if len(c.Examples) > 0 {
		w = append(w, "Examples")
		for _, e := range c.Examples {
			w = append(w, e.Run, e.Why)
		}
	}
	if c.Detail != "" {
		w = append(w, "Detail", c.Detail)
	}
	if len(c.Operands) > 0 {
		w = append(w, "Operands")
		for _, o := range c.Operands {
			w = append(w, o.Name, o.Description)
			if o.Required {
				w = append(w, labelRequired)
			}
			if o.Repeatable {
				w = append(w, labelRepeatable)
			}
		}
	}
	w = append(w, "Flags")
	w = append(w, flagWords(c.Flags)...)
	w = append(w, "Prints", c.Prints)
	if c.Format != nil {
		w = append(w, "--json:", *c.Format)
	}
	w = append(w, "Refusals")
	w = append(w, refusalWords(c.Refusals)...)
	return words(w...)
}

// indexText is the whole text of the reference's overview, apart from the
// version line, which version is the text of.
func indexText(h helpDoc, version string) string {
	w := []string{"metais", h.Description, version, h.Usage, "Golden paths"}
	for _, p := range h.Paths {
		w = append(w, p.Title, p.Steps)
	}
	w = append(w, "Commands")
	var groups []string
	for _, c := range h.Commands {
		if !slices.Contains(groups, c.Group) {
			groups = append(groups, c.Group)
		}
	}
	for _, g := range groups {
		w = append(w, g)
		for _, c := range h.Commands {
			if c.Group == g {
				w = append(w, "metais "+c.Name, c.Description)
			}
		}
	}
	w = append(w, "Global flags")
	w = append(w, flagWords(h.GlobalFlags)...)
	w = append(w, "Topics")
	for _, t := range h.Topics {
		w = append(w, "metais help "+t.Name, t.Title)
	}
	w = append(w, "metais skill --print", "Exit codes")
	for _, e := range h.ExitCodes {
		w = append(w, strconv.Itoa(e.Code), e.Meaning)
	}
	w = append(w, "Refusals")
	w = append(w, refusalWords(h.Refusals)...)
	return words(w...)
}

// versionLine is the overview's line from metais version --json.
func versionLine(t *testing.T, data string) string {
	t.Helper()
	var v struct {
		Tool struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"tool"`
		Specification struct {
			Implements []string `json:"implements"`
		} `json:"specification"`
	}
	readJSON(t, filepath.Join(data, "tool.json"), &v)
	return "metais version : " + v.Tool.Version + ". Specification: " + strings.Join(v.Specification.Implements, ", ") + "."
}

// referencePages is every reference page the help gives, by its path in
// the published tree.
func referencePages(h helpDoc) []string {
	out := []string{"reference/index.html", "reference/skill/index.html"}
	for _, c := range h.Commands {
		out = append(out, "reference/commands/"+c.Name+"/index.html")
	}
	for _, tp := range h.Topics {
		out = append(out, "reference/topics/"+tp.Name+"/index.html")
	}
	slices.Sort(out)
	return out
}

// topicPre matches a topic page's text.
var topicPre = regexp.MustCompile(`(?s)<pre class="topic">(.*?)</pre>`)

// skillArticle matches the skill page's converted body.
var skillArticle = regexp.MustCompile(`(?s)<article class="skill">\n(.*)\n</article>`)

// TestReferencePages holds each reference page of the built site to the
// help it came from (ADR-008 A13 items 2 and 3, ADR-013 D8 item 4): one
// page per command, one per topic and one for the skill, and no other
// under reference/; each command's page and the overview say exactly what
// the help says, in its order, with nothing but the layout's headings and
// labels beside it; a topic's page holds its text byte for byte; and the
// skill's page holds the skill's body as the build step converted it.
func TestReferencePages(t *testing.T) {
	site := tree(t, builtSite(t))
	data, _ := exportedInputs(t)
	var h helpDoc
	readJSON(t, filepath.Join(data, "help.json"), &h)
	if len(h.Commands) == 0 || len(h.Topics) == 0 {
		t.Fatalf("the exported help has %d commands and %d topics; the check is blind", len(h.Commands), len(h.Topics))
	}
	for _, want := range []string{"templates", "manifest"} {
		if !slices.ContainsFunc(h.Topics, func(tp helpTopic) bool { return tp.Name == want }) {
			t.Errorf("the help has no topic %s, which ADR-008 A13 item 1 names", want)
		}
	}

	var got []string
	for p := range site {
		if strings.HasPrefix(p, "reference/") {
			got = append(got, p)
		}
	}
	slices.Sort(got)
	if want := referencePages(h); !slices.Equal(got, want) {
		t.Errorf("the site's reference pages are\n%q\nwant one per command and topic and the skill's:\n%q", got, want)
	}

	check := func(p, want string) {
		t.Helper()
		b, ok := site[p]
		if !ok {
			return
		}
		if got := mainText(b); got != want {
			t.Errorf("%s does not say what the help says, and nothing else:\n got: %s\nwant: %s", p, got, want)
		}
	}
	check("reference/index.html", indexText(h, versionLine(t, data)))
	for _, c := range h.Commands {
		check("reference/commands/"+c.Name+"/index.html", commandText(c))
	}
	for _, tp := range h.Topics {
		p := "reference/topics/" + tp.Name + "/index.html"
		check(p, words("metais help "+tp.Name, tp.Title, tp.Text))
		m := topicPre.FindSubmatch(site[p])
		if m == nil || html.UnescapeString(string(m[1])) != tp.Text {
			t.Errorf("%s does not hold the topic's text exactly as metais help %s prints it", p, tp.Name)
		}
	}

	var sk struct {
		Text string `json:"text"` // the skill whole, which the page does not show: its frontmatter is for a harness
		HTML string `json:"html"`
	}
	readJSON(t, filepath.Join(data, "skill.json"), &sk)
	m := skillArticle.FindSubmatch(site["reference/skill/index.html"])
	if m == nil || string(m[1]) != sk.HTML {
		t.Errorf("reference/skill/index.html does not hold the skill's body as the build step converted it")
	}
	if got, want := mainText(site["reference/skill/index.html"]), words("metais skill --print", mainText([]byte("<main>"+sk.HTML+"</main>"))); got != want {
		t.Errorf("reference/skill/index.html says more than the skill:\n got: %s\nwant: %s", got, want)
	}
}

// TestReferenceHeldToHelp pins the reference pages' check false first: a
// page that says less than the help, more, or something else is refused.
func TestReferenceHeldToHelp(t *testing.T) {
	short := "-h"
	c := helpCommand{
		Name: "version", Group: "The tool", Synopsis: "metais version [--json]", Description: "Print the version.",
		Flags:  []helpFlag{{Name: "--help", Short: &short, Description: "Print help."}},
		Prints: "Lines.",
	}
	page := func(body string) []byte { return []byte("<main>" + body + "</main>") }
	good := `<h1><code>metais version</code></h1><p>Print the version.</p><p>The tool</p><h2>Usage</h2><pre><code>metais version [--json]</code></pre>` +
		`<h2>Flags</h2><dl><dt><code>-h, --help</code></dt><dd>Print help.</dd></dl><h2>Prints</h2><p>Lines.</p><h2>Refusals</h2><dl></dl>`
	if got := mainText(page(good)); got != commandText(c) {
		t.Fatalf("a page that says what the help says is refused:\n got: %s\nwant: %s", got, commandText(c))
	}
	for name, bad := range map[string]string{
		"a flag missing":   strings.Replace(good, `<dt><code>-h, --help</code></dt><dd>Print help.</dd>`, "", 1),
		"words added":      strings.Replace(good, `<p>Lines.</p>`, `<p>Lines. And more.</p>`, 1),
		"words changed":    strings.Replace(good, `Print the version.`, `Prints the version.`, 1),
		"sections swapped": strings.Replace(good, `<h2>Prints</h2><p>Lines.</p><h2>Refusals</h2>`, `<h2>Refusals</h2><h2>Prints</h2><p>Lines.</p>`, 1),
	} {
		if mainText(page(bad)) == commandText(c) {
			t.Errorf("a page with %s passes the check", name)
		}
	}
}

// TestSpecPages requires the built site to serve one page at /spec/draft/
// per Markdown file of spec/, as the exported input says (ADR-011 A8 item
// 2): while the draft is served, each saying it is a draft and asking
// search engines not to index it, and while spec/ is the latest released
// revision's, each leading to that revision's page. While no revision is
// released, /spec/ says so and leads to the draft. The header and the
// footer lead to /spec/, never to the draft.
func TestSpecPages(t *testing.T) {
	site := tree(t, builtSite(t))
	entries, err := os.ReadDir("../spec")
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, e := range entries {
		if name, ok := strings.CutSuffix(e.Name(), ".md"); ok && e.Type().IsRegular() {
			if name == "README" {
				want = append(want, "spec/draft/index.html")
			} else {
				want = append(want, "spec/draft/"+name+"/index.html")
			}
		}
	}
	slices.Sort(want)
	var got []string
	for p := range site {
		if strings.HasPrefix(p, "spec/draft/") {
			got = append(got, p)
		}
	}
	slices.Sort(got)
	if len(want) < 2 || !slices.Equal(got, want) {
		t.Errorf("the draft's pages are %q, want one per file of spec/: %q", got, want)
	}

	data, _ := exportedInputs(t)
	var sp specInput
	readJSON(t, filepath.Join(data, "spec.json"), &sp)
	if sp.DraftServed {
		for _, p := range got {
			checkDraft(t, p, site[p])
		}
	} else {
		latest := sp.Releases[len(sp.Releases)-1]
		for _, f := range sp.Draft.Files {
			checkDraftRedirect(t, site, f, latest)
		}
	}
	if len(sp.Releases) == 0 {
		index := site["spec/index.html"]
		if !bytes.Contains(index, []byte(noindex)) || !bytes.Contains(index, []byte(`<a href="/spec/draft/">`)) {
			t.Errorf("with no revision released, /spec/ does not lead to the draft and ask not to be indexed:\n%s", index)
		}
	}
	checkSpecNav(t, site, sp)
}

// specInput is spec.json as the tests here read it.
type specInput struct {
	Draft        specRevision   `json:"draft"`
	DraftServed  bool           `json:"draft_served"`
	Releases     []specRevision `json:"releases"`
	ReleasePages []specPage     `json:"release_pages"`
}

type specRevision struct {
	Revision string     `json:"revision"`
	Ref      string     `json:"ref"`
	Path     string     `json:"path"`
	Files    []specFile `json:"files"`
}

type specFile struct {
	File     string `json:"file"`
	Path     string `json:"path"`
	Title    string `json:"title"`
	HTML     string `json:"html"`
	Headings []any  `json:"headings"`
}

type specPage struct {
	Revision string `json:"revision"`
	File     string `json:"file"`
	Path     string `json:"path"`
}

// noindex asks search engines not to index a page.
const noindex = `<meta name="robots" content="noindex">`

// draftNotice is what every page of the draft says first.
const draftNotice = `<p class="draft"><strong>Draft.</strong>`

// checkDraft requires the page at p to be a page of the draft.
func checkDraft(t *testing.T, p string, b []byte) {
	t.Helper()
	head, _, _ := bytes.Cut(b, []byte("</head>"))
	if bytes.Count(head, []byte(noindex)) != 1 {
		t.Errorf("%s does not ask search engines not to index it", p)
	}
	if m := mainElement.FindSubmatch(b); m == nil || !bytes.HasPrefix(bytes.TrimSpace(m[1]), []byte(draftNotice)) {
		t.Errorf("%s does not begin by saying it is a draft", p)
	}
}

// checkDraftRedirect requires the page of the draft's file f to lead to
// the same chapter of latest, or to latest's contents when latest has no
// such chapter, as /spec/ leads to latest: a meta refresh and a canonical
// link, asking search engines not to index it, and not a page of the
// draft.
func checkDraftRedirect(t *testing.T, site map[string][]byte, f specFile, latest specRevision) {
	t.Helper()
	p := strings.TrimPrefix(f.Path, "/") + "index.html"
	b, ok := site[p]
	if !ok {
		t.Errorf("the draft's %s is not served at %s", f.File, f.Path)
		return
	}
	target := latest.Path
	for _, lf := range latest.Files {
		if lf.File == f.File {
			target = lf.Path
		}
	}
	head, _, _ := bytes.Cut(b, []byte("</head>"))
	for _, want := range []string{
		`<meta http-equiv="refresh" content="0; url=` + target + `">`,
		`<link rel="canonical" href="https://metais.org` + target + `">`,
		noindex,
	} {
		if bytes.Count(head, []byte(want)) != 1 {
			t.Errorf("%s does not lead to %s with %s:\n%s", p, target, want, head)
		}
	}
	if bytes.Contains(b, []byte(draftNotice)) || bytes.Contains(b, []byte(f.HTML)) {
		t.Errorf("%s is a page of the draft, not a page leading to %s", p, target)
	}
}

// checkSpecNav requires the header and the footer of every page to lead
// to /spec/ and never to the draft, and the landing page to name the latest
// released revision at /spec/ and the draft only while it is served,
// calling no released revision a draft.
func checkSpecNav(t *testing.T, site map[string][]byte, sp specInput) {
	t.Helper()
	for p, b := range site {
		if !strings.HasSuffix(p, ".html") || bytes.Contains(b, []byte(`http-equiv="refresh"`)) {
			continue
		}
		for _, l := range specNav.FindAllSubmatch(b, -1) {
			if string(l[1]) != "/spec/" {
				t.Errorf("%s's %s link leads to %s, not /spec/", p, l[2], l[1])
			}
		}
		if n := len(specNav.FindAll(b, -1)); n != 2 {
			t.Errorf("%s has %d links named Spec or Specification, want the header's and the footer's", p, n)
		}
	}
	index := site["index.html"]
	draftLink := []byte(`<a href="/spec/draft/">`)
	if got := bytes.Contains(index, draftLink); got != sp.DraftServed {
		t.Errorf("the landing page links the draft %v while the draft is served %v", got, sp.DraftServed)
	}
	if len(sp.Releases) > 0 {
		latest := sp.Releases[len(sp.Releases)-1]
		want := `<a href="/spec/"><span class="sec">§</span><span class="ttl">The specification, revision ` + latest.Revision + `</span>`
		if !bytes.Contains(index, []byte(want)) {
			t.Errorf("the landing page does not name the latest revision, %s, at /spec/: want %s", latest.Revision, want)
		}
	}
	if bytes.Contains(index, []byte("1.0 draft")) {
		t.Errorf("the landing page calls a revision a draft")
	}
}

// specNav matches the header's Spec link and the footer's Specification
// link, and captures where each leads and its name.
var specNav = regexp.MustCompile(`<a href="([^"]*)"[^>]*>(Spec|Specification)</a>`)

// renderSpec renders the site package from the exported inputs with the
// specification's input's releases and draft_served replaced, as the
// build step exports them for the tags spec-1.2 and spec-1.10, and returns
// the built site with the fixture's pages in place of every page of the
// specification. The releases' chapters are README.md and a chapter of the
// draft's, 01-introduction.md, so every other chapter of the draft has no
// counterpart in them.
func renderSpec(t *testing.T, draftServed bool) (map[string][]byte, specInput) {
	t.Helper()
	data, metais := exportedInputs(t)
	var sp specInput
	readJSON(t, filepath.Join(data, "spec.json"), &sp)
	sp.Releases, sp.ReleasePages, sp.DraftServed = nil, nil, draftServed
	for _, r := range []string{"1.2", "1.10"} {
		base := "/spec/" + r + "/"
		rev := specRevision{Revision: r, Ref: "spec-" + r, Path: base, Files: []specFile{
			{File: "01-introduction.md", Path: base + "01-introduction/", Title: "1. Intro", HTML: `<h1 id="1-intro">1. Intro</h1><p>Text of ` + r + `. <a href="` + base + `#the-specification">Contents</a>.</p>`, Headings: []any{}},
			{File: "README.md", Path: base, Title: "The specification", HTML: `<h1 id="the-specification">The specification</h1><p><a href="` + base + `01-introduction/">Intro</a></p>`, Headings: []any{}},
		}}
		sp.Releases = append(sp.Releases, rev)
		for _, f := range rev.Files {
			sp.ReleasePages = append(sp.ReleasePages, specPage{Revision: r, File: f.File, Path: f.Path})
		}
	}
	fixture := filepath.Join(testDir(t), "data")
	if err := copyDir(data, fixture); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(sp)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "spec.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(testDir(t), "site")
	cmd := exec.Command(metais, "render", "package", "--data", fixture, "--out", out)
	cmd.Env = []string{}
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("metais render: %v\n%s", err, b)
	}
	// The static files and schemas are the build step's, beside what the
	// package renders: the built site's, with the fixture's pages in place
	// of every page of the specification.
	site := tree(t, builtSite(t))
	for p := range site {
		if strings.HasPrefix(p, "spec/") {
			delete(site, p)
		}
	}
	maps.Copy(site, tree(t, out))
	return site, sp
}

// TestSpecReleases renders the site with two released revisions and a
// draft that differs from the latest, and requires the rule of ADR-011 A8
// item 1: each revision served at /spec/<M>.<m>/, one page per chapter,
// from its own text, indexed and not called a draft; /spec/ leading to the
// latest, 1.10; and, by item 2, the draft still served at /spec/draft/.
func TestSpecReleases(t *testing.T) {
	site, sp := renderSpec(t, true)
	for _, r := range sp.Releases {
		for _, f := range r.Files {
			p := strings.TrimPrefix(f.Path, "/") + "index.html"
			b, ok := site[p]
			if !ok {
				t.Errorf("revision %s's %s is not served at %s", r.Revision, f.File, f.Path)
				continue
			}
			if !bytes.Contains(b, []byte(f.HTML)) {
				t.Errorf("%s does not hold %s's text at %s", p, f.File, r.Ref)
			}
			if bytes.Contains(b, []byte(noindex)) || bytes.Contains(b, []byte(draftNotice)) {
				t.Errorf("%s, a released revision's page, is called a draft or asks not to be indexed", p)
			}
			if !bytes.Contains(b, []byte(`released at the tag <code>`+r.Ref+`</code>`)) {
				t.Errorf("%s does not say which revision it is", p)
			}
		}
	}
	index := site["spec/index.html"]
	for _, want := range []string{`<meta http-equiv="refresh" content="0; url=/spec/1.10/">`, `<link rel="canonical" href="https://metais.org/spec/1.10/">`} {
		if !bytes.Contains(index, []byte(want)) {
			t.Errorf("/spec/ does not lead to the latest revision, 1.10, with %s:\n%s", want, index)
		}
	}
	if bytes.Contains(index, []byte(noindex)) {
		t.Errorf("/spec/ asks not to be indexed once a revision is released")
	}
	if len(sp.Draft.Files) < 2 {
		t.Fatalf("the draft has %d files", len(sp.Draft.Files))
	}
	for _, f := range sp.Draft.Files {
		p := strings.TrimPrefix(f.Path, "/") + "index.html"
		if b, ok := site[p]; !ok {
			t.Errorf("the draft's %s is not served beside the releases", f.File)
		} else {
			checkDraft(t, p, b)
		}
	}
	checkSpecNav(t, site, sp)
	checkLinks(t, "the site with two releases and a draft", site)
}

// TestSpecDraftRedirects renders the site with two released revisions and
// a draft identical to the latest, and requires the rule of ADR-011 A8 item
// 2: no page of the draft is served, and each of its URLs leads to the
// same chapter of the latest revision, 1.10, or to 1.10's contents when
// 1.10 has no such chapter, so a link to the draft keeps working; the
// landing page names 1.10 at /spec/ and no draft.
func TestSpecDraftRedirects(t *testing.T) {
	site, sp := renderSpec(t, false)
	latest := sp.Releases[len(sp.Releases)-1]
	counterparts := 0
	for _, f := range sp.Draft.Files {
		if f.File == "README.md" || f.File == "01-introduction.md" {
			counterparts++
		}
		checkDraftRedirect(t, site, f, latest)
	}
	if counterparts != 2 || len(sp.Draft.Files) <= counterparts {
		t.Fatalf("the draft's files are %d, %d of them in 1.10; want both kinds of page", len(sp.Draft.Files), counterparts)
	}
	checkSpecNav(t, site, sp)
	checkLinks(t, "the site with two releases and no draft", site)
}

// testDir is a new directory beneath the shared one, for one test.
func testDir(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp(sharedDir, "test-")
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// copyDir copies the regular files of src, a flat directory, into dst,
// which it makes.
func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.Mkdir(dst, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

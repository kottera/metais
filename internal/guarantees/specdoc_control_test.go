// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// The controls of the document tripwires. Each builds a small
// specification that every tripwire passes, breaks it in one way per
// failure condition of the tripwire's row of ADR-011 D14's table, and
// requires the tripwire to report that break. A tripwire gone blind
// fails here, not on the day a chapter breaks it.

// fixtureBCP14 is the boilerplate as a chapter wraps it.
const fixtureBCP14 = `The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT",
"SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and
"OPTIONAL" in this document are to be interpreted as described in
BCP 14 [RFC2119] [RFC8174] when, and only when, they appear in all
capitals, as shown here.`

var fixtureFiles = map[string]string{
	"README.md": "```\nVersion: 1.1\nStatus: Draft\nLatest release: 1.0, tag spec-1.0\n```\n\n# The fixture specification\n\nIt is read as the [outputs chapter](09-outputs.md) says.\n",

	"CHANGELOG.md": "# Changelog\n\n## 1.1 — unreleased\n\n### Added\n\n- MTS-OUT-003 — the compressor's freedom.\n\n### Editorial\n\n- Chapter 9 reworded.\n\n## 1.0 — 2027-01-15\n\nFirst revision.\n",

	"ERRATA.md": "# Errata\n\n## E-1.0-001 — a rule nobody could obey\n- Kind: impossibility\n- Affects: MTS-OUT-005 (1.0)\n- Files: none\n",

	"01-introduction.md": "# 1. Introduction\n\n**Normative.** Areas: GEN.\n\nA summary.\n\n## 1.3 Normative language\n\n" + fixtureBCP14 + "\n\n" +
		"## 1.8 Area registry\n\n| Area | Chapter | Subject |\n|---|---|---|\n" +
		"| `GEN` | [1. Introduction](01-introduction.md) | general |\n" +
		"| `OUT` | [9. Outputs](09-outputs.md) | outputs |\n" +
		"| `FND` | [12. Findings](12-findings.md) | findings |\n",

	"09-outputs.md": `# 9. Outputs

**Normative.** Areas: OUT.

A summary of the chapter.

## 9.1 Paths

<a id="mts-out-001"></a>**MTS-OUT-001** An output's path MUST be
relative.

- <a id="mts-out-002"></a>**MTS-OUT-002** *(not observable)* An author SHOULD keep a
  path short.
  - nested detail.
- <a id="mts-out-003"></a>**MTS-OUT-003** *(since 1.1)* A processor MAY compress.
- a plain item.

| Id | Name | Rule |
|---|---|---|
| <a id="mts-out-004"></a>MTS-OUT-004 | ` + "`slug`" + ` | the result MUST be lower case |
| none | ` + "`plain`" + ` | unchanged |

## 9.2 Withdrawn and unobservable

<a id="mts-out-005"></a>**MTS-OUT-005** *(withdrawn by E-1.0-001)* A path MUST be short.

<a id="mts-out-006"></a>**MTS-OUT-006** *(not observable; amended by E-1.0-002)* A processor MUST
NOT open a network connection.

> **Note.** See [MTS-OUT-001](#mts-out-001) and [the slug rule](09-outputs.md#mts-out-004).

<!-- example: out/relative-path -->

` + "```yaml path=package/metais.yaml\nname: x\n```\n\n```text path=expected/out/a.txt\nA\n```\n\nThe end.\n",

	"12-findings.md": "# 12. Findings\n\n**Normative.** Areas: FND.\n\nA summary.\n\n## 12.1 The code catalog\n\n" + catalogBegin + "\n" + catalogEnd + "\n",

	"A-references.md": "# Appendix A. References\n\n**Normative.** No statement area.\n\nA list.\n",

	registryFile: `{
  "codes": [
    {"code": "internal.bug", "severity": "error", "class": "internal", "subject": "invocation", "params": [], "since": "1.0", "statements": []},
    {"code": "limit.invalid", "severity": "error", "class": "usage", "subject": "invocation", "params": [{"name": "reason", "always": true}, {"name": "name", "always": false}], "since": "1.0", "statements": []},
    {"code": "output.old", "severity": "error", "class": "refused", "subject": "output", "params": [], "since": "1.0", "retired": "1.1", "statements": []},
    {"code": "output.path_absolute", "severity": "error", "class": "refused", "subject": "output", "params": [], "since": "1.0", "statements": ["MTS-OUT-001", "MTS-OUT-004"]},
    {"code": "report.truncated", "severity": "info", "class": "content", "subject": "report", "params": [{"name": "listed", "always": true}], "since": "1.0", "statements": []}
  ]
}
`,

	"schemas/1.0/x.schema.json": `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://metais.org/schemas/1.0/x.schema.json",
  "type": "object",
  "properties": {
    "a": { "$ref": "#/$defs/a" },
    "b": { "$ref": "x.schema.json#/$defs/a" },
    "$ref": { "const": { "$ref": "https://example.com/data" } }
  },
  "$defs": {
    "a": { "enum": ["x", "y"] }
  }
}
`,

	"conformance/vectors/out/relative-path/vector.json": `{
  "$schema": "https://metais.org/schemas/conformance/1/vector.schema.json",
  "id": "out/relative-path",
  "title": "a path is relative",
  "statements": ["MTS-OUT-001", "MTS-OUT-004"],
  "since": "1.0",
  "class": "processor",
  "tags": [],
  "requires": [],
  "notes": "",
  "cases": [
    {
      "name": "render",
      "setup": { "copy": null, "dirs": [], "symlinks": [] },
      "steps": [
        { "op": "open", "args": { "package": "package" }, "expect": { "ok": true } },
        { "op": "render", "args": { "package": "package", "out": "out" }, "expect": { "ok": false, "findings": "expected/findings.json" } }
      ]
    }
  ]
}
`,
	"conformance/vectors/out/relative-path/expected/findings.json": `[{"code": "output.path_absolute", "severity": "error"}]` + "\n",
	"conformance/vectors/out/relative-path/expected/out/a.txt":     "A\n",
	"conformance/vectors/out/relative-path/package/metais.yaml":    "name: x\n",
	"conformance/vectors/out/relative-path/package/README.md":      "A package file the document never reads: it MUST be ignored.\n",
}

// specFixture is a specification every tripwire passes, in temporary
// directories.
type specFixture struct {
	root, embedded, unwired string
}

func writeFile(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newSpecFixture(t *testing.T) *specFixture {
	t.Helper()
	f := &specFixture{root: testenv.Dir(t), embedded: testenv.Dir(t)}
	f.unwired = filepath.Join(testenv.Dir(t), "unwired.txt")
	for name, body := range fixtureFiles {
		f.write(t, name, body)
	}
	writeFile(t, filepath.Join(f.embedded, "1.0", "x.schema.json"), fixtureFiles["schemas/1.0/x.schema.json"])
	writeFile(t, f.unwired, "# A comment.\n\nreport.truncated W4\n")
	f.regenerate(t)
	f.write(t, releasedDir+"/1.0.sha256",
		sha256Hex([]byte(fixtureFiles["conformance/vectors/out/relative-path/vector.json"]))+"  conformance/vectors/out/relative-path/vector.json\n"+
			sha256Hex([]byte(fixtureFiles["schemas/1.0/x.schema.json"]))+"  schemas/1.0/x.schema.json\n")
	return f
}

// regenerate writes the fixture's generated files, as go generate does.
func (f *specFixture) regenerate(t *testing.T) {
	t.Helper()
	g, err := generated(f.root)
	if err != nil {
		t.Fatal(err)
	}
	for name, b := range g {
		f.write(t, name, string(b))
	}
}

func (f *specFixture) path(name string) string {
	return filepath.Join(f.root, filepath.FromSlash(name))
}

func (f *specFixture) write(t *testing.T, name, body string) {
	t.Helper()
	writeFile(t, f.path(name), body)
}

func (f *specFixture) read(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(f.path(name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// edit replaces the one occurrence of old in the file name with new. A
// control whose old text has gone is stale, and fails.
func (f *specFixture) edit(t *testing.T, name, old, new string) {
	t.Helper()
	editFile(t, f.path(name), old, new)
}

func editFile(t *testing.T, p, old, new string) {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(b), old); n != 1 {
		t.Fatalf("%s holds %q %d times, not once: the control is stale", p, old, n)
	}
	writeFile(t, p, strings.Replace(string(b), old, new, 1))
}

func (f *specFixture) docs(t *testing.T) []*mdDoc { return mustDocs(t, f.root) }

func (f *specFixture) statements(t *testing.T) []statement {
	t.Helper()
	stmts, _ := specStatements(f.docs(t))
	return stmts
}

// tripwireCase is one break of the fixture, and what the tripwire must say
// about it. An empty want is a variation the tripwire must accept.
type tripwireCase struct {
	name   string
	break_ func(t *testing.T, f *specFixture)
	want   []string
}

// runControl requires check to pass the fixture, then to report each
// case's break with every substring the case wants.
func runControl(t *testing.T, check func(t *testing.T, f *specFixture) []string, cases []tripwireCase) {
	t.Helper()
	if p := check(t, newSpecFixture(t)); len(p) > 0 {
		t.Fatalf("the fixture does not pass: %q", p)
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newSpecFixture(t)
			c.break_(t, f)
			got := check(t, f)
			if len(c.want) == 0 {
				if len(got) > 0 {
					t.Errorf("reports %q, want nothing", got)
				}
				return
			}
			if len(got) == 0 {
				t.Fatalf("reports nothing, want %q", c.want)
			}
			for _, w := range c.want {
				if !slices.ContainsFunc(got, func(p string) bool { return strings.Contains(p, w) }) {
					t.Errorf("reports %q, want one saying %q", got, w)
				}
			}
		})
	}
}

func edit(name, old, new string) func(*testing.T, *specFixture) {
	return func(t *testing.T, f *specFixture) { f.edit(t, name, old, new) }
}

func appendTo(name, text string) func(*testing.T, *specFixture) {
	return func(t *testing.T, f *specFixture) { f.write(t, name, f.read(t, name)+text) }
}

func TestStatementsWellFormedControl(t *testing.T) {
	runControl(t, func(t *testing.T, f *specFixture) []string { return wellFormedProblems(f.docs(t)) }, []tripwireCase{
		{"no space after the head", edit("09-outputs.md", "**MTS-OUT-001** An", "**MTS-OUT-001**An"), []string{"does not match ADR-011 D3 item 1's grammar"}},
		{"an anchor without its closing tag", edit("09-outputs.md", `<a id="mts-out-001"></a>`, `<a id="mts-out-001">`), []string{"grammar"}},
		{"bold without the anchor", edit("09-outputs.md", `<a id="mts-out-001"></a>**MTS`, `**MTS`), []string{"grammar"}},
		{"an attribute D3 does not define", edit("09-outputs.md", "*(since 1.1)*", "*(since one)*"), []string{`the attribute "since one"`}},
		{"an empty attribute list", edit("09-outputs.md", "*(since 1.1)*", "*()*"), []string{"which ADR-011 D3 item 1 does not define"}},
		{"an attribute given twice", edit("09-outputs.md", "*(since 1.1)*", "*(since 1.1; since 1.2)*"), []string{`the attribute "since" twice`}},
		{"attributes not separated by a semicolon and a space", edit("09-outputs.md", "*(not observable; amended", "*(not observable;amended"), []string{"does not define"}},
		{"an identifier not bold", edit("09-outputs.md", "**MTS-OUT-001** An", "MTS-OUT-001 An"), []string{"is not bold"}},
		{"an identifier bold in a table cell", edit("09-outputs.md", "</a>MTS-OUT-004 |", "</a>**MTS-OUT-004** |"), []string{"is bold inside a table cell"}},
		{"a bare identifier with no anchor in a table cell", edit("09-outputs.md", `| <a id="mts-out-004"></a>MTS-OUT-004 |`, `| MTS-OUT-004 |`), []string{"grammar"}},
		{"a number of two digits", edit("09-outputs.md", `<a id="mts-out-001"></a>**MTS-OUT-001**`, `<a id="mts-out-01"></a>**MTS-OUT-01**`), []string{"at least three digits"}},
		{"a number padded past three digits", edit("09-outputs.md", `<a id="mts-out-001"></a>**MTS-OUT-001**`, `<a id="mts-out-0001"></a>**MTS-OUT-0001**`), []string{"zero-padded only to three"}},
		{"an anchor in upper case", edit("09-outputs.md", `<a id="mts-out-001">`, `<a id="MTS-OUT-001">`), []string{`has the anchor "MTS-OUT-001"`}},
		{"an anchor naming another statement", edit("09-outputs.md", `<a id="mts-out-001">`, `<a id="mts-out-009">`), []string{`has the anchor "mts-out-009"`}},
		{"an identifier used twice", edit("09-outputs.md", `<a id="mts-out-004"></a>MTS-OUT-004`, `<a id="mts-out-001"></a>MTS-OUT-001`), []string{"MTS-OUT-001 is already used at 09-outputs.md:9"}},
		{"an identifier used in two chapters", appendTo("12-findings.md", "\n<a id=\"mts-out-001\"></a>**MTS-OUT-001** A path MUST be relative.\n"), []string{"already used", "which 09-outputs.md owns"}},
		{"an area not in the registry", appendTo("09-outputs.md", "\n<a id=\"mts-zzz-001\"></a>**MTS-ZZZ-001** A thing MUST hold.\n"), []string{"the area ZZZ, which is not in the registry"}},
		{"an area another chapter owns", appendTo("12-findings.md", "\n<a id=\"mts-out-007\"></a>**MTS-OUT-007** A thing MUST hold.\n"), []string{"in the area OUT, which 09-outputs.md owns"}},
		{"a statement in a document that owns no area", appendTo("README.md", "\n<a id=\"mts-gen-001\"></a>**MTS-GEN-001** A thing MUST hold.\n"), []string{"which 01-introduction.md owns"}},
		{"an anchor inside a paragraph", appendTo("09-outputs.md", "\nSee <a id=\"mts-out-008\"></a>**MTS-OUT-008** here.\n"), []string{"09-outputs.md:44: a statement anchor that does not begin"}},
		{"a second anchor on a head's line", edit("09-outputs.md", "A processor MAY compress.", `A processor MAY <a id="mts-out-009"></a>compress.`), []string{"does not begin a paragraph"}},
		{"an anchor in a block quote", edit("09-outputs.md", "> **Note.** See", `> <a id="mts-out-010"></a>**MTS-OUT-010** See`), []string{"does not begin"}},
		{"a statement with no key word", edit("09-outputs.md", "MUST be\nrelative", "is\nrelative"), []string{"MTS-OUT-001 holds no key word"}},
		{"no area registry", edit("01-introduction.md", "| `GEN` | [1. Introduction](01-introduction.md) | general |\n| `OUT` | [9. Outputs](09-outputs.md) | outputs |\n| `FND` | [12. Findings](12-findings.md) | findings |\n", ""), []string{"no area registry"}},
		{"an area registered twice", appendTo("01-introduction.md", "| `OUT` | [12. Findings](12-findings.md) | again |\n"), []string{"the area OUT is registered twice"}},
		{"a status line naming another area", edit("09-outputs.md", "Areas: OUT.", "Areas: OUT, FND."), []string{`the status line names the areas ["FND" "OUT"]`}},
		{"a chapter with no status line", edit("09-outputs.md", "**Normative.** Areas: OUT.", "Normative."), []string{"09-outputs.md: no status line"}},
		{"a head inside a fence is no statement", appendTo("09-outputs.md", "\n```\n<a id=\"mts-out-001\"></a>**MTS-OUT-001** x\n```\n"), nil},
		{"a numbered list item may open one", appendTo("09-outputs.md", "\n1. <a id=\"mts-out-011\"></a>**MTS-OUT-011** A thing MUST hold.\n"), nil},
	})
}

func TestStatementIndexControl(t *testing.T) {
	check := func(t *testing.T, f *specFixture) []string { return indexProblems(f.root) }
	cases := []tripwireCase{
		{"a statement added and not indexed", appendTo("09-outputs.md", "\n<a id=\"mts-out-012\"></a>**MTS-OUT-012** A thing MUST hold.\n"), []string{"spec/statements.json differs"}},
		{"a statement's attribute changed", edit("09-outputs.md", "*(since 1.1)*", "*(since 1.2)*"), []string{"differs"}},
		{"a statement moved to another section", edit("09-outputs.md", "## 9.2 Withdrawn", "## 9.3 Withdrawn"), []string{"differs"}},
		{"the index removed", func(t *testing.T, f *specFixture) { os.Remove(f.path(statementsFile)) }, []string{"no such file"}},
		{"a wording change is no index change", edit("09-outputs.md", "A processor MAY compress.", "A processor MAY compress a file."), nil},
	}
	// A one-byte difference anywhere in the index.
	idx := statementIndex(newSpecFixture(t).statements(t))
	for _, at := range []int{0, len(idx) / 3, len(idx) / 2, len(idx) - 1} {
		cases = append(cases, tripwireCase{fmt.Sprintf("byte %d changed", at), func(t *testing.T, f *specFixture) {
			b := []byte(f.read(t, statementsFile))
			b[at] ^= 1
			f.write(t, statementsFile, string(b))
		}, []string{"differs"}})
	}
	cases = append(cases, tripwireCase{"a byte added", appendTo(statementsFile, "\n"), []string{"differs"}})
	runControl(t, check, cases)
}

// TestStatementIndexForm pins the index's form: one statement a line, in
// identifier order by number, with every member ADR-011 D3 item 7 names.
func TestStatementIndexForm(t *testing.T) {
	f := newSpecFixture(t)
	f.write(t, "09-outputs.md", f.read(t, "09-outputs.md")+"\n<a id=\"mts-out-1000\"></a>**MTS-OUT-1000** A thing MUST hold.\n")
	want := `{
  "statements": [
    {"id": "MTS-OUT-001", "chapter": "09-outputs.md", "section": "9.1", "kind": "requirement", "since": "1.0", "observable": true, "status": "active"},
    {"id": "MTS-OUT-002", "chapter": "09-outputs.md", "section": "9.1", "kind": "recommendation", "since": "1.0", "observable": false, "status": "active"},
    {"id": "MTS-OUT-003", "chapter": "09-outputs.md", "section": "9.1", "kind": "permission", "since": "1.1", "observable": true, "status": "active"},
    {"id": "MTS-OUT-004", "chapter": "09-outputs.md", "section": "9.1", "kind": "requirement", "since": "1.0", "observable": true, "status": "active"},
    {"id": "MTS-OUT-005", "chapter": "09-outputs.md", "section": "9.2", "kind": "requirement", "since": "1.0", "observable": true, "status": "withdrawn"},
    {"id": "MTS-OUT-006", "chapter": "09-outputs.md", "section": "9.2", "kind": "requirement", "since": "1.0", "observable": false, "status": "active", "amended_by": ["E-1.0-002"]},
    {"id": "MTS-OUT-1000", "chapter": "09-outputs.md", "section": "9.2", "kind": "requirement", "since": "1.0", "observable": true, "status": "active"}
  ]
}
`
	if got := string(statementIndex(f.statements(t))); got != want {
		t.Errorf("the index is\n%s\nwant\n%s", got, want)
	}
	if got := string(statementIndex(nil)); got != "{\n  \"statements\": []\n}\n" {
		t.Errorf("the empty index is %q", got)
	}
}

func TestKeywordsOnlyInStatementsControl(t *testing.T) {
	runControl(t, func(t *testing.T, f *specFixture) []string { return keywordProblems(f.docs(t)) }, []tripwireCase{
		{"a key word in a paragraph", appendTo("09-outputs.md", "\nAn author MUST read this.\n"), []string{`09-outputs.md:44: the key word "MUST" stands outside a statement`}},
		{"a key word in a note", edit("09-outputs.md", "> **Note.** See", "> **Note.** You MAY see"), []string{`"MAY" stands outside`}},
		{"a key word in a fence", appendTo("09-outputs.md", "\n```\nit MAY\n```\n"), []string{`"MAY" stands outside`}},
		{"a key word in an example block", edit("09-outputs.md", "```text path=expected/out/a.txt\nA\n", "```text path=expected/out/a.txt\nA MUST\n"), []string{`"MUST" stands outside`}},
		{"a key word on the front page", appendTo("README.md", "\nA reader SHOULD start here.\n"), []string{`README.md:11: the key word "SHOULD"`}},
		{"a key word in the changelog", appendTo("CHANGELOG.md", "\n- a tool MUST NOT change.\n"), []string{`"MUST NOT" stands outside`}},
		{"a key word in a table row that opens no statement", edit("09-outputs.md", "| unchanged |", "| MUST be unchanged |"), []string{`"MUST" stands outside`}},
		{"a key word in the list item after a statement's", edit("09-outputs.md", "- a plain item.", "- a plain item MUST hold."), []string{`"MUST" stands outside`}},
		{"a key word in a heading", edit("09-outputs.md", "## 9.1 Paths", "## 9.1 Paths MUST be relative"), []string{`"MUST" stands outside`}},
		{"a synonym in a statement", edit("09-outputs.md", "path MUST be\nrelative", "path SHALL be\nrelative"), []string{`"SHALL" is a synonym ADR-011 D2 item 2 excludes`}},
		{"a synonym outside a statement", appendTo("README.md", "\nA status block is REQUIRED.\n"), []string{`"REQUIRED" is a synonym`}},
		{"each synonym", appendTo("09-outputs.md", "\n<a id=\"mts-out-013\"></a>**MTS-OUT-013** It MUST: SHALL NOT, NOT RECOMMENDED, OPTIONAL.\n"), []string{`"SHALL NOT" is a synonym`, `"NOT RECOMMENDED" is a synonym`, `"OPTIONAL" is a synonym`}},
		{"the boilerplate reworded", edit("01-introduction.md", "as shown here.", "as shown below."), []string{"holds RFC 8174's boilerplate 0 times", `"REQUIRED" is a synonym`, `"MUST" stands outside`}},
		{"the boilerplate twice", appendTo("01-introduction.md", "\n"+fixtureBCP14+"\n"), []string{"holds RFC 8174's boilerplate 2 times"}},
		{"the boilerplate in another chapter", appendTo("09-outputs.md", "\n"+fixtureBCP14+"\n"), []string{"09-outputs.md: holds RFC 8174's boilerplate, which only 01-introduction.md carries"}},
		{"a key word in a lower-case word is none", appendTo("09-outputs.md", "\nAuthors must, should and may; MUSTARD and MAYBE are words.\n"), nil},
		{"a key word in a statement's nested list", edit("09-outputs.md", "  - nested detail.", "  - nested detail MUST hold."), nil},
		{"a key word in a statement's continuation paragraph", edit("09-outputs.md", "  - nested detail.", "  - nested detail.\n\n  It MAY continue."), nil},
		{"the vector suite's Markdown is not the document", appendTo("conformance/vectors/out/relative-path/package/README.md", "It SHALL be.\n"), nil},
		{"a key word in the conformance README", func(t *testing.T, f *specFixture) {
			f.write(t, "conformance/README.md", "# The suite\n\nA driver MUST answer.\n")
		}, []string{`conformance/README.md:3: the key word "MUST"`}},
	})
}

func TestRecommendationsNotObservableControl(t *testing.T) {
	runControl(t, func(t *testing.T, f *specFixture) []string { return recommendationProblems(f.statements(t)) }, []tripwireCase{
		{"a SHOULD statement observable", edit("09-outputs.md", "**MTS-OUT-002** *(not observable)* ", "**MTS-OUT-002** "), []string{"MTS-OUT-002 says SHOULD and is not marked not observable"}},
		{"a SHOULD NOT statement observable", appendTo("09-outputs.md", "\n<a id=\"mts-out-014\"></a>**MTS-OUT-014** An author SHOULD NOT nest.\n"), []string{"MTS-OUT-014 says SHOULD"}},
		{"a SHOULD in a requirement's nested list", edit("09-outputs.md", "  - nested detail.", "  - nested detail.\n- <a id=\"mts-out-015\"></a>**MTS-OUT-015** A tool MUST warn.\n  - an author SHOULD listen."), []string{"MTS-OUT-015 says SHOULD"}},
		{"a MAY statement may be observable", appendTo("09-outputs.md", "\n<a id=\"mts-out-016\"></a>**MTS-OUT-016** A processor MAY stop.\n"), nil},
	})
}

func TestCoverageControl(t *testing.T) {
	check := func(t *testing.T, f *specFixture) []string {
		p, _ := coverageProblems(f.statements(t), mustVectors(t, f.root))
		return p
	}
	runControl(t, check, []tripwireCase{
		{"a requirement no vector cites", edit(vectorsDir+"/out/relative-path/vector.json", `["MTS-OUT-001", "MTS-OUT-004"]`, `["MTS-OUT-001"]`), []string{"MTS-OUT-004 is an observable requirement, and no vector cites it"}},
		{"a requirement in a table row", edit(vectorsDir+"/out/relative-path/vector.json", `["MTS-OUT-001", "MTS-OUT-004"]`, `["MTS-OUT-004"]`), []string{"MTS-OUT-001 is an observable requirement"}},
		{"a not-observable requirement made observable", edit("09-outputs.md", "*(not observable; amended by E-1.0-002)*", "*(amended by E-1.0-002)*"), []string{"MTS-OUT-006 is an observable requirement"}},
		{"a withdrawn requirement restored", edit("09-outputs.md", "*(withdrawn by E-1.0-001)* ", ""), []string{"MTS-OUT-005 is an observable requirement"}},
		{"a vector in another area counts", func(t *testing.T, f *specFixture) {
			f.write(t, vectorsDir+"/gen/worked/vector.json", `{"id": "gen/worked", "statements": ["MTS-OUT-004"], "cases": []}`)
			f.edit(t, vectorsDir+"/out/relative-path/vector.json", `["MTS-OUT-001", "MTS-OUT-004"]`, `["MTS-OUT-001"]`)
		}, nil},
	})
}

func TestCoverageTable(t *testing.T) {
	f := newSpecFixture(t)
	_, citing := coverageProblems(f.statements(t), mustVectors(t, f.root))
	if got := citing["MTS-OUT-004"]; !slices.Equal(got, []string{"out/relative-path"}) {
		t.Errorf("MTS-OUT-004 is cited by %q", got)
	}
}

func TestCitationsResolveControl(t *testing.T) {
	check := func(t *testing.T, f *specFixture) []string {
		docs := f.docs(t)
		stmts, _ := specStatements(docs)
		return citationProblems(docs, stmts, mustVectors(t, f.root), mustRegistry(t, f.root))
	}
	vj := vectorsDir + "/out/relative-path/vector.json"
	runControl(t, check, []tripwireCase{
		{"a vector citing an unknown statement", edit(vj, `"MTS-OUT-004"]`, `"MTS-OUT-004", "MTS-OUT-099"]`), []string{"vector out/relative-path cites MTS-OUT-099, which no chapter states"}},
		{"a vector citing a withdrawn statement", edit(vj, `"MTS-OUT-004"]`, `"MTS-OUT-004", "MTS-OUT-005"]`), []string{"cites MTS-OUT-005, which E-1.0-001 withdrew"}},
		{"a registry entry naming an unknown statement", edit(registryFile, `["MTS-OUT-001", "MTS-OUT-004"]`, `["MTS-OUT-097"]`), []string{"the registry's output.path_absolute names MTS-OUT-097"}},
		{"a link to an unknown anchor", edit("09-outputs.md", "(09-outputs.md#mts-out-004)", "(09-outputs.md#mts-out-098)"), []string{"the link [the slug rule](09-outputs.md#mts-out-098) does not resolve to a statement"}},
		{"a link to another chapter's anchor", edit("09-outputs.md", "(09-outputs.md#mts-out-004)", "(12-findings.md#mts-out-004)"), []string{"does not resolve"}},
		{"a link to a missing chapter", edit("09-outputs.md", "(09-outputs.md#mts-out-004)", "(99-none.md#mts-out-004)"), []string{"does not resolve"}},
		{"a link whose text names another statement", edit("09-outputs.md", "[MTS-OUT-001](#mts-out-001)", "[MTS-OUT-001](#mts-out-004)"), []string{"names MTS-OUT-001 and points at #mts-out-004"}},
		{"a link to an identifier with no anchor", edit("09-outputs.md", "[MTS-OUT-001](#mts-out-001)", "[MTS-OUT-001](09-outputs.md)"), []string{"names MTS-OUT-001 and points at #"}},
		{"an anchor in upper case", edit("09-outputs.md", "(09-outputs.md#mts-out-004)", "(09-outputs.md#MTS-OUT-004)"), []string{"does not resolve"}},
		{"a link out of the specification", edit("09-outputs.md", "(09-outputs.md#mts-out-004)", "(https://example.com/09-outputs.md#mts-out-004)"), []string{"leaves the specification"}},
		{"a link from a subdirectory", func(t *testing.T, f *specFixture) {
			f.write(t, "conformance/README.md", "See [MTS-OUT-001](../09-outputs.md#mts-out-001).\n")
		}, nil},
		{"a broken link from a subdirectory", func(t *testing.T, f *specFixture) {
			f.write(t, "conformance/README.md", "See [MTS-OUT-001](09-outputs.md#mts-out-001).\n")
		}, []string{"conformance/README.md:1: the link"}},
		{"a link in a fence is no link", appendTo("09-outputs.md", "\n```\n[MTS-OUT-999](#mts-out-999)\n```\n"), nil},
	})
}

func TestFindingCatalogControl(t *testing.T) {
	runControl(t, func(t *testing.T, f *specFixture) []string { return catalogProblems(f.root) }, []tripwireCase{
		{"a code added to the registry", edit(registryFile, `    {"code": "report.truncated"`, `    {"code": "report.extra", "severity": "info", "class": "content", "subject": "report", "params": [], "since": "1.0", "statements": []},
    {"code": "report.truncated"`), []string{"spec/12-findings.md differs from what the specification generates"}},
		{"a code's severity changed", edit(registryFile, `"code": "report.truncated", "severity": "info"`, `"code": "report.truncated", "severity": "warning"`), []string{"differs"}},
		{"a code's statements changed", edit(registryFile, `["MTS-OUT-001", "MTS-OUT-004"]`, `["MTS-OUT-001"]`), []string{"differs"}},
		{"a statement a code names moved", func(t *testing.T, f *specFixture) {
			f.edit(t, "09-outputs.md", "<a id=\"mts-out-004\"></a>MTS-OUT-004 | `slug` | the result MUST be lower case |\n", "")
			f.write(t, "12-findings.md", f.read(t, "12-findings.md")+"\n<a id=\"mts-out-004\"></a>**MTS-OUT-004** It MUST.\n")
		}, []string{"differs"}},
		{"a byte of the table changed", edit("12-findings.md", "| `report.truncated` | info |", "| `report.truncated` | inf0 |"), []string{"differs"}},
		{"a row added by hand", edit("12-findings.md", catalogEnd, "| `x.y` | error | refused | output | — | 1.0 | — |\n"+catalogEnd), []string{"differs"}},
		{"the end marker removed", edit("12-findings.md", catalogEnd+"\n", ""), []string{"the catalog markers appear 1 and 0 times"}},
		{"the markers twice", appendTo("12-findings.md", catalogBegin+"\n"+catalogEnd+"\n"), []string{"appear 2 and 2 times"}},
		{"the markers reversed", func(t *testing.T, f *specFixture) {
			f.write(t, "12-findings.md", "# 12. Findings\n\n"+catalogEnd+"\n"+catalogBegin+"\n")
		}, []string{"comes before its beginning"}},
		{"prose around the catalog may change", edit("12-findings.md", "A summary.", "A longer summary."), nil},
	})
}

// TestFindingCatalogForm pins the catalog's rows.
func TestFindingCatalogForm(t *testing.T) {
	f := newSpecFixture(t)
	got := catalogTable(mustRegistry(t, f.root), f.statements(t))
	want := "| Code | Severity | Class | Subject | Parameters | Since | Statements |\n" +
		"|---|---|---|---|---|---|---|\n" +
		"| `internal.bug` | error | internal | invocation | — | 1.0 | — |\n" +
		"| `limit.invalid` | error | usage | invocation | `reason`; optional: `name` | 1.0 | — |\n" +
		"| `output.old` | error | refused | output | — | 1.0, retired 1.1 | — |\n" +
		"| `output.path_absolute` | error | refused | output | — | 1.0 | [MTS-OUT-001](09-outputs.md#mts-out-001), [MTS-OUT-004](09-outputs.md#mts-out-004) |\n" +
		"| `report.truncated` | info | content | report | `listed` | 1.0 | — |\n"
	if got != want {
		t.Errorf("the catalog is\n%s\nwant\n%s", got, want)
	}
}

func TestFindingCodesCoveredControl(t *testing.T) {
	check := func(t *testing.T, f *specFixture) []string { return codesCoveredProblems(f.root, f.unwired) }
	findings := vectorsDir + "/out/relative-path/expected/findings.json"
	unwiredEdit := func(old, new string) func(*testing.T, *specFixture) {
		return func(t *testing.T, f *specFixture) { editFile(t, f.unwired, old, new) }
	}
	runControl(t, check, []tripwireCase{
		{"a listed code a vector produces", edit(findings, `}]`, `}, {"code": "report.truncated"}]`), []string{"unwired.txt:3: report.truncated is produced by out/relative-path; take it off the list"}},
		{"a listed code not registered", unwiredEdit("report.truncated W4\n", "report.truncated W4\nzzz.unknown W5\n"), []string{"unwired.txt:4: zzz.unknown is not registered"}},
		{"a refused code no vector produces, unlisted", edit(findings, `"output.path_absolute"`, `"internal.bug"`), []string{"output.path_absolute is registered, of class refused, and no vector's expected findings hold it"}},
		{"a content code no vector produces, unlisted", unwiredEdit("report.truncated W4\n", ""), []string{"report.truncated is registered, of class content"}},
		{"a destination code no vector produces, unlisted", edit(registryFile, `"class": "content"`, `"class": "destination"`), nil},
		{"a destination code unlisted", func(t *testing.T, f *specFixture) {
			f.edit(t, registryFile, `"class": "content"`, `"class": "destination"`)
			editFile(t, f.unwired, "report.truncated W4\n", "")
		}, []string{"report.truncated is registered, of class destination"}},
		{"a produced code naming no statement", edit(registryFile, `["MTS-OUT-001", "MTS-OUT-004"]`, `[]`), []string{"output.path_absolute names no statement"}},
		{"a listed code need not name a statement", func(*testing.T, *specFixture) {}, nil},
		{"a usage code naming a statement", edit(registryFile, `"class": "usage", "subject": "invocation", "params": [{"name": "reason", "always": true}, {"name": "name", "always": false}], "since": "1.0", "statements": []`, `"class": "usage", "subject": "invocation", "params": [], "since": "1.0", "statements": ["MTS-OUT-001"]`), []string{"limit.invalid is of class usage, and names the statements"}},
		{"an internal code naming a statement", edit(registryFile, `"class": "internal", "subject": "invocation", "params": [], "since": "1.0", "statements": []`, `"class": "internal", "subject": "invocation", "params": [], "since": "1.0", "statements": ["MTS-OUT-001"]`), []string{"internal.bug is of class internal"}},
		{"a usage code listed", unwiredEdit("report.truncated W4\n", "limit.invalid W3\nreport.truncated W4\n"), []string{"unwired.txt:3: limit.invalid is of class usage"}},
		{"a retired code listed", unwiredEdit("report.truncated W4\n", "output.old W5\nreport.truncated W4\n"), []string{"output.old is retired"}},
		{"a line with no wave", unwiredEdit("report.truncated W4\n", "report.truncated\n"), []string{"unwired.txt:3: \"report.truncated\" is not a code and the wave"}},
		{"a wave already past", unwiredEdit("W4", "W2"), []string{"is not a code and the wave, W3 to W7"}},
		{"a wave past the plan's last", unwiredEdit("W4", "W8"), []string{"W3 to W7"}},
		{"codes out of order", unwiredEdit("report.truncated W4\n", "report.truncated W4\noutput.path_absolute W5\n"), []string{"output.path_absolute is out of order or listed twice"}},
		{"a code listed twice", unwiredEdit("report.truncated W4\n", "report.truncated W4\nreport.truncated W5\n"), []string{"listed twice"}},
		{"no unwired list", func(t *testing.T, f *specFixture) { os.Remove(f.unwired) }, []string{"no such file"}},
		{"a findings file outside its vector", edit(vectorsDir+"/out/relative-path/vector.json", `"expected/findings.json"`, `"../findings.json"`), []string{"is not inside the vector"}},
	})
}

func TestExamplesMatchVectorsControl(t *testing.T) {
	check := func(t *testing.T, f *specFixture) []string { return exampleProblems(f.root, f.docs(t)) }
	runControl(t, check, []tripwireCase{
		{"a block that differs from its file", edit("09-outputs.md", "name: x\n```", "name: y\n```"), []string{"09-outputs.md:34: the example's package/metais.yaml differs from the vector out/relative-path's file"}},
		{"a file changed under its block", edit(vectorsDir+"/out/relative-path/package/metais.yaml", "name: x\n", "name: x \n"), []string{"differs"}},
		{"a file with no final newline", edit(vectorsDir+"/out/relative-path/package/metais.yaml", "name: x\n", "name: x"), []string{"differs"}},
		{"a file with a CRLF", edit(vectorsDir+"/out/relative-path/package/metais.yaml", "name: x\n", "name: x\r\n"), []string{"differs"}},
		{"a missing vector", edit("09-outputs.md", "<!-- example: out/relative-path -->", "<!-- example: out/missing -->"), []string{"names the vector out/missing, which does not exist"}},
		{"a missing file", edit("09-outputs.md", "path=package/metais.yaml", "path=package/other.yaml"), []string{"shows package/other.yaml, which the vector out/relative-path does not hold"}},
		{"a path out of the vector", edit("09-outputs.md", "path=package/metais.yaml", "path=../relative-path/package/metais.yaml"), []string{"is not inside the vector"}},
		{"an example with no block", edit("09-outputs.md", "```yaml path=package/metais.yaml", "Text.\n\n```yaml path=package/metais.yaml"), []string{"shows no file of it", "not inside an example"}},
		{"a malformed example comment", edit("09-outputs.md", "<!-- example: out/relative-path -->", "<!-- example: Out/Relative -->"), []string{"is not an example comment naming a vector"}},
		{"a path block outside any example", appendTo("09-outputs.md", "\n```yaml path=package/metais.yaml\nname: x\n```\n"), []string{"a block naming a vector file with path= is not inside an example"}},
		{"an example in a list item", appendTo("09-outputs.md", "\n- An item:\n\n  <!-- example: out/relative-path -->\n\n  ```text path=expected/out/a.txt\n  A\n  ```\n"), nil},
		{"a subset of the vector's files", edit("09-outputs.md", "\n```text path=expected/out/a.txt\nA\n```\n", ""), nil},
		{"a block with no path ends the example", edit("09-outputs.md", "```text path=expected/out/a.txt\nA\n```", "```text\nanything\n```"), nil},
	})
}

func TestReleasedFilesUnchangedControl(t *testing.T) {
	schema := "schemas/1.0/x.schema.json"
	changed := strings.Replace(fixtureFiles[schema], `"y"`, `"z"`, 1)
	runControl(t, func(t *testing.T, f *specFixture) []string { return releasedProblems(f.root) }, []tripwireCase{
		{"a released file changed", edit(schema, `"y"`, `"z"`), []string{"1.0.sha256:2: the released file schemas/1.0/x.schema.json has changed, to sha256:" + sha256Hex([]byte(changed)) + ", and no erratum names it"}},
		{"a released vector changed", edit(vectorsDir+"/out/relative-path/vector.json", `"a path is relative"`, `"a path is relative."`), []string{"the released file conformance/vectors/out/relative-path/vector.json has changed"}},
		{"a released file changed with its erratum", func(t *testing.T, f *specFixture) {
			f.edit(t, schema, `"y"`, `"z"`)
			f.edit(t, "ERRATA.md", "- Files: none\n", "- Files: conformance/vectors/x.json sha256:"+strings.Repeat("0", 64)+"\n  "+schema+" sha256:"+sha256Hex([]byte(changed))+"\n")
		}, nil},
		{"an erratum with another digest", func(t *testing.T, f *specFixture) {
			f.edit(t, schema, `"y"`, `"z"`)
			f.edit(t, "ERRATA.md", "- Files: none\n", "- Files: "+schema+" sha256:"+strings.Repeat("0", 64)+"\n")
		}, []string{"has changed"}},
		{"an erratum naming another file", func(t *testing.T, f *specFixture) {
			f.edit(t, schema, `"y"`, `"z"`)
			f.edit(t, "ERRATA.md", "- Files: none\n", "- Files: schemas/1.0/y.schema.json sha256:"+sha256Hex([]byte(changed))+"\n")
		}, []string{"has changed"}},
		{"a released file removed", func(t *testing.T, f *specFixture) { os.Remove(f.path(schema)) }, []string{"the released file schemas/1.0/x.schema.json"}},
		{"a list out of order", func(t *testing.T, f *specFixture) {
			l := strings.Split(strings.TrimSuffix(f.read(t, releasedDir+"/1.0.sha256"), "\n"), "\n")
			f.write(t, releasedDir+"/1.0.sha256", l[1]+"\n"+l[0]+"\n")
		}, []string{"out of order"}},
		{"a list line not in sha256sum's form", appendTo(releasedDir+"/1.0.sha256", "deadbeef schemas/1.0/x.schema.json\n"), []string{"1.0.sha256:3: not a line of sha256sum's format"}},
		{"a list path out of spec/", appendTo(releasedDir+"/1.0.sha256", strings.Repeat("0", 64)+"  zz/../../x\n"), []string{"is not a path below spec/"}},
		{"a file in released/ that is no list", func(t *testing.T, f *specFixture) { f.write(t, releasedDir+"/notes.txt", "x\n") }, []string{"notes.txt is not a released revision's digest list"}},
		{"an unlisted file may change", edit("09-outputs.md", "A summary", "The summary"), nil},
	})
}

func TestChangelogControl(t *testing.T) {
	check := func(t *testing.T, f *specFixture) []string { return changelogProblems(f.root, f.statements(t)) }
	runControl(t, check, []tripwireCase{
		{"a since statement missing from Added", edit("CHANGELOG.md", "- MTS-OUT-003 — the compressor's freedom.\n", ""), []string{"MTS-OUT-003 is since 1.1 and missing from its Added"}},
		{"Added listing a statement of 1.0", edit("CHANGELOG.md", "- MTS-OUT-003 — the compressor's freedom.\n", "- MTS-OUT-003 — the compressor's freedom.\n- MTS-OUT-001 — paths.\n"), []string{"1.1 lists MTS-OUT-001 as Added, and its since is not 1.1"}},
		{"Added listing an unknown statement", edit("CHANGELOG.md", "- MTS-OUT-003 — the compressor's freedom.\n", "- MTS-OUT-003 — the compressor's freedom.\n- MTS-OUT-777 — nothing.\n"), []string{"lists MTS-OUT-777 as Added"}},
		{"Added listing one twice", edit("CHANGELOG.md", "- MTS-OUT-003 — the compressor's freedom.\n", "- MTS-OUT-003 — the compressor's freedom.\n- MTS-OUT-003 — again.\n"), []string{"lists a statement twice"}},
		{"an Added line malformed", edit("CHANGELOG.md", "- MTS-OUT-003 — the", "- MTS-OUT-003: the"), []string{"an Added line is not"}},
		{"a since with no section", edit("09-outputs.md", "*(since 1.1)*", "*(since 1.2)*"), []string{"MTS-OUT-003 are since 1.2, which CHANGELOG.md has no section for", "MTS-OUT-003 as Added"}},
		{"the first revision listing Added", edit("CHANGELOG.md", "First revision.\n", "First revision.\n\n### Added\n\n- MTS-OUT-001 — paths.\n"), []string{"1.0 opens its major"}},
		{"the first revision not saying so", edit("CHANGELOG.md", "First revision.\n", "The start.\n"), []string{"1.0 opens its major, and says \"First revision.\""}},
		{"sections oldest first", func(t *testing.T, f *specFixture) {
			f.write(t, "CHANGELOG.md", "# Changelog\n\n## 1.0 — 2027-01-15\n\nFirst revision.\n\n## 1.1 — unreleased\n\n### Added\n\n- MTS-OUT-003 — x.\n")
		}, []string{"1.1 comes after 1.0", "1.1 is unreleased, and only the newest section may be"}},
		{"a section heading malformed", edit("CHANGELOG.md", "## 1.0 — 2027-01-15", "## 1.0 - 2027-01-15"), []string{"a section heading is not"}},
		{"a part the changelog does not have", edit("CHANGELOG.md", "### Editorial", "### Fixed"), []string{`"Fixed" is not a part of a revision's section`}},
		{"the front page's version behind", edit("README.md", "Version: 1.1", "Version: 1.0"), []string{"README.md says Version: 1.0, and the newest section of CHANGELOG.md is 1.1"}},
		{"the front page released while the changelog is not", edit("README.md", "Status: Draft", "Status: Released 2027-01-15, tag spec-1.1"), []string{"README.md says Status: Released 2027-01-15, tag spec-1.1, and CHANGELOG.md gives Draft"}},
		{"the front page naming no release", edit("README.md", "Latest release: 1.0, tag spec-1.0", "Latest release: none"), []string{"README.md says Latest release: none, and CHANGELOG.md gives 1.0, tag spec-1.0"}},
		{"the changelog released while the front page is not", func(t *testing.T, f *specFixture) {
			f.edit(t, "CHANGELOG.md", "## 1.1 — unreleased", "## 1.1 — 2027-06-01")
		}, []string{"CHANGELOG.md gives Released 2027-06-01, tag spec-1.1", "CHANGELOG.md gives 1.1, tag spec-1.1"}},
		{"a released revision on both", func(t *testing.T, f *specFixture) {
			f.edit(t, "CHANGELOG.md", "## 1.1 — unreleased", "## 1.1 — 2027-06-01")
			f.edit(t, "README.md", "Status: Draft\nLatest release: 1.0, tag spec-1.0", "Status: Released 2027-06-01, tag spec-1.1\nLatest release: 1.1, tag spec-1.1")
		}, nil},
		{"a front page with no status block", edit("README.md", "```\nVersion: 1.1\nStatus: Draft\nLatest release: 1.0, tag spec-1.0\n```\n\n", ""), []string{"README.md does not open with its status block"}},
		{"a status block missing a line", edit("README.md", "Status: Draft\n", ""), []string{"lacks Version, Status or Latest release"}},
		{"a status block with another line", edit("README.md", "Status: Draft\n", "Status: Draft\nEditor: x\n"), []string{`"Editor: x" is not Version, Status or Latest release`}},
	})
}

func TestSchemasPublishedControl(t *testing.T) {
	schema := "schemas/1.0/x.schema.json"
	check := func(t *testing.T, f *specFixture) []string { return schemaProblems(f.root, f.embedded) }
	both := func(old, new string) func(*testing.T, *specFixture) {
		return func(t *testing.T, f *specFixture) {
			f.edit(t, schema, old, new)
			editFile(t, filepath.Join(f.embedded, "1.0", "x.schema.json"), old, new)
		}
	}
	runControl(t, check, []tripwireCase{
		{"another $schema", both("draft/2020-12/schema", "draft/2019-09/schema"), []string{`$schema is "https://json-schema.org/draft/2019-09/schema"`}},
		{"no $schema", both("  \"$schema\": \"https://json-schema.org/draft/2020-12/schema\",\n", ""), []string{`$schema is ""`}},
		{"an $id its path does not give", both("schemas/1.0/x.schema.json\"", "schemas/1.1/x.schema.json\""), []string{`$id is "https://metais.org/schemas/1.1/x.schema.json", not "https://metais.org/schemas/1.0/x.schema.json"`}},
		{"invalid against the meta-schema", both(`"type": "object"`, `"type": 5`), []string{"does not compile against the 2020-12 meta-schema"}},
		{"an external $ref", both(`{ "$ref": "#/$defs/a" }`, `{ "$ref": "https://example.com/a.json" }`), []string{`the external $ref ["https://example.com/a.json"]`}},
		{"a $ref to a sibling schema", both(`{ "$ref": "#/$defs/a" }`, `{ "$ref": "finding.schema.json#/$defs/a" }`), []string{`the external $ref ["finding.schema.json#/$defs/a"]`}},
		{"an external $dynamicRef", both(`{ "$ref": "#/$defs/a" }`, `{ "$dynamicRef": "https://example.com/a.json#meta" }`), []string{"the external $ref"}},
		{"four-space indentation", both("\n  \"type\": \"object\",", "\n    \"type\": \"object\","), []string{"line 4 is indented 4 spaces, not 2"}},
		{"a closing brace misindented", both("\n  },\n  \"$defs\"", "\n},\n  \"$defs\""), []string{"is indented 0 spaces, not 2"}},
		{"a tab", both("\n  \"type\"", "\n\t\"type\""), []string{"holds a CR or a tab"}},
		{"CRLF line ends", func(t *testing.T, f *specFixture) {
			f.write(t, schema, strings.ReplaceAll(fixtureFiles[schema], "\n", "\r\n"))
		}, []string{"holds a CR or a tab"}},
		{"no final LF", both("}\n}\n", "}\n}"), []string{"does not end in one LF"}},
		{"two final LFs", both("}\n}\n", "}\n}\n\n"), []string{"does not end in one LF"}},
		{"not JSON", both(`"type": "object",`, `"type": "object",,`), []string{"spec/schemas/1.0/x.schema.json:"}},
		{"a file that is not a schema", func(t *testing.T, f *specFixture) { f.write(t, "schemas/1.0/notes.txt", "x\n") }, []string{"spec/schemas/1.0/notes.txt: a file under spec/schemas/ is a schema"}},
		{"a conformance schema published wrongly", func(t *testing.T, f *specFixture) {
			f.write(t, "schemas/conformance/1/vector.schema.json", strings.Replace(fixtureFiles[schema], "1.0/x.schema.json", "conformance/1/other.schema.json", 1))
		}, []string{`spec/schemas/conformance/1/vector.schema.json: $id is "https://metais.org/schemas/conformance/1/other.schema.json"`}},
		{"a conformance schema published rightly", func(t *testing.T, f *specFixture) {
			f.write(t, "schemas/conformance/1/vector.schema.json", strings.ReplaceAll(strings.Replace(fixtureFiles[schema], "1.0/x", "conformance/1/x", 1), "x.schema.json", "vector.schema.json"))
		}, nil},
		{"an embedded copy that differs", func(t *testing.T, f *specFixture) {
			editFile(t, filepath.Join(f.embedded, "1.0", "x.schema.json"), `"y"`, `"z"`)
		}, []string{"the embedded schema 1.0/x.schema.json differs from spec/schemas/1.0/x.schema.json"}},
		{"a published file changed under its copy", edit(schema, `"y"`, `"z"`), []string{"the embedded schema 1.0/x.schema.json differs"}},
		{"an embedded copy with no published file", func(t *testing.T, f *specFixture) {
			writeFile(t, filepath.Join(f.embedded, "1.0", "gone.schema.json"), "{}\n")
		}, []string{"the embedded schema 1.0/gone.schema.json has no published file"}},
		{"no schema at all", func(t *testing.T, f *specFixture) { os.RemoveAll(f.path("schemas")) }, []string{"spec/schemas holds no schema"}},
	})
}

// TestExternalRefs pins which references leave a document.
func TestExternalRefs(t *testing.T) {
	doc := map[string]any{
		"$id": "https://metais.org/schemas/1.0/a.schema.json",
		"properties": map[string]any{
			"local":    map[string]any{"$ref": "#/$defs/x"},
			"self":     map[string]any{"$ref": "a.schema.json#/$defs/x"},
			"absolute": map[string]any{"$ref": "https://metais.org/schemas/1.0/a.schema.json"},
			"sibling":  map[string]any{"$ref": "b.schema.json"},
			"embedded": map[string]any{"$id": "inner.json", "$ref": "#/x"},
			"toInner":  map[string]any{"$ref": "inner.json#/x"},
			"data":     map[string]any{"const": map[string]any{"$ref": "https://example.com"}},
			"$ref":     map[string]any{"type": "string"},
		},
	}
	if got, want := externalRefs(doc), []string{"b.schema.json"}; !slices.Equal(got, want) {
		t.Errorf("externalRefs = %q, want %q", got, want)
	}
}

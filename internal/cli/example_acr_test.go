// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/testenv"
	"metais.org/metais/internal/zipfmt"
)

// acrPackage is the §4.1 example, the accessibility conformance report
// (plan §13).
const acrPackage = examplePackagesDir + "/accessibility-conformance-report"

// TestACREditionDecidesTables holds the §4.1 example's input constraints
// to what its guide promises: every edition takes its own tables and no
// other, validation refusing each table the edition needs and nothing
// supplied as input.missing about it, and each one supplied that the
// edition does not hold as input.forbidden about it, all in one report.
// An edition left out is wcag, by its default.
func TestACREditionDecidesTables(t *testing.T) {
	testenv.Home(t)
	type finding struct{ code, input string }
	tables := []string{"wcag", "section508", "en301549"}
	needs := map[string][]string{
		"":              {"wcag"},
		"wcag":          {"wcag"},
		"section508":    {"wcag", "section508"},
		"en301549":      {"wcag", "en301549"},
		"international": {"wcag", "section508", "en301549"},
	}
	src := filepath.Join(acrPackage, "examples/international/inputs")
	for _, edition := range []string{"", "wcag", "section508", "en301549", "international"} {
		// Every subset of the tables, each supplied as the international
		// example's own file.
		for mask := range 1 << len(tables) {
			var supplied []string
			for i, name := range tables {
				if mask&(1<<i) != 0 {
					supplied = append(supplied, name)
				}
			}
			var want []finding
			for _, name := range tables {
				switch in, needed := slices.Contains(supplied, name), slices.Contains(needs[edition], name); {
				case needed && !in:
					want = append(want, finding{"input.missing", name})
				case in && !needed:
					want = append(want, finding{"input.forbidden", name})
				}
			}
			name := "edition " + edition
			if edition == "" {
				name = "no edition"
			}
			t.Run(name+", tables "+strings.Join(supplied, "+"), func(t *testing.T) {
				data := testenv.Dir(t)
				report := acrReport(t, src, edition)
				if err := os.WriteFile(filepath.Join(data, "report.yaml"), report, 0o644); err != nil {
					t.Fatal(err)
				}
				for _, table := range supplied {
					b, err := os.ReadFile(filepath.Join(src, table+".csv"))
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(data, table+".csv"), b, 0o644); err != nil {
						t.Fatal(err)
					}
				}
				r := runTool(t, "validate", acrPackage, "--data", data, "--json")
				var doc struct {
					Findings []struct {
						Code    string
						Subject struct{ Kind, Name string }
					}
				}
				if err := json.Unmarshal([]byte(r.stdout), &doc); err != nil {
					t.Fatalf("validate's document: %v\n%s", err, r.stdout)
				}
				var got []finding
				for _, f := range doc.Findings {
					got = append(got, finding{f.Code, f.Subject.Name})
				}
				cmp := func(a, b finding) int { return strings.Compare(a.input+a.code, b.input+b.code) }
				slices.SortFunc(got, cmp)
				slices.SortFunc(want, cmp)
				if !slices.Equal(got, want) {
					t.Errorf("validate reports %v, want %v", got, want)
				}
				if ok := len(want) == 0; (r.status == 0) != ok {
					t.Errorf("validate exits %d, and its findings are %v", r.status, got)
				}
			})
		}
	}
}

// TestACRRowsRefused holds the §4.1 example's tables to the criteria and
// levels they take: a criterion number its standard does not have, a
// conformance level outside the five, a row without either, a remark
// holding a control character a Word document cannot carry, and a
// column the table does not have are each refused, about the row and
// column they are in, all in one report.
func TestACRRowsRefused(t *testing.T) {
	testenv.Home(t)
	src := filepath.Join(acrPackage, "examples/wcag_only/inputs")
	data := testenv.Dir(t)
	b, err := os.ReadFile(filepath.Join(src, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "report.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	// The rows are put into the example's own table, which answers every
	// criterion the table covers, so the findings are the rows' alone.
	table, at := acrTable(t, filepath.Join(src, "wcag.csv"),
		"1.1.1,Supports,",            // a valid row
		"302.1,Supports,",            // a Section 508 number in the WCAG table
		"1.4.3,Mostly,",              // a level outside the five
		",Supports,",                 // no criterion
		"2.1.1,,",                    // no level
		"4.1.2,Supports,\"bell \a\"", // a control character
	)
	if err := os.WriteFile(filepath.Join(data, "wcag.csv"), table, 0o644); err != nil {
		t.Fatal(err)
	}
	r := runTool(t, "validate", acrPackage, "--data", data, "--json")
	var doc struct {
		Findings []struct {
			Code     string
			Location struct{ Pointer string }
		}
	}
	if err := json.Unmarshal([]byte(r.stdout), &doc); err != nil {
		t.Fatalf("validate's document: %v\n%s", err, r.stdout)
	}
	var got []string
	for _, f := range doc.Findings {
		got = append(got, f.Code+" "+f.Location.Pointer)
	}
	slices.Sort(got)
	want := []string{
		"input.enum /" + strconv.Itoa(at[1]) + "/criterion",
		"input.enum /" + strconv.Itoa(at[2]) + "/conformance",
		"input.pattern /" + strconv.Itoa(at[5]) + "/remarks",
		"input.required /" + strconv.Itoa(at[3]) + "/criterion",
		"input.required /" + strconv.Itoa(at[4]) + "/conformance",
	}
	slices.Sort(want)
	if !slices.Equal(got, want) || r.status == 0 {
		t.Errorf("validate exits %d and reports %q, want %q", r.status, got, want)
	}
}

// acrReport is the report.yaml of the example inputs in src with its
// edition replaced by edition, or removed when edition is empty.
func acrReport(t *testing.T, src, edition string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(src, "report.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	found := false
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "edition: ") {
			found = true
			if edition == "" {
				continue
			}
			line = "edition: " + edition
		}
		out = append(out, line)
	}
	if !found {
		t.Fatalf("%s/report.yaml has no edition line", src)
	}
	return []byte(strings.Join(out, "\n"))
}

// TestACRCatalogueInStep holds the three copies of each table's catalogue
// in the §4.1 example to one list, in one order: the criterion numbers
// its schema's enum allows, the numbers its label partial names, and the
// numbers its reference file's sections give. The WCAG level partial
// names each WCAG number once, and the reference gives each the level the
// partial does.
func TestACRCatalogueInStep(t *testing.T) {
	read := func(p string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(acrPackage, p))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	whens := regexp.MustCompile(`\{%- when ((?:"[^"]+"(?:, )?)+) -%\}([^\n]*)`)
	quoted := regexp.MustCompile(`"([^"]+)"`)
	sections := regexp.MustCompile(`(?m)^## ([0-9.]+) ([^\n]*)$`)
	levels := map[string]string{}
	for _, m := range whens.FindAllStringSubmatch(read("templates/criteria/wcag-level.liquid"), -1) {
		for _, q := range quoted.FindAllStringSubmatch(m[1], -1) {
			if _, twice := levels[q[1]]; twice {
				t.Errorf("wcag-level.liquid names %s twice", q[1])
			}
			levels[q[1]] = m[2]
		}
	}
	for _, c := range []struct{ input, reference string }{
		{"wcag", "wcag-2.2.md"},
		{"section508", "section-508.md"},
		{"en301549", "en-301-549.md"},
	} {
		var schema struct {
			Items struct {
				Properties struct {
					Criterion struct{ Enum []string }
				}
			}
			MaxItems int
		}
		if err := json.Unmarshal([]byte(read("inputs/"+c.input+".schema.json")), &schema); err != nil {
			t.Fatal(err)
		}
		enum := schema.Items.Properties.Criterion.Enum
		if len(enum) == 0 || schema.MaxItems != len(enum) {
			t.Errorf("%s: the schema allows %d numbers and %d rows", c.input, len(enum), schema.MaxItems)
		}
		labels := map[string]string{}
		var named []string
		for _, m := range whens.FindAllStringSubmatch(read("templates/criteria/"+c.input+".liquid"), -1) {
			named = append(named, m[1][1:len(m[1])-1])
			labels[m[1][1:len(m[1])-1]] = m[2]
		}
		var given []string
		for _, m := range sections.FindAllStringSubmatch(read("references/"+c.reference), -1) {
			given = append(given, m[1])
			want := labels[m[1]]
			if c.input == "wcag" {
				want += " (Level " + levels[m[1]] + ")"
			}
			if m[2] != want {
				t.Errorf("%s: references/%s names %s %q, and the partials %q", c.input, c.reference, m[1], m[2], want)
			}
		}
		if !slices.Equal(named, enum) || !slices.Equal(given, enum) {
			t.Errorf("%s: the schema allows %q,\nthe partial names %q,\nand the reference gives %q", c.input, enum, named, given)
		}
		if c.input == "wcag" && len(levels) != len(enum) {
			t.Errorf("wcag-level.liquid names %d numbers, and the schema allows %d", len(levels), len(enum))
		}
	}
}

// TestACREditions holds the §4.1 example to the editions of the standards
// its terms check ruled it reports against (plan §6, W7-A1): every file
// that names EN 301 549's edition names V4.1.1 and no other, and every
// file that copies WCAG 2.2's names carries W3C's notice.
func TestACREditions(t *testing.T) {
	read := func(p string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(acrPackage, p))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	other := regexp.MustCompile(`EN 301 549 V[0-9]+\.[0-9]+\.[0-9]+`)
	for _, p := range []string{
		"metais.yaml",
		"inputs/en301549.schema.json",
		"inputs/report.md",
		"inputs/report.schema.json",
		"inputs/tables.md",
		"references/en-301-549.md",
		"references/sources.md",
		"templates/docx/document.xml.liquid",
	} {
		s := read(p)
		named := other.FindAllString(s, -1)
		if len(named) == 0 {
			t.Errorf("%s does not name EN 301 549's edition", p)
		}
		for _, n := range named {
			if n != "EN 301 549 V4.1.1" {
				t.Errorf("%s names %s, and the package reports against V4.1.1", p, n)
			}
		}
	}
	// Every copy of WCAG's names or levels carries W3C's notice, link and
	// status: the reference, the two partials, and both outputs a render
	// of the international example writes, which print them.
	data := testenv.Dir(t)
	if err := os.CopyFS(data, os.DirFS(filepath.Join(acrPackage, "examples/international/inputs"))); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(testenv.Dir(t), "out")
	if r := runTool(t, "render", acrPackage, "--data", data, "--out", out); r.status != 0 {
		t.Fatalf("render exits %d: %s", r.status, r.stderr)
	}
	b, err := os.ReadFile(filepath.Join(out, "acr.docx"))
	if err != nil {
		t.Fatal(err)
	}
	var document bytes.Buffer
	sink := func(name string) io.Writer {
		if name == "word/document.xml" {
			return &document
		}
		return nil
	}
	if _, fs, err := zipfmt.OpenContainer(bytes.NewReader(b), int64(len(b)), limits.Defaults(), sink); err != nil || len(fs) > 0 {
		t.Fatalf("the rendered acr.docx: %v %v", err, fs)
	}
	acrJSON, err := os.ReadFile(filepath.Join(out, "acr.json"))
	if err != nil {
		t.Fatal(err)
	}
	copies := map[string]string{
		"references/wcag-2.2.md":               read("references/wcag-2.2.md"),
		"templates/criteria/wcag.liquid":       read("templates/criteria/wcag.liquid"),
		"templates/criteria/wcag-level.liquid": read("templates/criteria/wcag-level.liquid"),
		"the rendered acr.docx":                document.String(),
		"the rendered acr.json":                string(acrJSON),
	}
	space := regexp.MustCompile(`\s+`)
	for _, p := range slices.Sorted(maps.Keys(copies)) {
		s := space.ReplaceAllString(copies[p], " ")
		for _, want := range []string{
			"Copyright © 2020-2024 World Wide Web Consortium. W3C® liability, trademark and document use rules apply.",
			"https://www.w3.org/TR/2024/REC-WCAG22-20241212/",
			"W3C Recommendation 12 December 2024",
		} {
			if !strings.Contains(s, want) {
				t.Errorf("%s copies WCAG 2.2's names or levels without %q", p, want)
			}
		}
	}
	if !strings.Contains(document.String(), "Non-text Content") || !strings.Contains(string(acrJSON), "Non-text Content") {
		t.Errorf("the rendered outputs print no WCAG name, so this test sees no copy of one")
	}
}

// acrLevels are the five conformance levels PRD-001 §4.1 names, in its
// order.
var acrLevels = []string{"Supports", "Partially Supports", "Does Not Support", "Not Applicable", "Not Evaluated"}

// TestACRFiveLevels holds the §4.1 example to the five conformance levels
// and no other: the level every table's schema takes, the level acr.json's
// schema allows, and the levels the report and its input's meaning define.
func TestACRFiveLevels(t *testing.T) {
	read := func(p string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(acrPackage, p))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	var conformance struct{ Enum []string }
	if err := json.Unmarshal([]byte(read("inputs/conformance.schema.json")), &conformance); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(conformance.Enum, acrLevels) {
		t.Errorf("inputs/conformance.schema.json allows %q, want %q", conformance.Enum, acrLevels)
	}
	for _, input := range []string{"wcag", "section508", "en301549"} {
		var schema struct {
			Items struct {
				Properties struct {
					Conformance struct {
						Ref  string `json:"$ref"`
						Enum []string
					}
				}
			}
		}
		if err := json.Unmarshal([]byte(read("inputs/"+input+".schema.json")), &schema); err != nil {
			t.Fatal(err)
		}
		if c := schema.Items.Properties.Conformance; c.Ref != "conformance.schema.json" || c.Enum != nil {
			t.Errorf("inputs/%s.schema.json's conformance is %+v, not inputs/conformance.schema.json's levels", input, c)
		}
	}
	var acr struct {
		Defs struct {
			Criterion struct {
				Properties struct {
					Conformance struct{ Enum []string }
				}
			}
		} `json:"$defs"`
	}
	if err := json.Unmarshal([]byte(read("references/acr.schema.json")), &acr); err != nil {
		t.Fatal(err)
	}
	if got := acr.Defs.Criterion.Properties.Conformance.Enum; !slices.Equal(got, acrLevels) {
		t.Errorf("references/acr.schema.json allows %q, want %q", got, acrLevels)
	}
	for p, re := range map[string]*regexp.Regexp{
		"inputs/conformance.md":              regexp.MustCompile(`(?m)^- \*\*([^*]+)\*\*:`),
		"templates/docx/document.xml.liquid": regexp.MustCompile(`<w:b/></w:rPr><w:t xml:space="preserve">([^:<]+): </w:t></w:r><w:r><w:t xml:space="preserve">the `),
	} {
		var defined []string
		for _, m := range re.FindAllStringSubmatch(read(p), -1) {
			defined = append(defined, m[1])
		}
		if !slices.Equal(defined, acrLevels) {
			t.Errorf("%s defines %q, want %q", p, defined, acrLevels)
		}
	}
}

// TestACRJSONValidated holds the §4.1 example to its guide's promise that
// every render checks acr.json against references/acr.schema.json before it
// writes anything: a copy of the package whose acr.json template writes a
// format the schema does not allow renders nothing, and reports the
// output's refusal.
func TestACRJSONValidated(t *testing.T) {
	testenv.Home(t)
	pkg := copyTree(t, acrPackage)
	tmpl := filepath.Join(pkg, "templates/acr.json.liquid")
	b, err := os.ReadFile(tmpl)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(b), `"format": "metais-example-acr/1"`, `"format": "metais-example-acr/0"`, 1)
	if broken == string(b) {
		t.Fatal("templates/acr.json.liquid writes no format to break")
	}
	if err := os.WriteFile(tmpl, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(pkg, "examples/wcag_only/inputs")
	out := filepath.Join(testenv.Dir(t), "out")
	r := runTool(t, "render", pkg, "--data", data, "--out", out, "--json")
	if r.status == 0 || !slices.Contains(r.codes(), "output.const") {
		t.Errorf("render of an acr.json the schema refuses exits %d and reports %q, want output.const", r.status, r.codes())
	}
	if ents, err := os.ReadDir(out); err == nil && len(ents) > 0 {
		t.Errorf("render of an acr.json the schema refuses wrote %d entries into the output directory", len(ents))
	}
}

// TestACRClaimsFromRows holds the §4.1 example's report to the rows it
// was given: the WCAG levels and the Revised Section 508 chapters its
// Standards section names are the ones its tables list, and no other; its
// summary counts each table's rows by conformance level, in acr.json as in
// acr.docx; and its date stands under its title, before the product. Each
// case varies one table's rows just past where a fixed claim shows: a
// level or a chapter left out, one alone, all of them, and counts of zero.
//
// A table that leaves a level or a chapter out is one coverage refuses,
// so the cases render a copy of the package whose tables carry no
// coverage: the claims are the templates' work, and coverage only decides
// which tables reach them.
func TestACRClaimsFromRows(t *testing.T) {
	testenv.Home(t)
	pkg := acrUncovered(t)
	src := filepath.Join(acrPackage, "examples/international/inputs")
	en301549, err := os.ReadFile(filepath.Join(src, "en301549.csv"))
	if err != nil {
		t.Fatal(err)
	}
	header := "criterion,conformance,remarks\n"
	for _, c := range []struct {
		name, wcag, section508 string
		standards              []string
		summary                []string
		counts                 map[string]map[string]int
	}{
		{
			name:       "levels A and AA, chapters 3 and 5",
			wcag:       "1.1.1,Supports,\n1.4.3,Partially Supports,\n2.1.1,Supports,\n",
			section508: "302.1,Supports,\n502.3.1,Does Not Support,\n",
			standards: []string{
				"Web Content Accessibility Guidelines (WCAG) 2.2, W3C Recommendation: its table lists success criteria at Levels A and AA.",
				"Revised Section 508 Standards, 36 CFR part 1194: its table lists criteria of chapters 3 and 5.",
			},
			summary: []string{
				"WCAG 2.2: 2 Supports, 1 Partially Supports (3 criteria)",
				"Revised Section 508: 1 Supports, 1 Does Not Support (2 criteria)",
			},
			counts: map[string]map[string]int{
				"wcag":       {"Supports": 2, "Partially Supports": 1},
				"section508": {"Supports": 1, "Does Not Support": 1},
			},
		},
		{
			name:       "level AAA alone, chapter 6 alone",
			wcag:       "2.4.13,Not Evaluated,Out of the evaluation's scope.\n",
			section508: "602.3,Not Applicable,\n",
			standards: []string{
				"Web Content Accessibility Guidelines (WCAG) 2.2, W3C Recommendation: its table lists success criteria at Level AAA.",
				"Revised Section 508 Standards, 36 CFR part 1194: its table lists criteria of chapter 6.",
			},
			summary: []string{
				"WCAG 2.2: 1 Not Evaluated (1 criterion)",
				"Revised Section 508: 1 Not Applicable (1 criterion)",
			},
			counts: map[string]map[string]int{
				"wcag":       {"Not Evaluated": 1},
				"section508": {"Not Applicable": 1},
			},
		},
		{
			name:       "every level, every chapter",
			wcag:       "1.1.1,Supports,\n1.4.3,Supports,\n2.4.13,Supports,\n",
			section508: "302.1,Supports,\n402.1,Supports,\n502.3.1,Supports,\n602.3,Supports,\n",
			standards: []string{
				"Web Content Accessibility Guidelines (WCAG) 2.2, W3C Recommendation: its table lists success criteria at Levels A, AA and AAA.",
				"Revised Section 508 Standards, 36 CFR part 1194: its table lists criteria of chapters 3 to 6.",
			},
			summary: []string{
				"WCAG 2.2: 3 Supports (3 criteria)",
				"Revised Section 508: 4 Supports (4 criteria)",
			},
			counts: map[string]map[string]int{
				"wcag":       {"Supports": 3},
				"section508": {"Supports": 4},
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			data := testenv.Dir(t)
			files := map[string][]byte{
				"report.yaml":    acrReport(t, src, "international"),
				"wcag.csv":       []byte(header + c.wcag),
				"section508.csv": []byte(header + c.section508),
				"en301549.csv":   en301549,
			}
			for name, b := range files {
				if err := os.WriteFile(filepath.Join(data, name), b, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			out := filepath.Join(testenv.Dir(t), "out")
			if r := runTool(t, "render", pkg, "--data", data, "--out", out); r.status != 0 {
				t.Fatalf("render exits %d: %s", r.status, r.stderr)
			}
			paragraphs := acrParagraphs(t, filepath.Join(out, "acr.docx"))
			for _, want := range append(slices.Clone(c.standards), c.summary...) {
				if !slices.Contains(paragraphs, want) {
					t.Errorf("acr.docx has no paragraph %q; it has:\n%s", want, strings.Join(paragraphs, "\n"))
				}
			}
			date := slices.Index(paragraphs, "Report date: 1 October 2026")
			product := slices.Index(paragraphs, "Product")
			if date < 0 || product < 0 || date > product || date > 3 {
				t.Errorf("acr.docx's report date is paragraph %d and its Product heading %d, want the date under the title, before the product", date, product)
			}
			b, err := os.ReadFile(filepath.Join(out, "acr.json"))
			if err != nil {
				t.Fatal(err)
			}
			var doc struct {
				Tables []struct {
					Standard string
					Summary  struct {
						Criteria    int
						Conformance map[string]int
					}
					Criteria []json.RawMessage
				}
			}
			if err := json.Unmarshal(b, &doc); err != nil {
				t.Fatal(err)
			}
			for _, table := range doc.Tables {
				want, ok := c.counts[table.Standard]
				if !ok {
					continue
				}
				if table.Summary.Criteria != len(table.Criteria) {
					t.Errorf("acr.json's %s summary counts %d criteria, and the table lists %d", table.Standard, table.Summary.Criteria, len(table.Criteria))
				}
				for _, level := range acrLevels {
					if got := table.Summary.Conformance[level]; got != want[level] {
						t.Errorf("acr.json's %s summary counts %d %s, want %d", table.Standard, got, level, want[level])
					}
				}
			}
		})
	}
}

// acrParagraphs is the text of each paragraph of the acr.docx at path's
// word/document.xml, in order, with its XML character references
// resolved.
func acrParagraphs(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document bytes.Buffer
	sink := func(name string) io.Writer {
		if name == "word/document.xml" {
			return &document
		}
		return nil
	}
	if _, fs, err := zipfmt.OpenContainer(bytes.NewReader(b), int64(len(b)), limits.Defaults(), sink); err != nil || len(fs) > 0 {
		t.Fatalf("%s: %v %v", path, err, fs)
	}
	tag := regexp.MustCompile(`<[^>]*>`)
	var paragraphs []string
	for _, p := range strings.Split(document.String(), "</w:p>") {
		if s := html.UnescapeString(tag.ReplaceAllString(p, "")); strings.TrimSpace(s) != "" {
			paragraphs = append(paragraphs, strings.TrimSpace(s))
		}
	}
	return paragraphs
}

// TestACRRequiredLevelsEvaluated holds the §4.1 example to ADR-003 A2
// item 8's per-row rule: a WCAG 2.2 criterion at Level A or AA is never
// Not Evaluated. One table answers every criterion the level partial
// names Not Evaluated, each saying why, and validation refuses each Level A
// and AA row and no Level AAA one, at the row's conformance cell, each failure naming
// the condition that selected it, all in one report.
func TestACRRequiredLevelsEvaluated(t *testing.T) {
	testenv.Home(t)
	b, err := os.ReadFile(filepath.Join(acrPackage, "templates/criteria/wcag-level.liquid"))
	if err != nil {
		t.Fatal(err)
	}
	whens := regexp.MustCompile(`\{%- when ((?:"[^"]+"(?:, )?)+) -%\}([^\n]*)`)
	quoted := regexp.MustCompile(`"([^"]+)"`)
	csv := "criterion,conformance,remarks\n"
	var want []string
	row := 0
	for _, m := range whens.FindAllStringSubmatch(string(b), -1) {
		for _, q := range quoted.FindAllStringSubmatch(m[1], -1) {
			csv += q[1] + ",Not Evaluated,Out of the evaluation's scope.\n"
			if m[2] == "A" || m[2] == "AA" {
				want = append(want, "input.not /"+strconv.Itoa(row)+"/conformance")
			}
			row++
		}
	}
	src := filepath.Join(acrPackage, "examples/wcag_only/inputs")
	data := testenv.Dir(t)
	report, err := os.ReadFile(filepath.Join(src, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	for name, b := range map[string][]byte{"report.json": report, "wcag.csv": []byte(csv)} {
		if err := os.WriteFile(filepath.Join(data, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := runTool(t, "validate", acrPackage, "--data", data, "--json")
	var doc struct {
		Findings []struct {
			Code     string
			Location struct{ Pointer string }
			Via      []struct {
				Keyword   string
				Condition struct{ Title string }
			}
		}
	}
	if err := json.Unmarshal([]byte(r.stdout), &doc); err != nil {
		t.Fatalf("validate's document: %v\n%s", err, r.stdout)
	}
	var got []string
	for _, f := range doc.Findings {
		got = append(got, f.Code+" "+f.Location.Pointer)
		if len(f.Via) != 1 || f.Via[0].Keyword != "then" || f.Via[0].Condition.Title != "A Level A or AA criterion" {
			t.Errorf("the finding at %s is reached by %+v, want a then whose if is titled %q", f.Location.Pointer, f.Via, "A Level A or AA criterion")
		}
	}
	if !slices.Equal(got, want) || r.status == 0 {
		t.Errorf("validate exits %d and reports %q,\nwant %q", r.status, got, want)
	}
}

// TestACRNotEvaluatedSaysWhy holds the §4.1 example to Ed Funnekotter's
// ruling of 2026-10-23 on W1-3: in every table a row answered Not
// Evaluated says why in its remarks, so a Revised Section 508 or EN 301
// 549 criterion an evaluator cannot assess stays honestly Not Evaluated
// and a bare one is refused. In each table a Not Evaluated row with no
// remarks, and one whose remarks are only spaces, are refused at the
// row's remarks, each failure naming the condition that selected it;
// one that says why, and a row of another level with no remarks, are
// accepted, all in one report.
func TestACRNotEvaluatedSaysWhy(t *testing.T) {
	testenv.Home(t)
	src := filepath.Join(acrPackage, "examples/international/inputs")
	data := testenv.Dir(t)
	// Each table's rows are put into the example's own table, which
	// answers every criterion the table covers, so the findings are the
	// rows' alone.
	rows := map[string][]string{
		// Level AAA criteria: a Level A or AA one is never Not Evaluated.
		"wcag":       {"1.2.6,Not Evaluated,", "1.2.7,Not Evaluated,\"   \"", "1.2.8,Not Evaluated,Out of the evaluation's scope.", "1.2.9,Supports,"},
		"section508": {"602.2,Not Evaluated,", "602.3,Not Evaluated,\"   \"", "602.4,Not Evaluated,The evaluator had no access to the vendor's documentation.", "603.2,Supports,"},
		"en301549":   {"12.1,Not Evaluated,", "12.2,Not Evaluated,\"   \"", "12.3,Not Evaluated,The evaluator had no access to the vendor's support services.", "12.4,Supports,"},
	}
	if err := os.WriteFile(filepath.Join(data, "report.yaml"), acrReport(t, src, "international"), 0o644); err != nil {
		t.Fatal(err)
	}
	var want []string
	for table, rows := range rows {
		b, at := acrTable(t, filepath.Join(src, table+".csv"), rows...)
		if err := os.WriteFile(filepath.Join(data, table+".csv"), b, 0o644); err != nil {
			t.Fatal(err)
		}
		want = append(want, table+" input.required /"+strconv.Itoa(at[0])+"/remarks", table+" input.pattern /"+strconv.Itoa(at[1])+"/remarks")
	}
	slices.Sort(want)
	r := runTool(t, "validate", acrPackage, "--data", data, "--json")
	var doc struct {
		Findings []struct {
			Code     string
			Subject  struct{ Name string }
			Location struct{ Pointer string }
			Via      []struct {
				Keyword   string
				Condition struct{ Title string }
			}
		}
	}
	if err := json.Unmarshal([]byte(r.stdout), &doc); err != nil {
		t.Fatalf("validate's document: %v\n%s", err, r.stdout)
	}
	var got []string
	for _, f := range doc.Findings {
		got = append(got, f.Subject.Name+" "+f.Code+" "+f.Location.Pointer)
		if len(f.Via) != 1 || f.Via[0].Keyword != "then" || f.Via[0].Condition.Title != "Answered Not Evaluated" {
			t.Errorf("the %s finding at %s is reached by %+v, want a then whose if is titled %q", f.Subject.Name, f.Location.Pointer, f.Via, "Answered Not Evaluated")
		}
	}
	slices.Sort(got)
	if !slices.Equal(got, want) || r.status == 0 {
		t.Errorf("validate exits %d and reports %q,\nwant %q", r.status, got, want)
	}
}

// acrWCAGRequired is every WCAG 2.2 criterion that
// templates/criteria/wcag-level.liquid names Level A or AA, in the
// partial's order.
func acrWCAGRequired(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(acrPackage, "templates/criteria/wcag-level.liquid"))
	if err != nil {
		t.Fatal(err)
	}
	whens := regexp.MustCompile(`\{%- when ((?:"[^"]+"(?:, )?)+) -%\}([^\n]*)`)
	quoted := regexp.MustCompile(`"([^"]+)"`)
	var required []string
	for _, m := range whens.FindAllStringSubmatch(string(b), -1) {
		if m[2] != "A" && m[2] != "AA" {
			continue
		}
		for _, q := range quoted.FindAllStringSubmatch(m[1], -1) {
			required = append(required, q[1])
		}
	}
	return required
}

// TestACRTesterTableRefused holds the §4.1 example to the table the
// external tester's run validated on 2026-10-23 (plan §1): a WCAG 2.2
// table that answers 1.1.1 once Supports and once Does Not Support, and
// no other criterion. Validation refuses it, all in one report: one
// input.unique_keys at the second row's criterion, and one input.coverage
// at the table for each other Level A and AA criterion the level partial
// names, its missing parameter that criterion.
func TestACRTesterTableRefused(t *testing.T) {
	testenv.Home(t)
	src := filepath.Join(acrPackage, "examples/wcag_only/inputs")
	data := testenv.Dir(t)
	report, err := os.ReadFile(filepath.Join(src, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	csv := "criterion,conformance,remarks\n" +
		"1.1.1,Supports,Every image has a text alternative.\n" +
		"1.1.1,Does Not Support,The dashboard's chart has no text alternative.\n"
	for name, b := range map[string][]byte{"report.json": report, "wcag.csv": []byte(csv)} {
		if err := os.WriteFile(filepath.Join(data, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := runTool(t, "validate", acrPackage, "--data", data, "--json")
	var doc struct {
		Findings []struct {
			Code     string
			Subject  struct{ Name string }
			Location struct{ Pointer string }
			Params   map[string]any
		}
	}
	if err := json.Unmarshal([]byte(r.stdout), &doc); err != nil {
		t.Fatalf("validate's document: %v\n%s", err, r.stdout)
	}
	var got, missing []string
	for _, f := range doc.Findings {
		got = append(got, f.Subject.Name+" "+f.Code+" "+f.Location.Pointer)
		if f.Code == "input.coverage" {
			// missing is the value as canonical JSON text.
			var m string
			text, _ := f.Params["missing"].(string)
			if err := json.Unmarshal([]byte(text), &m); err != nil {
				t.Errorf("an input.coverage finding's missing is %q, not a JSON string", text)
			}
			missing = append(missing, m)
		}
	}
	required := acrWCAGRequired(t)
	want := []string{"wcag input.unique_keys /1/criterion"}
	var wantMissing []string
	for _, c := range required {
		if c != "1.1.1" {
			want = append(want, "wcag input.coverage ")
			wantMissing = append(wantMissing, c)
		}
	}
	slices.Sort(got)
	slices.Sort(want)
	slices.Sort(missing)
	slices.Sort(wantMissing)
	if len(required) != 55 || len(wantMissing) != 54 {
		t.Errorf("the level partial names %d Level A and AA criteria, want 55", len(required))
	}
	if !slices.Equal(got, want) || r.status == 0 {
		t.Errorf("validate exits %d and reports %q,\nwant %q", r.status, got, want)
	}
	if !slices.Equal(missing, wantMissing) {
		t.Errorf("validate's input.coverage findings miss %q,\nwant %q", missing, wantMissing)
	}
}

// acrTable is the table in the CSV file at path, every criterion of its
// table answered, with rows put in: each row, a CSV record, replaces the
// file's row for its criterion, or follows the file's rows when the file
// has none. It returns the table and each row's index in it, in rows'
// order, so that a test of one rule sees no input.coverage finding.
func acrTable(t *testing.T, path string, rows ...string) ([]byte, []int) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(bytes.NewReader(b)).ReadAll()
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	var at []int
	for _, row := range rows {
		r := csv.NewReader(strings.NewReader(row))
		r.FieldsPerRecord = 3
		record, err := r.Read()
		if err != nil {
			t.Fatalf("the row %q: %v", row, err)
		}
		i := slices.IndexFunc(records[1:], func(rec []string) bool { return record[0] != "" && rec[0] == record[0] })
		if i < 0 {
			records = append(records, record)
			i = len(records) - 2
		} else {
			records[i+1] = record
		}
		at = append(at, i)
	}
	var out bytes.Buffer
	w := csv.NewWriter(&out)
	if err := w.WriteAll(records); err != nil {
		t.Fatal(err)
	}
	return out.Bytes(), at
}

// acrUncovered is a copy of the §4.1 example whose criteria tables carry
// no coverage, so that a table may leave criteria out.
func acrUncovered(t *testing.T) string {
	t.Helper()
	pkg := copyTree(t, acrPackage)
	for _, table := range []string{"wcag", "section508", "en301549"} {
		p := filepath.Join(pkg, "inputs", table+".schema.json")
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(b, &schema); err != nil {
			t.Fatal(err)
		}
		if _, ok := schema["coverage"]; !ok {
			t.Fatalf("inputs/%s.schema.json carries no coverage to take out", table)
		}
		delete(schema, "coverage")
		if b, err = json.Marshal(schema); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return pkg
}

// TestACRTablesCovered holds each criteria table of the §4.1 example to
// the rows it must hold: its array is unique on its criterion, and covers
// the criteria the report answers in full. For WCAG 2.2 that is every
// criterion the level partial names Level A or AA, and none at Level AAA,
// so a Level A or AA criterion added to or dropped from the one list
// without the other fails here. For the Revised Section 508 and EN 301
// 549 tables it is every criterion the table takes, as inputs/tables.md
// asks.
func TestACRTablesCovered(t *testing.T) {
	required := acrWCAGRequired(t)
	for _, input := range []string{"wcag", "section508", "en301549"} {
		b, err := os.ReadFile(filepath.Join(acrPackage, "inputs", input+".schema.json"))
		if err != nil {
			t.Fatal(err)
		}
		var schema struct {
			UniqueKeys []string
			Coverage   struct {
				Key    string
				Values []string
			}
			Items struct {
				Properties struct {
					Criterion struct{ Enum []string }
				}
			}
		}
		if err := json.Unmarshal(b, &schema); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(schema.UniqueKeys, []string{"/criterion"}) {
			t.Errorf("inputs/%s.schema.json's uniqueKeys is %q, want [\"/criterion\"]", input, schema.UniqueKeys)
		}
		if schema.Coverage.Key != "/criterion" {
			t.Errorf("inputs/%s.schema.json's coverage key is %q, want /criterion", input, schema.Coverage.Key)
		}
		want := schema.Items.Properties.Criterion.Enum
		if input == "wcag" {
			want = slices.Clone(required)
			slices.SortFunc(want, func(a, b string) int {
				return slices.Index(schema.Items.Properties.Criterion.Enum, a) - slices.Index(schema.Items.Properties.Criterion.Enum, b)
			})
		}
		if !slices.Equal(schema.Coverage.Values, want) {
			t.Errorf("inputs/%s.schema.json covers %q,\nwant %q", input, schema.Coverage.Values, want)
		}
	}
	if len(required) != 55 {
		t.Errorf("the level partial names %d Level A and AA criteria, and WCAG 2.2 has 55", len(required))
	}
}

// TestACRCriterionAnsweredOnce holds each criteria table of the §4.1
// example to answering a criterion once: a complete table that answers
// one criterion again twice more, at its end, is refused with one
// input.unique_keys for each repeat, at the repeat's criterion, naming
// the first row as the one it repeats, all in one report. Its only other
// finding is the input.max_items of a table whose rows already number its
// criteria.
func TestACRCriterionAnsweredOnce(t *testing.T) {
	testenv.Home(t)
	src := filepath.Join(acrPackage, "examples/international/inputs")
	data := testenv.Dir(t)
	if err := os.WriteFile(filepath.Join(data, "report.yaml"), acrReport(t, src, "international"), 0o644); err != nil {
		t.Fatal(err)
	}
	var want []string
	for table, criterion := range map[string]string{"wcag": "1.1.1", "section508": "302.1", "en301549": "4.2.1"} {
		b, at := acrTable(t, filepath.Join(src, table+".csv"), criterion+",Supports,")
		rows := strings.Count(string(b), "\n") - 1
		b = append(b, criterion+",Does Not Support,\n"+criterion+",Partially Supports,\n"...)
		if err := os.WriteFile(filepath.Join(data, table+".csv"), b, 0o644); err != nil {
			t.Fatal(err)
		}
		for _, i := range []int{rows, rows + 1} {
			want = append(want, table+" input.unique_keys /"+strconv.Itoa(i)+"/criterion of "+strconv.Itoa(at[0]))
		}
		// The Revised Section 508 and EN 301 549 tables hold as many rows
		// as they have criteria, so the repeats also make one too long.
		if table != "wcag" {
			want = append(want, table+" input.max_items  of <nil>")
		}
	}
	r := runTool(t, "validate", acrPackage, "--data", data, "--json")
	var doc struct {
		Findings []struct {
			Code     string
			Subject  struct{ Name string }
			Location struct{ Pointer string }
			Params   map[string]any
		}
	}
	if err := json.Unmarshal([]byte(r.stdout), &doc); err != nil {
		t.Fatalf("validate's document: %v\n%s", err, r.stdout)
	}
	var got []string
	for _, f := range doc.Findings {
		got = append(got, f.Subject.Name+" "+f.Code+" "+f.Location.Pointer+" of "+fmt.Sprint(f.Params["duplicate_of"]))
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) || r.status == 0 {
		t.Errorf("validate exits %d and reports %q,\nwant %q", r.status, got, want)
	}
}

// TestACRSampleRendered holds the §4.1 example's preview to its tables'
// coverage: render --sample, with --items below the 55 criteria WCAG 2.2's
// table covers, renders with no finding, every covered criterion
// answered once in each table of acr.json, and two runs with one seed
// write the same bytes. TestExamplePackages previews at the default count
// and compares a preview with a render of kept sample data; this holds the
// seed's determinism and the counts below and just under the coverage.
func TestACRSampleRendered(t *testing.T) {
	testenv.Home(t)
	required := acrWCAGRequired(t)
	for _, items := range []string{"1", "3", "54"} {
		t.Run("items "+items, func(t *testing.T) {
			var outs [2]string
			for i := range outs {
				outs[i] = filepath.Join(testenv.Dir(t), "out")
				r := runTool(t, "render", acrPackage, "--sample", "--seed", "7", "--items", items, "--out", outs[i], "--json")
				if r.status != 0 || len(r.codes()) > 0 {
					t.Fatalf("render --sample --items %s exits %d and reports %q", items, r.status, r.codes())
				}
			}
			for _, name := range []string{"acr.docx", "acr.json"} {
				a, errA := os.ReadFile(filepath.Join(outs[0], name))
				b, errB := os.ReadFile(filepath.Join(outs[1], name))
				if errA != nil || errB != nil || !bytes.Equal(a, b) {
					t.Errorf("two renders with seed 7 write %s differently (%v, %v)", name, errA, errB)
				}
			}
			b, err := os.ReadFile(filepath.Join(outs[0], "acr.json"))
			if err != nil {
				t.Fatal(err)
			}
			var doc struct {
				Tables []struct {
					Standard string
					Criteria []struct{ Criterion string }
				}
			}
			if err := json.Unmarshal(b, &doc); err != nil {
				t.Fatal(err)
			}
			for _, table := range doc.Tables {
				var listed []string
				for _, c := range table.Criteria {
					if slices.Contains(listed, c.Criterion) {
						t.Errorf("the preview's %s table answers %s twice", table.Standard, c.Criterion)
					}
					listed = append(listed, c.Criterion)
				}
				if table.Standard != "wcag" {
					continue
				}
				for _, c := range required {
					if !slices.Contains(listed, c) {
						t.Errorf("the preview's wcag table leaves out %s", c)
					}
				}
			}
		})
	}
}

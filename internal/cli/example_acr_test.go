// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
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
	csv := "criterion,conformance,remarks\n" +
		"1.1.1,Supports,\n" + // a valid row
		"302.1,Supports,\n" + // a Section 508 number in the WCAG table
		"1.4.3,Mostly,\n" + // a level outside the five
		",Supports,\n" + // no criterion
		"2.1.1,,\n" + // no level
		"4.1.2,Supports,\"bell \a\"\n" // a control character
	if err := os.WriteFile(filepath.Join(data, "wcag.csv"), []byte(csv), 0o644); err != nil {
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
		"input.enum /1/criterion",
		"input.enum /2/conformance",
		"input.pattern /5/remarks",
		"input.required /3/criterion",
		"input.required /4/conformance",
	}
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

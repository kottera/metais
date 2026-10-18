// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
)

// orderedFindings returns findings in ADR-006 D8 item 1's order, one or
// more pairs differing in each member of the key in turn.
func orderedFindings() []Finding {
	set := func(arg int, ptr string) *Location {
		l := Location{Origin: OriginSet, Argument: arg}.WithPointer(ptr)
		return &l
	}
	pkg := func(file string, line, col int, ptr string) *Location {
		l := Location{Origin: OriginPackage, File: file, Line: line, Column: col}.WithPointer(ptr)
		return &l
	}
	out := func(file string) *Location {
		l := Location{Origin: OriginOutput, File: file}.WithPointer("")
		return &l
	}
	typ := func(actual string) Params {
		return Params{"expected": List(String("string")), "actual": String(actual)}
	}
	tmpl := Subject{Kind: SubjectTemplate, Name: "templates/a.liquid"}
	tmplB := Subject{Kind: SubjectTemplate, Name: "templates/b.liquid"}
	return []Finding{
		// 1. subject kind: invocation first.
		New(codeUsageFlag, Subject{Kind: SubjectInvocation}, nil, nil),
		New(codePackageName, Subject{Kind: SubjectPackage}, pkg("a\u00e9", 0, 0, ""), nil),
		New(codeManifestField, Subject{Kind: SubjectManifest}, pkg("metais.yaml", 3, 1, "/x"), Params{"member": String("x")}),
		// 2. subject name, by bytes: "Zeta" before "alpha".
		New(codeInputType, input("Zeta"), set(1, ""), typ("null")),
		// 3. origin: data_dir before set.
		minLength("alpha", "alpha.csv", 0, "/0/a", 4),
		// 5. a finding with no line before one with a line, then lines.
		minLength("alpha", "alpha.csv", 2, "/0/b", 4),
		minLength("alpha", "alpha.csv", 9, "/7/a", 4),
		minLength("alpha", "alpha.csv", 10, "/8/a", 4),
		// 4. file.
		minLength("alpha", "beta.csv", 1, "/0/a", 4),
		// 4. argument, after file; 6. pointers: /9 before /10, a prefix
		// before its extensions, an index before a member name.
		New(codeInputType, input("alpha"), set(1, ""), typ("null")),
		New(codeInputType, input("alpha"), set(1, "/9"), typ("null")),
		New(codeInputType, input("alpha"), set(1, "/10"), typ("null")),
		New(codeInputType, input("alpha"), set(1, "/10/x"), typ("null")),
		New(codeInputType, input("alpha"), set(1, "/10/x~1y"), typ("null")),
		New(codeInputType, input("alpha"), set(1, "/a"), typ("null")),
		// 7. code: input.min_length before input.required before input.type.
		New(codeInputMinLength, input("alpha"), set(2, "/0"), Params{"limit": Int(10), "length": Int(4)}),
		New(codeInputRequired, input("alpha"), set(2, "/0"), Params{"property": String("z")},
			WithSchema(Schema{Entry: "inputs/alpha.schema.json", Pointer: "/a/required", Keyword: "required", KeywordLocation: "/a/required"})),
		// 8. schema pointer before params: "z" above sorts before "a" here.
		New(codeInputRequired, input("alpha"), set(2, "/0"), Params{"property": String("a")},
			WithSchema(Schema{Entry: "inputs/alpha.schema.json", Pointer: "/b/required", Keyword: "required", KeywordLocation: "/b/required"})),
		// 9. params, canonically: "array" before "number".
		New(codeInputType, input("alpha"), set(2, "/0"), typ("array")),
		New(codeInputType, input("alpha"), set(2, "/0"), typ("number")),
		New(codeInputType, input("beta"), set(1, "/0"), typ("null")),
		// Last, the subject's title, which only breaks a tie.
		New(codeInputType, Subject{Kind: SubjectInput, Name: "beta", Title: "BETA2"}, set(1, "/0"), typ("null")),
		// Severity is not in the key: a warning sorts by its subject.
		New(codeWarnUnused, tmpl, pkg("templates/a.liquid", 1, 1, ""), nil),
		New(codeTemplateParse, tmpl, pkg("templates/a.liquid", 2, 1, ""), nil),
		New(codeTemplateParse, tmplB, pkg("templates/b.liquid", 1, 1, ""), nil),
		New(codeOutputPath, Subject{Kind: SubjectOutput, Name: "page"}, out("page.html"), nil),
	}
}

// describe is a finding as the ordering tests compare it: every member of
// the sort key and the duplicate key, and the message.
func describe(f Finding) string {
	return fmt.Sprintf("%s %+v %+v %+v %s %s", f.Code(), f.subject, f.locationOrZero(), f.schemaOrZero(), f.canonicalParams(), f.message)
}

func describeAll(fs []Finding) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = describe(f)
	}
	return out
}

// outOfOrder returns the first place fs differs from want, or "" when they
// are the same list. It is the detector the ordering test applies.
func outOfOrder(fs, want []Finding) string {
	got, w := describeAll(fs), describeAll(want)
	for i := range min(len(got), len(w)) {
		if got[i] != w[i] {
			return fmt.Sprintf("at %d:\n got  %s\n want %s", i, got[i], w[i])
		}
	}
	if len(got) != len(w) {
		return fmt.Sprintf("got %d findings, want %d", len(got), len(w))
	}
	return ""
}

// TestReportOrder is ADR-006 D13 item 5's ordering test. It adds the same
// findings in many orders and requires D8's order every time. It was red
// against a report that keeps the order findings are added in; its control
// shows the detector still catches that order.
func TestReportOrder(t *testing.T) {
	withTestCodes(t)
	want := orderedFindings()
	rng := rand.New(rand.NewPCG(1, 2))
	scrambled := 0
	for i := range 500 {
		in := slices.Clone(want)
		if i > 0 {
			rng.Shuffle(len(in), func(a, b int) { in[a], in[b] = in[b], in[a] })
		}
		r := NewReport(1000)
		for _, f := range in {
			r.Add(f)
		}
		if d := outOfOrder(r.Findings(), want); d != "" {
			t.Fatalf("permutation %d: the report is out of D8's order %s", i, d)
		}
		// Control: the order the findings were added in is caught.
		if outOfOrder(in, want) != "" {
			scrambled++
		}
	}
	if scrambled == 0 {
		t.Fatal("control: no permutation was out of order, so the test proves nothing")
	}
}

// TestCompareTotal checks Compare is a total order over the fixture: it is
// antisymmetric, transitive, and zero only for a finding and itself.
func TestCompareTotal(t *testing.T) {
	withTestCodes(t)
	fs := orderedFindings()
	for i, a := range fs {
		for j, b := range fs {
			c := Compare(a, b)
			want := 0
			if i < j {
				want = -1
			} else if i > j {
				want = 1
			}
			if c != want {
				t.Errorf("Compare(#%d, #%d) = %d, want %d", i, j, c, want)
			}
		}
	}
}

func TestComparePointers(t *testing.T) {
	// Tokens are decoded before they compare: ~1 is "/" and ~0 is "~". The
	// empty member name "/" is not an index.
	ordered := []string{"", "/0", "/1", "/9", "/10", "/10/0", "/10/a", "/100", "/", "/00", "/01", "/A", "/a", "/a/b", "/a~1", "/a~0", "/b", "/\u00e9"}
	for i, a := range ordered {
		for j, b := range ordered {
			got := comparePointers(a, b)
			want := 0
			if i < j {
				want = -1
			} else if i > j {
				want = 1
			}
			if got != want {
				t.Errorf("comparePointers(%q, %q) = %d, want %d", a, b, got, want)
			}
		}
	}
}

func TestPointer(t *testing.T) {
	tests := []struct {
		tokens []string
		want   string
	}{
		{nil, ""},
		{[]string{""}, "/"},
		{[]string{"a/b~c"}, "/a~1b~0c"},
		{[]string{"5", "remarks"}, "/5/remarks"},
		{[]string{"~1"}, "/~01"},
	}
	for _, tt := range tests {
		if got := Pointer(tt.tokens...); got != tt.want {
			t.Errorf("Pointer(%q) = %q, want %q", tt.tokens, got, tt.want)
		}
		if got := pointerTokens(tt.want); !slices.Equal(got, tt.tokens) && len(got)+len(tt.tokens) > 0 {
			t.Errorf("pointerTokens(%q) = %q, want %q", tt.want, got, tt.tokens)
		}
	}
}

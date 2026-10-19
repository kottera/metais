// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
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
		// 7. code: input.fixture_min_length before input.fixture_required before input.fixture_type.
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
	return fmt.Sprintf("%s %+v %+v %+v %s %s", f.Code(), f.subject, f.locationOrZero(), f.schemaOrZero(), f.canonicalParams(), f.Message())
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

func TestComparePaths(t *testing.T) {
	// Tokens are decoded before they compare: ~1 is "/" and ~0 is "~". The
	// empty member name "/" is not an index.
	ordered := []string{"", "/0", "/1", "/9", "/10", "/10/0", "/10/a", "/100", "/", "/00", "/01", "/A", "/a", "/a/b", "/a~1", "/a~0", "/b", "/\u00e9"}
	for i, a := range ordered {
		for j, b := range ordered {
			got := comparePaths(parsePath(a), parsePath(b))
			want := 0
			if i < j {
				want = -1
			} else if i > j {
				want = 1
			}
			if got != want {
				t.Errorf("comparePaths(%q, %q) = %d, want %d", a, b, got, want)
			}
		}
	}
}

// TestPathShared pins that a Path built a token at a time is the pointer
// of its tokens: it writes the same string, has the same digest as the
// same tokens built apart, and orders the same against a path that shares
// its nodes as against one that does not.
func TestPathShared(t *testing.T) {
	long := strings.Repeat("x", 1000)
	shared := parsePath("/a").Child(long)
	for _, tt := range []struct {
		p    *Path
		want string
	}{
		{nil, ""},
		{parsePath("/"), "/"},
		{shared.Child("b/c"), "/a/" + long + "/b~1c"},
		{shared.Child("~"), "/a/" + long + "/~0"},
		{parsePath("/a").Child("\xff"), "/a/\ufffd"},
	} {
		if got := tt.p.String(); got != tt.want {
			t.Errorf("String() = %q, want %q", got, tt.want)
		}
		if apart := parsePath(tt.want); apart.digest() != tt.p.digest() || comparePaths(apart, tt.p) != 0 {
			t.Errorf("%q built apart is not the same path", tt.want)
		}
	}
	if parsePath("").digest() == parsePath("/").digest() {
		t.Error("the root and the empty token share a digest")
	}
	b, c := shared.Child("10"), shared.Child("9")
	for _, pair := range [][2]*Path{{b, c}, {parsePath(b.String()), c}, {b, parsePath(c.String())}} {
		if got := comparePaths(pair[0], pair[1]); got != 1 {
			t.Errorf("comparePaths(%q, %q) = %d, want 1", pair[0], pair[1], got)
		}
		if got := comparePaths(pair[1], pair[0]); got != -1 {
			t.Errorf("comparePaths(%q, %q) = %d, want -1", pair[1], pair[0], got)
		}
	}
	if got := comparePaths(shared, b); got != -1 {
		t.Errorf("a prefix compares %d to its extension, want -1", got)
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

// TestWithPathIsWithPointer pins that a location given its pointer as a
// Path makes the finding a pointer string makes: the same members, the
// same encodings, the same place in the order and the same duplicate key,
// with its related locations' pointers written out alike.
func TestWithPathIsWithPointer(t *testing.T) {
	withTestCodes(t)
	subj := Subject{Kind: SubjectInput, Name: "vendor"}
	at := Location{Origin: OriginDataDir, File: "vendor.json", Line: 3, Column: 9}
	params := Params{"limit": Int(10), "length": Int(4)}
	for _, ptr := range []string{"", "/", "/a~1b/0/é", "/~0/" + strings.Repeat("x", 300)} {
		loc := at.WithPointer(ptr)
		byString := New(codeInputMinLength, subj, &loc, params, WithRelated(Related{Location: at.WithPointer(ptr), Note: "n"}))
		loc = at.WithPath(parsePath(ptr))
		byPath := New(codeInputMinLength, subj, &loc, params, WithRelated(Related{Location: at.WithPath(parsePath(ptr)), Note: "n"}))
		if a, b := string(AppendJSON(nil, byString)), string(AppendJSON(nil, byPath)); a != b {
			t.Errorf("%q: JSON\n%s\nand\n%s", ptr, a, b)
		}
		if a, b := string(AppendHuman(nil, false, byString)), string(AppendHuman(nil, false, byPath)); a != b {
			t.Errorf("%q: human\n%s\nand\n%s", ptr, a, b)
		}
		if Compare(byString, byPath) != 0 || byString.duplicateKey() != byPath.duplicateKey() {
			t.Errorf("%q: the two findings order or de-duplicate apart", ptr)
		}
		l, _ := byPath.Location()
		if want := at.WithPointer(ptr); l != want {
			t.Errorf("%q: Location() = %+v, want %+v", ptr, l, want)
		}
		if r := byPath.Related(); len(r) != 1 || r[0].Location != at.WithPointer(ptr) {
			t.Errorf("%q: Related() = %+v", ptr, r)
		}
	}
	noPtr := at
	noPtr.path = parsePath("/a")
	if f := New(codeInputMinLength, subj, &noPtr, params); Compare(f, New(codeInputMinLength, subj, &at, params)) != 0 {
		t.Error("a path on a location with no pointer is ordered")
	}
}

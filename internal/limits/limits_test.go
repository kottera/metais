// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package limits

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
)

// adr009 is ADR-009 D3's three tables as the ADR writes them, with
// the limits its amendments add: name, default and table, in order.
var adr009 = []struct{ name, def, class string }{
	{"package-size", "256 MiB", "package"},
	{"package-uncompressed", "512 MiB", "package"},
	{"entries", "10,000", "package"},
	{"entry-size", "64 MiB", "package"},
	{"entry-ratio", "200", "package"},
	{"path-length", "512 bytes", "package"},
	{"path-depth", "32 components", "package"},
	{"manifest-size", "1 MiB", "package"},
	{"schema-size", "4 MiB", "package"},
	{"schema-yaml-size", "1 MiB", "package"},
	{"combined-schema-size", "16 MiB", "package"},
	{"schema-depth", "64", "package"},
	{"schema-subschemas", "20,000", "package"},
	{"schema-location-length", "512 bytes", "package"},
	{"ref-depth", "32", "package"},
	{"pattern-length", "4,096 bytes", "package"},
	{"regex-program", "4,096 instructions", "package"},
	{"template-size", "8 MiB", "package"},
	{"template-nesting", "100", "package"},
	{"partial-depth", "32", "package"},
	{"value-depth", "64", "package"},
	{"number-length", "1,024 bytes", "package"},

	{"input-size", "32 MiB", "data"},
	{"inputs-total", "128 MiB", "data"},
	{"input-values", "500,000", "data"},
	{"yaml-size", "1 MiB", "data"},
	{"csv-rows", "1,000,000", "data"},
	{"csv-columns", "1,024", "data"},
	{"csv-cell", "1 MiB", "data"},
	{"collection-items", "100,000", "data"},
	{"frontmatter-size", "1 MiB", "data"},

	{"render-work", "2^32 units", "work"},
	{"loop-iterations", "10,000,000", "work"},
	{"value-size", "64 MiB", "work"},
	{"array-size", "1,000,000 elements", "work"},
	{"assign-bytes", "512 MiB", "work"},
	{"schema-work", "450,000", "work"},
	{"regex-work", "2^33 units", "work"},
	{"outputs", "100,000", "work"},
	{"container-depth", "3", "work"},
	{"output-size", "128 MiB", "work"},
	{"output-total", "256 MiB", "work"},
	{"output-values", "400,000", "work"},
	{"findings", "100,000", "work"},
	{"grep-matches", "10,000", "work"},
	{"grep-context", "7 lines", "work"},
}

// parseADR reads a default as ADR-009 writes it, and its unit: a size in
// bytes, KiB, MiB or GiB is bytes, a ratio is entry-ratio's, and anything
// else counts.
func parseADR(t *testing.T, name, s string) (int64, Unit) {
	t.Helper()
	num, word, _ := strings.Cut(s, " ")
	var n int64
	if b, e, ok := strings.Cut(num, "^"); ok {
		base, _ := strconv.ParseInt(b, 10, 64)
		exp, _ := strconv.Atoi(e)
		n = 1
		for range exp {
			n *= base
		}
	} else {
		var err error
		if n, err = strconv.ParseInt(strings.ReplaceAll(num, ",", ""), 10, 64); err != nil {
			t.Fatalf("%s: %q", name, s)
		}
	}
	switch word {
	case "KiB":
		return n << 10, Bytes
	case "MiB":
		return n << 20, Bytes
	case "GiB":
		return n << 30, Bytes
	case "bytes":
		return n, Bytes
	}
	if name == "entry-ratio" {
		return n, Ratio
	}
	return n, Count
}

// TestTable holds the table to ADR-009 D3 exactly: every limit, in order,
// with its name, default, unit and class (D3 item 2).
func TestTable(t *testing.T) {
	all := All()
	if len(all) != len(adr009) {
		t.Fatalf("%d limits, ADR-009 D3 has %d", len(all), len(adr009))
	}
	for i, want := range adr009 {
		got := all[i]
		def, unit := parseADR(t, want.name, want.def)
		if got.Name != want.name || got.Default != def || got.Unit != unit || got.Class.String() != want.class || got.ID != ID(i) {
			t.Errorf("limit %d is %s %d %s %s (ID %d), ADR-009 D3 has %s %d %s %s", i, got.Name, got.Default, got.Unit, got.Class, got.ID, want.name, def, unit, want.class)
		}
		if l, ok := Lookup(want.name); !ok || l != got {
			t.Errorf("Lookup(%q) = %+v, %v", want.name, l, ok)
		}
		if Get(ID(i)) != got {
			t.Errorf("Get(%d) = %+v", i, Get(ID(i)))
		}
		if d := Defaults(); d.Value(ID(i)) != def || d.Lowered(ID(i)) {
			t.Errorf("Defaults: %s is %d, lowered %v", want.name, d.Value(ID(i)), d.Lowered(ID(i)))
		}
	}
	if _, ok := Lookup("Findings"); ok {
		t.Error("Lookup is not exact about case")
	}
	all[0].Default = 1
	if Get(PackageSize).Default == 1 {
		t.Error("All returned the table itself")
	}
}

func TestLower(t *testing.T) {
	tests := []struct {
		arg  string
		id   ID
		want int64
	}{
		{"findings=3", Findings, 3},
		{"findings=0", Findings, 0},
		{"findings=100000", Findings, 100_000},
		{"entry-size=1MiB", EntrySize, 1 << 20},
		{"entry-size=64MiB", EntrySize, 64 << 20},
		{"path-length=2KiB" /* above 512 */, PathLength, -1},
		{"path-length=511", PathLength, 511},
		{"output-total=256MiB", OutputTotal, 256 << 20},
		{"output-total=1GiB" /* above 256 MiB */, OutputTotal, -1},
		{"regex-work=8589934592", RegexWork, 1 << 33},
		{"entry-ratio=100", EntryRatio, 100},
	}
	for _, tt := range tests {
		s := Defaults()
		f, ok := s.Lower(tt.arg)
		if tt.want < 0 {
			if ok {
				t.Errorf("Lower(%q) accepted a raise", tt.arg)
			}
			continue
		}
		if !ok {
			t.Errorf("Lower(%q) refused: %s", tt.arg, finding.AppendJSON(nil, f))
			continue
		}
		if got := s.Value(tt.id); got != tt.want {
			t.Errorf("Lower(%q): %s is %d, want %d", tt.arg, Get(tt.id).Name, got, tt.want)
		}
		if s.Lowered(tt.id) != (tt.want < Get(tt.id).Default) {
			t.Errorf("Lower(%q): Lowered = %v", tt.arg, s.Lowered(tt.id))
		}
		for _, l := range All() {
			if l.ID != tt.id && (s.Value(l.ID) != l.Default || s.Lowered(l.ID)) {
				t.Errorf("Lower(%q) changed %s", tt.arg, l.Name)
			}
		}
	}

	// A later lowering of one limit replaces an earlier, both within the
	// default.
	s := Defaults()
	for _, arg := range []string{"findings=5", "findings=50"} {
		if _, ok := s.Lower(arg); !ok {
			t.Fatalf("Lower(%q) refused", arg)
		}
	}
	if s.Value(Findings) != 50 {
		t.Errorf("findings is %d after 5 then 50, want 50", s.Value(Findings))
	}
}

// TestLowerRefuses is ADR-009 D4 item 2: a raise, an unknown name and a
// malformed value are each refused as limit.invalid, class usage, with a
// finding naming the default wherever the argument names a limit, and the
// set is left as it was.
func TestLowerRefuses(t *testing.T) {
	type want struct {
		reason string
		params string
	}
	findingsLimit := `"name":"findings","default":100000,"unit":"count"`
	tests := []struct {
		arg  string
		want want
	}{
		// A raise.
		{"findings=100001", want{ReasonAboveDefault, findingsLimit + `,"value":100001`}},
		{"entry-size=65MiB", want{ReasonAboveDefault, `"name":"entry-size","default":67108864,"unit":"bytes","value":68157440`}},
		{"output-total=2GiB", want{ReasonAboveDefault, `"name":"output-total","default":268435456,"unit":"bytes","value":2147483648`}},
		{"findings=9007199254740993", want{ReasonAboveDefault, findingsLimit}},
		{"findings=99999999999999999999999", want{ReasonAboveDefault, findingsLimit}},
		// Amendment A1's two, each one above its default.
		{"schema-subschemas=20001", want{ReasonAboveDefault, `"name":"schema-subschemas","default":20000,"unit":"count","value":20001`}},
		{"schema-location-length=513", want{ReasonAboveDefault, `"name":"schema-location-length","default":512,"unit":"bytes","value":513`}},
		{"schema-location-length=1KiB", want{ReasonAboveDefault, `"name":"schema-location-length","default":512,"unit":"bytes","value":1024`}},
		{"entry-size=9007199254740992GiB", want{ReasonAboveDefault, `"name":"entry-size","default":67108864,"unit":"bytes"`}},
		// An unknown name.
		{"nope=1", want{ReasonUnknownName, ""}},
		{"Findings=1", want{ReasonUnknownName, ""}},
		{"=1", want{ReasonUnknownName, ""}},
		{"", want{ReasonUnknownName, ""}},
		{" findings=1", want{ReasonUnknownName, ""}},
		// A malformed value.
		{"findings", want{ReasonMalformedValue, findingsLimit}},
		{"findings=", want{ReasonMalformedValue, findingsLimit}},
		{"findings=abc", want{ReasonMalformedValue, findingsLimit}},
		{"findings=-1", want{ReasonMalformedValue, findingsLimit}},
		{"findings=+1", want{ReasonMalformedValue, findingsLimit}},
		{"findings=03", want{ReasonMalformedValue, findingsLimit}},
		{"findings=1.5", want{ReasonMalformedValue, findingsLimit}},
		{"findings=1e3", want{ReasonMalformedValue, findingsLimit}},
		{"findings= 1", want{ReasonMalformedValue, findingsLimit}},
		{"findings=1 ", want{ReasonMalformedValue, findingsLimit}},
		{"findings=1_000", want{ReasonMalformedValue, findingsLimit}},
		{"findings=1=2", want{ReasonMalformedValue, findingsLimit}},
		{"findings=1KiB", want{ReasonMalformedValue, findingsLimit}},
		{"entry-size=1kib", want{ReasonMalformedValue, `"name":"entry-size","default":67108864,"unit":"bytes"`}},
		{"entry-size=MiB", want{ReasonMalformedValue, `"name":"entry-size","default":67108864,"unit":"bytes"`}},
		{"entry-size=1MiBMiB", want{ReasonMalformedValue, `"name":"entry-size","default":67108864,"unit":"bytes"`}},
		{"entry-size=1KB", want{ReasonMalformedValue, `"name":"entry-size","default":67108864,"unit":"bytes"`}},
		{"entry-ratio=1KiB", want{ReasonMalformedValue, `"name":"entry-ratio","default":200,"unit":"ratio"`}},
		{"findings=１", want{ReasonMalformedValue, findingsLimit}},
	}
	for _, tt := range tests {
		s := Defaults()
		f, ok := s.Lower(tt.arg)
		if ok {
			t.Errorf("Lower(%q) accepted it", tt.arg)
			continue
		}
		if f.Code() != finding.CodeLimitInvalid || f.Class() != finding.ClassUsage || f.Severity() != finding.SeverityError {
			t.Errorf("Lower(%q): %s, class %s, severity %s", tt.arg, f.Code(), f.Class(), f.Severity())
		}
		params := `"params":{"reason":"` + tt.want.reason + `"`
		if tt.want.params != "" {
			params += "," + tt.want.params
		}
		params += "}"
		if j := string(finding.AppendJSON(nil, f)); !strings.Contains(j, params) {
			t.Errorf("Lower(%q):\n got  %s\n want %s", tt.arg, j, params)
		}
		if s != Defaults() {
			t.Errorf("Lower(%q) changed the set", tt.arg)
		}
		r := finding.NewReport(10)
		r.Add(f)
		if got := r.Exit(finding.RoleConsumer, finding.SeverityError); got != finding.ExitUsage {
			t.Errorf("Lower(%q): exit %d, want %d", tt.arg, got, finding.ExitUsage)
		}
	}
}

// TestLowerEchoesNothingUnknown: an argument that names no limit puts
// nothing of itself in the finding.
func TestLowerEchoesNothingUnknown(t *testing.T) {
	s := Defaults()
	const sentinel = "SENTINEL7f3a"
	for _, arg := range []string{sentinel + "=1", sentinel, "findings" + sentinel + "=1"} {
		f, _ := s.Lower(arg)
		out := string(finding.AppendJSON(nil, f)) + string(finding.AppendHuman(nil, false, f))
		if strings.Contains(out, sentinel) {
			t.Errorf("Lower(%q) echoes the argument: %s", arg, out)
		}
	}
}

// TestCapUnderLoweredFindings is plan §7's acceptance: lowered with
// --limit findings=3, a report of ten errors lists three, adds
// report.truncated with omitted 7, and keeps the exact totals.
func TestCapUnderLoweredFindings(t *testing.T) {
	s := Defaults()
	if f, ok := s.Lower("findings=3"); !ok {
		t.Fatalf("findings=3 refused: %s", finding.AppendJSON(nil, f))
	}
	r := finding.NewReport(int(s.Value(Findings)))
	for i := range 10 {
		// Ten distinct errors: raises of findings by ten amounts.
		probe := Defaults()
		f, ok := probe.Lower(fmt.Sprintf("findings=%d", 100_001+i))
		if ok {
			t.Fatal("a raise was accepted")
		}
		r.Add(f)
		r.Add(f) // and a duplicate of each, counted once
	}
	fs := r.Findings()
	if len(fs) != 4 {
		t.Fatalf("%d findings, want 3 listed and report.truncated", len(fs))
	}
	for i, f := range fs[:3] {
		if v, _ := f.Param("value"); func() int64 { n, _ := v.Int(); return n }() != int64(100_001+i) {
			t.Errorf("finding %d is not the %d-th in order", i, i)
		}
	}
	if got, want := string(finding.AppendJSON(nil, fs[3])), `"params":{"listed":3,"omitted":7}`; !strings.Contains(got, want) {
		t.Errorf("last finding %s, want report.truncated with %s", got, want)
	}
	if got, want := string(r.Summary().AppendJSON(nil)), `{"errors":10,"warnings":0,"info":1,"listed":3,"omitted":7,"by_code":{"limit.invalid":10,"report.truncated":1}}`; got != want {
		t.Errorf("summary\n got  %s\n want %s", got, want)
	}
	if got := r.Exit(finding.RoleNoPackage, finding.SeverityError); got != finding.ExitUsage {
		t.Errorf("exit %d, want %d", got, finding.ExitUsage)
	}
}

func TestBadUse(t *testing.T) {
	for name, f := range map[string]func(){
		"Get(-1)":        func() { Get(-1) },
		"Get(count)":     func() { Get(count) },
		"zero Set Value": func() { var s Set; s.Value(Findings) },
		"zero Set Lower": func() { var s Set; s.Lower("findings=1") },
		"Value(count)":   func() { Defaults().Value(count) },
		"Lowered(-1)":    func() { Defaults().Lowered(-1) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s did not panic", name)
				}
			}()
			f()
		}()
	}
}

// TestBreached holds limit.package to ADR-009 D2 item 5: the limit's name,
// its bound in this command, whether it was lowered, and the value found
// when there is one.
func TestBreached(t *testing.T) {
	s := Defaults()
	loc := finding.Location{Origin: finding.OriginPackage, File: "assets/big.bin"}
	f := s.Breached(EntrySize, loc, 70<<20, true)
	if f.Code() != finding.CodeLimitPackage || f.Class() != finding.ClassRefused || f.Subject().Kind != finding.SubjectPackage {
		t.Errorf("Breached gave %s, class %s, about a %s", f.Code(), f.Class(), f.Subject().Kind)
	}
	want := map[string]string{"limit": "entry-size", "bound": "67108864", "lowered": "false", "observed": "73400320"}
	for name, w := range want {
		v, ok := f.Param(name)
		if got := paramText(v); !ok || got != w {
			t.Errorf("%s = %q, want %q", name, got, w)
		}
	}
	if l, _ := f.Location(); l != loc {
		t.Errorf("location %+v, want %+v", l, loc)
	}

	if _, ok := s.Lower("entries=3"); !ok {
		t.Fatal("Lower(entries=3) refused")
	}
	f = s.Breached(Entries, finding.Location{Origin: finding.OriginPackage}, 0, false)
	if v, _ := f.Param("lowered"); paramText(v) != "true" {
		t.Error("a lowered limit is not reported as lowered")
	}
	if v, _ := f.Param("bound"); paramText(v) != "3" {
		t.Error("the bound is not the lowered value")
	}
	if _, ok := f.Param("observed"); ok {
		t.Error("a reading that stopped at the bound reports an observed value")
	}
}

func paramText(v finding.Value) string {
	if n, ok := v.Int(); ok {
		return strconv.FormatInt(n, 10)
	}
	if s, ok := v.Str(); ok {
		return s
	}
	for _, b := range []bool{false, true} {
		if reflect.DeepEqual(v, finding.Bool(b)) {
			return strconv.FormatBool(b)
		}
	}
	return "?"
}

// TestBreachFindings holds Breached and Exceeded alike to ADR-009 D2 item 5: the
// limit's name, its bound in the Set, whether it was lowered, and the
// observed value only when there is one.
func TestBreachFindings(t *testing.T) {
	s := Defaults()
	if f, ok := s.Lower("value-depth=8"); !ok {
		t.Fatal(f.Message())
	}
	loc := finding.Location{Origin: finding.OriginPackage, File: "metais.yaml", Line: 3, Column: 5}
	input := finding.Subject{Kind: finding.SubjectInput, Name: "vendor"}
	for _, tc := range []struct {
		f           finding.Finding
		code        finding.Code
		subj        finding.SubjectKind
		name        string
		bound       int64
		lowered     bool
		observed    int64
		hasObserved bool
	}{
		{s.Breached(ValueDepth, loc, 9, true), finding.CodeLimitPackage, finding.SubjectPackage, "value-depth", 8, true, 9, true},
		{s.Breached(ManifestSize, loc, 0, false), finding.CodeLimitPackage, finding.SubjectPackage, "manifest-size", 1 << 20, false, 0, false},
		{s.Exceeded(InputSize, input, loc, 1<<26, true), finding.CodeLimitExceeded, finding.SubjectInput, "input-size", 32 << 20, false, 1 << 26, true},
		{s.Exceeded(ValueDepth, input, loc, 0, false), finding.CodeLimitExceeded, finding.SubjectInput, "value-depth", 8, true, 0, false},
	} {
		if tc.f.Code() != tc.code || tc.f.Subject().Kind != tc.subj {
			t.Errorf("%s: code %s about a %s, want %s about a %s", tc.name, tc.f.Code(), tc.f.Subject().Kind, tc.code, tc.subj)
		}
		if v, _ := tc.f.Param("limit"); !reflect.DeepEqual(v, finding.String(tc.name)) {
			t.Errorf("%s: limit %v", tc.name, v)
		}
		if v, _ := tc.f.Param("bound"); !reflect.DeepEqual(v, finding.Int(tc.bound)) {
			t.Errorf("%s: bound %v, want %d", tc.name, v, tc.bound)
		}
		if v, _ := tc.f.Param("lowered"); !reflect.DeepEqual(v, finding.Bool(tc.lowered)) {
			t.Errorf("%s: lowered %v, want %v", tc.name, v, tc.lowered)
		}
		v, ok := tc.f.Param("observed")
		if ok != tc.hasObserved || (ok && !reflect.DeepEqual(v, finding.Int(tc.observed))) {
			t.Errorf("%s: observed %v %v, want %d %v", tc.name, v, ok, tc.observed, tc.hasObserved)
		}
		if l, ok := tc.f.Location(); !ok || l.Line != 3 || l.File != "metais.yaml" {
			t.Errorf("%s: location %+v", tc.name, l)
		}
	}
}

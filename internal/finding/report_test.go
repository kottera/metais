// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
)

// reference is ADR-006 D8 computed the slow way, holding everything: keep
// the one of each set of duplicates whose keyword_location sorts first
// (then the lower digest of its encoding), sort, and cut at the cap.
func reference(fs []Finding, max int) (listed []Finding, total int) {
	best := map[[sha256.Size]byte]Finding{}
	var keys [][sha256.Size]byte
	for _, f := range fs {
		k := f.duplicateKey()
		old, ok := best[k]
		if !ok {
			keys = append(keys, k)
			best[k] = f
			continue
		}
		if f.keywordLocation() < old.keywordLocation() {
			best[k] = f
		} else if f.keywordLocation() == old.keywordLocation() {
			a, b := sha256.Sum256(AppendJSON(nil, f)), sha256.Sum256(AppendJSON(nil, old))
			if bytes.Compare(a[:], b[:]) < 0 {
				best[k] = f
			}
		}
	}
	for _, k := range keys {
		listed = append(listed, best[k])
	}
	slices.SortFunc(listed, Compare)
	total = len(listed)
	if len(listed) > max {
		listed = listed[:max]
	}
	return listed, total
}

// randomFindings draws n findings from a small space, so duplicates are
// common, and duplicates differ in schema pointer and keyword location,
// which is what moves a kept finding within the order.
func randomFindings(rng *rand.Rand, n int) []Finding {
	out := make([]Finding, n)
	for i := range out {
		ptr := Pointer(fmt.Sprint(rng.IntN(3)), []string{"a", "b"}[rng.IntN(2)])
		sch := Schema{
			Entry:           "inputs/x.schema.json",
			Pointer:         fmt.Sprintf("/p%d/minLength", rng.IntN(4)),
			Keyword:         "minLength",
			KeywordLocation: fmt.Sprintf("/k%d/minLength", rng.IntN(4)),
		}
		name := []string{"a", "b"}[rng.IntN(2)]
		out[i] = New(codeInputMinLength, input(name), dataDir("x.csv", 0, 0, ptr),
			Params{"limit": Int(10), "length": Int(int64(rng.IntN(2)))}, WithSchema(sch))
	}
	return out
}

// TestReportMatchesReference holds the bounded report to the reference
// over many random streams and caps: the same findings listed, the same
// totals, whatever order duplicates arrive in.
func TestReportMatchesReference(t *testing.T) {
	withTestCodes(t)
	rng := rand.New(rand.NewPCG(3, 4))
	for trial := range 3000 {
		fs := randomFindings(rng, 1+rng.IntN(40))
		max := rng.IntN(8)
		want, total := reference(fs, max)
		r := NewReport(max)
		for _, f := range fs {
			r.Add(f)
		}
		got := r.Findings()
		if total > max {
			if n := len(got); n == 0 || got[n-1].Code() != CodeReportTruncated {
				t.Fatalf("trial %d: %d findings over a cap of %d and no report.truncated last", trial, total, max)
			}
			got = got[:len(got)-1]
		}
		if d := outOfOrder(got, want); d != "" {
			t.Fatalf("trial %d (cap %d, %d added): %s", trial, max, len(fs), d)
		}
		s := r.Summary()
		if s.Listed != len(want) || s.Omitted != total-len(want) || s.Errors != total {
			t.Fatalf("trial %d: summary %+v, want listed %d omitted %d errors %d", trial, s, len(want), total-len(want), total)
		}
		if r.listed.Len() > max {
			t.Fatalf("trial %d: %d findings held listed over a cap of %d", trial, r.listed.Len(), max)
		}
	}
}

// TestReportCap is ADR-006's cap vector in miniature: ten errors under a
// cap of three list three, then report.truncated with listed 3 and omitted
// 7, and the summary counts all ten.
func TestReportCap(t *testing.T) {
	withTestCodes(t)
	r := NewReport(3)
	for i := range 10 {
		r.Add(minLength("alpha", "a.csv", 10-i, Pointer(fmt.Sprint(9-i), "remarks"), 4))
	}
	fs := r.Findings()
	if len(fs) != 4 {
		t.Fatalf("got %d findings, want 3 and report.truncated", len(fs))
	}
	for i, f := range fs[:3] {
		if l, _ := f.Location(); l.Line != i+1 {
			t.Errorf("finding %d is at line %d, want %d: the first three in order are kept", i, l.Line, i+1)
		}
	}
	tr := fs[3]
	listed, _ := tr.Param("listed")
	omitted, _ := tr.Param("omitted")
	if tr.Code() != CodeReportTruncated || tr.Severity() != SeverityInfo || tr.Subject().Kind != SubjectReport {
		t.Errorf("last finding is %s %s about %s, want report.truncated info about report", tr.Code(), tr.Severity(), tr.Subject().Kind)
	}
	if n, _ := listed.Int(); n != 3 {
		t.Errorf("listed = %d, want 3", n)
	}
	if n, _ := omitted.Int(); n != 7 {
		t.Errorf("omitted = %d, want 7", n)
	}
	got := string(r.Summary().AppendJSON(nil))
	want := `{"errors":10,"warnings":0,"info":1,"listed":3,"omitted":7,"by_code":{"input.min_length":10,"report.truncated":1}}`
	if got != want {
		t.Errorf("summary\n got  %s\n want %s", got, want)
	}
	if _, ok := tr.Location(); ok {
		t.Error("report.truncated has a location")
	}
}

func TestReportNoTruncation(t *testing.T) {
	withTestCodes(t)
	for _, max := range []int{2, 3} {
		r := NewReport(max)
		r.Add(minLength("a", "a.csv", 1, "/0", 4))
		r.Add(minLength("a", "a.csv", 2, "/1", 4))
		if fs := r.Findings(); len(fs) != 2 || fs[1].Code() == CodeReportTruncated {
			t.Errorf("cap %d: got %d findings, want the 2 added and no truncation", max, len(fs))
		}
		if s := r.Summary(); s.Omitted != 0 || s.Info != 0 || s.Listed != 2 {
			t.Errorf("cap %d: summary %+v", max, s)
		}
	}
	r := NewReport(0)
	r.Add(minLength("a", "a.csv", 1, "/0", 4))
	if fs := r.Findings(); len(fs) != 1 || fs[0].Code() != CodeReportTruncated {
		t.Errorf("cap 0: got %v, want report.truncated alone", describeAll(fs))
	}
	if s := NewReport(5).Summary(); s.Errors+s.Warnings+s.Info+s.Listed+s.Omitted != 0 || len(s.ByCode) != 0 {
		t.Errorf("empty report summary %+v", s)
	}
}

// TestReportDuplicates is ADR-006 D8 item 3: duplicates are reported once,
// the one with the first keyword_location kept, and a finding that differs
// in a member of the duplicate key is not a duplicate.
func TestReportDuplicates(t *testing.T) {
	withTestCodes(t)
	req := func(kl string, opts ...Option) Finding {
		loc := Location{Origin: OriginDataDir, File: "v.yaml", Line: 1, Column: 1}.WithPointer("/vendor/name")
		return New(codeInputRequired, input("vendor"), &loc, Params{"property": String("name")},
			append([]Option{WithSchema(Schema{Entry: "inputs/vendor.schema.json", Pointer: "/required", Keyword: "required", KeywordLocation: kl})}, opts...)...)
	}
	r := NewReport(10)
	r.Add(req("/allOf/1/$ref/required"))
	r.Add(req("/allOf/0/$ref/required", WithVia(Via{Keyword: "then", SchemaPointer: "/then", Condition: &Condition{SchemaPointer: "/if"}})))
	r.Add(req("/allOf/2/$ref/required"))
	fs := r.Findings()
	if len(fs) != 1 {
		t.Fatalf("got %d findings, want the duplicates once", len(fs))
	}
	if s, _ := fs[0].Schema(); s.KeywordLocation != "/allOf/0/$ref/required" {
		t.Errorf("kept keyword_location %q, want the first in order", s.KeywordLocation)
	}
	if s := r.Summary(); s.Errors != 1 {
		t.Errorf("errors = %d, want 1", s.Errors)
	}

	// Not duplicates: another origin, file, argument, pointer, subject or
	// param.
	base := Location{Origin: OriginDataDir, File: "v.yaml"}.WithPointer("/a")
	variants := []Location{base}
	l := base
	l.Origin = OriginExample
	variants = append(variants, l)
	l = base
	l.File = "w.yaml"
	variants = append(variants, l)
	l = base
	l.Argument = 2
	variants = append(variants, l)
	variants = append(variants, base.WithPointer("/b"))
	l = base
	l.HasPointer, l.Pointer = false, ""
	variants = append(variants, l)
	r = NewReport(100)
	for _, v := range variants {
		r.Add(New(codeInputRequired, input("v"), &v, Params{"property": String("a")}))
	}
	r.Add(New(codeInputRequired, input("w"), &base, Params{"property": String("a")}))
	r.Add(New(codeInputRequired, Subject{Kind: SubjectInput, Name: "v", Title: "other"}, &base, Params{"property": String("a")}))
	r.Add(New(codeInputRequired, input("v"), &base, Params{"property": String("b")}))
	if got, want := len(r.Findings()), len(variants)+3; got != want {
		t.Errorf("got %d findings, want %d distinct ones", got, want)
	}
}

// TestReportSameBytes is ADR-006 D10's promise: the same findings give the
// same bytes, in JSON and for a person, whatever order they were added in.
func TestReportSameBytes(t *testing.T) {
	withTestCodes(t)
	all := append(orderedFindings(), randomFindings(rand.New(rand.NewPCG(5, 6)), 60)...)
	encode := func(in []Finding, max int) string {
		r := NewReport(max)
		for _, f := range in {
			r.Add(f)
		}
		fs := r.Findings()
		return string(AppendJSON(nil, fs...)) + string(r.Summary().AppendJSON(nil)) +
			string(AppendHuman(nil, true, fs...)) + string(r.Summary().AppendHuman(nil))
	}
	rng := rand.New(rand.NewPCG(7, 8))
	for _, max := range []int{5, 1000} {
		want := encode(all, max)
		for range 200 {
			in := slices.Clone(all)
			rng.Shuffle(len(in), func(a, b int) { in[a], in[b] = in[b], in[a] })
			if got := encode(in, max); got != want {
				t.Fatalf("cap %d: a reordering changed the bytes", max)
			}
		}
	}
}

// TestReportBounded holds the report to its cap while it runs: a stream
// of distinct findings far larger than the cap never holds more than the
// cap.
func TestReportBounded(t *testing.T) {
	withTestCodes(t)
	r := NewReport(10)
	for i := range 20000 {
		r.Add(minLength("a", "a.csv", 1+i%997, Pointer(fmt.Sprint(i)), 4))
		if r.listed.Len() > 10 || r.spill.Len() > 1 {
			t.Fatalf("after %d findings: %d listed, %d spilled", i+1, r.listed.Len(), r.spill.Len())
		}
	}
	if s := r.Summary(); s.Errors != 20000 || s.Listed != 10 || s.Omitted != 19990 {
		t.Errorf("summary %+v", s)
	}
}

// TestReportRefusesMovedDuplicate: duplicates are at one place, since a
// value's position is fixed by its file and pointer; two that are not
// show a defect in what positioned them.
func TestReportRefusesMovedDuplicate(t *testing.T) {
	withTestCodes(t)
	r := NewReport(5)
	r.Add(minLength("a", "a.csv", 2, "/0", 4))
	defer func() {
		if recover() == nil {
			t.Error("a duplicate at another line was accepted")
		}
	}()
	r.Add(minLength("a", "a.csv", 3, "/0", 4))
}

func TestReportAddRefusesZeroFinding(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Add of a zero Finding did not panic")
		}
	}()
	NewReport(1).Add(Finding{})
}

func TestExit(t *testing.T) {
	withTestCodes(t)
	loc := Location{Origin: OriginPackage, File: "f"}.WithPointer("")
	usage := New(codeUsageFlag, Subject{Kind: SubjectInvocation}, nil, nil)
	internal := New(codeInternalError, Subject{Kind: SubjectInvocation}, nil, nil)
	refused := New(codePackageName, Subject{Kind: SubjectPackage}, &loc, nil)
	inputErr := minLength("a", "a.csv", 1, "/0", 4)
	pkgContent := New(codeTemplateParse, Subject{Kind: SubjectTemplate, Name: "t"}, &loc, nil)
	manifest := New(codeManifestField, Subject{Kind: SubjectManifest}, &loc, Params{"member": String("x")})
	outputPath := New(codeOutputPath, Subject{Kind: SubjectOutput, Name: "o"}, &loc, nil)
	dest := New(codeDestNonEmpty, Subject{Kind: SubjectOutput, Name: "o"}, &loc, nil)
	warning := New(codeWarnUnused, Subject{Kind: SubjectTemplate, Name: "t"}, &loc, nil)
	tests := []struct {
		name   string
		fs     []Finding
		role   Role
		failOn Severity
		want   int
	}{
		{"nothing", nil, RoleConsumer, SeverityError, 0},
		{"a warning", []Finding{warning}, RoleConsumer, SeverityError, 0},
		{"a warning under --fail-on warning", []Finding{warning}, RoleConsumer, SeverityWarning, 1},
		{"an input error", []Finding{inputErr}, RoleConsumer, SeverityError, 1},
		{"an output path, content", []Finding{outputPath}, RoleConsumer, SeverityError, 1},
		{"internal", []Finding{internal}, RoleConsumer, SeverityError, 2},
		{"usage", []Finding{usage}, RoleNoPackage, SeverityError, 3},
		{"refused, consumer", []Finding{refused}, RoleConsumer, SeverityError, 4},
		{"refused, authoring", []Finding{refused}, RoleAuthoring, SeverityError, 4},
		{"destination", []Finding{dest}, RoleConsumer, SeverityError, 5},
		// D12 item 2: the package's own content error.
		{"template error, consumer", []Finding{pkgContent}, RoleConsumer, SeverityError, 4},
		{"template error, authoring", []Finding{pkgContent}, RoleAuthoring, SeverityError, 1},
		{"manifest defect under validate", []Finding{manifest}, RoleConsumer, SeverityError, 4},
		{"manifest defect under check", []Finding{manifest}, RoleAuthoring, SeverityError, 1},
		// D12 item 3: 2, 3, 4, 1, 5.
		{"internal over usage", []Finding{usage, internal}, RoleConsumer, SeverityError, 2},
		{"usage over refused", []Finding{refused, usage}, RoleConsumer, SeverityError, 3},
		{"refused over content", []Finding{inputErr, refused}, RoleConsumer, SeverityError, 4},
		{"content over destination", []Finding{dest, inputErr}, RoleConsumer, SeverityError, 1},
		{"warning under --fail-on warning over destination", []Finding{dest, warning}, RoleConsumer, SeverityWarning, 1},
		{"package content under authoring over destination", []Finding{dest, pkgContent}, RoleAuthoring, SeverityError, 1},
		{"package content under consumer over input error", []Finding{inputErr, pkgContent}, RoleConsumer, SeverityError, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A cap of zero lists nothing: the status counts what is not
			// listed (ADR-009 D2 item 6).
			r := NewReport(0)
			for _, f := range tt.fs {
				r.Add(f)
			}
			if got := r.Exit(tt.role, tt.failOn); got != tt.want {
				t.Errorf("Exit = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestExitInfoNeverFails(t *testing.T) {
	r := NewReport(0)
	r.Add(New(CodeLimitInvalid, Subject{Kind: SubjectInvocation}, nil, Params{"reason": String("unknown_name")}))
	r2 := NewReport(0)
	for _, f := range r.Findings() {
		if f.Code() == CodeReportTruncated {
			r2.Add(f)
		}
	}
	for _, failOn := range []Severity{SeverityError, SeverityWarning} {
		if got := r2.Exit(RoleConsumer, failOn); got != 0 {
			t.Errorf("a report holding only report.truncated exits %d under fail-on %s", got, failOn)
		}
	}
	if got := r.Exit(RoleNoPackage, SeverityError); got != ExitUsage {
		t.Errorf("an unlisted usage error exits %d, want 3", got)
	}
}

func TestExitRefusesBadArguments(t *testing.T) {
	for _, call := range []func(){
		func() { NewReport(1).Exit(0, SeverityError) },
		func() { NewReport(1).Exit(RoleConsumer, SeverityInfo) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("Exit accepted a bad argument")
				}
			}()
			call()
		}()
	}
}

// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"bytes"
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
)

// nestedFinding is a finding at ptr whose alternatives hold the findings
// of each branch of branches.
func nestedFinding(ptr string, branches ...[]Finding) Finding {
	alts := make([]Alternative, len(branches))
	for b, fs := range branches {
		alts[b] = Alternative{Branch: b, Findings: fs}
	}
	return New(codeInputMinLength, input("a"), dataDir("x.csv", 0, 0, ptr), Params{"limit": Int(10), "length": Int(1)}, WithAlternatives(alts...))
}

// leaf is a finding with no alternatives, at ptr.
func leaf(ptr string) Finding {
	return New(codeInputMinLength, input("a"), dataDir("x.csv", 0, 0, ptr), Params{"limit": Int(10), "length": Int(0)})
}

// shape is what f lists, as text: each finding's pointer, then its
// branches in brackets, each with what it omits after a slash.
func shape(f Finding) string {
	l, _ := f.Location()
	s := l.Pointer
	for _, a := range f.alternatives {
		s += "["
		for i, g := range a.Findings {
			if i > 0 {
				s += " "
			}
			s += shape(g)
		}
		s += fmt.Sprintf("/%d]", a.Omitted)
	}
	return s
}

// tree is the finding the fnd/nested-findings-cap vector reports about v:
// two branches, each a choice of two branches of one finding.
func tree() Finding {
	return nestedFinding("/t",
		[]Finding{nestedFinding("/u", []Finding{leaf("/u0")}, []Finding{leaf("/u1")})},
		[]Finding{nestedFinding("/v", []Finding{leaf("/v0")}, []Finding{leaf("/v1")})},
	)
}

// TestListAtMost is MTS-FND-020 and MTS-FND-021 over one finding: what it
// lists is counted in the order a report lists it, itself first, and what
// a bound leaves out is counted in its branch's omitted.
func TestListAtMost(t *testing.T) {
	withTestCodes(t)
	f := tree()
	if f.listed != 7 {
		t.Fatalf("the finding lists %d, want 7: itself, two choices and four leaves", f.listed)
	}
	for n, want := range map[int]string{
		1: "/t[/1][/1]",
		2: "/t[/u[/1][/1]/0][/1]",
		3: "/t[/u[/u0/0][/1]/0][/1]",
		4: "/t[/u[/u0/0][/u1/0]/0][/1]",
		5: "/t[/u[/u0/0][/u1/0]/0][/v[/1][/1]/0]",
		6: "/t[/u[/u0/0][/u1/0]/0][/v[/v0/0][/1]/0]",
		7: "/t[/u[/u0/0][/u1/0]/0][/v[/v0/0][/v1/0]/0]",
		8: "/t[/u[/u0/0][/u1/0]/0][/v[/v0/0][/v1/0]/0]",
	} {
		g := f.listAtMost(n)
		if got := shape(g); got != want {
			t.Errorf("at most %d: lists %s, want %s", n, got, want)
		}
		if g.listed != min(n, 7) || g.countListed() != g.listed {
			t.Errorf("at most %d: counts %d listed, and lists %d, want %d", n, g.listed, g.countListed(), min(n, 7))
		}
		// New under the same bound builds the same finding.
		if b := New(f.entry.Code, f.subject, f.location, f.params, WithAlternatives(f.alternatives...), WithListedAtMost(n)); shape(b) != want {
			t.Errorf("New at most %d: lists %s, want %s", n, shape(b), want)
		}
	}
	if got := shape(f); got != "/t[/u[/u0/0][/u1/0]/0][/v[/v0/0][/v1/0]/0]" {
		t.Errorf("listAtMost changed the finding it was given: %s", got)
	}
}

// TestReportCountsNested is MTS-FND-020 and MTS-FND-021 in a report: the
// findings within alternatives count toward the cap, the finding the cap
// is reached in is cut short, and no later finding is listed.
func TestReportCountsNested(t *testing.T) {
	withTestCodes(t)
	for _, tc := range []struct {
		max    int
		want   []string
		listed int
	}{
		{0, nil, 0},
		{1, []string{"/t[/1][/1]"}, 1},
		{5, []string{"/t[/u[/u0/0][/u1/0]/0][/v[/1][/1]/0]"}, 1},
		{7, []string{"/t[/u[/u0/0][/u1/0]/0][/v[/v0/0][/v1/0]/0]"}, 1},
		{8, []string{"/t[/u[/u0/0][/u1/0]/0][/v[/v0/0][/v1/0]/0]", "/w"}, 2},
	} {
		r := NewReport(tc.max)
		r.Add(leaf("/w"))
		r.Add(tree())
		fs := r.Findings()
		var got []string
		for _, f := range fs {
			if f.Code() != CodeReportTruncated {
				got = append(got, shape(f))
			}
		}
		if fmt.Sprint(got) != fmt.Sprint(tc.want) {
			t.Errorf("cap %d: lists %q, want %q", tc.max, got, tc.want)
		}
		if s := r.Summary(); s.Listed != tc.listed || s.Omitted != 2-tc.listed {
			t.Errorf("cap %d: summary lists %d and omits %d, want %d and %d", tc.max, s.Listed, s.Omitted, tc.listed, 2-tc.listed)
		}
	}
}

// weightedFindings draws n findings as randomFindings does, each with up
// to two branches of up to three findings, so a finding lists from one to
// seven.
func weightedFindings(rng *rand.Rand, n int) []Finding {
	out := randomFindings(rng, n)
	for i, f := range out {
		branches := make([][]Finding, rng.IntN(3))
		for b := range branches {
			for range rng.IntN(4) {
				branches[b] = append(branches[b], leaf(fmt.Sprintf("/l%d", rng.IntN(5))))
			}
		}
		alts := make([]Alternative, len(branches))
		for b, fs := range branches {
			alts[b] = Alternative{Branch: b, Findings: fs}
		}
		s, _ := f.Schema()
		out[i] = New(f.entry.Code, f.subject, f.location, f.params, WithSchema(s), WithAlternatives(alts...))
	}
	return out
}

// TestReportNestedMatchesReference holds the report to the reference with
// findings that list others: the distinct findings in D8's order, each
// listed while the findings before it leave the cap room, the last cut
// short to what is left.
func TestReportNestedMatchesReference(t *testing.T) {
	withTestCodes(t)
	rng := rand.New(rand.NewPCG(5, 6))
	for trial := range 3000 {
		fs := weightedFindings(rng, 1+rng.IntN(30))
		max := rng.IntN(24)
		all, total := reference(fs, math.MaxInt)
		var want []Finding
		start := 0
		for _, f := range all {
			if start >= max {
				break
			}
			f = f.listAtMost(max - start)
			start += f.listed
			want = append(want, f)
		}
		r := NewReport(max)
		for _, f := range fs {
			r.Add(f)
		}
		got := r.Findings()
		if len(want) < total {
			got = got[:len(got)-1]
		}
		if !bytes.Equal(AppendJSON(nil, got...), AppendJSON(nil, want...)) {
			t.Fatalf("trial %d (cap %d, %d added):\n got  %s\n want %s", trial, max, len(fs), AppendJSON(nil, got...), AppendJSON(nil, want...))
		}
		if s := r.Summary(); s.Listed != len(want) || s.Omitted != total-len(want) {
			t.Fatalf("trial %d: summary %+v, want listed %d omitted %d", trial, s, len(want), total-len(want))
		}
		if r.listed.Len() > max {
			t.Fatalf("trial %d: %d findings held listed over a cap of %d", trial, r.listed.Len(), max)
		}
	}
}

// TestBranchKeepsWhatItLists is a branch gathered as New keeps it: sorted,
// de-duplicated, its first maxBranchFindings held and the rest counted.
func TestBranchKeepsWhatItLists(t *testing.T) {
	withTestCodes(t)
	b := NewBranch(1, "One")
	for i := range 40 {
		b.Add(leaf(fmt.Sprintf("/n%02d", 39-i)))
		b.Add(leaf(fmt.Sprintf("/n%02d", 39-i)))
	}
	a := b.Alternative()
	if a.Branch != 1 || a.Title != "One" || len(a.Findings) != maxBranchFindings || a.Omitted != 40-maxBranchFindings {
		t.Fatalf("the branch lists %d and omits %d, want %d and %d", len(a.Findings), a.Omitted, maxBranchFindings, 40-maxBranchFindings)
	}
	if l, _ := a.Findings[0].Location(); l.Pointer != "/n00" {
		t.Errorf("the branch lists %s first, want /n00", l.Pointer)
	}
	if b.r.listed.Len() > maxBranchFindings {
		t.Errorf("the branch holds %d findings, more than it lists", b.r.listed.Len())
	}
}

// TestEncodeFlushes holds EncodeJSON and EncodeHuman to AppendJSON's and
// AppendHuman's bytes, and to handing them over a finding at a time: a
// finding whose alternatives list many is written without being held as
// one encoding.
func TestEncodeFlushes(t *testing.T) {
	withTestCodes(t)
	f := tree()
	for _, tc := range []struct {
		name   string
		want   []byte
		encode func(b []byte, flush func([]byte) []byte) []byte
	}{
		{"json", AppendJSON(nil, f)[1 : len(AppendJSON(nil, f))-1], func(b []byte, flush func([]byte) []byte) []byte { return EncodeJSON(b, f, flush) }},
		{"human", AppendHuman(nil, false, f), func(b []byte, flush func([]byte) []byte) []byte { return EncodeHuman(b, false, f, flush) }},
	} {
		var got []byte
		chunks := 0
		flush := func(b []byte) []byte {
			got = append(got, b...)
			chunks++
			return b[:0]
		}
		flush(tc.encode(nil, flush))
		if !bytes.Equal(got, tc.want) {
			t.Errorf("%s: written a piece at a time\n got  %s\n want %s", tc.name, got, tc.want)
		}
		// One piece before each of the six findings within, and the last.
		if chunks != 7 {
			t.Errorf("%s: written in %d pieces, want 7", tc.name, chunks)
		}
	}
}

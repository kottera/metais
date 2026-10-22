// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"strings"
	"testing"
	"unicode/utf8"

	"metais.org/metais/internal/fault"
)

// TestMessagesWithinTheBound is ADR-006 D9 items 1 and 2: a message and a
// hint are at most 240 code points however long the pointer {field}
// names or a parameter it echoes, and the finding's members keep the
// whole of each.
func TestMessagesWithinTheBound(t *testing.T) {
	long := "/" + strings.Repeat("member/", 200) + "remarks"
	loc := Location{Origin: OriginDataDir, File: "a.json"}.WithPointer(long)
	f := New("input.min_length", input("a"), &loc, Params{"limit": Int(10), "length": Int(4)})
	msg := f.Message()
	if n := utf8.RuneCountInString(msg); n > maxText {
		t.Fatalf("the message has %d code points: %q", n, msg)
	}
	if !strings.HasPrefix(msg, "`…") || !strings.Contains(msg, "/member/remarks` must be at least 10 characters long.") {
		t.Errorf("the message does not keep the pointer's end: %q", msg)
	}
	if l, _ := f.Location(); l.Pointer != long {
		t.Errorf("the location's pointer is %d bytes, want the whole %d", len(l.Pointer), len(long))
	}
	if !strings.Contains(string(AppendJSON(nil, f)), `"pointer":"`+long+`"`) {
		t.Error("the JSON finding does not carry the whole pointer")
	}

	// A member name the supplier wrote, echoed twice: each is cut to its
	// share, and the parameter keeps it whole.
	name := strings.Repeat("ü", 300)
	at := Location{Origin: OriginDataDir, File: "a.json"}.WithPointer("")
	g := New("input.additional_property", input("a"), &at, Params{"property": String(name)})
	for what, text := range map[string]string{"message": g.Message(), "hint": g.Hint()} {
		if n := utf8.RuneCountInString(text); n > maxText {
			t.Errorf("the %s has %d code points: %q", what, n, text)
		}
		if !strings.Contains(text, "…") {
			t.Errorf("the %s marks no cut: %q", what, text)
		}
	}
	if v, _ := g.Param("property"); v.text != name {
		t.Error("the property parameter was shortened")
	}

	// A message within the bound is written as it is.
	short := Location{Origin: OriginDataDir, File: "a.json"}.WithPointer("/remarks")
	if m := New("input.min_length", input("a"), &short, Params{"limit": Int(10), "length": Int(4)}).Message(); m != "`/remarks` must be at least 10 characters long." {
		t.Errorf("a short message is %q", m)
	}
}

// TestFairShare holds the shares to what the literal text leaves: a short
// value gives what it does not use to the long ones.
func TestFairShare(t *testing.T) {
	for _, c := range []struct {
		lens   []int
		budget int
		want   int
	}{
		{[]int{10, 500}, 100, 90},
		{[]int{500, 500}, 100, 50},
		{[]int{1, 2}, 100, int(^uint(0) >> 1)},
		{[]int{500}, -5, 0},
	} {
		if got := fairShare(c.lens, c.budget); got != c.want {
			t.Errorf("fairShare(%v, %d) = %d, want %d", c.lens, c.budget, got, c.want)
		}
	}
	if got := shorten("`/a/b/c/d`", 6, true); got != "`…c/d`" {
		t.Errorf("shorten of a pointer = %q", got)
	}
	if got := shorten("abcdef", 4, false); got != "abc…" {
		t.Errorf("shorten = %q", got)
	}
}

// TestFirstFindingFault is the fault.FirstFinding seam: under it a report
// keeps the first error it is given and drops every later one, and
// without it keeps them all.
func TestFirstFindingFault(t *testing.T) {
	add := func(r *Report) {
		for _, p := range []string{"/a", "/b", "/c"} {
			l := Location{Origin: OriginDataDir, File: "a.json"}.WithPointer(p)
			r.Add(New("input.min_length", input("a"), &l, Params{"limit": Int(10), "length": Int(4)}))
		}
		r.Add(New(CodeReportTruncated, Subject{Kind: SubjectReport}, nil, Params{"listed": Int(1), "omitted": Int(1)}))
	}
	r := NewReportFaults(100, 0)
	add(r)
	if s := r.Summary(); s.Errors != 3 || s.Info != 1 {
		t.Errorf("without the fault: %+v", s)
	}
	r = NewReportFaults(100, fault.FirstFinding)
	add(r)
	if s := r.Summary(); s.Errors != 1 || s.Info != 1 {
		t.Errorf("with FirstFinding: %+v, want one error and the info kept", s)
	}
}

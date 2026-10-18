// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"bytes"
	"container/heap"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
)

// Report gathers one command's findings (ADR-006 D8): it removes
// duplicates, keeps the first maxFindings in D8's order, and counts every
// finding exactly, listed or not.
//
// It holds the listed findings, at most the cap, in a bounded heap. Beside
// them it keeps, per distinct finding, the SHA-256 digest of its duplicate
// key and the members that decide which duplicate is kept (D8 item 6). It
// also keeps the findings past the cap that tie with the last listed one on
// every member of the key before schema: a later duplicate of a listed
// finding that sorts after its first copy lets the first of them back in,
// and the order of the key leaves no other finding past the cap able to
// return. Those are the distinct findings one code reports about one value,
// which the input and schema limits bound, so exact totals and an exact
// list never cost more than that beside the cap.
type Report struct {
	max    int
	groups map[[sha256.Size]byte]*group
	listed itemHeap // the last listed finding first
	spill  itemHeap // past the cap, tying with listed's first on the prefix; the first in order first
	sum    counts
}

// group is one distinct finding: every finding with one duplicate key.
type group struct {
	keywordLocation string            // the kept finding's, which sorts first
	digest          [sha256.Size]byte // the kept finding's encoding, which breaks a tie
	line, column    int               // every duplicate's
	it              *item             // the kept finding while it is held; nil once it cannot be listed
}

type item struct {
	f       Finding
	g       *group
	index   int // in listed, or in spill
	spilled bool
}

type counts struct {
	errors, warnings, info int
	byCode                 map[Code]int
	// The classes of the errors, for Exit. A content error about the
	// package's own content is kept apart, since its status depends on
	// the command's role (ADR-006 D12 item 2).
	classes        [ClassInternal + 1]bool
	packageContent bool
}

// NewReport returns an empty report that lists at most maxFindings, which
// is ADR-009's findings limit, as the command's limits give it. A
// negative cap lists nothing.
func NewReport(maxFindings int) *Report {
	return &Report{
		max:    max(maxFindings, 0),
		groups: make(map[[sha256.Size]byte]*group),
		listed: itemHeap{last: true},
		spill:  itemHeap{spilled: true},
		sum:    counts{byCode: make(map[Code]int)},
	}
}

// Add adds f to the report. A duplicate of a finding already added
// (ADR-006 D8 item 3) is counted once; of the two, the report keeps the
// one whose schema.keyword_location sorts first, and of two with the same
// keyword_location, the one whose JSON encoding has the lower SHA-256
// digest, so which is kept never depends on the order they came in.
//
// Add panics when f was not built by New, or when f is a duplicate at
// another line or column: a value's position is fixed by its file and
// pointer, so that is a defect in whatever positioned it.
func (r *Report) Add(f Finding) {
	if f.entry == nil {
		panic("finding: Report.Add of a Finding New did not build")
	}
	key := f.duplicateKey()
	kl := f.keywordLocation()
	l := f.locationOrZero()
	g, seen := r.groups[key]
	if !seen {
		g = &group{keywordLocation: kl, digest: sha256.Sum256(AppendJSON(nil, f)), line: l.Line, column: l.Column}
		r.groups[key] = g
		r.count(f)
	} else {
		if l.Line != g.line || l.Column != g.column {
			panic(fmt.Sprintf("finding: duplicates of %s at %s are at %d:%d and %d:%d", f.entry.Code, l.Pointer, g.line, g.column, l.Line, l.Column))
		}
		if kl > g.keywordLocation {
			return
		}
		d := sha256.Sum256(AppendJSON(nil, f))
		if kl == g.keywordLocation && bytes.Compare(d[:], g.digest[:]) >= 0 {
			return
		}
		g.keywordLocation, g.digest = kl, d
		if g.it != nil {
			r.remove(g.it)
		}
	}
	it := &item{f: f, g: g}
	g.it = it
	r.insert(it)
}

func (r *Report) count(f Finding) {
	r.sum.byCode[f.entry.Code]++
	switch f.entry.Severity {
	case SeverityError:
		r.sum.errors++
		if f.entry.Class == ClassContent && packageSubject(f.subject.Kind) {
			r.sum.packageContent = true
		} else {
			r.sum.classes[f.entry.Class] = true
		}
	case SeverityWarning:
		r.sum.warnings++
	case SeverityInfo:
		r.sum.info++
	}
}

// insert lists it if the list has room or it sorts before the last listed
// finding, and keeps whatever that pushes past the cap in the spill while
// it ties with the new last listed finding on the prefix. Every finding in
// the spill ties with the last listed one, so when that changes to one
// earlier on the prefix, the whole spill goes: it sorts after every
// finding that can still be listed.
func (r *Report) insert(it *item) {
	heap.Push(&r.listed, it)
	if r.listed.Len() <= r.max {
		return
	}
	out := heap.Pop(&r.listed).(*item)
	if r.listed.Len() == 0 {
		drop(out)
		return
	}
	last := r.listed.items[0].f
	if comparePrefix(out.f, last) == 0 {
		heap.Push(&r.spill, out)
	} else {
		drop(out)
	}
	if r.spill.Len() > 0 && comparePrefix(r.spill.items[0].f, last) != 0 {
		for _, s := range r.spill.items {
			drop(s)
		}
		r.spill.items = nil
	}
}

func drop(it *item) { it.g.it = nil }

// remove takes it out of the report, listing the first of the spill in its
// place when it was listed.
func (r *Report) remove(it *item) {
	drop(it)
	if it.spilled {
		heap.Remove(&r.spill, it.index)
		return
	}
	heap.Remove(&r.listed, it.index)
	if r.spill.Len() > 0 {
		heap.Push(&r.listed, heap.Pop(&r.spill))
	}
}

// omitted is how many distinct findings are counted and not listed.
func (r *Report) omitted() int { return len(r.groups) - r.listed.Len() }

// Findings returns the listed findings in D8's order, followed, when any
// were omitted, by one report.truncated (D8 item 6).
func (r *Report) Findings() []Finding {
	out := r.sorted()
	if n := r.omitted(); n > 0 {
		out = append(out, r.truncated(n))
	}
	return out
}

// sorted is the listed findings in D8's order.
func (r *Report) sorted() []Finding {
	out := make([]Finding, 0, r.listed.Len()+1)
	for _, it := range r.listed.items {
		out = append(out, it.f)
	}
	slices.SortFunc(out, Compare)
	return out
}

func (r *Report) truncated(omitted int) Finding {
	return New(CodeReportTruncated, Subject{Kind: SubjectReport}, nil,
		Params{"listed": Int(int64(r.listed.Len())), "omitted": Int(int64(omitted))})
}

// CodeCount is how many distinct findings of one code a report counted.
type CodeCount struct {
	Code  Code
	Count int
}

// Summary is a report's exact totals (ADR-006 D10 item 3). Errors,
// Warnings, Info and ByCode count every distinct finding, listed or
// omitted, and the report.truncated finding when there is one; Listed and
// Omitted are its params, which count the other findings. ByCode is in
// code order.
type Summary struct {
	Errors, Warnings, Info int
	Listed, Omitted        int
	ByCode                 []CodeCount
}

// Summary returns the report's totals.
func (r *Report) Summary() Summary {
	s := Summary{
		Errors:   r.sum.errors,
		Warnings: r.sum.warnings,
		Info:     r.sum.info,
		Listed:   r.listed.Len(),
		Omitted:  r.omitted(),
	}
	by := make(map[Code]int, len(r.sum.byCode)+1)
	for c, n := range r.sum.byCode {
		by[c] = n
	}
	if s.Omitted > 0 {
		s.Info++
		by[CodeReportTruncated]++
	}
	for c, n := range by {
		s.ByCode = append(s.ByCode, CodeCount{c, n})
	}
	slices.SortFunc(s.ByCode, func(a, b CodeCount) int { return strings.Compare(string(a.Code), string(b.Code)) })
	return s
}

// itemHeap is a heap of items by Compare, for container/heap: the last
// in D8's order first when last is set, otherwise the first. Its items
// record their index, and spilled says which heap they are in.
type itemHeap struct {
	items   []*item
	last    bool
	spilled bool
}

func (h *itemHeap) Len() int { return len(h.items) }
func (h *itemHeap) Less(i, j int) bool {
	c := Compare(h.items[i].f, h.items[j].f)
	if h.last {
		return c > 0
	}
	return c < 0
}
func (h *itemHeap) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	h.items[i].index, h.items[j].index = i, j
}
func (h *itemHeap) Push(x any) {
	it := x.(*item)
	it.index, it.spilled = len(h.items), h.spilled
	h.items = append(h.items, it)
}
func (h *itemHeap) Pop() any {
	n := len(h.items) - 1
	it := h.items[n]
	h.items[n] = nil
	h.items = h.items[:n]
	return it
}

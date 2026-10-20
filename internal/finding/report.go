// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"bytes"
	"cmp"
	"container/heap"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"

	"github.com/kottera/metais/internal/fault"
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
	max int
	// flat counts each finding as one, whatever its alternatives list: a
	// branch's cap counts its own findings (ADR-006 D6 item 6.4). A
	// report's cap counts every finding it lists, those within
	// alternatives too (MTS-FND-020), and lists the findings it holds
	// only while they leave the cap room.
	flat bool
	groups map[[sha256.Size]byte]*group
	listed itemHeap // the last listed finding first
	spill  itemHeap // past the cap, tying with listed's first on the prefix; the first in order first
	sum    counts
	lost   bool // a finding has gone past the cap
	// firstOnly is fault.FirstFinding: every error after the first is
	// dropped, as a report that stops at its first error would.
	firstOnly bool
}

// group is one distinct finding: every finding with one duplicate key.
// Beside the kept finding's schema.keyword_location it holds the members
// of D8 item 1's key in which two duplicates can differ, so a later
// duplicate is judged against the kept one even once the cap has let it
// go.
type group struct {
	keywordLocation string // the kept finding's, which sorts first
	line, column    int    // the kept finding's
	hasSchema       bool   // the kept finding's
	entry, pointer  string // the kept finding's schema.entry and schema.pointer
	it              *item  // the kept finding while it is held; nil once it cannot be listed
}

// keep records f as the group's kept finding.
func (g *group) keep(f Finding) {
	l, s := f.locationOrZero(), f.schemaOrZero()
	g.keywordLocation, g.line, g.column = f.keywordLocation(), l.Line, l.Column
	g.hasSchema, g.entry, g.pointer = f.schema != nil, s.Entry, s.Pointer
}

// order compares f, a duplicate of the group's kept finding, with it by
// D8 item 1's key. The members a duplicate shares with it compare equal,
// so only these can decide: line and column, then schema.entry and
// schema.pointer.
func (g *group) order(f Finding) int {
	l, s := f.locationOrZero(), f.schemaOrZero()
	return cmp.Or(
		cmp.Compare(l.Line, g.line),
		cmp.Compare(l.Column, g.column),
		cmpBool(f.schema != nil, g.hasSchema),
		strings.Compare(s.Entry, g.entry),
		strings.Compare(s.Pointer, g.pointer),
	)
}

type item struct {
	f       Finding
	g       *group
	index   int // in listed, or in spill
	spilled bool
	digest  *[sha256.Size]byte // f's encoding's, once a tie has needed it
}

// encodingDigest returns the SHA-256 digest of the item's JSON encoding,
// which breaks a tie between duplicates. It is computed only when a tie
// needs it, since the encoding holds the pointer, which can be as long as
// the input.
func (it *item) encodingDigest() [sha256.Size]byte {
	if it.digest == nil {
		d := sha256.Sum256(AppendJSON(nil, it.f))
		it.digest = &d
	}
	return *it.digest
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

// NewReportFaults returns NewReport(maxFindings) under faults. It is the
// seam of fault.FirstFinding (ADR-010 D13 item 2): with that fault the
// report drops every error after the first it is given, so the SC8
// detector can be shown to catch a report that stops at one.
func NewReportFaults(maxFindings int, faults fault.Set) *Report {
	r := NewReport(maxFindings)
	r.firstOnly = faults.Has(fault.FirstFinding)
	return r
}

// Add adds f to the report. A duplicate of a finding already added
// (ADR-006 D8 item 3) is counted once; of the two, the report keeps the
// one whose schema.keyword_location sorts first, and of two with the same
// keyword_location, the one whose JSON encoding has the lower SHA-256
// digest, so which is kept never depends on the order they came in.
//
// Duplicates at different places are one finding too, as D8 item 3's key,
// which holds no line or column, makes them: a reader reports every
// repeat of one member name in one object, each at its own name, with the
// same pointer and params. Of two that tie on keyword_location, or have
// none, the one that sorts first by D8 item 1's key is kept; of two that
// tie on that as well, the one whose encoding has the lower digest.
//
// Add panics when f was not built by New. It also panics when a duplicate
// at a place after the kept finding's wins on keyword_location while the
// kept one is listed and a finding has already gone past the cap:
// the list could then be made exact only from findings the report no
// longer holds. A value's place is fixed by its file and pointer, and only
// a schema finding has a keyword_location, so that is a defect in
// whatever positioned the two.
//
// Add costs a finding nothing for the length of its pointer: the pointer
// is de-duplicated by its path's digest, compared only against findings at
// the same place, and encoded only to break a tie between duplicates that
// are both held.
func (r *Report) Add(f Finding) {
	if f.entry == nil {
		panic("finding: Report.Add of a Finding New did not build")
	}
	if r.firstOnly && f.entry.Severity == SeverityError && r.sum.errors > 0 {
		return
	}
	key := f.duplicateKey()
	kl := f.keywordLocation()
	g, seen := r.groups[key]
	it := &item{f: f}
	if !seen {
		g = &group{}
		g.keep(f)
		r.groups[key] = g
		r.count(f)
	} else {
		order := g.order(f)
		c := cmp.Or(strings.Compare(kl, g.keywordLocation), order)
		if c == 0 {
			// A group the cap has let go was past the last listed finding
			// on the members before schema, which a duplicate at the same
			// place shares and the last listed finding only ever moves
			// back from, so no such duplicate of it can be listed either.
			if g.it == nil {
				return
			}
			d, kept := it.encodingDigest(), g.it.encodingDigest()
			c = bytes.Compare(d[:], kept[:])
		}
		if c >= 0 {
			return
		}
		l := f.locationOrZero()
		if cmp.Or(cmp.Compare(l.Line, g.line), cmp.Compare(l.Column, g.column)) > 0 && g.it != nil && !g.it.spilled && r.lost {
			panic(fmt.Sprintf("finding: duplicates of %s at %s, at %d:%d and %d:%d, differ in keyword_location", f.entry.Code, l.pointer(), g.line, g.column, l.Line, l.Column))
		}
		g.keep(f)
		if g.it != nil {
			r.remove(g.it)
		}
	}
	it.g = g
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
//
// The list holds the first findings the cap could list were each to list
// only itself. What each lists within its alternatives is counted when the
// report is read (MTS-FND-020), since a duplicate that replaces a listed
// finding can list less than it did, and so let in a finding after it.
func (r *Report) insert(it *item) {
	heap.Push(&r.listed, it)
	if r.listed.Len() <= r.max {
		return
	}
	out := heap.Pop(&r.listed).(*item)
	r.lost = true
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

// omitted is how many distinct findings a flat report counts and does not
// list.
func (r *Report) omitted() int { return len(r.groups) - r.listed.Len() }

// Findings returns the listed findings in D8's order, followed, when any
// were omitted, by one report.truncated (D8 item 6).
func (r *Report) Findings() []Finding {
	out := r.sorted()
	if n := len(r.groups) - len(out); n > 0 {
		out = append(out, r.truncated(len(out), n))
	}
	return out
}

// sorted is the listed findings in D8's order. Unless the report is flat,
// each is listed while the findings before it, and what they list within
// their alternatives, leave the cap room, and the last is cut short to
// what is left (MTS-FND-020, MTS-FND-021).
func (r *Report) sorted() []Finding {
	out := make([]Finding, 0, r.listed.Len()+1)
	for _, it := range r.listed.items {
		out = append(out, it.f)
	}
	slices.SortFunc(out, Compare)
	if r.flat {
		return out
	}
	left := r.max
	for i, f := range out {
		if left == 0 {
			return out[:i]
		}
		out[i] = f.listAtMost(left)
		left -= out[i].listed
	}
	return out
}

func (r *Report) truncated(listed, omitted int) Finding {
	return New(CodeReportTruncated, Subject{Kind: SubjectReport}, nil,
		Params{"listed": Int(int64(listed)), "omitted": Int(int64(omitted))})
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
	listed := len(r.sorted())
	s := Summary{
		Errors:   r.sum.errors,
		Warnings: r.sum.warnings,
		Info:     r.sum.info,
		Listed:   listed,
		Omitted:  len(r.groups) - listed,
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

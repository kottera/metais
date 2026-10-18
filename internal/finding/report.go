// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"bytes"
	"container/heap"
	"crypto/sha256"
	"slices"
	"strings"
)

// Report gathers one command's findings (ADR-006 D8): it removes
// duplicates, keeps the first maxFindings in D8's order, and counts every
// finding exactly, listed or not.
//
// Memory is bounded by the cap, not by the input. Beside the listed
// findings the report keeps, per distinct finding, the SHA-256 digest of
// its duplicate key and the member that decides which duplicate is kept
// (D8 item 6). It also keeps the findings just past the cap that tie with
// the last listed one on every member of the key before schema: a later
// duplicate of a listed finding that sorts after its first copy can let
// one of them back in, and only one of those can be it. The order of the
// key makes any other finding past the cap unable to return.
type Report struct {
	max    int
	groups map[[sha256.Size]byte]*group
	listed itemHeap // a max-heap: the last listed finding is at the top
	spill  []*item  // past the cap, tying with the top on the prefix
	sum    counts
}

// group is one distinct finding: every finding with one duplicate key.
type group struct {
	keywordLocation string            // the kept finding's, which sorts first
	digest          [sha256.Size]byte // the kept finding's encoding, which breaks a tie
	it              *item             // the kept finding while it is held; nil once it cannot be listed
}

type item struct {
	f     Finding
	g     *group
	index int // in listed; -1 in spill
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
		sum:    counts{byCode: make(map[Code]int)},
	}
}

// Add adds f to the report. A duplicate of a finding already added
// (ADR-006 D8 item 3) is counted once; of the two, the report keeps the
// one whose schema.keyword_location sorts first, and of two with the same
// keyword_location, the one whose JSON encoding has the lower SHA-256
// digest, so which is kept never depends on the order they came in.
func (r *Report) Add(f Finding) {
	if f.entry == nil {
		panic("finding: Report.Add of a Finding New did not build")
	}
	key := f.duplicateKey()
	kl := f.keywordLocation()
	g, seen := r.groups[key]
	if !seen {
		g = &group{keywordLocation: kl, digest: sha256.Sum256(AppendJSON(nil, f))}
		r.groups[key] = g
		r.count(f)
	} else {
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

// insert lists it if it sorts before the last listed finding or the list
// has room, and moves whatever that pushes past the cap to the spill.
func (r *Report) insert(it *item) {
	heap.Push(&r.listed, it)
	if len(r.listed) <= r.max {
		return
	}
	out := heap.Pop(&r.listed).(*item)
	out.index = -1
	r.spill = append(r.spill, out)
	r.prune()
}

// prune drops from the spill every finding that no longer ties with the
// last listed one on the prefix; it sorts after every finding that can
// still be listed, and can never return.
func (r *Report) prune() {
	keep := r.spill[:0]
	for _, it := range r.spill {
		if len(r.listed) > 0 && comparePrefix(it.f, r.listed[0].f) == 0 {
			keep = append(keep, it)
		} else {
			it.g.it = nil
		}
	}
	clear(r.spill[len(keep):])
	r.spill = keep
}

// remove takes it out of the report, refilling the list from the spill
// when it was listed.
func (r *Report) remove(it *item) {
	it.g.it = nil
	if it.index < 0 {
		i := slices.Index(r.spill, it)
		r.spill = slices.Delete(r.spill, i, i+1)
		return
	}
	heap.Remove(&r.listed, it.index)
	if len(r.spill) == 0 {
		return
	}
	i := 0
	for j := range r.spill {
		if Compare(r.spill[j].f, r.spill[i].f) < 0 {
			i = j
		}
	}
	best := r.spill[i]
	r.spill = slices.Delete(r.spill, i, i+1)
	heap.Push(&r.listed, best)
}

// omitted is how many distinct findings are counted and not listed.
func (r *Report) omitted() int { return len(r.groups) - len(r.listed) }

// Findings returns the listed findings in D8's order, followed, when any
// were omitted, by one report.truncated (D8 item 6).
func (r *Report) Findings() []Finding {
	out := make([]Finding, 0, len(r.listed)+1)
	for _, it := range r.listed {
		out = append(out, it.f)
	}
	slices.SortFunc(out, Compare)
	if n := r.omitted(); n > 0 {
		out = append(out, r.truncated(n))
	}
	return out
}

func (r *Report) truncated(omitted int) Finding {
	return New(CodeReportTruncated, Subject{Kind: SubjectReport}, nil,
		Params{"listed": Int(int64(len(r.listed))), "omitted": Int(int64(omitted))})
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
		Listed:   len(r.listed),
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

// itemHeap is a max-heap of items by Compare, for container/heap.
type itemHeap []*item

func (h itemHeap) Len() int           { return len(h) }
func (h itemHeap) Less(i, j int) bool { return Compare(h[i].f, h[j].f) > 0 }
func (h itemHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index, h[j].index = i, j
}
func (h *itemHeap) Push(x any) {
	it := x.(*item)
	it.index = len(*h)
	*h = append(*h, it)
}
func (h *itemHeap) Pop() any {
	old := *h
	it := old[len(old)-1]
	old[len(old)-1] = nil
	*h = old[:len(old)-1]
	return it
}

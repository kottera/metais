// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package ooxml

import (
	"bytes"
	"slices"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/liquid"
)

// FuzzScan runs the scan over arbitrary bytes as a template and as a
// partial it may render, named part.xml (ADR-009 D10, ADR-010 D14 item
// 6). Its seed corpus is testdata/fuzz/FuzzScan. It asserts no panic; that
// the tokenizer's events cover the source in order, with no gap and no
// overlap; that the text pieces are in order and never overlap; that every
// finding is located at a line and column of its template and has the
// parameters its code declares, with values from their closed sets; and
// that a second scan gives the same findings.
func FuzzScan(f *testing.F) {
	f.Fuzz(func(t *testing.T, root, part []byte) {
		ts := []Template{{Path: "document.xml", Text: root}, {Path: "part.xml", Text: part}}
		for _, tm := range ts {
			checkEvents(t, string(tm.Text))
		}
		fs := Scan(ts)
		for _, f := range fs {
			checkFinding(t, ts, f)
		}
		again := Scan(ts)
		if len(again) != len(fs) {
			t.Fatalf("a second scan gave %d findings, the first %d", len(again), len(fs))
		}
		for i := range fs {
			if finding.Compare(fs[i], again[i]) != 0 {
				t.Fatalf("a second scan's finding %d differs", i)
			}
		}
	})
}

// checkEvents holds the tokenizer and the scanner's pieces to their
// invariants over src.
func checkEvents(t *testing.T, src string) {
	t.Helper()
	cons := liquid.Constructs(src)
	for i, c := range cons {
		if c.Span.Start < 0 || c.Span.End > len(src) || c.Span.End <= c.Span.Start || i > 0 && c.Span.Start < cons[i-1].Span.End {
			t.Fatalf("construct %d [%d, %d) is out of order or out of the source of %d bytes", i, c.Span.Start, c.Span.End, len(src))
		}
	}
	z := newTokenizer(src, cons)
	at := 0
	for ev, ok := z.next(); ok; ev, ok = z.next() {
		if ev.start != at || ev.end <= ev.start || ev.end > len(src) {
			t.Fatalf("event [%d, %d) after offset %d, in a source of %d bytes", ev.start, ev.end, at, len(src))
		}
		if (ev.kind == evText || ev.kind == evCDATA) && (ev.textStart < ev.start || ev.textEnd > ev.end || ev.textEnd < ev.textStart) {
			t.Fatalf("content [%d, %d) outside its event [%d, %d)", ev.textStart, ev.textEnd, ev.start, ev.end)
		}
		at = ev.end
	}
	if at != len(src) {
		t.Fatalf("the events end at %d of %d bytes", at, len(src))
	}
	p := &template{src: src, cons: cons}
	s := newScanner(p, bindings{"": famW | famS | famA, "w": famW, "a": famA, "x": famS})
	s.walk()
	for i, pc := range s.pieces {
		if pc.end <= pc.start || i > 0 && pc.start < s.pieces[i-1].end {
			t.Fatalf("piece %d [%d, %d) is empty or out of order", i, pc.start, pc.end)
		}
	}
}

// causeValues is the cause parameter's closed set.
var causeValues = []string{
	"spelling or grammar marks", "tracked changes (accept or reject them first)",
	"a bookmark or comment", "a field", "a change of run properties (formatting)",
}

func checkFinding(t *testing.T, ts []Template, f finding.Finding) {
	t.Helper()
	loc, ok := f.Location()
	var src []byte
	for _, tm := range ts {
		if TemplatesDir+tm.Path == loc.File {
			src = tm.Text
		}
	}
	if !ok || src == nil || f.Subject().Name != loc.File {
		t.Fatalf("%s located at %+v, about %+v", f.Code(), loc, f.Subject())
	}
	lines := 1 + bytes.Count(src, []byte("\n")) + bytes.Count(src, []byte("\r")) - bytes.Count(src, []byte("\r\n"))
	if loc.Line < 1 || loc.Line > lines || loc.Column < 1 || loc.Column > len(src) {
		t.Fatalf("%s at %d:%d of a %d-line template", f.Code(), loc.Line, loc.Column, lines)
	}
	cause, _ := f.Param("cause")
	if c, _ := cause.Str(); !slices.Contains(causeValues, c) {
		t.Fatalf("%s: cause %q", f.Code(), c)
	}
	n, _ := f.Param("text_elements")
	if k, ok := n.Int(); !ok || k < 1 {
		t.Fatalf("%s: text_elements %v", f.Code(), k)
	}
	if f.Code() == CodeSplitDelimiter {
		d, _ := f.Param("delimiter")
		if s, _ := d.Str(); !slices.Contains([]string{"{{", "{%", "}}", "%}"}, s) {
			t.Fatalf("delimiter %q", s)
		}
	}
}

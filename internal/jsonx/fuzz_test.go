// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package jsonx

import (
	"strconv"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
)

// FuzzJSONParse is ADR-010 D14 item 6's target: whatever the bytes, Parse
// does not panic, every position it gives is within the input, and a
// value it accepts round-trips through Canonical, which Encode matches.
// Its seed corpus, under testdata/fuzz/FuzzJSONParse, runs in every go
// test.
func FuzzJSONParse(f *testing.F) {
	small := limits.Defaults()
	for _, a := range []string{"value-depth=4", "number-length=8"} {
		if _, ok := small.Lower(a); !ok {
			f.Fatalf("Lower(%q)", a)
		}
	}
	f.Fuzz(func(t *testing.T, src []byte) {
		for _, lim := range []limits.Set{limits.Defaults(), small} {
			d, fs := Parse(src, vendor, lim)
			x := newLineIndex(src)
			within := func(what string, p finding.Pos) {
				if p.Line < 1 || p.Line > len(x.starts) || p.Column < 1 {
					t.Fatalf("%s at %+v, outside the input's %d lines", what, p, len(x.starts))
				}
				end := len(src)
				if p.Line < len(x.starts) {
					end = x.starts[p.Line]
				}
				if p.Column > len(src[x.starts[p.Line-1]:end])+1 {
					t.Fatalf("%s at %+v, beyond the end of its line", what, p)
				}
			}
			for _, fd := range fs {
				l, ok := fd.Location()
				if !ok {
					t.Fatalf("%s has no location", fd.Code())
				}
				within(string(fd.Code()), finding.Pos{Line: l.Line, Column: l.Column})
				for _, r := range fd.Related() {
					within(string(fd.Code())+" related", finding.Pos{Line: r.Location.Line, Column: r.Location.Column})
				}
			}
			if (d.Value == nil) != (len(fs) > 0) {
				t.Fatalf("value %v beside %d findings", d.Value, len(fs))
			}
			if d.Value == nil {
				continue
			}
			walk(d.Value, "", func(ptr string, isMember bool) {
				off, ok := d.ValueOffset(ptr)
				if !ok || off < 0 || off >= len(src) {
					t.Fatalf("value %q at offset %d (%v), outside %d bytes", ptr, off, ok, len(src))
				}
				p, _ := d.Positions().Value(ptr)
				within("value "+ptr, p)
				if isMember {
					off, ok := d.KeyOffset(ptr)
					if !ok || off < 0 || off >= len(src) || src[off] != '"' {
						t.Fatalf("key %q at offset %d (%v)", ptr, off, ok)
					}
					p, _ := d.Positions().Key(ptr)
					within("key "+ptr, p)
				}
			})
			c := Canonical(d.Value)
			e, err := Encode(d.Value, 0)
			if err != nil || string(e) != string(c) {
				t.Fatalf("Encode %q (%v) is not Canonical %q", e, err, c)
			}
			// The canonical text reads back as the value, except that
			// ADR-003 D8 item 5 refuses an integer beyond 2^53-1 only
			// when it is written without a fraction or exponent: 11e17
			// is read, and its canonical text 1100000000000000000 is
			// refused.
			again, fs := Parse(c, vendor, limits.Defaults())
			for _, fd := range fs {
				r, _ := fd.Param("reason")
				if reason, _ := r.Str(); fd.Code() != CodeNumberOutOfRange || reason != reasonUnsafeInteger {
					t.Fatalf("Canonical %q does not read back: %s", c, describe(fs))
				}
			}
			if len(fs) == 0 && !value.Equal(again.Value, d.Value) {
				t.Fatalf("Canonical %q reads back as another value", c)
			}
		}
	})
}

// walk calls visit with the pointer of every value in v, and whether it
// is an object member.
func walk(v value.Value, ptr string, visit func(ptr string, isMember bool)) {
	visit(ptr, false)
	walkChildren(v, ptr, visit)
}

func walkChildren(v value.Value, ptr string, visit func(string, bool)) {
	switch v := v.(type) {
	case value.Array:
		for i, el := range v.All() {
			p := ptr + finding.Pointer(strconv.Itoa(i))
			visit(p, false)
			walkChildren(el, p, visit)
		}
	case value.Object:
		for name, m := range v.All() {
			p := ptr + finding.Pointer(name)
			visit(p, true)
			walkChildren(m, p, visit)
		}
	}
}

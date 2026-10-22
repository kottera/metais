// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package unitext

import (
	"strings"
	"testing"
	"unicode/utf8"

	"metais.org/metais/internal/limits"
)

// FuzzPortablePath runs the predicate and the collision set over arbitrary
// bytes (ADR-010 D14 item 6). Its seed corpus is testdata/fuzz/FuzzPortablePath.
// It asserts no panic, and that an accepted path has the properties every
// reader of it relies on: valid UTF-8 in NFC, well shaped, within the
// limits, built of path characters, colliding with nothing alone and
// reported once as a duplicate when given twice. Its input is also split
// at line feeds into a tree for Collisions, whose every report must pair
// two caseless matches.
func FuzzPortablePath(f *testing.F) {
	lim := limits.Defaults()
	f.Fuzz(func(t *testing.T, p string) {
		got := CheckPath(p, lim)
		for _, pr := range got {
			if pr.Kind < Encoding || pr.Kind > Depth {
				t.Fatalf("CheckPath(%+q) gave a problem of kind %d", p, pr.Kind)
			}
		}
		paths := strings.Split(p, "\n")
		cs := Collisions(paths)
		for _, c := range cs {
			if CaselessKey(c.Member) != CaselessKey(c.OtherMember) {
				t.Fatalf("Collisions(%+q) reported %+v, whose members do not match", paths, c)
			}
		}
		if len(got) > 0 {
			return
		}
		switch {
		case !utf8.ValidString(p), !IsNFC(p):
			t.Fatalf("CheckPath accepted %+q, which is not NFC UTF-8", p)
		case p == "", strings.HasPrefix(p, "/"), strings.HasSuffix(p, "/"), strings.Contains(p, "//"),
			strings.ContainsAny(p, "\\:<>\"|?*\x00"):
			t.Fatalf("CheckPath accepted %+q, which is not well shaped", p)
		case int64(len(p)) > lim.Value(limits.PathLength):
			t.Fatalf("CheckPath accepted %+q, which is longer than path-length", p)
		}
		for seg := range strings.SplitSeq(p, "/") {
			if seg == "." || seg == ".." || len(seg) > MaxSegment || isReservedName(seg) || isGit(seg) {
				t.Fatalf("CheckPath accepted %+q, whose segment %+q is refused", p, seg)
			}
			for _, r := range seg {
				if !IsPathChar(r) {
					t.Fatalf("CheckPath accepted %+q, which holds %U", p, r)
				}
			}
		}
		if cs := Collisions([]string{p}); len(cs) != 0 {
			t.Fatalf("an accepted path collides with itself: %+v", cs)
		}
		if cs := Collisions([]string{p, p}); len(cs) != 1 || cs[0].Kind != Duplicate {
			t.Fatalf("an accepted path given twice is not one duplicate: %+v", cs)
		}
	})
}

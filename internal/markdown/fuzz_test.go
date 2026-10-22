// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package markdown

import (
	"bytes"
	"testing"

	"metais.org/metais/internal/limits"
)

// FuzzFrontmatter runs SplitFrontmatter over arbitrary bytes (ADR-010 D14
// item 6). Its seed corpus is testdata/fuzz/FuzzFrontmatter. It asserts no
// panic; that a split puts the file back together, the frontmatter at a
// line within the file; that a file with no frontmatter is its body; that a
// finding comes alone, with nothing returned, at a line within the file;
// and that a lowered frontmatter-size holds the reading to the bound.
func FuzzFrontmatter(f *testing.F) {
	lowered := limits.Defaults()
	if _, ok := lowered.Lower("frontmatter-size=16"); !ok {
		f.Fatal("cannot lower frontmatter-size")
	}
	f.Fuzz(func(t *testing.T, src []byte) {
		norm := normalize(src)
		lines := lineCount(norm)
		for _, lim := range []limits.Set{limits.Defaults(), lowered} {
			front, frontLine, body, fs := SplitFrontmatter(src, testSource, lim)
			if bytes.ContainsRune(front, '\r') || bytes.ContainsRune(body, '\r') {
				t.Fatalf("SplitFrontmatter(%q) left a CR in %q, %q", src, front, body)
			}
			switch {
			case len(fs) > 0:
				if len(fs) != 1 || front != nil || frontLine != 0 || body != nil {
					t.Fatalf("SplitFrontmatter(%q) = %q, %d, %q with findings %v", src, front, frontLine, body, codes(fs))
				}
				if l, ok := fs[0].Location(); !ok || l.Line < 1 || l.Line > lines {
					t.Fatalf("SplitFrontmatter(%q): finding at %+v of a %d-line file", src, l, lines)
				}
			case frontLine == 0:
				if front != nil || !bytes.Equal(body, norm) {
					t.Fatalf("SplitFrontmatter(%q) = %q, %d, %q; want the whole file as the body", src, front, frontLine, body)
				}
			default:
				if frontLine != 2 || frontLine > lines {
					t.Fatalf("SplitFrontmatter(%q): frontmatter at line %d of a %d-line file", src, frontLine, lines)
				}
				if int64(len(front)) > lim.Value(limits.FrontmatterSize) {
					t.Fatalf("SplitFrontmatter(%q): %d bytes of frontmatter passed a bound of %d", src, len(front), lim.Value(limits.FrontmatterSize))
				}
				head := append(append([]byte("---\n"), front...), "---"...)
				if !bytes.HasPrefix(norm, head) || !bytes.HasSuffix(norm, body) ||
					(len(norm) != len(head)+len(body) && len(norm) != len(head)+1+len(body)) {
					t.Fatalf("SplitFrontmatter(%q) = %q, %q, which do not make the file again", src, front, body)
				}
			}
		}
		if b := delimit(src, 16); b.scanned > len("---\r\n")+2*(16+1) {
			t.Fatalf("delimit(%q) examined %d bytes at a bound of 16", src, b.scanned)
		}
	})
}

// FuzzOutline runs Outline over arbitrary bytes (ADR-010 D14 item 6). Its
// seed corpus is testdata/fuzz/FuzzOutline. It asserts no panic, that the
// result is a list, and that every heading is at a level of 1 to 6 and at
// a line within the input, in order.
func FuzzOutline(f *testing.F) {
	f.Fuzz(func(t *testing.T, src []byte) {
		got := Outline(src)
		if got == nil {
			t.Fatalf("Outline(%q) = nil", src)
		}
		checkOutline(t, src)
	})
}

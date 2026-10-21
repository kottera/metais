// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
)

// TestManifestSchemaWork is ADR-009 Amendment A6 item 2, pinned false
// first: a manifest whose validation would cost more than schema-work, an
// unknown member holding an array of 500,000 items within manifest-size,
// is refused naming schema-work, never as a defect in the tool.
func TestManifestSchemaWork(t *testing.T) {
	files := minimal()
	files["metais.yaml"] = append(files["metais.yaml"], []byte("x: ["+strings.Repeat("0,", 499_999)+"0]\n")...)
	p, fs := OpenFiles(files, defaults)
	if p != nil {
		t.Fatal("the package opened")
	}
	if len(fs) != 1 || fs[0].Code() != finding.CodeLimitPackage {
		t.Fatalf("findings %v, want one limit.package", codes(fs))
	}
	l, _ := fs[0].Param("limit")
	bound, _ := fs[0].Param("bound")
	if s, _ := l.Str(); s != "schema-work" {
		t.Errorf("limit %q, want schema-work", s)
	}
	if n, _ := bound.Int(); n != 450_000 {
		t.Errorf("bound %d, want the default, 450,000", n)
	}
	if loc, _ := fs[0].Location(); loc.File != "metais.yaml" {
		t.Errorf("located at %q, want metais.yaml", loc.File)
	}
}

// TestReferenceTitleStops is ADR-009 Amendment A6 item 2, pinned false
// first: the references index the open builds takes a Markdown entry's
// title alone, and keeps no heading after it, so what a title costs does
// not grow with the headings that follow it.
func TestReferenceTitleStops(t *testing.T) {
	cost := func(after int) float64 {
		src := []byte("## Before\n\n# The [title]\n\n" + strings.Repeat("## After\n", after) + "\n[title]: /t\n")
		var got *string
		allocs := testing.AllocsPerRun(5, func() { got = referenceTitle(src) })
		if got == nil || *got != "The title" {
			t.Fatalf("after %d headings, the title is %v", after, got)
		}
		return allocs
	}
	if few, many := cost(10), cost(10_000); many > few+16 {
		t.Errorf("a title followed by 10 headings costs %v allocations, and by 10,000, %v", few, many)
	}
}

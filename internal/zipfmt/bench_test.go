// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package zipfmt

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/testenv"
)

// benchEntries is a package's worth of entries at the entries limit's
// default, 10,000, as pack writes them: in byte order of their names,
// every one method auto, Markdown that deflates and bytes that do not.
func benchEntries() []Entry {
	n := int(limits.Defaults().Value(limits.Entries))
	var out []Entry
	for i := range n {
		var name string
		var content []byte
		switch {
		case i%10 == 9:
			name = fmt.Sprintf("assets/%02d/%04d.bin", i%37, i)
			content = make([]byte, 512+i%1024)
			for j := range content {
				content[j] = byte((i*131 + j*j*7) >> 3)
			}
		default:
			name = fmt.Sprintf("references/%02d/%04d.md", i%37, i)
			content = []byte(fmt.Sprintf("# Reference %d\n\n%s", i, strings.Repeat(fmt.Sprintf("Line of reference %d, which says what it says.\n", i), 8+i%32)))
		}
		out = append(out, Entry{Name: name, Content: content, Method: Auto})
	}
	slices.SortFunc(out, func(a, b Entry) int { return strings.Compare(a.Name, b.Name) })
	return out
}

// BenchmarkOpen is ADR-010 D15 item 1's zip reading of a package of
// 10,000 entries: the open, which reads every byte once, decodes and
// checks every entry, and builds the tree (ADR-001 D8).
func BenchmarkOpen(b *testing.B) {
	testenv.Home(b)
	es := benchEntries()
	pkg := write(b, es)
	b.SetBytes(int64(len(pkg)))
	b.ReportAllocs()
	for b.Loop() {
		a, found, err := Open(bytes.NewReader(pkg), int64(len(pkg)), limits.Defaults())
		if err != nil || len(found) > 0 || len(a.List()) != len(es) {
			b.Fatalf("Open: %v %q", err, codes(found))
		}
	}
}

// TestBenchPackage holds the benchmark's package to its 10,000 entries,
// opened without a finding, in every go test.
func TestBenchPackage(t *testing.T) {
	testenv.Home(t)
	es := benchEntries()
	pkg := write(t, es)
	a, found, err := Open(bytes.NewReader(pkg), int64(len(pkg)), limits.Defaults())
	if err != nil || len(found) > 0 {
		t.Fatalf("Open: %v %q", err, codes(found))
	}
	if len(es) != 10_000 || len(a.List()) != len(es) || a.Decodes() != len(es) {
		t.Errorf("the package lists %d of %d entries, decoded %d times; want 10,000", len(a.List()), len(es), a.Decodes())
	}
}

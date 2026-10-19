// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package tarw

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the tar golden files")

// goldens are ADR-010 D14 item 5's tar layout: the exact bytes archive/tar
// writes, under the pinned toolchain, for a path that fits ustar and for
// one that requires a pax path record.
var goldens = []struct {
	file string
	f    file
	pax  bool
}{
	{"ustar.tar", file{"docs/readme.txt", false, "hello, archive\n"}, false},
	{"pax.tar", file{"café/menü.sh", true, "#!/bin/sh\necho ok\n"}, true},
}

// sameBytes compares an archive with its golden file, naming the first
// offset where they differ.
func sameBytes(got, want []byte) error {
	for i := range min(len(got), len(want)) {
		if got[i] != want[i] {
			return fmt.Errorf("byte %d (block %d, offset %d) is %#02x, and the golden file has %#02x", i, i/blockSize, i%blockSize, got[i], want[i])
		}
	}
	if len(got) != len(want) {
		return fmt.Errorf("%d bytes, and the golden file has %d", len(got), len(want))
	}
	return nil
}

// TestGolden is W2-13.1. Run with -update to rewrite the golden files
// after a deliberate change, such as a toolchain whose archive/tar writes
// other bytes, and read the diff: that is a change of the tool's output.
func TestGolden(t *testing.T) {
	for _, g := range goldens {
		path := filepath.Join("testdata", g.file)
		got := write(t, g.f)
		if *update {
			if err := os.WriteFile(path, got, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%v (run go test -run TestGolden -args -update)", err)
		}
		if err := sameBytes(got, want); err != nil {
			t.Errorf("%s: %v", g.file, err)
		}

		// The golden bytes are themselves D17's: every fixed field, and an
		// extended header exactly where the path needs one.
		ms, err := members(want)
		if err != nil {
			t.Fatalf("%s: %v", g.file, err)
		}
		if len(ms) != 1 {
			t.Fatalf("%s: %d members, want 1", g.file, len(ms))
		}
		for _, fd := range fields {
			if err := fd.check(ms[0], g.f); err != nil {
				t.Errorf("%s: %s: %v", g.file, fd.name, err)
			}
		}
		if (ms[0].paxHeader != nil) != g.pax {
			t.Errorf("%s: extended header %v, want %v", g.file, ms[0].paxHeader != nil, g.pax)
		}

		// The comparison is red on any one changed byte, in a header, in
		// the content and in the end blocks.
		for _, i := range []int{0, 156, blockSize, len(want) - 1} {
			changed := slices.Clone(want)
			changed[i] ^= 0x01
			if sameBytes(got, changed) == nil {
				t.Errorf("%s: the comparison passes with byte %d changed", g.file, i)
			}
		}
		if sameBytes(got, want[:len(want)-1]) == nil {
			t.Errorf("%s: the comparison passes a truncated golden file", g.file)
		}
	}
}

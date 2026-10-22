// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package zipfmt

import (
	"bytes"
	"io"
	"testing"

	"metais.org/metais/internal/limits"
)

// FuzzZipRead runs the strict reader over arbitrary bytes (ADR-010 D14
// item 6). Its seed corpus, testdata/fuzz/FuzzZipRead, is valid packages
// and the hostile cases of TestOpenRefuses. It asserts no panic, no error
// from a reader that cannot fail, an archive exactly when there is no
// finding, every finding's offset within the input, and that an archive's
// every entry reads back as its size.
func FuzzZipRead(f *testing.F) {
	lim := limits.Defaults()
	f.Fuzz(func(t *testing.T, b []byte) {
		a, found, err := Open(bytes.NewReader(b), int64(len(b)), lim)
		if err != nil {
			t.Fatalf("Open over a bytes.Reader: %v", err)
		}
		if (a == nil) == (len(found) == 0) {
			t.Fatalf("Open returned an archive %v beside %d findings", a != nil, len(found))
		}
		for _, fd := range found {
			if v, ok := fd.Param("offset"); ok {
				if n, _ := v.Int(); n < 0 || n >= int64(len(b)) {
					t.Fatalf("%s: offset %d is outside the input's %d bytes", fd.Code(), n, len(b))
				}
			}
		}
		if a == nil {
			return
		}
		for _, p := range a.List() {
			info, err := a.Stat(p)
			if err != nil {
				t.Fatal(err)
			}
			r, err := a.Open(p)
			if err != nil {
				t.Fatal(err)
			}
			n, err := io.Copy(io.Discard, r)
			if err != nil || n != info.Size {
				t.Fatalf("%q: read %d bytes and %v, want %d", p, n, err, info.Size)
			}
		}
	})
}

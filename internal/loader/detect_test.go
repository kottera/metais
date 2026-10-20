// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/specdata"
)

// detectionKey is the leading bytes every METAIS 1.0 package shares
// (ADR-001 A1 item 8, chapter 2 §2.1): the local header signature, then,
// at offset 30, the name README.md, and at offset 39 the README text's
// first line.
func detectionKey(b []byte) bool {
	first, _, _ := bytes.Cut(specdata.Readme10, []byte("\n"))
	return len(b) >= 39+len(first) && string(b[:4]) == "PK\x03\x04" &&
		string(b[30:39]) == "README.md" && bytes.Equal(b[39:39+len(first)], first)
}

// TestDetectionKey holds the detection rule chapter 2 states to every
// committed packed package that opens: each begins with the key. Its
// control is every committed package refused for its README.md, none of
// which a detector keyed this way needs to find, and two of which it
// does not.
func TestDetectionKey(t *testing.T) {
	if first, _, _ := bytes.Cut(specdata.Readme10, []byte("\n")); string(first) != "# This is a METAIS package" {
		t.Fatalf("the README text's first line is %q, not the one chapter 2 names", first)
	}
	opened, missed := 0, 0
	err := filepath.WalkDir(vectorsDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".metais") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		pkg, _ := openPacked(bytes.NewReader(b), int64(len(b)), defaults, 0)
		switch {
		case pkg != nil && !detectionKey(b):
			t.Errorf("%s opens, and does not begin with the detection key: %q", p, b[:min(len(b), 66)])
		case pkg != nil:
			opened++
		case strings.Contains(p, "/readme-missing/") || strings.Contains(p, "/readme-not-first/"):
			if detectionKey(b) {
				t.Errorf("%s begins with the detection key", p)
			}
			missed++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if opened < 10 || missed != 2 {
		t.Errorf("%d packages opened and %d were held to lacking the key; the walk found too few", opened, missed)
	}
}

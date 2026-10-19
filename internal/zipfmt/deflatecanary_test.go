// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package zipfmt

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/kottera/metais/internal/limits"
)

// canaryDigest is the SHA-256 of the package Write makes of
// testdata/canary/ with the pinned toolchain's encoder (ADR-010 D1 item
// 3). A toolchain whose deflate encoder writes other bytes changes it; that
// is a change of the tool's output, taken deliberately with the toolchain
// and recorded here, never discovered in a user's diff (ADR-001 D4 item 6).
const canaryDigest = "73d3b2ce421591b1acd8df0eb30706b02752f4018b8421d1a8701e309df410c5"

// canaryPackage packs the tree fsys as pack would: every regular file, in
// ascending byte order of its path, with method Auto (ADR-001 D4 item 7).
func canaryPackage(fsys fs.FS) ([]byte, error) {
	files := map[string][]byte{}
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(fsys, p)
		files[p] = b
		return err
	})
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, p := range slices.Sorted(maps.Keys(files)) {
		entries = append(entries, Entry{Name: p, Content: files[p], Method: Auto})
	}
	var buf bytes.Buffer
	err = Write(&buf, entries)
	return buf.Bytes(), err
}

// checkCanary packs fsys and compares the package's digest with want.
func checkCanary(fsys fs.FS, want string) error {
	b, err := canaryPackage(fsys)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(b)
	if got := hex.EncodeToString(sum[:]); got != want {
		return fmt.Errorf("the canary packs to sha256 %s, and %s is committed: the deflate encoder or the canary tree changed", got, want)
	}
	return nil
}

// TestDeflateCanary is ADR-010 D1 item 3's golden-bytes canary.
func TestDeflateCanary(t *testing.T) {
	if err := checkCanary(os.DirFS("testdata/canary"), canaryDigest); err != nil {
		t.Error(err)
	}
	// The tree exercises the encoder: its package holds deflated entries,
	// and entries Auto stores.
	b, err := canaryPackage(os.DirFS("testdata/canary"))
	if err != nil {
		t.Fatal(err)
	}
	a, found := open(t, b, limits.Defaults())
	if len(found) > 0 {
		t.Fatalf("the canary package is refused: %q", codes(found))
	}
	methods := map[uint16]int{}
	for _, e := range a.entries {
		methods[e.method]++
	}
	if methods[methodDeflated] < 2 || methods[methodStored] < 2 {
		t.Errorf("the canary package's methods are %v; it needs deflated and stored entries", methods)
	}
}

// TestDeflateCanaryControl holds the canary red on a changed value
// (W2-4.1): a byte of the tree, a file added, or the committed digest.
func TestDeflateCanaryControl(t *testing.T) {
	tree := fstest.MapFS{}
	err := fs.WalkDir(os.DirFS("testdata/canary"), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile("testdata/canary/" + p)
		tree[p] = &fstest.MapFile{Data: b}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := checkCanary(tree, canaryDigest); err != nil {
		t.Fatalf("the copy of the tree: %v", err)
	}
	long := tree["references/long.txt"].Data
	flipped := bytes.Clone(long)
	flipped[len(flipped)/2] ^= 1
	for name, changed := range map[string]fstest.MapFS{
		"a byte of a deflated file": withFile(tree, "references/long.txt", flipped),
		"a file added":              withFile(tree, "assets/extra.txt", []byte("x")),
	} {
		if checkCanary(changed, canaryDigest) == nil {
			t.Errorf("%s: the canary passed", name)
		}
	}
	digest := []byte(canaryDigest)
	digest[0] ^= 1
	if checkCanary(tree, string(digest)) == nil {
		t.Error("a changed digest: the canary passed")
	}
}

func withFile(tree fstest.MapFS, p string, b []byte) fstest.MapFS {
	out := maps.Clone(tree)
	out[p] = &fstest.MapFile{Data: b}
	return out
}

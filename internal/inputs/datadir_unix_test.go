// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package inputs

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/testenv"
)

// TestDataDirOpensOnlyCandidates: in a data directory the tool "lists the
// entries, then opens only those whose names ADR-003's naming rule maps to
// a declared input … and the items of those collections", and "a FIFO,
// socket or device is refused from its Lstat mode without being opened,
// so a FIFO cannot block the tool" (ADR-009 D7 items 1 and 3). A FIFO
// that names a declared input, one that names none, and one among a
// collection's items would each block an open that has no writer; a file
// no one may read, naming no input, would fail one.
func TestDataDirOpensOnlyCandidates(t *testing.T) {
	testenv.Home(t)
	dir := dataDir(t, map[string]string{"articles/a.json": `1`, "private.json": `{}`})
	for _, name := range []string{"vendor.json", "other.json", "articles/b.json"} {
		if err := syscall.Mkfifo(filepath.Join(dir, filepath.FromSlash(name)), 0o644); err != nil {
			t.Skipf("no FIFO here: %v", err)
		}
	}
	if err := os.Chmod(filepath.Join(dir, "private.json"), 0); err != nil {
		t.Fatal(err)
	}
	res := run(t, layer(t, dirPkg), limits.Defaults(), args{data: dir})
	wantFindings(t, res.Findings,
		`input.not_regular_file vendor data_dir:vendor.json {"kind":"special"}`,
		`input.file_unknown other data_dir:other.json`,
		`input.file_unknown private data_dir:private.json`,
		`input.not_regular_file articles data_dir:articles/b.json {"kind":"special"}`,
		`input.validation.partial articles data_dir:articles # {"unparsed":1,"suppressed":0}`,
	)
}

// TestCollectionItemNames: an item whose file name holds a backslash or a
// colon, which hostfs will not resolve and no file system a package is
// portable to can hold, is refused and counted among the items that did not
// parse (ADR-006 D7 item 4).
func TestCollectionItemNames(t *testing.T) {
	testenv.Home(t)
	res := run(t, layer(t, dirPkg), limits.Defaults(), args{data: dataDir(t, map[string]string{
		"articles/a:b.json": `1`,
		`articles/c\d.json`: `2`,
		"articles/ok.json":  `3`,
	})})
	wantFindings(t, res.Findings,
		`collection.item_name articles data_dir:articles/a:b.json {"reason":"character"}`,
		`collection.item_name articles data_dir:articles/c\d.json {"reason":"character"}`,
		`input.validation.partial articles data_dir:articles # {"unparsed":2,"suppressed":0}`,
	)
}

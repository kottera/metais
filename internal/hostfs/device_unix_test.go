// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package hostfs

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// The device cases (ADR-009 D17): a device cannot be created without
// privilege, so a vector cannot carry one, and these unit tests are where
// a FIFO, a socket and a device are refused. Each is refused from its
// Lstat mode without being opened: mustNotOpen fails the test if the read
// gets as far as an open, which for a FIFO would block and for /dev/zero
// would never end.

func mkfifo(t *testing.T, root, name string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(p, 0o644); err != nil {
		t.Skipf("cannot make a FIFO here: %v", err)
	}
}

// mksocket binds a unix socket at name beneath root. The path is bound
// relative to root, from inside it, so a long temporary directory does not
// overrun sun_path.
func mksocket(t *testing.T, root, name string) {
	t.Helper()
	t.Chdir(root)
	l, err := net.Listen("unix", name)
	if err != nil {
		t.Skipf("cannot bind a unix socket here: %v", err)
	}
	t.Cleanup(func() { l.Close() })
}

func TestFIFORefusedUnopened(t *testing.T) {
	root := dir(t)
	mkfifo(t, root, "input.json")
	d := mustOpenDir(t, root)
	mustNotOpen(t, d)
	_, err := d.ReadFile("input.json", 1<<20)
	wantReason(t, err, NotRegularFile, root, "input.json")
	e, err := d.Lstat("input.json")
	if err != nil || e.Kind != KindOther {
		t.Errorf("Lstat = %v, %v, want KindOther", e, err)
	}
}

func TestSocketRefusedUnopened(t *testing.T) {
	root := dir(t)
	mksocket(t, root, "sock")
	d := mustOpenDir(t, root)
	mustNotOpen(t, d)
	_, err := d.ReadFile("sock", 1<<20)
	wantReason(t, err, NotRegularFile, root)
}

// /dev/null and /dev/zero are character devices on every unix host; read
// as a data directory's files, both are refused unopened. /dev/zero opened
// would yield bytes without end.
func TestDeviceRefusedUnopened(t *testing.T) {
	// The mode alone refuses a device, wherever it is.
	for _, p := range []string{"/dev/null", "/dev/zero"} {
		info, err := os.Lstat(p)
		if err != nil {
			t.Skipf("no %s here: %v", p, err)
		}
		if k, r := kindOf(info.Mode()), fileRefusal(info.Mode()); k != KindOther || r != NotRegularFile {
			t.Errorf("%s: kind %v, refusal %q; want KindOther and not_regular_file", p, k, r)
		}
	}
}

// The same devices read through a Dir on /dev.
func TestDeviceInADirRefusedUnopened(t *testing.T) {
	d, err := OpenDir("/dev")
	if err != nil {
		// A sandbox may refuse /dev as a root; TestDeviceRefusedUnopened
		// has still held the mode to its refusal.
		t.Skipf("cannot open /dev as a root here: %v", err)
	}
	defer d.Close()
	mustNotOpen(t, d)
	for _, name := range []string{"null", "zero"} {
		e, err := d.Lstat(name)
		if err != nil {
			t.Skipf("no /dev/%s here: %v", name, err)
		}
		if e.Kind != KindOther {
			t.Fatalf("/dev/%s Lstat kind %v, want KindOther", name, e.Kind)
		}
		_, err = d.ReadFile(name, 1<<20)
		wantReason(t, err, NotRegularFile, "/dev")
	}
}

// The caller's own package operand must be a regular file too: a FIFO or a
// device named on the command line is refused from its Stat, before an
// open that would block or never end.
func TestOpenFileRefusesDevices(t *testing.T) {
	root := dir(t)
	mkfifo(t, root, "p.metais")
	for _, p := range []string{filepath.Join(root, "p.metais"), "/dev/zero"} {
		_, err := OpenFile(p)
		wantReason(t, err, NotRegularFile, root)
	}
}

// A regular file swapped for a FIFO between Lstat and the open does not
// block the tool: the open is non-blocking, and the handle's Stat refuses
// what it opened.
func TestReadSwappedForFIFODoesNotBlock(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"input.json": "{}"})
	d := mustOpenDir(t, root)
	d.beforeOpen = func(name string) {
		p := filepath.Join(root, name)
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
		if err := syscall.Mkfifo(p, 0o644); err != nil {
			t.Skipf("cannot make a FIFO here: %v", err)
		}
	}
	done := make(chan error, 1)
	go func() {
		_, err := d.ReadFile("input.json", 1<<20)
		done <- err
	}()
	// A blocked open never returns; the test binary's own timeout is the
	// backstop, and an answer at all is the property.
	err := <-done
	wantReason(t, err, Changed, root)
}

// The source tree refuses a FIFO and a socket anywhere in the layout's
// roots without opening either, and opens one that holds them only outside
// the roots.
func TestSourceRefusesDevices(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"metais.yaml": "metais: '1.0'\n", "templates/a.liquid": "a"})
	mkfifo(t, root, "templates/pipe")
	if err := os.Mkdir(filepath.Join(root, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	mksocket(t, root, "assets/sock")
	_, err := OpenSource(root, sourceOptions(t))
	var se *SourceError
	if !errors.As(err, &se) {
		t.Fatalf("OpenSource = %v, want a *SourceError", err)
	}
	want := []Problem{
		{Kind: ProblemNonRegularFile, Path: "assets/sock"},
		{Kind: ProblemNonRegularFile, Path: "templates/pipe"},
	}
	wantProblems(t, se, want)

	outside := dir(t)
	files(t, outside, map[string]string{"metais.yaml": "metais: '1.0'\n", ".git/HEAD": "ref"})
	mkfifo(t, outside, ".git/fifo")
	mkfifo(t, outside, "notes")
	s, err := OpenSource(outside, sourceOptions(t))
	if err != nil {
		t.Fatalf("a FIFO outside the layout's roots refused the tree: %v", err)
	}
	s.Close()
}

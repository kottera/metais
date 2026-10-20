// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package hostfs

import (
	"os"
	"path/filepath"
	"testing"
)

// The device cases of the hostile corpus's inputs (ADR-009 D17 item 9):
// a device cannot be created without privilege, so these read the host's
// own, beside device_unix_test.go's /dev/null and /dev/zero. Every one is
// refused from its mode without being opened: /dev/tty opened would take
// the terminal, /dev/random and /dev/urandom yield bytes without end, and
// a block device is a disk.

// hostileDevices are device paths a unix host commonly has: character
// devices that never end or that reach the terminal, and block devices.
var hostileDevices = []string{
	"/dev/tty", "/dev/random", "/dev/urandom", "/dev/full",
	"/dev/loop0", "/dev/sda", "/dev/vda", "/dev/nvme0n1", "/dev/disk0",
}

// TestHostileDevicesRefusedByMode holds every device of hostileDevices
// the host has, character or block, to KindOther and not_regular_file
// from its Lstat mode alone, which is what a data directory's read and a
// collection's listing decide by. At least one must be present.
func TestHostileDevicesRefusedByMode(t *testing.T) {
	var seen int
	for _, p := range hostileDevices {
		info, err := os.Lstat(p)
		if err != nil || info.Mode()&os.ModeDevice == 0 {
			continue
		}
		seen++
		if k, r := kindOf(info.Mode()), fileRefusal(info.Mode()); k != KindOther || r != NotRegularFile {
			t.Errorf("%s (%v): kind %v, refusal %q; want KindOther and not_regular_file", p, info.Mode(), k, r)
		}
	}
	if seen == 0 {
		t.Skip("none of the devices is present on this host")
	}
}

// TestHostileDevicesInADirRefusedUnopened reads the same devices through a
// Dir on /dev, as a data directory's entries, and refuses each unopened.
func TestHostileDevicesInADirRefusedUnopened(t *testing.T) {
	d, err := OpenDir("/dev")
	if err != nil {
		// A sandbox may refuse /dev as a root; the mode test has still
		// held each device to its refusal.
		t.Skipf("cannot open /dev as a root here: %v", err)
	}
	defer d.Close()
	mustNotOpen(t, d)
	for _, p := range hostileDevices {
		name := filepath.Base(p)
		e, err := d.Lstat(name)
		if err != nil || e.Kind != KindOther {
			continue
		}
		_, err = d.ReadFile(name, 1<<20)
		wantReason(t, err, NotRegularFile, "/dev")
	}
}

// TestHostileDeviceListedUnopened lists /dev as a collection's directory
// is listed: every device is listed as KindOther, and reading any of them
// is refused unopened.
func TestHostileDeviceListedUnopened(t *testing.T) {
	d, err := OpenDir("/dev")
	if err != nil {
		t.Skipf("cannot open /dev as a root here: %v", err)
	}
	defer d.Close()
	es, err := d.List(".", 1<<16)
	if err != nil {
		t.Skipf("cannot list /dev here: %v", err)
	}
	mustNotOpen(t, d)
	var devices int
	for _, e := range es {
		info, err := os.Lstat(filepath.Join("/dev", e.Name))
		if err != nil || info.Mode()&os.ModeDevice == 0 {
			continue
		}
		devices++
		if e.Kind != KindOther {
			t.Errorf("/dev/%s is listed as %v, want KindOther", e.Name, e.Kind)
		}
		_, err = d.ReadFile(e.Name, 1<<20)
		wantReason(t, err, NotRegularFile, "/dev")
	}
	if devices == 0 {
		t.Error("/dev lists no device")
	}
}

// TestHostileLinkToDeviceRefusedUnfollowed is a data directory's entry,
// and a collection's directory, that are links to a device and to /dev:
// each is refused as a link, from its own mode, and neither is followed
// nor opened.
func TestHostileLinkToDeviceRefusedUnfollowed(t *testing.T) {
	root := dir(t)
	for name, target := range map[string]string{"input.json": "/dev/zero", "tty.json": "/dev/tty", "items": "/dev"} {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			t.Skipf("cannot make a link here: %v", err)
		}
	}
	d := mustOpenDir(t, root)
	mustNotOpen(t, d)
	for _, name := range []string{"input.json", "tty.json"} {
		e, err := d.Lstat(name)
		if err != nil || e.Kind != KindLink {
			t.Errorf("Lstat(%q) = %v, %v; want KindLink", name, e, err)
		}
		_, err = d.ReadFile(name, 1<<20)
		wantReason(t, err, Link, root)
	}
	_, err := d.List("items", 1<<16)
	wantReason(t, err, Link, root)
}

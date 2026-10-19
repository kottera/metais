// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"errors"
	"strings"
)

// RemoveFile removes the regular file name beneath the directory at dir,
// the caller's own name for it, through an os.Root on that directory: the
// one removal an authoring command makes, test --update's of a file its
// render no longer produces from examples/<name>/expected/ (ADR-007 D5
// item 5, RULINGS R1-006). It is its own operation: no write shape uses it.
//
// dir may be a symbolic link to a directory, because the caller named it.
// Beneath it, every segment of name is examined with Lstat through the
// root: a link, a junction or another reparse point at any segment is
// refused with Link, a segment above the last that is not a directory with
// NotDirectory, and a last segment that is a directory or another file
// that is not regular with IsDirectory or NotRegularFile. The removal
// itself is made relative to a handle on the file's directory, opened
// through the root, with a call that refuses a directory, so it never
// reaches outside dir and never removes a directory.
func RemoveFile(dir, name string) error {
	root, err := openRoot(dir)
	if err != nil {
		return opAs("remove", err)
	}
	defer root.Close()
	n, ok := osName(name)
	if !ok || name == "." {
		return fail("remove", BadName)
	}
	segs := strings.Split(name, "/")
	for k := range segs {
		sub, _ := osName(strings.Join(segs[:k+1], "/"))
		info, err := root.Lstat(sub)
		if err != nil {
			return mapErr("remove", err)
		}
		if linkLike(info) {
			return fail("remove", Link)
		}
		if k < len(segs)-1 {
			if !info.IsDir() {
				return fail("remove", NotDirectory)
			}
			continue
		}
		if r := fileRefusal(info.Mode()); r != "" {
			return fail("remove", r)
		}
	}
	w := &Writer{root: root}
	parent, base := split(name)
	d, path, err := w.openDirAt(parent)
	if err != nil {
		return opAs("remove", err)
	}
	defer d.Close()
	err = platformUnlink(d, path, base)
	if errors.Is(err, errHandleUnsupported) {
		err = root.Remove(n)
	}
	return mapErr("remove", err)
}

// RemoveEmptyDir removes the directory name beneath the directory at dir,
// the caller's own name for it, when it holds no entry, and leaves it when
// it holds any: test --update's removal of a directory of
// examples/<name>/expected/ that its removals of stale files emptied, since
// a source directory holds no empty directory (ADR-001 D2 item 6). It is
// confined as RemoveFile is: dir may be a symbolic link, and beneath it
// every segment of name is examined with Lstat through a root on dir, a
// link, a junction or another reparse point at any segment refused with
// Link and a segment that is not a directory with NotDirectory. The
// removal is made relative to a handle on its parent, opened through the
// root, with a call that refuses anything but an empty directory, so it
// never reaches outside dir and never removes a file or an entry inside
// the directory.
func RemoveEmptyDir(dir, name string) error {
	root, err := openRoot(dir)
	if err != nil {
		return opAs("remove", err)
	}
	defer root.Close()
	n, ok := osName(name)
	if !ok || name == "." {
		return fail("remove", BadName)
	}
	segs := strings.Split(name, "/")
	for k := range segs {
		sub, _ := osName(strings.Join(segs[:k+1], "/"))
		info, err := root.Lstat(sub)
		if err != nil {
			return mapErr("remove", err)
		}
		if linkLike(info) {
			return fail("remove", Link)
		}
		if !info.IsDir() {
			return fail("remove", NotDirectory)
		}
	}
	w := &Writer{root: root}
	parent, base := split(name)
	d, path, err := w.openDirAt(parent)
	if err != nil {
		return opAs("remove", err)
	}
	defer d.Close()
	err = platformRmdir(d, path, base)
	if errors.Is(err, errHandleUnsupported) {
		err = root.Remove(n)
	}
	if err != nil && notEmpty(err) {
		return nil
	}
	return mapErr("remove", err)
}

// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package pkgtreetest

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"strconv"
	"strings"

	"metais.org/metais/internal/pkgtree"
)

// Check holds t to the pkgtree.Tree contract, given want, the entries t
// should hold: each key an entry path, each value its bytes. It returns nil
// when t meets every clause, and otherwise one error for every breach it
// found, joined.
//
// It checks that List returns exactly want's paths, once each, in byte
// order, well shaped and with no path a directory prefix of another, in a
// slice the caller owns; that Stat and Open agree with want and with each
// other, and Open's readers are independent and end at io.EOF; and that
// every lookup of a path that is not an entry is refused with a
// *pkgtree.PathError wrapping pkgtree.ErrNotExist. The refused paths are
// derived from want's: each path's directory prefixes, its case variants,
// and its malformed spellings ("/a", "a/", "./a", "a/../a", "a\b").
//
// Every implementation of pkgtree.Tree runs Check over a tree built for the
// purpose, with want the entries it was built from.
func Check(t pkgtree.Tree, want map[string][]byte) error {
	c := &checker{t: t}
	paths := make([]string, 0, len(want))
	for p := range want {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	c.list(want, paths)
	for _, p := range paths {
		c.entry(p, want[p])
	}
	for _, p := range refusalProbes(paths) {
		c.refused(p)
	}
	return errors.Join(c.errs...)
}

type checker struct {
	t    pkgtree.Tree
	errs []error
}

func (c *checker) fail(msg string) {
	c.errs = append(c.errs, errors.New("pkgtreetest: "+msg))
}

// list checks List, given want and its paths in byte order, the order in
// which a missing entry is reported.
func (c *checker) list(want map[string][]byte, paths []string) {
	got := c.t.List()
	for i, p := range got {
		if i > 0 && got[i-1] >= p {
			c.fail("List: " + q(got[i-1]) + " is not before " + q(p) + " in byte order, or is listed twice")
		}
	}
	for _, err := range pathProblems(slices.Compact(slices.Sorted(slices.Values(got)))) {
		c.fail("List: " + strings.TrimPrefix(err.Error(), "pkgtreetest: "))
	}
	for _, p := range got {
		if _, ok := want[p]; !ok {
			c.fail("List: " + q(p) + " is listed but is not a wanted entry")
		}
	}
	for _, p := range paths {
		if !slices.Contains(got, p) {
			c.fail("List: wanted entry " + q(p) + " is not listed")
		}
	}

	// The slice is the caller's: spoiling every element of it must not
	// reach the next List.
	before := slices.Clone(got)
	for i := range got {
		got[i] = "\x00spoiled"
	}
	if again := c.t.List(); !slices.Equal(again, before) {
		c.fail("List: changing the slice List returned changed what List returns next")
	}
}

// entry checks Stat and Open of the wanted entry p, whose bytes are b.
func (c *checker) entry(p string, b []byte) {
	info, err := c.t.Stat(p)
	switch {
	case err != nil:
		c.fail("Stat(" + q(p) + "): " + err.Error())
	case info.Path != p:
		c.fail("Stat(" + q(p) + ").Path = " + q(info.Path))
	case info.Size != int64(len(b)):
		c.fail("Stat(" + q(p) + ").Size = " + strconv.FormatInt(info.Size, 10) + ", want " + strconv.Itoa(len(b)))
	}

	// Two readers, interleaved: the first reads its first half, the second
	// reads to the end a byte at a time, then the first reads its rest.
	r1, ok := c.open(p)
	if !ok {
		return
	}
	defer func() {
		if err := r1.Close(); err != nil {
			c.fail("Open(" + q(p) + "): Close: " + err.Error())
		}
	}()
	r2, ok := c.open(p)
	if !ok {
		return
	}
	half := len(b) / 2
	head, err := readAll(io.LimitReader(r1, int64(half)), half, 4096)
	if err != nil {
		c.fail("Open(" + q(p) + "): reading the first " + strconv.Itoa(half) + " bytes: " + err.Error())
		return
	}
	whole, err := readAll(r2, len(b), 1)
	if err != nil {
		c.fail("Open(" + q(p) + "): reading a byte at a time: " + err.Error())
	} else if !bytes.Equal(whole, b) {
		c.fail("Open(" + q(p) + "): a second reader, read while the first was open, yields " + strconv.Itoa(len(whole)) + " bytes that are not the entry's")
	}
	if err := r2.Close(); err != nil {
		c.fail("Open(" + q(p) + "): Close: " + err.Error())
	}
	tail, err := readAll(r1, len(b)-half, 4096)
	if err != nil {
		c.fail("Open(" + q(p) + "): reading to the end: " + err.Error())
		return
	}
	if got := append(head, tail...); !bytes.Equal(got, b) {
		c.fail("Open(" + q(p) + "): yields " + strconv.Itoa(len(got)) + " bytes that are not the entry's")
	}
	if n, err := r1.Read(make([]byte, 1)); n != 0 || err != io.EOF {
		c.fail("Open(" + q(p) + "): Read after the end returned " + strconv.Itoa(n) + " bytes and " + errText(err) + ", want 0 and EOF")
	}
}

func (c *checker) open(p string) (io.ReadCloser, bool) {
	r, err := c.t.Open(p)
	if err != nil {
		c.fail("Open(" + q(p) + "): " + err.Error())
		return nil, false
	}
	if r == nil {
		c.fail("Open(" + q(p) + ") returned a nil reader and no error")
		return nil, false
	}
	return r, true
}

// readAll reads r to io.EOF, chunk bytes at a time, and refuses to read
// more than one byte past size or to call Read without end, so a reader
// that never ends cannot hang the check.
func readAll(r io.Reader, size, chunk int) ([]byte, error) {
	var got []byte
	buf := make([]byte, chunk)
	for calls := 0; calls <= size+64; calls++ {
		n, err := r.Read(buf)
		got = append(got, buf[:n]...)
		if len(got) > size {
			return got, errors.New("yields more than the entry's " + strconv.Itoa(size) + " bytes")
		}
		if err == io.EOF {
			return got, nil
		}
		if err != nil {
			return got, err
		}
	}
	return got, errors.New("does not reach EOF")
}

// refused checks that p, which is not an entry, is refused by Stat and
// Open as the contract requires.
func (c *checker) refused(p string) {
	info, err := c.t.Stat(p)
	if why := refusalProblem("stat", p, err); why != "" {
		c.fail("Stat(" + q(p) + "), not an entry, " + why)
	}
	if info != (pkgtree.Info{}) {
		c.fail("Stat(" + q(p) + "), not an entry, returned an Info beside its error")
	}
	r, err := c.t.Open(p)
	if r != nil {
		r.Close()
		c.fail("Open(" + q(p) + "), not an entry, returned a reader")
	}
	if why := refusalProblem("open", p, err); why != "" {
		c.fail("Open(" + q(p) + "), not an entry, " + why)
	}
}

// refusalProblem says how err falls short of the refusal op must give for
// p, or returns "" if it does not.
func refusalProblem(op, p string, err error) string {
	if err == nil {
		return "succeeded"
	}
	if !errors.Is(err, pkgtree.ErrNotExist) {
		return "returned " + q(err.Error()) + ", which does not wrap pkgtree.ErrNotExist"
	}
	var pe *pkgtree.PathError
	if !errors.As(err, &pe) {
		return "returned " + q(err.Error()) + ", which is not a *pkgtree.PathError"
	}
	if pe.Op != op || pe.Path != p {
		return "returned a *pkgtree.PathError for " + q(pe.Op) + " of " + q(pe.Path)
	}
	return ""
}

// refusalProbes returns, in byte order and once each, paths that are not
// entries of a tree whose entries are paths but that a lookup by
// normalisation, case folding, path cleaning or a file system would take
// for one.
func refusalProbes(paths []string) []string {
	probes := []string{"", ".", "..", "/", "\x00", "\xff", "no-such-entry"}
	for _, p := range paths {
		probes = append(probes,
			"/"+p, p+"/", "./"+p, p+"/.", p+"/..", "x/../"+p,
			strings.ReplaceAll(p, "/", "\\"), strings.ReplaceAll(p, "/", "//"),
			p+"\x00", " "+p, p+" ", "\uFEFF"+p,
			strings.ToUpper(p), strings.ToLower(p),
		)
		for i := range len(p) {
			if p[i] == '/' {
				probes = append(probes, p[:i], p[:i+1])
			}
		}
	}
	slices.Sort(probes)
	probes = slices.Compact(probes)
	return slices.DeleteFunc(probes, func(probe string) bool {
		_, found := slices.BinarySearch(paths, probe)
		return found
	})
}

func q(s string) string {
	return strconv.Quote(s)
}

func errText(err error) string {
	if err == nil {
		return "no error"
	}
	return q(err.Error())
}

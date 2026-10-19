// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package zipfmt

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/kottera/metais/internal/limits"
)

// counting is an io.ReaderAt that counts how many times each byte of b is
// read.
type counting struct {
	b     []byte
	reads []int
}

func newCounting(b []byte) *counting { return &counting{b: b, reads: make([]int, len(b))} }

func (c *counting) ReadAt(p []byte, off int64) (int, error) {
	n, err := bytes.NewReader(c.b).ReadAt(p, off)
	for i := range n {
		c.reads[off+int64(i)]++
	}
	return n, err
}

// readOnce says how c's file was read: nil when every byte was read
// exactly once.
func (c *counting) readOnce() error {
	for i, n := range c.reads {
		if n != 1 {
			return fmt.Errorf("byte %d of %d was read %d times", i, len(c.reads), n)
		}
	}
	return nil
}

// TestOpenReadsEveryByteOnce is ADR-010 D15 item 2's deterministic budget
// (W2-4.4): opening a packed package reads every byte of the file once
// and decodes every entry once. It runs over a package smaller than the
// end-record search, whose every byte is in the tail, and one larger,
// whose entries straddle the tail's start.
func TestOpenReadsEveryByteOnce(t *testing.T) {
	small := sample()
	var large []Entry
	for i := range 40 {
		content := bytes.Repeat([]byte(fmt.Sprintf("entry %d line\n", i)), 300+i*7)
		method := Auto
		if i%3 == 0 {
			method = Store
		}
		large = append(large, Entry{Name: fmt.Sprintf("assets/%02d.txt", i), Content: content, Method: method})
	}
	for name, entries := range map[string][]Entry{"small": small, "large": large} {
		b := write(t, entries)
		if (name == "large") != (len(b) > endSearch) {
			t.Fatalf("%s: %d bytes is on the wrong side of the end-record search", name, len(b))
		}
		c := newCounting(b)
		a, found, err := Open(c, int64(len(b)), limits.Defaults())
		if err != nil || len(found) > 0 {
			t.Fatalf("%s: Open: %v %q", name, err, codes(found))
		}
		if err := c.readOnce(); err != nil {
			t.Errorf("%s (%d bytes): %v", name, len(b), err)
		}
		if a.decodes != len(entries) {
			t.Errorf("%s: %d entries decoded %d times", name, len(entries), a.decodes)
		}
		// Reading an entry after open decodes it once more, and no more.
		r, _ := a.Open(entries[1].Name)
		io.Copy(io.Discard, r)
		io.Copy(io.Discard, r)
		if a.decodes != len(entries)+1 {
			t.Errorf("%s: reading one entry twice through one reader decoded %d times in all", name, a.decodes)
		}
	}
}

// TestReadOnceControl is the detector's control: a reader that reads a
// byte twice, or skips one, is caught.
func TestReadOnceControl(t *testing.T) {
	b := write(t, sample())
	c := newCounting(b)
	c.ReadAt(make([]byte, len(b)), 0)
	c.ReadAt(make([]byte, 1), 7)
	if c.readOnce() == nil {
		t.Error("a byte read twice is not caught")
	}
	c = newCounting(b)
	c.ReadAt(make([]byte, len(b)-1), 0)
	if c.readOnce() == nil {
		t.Error("a byte never read is not caught")
	}
}

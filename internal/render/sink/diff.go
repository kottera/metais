// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sink

import (
	"strconv"
	"strings"
)

// diffContext is the lines of context around each change in a unified
// diff (ADR-007 D5 item 4).
const diffContext = 3

// maxEditDistance bounds the line edits the diff searches for. Beyond it
// the lines between the common head and tail are shown as all removed and
// all added: the diff stays correct, and its work stays bounded whatever
// the files hold.
const maxEditDistance = 512

// edit is one line of an edit script: ' ' kept, '-' removed from the
// expected file, '+' added by the rendered one, with the number of lines of
// each file before it.
type edit struct {
	op     byte
	line   string
	before [2]int
}

// unifiedDiff is a unified diff from want, the file labelled from, to got,
// labelled to: the ---, +++ and @@ lines and three lines of context. A
// line without a final line feed is followed by "\ No newline at end of
// file". It stops once the diff is longer than max bytes and reports that
// it did; the caller cuts it at max.
func unifiedDiff(want, got []byte, from, to string, max int) (string, bool) {
	ed := editScript(splitLines(string(want)), splitLines(string(got)))
	var b strings.Builder
	b.WriteString("--- " + from + "\n+++ " + to + "\n")
	for i := 0; i < len(ed); {
		if ed[i].op == ' ' {
			i++
			continue
		}
		start, end := max0(i-diffContext), i+1
		for j := i + 1; j < len(ed); {
			if ed[j].op != ' ' {
				end, j = j+1, j+1
				continue
			}
			k := j
			for k < len(ed) && ed[k].op == ' ' {
				k++
			}
			if k == len(ed) || k-j > 2*diffContext {
				break
			}
			j = k
		}
		stop := min(len(ed), end+diffContext)
		var count [2]int
		for _, e := range ed[start:stop] {
			if e.op != '+' {
				count[0]++
			}
			if e.op != '-' {
				count[1]++
			}
		}
		at := ed[start].before
		for side := range at {
			if count[side] > 0 {
				at[side]++
			}
		}
		b.WriteString("@@ -" + hunkRange(at[0], count[0]) + " +" + hunkRange(at[1], count[1]) + " @@\n")
		for _, e := range ed[start:stop] {
			b.WriteByte(e.op)
			b.WriteString(e.line)
			if !strings.HasSuffix(e.line, "\n") {
				b.WriteString("\n\\ No newline at end of file\n")
			}
			if b.Len() > max {
				return b.String(), true
			}
		}
		i = stop
	}
	return b.String(), false
}

func max0(n int) int { return max(n, 0) }

// hunkRange is a hunk header's start,count, the count omitted when it is 1.
func hunkRange(start, count int) string {
	if count == 1 {
		return strconv.Itoa(start)
	}
	return strconv.Itoa(start) + "," + strconv.Itoa(count)
}

// splitLines splits s after each line feed, keeping it, so a CR before it
// stays part of its line and a missing final line feed shows.
func splitLines(s string) []string {
	lines := strings.SplitAfter(s, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// editScript is an edit script from a to b: their common head and tail
// kept, and between them the shortest script Myers' algorithm finds within
// maxEditDistance edits, or every line removed and then every line added.
func editScript(a, b []string) []edit {
	head := 0
	for head < len(a) && head < len(b) && a[head] == b[head] {
		head++
	}
	tail := 0
	for tail < len(a)-head && tail < len(b)-head && a[len(a)-1-tail] == b[len(b)-1-tail] {
		tail++
	}
	am, bm := a[head:len(a)-tail], b[head:len(b)-tail]
	ops, ok := myers(am, bm)
	if !ok {
		ops = ops[:0]
		for range am {
			ops = append(ops, '-')
		}
		for range bm {
			ops = append(ops, '+')
		}
	}
	var out []edit
	x, y := 0, 0
	add := func(op byte) {
		e := edit{op: op, before: [2]int{x, y}}
		switch op {
		case ' ':
			e.line = a[x]
			x, y = x+1, y+1
		case '-':
			e.line = a[x]
			x++
		case '+':
			e.line = b[y]
			y++
		}
		out = append(out, e)
	}
	for range head {
		add(' ')
	}
	for _, op := range ops {
		add(op)
	}
	for range tail {
		add(' ')
	}
	return out
}

// myers is the shortest edit script from a to b, as ' ', '-' and '+', by
// Myers' O(ND) algorithm, or false when it needs more than
// maxEditDistance edits.
func myers(a, b []string) ([]byte, bool) {
	n, m := len(a), len(b)
	if n == 0 && m == 0 {
		return nil, true
	}
	off := maxEditDistance + 1
	v := make([]int, 2*maxEditDistance+3)
	var trace [][]int
	for d := 0; d <= maxEditDistance; d++ {
		trace = append(trace, append([]int(nil), v[off-d-1:off+d+2]...))
		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || k != d && v[off+k-1] < v[off+k+1] {
				x = v[off+k+1]
			} else {
				x = v[off+k-1] + 1
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x, y = x+1, y+1
			}
			v[off+k] = x
			if x >= n && y >= m {
				return backtrack(trace, n, m), true
			}
		}
	}
	return nil, false
}

// backtrack reads the edit script from the frontiers myers recorded, the
// d-th as it stood when the d-th round began.
func backtrack(trace [][]int, n, m int) []byte {
	var ops []byte
	x, y := n, m
	for d := len(trace) - 1; d >= 0; d-- {
		vd := trace[d]
		at := func(k int) int { return vd[k+d+1] }
		k := x - y
		prev := k - 1
		if k == -d || k != d && at(k-1) < at(k+1) {
			prev = k + 1
		}
		px := at(prev)
		py := px - prev
		for x > px && y > py {
			ops = append(ops, ' ')
			x, y = x-1, y-1
		}
		if d > 0 {
			if x == px {
				ops = append(ops, '+')
			} else {
				ops = append(ops, '-')
			}
		}
		x, y = px, py
	}
	for i, j := 0, len(ops)-1; i < j; i, j = i+1, j-1 {
		ops[i], ops[j] = ops[j], ops[i]
	}
	return ops
}

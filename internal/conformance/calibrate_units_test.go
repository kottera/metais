// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// renderWorkUnitCases are render-work's costliest units (ADR-009
// Amendment A13): each repeats one unit whose CPU time is the most a unit
// of its kind costs, until render-work refuses the render. The
// render-work case is the cheapest node over and over; these hold the
// default against the costliest, which is what a refusal at the default
// costs. Each builds its value once, under the render-work it leaves, and
// then spends the rest of the limit on the unit.
var renderWorkUnitCases = []calCase{
	{
		// A range of array-size elements materialised by assign: one unit
		// an element, and an element is a number made and boxed.
		name: "render-work-range", limit: "render-work",
		lowered: calLimits{"render-work": 268_435, "array-size": 1_000},
		build: func(dir string, lim calLimits) ([]string, error) {
			return renderTemplate(dir, fmt.Sprintf("{%% for i in (1..%d) %%}{%% assign r = (1..%d) %%}{%% endfor %%}",
				lim["render-work"]/lim["array-size"]+1, lim["array-size"]))
		},
		reached: refusedBy("render-work"),
	},
	{
		// An array of empty strings written whole: one unit an element,
		// and no byte of output to bound it otherwise.
		name: "render-work-write", limit: "render-work",
		lowered: calLimits{"render-work": 268_435, "array-size": 1_000},
		build: func(dir string, lim calLimits) ([]string, error) {
			commas := lim["array-size"] / 2
			return renderTemplate(dir, doubled("c", ",", commas)+`{% assign r = "x" | append: c | append: "x" | split: "," %}`+
				fmt.Sprintf("{%% for i in (1..%d) %%}{{ r }}{%% endfor %%}", lim["render-work"]/commas+1))
		},
		reached: refusedBy("render-work"),
	},
	{
		// size of a string of two-byte characters, the costliest to count
		// of the widths a character has: one unit a byte.
		name: "render-work-size", limit: "render-work",
		lowered: calLimits{"render-work": 268_435, "value-size": 64 << 10},
		build: func(dir string, lim calLimits) ([]string, error) {
			n := lim["value-size"] / 16
			return renderTemplate(dir, doubled("s", "é", n)+
				fmt.Sprintf("{%% for i in (1..%d) %%}{%% assign n = s.size %%}{%% endfor %%}", lim["render-work"]/n+1))
		},
		reached: refusedBy("render-work"),
	},
	{
		// split of a text of commas into empty strings, over and over.
		name: "render-work-split", limit: "render-work",
		lowered: calLimits{"render-work": 268_435, "array-size": 1_000},
		build: func(dir string, lim calLimits) ([]string, error) {
			commas := lim["array-size"] / 2
			return renderTemplate(dir, doubled("c", ",", commas)+`{% assign s = "x" | append: c | append: "x" %}`+
				fmt.Sprintf(`{%% for i in (1..%d) %%}{%% assign r = s | split: "," %%}{%% endfor %%}`, lim["render-work"]/commas+1))
		},
		reached: refusedBy("render-work"),
	},
	{
		// sum of array-size integers, the costliest terms to add now that
		// a string term is charged its bytes.
		name: "render-work-sum", limit: "render-work",
		lowered: calLimits{"render-work": 268_435, "array-size": 1_000},
		build: func(dir string, lim calLimits) ([]string, error) {
			return renderTemplate(dir, fmt.Sprintf("{%% assign r = (1..%d) %%}{%% for i in (1..%d) %%}{%% assign t = r | sum %%}{%% endfor %%}",
				lim["array-size"], lim["render-work"]/lim["array-size"]+1))
		},
		reached: refusedBy("render-work"),
	},
	{
		// uniq of array-size distinct integers: every element a new key.
		name: "render-work-uniq", limit: "render-work",
		lowered: calLimits{"render-work": 268_435, "array-size": 1_000},
		build: func(dir string, lim calLimits) ([]string, error) {
			return renderTemplate(dir, fmt.Sprintf("{%% assign r = (1..%d) %%}{%% for i in (1..%d) %%}{%% assign t = r | uniq %%}{%% endfor %%}",
				lim["array-size"], lim["render-work"]/lim["array-size"]+1))
		},
		reached: refusedBy("render-work"),
	},
	{
		// slug of thousands of code points that have a decomposition, once
		// each, so each call decomposes all of them; a letter at the end
		// keeps the slug from being empty.
		name: "render-work-slug", limit: "render-work",
		lowered: calLimits{"render-work": 268_435},
		build: func(dir string, lim calLimits) ([]string, error) {
			s := decomposing() + "a"
			return renderTemplate(dir, fmt.Sprintf(`{%% assign s = "%s" %%}{%% for i in (1..%d) %%}{%% assign t = s | slug %%}{%% endfor %%}`,
				s, lim["render-work"]/int64(len(s))+1))
		},
		reached: refusedBy("render-work"),
	},
	{
		// strip_html of tags with nothing between them, the most matches a
		// byte of text holds.
		name: "render-work-strip-html", limit: "render-work",
		lowered: calLimits{"render-work": 268_435, "value-size": 64 << 10},
		build: func(dir string, lim calLimits) ([]string, error) {
			n := lim["value-size"] / 16
			return renderTemplate(dir, doubled("s", "<a>", n)+
				fmt.Sprintf("{%% for i in (1..%d) %%}{%% assign t = s | strip_html %%}{%% endfor %%}", lim["render-work"]/n+1))
		},
		reached: refusedBy("render-work"),
	},
}

// decomposing is the code points of the blocks in which nearly every
// code point has a decomposition: Latin letters with marks, enclosed
// alphanumerics, CJK compatibility squares, ligatures and Arabic
// presentation forms, half- and full-width forms, mathematical letters and
// CJK compatibility ideographs. They are named here, not read from a
// normalization library: the runner drives the tool as a black box
// (ADR-011 D11 item 2), and normalization is internal/unitext's alone
// (ADR-010 D11 item 7).
func decomposing() string {
	var b strings.Builder
	for _, block := range [][2]rune{
		{0x00C0, 0x017F}, {0x1E00, 0x1EFF}, {0x2460, 0x24FF}, {0x3300, 0x33FF},
		{0xF900, 0xFAFF}, {0xFB00, 0xFDFF}, {0xFE70, 0xFEFF}, {0xFF01, 0xFFEE},
		{0x1D400, 0x1D7FF},
	} {
		for r := block[0]; r <= block[1]; r++ {
			if utf8.ValidRune(r) {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

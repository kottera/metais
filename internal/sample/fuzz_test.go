// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sample

import (
	"errors"
	"regexp/syntax"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/inputs/csv"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/regexprofile"
	"github.com/kottera/metais/internal/testenv"
)

// FuzzRoundTrip is W3-5.4's property over any schema: whatever the schema
// file, the seed, the mode, the item count and the form, the generator
// does not panic, gives the same value twice, and a value Files writes
// reads back through the form's reader as the same value. Its input-size
// is lowered, so that a schema asking for much is refused quickly. Its
// seed corpus, under testdata/fuzz, runs in every go test (W3-5.5).
func FuzzRoundTrip(f *testing.F) {
	testenv.Home(f)
	lim := limits.Defaults()
	if _, ok := lim.Lower("input-size=65536"); !ok {
		f.Fatal("input-size cannot be lowered")
	}
	forms := []Form{FormJSON, FormYAML, FormCSV, FormText, FormMarkdown}
	delims := []csv.Delimiter{csv.Comma, csv.Semicolon, csv.Pipe, csv.Tab}
	f.Fuzz(func(t *testing.T, schemaText []byte, seed uint64, mode, items, form uint8, collection bool) {
		p, fs := compileErr(t, map[string]string{entry: string(schemaText)})
		if len(fs) > 0 || p.c == nil {
			return
		}
		in := Input{
			Name: "data", Schema: entry, Forms: []Form{forms[int(form)%len(forms)]},
			Collection: collection, Delimiter: delims[int(form/8)%len(delims)], Required: true,
		}
		opts := Options{Seed: seed, Optional: Mode(mode % 3), Items: int(items % 5)}
		gen := func() (value.Value, []finding.Finding, error) {
			g := New(p.c, func(e string) finding.Positions { return p.pos[e] }, lim, opts)
			v, _, fs, err := g.Generate(in)
			return v, fs, err
		}
		v, fs, err := gen()
		again, fs2, err2 := gen()
		if (err == nil) != (err2 == nil) || len(fs) != len(fs2) || text(v) != text(again) {
			t.Fatalf("two runs differ: %s, then %s", text(v), text(again))
		}
		if err != nil || len(fs) > 0 {
			return
		}
		written, err := New(p.c, nil, lim, opts).Files(in, v)
		var ue *UnwritableError
		if errors.As(err, &ue) {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		if back := readBack(t, p, in, written); !value.Equal(back, v) {
			t.Fatalf("%s reads back as %s from %q", text(v), text(back), written)
		}
	})
}

// FuzzPatternText is item 18's pattern walk over any pattern the profile
// accepts: it does not panic, gives the same string for the same stream,
// and, for a pattern with no anchor and no class that matches nothing,
// gives a string the pattern matches.
func FuzzPatternText(f *testing.F) {
	testenv.Home(f)
	g := New(nil, nil, limits.Defaults(), Options{})
	f.Fuzz(func(t *testing.T, pattern string, seed uint64) {
		re, err := regexprofile.Compile(pattern, limits.Defaults(), nil)
		if err != nil {
			return
		}
		a, okA := g.patternText(pattern, NewSplitMix64(seed), 0, 0)
		b, okB := g.patternText(pattern, NewSplitMix64(seed), 0, 0)
		if a != b || okA != okB {
			t.Fatalf("%q: %q, then %q", pattern, a, b)
		}
		if okA && plainTree(re.Syntax()) && !re.MatchString(a) {
			t.Fatalf("%q gave %q, which it does not match", pattern, a)
		}
	})
}

// plainTree reports whether re holds no anchor, no assertion and no class
// or operator that matches nothing.
func plainTree(re *syntax.Regexp) bool {
	switch re.Op {
	case syntax.OpBeginLine, syntax.OpEndLine, syntax.OpBeginText, syntax.OpEndText,
		syntax.OpWordBoundary, syntax.OpNoWordBoundary, syntax.OpNoMatch:
		return false
	case syntax.OpCharClass:
		if len(re.Rune) == 0 {
			return false
		}
	}
	for _, sub := range re.Sub {
		if !plainTree(sub) {
			return false
		}
	}
	return true
}

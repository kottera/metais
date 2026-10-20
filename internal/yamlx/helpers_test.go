// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package yamlx

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/testenv"
)

// The two families a document is read in.
var (
	manifestSrc = Source{
		Subject: finding.Subject{Kind: finding.SubjectManifest},
		At:      finding.Location{Origin: finding.OriginPackage, File: "metais.yaml"},
		Size:    limits.ManifestSize,
	}
	inputSrc = Source{
		Subject: finding.Subject{Kind: finding.SubjectInput, Name: "vendor"},
		At:      finding.Location{Origin: finding.OriginDataDir, File: "vendor.yaml"},
		Size:    limits.InputSize,
	}
)

// parse reads text under src at the default limits, with a temporary
// HOME, and checks that every position Parse gives is within text.
func parse(t *testing.T, src Source, text string) (Doc, []finding.Finding) {
	t.Helper()
	testenv.Home(t)
	return parseLim(t, src, text, limits.Defaults())
}

func parseLim(t testing.TB, src Source, text string, lim limits.Set) (Doc, []finding.Finding) {
	t.Helper()
	doc, fs := Parse([]byte(text), src, lim)
	if err := positionsWithin(src, text, doc, fs); err != nil {
		t.Fatalf("%q: %v", text, err)
	}
	if (doc.Value == nil) == (len(fs) == 0) {
		t.Fatalf("%q: a document %v with %d findings", text, doc.Value != nil, len(fs))
	}
	return doc, fs
}

// lowered is the default limits with each NAME=VALUE of args applied.
func lowered(t testing.TB, args ...string) limits.Set {
	t.Helper()
	s := limits.Defaults()
	for _, a := range args {
		if f, ok := s.Lower(a); !ok {
			t.Fatalf("--limit %s: %s", a, f.Message())
		}
	}
	return s
}

// lineLengths is the length in code points of each line of text, a line
// ending at LF, CR LF or CR.
func lineLengths(text string) []int {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	var n []int
	for _, l := range strings.Split(text, "\n") {
		n = append(n, utf8.RuneCountInString(l))
	}
	return n
}

// positionsWithin checks every position in doc's table and in fs is a
// line of text, offset by src.FirstLine, and a column of that line or
// just past its end.
func positionsWithin(src Source, text string, doc Doc, fs []finding.Finding) error {
	lens := lineLengths(text)
	first := max(src.FirstLine, 1)
	check := func(what string, p finding.Pos) error {
		if !p.Known() {
			return nil
		}
		i := p.Line - first
		if i < 0 || i >= len(lens) || p.Column < 0 || p.Column > lens[i]+1 {
			return fmt.Errorf("%s at line %d column %d is outside the text", what, p.Line, p.Column)
		}
		return nil
	}
	if doc.Value != nil {
		values, keys := doc.pos.entries()
		for ptr, p := range values {
			if p.Column < 1 {
				return fmt.Errorf("value %q has no column", ptr)
			}
			if err := check("value "+ptr, p); err != nil {
				return err
			}
		}
		for ptr, p := range keys {
			if err := check("key "+ptr, p); err != nil {
				return err
			}
		}
	}
	for _, f := range fs {
		locs := []finding.Location{}
		if l, ok := f.Location(); ok {
			locs = append(locs, l)
		}
		for _, r := range f.Related() {
			locs = append(locs, r.Location)
		}
		for _, l := range locs {
			if err := check(string(f.Code()), finding.Pos{Line: l.Line, Column: l.Column}); err != nil {
				return err
			}
		}
	}
	return nil
}

// Values for expectations.

func num(lit string) value.Value {
	n, err := value.ParseNumber(lit)
	if err != nil {
		panic(err)
	}
	return n
}

func str(s string) value.Value {
	v, err := value.NewString(s)
	if err != nil {
		panic(err)
	}
	return v
}

func arr(elems ...value.Value) value.Value {
	a, err := value.NewArray(elems...)
	if err != nil {
		panic(err)
	}
	return a
}

// obj is the object of name, value pairs.
func obj(pairs ...any) value.Value {
	var ms []value.Member
	for i := 0; i < len(pairs); i += 2 {
		ms = append(ms, value.Member{Name: pairs[i].(string), Value: pairs[i+1].(value.Value)})
	}
	o, err := value.NewObject(ms...)
	if err != nil {
		panic(err)
	}
	return o
}

// describe is fs as one line per finding: code, line:column, pointer.
func describe(fs []finding.Finding) string {
	var b strings.Builder
	for _, f := range fs {
		l, _ := f.Location()
		fmt.Fprintf(&b, "%s %d:%d", f.Code(), l.Line, l.Column)
		if l.HasPointer {
			fmt.Fprintf(&b, " %q", l.Pointer)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// entries returns every position the table holds, by pointer: the
// values', and the member names'.
func (p *positions) entries() (values, keys map[string]finding.Pos) {
	values, keys = map[string]finding.Pos{}, map[string]finding.Pos{}
	var walk func(pl *place, ptr string)
	walk = func(pl *place, ptr string) {
		if pl.hasValue {
			values[ptr] = pl.value.pos()
		}
		if pl.hasKey {
			keys[ptr] = pl.key.pos()
		}
		for tok, c := range pl.children {
			walk(c, ptr+finding.Pointer(tok))
		}
		for i, c := range pl.elems {
			if c != nil {
				walk(c, ptr+finding.Pointer(strconv.Itoa(i)))
			}
		}
	}
	walk(p.root, "")
	return values, keys
}

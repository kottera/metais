// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"bufio"
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/liquid/ast"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/testenv"
)

// goldenDir holds Golden Liquid, vendored from jg-rp/golden-liquid at
// 65c2f76 with its licence and a SOURCE file (ADR-010 D14 item 4).
const goldenDir = "testdata/golden-liquid"

// goldenCase is one case of golden_liquid.json, as far as parsing reads
// it: the template, the partials it may render, and whether Shopify
// refuses it.
type goldenCase struct {
	Name      string            `json:"name"`
	Template  string            `json:"template"`
	Templates map[string]string `json:"templates"`
	Invalid   bool              `json:"invalid"`
}

func loadGolden(t *testing.T) []goldenCase {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(goldenDir, "golden_liquid.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Tests []goldenCase `json:"tests"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Tests
}

// exclusionReasons is ADR-010 D4 item 7's closed set.
var exclusionReasons = []string{"include", "lax", "error-string", "dialect", "version"}

// listLine is one line of exclusions.txt or render-time.txt: a case name
// and the reason, tab-separated.
type listLine struct {
	name, reason, note string
	line               int
}

// readList reads a list of case names, one per line as "name<TAB>reason"
// or "name<TAB>reason<TAB>note". Blank lines and lines beginning with "#"
// are skipped.
func readList(t *testing.T, file string) map[string]listLine {
	t.Helper()
	f, err := os.Open(filepath.Join(goldenDir, file))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	out := map[string]listLine{}
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		l := sc.Text()
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		parts := strings.Split(l, "\t")
		if len(parts) < 2 || len(parts) > 3 || parts[0] == "" || parts[1] == "" {
			t.Errorf("%s:%d: want name<TAB>reason[<TAB>note], got %q", file, n, l)
			continue
		}
		e := listLine{name: parts[0], reason: parts[1], line: n}
		if len(parts) == 3 {
			e.note = parts[2]
		}
		if _, dup := out[e.name]; dup {
			t.Errorf("%s:%d: %q listed twice", file, n, e.name)
		}
		out[e.name] = e
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// goldenDialect is the dialect the corpus is parsed with: every filter
// name of Shopify's standard set at 5.14.0 (ADR-010 D4 item 2), each a
// stub that accepts every application. The catalog's own definitions,
// and their checks, are Wave 3's; parsing needs only the names.
func goldenDialect(t testing.TB) *Dialect {
	t.Helper()
	var fs []FilterDef
	for _, n := range standardFilterNames {
		fs = append(fs, stubFilter(n))
	}
	d, err := NewDialect(Spec{Filters: fs})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// standardFilterNames is ADR-010 D4 item 2's list, with escape's alias h,
// which ADR-004's Context lists.
var standardFilterNames = strings.Fields(`abs append at_least at_most
	base64_decode base64_encode base64_url_safe_decode base64_url_safe_encode
	capitalize ceil compact concat date default divided_by downcase escape h
	escape_once find find_index first floor has join last lstrip map minus
	modulo newline_to_br plus prepend reject remove remove_first remove_last
	replace replace_first replace_last reverse round rstrip size slice sort
	sort_natural split squish strip strip_html strip_newlines sum times
	truncate truncatewords uniq upcase url_decode url_encode where`)

// stubFilter is a filter that only has a name.
type stubFilter string

func (f stubFilter) Name() string           { return string(f) }
func (stubFilter) Check(*ast.Filter) *Error { return nil }
func (stubFilter) Apply(FilterContext, value.Value, []Arg) (value.Value, *Error) {
	return nil, &Error{Code: CodeInternal, Detail: "a stub filter does not render"}
}

// parseGolden parses a case's template and every partial it carries, and
// returns every error.
func parseGolden(d *Dialect, c goldenCase) []*Error {
	_, errs := Parse(d, Source{Name: "template", Text: []byte(c.Template)}, ParseLimits{})
	for _, name := range slices.Sorted(maps.Keys(c.Templates)) {
		_, e := Parse(d, Source{Name: name, Text: []byte(c.Templates[name])}, ParseLimits{})
		errs = append(errs, e...)
	}
	return errs
}

// TestGoldenLiquidParse is ADR-010 D4 item 7's breadth check, at parse:
// every case of Golden Liquid is parsed under Core(), with its partials,
// and
//
//   - a valid case parses without error;
//   - an invalid case is refused at parse, unless render-time.txt lists
//     it: those are the cases whose refusal ADR-004 places at render
//     (D6 item 8), each line naming the ADR-004 item that does. They must
//     parse, and Wave 3's render run proves their refusal;
//   - a case that cannot do either is listed in exclusions.txt with a
//     reason from D4 item 7's closed set, and must still fail, so a stale
//     line is found.
//
// No case may recover an internal error, listed or not. A case that
// applies date is selected out (goldenDateSelector), and counted.
func TestGoldenLiquidParse(t *testing.T) {
	testenv.Home(t)
	cases := loadGolden(t)
	excluded := readList(t, "exclusions.txt")
	renderTime := readList(t, "render-time.txt")
	d := Core()
	appliesDate, err := goldenDateSelector(d)
	if err != nil {
		t.Fatal(err)
	}

	names := map[string]goldenCase{}
	for _, c := range cases {
		if _, dup := names[c.Name]; dup {
			t.Fatalf("the corpus names %q twice", c.Name)
		}
		names[c.Name] = c
	}
	for _, l := range excluded {
		if !slices.Contains(exclusionReasons, l.reason) {
			t.Errorf("exclusions.txt:%d: reason %q is not one of ADR-010 D4 item 7's %q", l.line, l.reason, exclusionReasons)
		}
		if l.reason == "dialect" && !strings.HasPrefix(l.note, "ADR-004 ") {
			t.Errorf("exclusions.txt:%d: a dialect line names its ADR-004 item", l.line)
		}
		if _, ok := names[l.name]; !ok {
			t.Errorf("exclusions.txt:%d: no case %q", l.line, l.name)
		}
		if _, ok := renderTime[l.name]; ok {
			t.Errorf("exclusions.txt:%d: %q is in render-time.txt too", l.line, l.name)
		}
	}
	for _, l := range renderTime {
		if !strings.HasPrefix(l.reason, "ADR-004 ") {
			t.Errorf("render-time.txt:%d: the line names the ADR-004 item that places the refusal at render", l.line)
		}
		if c, ok := names[l.name]; !ok {
			t.Errorf("render-time.txt:%d: no case %q", l.line, l.name)
		} else if !c.Invalid {
			t.Errorf("render-time.txt:%d: %q is a valid case", l.line, l.name)
		}
	}

	var passed, refusedAtParse, atRender, skipped, selectedOut int
	for _, c := range cases {
		if appliesDate(c) {
			selectedOut++
			continue
		}
		errs := parseGolden(d, c)
		for _, e := range errs {
			if e.Code == CodeInternal {
				t.Errorf("%q: internal error: %v", c.Name, e)
			}
		}
		_, deferred := renderTime[c.Name]
		var ok bool
		switch {
		case !c.Invalid:
			ok = len(errs) == 0
		case deferred:
			ok = len(errs) == 0
		default:
			ok = len(errs) > 0
		}
		if x, isExcluded := excluded[c.Name]; isExcluded {
			if ok {
				t.Errorf("exclusions.txt:%d: %q passes; remove the line", x.line, c.Name)
			}
			skipped++
			continue
		}
		if !ok {
			switch {
			case !c.Invalid:
				t.Errorf("%q is valid and refused: %v\n%s", c.Name, errs[0], c.Template)
			case deferred:
				t.Errorf("%q is in render-time.txt and refused at parse: %v", c.Name, errs[0])
			default:
				t.Errorf("%q is invalid and parses\n%s", c.Name, c.Template)
			}
			continue
		}
		passed++
		switch {
		case c.Invalid && deferred:
			atRender++
		case c.Invalid:
			refusedAtParse++
		}
	}
	t.Logf("%d cases: %d pass (%d invalid refused at parse, %d invalid parsed for render), %d excluded, %d selected out for applying date",
		len(cases), passed, refusedAtParse, atRender, skipped, selectedOut)
}

// goldenDateSelector returns the selector every Golden Liquid run under d
// applies: a case is out of the run exactly when its template or one of
// its partials applies date, which is Wave 5's (plan §8 and §9). The
// selection is mechanical: each source is parsed under d with a stub date
// added, and its tree walked for an application of date. It names no case
// and needs no list.
//
// It refuses a dialect that has date: once date is registered, its cases
// are in scope, and the selector must go rather than keep them out.
func goldenDateSelector(d *Dialect) (func(goldenCase) bool, error) {
	if _, ok := d.Filter("date"); ok {
		return nil, errors.New("the dialect registers date, so Golden Liquid's date cases are in scope: remove goldenDateSelector")
	}
	probe, err := d.Extend(Spec{Filters: []FilterDef{stubFilter("date")}})
	if err != nil {
		return nil, err
	}
	return func(c goldenCase) bool {
		srcs := []string{c.Template}
		for _, name := range slices.Sorted(maps.Keys(c.Templates)) {
			srcs = append(srcs, c.Templates[name])
		}
		for _, src := range srcs {
			tpl, _ := Parse(probe, Source{Text: []byte(src)}, ParseLimits{})
			if tpl == nil {
				continue
			}
			found := false
			ast.Walk(tpl.Root, func(n ast.Node) bool {
				if f, ok := n.(*ast.Filter); ok && f.Name == "date" {
					found = true
				}
				return !found
			})
			if found {
				return true
			}
		}
		return false
	}, nil
}

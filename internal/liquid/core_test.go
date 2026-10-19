// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// testMeter is a Meter with a limit per kind of work, none when 0. It
// keeps Meter's contract: a charge that would pass the limit refuses, the
// total stays where it was, and every later charge of that kind refuses.
type testMeter struct {
	limit   map[Work]int64
	used    map[Work]int64
	tripped map[Work]bool
}

func newTestMeter(limits map[Work]int64) *testMeter {
	return &testMeter{limit: limits, used: map[Work]int64{}, tripped: map[Work]bool{}}
}

func (m *testMeter) Charge(w Work, n int64) *Error {
	if n < 0 {
		panic("a negative charge")
	}
	if lim := m.limit[w]; m.tripped[w] || (lim > 0 && m.used[w]+n > lim) {
		m.tripped[w] = true
		return &Error{Code: CodeLimit, Args: map[string]string{"limit": strconv.Itoa(int(w))}}
	}
	m.used[w] += n
	return nil
}

func (m *testMeter) Used(w Work) int64 { return m.used[w] }

// testContext is a FilterContext of a testMeter.
type testContext struct{ m *testMeter }

func (c testContext) Meter() Meter { return c.m }

// adrList returns the backquoted identifiers of the paragraph of doc that
// starts at the first line beginning with from, up to the text until.
func adrList(t *testing.T, doc, from, until string) []string {
	t.Helper()
	i := strings.Index(doc, from)
	if i < 0 {
		t.Fatalf("the document has no %q", from)
	}
	para := doc[i+len(from):]
	j := strings.Index(para, until)
	if j < 0 {
		t.Fatalf("the document has no %q after %q", until, from)
	}
	var names []string
	for _, m := range regexp.MustCompile("`([^`]+)`").FindAllStringSubmatch(para[:j], -1) {
		if n := m[1]; n == "#" || isIdentifier(n) {
			names = append(names, n)
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

func readDoc(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", path))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(strings.Fields(string(b)), " ")
}

// TestCoreIsADR004Catalog compares Core() with ADR-004's catalog as its
// Context writes it (ADR-010 D4 item 2): the core tags are Shopify's
// STANDARD_TAGS without include, and the liquid tag; the filters are
// standardfilters.rb's, escape's alias h with them, without date (plan
// §8). It reads the lists from the ADR's own text, and holds ADR-010 D4
// item 2's filter list to the same set.
func TestCoreIsADR004Catalog(t *testing.T) {
	testenv.Home(t)
	adr004 := readDoc(t, "docs/adr/ADR-004-the-template-environment.md")
	tags := adrList(t, adr004, "`STANDARD_TAGS`):", "is parsed by `BlockBody` itself")
	tags = slices.DeleteFunc(tags, func(n string) bool { return n == "include" })
	if !slices.Contains(tags, "liquid") {
		t.Fatalf("ADR-004's tag paragraph names no liquid tag: %q", tags)
	}
	if got := CoreTags(); !slices.Equal(got, tags) {
		t.Errorf("CoreTags() = %q, ADR-004's core tags are %q", got, tags)
	}
	if got := Core().TagNames(); len(got) != 0 {
		t.Errorf("Core() adds tags %q; the core tags are the grammar's", got)
	}

	filters := adrList(t, adr004, "**Standard filters** (`lib/liquid/standardfilters.rb`):", "The documentation site's filter index")
	if !slices.Contains(filters, "date") || !slices.Contains(filters, "h") {
		t.Fatalf("ADR-004's filter list %q lacks date or h; the paragraph moved", filters)
	}
	want := slices.DeleteFunc(slices.Clone(filters), func(n string) bool { return n == "date" })
	if got := Core().FilterNames(); !slices.Equal(got, want) {
		t.Errorf("Core().FilterNames() = %q, ADR-004's catalog without date is %q", got, want)
	}

	adr010 := readDoc(t, "docs/adr/ADR-010-the-implementation.md")
	from := "the version ADR-004 pins — at v5.14.0 `"
	i := strings.Index(adr010, from)
	if i < 0 {
		t.Fatalf("ADR-010 has no %q", from)
	}
	list, _, _ := strings.Cut(adr010[i+len(from):], "`")
	d4 := append(strings.Fields(list), "h")
	slices.Sort(d4)
	if !slices.Equal(d4, filters) {
		t.Errorf("ADR-010 D4 item 2's filters with h are %q, ADR-004's are %q", d4, filters)
	}
	for _, n := range []string{"date", "include", "json", "safe", "slug"} {
		if _, ok := Core().Filter(n); ok {
			t.Errorf("Core() has %s", n)
		}
	}
}

// TestCoreCheck holds each filter's Check to ADR-004 D16 item 1 and to
// FilterDef's contract: an application wrong whatever the data is refused
// at parse, located at the "{{" of its token (ADR-004 D16 item 2), and
// the parse goes on to report the next; an application whose fault
// depends on a value parses.
func TestCoreCheck(t *testing.T) {
	testenv.Home(t)
	refused := []struct {
		src  string
		code Code
		arg  string
	}{
		{`{{ x | upcase: 1 }}`, CodeFilterArgument, "1"},
		{`{{ x | append }}`, CodeFilterArgument, "1"},
		{`{{ x | append: 1, 2 }}`, CodeFilterArgument, "2"},
		{`{{ x | replace_last: "a" }}`, CodeFilterArgument, "2"},
		{`{{ x | slice }}`, CodeFilterArgument, "1"},
		{`{{ x | join: ",", "," }}`, CodeFilterArgument, "2"},
		{`{{ x | where: "a", 1, 2 }}`, CodeFilterArgument, "3"},
		{`{{ x | truncate: 1, "a", "b" }}`, CodeFilterArgument, "3"},
		{`{{ x | sort: by: "a" }}`, CodeFilterArgument, "by"},
		{`{{ x | default: 1, allow_nil: true }}`, CodeFilterArgument, "allow_nil"},
		{`{{ x | default: 1, allow_false: true, allow_false: false }}`, CodeFilterArgument, "allow_false"},
		{`{{ x | slice: 1.5 }}`, CodeFilterArgument, "1"},
		{`{{ x | slice: "a" }}`, CodeFilterArgument, "1"},
		{`{{ x | slice: "01" }}`, CodeFilterArgument, "1"},
		{`{{ x | slice: nil }}`, CodeFilterArgument, "1"},
		{`{{ x | slice: 1, 2.0 }}`, CodeFilterArgument, "2"},
		{`{{ x | slice: 1, true }}`, CodeFilterArgument, "2"},
		{`{{ x | truncate: 2.5 }}`, CodeFilterArgument, "1"},
		{`{{ x | truncatewords: "x" }}`, CodeFilterArgument, "1"},
		{`{{ x | truncatewords: empty }}`, CodeFilterArgument, "1"},
		{`{{ x | concat: "a" }}`, CodeFilterArgument, "1"},
		{`{{ x | concat: 1 }}`, CodeFilterArgument, "1"},
		{`{{ x | concat: nil }}`, CodeFilterArgument, "1"},
		{`{{ x | divided_by: 0 }}`, CodeDivisionByZero, ""},
		{`{{ x | divided_by: 0.0 }}`, CodeDivisionByZero, ""},
		{`{{ x | modulo: "foo" }}`, CodeDivisionByZero, ""},
		{`{{ x | modulo: nil }}`, CodeDivisionByZero, ""},
	}
	for _, c := range refused {
		_, errs := Parse(Core(), Source{Name: "t", Text: []byte("a\n" + c.src + " {{ y | nope }}")}, ParseLimits{})
		if len(errs) != 2 {
			t.Errorf("%s: %d errors %v, want the refusal and the unknown filter after it", c.src, len(errs), errs)
			continue
		}
		e := errs[0]
		name := strings.TrimSpace(strings.SplitN(strings.SplitN(c.src, "|", 2)[1], ":", 2)[0])
		name = strings.TrimSuffix(name, " }}")
		if e.Code != c.code || e.Args["filter"] != name || e.Args["argument"] != c.arg || e.Line != 2 || e.Col != 1 {
			t.Errorf("%s: got %v, want %s at 2:1 filter=%s argument=%q", c.src, e, c.code, name, c.arg)
		}
		if errs[1].Code != CodeUnknownFilter {
			t.Errorf("%s: the parse did not go on: %v", c.src, errs[1])
		}
	}
	for _, src := range []string{
		`{{ x | upcase }}`, `{{ x | join }}`, `{{ x | join: y }}`, `{{ x | default }}`,
		`{{ x | default: allow_false: true, "b" }}`, `{{ x | slice: "2", " 3 " }}`, `{{ x | slice: 1, nil }}`,
		`{{ x | slice: -1, false }}`, `{{ x | slice: y, z }}`, `{{ x | truncate }}`, `{{ x | truncate: 5, nil }}`,
		`{{ x | concat: (1..2) }}`, `{{ x | concat: y }}`, `{{ x | divided_by: y }}`, `{{ x | divided_by: "2" }}`,
		`{{ x | modulo: -0.5 }}`, `{{ x | round: "a" }}`, `{{ x | where: "a" }}`, `{{ x | replace: "a" }}`,
		`{{ x | map: nil }}`, `{{ x | split: false }}`, `{{ x | h }}`,
	} {
		if _, errs := Parse(Core(), Source{Text: []byte(src)}, ParseLimits{}); len(errs) != 0 {
			t.Errorf("%s: refused: %v", src, errs[0])
		}
	}
}

// TestFiltersReadNothingElse pins ADR-004 D7 item 1 and ADR-010 D4 item
// 5 for the catalog: no file of it imports a package that reaches the
// host, the clock, randomness or another goroutine's state, so a filter
// reads nothing but its input, its arguments and its context. Golden
// Liquid and the unit tests hold what they compute to their arguments.
func TestFiltersReadNothingElse(t *testing.T) {
	testenv.Home(t)
	allowed := []string{
		"encoding/base64", "fmt", "maps", "math", "math/big", "math/bits", "regexp", "slices",
		"strconv", "strings", "unicode", "unicode/utf8",
		"github.com/kottera/metais/internal/liquid/ast", "github.com/kottera/metais/internal/liquid/value",
	}
	files, err := filepath.Glob("filter*.go")
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, "core.go")
	var seen int
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		seen++
		af, err := goparser.ParseFile(token.NewFileSet(), f, nil, goparser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range af.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if !slices.Contains(allowed, p) {
				t.Errorf("%s imports %s; a filter reads only its input, its arguments and its context", f, p)
			}
		}
	}
	if seen < 4 {
		t.Fatalf("found %d catalog files; the glob is stale", seen)
	}
}

// TestEngineReadsNothingElse holds the whole engine to what
// TestFiltersReadNothingElse holds the catalog to: every non-test file of
// internal/liquid/..., on every platform, the parser, the renderer, the
// evaluator and the meter among them, imports only packages that compute.
// None reaches the clock, randomness, the host or another goroutine's
// state, so a render's output and the work it is charged follow from its
// template and its inputs alone, and the meter counts work, never time
// (W3-3.3). The list is of what is allowed, so a new file or a new import
// is held to it unread.
func TestEngineReadsNothingElse(t *testing.T) {
	testenv.Home(t)
	allowed := []string{
		"cmp", "encoding/base64", "errors", "fmt", "io", "iter", "maps", "math", "math/big", "math/bits",
		"regexp", "slices", "sort", "strconv", "strings", "unicode", "unicode/utf8",
		// A recovered panic's stack, for CodeInternal's detail alone.
		"runtime/debug",
		"github.com/kottera/metais/internal/liquid/ast", "github.com/kottera/metais/internal/liquid/value",
	}
	var seen []string
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		seen = append(seen, filepath.ToSlash(path))
		af, err := goparser.ParseFile(token.NewFileSet(), path, nil, goparser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range af.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if !slices.Contains(allowed, p) {
				t.Errorf("%s imports %s; the engine reads only its template, its inputs and its context", path, p)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"meter.go", "render.go", "eval.go", "parse.go", "ast/ast.go", "value/value.go"} {
		if !slices.Contains(seen, want) {
			t.Errorf("%s was not checked; the walk is stale (saw %q)", want, seen)
		}
	}
}

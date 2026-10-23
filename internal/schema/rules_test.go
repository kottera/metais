// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/schemafail"
	"metais.org/metais/internal/testenv"
)

// TestUniqueKeysGrammar is MTS-DAT-015's negative test: a uniqueKeys
// that is not an array of 1 to 8 key pointers, no two equal, is refused at
// open with one schema.invalid naming the keyword, at its value, wherever
// the schema object holding it is (A2 item 14).
func TestUniqueKeysGrammar(t *testing.T) {
	testenv.Home(t)
	nine := `["/a","/b","/c","/d","/e","/f","/g","/h","/i"]`
	for _, c := range []struct{ name, schema, ptr string }{
		{"not an array", `{"uniqueKeys": "/a"}`, "/uniqueKeys"},
		{"no pointer", `{"uniqueKeys": []}`, "/uniqueKeys"},
		{"nine pointers", `{"uniqueKeys": ` + nine + `}`, "/uniqueKeys"},
		{"a member that is not a string", `{"uniqueKeys": ["/a", 1]}`, "/uniqueKeys"},
		{"the empty pointer", `{"uniqueKeys": [""]}`, "/uniqueKeys"},
		{"a pointer with no leading slash", `{"uniqueKeys": ["a"]}`, "/uniqueKeys"},
		{"an escape RFC 6901 does not define", `{"uniqueKeys": ["/a~2"]}`, "/uniqueKeys"},
		{"two equal pointers", `{"uniqueKeys": ["/a", "/b", "/a"]}`, "/uniqueKeys"},
		{"in a subschema", `{"properties": {"rows": {"uniqueKeys": {}}}}`, "/properties/rows/uniqueKeys"},
		{"in $defs", `{"$defs": {"t": {"uniqueKeys": [true]}}}`, "/$defs/t/uniqueKeys"},
	} {
		comp, fs := compileFiles(t, limits.Defaults(), map[string]string{"inputs/s.json": c.schema})
		if comp != nil || len(fs) != 1 {
			t.Errorf("%s: %d findings, want 1:\n%s", c.name, len(fs), describe(fs))
			continue
		}
		l, _ := fs[0].Location()
		if fs[0].Code() != CodeInvalid || l.Pointer != c.ptr {
			t.Errorf("%s: got %s at #%s, want %s at #%s", c.name, fs[0].Code(), l.Pointer, CodeInvalid, c.ptr)
		}
		if got := param(t, fs[0], "keyword"); got != "uniqueKeys" {
			t.Errorf("%s: keyword is %q, want uniqueKeys", c.name, got)
		}
	}
	eight := `["/a","/b","/c","/d","/e","/f","/g","/h"]`
	for _, s := range []string{
		`{"uniqueKeys": ["/a"]}`,
		`{"uniqueKeys": ` + eight + `}`,
		`{"uniqueKeys": ["/a~1b", "/a/b", "/~0", "/0", "/"]}`,
	} {
		if _, fs := compileFiles(t, limits.Defaults(), map[string]string{"inputs/s.json": s}); len(fs) > 0 {
			t.Errorf("%s is refused:\n%s", s, describe(fs))
		}
	}
}

// TestCoverageGrammar is MTS-DAT-016's negative test: a coverage that is
// not an object of exactly key, a key pointer, and values, one or more
// values no two of which are equal as MTS-DAT-013 compares them, is
// refused at open with one schema.invalid naming the keyword, at its
// value, wherever the schema object holding it is (A2 item 14). Inside
// the window W1-6 opened it was refused as an unknown keyword; it is now
// a keyword of this dialect.
func TestCoverageGrammar(t *testing.T) {
	testenv.Home(t)
	for _, c := range []struct{ name, schema, ptr string }{
		{"not an object", `{"coverage": ["/a", 1]}`, "/coverage"},
		{"no member", `{"coverage": {}}`, "/coverage"},
		{"no values", `{"coverage": {"key": "/a"}}`, "/coverage"},
		{"no key", `{"coverage": {"values": [1]}}`, "/coverage"},
		{"a third member", `{"coverage": {"key": "/a", "values": [1], "x-note": 1}}`, "/coverage"},
		{"a misnamed member", `{"coverage": {"keys": "/a", "values": [1]}}`, "/coverage"},
		{"a key that is not a string", `{"coverage": {"key": ["/a"], "values": [1]}}`, "/coverage"},
		{"the empty pointer", `{"coverage": {"key": "", "values": [1]}}`, "/coverage"},
		{"a pointer with no leading slash", `{"coverage": {"key": "a", "values": [1]}}`, "/coverage"},
		{"values not an array", `{"coverage": {"key": "/a", "values": 1}}`, "/coverage"},
		{"no value", `{"coverage": {"key": "/a", "values": []}}`, "/coverage"},
		{"two equal values", `{"coverage": {"key": "/a", "values": ["x", "y", "x"]}}`, "/coverage"},
		{"a number by its value", `{"coverage": {"key": "/a", "values": [1, 1.0]}}`, "/coverage"},
		{"an object whatever its order", `{"coverage": {"key": "/a", "values": [{"x": 1, "y": 2}, {"y": 2, "x": 1}]}}`, "/coverage"},
		{"in a subschema", `{"properties": {"rows": {"coverage": {"key": "/a"}}}}`, "/properties/rows/coverage"},
		{"in $defs", `{"$defs": {"t": {"coverage": true}}}`, "/$defs/t/coverage"},
	} {
		comp, fs := compileFiles(t, limits.Defaults(), map[string]string{"inputs/s.json": c.schema})
		if comp != nil || len(fs) != 1 {
			t.Errorf("%s: %d findings, want 1:\n%s", c.name, len(fs), describe(fs))
			continue
		}
		l, _ := fs[0].Location()
		if fs[0].Code() != CodeInvalid || l.Pointer != c.ptr {
			t.Errorf("%s: got %s at #%s, want %s at #%s", c.name, fs[0].Code(), l.Pointer, CodeInvalid, c.ptr)
		}
		if got := param(t, fs[0], "keyword"); got != "coverage" {
			t.Errorf("%s: keyword is %q, want coverage", c.name, got)
		}
	}
	for _, s := range []string{
		`{"type": "array", "coverage": {"key": "/a", "values": [1]}}`,
		`{"coverage": {"values": [null, false, 0, "0", "", [], {}], "key": "/a~1b/0"}}`,
		`{"coverage": {"key": "/", "values": [1, 1.5, "1"]}}`,
	} {
		if _, fs := compileFiles(t, limits.Defaults(), map[string]string{"inputs/s.json": s}); len(fs) > 0 {
			t.Errorf("%s is refused:\n%s", s, describe(fs))
		}
	}
}

// coverageFailures validates v against schema and returns every coverage
// failure in the tree as its array's instance pointer, the keyword's
// schema pointer and the member missing, sorted, failing t on any other.
func coverageFailures(t *testing.T, schema, v string) []string {
	t.Helper()
	s := mustCompile(t, map[string]string{"inputs/s.json": schema}, "inputs/s.json")
	fails, err := s.Validate(mustValue(t, v))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for i := range fails {
		fails[i].Walk(func(_ []*schemafail.Failure, f *schemafail.Failure) bool {
			if len(f.Children) > 0 {
				return true
			}
			if f.Keyword != "coverage" {
				t.Errorf("a failure of %s at %s", f.Keyword, f.SchemaPointer)
				return true
			}
			out = append(out, fmt.Sprintf("/%s %s %v", strings.Join(f.InstancePointer, "/"), f.SchemaPointer, f.Data["missing"]))
			return true
		})
	}
	slices.Sort(out)
	return out
}

// TestCoverage is MTS-DAT-017 and MTS-DAT-019 against the adapter's tree:
// each member of values that no item's value at key equals is one failure
// at the array naming that member, values compare as MTS-DAT-013 compares
// them, an item with no value at key covers nothing, and the keyword
// applies wherever its schema object applies to an array, and never to
// anything else.
func TestCoverage(t *testing.T) {
	testenv.Home(t)
	abc := `{"coverage": {"key": "/id", "values": ["a", "b", "c"]}}`
	for _, c := range []struct {
		name, schema, v string
		want            []string
	}{
		{"every value covered", abc, `[{"id": "c"}, {"id": "a"}, {"id": "b"}, {"id": "d"}]`, nil},
		{"every value covered more than once", abc, `[{"id": "a"}, {"id": "a"}, {"id": "b"}, {"id": "c"}]`, nil},
		{"one gap", abc, `[{"id": "a"}, {"id": "c"}]`, []string{`/ /coverage "b"`}},
		{"several gaps", abc, `[{"id": "b"}]`, []string{`/ /coverage "a"`, `/ /coverage "c"`}},
		{"the empty array misses every value", abc, `[]`, []string{`/ /coverage "a"`, `/ /coverage "b"`, `/ /coverage "c"`}},
		{"no value at key covers nothing", abc, `[{"n": "a"}, {"id": "b"}, {}, "a", ["a"]]`, []string{`/ /coverage "a"`, `/ /coverage "c"`}},
		{"a number by its value", `{"coverage": {"key": "/id", "values": [1, 2, 0.5]}}`, `[{"id": 1.0}, {"id": 5e-1}, {"id": "2"}]`, []string{`/ /coverage 2`}},
		{"null is a value", `{"coverage": {"key": "/id", "values": [null, false]}}`, `[{"id": null}, {"id": 0}]`, []string{`/ /coverage false`}},
		{"an object whatever its order", `{"coverage": {"key": "/id", "values": [{"x": 1, "y": [2]}]}}`, `[{"id": {"y": [2.0], "x": 1}}]`, nil},
		{"a pointer into an item's depth", `{"coverage": {"key": "/k/1", "values": [2, 3]}}`, `[{"k": [1, 2]}, {"k": [3]}]`, []string{`/ /coverage 3`}},
		{"not an array", abc, `{"id": "x"}`, nil},
		{"an array beneath properties", `{"properties": {"rows": {"coverage": {"key": "/id", "values": ["a"]}}}}`, `{"rows": [{"id": "b"}]}`,
			[]string{`/rows /properties/rows/coverage "a"`}},
		{"through $ref", `{"$ref": "#/$defs/t", "$defs": {"t": {"coverage": {"key": "/id", "values": [1]}}}}`, `[{"id": 2}]`,
			[]string{`/ /$defs/t/coverage 1`}},
		{"through allOf", `{"allOf": [{"coverage": {"key": "/id", "values": [1]}}]}`, `[]`,
			[]string{`/ /allOf/0/coverage 1`}},
		{"under a conditional", `{"if": {"minItems": 1}, "then": {"coverage": {"key": "/id", "values": [1]}}}`, `[{"id": 2}]`,
			[]string{`/ /then/coverage 1`}},
		{"under a conditional not taken", `{"if": {"minItems": 2}, "then": {"coverage": {"key": "/id", "values": [1]}}}`, `[{"id": 2}]`, nil},
		{"on items' items", `{"items": {"coverage": {"key": "/id", "values": [1]}}}`, `[[{"id": 1}], [{"id": 2}]]`,
			[]string{`/1 /items/coverage 1`}},
		{"beside uniqueKeys, which passes", `{"uniqueKeys": ["/id"], "coverage": {"key": "/id", "values": [1, 2]}}`, `[{"id": 1}]`,
			[]string{`/ /coverage 2`}},
	} {
		got := coverageFailures(t, c.schema, c.v)
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: got\n\t%s\nwant\n\t%s", c.name, strings.Join(got, "\n\t"), strings.Join(c.want, "\n\t"))
		}
	}
}

// TestCoverageLinear is A2 item 18's property for coverage: each item's
// value at key is encoded once and each member of values looked up once,
// so the evaluation is linear in the items and the package's own values.
// Its size is the smallest that fails a check comparing values with
// items: the items are distinct and each value's own item comes after the
// 50,000 that cover nothing, so even a scan that stops at its match makes
// 2,500,000,000 comparisons, more than 3 seconds of CPU, where the linear
// evaluation takes under a second under the race detector. It is bounded
// work, held to a CPU time. One value is missing, the last, so the whole
// of values is seen.
func TestCoverageLinear(t *testing.T) {
	testenv.Home(t)
	const k = 50_000
	const cpuBound = 3 * time.Second
	var items []value.Value
	add := func(id int) {
		num, err := value.ParseNumber(strconv.Itoa(id))
		if err != nil {
			t.Fatal(err)
		}
		o, err := value.NewObject(value.Member{Name: "id", Value: num})
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, o)
	}
	for i := range k {
		add(k + i)
	}
	for i := range k - 1 {
		add(i)
	}
	n := len(items)
	arr, err := value.NewArray(items...)
	if err != nil {
		t.Fatal(err)
	}
	values := make([]string, k)
	for i := range k {
		values[i] = strconv.Itoa(i)
	}
	schema := `{"coverage": {"key": "/id", "values": [` + strings.Join(values, ",") + `]}}`
	s := mustCompile(t, map[string]string{"inputs/s.json": schema}, "inputs/s.json")
	before, measured := selfCPU()
	fails, err := s.Validate(arr)
	if after, _ := selfCPU(); measured {
		t.Logf("coverage of %d values over %d items took %v of CPU", k, n, after-before)
		if after-before > cpuBound {
			t.Errorf("coverage of %d values over %d items took %v of CPU, more than %v", k, n, after-before, cpuBound)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(fails) != 1 || fmt.Sprint(fails[0].Data["missing"]) != strconv.Itoa(k-1) {
		t.Fatalf("got %+v, want one failure missing %d", fails, k-1)
	}
}

// uniqueKeysFailure is one uniqueKeys failure as the adapter gives it.
type uniqueKeysFailure struct {
	inst       string // the array's instance pointer
	schema     string // the keyword's schema pointer
	duplicates []int
	at         string // the key pointer's tokens, joined by /; "-" for several
}

func (f uniqueKeysFailure) String() string {
	return fmt.Sprintf("%s %s %v %s", f.inst, f.schema, f.duplicates, f.at)
}

// uniqueKeysFailures validates v against schema and returns every
// uniqueKeys failure in the tree, sorted, failing t on any other.
func uniqueKeysFailures(t *testing.T, schema, v string) []string {
	t.Helper()
	s := mustCompile(t, map[string]string{"inputs/s.json": schema}, "inputs/s.json")
	fails, err := s.Validate(mustValue(t, v))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for i := range fails {
		fails[i].Walk(func(_ []*schemafail.Failure, f *schemafail.Failure) bool {
			if len(f.Children) > 0 {
				return true
			}
			if f.Keyword != "uniqueKeys" {
				t.Errorf("a failure of %s at %s", f.Keyword, f.SchemaPointer)
				return true
			}
			u := uniqueKeysFailure{inst: "/" + strings.Join(f.InstancePointer, "/"), schema: f.SchemaPointer, at: "-"}
			u.duplicates, _ = f.Data["duplicates"].([]int)
			if at, ok := f.Data["at"].([]string); ok {
				u.at = strings.Join(at, "/")
			}
			out = append(out, u.String())
			return true
		})
	}
	slices.Sort(out)
	return out
}

// TestUniqueKeys is MTS-DAT-017 and MTS-DAT-018 against the adapter's
// tree: each item whose key repeats is one failure naming the first item
// with the key, a key is compared value by value as MTS-DAT-013 compares
// values, an item with no key takes no part, and the keyword applies
// wherever its schema object applies to an array, and never to anything
// else.
func TestUniqueKeys(t *testing.T) {
	testenv.Home(t)
	one := `{"uniqueKeys": ["/id"]}`
	two := `{"uniqueKeys": ["/a", "/b"]}`
	for _, c := range []struct {
		name, schema, v string
		want            []string
	}{
		{"distinct keys", one, `[{"id": 1}, {"id": 2}]`, nil},
		{"a repeated key", one, `[{"id": 1, "n": "a"}, {"id": 2}, {"id": 1, "n": "b"}]`,
			[]string{"/ /uniqueKeys [0 2] id"}},
		{"every repeat, each naming the first", one, `[{"id": 1}, {"id": 1}, {"id": 2}, {"id": 1}, {"id": 2}]`,
			[]string{"/ /uniqueKeys [0 1] id", "/ /uniqueKeys [0 3] id", "/ /uniqueKeys [2 4] id"}},
		{"a number by its value", one, `[{"id": 1.0}, {"id": 1}, {"id": 1e0}, {"id": 10e-1}]`,
			[]string{"/ /uniqueKeys [0 1] id", "/ /uniqueKeys [0 2] id", "/ /uniqueKeys [0 3] id"}},
		{"null equals null", one, `[{"id": null}, {"id": null}]`, []string{"/ /uniqueKeys [0 1] id"}},
		{"a value of each type, distinct", one, `[{"id": null}, {"id": false}, {"id": 0}, {"id": "0"}, {"id": ""}, {"id": []}, {"id": {}}]`, nil},
		{"an object whatever its order", one, `[{"id": {"x": 1, "y": [2]}}, {"id": {"y": [2.0], "x": 1}}]`,
			[]string{"/ /uniqueKeys [0 1] id"}},
		{"no key, no part", one, `[{"n": 1}, {"n": 1}, {"id": 1}, {}, {"id": 1}]`, []string{"/ /uniqueKeys [2 4] id"}},
		{"an item that is not an object has no key", one, `[1, 1, "id", "id"]`, nil},
		{"a composite key", two, `[{"a": 1, "b": 1}, {"a": 1, "b": 2}, {"a": 2, "b": 1}, {"a": 1, "b": 1}]`,
			[]string{"/ /uniqueKeys [0 3] -"}},
		{"a composite key with a part missing", two, `[{"a": 1}, {"a": 1}, {"a": 1, "b": null}, {"a": 1, "b": null}]`,
			[]string{"/ /uniqueKeys [2 3] -"}},
		{"a composite key's parts never run together", `{"uniqueKeys": ["/a", "/b"]}`, `[{"a": "ab", "b": "c"}, {"a": "a", "b": "bc"}]`, nil},
		{"a pointer into an item's depth", `{"uniqueKeys": ["/k/0", "/k~1x"]}`, `[{"k": [1, 2], "k/x": 3}, {"k": [1], "k/x": 3}, {"k": [1, 3]}]`,
			[]string{"/ /uniqueKeys [0 1] -"}},
		{"an index with a leading zero names nothing", `{"uniqueKeys": ["/k/01"]}`, `[{"k": [1, 2]}, {"k": [1, 2]}]`, nil},
		{"not an array", one, `{"id": 1, "x": {"id": 1}}`, nil},
		{"an array beneath properties", `{"properties": {"rows": {"uniqueKeys": ["/id"]}}}`, `{"rows": [{"id": "a"}, {"id": "a"}]}`,
			[]string{"/rows /properties/rows/uniqueKeys [0 1] id"}},
		{"through $ref", `{"$ref": "#/$defs/t", "$defs": {"t": {"uniqueKeys": ["/id"]}}}`, `[{"id": 1}, {"id": 1}]`,
			[]string{"/ /$defs/t/uniqueKeys [0 1] id"}},
		{"through allOf", `{"allOf": [{"uniqueKeys": ["/id"]}]}`, `[{"id": 1}, {"id": 1}]`,
			[]string{"/ /allOf/0/uniqueKeys [0 1] id"}},
		{"under a conditional", `{"if": {"minItems": 2}, "then": {"uniqueKeys": ["/id"]}}`, `[{"id": 1}, {"id": 1}]`,
			[]string{"/ /then/uniqueKeys [0 1] id"}},
		{"under a conditional not taken", `{"if": {"minItems": 3}, "then": {"uniqueKeys": ["/id"]}}`, `[{"id": 1}, {"id": 1}]`, nil},
		{"on items' items", `{"items": {"uniqueKeys": ["/id"]}}`, `[[{"id": 1}], [{"id": 2}, {"id": 2}]]`,
			[]string{"/1 /items/uniqueKeys [0 1] id"}},
	} {
		got := uniqueKeysFailures(t, c.schema, c.v)
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: got\n\t%s\nwant\n\t%s", c.name, strings.Join(got, "\n\t"), strings.Join(c.want, "\n\t"))
		}
	}
}

// TestUniqueKeysLinear is A2 item 18's property that keyed evaluation is
// linear in the items: each item's key is encoded and looked up once. Its
// size is the smallest that fails a check comparing every pair of keys:
// about 5,000,000,000 comparisons at 100,000 distinct items, which take
// more than 7 seconds of CPU even as comparisons of short strings, while
// the linear evaluation takes about a tenth of a second, and less than a
// second under the race detector. It is bounded work, held to a CPU time.
// The keys are distinct, the worst case for a pairwise check, which finds
// no duplicate to stop at, but for one repeat at the end, to show the
// whole array is seen.
func TestUniqueKeysLinear(t *testing.T) {
	testenv.Home(t)
	const n = 100_000
	const cpuBound = 3 * time.Second
	items := make([]value.Value, n)
	for i := range n {
		id := i
		if i == n-1 {
			id = 0
		}
		num, err := value.ParseNumber(strconv.Itoa(id))
		if err != nil {
			t.Fatal(err)
		}
		o, err := value.NewObject(value.Member{Name: "id", Value: num})
		if err != nil {
			t.Fatal(err)
		}
		items[i] = o
	}
	arr, err := value.NewArray(items...)
	if err != nil {
		t.Fatal(err)
	}
	s := mustCompile(t, map[string]string{"inputs/s.json": `{"uniqueKeys": ["/id"]}`}, "inputs/s.json")
	before, measured := selfCPU()
	fails, err := s.Validate(arr)
	if after, _ := selfCPU(); measured {
		t.Logf("uniqueKeys over %d items took %v of CPU", n, after-before)
		if after-before > cpuBound {
			t.Errorf("uniqueKeys over %d items took %v of CPU, more than %v", n, after-before, cpuBound)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(fails) != 1 || fmt.Sprint(fails[0].Data["duplicates"]) != fmt.Sprint([]int{0, n - 1}) {
		t.Fatalf("got %+v, want one failure naming items 0 and %d", fails, n-1)
	}
}

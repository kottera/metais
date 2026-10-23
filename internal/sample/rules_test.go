// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sample

import (
	"slices"
	"testing"

	"metais.org/metais/internal/liquid/value"
)

// rows is the schema of an array of rows whose criterion column is an
// enum of four, with the rules given, which are spliced in before items.
func rows(rules string) string {
	return `{"type":"array",` + rules + `"items":{"type":"object","required":["criterion","level"],"properties":{
		"criterion":{"enum":["1.1.1","1.2.1","1.3.1","1.4.1"]},
		"level":{"type":"string","pattern":"^[A-Z]{1,3}$"}}}}`
}

// column returns the value each item of v holds at name, in order.
func column(t *testing.T, v value.Value, name string) []string {
	t.Helper()
	a, ok := v.(value.Array)
	if !ok {
		t.Fatalf("not an array: %s", text(v))
	}
	var out []string
	for _, e := range a.All() {
		m, _ := at(e, []string{name})
		s, ok := m.(value.String)
		if !ok {
			t.Fatalf("item %s holds no string at %s", text(e), name)
		}
		out = append(out, s.Text())
	}
	return out
}

// TestCoverageItems is MTS-SMP-010, by ADR-003 Amendment A2 item 17: one
// item per member of coverage's values comes first, in values' order,
// holding the member at the key, whatever --items asks; then the array
// fills up to --items. Below, at and above the number of values.
func TestCoverageItems(t *testing.T) {
	sch := rows(`"coverage":{"key":"/criterion","values":["1.3.1","1.1.1","1.4.1"]},`)
	for _, tc := range []struct {
		items, length int
	}{{0, 3}, {1, 3}, {3, 3}, {5, 5}} {
		v := mustOne(t, sch, Options{Items: tc.items})
		got := column(t, v, "criterion")
		if len(got) != tc.length || !slices.Equal(got[:3], []string{"1.3.1", "1.1.1", "1.4.1"}) {
			t.Errorf("--items %d: criteria %v, want 1.3.1, 1.1.1, 1.4.1 first and %d items", tc.items, got, tc.length)
		}
	}
}

// TestCoverageNested: a key below the item's top level is held where it
// points, and the item's other fields are generated around it.
func TestCoverageNested(t *testing.T) {
	v := mustOne(t, `{"type":"array","coverage":{"key":"/ref/id","values":[7,9]},
		"items":{"type":"object","required":["ref","note"],"properties":{
			"ref":{"type":"object","required":["id","by"],"properties":{"id":{"type":"integer"},"by":{"type":"string"}}},
			"note":{"type":"string"}}}}`, Options{Items: 2})
	a := v.(value.Array)
	for i, want := range []float64{7, 9} {
		id, ok := at(a.At(i), []string{"ref", "id"})
		if n, isNum := id.(value.Number); !ok || !isNum || n.Float64() != want {
			t.Errorf("item %d: %s, want /ref/id %v", i, text(a.At(i)), want)
		}
	}
}

// TestUniqueKeysItems is MTS-SMP-011: no two items have equal keys, for a
// key of one pointer whose values the generator draws, or takes from an
// enum, and for a composite key, compared pointer by pointer.
func TestUniqueKeysItems(t *testing.T) {
	for _, tc := range []struct {
		schema string
		key    [][]string
		items  int
	}{
		{`{"type":"array","uniqueKeys":["/id"],"items":{"type":"object","required":["id"],"properties":{"id":{"type":"integer","minimum":1,"maximum":6}}}}`,
			[][]string{{"id"}}, 6},
		{`{"type":"array","uniqueKeys":["/id"],"items":{"type":"object","required":["id"],"properties":{"id":{"type":"string","pattern":"^[a-c]$"}}}}`,
			[][]string{{"id"}}, 3},
		{rows(`"uniqueKeys":["/criterion"],`), [][]string{{"criterion"}}, 4},
		{`{"type":"array","uniqueKeys":["/edition","/criterion"],"items":{"type":"object","required":["edition","criterion"],"properties":{
			"edition":{"enum":["wcag","en"]},"criterion":{"enum":["1.1.1","1.2.1"]}}}}`,
			[][]string{{"edition"}, {"criterion"}}, 4},
	} {
		v := mustOne(t, tc.schema, Options{Items: tc.items})
		a := v.(value.Array)
		if a.Len() != tc.items {
			t.Errorf("%s: %d items, want %d: %s", tc.schema, a.Len(), tc.items, text(v))
		}
		var earlier []value.Value
		for _, e := range a.All() {
			if keyRepeats(earlier, tc.key, e) {
				t.Errorf("%s: the key of %s repeats: %s", tc.schema, text(e), text(v))
			}
			earlier = append(earlier, e)
		}
	}
}

// TestKeyRepeats is MTS-DAT-018's comparison, as the generator applies
// it: 1.0 equals 1, null equals null, a composite key repeats only when
// every pointer's value does, and an item without the key repeats none.
func TestKeyRepeats(t *testing.T) {
	items := []value.Value{parse(t, `{"a":1,"b":null}`), parse(t, `{"a":2}`)}
	for _, tc := range []struct {
		key  [][]string
		item string
		want bool
	}{
		{[][]string{{"a"}}, `{"a":1.0}`, true},
		{[][]string{{"a"}}, `{"a":3}`, false},
		{[][]string{{"b"}}, `{"b":null}`, true},
		{[][]string{{"a"}, {"b"}}, `{"a":1,"b":null}`, true},
		{[][]string{{"a"}, {"b"}}, `{"a":2,"b":null}`, false},
		{[][]string{{"a"}, {"b"}}, `{"a":2}`, false},
		{[][]string{{"c"}}, `{"c":1}`, false},
	} {
		if got := keyRepeats(items, tc.key, parse(t, tc.item)); got != tc.want {
			t.Errorf("%v %s: %v, want %v", tc.key, tc.item, got, tc.want)
		}
	}
}

// TestBothKeywords: one column both the unique key and the covered key.
// The covered items come first; an item filling up to --items takes a
// key no earlier item holds; and when --items asks for more items than
// there are distinct keys, the array is shortened to the keys there are,
// as ADR-007 D9 item 16 shortens a uniqueItems array, since minItems
// allows it.
func TestBothKeywords(t *testing.T) {
	sch := rows(`"uniqueKeys":["/criterion"],"coverage":{"key":"/criterion","values":["1.2.1","1.1.1"]},`)
	for _, tc := range []struct {
		items int
		want  []string
	}{
		{1, []string{"1.2.1", "1.1.1"}},
		{3, []string{"1.2.1", "1.1.1", "1.3.1"}},
		{4, []string{"1.2.1", "1.1.1", "1.3.1", "1.4.1"}},
		{9, []string{"1.2.1", "1.1.1", "1.3.1", "1.4.1"}},
	} {
		v := mustOne(t, sch, Options{Items: tc.items})
		got := column(t, v, "criterion")
		if len(got) < 2 || !slices.Equal(got[:2], tc.want[:2]) || len(got) != len(tc.want) {
			t.Errorf("--items %d: criteria %v, want %v", tc.items, got, tc.want)
			continue
		}
		rest := slices.Sorted(slices.Values(got[2:]))
		if !slices.Equal(rest, tc.want[2:]) {
			t.Errorf("--items %d: criteria %v, want %v", tc.items, got, tc.want)
		}
	}
}

// TestRulesDeterministic is the generator's determinism with both
// keywords (ADR-007 D9 item 16): the same seed gives the same items, and
// these, on every platform; another seed may give others. The values are
// pinned, so a change to how either rule is honoured is seen here.
func TestRulesDeterministic(t *testing.T) {
	sch := rows(`"uniqueKeys":["/criterion"],"coverage":{"key":"/criterion","values":["1.2.1","1.1.1"]},`)
	for _, tc := range []struct {
		seed uint64
		want string
	}{
		{0, `[{"criterion":"1.2.1","level":"F"},{"criterion":"1.1.1","level":"GBO"},{"criterion":"1.3.1","level":"W"}]`},
		{7, `[{"criterion":"1.2.1","level":"I"},{"criterion":"1.1.1","level":"BM"},{"criterion":"1.3.1","level":"I"}]`},
	} {
		for range 2 {
			if got := text(mustOne(t, sch, Options{Seed: tc.seed, Items: 3})); got != tc.want {
				t.Errorf("seed %d: %s, want %s", tc.seed, got, tc.want)
			}
		}
	}
}

// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/regexprofile"
	"github.com/kottera/metais/internal/schemafail"
)

// loud is an error kind whose text cannot be rendered: it embeds a nil
// *kind.MaxLength, whose LocalizedString it promotes, so rendering it
// panics. Its Err field is an error whose text cannot be read either. (The
// test reaches the method by embedding because only internal/unitext may
// import golang.org/x/text, whose Printer the method takes.)
type loud struct {
	*kind.MaxLength
	Limit  int
	Detail string
	Err    error
}

func (*loud) KeywordPath() []string { return []string{"maxLength"} }

type loudErr struct{}

func (loudErr) Error() string { panic("the adapter read an error's text") }

const rootURL = "metais:///inputs/s.json"

// TestAdapterReadsNoText holds ADR-010 D5 item 5's first property: the
// adapter builds its tree without rendering any of the library's text.
// The tree's format failure holds an error that panics when its text is
// read, which rendering the failure does, so the property is red against
// an adapter that reads a message: the control renders it. A kind the
// adapter does not know is refused with an error, unread.
func TestAdapterReadsNoText(t *testing.T) {
	c := compiled(t)
	ve := &jsonschema.ValidationError{
		SchemaURL: rootURL, ErrorKind: &kind.Schema{Location: rootURL},
		Causes: []*jsonschema.ValidationError{
			{SchemaURL: rootURL + "#/properties/a", InstanceLocation: []string{"a"},
				ErrorKind: &kind.MinLength{Got: 1, Want: 3}},
			{SchemaURL: rootURL + "#/properties/b", InstanceLocation: []string{"b"},
				ErrorKind: &kind.Format{Got: "x", Want: "date", Err: loudErr{}}},
		},
	}
	fails, err := adaptQuietly(t, c, ve)
	if err != nil {
		t.Fatal(err)
	}
	if len(fails) != 2 {
		t.Fatalf("got %d failures, want 2", len(fails))
	}
	a := fails[0]
	if a.Keyword != "minLength" || a.Entry != "inputs/s.json" || a.SchemaPointer != "/properties/a/minLength" ||
		a.KeywordLocation != "/properties/a/minLength" || a.Instance() != "/a" {
		t.Errorf("got %+v", a)
	}
	if !reflect.DeepEqual(fails[1].Data, schemafail.Data{"format": "date"}) {
		t.Errorf("format data %v", fails[1].Data)
	}
	// The control: rendering the tree as the library does reads the
	// error's text, which panics.
	func() {
		defer func() {
			if recover() == nil {
				t.Error("control: rendering the tree's text did not panic")
			}
		}()
		_ = ve.Error()
	}()
	unknown := &jsonschema.ValidationError{SchemaURL: rootURL, ErrorKind: &kind.Schema{Location: rootURL},
		Causes: []*jsonschema.ValidationError{{SchemaURL: rootURL, ErrorKind: &loud{Err: loudErr{}}}}}
	if _, err := adaptQuietly(t, c, unknown); err == nil {
		t.Error("a kind the adapter does not know was adapted")
	}
}

// adaptQuietly adapts ve, reporting a panic, which a loud kind or error
// raises when its text is read, as a failure of the test.
func adaptQuietly(t *testing.T, c *Compiler, ve *jsonschema.ValidationError) (fails []schemafail.Failure, err error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("the adapter read the library's text: %v", r)
		}
	}()
	return c.adapt(ve)
}

// compiled is a compiler over one schema file, for adapting hand-built
// trees.
func compiled(t *testing.T) *Compiler {
	t.Helper()
	c, fs := compileFiles(t, limits.Defaults(), map[string]string{"inputs/s.json": `{}`})
	if len(fs) > 0 {
		t.Fatal(describe(fs))
	}
	return c
}

// TestAdapterOrderIndependent holds ADR-010 D5 item 5's second property:
// the adapter's tree owes nothing to the order the library gives causes
// in. Every permutation of a tree's causes, at every level, adapts to the
// same tree, so the test is red against an adapter that keeps the
// library's order.
func TestAdapterOrderIndependent(t *testing.T) {
	c := compiled(t)
	leaf := func(ptr, inst string, k jsonschema.ErrorKind) *jsonschema.ValidationError {
		return &jsonschema.ValidationError{SchemaURL: rootURL + "#" + ptr, InstanceLocation: []string{inst}, ErrorKind: k}
	}
	causes := func() []*jsonschema.ValidationError {
		return []*jsonschema.ValidationError{
			leaf("/properties/a", "a", &kind.MinLength{Got: 1, Want: 2}),
			leaf("/properties/b", "b", &kind.Type{Got: "number", Want: []string{"string"}}),
			{SchemaURL: rootURL + "#/properties/c", InstanceLocation: []string{"c"}, ErrorKind: &kind.AllOf{},
				Causes: []*jsonschema.ValidationError{
					leaf("/properties/c/allOf/1", "c", &kind.Required{Missing: []string{"x", "y"}}),
					leaf("/properties/c/allOf/0", "c", &kind.MaxProperties{Got: 3, Want: 2}),
				}},
			leaf("/properties/a", "a", &kind.Pattern{Got: "q", Want: "^p"}),
		}
	}
	var first []schemafail.Failure
	perms := permutations(4)
	for i, p := range perms {
		cs := causes()
		ordered := make([]*jsonschema.ValidationError, len(p))
		for j, k := range p {
			ordered[j] = cs[k]
		}
		if i%2 == 1 {
			slices.Reverse(ordered[2].Causes)
		}
		fails, err := c.adapt(&jsonschema.ValidationError{SchemaURL: rootURL, ErrorKind: &kind.Schema{Location: rootURL}, Causes: ordered})
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = fails
			continue
		}
		if !reflect.DeepEqual(fails, first) {
			t.Fatalf("permutation %v adapted differently:\n%+v\nwant\n%+v", p, fails, first)
		}
	}
	// The control: the library's own order differs between these trees,
	// so an adapter that kept it would differ too.
	if reflect.DeepEqual(perms[0], perms[1]) {
		t.Fatal("control: the permutations are not distinct")
	}
	var got []string
	for _, f := range first {
		f.Walk(func(_ []*schemafail.Failure, n *schemafail.Failure) bool {
			got = append(got, n.KeywordLocation)
			return true
		})
	}
	want := []string{"/properties/a/minLength", "/properties/a/pattern", "/properties/b/type",
		"/properties/c/allOf", "/properties/c/allOf/0/maxProperties", "/properties/c/allOf/1/required"}
	if !slices.Equal(got, want) {
		t.Errorf("adapted in the order %q, want %q", got, want)
	}
}

func permutations(n int) [][]int {
	if n == 0 {
		return [][]int{{}}
	}
	var out [][]int
	for _, p := range permutations(n - 1) {
		for i := 0; i <= len(p); i++ {
			q := append(append(append([]int{}, p[:i]...), n-1), p[i:]...)
			out = append(out, q)
		}
	}
	return out
}

// TestAdapterRepeatable validates one instance with many sibling failures
// many times: the library's order follows map iteration and varies, and
// the adapted tree does not.
func TestAdapterRepeatable(t *testing.T) {
	s := mustCompile(t, map[string]string{"inputs/s.json": `{"properties": {
		"a": {"type": "string"}, "b": {"minimum": 3}, "c": {"maxLength": 1}, "d": {"enum": [1]},
		"e": {"const": 2}, "f": {"required": ["x"]}, "g": {"pattern": "^z"}, "h": {"format": "date"}},
		"required": ["i", "j", "k"]}`}, "inputs/s.json")
	inst := mustValue(t, `{"a": 1, "b": 1, "c": "ccc", "d": 2, "e": 3, "f": {}, "g": "y", "h": "nope"}`)
	var first []schemafail.Failure
	for i := 0; i < 50; i++ {
		fails, err := s.Validate(inst)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = fails
			if len(fails) != 9 {
				t.Fatalf("got %d failures, want 9", len(fails))
			}
			continue
		}
		if !reflect.DeepEqual(fails, first) {
			t.Fatalf("validation %d adapted differently", i)
		}
	}
}

// TestAdapterData holds the adapter to schemafail's Data contract
// (schemafail/data.go) on the library's own failures, kind by kind, and to
// the shapes the contract asks for: one false failure per member refused,
// a then or else node over a conditional's failures, and a oneOf with two
// branches matching that has no children.
func TestAdapterData(t *testing.T) {
	s := mustCompile(t, map[string]string{"inputs/s.json": `{"properties": {
		"type": {"type": ["string", "null"]},
		"enum": {"enum": [{"b": 1, "a": [true, null]}, "x", 1.50]},
		"const": {"const": {"z": "q", "a": 2}},
		"required": {"required": ["y", "x"]},
		"dependentRequired": {"dependentRequired": {"a": ["c", "b"]}},
		"minProperties": {"minProperties": 2},
		"maxItems": {"maxItems": 1},
		"minContains": {"contains": {"type": "string"}, "minContains": 2},
		"maxLength": {"maxLength": 2},
		"pattern": {"pattern": "^a"},
		"format": {"format": "date"},
		"uniqueItems": {"uniqueItems": true},
		"minimum": {"minimum": 0.1},
		"exclusiveMaximum": {"exclusiveMaximum": 1e2},
		"multipleOf": {"multipleOf": 0.25},
		"anyOf": {"anyOf": [{"type": "string"}, {"type": "boolean"}, false]},
		"oneOf": {"oneOf": [{}, true, {"type": "number"}]},
		"not": {"not": {"type": "number"}},
		"propertyNames": {"propertyNames": {"maxLength": 1}},
		"additionalProperties": {"properties": {"k": {}}, "additionalProperties": false},
		"items": {"prefixItems": [{}, {}], "items": false},
		"conditional": {"if": {"type": "object"}, "then": {"required": ["t"]}, "else": {"maxLength": 0}}
	}}`}, "inputs/s.json")
	fails, err := s.Validate(mustValue(t, `{
		"type": 1, "enum": "y", "const": 0, "required": {}, "dependentRequired": {"a": 1},
		"minProperties": {}, "maxItems": [1, 2], "minContains": ["a", 1], "maxLength": "abc",
		"pattern": "b", "format": "nope", "uniqueItems": [1, 2, 1], "minimum": 0.05,
		"exclusiveMaximum": 100, "multipleOf": 1e-1, "anyOf": 1, "oneOf": 5, "not": 3,
		"propertyNames": {"ab": 1}, "additionalProperties": {"k": 1, "zz": 2, "aa": 3},
		"items": [1, 2, 3, 4], "conditional": {}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range fails {
		f.Walk(func(path []*schemafail.Failure, n *schemafail.Failure) bool {
			got = append(got, fmt.Sprintf("%s%s %s %v", strings.Repeat("  ", len(path)), n.KeywordLocation, n.Instance(), n.Data))
			return true
		})
	}
	want := []string{
		"/properties/additionalProperties/additionalProperties /additionalProperties/aa map[]",
		"/properties/additionalProperties/additionalProperties /additionalProperties/zz map[]",
		"/properties/anyOf/anyOf /anyOf map[branch_count:3]",
		"  /properties/anyOf/anyOf/0/type /anyOf map[got:number want:[string]]",
		"  /properties/anyOf/anyOf/1/type /anyOf map[got:number want:[boolean]]",
		"  /properties/anyOf/anyOf/2 /anyOf map[]",
		"/properties/conditional/then /conditional map[]",
		"  /properties/conditional/then/required /conditional map[missing:[t]]",
		"/properties/const/const /const map[expected:{\"a\":2,\"z\":\"q\"}]",
		"/properties/dependentRequired/dependentRequired /dependentRequired map[because:a missing:[b c]]",
		"/properties/enum/enum /enum map[allowed:[{\"a\":[true,null],\"b\":1} \"x\" 1.5] got:y]",
		"/properties/exclusiveMaximum/exclusiveMaximum /exclusiveMaximum map[limit:100 value:100]",
		"/properties/format/format /format map[format:date]",
		"/properties/items/items /items/2 map[prefix_count:2]",
		"/properties/items/items /items/3 map[prefix_count:2]",
		"/properties/maxItems/maxItems /maxItems map[count:2 limit:1]",
		"/properties/maxLength/maxLength /maxLength map[length:3 limit:2]",
		"/properties/minContains/minContains /minContains map[count:1 limit:2]",
		"/properties/minProperties/minProperties /minProperties map[count:0 limit:2]",
		"/properties/minimum/minimum /minimum map[limit:0.1 value:0.05]",
		"/properties/multipleOf/multipleOf /multipleOf map[divisor:0.25 value:0.1]",
		"/properties/not/not /not map[]",
		"/properties/oneOf/oneOf /oneOf map[branch_count:3 matched:[0 1]]",
		"/properties/pattern/pattern /pattern map[pattern:^a]",
		"/properties/propertyNames/propertyNames /propertyNames map[property:ab]",
		"/properties/required/required /required map[missing:[x y]]",
		"/properties/type/type /type map[got:number want:[null string]]",
		"/properties/uniqueItems/uniqueItems /uniqueItems map[duplicates:[0 2]]",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for i := range fails {
		if err := fails[i].Check(); err != nil {
			t.Error(err)
		}
	}
	// schemafail.Map reads every node as its contract says.
	if _, err := schemafail.Map(target(s.c), fails...); err != nil {
		t.Errorf("schemafail.Map refuses the adapter's tree: %v", err)
	}
}

// TestDecimal writes a rational exactly, or not at all.
func TestDecimal(t *testing.T) {
	for in, want := range map[string]string{
		"0": "0", "1": "1", "-3": "-3", "1/10": "0.1", "1/8": "0.125", "-1/1024": "-0.0009765625",
		"12345678901234567890123/1000": "12345678901234567890.123",
	} {
		r, _ := new(big.Rat).SetString(in)
		if got, ok := decimal(r); !ok || string(got) != want {
			t.Errorf("decimal(%s) = %q, %v; want %q", in, got, ok, want)
		}
	}
	if _, ok := decimal(big.NewRat(1, 3)); ok {
		t.Error("decimal(1/3) has no exact form, and was given one")
	}
}

// toFloat is toLibrary as it must not be written: numbers as float64. It
// is the broken implementation TestNumbersKeepLiteral is red against.
func toFloat(v value.Value) any {
	switch v := v.(type) {
	case value.Number:
		return v.Float64()
	case value.Object:
		out := map[string]any{}
		for k, e := range v.All() {
			out[k] = toFloat(e)
		}
		return out
	}
	return toLibrary(v)
}

// TestNumbersKeepLiteral holds ADR-010 D5 item 4: a number reaches the
// library as json.Number holding its literal text, never float64, so the
// library judges the number written. Each case is one a float64 gets
// wrong, which the control shows.
func TestNumbersKeepLiteral(t *testing.T) {
	cases := []struct{ schema, instance string }{
		{`{"maximum": 9007199254740992}`, `9007199254740993`},
		{`{"const": 9007199254740992}`, `9007199254740993`},
		{`{"maximum": 0.1}`, `0.10000000000000000001`},
		{`{"multipleOf": 0.01}`, `0.0700000000000000001`},
	}
	for _, c := range cases {
		s := mustCompile(t, map[string]string{"inputs/s.json": c.schema}, "inputs/s.json")
		inst := mustValue(t, c.instance)
		if got := toLibrary(inst); got != json.Number(c.instance) {
			t.Errorf("toLibrary(%s) = %#v, want json.Number(%q)", c.instance, got, c.instance)
		}
		fails, err := s.Validate(inst)
		if err != nil || len(fails) == 0 {
			t.Errorf("%s: %s is valid, which only a float64 makes it", c.schema, c.instance)
		}
		if err := s.lib.Validate(toFloat(inst)); err != nil {
			t.Errorf("control: %s: %s as a float64 is invalid, so the case shows nothing", c.schema, c.instance)
		}
	}
	// Nowhere in a converted tree is there a float64.
	var walk func(any)
	walk = func(x any) {
		switch x := x.(type) {
		case float64, float32:
			t.Errorf("a %T in the converted tree", x)
		case []any:
			for _, e := range x {
				walk(e)
			}
		case map[string]any:
			for _, e := range x {
				walk(e)
			}
		}
	}
	walk(toLibrary(mustValue(t, `{"a": [1, 1.5, -0, 1e300, {"b": 2E-3}], "c": null, "d": true, "e": "s"}`)))
}

// TestRegexWork refuses a validation once the command's matching budget is
// spent (ADR-009 D10 item 7), and gives no failures with the refusal.
func TestRegexWork(t *testing.T) {
	c, fs := compileFiles(t, lower(t, "regex-work=100"), map[string]string{"inputs/s.json": `{"pattern": "a"}`})
	if len(fs) > 0 {
		t.Fatal(describe(fs))
	}
	s, _ := c.Schema("inputs/s.json")
	if fails, err := s.Validate(mustValue(t, `"a"`)); err != nil || len(fails) > 0 {
		t.Fatalf("a match within the budget: %v, %v", fails, err)
	}
	fails, err := s.Validate(mustValue(t, `"`+string(make([]byte, 0))+`aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`))
	var le *regexprofile.LimitError
	if !errors.As(err, &le) || le.Limit != limits.RegexWork || fails != nil {
		t.Errorf("a match beyond the budget: %v, %v; want regex-work and no failures", fails, err)
	}
}

// TestKeywordLocations follows the evaluation path through applicators
// on the library's own output.
func TestKeywordLocations(t *testing.T) {
	s := mustCompile(t, map[string]string{"inputs/s.json": `{
		"allOf": [{"properties": {"a": {"minimum": 2}}}, {"required": ["b"]}],
		"properties": {"c": {"anyOf": [{"type": "string"}, {"maxLength": 1, "type": "string"}]},
			"d": {"not": {"type": "null"}}, "e": false, "f": {"oneOf": [{}, true]}}}`}, "inputs/s.json")
	fails, err := s.Validate(mustValue(t, `{"a": 1, "c": 5, "d": null, "e": 0, "f": 1}`))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range fails {
		f.Walk(func(_ []*schemafail.Failure, n *schemafail.Failure) bool {
			got = append(got, n.Keyword+" "+n.KeywordLocation+" "+n.SchemaPointer+" @"+n.Instance())
			return true
		})
	}
	want := []string{
		"allOf /allOf /allOf @",
		"minimum /allOf/0/properties/a/minimum /allOf/0/properties/a/minimum @/a",
		"required /allOf/1/required /allOf/1/required @",
		"anyOf /properties/c/anyOf /properties/c/anyOf @/c",
		"type /properties/c/anyOf/0/type /properties/c/anyOf/0/type @/c",
		"type /properties/c/anyOf/1/type /properties/c/anyOf/1/type @/c",
		"not /properties/d/not /properties/d/not @/d",
		"false /properties/e /properties/e @/e",
		"oneOf /properties/f/oneOf /properties/f/oneOf @/f",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for i := range fails {
		if err := fails[i].Check(); err != nil {
			t.Error(err)
		}
	}
}

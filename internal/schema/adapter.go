// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"

	"github.com/kottera/metais/internal/schemafail"
)

// The adapter turns the library's *jsonschema.ValidationError tree into
// ADR-006's schemafail tree (ADR-010 D5 item 5) from its structure alone:
// the SchemaURL, the InstanceLocation, the ErrorKind's type, KeywordPath
// and exported fields, and the Causes. It never reads the library's
// message text, so a library update cannot change a finding's wording:
// it never calls Error, LocalizedString or any other method that renders
// text, and it reads no field of type error. And it never relies on the
// library's order, which follows map iteration: every list it builds is
// sorted by the failure's own content.
//
// reflect is used here, to read a kind's fields, and nowhere else in the
// module (ADR-010 D11 item 9).

// unnamed are the kinds whose KeywordPath is empty, and the keyword each
// is a failure of.
var unnamed = map[reflect.Type]string{
	reflect.TypeFor[*kind.Not]():         "not",
	reflect.TypeFor[*kind.FalseSchema](): "false",
	reflect.TypeFor[*kind.RefCycle]():    "$ref",
}

// transparent are the kinds that stand for a schema, not a keyword: the
// root of a validation, and a subschema that failed in several ways.
// Their causes take their place.
var transparent = map[reflect.Type]bool{
	reflect.TypeFor[*kind.Schema](): true,
	reflect.TypeFor[*kind.Group]():  true,
}

// adapter carries what one conversion needs: the compiler, to place a
// schema URL in the package, and whether the schema is a package file at
// all (a meta-schema is not).
type adapter struct {
	c    *Compiler
	meta bool
}

// adapt converts the error of a validation against a package schema.
func (c *Compiler) adapt(ve *jsonschema.ValidationError) ([]schemafail.Failure, error) {
	return adapter{c: c}.convert(ve, "", ve.SchemaURL)
}

// adaptMeta converts the error of a schema file's validation against the
// 2020-12 meta-schema, whose keywords are no package entry's: the
// failures carry the meta-schema's keywords and the instance pointers into
// the schema file.
func (c *Compiler) adaptMeta(ve *jsonschema.ValidationError) ([]schemafail.Failure, error) {
	return adapter{c: c, meta: true}.convert(ve, "", ve.SchemaURL)
}

// convert returns the failures ve stands for: itself, or, for a
// transparent kind, its causes. kwLoc is the evaluation path to the
// schema at parentURL, which ve's SchemaURL extends.
func (a adapter) convert(ve *jsonschema.ValidationError, kwLoc, parentURL string) ([]schemafail.Failure, error) {
	if ve == nil || ve.ErrorKind == nil {
		return nil, fmt.Errorf("schema: the library gave a failure with no kind")
	}
	t := reflect.TypeOf(ve.ErrorKind)
	if strings.HasPrefix(ve.SchemaURL, parentURL) {
		kwLoc += unescapePointer(ve.SchemaURL[len(parentURL):])
	}
	if transparent[t] {
		return a.children(ve.Causes, kwLoc, ve.SchemaURL)
	}
	f := schemafail.Failure{InstancePointer: slices.Clone(ve.InstanceLocation)}
	if f.InstancePointer == nil {
		f.InstancePointer = []string{}
	}
	if !a.meta {
		entry, ptr, ok := a.c.locate(ve.SchemaURL)
		if !ok {
			base, frag, _ := strings.Cut(ve.SchemaURL, "#")
			if !metaSchemas[base] {
				return nil, fmt.Errorf("schema: a failure is in no schema file of the package")
			}
			// A keyword of a 2020-12 meta-schema a package schema refers
			// to: no entry holds it.
			ptr = unescapePointer(frag)
		}
		f.Entry, f.SchemaPointer = entry, ptr
	}
	childURL := ve.SchemaURL
	if ref, ok := ve.ErrorKind.(*kind.Reference); ok {
		f.Keyword = ref.Keyword
		f.SchemaPointer += "/" + escapeToken(ref.Keyword)
		kwLoc += "/" + escapeToken(ref.Keyword)
		f.KeywordLocation = kwLoc
		childURL = ref.URL
		kids, err := a.children(ve.Causes, kwLoc, childURL)
		if err != nil {
			return nil, err
		}
		f.Children = kids
		return []schemafail.Failure{f}, nil
	}
	path := ve.ErrorKind.KeywordPath()
	if len(path) == 0 {
		name, ok := unnamed[t]
		if !ok {
			return nil, fmt.Errorf("schema: the library gave a failure of kind %v with no keyword", t)
		}
		path = []string{name}
		if name == "false" {
			// A false schema is the schema itself, not a keyword in it.
			path = nil
		}
		f.Keyword = name
	} else {
		f.Keyword = path[0]
	}
	for _, p := range path {
		f.SchemaPointer += "/" + escapeToken(p)
		kwLoc += "/" + escapeToken(p)
	}
	f.KeywordLocation = kwLoc
	f.Data = fields(ve.ErrorKind)
	kids, err := a.children(ve.Causes, kwLoc, childURL)
	if err != nil {
		return nil, err
	}
	f.Children = kids
	return []schemafail.Failure{f}, nil
}

// unescapePointer decodes the percent-encoding the library writes each
// token of a URL's fragment in.
func unescapePointer(p string) string {
	toks := strings.Split(p, "/")
	for i, t := range toks {
		if d, err := url.PathUnescape(t); err == nil {
			toks[i] = d
		}
	}
	return strings.Join(toks, "/")
}

// children converts causes and sorts them.
func (a adapter) children(causes []*jsonschema.ValidationError, kwLoc, parentURL string) ([]schemafail.Failure, error) {
	var out []schemafail.Failure
	for _, c := range causes {
		fs, err := a.convert(c, kwLoc, parentURL)
		if err != nil {
			return nil, err
		}
		out = append(out, fs...)
	}
	slices.SortFunc(out, compareFailures)
	return out, nil
}

// compareFailures orders failures by their own content: where they are
// in the schemas, where in the instance, the keyword, the data and then
// the children, so that two trees holding the same failures sort the same
// way whatever order the library gave them in.
func compareFailures(a, b schemafail.Failure) int {
	if c := cmp.Or(
		strings.Compare(a.KeywordLocation, b.KeywordLocation),
		strings.Compare(a.Entry, b.Entry),
		strings.Compare(a.SchemaPointer, b.SchemaPointer),
		slices.Compare(a.InstancePointer, b.InstancePointer),
		strings.Compare(a.Keyword, b.Keyword),
		strings.Compare(dataKey(a.Data), dataKey(b.Data)),
		cmp.Compare(len(a.Children), len(b.Children)),
	); c != 0 {
		return c
	}
	for i := range a.Children {
		if c := compareFailures(a.Children[i], b.Children[i]); c != 0 {
			return c
		}
	}
	return 0
}

// dataKey is a canonical text of d, for ordering only.
func dataKey(d schemafail.Data) string {
	var b strings.Builder
	for _, k := range slices.Sorted(func(yield func(string) bool) {
		for k := range d {
			if !yield(k) {
				return
			}
		}
	}) {
		fmt.Fprintf(&b, "%s=%#v;", k, d[k])
	}
	return b.String()
}

// fields reads a kind's exported fields into Data, each under its name in
// snake case: strings, booleans, integers, exact numbers and slices of
// them. A field of another type is left out; none of them is text the
// library wrote, and an error's text is never read.
func fields(k jsonschema.ErrorKind) schemafail.Data {
	v := reflect.ValueOf(k)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	d := schemafail.Data{}
	for i := 0; i < v.NumField(); i++ {
		sf := v.Type().Field(i)
		if !sf.IsExported() {
			continue
		}
		if x, ok := datum(v.Field(i)); ok {
			d[snake(sf.Name)] = x
		}
	}
	if len(d) == 0 {
		return nil
	}
	return d
}

var (
	errorType = reflect.TypeFor[error]()
	ratType   = reflect.TypeFor[*big.Rat]()
)

// datum converts one field's value to a type schemafail.Data carries.
func datum(v reflect.Value) (any, bool) {
	if v.Type() == errorType || v.Type().Implements(errorType) && v.Kind() != reflect.Struct {
		return nil, false
	}
	if v.Type() == ratType {
		if v.IsNil() {
			return nil, false
		}
		n, ok := decimal(v.Interface().(*big.Rat))
		return n, ok
	}
	switch v.Kind() {
	case reflect.String:
		return v.String(), true
	case reflect.Bool:
		return v.Bool(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(v.Int()), true
	case reflect.Interface:
		if v.IsNil() {
			return nil, false
		}
		return scalar(v.Elem().Interface())
	case reflect.Slice, reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			// Bytes are content the library decoded, not a field.
			return nil, false
		}
		return list(v)
	}
	return nil, false
}

// scalar converts a value the library holds as any: a JSON scalar of the
// instance or the schema.
func scalar(x any) (any, bool) {
	switch x := x.(type) {
	case string:
		return x, true
	case bool:
		return x, true
	case json.Number:
		// The literal text toLibrary gave the library.
		return schemafail.Number(x), true
	}
	return nil, false
}

// list converts a slice or array whose elements all convert to one type.
func list(v reflect.Value) (any, bool) {
	var strs []string
	var bools []bool
	var ints []int
	var nums []schemafail.Number
	for i := 0; i < v.Len(); i++ {
		x, ok := datum(v.Index(i))
		if !ok {
			return nil, false
		}
		switch x := x.(type) {
		case string:
			strs = append(strs, x)
		case bool:
			bools = append(bools, x)
		case int:
			ints = append(ints, x)
		case schemafail.Number:
			nums = append(nums, x)
		default:
			return nil, false
		}
	}
	switch n := v.Len(); {
	case len(strs) == n:
		return nonNil(strs), true
	case len(bools) == n:
		return bools, true
	case len(ints) == n:
		return ints, true
	case len(nums) == n:
		return nums, true
	}
	return nil, false
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// decimal writes r exactly as a JSON number. Every number the library
// builds a big.Rat from is a decimal, whose denominator has no prime
// factor but 2 and 5; any other has no exact decimal form and is left out.
func decimal(r *big.Rat) (schemafail.Number, bool) {
	den := new(big.Int).Set(r.Denom())
	places := 0
	for _, p := range []int64{2, 5} {
		n, m := 0, new(big.Int)
		for {
			q, rem := new(big.Int).QuoRem(den, big.NewInt(p), m)
			if rem.Sign() != 0 {
				break
			}
			den, n = q, n+1
		}
		places = max(places, n)
	}
	if den.Cmp(big.NewInt(1)) != 0 {
		return "", false
	}
	s := r.FloatString(places)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	if s == "-0" {
		s = "0"
	}
	return schemafail.Number(s), true
}

// snake converts a Go field name to snake case: Got is got, KeywordLocation1
// is keyword_location1.
func snake(name string) string {
	var b strings.Builder
	for i, r := range name {
		if unicode.IsUpper(r) {
			if i > 0 && !unicode.IsUpper(rune(name[i-1])) {
				b.WriteByte('_')
			}
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

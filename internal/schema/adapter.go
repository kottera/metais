// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"

	"metais.org/metais/internal/jsonx"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/schemafail"
)

// The adapter turns the library's *jsonschema.ValidationError tree into
// ADR-006's schemafail tree (ADR-010 D5 item 5), meeting the Data contract
// schemafail.Map reads, from the tree's structure alone: the SchemaURL,
// the InstanceLocation, the ErrorKind's type and typed fields, and the
// Causes. It never reads the library's message text, so a library update
// cannot change a finding's wording: it calls no method that renders text
// and reads no field of type error. And it never relies on the library's
// order, which follows map iteration: every list it builds is sorted by
// the failure's own content, and so is every list of names a kind holds.

// adapter carries what one conversion needs: the compiler, to place a
// schema URL in the package and read a schema file as authored, and
// whether the schema is a package file at all (a meta-schema is not).
type adapter struct {
	c    *Compiler
	meta bool
}

// adapt converts the error of a validation against a package schema.
func (c *Compiler) adapt(ve *jsonschema.ValidationError) ([]schemafail.Failure, error) {
	return adapter{c: c}.convert(ve, "", ve.SchemaURL, nil)
}

// adaptMeta converts the error of a schema file's validation against the
// 2020-12 meta-schema, whose keywords are no package entry's: the failures
// carry the meta-schema's keywords and the instance pointers into the
// schema file.
func (c *Compiler) adaptMeta(ve *jsonschema.ValidationError) ([]schemafail.Failure, error) {
	return adapter{c: c, meta: true}.convert(ve, "", ve.SchemaURL, nil)
}

// convert returns the failures ve stands for: itself, its causes for a
// kind that stands for a schema rather than a keyword, or one failure per
// member for additionalProperties: false. kwLoc is the evaluation path to
// the schema at parentURL, which ve's SchemaURL extends.
func (a adapter) convert(ve *jsonschema.ValidationError, kwLoc, parentURL string, inst []string) ([]schemafail.Failure, error) {
	if ve == nil || ve.ErrorKind == nil {
		return nil, fmt.Errorf("schema: the library gave a failure with no kind")
	}
	parentLoc := kwLoc
	if strings.HasPrefix(ve.SchemaURL, parentURL) {
		kwLoc += unescapePointer(strings.TrimPrefix(ve.SchemaURL[len(parentURL):], "#"))
	}
	switch ve.ErrorKind.(type) {
	case *kind.Schema, *kind.Group:
		// The root of a validation, or a subschema that failed in several
		// ways: its causes take its place.
		return a.children(ve.Causes, kwLoc, ve.SchemaURL, ve.InstanceLocation)
	}
	f := schemafail.Failure{InstancePointer: slices.Clone(ve.InstanceLocation), KeywordLocation: kwLoc}
	if f.InstancePointer == nil {
		f.InstancePointer = []string{}
	}
	if err := a.place(&f, ve.SchemaURL); err != nil {
		return nil, err
	}
	// keyword makes f the failure of keyword kw of the schema it is in.
	keyword := func(kw string) {
		f.Keyword = kw
		f.SchemaPointer += "/" + escapeToken(kw)
		f.KeywordLocation += "/" + escapeToken(kw)
	}
	childURL, keepChildren := ve.SchemaURL, true
	switch k := ve.ErrorKind.(type) {
	case *kind.Reference:
		keyword(k.Keyword)
		childURL, kwLoc = k.URL, f.KeywordLocation
	case *kind.FalseSchema:
		f.Keyword = schemafail.KeywordFalse
		if a.isItems(f.SchemaPointer) {
			f.Data = schemafail.Data{"prefix_count": a.length(f.Entry, strings.TrimSuffix(f.SchemaPointer, "/items")+"/prefixItems")}
		}
	case *kind.AdditionalProperties:
		var out []schemafail.Failure
		for _, p := range slices.Sorted(slices.Values(k.Properties)) {
			out = append(out, schemafail.Failure{
				Keyword: schemafail.KeywordFalse, KeywordLocation: f.KeywordLocation + "/additionalProperties",
				Entry: f.Entry, SchemaPointer: f.SchemaPointer + "/additionalProperties",
				InstancePointer: append(slices.Clone(f.InstancePointer), p),
			})
		}
		return out, nil
	case *kind.PropertyNames:
		// The library's SchemaURL is the propertyNames subschema itself,
		// and its InstanceLocation shares a slice later validations write
		// to: the instance is the one the path from the parent reaches.
		f.Keyword = "propertyNames"
		f.Data = schemafail.Data{"property": k.Property}
		if in, ok := instanceAlong(inst, pointerTokens(strings.TrimPrefix(kwLoc, parentLoc))); ok && strings.HasPrefix(kwLoc, parentLoc) {
			f.InstancePointer = in
		}
		keepChildren = false
	case *kind.Not:
		keyword("not")
		keepChildren = false
	case *kind.AllOf:
		keyword("allOf")
	case *kind.AnyOf:
		keyword("anyOf")
		f.Data = schemafail.Data{"branch_count": a.length(f.Entry, f.SchemaPointer)}
	case *kind.OneOf:
		keyword("oneOf")
		f.Data = schemafail.Data{"branch_count": a.length(f.Entry, f.SchemaPointer)}
		if len(k.Subschemas) > 0 {
			f.Data["matched"] = slices.Sorted(slices.Values(k.Subschemas))
			keepChildren = false
		}
	case *kind.Contains:
		keyword("contains")
		keepChildren = false
	case *kind.RefCycle:
		keyword("$ref")
		keepChildren = false
	case *uniqueKeysKind:
		keyword("uniqueKeys")
		f.Data = schemafail.Data{"duplicates": []int{k.Duplicates[0], k.Duplicates[1]}, "keys": slices.Clone(k.Keys)}
		if k.At != nil {
			f.Data["at"] = slices.Clone(k.At)
		}
		keepChildren = false
	case *coverageKind:
		keyword("coverage")
		f.Data = schemafail.Data{"missing": k.Missing, "keys": []string{k.Key}}
		keepChildren = false
	default:
		kw, data, err := leaf(ve.ErrorKind)
		if err != nil {
			return nil, err
		}
		keyword(kw)
		f.Data = data
		// minContains' and maxContains' causes are the items contains
		// refused, a yes-or-no context (ADR-003 D3 item 7).
		keepChildren = false
	}
	if keepChildren && len(ve.Causes) > 0 {
		kids, err := a.children(ve.Causes, kwLoc, childURL, ve.InstanceLocation)
		if err != nil {
			return nil, err
		}
		f.Children = kids
	}
	if n, ok := f.Data["branch_count"]; ok && n == 0 {
		// An anyOf or oneOf of no package file, a meta-schema's: with no
		// branch matching, every branch failed and is a child, so the
		// highest branch is the last.
		f.Data["branch_count"] = branches(f)
	}
	return []schemafail.Failure{f}, nil
}

// leaf returns the keyword and Data of a kind that has no subschema.
func leaf(k jsonschema.ErrorKind) (string, schemafail.Data, error) {
	count := func(limit, n int) schemafail.Data { return schemafail.Data{"limit": limit, "count": n} }
	switch k := k.(type) {
	case *kind.Type:
		return "type", schemafail.Data{"want": slices.Sorted(slices.Values(k.Want)), "got": k.Got}, nil
	case *kind.Enum:
		allowed := make([]string, len(k.Want))
		for i, w := range k.Want {
			allowed[i] = canonicalText(w)
		}
		d := schemafail.Data{"allowed": allowed}
		if s, ok := k.Got.(string); ok {
			d["got"] = s
		}
		return "enum", d, nil
	case *kind.Const:
		return "const", schemafail.Data{"expected": canonicalText(k.Want)}, nil
	case *kind.Required:
		return "required", schemafail.Data{"missing": slices.Sorted(slices.Values(k.Missing))}, nil
	case *kind.DependentRequired:
		return "dependentRequired", schemafail.Data{"missing": slices.Sorted(slices.Values(k.Missing)), "because": k.Prop}, nil
	case *kind.MinProperties:
		return "minProperties", count(k.Want, k.Got), nil
	case *kind.MaxProperties:
		return "maxProperties", count(k.Want, k.Got), nil
	case *kind.MinItems:
		return "minItems", count(k.Want, k.Got), nil
	case *kind.MaxItems:
		return "maxItems", count(k.Want, k.Got), nil
	case *kind.MinContains:
		return "minContains", count(k.Want, len(k.Got)), nil
	case *kind.MaxContains:
		return "maxContains", count(k.Want, len(k.Got)), nil
	case *kind.MinLength:
		return "minLength", schemafail.Data{"limit": k.Want, "length": k.Got}, nil
	case *kind.MaxLength:
		return "maxLength", schemafail.Data{"limit": k.Want, "length": k.Got}, nil
	case *kind.Pattern:
		return "pattern", schemafail.Data{"pattern": k.Want}, nil
	case *kind.Format:
		return "format", schemafail.Data{"format": k.Want}, nil
	case *kind.UniqueItems:
		d := []int{k.Duplicates[0], k.Duplicates[1]}
		slices.Sort(d)
		return "uniqueItems", schemafail.Data{"duplicates": d}, nil
	case *kind.Minimum:
		return bound("minimum", "limit", k.Want, k.Got)
	case *kind.Maximum:
		return bound("maximum", "limit", k.Want, k.Got)
	case *kind.ExclusiveMinimum:
		return bound("exclusiveMinimum", "limit", k.Want, k.Got)
	case *kind.ExclusiveMaximum:
		return bound("exclusiveMaximum", "limit", k.Want, k.Got)
	case *kind.MultipleOf:
		return bound("multipleOf", "divisor", k.Want, k.Got)
	}
	// The content keywords are never asserted (ADR-003 D3 item 4), and the
	// rest are earlier drafts'.
	return "", nil, fmt.Errorf("schema: the library gave a failure of kind %T, which the adapter does not know", k)
}

// bound is the Data of a numeric keyword: the schema's number under name,
// and the instance's under value.
func bound(kw, name string, limit, got *big.Rat) (string, schemafail.Data, error) {
	l, ok1 := decimal(limit)
	v, ok2 := decimal(got)
	if !ok1 || !ok2 {
		return "", nil, fmt.Errorf("schema: a %s failure holds a number with no decimal form", kw)
	}
	return kw, schemafail.Data{name: l, "value": v}, nil
}

// canonicalText is a value as the library holds it, in RFC 8785's text.
func canonicalText(v any) string {
	return string(jsonx.Canonical(fromLibrary(v)))
}

// fromLibrary converts a value the library holds, which toLibrary or the
// library's own reading of a schema made, back to the data model.
func fromLibrary(v any) value.Value {
	switch v := v.(type) {
	case nil:
		return value.Null{}
	case bool:
		return value.Bool(v)
	case json.Number:
		if n, err := value.ParseNumber(string(v)); err == nil {
			return n
		}
	case string:
		if s, err := value.NewString(v); err == nil {
			return s
		}
	case []any:
		elems := make([]value.Value, len(v))
		for i, e := range v {
			elems[i] = fromLibrary(e)
		}
		if a, err := value.NewArray(elems...); err == nil {
			return a
		}
	case map[string]any:
		members := make([]value.Member, 0, len(v))
		for k, e := range v {
			members = append(members, value.Member{Name: k, Value: fromLibrary(e)})
		}
		if o, err := value.NewObject(members...); err == nil {
			return o
		}
	}
	panic(fmt.Sprintf("schema: the library holds a %T that is no value of the data model", v))
}

// place sets f's entry and schema pointer from the library's URL for the
// schema f is in.
func (a adapter) place(f *schemafail.Failure, schemaURL string) error {
	if a.meta {
		return nil
	}
	entry, ptr, ok := a.c.locate(schemaURL)
	if !ok {
		base, frag, _ := strings.Cut(schemaURL, "#")
		if !metaSchemas[base] {
			return fmt.Errorf("schema: a failure is in no schema file of the package")
		}
		// A keyword of a 2020-12 meta-schema a package schema refers to:
		// no entry holds it.
		ptr = unescapePointer(frag)
	}
	f.Entry, f.SchemaPointer = entry, ptr
	return nil
}

// length returns the length of the array at ptr in entry, or 0.
func (a adapter) length(entry, ptr string) int {
	if arr, ok := a.c.lookup(entry, ptr).(value.Array); ok {
		return arr.Len()
	}
	return 0
}

// isItems reports whether the schema at ptr is the value of an items
// keyword, rather than, say, a property named items.
func (a adapter) isItems(ptr string) bool {
	toks := pointerTokens(ptr)
	return len(toks) > 0 && toks[len(toks)-1] == "items" && schemaPath(toks[:len(toks)-1])
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

// pointerTokens decodes an RFC 6901 pointer.
func pointerTokens(p string) []string {
	if p == "" {
		return nil
	}
	toks := strings.Split(strings.TrimPrefix(p, "/"), "/")
	for i, t := range toks {
		toks[i] = tokenUnescaper.Replace(t)
	}
	return toks
}

var tokenUnescaper = strings.NewReplacer("~1", "/", "~0", "~")

// schemaPath reports whether toks, read from a schema, end at a schema.
func schemaPath(toks []string) bool {
	return walkShape(toks, func(string) bool { return false }) == len(toks)
}

// conditional returns the index of the first then or else keyword along
// toks, read from a schema, or -1 when the path reaches none while it is
// among schemas.
func conditional(toks []string) int {
	i := walkShape(toks, func(t string) bool { return t == "then" || t == "else" })
	if i < len(toks) && (toks[i] == "then" || toks[i] == "else") {
		return i
	}
	return -1
}

// walkShape reads toks from a schema, keyword by keyword, each entered by
// its shape in the keywords table, and returns the index of the keyword
// at which stop held or the path left the schemas for a keyword's value,
// or len(toks) when it ends at a schema.
func walkShape(toks []string, stop func(string) bool) int {
	inSchema := true
	for i, t := range toks {
		if !inSchema {
			// An array index or a member name, which names a subschema.
			inSchema = true
			continue
		}
		if stop(t) {
			return i
		}
		switch keywords[t] {
		case shapeSchema:
		case shapeSchemaArray, shapeSchemaMap:
			inSchema = false
		default:
			return i
		}
	}
	if !inSchema {
		return len(toks) - 1
	}
	return len(toks)
}

// branches returns one more than the highest branch index among f's
// children and their matched branches.
func branches(f schemafail.Failure) int {
	n := 0
	if m, ok := f.Data["matched"].([]int); ok {
		for _, i := range m {
			n = max(n, i+1)
		}
	}
	for _, c := range f.Children {
		toks := pointerTokens(strings.TrimPrefix(c.KeywordLocation, f.KeywordLocation))
		if len(toks) > 0 {
			if i, err := strconv.Atoi(toks[0]); err == nil {
				n = max(n, i+1)
			}
		}
	}
	return n
}

// instanceAlong returns inst extended by the instance tokens the schema
// path toks enters, read from a schema: a properties member's name. It
// returns false when the path enters a member or item it does not name.
func instanceAlong(inst, toks []string) ([]string, bool) {
	out := append([]string{}, inst...)
	for i := 0; i < len(toks); i++ {
		switch t := toks[i]; {
		case t == "properties":
			if i+1 < len(toks) {
				out = append(out, toks[i+1])
				i++
			}
		case consumes[t]:
			return nil, false
		case keywords[t] == shapeSchemaArray || keywords[t] == shapeSchemaMap:
			i++
		case keywords[t] == shapeValue:
			return out, true
		}
	}
	return out, true
}

// entered counts the instance tokens the schema path toks enters.
func entered(toks []string) int {
	n := 0
	for i := 0; i < len(toks); i++ {
		switch t := toks[i]; {
		case t == "properties" || consumes[t]:
			n++
			if keywords[t] != shapeSchema {
				i++
			}
		case keywords[t] == shapeSchemaArray || keywords[t] == shapeSchemaMap:
			i++
		case keywords[t] == shapeValue:
			return n
		}
	}
	return n
}

// consumes are the keywords, besides properties, whose subschemas apply to
// a member or an item of the instance.
var consumes = map[string]bool{
	"patternProperties": true, "additionalProperties": true, "unevaluatedProperties": true,
	"prefixItems": true, "items": true, "unevaluatedItems": true, "contains": true,
}

// children converts causes, sets each conditional's failures beneath a
// then or else node, which the library does not make and schemafail.Map
// needs, and sorts them. kwLoc is the evaluation path to the schema the
// causes are in, at the instance location inst.
func (a adapter) children(causes []*jsonschema.ValidationError, kwLoc, parentURL string, inst []string) ([]schemafail.Failure, error) {
	var out []schemafail.Failure
	for _, c := range causes {
		fs, err := a.convert(c, kwLoc, parentURL, inst)
		if err != nil {
			return nil, err
		}
		out = append(out, fs...)
	}
	return nest(out, kwLoc, inst), nil
}

// nest puts each failure of fs whose evaluation path from loc passes a
// then or else beneath a node for it, and sorts every level.
func nest(fs []schemafail.Failure, loc string, inst []string) []schemafail.Failure {
	var out []schemafail.Failure
	nodes := map[string]int{}
	for _, f := range fs {
		toks := pointerTokens(strings.TrimPrefix(f.KeywordLocation, loc))
		j := conditional(toks)
		// A failure of the then or else schema itself, a false one, is the
		// node's child too.
		if !strings.HasPrefix(f.KeywordLocation, loc) || j < 0 || (j == len(toks)-1 && f.Keyword == toks[j]) {
			out = append(out, f)
			continue
		}
		var rel, rest strings.Builder
		for _, t := range toks[:j+1] {
			rel.WriteString("/" + escapeToken(t))
		}
		for _, t := range toks[j+1:] {
			rest.WriteString("/" + escapeToken(t))
		}
		nodeLoc := loc + rel.String()
		i, ok := nodes[nodeLoc]
		if !ok {
			// The conditional validates the instance its schema does: the
			// one the path reaches from loc, or else the child's, less what
			// the child's own path entered.
			at, known := instanceAlong(inst, toks[:j])
			if n := len(f.InstancePointer) - entered(toks[j+1:]); !known && n >= 0 {
				at = slices.Clone(f.InstancePointer[:n])
			}
			i = len(out)
			nodes[nodeLoc] = i
			out = append(out, schemafail.Failure{
				Keyword: toks[j], KeywordLocation: nodeLoc, Entry: f.Entry,
				SchemaPointer:   strings.TrimSuffix(f.SchemaPointer, rest.String()),
				InstancePointer: at,
			})
		}
		out[i].Children = append(out[i].Children, f)
	}
	for _, i := range nodes {
		out[i].Children = nest(out[i].Children, out[i].KeywordLocation, out[i].InstancePointer)
	}
	slices.SortFunc(out, compareFailures)
	return out
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
	); c != 0 {
		return c
	}
	// The data's text is written only for failures the fields above do
	// not order, which keeps a sort of many failures from writing it at
	// every comparison.
	if c := cmp.Or(
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

// decimal writes r exactly as a JSON number. Every number the library
// builds a big.Rat from is a decimal, whose denominator has no prime
// factor but 2 and 5; any other has no exact decimal form.
func decimal(r *big.Rat) (schemafail.Number, bool) {
	if r == nil {
		return "", false
	}
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

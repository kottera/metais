// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import (
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/regexprofile"
	"github.com/kottera/metais/internal/schema"
)

// allTypes is every JSON type: the types of a schema with no `type`.
const allTypes = schema.TypeNull | schema.TypeBoolean | schema.TypeObject | schema.TypeArray |
	schema.TypeNumber | schema.TypeString | schema.TypeInteger

// annotations are the keywords a schema may hold and still be open
// (ADR-007 D3 item 4.2): the author said nothing checkable.
var annotations = map[string]bool{
	"title": true, "description": true, "$comment": true, "examples": true, "default": true,
	"deprecated": true, "readOnly": true, "writeOnly": true, "$id": true, "$anchor": true,
	"$schema": true, "$defs": true,
}

// member is one node of an applicable set: its pointer, and whether it
// applies whenever n does, being n itself or reached only through `$ref`
// and `allOf`. Only such a node's maxItems or `items: false` rules an
// index out (ADR-007 D3 item 6): a branch of anyOf, oneOf or a
// conditional may not apply.
type member struct {
	ptr  string
	must bool
}

// applicable is a node's applicable set (ADR-007 D3 item 4.1) and whether
// it is open (item 4.2).
type applicable struct {
	members []member
	open    bool
}

// applicableSet returns A(n) for the node at ptr of input's schema: n,
// and every subschema reached from it through `$ref`, `$dynamicRef`,
// `allOf`, `anyOf`, `oneOf`, `if`, `then`, `else` and the values of
// `dependentSchemas`, never through `not`. A `$dynamicRef` reaches its
// static target and, where the input's schema resources declare more than
// one `$dynamicAnchor` of its name, every one of them.
func (a *analyzer) applicableSet(input, ptr string) *applicable {
	key := input + "\x00" + ptr
	if ap, ok := a.appl[key]; ok {
		return ap
	}
	ap := &applicable{}
	index := map[string]int{}
	var visit func(ptr string, must bool)
	visit = func(ptr string, must bool) {
		if i, seen := index[ptr]; seen {
			if must && !ap.members[i].must {
				ap.members[i].must = true
				a.follow(input, ptr, true, visit)
			}
			return
		}
		v, ok := a.walker.Lookup(ptr)
		if !ok {
			return
		}
		index[ptr] = len(ap.members)
		ap.members = append(ap.members, member{ptr, must})
		if isOpen(v) {
			ap.open = true
		}
		a.follow(input, ptr, must, visit)
	}
	visit(ptr, true)
	a.appl[key] = ap
	return ap
}

// follow visits every subschema A(n) takes from the schema at ptr.
func (a *analyzer) follow(input, ptr string, must bool, visit func(string, bool)) {
	o, ok := a.object(ptr)
	if !ok {
		return
	}
	if t, ok, err := a.walker.Reference(ptr, "$ref"); err == nil && ok {
		visit(t, must)
	}
	if t, ok, err := a.walker.Reference(ptr, "$dynamicRef"); err == nil && ok {
		visit(t, false)
		for _, d := range a.dynamicTargets(input, o) {
			visit(d, false)
		}
	}
	for _, kw := range []string{"allOf", "anyOf", "oneOf"} {
		if arr, ok := get[value.Array](o, kw); ok {
			for i := range arr.Len() {
				visit(ptr+"/"+kw+"/"+itoa(i), must && kw == "allOf")
			}
		}
	}
	for _, kw := range []string{"if", "then", "else"} {
		if _, ok := o.Get(kw); ok {
			visit(ptr+"/"+kw, false)
		}
	}
	if deps, ok := get[value.Object](o, "dependentSchemas"); ok {
		for _, name := range deps.Names() {
			visit(ptr+"/dependentSchemas/"+escape(name), false)
		}
	}
}

// dynamicTargets returns every schema of input's resources whose
// `$dynamicAnchor` is the name the `$dynamicRef` of o names, when there is
// more than one; none otherwise, the static target being the only one.
func (a *analyzer) dynamicTargets(input string, o value.Object) []string {
	r, ok := get[value.String](o, "$dynamicRef")
	if !ok {
		return nil
	}
	u, err := url.Parse(r.Text())
	if err != nil || u.Fragment == "" || strings.HasPrefix(u.Fragment, "/") {
		return nil
	}
	files := a.resources(input)
	var out []string
	for _, p := range a.walker.DynamicAnchors(u.Fragment) {
		if files[fileOf(p)] {
			out = append(out, p)
		}
	}
	if len(out) < 2 {
		return nil
	}
	return out
}

// resources returns the files of input's schema resources: its schema
// file, and every file a reference from one of them reaches.
func (a *analyzer) resources(input string) map[string]bool {
	if r, ok := a.files[input]; ok {
		return r
	}
	files := map[string]bool{fileOf(a.inputs[input].root): true}
	refs := a.walker.References()
	for changed := true; changed; {
		changed = false
		for _, site := range refs {
			if !files[fileOf(site)] {
				continue
			}
			for _, kw := range []string{"$ref", "$dynamicRef"} {
				if t, ok, err := a.walker.Reference(site, kw); err == nil && ok && !files[fileOf(t)] {
					files[fileOf(t)] = true
					changed = true
				}
			}
		}
	}
	a.files[input] = files
	return files
}

// isOpen reports whether a schema is open (ADR-007 D3 item 4.2): the
// boolean schema true, or an object schema holding annotations only.
func isOpen(v value.Value) bool {
	switch s := v.(type) {
	case value.Bool:
		return bool(s)
	case value.Object:
		for _, name := range s.Names() {
			if !annotations[name] {
				return false
			}
		}
		return true
	}
	return false
}

// typesOf returns the types of the node at ptr (ADR-007 D3 item 5): the
// types its `type` allows, all when it has none, intersected through
// `$ref` and `allOf` and with the union of its anyOf branches and of its
// oneOf branches. `if`, `then` and `else` do not narrow them. A number
// type includes the integers.
func (a *analyzer) typesOf(ptr string) schema.TypeSet {
	if t, ok := a.types[ptr]; ok {
		return t
	}
	a.types[ptr] = allTypes // a cycle through ptr narrows nothing
	t := a.computeTypes(ptr)
	a.types[ptr] = t
	return t
}

func (a *analyzer) computeTypes(ptr string) schema.TypeSet {
	v, ok := a.walker.Lookup(ptr)
	if !ok {
		return allTypes
	}
	switch s := v.(type) {
	case value.Bool:
		if s {
			return allTypes
		}
		return 0
	case value.Object:
	default:
		return allTypes
	}
	o := v.(value.Object)
	t := allTypes
	if tv, ok := o.Get("type"); ok {
		t = typeSet(tv)
	}
	if target, ok, err := a.walker.Reference(ptr, "$ref"); err == nil && ok {
		t &= a.typesOf(target)
	}
	if arr, ok := get[value.Array](o, "allOf"); ok {
		for i := range arr.Len() {
			t &= a.typesOf(ptr + "/allOf/" + itoa(i))
		}
	}
	for _, kw := range []string{"anyOf", "oneOf"} {
		if arr, ok := get[value.Array](o, kw); ok {
			var u schema.TypeSet
			for i := range arr.Len() {
				u |= a.typesOf(ptr + "/" + kw + "/" + itoa(i))
			}
			t &= u
		}
	}
	return t
}

// typeSet reads a `type` value, a number including the integers.
func typeSet(v value.Value) schema.TypeSet {
	var names []value.Value
	switch t := v.(type) {
	case value.String:
		names = []value.Value{t}
	case value.Array:
		for _, e := range t.All() {
			names = append(names, e)
		}
	default:
		return allTypes
	}
	var set schema.TypeSet
	for _, n := range names {
		s, _ := n.(value.String)
		switch s.Text() {
		case "null":
			set |= schema.TypeNull
		case "boolean":
			set |= schema.TypeBoolean
		case "object":
			set |= schema.TypeObject
		case "array":
			set |= schema.TypeArray
		case "number":
			set |= schema.TypeNumber | schema.TypeInteger
		case "string":
			set |= schema.TypeString
		case "integer":
			set |= schema.TypeInteger
		}
	}
	return set
}

// typeNames returns the names of t's types, in the order of schema's type
// bits, with integer named only where number is not.
func typeNames(t schema.TypeSet) []string {
	var out []string
	for _, n := range []struct {
		t    schema.TypeSet
		name string
	}{
		{schema.TypeNull, "null"}, {schema.TypeBoolean, "boolean"}, {schema.TypeObject, "object"},
		{schema.TypeArray, "array"}, {schema.TypeNumber, "number"}, {schema.TypeString, "string"},
		{schema.TypeInteger, "integer"},
	} {
		if t&n.t == 0 || (n.t == schema.TypeInteger && t&schema.TypeNumber != 0) {
			continue
		}
		out = append(out, n.name)
	}
	return out
}

// numeric reports whether t holds a number or an integer.
func numeric(t schema.TypeSet) bool {
	return t&(schema.TypeNumber|schema.TypeInteger) != 0
}

// rejection is why one alternative rejects a segment.
type rejection struct {
	code     codeKind
	input    string
	ptr      string // the schema node, "" for a value that is not a schema's
	declared []string
	types    schema.TypeSet
}

// codeKind is a rejection's finding code.
type codeKind uint8

const (
	rejUnknownField codeKind = iota + 1
	rejNotAnObject
	rejNotAnArray
	rejOutOfRange
)

// schemaProperty resolves the property segment p on the node at ptr of
// input's schema (ADR-007 D3 item 4). Liquid's command names apply only
// to a ".p" segment, for which dot is set.
func (a *analyzer) schemaProperty(input, ptr, p string, dot bool) (set, *rejection) {
	ap := a.applicableSet(input, ptr)
	if ap.open {
		return open, nil
	}
	var out set
	var declared []string
	for _, m := range ap.members {
		o, ok := a.object(m.ptr)
		if !ok {
			continue
		}
		matched := false
		if props, ok := get[value.Object](o, "properties"); ok {
			declared = append(declared, props.Names()...)
			if _, ok := props.Get(p); ok {
				out = append(out, schemaAlt(input, m.ptr+"/properties/"+escape(p)))
				matched = true
			}
		}
		if pats, ok := get[value.Object](o, "patternProperties"); ok {
			for _, pat := range pats.Names() {
				if a.matches(pat, p) {
					out = append(out, schemaAlt(input, m.ptr+"/patternProperties/"+escape(pat)))
					matched = true
				}
			}
		}
		if matched {
			continue
		}
		for _, kw := range []string{"additionalProperties", "unevaluatedProperties"} {
			v, ok := o.Get(kw)
			if !ok {
				continue
			}
			if v != value.Bool(false) {
				out = append(out, schemaAlt(input, m.ptr+"/"+kw))
			}
			break
		}
	}
	if len(out) > 0 {
		return union(out), nil
	}
	t := a.typesOf(ptr)
	switch {
	case !dot:
	case p == "size":
		if t&(schema.TypeArray|schema.TypeObject|schema.TypeString) != 0 {
			return num, nil
		}
	case p == "first" || p == "last":
		var r set
		if t&schema.TypeArray != 0 {
			i := 0
			if p == "last" {
				i = -1
			}
			if s, rej := a.schemaIndex(input, ptr, i); rej == nil {
				r = union(r, s)
			}
		}
		if t&schema.TypeString != 0 {
			r = union(r, str)
		}
		if len(r) > 0 {
			return r, nil
		}
	}
	slices.Sort(declared)
	declared = slices.Compact(declared)
	if t&schema.TypeObject == 0 {
		return nil, &rejection{code: rejNotAnObject, input: input, ptr: ptr, types: t}
	}
	return nil, &rejection{code: rejUnknownField, input: input, ptr: ptr, declared: declared}
}

// schemaIndex resolves the index segment [i] on the node at ptr of
// input's schema (ADR-007 D3 item 6).
func (a *analyzer) schemaIndex(input, ptr string, i int) (set, *rejection) {
	ap := a.applicableSet(input, ptr)
	if ap.open {
		return open, nil
	}
	var out set
	outOfRange := false
	for _, m := range ap.members {
		o, ok := a.object(m.ptr)
		if !ok {
			continue
		}
		prefix, _ := get[value.Array](o, "prefixItems")
		items, hasItems := o.Get("items")
		switch {
		case i < 0:
			for j := range prefix.Len() {
				out = append(out, schemaAlt(input, m.ptr+"/prefixItems/"+itoa(j)))
			}
			if hasItems && items != value.Bool(false) {
				out = append(out, schemaAlt(input, m.ptr+"/items"))
			}
		case i < prefix.Len():
			out = append(out, schemaAlt(input, m.ptr+"/prefixItems/"+itoa(i)))
		case hasItems && items == value.Bool(false):
			outOfRange = outOfRange || m.must
		case hasItems:
			out = append(out, schemaAlt(input, m.ptr+"/items"))
		}
		if mx, ok := get[value.Number](o, "maxItems"); ok && m.must && i >= 0 {
			if n, ok := mx.SafeInt(); ok && int64(i) >= n {
				outOfRange = true
			}
		}
	}
	t := a.typesOf(ptr)
	switch {
	case outOfRange:
		return nil, &rejection{code: rejOutOfRange, input: input, ptr: ptr}
	case len(out) > 0:
		return union(out), nil
	case t&schema.TypeArray == 0:
		return nil, &rejection{code: rejNotAnArray, input: input, ptr: ptr, types: t}
	}
	return open, nil
}

// schemaDynamic resolves a computed segment [expr] on the node at ptr of
// input's schema (ADR-007 D3 item 6): every candidate a property of any
// name could resolve to, and every array element's schema. It is empty
// when there is none.
func (a *analyzer) schemaDynamic(input, ptr string) set {
	ap := a.applicableSet(input, ptr)
	if ap.open {
		return open
	}
	var out set
	for _, m := range ap.members {
		o, ok := a.object(m.ptr)
		if !ok {
			continue
		}
		for _, kw := range []string{"properties", "patternProperties"} {
			if ps, ok := get[value.Object](o, kw); ok {
				for _, name := range ps.Names() {
					out = append(out, schemaAlt(input, m.ptr+"/"+kw+"/"+escape(name)))
				}
			}
		}
		for _, kw := range []string{"additionalProperties", "unevaluatedProperties"} {
			if v, ok := o.Get(kw); ok {
				if v != value.Bool(false) {
					out = append(out, schemaAlt(input, m.ptr+"/"+kw))
				}
				break
			}
		}
		out = append(out, a.itemSchemas(input, m.ptr, o)...)
	}
	return union(out)
}

// schemaElements returns the element of an array at the node at ptr of
// input's schema, as a for loop binds it and a filter reads it: every
// `prefixItems` and `items` schema in its applicable set; open for a node
// that may be an array and declares neither; unknown for one that is
// never an array.
func (a *analyzer) schemaElements(input, ptr string) set {
	ap := a.applicableSet(input, ptr)
	if ap.open {
		return open
	}
	var out set
	for _, m := range ap.members {
		if o, ok := a.object(m.ptr); ok {
			out = append(out, a.itemSchemas(input, m.ptr, o)...)
		}
	}
	switch {
	case len(out) > 0:
		return union(out)
	case a.typesOf(ptr)&schema.TypeArray != 0:
		return open
	}
	return unknown
}

// itemSchemas returns o's `prefixItems` schemas and its `items` unless it
// is false.
func (a *analyzer) itemSchemas(input, ptr string, o value.Object) []alt {
	var out []alt
	if prefix, ok := get[value.Array](o, "prefixItems"); ok {
		for j := range prefix.Len() {
			out = append(out, schemaAlt(input, ptr+"/prefixItems/"+itoa(j)))
		}
	}
	if items, ok := o.Get("items"); ok && items != value.Bool(false) {
		out = append(out, schemaAlt(input, ptr+"/items"))
	}
	return out
}

// matches reports whether p matches the patternProperties pattern pat,
// read by the package's regular-expression profile (ADR-009 D10). The
// schema compiler has already accepted every pattern; one that does not
// compile here matches, so that no reference is wrongly an error.
func (a *analyzer) matches(pat, p string) bool {
	re, ok := a.patterns[pat]
	if !ok {
		var err error
		re, err = regexprofile.Compile(pat, a.pkg.Limits, nil)
		if err != nil {
			re = nil
		}
		a.patterns[pat] = re
	}
	return re == nil || re.MatchString(p)
}

// object returns the object schema at ptr, and false when the schema
// there is a boolean or there is none.
func (a *analyzer) object(ptr string) (value.Object, bool) {
	v, ok := a.walker.Lookup(ptr)
	if !ok {
		return value.Object{}, false
	}
	o, ok := v.(value.Object)
	return o, ok
}

// get returns the member name of o as a T, if o has one of that kind.
func get[T value.Value](o value.Object, name string) (T, bool) {
	var zero T
	v, ok := o.Get(name)
	if !ok {
		return zero, false
	}
	t, ok := v.(T)
	return t, ok
}

// itoa returns i in decimal.
func itoa(i int) string { return strconv.Itoa(i) }

// escape returns name as a JSON Pointer reference token.
func escape(name string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(name)
}

// unescape returns the member name a JSON Pointer reference token names.
func unescape(tok string) string {
	return strings.NewReplacer("~1", "/", "~0", "~").Replace(tok)
}

// fileOf returns the schema file a pointer in the combined schema is in,
// "" for one outside its $defs.
func fileOf(ptr string) string {
	rest, ok := strings.CutPrefix(ptr, "/$defs/")
	if !ok {
		return ""
	}
	tok, _, _ := strings.Cut(rest, "/")
	return unescape(tok)
}

// inFile returns a pointer in the combined schema as the schema file and
// the pointer within it (ADR-007 D3 item 4.6's schema location).
func inFile(ptr string) (file, p string) {
	rest, ok := strings.CutPrefix(ptr, "/$defs/")
	if !ok {
		return "", ptr
	}
	tok, after, found := strings.Cut(rest, "/")
	if found {
		after = "/" + after
	}
	return unescape(tok), after
}

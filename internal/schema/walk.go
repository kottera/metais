// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/liquid/value"
)

// TypeSet is a set of JSON Schema type names (validation §6.1.1): a
// declared type set of ADR-003 D14 item 1. The zero TypeSet is empty.
type TypeSet uint8

// The type names, one bit each.
const (
	TypeNull TypeSet = 1 << iota
	TypeBoolean
	TypeObject
	TypeArray
	TypeNumber
	TypeString
	TypeInteger
)

// typeNames is each type's name, in bit order.
var typeNames = [...]string{"null", "boolean", "object", "array", "number", "string", "integer"}

// Has reports whether every type of u is in t.
func (t TypeSet) Has(u TypeSet) bool { return t&u == u }

// String returns the set's names in bit order, as a `type` array would
// list them: "[]" for the empty set.
func (t TypeSet) String() string {
	var names []string
	for i, n := range typeNames {
		if t&(1<<i) != 0 {
			names = append(names, strconv.Quote(n))
		}
	}
	return "[" + strings.Join(names, ",") + "]"
}

// typeNamed returns the type whose name is n.
func typeNamed(n string) (TypeSet, bool) {
	for i, name := range typeNames {
		if n == name {
			return 1 << i, true
		}
	}
	return 0, false
}

// Step is one step of an instance location: an object member by name or
// an array element by index. The walk needs to know which, because
// `properties` answers a member and `prefixItems` and `items` answer an
// index, and a token such as "0" can name either.
type Step struct {
	name    string
	index   int
	isIndex bool
}

// Member returns the step to the object member name.
func Member(name string) Step { return Step{name: name} }

// Index returns the step to the array element at i, which is not
// negative.
func Index(i int) Step { return Step{index: i, isIndex: true} }

// String returns the step as a JSON Pointer token.
func (s Step) String() string {
	if s.isIndex {
		return strconv.Itoa(s.index)
	}
	return escapeToken(s.name)
}

// Static is the static subschema at an instance location (ADR-003 D14
// item 1).
type Static struct {
	// Schema is the subschema the walk reached, before any `$ref` of its
	// own is followed: the schema an author wrote for the location.
	Schema value.Value
	// Pointer is Schema's JSON Pointer in the walked document.
	Pointer string
	// Types is the declared type set: the `type` of the first schema that
	// declares one in the chain from Schema through `$ref` targets, or the
	// empty set when none does.
	Types TypeSet
}

// Walk errors. Each is a schema the compiler refuses (ADR-003 D3 and D5),
// met here because the walk reads a schema without compiling it.
var (
	// ErrMalformed is returned for a keyword the walk reads whose value
	// does not have the form 2020-12 gives it.
	ErrMalformed = errors.New("schema: malformed keyword")
	// ErrUnresolved is returned for a `$ref` that names no resource,
	// anchor or location in the walked document. The 2020-12
	// meta-schemas are not in it: a walk does not follow a `$ref` to one.
	ErrUnresolved = errors.New("schema: unresolved reference")
	// ErrRefCycle is returned for a chain of `$ref`s that returns to a
	// schema it has passed through without taking a step.
	ErrRefCycle = errors.New("schema: $ref cycle")
	// ErrDuplicateURI is returned when two resources, or two anchors of
	// one resource, have one URI (ADR-003 D5 item 6).
	ErrDuplicateURI = errors.New("schema: duplicate URI")
)

// Walker finds static subschemas in one schema document: an input's
// schema file, or the combined schema that bundles them all (Bundle).
type Walker struct {
	doc       value.Value
	bases     map[string]*url.URL // schema pointer → its base URI
	resources map[string]string   // resource URI → pointer
	anchors   map[string]string   // resource URI "#" anchor → pointer
	dynamic   map[string][]string // $dynamicAnchor name → pointers
	refSites  []string            // every schema holding $ref or $dynamicRef
}

// NewWalker indexes doc, whose retrieval URI is base, for walking: every
// resource it holds by its `$id`, and every `$anchor` and
// `$dynamicAnchor`, each found only where a schema may stand. Walking the
// combined schema, base is CombinedURI; walking one schema file, it is
// that file's RetrievalURI.
func NewWalker(doc value.Value, base string) (*Walker, error) {
	b, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("%w: base %q: %v", ErrMalformed, base, err)
	}
	w := &Walker{
		doc:       doc,
		bases:     map[string]*url.URL{},
		resources: map[string]string{},
		anchors:   map[string]string{},
		dynamic:   map[string][]string{},
	}
	if _, hasID := idOf(doc); !hasID {
		w.resources[b.String()] = ""
	}
	if err := w.index(doc, "", b); err != nil {
		return nil, err
	}
	return w, nil
}

// idOf returns s's `$id` member, if s is an object schema with one.
func idOf(s value.Value) (value.Value, bool) {
	o, ok := s.(value.Object)
	if !ok {
		return nil, false
	}
	return o.Get("$id")
}

// The keywords whose values are schemas, by the shape of the value.
var (
	schemaKeywords = []string{
		"additionalProperties", "contains", "contentSchema", "else", "if",
		"items", "not", "propertyNames", "then", "unevaluatedItems",
		"unevaluatedProperties",
	}
	schemaArrayKeywords = []string{"allOf", "anyOf", "oneOf", "prefixItems"}
	schemaMapKeywords   = []string{"$defs", "dependentSchemas", "patternProperties", "properties"}
)

// index records the schema s at ptr, whose enclosing base URI is base,
// and every schema beneath it.
func (w *Walker) index(s value.Value, ptr string, base *url.URL) error {
	o, ok := s.(value.Object)
	if !ok {
		w.bases[ptr] = base
		return nil
	}
	if idv, ok := o.Get("$id"); ok {
		id, isStr := idv.(value.String)
		if !isStr {
			return fmt.Errorf("%w: %s/$id is a %s, not a string", ErrMalformed, ptr, idv.Kind())
		}
		uri, err := resolveID(base, id.Text(), "#"+ptr)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrMalformed, err)
		}
		if other, dup := w.resources[uri]; dup {
			return fmt.Errorf("%w: %s is the $id of both %q and %q", ErrDuplicateURI, uri, other, ptr)
		}
		w.resources[uri] = ptr
		base, _ = url.Parse(uri) // uri is url.URL's own text
	}
	w.bases[ptr] = base
	for _, kw := range []string{"$anchor", "$dynamicAnchor"} {
		av, ok := o.Get(kw)
		if !ok {
			continue
		}
		a, isStr := av.(value.String)
		if !isStr {
			return fmt.Errorf("%w: %s/%s is a %s, not a string", ErrMalformed, ptr, kw, av.Kind())
		}
		key := base.String() + "#" + a.Text()
		if other, dup := w.anchors[key]; dup && other != ptr {
			return fmt.Errorf("%w: %s names both %q and %q", ErrDuplicateURI, key, other, ptr)
		}
		w.anchors[key] = ptr
		if kw == "$dynamicAnchor" {
			w.dynamic[a.Text()] = append(w.dynamic[a.Text()], ptr)
		}
	}
	if _, ok := o.Get("$ref"); ok {
		w.refSites = append(w.refSites, ptr)
	} else if _, ok := o.Get("$dynamicRef"); ok {
		w.refSites = append(w.refSites, ptr)
	}
	for _, kw := range schemaKeywords {
		if sub, ok := o.Get(kw); ok {
			if err := w.index(sub, ptr+"/"+kw, base); err != nil {
				return err
			}
		}
	}
	for _, kw := range schemaArrayKeywords {
		v, ok := o.Get(kw)
		if !ok {
			continue
		}
		arr, isArr := v.(value.Array)
		if !isArr {
			return fmt.Errorf("%w: %s/%s is a %s, not an array", ErrMalformed, ptr, kw, v.Kind())
		}
		for i, sub := range arr.All() {
			if err := w.index(sub, ptr+"/"+kw+"/"+strconv.Itoa(i), base); err != nil {
				return err
			}
		}
	}
	for _, kw := range schemaMapKeywords {
		v, ok := o.Get(kw)
		if !ok {
			continue
		}
		m, isObj := v.(value.Object)
		if !isObj {
			return fmt.Errorf("%w: %s/%s is a %s, not an object", ErrMalformed, ptr, kw, v.Kind())
		}
		for name, sub := range m.All() {
			if err := w.index(sub, ptr+"/"+kw+"/"+escapeToken(name), base); err != nil {
				return err
			}
		}
	}
	return nil
}

// node is a schema in the walked document and where it is.
type node struct {
	v   value.Value
	ptr string
}

// Walk returns the static subschema at the instance location path, walked
// from the document's root through `$ref`, `properties`, `prefixItems` and
// `items` and through nothing else (ADR-003 D14 item 1). At each step the
// schema's own `properties` (for a member) or `prefixItems` and `items`
// (for an index) are tried first, and its `$ref` target when they do not
// answer the step. ok is false when no schema is statically declared at
// path. The walk never evaluates `allOf`, `anyOf`, `oneOf` or a
// conditional, so its answer does not depend on any data.
func (w *Walker) Walk(path []Step) (s Static, ok bool, err error) {
	cur := node{v: w.doc}
	for _, st := range path {
		cur, ok, err = w.step(cur, st)
		if err != nil || !ok {
			return Static{}, false, err
		}
	}
	types, err := w.declared(cur)
	if err != nil {
		return Static{}, false, err
	}
	return Static{Schema: cur.v, Pointer: cur.ptr, Types: types}, true, nil
}

// step returns the subschema that answers st from cur, following cur's
// `$ref` chain until a schema answers it or the chain ends.
func (w *Walker) step(cur node, st Step) (node, bool, error) {
	seen := map[string]bool{}
	for {
		if seen[cur.ptr] {
			return node{}, false, fmt.Errorf("%w: through %q", ErrRefCycle, cur.ptr)
		}
		seen[cur.ptr] = true
		o, isObj := cur.v.(value.Object)
		if !isObj {
			return node{}, false, nil
		}
		next, ok, err := answer(o, cur.ptr, st)
		if err != nil || ok {
			return next, ok, err
		}
		if cur, ok, err = w.ref(o, cur.ptr); err != nil || !ok {
			return node{}, false, err
		}
	}
}

// answer returns the subschema o itself gives for st: its `properties`
// member for a member step; its `prefixItems` element when the index is
// one it covers, and otherwise its `items`, for an index step.
func answer(o value.Object, ptr string, st Step) (node, bool, error) {
	if !st.isIndex {
		v, ok := o.Get("properties")
		if !ok {
			return node{}, false, nil
		}
		props, isObj := v.(value.Object)
		if !isObj {
			return node{}, false, fmt.Errorf("%w: %s/properties is a %s, not an object", ErrMalformed, ptr, v.Kind())
		}
		sub, ok := props.Get(st.name)
		if !ok {
			return node{}, false, nil
		}
		return node{v: sub, ptr: ptr + "/properties/" + escapeToken(st.name)}, true, nil
	}
	if st.index < 0 {
		return node{}, false, nil
	}
	if v, ok := o.Get("prefixItems"); ok {
		prefix, isArr := v.(value.Array)
		if !isArr {
			return node{}, false, fmt.Errorf("%w: %s/prefixItems is a %s, not an array", ErrMalformed, ptr, v.Kind())
		}
		if st.index < prefix.Len() {
			return node{v: prefix.At(st.index), ptr: ptr + "/prefixItems/" + strconv.Itoa(st.index)}, true, nil
		}
	}
	if v, ok := o.Get("items"); ok {
		switch v.(type) {
		case value.Object, value.Bool:
			return node{v: v, ptr: ptr + "/items"}, true, nil
		default:
			return node{}, false, fmt.Errorf("%w: %s/items is a %s, not a schema", ErrMalformed, ptr, v.Kind())
		}
	}
	return node{}, false, nil
}

// declared returns the declared type set at n: the `type` of the first
// schema in the chain from n through `$ref` targets that declares one.
func (w *Walker) declared(n node) (TypeSet, error) {
	seen := map[string]bool{}
	for {
		if seen[n.ptr] {
			return 0, fmt.Errorf("%w: through %q", ErrRefCycle, n.ptr)
		}
		seen[n.ptr] = true
		o, isObj := n.v.(value.Object)
		if !isObj {
			return 0, nil
		}
		if tv, ok := o.Get("type"); ok {
			return typeSet(tv, n.ptr)
		}
		next, ok, err := w.ref(o, n.ptr)
		if err != nil || !ok {
			return 0, err
		}
		n = next
	}
}

// typeSet reads a `type` value: one type name, or an array of them.
func typeSet(v value.Value, ptr string) (TypeSet, error) {
	var names []value.Value
	switch t := v.(type) {
	case value.String:
		names = []value.Value{t}
	case value.Array:
		for _, e := range t.All() {
			names = append(names, e)
		}
	default:
		return 0, fmt.Errorf("%w: %s/type is a %s, not a string or an array", ErrMalformed, ptr, v.Kind())
	}
	var set TypeSet
	for _, n := range names {
		s, isStr := n.(value.String)
		if !isStr {
			return 0, fmt.Errorf("%w: %s/type holds a %s, not a type name", ErrMalformed, ptr, n.Kind())
		}
		t, known := typeNamed(s.Text())
		if !known {
			return 0, fmt.Errorf("%w: %s/type holds %q, which is no type name", ErrMalformed, ptr, s.Text())
		}
		set |= t
	}
	return set, nil
}

// ref returns the target of the `$ref` of the schema o at ptr, resolved
// against ptr's base URI (ADR-003 D5 item 3); ok is false when o has no
// `$ref`. `$dynamicRef` is not followed (ADR-003 D14 item 1).
func (w *Walker) ref(o value.Object, ptr string) (node, bool, error) {
	return w.reference(o, ptr, "$ref")
}

// reference returns the static target of the reference keyword kw of the
// schema o at ptr: what ref returns for `$ref`, and for `$dynamicRef` its
// initial target, resolved as a `$ref` is, with no dynamic scope.
func (w *Walker) reference(o value.Object, ptr, kw string) (node, bool, error) {
	rv, ok := o.Get(kw)
	if !ok {
		return node{}, false, nil
	}
	r, isStr := rv.(value.String)
	if !isStr {
		return node{}, false, fmt.Errorf("%w: %s/%s is a %s, not a string", ErrMalformed, ptr, kw, rv.Kind())
	}
	ref, err := url.Parse(r.Text())
	if err != nil {
		return node{}, false, fmt.Errorf("%w: %s/%s %q is not a URI reference: %v", ErrMalformed, ptr, kw, r.Text(), err)
	}
	target := w.baseAt(ptr).ResolveReference(ref)
	frag := target.Fragment
	target.Fragment, target.RawFragment = "", ""
	res := target.String()
	rptr, found := w.resources[res]
	switch {
	case !found:
	case frag == "":
	case strings.HasPrefix(frag, "/"):
		rptr += frag
	default:
		rptr, found = w.anchors[res+"#"+frag]
	}
	if !found {
		return node{}, false, fmt.Errorf("%w: %s/%s %q", ErrUnresolved, ptr, kw, r.Text())
	}
	v, found := lookup(w.doc, rptr)
	if !found {
		return node{}, false, fmt.Errorf("%w: %s/%s %q", ErrUnresolved, ptr, kw, r.Text())
	}
	return node{v: v, ptr: rptr}, true, nil
}

// baseAt returns the base URI at ptr: that of the nearest schema at or
// above it that index recorded. A pointer fragment may name a location no
// keyword makes a schema, under an `x-` keyword for one; such a location
// takes its enclosing schema's base.
func (w *Walker) baseAt(ptr string) *url.URL {
	for {
		if b, ok := w.bases[ptr]; ok {
			return b
		}
		ptr = ptr[:strings.LastIndexByte(ptr, '/')]
	}
}

// lookup returns the value at the JSON Pointer ptr in doc (RFC 6901).
func lookup(doc value.Value, ptr string) (value.Value, bool) {
	if ptr == "" {
		return doc, true
	}
	cur := doc
	for _, tok := range strings.Split(ptr[1:], "/") {
		tok = strings.NewReplacer("~1", "/", "~0", "~").Replace(tok)
		switch c := cur.(type) {
		case value.Object:
			v, ok := c.Get(tok)
			if !ok {
				return nil, false
			}
			cur = v
		case value.Array:
			i, err := strconv.Atoi(tok)
			if err != nil || i < 0 || i >= c.Len() || strconv.Itoa(i) != tok {
				return nil, false
			}
			cur = c.At(i)
		default:
			return nil, false
		}
	}
	return cur, true
}

// escapeToken returns name as a JSON Pointer reference token.
func escapeToken(name string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(name)
}

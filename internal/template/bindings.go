// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"slices"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/dates"
	"github.com/kottera/metais/internal/digest"
	"github.com/kottera/metais/internal/inputs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/regexprofile"
	"github.com/kottera/metais/internal/schema"
)

// Package is what the template environment reads of an opened package:
// its tree, for the templates; its manifest's data model, for package; its
// digest; and its combined schema, from which the declared properties of
// every object are found (ADR-004 D6 item 2).
type Package struct {
	Tree          pkgtree.Tree
	Manifest      *manifest.Manifest
	ManifestValue value.Value
	Digest        digest.Sum
	Combined      value.Value
}

// canaryGlobals, when set, adds globals to every render. Only a canary
// build sets it (ADR-009 D16 item 2): the env canary binds the host's
// environment, which the SC7 harness must catch. The tool's own build
// never does.
var canaryGlobals func() []value.Member

// The manifest members package exposes, beside digest (ADR-004 D7 item 3).
var packageMembers = []string{"metais", "name", "version", "title", "description", "guide", "authors", "license", "homepage"}

// Bindings is the globals every template body and partial of one render
// sees but output (ADR-004 D7 items 2, 3, 7 and 9): inputs, package, now
// and sources, built from the validated inputs, the manifest and --now,
// and from nothing else (ADR-009 D13 item 2). Each is a plain value of
// the data model, its numbers as a template reads them (D4 item 2).
type Bindings struct {
	members []value.Member
}

// NewBindings builds the globals of a render of pkg with the records of
// its inputs, in manifest order, and now, --now's value as given, or ""
// when none was. now has been checked as an RFC 3339 date-time.
func NewBindings(pkg Package, records []inputs.Record, now string, lim limits.Set) (*Bindings, error) {
	m, err := newMaterializer(pkg.Combined, lim)
	if err != nil {
		return nil, err
	}
	in := make([]value.Member, 0, len(records))
	for _, rec := range records {
		v := rec.Value
		if v == nil {
			v = value.Null{}
		}
		in = append(in, value.Member{Name: rec.Name, Value: v})
	}
	obj, err := value.NewObject(in...)
	if err != nil {
		return nil, err
	}
	b := &Bindings{}
	b.add("inputs", liquid.Data(m.materialize(obj, []string{""})))
	b.add("package", packageValue(pkg))
	nowValue := value.Value(value.Null{})
	if now != "" {
		t, err := dates.ParseNow(now)
		if err != nil {
			return nil, err
		}
		nowValue = str(t.Normalized())
	}
	b.add("now", nowValue)
	b.add("sources", sourcesValue(pkg.Manifest.Inputs, records))
	if canaryGlobals != nil {
		b.members = append(b.members, canaryGlobals()...)
	}
	return b, nil
}

func (b *Bindings) add(name string, v value.Value) {
	b.members = append(b.members, value.Member{Name: name, Value: v})
}

// globals returns the globals of a template body rendered as the output
// at path: b's, and output (ADR-004 D7 item 6).
func (b *Bindings) globals(path string) value.Object {
	root := strings.Repeat("../", strings.Count(path, "/"))
	out := obj(value.Member{Name: "path", Value: str(path)}, value.Member{Name: "root", Value: str(root)})
	ms := append(slices.Clone(b.members), value.Member{Name: "output", Value: out})
	o, err := value.NewObject(ms...)
	if err != nil {
		panic("template: " + err.Error())
	}
	return o
}

// Each returns every global of b, by name, in canonical order, for a test
// that holds the set to the data model's plain values (ADR-009 D13 item 3).
func (b *Bindings) Each(path string) value.Object { return b.globals(path) }

// packageValue is package: the canonical manifest's members ADR-004 D7
// item 3 lists, an omitted one nil, and digest.
func packageValue(pkg Package) value.Value {
	mv, _ := pkg.ManifestValue.(value.Object)
	ms := []value.Member{{Name: "digest", Value: str(digest.Format(pkg.Digest))}}
	for _, name := range packageMembers {
		v, ok := mv.Get(name)
		if !ok {
			v = value.Null{}
		}
		ms = append(ms, value.Member{Name: name, Value: liquid.Data(v)})
	}
	return obj(ms...)
}

// sourcesValue is sources (ADR-004 D7 item 9): per input, the origin of
// one that is not a collection, or an array parallel to a collection's
// items, each its file's name, stem and form, or nil for an item supplied
// on the command line; nil for an absent input.
func sourcesValue(decls []manifest.Input, records []inputs.Record) value.Value {
	var ms []value.Member
	for i, rec := range records {
		var v value.Value = value.Null{}
		switch {
		case rec.Value == nil:
		case decls[i].Collection:
			elems := make([]value.Value, len(rec.Items))
			for i, it := range rec.Items {
				if !it.FromFile {
					elems[i] = value.Null{}
					continue
				}
				elems[i] = obj(
					value.Member{Name: "name", Value: str(it.Name)},
					value.Member{Name: "stem", Value: str(it.Stem)},
					value.Member{Name: "form", Value: str(string(it.Form))},
				)
			}
			a, err := value.NewArray(elems...)
			if err != nil {
				panic("template: " + err.Error())
			}
			v = a
		default:
			v = obj(value.Member{Name: "origin", Value: str(origin(rec.Source.Kind))})
		}
		ms = append(ms, value.Member{Name: rec.Name, Value: v})
	}
	return obj(ms...)
}

// origin is ADR-004 D7 item 9's name of a source kind.
func origin(k inputs.SourceKind) string {
	switch k {
	case inputs.SourceFile:
		return "file"
	case inputs.SourceCommandLine:
		return "command-line"
	case inputs.SourceDefault:
		return "default"
	}
	panic("template: an input with a value has no source")
}

func str(s string) value.Value {
	v, err := value.NewString(s)
	if err != nil {
		panic("template: " + err.Error())
	}
	return v
}

func obj(ms ...value.Member) value.Object {
	o, err := value.NewObject(ms...)
	if err != nil {
		panic("template: " + err.Error())
	}
	return o
}

// materializer makes every declared property absent from an object of the
// data present with the value nil (ADR-004 D6 items 2 and 3). An object's
// declared properties are the names in `properties` of the subschemas at
// its location in the combined schema and of every subschema reachable
// from them through `$ref`, `$dynamicRef`, `allOf`, `anyOf`, `oneOf`,
// `if`, `then`, `else` and `dependentSchemas`; a member's location is its
// `properties` entry, each `patternProperties` entry it matches, and
// otherwise `additionalProperties`; an element's is its `prefixItems`
// entry, or `items`. An absent object is not created.
type materializer struct {
	w        *schema.Walker
	lim      limits.Set
	sets     map[string][]string
	patterns map[string]*regexprofile.Regexp
}

func newMaterializer(combined value.Value, lim limits.Set) (*materializer, error) {
	w, err := schema.NewWalker(combined, schema.CombinedURI)
	if err != nil {
		return nil, err
	}
	return &materializer{w: w, lim: lim, sets: map[string][]string{}, patterns: map[string]*regexprofile.Regexp{}}, nil
}

// materialize returns v, the value at the schema locations at, with every
// declared property of every object in it present.
func (m *materializer) materialize(v value.Value, at []string) value.Value {
	set := m.applicable(at)
	switch v := v.(type) {
	case value.Object:
		var ms []value.Member
		for name, mv := range v.All() {
			ms = append(ms, value.Member{Name: name, Value: m.materialize(mv, m.member(set, name))})
		}
		for _, name := range m.declared(set) {
			if _, ok := v.Get(name); !ok {
				ms = append(ms, value.Member{Name: name, Value: value.Null{}})
			}
		}
		return obj(ms...)
	case value.Array:
		elems := make([]value.Value, v.Len())
		for i, e := range v.All() {
			elems[i] = m.materialize(e, m.element(set, i))
		}
		a, err := value.NewArray(elems...)
		if err != nil {
			panic("template: " + err.Error())
		}
		return a
	}
	return v
}

// applicable is the applicable set of the schemas at ptrs: each, and every
// subschema reachable from one through the applicators, each once, in the
// order first reached.
func (m *materializer) applicable(ptrs []string) []string {
	// The count first, so that no set's key is another's: the empty set's
	// is not the root's.
	key := strconv.Itoa(len(ptrs)) + "\x00" + strings.Join(ptrs, "\x00")
	if s, ok := m.sets[key]; ok {
		return s
	}
	var out []string
	seen := map[string]bool{}
	var visit func(string)
	visit = func(ptr string) {
		if seen[ptr] {
			return
		}
		seen[ptr] = true
		o, ok := m.object(ptr)
		if !ok {
			return
		}
		out = append(out, ptr)
		if t, ok, err := m.w.Reference(ptr, "$ref"); err == nil && ok {
			visit(t)
		}
		if t, ok, err := m.w.Reference(ptr, "$dynamicRef"); err == nil && ok {
			visit(t)
			if ref, ok := o.Get("$dynamicRef"); ok {
				if s, ok := ref.(value.String); ok {
					if _, name, found := strings.Cut(s.Text(), "#"); found {
						for _, d := range m.w.DynamicAnchors(name) {
							visit(d)
						}
					}
				}
			}
		}
		for _, kw := range []string{"allOf", "anyOf", "oneOf"} {
			if a, ok := get[value.Array](o, kw); ok {
				for i := range a.Len() {
					visit(ptr + "/" + kw + "/" + strconv.Itoa(i))
				}
			}
		}
		for _, kw := range []string{"if", "then", "else"} {
			if _, ok := o.Get(kw); ok {
				visit(ptr + "/" + kw)
			}
		}
		if deps, ok := get[value.Object](o, "dependentSchemas"); ok {
			for _, name := range deps.Names() {
				visit(ptr + "/dependentSchemas/" + escape(name))
			}
		}
	}
	for _, p := range ptrs {
		visit(p)
	}
	m.sets[key] = out
	return out
}

// declared is every name in `properties` of the schemas of set, each once,
// in the order first met.
func (m *materializer) declared(set []string) []string {
	var out []string
	for _, ptr := range set {
		o, _ := m.object(ptr)
		if props, ok := get[value.Object](o, "properties"); ok {
			for _, name := range props.Names() {
				if !slices.Contains(out, name) {
					out = append(out, name)
				}
			}
		}
	}
	return out
}

// member is the schema locations of the member name of an object at set.
func (m *materializer) member(set []string, name string) []string {
	var out []string
	for _, ptr := range set {
		o, _ := m.object(ptr)
		matched := false
		if props, ok := get[value.Object](o, "properties"); ok {
			if _, ok := props.Get(name); ok {
				out, matched = append(out, ptr+"/properties/"+escape(name)), true
			}
		}
		if pats, ok := get[value.Object](o, "patternProperties"); ok {
			for _, pat := range pats.Names() {
				if m.matches(pat, name) {
					out, matched = append(out, ptr+"/patternProperties/"+escape(pat)), true
				}
			}
		}
		if _, ok := o.Get("additionalProperties"); ok && !matched {
			out = append(out, ptr+"/additionalProperties")
		}
	}
	return out
}

// element is the schema locations of the element at index i of an array
// at set.
func (m *materializer) element(set []string, i int) []string {
	var out []string
	for _, ptr := range set {
		o, _ := m.object(ptr)
		n := 0
		if prefix, ok := get[value.Array](o, "prefixItems"); ok {
			n = prefix.Len()
			if i < n {
				out = append(out, ptr+"/prefixItems/"+strconv.Itoa(i))
				continue
			}
		}
		if _, ok := o.Get("items"); ok {
			out = append(out, ptr+"/items")
		}
	}
	return out
}

// matches reports whether name matches the patternProperties pattern pat,
// read by the package's regular-expression profile, which the schema
// compiler has already accepted it under.
func (m *materializer) matches(pat, name string) bool {
	re, ok := m.patterns[pat]
	if !ok {
		var err error
		if re, err = regexprofile.Compile(pat, m.lim, nil); err != nil {
			panic("template: a compiled schema's pattern does not compile: " + err.Error())
		}
		m.patterns[pat] = re
	}
	return re.MatchString(name)
}

// object returns the object schema at ptr, and false when the schema there
// is a boolean or there is none.
func (m *materializer) object(ptr string) (value.Object, bool) {
	v, ok := m.w.Lookup(ptr)
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

// escape escapes name as a JSON Pointer token.
func escape(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
}

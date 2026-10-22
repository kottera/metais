// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package schema is METAIS's JSON Schema 2020-12 layer (ADR-003 D3 to D7,
// ADR-010 D5): the closed compiler, the combined schema that bundles every
// input's schema into one document, and the static walk and coercion of
// ADR-003 D14 that read a schema without evaluating it.
package schema

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"metais.org/metais/internal/liquid/value"
)

// Dialect is the one `$schema` value METAIS accepts (ADR-003 D3 item 2).
const Dialect = "https://json-schema.org/draft/2020-12/schema"

// CombinedURI is the combined schema's own `$id`: the retrieval URI of
// `.compiled/schema.json`, where `pack` writes it (ADR-003 D7 item 1).
const CombinedURI = "metais:///.compiled/schema.json"

// InputsDir is the directory every input schema file lies under (ADR-003
// D1 item 4, D3 item 1).
const InputsDir = "inputs/"

// Input is one input slot as bundling reads it: the manifest's
// declaration (ADR-003 D1 item 2), already checked by the manifest's rules.
type Input struct {
	// Name is the input's name, the member it has in the inputs object.
	Name string
	// Title is the slot's title.
	Title string
	// Description is the slot's description, nil when the manifest gives
	// none.
	Description *string
	// Schema is the package path of the input's schema file.
	Schema string
	// Required is the slot's `required`, with the default of true already
	// applied (ADR-003 D1 item 7).
	Required bool
}

// Declarations is what the manifest declares of a package's inputs.
type Declarations struct {
	// Inputs is every input slot, in manifest order.
	Inputs []Input
	// Constraints is the package path of `input_constraints` (ADR-003 D6
	// item 2), or "" when the manifest declares none.
	Constraints string
}

// File is one schema file under `inputs/`, already parsed (ADR-003 D9 and
// D10): a YAML file is the same data-model value a JSON file would be, so
// it is embedded as JSON with nothing further to convert.
type File struct {
	// Path is the file's package path.
	Path string
	// Root is the file's value.
	Root value.Value
}

// ErrBundle is wrapped by every refusal of Bundle. Each one is a rule
// another layer reports as a finding before it bundles (the manifest's
// paths, the compiler's `$schema` and `$id` checks), so meeting one here
// means that layer let a package through.
var ErrBundle = errors.New("schema: cannot bundle")

// RetrievalURI returns the retrieval URI of the package file at path:
// `metais:///` followed by the path, its characters percent-encoded where
// RFC 3986 requires (ADR-003 D5 item 1).
func RetrievalURI(path string) string {
	return (&url.URL{Scheme: "metais", Path: "/" + path}).String()
}

// CanonicalURI returns the canonical URI of the schema file at path whose
// value is root: its root `$id` resolved against its retrieval URI, or its
// retrieval URI when it has none (ADR-003 D5 item 2). An `$id` that is not
// a string, is not a URI reference, or has a non-empty fragment is
// refused.
func CanonicalURI(path string, root value.Value) (string, error) {
	retrieval := RetrievalURI(path)
	obj, ok := root.(value.Object)
	if !ok {
		return retrieval, nil
	}
	idv, ok := obj.Get("$id")
	if !ok {
		return retrieval, nil
	}
	id, ok := idv.(value.String)
	if !ok {
		return "", fmt.Errorf("%w: %s: $id is a %s, not a string", ErrBundle, path, idv.Kind())
	}
	base, err := url.Parse(retrieval)
	if err != nil {
		// A retrieval URI is built by url.URL itself.
		return "", fmt.Errorf("%w: %s: retrieval URI %q: %v", ErrBundle, path, retrieval, err)
	}
	return resolveID(base, id.Text(), path)
}

// resolveID resolves the `$id` text id against base, refusing a non-empty
// fragment (core §8.2.1), and returns the result without its empty
// fragment.
func resolveID(base *url.URL, id, where string) (string, error) {
	ref, err := url.Parse(id)
	if err != nil {
		return "", fmt.Errorf("%w: %s: $id %q is not a URI reference: %v", ErrBundle, where, id, err)
	}
	if ref.Fragment != "" {
		return "", fmt.Errorf("%w: %s: $id %q has a non-empty fragment", ErrBundle, where, id)
	}
	u := base.ResolveReference(ref)
	u.Fragment, u.RawFragment = "", ""
	return u.String(), nil
}

// Bundle returns the combined schema of ADR-003 D7: one object schema with
// a property per input, every schema file under `inputs/` embedded in
// `$defs` as its own resource, and `allOf` naming `input_constraints` when
// the manifest declares it. files is every schema file under `inputs/`,
// whether or not an input names it (ADR-003 D3 item 1), in any order.
//
// No reference is rewritten or substituted (D7 items 3 and 4). Each
// resource's `$id` becomes its canonical URI, so a relative `$id` resolves
// to what it did in its own file, and its `$schema` is set to Dialect. A
// boolean schema file has no members to carry an `$id`, so it is embedded
// as the object schema that means the same: `true` as `{}`, `false` as
// `{"not": {}}`.
//
// Bundle refuses, wrapping ErrBundle: a file outside `inputs/` or given
// twice; a file whose root is not a schema, whose `$schema` is not
// Dialect, or whose `$id` is malformed; two files with one canonical URI
// (ADR-003 D5 item 6); an input or `input_constraints` naming a path no
// file has; and two inputs with one name.
func Bundle(decl Declarations, files []File) (value.Object, error) {
	canonical := make(map[string]string, len(files)) // path → canonical URI
	owner := make(map[string]string, len(files))     // canonical URI → path
	defs := make([]value.Member, 0, len(files))
	for _, f := range files {
		if !strings.HasPrefix(f.Path, InputsDir) {
			return value.Object{}, fmt.Errorf("%w: %s is not under %s", ErrBundle, f.Path, InputsDir)
		}
		if _, dup := canonical[f.Path]; dup {
			return value.Object{}, fmt.Errorf("%w: %s is given twice", ErrBundle, f.Path)
		}
		uri, err := CanonicalURI(f.Path, f.Root)
		if err != nil {
			return value.Object{}, err
		}
		if other, dup := owner[uri]; dup {
			return value.Object{}, fmt.Errorf("%w: %s and %s have one canonical URI, %s", ErrBundle, other, f.Path, uri)
		}
		canonical[f.Path], owner[uri] = uri, f.Path
		res, err := resource(f, uri)
		if err != nil {
			return value.Object{}, err
		}
		defs = append(defs, value.Member{Name: f.Path, Value: res})
	}

	props := make([]value.Member, 0, len(decl.Inputs))
	var required []value.Value
	for _, in := range decl.Inputs {
		uri, ok := canonical[in.Schema]
		if !ok {
			return value.Object{}, fmt.Errorf("%w: input %q names schema %s, which is not a schema file under %s", ErrBundle, in.Name, in.Schema, InputsDir)
		}
		prop, err := property(in, uri)
		if err != nil {
			return value.Object{}, err
		}
		props = append(props, value.Member{Name: in.Name, Value: prop})
		if in.Required {
			name, err := value.NewString(in.Name)
			if err != nil {
				return value.Object{}, fmt.Errorf("%w: input %q: %v", ErrBundle, in.Name, err)
			}
			required = append(required, name)
		}
	}
	properties, err := value.NewObject(props...)
	if err != nil {
		return value.Object{}, fmt.Errorf("%w: inputs: %v", ErrBundle, err)
	}
	requiredArr, err := value.NewArray(required...)
	if err != nil {
		return value.Object{}, fmt.Errorf("%w: required: %v", ErrBundle, err)
	}
	defsObj, err := value.NewObject(defs...)
	if err != nil {
		return value.Object{}, fmt.Errorf("%w: $defs: %v", ErrBundle, err)
	}

	members := []value.Member{
		{Name: "$schema", Value: str(Dialect)},
		{Name: "$id", Value: str(CombinedURI)},
		{Name: "type", Value: str("object")},
		{Name: "properties", Value: properties},
		{Name: "required", Value: requiredArr},
		{Name: "additionalProperties", Value: value.Bool(false)},
		{Name: "$defs", Value: defsObj},
	}
	if decl.Constraints != "" {
		uri, ok := canonical[decl.Constraints]
		if !ok {
			return value.Object{}, fmt.Errorf("%w: input_constraints names %s, which is not a schema file under %s", ErrBundle, decl.Constraints, InputsDir)
		}
		allOf, err := value.NewArray(mustObject(value.Member{Name: "$ref", Value: str(uri)}))
		if err != nil {
			return value.Object{}, fmt.Errorf("%w: allOf: %v", ErrBundle, err)
		}
		members = append(members, value.Member{Name: "allOf", Value: allOf})
	}
	return value.NewObject(members...)
}

// property returns the combined schema's member for in: the slot's title
// and description beside a `$ref` to its schema's canonical URI (ADR-003
// D7 item 2).
func property(in Input, uri string) (value.Object, error) {
	title, err := value.NewString(in.Title)
	if err != nil {
		return value.Object{}, fmt.Errorf("%w: input %q: title: %v", ErrBundle, in.Name, err)
	}
	ms := []value.Member{
		{Name: "title", Value: title},
		{Name: "$ref", Value: str(uri)},
	}
	if in.Description != nil {
		desc, err := value.NewString(*in.Description)
		if err != nil {
			return value.Object{}, fmt.Errorf("%w: input %q: description: %v", ErrBundle, in.Name, err)
		}
		ms = append(ms, value.Member{Name: "description", Value: desc})
	}
	o, err := value.NewObject(ms...)
	if err != nil {
		return value.Object{}, fmt.Errorf("%w: input %q: %v", ErrBundle, in.Name, err)
	}
	return o, nil
}

// resource returns f's root as an embedded resource whose `$id` is uri and
// whose `$schema` is Dialect (ADR-003 D7 item 3). Every other member is
// carried unchanged.
func resource(f File, uri string) (value.Object, error) {
	var root value.Object
	switch r := f.Root.(type) {
	case value.Object:
		root = r
	case value.Bool:
		if !r {
			root = mustObject(value.Member{Name: "not", Value: value.Object{}})
		}
	default:
		return value.Object{}, fmt.Errorf("%w: %s: a schema is an object or a boolean, not a %s", ErrBundle, f.Path, kindOf(f.Root))
	}
	if s, ok := root.Get("$schema"); ok {
		if t, isStr := s.(value.String); !isStr || t.Text() != Dialect {
			return value.Object{}, fmt.Errorf("%w: %s: $schema is not %s", ErrBundle, f.Path, Dialect)
		}
	}
	ms := []value.Member{
		{Name: "$id", Value: str(uri)},
		{Name: "$schema", Value: str(Dialect)},
	}
	for name, v := range root.All() {
		if name != "$id" && name != "$schema" {
			ms = append(ms, value.Member{Name: name, Value: v})
		}
	}
	o, err := value.NewObject(ms...)
	if err != nil {
		return value.Object{}, fmt.Errorf("%w: %s: %v", ErrBundle, f.Path, err)
	}
	return o, nil
}

// kindOf names v's kind, or says there is no value.
func kindOf(v value.Value) string {
	if v == nil {
		return "missing value"
	}
	return v.Kind().String()
}

// str returns s as a value, for a constant or a URI: a URI is ASCII, its
// other characters percent-encoded.
func str(s string) value.String {
	v, err := value.NewString(s)
	if err != nil {
		panic(fmt.Sprintf("schema: %q: %v", s, err))
	}
	return v
}

// mustObject returns the object of ms, whose names are distinct
// constants.
func mustObject(ms ...value.Member) value.Object {
	o, err := value.NewObject(ms...)
	if err != nil {
		panic(fmt.Sprintf("schema: %v", err))
	}
	return o
}

// InputDocument returns the schema of the input called name as one
// self-contained Compound Schema Document (ADR-008 D5, schema item 2),
// cut from combined, the combined schema Bundle returned: the slot's
// title and description beside the `$ref` to its schema's canonical URI,
// as the combined schema's property carries them, with `$defs` embedding
// every resource of combined's `$defs` that the `$ref` reaches, in their
// order there. Nothing is rewritten, so every reference resolves as it
// does in the combined schema.
//
// A resource is reached by a `$ref` or `$dynamicRef` whose URI, resolved
// against the base the enclosing `$id`s give it and without its
// fragment, is the canonical URI of a resource or an `$id` within one.
// A string that only looks like a reference, as an `enum` value may,
// can embed a resource no reference needs, which leaves the document
// self-contained.
func InputDocument(combined value.Object, name string) value.Object {
	props, _ := combined.Get("properties")
	prop, _ := props.(value.Object).Get(name)
	defsV, _ := combined.Get("$defs")
	defs := defsV.(value.Object)

	owner := map[string]int{} // an $id's URI → the index of its resource in defs
	refs := make([][]string, defs.Len())
	for i := range defs.Len() {
		m := defs.At(i)
		walkRefs(m.Value, nil, func(id string) { owner[id] = i }, func(ref string) { refs[i] = append(refs[i], ref) })
	}
	reached := make([]bool, defs.Len())
	var reach func(ref string)
	reach = func(ref string) {
		i, ok := owner[ref]
		if !ok || reached[i] {
			return
		}
		reached[i] = true
		for _, r := range refs[i] {
			reach(r)
		}
	}
	walkRefs(prop, nil, func(string) {}, reach)

	var embedded []value.Member
	for i := range defs.Len() {
		if reached[i] {
			embedded = append(embedded, defs.At(i))
		}
	}
	members := []value.Member{{Name: "$schema", Value: str(Dialect)}}
	for n, v := range prop.(value.Object).All() {
		members = append(members, value.Member{Name: n, Value: v})
	}
	members = append(members, value.Member{Name: "$defs", Value: mustObject(embedded...)})
	return mustObject(members...)
}

// walkRefs walks v, a schema or a value within one, whose base URI is
// base, calling id with each `$id` it declares and ref with each `$ref`
// and `$dynamicRef`, both resolved and without their fragments.
func walkRefs(v value.Value, base *url.URL, id, ref func(string)) {
	switch v := v.(type) {
	case value.Object:
		if s, ok := v.Get("$id"); ok {
			if t, ok := s.(value.String); ok {
				if u, err := resolveURL(base, t.Text()); err == nil {
					base = u
					id(u.String())
				}
			}
		}
		for n, m := range v.All() {
			if t, ok := m.(value.String); ok && (n == "$ref" || n == "$dynamicRef") {
				if u, err := resolveURL(base, t.Text()); err == nil {
					ref(u.String())
				}
				continue
			}
			walkRefs(m, base, id, ref)
		}
	case value.Array:
		for _, e := range v.All() {
			walkRefs(e, base, id, ref)
		}
	}
}

// resolveURL resolves the URI reference s against base, when there is
// one, and drops its fragment.
func resolveURL(base *url.URL, s string) (*url.URL, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	if base != nil {
		u = base.ResolveReference(u)
	}
	u.Fragment, u.RawFragment = "", ""
	return u, nil
}

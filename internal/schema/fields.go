// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"metais.org/metais/internal/jsonx"
	"metais.org/metais/internal/liquid/value"
)

// Field is one line of the compact field listing (ADR-008 A7): one
// declaration of a field, at an instance location, by the schemas that
// declare it there under one set of conditions.
type Field struct {
	// Pointer is the field's JSON Pointer, its first token the input's
	// name, as --set takes it; a token "*" stands for every element of
	// an array, or every member of an object not named otherwise.
	Pointer string
	// Type is the field's declared type names, in `type`'s order; empty
	// when no schema declaring it names a type.
	Type []string
	// Required is whether the object holding the field requires it.
	Required bool
	// Constraints is each assertion and `default` the declaring schemas
	// make of the field's value, as keyword=<canonical JSON>.
	Constraints []string
	// Title is the first `title` of the declaring schemas, or nil.
	Title *string
	// When is the conditions the declaration applies under, outermost
	// first; empty for a declaration that always applies.
	When []string
	// SameAs is the pointer where the fields beneath a schema that also
	// declares this one were listed, or "" when none was listed before.
	SameAs string
	// BeyondValueDepth is true when the declaring schemas declare fields
	// beneath this one that are not listed, because they lie deeper than
	// value-depth beneath the input, where no input is accepted.
	BeyondValueDepth bool
}

// The listing's cuts (ADR-008 A7 item 5, A16): a title, a condition and
// one constraint are each cut at MaxShown bytes, as grep cuts a line, and
// a line's constraints together at MaxConstraints. The bound is in bytes,
// not in values: an enum is cut where its canonical JSON passes MaxShown.
// Each mark names the command that prints the value whole.
const (
	MaxShown       = 1024
	MaxConstraints = 4096
)

// Fields returns the compact field listing of the schema at the root of
// the walked document (ADR-008 A7). With an input's name, the root is
// that input's document (InputDocument) and is listed as the field
// /<input>, required as the manifest declares; with "", the root is the
// combined schema, and its members, the inputs, are listed beneath it.
// depth is value-depth: no field more than depth tokens beneath an
// input's name is listed.
//
// Each schema is expanded once: the fields beneath it are listed where
// it is first met, and every later declaration it makes names that
// pointer as SameAs. So a recursive schema ends, and a schema reached
// along many paths is listed once; the listing has at most one line per
// subschema of the document and per name a `required` lists.
func (w *Walker) Fields(input string, required bool, depth int) []Field {
	l := &lister{w: w, depth: depth, sums: map[string]*summary{}, expanded: map[string]string{}, whole: WholeCommand(input)}
	if input == "" {
		_, _, kids, branches := l.gather("", -1, []string{""})
		l.emit("", -1, nil, kids, branches)
	} else {
		l.field("/"+escapeToken(input), 0, nil, []string{""}, required, nil, true)
	}
	return l.out
}

type lister struct {
	w        *Walker
	depth    int
	sums     map[string]*summary // a schema's pointer → what it declares at its own location
	expanded map[string]string   // a schema's pointer → the instance pointer its fields are listed beneath
	whole    string              // the command that prints the listed schema whole, named by each cut's mark
	out      []Field
}

// summary is what a schema, and the schemas it reaches without a
// condition ($ref, $dynamicRef, allOf), declare of the value at their
// own instance location.
type summary struct {
	types       []string
	typed       bool
	title       *string
	constraints []string
	size        int
	cut         bool
	// beneath is true when the schema, or one it reaches without a
	// condition, declares a field beneath its location or a conditional
	// branch.
	beneath bool
	busy    bool
}

// The keywords a summary does not list as constraints: identifiers,
// annotations other than `default`, `type` and `title`, which have their
// own columns, and the applicators the listing walks.
var unlisted = map[string]bool{
	"$schema": true, "$id": true, "$anchor": true, "$dynamicAnchor": true, "$ref": true,
	"$dynamicRef": true, "$defs": true, "$comment": true, "title": true, "description": true,
	"examples": true, "deprecated": true, "readOnly": true, "writeOnly": true, "type": true,
	"properties": true, "patternProperties": true, "required": true, "prefixItems": true,
	"allOf": true, "anyOf": true, "oneOf": true, "if": true, "then": true, "else": true,
	"dependentSchemas": true, "contentEncoding": true, "contentMediaType": true, "contentSchema": true,
}

// The keywords whose schema value is walked as the fields beneath: when
// the value is `false` it is listed as a constraint instead.
var walkedSchemas = map[string]bool{
	"additionalProperties": true, "unevaluatedProperties": true, "items": true, "unevaluatedItems": true,
}

func (l *lister) summary(ptr string) *summary {
	if s, ok := l.sums[ptr]; ok {
		if s.busy {
			return &summary{}
		}
		return s
	}
	s := &summary{busy: true}
	l.sums[ptr] = s
	v, _ := l.w.Lookup(ptr)
	switch v := v.(type) {
	case value.Bool:
		if !v {
			s.add("false")
		}
	case value.Object:
		for kw, m := range v.All() {
			switch {
			case kw == "type":
				s.types, s.typed = declaredNames(m), true
			case kw == "title":
				if t, ok := m.(value.String); ok {
					title := l.shown(t.Text())
					s.title = &title
				}
			case walkedSchemas[kw]:
				if b, ok := m.(value.Bool); ok {
					if !b {
						s.add(kw + "=false")
					}
					continue
				}
				s.beneath = true
			case kw == "properties" || kw == "patternProperties" || kw == "prefixItems" ||
				kw == "anyOf" || kw == "oneOf" || kw == "dependentSchemas" || kw == "required":
				s.beneath = true
			case kw == "then" || kw == "else":
				if _, ok := v.Get("if"); ok {
					s.beneath = true
				}
			case unlisted[kw] || strings.HasPrefix(kw, "x-"):
			default:
				s.add(kw + "=" + l.shown(string(jsonx.Canonical(m))))
			}
		}
		for _, kw := range []string{"$ref", "$dynamicRef"} {
			if t, ok := l.target(v, ptr, kw); ok {
				s.merge(l.summary(t))
			} else if r, has := v.Get(kw); has {
				s.add(kw + "=" + l.shown(string(jsonx.Canonical(r))))
			}
		}
		if all, ok := v.Get("allOf"); ok {
			if arr, ok := all.(value.Array); ok {
				for i := range arr.Len() {
					s.merge(l.summary(ptr + "/allOf/" + strconv.Itoa(i)))
				}
			}
		}
	}
	s.busy = false
	return s
}

// target is the pointer of the schema the reference keyword kw of o, at
// ptr, names; false when o has none, or names a schema the document does
// not hold, such as a 2020-12 meta-schema.
func (l *lister) target(o value.Object, ptr, kw string) (string, bool) {
	n, ok, err := l.w.reference(o, ptr, kw)
	if err != nil || !ok {
		return "", false
	}
	return n.ptr, true
}

func declaredNames(v value.Value) []string {
	switch t := v.(type) {
	case value.String:
		return []string{t.Text()}
	case value.Array:
		var out []string
		for _, e := range t.All() {
			if s, ok := e.(value.String); ok {
				out = append(out, s.Text())
			}
		}
		return out
	}
	return nil
}

// add appends one constraint, once, within MaxConstraints.
func (s *summary) add(c string) {
	if slices.Contains(s.constraints, c) || s.cut {
		return
	}
	if s.size+len(c) > MaxConstraints {
		s.cut = true
		return
	}
	s.constraints = append(s.constraints, c)
	s.size += len(c) + 1
}

// merge adds what t declares to s: its type when s has none, and
// otherwise a `type` of its own that differs as a constraint; its title
// when s has none; and its constraints.
func (s *summary) merge(t *summary) {
	if t.typed {
		if !s.typed {
			s.types, s.typed = t.types, true
		} else if !slices.Equal(s.types, t.types) {
			s.add("type=" + typeJSON(t.types))
		}
	}
	if s.title == nil {
		s.title = t.title
	}
	for _, c := range t.constraints {
		s.add(c)
	}
	if t.cut {
		s.cut = true
	}
	s.beneath = s.beneath || t.beneath
}

func typeJSON(types []string) string {
	q := make([]string, len(types))
	for i, t := range types {
		q[i] = strconv.Quote(t)
	}
	if len(q) == 1 {
		return q[0]
	}
	return "[" + strings.Join(q, ",") + "]"
}

// WholeCommand is the command that prints whole the schema whose fields
// Fields lists for input: the input's schema, or with "" the combined
// schema.
func WholeCommand(input string) string {
	if input == "" {
		return "metais schema <package>"
	}
	return "metais schema <package> " + input
}

// cutMark marks a value's cut: omitted bytes follow what is shown, and
// whole prints them.
func cutMark(omitted int, whole string) string {
	return "[…+" + strconv.Itoa(omitted) + " bytes; " + whole + " prints it whole]"
}

// constraintsMark ends a line's constraints cut at MaxConstraints.
func constraintsMark(whole string) string {
	return "[…; " + whole + " prints every constraint]"
}

// shown is s cut at MaxShown bytes, on a character boundary, and marked.
func (l *lister) shown(s string) string {
	if len(s) <= MaxShown {
		return s
	}
	n := MaxShown
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n] + cutMark(len(s)-n, l.whole)
}

// child is a field beneath the one being expanded, gathered from every
// schema that declares it.
type child struct {
	token      string
	roots      []string
	required   bool
	constraint string
}

// branch is a conditional subschema at the location being expanded.
type branch struct {
	when string
	root string
}

// field lists the field at ptr, depth tokens beneath its input, as the
// schemas at roots declare it under when, and the fields beneath it.
// show is false for a conditional branch, whose line is listed only when
// the branch itself declares something of the field.
func (l *lister) field(ptr string, depth int, when []string, roots []string, required bool, extra []string, show bool) {
	s := &summary{}
	for _, c := range extra {
		s.add(c)
	}
	for _, r := range roots {
		s.merge(l.summary(r))
	}
	f := Field{Pointer: ptr, Type: s.types, Required: required, Title: s.title, When: when}
	f.Constraints = slices.Clone(s.constraints)
	if s.cut {
		f.Constraints = append(f.Constraints, constraintsMark(l.whole))
	}
	at := len(l.out)
	l.out = append(l.out, f)
	same, beyond, kids, branches := l.gather(ptr, depth, roots)
	g := &l.out[at]
	g.SameAs, g.BeyondValueDepth = same, beyond
	if !show && !s.typed && s.title == nil && len(g.Constraints) == 0 && !required && same == "" && !beyond {
		l.out = l.out[:at]
	}
	l.emit(ptr, depth, when, kids, branches)
}

func (l *lister) emit(ptr string, depth int, when []string, kids []*child, branches []branch) {
	for _, k := range kids {
		var extra []string
		if k.constraint != "" {
			extra = []string{k.constraint}
		}
		l.field(ptr+"/"+k.token, depth+1, when, k.roots, k.required, extra, len(when) == 0)
	}
	for _, b := range branches {
		l.field(ptr, depth, append(slices.Clone(when), b.when), []string{b.root}, false, nil, false)
	}
}

// gather walks the schemas at roots, and those they reach without a
// condition, for the fields beneath ptr and the conditional branches at
// it. A schema already expanded is not walked again: same is the pointer
// it was expanded at. beyond is true when fields beneath are not listed
// because ptr is value-depth tokens beneath its input.
func (l *lister) gather(ptr string, depth int, roots []string) (same string, beyond bool, kids []*child, branches []branch) {
	byKey := map[string]*child{}
	kid := func(key, token string) *child {
		if c, ok := byKey[key]; ok {
			return c
		}
		c := &child{token: token}
		byKey[key] = c
		kids = append(kids, c)
		return c
	}
	var visit func(sp string)
	visit = func(sp string) {
		if at, ok := l.expanded[sp]; ok {
			if at != ptr && same == "" {
				same = at
			}
			return
		}
		if !l.summary(sp).beneath {
			return
		}
		if depth >= l.depth {
			beyond = true
			return
		}
		l.expanded[sp] = ptr
		v, _ := l.w.Lookup(sp)
		o, ok := v.(value.Object)
		if !ok {
			return
		}
		if m, ok := o.Get("properties"); ok {
			if props, ok := m.(value.Object); ok {
				for name := range props.All() {
					c := kid("p:"+name, escapeToken(name))
					c.roots = append(c.roots, sp+"/properties/"+escapeToken(name))
				}
			}
		}
		if m, ok := o.Get("required"); ok {
			if names, ok := m.(value.Array); ok {
				for _, n := range names.All() {
					if name, ok := n.(value.String); ok {
						kid("p:"+name.Text(), escapeToken(name.Text())).required = true
					}
				}
			}
		}
		if m, ok := o.Get("patternProperties"); ok {
			if pats, ok := m.(value.Object); ok {
				for re := range pats.All() {
					c := kid("r:"+re, "*")
					c.roots = append(c.roots, sp+"/patternProperties/"+escapeToken(re))
					c.constraint = "name-matches=" + l.shown(strconv.Quote(re))
				}
			}
		}
		if m, ok := o.Get("prefixItems"); ok {
			if arr, ok := m.(value.Array); ok {
				for i := range arr.Len() {
					c := kid("i:"+strconv.Itoa(i), strconv.Itoa(i))
					c.roots = append(c.roots, sp+"/prefixItems/"+strconv.Itoa(i))
				}
			}
		}
		for _, kw := range []string{"additionalProperties", "unevaluatedProperties", "items", "unevaluatedItems"} {
			if m, ok := o.Get(kw); ok {
				if _, isObj := m.(value.Object); isObj {
					c := kid(kw, "*")
					c.roots = append(c.roots, sp+"/"+kw)
				}
			}
		}
		for _, kw := range []string{"$ref", "$dynamicRef"} {
			if t, ok := l.target(o, sp, kw); ok {
				visit(t)
			}
		}
		if m, ok := o.Get("allOf"); ok {
			if arr, ok := m.(value.Array); ok {
				for i := range arr.Len() {
					visit(sp + "/allOf/" + strconv.Itoa(i))
				}
			}
		}
		at := ptr
		if at == "" {
			at = `""`
		}
		for _, kw := range []string{"anyOf", "oneOf"} {
			if m, ok := o.Get(kw); ok {
				if arr, ok := m.(value.Array); ok {
					for i := range arr.Len() {
						branches = append(branches, branch{at + " matches " + kw + "/" + strconv.Itoa(i), sp + "/" + kw + "/" + strconv.Itoa(i)})
					}
				}
			}
		}
		if cond, ok := o.Get("if"); ok {
			c := l.shown(string(jsonx.Canonical(cond)))
			if _, ok := o.Get("then"); ok {
				branches = append(branches, branch{at + " matches " + c, sp + "/then"})
			}
			if _, ok := o.Get("else"); ok {
				branches = append(branches, branch{at + " does not match " + c, sp + "/else"})
			}
		}
		if m, ok := o.Get("dependentSchemas"); ok {
			if deps, ok := m.(value.Object); ok {
				for name := range deps.All() {
					branches = append(branches, branch{ptr + "/" + escapeToken(name) + " is present", sp + "/dependentSchemas/" + escapeToken(name)})
				}
			}
		}
	}
	for _, r := range roots {
		visit(r)
	}
	return same, beyond, kids, branches
}

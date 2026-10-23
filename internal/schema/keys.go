// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"fmt"
	"slices"
	"strconv"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/regexprofile"
)

// Node is a schema node: the place of a schema in a schema file.
type Node struct{ Entry, Ptr string }

// KeyResolver resolves the key pointers of the rules vocabulary's
// uniqueKeys and coverage against an array's item schemas, token by token
// as MTS-AUT-013 resolves a property segment. check reports what it
// reaches (MTS-AUT-081 to MTS-AUT-083), and the sample generator refuses
// a coverage member the schemas it reaches refuse (MTS-SMP-010): both by
// this one resolution.
type KeyResolver struct {
	c        *Compiler
	lim      limits.Set
	patterns map[string]*regexprofile.Regexp
}

// NewKeyResolver returns the resolver over c, whose patterns were compiled
// under lim.
func NewKeyResolver(c *Compiler, lim limits.Set) *KeyResolver {
	return &KeyResolver{c: c, lim: lim, patterns: map[string]*regexprofile.Regexp{}}
}

// Applicable returns the applicable set of the node n (ADR-007 D3 item
// 4.1): n, and every node $ref, $dynamicRef, allOf, anyOf, oneOf, if,
// then, else and dependentSchemas reach from it. It reports whether the
// set is open (item 4.2): some node of it is open, or names nothing the
// package holds.
func (r *KeyResolver) Applicable(n Node) ([]Node, bool) {
	seen := map[Node]bool{}
	var set []Node
	open := false
	var visit func(n Node)
	visit = func(n Node) {
		if seen[n] || open {
			return
		}
		seen[n] = true
		v, ok := r.c.Lookup(n.Entry, n.Ptr)
		if !ok || IsOpen(v) {
			open = true
			return
		}
		set = append(set, n)
		o, ok := v.(value.Object)
		if !ok {
			return
		}
		if e, p, ok := r.c.Ref(n.Entry, n.Ptr); ok {
			visit(Node{e, p})
		}
		if e, p, ok := r.c.DynamicRef(n.Entry, n.Ptr); ok {
			visit(Node{e, p})
		}
		for _, kw := range []string{"if", "then", "else"} {
			if _, ok := o.Get(kw); ok {
				visit(Node{n.Entry, n.Ptr + "/" + kw})
			}
		}
		for _, kw := range []string{"allOf", "anyOf", "oneOf"} {
			if a, ok := o.Get(kw); ok {
				if aa, ok := a.(value.Array); ok {
					for i := range aa.Len() {
						visit(Node{n.Entry, n.Ptr + "/" + kw + "/" + strconv.Itoa(i)})
					}
				}
			}
		}
		if d, ok := o.Get("dependentSchemas"); ok {
			if do, ok := d.(value.Object); ok {
				for dn := range do.All() {
					visit(Node{n.Entry, n.Ptr + "/dependentSchemas/" + escapeToken(dn)})
				}
			}
		}
	}
	visit(n)
	return set, open
}

// Property resolves the property segment name on the node n as
// MTS-AUT-013 does, and reports whether it is declared: the applicable set
// of n is open, or some node of it has name in properties, a pattern of
// patternProperties that matches it, or an additionalProperties other
// than false — or, with no additionalProperties, an
// unevaluatedProperties other than false. It returns the subschemas the
// segment reaches, none when the set is open.
func (r *KeyResolver) Property(n Node, name string) ([]Node, bool) {
	set, open := r.Applicable(n)
	if open {
		return nil, true
	}
	var out []Node
	for _, m := range set {
		v, _ := r.c.Lookup(m.Entry, m.Ptr)
		o, ok := v.(value.Object)
		if !ok {
			continue
		}
		matched := false
		if p, ok := o.Get("properties"); ok {
			if po, ok := p.(value.Object); ok {
				if _, ok := po.Get(name); ok {
					out = append(out, Node{m.Entry, m.Ptr + "/properties/" + escapeToken(name)})
					matched = true
				}
			}
		}
		if pp, ok := o.Get("patternProperties"); ok {
			if po, ok := pp.(value.Object); ok {
				for pat := range po.All() {
					if r.matches(pat, name) {
						out = append(out, Node{m.Entry, m.Ptr + "/patternProperties/" + escapeToken(pat)})
						matched = true
					}
				}
			}
		}
		if matched {
			continue
		}
		for _, kw := range []string{"additionalProperties", "unevaluatedProperties"} {
			if ap, ok := o.Get(kw); ok {
				if ap != value.Bool(false) {
					out = append(out, Node{m.Entry, m.Ptr + "/" + kw})
				}
				break
			}
		}
	}
	return out, len(out) > 0
}

// ItemSchemas returns the item schemas of the array whose schema is the
// node n: the items and each prefixItems member of every node of its
// applicable set. It reports whether an item may be any value, the set
// being open or naming no item schema.
func (r *KeyResolver) ItemSchemas(n Node) ([]Node, bool) {
	set, open := r.Applicable(n)
	if open {
		return nil, true
	}
	var out []Node
	for _, m := range set {
		v, _ := r.c.Lookup(m.Entry, m.Ptr)
		o, ok := v.(value.Object)
		if !ok {
			continue
		}
		if pi, ok := o.Get("prefixItems"); ok {
			if a, ok := pi.(value.Array); ok {
				for i := range a.Len() {
					out = append(out, Node{m.Entry, m.Ptr + "/prefixItems/" + strconv.Itoa(i)})
				}
			}
		}
		if _, ok := o.Get("items"); ok {
			out = append(out, Node{m.Entry, m.Ptr + "/items"})
		}
	}
	return out, len(out) == 0
}

// Key resolves the key pointer p against the item schemas items, each
// token as a property segment, and returns the schemas it reaches, none
// when one of them is open. It reports whether every token is declared by
// some schema the tokens before it reach.
func (r *KeyResolver) Key(items []Node, p string) ([]Node, bool) {
	path, ok := finding.ParsePath(p)
	if !ok {
		return nil, true // the pre-check refused it, as schema.invalid
	}
	cur := items
	for _, tok := range path.Tokens() {
		var next []Node
		declared := false
		for _, n := range cur {
			reached, d := r.Property(n, tok)
			if d && len(reached) == 0 {
				return nil, true
			}
			if d {
				declared = true
				for _, m := range reached {
					if !slices.Contains(next, m) {
						next = append(next, m)
					}
				}
			}
		}
		if !declared {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

// CoverageKey returns the schemas the key of the coverage of the array
// schema n reaches, the key's pointer text being key: none when an item,
// or the key's value, may be any value, or a token of the key is not
// declared. A member of the coverage's values that every schema returned
// refuses is one no item can hold (MTS-AUT-081, MTS-SMP-010).
func (r *KeyResolver) CoverageKey(n Node, key string) []Node {
	items, open := r.ItemSchemas(n)
	if open {
		return nil
	}
	reached, _ := r.Key(items, key)
	return reached
}

// matches reports whether the patternProperties pattern pat matches name,
// by the package regular-expression profile the compiler compiled it with.
func (r *KeyResolver) matches(pat, name string) bool {
	re, ok := r.patterns[pat]
	if !ok {
		var err error
		re, err = regexprofile.Compile(pat, r.lim, nil)
		if err != nil {
			// The compiler compiled the pattern under the same limits.
			panic(fmt.Sprintf("schema: compiling the pattern %q, which compiled: %v", pat, err))
		}
		r.patterns[pat] = re
	}
	return re.MatchString(name)
}

// annotations are the keywords an open schema holds (ADR-007 D3 item 4.2).
var annotations = map[string]bool{
	"title": true, "description": true, "$comment": true, "examples": true, "default": true, "deprecated": true,
	"readOnly": true, "writeOnly": true, "$id": true, "$anchor": true, "$schema": true, "$defs": true,
}

// IsOpen reports whether a schema is open: the boolean schema true, or an
// object schema holding annotations only (ADR-007 D3 item 4.2).
func IsOpen(v value.Value) bool {
	switch s := v.(type) {
	case value.Bool:
		return bool(s)
	case value.Object:
		for name := range s.All() {
			if !annotations[name] {
				return false
			}
		}
		return true
	}
	return false
}

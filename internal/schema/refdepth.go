// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"math"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
)

// unbounded is the length of a chain that reaches a cycle.
const unbounded = math.MaxInt64

// inPlace are the applicators whose subschemas apply at the instance
// location of the schema object holding them: a chain of references
// carries on through them. Every other applicator moves to a member, an
// item or a name, and starts a chain anew.
var inPlace = []string{"allOf", "anyOf", "oneOf", "not", "if", "then", "else", "dependentSchemas"}

// chains is ref-depth over one compilation's schema files (ADR-009 D3): for
// each schema object, the most $ref and $dynamicRef references followed
// along a chain of subschemas each applying at the instance location of
// the one before. Its graph is the files as authored, each reference
// resolved as the pre-check resolved it.
type chains struct {
	c       *Compiler
	hops    map[root]int64
	state   map[root]uint8
	dynamic map[string][]root // each $dynamicAnchor's schema object, by name
}

const (
	visiting = 1
	visited  = 2
)

// checkRefDepth refuses a compilation in which a reference begins a chain
// of more than ref-depth references, or one that never ends because it
// meets a cycle that consumes no instance (ADR-009 D3, D17 item 6). It is
// reported once, with limit.package at the first such reference in the
// order the pre-check walks the files, observing the chain's length when it
// has one. The library would otherwise follow a chain without bound, and
// a cycle reaches its own error at validation, which is no finding.
func (c *Compiler) checkRefDepth(files []*file) []finding.Finding {
	a := &chains{c: c, hops: map[root]int64{}, state: map[root]uint8{}, dynamic: map[string][]root{}}
	for _, f := range files {
		for _, d := range f.walk.dynamic {
			a.dynamic[d.name] = append(a.dynamic[d.name], root{f.entry, d.ptr})
		}
	}
	bound := c.lim.Value(limits.RefDepth)
	for _, f := range files {
		for _, r := range f.walk.refs {
			n := a.from(f.entry, r.ptr)
			if n <= bound {
				continue
			}
			loc := *at(f.entry, r.ptr)
			return []finding.Finding{c.lim.Breached(limits.RefDepth, loc, n, n != unbounded)}
		}
	}
	return nil
}

// from returns the length of the longest chain the reference keyword at
// ptr in entry begins, counting itself.
func (a *chains) from(entry, ptr string) int64 {
	holder, kw := ptr[:strings.LastIndexByte(ptr, '/')], ptr[strings.LastIndexByte(ptr, '/')+1:]
	best := int64(0)
	for _, t := range a.targets(root{entry, holder}, kw) {
		best = max(best, plus(a.chain(t), 1))
	}
	return best
}

// chain returns the most references along a chain from the schema at n:
// unbounded when one reaches a cycle, which, since a file is a tree, holds
// a reference.
func (a *chains) chain(n root) int64 {
	switch a.state[n] {
	case visited:
		return a.hops[n]
	case visiting:
		return unbounded
	}
	o, ok := a.c.object(n.entry, n.ptr)
	if !ok {
		// A boolean schema, or a target the library will refuse, applies
		// nothing further.
		return 0
	}
	a.state[n] = visiting
	best := int64(0)
	for _, kw := range inPlace {
		v, ok := o.Get(kw)
		if !ok {
			continue
		}
		at := n.ptr + "/" + kw
		switch v := v.(type) {
		case value.Array:
			for i := range v.Len() {
				best = max(best, a.chain(root{n.entry, at + "/" + strconv.Itoa(i)}))
			}
		case value.Object:
			if kw == "dependentSchemas" {
				for name := range v.All() {
					best = max(best, a.chain(root{n.entry, at + "/" + escapeToken(name)}))
				}
			} else {
				best = max(best, a.chain(root{n.entry, at}))
			}
		}
	}
	for _, kw := range []string{"$dynamicRef", "$ref"} {
		for _, t := range a.targets(n, kw) {
			best = max(best, plus(a.chain(t), 1))
		}
	}
	a.state[n], a.hops[n] = visited, best
	return best
}

// targets returns the schema objects the reference keyword kw of the
// schema at n may resolve to: none when it has none, or when it names a
// meta-schema, which ends a chain. A $dynamicRef whose target declares the
// $dynamicAnchor it names may resolve, by the dynamic scope, to any schema
// object declaring that anchor, so each is a target.
func (a *chains) targets(n root, kw string) []root {
	refs := a.c.refs
	if kw == "$dynamicRef" {
		refs = a.c.dynRefs
	}
	resolved, ok := refs[n]
	if !ok {
		return nil
	}
	t, ok := a.c.resolve(resolved)
	if !ok {
		// The zero root is no schema object: the reference is followed,
		// and the chain ends there.
		return []root{{}}
	}
	out := []root{t}
	if _, frag, _ := strings.Cut(resolved, "#"); kw == "$dynamicRef" && frag != "" {
		if o, ok := a.c.object(t.entry, t.ptr); ok {
			if d, ok := o.Get("$dynamicAnchor"); ok {
				if d, ok := d.(value.String); ok && d.Text() == frag {
					out = append(out, a.dynamic[frag]...)
				}
			}
		}
	}
	return out
}

// resolve returns the schema file and pointer a resolved reference names,
// and false for a meta-schema or a target the library will refuse.
func (c *Compiler) resolve(uri string) (root, bool) {
	base, frag, _ := strings.Cut(uri, "#")
	r, ok := c.roots[base]
	if !ok {
		return root{}, false
	}
	switch {
	case frag == "":
		return r, true
	case strings.HasPrefix(frag, "/"):
		return root{r.entry, r.ptr + unescapePointer(frag)}, true
	}
	t, ok := c.anchors[base+"#"+frag]
	return t, ok
}

// plus adds n to a chain's length, which stays unbounded.
func plus(h, n int64) int64 {
	if h == unbounded {
		return h
	}
	return h + n
}

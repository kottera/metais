// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/schemafail"
)

// The compiler is the schemafail.Schemas of its package: it holds every
// schema file as authored, and answers what ADR-006 D6's mapping reads of
// one (field lookup, D6 item 9).
var _ schemafail.Schemas = (*Compiler)(nil)

// lookup returns the value at the RFC 6901 pointer ptr in the schema file
// entry, or nil when there is none.
func (c *Compiler) lookup(entry, ptr string) value.Value {
	v := c.docs[entry]
	for _, tok := range pointerTokens(ptr) {
		switch x := v.(type) {
		case value.Object:
			v, _ = x.Get(tok)
		case value.Array:
			i, err := strconv.Atoi(tok)
			if err != nil || i < 0 || i >= x.Len() || tok != strconv.Itoa(i) {
				return nil
			}
			v = x.At(i)
		default:
			return nil
		}
	}
	return v
}

func (c *Compiler) object(entry, ptr string) (value.Object, bool) {
	o, ok := c.lookup(entry, ptr).(value.Object)
	return o, ok
}

// Annotations returns the title, description and writeOnly of the schema
// object at pointer in entry.
func (c *Compiler) Annotations(entry, pointer string) schemafail.Annotations {
	o, ok := c.object(entry, pointer)
	if !ok {
		return schemafail.Annotations{}
	}
	var a schemafail.Annotations
	if s, ok := o.Get("title"); ok {
		if s, ok := s.(value.String); ok {
			a.Title = s.Text()
		}
	}
	if s, ok := o.Get("description"); ok {
		if s, ok := s.(value.String); ok {
			a.Description = s.Text()
		}
	}
	if b, ok := o.Get("writeOnly"); ok {
		a.WriteOnly = b == value.Bool(true)
	}
	return a
}

// Ref returns the entry and pointer of the schema object the $ref of the
// object at pointer in entry names, and false when it has none or names a
// meta-schema.
func (c *Compiler) Ref(entry, pointer string) (string, string, bool) {
	target, ok := c.refs[root{entry, pointer}]
	if !ok {
		return "", "", false
	}
	return c.target(target)
}

// DynamicRef returns the entry and pointer of the schema object the
// $dynamicRef of the object at pointer in entry names as written, resolved
// against its base URI as a $ref is, and false when it has none or names a
// meta-schema. The dynamic scope of an evaluation can select another
// target; the library's evaluation does, and DynamicRef does not.
func (c *Compiler) DynamicRef(entry, pointer string) (string, string, bool) {
	target, ok := c.dynRefs[root{entry, pointer}]
	if !ok {
		return "", "", false
	}
	return c.target(target)
}

// target places a reference's resolved URI in the package: the entry and
// pointer of the schema object it names.
func (c *Compiler) target(target string) (string, string, bool) {
	base, frag, _ := strings.Cut(target, "#")
	if frag != "" && !strings.HasPrefix(frag, "/") {
		r, ok := c.anchors[target]
		return r.entry, r.ptr, ok
	}
	r, ok := c.roots[base]
	if !ok {
		return "", "", false
	}
	if frag != "" {
		d, err := url.PathUnescape(frag)
		if err != nil {
			return "", "", false
		}
		frag = d
	}
	return r.entry, r.ptr + frag, true
}

// Lookup returns the value at pointer in the schema file entry, as
// authored, and false when there is none.
func (c *Compiler) Lookup(entry, pointer string) (value.Value, bool) {
	v := c.lookup(entry, pointer)
	return v, v != nil
}

// Subschema returns the schema at pointer in the schema file entry,
// compiled, so that a value can be validated against it alone; the sample
// generator checks each value it makes so (ADR-007 D9 item 12). It is
// evaluated as a root: a keyword whose answer depends on the schema
// around it, such as unevaluatedProperties seeing an enclosing applicator
// or a $dynamicRef the enclosing scope redirects, is answered from the
// subschema alone. Each subschema is compiled once.
func (c *Compiler) Subschema(entry, pointer string) (*Schema, error) {
	key := root{entry, pointer}
	if s, ok := c.subs[key]; ok {
		return s, nil
	}
	if _, ok := c.docs[entry]; !ok {
		return nil, fmt.Errorf("schema: %s is not a schema file of the package", entry)
	}
	u, err := url.Parse(RetrievalURI(entry))
	if err != nil {
		return nil, err
	}
	if entry == CombinedEntry {
		u, _ = url.Parse(CombinedURI)
	}
	u.Fragment = pointer
	lib, err := c.lib.Compile(u.String())
	if err != nil {
		return nil, fmt.Errorf("schema: compiling %s#%s: %w", entry, pointer, err)
	}
	s := &Schema{c: c, lib: lib}
	c.subs[key] = s
	return s, nil
}

// Properties returns the member names the properties of the object at
// pointer in entry declares.
func (c *Compiler) Properties(entry, pointer string) []string {
	o, ok := c.object(entry, pointer)
	if !ok {
		return nil
	}
	ps, ok := o.Get("properties")
	if !ok {
		return nil
	}
	if po, ok := ps.(value.Object); ok {
		return po.Names()
	}
	return nil
}

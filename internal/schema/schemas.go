// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
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

// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/schema"
)

// state is one input while the inputs object is assembled.
type state struct {
	// refused is set when no path set is applied into the input: its
	// file, or an argument for it, did not parse (ADR-003 D18 item 4).
	refused bool
	// fromFile is set while the input's whole value is its data-directory
	// file's, untouched by any argument (ADR-003 D21 item 2).
	fromFile bool
	// touched is set once a path set has changed the input's value, which
	// removes its raw form; touchedItems are the items of a collection a
	// path set changed, which lose theirs (ADR-003 D21 item 2).
	touched      bool
	touchedItems []int
	// replaced is set once an --inputs-json member has replaced the
	// input's value.
	replaced bool
	// partial is the input's array validated in part (ADR-006 D7 item 4),
	// while its value is the one the data directory gave.
	partial *partial
	// from is every source the input's value came from, in the order
	// applied (ADR-008 D7 item 5).
	from []From
}

// assemble builds the inputs object (ADR-003 D18): the data directory,
// then every --inputs-json in order, then every path set in order. It
// returns the object's root and each input's state.
func (s *Supply) assemble() (*node, map[string]*state) {
	root := newObject(nil)
	states := map[string]*state{}
	for _, in := range s.l.inputs {
		st := &state{refused: s.broken[in.Name]}
		states[in.Name] = st
		fi, ok := s.files[in.Name]
		if !ok {
			continue
		}
		st.refused = st.refused || fi.broken
		if !fi.contributes() {
			continue
		}
		var v *node
		switch {
		case fi.collection:
			v = &node{isArr: true, at: plainLocator{s.fileLoc(fi.file)}}
			var kept []int
			for _, it := range fi.items {
				if !it.ok {
					continue
				}
				it.value.index, it.value.item = it.index, it
				v.arr = append(v.arr, it.value)
				kept = append(kept, it.index)
			}
			if len(kept) < len(fi.items) {
				st.partial = &partial{kept: kept, unparsed: len(fi.items) - len(kept)}
			}
		default:
			v, st.partial = fi.value, fi.partial
		}
		root.obj[in.Name] = v
		st.fromFile = true
		st.from = []From{{Kind: FromFile, File: fi.file}}
	}
	for _, a := range s.jsonArgs {
		for name, v := range a.members {
			root.obj[name] = fromValue(v, docLocator{at: a.at.at, pos: a.at.pos, prefix: (*finding.Path)(nil).Child(name)})
			st := states[name]
			st.fromFile, st.partial, st.replaced = false, nil, true
			st.from = []From{{Kind: FromInputsJSON, Argument: a.ordinal}}
		}
	}
	for _, ps := range s.sets {
		s.apply(root, ps, states[ps.input])
	}
	return root, states
}

// apply applies the path set ps to the inputs object at root (ADR-003 D17
// item 4): each token but the last must reach an existing object or
// array, except that a missing object member is created as an empty
// object; an index must be an existing one, written without a leading
// zero; `-` is accepted only as the last token, and appends; the last
// token's member or element is created or replaced. A set that cannot be
// applied is a finding and changes nothing.
func (s *Supply) apply(root *node, ps *pathSet, st *state) {
	subj := s.l.subject(ps.input)
	loc := finding.Location{Origin: finding.OriginSet, Argument: ps.ordinal}
	at := plainLocator{loc}
	if st.refused || st.partial != nil {
		// A set into an input whose file did not parse is not applied
		// (D18 item 4). Nor is one into an array some of whose items did
		// not parse: an index would not name the item it says.
		s.add(finding.New(CodeSetIntoUnparsed, subj, &loc, finding.Params{}))
		return
	}
	fail := func(reason string) {
		s.add(finding.New(CodeSetTraversal, subj, &loc, finding.Params{"reason": finding.String(reason)}))
	}
	// cur holds the target: its member key, or its element idx when
	// isIdx. Nothing changes until every token is checked.
	toks := ps.tokens
	path := []schema.Step{schema.Member(ps.input)}
	cur, key, isIdx, idx := root, ps.input, false, 0
	missing := -1 // the token from which every object is created
	for i, tok := range toks {
		var child *node
		if !isIdx {
			child = cur.obj[key]
		} else {
			child = cur.arr[idx]
		}
		if child == nil {
			// Only an object member can be missing: every index is an
			// existing one, and `-` is only the last token. From here
			// each token is a member of an object the set creates.
			missing = i
			for _, t := range toks[i:] {
				path = append(path, schema.Member(t))
			}
			break
		}
		cur = child
		switch {
		case cur.isObj:
			key, isIdx = tok, false
			path = append(path, schema.Member(tok))
		case cur.isArr:
			n, ok := arrayIndex(tok)
			switch {
			case tok == "-" && i == len(toks)-1:
				n = len(cur.arr)
			case tok == "-":
				fail(traverseDash)
				return
			case !ok:
				fail(traverseIndex)
				return
			case n >= len(cur.arr):
				fail(traverseOutOfRange)
				return
			}
			idx, isIdx = n, true
			path = append(path, schema.Index(n))
		default:
			fail(traverseScalar)
			return
		}
	}
	var v *node
	if ps.json {
		v = fromValue(ps.value, at)
		if d := len(toks) + v.depth(); int64(d) > s.lim.Value(limits.ValueDepth) {
			s.add(s.lim.Exceeded(limits.ValueDepth, subj, loc, int64(d), true))
			return
		}
	} else {
		c, err := schema.Coerce(ps.text, s.l.walker.types(path))
		if err != nil {
			panic("inputs: " + err.Error())
		}
		if f, refused := refuseNumber(c, subj, loc.WithPointer(finding.Pointer(toks...)), s.lim); refused {
			s.add(f)
			return
		}
		v = &node{scalar: c, at: at}
	}
	switch {
	case missing >= 0:
		parent, k := cur, key
		for _, t := range toks[missing:] {
			o := newObject(at)
			parent.obj[k] = o
			parent, k = o, t
		}
		parent.obj[k] = v
	case !isIdx:
		cur.obj[key] = v
	case idx == len(cur.arr):
		v.index = idx
		cur.arr = append(cur.arr, v)
	default:
		v.index = idx
		cur.arr[idx] = v
	}
	if len(toks) == 0 {
		st.fromFile, st.from = false, nil
	}
	if st.fromFile && len(toks) > 0 {
		if i, ok := arrayIndex(toks[0]); ok {
			st.touchedItems = append(st.touchedItems, i)
		}
	}
	st.touched = true
	kind := FromSet
	if ps.json {
		kind = FromSetJSON
	}
	st.from = append(st.from, From{Kind: kind, Argument: ps.ordinal})
}

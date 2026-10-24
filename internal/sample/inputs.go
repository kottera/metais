// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sample

import (
	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/liquid/value"
)

// Generated is what the generator made of one input, as Generate returns
// it: the value, whether the input is present, every field it could not
// satisfy, and a failure that leaves the value unknown.
type Generated struct {
	Value    value.Value
	Present  bool
	Findings []finding.Finding
	Err      error
}

// Inputs generates every input of ins, in order, under constraints, the
// package path of the package's input_constraints schema, or "" when it
// has none (ADR-003 D7), and returns what it made of each, in the same
// order.
//
// Without constraints each input is Generate's. With them, the inputs
// object is made as an object is (ADR-007 D9 item 15, the modes applying
// to inputs as ADR-003 D1's required lets them), and its conditionals
// settle as an object's do (item 9). Each input is generated from its own
// schema and the schemas the constraints give its name. An optional input
// is decided by its include stream, as Generate decides it, unless the
// constraints require it, when it is generated, or give it the false
// schema, when it is omitted. Then every if of the constraints is
// evaluated against the inputs object, the branches it selects are added,
// and the inputs whose schemas or requirement changed are decided again,
// until nothing changes, at most maxPasses times. So a sample never holds
// an input the constraints refuse, nor leaves out one they require, when
// the other inputs decide which: an edition that names the tables a
// report holds.
func (s *Generator) Inputs(ins []Input, constraints string) []Generated {
	out := make([]Generated, len(ins))
	if constraints == "" {
		for i, in := range ins {
			v, present, fs, err := s.Generate(in)
			out[i] = Generated{v, present, fs, err}
		}
		return out
	}
	// The constraints are about no one input, so their own decisions draw
	// from streams of the empty input name.
	root := &place{ptr: "", refs: map[node]int{}}
	at := func() { s.reset(Input{Schema: constraints}) }
	failed := func() []Generated {
		for i := range out {
			out[i] = Generated{Present: true, Err: s.err}
		}
		return out
	}
	at()
	base := s.effective(root, []node{{constraints, ""}}, nil)
	made := make([]string, len(ins)) // the key each input was last decided from
	var extra []node
	var lastRequired map[string]bool
	changed := -1
	for pass := 0; pass < maxPasses; pass++ {
		at()
		c := base
		if len(extra) > 0 {
			c = s.with(root, base, extra)
		}
		required := s.required(root, c, members(ins, out))
		if s.err != nil {
			return failed()
		}
		changed = -1
		for i, in := range ins {
			at()
			starts := s.propStarts(c, in.Name)
			req := in.Required || required[in.Name]
			k := key(starts)
			if req {
				k += "required"
			}
			if s.err != nil {
				return failed()
			}
			if made[i] == k && pass > 0 {
				continue
			}
			made[i] = k
			if changed < 0 {
				changed = i
			}
			out[i] = s.decide(in, starts, req, c)
		}
		at()
		obj := members(ins, out)
		sel := s.selected(c, objectOf(obj))
		reqNow := s.required(root, c, obj)
		if s.err != nil {
			return failed()
		}
		if sameNodes(sel, extra) && sameSet(reqNow, required) && (lastRequired == nil || sameSet(reqNow, lastRequired)) {
			return out
		}
		extra, lastRequired = sel, reqNow
	}
	// The conditionals did not settle: the finding is about the first input
	// the last pass decided again.
	if changed < 0 {
		changed = 0
	}
	s.reset(ins[changed])
	s.unsatisfiable(root, node{constraints, ""}, ReasonConditionals)
	out[changed] = Generated{Present: true, Findings: s.findings}
	return out
}

// decide makes in from its schema and starts, the schemas the input
// constraints C give its name, when it is present: when it was supplied,
// when req, it is required, or when its include stream says so and starts
// does not hold the false schema.
func (s *Generator) decide(in Input, starts []node, req bool, c *conj) Generated {
	s.reset(in)
	if in.Supplied == nil {
		switch forbidden := s.forbidden(starts); {
		case req && forbidden:
			s.unsatisfiable(&place{ptr: ""}, s.holder(c, "required"), ReasonType)
			return Generated{Present: true, Findings: s.findings}
		case !req && (forbidden || !s.include("")):
			return Generated{}
		}
	}
	v, present, fs, err := s.make(in, starts)
	return Generated{v, present, fs, err}
}

// members is the inputs object out makes of ins: each input present and
// made in full, by its name.
func members(ins []Input, out []Generated) map[string]value.Value {
	m := map[string]value.Value{}
	for i, in := range ins {
		if g := out[i]; g.Present && g.Value != nil && len(g.Findings) == 0 && g.Err == nil {
			m[in.Name] = g.Value
		}
	}
	return m
}

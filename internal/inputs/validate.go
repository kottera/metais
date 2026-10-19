// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/regexprofile"
	"github.com/kottera/metais/internal/schemafail"
)

// Result is what a supply gave: each declared input's record, and every
// finding of the supply's parsing, assembly and validation together
// (ADR-003 D22 item 5).
type Result struct {
	// Inputs is one record per declared input, in manifest order.
	Inputs []Record
	// Findings is every finding, in no particular order: a
	// finding.Report orders, de-duplicates and caps them (ADR-006 D8).
	// Render proceeds only when none is an error (ADR-006 D7 item 5).
	Findings []finding.Finding
}

// constraintsPrefix is where the combined schema evaluates the
// `input_constraints` subschema: its allOf, which holds nothing else
// (ADR-003 D7 item 2).
const constraintsPrefix = "/allOf/"

// Validate assembles the supply into the inputs object (ADR-003 D18),
// applies its defaults (D19) and validates it against the combined schema
// (D22 steps 3 and 4), and returns every input's record with every
// finding the supply gave.
//
// An input whose file or argument did not parse is left out of the
// instance, and is not reported missing: its parse findings already are.
// A collection some of whose items did not parse, and a CSV file some of
// whose records the reader rejected alone, are validated in part (ADR-006
// D7 item 4). When any input is left out, or validated in part, or an
// --inputs-json argument did not parse, the input constraints are not
// evaluated and input.constraints_not_evaluated says so.
//
// Its error is a failure tree the mapping refuses, which is a defect in
// the tool, not in the inputs. Validate may be called once per supply.
func (s *Supply) Validate() (*Result, error) {
	if s.validated {
		panic("inputs: Validate called twice on one supply")
	}
	s.validated = true
	root, states := s.assemble()

	leftOut := map[string]bool{}
	incomplete := s.incomplete
	for _, in := range s.l.inputs {
		st := states[in.Name]
		fi := s.files[in.Name]
		switch {
		case s.broken[in.Name], fi != nil && !fi.contributes() && !st.replaced:
			// D22 item 3: unparsed, and left out of the instance.
			leftOut[in.Name] = true
			delete(root.obj, in.Name)
			incomplete = true
		case st.partial != nil:
			incomplete = true
		}
	}
	defaulted := s.l.applyDefaults(root, leftOut)

	instance := root.value().(value.Object)
	var vfs []finding.Finding
	failures, err := s.l.pkg.Schema.Validate(instance)
	var le *regexprofile.LimitError
	switch {
	case errors.As(err, &le):
		// The command's matching budget is spent, so no input's
		// validation has an answer: each input present says so.
		for _, in := range s.l.inputs {
			if n, ok := root.obj[in.Name]; ok {
				l := n.locate("", false).WithPointer("")
				vfs = append(vfs, s.lim.Exceeded(limits.RegexWork, s.l.subject(in.Name), l, 0, false))
			}
		}
	case err != nil:
		return nil, err
	default:
		vfs, err = s.mapFailures(failures, root, states, leftOut, incomplete)
		if err != nil {
			return nil, err
		}
	}
	if incomplete && s.l.pkg.Manifest.InputConstraints != "" {
		vfs = append(vfs, finding.New(CodeConstraintsSkipped, finding.Subject{Kind: finding.SubjectInvocation}, nil, finding.Params{}))
	}
	res := &Result{Findings: append(slices.Clone(s.findings), vfs...)}
	for _, in := range s.l.inputs {
		res.Inputs = append(res.Inputs, s.record(in, root, instance, states[in.Name], leftOut[in.Name], defaulted[in.Name]))
	}
	return res, nil
}

// unit is one part of the combined schema's failure tree: a subtree about
// one input, its instance pointers made the input's own (ADR-003 D6 item
// 2), with the conditions above it at the inputs object's root.
type unit struct {
	input      string
	tree       schemafail.Failure
	via        []finding.Via
	constraint bool
}

// mapFailures maps the combined schema's failures to findings (ADR-006
// D6): each subtree about one input with that input as its target, and
// what the inputs object's root itself fails, which only its `required`
// and the input constraints can, by the rules below. incomplete skips the
// constraints' failures; an input the inputs object lacks is not missing
// when it was left out, or when an --inputs-json argument that did not
// parse may have held it.
func (s *Supply) mapFailures(failures []schemafail.Failure, root *node, states map[string]*state, leftOut map[string]bool, incomplete bool) ([]finding.Finding, error) {
	var units []unit
	var out []finding.Finding
	missing := map[string]bool{}
	var walk func(n *schemafail.Failure, via []finding.Via) error
	walk = func(n *schemafail.Failure, via []finding.Via) error {
		constraint := strings.HasPrefix(n.KeywordLocation, constraintsPrefix)
		if constraint && incomplete {
			// D22 item 4: cross-input rules over an incomplete instance
			// would report errors that are not there.
			return nil
		}
		if len(n.InstancePointer) > 0 {
			name := n.InstancePointer[0]
			if _, declared := s.l.byName[name]; !declared {
				return fmt.Errorf("inputs: a failure at an undeclared input %q", name)
			}
			units = append(units, unit{input: name, tree: s.strip(n, name, constraint), via: via, constraint: constraint})
			return nil
		}
		switch n.Keyword {
		case "$ref", "$dynamicRef", "allOf", "properties", "patternProperties", "additionalProperties",
			"unevaluatedProperties", "dependentSchemas":
		case "if":
			return nil
		case "then", "else":
			via = append(slices.Clip(via), s.condition(n))
		case "required", "dependentRequired":
			// The inputs object lacks a member: an input not supplied.
			ms, ok := n.Data["missing"].([]string)
			if !ok {
				return fmt.Errorf("inputs: a %s failure without its missing members", n.Keyword)
			}
			for _, name := range ms {
				if leftOut[name] || s.incomplete || missing[name] && len(via) > 0 {
					continue
				}
				if _, declared := s.l.byName[name]; !declared {
					continue
				}
				if len(via) == 0 {
					missing[name] = true
				}
				out = append(out, finding.New(CodeMissing, s.l.subject(name), nil, finding.Params{}, finding.WithVia(via...)))
			}
			return nil
		default:
			// A rule of the input constraints over the inputs object as a
			// whole, which names no one input (ADR-006 D4 has no subject
			// for the inputs object).
			p := finding.Params{"keyword": finding.String(n.Keyword)}
			out = append(out, finding.New(CodeConstraintsFailed, finding.Subject{Kind: finding.SubjectInvocation}, nil, p,
				finding.WithSchema(finding.Schema{Entry: n.Entry, Pointer: n.SchemaPointer, Keyword: n.Keyword, KeywordLocation: n.KeywordLocation}),
				finding.WithVia(via...)))
			return nil
		}
		for i := range n.Children {
			if err := walk(&n.Children[i], via); err != nil {
				return err
			}
		}
		return nil
	}
	for i := range failures {
		if err := walk(&failures[i], nil); err != nil {
			return nil, err
		}
	}
	// An input missing unconditionally is reported once, without the
	// conditions that would also have required it.
	out = slices.DeleteFunc(out, func(f finding.Finding) bool {
		return f.Code() == CodeMissing && len(f.Via()) > 0 && missing[f.Subject().Name]
	})

	for _, in := range s.l.inputs {
		n, present := root.obj[in.Name]
		var trees []schemafail.Failure
		var cons []unit
		for _, u := range units {
			switch {
			case u.input != in.Name:
			case u.constraint || len(u.via) > 0:
				cons = append(cons, u)
			default:
				trees = append(trees, u.tree)
			}
		}
		st := states[in.Name]
		if !present || len(trees) == 0 && len(cons) == 0 && st.partial == nil {
			continue
		}
		t := schemafail.Target{
			Family:  schemafail.FamilyInput,
			Subject: s.l.subject(in.Name),
			Schemas: s.l.pkg.Compiler,
			Locate:  n.locate,
		}
		if st.partial != nil {
			t.Partial = &schemafail.Partial{Kept: st.partial.kept, Unparsed: st.partial.unparsed}
		}
		fs, err := schemafail.Map(t, trees...)
		if err != nil {
			return nil, err
		}
		out = append(out, fs...)
		t.Partial = nil
		for _, u := range cons {
			fs, err := schemafail.Map(t, u.tree)
			if err != nil {
				return nil, err
			}
			for _, f := range fs {
				out = append(out, rebuild(f, f.Subject(), keep, u.via...))
			}
		}
	}
	return out, nil
}

// condition is the via entry of a then or else at the inputs object's
// root (ADR-006 D6 item 7), as the mapping writes one.
func (s *Supply) condition(n *schemafail.Failure) finding.Via {
	toks := pointerTokens(n.SchemaPointer)
	cond := finding.Pointer(append(slices.Clip(toks[:max(len(toks)-1, 0)]), "if")...)
	return finding.Via{
		Keyword:       n.Keyword,
		SchemaPointer: n.SchemaPointer,
		Condition:     &finding.Condition{SchemaPointer: cond, Title: s.l.pkg.Compiler.Annotations(n.Entry, cond).Title},
	}
}

// strip returns the subtree n about the input name, its instance pointers
// made the input's own. A subtree the combined schema evaluates through
// the input's own `properties` member has that step taken off its
// evaluation path too, which then runs from the input's root schema
// (ADR-006 D6 item 8), as D7 item 4's suppression reads it.
func (s *Supply) strip(n *schemafail.Failure, name string, constraint bool) schemafail.Failure {
	prefix := "/properties/" + finding.Pointer(name)[1:]
	var cp func(f schemafail.Failure) schemafail.Failure
	cp = func(f schemafail.Failure) schemafail.Failure {
		out := f
		out.InstancePointer = slices.Clone(f.InstancePointer[1:])
		if !constraint {
			if rest, ok := strings.CutPrefix(f.KeywordLocation, prefix); ok && (rest == "" || rest[0] == '/') {
				out.KeywordLocation = rest
			}
		}
		out.Children = make([]schemafail.Failure, len(f.Children))
		for i, c := range f.Children {
			out.Children[i] = cp(c)
		}
		return out
	}
	return cp(*n)
}

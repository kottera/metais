// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
)

// jsonArg is one --inputs-json argument, parsed (ADR-003 D17 item 1).
type jsonArg struct {
	ordinal int
	members map[string]value.Value // its members naming declared inputs
	at      docLocator
}

// pathSet is one --set or --set-json argument, parsed (ADR-003 D17 items 2
// and 3).
type pathSet struct {
	ordinal int
	json    bool
	input   string
	tokens  []string    // the pointer's tokens after the input's name
	text    string      // a text set's value, coerced when it is applied
	value   value.Value // a JSON set's value
}

// argSubject is the subject a finding about an --inputs-json argument has
// while it is parsed: every finding the argument gives is then placed in
// the input its pointer names (ADR-003 D6 item 2's split, applied to the
// argument), or reported about the argument as a whole.
var argSubject = finding.Subject{Kind: finding.SubjectInput, Name: "_"}

// AddInputsJSON parses text, the value of the next --inputs-json flag:
// inline JSON, or standard input's bytes for `-`, which the command line
// reads within input-size (ADR-008 D7). It follows ADR-003 D9's rules and
// must be an object whose members are declared inputs (D17 item 1). A
// finding inside a member is about that member's input, at its pointer
// within the input; one outside every member is input.json_arg_invalid,
// about the argument.
func (s *Supply) AddInputsJSON(text []byte) {
	a := &jsonArg{ordinal: len(s.jsonArgs) + 1, members: map[string]value.Value{}}
	s.jsonArgs = append(s.jsonArgs, a)
	loc := finding.Location{Origin: finding.OriginInputsJSON, Argument: a.ordinal}
	invalid := func(l finding.Location, problem finding.Code, limit string) {
		p := finding.Params{"problem": finding.String(string(problem))}
		if limit != "" {
			p["limit"] = finding.String(limit)
		}
		l.Pointer, l.HasPointer = "", false
		s.add(finding.New(CodeJSONArgInvalid, finding.Subject{Kind: finding.SubjectInvocation}, &l, p))
		s.incomplete = true
	}
	if s.total+int64(len(text)) > s.lim.Value(limits.InputsTotal) {
		invalid(loc, CodeLimitExceeded, limits.Get(limits.InputsTotal).Name)
		return
	}
	s.total += int64(len(text))
	doc, fs := jsonx.Parse(text, jsonx.Source{Input: argSubject, At: loc}, s.lim)
	unknown := map[string]bool{}
	for _, f := range fs {
		l, _ := f.Location()
		name := firstToken(l.Pointer)
		limit := ""
		if v, ok := f.Param("limit"); ok {
			limit, _ = v.Str()
		}
		if !l.HasPointer || l.Pointer == "" {
			invalid(l, f.Code(), limit)
			continue
		}
		if _, declared := s.l.byName[name]; !declared {
			// A member naming no input is reported as one, and what is
			// wrong inside it is about the argument.
			if !unknown[name] {
				unknown[name] = true
				s.unknownMember(loc, doc, name)
			}
			invalid(l, f.Code(), limit)
			continue
		}
		s.broken[name] = true
		s.add(rebuild(f, s.l.subject(name), below(name)))
	}
	if len(fs) > 0 {
		// The argument gave no value, so which inputs its other members
		// would have supplied is not known.
		s.incomplete = true
		return
	}
	a.at = docLocator{at: loc, pos: doc.Positions()}
	obj, ok := doc.Value.(value.Object)
	if !ok {
		l := loc
		if p, ok := doc.Positions().Value(""); ok {
			l = l.At(p)
		}
		s.add(finding.New(CodeJSONArgNotObject, finding.Subject{Kind: finding.SubjectInvocation}, &l,
			finding.Params{"actual": finding.String(doc.Value.Kind().String())}))
		s.incomplete = true
		return
	}
	for name, v := range obj.All() {
		if _, declared := s.l.byName[name]; !declared {
			s.unknownMember(loc, doc, name)
			continue
		}
		a.members[name] = v
	}
}

// unknownMember reports the member name of an --inputs-json argument at
// loc, which names no declared input, at its name.
func (s *Supply) unknownMember(loc finding.Location, doc jsonx.Doc, name string) {
	if p, ok := doc.Positions().Key(finding.Pointer(name)); ok {
		loc = loc.At(p)
	}
	s.add(finding.New(CodeJSONArgUnknown, finding.Subject{Kind: finding.SubjectInput, Name: name}, &loc, finding.Params{}))
}

// below is rebuild's loc for a finding of an --inputs-json argument inside
// the member name: its pointer becomes one within the input.
func below(name string) func(finding.Location) finding.Location {
	prefix := finding.Pointer(name)
	return func(l finding.Location) finding.Location {
		if l.HasPointer {
			l.Pointer = strings.TrimPrefix(l.Pointer, prefix)
		}
		return l
	}
}

// AddSet parses arg, the value of the next --set flag: a text set, an
// inputs pointer, `=` and the text, which is coerced by the schema when
// the set is applied (ADR-003 D17 items 2 and 3).
func (s *Supply) AddSet(arg string) { s.addSet(arg, false) }

// AddSetJSON parses arg, the value of the next --set-json flag: a JSON
// set, an inputs pointer, `=` and a JSON text placed as it is (ADR-003
// D17 item 3).
func (s *Supply) AddSetJSON(arg string) { s.addSet(arg, true) }

func (s *Supply) addSet(arg string, isJSON bool) {
	s.nSets++
	ps := &pathSet{ordinal: s.nSets, json: isJSON}
	loc := finding.Location{Origin: finding.OriginSet, Argument: ps.ordinal}
	bad := func(reason string) {
		s.add(finding.New(CodeSetPointerInvalid, finding.Subject{Kind: finding.SubjectInvocation}, &loc,
			finding.Params{"reason": finding.String(reason)}))
	}
	ptr, text, hasValue := strings.Cut(arg, "=")
	switch {
	case !hasValue:
		bad(setNoValue)
		return
	case !strings.HasPrefix(ptr, "/"):
		bad(setNotPointer)
		return
	case !utf8.ValidString(ptr) || !validEscapes(ptr):
		bad(setMalformed)
		return
	}
	toks := pointerTokens(ptr)
	if _, declared := s.l.byName[toks[0]]; !declared {
		bad(setUnknownInput)
		return
	}
	ps.input, ps.tokens = toks[0], toks[1:]
	subj := s.l.subject(ps.input)
	if int64(len(ps.tokens)) > s.lim.Value(limits.ValueDepth) {
		s.add(s.lim.Exceeded(limits.ValueDepth, subj, loc, int64(len(ps.tokens)), true))
		return
	}
	n := int64(len(text))
	if n > s.lim.Value(limits.InputSize) {
		s.add(s.lim.Exceeded(limits.InputSize, subj, loc, n, true))
		return
	}
	if !s.charge(n, subj, loc) {
		return
	}
	target := finding.Pointer(ps.tokens...)
	if !isJSON {
		if f, ok := encoding([]byte(text), subj, loc); !ok {
			s.add(f)
			return
		}
		if f, ok := noncharacters([]byte(text), subj, loc, 1); !ok {
			s.add(f)
			return
		}
		ps.text = text
		s.sets = append(s.sets, ps)
		return
	}
	doc, fs := jsonx.Parse([]byte(text), jsonx.Source{Input: subj, At: loc}, s.lim)
	if len(fs) > 0 {
		for _, f := range fs {
			s.add(rebuild(f, subj, under(target)))
		}
		s.broken[ps.input] = true
		return
	}
	ps.value = doc.Value
	s.sets = append(s.sets, ps)
}

// validEscapes reports whether every ~ in ptr begins ~0 or ~1 (RFC 6901
// §3).
func validEscapes(ptr string) bool {
	for i := 0; i < len(ptr); i++ {
		if ptr[i] == '~' && (i+1 == len(ptr) || ptr[i+1] != '0' && ptr[i+1] != '1') {
			return false
		}
	}
	return true
}

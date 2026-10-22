// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schemafail

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"metais.org/metais/internal/finding"
)

// Family is the code family a document's failures are reported in
// (ADR-006 D6 item 1): the same keyword codes under each prefix.
type Family uint8

const (
	FamilyInput Family = iota + 1
	FamilyManifest
	FamilyOutput
)

var familyPrefixes = [...]string{FamilyInput: "input", FamilyManifest: "manifest", FamilyOutput: "output"}

func (f Family) String() string {
	if f >= FamilyInput && f <= FamilyOutput {
		return familyPrefixes[f]
	}
	return fmt.Sprintf("Family(%d)", uint8(f))
}

// Annotations are what a schema object says about the value it describes
// that the mapping reports: its title and description (ADR-006 D6 item 9)
// and whether it is writeOnly (D9 item 5).
type Annotations struct {
	Title       string
	Description string
	WriteOnly   bool
}

// Schemas reads the schema entries a failure tree's pointers name, as they
// were authored. Each method is given an entry and the RFC 6901 pointer of
// a schema object within it, and answers the zero value when the pointer
// names no schema object.
type Schemas interface {
	// Annotations returns the object's annotations.
	Annotations(entry, pointer string) Annotations
	// Ref returns the entry and pointer of the schema object the object's
	// $ref names, and false when it has no $ref.
	Ref(entry, pointer string) (string, string, bool)
	// Properties returns the member names the object's properties
	// declares.
	Properties(entry, pointer string) []string
}

// Target is the document a failure tree was validated from.
type Target struct {
	Family  Family
	Subject finding.Subject
	Schemas Schemas
	// Locate returns where the value at the instance pointer p is: its
	// origin and file, and its line and column; those of the member's name
	// when key is set (ADR-006 D5 item 6). Map sets the pointer. Map gives
	// the paths under one member name the same tokens, so a Locate that
	// reads a path by its parent's costs each its depth, never its length.
	Locate func(p *finding.Path, key bool) finding.Location
	// Partial, when set, says the document is an array validated without
	// the items that did not parse (ADR-006 D7 item 4).
	Partial *Partial
	// Listed, when positive, is the command's findings limit: the most
	// one input.any_of or input.one_of lists, itself and the findings
	// within its alternatives together (MTS-FND-020), since a report lists
	// no more than that in all. Map builds no more of a finding than that.
	Listed int
}

// Partial describes an array validated in part: Kept[i] is the original
// index of the item validated at index i, and Unparsed counts the items
// left out.
type Partial struct {
	Kept     []int
	Unparsed int
}

// maxAllowed is how many allowed values input.enum lists (ADR-006 D6 item
// 4).
const maxAllowed = 20

// noteFirst is the related note of the earlier of two equal items
// (ADR-006 D5 item 9).
const noteFirst = "first occurrence"

// Map maps the failures of one validation to findings by ADR-006 D6's
// rules, about t's subject, and, when t.Partial is set, by D7 item 4's. It
// never reads a message, since a Failure has none, and never relies on the
// order of a failure's children: the findings are the same for any order,
// and a finding.Report orders them.
//
// An error means the tree is not one an adapter may give: a node Check
// refuses, a keyword the mapping does not know, or Data missing a field
// its keyword has. It is a defect in the tool, not in the document.
func Map(t Target, fs ...Failure) ([]finding.Finding, error) {
	if t.Family < FamilyInput || t.Family > FamilyOutput || t.Schemas == nil || t.Locate == nil {
		return nil, fmt.Errorf("schemafail: an incomplete target")
	}
	if t.Partial != nil && t.Family != FamilyInput {
		return nil, fmt.Errorf("schemafail: only an input is validated in part")
	}
	m := &mapper{t: t}
	// kept holds each distinct finding once, as the command's report
	// would keep it, so a tree that reaches one keyword by many paths is
	// never held as many findings.
	kept := finding.NewReport(math.MaxInt)
	// suppressed counts the distinct findings D7 item 4 suppresses, as a
	// report would count them; it lists none of them.
	suppressed := finding.NewReport(1)
	for i := range fs {
		if err := fs[i].Check(); err != nil {
			return nil, err
		}
		m.visit(nil, &fs[i], nil, func(e emitted) {
			switch {
			case !e.ok:
			case e.suppressed:
				suppressed.Add(e.f)
			default:
				kept.Add(e.f)
			}
		})
	}
	if m.err != nil {
		return nil, m.err
	}
	out := kept.Findings()
	if t.Partial != nil {
		s := suppressed.Summary()
		loc := t.Locate(nil, false).WithPath(nil)
		out = append(out, finding.New(finding.CodeInputValidationPartial, t.Subject, &loc, finding.Params{
			"unparsed":   finding.Int(int64(t.Partial.Unparsed)),
			"suppressed": finding.Int(int64(s.Listed + s.Omitted)),
		}))
	}
	return out, nil
}

type mapper struct {
	t   Target
	err error
	// paths makes each finding's instance path from the one it made
	// before, so the failures under one member name read it once.
	paths finding.PathCache
	// fields holds field's answer for each failure on the path being
	// walked that it has looked up, so the failures beneath one schema
	// object read its chain once.
	fields map[*Failure]fieldAt
}

// emitted is one finding, and whether D7 item 4 suppresses it. A failure
// the mapping refuses emits none: ok is false.
type emitted struct {
	f          finding.Finding
	suppressed bool
	ok         bool
}

func (m *mapper) fail(n *Failure, format string, args ...any) {
	if m.err == nil {
		m.err = fmt.Errorf("schemafail: %s at %q: %s", n.Keyword, n.KeywordLocation, fmt.Sprintf(format, args...))
	}
}

// boundKeywords are the keywords whose code reports a bound and a count.
var boundKeywords = map[string]string{
	"minProperties": "min_properties", "maxProperties": "max_properties",
	"minItems": "min_items", "maxItems": "max_items",
	"minContains": "min_contains", "maxContains": "max_contains",
}

// numericKeywords are the keywords whose code reports a limit and the
// value.
var numericKeywords = map[string]string{
	"minimum": "minimum", "maximum": "maximum",
	"exclusiveMinimum": "exclusive_minimum", "exclusiveMaximum": "exclusive_maximum",
}

// visit maps n, beneath path, and gives sink each of its findings, each
// with via, the conditionals and alternatives above it, outermost first.
// A transparent keyword hands its children's findings straight on, so a
// tree of failures is never held as findings beyond what sink keeps.
func (m *mapper) visit(path []*Failure, n *Failure, via []finding.Via, sink func(emitted)) {
	// Once n is mapped, nothing beneath it asks for its field again, so
	// field's answers are held for the path being walked and no more.
	defer delete(m.fields, n)
	children := func(via []finding.Via) {
		p := append(slices.Clip(path), n)
		for i := range n.Children {
			m.visit(p, &n.Children[i], via, sink)
		}
	}
	switch k := n.Keyword; k {
	// D6 item 5: transparent.
	case "$ref", "$dynamicRef", "allOf", "properties", "patternProperties", "additionalProperties",
		"unevaluatedProperties", "items", "prefixItems", "unevaluatedItems", "dependentSchemas":
		children(via)
	case "contains":
		sink(m.emit(path, n, n.InstancePointer, "contains", k, finding.Params{}, via))
		children(via)

	// D6 item 7: transparent with a reason; D6 item 10: if never reports.
	case "if":
	case "then", "else":
		toks := tokens(n.SchemaPointer)
		cond := pointer(append(slices.Clip(toks[:max(len(toks)-1, 0)]), "if"))
		v := append(slices.Clip(via), finding.Via{
			Keyword:       k,
			SchemaPointer: n.SchemaPointer,
			Condition:     &finding.Condition{SchemaPointer: cond, Title: m.t.Schemas.Annotations(n.Entry, cond).Title},
		})
		children(v)

	case "anyOf", "oneOf":
		m.choice(path, n, via, sink)
	default:
		for _, e := range m.node(path, n, via) {
			sink(e)
		}
	}
}

// node maps n, a failure of a keyword visit does not pass through, beneath
// path, to its findings, each with via.
func (m *mapper) node(path []*Failure, n *Failure, via []finding.Via) []emitted {
	inst := n.InstancePointer
	one := func(suffix string, p finding.Params, opts ...finding.Option) []emitted {
		return []emitted{m.emit(path, n, inst, suffix, n.Keyword, p, via, opts...)}
	}
	switch k := n.Keyword; k {
	// D6 item 10: the subschemas of not and propertyNames report nothing
	// from inside them.
	case "not":
		return one("not", finding.Params{})
	case "propertyNames":
		name := m.str(n, dataProperty)
		return []emitted{m.emit(path, n, append(slices.Clip(inst), name), "property_name", k,
			finding.Params{"property": finding.String(name)}, via)}

	case KeywordFalse:
		return m.falseSchema(path, n, via)

	// ADR-003 D3 item 4: the content vocabulary is never asserted.
	case "contentEncoding", "contentMediaType", "contentSchema":
		return nil

	// D6 item 3: one finding per missing member.
	case "required", "dependentRequired":
		suffix, p := "required", finding.Params{}
		if k == "dependentRequired" {
			suffix, p = "dependent_required", finding.Params{"because": finding.String(m.str(n, dataBecause))}
		}
		var out []emitted
		for _, name := range m.strs(n, dataMissing) {
			q := maps.Clone(p)
			q["property"] = finding.String(name)
			out = append(out, m.emit(path, n, append(slices.Clip(inst), name), suffix, k, q, via))
		}
		return out

	case "type":
		var want []finding.Value
		for _, t := range m.strs(n, dataWant) {
			want = append(want, finding.String(t))
		}
		return one("type", finding.Params{"expected": finding.List(want...), "actual": finding.String(m.str(n, dataGot))})
	case "enum":
		return one("enum", m.enum(n))
	case "const":
		return one("const", finding.Params{"expected": finding.String(m.str(n, dataExpected))})
	case "minLength", "maxLength":
		suffix := map[string]string{"minLength": "min_length", "maxLength": "max_length"}[k]
		return one(suffix, finding.Params{"limit": m.num(n, dataLimit), "length": m.num(n, dataLength)})
	case "pattern":
		return one("pattern", finding.Params{"pattern": finding.String(m.str(n, dataPattern))})
	case "format":
		return one("format", finding.Params{"format": finding.String(m.str(n, dataFormat))})
	case "multipleOf":
		return one("multiple_of", finding.Params{"divisor": m.num(n, dataDivisor), "value": m.num(n, dataValue)})
	case "uniqueItems":
		return []emitted{m.uniqueItems(path, n, via)}
	}
	if suffix, ok := boundKeywords[n.Keyword]; ok {
		return one(suffix, finding.Params{"limit": m.num(n, dataLimit), "count": m.num(n, dataCount)})
	}
	if suffix, ok := numericKeywords[n.Keyword]; ok {
		return one(suffix, finding.Params{"limit": m.num(n, dataLimit), "value": m.num(n, dataValue)})
	}
	m.fail(n, "a keyword the mapping does not know")
	return nil
}

// choice maps an anyOf or oneOf none of whose branches matched by ADR-006
// D6 item 6, and a oneOf more than one of whose did by its item 5, and
// gives sink the findings.
func (m *mapper) choice(path []*Failure, n *Failure, via []finding.Via, sink func(emitted)) {
	suffix := map[string]string{"anyOf": "any_of", "oneOf": "one_of"}[n.Keyword]
	if _, ok := n.Data[dataMatched]; ok && n.Keyword == "oneOf" {
		var bs []finding.Value
		for _, b := range m.ints(n, dataMatched) {
			bs = append(bs, finding.Int(int64(b)))
		}
		sink(m.emit(path, n, n.InstancePointer, "one_of.ambiguous", n.Keyword, finding.Params{"branches": finding.List(bs...)}, via))
		return
	}
	count := m.int(n, dataBranchCount)
	branches := make([][]*Failure, max(count, 0))
	for i := range n.Children {
		c := &n.Children[i]
		b, ok := branchOf(n, c)
		if !ok || b >= count {
			m.fail(n, "a child at %q is in no branch of %d", c.KeywordLocation, count)
			return
		}
		branches[b] = append(branches[b], c)
	}
	p := append(slices.Clip(path), n)
	var open []int
	for b, cs := range branches {
		if !slices.ContainsFunc(cs, func(c *Failure) bool { return discriminates(n.InstancePointer, c) }) {
			open = append(open, b)
		}
	}
	// D6 item 6.3: the one branch not rejected is reported as if the
	// choice were not there, with a reason.
	if len(open) == 1 {
		b := open[0]
		v := append(slices.Clip(via), finding.Via{Keyword: n.Keyword, SchemaPointer: n.SchemaPointer, Branch: b, Title: m.branchTitle(n, b)})
		given := 0
		for _, c := range branches[b] {
			m.visit(p, c, v, func(e emitted) {
				given++
				sink(e)
			})
		}
		if given > 0 {
			return
		}
	}
	// D6 item 6.4: every branch, with its findings, each branch's gathered
	// as the finding keeps them, so no more of a branch is held than it
	// lists.
	alts := make([]finding.Alternative, len(branches))
	for b, cs := range branches {
		br := finding.NewBranch(b, m.branchTitle(n, b))
		for _, c := range cs {
			m.visit(p, c, nil, func(e emitted) {
				if e.ok && !e.suppressed {
					br.Add(e.f)
				}
			})
		}
		alts[b] = br.Alternative()
	}
	sink(m.emit(path, n, n.InstancePointer, suffix, n.Keyword,
		finding.Params{"branch_count": finding.Int(int64(count))}, via, finding.WithAlternatives(alts...), finding.WithListedAtMost(m.t.Listed)))
}

// branchOf returns the index of the branch of choice c is evaluated in,
// from c's evaluation path, which an adapter gives whether or not it
// gives a node for each $ref between them.
//
// It compares the two paths as written, a token at a time being equal
// exactly when its escaped form is, so that a deep tree's paths are never
// split for each of its failures.
func branchOf(choice, c *Failure) (int, bool) {
	rest, ok := strings.CutPrefix(c.KeywordLocation, choice.KeywordLocation+"/")
	if !ok {
		return 0, false
	}
	tok, _, _ := strings.Cut(rest, "/")
	b, err := strconv.Atoi(tok)
	if err != nil || b < 0 || strconv.Itoa(b) != tok {
		return 0, false
	}
	return b, true
}

// discriminates reports whether c, or a failure beneath it, rejects its
// branch of a choice applied at inst (ADR-006 D6 item 6.1): a type failure
// at inst, or a const, enum or false-schema failure at inst or at a direct
// member of it. A nested choice, and the contexts that never report, are
// not looked into.
func discriminates(inst []string, c *Failure) bool {
	found := false
	c.Walk(func(_ []*Failure, f *Failure) bool {
		at := slices.Equal(f.InstancePointer, inst)
		member := len(f.InstancePointer) == len(inst)+1 && slices.Equal(f.InstancePointer[:len(inst)], inst)
		switch f.Keyword {
		case "type":
			found = found || at
		case "const", "enum", KeywordFalse:
			found = found || at || member
		case "anyOf", "oneOf", "not", "if", "propertyNames":
			return false
		}
		return !found
	})
	return found
}

// falseSchema maps a false schema the instance reached, named by the
// applicator that reached it (ADR-006 D6 item 4).
func (m *mapper) falseSchema(path []*Failure, n *Failure, via []finding.Via) []emitted {
	toks := tokens(n.SchemaPointer)
	ss, rest := steps(toks)
	kw := KeywordFalse
	holder := toks
	if len(ss) > 0 && rest == len(toks) {
		last := ss[len(ss)-1]
		kw = last.keyword
		start := 0
		if len(ss) > 1 {
			start = ss[len(ss)-2].end
		}
		holder = toks[:start]
	}
	inst := n.InstancePointer
	lastInst := ""
	if len(inst) > 0 {
		lastInst = inst[len(inst)-1]
	}
	switch kw {
	case "additionalProperties", "unevaluatedProperties":
		if len(inst) == 0 {
			m.fail(n, "a refused member with no name")
			return nil
		}
		suffix := map[string]string{"additionalProperties": "additional_property", "unevaluatedProperties": "unevaluated_property"}[kw]
		p := finding.Params{"property": finding.String(lastInst)}
		if s, ok := suggest(lastInst, m.t.Schemas.Properties(n.Entry, pointer(holder)), false); ok {
			p["suggestion"] = finding.String(s)
		}
		return []emitted{m.emit(path, n, inst, suffix, kw, p, via)}
	case "items", "unevaluatedItems":
		i, err := strconv.Atoi(lastInst)
		if len(inst) == 0 || err != nil {
			m.fail(n, "a refused item with no index")
			return nil
		}
		if len(inst) == 1 {
			i = m.original(n, i)
		}
		if kw == "items" {
			return []emitted{m.emit(path, n, inst, "additional_item", kw,
				finding.Params{"index": finding.Int(int64(i)), "prefix_count": m.num(n, dataPrefixCount)}, via)}
		}
		return []emitted{m.emit(path, n, inst, "unevaluated_item", kw, finding.Params{"index": finding.Int(int64(i))}, via)}
	}
	p := finding.Params{}
	if kw == "properties" {
		p["property"] = finding.String(toks[len(toks)-1])
	}
	return []emitted{m.emit(path, n, inst, "forbidden", kw, p, via)}
}

// uniqueItems maps a uniqueItems failure to a finding at the later of the
// two equal items, related to the earlier.
func (m *mapper) uniqueItems(path []*Failure, n *Failure, via []finding.Via) emitted {
	d := m.ints(n, dataDuplicates)
	if len(d) != 2 || d[0] < 0 || d[1] < 0 || d[0] == d[1] {
		m.fail(n, "duplicates is not two indexes")
		return emitted{}
	}
	first, later := min(d[0], d[1]), max(d[0], d[1])
	inst := n.InstancePointer
	at := append(slices.Clip(inst), strconv.Itoa(first))
	if len(inst) == 0 {
		first, later = m.original(n, first), m.original(n, later)
	}
	firstPath := m.paths.Path(nil, m.remap(n, at))
	rel := finding.Related{Location: m.t.Locate(firstPath, false).WithPath(firstPath), Note: noteFirst}
	return m.emit(path, n, append(slices.Clip(inst), strconv.Itoa(max(d[0], d[1]))), "unique_items", n.Keyword,
		finding.Params{"index": finding.Int(int64(later)), "duplicate_of": finding.Int(int64(first))}, via, finding.WithRelated(rel))
}

// emit builds the finding for the keyword code suffix of n, at instance
// location inst, its schema keyword kw.
func (m *mapper) emit(path []*Failure, n *Failure, inst []string, suffix, kw string, p finding.Params, via []finding.Via, opts ...finding.Option) emitted {
	if m.err != nil {
		return emitted{}
	}
	code := finding.Code(m.t.Family.String() + "." + suffix)
	entry, ok := finding.Lookup(code)
	if !ok {
		m.fail(n, "%s is not registered", code)
		return emitted{}
	}
	var fd finding.Field
	writeOnly := false
	switch suffix {
	case "required", "dependent_required":
		name, _ := p["property"].Str()
		fd = m.memberField(n, name)
	default:
		fd, writeOnly = m.field(path, n, inst)
	}
	// D9 item 5: under writeOnly, no value, no length and no suggestion.
	if writeOnly {
		delete(p, "value")
		delete(p, "length")
		delete(p, "suggestion")
	}
	suppressed := m.suppressed(n, inst)
	at := m.paths.Path(nil, m.remap(n, inst))
	var loc finding.Location
	switch entry.Span {
	case finding.SpanKey:
		loc = m.t.Locate(at, true)
	case finding.SpanParent:
		loc = m.t.Locate(at.Parent(), false)
	default:
		loc = m.t.Locate(at, false)
	}
	loc = loc.WithPath(at)
	opts = append([]finding.Option{
		finding.WithSchema(finding.Schema{Entry: n.Entry, Pointer: n.SchemaPointer, Keyword: kw, KeywordLocation: n.KeywordLocation}),
		finding.WithField(fd),
		finding.WithVia(via...),
	}, opts...)
	if m.err != nil {
		return emitted{}
	}
	return emitted{finding.New(code, m.t.Subject, &loc, p, opts...), suppressed, true}
}

// suppressed reports whether D7 item 4 suppresses a finding of n at inst:
// one about the array itself, or one whose evaluation path passes through
// prefixItems or unevaluatedItems applied to the array.
func (m *mapper) suppressed(n *Failure, inst []string) bool {
	if m.t.Partial == nil {
		return false
	}
	if len(inst) == 0 {
		return true
	}
	ss, _ := steps(tokens(n.KeywordLocation))
	depth := 0
	for _, s := range ss {
		if depth == 0 && (s.keyword == "prefixItems" || s.keyword == "unevaluatedItems") {
			return true
		}
		if s.descends {
			depth++
		}
	}
	return false
}

// remap returns inst with its first token, an index of the array validated
// in part, made the item's original index (ADR-006 D7 item 4).
func (m *mapper) remap(n *Failure, inst []string) []string {
	if m.t.Partial == nil || len(inst) == 0 {
		return inst
	}
	i, err := strconv.Atoi(inst[0])
	if err != nil {
		m.fail(n, "the array validated in part has a member %q", inst[0])
		return inst
	}
	return append([]string{strconv.Itoa(m.original(n, i))}, inst[1:]...)
}

// original returns the original index of the item validated at index i.
func (m *mapper) original(n *Failure, i int) int {
	if m.t.Partial == nil {
		return i
	}
	if i < 0 || i >= len(m.t.Partial.Kept) {
		m.fail(n, "item %d is beyond the %d items validated", i, len(m.t.Partial.Kept))
		return i
	}
	return m.t.Partial.Kept[i]
}

// enum is input.enum's parameters: the first allowed values, how many
// there are, and the suggestion when the value is a string.
func (m *mapper) enum(n *Failure) finding.Params {
	allowed := m.strs(n, dataAllowed)
	var list []finding.Value
	for _, a := range allowed[:min(len(allowed), maxAllowed)] {
		list = append(list, finding.String(a))
	}
	p := finding.Params{"allowed": finding.List(list...), "allowed_count": finding.Int(int64(len(allowed)))}
	if v, ok := n.Data[dataGot]; ok {
		got, ok := v.(string)
		if !ok {
			m.fail(n, "%s is a %T", dataGot, v)
			return p
		}
		if s, ok := suggest(got, jsonStrings(allowed), true); ok {
			p["suggestion"] = finding.String(s)
		}
	}
	return p
}

func (m *mapper) str(n *Failure, key string) string {
	s, ok := n.Data[key].(string)
	if !ok {
		m.fail(n, "%s is not a string", key)
	}
	return s
}

func (m *mapper) strs(n *Failure, key string) []string {
	s, ok := n.Data[key].([]string)
	if !ok {
		m.fail(n, "%s is not a list of strings", key)
	}
	return s
}

func (m *mapper) ints(n *Failure, key string) []int {
	s, ok := n.Data[key].([]int)
	if !ok {
		m.fail(n, "%s is not a list of ints", key)
	}
	return s
}

func (m *mapper) int(n *Failure, key string) int {
	i, ok := n.Data[key].(int)
	if !ok {
		m.fail(n, "%s is not an int", key)
	}
	return i
}

// num is the number at key: an int, or a Number in its canonical form.
func (m *mapper) num(n *Failure, key string) finding.Value {
	switch v := n.Data[key].(type) {
	case int:
		return finding.Int(int64(v))
	case Number:
		f, err := finding.Num(string(v))
		if err != nil {
			m.fail(n, "%s: %v", key, err)
		}
		return f
	}
	m.fail(n, "%s is not a number", key)
	return finding.Int(0)
}

// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package sample is the reference tool's sample-data generator (ADR-007
// D9): it makes a value for each input from the input's schema, and
// serializes it in the input's first form, as a data directory holds it.
//
// Every decision draws from its own SplitMix64 stream, seeded by the seed,
// the input, the instance location and the decision's purpose, so the same
// schema, seed, mode and item count give the same value on every run and
// platform, and a value depends only on where it sits (items 4 to 6).
// Numbers are exact rationals until they are written. At each location the
// generator works on the conjunction of the schemas there (items 7 to 11),
// prefers the author's const, examples, default and enum to a value of its
// own (items 12 and 13), and otherwise generates one by type (items 14 to
// 20): strings from formats, from a walk of the pattern's regexp/syntax
// tree as regexprofile translates it, or from a fixed placeholder corpus.
// Each value it keeps is validated against the schemas of its location by
// the package's compiled schemas (internal/schema). A field it cannot
// satisfy is schema.sample_unsatisfiable, and every such field is reported
// in one run (item 23).
//
// The generator is not normative (item 3): the specification says only
// that a sample mode produces data that validates.
package sample

import (
	"fmt"
	"strconv"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/inputs/csv"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/regexprofile"
	"metais.org/metais/internal/schema"
)

// Mode is the optional-field mode (ADR-007 D9 items 1, 14 and 15).
type Mode uint8

// The modes. The zero Mode is All, the default.
const (
	// All generates every optional property and input, and never chooses
	// null.
	All Mode = iota
	// None omits every optional property and input, and chooses null
	// wherever null is allowed.
	None
	// Mixed decides each by its include or type stream.
	Mixed
)

// ParseMode returns the mode --optional names: all, none or mixed.
func ParseMode(s string) (Mode, bool) {
	switch s {
	case "all":
		return All, true
	case "none":
		return None, true
	case "mixed":
		return Mixed, true
	}
	return 0, false
}

func (m Mode) String() string {
	switch m {
	case All:
		return "all"
	case None:
		return "none"
	case Mixed:
		return "mixed"
	}
	return "Mode(" + strconv.Itoa(int(m)) + ")"
}

// DefaultItems is --items's default, the target count of every array and
// collection.
const DefaultItems = 3

// Options are the generator's options (ADR-007 D9 item 1).
type Options struct {
	// Seed is --seed.
	Seed uint64
	// Optional is --optional.
	Optional Mode
	// Items is --items, at least 0.
	Items int
}

// Form is one of ADR-003's input forms.
type Form string

// The forms (ADR-003 D9 to D13).
const (
	FormJSON     Form = "json"
	FormYAML     Form = "yaml"
	FormCSV      Form = "csv"
	FormText     Form = "text"
	FormMarkdown Form = "markdown"
)

// Input is one input slot as the generator reads it: the manifest's
// declaration (ADR-003 D1), already checked.
type Input struct {
	// Name is the input's name.
	Name string
	// Schema is the package path of the input's schema file.
	Schema string
	// Forms is the declaration's forms, in order. The first is the one
	// the input is written in (item 21).
	Forms []Form
	// Collection is the declaration's collection.
	Collection bool
	// Delimiter is the declaration's csv.delimiter, for a csv form.
	Delimiter csv.Delimiter
	// Required is the declaration's required, its default applied.
	Required bool
	// Supplied is the value the caller supplied for the input, merged by
	// ADR-003's precedence, or nil when none was: a fixed point the
	// generator fills around (item 2).
	Supplied value.Value
}

// form is the form the input is written in: its first (item 21).
func (in Input) form() Form {
	if len(in.Forms) == 0 {
		return FormJSON
	}
	return in.Forms[0]
}

// fileAt marks loc, the value of one file of the input, by what the
// input's form makes of it: a whole text file or Markdown document is
// prose (item 18), and a CSV file is a table.
func (in Input) fileAt(loc *place) {
	switch in.form() {
	case FormText:
		loc.prose = proseText
	case FormMarkdown:
		loc.prose, loc.markdown = proseMarkdown, true
	case FormCSV:
		loc.table = true
	}
}

// Positions returns the position table of a schema file as the package
// loader read it, by its package path, or nil when it has none. The
// generator reads from it the order in which an object schema's
// properties are written, which its value of the file does not keep
// (item 15), and the line and column a finding names.
type Positions func(entry string) finding.Positions

// Generator makes the sample values of one package's inputs.
type Generator struct {
	c    *schema.Compiler
	pos  Positions
	lim  limits.Set
	opts Options

	// The input being generated, and what its generation found.
	input    Input
	findings []finding.Finding
	err      error
	// size is what the generation has made so far, held to input-size
	// (spend), and over is the finding for its breach.
	size int64
	over []finding.Finding
	// patterns are the patterns compiled so far, by their text.
	patterns map[string]*regexprofile.Regexp
}

// New returns the generator over c, the package's compiled schemas, with
// pos, the position table of each of its schema files, under lim, the
// limits a pattern is compiled under.
func New(c *schema.Compiler, pos Positions, lim limits.Set, opts Options) *Generator {
	if opts.Items < 0 {
		panic("sample: a negative item count")
	}
	return &Generator{c: c, pos: pos, lim: lim, opts: opts}
}

// Generate returns the value the generator makes for in, and whether the
// input is present at all: an optional input is omitted under None, and
// under Mixed when its include stream's first draw is odd (ADR-007 D9
// item 15, ADR-003 D1). An input the caller supplied is present.
//
// Every field the generator cannot satisfy is reported, as
// schema.sample_unsatisfiable (item 23); when there is any, the value is
// nil. err is a failure of the schema library, or the regex-work budget
// spent, either of which leaves the value unknown.
func (s *Generator) Generate(in Input) (v value.Value, present bool, fs []finding.Finding, err error) {
	s.input, s.findings, s.err, s.size, s.over = in, nil, nil, 0, nil
	if in.Supplied == nil && !in.Required && !s.include("") {
		return nil, false, nil, nil
	}
	loc := &place{ptr: "", fixed: in.Supplied, refs: map[node]int{}}
	form := in.form()
	loc.noCR = form == FormCSV || form == FormText || form == FormMarkdown
	if in.Collection {
		loc.collection = true
	} else {
		in.fileAt(loc)
	}
	v, ok := s.value(loc, []node{{in.Schema, ""}})
	if s.err != nil {
		return nil, false, nil, s.err
	}
	if fs := append(s.findings, s.over...); !ok || len(fs) > 0 {
		if len(fs) == 0 {
			panic("sample: a value failed with no finding")
		}
		return nil, true, fs, nil
	}
	return v, true, nil, nil
}

// prose is the kind of prose a string at a place is (item 18).
type prose uint8

const (
	proseNone prose = iota
	proseText
	proseMarkdown
)

// place is an instance location being generated, and what the path to it
// decides there.
type place struct {
	// ptr is the instance JSON Pointer.
	ptr string
	// k is the index of the item within the innermost enclosing array, 0
	// outside any (item 12).
	k int
	// earlier are the items before this one, when the enclosing array is
	// uniqueItems.
	earlier []value.Value
	// avoid are values that a value below this place must not equal,
	// each at its path from here: the keys earlier items of an enclosing
	// array hold under a uniqueKeys of one pointer.
	avoid []avoided
	// refs counts each $ref target entered on the path from the root,
	// this place's own included (item 20).
	refs map[node]int
	// stop is set once a target has been entered stopRecursion times on
	// the path: optional properties are omitted and arrays take their
	// minimum length.
	stop bool
	// purpose is the purpose of the place's value stream: value, or an
	// attempt's.
	purpose string
	// attempt is the attempt the enclosing item is regenerated under,
	// when a uniqueKeys key repeated: every place below it inherits it
	// as its value stream's purpose, so the key's own value changes.
	attempt string
	// fixed is the value supplied for this place, nil when none was.
	fixed value.Value
	// prose is whether a string here is prose, and of which kind.
	prose prose
	// markdown is set on an object that is a Markdown document, whose
	// body is required and Markdown prose.
	markdown bool
	// collection is set on a collection input's root, whose items are
	// each a file of the input's form.
	collection bool
	// item is set on each item of a collection, whose body is prose.
	item bool
	// table is set on a CSV file's value, an array of records; record on
	// each of its records; and cell on each field of one, which holds no
	// null, array or object (ADR-003 D11 item 7).
	table, record, cell bool
	// noCR is set throughout an input whose form normalizes line breaks
	// before it is read (ADR-003 D8 item 2), where a CR in a string would
	// not read back.
	noCR bool
}

// stopRecursion is the entry count at which a $ref target stops growing,
// and failRecursion the one at which its path is unsatisfiable (item 20).
const (
	stopRecursion = 4
	failRecursion = 8
)

// stream returns the stream of a decision at the instance pointer ptr.
func (s *Generator) stream(ptr, purpose string) *SplitMix64 {
	return Stream(s.opts.Seed, s.input.Name, ptr, purpose)
}

// include decides an optional property or input at ptr by the mode
// (item 15).
func (s *Generator) include(ptr string) bool {
	switch s.opts.Optional {
	case None:
		return false
	case Mixed:
		return s.stream(ptr, purposeInclude).Next()%2 == 0
	}
	return true
}

// spend charges n, a value's size, to input-size (ADR-009 D10): the data
// the generator makes is held to the limits supplied data is (item 22),
// so making more than an input may hold is work for nothing, and the
// generator's work is bounded by the limit rather than by the schema. A
// string counts its bytes, and every other value one. The first breach is
// limit.exceeded about the input, at the place that breached it, and ends
// the generation.
func (s *Generator) spend(loc *place, n int64) bool {
	if !s.affords(loc, n) {
		return false
	}
	s.size += n
	return true
}

// affords reports whether n more can be made at loc within input-size, as
// spend does without charging it: a bound a value is about to be made to,
// such as minItems, is checked before any of it is made.
func (s *Generator) affords(loc *place, n int64) bool {
	if s.over != nil {
		return false
	}
	if n <= s.lim.Value(limits.InputSize)-s.size {
		return true
	}
	at := finding.Location{Origin: finding.OriginSample}.WithPointer(loc.ptr)
	s.over = []finding.Finding{s.lim.Exceeded(limits.InputSize, finding.Subject{Kind: finding.SubjectInput, Name: s.input.Name}, at, 0, false)}
	return false
}

// stopped reports whether the generation can go no further: the schema
// library failed, or input-size is spent.
func (s *Generator) stopped() bool { return s.err != nil || s.over != nil }

// valid reports whether v validates against each schema of starts.
func (s *Generator) valid(starts []node, v value.Value) bool {
	for _, n := range starts {
		if s.err != nil {
			return false
		}
		sch, err := s.c.Subschema(n.entry, n.ptr)
		if err != nil {
			s.err = err
			return false
		}
		fails, err := sch.Validate(v)
		if err != nil {
			s.err = err
			return false
		}
		if len(fails) > 0 {
			return false
		}
	}
	return true
}

// The reasons of schema.sample_unsatisfiable (ADR-007 D9 item 23, and the
// two ADR-003 Amendment A2 item 17 adds: unique_keys and coverage).
const (
	ReasonType               = "type"
	ReasonRange              = "range"
	ReasonLength             = "length"
	ReasonEnum               = "enum"
	ReasonConst              = "const"
	ReasonMultipleOf         = "multiple_of"
	ReasonPatternUnsupported = "pattern_unsupported"
	ReasonPatternUnmet       = "pattern_unmet"
	ReasonFormat             = "format"
	ReasonUniqueItems        = "unique_items"
	ReasonUniqueKeys         = "unique_keys"
	ReasonCoverage           = "coverage"
	ReasonNot                = "not"
	ReasonConditionals       = "conditionals"
	ReasonRecursion          = "recursion"
)

// CodeUnsatisfiable is the finding for a field the generator cannot make a
// value for (ADR-007 D9 item 23).
const CodeUnsatisfiable finding.Code = "schema.sample_unsatisfiable"

// unsatisfiable reports the field at loc, whose schema at n the generator
// cannot satisfy, for reason. The finding is about the input; its
// location is the subschema, in its schema file, and its instance
// parameter the field's place in the input's value.
func (s *Generator) unsatisfiable(loc *place, n node, reason string) {
	at := finding.Location{Origin: finding.OriginPackage, File: n.entry}.WithPointer(n.ptr)
	if s.pos != nil {
		if t := s.pos(n.entry); t != nil {
			if p, ok := t.Value(n.ptr); ok {
				at = at.At(p)
			}
		}
	}
	s.findings = append(s.findings, finding.New(CodeUnsatisfiable,
		finding.Subject{Kind: finding.SubjectInput, Name: s.input.Name}, &at,
		finding.Params{"instance": finding.String(loc.ptr), "reason": finding.String(reason)}))
}

// itoa is strconv.Itoa.
func itoa(i int) string { return strconv.Itoa(i) }

// child is the place of the member or element token below loc, on the
// path loc's targets were entered on.
func (loc *place) child(token string) *place {
	child := &place{ptr: loc.ptr + finding.Pointer(token), k: loc.k, refs: loc.refs, stop: loc.stop, noCR: loc.noCR, attempt: loc.attempt}
	for _, a := range loc.avoid {
		if len(a.path) > 0 && a.path[0] == token {
			child.avoid = append(child.avoid, avoided{a.path[1:], a.held})
		}
	}
	return child
}

// avoided is the values that the value at path below a place must not
// equal, each by its digest.
type avoided struct {
	path []string
	held map[string]bool
}

// avoids reports whether loc's own value must not be v.
func (loc *place) avoids(v value.Value) bool {
	for _, a := range loc.avoid {
		if len(a.path) == 0 && a.held[digest(v)] {
			return true
		}
	}
	return false
}

// valuePurpose is the purpose of loc's value stream.
func (loc *place) valuePurpose() string {
	switch {
	case loc.purpose != "":
		return loc.purpose
	case loc.attempt != "":
		return loc.attempt
	}
	return purposeValue
}

// errorf is a defect in the generator, never in its input.
func errorf(format string, args ...any) error { return fmt.Errorf("sample: "+format, args...) }

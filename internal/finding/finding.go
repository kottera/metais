// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package finding is the one way the tool reports a problem (ADR-006): the
// finding model, the registry of codes every finding is built from, the
// report that orders, de-duplicates and caps a command's findings, their
// JSON and human encodings, and the exit status they give.
//
// A Finding is built only by New, from a registered code and its
// parameters. Its message and hint are rendered from the registry's
// templates, so no code path can put free text, and with it a value from
// the inputs, into a finding (ADR-006 D1 item 4, D9 item 4).
package finding

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// Severity is how serious a finding is (ADR-006 D2). It is fixed by the
// code.
type Severity uint8

const (
	SeverityError Severity = iota + 1
	SeverityWarning
	SeverityInfo
)

var severityNames = [...]string{SeverityError: "error", SeverityWarning: "warning", SeverityInfo: "info"}

func (s Severity) String() string {
	if s >= SeverityError && s <= SeverityInfo {
		return severityNames[s]
	}
	return fmt.Sprintf("Severity(%d)", uint8(s))
}

// Class is the kind of failure a code reports, which decides the exit
// status (ADR-006 D3 item 5, D12).
type Class uint8

const (
	ClassUsage Class = iota + 1
	ClassRefused
	ClassContent
	ClassDestination
	ClassInternal
)

var classNames = [...]string{ClassUsage: "usage", ClassRefused: "refused", ClassContent: "content", ClassDestination: "destination", ClassInternal: "internal"}

func (c Class) String() string {
	if c >= ClassUsage && c <= ClassInternal {
		return classNames[c]
	}
	return fmt.Sprintf("Class(%d)", uint8(c))
}

// SubjectKind is the kind of thing a finding is about (ADR-006 D4 item 1).
// The constants are declared in ADR-006 D8 item 1's sort order, so a kind's
// value is its rank in the report.
type SubjectKind uint8

const (
	SubjectInvocation SubjectKind = iota + 1
	SubjectPackage
	SubjectManifest
	SubjectSchema
	SubjectInput
	SubjectTemplate
	SubjectAsset
	SubjectReference
	SubjectOutput
	SubjectExample
	SubjectReport
)

var subjectNames = [...]string{
	SubjectInvocation: "invocation", SubjectPackage: "package", SubjectManifest: "manifest",
	SubjectSchema: "schema", SubjectInput: "input", SubjectTemplate: "template",
	SubjectAsset: "asset", SubjectReference: "reference", SubjectOutput: "output",
	SubjectExample: "example", SubjectReport: "report",
}

func (k SubjectKind) String() string {
	if k >= SubjectInvocation && k <= SubjectReport {
		return subjectNames[k]
	}
	return fmt.Sprintf("SubjectKind(%d)", uint8(k))
}

// Origin is the source a located value or file came from (ADR-006 D4 item
// 4). The constants are declared in ADR-006 D8 item 1's sort order.
type Origin uint8

const (
	OriginPackage Origin = iota + 1
	OriginExample
	OriginDataDir
	OriginInputsJSON
	OriginSet
	OriginDefault
	OriginSample
	OriginOutput
)

var originNames = [...]string{
	OriginPackage: "package", OriginExample: "example", OriginDataDir: "data_dir",
	OriginInputsJSON: "inputs_json", OriginSet: "set", OriginDefault: "default",
	OriginSample: "sample", OriginOutput: "output",
}

func (o Origin) String() string {
	if o >= OriginPackage && o <= OriginOutput {
		return originNames[o]
	}
	return fmt.Sprintf("Origin(%d)", uint8(o))
}

// SpanKind is which character a code's line and column mark (ADR-006 D5
// item 6). SpanNone registers a code whose findings never carry a
// location: a usage finding, or the report's own (D1 item 2).
type SpanKind uint8

const (
	SpanValue SpanKind = iota + 1
	SpanKey
	SpanParent
	SpanNone
)

var spanNames = [...]string{SpanValue: "value", SpanKey: "key", SpanParent: "parent", SpanNone: "none"}

func (s SpanKind) String() string {
	if s >= SpanValue && s <= SpanNone {
		return spanNames[s]
	}
	return fmt.Sprintf("SpanKind(%d)", uint8(s))
}

// Code is a registered finding code, such as input.min_length (ADR-006 D3).
type Code string

// Subject is what a finding is about (ADR-006 D4). Name and Title are
// empty when absent.
type Subject struct {
	Kind  SubjectKind
	Name  string
	Title string
}

// Location is where the value or file a finding reports is (ADR-006 D5).
// File is relative to its origin's root and empty when absent; Argument is
// the 1-based ordinal of a command-line argument, zero when absent; Line
// and Column are 1-based, zero when unknown. Pointer is an RFC 6901 JSON
// Pointer, present when HasPointer is set: "" is the root.
type Location struct {
	Origin     Origin
	File       string
	Argument   int
	Line       int
	Column     int
	Pointer    string
	HasPointer bool
}

// At returns l with its line and column set from p.
func (l Location) At(p Pos) Location {
	l.Line, l.Column = p.Line, p.Column
	return l
}

// WithPointer returns l pointing at ptr.
func (l Location) WithPointer(ptr string) Location {
	l.Pointer, l.HasPointer = ptr, true
	return l
}

// Schema is the schema keyword a finding's value broke (ADR-006 D6 item 8).
type Schema struct {
	Entry           string
	Pointer         string
	Keyword         string
	KeywordLocation string
}

// Field is the title and description of the field a finding is about
// (ADR-006 D6 item 9). Either may be empty.
type Field struct {
	Title       string
	Description string
}

// Condition is the if subschema that selected a then or else (ADR-006 D6
// item 7).
type Condition struct {
	SchemaPointer string
	Title         string
}

// Via is one conditional or alternative that made a rule apply (ADR-006 D6
// items 6 and 7). Keyword is anyOf or oneOf, with Branch, or then or else,
// with Condition.
type Via struct {
	Keyword       string
	SchemaPointer string
	Branch        int
	Title         string
	Condition     *Condition
}

// Alternative is one branch of an anyOf or oneOf none of whose branches
// matched (ADR-006 D6 item 6): its findings, sorted, and how many more there
// were.
type Alternative struct {
	Branch   int
	Title    string
	Findings []Finding
	Omitted  int
}

// Related is another location a finding concerns, with a fixed note
// (ADR-006 D5 item 9).
type Related struct {
	Location Location
	Note     string
}

// Finding is one problem, built only by New (ADR-006 D1). Its members are
// read through its methods.
type Finding struct {
	entry        *Entry
	subject      Subject
	location     *Location
	params       Params
	message      string
	hint         string
	schema       *Schema
	field        *Field
	via          []Via
	alternatives []Alternative
	related      []Related

	ptrTokens []string // the location's pointer, decoded, for Compare
	paramsKey string   // the canonical form of params, for Compare and the duplicate key
}

func (f Finding) Code() Code                  { return f.entry.Code }
func (f Finding) Severity() Severity          { return f.entry.Severity }
func (f Finding) Class() Class                { return f.entry.Class }
func (f Finding) Message() string             { return f.message }
func (f Finding) Hint() string                { return f.hint }
func (f Finding) Subject() Subject            { return f.subject }
func (f Finding) Via() []Via                  { return slices.Clone(f.via) }
func (f Finding) Alternatives() []Alternative { return slices.Clone(f.alternatives) }
func (f Finding) Related() []Related          { return slices.Clone(f.related) }

// Location returns the finding's location, and false when it has none.
func (f Finding) Location() (Location, bool) {
	if f.location == nil {
		return Location{}, false
	}
	return *f.location, true
}

// Schema returns the schema keyword the finding reports, and false when it
// has none.
func (f Finding) Schema() (Schema, bool) {
	if f.schema == nil {
		return Schema{}, false
	}
	return *f.schema, true
}

// Field returns the field's title and description, and false when neither
// is known.
func (f Finding) Field() (Field, bool) {
	if f.field == nil {
		return Field{}, false
	}
	return *f.field, true
}

// Param returns the parameter called name, and false when it is absent.
func (f Finding) Param(name string) (Value, bool) {
	v, ok := f.params[name]
	return v, ok
}

// Option adds a member to a finding New builds.
type Option func(*Finding)

// WithSchema records the schema keyword the finding reports.
func WithSchema(s Schema) Option { return func(f *Finding) { f.schema = &s } }

// WithField records the field's title and description. A field with
// neither is no field.
func WithField(fd Field) Option {
	return func(f *Finding) {
		if fd.Title != "" || fd.Description != "" {
			f.field = &fd
		}
	}
}

// WithVia records the conditionals and alternatives, outermost first.
func WithVia(v ...Via) Option { return func(f *Finding) { f.via = append(f.via, v...) } }

// WithAlternatives records the branches of an anyOf or oneOf.
func WithAlternatives(a ...Alternative) Option {
	return func(f *Finding) { f.alternatives = append(f.alternatives, a...) }
}

// WithRelated records other locations the finding concerns.
func WithRelated(r ...Related) Option {
	return func(f *Finding) { f.related = append(f.related, r...) }
}

// New builds the finding for code, about subj, at loc (nil when the
// subject has no place), with the code's parameters p. It renders the
// message and hint from the code's templates.
//
// New panics when code is not registered, when subj is not the code's
// subject kind, when p names a parameter the code does not declare, gives
// one the wrong type or lacks one the code always has, or when a location
// is given for a code registered with no span, or withheld from one with a
// span. Each is a defect in the tool, never in its input, and the command
// line reports a panic as internal.error.
func New(code Code, subj Subject, loc *Location, p Params, opts ...Option) Finding {
	e, ok := lookup(code)
	if !ok {
		panic(fmt.Sprintf("finding: code %q is not registered", code))
	}
	if subj.Kind != e.Subject {
		panic(fmt.Sprintf("finding: %s is reported about a %s, not a %s", code, e.Subject, subj.Kind))
	}
	if (loc == nil) != (e.Span == SpanNone) {
		panic(fmt.Sprintf("finding: %s registers span %s, and New was given location %v", code, e.Span, loc))
	}
	if err := checkSubject(subj); err != nil {
		panic(fmt.Sprintf("finding: %s: %v", code, err))
	}
	if err := e.checkParams(p); err != nil {
		panic(fmt.Sprintf("finding: %s: %v", code, err))
	}
	f := Finding{entry: e, subject: cleanSubject(subj), params: cleanParams(p)}
	if loc != nil {
		l := cleanLocation(*loc)
		f.location = &l
	}
	for _, o := range opts {
		o(&f)
	}
	f.clean()
	if err := f.checkPointers(); err != nil {
		panic(fmt.Sprintf("finding: %s: %v", code, err))
	}
	if f.location != nil && f.location.HasPointer {
		f.ptrTokens = pointerTokens(f.location.Pointer)
	}
	f.paramsKey = canonicalForm(f.params)
	var err error
	if f.message, err = f.render(e.Message); err != nil {
		panic(fmt.Sprintf("finding: %s message: %v", code, err))
	}
	if f.hint, err = f.render(e.Hint); err != nil {
		panic(fmt.Sprintf("finding: %s hint: %v", code, err))
	}
	return f
}

// checkSubject holds a subject to ADR-006 D4 items 2 and 3: a name exactly
// for the kinds that have one, and a title only for an input or a schema.
func checkSubject(s Subject) error {
	switch s.Kind {
	case SubjectInvocation, SubjectPackage, SubjectManifest, SubjectReport:
		if s.Name != "" {
			return fmt.Errorf("a subject of kind %s takes no name", s.Kind)
		}
	default:
		if s.Name == "" {
			return fmt.Errorf("a subject of kind %s needs a name", s.Kind)
		}
	}
	if s.Title != "" && s.Kind != SubjectInput && s.Kind != SubjectSchema {
		return fmt.Errorf("a subject of kind %s takes no title", s.Kind)
	}
	return nil
}

// checkPointers reports the first pointer of f that is not an RFC 6901
// pointer: its location's and its related locations', and every schema
// pointer it names.
func (f Finding) checkPointers() error {
	var ptrs []string
	locs := []*Location{f.location}
	for i := range f.related {
		locs = append(locs, &f.related[i].Location)
	}
	for _, l := range locs {
		if l != nil && l.HasPointer {
			ptrs = append(ptrs, l.Pointer)
		}
	}
	if f.schema != nil {
		ptrs = append(ptrs, f.schema.Pointer, f.schema.KeywordLocation)
	}
	for _, v := range f.via {
		ptrs = append(ptrs, v.SchemaPointer)
		if v.Condition != nil {
			ptrs = append(ptrs, v.Condition.SchemaPointer)
		}
	}
	for _, p := range ptrs {
		if !validPointer(p) {
			return fmt.Errorf("%q is not a JSON Pointer", p)
		}
	}
	return nil
}

// validPointer reports whether p is an RFC 6901 pointer: empty, or each
// token prefixed by / with ~ only in ~0 and ~1.
func validPointer(p string) bool {
	if p == "" {
		return true
	}
	if p[0] != '/' {
		return false
	}
	for i := 0; i < len(p); i++ {
		if p[i] == '~' && (i+1 == len(p) || (p[i+1] != '0' && p[i+1] != '1')) {
			return false
		}
	}
	return true
}

// render expands a message or hint template (ADR-006 D3 item 7): {name}
// is a parameter or one of the built-ins field, input, output and
// package; {{ and }} are literal braces.
func (f Finding) render(tmpl string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(tmpl); {
		c := tmpl[i]
		switch {
		case c == '{' && strings.HasPrefix(tmpl[i:], "{{"):
			b.WriteByte('{')
			i += 2
		case c == '}' && strings.HasPrefix(tmpl[i:], "}}"):
			b.WriteByte('}')
			i += 2
		case c == '{':
			end := strings.IndexByte(tmpl[i:], '}')
			if end < 0 {
				return "", fmt.Errorf("unclosed placeholder at byte %d", i)
			}
			s, err := f.placeholder(tmpl[i+1 : i+end])
			if err != nil {
				return "", err
			}
			b.WriteString(s)
			i += end + 1
		case c == '}':
			return "", fmt.Errorf("unpaired } at byte %d", i)
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String(), nil
}

func (f Finding) placeholder(name string) (string, error) {
	switch name {
	case "package":
		return "<package>", nil
	case "field":
		return f.fieldText(), nil
	case "input":
		if (f.subject.Kind == SubjectInput || f.subject.Kind == SubjectSchema) && f.subject.Name != "" {
			return f.subject.Name, nil
		}
		return "", fmt.Errorf("{input} in a finding about a %s with no input name", f.subject.Kind)
	case "output":
		if f.subject.Kind == SubjectOutput && f.subject.Name != "" {
			return f.subject.Name, nil
		}
		return "", fmt.Errorf("{output} in a finding about a %s with no output name", f.subject.Kind)
	}
	v, ok := f.params[name]
	if !ok {
		return "", fmt.Errorf("placeholder {%s} names no parameter given", name)
	}
	return v.render(), nil
}

// fieldText is {field} (ADR-006 D9 item 3): the field's title, else its
// pointer in backticks, else the input itself.
func (f Finding) fieldText() string {
	if f.field != nil && f.field.Title != "" {
		return f.field.Title
	}
	if f.location != nil && f.location.HasPointer && f.location.Pointer != "" {
		return "`" + f.location.Pointer + "`"
	}
	if f.subject.Name != "" {
		return "the " + f.subject.Kind.String() + " `" + f.subject.Name + "`"
	}
	return "the " + f.subject.Kind.String()
}

// clean replaces every invalid UTF-8 byte in the options' strings with
// U+FFFD (ADR-006 D9 item 9), so the finding compares and encodes as
// valid text.
func (f *Finding) clean() {
	if f.schema != nil {
		s := *f.schema
		s.Entry, s.Pointer, s.Keyword, s.KeywordLocation = validUTF8(s.Entry), validUTF8(s.Pointer), validUTF8(s.Keyword), validUTF8(s.KeywordLocation)
		f.schema = &s
	}
	if f.field != nil {
		fd := Field{validUTF8(f.field.Title), validUTF8(f.field.Description)}
		f.field = &fd
	}
	via := make([]Via, len(f.via))
	for i, v := range f.via {
		v.Keyword, v.SchemaPointer, v.Title = validUTF8(v.Keyword), validUTF8(v.SchemaPointer), validUTF8(v.Title)
		if v.Condition != nil {
			c := Condition{validUTF8(v.Condition.SchemaPointer), validUTF8(v.Condition.Title)}
			v.Condition = &c
		}
		via[i] = v
	}
	f.via = via
	alts := make([]Alternative, len(f.alternatives))
	for i, a := range f.alternatives {
		a.Title = validUTF8(a.Title)
		a.Findings, a.Omitted = branchFindings(a.Findings, a.Omitted)
		alts[i] = a
	}
	f.alternatives = alts
	rel := make([]Related, len(f.related))
	for i, r := range f.related {
		rel[i] = Related{cleanLocation(r.Location), validUTF8(r.Note)}
	}
	f.related = rel
	if len(f.via) == 0 {
		f.via = nil
	}
	if len(f.alternatives) == 0 {
		f.alternatives = nil
	}
	if len(f.related) == 0 {
		f.related = nil
	}
}

// maxBranchFindings is how many findings one alternative lists (ADR-006
// D6 item 6).
const maxBranchFindings = 16

// branchFindings sorts and de-duplicates one alternative's findings within
// the branch (ADR-006 D8 item 4) and keeps the first maxBranchFindings,
// adding the rest to omitted.
func branchFindings(fs []Finding, omitted int) ([]Finding, int) {
	r := NewReport(maxBranchFindings)
	for _, f := range fs {
		r.Add(f)
	}
	return r.sorted(), omitted + r.omitted()
}

func cleanSubject(s Subject) Subject {
	return Subject{s.Kind, validUTF8(s.Name), validUTF8(s.Title)}
}

func cleanLocation(l Location) Location {
	l.File, l.Pointer = validUTF8(l.File), validUTF8(l.Pointer)
	return l
}

func cleanParams(p Params) Params {
	out := make(Params, len(p))
	for k, v := range p {
		out[k] = v.clean()
	}
	return out
}

// validUTF8 returns s with each byte that is not part of valid UTF-8
// replaced by U+FFFD.
func validUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		r, n := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && n == 1 {
			b.WriteRune(utf8.RuneError)
		} else {
			b.WriteString(s[i : i+n])
		}
		i += n
	}
	return b.String()
}

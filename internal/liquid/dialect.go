// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"errors"
	"fmt"
	"slices"

	"github.com/kottera/metais/internal/liquid/ast"
	"github.com/kottera/metais/internal/liquid/value"
)

// Dialect is the set of names a template may use beyond the grammar
// itself: the filters, the tags a caller adds to the core tags, and the
// names no template-scoped binding may take. The core tags of ADR-004 D2
// are the grammar's and every Dialect has them; include never is one. A
// Dialect is immutable: Extend returns a new one.
//
// The parser consults a Dialect for three things only: whether a filter
// exists (ADR-004 D1 item 6), the added tags and their static checks, and
// the reserved names (ADR-004 D7 item 8). Everything a filter or tag does
// is the renderer's, through the same definitions.
type Dialect struct {
	tags     map[string]TagDef
	filters  map[string]FilterDef
	reserved map[string]bool
}

// Spec is what NewDialect and Extend add: tags, filters and reserved
// names.
type Spec struct {
	Tags     []TagDef
	Filters  []FilterDef
	Reserved []string
}

// ErrDialect is returned for a Spec that names a definition twice, uses a
// name that is not an identifier, or adds a tag the grammar already has.
var ErrDialect = errors.New("invalid dialect")

// NewDialect returns the dialect of s. With an empty Spec it is the bare
// grammar: every core tag, no filter and no reserved name.
func NewDialect(s Spec) (*Dialect, error) {
	return (&Dialect{}).Extend(s)
}

// Extend returns a dialect holding d's definitions and s's. It refuses a
// name d or s already defines, a tag named as a core tag, as include or
// as an end tag ("end" followed by anything), and any name that is not an
// identifier.
func (d *Dialect) Extend(s Spec) (*Dialect, error) {
	n := &Dialect{
		tags:     make(map[string]TagDef, len(d.tags)+len(s.Tags)),
		filters:  make(map[string]FilterDef, len(d.filters)+len(s.Filters)),
		reserved: make(map[string]bool, len(d.reserved)+len(s.Reserved)),
	}
	for k, v := range d.tags {
		n.tags[k] = v
	}
	for k, v := range d.filters {
		n.filters[k] = v
	}
	for k := range d.reserved {
		n.reserved[k] = true
	}
	for _, t := range s.Tags {
		name := t.Name()
		switch {
		case !isIdentifier(name):
			return nil, fmt.Errorf("%w: tag name %q is not an identifier", ErrDialect, name)
		case slices.Contains(coreTags, name), name == "include", len(name) >= 3 && name[:3] == "end":
			return nil, fmt.Errorf("%w: tag %q is the grammar's", ErrDialect, name)
		case n.tags[name] != nil:
			return nil, fmt.Errorf("%w: tag %q defined twice", ErrDialect, name)
		}
		n.tags[name] = t
	}
	for _, f := range s.Filters {
		name := f.Name()
		switch {
		case !isIdentifier(name):
			return nil, fmt.Errorf("%w: filter name %q is not an identifier", ErrDialect, name)
		case n.filters[name] != nil:
			return nil, fmt.Errorf("%w: filter %q defined twice", ErrDialect, name)
		}
		n.filters[name] = f
	}
	for _, name := range s.Reserved {
		switch {
		case !isIdentifier(name):
			return nil, fmt.Errorf("%w: reserved name %q is not an identifier", ErrDialect, name)
		case n.reserved[name]:
			return nil, fmt.Errorf("%w: name %q reserved twice", ErrDialect, name)
		}
		n.reserved[name] = true
	}
	return n, nil
}

// Filter returns the filter named name, if the dialect defines it.
func (d *Dialect) Filter(name string) (FilterDef, bool) {
	f, ok := d.filters[name]
	return f, ok
}

// Tag returns the added tag named name, if the dialect defines it.
func (d *Dialect) Tag(name string) (TagDef, bool) {
	t, ok := d.tags[name]
	return t, ok
}

// Reserved reports whether name is reserved.
func (d *Dialect) Reserved(name string) bool { return d.reserved[name] }

// CoreTags returns the tags of ADR-004 D2 that the grammar parses, sorted:
// every one of Shopify's 5.14.0 tags but include, and the liquid tag.
func CoreTags() []string { return slices.Clone(coreTags) }

// coreTags is sorted, and holds the opening tag of every block.
var coreTags = []string{
	"#", "assign", "break", "capture", "case", "comment", "continue",
	"cycle", "decrement", "doc", "echo", "for", "if", "ifchanged",
	"increment", "liquid", "raw", "render", "tablerow", "unless",
}

// TagDef defines a tag a Dialect adds to the core tags: METAIS's embed,
// asset and provenance (ADR-004 D12 to D14), which internal/template
// defines. An added tag is never a block: it has no body and no end tag.
// Its markup is an argument list, positional arguments before keyword
// arguments, each a Primary, with optional commas between them and no
// keyword given twice: "{% asset "logo.svg", as: "path" %}". The parser
// builds an *ast.Tag of it.
//
// The contract, for both sides:
//
//   - Name is the tag's name, an identifier, and is constant.
//   - Check is called once per occurrence when the template is parsed,
//     with no data, after the markup has parsed. It refuses what is wrong
//     whatever the data: the static arguments ADR-004 D6 item 8 lists,
//     such as a non-literal name or an unknown keyword. It returns nil to
//     accept. A refusal is reported as it is given, its Source, Line, Col
//     and Offset filled in from the tag's position; the parse continues.
//   - Render is called by the renderer each time the tag executes. It
//     evaluates its arguments only through c, writes only through c, and
//     charges its work to c's Meter. It reads nothing else: no host, no
//     clock, no global state (ADR-004 D7 item 1, ADR-010 D4 item 5). It
//     returns nil or the error that stops the render.
//
// A TagDef must be safe to call from one goroutine at a time; the engine
// never calls one concurrently with itself.
type TagDef interface {
	Name() string
	Check(t *ast.Tag) *Error
	Render(c TagContext, t *ast.Tag) *Error
}

// TagContext is what the renderer gives a TagDef's Render. Wave 3's
// renderer implements it.
//
//   - Eval evaluates one of the tag's arguments in the current scope,
//     with the strictness of ADR-004 D6: an undefined name is an error,
//     never nil.
//   - Write writes b to the output exactly as given, never escaped
//     (ADR-004 D9 item 4), counting its bytes against the output's limits.
//   - Meter is the render's meter.
type TagContext interface {
	Eval(e ast.Primary) (value.Value, *Error)
	Write(b []byte) *Error
	Meter() Meter
}

// FilterDef defines a filter: every filter of the standard catalog
// (ADR-010 D4 item 2) and METAIS's own (ADR-004 D10), which Wave 3's
// filter catalog and internal/template implement.
//
// The contract, for both sides:
//
//   - Name is the filter's name, an identifier, and is constant.
//   - Check is called once per application when the template is parsed,
//     with no data. It refuses an application that is wrong whatever the
//     data: an argument count the filter never takes, a keyword it does not
//     know, or a literal argument of the wrong type. It returns nil to
//     accept. A refusal is reported as it is given, located at the
//     application, and the parse continues. Anything that depends on a
//     value is Apply's to refuse.
//   - Apply is called by the renderer for each application, with the
//     filter's input and its evaluated arguments in source order. It
//     returns the result, or an error that stops the render. It charges
//     its work to c's Meter as ADR-009 D11 item 2's table says, before
//     doing work the charge would refuse, and reads nothing but its input,
//     its arguments and c (ADR-004 D7 item 1).
//
// A FilterDef must be safe to call from one goroutine at a time; the
// engine never calls one concurrently with itself.
type FilterDef interface {
	Name() string
	Check(f *ast.Filter) *Error
	Apply(c FilterContext, in value.Value, args []Arg) (value.Value, *Error)
}

// Arg is one evaluated argument of a filter. Name is empty for a
// positional argument.
type Arg struct {
	Name  string
	Value value.Value
}

// FilterContext is what the renderer gives a FilterDef's Apply. Wave 3's
// renderer implements it. Meter is the render's meter.
type FilterContext interface {
	Meter() Meter
}

// Work is a kind of work a Meter counts (ADR-010 D4 item 4, ADR-009 D11).
type Work uint8

// The kinds of work.
const (
	// WorkUnits is ADR-009's render-work: units charged per node
	// evaluated, filter application and operator, and per size of their
	// operands, as ADR-009 D11 item 2 defines them.
	WorkUnits Work = iota + 1
	// WorkOutputBytes is bytes written to the output (output-size).
	WorkOutputBytes
	// WorkHeldBytes is bytes held by strings built by assign, capture and
	// filters (assign-bytes, value-size).
	WorkHeldBytes
	// WorkIterations is loop iterations, charged for a range before it is
	// materialized (loop-iterations).
	WorkIterations
	// WorkPartials is partials rendered (partial-depth is the renderer's
	// own stack, not a count).
	WorkPartials
)

// Meter counts a render's work against its limits. It is a counter, never
// a clock: the same template and data exceed the same limit at the same
// node on every machine (ADR-010 D4 item 4, R10 item 2). Wave 3's
// renderer implements it.
//
// The contract, for both sides:
//
//   - Charge adds n units of w and returns nil while the total is within
//     w's limit. The first charge that would exceed it returns a CodeLimit
//     error whose Args "limit" names the limit as ADR-009 does; the total
//     is not raised past the limit, and every later Charge of w refuses
//     too. n is never negative.
//   - Used returns the total charged so far of w, so a test asserts the
//     count rather than a time (ADR-010 D14 item 7).
//
// The renderer charges every step it takes; a FilterDef or TagDef charges
// the work its own definition states. A Meter must be safe to call from
// one goroutine at a time.
type Meter interface {
	Charge(w Work, n int64) *Error
	Used(w Work) int64
}

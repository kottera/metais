// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package manifest is the manifest's model and its rules (ADR-002 D4 to D7
// and D10): the version gate, the manifest rules the manifest schema does
// not express, and the typed model a package's later layers read.
//
// The package reads a value tree another layer has already parsed, under
// ADR-002 D2's YAML profile or D3's JSON profile, and never bytes. It does
// not call the JSON Schema validator: validating the manifest against
// spec/schemas/1.0/manifest.schema.json, which internal/specdata embeds, is
// the loader's step, and so is reporting what it finds. The loader composes
// the steps in ADR-002 D11's order:
//
//  1. Gate, whose refusal ends processing (D11 step 4);
//  2. the schema of the declared minor and Check, in one pass, reporting
//     every finding of both (steps 5 and 6);
//  3. Build, which turns the tree both accepted into a Manifest (step 7).
//
// Check runs Gate first and reports nothing else when it refuses, and
// Build re-judges every rule before it returns a model, the gate first, so
// neither can be misused into judging a manifest by the wrong rules or
// handing on a model that breaks one.
package manifest

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kottera/metais/internal/liquid/value"
)

// Manifest is a package's manifest under METAIS 1.0 (ADR-002 D4). Text
// members hold the text the manifest writes; an optional member it omits
// is "" (no optional text member may be empty) or nil.
type Manifest struct {
	// Metais is the specification version the package declares.
	Metais Spec
	// Name is the package's name (D5).
	Name string
	// Version is the package's own semantic version (D6).
	Version string
	// Title is a line of 1 to 200 characters.
	Title string
	// Description is prose of 1 to 1000 characters.
	Description string
	// Guide is the supplier's guide, CommonMark prose of 1 to 16 384
	// characters.
	Guide string
	// Authors is nil when the manifest names none.
	Authors []Author
	// License is an SPDX license expression, or "".
	License string
	// Homepage is an absolute http or https URI, or "".
	Homepage string
	// Inputs is every input slot, in manifest order (ADR-003 D1).
	Inputs []Input
	// InputConstraints is the package path of the cross-input schema
	// (ADR-003 D6 item 2), or "".
	InputConstraints string
	// Outputs is every top-level output declaration, in manifest order
	// (ADR-005 D1).
	Outputs []Output
}

// Author is one member of `authors` (D4 item 10).
type Author struct {
	Name  string
	Email string // "" when omitted
	URL   string // "" when omitted
}

// Form is a file form an input may be supplied in (ADR-003 D1 item 2).
type Form string

const (
	FormJSON     Form = "json"
	FormYAML     Form = "yaml"
	FormCSV      Form = "csv"
	FormText     Form = "text"
	FormMarkdown Form = "markdown"
)

// Input is one input slot (ADR-003 D1). The model applies the defaults of
// ADR-003 D1 item 2's table, which are the specification's: the canonical
// manifest carries only what the author wrote (D1 item 7), and is made
// from the value tree, never from this model.
type Input struct {
	// Pointer is the slot's JSON Pointer in the manifest, /inputs/<i>.
	Pointer string
	Name    string
	Title   string
	// Description is "" when omitted.
	Description string
	// Schema is the package path of the input's schema file.
	Schema string
	// Meaning is the package path of the meaning document, or "".
	Meaning string
	// Forms is the forms written, or json and yaml when omitted.
	Forms []Form
	// Delimiter is the CSV delimiter when Forms contains csv: the one
	// written, or "," when omitted. It is "" otherwise.
	Delimiter string
	// Collection is false when omitted.
	Collection bool
	// Required is true when omitted.
	Required bool
}

// OutputForm is the form of an output declaration, named by its one
// source key (ADR-005 D1 item 2).
type OutputForm string

const (
	OutputTemplate  OutputForm = "template"
	OutputAsset     OutputForm = "asset"
	OutputMirror    OutputForm = "mirror"
	OutputContainer OutputForm = "container"
)

// Output is one output declaration, at the top level or as a container's
// entry (ADR-005 D1 to D9). A field its form does not allow is its zero
// value.
type Output struct {
	// Pointer is the declaration's JSON Pointer in the manifest.
	Pointer string
	// Form is the declaration's form.
	Form OutputForm
	// ID is required at the top level, and "" on an entry that omits it.
	ID string
	// Source is the value of the source key: the template's path relative
	// to templates/, the asset's relative to assets/, or the mirror's
	// directory relative to templates/. It is "" for a container.
	Source string
	// Path is the destination template, or "" when omitted (ADR-005 D4
	// item 4 gives the default).
	Path string
	// When and ForEach are Liquid expressions, "" when omitted.
	When    string
	ForEach string
	// Escape is none or html; none when omitted.
	Escape string
	// Newline is as-written or crlf; as-written when omitted.
	Newline string
	// Validate is the package path of the output's schema, or "".
	Validate string
	// Kind is the output kind written, or "" to derive it from the path
	// (ADR-005 D7).
	Kind string
	// Executable, First and Stored are false when omitted.
	Executable bool
	First      bool
	Stored     bool
	// Container is the container's shape, for a container only.
	Container *Container
}

// Container is a zip container's shape (ADR-005 D8 item 1).
type Container struct {
	// Format is zip, written or not.
	Format string
	// Profile is ooxml, odf or epub, or "" (ADR-005 D9).
	Profile string
	// Mimetype is the odf profile's media type, and "" for any other.
	Mimetype string
	// Entries is every entry, in declaration order.
	Entries []Output
}

// ErrInvalid is wrapped by every refusal of Build.
var ErrInvalid = errors.New("manifest: the value tree is not a valid manifest")

// Violation is one rule a value tree breaks: where, and which rule. Rule is
// the name of the rule; for one of Gate's or Check's it is the code they
// report.
type Violation struct {
	Pointer string
	Rule    string
}

// InvalidError is Build's refusal: every rule the tree breaks.
type InvalidError struct {
	Violations []Violation
}

func (e *InvalidError) Error() string {
	var b strings.Builder
	b.WriteString(ErrInvalid.Error())
	for i, v := range e.Violations {
		if i == 8 {
			fmt.Fprintf(&b, "; and %d more", len(e.Violations)-i)
			break
		}
		ptr := v.Pointer
		if ptr == "" {
			ptr = "the root"
		}
		fmt.Fprintf(&b, "; %s: %s", ptr, v.Rule)
	}
	return b.String()
}

func (e *InvalidError) Unwrap() error { return ErrInvalid }

// Build returns the manifest root holds, judged by every rule of ADR-002 D4
// to D7 and of the inputs and outputs sections (ADR-003 D1, ADR-005 D1,
// D2 and D7 to D9) under the specification version root declares.
//
// It judges the version gate first, and when the gate refuses it judges
// nothing else (D7 item 8). Otherwise it judges every other rule,
// those the manifest schema states and those Check reports alike, and
// refuses with an *InvalidError, wrapping ErrInvalid, naming each one the
// tree breaks. A tree the schema of its declared minor accepts and Check
// finds nothing in is never refused.
func Build(root value.Value) (*Manifest, error) {
	spec, gate := Gate(root, Source{})
	if len(gate) > 0 {
		ptr, _ := gate[0].Location()
		return nil, &InvalidError{[]Violation{{ptr.Pointer, string(gate[0].Code())}}}
	}
	switch spec {
	case Spec{1, 0}:
		d := &decoder{}
		m := d.manifest10(root, spec)
		for _, f := range Check(root, Source{}) {
			loc, _ := f.Location()
			d.fail(loc.Pointer, string(f.Code()))
		}
		if len(d.violations) > 0 {
			return nil, &InvalidError{d.violations}
		}
		return m, nil
	}
	// Gate accepts only a version this tool implements, and each one has
	// its rules here.
	panic(fmt.Sprintf("manifest: no rules for METAIS %s", spec))
}

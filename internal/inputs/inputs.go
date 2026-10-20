// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package inputs is the inputs layer (ADR-003 D8 to D22, ADR-010 D2): it
// reads the inputs a caller supplies to an opened package, assembles them
// into the inputs object, applies the schemas' defaults, validates the
// result against the combined schema, and gives each declared input its
// record.
//
// A Layer is built from what ADR-003 D22 step 1 produced when the package
// was opened: its manifest and its compiled schemas. A Supply is one
// command's inputs, given by its supply flags (ADR-008 D7) through one
// entry point each: ReadDataDir for --data, AddInputsJSON for each
// --inputs-json, AddSet for each --set and AddSetJSON for each --set-json,
// in command-line order. Each parses what it is given at once and records
// every finding (D22 step 2). Validate then assembles the inputs object in
// D18's order, applies D19's defaults, validates it (D22 steps 3 and 4)
// and returns every finding of the supply together (step 5), unsorted;
// the command's finding.Report orders, de-duplicates and caps them.
//
// The forms are read by Wave 2's readers: internal/jsonx, internal/yamlx,
// internal/inputs/csv and internal/markdown, with each value coerced by
// the static walk of internal/schema (D14). The data directory is read
// through internal/hostfs's read side, and nothing else of the host is
// read: no environment variable, path or clock is a source of any value or
// default (ADR-009 D7).
package inputs

import (
	"fmt"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/schema"
)

// Package is what the inputs layer reads of an opened package: its
// manifest and its schemas, as ADR-003 D22 step 1 read and compiled them.
type Package struct {
	// Manifest is the package's manifest.
	Manifest *manifest.Manifest
	// Compiler holds every schema file of the package, compiled. It is
	// the schemafail.Schemas the failures are mapped against.
	Compiler *schema.Compiler
	// Combined is the combined schema schema.Bundle derived from the
	// manifest and the schema files (ADR-003 D7).
	Combined value.Value
	// Schema is Combined, compiled by Compiler.Combined: what validation
	// evaluates (ADR-003 D7 item 5).
	Schema *schema.Schema
}

// Layer is the inputs layer of one opened package.
type Layer struct {
	pkg    Package
	inputs []manifest.Input
	byName map[string]int
	walker *walker
}

// New returns the inputs layer of p. Its error is a Package whose parts
// do not belong together, which is a defect in its caller.
func New(p Package) (*Layer, error) {
	if p.Manifest == nil || p.Compiler == nil || p.Combined == nil || p.Schema == nil {
		return nil, fmt.Errorf("inputs: an incomplete package")
	}
	w, err := newWalker(p.Combined)
	if err != nil {
		return nil, err
	}
	l := &Layer{pkg: p, inputs: p.Manifest.Inputs, byName: map[string]int{}, walker: w}
	for i, in := range l.inputs {
		l.byName[in.Name] = i
	}
	return l, nil
}

// subject is the finding subject of the declared input name, with its
// slot title (ADR-006 D4 items 2 and 3).
func (l *Layer) subject(name string) finding.Subject {
	return finding.Subject{Kind: finding.SubjectInput, Name: name, Title: l.inputs[l.byName[name]].Title}
}

// reader is a reader of in's files under lim, counting the values it
// decodes on values.
func (l *Layer) reader(in manifest.Input, lim limits.Set, values *limits.Counter) *reader {
	return &reader{in: in, subj: l.subject(in.Name), lim: lim, values: values, walker: l.walker}
}

// Supply is one command's supplied inputs, before assembly.
type Supply struct {
	l   *Layer
	lim limits.Set

	// total is the input bytes read so far, against inputs-total.
	total int64
	// values counts the values every reader of the command decodes,
	// against input-values (ADR-009 Amendment A4 item 1).
	values   *limits.Counter
	findings []finding.Finding

	dataDir bool
	files   map[string]*fileInput // by input name
	// locPrefix is the package entry path of the directory ReadPackageDir
	// reads, whose findings are located with origin example; "" for a
	// data directory on the host.
	locPrefix string

	jsonArgs []*jsonArg
	sets     []*pathSet
	nSets    int // the --set and --set-json flags given so far

	// broken is each input an argument for it did not parse in: an
	// --inputs-json member's value or a --set-json value.
	broken map[string]bool
	// incomplete is set when an --inputs-json argument did not parse or
	// was not an object, so which inputs it would have supplied is not
	// known.
	incomplete bool

	validated bool
}

// fileInput is what the data directory gave one input.
type fileInput struct {
	file       string // its entry, relative to the data directory
	collection bool
	listed     bool     // a collection's directory was listed
	value      *node    // an input's file's value
	items      []*item  // a collection's items, in order
	partial    *partial // a CSV file validated in part
	raw        string   // an input's file's bytes
	// broken is set when any finding refused the input's entry or any
	// item of it: no path set is applied into it (ADR-003 D18 item 4).
	broken bool
}

// contributes reports whether the entry gives its input a value: a file
// that parsed, or that a CSV reader read in part, or a collection's
// directory, whose items that did not parse are left out (ADR-006 D7 item
// 4).
func (fi *fileInput) contributes() bool {
	if fi.collection {
		return fi.listed
	}
	return fi.value != nil
}

// item is one file of a collection (ADR-003 D15).
type item struct {
	name  string // the file name, in NFC
	stem  string // name without its last extension
	form  manifest.Form
	file  string // the entry, relative to the data directory
	index int    // the item's index in the collection's order
	value *node
	raw   string
	ok    bool // the item parsed
}

// NewSupply returns an empty supply for one command, under lim: the
// command's limits, the same the package was opened under, whose
// regex-work the compiled schemas' matching already counts against.
func (l *Layer) NewSupply(lim limits.Set) *Supply {
	return &Supply{l: l, lim: lim, values: lim.Counter(limits.InputValues), files: map[string]*fileInput{}, broken: map[string]bool{}}
}

func (s *Supply) add(fs ...finding.Finding) { s.findings = append(s.findings, fs...) }

// charge counts n more bytes read against inputs-total, about subj at
// loc, and reports whether they are within it.
func (s *Supply) charge(n int64, subj finding.Subject, loc finding.Location) bool {
	if s.total+n > s.lim.Value(limits.InputsTotal) {
		s.add(s.lim.Exceeded(limits.InputsTotal, subj, loc, 0, false))
		return false
	}
	s.total += n
	return true
}

func itoa(i int) string { return fmt.Sprint(i) }

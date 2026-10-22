// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package xref is check's cross-reference of templates against schemas
// (ADR-007 D3): a static analysis, needing no data, of every variable path
// a package's templates and manifest expressions read, resolved against
// the input schemas with an exact definition of "declared".
//
// The analysis walks the engine's own syntax tree, internal/liquid/ast,
// and has no parser of its own (D3 item 1): templates arrive parsed, and
// the manifest's path, when and for_each are parsed by liquid.Parse,
// ParseCondition and ParseValue under the caller's dialect. The schemas
// are the input schema resources as schema.Bundle combines them, read
// through schema.Walker.
//
// Each expression is evaluated to a set of abstract alternatives (D3 item
// 2): a schema node, open, unknown, a scalar of a kind, a record of fixed
// fields, or an array of an element set. A segment is an error only when
// every alternative rejects it, and open and unknown reject nothing. The
// analysis follows no control flow: every branch is read, and a local's
// value is the union of everything ever assigned to it.
//
// The analysis is bounded by two fixed constants, MaxContexts and
// MaxSegments (D3 item 13), which are not ADR-009 limits: reaching one
// stops the analysis with template.xref.budget_exhausted, a warning, and
// refuses nothing.
package xref

import (
	"errors"
	"fmt"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/liquid/ast"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/manifest"
)

// The analysis's bounds (ADR-007 D3 item 13). They are constants of the
// analysis, never ADR-009 limits, and no caller can change them.
const (
	// MaxContexts is how many template-context analyses the analysis
	// makes: a template, a partial or a manifest expression analysed in
	// one context counts one.
	MaxContexts = 10_000
	// MaxSegments is how many path segments, property-name filter
	// arguments included, the analysis resolves.
	MaxSegments = 1_000_000
)

// TemplatesDir is the directory every template lies under (ADR-001 D5).
const TemplatesDir = "templates/"

// Package is what the analysis reads of a package.
type Package struct {
	// Manifest is the package's manifest, which Build has accepted.
	Manifest *manifest.Manifest
	// Positions locates values in the manifest, so a finding about a
	// manifest expression has a line; nil when the reader kept none.
	Positions finding.Positions
	// Schema is the combined schema of the package's inputs, as
	// schema.Bundle returns it. It is ignored when NoSchemas is set.
	Schema value.Object
	// NoSchemas reports that the package's schemas could not be read:
	// a schema file did not compile, so there is no combined schema.
	// Every input's value is then unknown (D3 item 2): nothing read
	// through it is checked, and every other reference is.
	NoSchemas bool
	// Templates is every entry under templates/, by its path relative to
	// templates/. Root is the template's syntax tree, or nil when it did
	// not parse; its parse findings are the caller's to report.
	Templates []Template
	// Assets is every entry under assets/, by its path relative to
	// assets/.
	Assets []string
	// Dialect parses the manifest's expressions: the dialect the
	// templates were parsed under.
	Dialect *liquid.Dialect
	// ParseLimits bounds those parses, as the templates' were.
	ParseLimits liquid.ParseLimits
	// Limits holds the regular-expression limits patternProperties'
	// patterns are compiled under, as the schema compiler compiled them.
	Limits limits.Set
}

// Template is one template of a package.
type Template struct {
	// Path is the template's path relative to templates/.
	Path string
	// Root is its syntax tree, nil when it did not parse.
	Root *ast.Block
}

// Result is the analysis's report.
type Result struct {
	// Findings is every finding, in the order the analysis made them.
	Findings []finding.Finding
	// Contexts is how many template-context analyses were made, at most
	// MaxContexts.
	Contexts int
	// Segments is how many segments were resolved, at most MaxSegments.
	Segments int
}

// ErrPackage is wrapped by Analyze's refusal of a Package that breaks a
// rule an earlier layer reports: a combined schema the walker cannot
// index, or an input it has no schema for. Meeting one means that layer
// let a package through.
var ErrPackage = errors.New("xref: the package is not one the earlier layers accept")

// Analyze cross-references p's templates and manifest expressions against
// its input schemas (ADR-007 D3) and reports what it finds.
func Analyze(p Package) (Result, error) {
	if p.Manifest == nil || p.Dialect == nil {
		return Result{}, fmt.Errorf("%w: no manifest or no dialect", ErrPackage)
	}
	a, err := newAnalyzer(&p)
	if err != nil {
		return Result{}, err
	}
	a.run()
	return Result{Findings: a.findings, Contexts: a.contexts, Segments: a.segments}, nil
}

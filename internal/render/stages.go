// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/loader"
	"metais.org/metais/internal/manifest"
	"metais.org/metais/internal/regexprofile"
	"metais.org/metais/internal/schema"
	"metais.org/metais/internal/template"
)

// Schemas are the output validation schemas of a package's declarations,
// by the name validate gives.
type Schemas struct {
	byName map[string]*schema.Compiler
}

// OutputSchemas compiles every output validation schema of pkg's
// declarations (ADR-005 D11 item 3), charged to the command's matching
// budget. A schema that does not compile refuses the render with its
// findings.
func OutputSchemas(pkg *loader.Package, lim limits.Set) (*Schemas, []finding.Finding) {
	var meter *regexprofile.Meter
	if pkg.Schemas != nil {
		meter = pkg.Schemas.Meter()
	}
	byName, fs := loader.OutputSchemas(pkg.Tree, pkg.Manifest.Outputs, lim, meter)
	return &Schemas{byName: byName}, fs
}

// Staging is what the stages of a render read: the package, its limits
// and its output validation schemas, and the renderer whose meter charged
// the bytes each file rendered.
type Staging struct {
	pkg     *loader.Package
	lim     limits.Set
	schemas *Schemas
	r       *template.Renderer
}

// Stage is a transform of a rendered file's bytes, in ADR-005 D13's phase
// 4 (ADR-010 Amendment A7 item 6): Applies reports whether it runs on a
// file the declaration d plans, and Run runs it, reading the file's bytes,
// its declaration and the package and reaching no host. A stage changes
// the bytes only through Staging.SetBytes.
type Stage struct {
	Name    string
	Applies func(d manifest.Output) bool
	Run     func(c *Staging, f *File) []finding.Finding
}

// stages are the stages, in their normative order: D10's CRLF conversion,
// then D11's validation, which reads the final bytes.
var stages = []Stage{
	crlfStage,
	validateStage,
}

// runStages is ADR-005 D13 item 4's transforms of f, a file that
// rendered, top-level or a container's entry: every stage that applies
// runs, in order, and every finding of each is reported (D11 item 4); a
// stage's findings do not stop the stages after it, but a file a stage
// left unrendered goes no further. It runs before a container holding f
// is measured, so the container holds the bytes transformed.
func runStages(c *Staging, f *File) []finding.Finding {
	var fs []finding.Finding
	for _, s := range stages {
		if f.failed {
			break
		}
		if s.Applies(f.decl) {
			fs = append(fs, s.Run(c, f)...)
		}
	}
	return fs
}

// SetBytes makes b f's bytes, charging the change in their length against
// output-size and output-total on the meter that charged f's rendered
// bytes (ADR-010 Amendment A7 item 8), so the bound holds on the bytes
// the render writes (ADR-005 D13 item 4, ADR-009 D3). A breach is f's
// limit.exceeded.output, at its path, a container's entry by its
// container's path and its own: f did not render, and no later stage
// runs on it.
func (c *Staging) SetBytes(f *File, b []byte) []finding.Finding {
	o := template.Output{ID: f.owner, Path: f.path, In: f.in}
	if breach := c.r.ChargeChange(o, int64(len(f.bytes)), int64(len(b))); breach != nil {
		f.bytes, f.failed = nil, true
		return []finding.Finding{*breach}
	}
	f.bytes = b
	return nil
}

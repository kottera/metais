// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/regexprofile"
	"github.com/kottera/metais/internal/schema"
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
// and its output validation schemas.
type Staging struct {
	pkg     *loader.Package
	lim     limits.Set
	schemas *Schemas
}

// runStages is ADR-005 D13 item 4's transforms of f, a file that
// rendered, top-level or a container's entry: D10's CRLF conversion, then
// D11's validation of its final bytes against the declaration's schema,
// every finding of which is reported (D11 item 4). It runs before a
// container holding f is measured, so the container holds the bytes
// transformed.
func runStages(c *Staging, f *File) []finding.Finding {
	if f.decl.Newline == "crlf" {
		f.bytes = crlf(f.bytes)
	}
	if f.decl.Validate == "" {
		return nil
	}
	return validateOutput(c.pkg, f, c.schemas.byName, c.lim)
}

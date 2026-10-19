// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package template is the template environment of ADR-004 over the engine
// of internal/liquid: the dialect a package's templates are parsed in, the
// globals a render binds (D7), the partials it loads (D8), the limits it
// holds (ADR-009 D11), and the mapping of every engine error to its
// finding, located at a template path and a line (D16).
//
// Templates is a package's templates/ entries, each parsed once, read in
// place from the package's tree; Parse is the one parse a template is
// given, which render and check share (ADR-007 D2 step 6). Bindings builds
// the globals from the validated inputs and the manifest and from nothing
// else (ADR-009 D13): a closed object of plain values. A Renderer renders
// one command's outputs under one meter.
package template

import (
	"github.com/kottera/metais/internal/liquid"
)

// The globals of ADR-004 D7, and the loop objects Liquid binds, which no
// template-scoped binding may take (D7 item 8).
var reserved = []string{
	"fan_out", "forloop", "inputs", "item", "now", "output", "package", "sources", "tablerowloop",
}

// dialect is Dialect's value.
var dialect = func() *liquid.Dialect {
	d, err := liquid.Core().Extend(liquid.Spec{Reserved: reserved})
	if err != nil {
		panic("template: " + err.Error())
	}
	return d
}()

// Dialect is the dialect a package's templates are parsed and rendered in:
// the standard catalog of internal/liquid, with the globals' names
// reserved (ADR-004 D1, D7 item 8).
func Dialect() *liquid.Dialect { return dialect }

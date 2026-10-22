// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import "metais.org/metais/internal/manifest"

// Form is a declaration form: how one evaluation of a declaration expands
// into planned files, and nothing else (ADR-010 A7 item 4). The planner
// evaluates when and for_each, renders the path and holds the outputs
// limit for every form; a form never does.
type Form interface {
	// DefaultPath is the path of a declaration that gives none (ADR-005
	// D4 item 4).
	DefaultPath(d manifest.Output) string
	// Expand is the files of the evaluation at, whose path is planned,
	// inside depth containers (D5, D6, D8), each recorded in p's plan and
	// given its producer.
	Expand(p *Planner, at *File, depth int) []*File
}

// formOf is the form of a declaration: the table of forms.
func formOf(form manifest.OutputForm) Form {
	switch form {
	case manifest.OutputTemplate:
		return templateForm{}
	case manifest.OutputAsset:
		return assetForm{}
	case manifest.OutputMirror:
		return mirrorForm{}
	case manifest.OutputContainer:
		return containerForm{}
	}
	panic("render: a declaration of no form: " + string(form))
}

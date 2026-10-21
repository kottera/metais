// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"strings"

	"github.com/kottera/metais/internal/manifest"
)

// templateForm is a template output's (ADR-005 D4): each evaluation is
// one file, rendered from the template it names.
type templateForm struct{}

// DefaultPath is the template's name with a final .liquid removed.
func (templateForm) DefaultPath(d manifest.Output) string {
	return strings.TrimSuffix(d.Source, ".liquid")
}

func (templateForm) Expand(p *Planner, at *File, _ int) []*File {
	at.template, at.producer = at.decl.Source, producerOf(byRendering)
	return p.plan(at)
}

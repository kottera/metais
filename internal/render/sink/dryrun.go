// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sink

import (
	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/render"
)

// DryRun is s as --dry-run runs it (ADR-005 D16 item 1): every check of
// s's phase 3, its refusals and its statuses, and never its emission, so
// nothing is written.
func DryRun(s Sink) Sink { return dryRun{s} }

type dryRun struct{ s Sink }

func (d dryRun) Check(paths []string) []finding.Finding { return d.s.Check(paths) }
func (d dryRun) Status(path string) string              { return d.s.Status(path) }
func (d dryRun) Close()                                 { d.s.Close() }

func (dryRun) Emit([]*render.File) Emission { return Emission{} }

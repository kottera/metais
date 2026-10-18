// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

// Report gathers one command's findings. This first form keeps them in
// the order they were added.
type Report struct {
	max int
	fs  []Finding
}

// NewReport returns an empty report that lists at most maxFindings.
func NewReport(maxFindings int) *Report { return &Report{max: maxFindings} }

// Add adds f to the report.
func (r *Report) Add(f Finding) { r.fs = append(r.fs, f) }

// Findings returns the findings in the order they were added.
func (r *Report) Findings() []Finding { return append([]Finding{}, r.fs...) }

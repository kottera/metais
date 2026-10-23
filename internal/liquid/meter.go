// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import "fmt"

// MeterLimits are the bounds a meter from NewMeter refuses at, one per
// kind of work. The values are the caller's — internal/template passes
// ADR-009's — and this package knows no other. Zero means no bound.
//
// WorkPartials has no limit of its own and never refuses: it is internal
// bookkeeping, not a charge of ADR-009 D11 (Amendment A10 item 5).
// ADR-009 bounds partials by partial-depth, which the renderer holds, and
// by render-work, which every render tag is charged. The meter counts
// them all the same, so a test can read how many were entered.
type MeterLimits struct {
	// RenderWork bounds WorkUnits (ADR-009 render-work).
	RenderWork int64
	// OutputSize bounds WorkOutputBytes (ADR-009 output-size).
	OutputSize int64
	// AssignBytes bounds WorkHeldBytes (ADR-009 assign-bytes).
	AssignBytes int64
	// LoopIterations bounds WorkIterations (ADR-009 loop-iterations).
	LoopIterations int64
}

// NewMeter returns a Meter that counts each kind of work against l. It
// keeps counts and nothing else: it reads no clock, so the same work is
// refused at the same charge on every machine (ADR-010 D4 item 4).
func NewMeter(l MeterLimits) Meter {
	return &counter{limits: [workKinds]int64{
		WorkUnits:       l.RenderWork,
		WorkOutputBytes: l.OutputSize,
		WorkHeldBytes:   l.AssignBytes,
		WorkIterations:  l.LoopIterations,
	}}
}

// workKinds is one more than the largest Work.
const workKinds = WorkPartials + 1

// workLimitNames names each kind's limit as ADR-009 does.
var workLimitNames = [workKinds]string{
	WorkUnits:       "render-work",
	WorkOutputBytes: "output-size",
	WorkHeldBytes:   "assign-bytes",
	WorkIterations:  "loop-iterations",
}

// counter is NewMeter's Meter.
type counter struct {
	limits [workKinds]int64
	used   [workKinds]int64
	// over records a kind whose limit a charge has exceeded: every later
	// charge of it refuses too.
	over [workKinds]bool
}

func (c *counter) Charge(w Work, n int64) *Error {
	if w == 0 || w >= workKinds {
		return &Error{Code: CodeInternal, Detail: fmt.Sprintf("charge of unknown work %d", w)}
	}
	if n < 0 {
		return &Error{Code: CodeInternal, Detail: fmt.Sprintf("negative charge %d of %s", n, workLimitNames[w])}
	}
	lim := c.limits[w]
	if lim <= 0 {
		c.used[w] = saturatingAdd(c.used[w], n)
		return nil
	}
	if c.over[w] || n > lim-c.used[w] {
		c.over[w] = true
		return &Error{Code: CodeLimit, Args: map[string]string{"limit": workLimitNames[w]},
			Detail: fmt.Sprintf("%s: %d charged of %d, and %d more refused", workLimitNames[w], c.used[w], lim, n)}
	}
	c.used[w] += n
	return nil
}

func (c *counter) Used(w Work) int64 {
	if w == 0 || w >= workKinds {
		return 0
	}
	return c.used[w]
}

// saturatingAdd adds two non-negative counts, stopping at the largest
// int64 rather than wrapping.
func saturatingAdd(a, b int64) int64 {
	if b > 1<<63-1-a {
		return 1<<63 - 1
	}
	return a + b
}

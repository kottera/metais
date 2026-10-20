// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid_test

import (
	"testing"
	"time"
)

// cpuSpan is the process's CPU time at the start of a benchmark's loop.
type cpuSpan struct {
	at time.Duration
	ok bool
}

func startCPU() cpuSpan {
	at, ok := selfCPU()
	return cpuSpan{at, ok}
}

// report reports the CPU time each iteration of b's loop took, as
// cpu-ns/op, where the platform measures it: the unit of ADR-010 D15
// item 2's budgets.
func (s cpuSpan) report(b *testing.B) {
	if now, ok := selfCPU(); ok && s.ok && b.N > 0 {
		b.ReportMetric(float64(now-s.at)/float64(b.N), "cpu-ns/op")
	}
}

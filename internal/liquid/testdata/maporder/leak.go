// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package leak is TestNoMapIterationControl's fixture: it iterates Go maps
// in each way TestNoMapIteration refuses, and once in the way it allows.
// It is under testdata, so it is never built.
package leak

import (
	"maps"
	"slices"
)

type table map[string]int

// Leak iterates maps.
func Leak(m map[string]int, t table) int {
	n := 0
	for k := range m {
		n += len(k)
	}
	for range t {
		n++
	}
	for k := range maps.Keys(m) {
		n += len(k)
	}
	n += len(slices.Sorted(maps.Keys(m)))
	return n
}

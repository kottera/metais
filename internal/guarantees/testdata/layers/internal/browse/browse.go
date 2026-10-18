// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package browse breaks single-entry-point twice and keeps its exception
// once.
package browse

import "github.com/kottera/metais/internal/hostfs"

// Extract is the single entry point, which may write.
func Extract() error { return hostfs.Write() }

// List must not reach hostfs.
func List() error {
	return hostfs.Write()
}

// T has a method named Extract, which is not the entry point; the rule
// reads top-level functions only, so it is flagged as T.Extract.
type T struct{}

// Extract is not the entry point.
func (T) Extract() error { return hostfs.Write() }

var write = hostfs.Write

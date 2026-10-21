// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package sink is a package of its own, which internal/render's
// single-entry-point rule does not bind: it may write.
package sink

import "github.com/kottera/metais/internal/hostfs"

// Emit writes.
func Emit() error { return hostfs.Write() }

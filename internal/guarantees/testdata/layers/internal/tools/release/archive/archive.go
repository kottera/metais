// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package archive, beneath the release builder, breaks only-referrers once:
// only the builder's main package is named beside cmd/metais. The binary
// importing it breaks never-depends.
package archive

import "os"

// Write writes to the standard output it was not given.
func Write() {
	_, _ = os.Stdout.Write(nil)
}

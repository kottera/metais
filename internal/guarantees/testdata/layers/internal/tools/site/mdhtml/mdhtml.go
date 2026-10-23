// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package mdhtml, beneath the site's build step, may import the Markdown
// library, and breaks only-referrers once: only the build step's main
// package is named beside cmd/metais. The binary importing it breaks
// never-depends.
package mdhtml

import (
	"os"

	"github.com/yuin/goldmark"
)

// Convert writes to the standard output it was not given.
func Convert() {
	_ = goldmark.New()
	_, _ = os.Stdout.Write(nil)
}

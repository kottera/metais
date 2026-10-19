// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package author keeps only-importers' exception for the cross-reference:
// check imports it. It keeps never-depends' exception too: only ooxml is
// held from the manifest, so author, which imports both, may depend on it.
package author

import (
	"github.com/kottera/metais/internal/author/ooxml"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/xref"
)

// Check stands for check.
var Check = xref.Analyze

// Read reads the manifest and scans.
var Read = func(manifest.Manifest) { ooxml.Scan() }

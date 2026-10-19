// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"archive/zip"

	"golang.org/x/text/cases"
)

// Fold is a second case folding, outside internal/unitext, and a zip
// reader outside internal/zipfmt: both break only-importers.
var (
	Fold   = cases.Fold
	Reader = zip.NewReader
)

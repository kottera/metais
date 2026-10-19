// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package author keeps only-importers' exception for the cross-reference:
// check imports it.
package author

import "github.com/kottera/metais/internal/xref"

// Check stands for check.
var Check = xref.Analyze

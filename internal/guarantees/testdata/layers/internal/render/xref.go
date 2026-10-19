// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import "github.com/kottera/metais/internal/xref"

// Check breaks only-importers: the cross-reference reached around check.
var Check = xref.Analyze

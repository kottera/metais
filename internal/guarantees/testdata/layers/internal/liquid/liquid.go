// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package liquid breaks standard-library-only: it imports another
// internal package.
package liquid

import "metais.org/metais/internal/finding"

// Code is a finding code, which the engine must not know.
var Code finding.Code

// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package render breaks test-only: release code imports the test-only
// helper.
package render

import "github.com/kottera/metais/internal/pkgtree/pkgtreetest"

// Tree is a tree built by the helper.
var Tree = pkgtreetest.New()

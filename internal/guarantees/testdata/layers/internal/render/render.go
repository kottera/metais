// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package render breaks test-only: release code imports the test-only
// helper. It breaks only-importers twice too, in text.go.
package render

import "metais.org/metais/internal/pkgtree/pkgtreetest"

// Tree is a tree built by the helper.
var Tree = pkgtreetest.New()

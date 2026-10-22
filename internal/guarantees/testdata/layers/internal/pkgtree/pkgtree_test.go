// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// pkgtree_test keeps test-only's exception: a test may import the
// helper.
package pkgtree_test

import (
	"testing"

	"metais.org/metais/internal/pkgtree/pkgtreetest"
)

func TestNew(t *testing.T) { _ = pkgtreetest.New() }

// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package leak is TestStandardLibraryOnlyControl's fixture: it imports what
// internal/liquid/value must not. It is under testdata, so it is never
// built.
package leak

import (
	"fmt"

	_ "github.com/kottera/metais/internal/testenv"
	_ "golang.org/x/term"
)

var _ = fmt.Sprint

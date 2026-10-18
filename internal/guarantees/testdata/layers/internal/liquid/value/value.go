// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package value breaks standard-library-only and only-importers: it
// imports golang.org/x/term.
package value

import "golang.org/x/term"

// Terminal is term.IsTerminal.
var Terminal = term.IsTerminal

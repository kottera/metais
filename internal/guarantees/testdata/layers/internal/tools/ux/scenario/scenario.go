// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package scenario, beneath the UX runner, is what the binary could import
// of it: the runner itself is a main package. The binary importing it
// breaks never-depends, and its importing net and net/http breaks
// only-importers, which the runner alone may.
package scenario

import (
	"net"
	"net/http"
)

// Load reads nothing.
func Load() {}

var _, _ = net.Listen, http.Get

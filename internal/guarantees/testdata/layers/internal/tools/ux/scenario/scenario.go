// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package scenario, beneath the UX runner, is what the binary could import
// of it: the runner itself is a main package. The binary importing it
// breaks never-depends.
package scenario

// Load reads nothing.
func Load() {}

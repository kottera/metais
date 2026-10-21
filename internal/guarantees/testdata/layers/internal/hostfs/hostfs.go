// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package hostfs keeps only-importers' exception: it may import x/sys.
package hostfs

import "golang.org/x/sys/unix"

// Write is a write function of hostfs.
func Write() error { return unix.Sync() }

// Read is a read function of hostfs.
func Read() {}

// Options is how a check or a write runs.
type Options struct{}

// WithFaults is an option, which a check of paths takes.
func WithFaults() Options { return Options{} }

// CheckPaths is the one call internal/render may make into hostfs.
func CheckPaths(opt Options, check func() []string) []string { return check() }

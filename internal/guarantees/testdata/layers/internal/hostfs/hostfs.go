// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package hostfs keeps only-importers' exception: it may import x/sys.
package hostfs

import "golang.org/x/sys/unix"

// Write is a write function of hostfs.
func Write() error { return unix.Sync() }

// Read is a read function of hostfs.
func Read() {}

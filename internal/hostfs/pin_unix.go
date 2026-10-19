// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package hostfs

// pinsBlockRename is false: a directory is renamed with entries beneath it
// held open, so every entry a render created keeps its pin until it is
// removed or committed.
const pinsBlockRename = false

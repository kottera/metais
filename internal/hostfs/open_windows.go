// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

// openFlags are added to every open for reading. Windows has no FIFO an
// open could wait on: a named pipe lives outside the file system's
// directories, so nothing is added.
const openFlags = 0

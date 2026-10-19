// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

// pinsBlockRename is true: a directory is not renamed while an entry
// beneath it is held open, so the pins beneath a directory are released
// before it is renamed. What their records say stays an identity on NTFS:
// a file id carries a sequence number that changes when its file record is
// used again, so it is never given to another file.
const pinsBlockRename = true

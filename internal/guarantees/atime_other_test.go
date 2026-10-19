// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build !darwin && !linux && !windows

package guarantees

import "io/fs"

// accessTime is not read on this platform: the existing-destination
// detector's read half is skipped.
func accessTime(fs.FileInfo) (int64, bool) { return 0, false }

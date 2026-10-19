// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build !unix

package guarantees

import "errors"

// mkfifo fails: named pipes are POSIX, and a setup that needs one is not
// replayed here, as the runner's fifos probe skips it.
func mkfifo(string) error { return errors.New("named pipes are POSIX") }

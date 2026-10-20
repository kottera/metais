// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build !linux && !darwin

package hostfs_test

import (
	"errors"
	"os"
	"runtime"
)

// openPTY opens no pseudo-terminal: this test opens one on linux and
// darwin only.
func openPTY() (master, slave *os.File, err error) {
	return nil, nil, errors.New("the test opens a pseudo-terminal on linux and darwin, and this is " + runtime.GOOS)
}

// rawOutput is never reached: openPTY opens nothing here.
func rawOutput(*os.File) error { return errors.New("no terminal") }

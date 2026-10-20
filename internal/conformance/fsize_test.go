// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
)

// fsizeStubName is the name of the stub that starts a step's process
// under its case's file-size limit (ADR-011 A6 item 3).
const fsizeStubName = "metais-fsizestub"

// stubDir is the directory TestMain builds the stubs into, or "" when it
// builds none.
var stubDir string

var (
	fsizeOnce sync.Once
	fsizePath string
	fsizeErr  error
)

// fsizeStub returns the path of testdata/fsizestub, built without the race
// detector into stubDir the first time a case or a probe needs it. It
// cannot be built where the platform has no file-size limit, and the
// file-size-limit requirement is not met there.
func fsizeStub() (string, error) {
	fsizeOnce.Do(func() {
		if stubDir == "" {
			fsizeErr = errors.New("the file-size stub has no directory to be built in")
			return
		}
		out := filepath.Join(stubDir, fsizeStubName)
		cmd, err := goCommand(filepath.Join(stubDir, goEnvDir), "build", "-trimpath", "-o", out, "./internal/conformance/testdata/fsizestub")
		if err != nil {
			fsizeErr = fmt.Errorf("building the file-size stub: %v", err)
			return
		}
		cmd.Dir = moduleRoot
		if b, err := cmd.CombinedOutput(); err != nil {
			fsizeErr = fmt.Errorf("building the file-size stub: %v\n%s", err, b)
			return
		}
		fsizePath = out
	})
	return fsizePath, fsizeErr
}

// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"os"
	"testing"
)

// TestMain runs the tests, unless this binary was started as the umask
// stub, which sets a umask and becomes the program it names.
func TestMain(m *testing.M) {
	if stub() {
		return
	}
	os.Exit(m.Run())
}

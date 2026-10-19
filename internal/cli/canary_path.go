// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build metais_canary_path

package cli

import (
	"bytes"
	"path/filepath"
)

// The path canary of ADR-009 D16 item 2: findings print the data
// directory's absolute path, made absolute from the working directory as
// the process found it, a realistic way for a caller's path to reach a
// message. Its probe vector is dat/required-input-missing, a missing
// required input with a data directory given; the SC7 harness must see
// the path by its checks 1 and 2.
func init() {
	canaryLeak = func(b []byte, dataDir string) []byte {
		abs, err := filepath.Abs(dataDir)
		if err != nil {
			return b
		}
		prefix := []byte(filepath.ToSlash(abs) + ": ")
		b = bytes.ReplaceAll(b, []byte(`"message":"`), append([]byte(`"message":"`), prefix...))
		return bytes.ReplaceAll(b, []byte(": error: "), append([]byte(": error: "), prefix...))
	}
}

// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package leak is the symbol rule's control: a non-test file calling
// os.Getenv outside the allow-list, which the rule must flag (ADR-010 D13
// item 4).
package leak

import "os"

// Home reads the host's home directory from the environment.
func Home() string {
	return os.Getenv("HOME")
}

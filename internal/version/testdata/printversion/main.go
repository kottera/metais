// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Command printversion prints version.String(), so a test can prove that a
// release build's -X flag reaches the release variable.
package main

import (
	"fmt"

	"github.com/kottera/metais/internal/version"
)

func main() {
	fmt.Print(version.String())
}

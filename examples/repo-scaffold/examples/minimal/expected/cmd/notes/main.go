// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Command notes: Keeps short notes in a plain-text file.
package main

import (
	"fmt"
	"os"

	"github.com/example/notes/internal/notes"
)

func main() {
	if err := notes.Run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "notes:", err)
		os.Exit(1)
	}
}

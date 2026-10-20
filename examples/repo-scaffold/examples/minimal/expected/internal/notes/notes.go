// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package notes is what the notes command does.
package notes

import (
	"fmt"
	"io"
)

// Description is what notes does.
const Description = "Keeps short notes in a plain-text file"

// Run runs notes with the arguments after the command's name,
// writing its output to w.
func Run(args []string, w io.Writer) error {
	if len(args) > 0 {
		return fmt.Errorf("unexpected argument %q", args[0])
	}
	_, err := fmt.Fprintln(w, Description)
	return err
}

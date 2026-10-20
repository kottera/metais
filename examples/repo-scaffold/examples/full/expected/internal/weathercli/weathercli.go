// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package weathercli is what the weather-cli command does.
package weathercli

import (
	"fmt"
	"io"
)

// Description is what weather-cli does.
const Description = "Prints the weather for a place"

// Run runs weather-cli with the arguments after the command's name,
// writing its output to w.
func Run(args []string, w io.Writer) error {
	if len(args) > 0 {
		return fmt.Errorf("unexpected argument %q", args[0])
	}
	_, err := fmt.Fprintln(w, Description)
	return err
}

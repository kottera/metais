// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Command weather-cli: Prints the weather for a place.
package main

import (
	"fmt"
	"os"

	"example.com/weather/internal/weathercli"
)

func main() {
	if err := weathercli.Run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "weather-cli:", err)
		os.Exit(1)
	}
}

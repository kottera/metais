// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Command site, the site's build step, keeps only-referrers' exception.
package main

import "os"

func main() {
	_, _ = os.Stdout.Write(nil)
	os.Exit(len(os.Args))
}

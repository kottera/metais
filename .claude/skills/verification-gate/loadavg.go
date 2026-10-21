// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build ignore

// loadavg prints the host's one-minute load average from sysinfo(2). gate.sh
// builds it on linux only when the host hides /proc/loadavg and uptime, as a
// sandbox can, and gate-exec.sh runs it to read the load a red package ran
// under.
package main

import (
	"fmt"
	"os"
	"syscall"
)

func main() {
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil {
		fmt.Fprintln(os.Stderr, "loadavg:", err)
		os.Exit(1)
	}
	// The kernel reports each load as a fixed-point number with 16
	// fractional bits (SI_LOAD_SHIFT).
	fmt.Printf("%.2f\n", float64(info.Loads[0])/65536)
}

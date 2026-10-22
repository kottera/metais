// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import "metais.org/metais/internal/hostfs"

// Paths keeps single-entry-point's exception: it calls hostfs.CheckPaths,
// with an option of hostfs among its arguments.
func Paths() []string {
	return hostfs.CheckPaths(hostfs.WithFaults(), func() []string { return nil })
}

// Smuggle breaks it inside the call: a function literal among the
// arguments is code of its own, not the call's.
func Smuggle() []string {
	return hostfs.CheckPaths(hostfs.WithFaults(), func() []string {
		_ = hostfs.Write()
		return nil
	})
}

// Stage breaks it outright, as a stage that wrote would.
func Stage() error { return hostfs.Write() }

// check breaks it by naming CheckPaths without calling it.
var check = hostfs.CheckPaths

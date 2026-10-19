// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build metais_canary_pid

package template

import (
	"os"
	"strconv"
)

// The pid canary (ADR-009 D16 item 2): every provenance line gains a pid
// field, the process's id, read when the process starts. It is a
// deliberate leak, built only under its tag, which the SC7 harness must
// catch by its differential check: the process id differs between the
// sentinel and the plain environment, and no sentinel can be planted in
// it.
func init() {
	pid := strconv.Itoa(os.Getpid())
	canaryProvenance = func() field { return field{name: "pid", value: pid} }
}

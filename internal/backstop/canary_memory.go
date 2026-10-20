// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build metais_canary_memory

package backstop

// The memory canary of ADR-009 Amendment A6 item 6: the ceiling is zero
// bytes, so every command the build runs passes it at the first read and
// exits through the backstop. It is how a subprocess is forced over a
// lowered ceiling: the calibration and corpus harnesses must fail a case
// that exits through it (ADR-009 D18 item 2), and cmd/metais's tests
// prove the stop at the process's own exit. A build tag, not a flag or the
// environment, because the backstop takes nothing from the host.
func init() {
	ceiling = 0
}

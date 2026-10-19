// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build race

package conformance

// raceEnabled reports whether this test binary is race-built, so TestMain
// builds the tool with -race too (ADR-011 D11 item 2).
const raceEnabled = true

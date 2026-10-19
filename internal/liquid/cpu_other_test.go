// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build !unix

package liquid_test

import "time"

// selfCPU reports that this platform's CPU time is not measured here.
func selfCPU() (time.Duration, bool) { return 0, false }

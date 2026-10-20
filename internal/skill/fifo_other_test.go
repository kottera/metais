// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build !unix

package skill

import "testing"

func mkfifo(t *testing.T, _ string) { t.Skip("no FIFO on this platform") }

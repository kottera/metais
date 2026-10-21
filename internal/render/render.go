// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package render is ADR-005 D13's render pipeline: planning, the path
// checks, producing every file's bytes, the destination check and
// emission (ADR-010 Amendment A7 item 2). internal/cli keeps each command
// that runs it: its flags, refusals, help and result records, and the
// choice of where the render goes.
package render

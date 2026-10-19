// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build metais_canary_clock

package dates

import "time"

// The clock canary (ADR-009 D16 item 2): date reads "now" and "today" from
// the host's clock instead of refusing them. It is a deliberate leak,
// built only under its tag, which the SC7 harness must catch on the
// clock's channel.
func init() {
	canaryNow = func() Time { return Time{t: time.Now().UTC().Truncate(time.Second)} }
}

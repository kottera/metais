// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build metais_canary_tz

package dates

import (
	"time"
	// The zone database, embedded, so the zone TZ names loads wherever the
	// canary runs: without it, a host whose zoneinfo files the process
	// cannot read gives UTC for every TZ, and the canary would leak
	// nothing there.
	_ "time/tzdata"
)

// The tz canary (ADR-009 D16 item 2): date formats in the host's zone. The
// zone is loaded when the process starts, from TZ, before hostenv.Isolate
// sets time.Local to UTC. It is a deliberate leak, built only under its
// tag, which the SC7 harness must catch on the time zone's channel.
func init() {
	host := time.Local
	// Using the zone loads it now, while TZ is still set.
	time.Unix(0, 0).In(host).Zone()
	canaryZone = func(t Time) Time { return Time{t: t.t.In(host), frac: t.frac} }
}

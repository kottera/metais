// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/kottera/metais/internal/testenv"
)

// The self-proof's rows met their 5 s wall clock under the full parallel
// go test -race on a loaded host, on processes that had nothing wrong with
// them (Wave 3's closeout, question 2). Two mechanisms, neither the
// implementation's, spent that clock, and these tests hold each away
// without moving the clock:
//
//   - the race runtime sleeps a second before every race-built process
//     exits (atexit_sleep_ms), so a fifth of each process's wall clock was
//     gone before it began; quietRaceExit sets it to 0;
//   - the self-proof starts every row's run together, a burst of some
//     eighty race-built processes on a few processors, so a process waited
//     its turn to start under a clock already running; procSlots starts at
//     most GOMAXPROCS processes at once, each clock starting with its own
//     process.

// TestRaceBuiltProcessesExitWithoutSleeping is the first mechanism, pinned
// both ways on a race-built binary: the fake run as the runner runs it
// gets GORACE's atexit_sleep_ms=0; and run without it, its exit alone
// takes the runtime's second, which is what the option takes away.
func TestRaceBuiltProcessesExitWithoutSleeping(t *testing.T) {
	t.Parallel()
	suite := writeFixture(t, testenv.Dir(t))
	r := fixtureRunner(t, suite, "cli", "clean")
	v, err := readVector(filepath.Join(suite, "vectors", filepath.FromSlash(fixValidate)))
	if err != nil {
		t.Fatal(err)
	}
	r.short = true
	res := r.runCase(v, &v.Cases[0], testenv.Dir(t))
	got, set := res.runs[envA].sb.vars["GORACE"]
	if raceEnabled != set || set && got != raceExit {
		t.Fatalf("a race-built %v program's sandbox sets GORACE %q (%v)", raceEnabled, got, set)
	}
	if !raceEnabled {
		return
	}
	fake := fakeProgram(t, testenv.Dir(t), "clean")
	sb, err := plainSandbox(testenv.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := runProc(fake, []string{"version"}, sb.work, sb.environ(), nil, defaultWall); err != nil {
		t.Fatal(err)
	}
	if took := time.Since(start); took < time.Second {
		t.Errorf("a race-built process without %s took %v, under the runtime's second at exit: the mechanism this test pins is gone", raceExit, took)
	}
}

// TestProcessesAreBounded is the second mechanism: however many runs start
// processes at once, no more than procSlots run together.
func TestProcessesAreBounded(t *testing.T) {
	fake := fakeProgram(t, testenv.Dir(t), "clean")
	sb, err := plainSandbox(testenv.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	sb.quietRaceExit()
	var wg sync.WaitGroup
	for range 3 * cap(procSlots) {
		wg.Go(func() {
			if _, err := runProc(fake, []string{"version"}, sb.work, sb.environ(), nil, defaultWall); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if peak := procPeak.Load(); peak > int64(cap(procSlots)) || peak < 1 {
		t.Errorf("%d processes ran at once; at most %d may", peak, cap(procSlots))
	}
}

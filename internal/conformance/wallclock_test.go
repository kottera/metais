// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"metais.org/metais/internal/testenv"
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

// TestCasesWaitTogether pins ADR-011 D11 item 9's budget against the
// mechanism that once spent it: a case that held its slot through the two
// seconds between A and B, so that the default run took one such wait per
// case, divided among the slots, whatever the host's speed, and grew with
// every case added. With a single slot and three cases in one vector,
// every case must be in that wait at once, which only a runner that holds
// no slot while it waits can reach; and every case must have started when
// runSuite returns, before any subtest waits for a parallel slot, so that
// no case waits for one behind a case that is waiting for its result.
// Neither depends on how fast the host runs the cases, short of the two
// minutes the three of them get to reach the wait.
func TestCasesWaitTogether(t *testing.T) {
	t.Parallel()
	suite := writeFixture(t, testenv.Dir(t))
	vectors := filepath.Join(suite, "vectors")
	dirs, err := vectorDirs(suite)
	if err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(vectors, filepath.FromSlash(fixValidate))
	for _, d := range dirs {
		if d != keep {
			if err := os.RemoveAll(d); err != nil {
				t.Fatal(err)
			}
		}
	}
	p := filepath.Join(keep, "vector.json")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	const cases = 3
	one := m["cases"].([]any)[0].(map[string]any)
	var many []any
	for i := range cases {
		c := maps.Clone(one)
		c["name"] = fmt.Sprintf("validate-%d", i+1)
		many = append(many, c)
	}
	m["cases"] = many
	if b, err = json.Marshal(m); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}

	r := fixtureRunner(t, suite, "cli", "clean")
	r.slots = make(chan struct{}, 1)
	var (
		mu       sync.Mutex
		waiting  int
		together = make(chan struct{})
		deadline time.Time
		apart    bool
	)
	r.wait = func(until time.Time) {
		mu.Lock()
		waiting++
		if waiting == cases {
			close(together)
		}
		if deadline.IsZero() {
			deadline = time.Now().Add(2 * time.Minute)
		}
		d := time.Until(deadline)
		mu.Unlock()
		select {
		case <-together:
		case <-time.After(d):
			mu.Lock()
			apart = true
			mu.Unlock()
		}
		time.Sleep(time.Until(until))
	}
	var started int64
	passed := t.Run("suite", func(t *testing.T) {
		stats := runSuite(t, r)
		started = stats.cases.Load()
	})
	if started != cases {
		t.Errorf("%d of %d cases had started when runSuite returned", started, cases)
	}
	mu.Lock()
	defer mu.Unlock()
	if apart || waiting != cases {
		t.Errorf("%d of %d cases waited between A and B, never all at once with one slot: a case holds its slot while it waits", waiting, cases)
	}
	if !passed {
		t.Error("the suite failed")
	}
}

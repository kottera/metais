// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"context"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// TestStubExitsAtOnce pins the measuring stub's own exit: when this test
// binary is race-built, the stub runs with GORACE set to raceExit, so the
// race runtime does not sleep its second after every process the stub
// starts. That second, paid by every process of a program that is not
// race-built, which quietRaceExit leaves alone, held a runner slot idle
// through most of TestHostIsolation's run of every vector with the
// release build, past the package's timeout. The process itself still
// gets exactly the environment it is given: GORACE only when the step's
// environment sets it.
func TestStubExitsAtOnce(t *testing.T) {
	envProg, err := exec.LookPath("env")
	if err != nil {
		t.Skipf("no env program to print the environment: %v", err)
	}
	for _, env := range [][]string{
		{"METAIS_A=1", "METAIS_B=two"},
		{"METAIS_A=1", "GORACE=" + raceExit},
		{},
	} {
		cmd, usage, err := measured(context.Background(), envProg, nil, env)
		if err != nil {
			t.Fatal(err)
		}
		usage(nil)
		stubRace := slices.Contains(cmd.Env, "GORACE="+raceExit)
		if raceEnabled && !stubRace {
			t.Errorf("given %q, the race-built stub runs with %q: its exit sleeps the race runtime's second", env, cmd.Env)
		}
		if !raceEnabled && !slices.Equal(cmd.Env, env) {
			t.Errorf("given %q, the stub runs with %q", env, cmd.Env)
		}

		p, err := runProc(envProg, nil, testenv.Dir(t), env, nil, defaultWall)
		if err != nil {
			t.Fatal(err)
		}
		got := strings.Fields(string(p.stdout))
		if !slices.Equal(got, env) {
			t.Errorf("given %q, the process received %q", env, got)
		}
	}
}

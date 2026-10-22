// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
)

// TestStubIsItsOwnProgram pins what every measured process pays beside
// itself on linux: its stub is testdata/procstub, built without the race
// detector, not this test binary. Re-executed as the stub, a race-built
// test binary cost each process the race runtime's start and, unless
// GORACE said otherwise, a second's sleep at its exit, which a program
// that is not race-built, such as the release build, was never given
// GORACE to stop: TestHostIsolation's run of every vector spent most of
// its CPU time in the stubs, and most of its wall time in their sleeps,
// past the package's timeout. The process still gets exactly the
// environment it is given.
func TestStubIsItsOwnProgram(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if procStub == "" || procStub == self {
		t.Fatalf("the measuring stub is %q; this test binary is %q", procStub, self)
	}
	if raceBuilt(procStub) {
		t.Errorf("the measuring stub %s is race-built", procStub)
	}
	envProg, err := exec.LookPath("env")
	if err != nil {
		t.Skipf("no env program to print the environment: %v", err)
	}
	for _, env := range [][]string{
		{"METAIS_A=1", "METAIS_B=two"},
		{"METAIS_A=1", "GORACE=" + raceExit},
		{},
	} {
		p, err := runProc(envProg, nil, testenv.Dir(t), env, nil, defaultWall)
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Fields(string(p.stdout)); !slices.Equal(got, env) {
			t.Errorf("given %q, the process received %q", env, got)
		}
	}
}

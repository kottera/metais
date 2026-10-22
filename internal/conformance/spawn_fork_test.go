// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build unix && !linux

package conformance

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
)

// forkStarts is whether Go starts a process here by fork(2), so that the
// runner starts its processes through the spawner (testdata/spawner).
const forkStarts = true

// TestProcessesStartFromTheSpawner pins what every process the runner
// runs costs this test binary where Go starts a process by fork(2): it
// is started by the spawner, a small program that is not race-built, and
// never forked from this binary. A fork marks every page of the forking
// process copy-on-write, and this race-built binary then took a fault on
// each page it wrote, its shadow memory's among them: started from here,
// TestHostIsolation's eleven thousand processes cost it some 400 s of
// system time in eleven million faults. The process still gets exactly
// the environment, working directory and standard input it is given.
func TestProcessesStartFromTheSpawner(t *testing.T) {
	c, err := spawner()
	if err != nil || c == nil {
		t.Fatalf("no spawner: %v", err)
	}
	if raceBuilt(spawnerPath) {
		t.Errorf("the spawner %s is race-built", spawnerPath)
	}
	dir := testenv.Dir(t)
	p, err := runProc("/bin/sh", []string{"-c", `echo "$PPID"; pwd -P; cat`}, dir, nil, []byte("the input\n"), defaultWall)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(p.stdout)), "\n")
	if len(lines) != 3 {
		t.Fatalf("the process wrote %q", p.stdout)
	}
	if ppid, _ := strconv.Atoi(lines[0]); ppid != c.cmd.Process.Pid || ppid == os.Getpid() {
		t.Errorf("the process's parent is %d; the spawner is %d, this test binary %d", ppid, c.cmd.Process.Pid, os.Getpid())
	}
	if want, err := filepath.EvalSymlinks(dir); err != nil || lines[1] != want {
		t.Errorf("the process ran in %q, want %q", lines[1], want)
	}
	if lines[2] != "the input" {
		t.Errorf("the process read %q from its standard input", lines[2])
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
		p, err := runProc(envProg, nil, dir, env, nil, defaultWall)
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Fields(string(p.stdout)); !slices.Equal(got, env) {
			t.Errorf("given %q, the process received %q", env, got)
		}
	}
}

// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// buildProcStub builds testdata/procstub, without the race detector, into
// dir, and returns its path.
func buildProcStub(dir string) (string, error) {
	out := filepath.Join(dir, procStubName)
	cmd := exec.Command("go", "build", "-trimpath", "-o", out, "./internal/conformance/testdata/procstub")
	cmd.Dir = moduleRoot
	if b, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("building the measuring stub: %v\n%s", err, b)
	}
	return out, nil
}

// measured is the command that runs path with args and exactly env, and
// the function that reads its usage once it has finished, or releases
// what it holds when given nil because the process never started.
//
// On linux a process's ru_maxrss counts the peak of the address space it
// was executed from (TestRSSIsTheProcessAlone), and Go executes a child
// from its parent's own address space, so read directly it would count
// this test process's peak. The process is therefore started by the stub,
// testdata/procstub, which starts it with its standard streams, working
// directory and environment, waits for it, and writes its usage to the
// pipe on its descriptor 3. What the process inherits is the stub's own
// few MiB, which no Go program's own peak falls below.
//
// The stub is a small program of its own, never race-built
// (TestStubIsItsOwnProgram): every process the runner starts pays for the stub's
// start and exit, and the test binary, re-executed as the stub, made that
// the race runtime's start and its second's sleep at exit.
func measured(ctx context.Context, path string, args, env []string) (*exec.Cmd, func(*os.ProcessState) (usage, error), error) {
	if procStub == "" {
		return nil, nil, errors.New("the measuring stub was not built")
	}
	// The path is resolved here, against this process's PATH, as running
	// it directly would; the stub runs in the step's environment.
	path, err := exec.LookPath(path)
	if err != nil && !errors.Is(err, exec.ErrDot) {
		return nil, nil, err
	}
	r, w, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	cmd := exec.CommandContext(ctx, procStub)
	cmd.Args = append([]string{procStubName, path}, args...)
	cmd.Env = env
	if env == nil {
		cmd.Env = []string{}
	}
	cmd.ExtraFiles = []*os.File{w}
	return cmd, func(ps *os.ProcessState) (usage, error) {
		w.Close()
		defer r.Close()
		if ps == nil {
			return usage{}, nil
		}
		report, err := io.ReadAll(io.LimitReader(r, 1<<10))
		if err != nil {
			return usage{}, err
		}
		// A stub that wrote nothing was killed, by the wall clock, and the
		// process it started with it: what that process used is unknown.
		u := usage{exit: ps.ExitCode(), cpu: ps.UserTime() + ps.SystemTime()}
		if len(report) == 0 {
			return u, fmt.Errorf("the measuring stub exited %d and reported nothing", ps.ExitCode())
		}
		var status uint32
		var utime, stime int64
		if _, err := fmt.Sscanf(string(report), "usage %d %d %d %d", &status, &utime, &stime, &u.rss); err != nil {
			return u, fmt.Errorf("the measuring stub: %s", report)
		}
		ws := syscall.WaitStatus(status)
		u.exit = -1
		if ws.Exited() {
			u.exit = ws.ExitStatus()
		}
		u.cpu, u.rssOK = time.Duration(utime+stime), true
		return u, nil
	}, nil
}

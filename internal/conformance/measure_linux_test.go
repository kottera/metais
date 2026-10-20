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
	"runtime"
	"slices"
	"strings"
	"syscall"
	"time"
)

// measured is the command that runs path with args, and the function that
// reads its usage once it has finished, or releases what it holds when
// given nil because the process never started.
//
// On linux a process's ru_maxrss counts the peak of the address space it
// was executed from (TestRSSIsTheProcessAlone), and Go executes a child
// from its parent's own address space, so read directly it would count
// this test process's peak. The process is therefore started by the stub:
// this test binary, executed afresh under procStubName, which starts it
// with its standard streams, working directory and environment, waits for
// it, and writes its usage to the pipe on its descriptor 3. What the
// process inherits is the stub's own few MiB, which no Go program's own
// peak falls below.
//
// The process gets exactly env. The stub is this test binary, so when it
// is race-built its own exit would sleep the race runtime's second
// (raceExit) after every process it starts, which the release build's
// run of every vector paid once per process; so the stub always runs
// with GORACE set to raceExit, and when env does not set GORACE the stub
// unsets it for the process, which never receives it (stubEnviron).
func measured(ctx context.Context, path string, args, env []string) (*exec.Cmd, func(*os.ProcessState) (usage, error), error) {
	stub, err := os.Executable()
	if err != nil {
		return nil, nil, err
	}
	// The path is resolved here, against this process's PATH, as running
	// it directly would; the stub runs in the step's environment.
	if path, err = exec.LookPath(path); err != nil && !errors.Is(err, exec.ErrDot) {
		return nil, nil, err
	}
	r, w, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	cmd := exec.CommandContext(ctx, stub)
	stubEnv, mode := stubEnviron(env)
	cmd.Args = append([]string{procStubName, mode, path}, args...)
	cmd.Env = stubEnv
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

// How the stub gives the process its environment: as the stub has it, or
// without GORACE, which only the stub's own exit was given.
const (
	stubKeepEnv    = "env"
	stubDropGORACE = "env-GORACE"
)

// stubEnviron is the stub's environment for a process given env, and how
// the stub gives the process its own: env with GORACE set to raceExit
// when the stub is race-built and env leaves GORACE unset, so the stub
// exits at once and the process still gets exactly env.
func stubEnviron(env []string) ([]string, string) {
	if env == nil {
		env = []string{}
	}
	if !raceEnabled || slices.ContainsFunc(env, func(kv string) bool { return strings.HasPrefix(kv, "GORACE=") }) {
		return env, stubKeepEnv
	}
	return append(slices.Clone(env), "GORACE="+raceExit), stubDropGORACE
}

// procStubMain is the stub measured starts: args are how it gives the
// process its environment (stubEnviron), the path of the process to run
// and its arguments. It runs it, and writes "usage" and its
// wait status, user and system CPU time in nanoseconds and ru_maxrss to
// descriptor 3, or "error" and why it could not start it.
//
// The process gets the stub's standard streams as they are and the stub's
// environment, without GORACE when the first argument says so, and is
// killed when the stub dies, so the runner's
// wall clock, which kills the stub, kills it too. The stub's thread is
// locked because the kernel sends that signal when the starting thread
// ends.
func procStubMain(args []string) int {
	report := os.NewFile(3, "report")
	if len(args) < 2 || args[0] != stubKeepEnv && args[0] != stubDropGORACE {
		fmt.Fprintln(report, "error no environment mode and process named")
		return 2
	}
	mode, args := args[0], args[1:]
	runtime.LockOSThread()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Err = nil // the runner resolved the path
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = append([]string{}, os.Environ()...)
	if mode == stubDropGORACE {
		cmd.Env = slices.DeleteFunc(cmd.Env, func(kv string) bool { return strings.HasPrefix(kv, "GORACE=") })
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
	err := cmd.Run()
	ps := cmd.ProcessState
	if ps == nil {
		fmt.Fprintln(report, "error", err)
		return 2
	}
	ru, ok := ps.SysUsage().(*syscall.Rusage)
	if !ok || ru == nil {
		fmt.Fprintln(report, "error no resource usage")
		return 2
	}
	fmt.Fprintf(report, "usage %d %d %d %d\n", uint32(ps.Sys().(syscall.WaitStatus)), ru.Utime.Nano(), ru.Stime.Nano(), ru.Maxrss)
	return 0
}

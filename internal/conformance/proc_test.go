// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"bytes"
	"context"
	"debug/buildinfo"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// The default per-step bounds of ADR-011 D13 item 1, and the factors of
// D13 item 3 for a race-built tool.
const (
	defaultCPU       = 10 * time.Second
	defaultRSS       = 1024 << 20
	defaultWall      = 120 * time.Second
	raceCPUFactor    = 20
	raceMemoryFactor = 10
)

// limits are the bounds one process of a step runs under.
type limits struct {
	cpu  time.Duration
	rss  int64 // bytes
	wall time.Duration
}

// stepLimits are a step's bounds: the defaults, lowered by the step's own
// bounds, multiplied when the program is race-built (D13 items 1 and 3).
// The wall clock is the runner's, and is not multiplied.
func stepLimits(b *bounds, race bool, wall time.Duration) limits {
	l := limits{cpu: defaultCPU, rss: defaultRSS, wall: wall}
	if b != nil {
		if b.CPUms > 0 && time.Duration(b.CPUms)*time.Millisecond < l.cpu {
			l.cpu = time.Duration(b.CPUms) * time.Millisecond
		}
		if b.RSSMiB > 0 && b.RSSMiB<<20 < l.rss {
			l.rss = b.RSSMiB << 20
		}
	}
	if race {
		l.cpu *= raceCPUFactor
		l.rss *= raceMemoryFactor
	}
	return l
}

// maxRSSBytes converts ru_maxrss to bytes: darwin counts bytes, linux
// KiB (getrusage(2) on each; ADR-011 D13 item 2).
func maxRSSBytes(goos string, raw int64) int64 {
	if goos == "darwin" || goos == "ios" {
		return raw
	}
	return raw * 1024
}

// raceBuilt reports whether the program at path is race-built, from its
// build information. A program without Go build information is not.
func raceBuilt(path string) bool {
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return false
	}
	for _, s := range info.Settings {
		if s.Key == "-race" && s.Value == "true" {
			return true
		}
	}
	return false
}

// proc is one finished process.
type proc struct {
	stdout, stderr []byte
	exit           int
	cpu            time.Duration
	rss            int64 // bytes; -1 when not measured
	killed         bool  // the wall clock ran out
}

// runProc runs path with argv0 and args in dir, with exactly env, feeding
// stdin, under lim's wall clock. A process that cannot be started is an
// error; one that runs and fails is a proc.
func runProc(path string, args []string, dir string, env []string, stdin []byte, wall time.Duration) (*proc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), wall)
	defer cancel()
	cmd, usage, err := measured(ctx, path, args)
	if err != nil {
		return nil, err
	}
	cmd.Dir = dir
	cmd.Env = env
	if env == nil {
		cmd.Env = []string{}
	}
	cmd.Stdin = bytes.NewReader(stdin)
	stdout, stderr := &capped{}, &capped{}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	cmd.WaitDelay = time.Second
	err = cmd.Run()
	p := &proc{stdout: stdout.Bytes(), stderr: stderr.Bytes(), rss: -1}
	if cmd.ProcessState == nil {
		usage(nil)
		return nil, fmt.Errorf("start %s: %w", path, err)
	}
	if ctx.Err() != nil {
		p.killed = true
	}
	u, uerr := usage(cmd.ProcessState)
	if uerr != nil && !p.killed {
		return nil, fmt.Errorf("run %s: %w", path, uerr)
	}
	if stdout.over || stderr.over {
		return nil, fmt.Errorf("%s wrote more than the runner's %d bytes to a standard stream", path, maxInflate)
	}
	p.exit, p.cpu = u.exit, u.cpu
	if u.rssOK {
		p.rss = maxRSSBytes(runtime.GOOS, u.rss)
	}
	var ee *exec.ExitError
	if err != nil && !errors.As(err, &ee) && !p.killed && !errors.Is(err, exec.ErrWaitDelay) {
		return nil, fmt.Errorf("run %s: %w", path, err)
	}
	return p, nil
}

// usage is what a finished process used and how it ended: its exit code,
// -1 when a signal ended it; its user plus system CPU time; and its
// ru_maxrss in the platform's unit, when the platform reports it.
type usage struct {
	exit  int
	cpu   time.Duration
	rss   int64
	rssOK bool
}

// capped holds what a process writes to one stream, up to maxInflate
// bytes, so a runaway tool fails its step rather than the runner.
type capped struct {
	bytes.Buffer
	over bool
}

func (c *capped) Write(b []byte) (int, error) {
	if room := maxInflate - c.Len(); len(b) > room {
		c.over = true
		c.Buffer.Write(b[:max(room, 0)])
		return len(b), nil
	}
	return c.Buffer.Write(b)
}

// boundProblems checks p against lim (ADR-011 D13).
func boundProblems(p *proc, lim limits) []problem {
	var ps []problem
	if p.killed {
		ps = append(ps, problem{check: checkWall, msg: fmt.Sprintf("the process ran past the %v wall clock and was killed", lim.wall)})
	}
	if p.cpu > lim.cpu {
		ps = append(ps, problem{check: checkCPU, msg: fmt.Sprintf("the process used %v of CPU time, above the bound of %v", p.cpu, lim.cpu)})
	}
	if p.rss > lim.rss {
		ps = append(ps, problem{check: checkRSS, msg: fmt.Sprintf("the process's peak resident memory was %d MiB, above the bound of %d MiB", p.rss>>20, lim.rss>>20)})
	}
	return ps
}

// executable reports whether path names a file this process can run.
func executable(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	return nil
}

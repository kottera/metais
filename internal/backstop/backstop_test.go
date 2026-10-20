// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package backstop

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// stopReport is what the tests' Stop writes, and the status it returns.
const (
	stopReport = "stopped\n"
	stopStatus = 2
)

func stopped(stdout, stderr io.Writer) int {
	io.WriteString(stdout, stopReport)
	return stopStatus
}

// TestCeiling is A6 item 6's figures: the ceiling is 1.5 GiB, the soft
// memory limit 1 GiB, and the memory is read at least every 10 ms. The
// memory canary's build is the one build with another ceiling.
func TestCeiling(t *testing.T) {
	if ceiling != 3<<29 {
		t.Errorf("the ceiling is %d bytes, want 1.5 GiB", ceiling)
	}
	if softLimit != 1<<30 {
		t.Errorf("the soft memory limit is %d bytes, want 1 GiB", softLimit)
	}
	if interval > 10*time.Millisecond {
		t.Errorf("the memory is read every %v, want at least every 10ms", interval)
	}
}

// TestRunSetsSoftLimit proves Run sets the runtime's soft memory limit,
// so collectable garbage does not count against the ceiling.
func TestRunSetsSoftLimit(t *testing.T) {
	was := debug.SetMemoryLimit(-1)
	t.Cleanup(func() { debug.SetMemoryLimit(was) })
	debug.SetMemoryLimit(was / 2)
	status := Run(io.Discard, io.Discard, func(context.Context, io.Writer, io.Writer) int { return 7 }, stopped)
	if status != 7 {
		t.Errorf("Run returned %d, want the command's 7", status)
	}
	if got := debug.SetMemoryLimit(-1); got != softLimit {
		t.Errorf("the soft memory limit is %d after Run, want %d", got, softLimit)
	}
}

// TestMeter holds the meter to what the runtime has mapped and not
// released: more than the heap a test holds live, and more than it
// before it is allocated.
func TestMeter(t *testing.T) {
	m := newMeter()
	before := m.read()
	hold := make([]byte, 256<<20)
	for i := 0; i < len(hold); i += 4096 {
		hold[i] = 1
	}
	after := m.read()
	if after < before+uint64(len(hold)) {
		t.Errorf("the meter read %d bytes with %d live, %d before", after, len(hold), before)
	}
	if allocs := testing.AllocsPerRun(10, func() { m.read() }); allocs != 0 {
		t.Errorf("a read allocates %v times, want none", allocs)
	}
	hold[0] = 0
}

// TestStopsMidAllocation is A6 item 6's stop at a lowered ceiling, with
// the real meter: a command that allocates without end is stopped once
// the memory passes the ceiling, its context ends, and the stop's status
// and report are the run's.
func TestStopsMidAllocation(t *testing.T) {
	m := newMeter()
	lowered := m.read() + 64<<20
	ended := make(chan struct{})
	var stdout bytes.Buffer
	status := watcher{ceiling: lowered, read: m.read, every: interval}.run(&stdout, io.Discard,
		func(ctx context.Context, stdout, stderr io.Writer) int {
			defer close(ended)
			io.WriteString(stdout, "started\n")
			var hold [][]byte
			for ctx.Err() == nil {
				if len(hold) == 1024 {
					return 99 // 1 GiB held and never stopped
				}
				b := make([]byte, 1<<20)
				for i := 0; i < len(b); i += 4096 {
					b[i] = 1
				}
				hold = append(hold, b)
			}
			return 0
		}, stopped)
	<-ended
	if status != stopStatus {
		t.Fatalf("the run returned %d, want the stop's %d", status, stopStatus)
	}
	if got := stdout.String(); got != "started\n"+stopReport {
		t.Errorf("standard output is %q", got)
	}
}

// TestSealed proves a stopped command writes nothing more: its streams
// are sealed before the stop writes, and a write after that never
// returns.
func TestSealed(t *testing.T) {
	var over atomic.Bool
	tripped, ended, writing := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var stdout, stderr bytes.Buffer
	var afterWrite atomic.Bool
	w := watcher{ceiling: 100, read: func() uint64 {
		if over.Load() {
			return 101
		}
		return 100
	}, every: time.Millisecond}
	status := w.run(&stdout, &stderr, func(ctx context.Context, stdout, stderr io.Writer) int {
		io.WriteString(stdout, "before\n")
		over.Store(true)
		close(tripped)
		<-ctx.Done()
		close(ended)
		go func() {
			close(writing)
			io.WriteString(stderr, "after\n")
			afterWrite.Store(true)
		}()
		io.WriteString(stdout, "after\n")
		afterWrite.Store(true)
		return 0
	}, stopped)
	<-tripped
	<-ended
	<-writing
	if status != stopStatus {
		t.Errorf("the run returned %d, want the stop's %d", status, stopStatus)
	}
	if stdout.String() != "before\n"+stopReport || stderr.Len() != 0 {
		t.Errorf("standard output is %q and standard error %q", stdout.String(), stderr.String())
	}
	if afterWrite.Load() {
		t.Error("a write of the stopped command returned")
	}
}

// TestFinishedFirst proves a command that ends below the ceiling is the
// run's: its status, and its streams as it wrote them.
func TestFinishedFirst(t *testing.T) {
	var stdout bytes.Buffer
	status := watcher{ceiling: 100, read: func() uint64 { return 100 }, every: time.Millisecond}.run(&stdout, io.Discard,
		func(_ context.Context, stdout, _ io.Writer) int {
			io.WriteString(stdout, "done\n")
			return 1
		}, stopped)
	if status != 1 || stdout.String() != "done\n" {
		t.Errorf("the run returned %d and wrote %q, want the command's 1 and %q", status, stdout.String(), "done\n")
	}
}

// TestOverAtStart proves the memory is read before the command starts: a
// process already past the ceiling runs no command.
func TestOverAtStart(t *testing.T) {
	var stdout bytes.Buffer
	ran := false
	status := watcher{ceiling: 0, read: func() uint64 { return 1 }, every: time.Hour}.run(&stdout, io.Discard,
		func(context.Context, io.Writer, io.Writer) int {
			ran = true
			return 0
		}, stopped)
	if ran || status != stopStatus || stdout.String() != stopReport {
		t.Errorf("the command ran: %v; the run returned %d and wrote %q", ran, status, stdout.String())
	}
}

// figure is the memory the leak tests' meter reads: past every ceiling
// they set, and with forms no report would hold by chance.
const figure uint64 = 1_717_986_939

// figureForms are the ways a careless report would print figure: its
// bytes, in decimal and in hexadecimal, and in KiB, MiB and GiB.
func figureForms() []string {
	return []string{
		strconv.FormatUint(figure, 10),
		fmt.Sprintf("%x", figure), fmt.Sprintf("%X", figure),
		strconv.FormatUint(figure>>10, 10),
		strconv.FormatUint(figure>>20, 10),
		fmt.Sprintf("%.1f", float64(figure)/(1<<30)),
		fmt.Sprintf("%.2f", float64(figure)/(1<<30)),
	}
}

// leaked is every form of figure that appears in b.
func leaked(b []byte) []string {
	var got []string
	for _, f := range figureForms() {
		if bytes.Contains(b, []byte(f)) {
			got = append(got, f)
		}
	}
	return got
}

// TestNoFigure is A6 item 6 and ADR-009 D15: the figure the stop was
// decided on reaches neither stream. It is pinned false first: the leak
// fault writes the figure, and the scan must see it.
func TestNoFigure(t *testing.T) {
	for _, leak := range []bool{true, false} {
		var stdout, stderr bytes.Buffer
		w := watcher{ceiling: 1 << 30, read: func() uint64 { return figure }, every: time.Hour, leak: leak}
		status := w.run(&stdout, &stderr, func(context.Context, io.Writer, io.Writer) int { return 0 },
			func(stdout, stderr io.Writer) int {
				io.WriteString(stdout, "metais 1.0.0 stopped the command\n")
				io.WriteString(stderr, "error internal.memory\n")
				return stopStatus
			})
		if status != stopStatus {
			t.Fatalf("leak %v: the run returned %d", leak, status)
		}
		got := leaked(append(stdout.Bytes(), stderr.Bytes()...))
		switch {
		case leak && len(got) == 0:
			t.Errorf("the leak fault wrote the figure, and the scan did not see it: %q %q", stdout.String(), stderr.String())
		case !leak && len(got) > 0:
			t.Errorf("the streams hold the figure as %q: %q %q", got, stdout.String(), stderr.String())
		}
	}
}

// TestKeepsTerminal proves a stream with a file descriptor keeps it
// behind its gate, so a command still tells a terminal apart.
func TestKeepsTerminal(t *testing.T) {
	g := seal(os.Stderr).writer()
	f, ok := g.(interface{ Fd() uintptr })
	if !ok || f.Fd() != os.Stderr.Fd() {
		t.Errorf("the gate over standard error has no file descriptor, or another: %T", g)
	}
	if _, ok := seal(&strings.Builder{}).writer().(interface{ Fd() uintptr }); ok {
		t.Error("the gate over a buffer claims a file descriptor")
	}
}

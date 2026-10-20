// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package backstop is metais's memory backstop (ADR-009 Amendment A6 item
// 6). It runs a command while it watches the process's Go-managed memory,
// and once that passes the ceiling it seals the command's streams, has the
// stop reported on them in the command's place, and returns, so that the
// process exits with the command where it stood.
//
// The backstop is not a limit (A6 item 6): it is not in ADR-009 D3's table,
// and nothing a command line, the environment or a package gives moves its
// ceiling. Reaching it is a defect in the tool. The figure it measures is
// compared with the ceiling and goes nowhere else: a Stop is given the
// streams and nothing of the memory (ADR-009 D15).
package backstop

import (
	"context"
	"io"
	"runtime/debug"
	"runtime/metrics"
	"strconv"
	"sync"
	"time"
)

const (
	// softLimit is the runtime's soft memory limit, 1 GiB, so the
	// collector frees garbage before it counts against the ceiling.
	softLimit = 1 << 30
	// interval is the longest the watcher waits between two reads.
	interval = 10 * time.Millisecond
)

// ceiling is the Go-managed memory, in bytes, past which a command stops:
// 1.5 GiB. Only the memory canary lowers it.
var ceiling uint64 = 3 << 29

// Command runs a command, writing to stdout and stderr, and returns its
// exit status. ctx ends when the backstop stops it.
type Command func(ctx context.Context, stdout, stderr io.Writer) int

// Stop reports a command the backstop stopped, on stdout and stderr, which
// the command can no longer write, and returns the exit status.
type Stop func(stdout, stderr io.Writer) int

// Run runs cmd under the backstop and returns its exit status, or, when
// the process's Go-managed memory passes the ceiling first, the status
// stop returns. It sets the runtime's soft memory limit, which holds for
// the rest of the process. The caller exits once Run returns: a stopped
// command is left running, unable to write either stream.
func Run(stdout, stderr io.Writer, cmd Command, stop Stop) int {
	debug.SetMemoryLimit(softLimit)
	return watcher{ceiling: ceiling, read: newMeter().read, every: interval}.run(stdout, stderr, cmd, stop)
}

// watcher is one backstop: its ceiling, how it reads the memory and how
// often.
type watcher struct {
	ceiling uint64
	read    func() uint64
	every   time.Duration
	// leak, which only the detector's test sets, writes the figure the
	// stop was decided on to standard error, so that the test that no
	// figure reaches a stream is seen to fail when one does.
	leak bool
}

// run runs cmd under w. The memory is read once before cmd starts, then
// at least every w.every while it runs; the first read past the ceiling
// stops it.
func (w watcher) run(stdout, stderr io.Writer, cmd Command, stop Stop) int {
	out, errs := seal(stdout), seal(stderr)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	over := make(chan uint64, 1)
	if m := w.read(); m > w.ceiling {
		over <- m
	} else {
		go func() {
			for ctx.Err() == nil {
				time.Sleep(w.every)
				if m := w.read(); m > w.ceiling {
					over <- m
					return
				}
			}
		}()
	}
	done := make(chan int, 1)
	if len(over) == 0 {
		go func() { done <- cmd(ctx, out.writer(), errs.writer()) }()
	}
	var m uint64
	select {
	case status := <-done:
		return status
	case m = <-over:
	}
	// Seal both streams: a write in progress ends first, and no write of
	// the command starts after it. A command that finished meanwhile has
	// written its whole result, which stands.
	out.mu.Lock()
	errs.mu.Lock()
	out.sealed, errs.sealed = true, true
	select {
	case status := <-done:
		return status
	default:
	}
	cancel()
	if w.leak {
		io.WriteString(stderr, strconv.FormatUint(m, 10)+"\n")
	}
	return stop(stdout, stderr)
}

// meter reads the process's Go-managed memory: what the runtime has mapped
// from the operating system, less what it has released to it.
type meter struct {
	samples [2]metrics.Sample
}

func newMeter() *meter {
	m := &meter{}
	m.samples[0].Name = "/memory/classes/total:bytes"
	m.samples[1].Name = "/memory/classes/heap/released:bytes"
	return m
}

// read reads the memory. It allocates nothing, and it is called by one
// goroutine at a time.
func (m *meter) read() uint64 {
	metrics.Read(m.samples[:])
	return m.samples[0].Value.Uint64() - m.samples[1].Value.Uint64()
}

// gate is a stream a command writes through, until the backstop seals it.
type gate struct {
	mu     sync.Mutex
	w      io.Writer
	sealed bool
}

func seal(w io.Writer) *gate { return &gate{w: w} }

// writer is the stream the command is given: g, and g's file descriptor
// when the stream has one, so the command still tells a terminal apart.
func (g *gate) writer() io.Writer {
	if f, ok := g.w.(interface{ Fd() uintptr }); ok {
		return fdGate{g, f}
	}
	return g
}

// Write writes p, or, once the stream is sealed, blocks for the rest of
// the process: the command writes nothing more, and its write returns no
// error it could go on to report.
func (g *gate) Write(p []byte) (int, error) {
	g.mu.Lock()
	if g.sealed {
		g.mu.Unlock()
		select {}
	}
	defer g.mu.Unlock()
	return g.w.Write(p)
}

// fdGate is a gate over a stream with a file descriptor.
type fdGate struct {
	*gate
	f interface{ Fd() uintptr }
}

func (g fdGate) Fd() uintptr { return g.f.Fd() }

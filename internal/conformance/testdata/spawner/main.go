// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build unix

// Command spawner starts the processes internal/conformance runs, on the
// platforms where Go starts a process by fork(2). It reads requests from
// its standard input and writes responses to its standard output, each a
// gob-encoded value, and serves every request at once. A request names a
// program, its arguments, its working directory, its environment exactly,
// what to feed its standard input, its wall clock and the most it may
// write to each standard stream. The response carries what it wrote to
// each, how it exited, its user plus system CPU time and its ru_maxrss,
// and whether the wall clock ran out and killed it.
//
// fork(2) marks every page of the forking process copy-on-write, and the
// forking process takes a fault on the next write to each one. The race
// detector writes its shadow memory on every access, so a race-built test
// binary that starts thousands of processes spends most of its time in
// those faults. This program is small and never race-built, so a fork
// from it is cheap, and the test binary forks only once, to start it.
//
// When its standard input ends it kills every process still running, and
// exits once each has ended.
package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// request and response are the two sides of the protocol; the runner's
// spawnRequest and spawnResponse have the same fields.
type request struct {
	ID    uint64
	Path  string
	Args  []string
	Dir   string
	Env   []string
	Stdin []byte
	Wall  time.Duration
	Cap   int
}

type response struct {
	ID       uint64
	StartErr string // why the process could not be started, or ""
	RunErr   string // how waiting for it failed other than by its exit, or ""
	Stdout   []byte
	Stderr   []byte
	Over     bool // a stream passed the cap
	Exit     int
	CPU      time.Duration
	RSS      int64
	RSSOK    bool
	Killed   bool // the wall clock ran out
}

func main() {
	// A runner gone before its responses would otherwise end this program
	// by SIGPIPE, leaving its processes running. The signal is handled,
	// not ignored, so the processes it starts do not inherit it ignored.
	signal.Notify(make(chan os.Signal, 1), syscall.SIGPIPE)
	ctx, cancel := context.WithCancel(context.Background())
	var mu sync.Mutex
	enc := gob.NewEncoder(os.Stdout)
	dec := gob.NewDecoder(os.Stdin)
	var wg sync.WaitGroup
	for {
		var req request
		if err := dec.Decode(&req); err != nil {
			if !errors.Is(err, io.EOF) {
				fmt.Fprintln(os.Stderr, "spawner:", err)
			}
			break
		}
		wg.Go(func() {
			resp := serve(ctx, &req)
			mu.Lock()
			defer mu.Unlock()
			if err := enc.Encode(resp); err != nil {
				fmt.Fprintln(os.Stderr, "spawner:", err)
			}
		})
	}
	cancel()
	wg.Wait()
}

// serve runs the process req names, as the runner runs one directly.
func serve(parent context.Context, req *request) *response {
	resp := &response{ID: req.ID}
	ctx, cancel := context.WithTimeout(parent, req.Wall)
	defer cancel()
	cmd := exec.CommandContext(ctx, req.Path, req.Args...)
	cmd.Dir = req.Dir
	cmd.Env = req.Env
	if cmd.Env == nil {
		cmd.Env = []string{}
	}
	cmd.Stdin = bytes.NewReader(req.Stdin)
	stdout, stderr := &capped{max: req.Cap}, &capped{max: req.Cap}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	if cmd.ProcessState == nil {
		resp.StartErr = err.Error()
		return resp
	}
	resp.Killed = ctx.Err() != nil
	resp.Stdout, resp.Stderr = stdout.buf.Bytes(), stderr.buf.Bytes()
	resp.Over = stdout.over || stderr.over
	ps := cmd.ProcessState
	resp.Exit, resp.CPU = ps.ExitCode(), ps.UserTime()+ps.SystemTime()
	if ru, ok := ps.SysUsage().(*syscall.Rusage); ok && ru != nil {
		resp.RSS, resp.RSSOK = int64(ru.Maxrss), true
	}
	var ee *exec.ExitError
	if err != nil && !errors.As(err, &ee) && !resp.Killed && !errors.Is(err, exec.ErrWaitDelay) {
		resp.RunErr = err.Error()
	}
	return resp
}

// capped holds what a process writes to one stream, up to max bytes. It
// has no ReadFrom, so every write passes through Write.
type capped struct {
	buf  bytes.Buffer
	max  int
	over bool
}

func (c *capped) Write(b []byte) (int, error) {
	if room := c.max - c.buf.Len(); len(b) > room {
		c.over = true
		c.buf.Write(b[:max(room, 0)])
		return len(b), nil
	}
	return c.buf.Write(b)
}

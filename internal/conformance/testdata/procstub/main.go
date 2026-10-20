// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build linux

// Command procstub is the stub through which internal/conformance starts
// every process it measures on linux. Its arguments are the path of the
// process to run and its arguments. It runs it, and writes "usage" and its
// wait status, user and system CPU time in nanoseconds and ru_maxrss to
// descriptor 3, or "error" and why it could not start it.
//
// The process gets the stub's standard streams as they are and the stub's
// environment exactly, and is killed when the stub dies, so the runner's
// wall clock, which kills the stub, kills it too. The stub's thread is
// locked because the kernel sends that signal when the starting thread
// ends.
//
// It is a program of its own, built without the race detector whatever
// the test binary is built with, so that what each process costs beside
// the process is a small program's start: the test binary re-executed
// cost each process the race runtime's start and exit.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	report := os.NewFile(3, "report")
	if len(args) == 0 {
		fmt.Fprintln(report, "error no process named")
		return 2
	}
	runtime.LockOSThread()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Err = nil // the runner resolved the path
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = append([]string{}, os.Environ()...)
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

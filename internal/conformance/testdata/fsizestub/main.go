// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build unix

// Command fsizestub is the stub through which internal/conformance starts
// a step's process under the file-size limit its case's setup names
// (ADR-011 A6 item 3). Its arguments are the limit in bytes, then the path
// of the process to run and its arguments. It sets its own file-size
// resource limit, RLIMIT_FSIZE, to the limit, ignores SIGXFSZ, and
// executes the process in its place, with its environment and standard
// streams as they are: the process inherits the limit and the ignored
// signal, so a write that would make a file larger than the limit fails
// with an error rather than ending the process.
//
// Given "-probe" and a directory in place of the process, it instead
// reports whether the limit refuses a write: it writes one byte more than
// the limit to a new file in the directory, and exits 0 when the write
// fails and the file holds no more than the limit, 1 when the write
// succeeds, and 2 when it cannot try.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "fsizestub: usage: fsizestub <bytes> <program> [<argument>...]")
		return 2
	}
	limit, err := strconv.ParseUint(args[0], 10, 63)
	if err != nil || limit == 0 {
		fmt.Fprintln(os.Stderr, "fsizestub: the limit is not a positive number of bytes:", args[0])
		return 2
	}
	signal.Ignore(syscall.SIGXFSZ)
	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{Cur: limit, Max: limit}); err != nil {
		fmt.Fprintln(os.Stderr, "fsizestub: setting the file-size limit:", err)
		return 2
	}
	if args[1] == "-probe" {
		if len(args) != 3 {
			fmt.Fprintln(os.Stderr, "fsizestub: usage: fsizestub <bytes> -probe <dir>")
			return 2
		}
		return probe(filepath.Join(args[2], "fsize-probe"), int64(limit))
	}
	if err := syscall.Exec(args[1], args[1:], os.Environ()); err != nil {
		fmt.Fprintln(os.Stderr, "fsizestub: executing", args[1]+":", err)
	}
	return 2
}

// probe writes limit+1 bytes to a new file at p, and reports whether the
// write was refused at the limit.
func probe(p string, limit int64) int {
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fsizestub:", err)
		return 2
	}
	defer os.Remove(p)
	_, werr := f.Write(make([]byte, limit+1))
	f.Close()
	fi, err := os.Stat(p)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fsizestub:", err)
		return 2
	}
	if werr == nil || fi.Size() > limit {
		return 1
	}
	return 0
}

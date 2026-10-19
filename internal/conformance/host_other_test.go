// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build !unix

package conformance

import (
	"errors"
	"os"
	"time"
)

// rawMaxRSS reports no peak memory: ru_maxrss is a POSIX field, and the
// runner runs on darwin and linux (ADR-011 D15 item 6).
func rawMaxRSS(*os.ProcessState) (int64, bool) { return 0, false }

var start = time.Now()

// selfCPU approximates this process's CPU time by its age.
func selfCPU() int64 { return int64(time.Since(start)) }

// mkfifo cannot make a named pipe here; the fifos probe reports it.
func mkfifo(string) error { return errors.New("named pipes are POSIX") }

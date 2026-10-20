// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build !linux

package conformance

import (
	"context"
	"os"
	"os/exec"
)

// measured is the command that runs path with args, and the function that
// reads its usage once it has finished, or releases what it holds when
// given nil because the process never started. Off linux the process is
// started directly, with exactly env, and its own usage is read.
func measured(ctx context.Context, path string, args, env []string) (*exec.Cmd, func(*os.ProcessState) (usage, error), error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = env
	if env == nil {
		cmd.Env = []string{}
	}
	return cmd, func(ps *os.ProcessState) (usage, error) {
		if ps == nil {
			return usage{}, nil
		}
		return directUsage(ps), nil
	}, nil
}

// directUsage is the usage of the process ps describes.
func directUsage(ps *os.ProcessState) usage {
	u := usage{exit: ps.ExitCode(), cpu: ps.UserTime() + ps.SystemTime()}
	u.rss, u.rssOK = rawMaxRSS(ps)
	return u
}

// buildProcStub builds nothing off linux, where no process is started
// through a stub.
func buildProcStub(string) (string, error) { return "", nil }

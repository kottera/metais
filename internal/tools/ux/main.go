// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Command ux runs the UX scenario set of PLAN-2026-10-metais-ux §5 in
// nested, isolated agent sessions (docs/dev/ux-testing.md):
//
//	go run ./internal/tools/ux -out <dir> [-scenarios C0,C1] [-models opus,sonnet,haiku]
//
// run from the top of a clean checkout. It builds metais from the checkout
// with -buildvcs=false, packs each scenario's fixtures with it, and runs
// every scenario × model as one session of the harness -harness names, in
// a fresh directory of its own outside any repository, with its own HOME
// and TMPDIR. It makes the isolation checks on every run, writes each
// run's record beneath -out, paces itself, and prints a summary. Run again
// with the same -out, it resumes: a run already recorded is kept, and one
// a quota pause or an interruption left unfinished is run again.
//
// The runner is a developer's tool, never linked into metais, and is the
// one package ADR-009 Amendment A8 lets start a session, read the
// environment, write files and read the clock outside the release
// builder. No go test starts a session: a UX run is evidence, not a gate.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func main() {
	os.Exit(cli(os.Args[1:], os.Stdout, os.Stderr, os.Environ()))
}

// cli parses args and runs the set from the working directory, the top of
// the checkout, writing its progress and summary to stdout and a refusal
// to stderr. It returns the exit status: 0 when every run is recorded, 3
// when a quota pause stopped the set and a second run resumes it, 1 on a
// refusal, 2 on a usage error.
func cli(args []string, stdout, stderr io.Writer, environ []string) int {
	fs := flag.NewFlagSet("ux", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "", "the run set's directory, an absolute path outside any repository; given again, the set resumes")
	scenarios := fs.String("scenarios", "", "the scenarios to run, by id, comma-separated (default every scenario)")
	models := fs.String("models", strings.Join(defaultModels, ","), "the models to run each scenario on, comma-separated, as the harness names them")
	harnessName := fs.String("harness", claudeCodeName, "the agent harness the sessions run in; "+claudeCodeName+" is the only one implemented")
	withSkill := fs.Bool("with-skill", false, "install metais skill for each session before it starts")
	pace := fs.Duration("pace", 30*time.Second, "how long to wait between two runs")
	quotaWait := fs.Duration("quota-wait", 0, "on a quota pause, how long to wait before running the run again; 0 stops the set, to be resumed later")
	timeout := fs.Duration("timeout", 30*time.Minute, "how long one session may run before it is stopped and recorded as timed out")
	allowDirty := fs.Bool("allow-dirty", false, "run from a checkout with uncommitted changes, recording the commit as dirty")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 || *out == "" {
		fmt.Fprintln(stderr, "usage: go run ./internal/tools/ux -out <dir> [-scenarios <ids>] [-models <models>] [-harness <name>] [-with-skill] [-pace <d>] [-quota-wait <d>]")
		return 2
	}
	h, err := harnessByName(*harnessName, environ)
	if err != nil {
		fmt.Fprintln(stderr, "ux:", err)
		return 2
	}
	c := &config{
		root:       ".",
		out:        *out,
		scenarios:  splitList(*scenarios),
		models:     splitList(*models),
		harness:    h,
		withSkill:  *withSkill,
		pace:       *pace,
		quotaWait:  *quotaWait,
		timeout:    *timeout,
		allowDirty: *allowDirty,
		environ:    environ,
		stdout:     stdout,
		now:        time.Now,
		sleep:      time.Sleep,
	}
	switch err := c.run(); {
	case err == errQuota:
		fmt.Fprintf(stderr, "ux: a quota pause stopped the set; run the same command again to resume it\n")
		return 3
	case err != nil:
		fmt.Fprintln(stderr, "ux:", err)
		return 1
	}
	return 0
}

// splitList splits a comma-separated flag value, dropping empty items.
func splitList(s string) []string {
	var items []string
	for item := range strings.SplitSeq(s, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

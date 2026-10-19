---
name: verification-gate
description: The pre-merge gate for metais — AGENTS.md's Development list, run by this skill's gate.sh. Read this BEFORE running the gate, BEFORE publishing a test run as evidence in a pull request, and ALWAYS when a step goes red. Covers when to run it (once, last, after directed tests, self-review and rebase), how to run it (in the background, watched with gate-wait.sh), why it caps `go test -p` on a shared host, why the gate never passes -count=1, what to re-run after a fix or a rebase, how to tell whether a red is yours, and what a green gate cannot settle.
---

# The verification gate

The gate is the list under "Development" in `AGENTS.md`: build, vet,
`go test -race ./...`, tidiness, gofmt, the static build, the six-target
cross vet and build, and spec-data drift. `gate.sh` beside this file runs
exactly that list. Every step runs even when an earlier one goes red, so one
run reports everything that is wrong, and the run ends with a summary of
every step's verdict.

Run the gate through `gate.sh`, not a hand-written copy of the list. A
hand-written copy drifts from AGENTS.md, polls with blind `sleep`s, and
reports a different shape in every pull request.

## When to run it: once, last, right before merge

The gate takes many minutes, and on a shared host much longer. Most of that
time goes to packages your change does not reach. So it runs once, on the
finished tree:

1. **Make the change, and prove it with directed tests.** These are your new
   tests, the packages you edited, and the packages that import them:

   ```bash
   pkg=github.com/kottera/metais/internal/hostfs   # what you changed
   go list -f '{{.ImportPath}} {{join .Imports " "}} {{join .TestImports " "}} {{join .XTestImports " "}}' ./... |
     awk -v p="$pkg" '{for (i = 2; i <= NF; i++) if ($i == p) { print $1; break }}'
   ```

   A change under `spec/`, to a finding code, or to a command's flags also
   reaches the spec tripwires. Run them directly:
   `go test ./internal/guarantees -run 'TestStatementsWellFormed|TestCoverage|TestCitationsResolve'`,
   `go test ./internal/conformance -run 'TestVectorsWellFormed|TestConformance'`
   and `go test ./internal/cli -run TestHelp`. A red here costs seconds; in the
   gate it costs a run.
2. **Review your own change and fix what the review finds.**
3. **Rebase onto `origin/main`**, regenerate every generated file (your brief
   names them), and push. Rebasing before the gate means the gate certifies
   the head you merge.
4. **Run the gate** (below).
5. **On a red, read it before re-running anything** (the ladder below). Fix
   what is yours, re-run the failing tests and the packages your fix edited
   directly, then run `gate.sh` once more for the head you merge.
6. **Merge.**

The expensive mistake is running the gate as a development loop: a full run,
a red a directed test would have shown in seconds, a fix, and the full run
again.

## Running it: in the background, watched from the foreground

A foreground shell call is cut off at its time bound, and a gate run can
outlast it, so the gate runs in the background and you watch its log:

1. **Start it** with a background shell call (in Claude Code, Bash with
   `run_in_background: true`):

   ```bash
   .claude/skills/verification-gate/gate.sh
   ```

   If the harness has no background option, run
   `nohup .claude/skills/verification-gate/gate.sh >/dev/null 2>&1 &`.
2. **Watch it** with a foreground call, with a 590-second timeout, **repeated
   until it says `DONE`**:

   ```bash
   .claude/skills/verification-gate/gate-wait.sh
   ```

Each `gate-wait.sh` call blocks for up to 570 seconds, and returns early as
soon as there is something to act on. Its first line is one word:

| First word | Exit | What it means | What you do |
|---|---|---|---|
| `DONE` | 0 / 1 | The run finished. The summary and every failure line follow. | Green: merge. Red: the ladder below. |
| `NEW-FAILURE` | 3 | A failure appeared, and the run goes on. Each failure is shown once. | Read it. Do not edit the tree. Call `gate-wait.sh` again. |
| `RUNNING` | 2 | Nothing new in this window. | Call `gate-wait.sh` again. |
| `DIED` | 4 | The run is gone and never finished. Its log's tail follows. | Find out why from the tail, then start a new run. |

These rules make the background run work:

- **Do not end your turn while a run is live.** A dispatched session is one
  turn: ending it kills the session, and the background gate with it.
- **Do not edit the tree while a run is live.** `go test` builds packages as
  it reaches them, so an edit mid-run gives later packages a tree you never
  meant to test. If a `NEW-FAILURE` is plainly yours, let the run finish (the
  other steps' verdicts are still worth having), then fix it.
- **One run per checkout.** `gate.sh` refuses to start beside a live run.
- **Do not wrap it in a shell `timeout`.** The hang bound is `go test`'s own
  `-timeout` (ten minutes per package binary), which panics with a goroutine
  dump that names the stuck test. A shell kill throws that dump away.
- **The logs are under `.local/gate/`** (git-ignored); `latest.log` is the
  newest run.

## Why the gate caps `go test -p`, and never passes `-count=1`

The hosts that run this gate run several sessions at once, each with its own
gate. `go test` starts as many package binaries at once as the host has
cores, and every binary runs its own parallel tests, so three gates on one
host ask for many times the cores it has. The long packages (`hostfs`, `cli`,
`guarantees`, `finding`, `conformance`) then stretch from a few minutes to past
the ten-minute timeout. That is a wrong verdict caused by the gate itself,
not by the code. `gate.sh` passes `-p` at half the host's cores by default;
`METAIS_GATE_P` overrides it. **Never raise `-timeout` to get past a
timeout.** A test that times out has a mechanism to find (`root-cause`).

The gate leaves Go's test cache on. A package whose inputs did not change
since its last pass is not run again, which is what makes the second run
after a small fix cheap. Use `-count=1` for a directed re-run of one test you
are investigating, never for the gate.

## Is this red mine? The ladder

Every red is a defect with a mechanism. "Flaky" is not a verdict, and a green
re-run closes nothing. A red that does not reproduce on its own tells you the
failure depends on timing, or on what ran beside it. That is a lead, not an
acquittal.

1. **Read the failure, and the lines around it.** A `-timeout` panic names
   the test that was running and dumps every goroutine. A spec-tripwire
   failure names the file, the line and the statement number.
2. **Does your diff reach it?** Compare `git diff --name-only origin/main...HEAD`
   with the failing package and what it depends on:
   `go list -deps -test ./internal/<pkg> | grep kottera/metais`. A spec file,
   vector or generated file you changed reaches every test that reads it.
3. **Run it on its own:**
   `go test -race -count=1 -run '^TestName$' ./internal/<pkg>`.
4. **Find the mechanism** with the `root-cause` skill. It applies whether or
   not the red is yours.
5. **If it is yours, fix it.** If it is not, do not merge over it and do not
   call it flaky. Report it to whoever owns your work: in a Work Order, ask
   the orchestrator that filed it. Include the failure output and what you
   established about the mechanism. They decide whether it blocks you.

Rebase reds are deterministic, and they are yours. A statement number a
sibling took first, a regenerated golden, or conflict markers in a chapter
all reproduce every time. Fix them as the brief says: renumber your unmerged
statements, and regenerate generated files rather than merging them by hand.

## What a green gate cannot settle

- **Other operating systems' branches.** The cross step compiles and vets
  them; it runs none of their tests. Claim only what ran on your host, and
  name GOOS/GOARCH in the pull request.
- **Bounds measured in time.** `TestConformance` has a time bound (ADR-011
  D11 item 9). Report its time on your run, and say so plainly when it is
  near the bound.
- **A race the run did not schedule.** `-race` reports data races that
  happened in this run, not ordering bugs, and not races on paths the tests
  did not take.

## Publishing the evidence

In the pull request, record `go version` and GOOS/GOARCH, the gate's
`=== SUMMARY` lines, and anything your brief asks for. Quote the commands
you ran, as you ran them. Claim nothing you did not run.

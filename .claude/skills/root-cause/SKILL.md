---
name: root-cause
description: How to find out why something failed, with an instrument instead of repetition. Read this the moment a test fails, a behaviour is wrong, or a failure looks intermittent and you do not yet know WHY, before re-running anything. Names the two moves that beat sampling (make the failure deterministic, or make one occurrence sufficient), a catalogue of instruments keyed by the kind of failure (seam logging, forcing an interleaving, the timeout's goroutine dump, stretching a timing failure's interval at its seam rather than loading the host, CPU time instead of wall time, environment diffs, bisect last), the TEMP-DEBUG rule for temporary instrumentation, and how to confirm a fix so the green means something.
---

# Root-cause: find out why

A failed test is a **fact about the code**. A re-run is a question about the
weather. When something is red and you do not know why, the lazy move is to
run it again, twenty times, and come back with a *rate*. A rate is not an
explanation. It is the number you report when you have declined to find one.
The run that finally reproduces the failure tells you only what the first one
did: it fails.

This project aims at **zero intermittent failures**. Every red is a defect
with a mechanism, and it is cheapest to find while it is fresh. Nothing gets
recorded as "known flaky" and left alone.

## The two moves

Every technique worth using is one of these:

1. **Make the failure deterministic.** A race fails 1 time in 50 because the
   scheduler rarely picks the bad interleaving. Put a yield or a sync point at
   the seam you suspect and it fails 50 times in 50, and keeps failing until
   you fix it. That is also how you know you fixed it.
2. **Make one occurrence sufficient.** If you cannot force it, arrange for
   the *first* time it happens to tell you everything. Log at every writer and
   reader of the value that is wrong, and dump the goroutines and the state
   at the moment the assertion fails.

Sampling is what remains when you have chosen neither. Do not start there.

**Never load the host to make a failure show up.** CPU burners, parallel
copies of the suite, `-count=50`, anything whose purpose is to make the
machine busy so a timing failure happens more often: that is sampling with a
different sample. The host is also shared with other sessions and their
gates, and your load lands on them.

## Before any instrument: read what is already there

The answer is often in the failure output. A `-timeout` panic names the test
that was running and prints every goroutine's stack, so the blocked one is in
it. A spec tripwire names the file, the line and the statement number. A
comparison failure prints both values. Read the failure and the lines around
it *before* you touch anything.

Then shrink the problem: one test (`-run '^TestName$'`), one package,
`-count=1`, so that what you instrument is small enough to read.

## The catalogue: instrument by kind of failure

| What is wrong | Reach for | The trap |
|---|---|---|
| A value is wrong somewhere between two packages | Log it at **every writer and every reader**, run once, and read the line where it goes wrong | Logging only where you *think* it changes; you already guessed wrong once |
| Race or ordering | `go test -race` first. Then **force** the interleaving with a `runtime.Gosched()`, a sleep or a channel wait at the suspected seam. Temporary logs carry a goroutine identifier and a monotonic timestamp, or they mislead | Trusting a quiet `-race`: it reports data races that happened in this run, not ordering bugs |
| Timing, or "fails under load" | **Find the interval, then stretch it at the seam** (below). Time each stage with seam logging, so one red names the slow stage | Making the host busy to raise the rate. Widening the assertion or raising the timeout deletes the report, not the cause |
| A bound on work (render work, limits, the conformance run's time) | Measure **CPU time**, not wall time (AGENTS.md: bounded work is measured). Wall time on a shared host measures the host | Calling a wall-clock overrun a regression, or a wall-clock pass a proof |
| Hang | The `-timeout` panic's goroutine dump names the stuck goroutine and the call it is blocked in; set `GOTRACEBACK=all` for every goroutine. Shorten `-timeout` for the one test you are investigating, so the dump arrives in seconds instead of ten minutes | Reading only the top frame, or killing the run from the shell and losing the dump |
| A host-facing failure (a file, a path, a link, a device, open-file limits) | The `internal/hostfs` seam and `internal/testenv`'s hermetic `HOME` and directories. Reproduce the exact host state the test builds, then step through it | Assuming the host is the same as the test's fixture: a umask, a case-insensitive filesystem, a lowered `RLIMIT_NOFILE` |
| "Only fails there": one operating system, one host, the dispatch sandbox | **Diff the environment.** `GOOS`-specific files (`*_unix_test.go`, `*_windows.go`), `TMPDIR`, `HOME`, the `go env` output, the umask, what the sandbox denies. Reproduce that constraint locally | Treating the environment as a mystery; it is a variable |
| A guarantee a detector should catch | The fault switches in `internal/fault`. A fault breaks one guarantee at one named seam, to show its detector catches it. Read the seam the fault names, and you have the path the guarantee takes | Adding a fault to debug something: faults are permanent, assigned in the brief, and prove detectors. Temporary instrumentation is TEMP-DEBUG (below) |
| Mechanism still unknown after all of the above | `git bisect` over **commits**: last, and never over re-runs | Bisecting first, because it feels like progress |

`dlv` (delve) is the instrument for hidden state, such as a wedged goroutine
or a `nil` from a caller you cannot identify. Check `dlv version` first. It
is not part of the Go toolchain and may not be installed. It also builds
without `-race` and perturbs timing, so it is the wrong tool for a race.

## "It fails under load" is a symptom, not a mechanism

Load does one thing: it stretches some interval. Somewhere a wait depends on
that interval being short: a deadline, a timeout, a polling loop, an order one
goroutine assumed it would win. Find which wait, and make the interval long
on purpose, at its seam:

1. **Read the bound in the failure.** The red names it: a `-timeout` panic,
   `context deadline exceeded`, a time limit in an assertion. Note the bound
   and what the test was waiting for.
2. **Find what it waits on.** Follow the awaited value or event back to the
   code that produces it, and list the stages between the start of the wait
   and that producer: a subprocess, a file write, a rename, a goroutine that
   must be scheduled.
3. **Time each stage with seam logging.** Put a `TEMP-DEBUG` line at each
   stage boundary, with a monotonic timestamp, so the next red names the
   stage that used up the time.
4. **Stretch that stage deliberately.** Add a `TEMP-DEBUG` hook at the seam:
   a `time.Sleep`, a channel the test holds closed, or a stub that sleeps
   before it answers. If the failure now happens every time on an idle host,
   you have the mechanism. If it does not, that stage was not the one; move
   the hook.

The fix makes the wait correct however long that stage takes, and the hook
proves it: with the delay still in place, the test passes.

Sometimes the "wait" is the gate itself. A package whose own run takes four
minutes reaches the ten-minute `go test` timeout when several gates share
the host. That mechanism is oversubscription, and the `verification-gate`
skill's `-p` cap is its fix. Establish it by measurement: the package's time
alone, and its time in the full run. Do not assume it.

## Temporary instrumentation: how it leaves

Temporary logs, forced yields and capture-at-failure hooks are encouraged.
They are also temporary, and a branch that ships them is worse than the bug.
So:

- **Tag every line of it `TEMP-DEBUG`**, in the comment, the log prefix or
  the identifier, so one search finds all of it.
- **Before you commit, `git grep -n TEMP-DEBUG` prints nothing.** Do not
  rename the marker to get past it.
- What you learned goes in the fix's commit message, or in a comment on the
  test that now pins it. Never in a permanent log line kept "in case it
  happens again".

## Confirm the fix so the green means something

A green you cannot explain is not evidence. The order matters:

1. **State the mechanism** in one sentence: what was wrong, and why the fix
   makes the failure impossible rather than unlikely.
2. **Pin it.** Where you could, keep a test that fails without the fix: the
   forced interleaving, or the stretched stage turned into a fixture. This is
   AGENTS.md's "pin the property false first", applied to a bug.
3. **Predict what else the fix would break**, and look there: the other
   operating systems' branches, a second fixture with different limits, a
   caller that relied on the old behaviour.
4. **Only now repeat it**, and state the base rate. "10 of 10 green, was 2 in
   5 red" is a result; "10 of 10 green" is just a number.

## When the red is not yours

Whether the red is yours is a separate question from why it happened. The
`verification-gate` skill's ladder answers whose it is. The mechanism still
needs finding either way. Report what you established, with the failure
output, to whoever owns your work, so that the person who fixes it starts
from your evidence and not from a re-run.

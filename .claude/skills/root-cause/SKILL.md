---
name: root-cause
description: How to find out why something failed, with an instrument instead of repetition. Read this the moment a test fails, a behaviour is wrong, or a failure looks intermittent and you do not yet know WHY, before re-running anything. Names the two moves that beat sampling (make the failure deterministic, or make one occurrence sufficient), a catalogue of instruments keyed by the kind of failure (seam logging, forcing an interleaving, the timeout's goroutine dump, stretching a timing failure's interval at its seam rather than loading the host, CPU time instead of wall time, environment diffs, bisect last), metais's own instruments and the failure each is for (internal/fault's bits, internal/testenv, the guarantee detectors and their fault controls, the host-isolation canaries, CPU-time measurement and the work counters, the gate's share line that says whether a red package was starved), why a failure under load is a defect with a mechanism and passing alone closes nothing, the TEMP-DEBUG rule for temporary instrumentation, and how to confirm a fix so the green means something.
---

# Root-cause: find out why

A failed test is a **fact about the code**. A re-run is a question about the
weather. When something is red and you do not know why, the lazy move is to
run it again, twenty times, and come back with a *rate*. A rate is not an
explanation. It is the number you report when you have declined to find one.
The run that finally reproduces the failure tells you only what the first one
did: it fails.

This project aims at **zero intermittent failures**. Every red is a defect
with a mechanism, and it is cheapest to find while it is fresh. There is no
list of tolerated failures, and nothing is recorded as "known flaky" and
left alone: a red is fixed at its mechanism, or it stays open as a defect
with the evidence you gathered.

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
| A guarantee a detector should catch | The fault switches in `internal/fault`. A fault breaks one guarantee at one named seam, to show its detector catches it. Read the seam the fault names, and you have the path the guarantee takes (metais's instruments, below) | Adding a fault to debug something: faults are permanent, assigned in the brief, and prove detectors. Temporary instrumentation is TEMP-DEBUG (below) |
| A host fact in an output, a finding or a stream | The host-isolation harness and its canaries (below): the sentinel names its channel, and the A/B differential names the stream | Grepping the output for your own user name: the sentinels exist so that the hit says where it came from |
| Mechanism still unknown after all of the above | `git bisect` over **commits**: last, and never over re-runs | Bisecting first, because it feels like progress |

`dlv` (delve) is the instrument for hidden state, such as a wedged goroutine
or a `nil` from a caller you cannot identify. Check `dlv version` first. It
is not part of the Go toolchain and may not be installed. It also builds
without `-race` and perturbs timing, so it is the wrong tool for a race.

## "It fails under load" is a symptom, not a mechanism

A test that times out, or fails only under load, is a defect with a
mechanism to find. Look first for one of these four, and fix the one you
find:

- **a timing assumption**: an order one goroutine or process assumes it
  wins, or a stage assumed to finish within some interval;
- **an unbounded wait**: a read, a lock or a process wait with nothing that
  ends it, which a busy host only makes long enough to see;
- **a test that charges wall time**: a bound on work asserted against the
  clock, so the host's other work is counted as the test's (CPU-time
  measurement, below);
- **a gate that oversubscribes the host**: more packages, processes or
  parallel tests at once than the cores can run, so every clock in them
  includes waiting for a turn.

**"It passed alone" closes nothing.** A test that passes run by itself and
fails in the full run has told you the mechanism needs company to show,
which narrows the four above; it has not told you the code is right. **No
assertion is widened and no timeout raised to hide it**: either deletes the
report and keeps the defect.

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
skill's `-p` cap is its fix. Establish it by measurement, and the gate
measures it on every run: read the red's share line (below). Do not assume
it.

### The gate's share line: was the red starved?

The gate runs every test binary through `gate-exec.sh`, which takes the
binary's CPU from the kernel's accounting of the processes it reaped, so
measuring runs nothing beside the test. A red package's `FAIL:` line in
the gate's output carries what it got:

```
FAIL: github.com/kottera/metais/internal/conformance: got 1.30 of 4 cores (CPU 812.4s over 615.0s wall, beside 0.2 other test binaries); host load 25.31 on 16 cores; last green scoped-20261020T101500Z got 4.10 of 4 cores (CPU 2350.1s over 493.0s wall), CPU now 0.35x it
```

- **got … of p cores** is the package's CPU over its wall time, against
  the gate's share `p`. The CPU is user plus system time of its test
  binary and of every process it waited for: the tool, stub and spawner
  processes its tests run, and the builds its tests start. Packages that
  run together share `p`, so **beside** says how many other test binaries
  ran beside it, on average over its wall time.
- **host load … on … cores** is the host's one-minute load average when
  the binary ended, against the host's cores. A load above the cores means
  every process on the host was waiting for a turn.
- **last green** is the same package's newest PASS in `.local/gate/`, from
  any earlier run, with its own figures, and **CPU now …x it** compares the
  two runs' CPU. A timed-out binary's CPU is what it had used when it was
  killed, so it can only be at most its full run's CPU.
- The `.record` keeps every tested package's figures as its fifth field,
  and `--resume` carries them with a verdict it reuses. The test step's
  `=== PASS test` or `=== FAIL test` line gives the cores the whole
  `go test` got against `p`, its builds included.

A package can get few cores and not be starved: one that mostly waits on a
sleep, a lock or a child gets few cores on an idle host too. That is why
the line compares with the package's own last green, not only with `p`.
Read it by these cases:

| The red's share line | What it is | What you do |
|---|---|---|
| **Starved:** its cores are well under its last green's, or with no green well under `p` with nothing beside it. Its CPU matches its last green's (for a timeout, it is no more than the green's). A host load above its cores is the usual cause; a limit on the run's own cores (a cgroup, an affinity mask) starves it with the load low | **No evidence about the change either way.** The host gave the gate too little. The red is not a pass and not a defect of the change: it is no verdict | Do not merge on it. Wait until the host can give the share (`gate-exec.sh --load` below the host's cores), then `gate.sh --resume`, which runs the red package again at the same share and timeout. Its new line decides: at its full share, its verdict stands, red or green. If the host stays loaded, report the red with its share lines to whoever owns your work, and do not merge |
| **At its full share:** its cores are those of its last green, or, with no green, the host's load is below its cores and nothing limits the run's cores | **A defect to root-cause.** The share did not cause it | This skill, from the top: one of the four mechanisms above, or the change |
| **More work:** its CPU is well above its last green's | **The package did more work**, whatever the share | Find by what, with the counters and a CPU-time measurement (below) |

None of these raises a timeout, enlarges the share or lowers `-p` for one
package: `METAIS_GATE_P` stays what it is. A starved red's re-run is the
gate's own verdict under the same share and the same timeout, taken when
the host can give that share. It counts only because the first red was
measured as starved, with its CPU unchanged. "It passed alone" still closes
nothing, and a red at its full share that passes on a re-run still has a
mechanism to find.

`internal/conformance/wallclock_test.go` is a worked example. The
self-proof's rows ran out their 5 s wall clock under the full parallel
`go test -race` on processes with nothing wrong with them, and two
mechanisms, neither the implementation's, were spending the clock: the race
runtime sleeps a second before every race-built process exits
(`atexit_sleep_ms`), and the self-proof started some eighty race-built
processes at once on a few processors. Each is fixed at its mechanism
(`GORACE` with `atexit_sleep_ms=0`, and `procSlots` starting at most
`GOMAXPROCS` processes together) and pinned by a test, and the clock did
not move.

## metais's own instruments

These are already in the tree. Each is built for a kind of failure; read
the code it names before you add anything of your own.

### `internal/fault`'s bits

`fault.Set` is a `uint64`, one bit per fault, and `Set.Has(f)` is true only
for a non-empty `f` wholly in the set. A fault breaks one headline
guarantee at one named seam, so that the guarantee's detector can be shown
to catch it in every `go test` (ADR-010 D13). Faults are injected through
options, never a flag or the environment: `internal/cli` builds a zero
`Set`, and `TestFaultsOnlyAtSeams` in `internal/guarantees` fails when a
non-test file outside `internal/fault` and the seams names one.
`TestFaultBits` pins each bit:

| Bit | Fault | Seam | What it breaks | The detector that must see it |
|---|---|---|---|---|
| 0 | `SkipDigest` | `internal/pkgtree` | digest verification returns success | `TestDigestDetector` |
| 1 | `FirstFinding` | `internal/finding` | a report keeps only its first error | `TestOnePassDetector` (SC8's planted errors) |
| 2 | `UnpackToTemp` | `internal/browse` | browse reads an entry by extracting it to a temporary file | `TestReadInPlaceDetector` |
| 3 | `SkipConfine` | `internal/hostfs` | the output path check returns success | `TestConfinementDetector` |
| 4 | `MapOrder` | `internal/template` (`Env.UnorderedObjects`) | object members iterate in Go map order | `TestDeterminismDetector` |
| 5 | `ReadExisting` | `internal/hostfs` | a write into an existing directory reads what it replaces | `TestExistingDestinationDetector` |
| 6 | `RemoveForeign` | `internal/hostfs` | a write into an existing directory removes files it did not plan | `TestExistingDestinationDetector` |
| 7 | `StdoutNotice` | `internal/cli` | a render to standard output writes a notice before the bytes | `TestStdoutPurity` |
| 8 | `UpdateBeyondExpected` | `internal/cli` | `test --update` removes files outside `expected/` | `TestUpdateConfinement` |

**The kind of failure it is for: a detector's red, and which half of it is
red.** Every detector runs twice: on the tool, where it must find nothing,
and with its fault, where it must find something. A red on the tool is the
guarantee broken; follow the seam the fault names, which is the path the
guarantee takes. A red on the control ("the detector caught 0 of …") is
the detector gone blind: a snapshot taken in the wrong place, a sentinel no
longer planted, a fixture that stopped exercising the seam. The tool may
be fine. Fix the detector, never by dropping the control.

### `internal/testenv`

`testenv.Home(t)` gives a test a fresh, empty home directory and points
`HOME` and `USERPROFILE` at it until the test ends; `testenv.Dir(t)` gives a
fresh directory removed when the test ends. The package is built only on
the testing package's own helpers, so it holds no host access of its own.
A test that calls `Home` sets the environment, so it never calls
`t.Parallel`.

**The kind of failure it is for: "passes on my machine", "fails only in the
sandbox", or a test whose result depends on what ran before it.** The first
question is whether the test is hermetic. One that reads the real `HOME`,
writes to a fixed path or shares a directory with another test is the
suspect; put it on `testenv` and the difference between hosts either goes
away or becomes a variable you can set.

### The detectors

The detectors live in `internal/guarantees/`, except `TestStdoutPurity`
and `TestUpdateConfinement`, which are in `internal/cli`. Each has its own
instrument in its failure output, which is the one occurrence that is
enough:

- **Confinement** (`TestConfinementDetector`) snapshots the tree around the
  output directory (path, type, size, SHA-256) before and after, and names
  every path that changed outside it.
- **Digests** (`TestDigestDetector`) runs the hostile corpus's
  digest-mismatch members, and names the member that was not refused.
- **Read in place** (`TestReadInPlaceDetector`) points `TMPDIR`, the
  working directory and `HOME` at snapshotted directories, and names what
  appeared in them.
- **Determinism** (`TestDeterminismDetector`) renders each vector three
  times in one process at `GOMAXPROCS` 1 and `runtime.NumCPU()`, and once
  in a fresh process under `TZ=Pacific/Chatham`, `LC_ALL=tr_TR.UTF-8`,
  `umask 077` and `GODEBUG=zipinsecurepath=0,tarinsecurepath=0`, and
  compares SHA-256s. When it is red, those settings are the variables:
  render the one vector under each alone, and the one that changes the
  bytes is the input the output wrongly depends on.
- **One pass** (`TestOnePassDetector`) plants SC8's errors and names those
  the report missed.
- **Existing destination** (`TestExistingDestinationDetector`), standard
  output purity (`TestStdoutPurity`) and `test --update`'s confinement
  (`TestUpdateConfinement`) each name the case and what it found.
- **No network, no execution** (`TestNoHostFacts`) runs the import rule and
  the host-API symbol rule over `cmd/metais`'s closure as `go list -deps`
  reports it, for darwin, linux and windows. Its controls are
  `testdata/netleak`, a main package importing `net`, and the
  `testdata/hostfacts` fixture, and a red names each violation and the
  target it was found for.

**The kind of failure they are for: a headline guarantee broken.** Read
what the red names before anything else. A guarantee red is never made
green by narrowing what the detector looks at: plan §4's "do not widen a
refusal to make a proof pass" holds for detectors as well.

### The host-isolation canaries

`TestHostIsolation` in `internal/conformance` runs the built binary over
every vector in a sentinel environment A and a plain environment B
(ADR-009 D16 item 1). Its checks are: no sentinel in any stream; none of
the real host's names or paths; A and B byte-identical; and nothing
written outside a declared output. A sentinel is letters only, begins
`MTSSENTINEL` and names its channel (`MTSSENTINELHOME`), so a hit says where
the fact came from.

The canaries keep that harness honest (ADR-009 D16 item 2). Under a build
tag `metais_canary_<channel>`, a file `canary_<channel>.go` inserts one
realistic leak: `env`, `cwd`, `home` and `pid` in `internal/template`, `tz`
and `clock` in `internal/dates`, and `path` in `internal/cli`.
`TestHostIsolationCanaries` builds every canary binary in every `go test`
and requires the harness to report a violation on that canary's channel,
and requires none of the release build. They are build tags rather than
faults because the harness drives a subprocess, which no injected option
reaches.

**The kind of failure they are for: a host fact reaching a template, an
output or a finding (SC7), and a harness that can no longer see one.** A
red `TestHostIsolation` names the channel by its sentinel, or the stream by
the A/B difference; the canary for that channel shows the shape such a
leak takes. A red `TestHostIsolationCanaries` is the harness gone blind on
one channel; build that canary with its tag and run its probe vector by
hand to see what the harness missed.

### CPU-time measurement, and counters before either

A bound on work is proved by counting the work, and only where a duration
must be asserted by the process's CPU time (ADR-010 D14 item 7):

- **Counters first.** The Liquid meter (`internal/liquid/meter.go`) counts
  each kind of work and reads no clock, so the same work is refused at the
  same charge on every machine. `TestOpenReadsEveryByteOnce` in
  `internal/zipfmt` counts the bytes a counting `ReaderAt` reads, with
  `TestReadOnceControl` as its control. A counted bound cannot be red
  because the host was busy.
- **CPU time.** `selfCPU` in `internal/cli/cpu_unix_test.go` is
  `getrusage(RUSAGE_SELF)`'s user plus system time, and reports that it did
  not measure off unix (`cpu_other_test.go`), so a bound there is skipped,
  not timed. `internal/conformance` reads its own (`host_unix_test.go`) and
  each child's: `measure_linux_test.go` starts the child through a stub,
  because on linux a child's `ru_maxrss` counts the address space it was
  executed from. Off unix, `host_other_test.go`'s `selfCPU` is the
  process's age: wall time, so it proves nothing about CPU there.

**The kind of failure it is for: a time bound red, above all one that is
red only under load.** Measure the stage in CPU time and in wall time. If
the CPU time is within the bound and the wall time is not, the host's other
work is in the clock: the test charges wall time, or the gate
oversubscribes the host, and that is the defect to fix. If the CPU time is
over the bound too, the work grew; find out by what, with the counters.

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

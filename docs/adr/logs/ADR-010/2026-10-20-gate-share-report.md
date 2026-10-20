# ADR-010 — the gate reports the share of the host a red package got

**2026-10-20, outside a train, in support of PLAN-2026-10-metais-v1.** In
Wave 7, W7-C2 and W7-C4 each spent most of a session proving by hand that a
package had timed out because the host gave the gate less than its share
(Amendment A5 item 7), not because of the change. C2 wrote a busy-work
probe. C4 compared four standalone runs of `internal/conformance`. Its first
comparison had already settled it: the same CPU on the branch and on
`main`, and only the wall time different. The gate measured neither, so
every session re-derived it at about ten minutes a sample. The gate now
measures it on every run, and one run says whether a red was starved.

**The measurement.** Every `go test` the gate runs, in the test step, the
tripwires and the reach groups, starts its test binaries through
`gate-exec.sh`, as `go test -exec`. For each binary the wrapper appends a
line to `<run>.cpu` in `.local/gate/`. It records the package's directory,
the start and the end, the CPU, the binary's exit status, and the host's
one-minute load average when the binary ended.

- **The CPU** is the binary's user plus system time and that of every
  process it waited for: the tool, stub and spawner processes, and the builds
  its tests start. The wrapper reads it with bash's `times`, from the
  kernel's accounting of the processes its shell reaped. So measuring runs
  nothing beside the test, and takes no CPU from the host. No probe burns
  CPU to sample the share. That is the finest grain the gate can measure
  honestly: one test binary, which is one package.
- **The start and the end** are microsecond times where bash has
  `EPOCHREALTIME`, and whole seconds where it does not.
- **The compiles are counted only for the step.** The CPU of compiling and
  linking the test binaries belongs to no one package. The test step's
  `=== PASS test` or `=== FAIL test` line gives the cores the whole
  `go test` got against `p`, its builds included. It is measured the same
  way, with `times` in the gate's own shell around the step.
- **The load** comes from what the host already exposes: `/proc/loadavg`,
  darwin's `vm.loadavg`, or `uptime`. A sandbox can hide all three, as the
  linux dispatch image does, and linux still answers `sysinfo(2)`. For
  that case `gate.sh` builds the build-ignored `loadavg.go` once per run.

`gate.sh` maps each line's directory to its import path with one
`go list`. From the line it derives:

- the package's **cores**, its CPU over its wall time;
- **beside**, how many other test binaries of the run ran beside it, on
  average over its wall time, from the start and end times. A package
  beside three others under `-p 4` shares four cores.

**What it reports.** Each package's verdict in `<run>.record` gains a fifth
field:

```
cpu=<s> wall=<s> cores=<cpu/wall> beside=<n> p=<share> load=<load>/<cores>
```

A tripwire or reach step's verdict gains it too. `--resume` carries the
field with every verdict it reuses. A red package's `FAIL:` line, which
`gate-wait.sh` shows among its failure lines, reads:

- the cores it got against `p`;
- its CPU-seconds and wall time, and what ran beside it;
- the host's load against its cores;
- its newest PASS in any earlier run under `.local/gate/`, with that run's
  figures and the ratio of the two CPUs.

Each green package gets a `measured <pkg>:` line in the log.

**How to read it** is in `root-cause` ("The gate's share line: was the red
starved?"), with the matching step in `verification-gate`'s ladder. The
reading compares a red with the package's own last green, not only with `p`.
A package that mostly waits gets few cores on an idle host too.

- **A starved red** has cores well under its last green's and its CPU no
  more than the green's. It is no evidence about the change either way.
  The next step is `gate.sh --resume` once the host can give the share.
  Nothing merges over the red meanwhile.
- **A red at its full share** is a defect to root-cause.
- **A red with more CPU than its green** did more work.

No timeout, assertion, test or share changed. "It passed alone" still
closes nothing.

**Pinned false first**, on the linux/amd64 dispatch host (12 cores, `-p 4`),
with `GOFLAGS=-timeout=60s` set for these two runs only, so that a package
goes red without loading the host:

- **At its full share.** `internal/guarantees` timed out at 1.26 of 4 cores
  against its last green's 1.41, with CPU 75.7 s at the kill:
  `got 1.26 of 4 cores (CPU 75.7s over 60.1s wall, beside 1.1 other test binaries); host load 12.45 on 12 cores; last green scoped-20261020T173156Z got 1.41 of 4 cores (CPU 156.7s over 111.4s wall), CPU now 0.48x it`.
- **Starved.** The same run under `taskset -c 0` had one core for a share
  of 2. `getconf _NPROCESSORS_ONLN` reports the affinity mask's count
  there, so the default share fell to its floor. The line reads
  `got 0.53 of 2 cores (CPU 32.2s over 60.2s wall, beside 1.0 other test binaries); host load 8.96 on 1 cores; last green … got 1.41 of 4 cores …, CPU now 0.21x it`.
  The step reads `the gate got 0.98 of 2 cores`.

**What it cannot see.**

- A binary the go command kills, rather than one that panics at its own
  `-timeout`, never returns to the wrapper. Its line is missing, and its
  `FAIL:` line says it was not measured. The go command kills only a
  minute after `-timeout`, so a timeout's own panic is measured.
- Where `getconf` counts only the cores a mask allows, the host load is
  compared with those cores, not with the host's.

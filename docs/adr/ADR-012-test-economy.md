# ADR-012: Test economy — each property is proved once, along each of its axes, at the smallest size that can fail it; the reference tool's conformance run is one pass checked against a committed record; and the work a test does is budgeted by count

**Status:** Accepted 2026-10-21 by Ed Funnekotter, every lean of the
plan's §11 as drafted. It was proposed 2026-10-20 by the
`metais-test-economy` design Work Order, under his direction of that day.
Each decision takes effect in the tree as the item of
[PLAN-2026-10-metais-test-economy](../plans/PLAN-2026-10-metais-test-economy.md)
that carries it lands.
**Date:** 2026-10-20
**Deciders:** Ed Funnekotter
**Designs:** how the reference implementation's own tests prove ADR-005,
ADR-009, ADR-010 and ADR-011's guarantees per change, and what they cost.
**Amends:** ADR-005 D19 item 2 (A2); ADR-009 D16 items 1 and 2 (A7);
ADR-010 D2 item 3, D13 item 4 and D14 item 8 (A6); ADR-011 D11 items 2, 7
and 9, D12 items 4 and 5, D14's `TestHostileVectorsOpenEverywhere` row,
D15 item 1 and D16 item 3 (A7). Each amendment states the new rule in its ADR and points
here for the reasons.
**Depends on:** ADR-010 Amendment A5 (the two gate tiers), and the
gate-selection fix filed beside this design in the same cohort, which
decides which packages a change reaches. This ADR states what each test
proves and where its inputs are. It does not redesign the selector.

## Context

### The incident

On 2026-10-20 the scoped gate of metais-v1 [W7-NB2] timed out on four runs
in a row. `internal/conformance` and `internal/guarantees` reached the
ten-minute `go test` limit, and `internal/cli` took 583 s. Four Wave 7 items
were sharing one 16-core runner with other work, at a load average of 41 to
44. Ed asked why the tests of a fairly straightforward tool take this long,
and had them audited.

The audit found that about 1,100 of the 1,211 top-level tests together cost
seconds. Nearly all of the time goes to about fifteen tests, and most of
that time buys nothing: in several places the expensive version proves
less than a cheap one would. Ed's direction: prove each property once, at
the size that can make it fail, rather than running minutes of testing
over and over. On the two seconds between environments A and B: "If the 2
seconds only happens a handful of times, then I am fine with that."

### Measured

Two hosts, each package run **alone** under the gate's settings
(`go test -race -count=1 -p 1 -parallel 4`), with a warm build cache. CPU
is user plus system time of the whole `go test`, children included
(`/usr/bin/time`). A parallel test's own time is counted from its first
`cont` event in `go test -json`, never from `run`: from `run` it includes
the time it queued for a slot. A test's own CPU is taken from a run of that
test alone (`-run '^Name$'`), less the package's `TestMain`.

| Package | Apple M, 12 cores (audit, `647c0cf`) | linux/amd64, 12 cores (this design, `7c3b362`) |
|---|---|---|
| `internal/conformance` | 275 s wall, 959 CPU-s | 287 s wall, 1,492 CPU-s (1,027 user, 466 system) |
| `internal/cli` | 98 s, 194 CPU-s | 180 s, 379 CPU-s |
| `internal/guarantees` | 88 s, 126 CPU-s | 138 s, 228 CPU-s |
| `internal/tools/release` | — | 96 s, 325 CPU-s |
| `internal/hostfs` | — | 36 s, 36 CPU-s |
| `internal/author` | — | 8 s, 8 CPU-s |
| `cmd/metais` | — | 4 s, 5 CPU-s |

The whole tree under plain `go test` took 149 s of wall time and 1,224
CPU-s, 644 s of it system time, on the audit's host. The per-test figures
behind each decision are in that decision and in the plan's items.

Two traps make the cost look other than it is, and both are now rules of
measurement (D12):

- **Queue time.** `TestDriverProtocol`'s time from `run` to `pass` is
  235 s; its own work, from its first `cont`, is about 52 s on the audit's
  host.
- **Contention.** Run beside the whole tree, the mid-cost tests inflate
  about five times. In this design's package run, `TestCasesWaitTogether`
  showed 55 s and `TestUnmetRequirementIsSkippedAndReported` 64 s from
  their first `cont`, because they ran beside `TestDriverProtocol`'s
  suite. Alone, each costs under a CPU-second beyond the package's
  `TestMain`, and its wall time is a sleep (D12).

### Where the time goes

`internal/conformance` runs the whole vector suite three times, and the
biggest of the three twice over:

- `TestConformance` (`internal/conformance/main_test.go:217`) runs every
  case in environments A and B against a race-built tool: about 5,881
  processes.
- `TestHostIsolation` (`hostisolation_test.go:105`) runs every case in A
  and B, in two path modes each, against a release build: **11,766
  processes**, logged at `hostisolation_test.go:112`.
- `TestDriverProtocol/suite` (`driver_test.go:150-157`) runs every case in
  A and B through the reference driver, which wraps the tool: about 3,200
  processes.

Of each environment's processes, 79 % come from one rule. Every `open`
step of a `security` vector runs as all 14 package-opening commands
(`adapter_test.go:199-209`, ADR-011 D16 item 3). There are 165 such steps
among the suite's 780, so they make 2,310 of each environment's 2,925
processes.

The other packages' cost sits in a handful of tests. Each brute-forces a
size or a corpus that its property does not need. D5 to D10 take them one
by one.

## Decision

### D1 — The principle

**A test proves a property along each of its axes, at the smallest size
that can make it fail. The proof is made once per change. Any repetition
is an axis the property has, never the same proof run again.**

1. **Name the axes.** A property is a claim about how something behaves as
   a quantity grows or a condition varies: handles held as directories,
   files and depth grow; bytes identical as the process, environment, path
   mode and day vary; every opening command refusing as the hostility
   varies. The test varies each axis past the point where a defect along it
   shows, and no further.
2. **Full-scale counts live in calibration.** Writing ADR-009 D3's default
   count of a limit proves that the default fits on the host that ran it.
   That is a calibration fact, measured per target at release (ADR-009 D3
   item 3, D18 item 4). It is not proved per change, because a change cannot
   make a host's limits smaller.
3. **A corpus walk names what one representative per code path would
   miss.** A test that runs every vector or every package through
   something must say, in its comment, what a representative of each code
   path it reaches would not catch. If nothing, it runs the
   representatives.
4. **A slow test is asked what it proves before it is made parallel.**
   Parallelism hides cost from the wall clock. It does not remove it from
   the host, which is shared (ADR-010 A5 item 7).
5. **An incident in the test infrastructure is answered with a unit test
   of the fix.** It is not answered with a run of the suite that would have
   shown the incident.
6. **No guarantee is dropped silently.** A test that is shrunk, merged,
   moved or deleted names, in its change, the property it proved and what
   still proves it. Where the reworked test guards a property, the change
   shows it red first, against the defect it exists to catch.

AGENTS.md carries items 1 to 5 as conventions (D13).

### D2 — The reference tool's conformance run is one pass, checked against a committed record of environment B

**On every change, each case runs once, in environment A, against the
release build. Its every observed byte is compared with the vector's
expectations, and with a committed record of the same case run in
environment B. Live A/B, with B two seconds after A, stays where it proves
something a record cannot: the canaries, the runner's own proof, and the
published method an implementation and a release are run by.**

1. **The pass.** `TestConformance`, against the reference tool built from
   source (its default), runs every case once:
   - in ADR-009 D16 item 1's environment A: ADR-011 D12's sentinels plus the
     harness's further variables and decoy trees;
   - with its data and output paths absolute, under a directory whose name
     holds a sentinel (ADR-009 D16's absolute path mode);
   - against the release build (D4).

   Each step is held to everything the two runs check today:
   - the vector's expectations (ADR-011 D10);
   - the sentinel scan (check 1) and the scan for the real host's facts
     (check 2);
   - nothing written outside a declared tree (check 4), over the whole
     sandbox as ADR-009 D16 check 4 already says, not only `home/`, `tmp/`
     and `work/`;
   - the bounds, and the crash and exit-code checks;
   - check 3, against the record (item 2).

   A repeatable step still runs a second time in A. The pass is
   `t.Parallel` at the top level, and counts its own processes (D9).
   `TestHostIsolation` keeps its name only for the every-command reach of
   ADR-009 D16 item 1, under `-conformance.reach`, so the wave tier's
   command in AGENTS.md is unchanged. Without the flag it runs nothing,
   as `TestCalibration` runs only under `-calibration.record`, and says
   that `TestConformance` is the run that covers it.
2. **The record.** One file per vector, at
   `internal/conformance/testdata/record/<area>/<slug>.json`. It is outside
   the suite tree, because it is a fact about the reference tool, not part
   of the specification. It holds:
   - the SHA-256 of the vector's directory, so that a vector that changes
     leaves its record stale;
   - the tool's name and its fixed release-line version, `0.0.0-sc7`
     (`hostisolation_test.go:172`);
   - for every step of every case, and for every command of the step's
     **full** plan (all 14 for a security `open` step), ADR-011 D15 item
     3's fields: `ok`, and the SHA-256 and length of standard output,
     standard error, the stream and the listing, and every file of every
     declared tree with its path, size, SHA-256 and execute bit.

   No exit status is recorded. A step's exit status is a function of `ok`
   and the findings' classes (ADR-008 D13). The adapter checks that function
   in A on every step, and the findings are part of standard output, whose
   digest is compared. So equal standard output and a passing exit check
   give an equal exit status.

   ADR-011 D14 rejected one committed coverage file because it would
   conflict on every pair of changes that each add a vector. A record per
   vector conflicts only when two changes alter the same vector's output.
   When they do, ADR-011 A5's merge driver keeps one side, and
   regeneration rewrites it.
3. **Regeneration.** `regenerate.sh` gains a step that runs
   `TestConformance` with `-conformance.regenerate`. It runs environment B
   (ADR-011 D12 item 4: plain, `TZ=UTC`, `LC_ALL=C`), with relative paths,
   over the full plan, and applies to B every check that applies there:
   expectations, check 2, check 4, the bounds and the crash check. It
   writes a vector's record only when every case of the vector passes, and
   it rewrites only records that are missing, stale or different. The new
   file gets a line in `.gitattributes`, as AGENTS.md requires.
4. **A record that differs is root-caused, never regenerated to make it
   pass.** When A differs from a record that is not stale, the failure says
   which. Either the change meant to alter that vector's output, and its
   author regenerates. Or something outside the change reached the output:
   the clock, the process, the platform or the host. The
   `verification-gate` skill's ladder treats the second as a defect, and
   says so.
5. **What the record catches, and what it does not.** Against live A/B on
   one host, minutes apart, a record made by another process, earlier,
   possibly on another host:
   - **Catches the same:** the process id, the time zone, the locale, the
     environment and `GODEBUG` (`A` sets `=1`, B leaves it unset), and the
     clock at any resolution down to the gap between the record and the
     run.
   - **Catches more:**
     - a leak of the binary's own path, which live A and B share;
     - a clock leak at date resolution, which a two-second gap almost
       never crosses;
     - a difference between platforms. ADR-011 D15's cross-platform
       agreement (SC5) is checked on every change, on whichever host runs
       the gate.
   - **Catches later:** a leak that only a relative path shows ("makes a
     path absolute") was caught in A's relative mode. It is now caught by
     check 2 over B at regeneration, which every change that alters an
     output runs. A date-resolution clock leak is caught on the first day
     after its record was made, not on the day it merged. Item 4 makes that
     red a defect to find, not a record to refresh.
   - **Catches nothing that live A/B on one host caught for a host fact
     that is the same at both times** (host name, user, home, checkout).
     Neither method sees those by comparison. Check 2's scan does, and it
     runs on A in the pass and on B at regeneration.
   - **Asserts something new:** that absolute-path output equals
     relative-path output. Check 1 already implies it for any output that
     echoes a caller's path, since such an output carries the sentinel in
     the absolute mode.
6. **A case the record does not cover runs B live.** That is a case whose
   `requires` the regenerating host did not meet (ADR-011 D9 item 6: an
   unwritable directory under root, a file system that folds case). Its
   record lists it as unmet. On a host that meets the requirement, that
   case alone runs A and B live, with the two-second gap. A tripwire,
   `TestRecordsCurrent`, fails when:
   - a vector has no record;
   - a record is stale, or names a case or command the full plan does not;
   - a record holds `MTSSENTINEL`, or a tree path that is absolute or
     leaves its tree.
7. **Live A/B stays, and is proved, where a record cannot stand in:**
   - **The canaries** (ADR-009 D16 item 2). Each of `pid`, `tz` and
     `clock` must be caught by the record comparison and by live A/B. The
     others keep their checks.
   - **The runner's own proof** (ADR-011 D18): the A/B-divergence fault,
     and `TestBStartsTwoSecondsAfterA`.
   - **`-conformance.live`**, the published method. It is the default
     whenever the tool is not the reference build from source: another
     `-conformance.tool`, or the `exec` adapter. The release act runs the
     reference tool with it on each matrix target (ADR-011 D15 item 2), in
     both path modes and with the full opening fan-out. The method that
     `spec/conformance/README.md` publishes is unchanged.

   The runner keeps what live A/B needs at volume: the slot released
   during the wait, and the case goroutines started before `t.Parallel`
   (`runner_test.go:101-107`, `:637-645`, `:683-686`). It also keeps the
   darwin spawner, whose saving scales with the process count. These cost
   nothing per run.
8. **What the specification's tree gains.** Nothing normative. No `MTS-`
   statement changes, and no vector changes.
   `spec/conformance/README.md`'s "Running the suite" says what the
   reference runner does by default against the reference tool, and that
   `-conformance.live` runs the method above. That is prose describing this
   repository's runner, beside the paragraph on `TestHostIsolation` it
   already has.

### D3 — The opening fan-out runs through a representative of each refusal

**Each finding code a security `open` step expects is proved through every
package-opening command, once for each package form, by a representative
the runner chooses. Every other security `open` step runs `inspect`.**

1. **Representatives.** For each pair of a finding code that some security
   `open` step expects and a package form (packed or a source directory),
   the runner picks the first such step in path order. That step runs
   every command of `cli.OpeningCommands` (`internal/cli/decl.go:363-371`).
   Each command must give ADR-011 D16 item 3's result: `ok: false`, the same
   findings, the exit code they map to, and nothing written. Today that is
   48 codes, about 50 pairs, and about 700 processes, against 2,310.
2. **Why not one in-process proof that every command uses the shared
   opener**, as the audit leaned. There is no single opener:
   - `inspect` and the reading commands go through `session.open`
     (`internal/cli/inspect.go:231`);
   - render, pack, test, sample, validate and unpack call `loader.Open`
     (`render.go:328`, `pack.go:204`, `test.go:267`, `sample.go:375`,
     `validate.go:123`, `unpack.go:123`);
   - check goes through `author.Check` to `loader.Check`;
   - the directory form has its own paths (`loader.go:220`, `check.go:150`).

   ADR-011 D16 item 3's per-command facts are only observable through the
   binary: nothing written before the open (a `--out` created early), the
   exit mapping, and the bounds before verification.
3. **Why per code, not per family.** Commands differ in how they reach
   the opener, and a family shares one check function. Per family would
   be about fifteen representatives, and would most likely catch the same
   defects. Per code is what the audit leaned to. It costs about 600
   processes more than per family, and it keeps the claim "every refusal
   through every command" exact.
4. **The record holds the full plan** (D2 item 2). When a vector added
   earlier in path order takes over as representative, no record goes
   stale.
5. **`TestHostileVectorsOpenEverywhere`** fails when a pair of code and
   form among the security `open` steps has no representative run through
   every opening command. Its control (`tripwires_test.go:476`) stays the
   runtime proof. The full fan-out of every step remains `-conformance.live`'s
   (D2 item 7).

### D4 — The suite runs the release build; `-race` runs on the in-process tests and on the tool's one goroutine

1. **One build of the tool per test binary:** the release build
   (`hostisolation_test.go:170-197`), linked once and shared by the pass,
   the reach, `TestSkillHostIsolation` and `TestCalibrationLowered`. No
   suite process runs a race-built `metais`. The canaries, the `fsizestub`
   and the `procstub` are built as they are now.
2. **Why race-building the tool proves nothing per process.** The shipped
   tool starts one goroutine (`internal/cli/render_write.go:125`). It writes
   a container's zip into an `io.Pipe` that the commit reads: one producer,
   one consumer, in a fixed order. None of the module's dependencies starts
   one, and `os/exec` is outside the binary's closure (`layers.json:14-18`).
   Thousands of processes run a race detector over code with nothing to
   race. ADR-010 D14 item 8 already says `-race` chiefly guards the test
   harness.
3. **What still runs under `-race`:** every in-process test, as now. A
   focused test of `content()`'s goroutine runs its paths, the early close
   among them. It is shown red against a deliberate unsynchronised write
   before it merges.
4. **ADR-010 D2 item 3 and the code disagree.** It says that no goroutine
   is started on any path that renders, and `render_write.go:125` starts
   one. This ADR amends D2 item 3 to name that goroutine and its proof
   (ADR-010 A6). Removing the goroutine was the alternative; Ed chose the
   amendment on 2026-10-21 (the plan's question Q4).
5. **Calibration's lowered cases** (`calibrate_test.go:423`) run on the
   release build, in parallel. They judge findings at a limit, never a
   time, so neither the race build nor serial running buys anything.

### D5 — Limits are proved by growth, at small counts

**A test of a bounded resource proves that what it holds does not grow
along each axis. Writing a limit's full default count moves to calibration.**

1. **What the tests prove today, and what they miss.**
   `TestExistingWritesItsLimitUnderTheOpenFileLimit`
   (`internal/hostfs/existing_posix_test.go:28`),
   `TestShapesWriteTheirLimitUnderTheOpenFileLimit`
   (`shapes_posix_test.go:63`) and
   `TestUnpackWritesTheEntriesLimitUnderTheOpenFileLimit`
   (`internal/author/unpack_posix_test.go:89`) write the `outputs` default
   (100,000) or the `entries` default (10,000) under `RLIMIT_NOFILE` 128.
   `manyNames` (`shapes_posix_test.go:47-53`) writes 1,000 files per
   directory, so they write 100 or 10 directories. A handle leaked per
   directory adds 100 or fewer, and with the process's own dozen or so the
   test still passes. A per-depth leak is never reached, because no path
   is deeper than two (three in `author`).
2. **The property is ADR-005 A1 item 7's:** a fixed set of handles
   "however many files it plans; no handle is held per planned file". It
   is a statement about growth. R1-015 (`RULINGS-v1.md:114`) governs the
   product's limits ("never raised"). It says nothing about the size of a
   test, so a test that writes fewer entries lowers no limit. The ADR-010
   log W5b-2 read it otherwise, and this decision corrects that reading.
3. **The tests.** A test-only `openHandles()` counts `/proc/self/fd` on
   linux and `/dev/fd` on darwin, under the existing `linux || darwin` tag.
   For each shape (`StageNew`, `StageNew` with abort, `WriteInPlace`,
   `CreateNew`, and `Existing` with directories present and replaced), the
   test writes along each axis past the soft limit of 128:
   - 200 directories of one file each;
   - one directory of 200 files;
   - one chain at ADR-009's `path-depth` of 32, with a file at each level.

   It asserts that the count never exceeds its value after the first
   file, and that it returns to its starting value after `Commit` or
   `Abort`. That is about 700 files instead of 110,000 or more. A writer
   with pins on, made to hold a handle per entry, must make the count grow,
   and that is the control. `Unpack` is one call with no hook between
   entries, so it keeps the open-file limit as its detector, at the same
   three axes. W5b-2 and W5b-NB1 chose the full counts on purpose ("fewer
   entries proves the bound at less than the limit"). A growth test proves
   item 2's wording more strongly than one count under 128 does. What it
   loses, that the default count fits on this host, is item 4's.
4. **Full counts move to the calibration suite** (`calibrate_test.go`,
   `docs/dev/calibration.md`). Render into a new and into an existing
   directory at the `outputs` default, and unpack at the `entries`
   default, run through the release binary under an open-file stub of 128,
   the pattern of `guarantees/umask_unix_test.go:33`. The release act runs
   them on each target.

### D6 — The release builder is tested once per run of each kind

1. **`TestReleaseBuildsTwiceAndWritesDist`**
   (`internal/tools/release/release_test.go:174`) runs the whole release
   twice: 26 cold builds. The second run (`:240-251`) is the only check that
   archiving is deterministic, since one run compares only the two copies'
   binaries (`release.go:143-145`). The audit was right about the cost and
   wrong about the purpose. The test runs once. `archive()` is run twice on
   the same input and its bytes compared, in-process, with no build. The
   PLAN-2026-10-metais-v1 Wave 7 criterion, "matching `SHA256SUMS` on two
   runs" (`:1436`), is what the release act itself does. It is no longer
   repeated per change.
2. **`TestReleaseRefusesAnUnreleasedRevision`** (`:385`) clones and
   cold-builds in every case to reach a git check in `released()`
   (`release.go:228`). `released()` uses only `git` against the original
   repository, so it becomes a table test over `fakeRepo`, with no clone and
   no build. One end-to-end case (a revision with no tag) proves that
   `checkImplements` calls it. "Implements nothing" stays end to end, or
   its JSON reading is split out and tabled.
3. **Placement.** The package is selected on every change today, because
   `gate-select.sh` (`:127-134`) counts any test that runs the go command
   as blind to imports. Its tests read only its own sources, `buildline`,
   `tarw`, `testenv`, and one file of `internal/conformance`
   (`TestHarnessBuildsByTheReleaseLine`, `release_test.go:577`). That test
   moves to `internal/conformance`, beside the build it checks. Selecting
   the package only when those inputs change is the gate-selection fix's
   work. No "wave tier only" category exists (ADR-010 A5 items 2 and 3), so
   this ADR adds none.

### D7 — The guarantee detectors keep what conformance does not check

1. **Determinism** (ADR-010 D13 item 4, ADR-005 D19 item 2). Today the
   detector renders every vector three times in one process, at
   `GOMAXPROCS` 1 and `NumCPU`, and once in a fresh process
   (`internal/guarantees/determinism_test.go:174-188`).
   - With D2, the pass compares every vector's bytes from a fresh process
     in A with the record of another process in B. The comparison covers
     different environment, `GODEBUG`, path mode, day and possibly
     platform, and Go randomises map order per process. That is the
     repetition proof, for every vector.
   - What the detector alone adds:
     - the in-process `cli.Run` path;
     - `GOMAXPROCS` 1, against the one goroutine D4 item 2 shows cannot
       reorder bytes;
     - umask 077, in its fresh process.
   - So the in-process loop shrinks to the control's own probe: the
     20-member object rendered under `MapOrder`, at most 20 times, which
     must be caught, and the same probe clean, which must not. This keeps
     the detector honest without re-running the suite.
   - The fresh process under umask 077 stays, for the probe and for every
     vector whose expected trees mark a file executable. The execute bit is
     the only observed byte a umask can reach.
     `TestRenderUnderAClosedUmask` (umask 777) proves something else: that
     a render needs no permission from the umask.
   - `TestPackDeterminism` (`packdeterminism_test.go:149-166`) keeps
     `disturb`, changed file times and modes, which no vector varies. It
     drops the `GOMAXPROCS` loop.
2. **Confinement** (`confinement_test.go`). With check 4 over the whole
   sandbox (D2 item 1), the pass sees a write anywhere beneath the
   sandbox, the parent of the work directory included, for every vector
   through the binary. The detector keeps the in-process proof over the
   hostile corpus's traversal members, with `SkipConfine` as its control.
   Its every-render main loop goes.
3. **The gate-script tests** (`gateselect_test.go:114,146,157`) prove the
   scripts under `.claude/skills/verification-gate/`, not the tool.
   `TestGateSelectWorkingTree` is a strict subset of
   `TestGateSelectWorkingTreeHostGit`: same assertions, and
   `testenv.GitEnv` already discards the host's git configuration. It is
   deleted. The rest run when the scripts change, by the gate-selection
   fix (D6 item 3's rule).
4. **One build of `metais` per test binary**, behind a `sync.Once`, for
   the four tests that only run it (`determinism_test.go:162`,
   `packdeterminism_test.go:142`, `openfiles_test.go:53` and `:83`).
   `TestGoCommandTakesNoHostGoEnv` keeps its own, because the build is what
   it tests.
5. **`go list` once per target, and module files parsed once, per test
   binary.** Each target's listing is cached under its own key (a
   `sync.Once` per target, not one mutex held across the call), and
   `moduleFiles` (`golist_test.go:328`) is parsed once.
6. **`TestGoCommandKeepsHostCToolchain`** (`goenv_test.go:94`, 17.8 s
   alone) cold-builds `runtime/cgo` with a nonce, to prove
   `testenv.GoEnv` keeps the host's C toolchain. It is a test of
   `internal/testenv`, a shared seam whose change already runs the wave
   tier. It moves to that package. The audit did not list it.

### D8 — The command line's corpus walks become representatives and a static check

1. **`TestReadingRefusalsTriggered`** (`internal/cli/browse_test.go:75`,
   walk at `:104`) runs 482 packages through seven commands: 3,374 runs
   in-process. On a package that opens cleanly it asserts only the generic
   document checks, and an `internal.error` passes (`:274`, `cli.go:85`).
   It becomes:
   - one refused package per finding family, through the six reading
     commands and `inspect`: `package.container`, `package.name`,
     `package.layout`, `package.digest`, `package.compiled`,
     `package.version`, `manifest.*`, `schema.*`, template, and
     `limit.package` with a lowered limit;
   - two clean packages, one packed and one a source directory, through
     every reading command, asserting that no `internal.error` occurs. This
     is a check the walk lacked;
   - a static check: every code in `packageRefusals(cmd)` for a reading
     command appears in some `open` step's expected findings. This is the
     narrow form of `TestFindingCodesCovered`.

   All seven commands refuse through the one `session.open`
   (`inspect.go:231`), and their refusals are generated from the registry
   (`inspect.go:86-111`). So one package per family reaches every refusal
   path the walk reached. The binary's every-command reach over every
   package stays in the wave tier.
2. **`TestInspectRefusalsTriggered`** (`inspect_test.go:66`) walks the same
   482 packages through `inspect` alone, a subset of the walk above. Its
   walk is deleted. Its fixed line cases stay.
3. **`TestACREditionDecidesTables`** (`example_acr_test.go:28`) runs 40
   cases, five editions by eight subsets of three tables. The constraint
   is an `allOf` of independent `if`/`then`/`else` branches, one per
   optional table
   (`examples/accessibility-conformance-report/inputs/edition.schema.json`).
   So "no tables" and "all tables" for each edition put every cell in both
   states: 10 cases.
4. **`TestPresentationNeverReachesJSON`** (`output_test.go:115`) runs 65
   cases. `colour()` folds the twelve combinations to one bool
   (`output.go:368-375`), and only `fault.ColourJSON` consults it on the
   JSON path. Per command line: the base case, `always` on a terminal,
   and `auto` on a terminal without `NO_COLOR`. That is 15 cases and the
   control.
5. **`TestHostileMarkdown`** (`hostile_markdown_test.go:156`) runs cmark's
   25 pathological cases through outline and validate, 125 runs.
   `internal/markdown`'s `TestPathological` covers the outline more
   strongly. What the cli test adds is the closed-frontmatter shape, which
   reaches `yamlx.Parse`, and the line bounds of headings after
   frontmatter. It keeps those, and two outline cases as the end-to-end
   check, unless `internal/yamlx`'s hostile tests already cover the
   shape. The item checks which.
6. **`cmd/metais`**: its three builds hit the cache and the package costs
   4 s, so they stay. `TestSC3StatusWidget` reads the status-widget
   example's input and expected output from `examples/` instead of
   inlining a copy of them (`sc3_test.go:22-45`). That is one source of
   truth, not a saving.

### D9 — A process budget that cannot flake

1. **The pass plans before it runs.** For each step it lists the commands
   it will start, in each environment, and it counts every process it
   starts through `runProc` (`proc_test.go:119`) on a counter of its own,
   not the package-global `procStarted`.
2. **`TestConformance` fails when the count differs from the plan** by
   more than the fixed preamble: `version`, `help --json` and the
   vocabulary probe. It also fails when the plan exceeds two processes per
   step, plus 13 for each representative of D3 and one for each repeatable
   step. Today's ratio is 7.5 per step in `TestConformance` and 15 in
   `TestHostIsolation`, and a third environment, a second path mode or a
   wider fan-out would trip it. The budget scales with the suite, so adding
   a vector never touches it.
3. **No time is budgeted.** ADR-011 D11 item 9's 60 s is replaced by item 2,
   and by the per-package CPU targets that the plan measures at its wave
   verification. A count cannot flake. A time measured on a shared host
   can.
4. **Other packages are not given a count tripwire.** `internal/cli` starts
   no processes, and its in-process runs have no natural denominator, so
   any ceiling would be raised by every new test. Their targets are
   measured, as item 3 says.

### D10 — Where each kind of test runs

1. **Per change (the scoped tier):** everything above that a change can
   break: the pass, the representatives, the growth tests, the detectors'
   probes and controls, and the static checks.
2. **Per wave (the wave tier):** what it runs today, the every-command
   reach in groups among it.
3. **Per release (the release act, on each matrix target):**
   - `-conformance.live` with the full fan-out in both path modes;
   - the driver protocol over the whole suite (D11);
   - the calibration suite with D5 item 4's full counts;
   - the release builder's two runs.
4. **Which packages a change reaches** is the gate-selection fix's to
   decide. This ADR relies on it for D6 item 3 and D7 item 3, and adds no
   tier.

### D11 — The driver protocol is proved by its round trip and its fixtures

**ADR-011 D11 item 7's whole-suite run through the reference driver
(`driver_test.go:150-157`, about 3,200 processes) is replaced by:**

1. **A static round trip.** For every step of the suite, the request the
   `exec` adapter would send is built, and it must validate against
   `driver-request.schema.json`. For every step's expected findings, a
   response carrying them must validate against
   `driver-response.schema.json` and parse back through the adapter
   unchanged.
2. **`/fixtures`** (`driver_test.go:171-196`): the self-proof's fixture
   vectors through the reference driver over the clean fake, which
   proves that every operation they use crosses the protocol. It gains a
   stream-rendering fixture if it has none.
3. **The release act** runs the whole suite through the reference driver,
   live, as an external implementation would be run.

The reference driver reuses `cliAdapter.commands` (`driver_test.go:58`),
and passes a vector with no tags, so a hostile `open` becomes `inspect`
alone (`:62`). The suite run therefore proved little beyond what item 1
proves, and the tool's own behaviour, which the pass proves.

### D12 — How a test's cost is measured

1. **Alone.** One package at a time, `-p 1`, at the gate's `-parallel`,
   with `-race -count=1` and a warm build cache. CPU is user plus system
   time of the whole `go test`, children included, from `/usr/bin/time`
   or `getrusage(RUSAGE_CHILDREN)`.
2. **A test's own figure** comes from a run of that test alone
   (`-run '^Name$'`), less the package's `TestMain` run alone
   (`-run '^$'`). In `go test -json`, a parallel test's time is counted
   from its first `cont`, never from its `run`.
3. **The host is named** with each figure: its OS, architecture, cores
   and `go version`. Figures from different hosts are compared only as
   ratios.
4. **`internal/conformance`'s process count** is the pass's own log line
   (D9 item 1). The package's total of processes and builds is logged by
   its `TestMain`.

### D13 — The conventions AGENTS.md gains

AGENTS.md's Development section gains these rules, beside "Pin the property
false first". They land with this ADR, because they govern every test
written from then on, including the plan's own:

- **Prove a property along each of its axes, at the smallest size that can
  fail it.** Full-scale counts of a limit live in the calibration suite,
  which runs at release.
- **Before adding a walk over the whole corpus, name what one
  representative per code path would miss**, in the test's comment. If
  nothing, run the representatives.
- **When a test is slow, ask what it proves before making it parallel.**
  Parallelism moves cost off the clock, not off the shared host.
- **Answer an incident in the test infrastructure with a unit test of the
  fix**, not with a run of the suite that would have shown it.
- **A shrunk, merged, moved or deleted test names what still proves its
  property**, in its pull request.

The `verification-gate` skill gains D2 item 4: a record that differs on a
change that did not mean to change that output is a red to root-cause, not
a record to regenerate. It also gains D12's method for publishing a cost.

## The audit's findings, and what this ADR does with each

| Finding | Verdict | Where |
|---|---|---|
| 1. The suite runs three times or more | Confirmed: 5,881, 11,766 and about 3,200 processes | D2, D3, D11 |
| 1a. One pass and a committed B record | **Adopted, adapted.** Exit status is implied, not recorded. The record holds the full plan. A case the record lacks runs live. A differing record is root-caused. | D2 |
| 1b. The security `open` fan-out | **Adapted.** A representative per code and form runs every command. The in-process opener proof is rejected: there is no single opener. | D3 |
| 1c. `TestDriverProtocol/suite` | **Adopted.** Static round trip plus `/fixtures`, with the whole suite at release | D11 |
| 1d. Builds | **Adopted.** The release build only, shared | D4 |
| 1e. The serial chain | **Adopted** for the pass and for calibration's lowered cases. `TestHostIsolation` no longer runs the suite. The canaries were already parallel inside. | D2 item 1, D4 item 5 |
| 1f. `TestSelfProofAccruingRowsIgnoreWall` | **Adapted.** A unit test of `rowWall` plus one end-to-end row (cpu, `cli`), which keeps the proof that a starved process is reported as CPU, not as wall time | plan |
| 1f. `TestUnmetRequirementIsSkippedAndReported` | **Rejected.** Its `runSuite` leg starts no tool process, since the probe fails, and it is the only proof that a record lists a skip. Its 64 s was contention. | — |
| 1f. `TestHostileVectorsOpenEverywhere` and Control | **Adapted.** The test is redefined by D3 item 5, and the Control stays as the runtime proof. Both are cheap. | D3 |
| 1f. `TestRecordAndCompare` | **Adopted.** One record run, compared with itself and with an edited copy | plan |
| 1f. `TestUnwritablePreState` | **Adopted.** `cli` only, its control in A only | plan |
| 1f. `TestBStartsTwoSecondsAfterA` | **Rejected.** Its time is a sleep, not CPU, and its `exec` leg guards an adapter that returns before its process ends | — |
| 1g. Machinery that absorbs volume | **Rejected for now.** `-conformance.live` still runs the whole suite at release, and the pieces cost nothing per run. | D2 item 7 |
| 2. Limit tests that brute-force the limit | **Adopted, adapted.** Growth at small counts with a handle counter; `Unpack` by the open-file limit at small counts; full counts to calibration. R1-015 governs limits, not tests. | D5 |
| 3. `internal/tools/release` | **Adapted.** One run plus an in-process archive-determinism test; `released()` as a table. "Move to the wave tier" is rejected as stated (no such tier), and placement is left to the selector. | D6 |
| 4. Determinism and pack-determinism loops | **Adopted.** Needs ADR-005 D19 item 2 and ADR-010 D13 item 4 amended | D7 item 1 |
| 4. Confinement duplicates conformance | **Adopted**, once check 4 covers the whole sandbox, which it does not today | D7 item 2 |
| 4. Gate-script tests | **Adopted.** `WorkingTree` deleted; the rest selected by the scripts | D7 item 3 |
| 4. Five builds | **Adopted** for four. One is the thing tested. | D7 item 4 |
| 4. `go list` behind a mutex | **Adapted.** The callers are serial, not the mutex. A per-target once, and module files parsed once. | D7 item 5 |
| 4. Liquid's import tests duplicate `layers.json` | **Rejected.** They cost milliseconds, and they read files under build tags outside the six targets, which `layers.json` does not | — |
| 5. The cli corpus walks | **Adopted, adapted.** One package per family, a static check, and a clean-package `internal.error` check the walk lacked | D8 |
| 5. ACR editions, presentation, hostile Markdown | **Adopted.** Hostile Markdown keeps the frontmatter shape | D8 |
| 5. `cmd/metais` builds, `TestSC3` | **Rejected** for builds (4 s, cached). **Adopted** for SC3's data. | D8 item 6 |
| 6. Race | **Adopted.** Plus a red-first focused test, and the ADR-010 D2 item 3 contradiction surfaced | D4 |
| 7. Budget by process count | **Adopted, adapted.** Equality with a plan, and a per-step ceiling | D9 |
| 8. AGENTS.md rules | **Adopted.** "An incident is answered with a unit test" sharpens the root-cause skill's "keep a test that fails without the fix" | D1, D13 |

## Alternatives rejected

### Keep live A/B, and only merge the three runs into one

One pass, with A and B live, against the release build, with the
representatives of D3. That makes about 2,900 processes and a two-second
wait per case, with no committed file. It is the smaller change, and it
catches nothing the record does not. It costs twice the processes on every
change, keeps the wait on every case, which Ed accepts only "a handful of
times", and gives up D2 item 5's every-change check of platform agreement
and of the binary's own path. The record's cost is a regeneration step
that every output change runs anyway, as vectorgen's numbers already do
(ADR-011 A5).

### Run the expensive tests in the wave tier only

A property that a change can break must be proved on that change. In the
wave tier, a change that breaks confinement merges, and its siblings
inherit the red (ADR-010 A5 item 8). This ADR makes the proofs cheap
instead, and moves to the release act only what no change can break: the
full counts of a limit on a host, and the published method's own volume.

### Raise the timeout, or the parallelism

The cost is on the host whatever the clock says. The shared-host cap of
ADR-010 A5 item 7 exists because the host is shared, and a higher cap
moves the timeout from this gate to a neighbour's.

### Budget by time

A CPU time measured on a shared host drifts with the host's scheduling. W5b-2 measured
the same subtest at 60 s to 131 s of CPU time across six runs. A budget
that fails on the host's load is a flake by construction. Counts do not
drift.

### Amendments only, with no new ADR

The decisions touch four ADRs and AGENTS.md. Spread over four amendments,
the principle would have no single statement, and the audit's findings no
single ledger, for the next wave to cite. Each amended ADR still gets an
amendment, so that its own text stays true to a reader who reads only it.

## Consequences

### Positive

- The gate's dominant package runs about one process per step instead of
  15 per step. The plan's targets give the measured figures.
- SC5's cross-platform agreement and the binary-path leak are checked on
  every change rather than at release.
- Three properties are proved more strongly than before:
  - bounded handles, along every axis rather than at one count;
  - confinement, over the whole sandbox;
  - clean reading commands, which may no longer pass on an
    `internal.error`.

### Negative

- A committed record per vector: about 480 generated files, a
  regeneration step, and a merge driver line.
- A date-resolution clock leak turns red a day late, possibly in someone
  else's change. D2 item 4 makes it legible.
- A platform difference that the release matrix would have found turns
  red on the other platform on every change until it is fixed, rather
  than once at release.
- The published method's full volume runs only at release. A defect in
  `-conformance.live` that no fixture or canary exercises waits for it.

### Neutral

- No `MTS-` statement, vector or published schema changes.

## What this does not decide

- **Gate selection**: which packages a change reaches, and whether a
  package that runs the go command is blind to its imports. That is the
  sibling Work Order's. D6 item 3 and D7 items 3 and 6 rely on it.
- **Whether the render keeps its one goroutine** (D4 item 4).
- **Per-platform records.** There is one record per vector. If a real
  platform difference is ever ruled acceptable, which SC5 forbids today,
  records per platform are a later amendment.

## Implementation log

This log is the entry files in `docs/adr/logs/ADR-012/`, one per work
item, named as AGENTS.md's "Writing documentation" says.

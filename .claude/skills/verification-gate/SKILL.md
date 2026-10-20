---
name: verification-gate
description: The pre-merge gate for metais, in two tiers (ADR-010 Amendment A5) — the scoped tier every change passes, and the wave tier, AGENTS.md's full list, once per wave and for a shared seam. Read this BEFORE running the gate, BEFORE publishing a test run as evidence in a pull request, and ALWAYS when a step goes red. Covers when to run it (once, last, after directed tests, self-review and rebase), how to run it (gate.sh in the background, watched with gate-wait.sh, resumed with --resume), what each tier runs and what a change reaches (gate-select.sh), why it caps `go test -p` on a shared host, what to re-run after a fix and after a rebase onto a sibling, the generated files' merge drivers (merge-setup.sh before the first rebase in a fresh clone) and regenerate.sh after every rebase, how to tell whether a red is yours by reading its mechanism (gate-why.sh, then root-cause), and what a green gate cannot settle.
---

# The verification gate

The gate has two tiers (ADR-010 Amendment A5). `gate.sh` beside this file
runs either one. Every step runs even when an earlier one goes red, so one
run reports everything that is wrong, and the run ends with a summary of
every step's verdict.

| Tier | Command | When |
|---|---|---|
| **Scoped** | `gate.sh` | Every change, before it merges |
| **Wave** | `gate.sh wave` | Once per wave, at the wave's verification, on a fresh clone of the merged tree; and before merging a change to a shared seam |

- **The scoped tier** runs build, vet and gofmt over the whole module,
  `go mod tidy -diff`, the static build, and the six-target cross vet and
  build. It runs `go test -race` over every package your change reaches,
  and the spec and generated-data tripwires on every run:
  - `internal/finding` and `internal/specdata`, whole;
  - `internal/conformance`'s well-formedness tests;
  - the help goldens;
  - `internal/guarantees`' tests of the specification's text;
  - the specdata drift.
- **The wave tier** is AGENTS.md's full list: every package under
  `go test -race`, and every other step. Its last steps, `reach <area>
  <first slug>`, run the SC7 harness's every-command reach over every
  vector (R-W6-3). Each is `TestHostIsolation` with `-conformance.reach`
  over at most 25 vectors of one area (`METAIS_GATE_REACH_CHUNK` sets the
  size), in its own `go test` invocation, so each stays within the
  ten-minute timeout. A group that runs fewer vectors than it selects
  fails. Every `go test` runs the reach over the canaries' probes and in
  `TestHarnessReach` alone, so a harness that stops catching a leak is
  red in either tier.
- **The shared seams** are `go.mod`, `go.sum`, `internal/fault`,
  `internal/testenv` and this skill's scripts. A change to one needs both
  tiers. The scoped run says so in its log, with a `NOTE`.

Run the gate through `gate.sh`, not a hand-written copy of either list. A
hand-written copy drifts from the amendment, polls with blind `sleep`s, and
reports a different shape in every pull request.

## What your change reaches

`gate-select.sh` prints what the scoped tier will test, and why. Run it
before the gate to see the cost coming:

```bash
.claude/skills/verification-gate/gate-select.sh          # since the merge base with origin/main
.claude/skills/verification-gate/gate-select.sh --base <ref>
```

The change is everything that differs from the merge base with
`origin/main`: commits, staged and unstaged edits, and untracked files.
Fetch `origin` first, or the base is stale. A path reaches packages by
these rules:

- **A path in a package's directory** (Go files, `testdata/`, embedded
  files) touches that package. Every package that imports it, tests
  included, is tested.
- **A path outside every package** (`spec/`, `docs/`, a root file) is data.
  It touches the packages whose test files name it as a path literal
  (`"spec/…"`, `"../../spec"`). A change under `spec/` reaches the spec
  readers: `cli`, `conformance`, `finding`, `guarantees`, `liquidspec`,
  `loader`, `specdata`, `template` and `vectorgen`.
- **A Go file outside every package** (build-ignored, or under a directory
  `go list ./...` does not see) also touches every package whose test files
  name the module root itself (`"../.."` from `internal/x`). Those tests
  walk the module's tree: `TestLicenceHeaders` and `TestNoHostFacts` read
  every Go file, whatever `go list` sees.
- **A generated file is a path like any other.** The tripwires and the
  drift step check every generated file on every run anyway.
- **The blind packages** (`blind` in the output) are the ones whose tests
  build the binary or list the module with the `go` command: `conformance`,
  `guarantees`, `hostfs`, `version`, `pkgtree` and `cmd/metais`. Any
  change to the module's code has them tested, because no import edge
  shows what they read.

`internal/guarantees`' `TestGateSelectReach` holds these rules to a probe
list, `testdata/gateselect/reach.txt`: each probe is a changed path and the
packages it must reach. A change to the rules adds its probe there, not in
a test file, whose path literals the data rule would read as what that
package reads.

## Generated files: the merge drivers, and regenerate.sh

A committed generated file is never merged by hand (ADR-011 A5).
`.gitattributes` names a merge driver for each one, and
`merge-generated.sh` beside this file is the driver:

- **`metais-generated`, for a file generated whole**, keeps one side's
  copy and reports a clean merge. These are `spec/statements.json`,
  `spec/findings/registry.json`, the specdata copies, the help, scaffold,
  tar, YAML-writer and vector-package goldens, and
  `THIRD-PARTY-NOTICES.txt`.
- **`metais-generated-region`, for `spec/12-findings.md`**, gives the
  chapter's prose git's own three-way merge, conflicts included, and keeps
  one side only between the code catalog's markers. When the markers are
  not one well-formed pair, it merges the whole chapter as text. It never
  takes one side of hand-written text.

Every merge a driver resolves prints a `merge-generated:` line naming the
file. The side it kept is stale until you regenerate:

```bash
.claude/skills/verification-gate/regenerate.sh
```

It rewrites every committed generated file from the merged sources, in the
finding chain's order: `go generate ./internal/finding ./internal/specdata
./internal/guarantees`, then each golden test's `-update`, then the
notices. It ends by listing the generated files that changed. Commit them.
On an unchanged tree it changes nothing. Each file's drift test fails until
it has run, so a forgotten regeneration is a red, not a silent stale file.

**Git reads drivers only from the clone's configuration, never from the
repository.** A clone without them falls back to an ordinary text merge
of generated files, and says nothing. So:

- **In a fresh clone, run `merge-setup.sh` before your first rebase or
  merge.** It installs the drivers in `.git/config`. `gate.sh`,
  `gate-select.sh`, `gate-why.sh` and `regenerate.sh` all run it first.
  If the drivers were missing, it prints
  `merge-setup: NOTE: installed the merge driver(s) …`, and `gate.sh` puts
  that line in its log. If you see it after a rebase, that rebase merged
  generated files as text. `regenerate.sh` makes them right either way.
- **`merge-setup.sh --check`** installs nothing, and exits 1 when a driver
  is missing.
- **A conflict that remains is in hand-written text**, the prose of
  `spec/12-findings.md` included. Resolve it, then regenerate.
- **A file one side deleted** is a modify/delete conflict, which git never
  hands to a driver. Take either side, then regenerate. The help, scaffold
  and vector-package tests fail on a golden file nothing writes.

**`merge-check.sh` proves the drivers** on a fresh clone of HEAD, so commit
first. It runs a scripted conflicting rebase of each kind, then
regeneration, then the drift tests. Run it after changing `.gitattributes`,
a generator or a driver. Its planted statements take the two numbers after
the highest `MTS-AUT` number in HEAD's tree, so no statement the
specification assigns can take them. A new generated file needs a line in
`.gitattributes` and a step in `regenerate.sh`, in the change that adds it.

**The scripts' git takes nothing from the host's configuration.** Every
script that runs git sources `git-env.sh` first. It drops the caller's
`GIT_*` variables, reads no system or global configuration, attributes or
excludes file, and turns `core.precomposeunicode` off, which git on darwin
writes into every repository it creates. The go command's git inherits it.
A test that runs git gives it the same environment through
`internal/testenv`'s `GitEnv`. A test that runs the go command gives it
`testenv.GoEnv`'s, which takes the host's module cache, read only, and no
proxy; the scripts' own go commands are the developer's build, and run in
the developer's environment.

## When to run it: once, last, right before merge

The scoped tier takes minutes, and most of them go to the packages your
change reaches. So it runs once, on the finished tree:

1. **Make the change, and prove it with directed tests.** These are your
   new tests and the packages you edited. A red here costs seconds; in the
   gate it costs a run.
2. **Review your own change and fix what the review finds.**
3. **Rebase onto `origin/main`**, run `regenerate.sh`, commit what it
   changed, and push. In a fresh clone, run `merge-setup.sh` before the
   rebase (below). Rebasing before the gate means the gate certifies the
   head you merge.
4. **Run the scoped tier** (below). If the change touches a shared seam,
   run the wave tier too.
5. **On a red, read it before re-running anything** (the ladder below). Fix
   what is yours, then `gate.sh --resume`.
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
   .claude/skills/verification-gate/gate.sh            # scoped tier
   .claude/skills/verification-gate/gate.sh wave       # wave tier
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
| `DIED` | 4 | The run is gone and never finished. Its log's tail follows. | Find out why from the tail, then `gate.sh --resume`. |

These rules make the background run work:

- **Do not end your turn while a run is live.** A dispatched session is one
  turn: ending it kills the session, and the background gate with it.
- **Do not edit the tree while a run is live.** `go test` builds packages as
  it reaches them, so an edit mid-run gives later packages a tree you never
  meant to test. A run whose tree moved says so at its end, and
  `--resume` will not reuse its verdicts.
- **One run per checkout.** `gate.sh` refuses to start beside a live run.
- **Do not wrap it in a shell `timeout`.** The hang bound is `go test`'s own
  `-timeout` (ten minutes per package binary), which panics with a goroutine
  dump that names the stuck test. A shell kill throws that dump away.
- **The files are under `.local/gate/`** (git-ignored). `latest.log` is the
  newest run's log. `<run>.record` holds one verdict per step and per
  package, each with the tree it was reached on.

## Resuming: re-run only what is still unverified

`gate.sh --resume` (or `gate.sh wave --resume`) starts a run that reuses a
PASS from the tier's last run when nothing since that verdict reaches it.
The paths that differ between the verdict's tree and yours are judged by
`gate-select.sh`'s rules. These all run again:

- a failure;
- a step or package the last run never reached, because it died, was
  killed, or was cut off with your session;
- a PASS your edit since then reaches;
- the cheap steps (gofmt, tidiness, the specdata drift), whenever the
  tree changed at all.

It is the run to start after you fix a red, and the first run to start
when a resumed session finds a gate that died with its predecessor. A
resume of a resume carries every verdict forward with the tree it was
reached on, so a chain of resumes never carries a stale PASS. Reused
verdicts show as `=== REUSED` in the log.

## After a rebase onto a sibling

A sibling merged while you worked, and you rebased onto it. Re-run what the
rebase changed, not the whole gate:

1. **Run `regenerate.sh`**, and commit what changed.
2. **`gate.sh --resume`.** The rebase changed paths from the sibling's
   change and from the regeneration. Resume re-runs the packages those
   paths reach, every tripwire, the drift step, and the build steps when
   code changed. It reuses the rest of your last run.

If your last run is gone, a plain `gate.sh` does the same work from
scratch. Run the wave tier after a rebase only when the sibling's change
or yours touches a shared seam.

Rebase reds are deterministic, and they are yours. A statement number a
sibling took first, a regenerated golden, or conflict markers in a chapter
all reproduce every time. Fix them as the brief says: renumber your unmerged
statements (a duplicate is `TestStatementsWellFormed`'s `is already used`),
and run `regenerate.sh` rather than merging a generated file by hand.

## The shared-host cap, and the test cache

The hosts that run this gate run several sessions at once. `go test` starts
as many package binaries at once as the host has cores, and three gates on
one host then ask for many times the cores it has. A time-bound test
stretches past its bound under that load: a wrong verdict caused by the
gate, not by the code.

- **The gate's share is a third of the host's cores** (at least 2), the
  share AGENTS.md gives a fuzzer. `gate.sh` passes it as `-p`, the package
  binaries at once, and as `-parallel`, the parallel tests at once in one
  binary. `-parallel` also bounds the tool processes `internal/conformance`
  runs. `METAIS_GATE_P` sets the share (Amendment A5 item 7).
- **A test must not hold a parallel slot while it waits.** One that does
  turns a lower `-parallel` into a timeout the gate caused. Fix the test,
  not the cap.
- **Never raise `-timeout` to get past a timeout.** A test that times out
  has a mechanism to find (`root-cause`).
- **Both tiers pass `-count=1`.** Go's test cache keys on the test binary,
  and the blind packages' binaries do not link the tool they build. So an
  edit to the tool left `internal/conformance` reporting `(cached)` over a
  binary it never ran. `--resume` is the gate's own reuse, and it can see
  that reach.

## Is this red mine? The ladder

Every red is a defect with a mechanism. A red is answered by reading that
mechanism, never by matching it against a list of known failures. There is
no such list, and none is kept. "It passed alone" closes nothing, and
neither does a green re-run. A red that does not reproduce on its own tells
you the failure depends on timing, or on what ran beside it: a lead, not an
acquittal.

1. **Read the failure, and the lines around it.** A `-timeout` panic names
   the test that was running and dumps every goroutine. A spec-tripwire
   failure names the file, the line and the statement number.
2. **Does your change reach it?** `gate-select.sh` says which packages your
   change reaches, and through which path. A spec file, vector or generated
   file you changed reaches every test that reads it.
3. **Re-run it alone, and read its times:**

   ```bash
   .claude/skills/verification-gate/gate-why.sh TestName ./internal/<pkg>
   .claude/skills/verification-gate/gate-why.sh 'TestName/subtest' ./internal/<pkg>
   ```

   It runs the one test with `-race -count=1 -v`, every level of the name
   anchored. It says whether your change reaches the package, and prints the
   run's wall time beside its CPU time. CPU far below wall means the test
   waits (a sleep, a lock, a child process). CPU near wall times the cores
   means it is bound by work.
4. **Find the mechanism** with the `root-cause` skill. It applies whether or
   not the red is yours. These are the usual mechanisms of a red that
   appears only under load or only beside other tests:
   - a timing assumption;
   - an unbounded wait;
   - a test that charges wall time where it means CPU time;
   - a gate that oversubscribes the host.

   Each one is a defect, and you fix the mechanism. Never widen an
   assertion, raise a timeout or add a skip to make the red go away.
5. **If it is yours, fix it**, then `gate.sh --resume`. If it is not, do not
   merge over it. Report it to whoever owns your work, with the failure
   output and the mechanism you found: in a Work Order, the orchestrator
   that filed it. They decide whether it blocks you.

## What a green gate cannot settle

- **What only the wave tier sees.** The scoped tier misses three kinds of
  failure:
  - a test that reads module data by a path the selector cannot see;
  - interference between packages that run together;
  - two siblings that each pass alone and fail together.

  The wave tier, on the merged tree, is what catches them (Amendment A5
  item 8).
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

In the pull request, record:

- `go version` and GOOS/GOARCH;
- the tier you ran, and its `=== SUMMARY` lines, the `REUSED` ones
  included;
- anything your brief asks for.

Quote the commands you ran, as you ran them. Claim nothing you did not run.

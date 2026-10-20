# AGENTS.md

Instructions for any coding agent working in this repository, and the
conventions every contributor follows. `README.md` says what the project is.
This file carries **conventions**; it is not a status log.

## Orientation

- **METAIS is two deliverables:** the specification of the package format,
  and `metais`, its reference tool in Go. The specification is normative; the
  tool implements it.
- **Requirements** live in `docs/prd/`, **decisions** in `docs/adr/`, **plans**
  (the trains that build them) in `docs/plans/`, and per-wave briefs in
  `docs/briefs/`. This repository's PRD and ADR numbers are its own and never
  share a number space with another repository's.
- `docs/spec-outline.md` is design input. Once the PRD and ADRs cover a topic,
  they govern it, not the outline.

## The rules every line obeys

These are the project's design stance. The ADRs refine each one; none of them
is relaxed by a work item.

- **The specification comes first.** A change to what a package means or to
  what the tool does changes the specification, and the conformance vectors
  that prove it, in the same change. Where the tool and the specification
  disagree, the tool is wrong.
- **Strict on the data, loose on where it comes from.** Every input is
  validated against its schema, and every finding is reported at once. The
  package never prescribes how its data is gathered; it explains what the
  data means.
- **Rendering is pure.** No hook, script or plugin from a package ever runs.
  No command touches the network. The same package and the same inputs give
  byte-identical outputs on every platform.
- **A package is read in place.** Listing, reading and searching a package
  never unpacks it to disk.
- **A package is untrusted input.** Every entry path and every output path is
  confined, and every size, count and ratio is bounded, before anything is
  read or written.
- **Liquid is the only template language.** The specification pins the
  dialect; the tool does not extend it with anything a package can't rely on
  in another conforming implementation.
- **The golden path is designed, not left over.** Every user-facing task —
  writing a package, validating data, rendering — has one documented,
  opinionated route a newcomer follows without having to choose. Alternatives
  are escape hatches.
- **METAIS knows nothing about tera.** Nothing here imports, names or
  requires Kottera's runtime. tera may use METAIS; METAIS never depends on
  tera.

## This is a product, not a toy

Every surface works end to end. When a feature needs something an underlying
part does not provide, fix that part rather than layering a workaround over
it. Do not return a plausible empty answer where the real one is missing, do
not synthesize data the source does not hold, and do not hide half-finished
work behind a flag. If you catch yourself writing "for now, we'll…", stop,
say so, and split the prerequisite into its own work item.

## Development

The module is `github.com/kottera/metais`. Its `go.mod` `toolchain` line
names the exact toolchain every build and test uses, fetched under the
default `GOTOOLCHAIN=auto` (ADR-010 D1). There is no CI. The gate is two
tiers (ADR-010 D15 item 3, as Amendment A5 divides it), and
`.claude/skills/verification-gate/gate.sh` runs either one:

- **The scoped tier, before every change merges:** `gate.sh`. It runs
  build, vet and gofmt over the whole module, `go mod tidy -diff`, the
  static build and the six-target cross vet and build. It runs
  `go test -race -count=1` over the packages the change touches and every
  package that imports them. On every run it also runs the spec and
  generated-data tripwires:
  - `internal/finding` and `internal/specdata`;
  - `internal/conformance`'s well-formedness tests;
  - the help goldens;
  - the specification's own text tests;
  - the specdata drift.

  `gate-select.sh` prints what a change reaches.
- **The wave tier:** `gate.sh wave`, the full list below. It runs once per
  wave, at the wave's verification, on a fresh clone of the merged tree.
  It also runs before merging any change to a shared seam: `go.mod`,
  `go.sum`, `internal/fault`, `internal/testenv`, or the gate's own scripts.

```bash
go build ./...
go vet ./...
go test -race -count=1 ./...
go mod tidy -diff    # go.mod and go.sum stay tidy
gofmt -l .           # prints nothing
CGO_ENABLED=0 go build -trimpath ./cmd/metais
for t in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do
  GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go vet ./... && \
  GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go build ./...
done
go generate ./internal/specdata && git diff --exit-code internal/specdata
# the SC7 reach: one invocation per group of at most 25 vectors of an area
go test -race -count=1 -run '^TestHostIsolation$/^<area>$/^(<slug>|…)$' \
  ./internal/conformance -args -conformance.reach
```

The last step is the SC7 harness's every-command reach (ADR-009 D16 item
1) over every vector. Every `go test` runs that reach only over the
canaries' probes and in `TestHarnessReach`. Over the whole suite it is
more than one `go test` binary can run within its ten-minute timeout at the
gate's share of the host. So the wave tier runs it in groups of one area's
vectors, each group in its own invocation under the same timeout and share
(R-W6-3, plan orchestrator, 2026-10-20). It runs at the wave's verification
and again before a release.

Run either tier in the background, and watch it with `gate-wait.sh` until
it says `DONE`. After a fix, or after a rebase onto a sibling,
`gate.sh --resume` re-runs only what the change since the last run reaches.
`go test -p` and `-parallel` are both capped at a third of the host's cores,
the share a fuzzer gets, because the host is shared. `METAIS_GATE_P` sets
that share. The
conformance vectors run as part of `go test`. The race detector needs cgo,
so `go test -race` runs with it enabled; the static build is the separate
check of a cgo-free binary.

**Generated files are regenerated, never merged by hand** (ADR-011 A5).
`.gitattributes` gives each committed generated file a merge driver that
keeps one side's copy. For the code catalog in `spec/12-findings.md`, only
the generated region keeps one side, and the chapter's prose gets an
ordinary three-way merge. Git reads drivers only from a clone's own
configuration, so:

- **In a fresh clone, before the first rebase or merge,** run
  `.claude/skills/verification-gate/merge-setup.sh`. Every gate script also
  runs it, and says `NOTE` when the drivers were missing. A clone without
  them merges generated files as text.
- **After every rebase or merge,** run
  `.claude/skills/verification-gate/regenerate.sh` and commit what it
  changes. It rewrites every committed generated file from its sources:
  - `spec/findings/registry.json`, the specdata copies,
    `spec/statements.json` and the code catalog, by `go generate`;
  - the help, scaffold, tar, YAML-writer and vector-package goldens, by
    their tests' `-update`;
  - `THIRD-PARTY-NOTICES.txt`.

  Each file's drift test fails until it has run.

A new generated file gets a line in `.gitattributes` and a step in
`regenerate.sh` in the change that adds it. `merge-check.sh` proves the
drivers on a fresh clone, so run it after changing either file.

Two skills under `.claude/skills/` carry the procedure behind the tiers.
Agents that do not load skills read them as plain files:

- **`verification-gate`**: read it before you run either tier, before you
  publish a test run as evidence, and whenever a step goes red. It covers
  `gate.sh`, `gate-wait.sh`, `gate-select.sh` and `gate-why.sh`, which
  re-runs one test alone. It also covers the generated files' merge
  drivers and `regenerate.sh`, resuming, what to re-run after a rebase, and
  how to tell from a red's mechanism whether it is yours.
- **`root-cause`**: read it the moment something fails and you do not yet
  know why, before re-running anything.

- **Test-driven.** Every behaviour has a test that fails without it, written
  against the specification's wording. Every refusal has a negative test.
- **Pin the property false first.** A headline guarantee — no write outside
  the output directory, no network, byte-identical outputs — gets a test
  that is red before the code that makes it true.
- **Debug the mechanism, not the blame.** A failure asks *why*. Make it
  deterministic or make one occurrence sufficient. The aim is no
  intermittent failures at all: a red that passes on a re-run still has a
  mechanism to find, and no test is closed by widening its assertion or
  raising its timeout.
- **Bounded work is measured.** A test that holds work to a time measures
  CPU time, not wall time alone.
- **A benchmarked package commits its baseline** in
  `testdata/bench/baseline.txt` beside its benchmarks, labelled with its
  host, CPU and `go version` as a sandbox figure, not a budget.
  `docs/dev/benchmarks.md` is the one command that runs every benchmark,
  and how a run is compared with the baselines.
- **A fuzz target is proved by its seed corpus.** The corpus under
  `testdata/fuzz/` runs in every `go test`, and that is the proof a change
  gives. A Work Order never runs `go test -fuzz`: Go starts a fuzz worker on
  every core, and the runner host is shared with other Work Orders, so two
  minutes of it stalls everything else on the machine. Fuzzing by hand runs
  one target for a minute at most, on a third of the cores:
  `go test -run '^$' -fuzz '^FuzzX$' -fuzztime 60s -parallel $(( $(getconf _NPROCESSORS_ONLN) / 3 )) ./internal/x`.
- **Isolate tests with `HOME` and temporary directories**, through
  `internal/testenv`, never with the developer's real configuration.
- **Dependencies are few and justified.** A new module dependency is named
  in the ADR that needs it.

## Writing documentation

- **Decisions go in ADRs, requirements in the PRD.** Language, layout and
  tooling choices are made by ADR; until one exists, ask.
- **Edit this file only when a convention changes.** Status belongs in the
  plans and the ADRs' implementation logs.
- **An implementation log entry is a file of its own.** A work item never
  appends to an ADR's "Implementation log" section: it writes its entry to
  `docs/adr/logs/ADR-<NNN>/<address>.md`, where `<address>` is the bracketed
  address of its title (`docs/adr/logs/ADR-010/W5b-1.md`). The file begins
  with a heading naming the ADR, the address and the change, and its first
  paragraph opens with the date and the plan, as a log entry always has. An
  item that logs against several ADRs writes one file in each ADR's
  directory. Work outside a train names its file by date and a short slug
  (`2026-11-02-zip64-reader.md`). A file is never overwritten: a later train
  whose address an earlier train's file already holds prefixes its plan's
  name (`PLAN-2027-01-metais-v2-W1-1.md`). Sibling items each add a file,
  so parallel work never edits the same lines; the entries an ADR held
  before this rule stay in it.
- **Never explain what the text used to say.** State the rule; where it is
  counterintuitive, say why it is right. A change log records *that*
  something moved, not the old wording.
- **Prefer prose that states no count it cannot own.** "The fifth command"
  is false the moment a sibling lands in another change, and git merges both
  sides without complaint. Name the thing, not its position.
- **The specification uses RFC 2119 keywords** (MUST, SHOULD, MAY) and only
  in normative text.

## Licensing

- The project is licensed under the Apache License 2.0. `NOTICE` carries the
  attribution to Kottera and is preserved in every distribution.
- Every source file begins with:

  ```go
  // Copyright 2026 Kottera
  // SPDX-License-Identifier: Apache-2.0
  ```

- **Never copy code** from another template engine, scaffolding tool or
  schema validator. Depend on a library through its module, or write the
  code here.
- The repository is private for now. Do not publish its contents outside it.

## Other repositories

Work happens in **this repository only**. `kottera/kottera` holds tera,
which may become a consumer of METAIS; read it when a question turns on what
it says, and never write to it from here. A change tera needs is filed
there as its own work.

## Git and pull requests

- Work on a branch; every change reaches `main` through a pull request.
- Name commits and pull requests `<type>(<feature>): [W<wave>-<item>] <change>`,
  following `docs/dev/work-order-and-pr-naming.md` in `kottera/kottera`; omit
  the bracketed address for work outside a train. Preserve the approved title
  as the merge subject.
- Never include `Co-Authored-By` lines in commit messages or pull request
  descriptions.
- Pass a multi-paragraph commit or pull request message by file
  (`git commit -F`, `gh pr create --body-file`), written under
  `${TMPDIR:-/tmp}` with a unique name, never through a heredoc. Leave the
  file there afterwards.
- Add files by name, never `git add -A`.

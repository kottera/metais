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

The module is `metais.org/metais`. Its `go.mod` `toolchain` line
names the exact toolchain every build and test uses, fetched under the
default `GOTOOLCHAIN=auto` (ADR-010 D1). The gate is three tiers
(ADR-010 D15 item 3, as Amendments A5 and A8 divide it). The first two run
on the hosts the work is done on, and
`.claude/skills/verification-gate/gate.sh` runs either one; the third is
CI:

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
- **CI, the gate of an outside pull request:** `.github/workflows/ci.yml`,
  on GitHub-hosted runners only (ADR-013 D5). It runs the wave tier's list
  on linux/amd64, the race-enabled tests and the build steps on
  darwin/arm64, the DCO check, `shellcheck` over the repository's shell
  scripts, and the site's build and tests. Its linux job runs `gate.sh
  wave` with `METAIS_GATE_P` at the runner's cores, since a hosted runner
  runs nothing else. It runs on every pull request and every push to
  `main`, and a dispatch with its `jobs` input re-runs one job alone. CI
  does not replace the other two: a train item still runs the scoped tier
  before it merges, and a wave still runs the wave tier at its
  verification, because they select what a change reaches. No workflow
  runs on a self-hosted runner or on the `pull_request_target` trigger,
  and every workflow declares a top-level `permissions` block;
  `internal/guarantees`' `TestWorkflows` holds each one to that.

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
ordinary three-way merge. A vector file holding numbers vectorgen writes
gets a three-way merge too, and a line both sides changed only in its
digits keeps one side. Git reads drivers only from a clone's own
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
  - the help, scaffold, tar, YAML-writer, formula and vector-package
    goldens, and the packages pack's vectors expect, by their tests'
    `-update`;
  - every number a vectorgen package's bytes fix in its vector: a
    finding's offset, size or length, and a limit lowered to the
    package's own size. `internal/vectorgen`'s `TestValues` derives each
    from where vectorgen placed the records, and fails on one it does not
    write, so no vector hard-codes an offset;
  - `THIRD-PARTY-NOTICES.txt`;
  - the site package's input schemas of a command's result, from
    `docs/cli/schemas/`, by `site`'s `TestCLIResultSchemas -update`.

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
- **Prove each property once, along each of its axes, at the smallest size
  that can fail it** (ADR-012 D1). Vary each axis a defect could grow along
  (directories, files, depth; process, environment, path mode, day) just
  past where the defect shows. Full-scale counts of a limit live in the
  calibration suite, which runs at release.
- **Before adding a walk over the whole corpus, name what one
  representative per code path would miss**, in the test's comment. If
  nothing, run the representatives.
- **When a test is slow, ask what it proves before making it parallel.**
  Parallelism takes cost off the clock, not off the shared host.
- **Answer an incident in the test infrastructure with a unit test of the
  fix**, not with a run of the suite that would have shown it.
- **A test that is shrunk, merged, moved or deleted names what still
  proves its property**, in its pull request.
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
  `internal/testenv`, never with the developer's real configuration. A
  test that runs the go command gives it `testenv.GoEnv`'s whole
  environment: the host's module cache, read only, and no proxy, so it
  never downloads a module. Of the host's configuration it keeps only the C
  toolchain's, so a race build's C compiler runs where it is a wrapper.
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
- **The repository is public, so everything committed is published**:
  every pushed commit on every branch, its message, and every pull
  request's title, body and comments. Git keeps a commit's contents in the
  history after a later commit removes them, so treat a push as permanent.
  Nothing internal enters it (ADR-013 D4 item 3): no host name, no private
  network's name or URL, and no note that describes a person's machine
  rather than a procedure. Describe a procedure, and say what a host must
  provide rather than which host provides it. Work Order IDs stay: they
  are opaque, and they tie each change to its decision.
- **No secret enters the tree**, not a token, a key or a password, not even
  one that is expired or meant only for a test. A test makes the
  credential it needs. A secret that reaches a pushed commit is rotated at
  once (ADR-013 D4 item 2): removing it from the tree does not take it out
  of the history, and only rotation makes it harmless.

## Other repositories

Work happens in **this repository only**. `kottera/kottera` holds tera,
which may become a consumer of METAIS; read it when a question turns on what
it says, and never write to it from here. A change tera needs is filed
there as its own work.

## Git and pull requests

- Work on a branch; every change reaches `main` through a pull request.
- Name commits and pull requests `<type>(<feature>): [W<wave>-<item>] <change>`,
  and a train's Work Orders `<feature> [W<wave>-<item>]: <action>`, by the
  naming rules below. Preserve the approved title as the merge subject.
- **Sign off every commit** under the Developer Certificate of Origin
  (ADR-013 D3 item 2): `git commit -s` adds the `Signed-off-by:` trailer
  that certifies you may contribute the change under the project's
  licence. A squash merge keeps the trailer in its commit's body. CI's DCO
  check fails a pull request with any commit that lacks one, and there is
  no exemption.
- Never include `Co-Authored-By` lines in commit messages, pull request
  titles and descriptions, or squash messages. CI's co-author check fails
  a pull request whose commits, or whose title and body, carry one.
- **Pass every squash merge's body explicitly.** GitHub's default squash
  body adds a `Co-authored-by` line for every commit author other than the
  merging account's own identity, so a merge without one adds the line
  even to clean commits, and CI cannot see a body typed at the merge.
  Write the pull request's description to a file whose only trailer is
  the `Signed-off-by:` line, then run `gh pr merge <n> --squash --subject
  '<title> (#<n>)' --body-file <file>`. Then check the merged commit's
  message.
- Pass a multi-paragraph commit or pull request message by file
  (`git commit -F`, `gh pr create --body-file`), written under
  `${TMPDIR:-/tmp}` with a unique name, never through a heredoc. Leave the
  file there afterwards.
- Add files by name, never `git add -A`.

### Naming

These rules name Work Orders, pull requests and merge subjects, squash
merges included. Follow them, never recent titles as precedent.

```text
Work Order:  <feature> [W<wave>-<item>]: <action>
PR / merge:  <type>(<feature>): [W<wave>-<item>] <change>
```

- **Feature:** a short lowercase kebab-case name aligned with the owning
  document's name. A plan declares it once as `**Feature:** <feature>`,
  and its briefs, Work Orders and pull requests carry it verbatim. Declare
  an abbreviation explicitly rather than inventing one per item. Work
  outside a train uses the owning feature's or subsystem's established
  name, and records a new name in its owning document when there is one.
- **Train address:** exactly as the breakdown assigned it, wave and item
  (`W1-2`, `W5b-NB1`). Assign it before filing an implementation Work
  Order, and carry it into the pull request and the merge subject. A retry
  or rework of that Work Order keeps it. A separately filed correction gets
  its own item address and links the original in its body; its merge
  carries the correction's address, not the verifier's. An address is
  never reused for another Work Order, nor renumbered to follow merge
  order.
- **Scope:** work outside a train omits the bracketed address, and so
  does plan orchestration. A wave's derivation, supervision, verification
  and closeout use `[W<wave>]` with no item, and say which activity in the
  prose:

  ```text
  metais-public [W1-2]: scan the history for secrets
  docs(metais-public): [W1-2] clear the tree of internal details
  metais-public [W1]: verify the completed wave
  docs(metais-public): [W1] record the wave's closeout
  metais-ux: orchestrate the plan
  ```

- **Type:** one of `feat` (a new capability), `fix` (a defect repaired),
  `refactor` (restructured, no behaviour changed), `perf`, `test` (tests
  only), `docs`, `build`, `ci`, `chore` (other maintenance) or `revert`.
  Classify the final change: a fix with its regression test is `fix`, not
  `test`. A Work Order's title has no type, because discovery may change
  its outcome.
- **The prose:** start with a concrete imperative verb and name the
  resulting behaviour or deliverable, never "address review feedback",
  "finish W1-2" or "improve handling". Aim for 90 characters before the
  appended pull request number; clarity wins over the limit. Identifiers,
  ADR references, dependencies and acceptance details go in the body.
  When the scope changes, rewrite the prose and the type, and keep the
  feature and the address.
- **When they apply:** check the title when filing a Work Order, when
  opening or updating a pull request, and before merging. The merge keeps
  the approved pull request title as its subject, with the pull request
  number appended once. A commit already in history, or a finished Work
  Order, is never renamed for these rules.

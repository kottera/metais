# SC9's round: a package written from the specification alone

This is the runbook for the round that proves PRD SC9 (plan §6). The plan
orchestrator files it as a `dogfood-round` act once the worked package of
`spec/B-worked-package.md` has merged, naming this page as its `runbook`
and the commit it is run against as its `build`. Whoever runs the round
reads this page whole first. The author does not read it: what the author
is given is below.

## What the round establishes

PRD SC9: an author working only from the specification, without the tool's
source, writes a package that the tool checks, tests and packs cleanly.

The round shows that the specification, starting at its worked package
(ADR-011 D4 item 4), is enough to write a package. It also shows that the
authoring golden path of PRD R17 item 2 can be followed from the text:
init, edit, check, test, pack. It does not show that any package is well
designed, and it does not rerun the conformance suite.

## Who does what

- **The runner** is whoever runs the round. They prepare what the author is
  given, collect the evidence and judge the result.
- **The author** writes the package: a person, or an AI harness started
  with nothing but what is given below. The author has not worked on this
  repository's source. An AI harness has no tool that reaches the
  repository, the network or any file outside its working directory.

## What the author is given

Two things, both from the commit the act names as its `build`. The runner
prepares them at the root of a clone of this repository, with the author's
directory, the author's working directory, named by an absolute path that
does not exist yet:

1. **The specification:** the `spec/` directory of that commit, and nothing
   else from the repository:

   ```bash
   mkdir <author's directory>
   git archive --format=tar <commit> spec | tar -x -C <author's directory>
   ```

   The author starts at `spec/B-worked-package.md`.
2. **A built `metais` binary of that commit**, built by the runner:

   ```bash
   git checkout <commit>
   CGO_ENABLED=0 go build -trimpath -o <author's directory>/metais ./cmd/metais
   <author's directory>/metais version
   ```

   The runner records the binary's SHA-256 and the output of
   `metais version`.

The task the author is set, in these words:

> Write a METAIS package of your own choosing, on a subject that is not the
> release notes of `spec/B-worked-package.md`. It must have a manifest, at
> least one input with its schema and its meaning document, at least one
> template, and at least one example with its expected outputs. Work only
> from `spec/` and the `metais` binary; `metais help` is part of the binary.
> When `metais check`, `metais test` and `metais pack` all accept the
> package, stop. Keep a record of every command you run and its full
> output. Each time the text did not tell you what to do, note where you
> were reading, what you needed, and what you did instead.

## What the author is not given

- **The tool's source:** no `cmd/`, no `internal/`, no `go.mod`. Neither
  the repository nor a clone, nor a copy of any file of it outside `spec/`.
- **This repository's other documents:** `docs/`, `README.md`, `AGENTS.md`,
  and the example packages under `examples/`. The worked package is the only
  example the author is given, and it is inside `spec/`.
- **Help from anyone who knows the tool.** A question the author asks is
  answered "the text is all there is", and recorded as a point where the
  text did not tell them what to do.

## The commands that judge the result

The runner runs these with the binary the author was given, on a fresh
copy of the author's package source directory, `<package directory>`. The
copy is `package` in a judging directory, named by an absolute path that
does not exist yet, and the commands run from the judging directory, the
copy's parent:

```bash
mkdir <judging directory>
cp -R <package directory> <judging directory>/package
cd <judging directory>
<author's directory>/metais check package
<author's directory>/metais test package
mkdir packed
<author's directory>/metais pack package --out packed
```

**The round passes when each command exits 0 and reports no finding of
error severity.** A warning does not fail the round. Each one is listed in
the report, with the author's reason for keeping it, or with "no reason"
when the author did not give one. The runner also runs each command with
`--json`, to keep the findings in full.

The round also fails when the package lacks any part the task asks for,
when it is the worked package with its names changed, or when the evidence
shows the author read something they were not given.

## The evidence to report back

Report evidence, not an attestation: what was run and what it printed,
exactly, so that a reader who was not there can judge the round. W4-A1R's
report in plan §10 is the model: the commit, the host, and the measured
outcome in the tool's own words.

1. **The build:** the commit, the binary's SHA-256, the output of
   `metais version`, and the host's OS, architecture and Go version.
2. **The package written:** the author's source directory, as an archive
   attached to the report, and the `.metais` file that `metais pack`
   wrote, with its SHA-256 and the package digest that `pack` printed.
3. **The judging run:** each command of the previous section, with its full
   standard output and standard error and its exit status, in human form
   and with `--json`.
4. **The author's record:** every command the author ran, in order, with
   its full output and exit status, the failed ones included. An AI
   harness's transcript serves, when it shows each command and its output.
5. **What the author read:** each file of `spec/` the author opened, and
   each `metais help` page, in the order read.
6. **Every point where the text did not tell the author what to do:** for
   each, the file and section being read, what the author needed to know,
   and what the author did instead: guessed, experimented with the binary,
   or gave up on part of the package. A point the binary's output answered
   counts too, because the text did not.
7. **The verdict:** pass or fail, by the rule above, with the failing
   command's findings quoted when it fails.

## What happens next

The plan orchestrator reads the report. Each point where the text did not
tell the author what to do becomes a fix item against the specification,
or is recorded as a ruling that the text is enough. A finding about the
tool becomes a fix item against the tool. The round is rerun when a fix
changes `spec/B-worked-package.md` or the authoring chapter.

## Evidence on the post-Wave-2 binary (PLAN-2026-10-metais-ux W3-5)

This is the harness's evidence for the round, recorded for whoever takes it
(PLAN-2026-10-metais-ux §8). It is not the round. The round is Work Order
`01a15abf-6347` of PLAN-2026-10-metais-v1, and it is not done until Ed
confirms it.

**What ran.** Scenario A4 of the UX set (`docs/dev/ux-testing.md`), an
A1-shaped run. Its prompt is this page's task, minus the request to keep a
record of every command, which the transcript keeps, and plus the words
"starting at spec/B-worked-package.md". The session ran in a fresh directory
holding only `spec/`, as `git archive` of the commit gives it, with the
binary under test first on `PATH`. It had no tool source, no `docs/`, no
`examples/` and no repository above it. The runner isolates the session as
the method's isolation checks say, and every one of them passed on both
runs.

| | |
|---|---|
| Commit | `29f077f3edbadb5f99ec3998f66803e99103a0f3`, on branch `feat/metais-ux-w3-5-sc1-sc9-evidence`. Its binary closure and its `spec/` are `main`'s at `63315621`: the branch changes only `internal/tools/ux` and documents |
| metais | `metais 0.0.0-devel`, `specification 1.0`; SHA-256 `6d0d8c1e7082cc5e5e191c08689cf512c6eeb153e2e529f4b553073ecdfff9ca` |
| Host | a tera dispatch, darwin/arm64, go1.26.2 |
| Harness | Claude Code 2.1.299, `claude -p`, no skill installed |
| Models | `opus` (claude-opus-5-5) and `sonnet` (claude-sonnet-5-5), one run each, 2026-10-22 |
| Transcripts | artifact `art_nof7la4jhanh3k5z3kfxevib4dujterdjhwhtq2ewli7cy2bsa2q`: the scrubbed set `ux-w3-5`, `runs/A4-spec-only-package/<model>/`, which holds the package source the session wrote under `work/recipe-card/`, and `judging/A4-<model>.log`, the judging commands below with their full output |

**The judging commands**, run as this page says, on a fresh copy of each
session's package source:

| | opus | sonnet |
|---|---|---|
| Package | `recipe-card`: one input, `recipe`, with its schema and meaning document; two templates, a recipe card and a shopping list; the example `pancakes` with both expected outputs | `recipe-card`: one input, `recipe`, with its schema and meaning document; one template, a recipe card; the example `soup` with its expected output |
| `check package` (human and `--json`) | exit 0; `check: 0 errors, 0 warnings, 0 notes` | exit 0; `check: 0 errors, 0 warnings, 0 notes` |
| `test package` (human and `--json`) | exit 0; `pancakes: pass` | exit 0; `soup: pass` |
| `pack package --out packed` (human and `--json`) | exit 0; `Packed recipe-card-1.0.0.metais: 7137 bytes, 14 entries` | exit 0; `Packed recipe-card-1.0.0.metais: 5207 bytes, 11 entries` |
| Package digest | `sha256:b5772c41173587075c77b63c3b665267de3266ac21c02a994309287f96f7334b` | `sha256:ce055db664b6527dbdac71815a68ce6d80983918196eb0b9b4f776a6a702aefc` |
| `.metais` SHA-256 | `1a16e8882a4bfa8ff9c3c779515967b6a08d5bdf7c29ddc94681a76673fabf2d` | `864084687ac1f8e505beb05d300326b61fc87cc506a2b14983dc7b15b881dc1d` |

No command reported a warning.

**What the transcripts show against the round's bar:**

- **`check`, `test` and `pack`.** Both runs pass, by the table above. Each
  session's own last `test`, `pack` and `check` passed after its last edit
  (opus turn 11, line 58; sonnet turn 16, line 63). The UX scorer's
  `checks-tests-packs` check passes on both.
- **Every part the task asks for.** Both packages have a manifest, an input
  with its schema and meaning document, a template, and an example with its
  expected outputs. Neither is the worked package with its names changed:
  each is a recipe card, not release notes, with its own schema, templates
  and example.
- **What the author read.** Each session read `spec/B-worked-package.md`
  first (opus turn 1, line 3; sonnet turn 2, line 4), then `metais help`
  pages (opus `help`, `help check`, `help test` and `help pack`, turn 2,
  line 8; sonnet `help`, turn 2, line 4, and `help render`, turn 14,
  line 55). Opus also searched chapters 5, 7 and 8 (turn 3, line 14), and
  read MTS-SUP-007 in chapter 6 (turn 8, line 48). No tool call names a
  file outside the session's directory, and neither session read anything
  but `spec/`, the binary's help and output, and its own files.
- **Where the text did not tell the author what to do.** Each session's
  last words list such points (opus line 66; sonnet line 65):
  - **Whether a bare number in YAML is a string** (B.7). Both met it. Opus
    found the answer in MTS-SUP-007, and sonnet found it when `render`
    refused `amount: 1` as `input.type` (turn 10, line 40).
  - **An `expected/` directory made before its files** (B.7, B.8). Both
    made one, and `check` (opus) or `render` (sonnet) refused the source as
    `package.layout.empty_directory`. Sonnet asks for B.8 to say that the
    directory must not exist before the first render.
  - **Check reports one finding where B.6 says every step's.** Opus's first
    `check` (turn 4, line 17) ran on a source whose files were all empty,
    because here-documents failed. It reported only the empty directory,
    and nothing of the empty `metais.yaml`. B.6 says check "reports every
    finding of every step in one run". MTS-AUT-003 says a refused walk of
    a source directory gets only its own findings, so the binary does what
    chapter 13 says, and B.6's sentence is what misled.
  - **Check accepts an example whose data its schema refuses** (sonnet, its
    point 3). Only `render` and `test` read the example's data.
  - Opus also lists loop counters (B.5), arrays of plain values (B.4), a
    path built from an input (B.3) and recording the expected files (B.8).
    Sonnet lists the whitespace of a `for` loop (B.5) and what `render`
    prints.

**What a transcript cannot show:**

- **Whether the text alone was enough for a person.** These are two
  harness runs, not an author's.
- **The runbook's exact set-up.** The author was not given `./metais` in
  its directory: it had `metais` on `PATH`. Its sandbox refused shell
  here-documents, which left every file opus and sonnet first wrote empty
  (opus turn 4, line 17; sonnet turn 3, line 9). Both rewrote the files with
  the Write tool. A person's host would not refuse them.

**Findings for the wave.** Neither run fails the round's bar. The points
above are the round's to rule on, by "What happens next". One of them is a
candidate fix against the text: B.6's "reports every finding of every step
in one run" against MTS-AUT-003.

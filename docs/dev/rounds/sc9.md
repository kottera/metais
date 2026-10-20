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

Two things, both from the commit the act names as its `build`:

1. **The specification:** the `spec/` directory of that commit, and nothing
   else from the repository:

   ```bash
   git archive --format=tar <commit> spec | tar -x -C <author's directory>
   ```

   The author starts at `spec/B-worked-package.md`.
2. **A built `metais` binary of that commit**, built by the runner:

   ```bash
   git checkout <commit>
   CGO_ENABLED=0 go build -trimpath -o <author's directory>/metais ./cmd/metais
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

The runner runs these on the author's source directory, with the binary the
author was given, from a fresh copy of the directory and from its parent:

```bash
metais check <dir>
metais test <dir>
metais pack <dir> --out <empty directory>
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

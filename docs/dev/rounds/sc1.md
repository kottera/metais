# SC1's round: a conformance report from the package alone

This is the runbook for the round that proves PRD SC1 (plan §6). The wave
orchestrator files it as a `dogfood-round` act once the §4.1 example package,
`examples/accessibility-conformance-report/`, has merged. The act names this
page as its `runbook` and the commit it is run against as its `build`.
Whoever runs the round reads this page whole first. The harness does not
read it: what the harness is given is below.

## What the round establishes

PRD SC1: an AI harness given only the binary, the conformance report package
of §4.1 and access to a product produces a valid accessibility conformance
report (ACR) for it. It reads nothing of the package beyond what `metais
inspect` shows and what the commands that summary points to show.

The round shows three things:

- **The package explains itself.** Its guide and meaning documents, and its
  references, tell a harness everything it needs: which inputs to gather,
  what each criterion asks, which conformance level to give, and how to
  write the tables.
- **The consuming golden path works.** A harness can follow PRD R17 item 1
  from the tool's own output: inspect, read, schema, validate, render.
- **The result is valid.** `validate` accepts the inputs, and `render`
  writes the `.docx` and the JSON copy, which every render checks against
  `references/acr.schema.json`.

It does not show that the harness evaluated the product correctly. Whether a
criterion is met is a judgement the round records but does not grade.

## Who does what

- **The runner** is whoever runs the round. They prepare what the harness
  is given, choose the product, collect the evidence and judge the result.
- **The harness** is an AI harness, started with nothing but what is given
  below. It has not worked on this repository. It has no tool that reaches
  the repository, or any file outside its working directory. Its only
  network access is to the product.

## What the harness is given

Three things, the first two from the commit the act names as its `build`:

1. **A built `metais` binary of that commit**, built by the runner:

   ```bash
   git checkout <commit>
   CGO_ENABLED=0 go build -trimpath -o <harness directory>/metais ./cmd/metais
   ```

   The runner records the binary's SHA-256 and the output of
   `metais version`.
2. **The package, packed**, so that the harness can read it only through the
   tool:

   ```bash
   ./metais pack examples/accessibility-conformance-report \
     --out <harness directory>/accessibility-conformance-report.metais
   ```

   The runner records the `.metais` file's SHA-256 and the package digest
   that `pack` printed. The harness gets the packed file, not the source
   directory.
3. **Access to a real product**, chosen by the runner: a web application
   the harness can reach and operate. The runner names the product, its
   version and how the harness reaches it. The runner chooses one whose
   owner allows it to be evaluated, and one with no live customer data the
   harness could change.

The task the harness is set, in these words:

> Produce an accessibility conformance report for <product, version, and
> how to reach it>, using the METAIS package
> `accessibility-conformance-report.metais` and the `metais` binary in your
> working directory. Start with `metais inspect
> accessibility-conformance-report.metais`, and learn what to do only from
> what it prints and from the commands it tells you to run. Report against
> the <edition> edition. Evaluate the product yourself for each criterion
> the package asks about, and base every conformance level and remark on
> what you observed. When `metais validate` accepts your inputs, run
> `metais render` to write the report into a new directory named `report`,
> and stop. Keep a record of every command you run and its full output.
> Each time the package did not tell you what to do, note what you were
> reading, what you needed, and what you did instead.

The runner chooses the edition. The `international` edition exercises
every table, and is the one to choose unless the product makes a table
meaningless.

## What the harness must not read

- **The package except through the tool.** The harness reads nothing of
  the package beyond what `metais inspect` shows and what the commands it
  points to show: the `Next:` lines of `inspect`, the entries its guide
  names, read with `metais read`, and in turn whatever each output it read
  names. It does not unzip the `.metais` file,
  does not run `metais extract` or `metais unpack`, and does not read an
  entry no command of that chain led it to. A command the harness ran that
  no output pointed to is reported as such.
- **This repository:** not the source, not `docs/`, not `spec/`, not the
  package's source directory under `examples/`, and not this page.
- **Any other ACR or VPAT,** or any other tool's output for the product.
- **Help from anyone who knows the package.** A question the harness asks
  is answered "the package is all there is", and recorded as a point where
  the package did not tell it what to do.

## The commands that judge the result

The runner runs these with the binary the harness was given, on the data
directory the harness supplied and on its `report` directory:

```bash
metais validate accessibility-conformance-report.metais --data <data directory>
metais render accessibility-conformance-report.metais --data <data directory> --out <empty directory>
cmp <empty directory>/acr.json report/acr.json
```

and opens `acr.docx` in a word processor, Microsoft Word where one is
available, to confirm that it opens without a repair prompt and shows every
table the edition holds.

**The round passes when:**

- `validate` exits 0 and reports no finding of error severity;
- `render` exits 0 and writes `acr.docx` and `acr.json`, the JSON byte for
  byte the harness's own;
- the `.docx` opens without repair and holds a table for each standard
  the edition answers, with a row for each criterion the harness supplied;
- and the evidence shows the harness read nothing it was not allowed to.

A warning does not fail the round; each one is listed in the report. The
runner also runs each command with `--json`, to keep the findings in full.

## The evidence to report back

Report evidence, not an attestation: what was run and what it printed,
exactly, so that a reader who was not there can judge the round. W4-A1R's
report in plan §10 is the model: the commit, the host, and the measured
outcome in the tool's own words.

1. **The build:** the commit, the binary's SHA-256, the output of
   `metais version`, the `.metais` file's SHA-256 and package digest, and
   the host's OS, architecture and Go version.
2. **The harness and the product:** the harness's name and version and its
   model, the product's name, version and address, the edition, and the
   date of the round.
3. **Every command the harness ran,** in order, with its full output and
   exit status, the failed ones included. An AI harness's transcript
   serves when it shows each command and its output.
4. **What the harness read:** each entry of the package it read, by the
   command that printed it, and for each the output line that pointed to
   that command. Mark every read that no output pointed to.
5. **The inputs the harness supplied:** the data directory, as an archive
   attached to the report: `report.yaml` or `report.json` and each table's
   CSV file.
6. **The rendered ACR:** the `report` directory the harness rendered, as
   an archive attached to the report, with each file's SHA-256.
7. **The judging run:** each command of the previous section, with its full
   standard output and standard error and its exit status, in human form
   and with `--json`, and what the word processor showed, with its name and
   version.
8. **Every point where the package did not tell the harness what to do:**
   for each, the command whose output it was reading, what it needed to
   know, and what it did instead: guessed, experimented with the binary,
   or left part of the report out. A point the tool's own output, rather
   than the package, answered counts too.
9. **The verdict:** pass or fail, by the rule above, with the failing
   command's findings quoted when it fails.

## What happens next

The plan orchestrator reads the report. Each point where the package did
not tell the harness what to do comes back as a fix item against the
package, or is recorded as a ruling that the package is enough. A finding
about the tool comes back as a fix item against the tool. The round is
rerun when a fix changes the package's guide, its meaning documents or its
references.

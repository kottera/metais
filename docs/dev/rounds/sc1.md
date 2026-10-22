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

Three things, the first two from the commit the act names as its `build`.
The runner prepares them at the root of a clone of this repository, with
the harness directory, the harness's working directory, named by an
absolute path that does not exist yet:

1. **A built `metais` binary of that commit**, built by the runner:

   ```bash
   git checkout <commit>
   mkdir <harness directory>
   CGO_ENABLED=0 go build -trimpath -o <harness directory>/metais ./cmd/metais
   <harness directory>/metais version
   ```

   The runner records the binary's SHA-256 and the output of
   `metais version`.
2. **The package, packed**, so that the harness can read it only through the
   tool:

   ```bash
   <harness directory>/metais pack examples/accessibility-conformance-report \
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
> working directory. Start with `./metais inspect
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

The runner runs these in the harness directory, with the binary the
harness was given, on the data directory the harness supplied and on its
`report` directory. The judging render writes into `judged`, a directory
that does not exist yet:

```bash
cd <harness directory>
./metais validate accessibility-conformance-report.metais --data <data directory>
./metais render accessibility-conformance-report.metais --data <data directory> --out judged
cmp judged/acr.json report/acr.json
```

and opens `judged/acr.docx` in a word processor, Microsoft Word where one is
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

## Evidence on the post-Wave-2 binary (PLAN-2026-10-metais-ux W3-5)

This is the harness's evidence for the round, recorded for whoever takes it
(PLAN-2026-10-metais-ux §8). It is not the round. The round is Work Order
`01a15abf-4d58` of PLAN-2026-10-metais-v1, and it is not done until Ed
confirms it.

**What ran.** Scenario C6 of the UX set (`docs/dev/ux-testing.md`): a
C2-shaped run, the prompt "render this file with new fictional data:
accessibility-conformance-report.metais", in a fresh directory holding only
`examples/accessibility-conformance-report` packed by the binary under
test, with that binary first on `PATH`. The runner isolates the session as
the method's isolation checks say, and every one of them passed on both
runs.

| | |
|---|---|
| Commit | `29f077f3edbadb5f99ec3998f66803e99103a0f3`, on branch `feat/metais-ux-w3-5-sc1-sc9-evidence`. Its binary closure is `main`'s at `63315621`: the branch changes only `internal/tools/ux` and documents, which `cmd/metais` does not import. It precedes W2-NB1 (#217), which changed three of the binary's sentences, among them the name of `inspect`'s next-step section |
| metais | `metais 0.0.0-devel`, `specification 1.0`; SHA-256 `6d0d8c1e7082cc5e5e191c08689cf512c6eeb153e2e529f4b553073ecdfff9ca` |
| Package | `accessibility-conformance-report.metais`, SHA-256 `95fa5d7eaeb29b3dd14482172e7854b7d54b9dbda6a28384ad42f2d4921e3d68`, package digest `sha256:1d29c7f0c90d4a4ade57ba4cc3ab68e430081dd04bfd15af32c8ba34f889b157` |
| Host | a tera dispatch, darwin/arm64, go1.26.2 |
| Harness | Claude Code 2.1.299, `claude -p`, no skill installed |
| Models | `opus` (claude-opus-5-5) and `sonnet` (claude-sonnet-5-5), one run each, 2026-10-22 |
| Transcripts | artifact `art_nof7la4jhanh3k5z3kfxevib4dujterdjhwhtq2ewli7cy2bsa2q`: the scrubbed set `ux-w3-5`, `runs/C6-render-acr-fictional/<model>/`, and `judging/C6-<model>.log`, the judging commands below with their full output |

**The judging commands**, run as this page says, on a copy of each run's
`data/` and of the directory it rendered into, standing for `report`:

| | opus | sonnet |
|---|---|---|
| Edition | `wcag` | `international` |
| `validate` (human and `--json`) | exit 0; `0 errors, 0 warnings, 0 info` | exit 0; `0 errors, 0 warnings, 0 info` |
| `render --out judged` (human and `--json`) | exit 0; writes `acr.docx` and `acr.json` | exit 0; writes `acr.docx` and `acr.json` |
| `cmp judged/acr.json report/acr.json` | exit 0 | exit 0 |
| `acr.json` SHA-256 | `5d544be39c3b54b374d052ffa308be3240167c72787b85547418122be83fa63e` | `57e2e08deab23b6694c09f691d42c60ea33df1603aab8f7036e149f1e6341cf2` |
| `acr.docx` SHA-256, the same as the session's | `f3fac2c66c33f08bca9449bee2cd8d2e755fdad8f3c637a1b78553d105f33ea5` | `42f47c56254782213e5d4a4827d873d8e50a0b709942fc371fde099b8542c96e` |
| Tables in `word/document.xml`, and their rows | 1 table, 56 rows: a header and the 55 WCAG criteria `wcag.csv` lists | 3 tables, 310 rows: a header each and the 55, 104 and 148 criteria of `wcag.csv`, `section508.csv` and `en301549.csv` |

**What the transcripts show against the round's bar:**

- **`validate` and `render`.** Both runs pass the bar's first two items: the
  table above. Each session's own `validate` and `render` passed too (opus
  turn 10, line 59; sonnet turn 10, line 44).
- **The golden path.** Each session ran `metais inspect` before any other
  `metais` command but `--version`, then read the entries its guide names
  with `metais read` and the fields with `metais schema --fields` (opus
  turns 3 to 6, lines 7 to 36; sonnet turns 3 and 4, lines 7 and 12). Sonnet
  also ran `metais sample` (turn 6, line 22), and read the shipped
  international example's tables with `metais read` to take the criterion
  numbers from them (turn 7, line 28). No session unzipped the `.metais`
  file or ran `metais extract` or `unpack`.
- **Reads no output pointed to.** Both sessions read the package's raw bytes
  once, before `inspect`. Opus ran `head -c 6000` on it (turn 2, line 4),
  and sonnet ran `head -c 3000` (turn 1, line 2) and then read it with the
  Read tool (turn 2, line 4). What they read was the stored `README.md`,
  which told them to run `metais inspect`, which they then did. By this
  page's letter, these are reads of the package that no output pointed to,
  so neither run meets the bar's last item as it stands.

  C6's prompt does not say what this round's task says: "Start with
  `./metais inspect accessibility-conformance-report.metais`". So these
  runs do not show what a session given the round's own task does here.
  Opus also listed names beside its working directory
  (`find .. -maxdepth 3 -name "*metais*"`, turn 1, line 2), which found only
  its own package.
- **Where the package left the session to choose.** Opus chose the `wcag`
  edition. Sonnet chose `international`, the edition this page says to
  choose. The C2-shaped prompt names no edition. Opus put the product
  description on one line after its first render showed that the YAML
  block's line breaks start new paragraphs (turn 10, line 57), as
  `inputs/report.md` says they do. Neither session's last words name a
  point where the package did not tell it what to do. C6's prompt does not
  ask for such points, as the round's task does.

**What a transcript cannot show:**

- **That the harness evaluated a real product.** The C2-shaped prompt asks
  for fictional data and names no product, so every level and remark is
  invented. Each session says so in the report.
- **That `acr.docx` opens in a word processor without a repair prompt.** No
  word processor was run. On the dispatch's host, macOS `textutil` read each
  `judged/acr.docx` as text, which is not the bar's check.
- **The runbook's exact set-up.** The session was not given `./metais` in
  its directory or the round's own task: it had `metais` on `PATH` and C6's
  prompt. Its sandbox also refused shell here-documents, which both
  sessions met and worked round with the Write tool (opus turn 7, line 41;
  sonnet turn 8, line 34). A person's host would not refuse them.

**Findings for the wave.** None is this item's to fix, and none is marked
fixed here.

- **SC1's bar, last item.** Both runs read the `.metais` file's raw bytes
  before `inspect` (above). Whether a run of the round's own task does the
  same is for the round to show.
- **A scorer mis-score.** The UX scorer fails opus's run on
  `no-zip-extraction` and `package-untouched` for
  `unzip -p out/acr.docx word/document.xml` (turn 9, line 51). That command
  reads the session's own rendered `.docx`, not the package. R-UX-8's
  "extracts the package" does not describe it, and `package-untouched`'s
  detail says "the package is extracted". `no-zip-extraction`'s wording,
  "an archive", does cover a `.docx`. Sonnet's run scores pass.

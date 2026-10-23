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

This is the harness's evidence for the round, recorded for whoever took it
(PLAN-2026-10-metais-ux §8). It is not the round. The round is Work Order
`01a15abf-4d58` of PLAN-2026-10-metais-v1, and its result is recorded below, in
"The round on 339864912a01 (PLAN-2026-10-metais-v1 W7-A2)".

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
  "an archive", does cover a `.docx`. The scorer at `dae1ef72`, which
  holds W3-1's settlement of the baseline's mis-scores, still scores it so.
  Sonnet's run scores pass.

## The round on 339864912a01 (PLAN-2026-10-metais-v1 W7-A2)

This is the round, Work Order `01a15abf-4d58`, run on 2026-10-22 by this
page as written, except where **Deviations** says otherwise. Ed ruled on
2026-10-22 that it runs on current `main` rather than on `dd179daa`, the
build the Work Order first named, and that it isolates the harness by the
method SC9's round used the same day. `main` was then 75 commits ahead of
`dd179daa`. This page and `examples/accessibility-conformance-report/` are
the same at the build as on `main` when this section was written.

**Verdict: pass.** `validate` exits 0 with no finding of any severity, in
human form and with `--json`. `render` exits 0 and writes `acr.docx` and
`acr.json`, and the JSON is byte for byte the harness's own. The `.docx`
opened without a repair prompt and showed a table for each of the
international edition's three standards, with a row for each criterion the
harness supplied; Ed checked it in Pages, since no Microsoft Word was
available (**The word processor**). The evidence shows the harness read
nothing it was not allowed to.

### The build

| | |
|---|---|
| Commit | `339864912a0150b351a66bdcb4d6798506d41134`, `main` at 2026-10-22 15:18:39 -0400 ("docs(metais-public): [W1-3] the community files (#244)") |
| Build | `CGO_ENABLED=0 go build -trimpath -o <harness directory>/metais ./cmd/metais`, from a fresh clone at the commit, under the default `GOTOOLCHAIN=auto` |
| metais | `metais 0.0.0-20261022191839-339864912a01`, `specification 1.0`, `METAIS is made by Kottera.`; SHA-256 `721fbc1d59c7064c5429dc49f70ac1a776b8716e53671ef98025448eb6bf8b30`, the same after the session, and the same binary SC9's round gave its author |
| Package | `<harness directory>/metais pack examples/accessibility-conformance-report --out <harness directory>/accessibility-conformance-report.metais`, exit 0: `Packed the package: 86708 bytes, 47 entries, package digest sha256:1d29c7f0c90d4a4ade57ba4cc3ab68e430081dd04bfd15af32c8ba34f889b157`, the package digest of W3-5's evidence above. The `.metais` file's SHA-256 is `d7bc52c8b6c75ef712ab9698ea58f065fda4026a485944743efd6dd746d1c941`, the same after the session |
| Harness directory | a new absolute path outside the clone, holding the binary and the packed package, and nothing else |
| Host | a developer laptop, Apple M2 Max, macOS 26.5.2 (25F84), Darwin 25.5.0, darwin/arm64; host Go go1.26.1 |
| Evidence kept | outside the repository, with the round's report on the host that ran it: the full `stream-json` transcript (SHA-256 `753a11cbfd0815b77868aea72029ea8ee42e540a73a61862eb8bd287d4954648`), the harness directory as the session left it (its `worklog/commands.log` and `NOTES.md`, its scripts and its evidence of the product), the data and report archives below, a record of every tool call with its full output, the judging logs, the runner's settings and the isolation probes. None is committed: they hold host paths, a user name and the test account's password, and "Scrubbing a set" in [ux-testing.md](../ux-testing.md) scrubs a set `internal/tools/ux` wrote, which a round run by hand does not have |

### The harness and the product

| | |
|---|---|
| Harness | Claude Code 2.1.301, `claude -p` with `--model opus`, which resolved to `claude-opus-5-5`. 60 turns and 57 tool calls (39 Bash, 10 Write, 6 Edit, 2 Read) in 789 s; it ended `success` with `end_turn` and no permission denial |
| Product | tera's web user interface: the issuer's sign-in page and its error states, the passkey offer after a password sign-in, Home, the account menu, and the Account and security page. tera `v0.8.0-rc.3-42-g48637addf5`, served by a disposable testbed daemon on the same laptop at `http://localhost:41080`, signed in as the testbed's test account. Kottera owns it and allows the evaluation, and it holds no customer data. No plugin was installed, so no plugin app was in scope |
| Edition | `international` |
| Date | 2026-10-22, 21:45 to 21:58 EDT |
| Task text | this page's, word for word, with the product and the edition filled in. The product slot also named how to reach the product from inside the sandbox: a `sc1-browser` module that launches the staged Chromium, and `curl --noproxy ''` (**Isolation**) |

### Every command the harness ran

The transcript holds every tool call with its input and full output, and
none failed at the tool level. The harness also ran every command but its
first few through a logger of its own, which wrote each command and its
full output and exit status to `worklog/commands.log`, and listed in its
`NOTES.md` the few that ran outside it. Its `metais` commands, in order:

| Call | Turn | Command | Result |
|---|---|---|---|
| 2 | 4 | `./metais inspect accessibility-conformance-report.metais` | exit 0 |
| 3 | 5 | `metais read` of `inputs/report.md`, then of `inputs/tables.md` | exit 0 |
| 4 | 6 | `metais read` of `inputs/conformance.md`; `metais schema` of `report`, `wcag`, `section508` and `en301549`, each `--fields` | exit 0 |
| 5 to 7 | 9 to 11 | `metais read` of `references/wcag-2.2.md`, `references/section-508.md`, `references/en-301-549.md` and `references/sources.md` | exit 0 |
| 54 | 121 | `./metais validate accessibility-conformance-report.metais --data data` | exit 0, `0 errors, 0 warnings, 0 info` |
| 55 | 122 | `./metais render accessibility-conformance-report.metais --data data --out report` | exit 0, writes `acr.docx` and `acr.json` |

The first call, before `inspect`, listed the harness directory and wrote
the logger; it read nothing of the package. Every other call evaluated the
product or wrote the data: `curl --noproxy ''` of the pages and
stylesheets, Playwright scripts it wrote under `scripts/`, `python3 -I`
summaries of its own evidence, and `scripts/tables.py`, which wrote the
tables' CSV files.

### What the harness read

| Entry | Command | The output line that pointed to it |
|---|---|---|
| `inputs/report.md` | `metais read` | `inspect`'s `Next:` line `metais read <package> inputs/report.md`, and the guide |
| `inputs/tables.md` | `metais read` | `inspect`'s `Next:` line `metais read <package> inputs/tables.md` |
| `inputs/conformance.md` | `metais read` | the guide's link `[what the conformance levels mean](inputs/conformance.md)` |
| the fields of `report`, `wcag`, `section508` and `en301549` | `metais schema … --fields` | `inspect`'s `Next:` line `metais schema <package> <input> --fields`, one per input |
| `references/wcag-2.2.md`, `references/section-508.md`, `references/en-301-549.md` | `metais read` | the guide, which names each, and `inspect`'s References list |
| `references/sources.md` | `metais read` | the guide, on W3C's notice, and `inspect`'s References list |

No read lacked a pointer. The harness did not read the `.metais` file's
bytes, unzip it, or run `extract`, `unpack`, `sample` or `render
--sample`, and it read no example. This answers the question W3-5's
evidence left for the round: given this page's task, which says to start
with `inspect`, the harness started there and read the package only
through the tool.

### The inputs and the render

The data directory, kept as `data.tar.gz`, SHA-256
`1a0f1872a63b081446e079ef22a3856d220ca69f3a10a9025d098ef3f38d8526`:

| File | SHA-256 | Rows | Levels |
|---|---|---|---|
| `report.yaml` | `9f38f57f02e9a414e236fe070404a3260b59deca7a1e4f0aae490ee43a2c38e2` | | edition `international`; the vendor `Not stated to the evaluator` (point 1 below) |
| `wcag.csv` | `ba4ce949d6b2d312f0c4f0920ab4afbaff3ea6784880baf30a2f3c7729308016` | 86 | Supports 45, Partially Supports 19, Does Not Support 1, Not Applicable 18, Not Evaluated 3 |
| `section508.csv` | `903fa46b95d720db0de9526992d0efced84f7af22340bbc0cbaa0176c5d367a3` | 104 | Supports 14, Partially Supports 7, Not Applicable 78, Not Evaluated 5 |
| `en301549.csv` | `d942901dc247358caa9a558ee4283d4bac51646c7a8954607a76ae563c4e1d34` | 148 | Supports 19, Partially Supports 7, Not Applicable 119, Not Evaluated 3 |

The `report` directory the harness rendered, kept as `report.tar.gz`,
SHA-256 `5d1d3c0dab4ce551599c32650ba869ed4c7a83f34b24138c4d81e80efc7aa88e`:
`acr.docx`, 20933 bytes, SHA-256
`bd8dc31fa8ef27d73594fa783e7d859eaa9f826555b01b46b39a3160aa9c1f90`, and
`acr.json`, 101874 bytes, SHA-256
`60d6f71b6f04d402d06ccab3cf080fb67ad4ed3bfc3418432477dab358d4585b`.

### The judging run

The commands of "The commands that judge the result", in the harness
directory, with the binary the harness was given, on its `data` and
`report` directories. Every standard error was empty.

`./metais validate accessibility-conformance-report.metais --data data`,
exit 0:

```text
0 errors, 0 warnings, 0 info

Next:
  metais render <package> --data <data dir> <your --set, --set-json and --inputs-json flags> --out <dir>  — Renders the outputs from the inputs just validated into a new directory.
```

`./metais render accessibility-conformance-report.metais --data data --out judged`,
exit 0:

```text
20933	sha256:bd8dc31fa8ef27d73594fa783e7d859eaa9f826555b01b46b39a3160aa9c1f90	acr.docx
  678	sha256:91de21695be901042e96150d83e008bd1dc54d31451eb944df53ef05e54e45dd	[Content_Types].xml
  447	sha256:3cc01f857b0a3040f32c96eb45310b27764a2c5072568ebbdd774805c3e208b6	_rels/.rels
  563	sha256:c53b80dd4828f254dfb26c4ff24cf42248a1699687cb1c8e4b1f1bd244910fb2	docProps/core.xml
  286	sha256:31b458343a6a7fccc15f14453ebfec632c964079a7a3aa6cae85aa2886cb22ba	word/_rels/document.xml.rels
  192819	sha256:b5df120c523adbc0c9962345ba6d83037133a8bc77a0caab65a091fe5fbc510a	word/document.xml
  2635	sha256:c466ed74dd34155617db34b764772f0b4382ce57748b7f9ddf52e2e4fff0b5a4	word/styles.xml
101874	sha256:60d6f71b6f04d402d06ccab3cf080fb67ad4ed3bfc3418432477dab358d4585b	acr.json
```

`cmp judged/acr.json report/acr.json`, exit 0, no output.

`validate … --json`, and `render … --out judged-json --json` followed by
`cmp judged-json/acr.json report/acr.json`, each exit 0. The two reports
open:

```json
{"format":"metais.validate/1","ok":true,"findings":[],"summary":{"errors":0,"warnings":0,"info":0,"listed":0,"omitted":0,"by_code":{}},…}
{"format":"metais.render/1","ok":true,"findings":[],"summary":{"errors":0,"warnings":0,"info":0,"listed":0,"omitted":0,"by_code":{}},…}
```

and the render report gives the same size and digest for each file and
part as the human render above. No command reported a warning. Every copy
of `acr.docx` and of `acr.json` is byte-identical to the harness's own.

`judged/acr.docx`'s `word/document.xml` holds three tables, of 87, 105 and
149 rows: a header row each, then one row for each of the 86, 104 and 148
criteria of `wcag.csv`, `section508.csv` and `en301549.csv`.

### The word processor

Ed opened the judged `acr.docx`, SHA-256
`bd8dc31fa8ef27d73594fa783e7d859eaa9f826555b01b46b39a3160aa9c1f90`, on
2026-10-23 in **Pages 13.0 on macOS 26.5.2**, because neither Microsoft
Word nor LibreOffice is installed on the laptop. It opened with no repair
prompt and showed the three tables with all the data. **The check passes,
in Pages rather than Word.**

During the first viewing, Pages crashed once while Ed dragged the
scrollbar: `SIGABRT` on the main thread inside AppKit's scroll handling,
in `NSViewHierarchyInvalidateVisibleRect` under `-[NSClipView
scrollToPoint:]`, called from `-[NSScroller sendAction:]`. No frame of the
crash touches document import or layout. Reopened, the document looked
and scrolled correctly and all the data was present. This is recorded as a
fault of the viewer, in AppKit's scroller, and not as a finding against
the document.

### The round's focus: this page's pack command

The Work Order's focus named this page's pack command, which exited 127
as W7-E1 first wrote it (W7-NB5). At the build it names the built binary's
path, and it ran as written, exit 0, with the output under **The build**.

### Isolation

The method of "Isolation: the checks the runner makes" in
[ux-testing.md](../ux-testing.md), applied by hand as SC9's round applied
it, since `internal/tools/ux` refuses to start without
`ANTHROPIC_BASE_URL` and this task is none of its scenarios. The session
started as:

```text
env -i PATH=<system PATH> HOME=<the runner's HOME> USER LOGNAME SHELL=/bin/bash LANG=en_US.UTF-8 \
  TMPDIR=<session tmp>/ CLAUDE_CODE_TMPDIR=<session tmp> CLAUDE_CODE_DISABLE_AUTO_MEMORY=1 \
  NODE_PATH=<staged Playwright>/node_modules PLAYWRIGHT_BROWSERS_PATH=<staged browsers> \
  CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1 DISABLE_AUTOUPDATER=1 \
  claude -p "<the task>" --model opus --output-format stream-json --verbose \
    --no-session-persistence --strict-mcp-config --setting-sources project \
    --safe-mode --disable-slash-commands --settings <settings.json> \
    --permission-mode dontAsk --tools "Bash,Read,Write,Edit" \
    --disallowedTools "WebFetch,WebSearch,Agent,Task" --max-turns 800
```

Its settings were generated from SC9's round's. They turned auto-memory
off; allowed Read, Edit and Write only in the harness directory; denied
WebFetch, WebSearch, every read under the users' directory, and the
clones' and the runner's directories. The Bash sandbox was on, with
`failIfUnavailable` and no unsandboxed commands. Its network allow list
was strict and held one entry, the product's `localhost:41080`. It denied
reads of the users' directory, the volumes, the per-user temporary tree
and every entry then in the system's temporary directory. It allowed reads
of the harness directory, the session's temporary directory and the staged
Playwright and Chromium, read only, and writes, beyond the harness
directory it ran in, only in the session's temporary directory.

The product was reached through Playwright 1.64.1 and Chromium's headless
shell 156, staged outside the harness directory. Under the sandbox's
profile Chromium cannot start its multi-process form, and the sandbox
refuses a direct connection to loopback. So the `sc1-browser` module, 15
lines, launches Chromium single-process through the sandbox's own proxy,
where the allow list admits `localhost:41080` and nothing else. `curl
--noproxy ''` goes through that proxy for the same reason.

Five probe sessions on the same harness version, before the round,
confirmed each refusal: the Read tool and Bash on the clones and the
runner's configuration, `example.com`, the daemon's management port,
`127.0.0.1:41080`, another testbed's port, writes outside the harness
directory, `ls ..`, and the temporary tree. They also confirmed that the
product loads through `sc1-browser` and `curl`.

| Check | Result |
|---|---|
| `outside-repository` | pass |
| `fresh-directory` | pass: exactly `metais` and the `.metais` file before the session, both unchanged after it; Claude Code made an empty `.claude/.cc-writes/` |
| `metais-first-on-path` | not applicable: as this page says, the harness runs `./metais`, which is not on `PATH` |
| `placeholder-credential` | **deviation**: the session used the runner's own login through the real `HOME`; no credential variable was set, as in SC9's round |
| `auto-memory-off` | pass |
| `project-settings-only` | pass, with the same deviation: `HOME` was the runner's, and `--safe-mode` kept out user instructions, skills, plugins, hooks, MCP servers and agents |
| `strict-mcp-config` | pass: the `init` event lists no MCP server |
| `no-session-persistence` | pass: the only file written under the real `HOME` was the harness's spill file of call 10's output |
| `session-as-started` | pass: the harness directory as `cwd`, harness 2.1.301, `claude-opus-5-5` |
| `no-memory-read` | pass |
| `writes-inside-run` | pass |
| Tools and network | Bash, Edit, Read and Write only; Bash's network reached `localhost:41080` alone |

**The deviation.** With the real `HOME`, the isolation rests on the flags,
the Bash sandbox and the Read rules, not on an empty `HOME`. Two paths
outside the harness directory appear in the transcript. Call 11 grepped
the harness's own spill file of call 10's output (the product's
stylesheets), under the real `HOME`, and the sandbox refused it, as in
SC9's round; the harness fetched the stylesheets again into its own
directory. Its Playwright scripts loaded the staged `node_modules`, which
the settings allowed, read only. Nothing of the repository was read, and
nothing reached the network but the product. Left open and, by the
transcript, unused: Bash could list names, not contents, in the system's
temporary directory; entries made there after the deny list was taken;
and the system paths.

### Every point where the package did not tell the harness what to do

From the harness's own `NOTES.md` and the transcript. The harness left no
part of the report out, and the tool's own output answered none of these.

1. **The vendor's name.** Reading `inputs/report.md` and `metais schema …
   report --fields`, it needed `vendor.name`, which is required. The task
   gave none and the product shows none. The only guidance, to ask the
   person for a missing fact, is in `inspect`'s Examples section, and
   under `claude -p` it could not ask. It wrote `Not stated to the
   evaluator`, explained it in `notes`, and asked for the name in its last
   words. This page's task text does not supply the vendor either.
2. **Level AAA criteria.** `inputs/tables.md` and `inputs/conformance.md`
   write the AAA rule for the vendor: the table holds the AAA criteria
   "the vendor chose to evaluate". The harness is not the vendor. It
   listed all 86 criteria and evaluated every AAA criterion it had
   evidence for; 1.3.6, 1.4.8 and 2.2.5 are Not Evaluated.
3. **Methods with no assistive technology.** `inputs/report.md` asks the
   methods to name the assistive technology used, and says nothing of an
   evaluation with none. It said plainly that this was a scripted
   evaluation by an AI agent, with no screen reader, magnifier or speech
   input.
4. **Section 508 502.3.x and EN 301 549 11.5.2.x for a web application.**
   The package says WCAG answers E207.2 and clauses 9 to 11.4, but not how
   to treat these rows for a web application. It evaluated them through
   the browser's accessibility tree and marked the platform-only rows Not
   Applicable.
5. **Documentation and support (602, 603, 12.x).** `inputs/conformance.md`
   discourages Not Evaluated for a required criterion, but a row out of
   the evaluation's scope has no other level. It used Not Evaluated, with
   the reason in the remarks.
6. **The report's date.** `inputs/report.md` says "the date the report was
   completed", not by which clock: local 2026-10-22, the server's
   2026-10-23 UTC. It used the local date.
7. **No evaluation tool named.** The package says to evaluate but names no
   tool, and none was installed. It wrote its own checks. This is the
   environment's, not the package's.
8. **The test set-up's limits** (not the package's). The single-process
   browser holds one context per launch, so each check launched its own.
   The product's Content Security Policy blocks injected styles, so the
   zoom and text-spacing checks used Playwright's `bypassCSP`, standing in
   for a user style sheet.

The harness's side effects on the product, from its notes: three
wrong-password attempts, three sign-ins each signed out by the end, and
one "Save profile" with nothing changed.

### Findings

None blocks the round. Each goes to the plan orchestrator by "What happens
next", as a fix item or a ruling that the package is enough.

| Finding | Evidence | Blocking |
|---|---|---|
| **Package text, the report details:** nothing in `inputs/report.md` says what to do with a required fact the evaluator was not given, such as `vendor.name`, when it cannot ask; the only guidance is in `inspect`'s Examples | point 1; `report.yaml`'s vendor | no |
| **Runbook:** this page's task slot does not supply the vendor | point 1 | no |
| **Package text, `inputs/tables.md` and `inputs/conformance.md`:** the AAA rule is written for the vendor, not for a third-party evaluator | point 2 | no |
| **Package text, `inputs/report.md`:** no guidance for methods with no assistive technology, or for which clock the report's date uses | points 3 and 6 | no |
| **Package text, the references:** they do not say how to treat Section 508 502.3.x and EN 301 549 11.5.2.x for a web application | point 4 | no |
| **Package text, Not Evaluated:** an out-of-scope documentation or support row (602, 603, 12.x) has no level but Not Evaluated, which `inputs/conformance.md` discourages for a required criterion | point 5 | no |
| **The golden path held:** `inspect` first, every read pointed to, no raw read of the `.metais` file, and `validate` and `render` clean at the first attempt | **What the harness read**; calls 54 and 55 | no |
| **Method:** under Claude Code's Bash sandbox, a round against a web product needs a single-process browser launched through the sandbox's proxy, and `curl --noproxy ''` | **Isolation**; point 8 | no |
| **Viewer:** Pages 13.0 crashed once in AppKit's scroller while the judged `acr.docx` was scrolled, and not on reopening | **The word processor** | no: no frame touches the document's import or layout |

### Deviations

1. The build is `339864912a01`, not `dd179daa`, by Ed's ruling.
2. Claude Code 2.1.301 rather than the host's packaged 2.1.236, as in
   SC9's round, whose Deviations say why.
3. `HOME` was the runner's own, with its login, rather than an empty one
   with a placeholder credential, by Ed's ruling, as in SC9's round; see
   **The deviation** under Isolation.
4. An earlier route with a placeholder token was dropped by Ed's ruling
   before it ran a session.
5. The task's product slot names `sc1-browser` and `curl --noproxy ''`,
   which the sandbox needs; the rest of the task text is this page's,
   verbatim.
6. The judging `--json` render wrote to a second new directory,
   `judged-json`; every copy is byte-identical.
7. The word processor was Pages, not Microsoft Word, which was not
   available; see **The word processor**.
8. The test account's password was in the task text and so is in the
   transcript. The testbed has been reset since, and the account no longer
   exists.

### What the round cannot show

- **Whether the harness evaluated the product correctly.** As this page
  says, the round records the levels and remarks and does not grade them.
- **Whether the package alone is enough for a person.** This is one
  harness run by one model, not a person's.
- **What Microsoft Word shows.** The document was opened in Pages only.
- **What an empty `HOME` would have guaranteed.** With the runner's own
  `HOME`, a read the settings did not foresee was refused only where a rule
  covered it. The transcript shows no such read.

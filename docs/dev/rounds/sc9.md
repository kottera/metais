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

This is the harness's evidence for the round, recorded for whoever took it
(PLAN-2026-10-metais-ux §8). It is not the round. The round is Work Order
`01a15abf-6347` of PLAN-2026-10-metais-v1, and its result is recorded below, in
"The round on 339864912a01 (PLAN-2026-10-metais-v1 W7-A3)".

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
| Commit | `29f077f3edbadb5f99ec3998f66803e99103a0f3`, on branch `feat/metais-ux-w3-5-sc1-sc9-evidence`. Its binary closure and its `spec/` are `main`'s at `63315621`: the branch changes only `internal/tools/ux` and documents. It precedes W2-NB1 (#217), which changed three of the binary's sentences, among them the name of `inspect`'s next-step section |
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

## §B.6 corrected after the evidence run (PLAN-2026-10-metais-ux W3-11)

The evidence run above was made against the old text of
`spec/B-worked-package.md` §B.6. It said that check "reports every finding
of every step in one run", with no word on a package whose read is refused,
and opus's first `check` showed that sentence misleading an author. The plan
orchestrator ruled that MTS-AUT-003 governs, and W3-11 corrected the text to
agree with it. No normative statement, behaviour or vector changed:

- **§B.6** now says that a package whose read is refused, such as a source
  holding an empty directory, gets only the findings of that refusal, so a
  check that reports only a refusal has not yet looked at the manifest, the
  schemas or the templates.
- **§13.1's opening** says the same, citing MTS-AUT-003.
- **`metais help check`'s summary** and **`metais help manifest`** say it
  too.

Whoever takes the round gives the author the `spec/` of a commit that holds
W3-11, and reads §B.6 there rather than the text the evidence run saw. The
evidence above stands for what it shows of the binary; its point on B.6 is
answered by this correction. The round itself is Work Order
`01a15abf-6347`, and its result is the next section.

## The round on 339864912a01 (PLAN-2026-10-metais-v1 W7-A3)

This is the round, Work Order `01a15abf-6347`, run on 2026-10-22 by this
page as written, except where **Deviations** says otherwise. Ed ruled on
2026-10-22 that it runs on current `main` rather than on `dd179daa`, the
build the Work Order first named. `main` was then 75 commits ahead of
`dd179daa`, the whole of PLAN-2026-10-metais-ux included, and so W3-11's
correction of §B.6.

**Verdict: pass.** `check`, `test` and `pack` each exit 0 and report no
finding of any severity, in human form and with `--json`. The package has
every part the task asks for and is not the worked package with its names
changed. The transcript shows no reading of anything the author was not
given.

### The build

| | |
|---|---|
| Commit | `339864912a0150b351a66bdcb4d6798506d41134`, `main` at 2026-10-22 15:18:39 -0400 ("docs(metais-public): [W1-3] the community files (#244)") |
| Build | `GOTOOLCHAIN=auto CGO_ENABLED=0 go build -trimpath -o <author's directory>/metais ./cmd/metais`, from a fresh clone at the commit. `GOTOOLCHAIN=auto` fetched `go.mod`'s `toolchain go1.26.9`, which built the binary |
| metais | `metais 0.0.0-20261022191839-339864912a01`, `specification 1.0`, `METAIS is made by Kottera.`; SHA-256 `721fbc1d59c7064c5429dc49f70ac1a776b8716e53671ef98025448eb6bf8b30`, the same after the session |
| Author's directory | a new absolute path holding `git archive --format=tar <commit> spec \| tar -x` and the binary, and nothing else. No `.git`, `.hg`, `.jj` or `.svn` in it or above it |
| Host | a developer laptop, Apple M2 Max, macOS 26.5.2 (25F84), Darwin 25.5.0, darwin/arm64; host Go go1.26.1 |
| Harness | Claude Code 2.1.301, `claude -p` with `--model opus`, which resolved to `claude-opus-5-5`. The task text was this page's, verbatim. The session ran 45 turns and 44 tool calls in 247 s |
| Evidence kept | outside the repository, with the round's report on the host that ran it: the author's source directory as `package-src.tar.gz` (SHA-256 `1c3fb8553f18d715ade87eaa10afde324ecff4c381aaf79e3e409fe8c2956399`), the packed package, the full `stream-json` transcript, the author's own record and list of gaps, the judging logs, the runner's settings and the isolation probes. None is committed: they hold host paths and a user name, and "Scrubbing a set" in [ux-testing.md](../ux-testing.md) scrubs a set `internal/tools/ux` wrote, which a round run by hand does not have |

### The package written

`recipe-card`: a recipe card in Markdown and a plain-text shopping list,
both rendered from one recipe.

- **Manifest.** `metais.yaml`: `metais: "1.0"`, the name `recipe-card`,
  version `1.0.0`, a title, a description and a guide.
- **Input.** `recipe`, with its schema `inputs/recipe.schema.json` (JSON
  Schema 2020-12: a name, servings, preparation and cooking minutes, an
  optional note, ingredients of quantity, unit and item under a
  `dependentRequired` rule, and steps) and its meaning document
  `inputs/recipe.md`.
- **Templates.** `templates/card.md.liquid` (output `card`) and
  `templates/shopping-list.txt.liquid` (output `shopping_list`).
- **Example.** `examples/flatbread/`: `example.yaml`, `inputs/recipe.yaml`,
  `expected/card.md` and `expected/shopping-list.txt`, written by hand,
  not by `test --update`.
- **Not the worked package renamed.** Its subject, schema, templates and
  example are its own; it shares only the layout every package has.
- **Packed:** `recipe-card-1.0.0.metais`, 7714 bytes, 14 entries, SHA-256
  `728efbe7f2ab0f9a58af6c47c10a658ea902925ffac82742ca68d46ce2ab7f25`,
  package digest
  `sha256:7879ae0d5f763543449647ca39a857bc42a04705ea6c80055892bca6a0db6afd`.
  The author's own pack and both judging packs are byte-identical. The
  entries are `README.md` first and stored, then `.compiled/digests.json`,
  `manifest.json`, `references.json` and `schema.json`, then the nine
  source files in name order.

### The judging run

The commands of "The commands that judge the result", on a fresh copy of
the author's source directory, with the binary the author was given. Every
standard error not shown below was empty.

`metais check package`, exit 0:

```text
check: 0 errors, 0 warnings, 0 notes

Next:
  metais test <package>  — Renders every example and compares it with the outputs it should produce.
```

`metais test package`, exit 0:

```text
0 errors, 0 warnings, 0 info
flatbread: pass

Next:
  metais pack <package>  — Packs the source directory, which checks and tests it again first.
```

`mkdir packed` and `metais pack package --out packed`, exit 0. Standard
output:

```text
Packed recipe-card-1.0.0.metais: 7714 bytes, 14 entries, package digest sha256:7879ae0d5f763543449647ca39a857bc42a04705ea6c80055892bca6a0db6afd
```

Standard error:

```text

Next:
  metais inspect <package>  — Reads the packed package as a supplier first reads it.
```

`check package --json`, `test package --json` and
`pack package --out packed-json --json`, each exit 0:

```json
{"format":"metais.check/1","ok":true,"findings":[],"summary":{"errors":0,"warnings":0,"info":0,"listed":0,"omitted":0,"by_code":{}},"result":{"counts":{"error":0,"warning":0,"info":0}},"next":[{"run":"metais test <package>","why":"Renders every example and compares it with the outputs it should produce."}]}
{"format":"metais.test/1","ok":true,"findings":[],"summary":{"errors":0,"warnings":0,"info":0,"listed":0,"omitted":0,"by_code":{}},"result":{"examples":[{"name":"flatbread","status":"pass","differences":[]}],"update":null},"next":[{"run":"metais pack <package>","why":"Packs the source directory, which checks and tests it again first."}]}
{"format":"metais.pack/1","ok":true,"findings":[],"summary":{"errors":0,"warnings":0,"info":0,"listed":0,"omitted":0,"by_code":{}},"result":{"dry_run":false,"file":"recipe-card-1.0.0.metais","size":7714,"digest":"sha256:7879ae0d5f763543449647ca39a857bc42a04705ea6c80055892bca6a0db6afd","entries":14},"next":[{"run":"metais inspect <package>","why":"Reads the packed package as a supplier first reads it."}]}
```

No command reported a warning, so none needs the author's reason.

### The author's record

The transcript holds every tool call with its input and output, and the
author kept its own record of 21 shell commands with their output and exit
status. In order, the session listed its directory; read `spec/`'s
README, changelog, errata and file list; read chapters 1 and 13; ran
`metais help`, `--version` and `help` for `check`, `test` and `pack`; ran
`help` for `manifest`, `templates` and `inputs`; read chapter 5, parts of
chapters 2, 3 and 7, and appendix B; wrote the package; then ran
`./metais check recipe-card`, `test recipe-card`, `test recipe-card
--json`, a `render` of the example's inputs compared with `expected/` by
`diff -r` (identical), and `pack recipe-card --out dist`. Every one exited
0. No `metais` command failed at any point, and the last `check`, `test`
and `pack` ran after the last edit to the source.

### What the author read, in order

1. `ls -la spec`.
2. `spec/README.md`, `spec/CHANGELOG.md`, `spec/ERRATA.md`, and a list of
   every file under `spec/`.
3. `find spec -maxdepth 3`.
4. `spec/01-introduction.md`.
5. `spec/13-authoring.md`.
6. `metais help`, `metais --version`, and `metais help` for `check`,
   `test` and `pack`.
7. `metais help` for `manifest`, `templates` and `inputs`.
8. `spec/05-data-specification.md`.
9. The headings of chapters 2, 6, 7 and 9 and of appendix B.
10. `spec/02-container.md` §2.4, §2.4.1 and §2.5.
11. `spec/B-worked-package.md`.
12. The headings of chapter 3.
13. `spec/03-manifest.md` §3.4.
14. `spec/07-template-environment.md`, lines 116 to 275 (§7.3 to §7.7.1).

This page says the author starts at `spec/B-worked-package.md`, but the
task text it gives the author does not say so, and this author read
appendix B eleventh.

### Isolation

The method of "Isolation: the checks the runner makes" in
[ux-testing.md](../ux-testing.md), applied by hand rather than by
`internal/tools/ux`, which refuses to start without `ANTHROPIC_BASE_URL`,
and whose A4 prompt is not this page's task text. The session started as:

```text
env -i PATH=<system PATH> HOME=<the runner's HOME> USER LOGNAME SHELL=/bin/bash LANG=en_US.UTF-8 \
  TMPDIR=<session tmp>/ CLAUDE_CODE_TMPDIR=<session tmp> CLAUDE_CODE_DISABLE_AUTO_MEMORY=1 \
  CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1 DISABLE_AUTOUPDATER=1 \
  claude -p "<the task>" --model opus --output-format stream-json --verbose \
    --no-session-persistence --strict-mcp-config --setting-sources project \
    --safe-mode --disable-slash-commands --settings <settings.json> \
    --permission-mode dontAsk --tools "Bash,Read,Write,Edit" \
    --disallowedTools "WebFetch,WebSearch,Agent,Task" --max-turns 400
```

Its settings turned auto-memory off; allowed Read, Edit and Write only in
the author's directory; denied WebFetch, WebSearch, every read under the
users' directory, and the clone's and the runner's directories. The Bash
sandbox was on, with `failIfUnavailable`, no unsandboxed commands and no
network (an empty, strict domain allow list). It denied reads of the users'
directory, the volumes, the per-user temporary tree and every entry then
in the system's temporary directory, and allowed reads and writes only in
the author's directory and the session's temporary directory. A probe
session on the same harness version confirmed each refusal and each
allowed action before the round.

| Check | Result |
|---|---|
| `outside-repository` | pass |
| `fresh-directory` | pass: exactly `spec/` and `metais` before the session, both unchanged after it; Claude Code made an empty `.claude/.cc-writes/` |
| `metais-first-on-path` | not applicable: as this page says, the author runs `./metais`, which is not on `PATH` |
| `placeholder-credential` | **deviation**: the session used the runner's own login through the real `HOME`; no credential variable was set |
| `auto-memory-off` | pass |
| `project-settings-only` | pass, with the same deviation: `HOME` was the runner's, and `--safe-mode` kept out user instructions, skills, plugins, hooks, MCP servers and agents |
| `strict-mcp-config` | pass: the `init` event lists no MCP server |
| `no-session-persistence` | pass |
| `session-as-started` | pass: the author's directory as `cwd`, harness 2.1.301 |
| `skill-as-declared` | pass: no skill; two built-in plugins safe mode keeps, never used |
| `no-memory-read` | pass |
| `writes-inside-run` | pass |
| Tools and network | Bash, Edit, Read and Write only; the sandbox refused Bash's network |

**The deviation.** With the real `HOME`, the isolation rests on the flags,
the Bash sandbox and the Read rules, not on an empty `HOME`. One read
outside the author's directory was attempted and refused: tool call 8, the
harness's own spill file of call 7's output (the help pages), under the
real `HOME`. The author read the same output back from its own record. No
other tool call names a path outside the author's directory, nothing of
the repository was read, and nothing reached the network. Left open and,
by the transcript, unused: Bash could list names, not contents, in the
system's temporary directory; entries made there after the deny list was
taken; and the system paths `/usr`, `/opt/homebrew` and `/etc`.

### Did §B.9's description of the packed package mislead the author?

**No.** The Work Order's focus asked whether §B.9, which says the packed
package begins with a stored `README.md`, misleads an author. This author
read §2.4, MTS-PKG-021 and §2.4.1 before appendix B, wrote no `README.md`,
and never raised it. The packed file does begin with `README.md`, stored.
The runner saw one thing §B.9 does not say: `metais read <package>
README.md` refuses with `usage.entry_not_found`, and `metais ls` does not
list it. Chapter 2 says so; §B.9 does not.

### Every point where the text did not tell the author what to do

From the author's own list and the transcript. The author gave up no part
of the package.

1. **Start from `init`, or write every file?** Reading `metais help`, whose
   authoring golden path is `init → edit → check → test → pack`, and §B.2's
   Note, which writes every file by hand. The author followed appendix B
   and never ran `init`.
2. **Is a hyphen allowed in a template's file name?** Reading `metais help
   manifest` and §3.4, which give the grammar of an output `id` (no hyphen)
   but not of a template path. §2.3 answers it, unread by the author. The
   author used `shopping-list.txt.liquid` and let `check` judge it, which
   accepted it: the binary answered.
3. **Is `dependentRequired` allowed?** Reading MTS-DAT-004, which allows the
   keywords of the JSON Schema 2020-12 vocabularies it names without
   listing them, and those documents are not in `spec/`. The author relied
   on its own knowledge of 2020-12; `check` accepted it.
4. **Does `forloop.index` exist, and does it count from 1?** Reading §7.3
   to §7.7.1. Chapter 7 lists `forloop`'s members in its tag table (§7.2,
   above the lines read), and no sentence says `index` counts from 1. The
   author used standard Liquid's meaning; the render numbered the steps 1
   to 6.
5. **Did `test` compare both expected files?** Reading `metais help test`
   and `test --json`'s report, neither of which names the files compared.
   The author rendered the example itself and compared it by `diff -r`.
6. **Saved output out of reach (the harness, not the text).** Claude Code
   spilled long command output to a file under the real `HOME`, which the
   isolation refuses. The author read it back from its own record.
7. **Two words for one severity.** `check` prints `0 notes` where `test`
   prints `0 info`, and the specification calls the severity `info`.

### Findings

None blocks the round. Each goes to the plan orchestrator by "What happens
next", as a fix item or a ruling that the text is enough.

| Finding | Evidence | Blocking |
|---|---|---|
| **Text:** chapter 7 lists `forloop`'s members only in §7.2's tag table, and does not say that `index` counts from 1 | point 4; `spec/07-template-environment.md` §7.2's `for` row | no |
| **Text:** MTS-DAT-004 leaves the list of schema keywords to the external JSON Schema 2020-12 vocabularies | point 3; the author's gap 3 | no |
| **Text and help:** `metais help`'s authoring golden path starts at `init`, while appendix B writes every file by hand | point 1; §B.2's Note explains why, but the two give different first steps | no |
| **Text:** §B.9 says the packed file begins with `README.md`, but not that `metais read` and `metais ls` do not reach it | the runner's `read` refusal, `usage.entry_not_found`; chapter 2 says it | no |
| **Tool:** `test`'s report, human and `--json`, does not name the expected files it compared | point 5; `"differences":[]` above | no |
| **Tool:** `check` says `notes` where `test` says `info` for the same severity | point 7; the judging output above | no |
| **Tool:** `pack` prints its human `Next:` block on standard error, `check` and `test` on standard output | the judging output above. ADR-008 D11 item 3 decides it so, so this is as designed and needs no fix | no |
| **Runbook:** this page says the author starts at appendix B, but the task text it gives does not say so | "What the author read" above | no |
| **Method:** `internal/tools/ux` cannot run on a host without a loopback proxy; and Claude Code 2.1.236 resolves `opus` to `claude-opus-5`, which refused this task text at its first turn | Deviations 2 to 4 | no |

### Deviations

1. The build is `339864912a01`, not `dd179daa`, by Ed's ruling.
2. Two attempts on Claude Code 2.1.236 never started: it resolves `opus`
   to `claude-opus-5`, which refused the task text at turn 1 with a
   `[reasoning_extraction]` safeguard.
3. A third attempt, naming `claude-opus-5-5`, never started: 2.1.236
   refuses that model and asks for 2.1.280 or newer.
4. The round ran on Claude Code 2.1.301, already on the host, where `opus`
   resolves to `claude-opus-5-5`. The probe was rerun on 2.1.301 first.
   The task text was the same in every attempt.
5. `HOME` was the runner's own, with its login, rather than an empty one
   with a placeholder credential; see **The deviation** under Isolation.

### What the round cannot show

- **Whether the text alone is enough for a person.** This is one harness
  run by one model, not a person's.
- **What an empty `HOME` would have guaranteed.** With the runner's own
  `HOME`, a read the settings did not foresee was refused only where a rule
  covered it. The transcript shows no such read.

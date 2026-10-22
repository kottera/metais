# PLAN-2026-10: METAIS UX — make the golden paths findable from the binary, and measure it with fresh agent sessions

**Author:** drafted in the 2026-10-20 design session with Ed Funnekotter
**Date:** 2026-10-20
**Feature:** `metais-ux`. Every brief, Work Order and pull request of this
train inherits it verbatim: `metais-ux [W2-C3]: <action>` for a Work Order,
and `<type>(metais-ux): [W2-C3] <change>` for a pull request and its squash
subject, by `kottera/kottera`'s `docs/dev/work-order-and-pr-naming.md`, which
AGENTS.md adopts.
**Status:** Complete 2026-10-22: every wave complete, R-UX-5 holding under R-W3-5, and the train closeout in §10. Ratified 2026-10-21 by Ed Funnekotter, with the rulings of §4; R-UX-8 amended and R-UX-9 added the same day.
**For PRD:** [PRD-001](../prd/PRD-001-metais-v1.md) R17 (the golden paths),
and its success criteria SC1 (an AI harness, given only the binary and a
package, produces a valid document) and SC9 (an author working only from the
specification writes a package `check`, `test` and `pack` accept).
**For ADR:** [ADR-008](../adr/ADR-008-the-command-line-and-its-machine-output.md)
(the command line and help), [ADR-007](../adr/ADR-007-authoring.md)
(`init`, `sample`, `render --sample`), [ADR-001](../adr/ADR-001-the-container-and-its-layout.md)
Amendment A2 (the README names `metais inspect`), and
[ADR-011](../adr/ADR-011-the-specification-document-and-conformance.md) (the
vectors).
**Repository:** `kottera/metais`, branch `main`. The train writes this
repository only.
**Depends on:** nothing. PLAN-2026-10-metais-v1's waves are all complete,
and nothing they left in flight touches this train's seams. The dependency
runs the other way: **the v1.0 release waits for this train's Wave 2**
(§4, R-UX-1).

## 1. Purpose

METAIS is used mostly by agents, so a golden path that an agent cannot find
from the binary is no golden path at all. On 2026-10-20 Ed ran fresh
`claude -p` sessions, each in an empty directory with nothing but a packed
package and `metais` on PATH. Each session had no memory, no user settings and
no MCP. The prompts were "fill this file: report.metais", with fictional or
new fictional data asked for in some runs, on Opus, Sonnet and Haiku. He then
had a session rewrite a real tool, `pr-report`, onto a package.

What the runs found:

- **Reaching `inspect`.** Before the README named `inspect` (ADR-001 A2), a
  session unzipped the package to find its way. After it, every model ran
  `metais inspect` before reading the package's contents.
- **Field lists.** `metais schema` prints about 39 KB of nested JSON for one
  input of the DORA package, too much for an agent to read at once.
  - Three of the four models worked around it. One unzipped the package and
    parsed the raw schema file.
  - Every extraction script had to discover that the input's fields sit under
    `result.schema.$defs["inputs/<name>.schema.json"]`.
- **Generating data.** No model found `render --sample` or `sample`, which
  generate data on the fly.
  - `inspect`'s next steps do not name them.
  - `metais help` files `sample` under authoring.
  - `render --help` calls `--sample` "the author's preview".
  - Asked for fictional data, one model re-rendered the package's shipped
    example, and the rest wrote every value by hand.
- **Guessed commands.** One model guessed `metais cat`.
- **The template language.** Authoring a real package, a session found the
  language, the manifest's fields and the escaping rules only in `spec/` and
  in the repository's example packages. `metais help` has no topic for any of
  them, and metais.org is not yet live.
  - The session scored "learning the template language from the tool" 3/10,
    and "help readability" 5/10. The help is complete and precise, but reads
    as specification, with examples last.
  - It scored the primitives themselves 9/10 (`escape: html`, `json_script`,
    deterministic renders), and the authoring loop 8/10.

**Done means:**

- A session with only the binary learns everything it needs from the
  binary: the template language, the manifest, and how to get an input's
  fields compactly. That covers the language's tags, filters, nil and
  escaping, and the context-safe filters.
- `inspect` and the help put on-the-fly data (`render --sample`, `sample`)
  in front of a supplier, not only an author. Every command's help leads
  with examples.
- `metais` installs its own agent skill for the harness in use
  (ADR-008 A2 and ADR-009 A5, drafted in Work Order `01a15ad7`).
- The byte-level conformance vectors are generated, so a change to the
  package README is an edit and `regenerate.sh` (Work Order `01a15ad6`).
- **A UX harness exists and its scorecard is green.** It runs scripted
  `claude -p` sessions per scenario and model, isolated from the host's
  memory and settings, and scores each transcript. The scorecard after
  Wave 2 meets §4's thresholds on every model the thresholds name, and its
  comparison with Wave 1's baseline is recorded.
- The SC1 and SC9 rounds of PLAN-2026-10-metais-v1 §6 (Work Orders
  `01a15abf-4d58` and `01a15abf-6347`) run on the post-Wave-2 binary, with
  the harness's transcripts as their evidence.

## 2. Execution model

PLAN-2026-10-metais-v1 §2 applies whole:
- waves run one at a time, and the items inside a wave run in parallel;
- each wave's brief is written late by the plan orchestrator, as
  `docs/briefs/brief-plan-2026-10-metais-ux-waveN.md`;
- the specification is written with the code;
- shared seams and statement numbers are allotted at breakdown.

Its harness policy also applies whole: Claude throughout, and the dev
pipeline's types by full key.

Two rules this train adds:

- **ADR amendment numbers are allotted at breakdown**, like statement
  numbers. Several Wave 2 items amend ADR-008. The wave orchestrator gives
  each item its amendment number, so two items never draft the same
  `A<n>`. The help goldens are a shared seam too: items that change the same
  command's help run in sequence.
- **This train runs beside [PLAN-2026-10-metais-test-economy](PLAN-2026-10-metais-test-economy.md),
  and shares two seams with it.**
  - That plan's W2-3 adds a per-vector record, with a step in
    `regenerate.sh` and a line in `.gitattributes`. Work Order `01a15ad6`
    (generated vectors) edits the same three files.
  - Its W1-4 reworks `internal/cli`'s refusal walks. That is the package
    where Wave 2's help changes and their goldens land.

  Before breakdown, each wave orchestrator reads that plan's §3 status. An
  item that would edit a file an in-flight test-economy item edits gets a
  `depends_on` edge to it, and merges after it.
- **A UX run is evidence, not a gate.** The harness costs model tokens and
  takes minutes per scenario, so it never runs in `gate.sh`. Its scorer does
  run in the gate, against recorded transcripts committed as fixtures. A
  scorecard is published like test evidence: the commit, the model IDs, the
  host, and where the transcripts are.

## 3. Overview and sequencing

| Wave | Objective | Depends on | Status |
|---|---|---|---|
| 0 | This plan, merged to `main` | — | **complete** (#183) |
| 1 | **The harness and the baseline**: `docs/dev/ux-testing.md`; the scenario set; the runner, with its isolation proof; the transcript scorer with its fixtures; a baseline scorecard on `main` | 0 | **complete** (see §5, Wave 1 verdict) |
| 2 | **The fixes**: the `templates` and `manifest` help topics; examples first in every command's help; a compact field listing from `schema`; on-the-fly data on `inspect`'s next steps and in the supplier's help; a minimal `init`; the agent skill (`01a15ad7`); generated vectors (`01a15ad6`) | 1 | **complete** (see §7, Wave 2 verdict) |
| 3 | **Measure, fix, and close**: the scorecard on post-Wave-2 `main`, with and without the skill installed; its comparison with the baseline; a fix item for each failing scenario, then a re-run until §4's thresholds hold; the SC1 and SC9 rounds | 2 | **complete** (see §8, Wave 3 verdict) |

Wave 1 comes before the fixes so that the baseline measures today's binary.
Wave 2's items all depend on Wave 1's merge, not on each other. Its two items
already in flight (`01a15ad6`, `01a15ad7`) join the wave by `depends_on`
edges from its verification, not as children.

## 4. Rulings

Made in the 2026-10-20 session:

- **R-UX-1.** The v1.0 release (PLAN-2026-10-metais-v1 §6, Work Order
  `01a15ac4-4d4e`, T-A3) waits for this train's Wave 2. When the plan
  orchestrator files Wave 2's orchestrator, it adds that Work Order to
  T-A3's `depends_on`. If it lacks the grant to edit T-A3, it parks and asks
  Ed to add the edge. The release does not wait for Wave 3's measurements.
- **R-UX-2.** A package's `examples/` stays optional. A supplier who asks for
  example or fictional data is served by on-the-fly generation
  (`render --sample`, `sample`), never by copying a shipped example's files
  out of the zip. A session asked for a real document never invents its
  facts.
- **R-UX-3.** The byte-level vectors' offsets and sizes are generated by
  `internal/vectorgen`, not written by hand (Work Order `01a15ad6`).
- **R-UX-4.** The package README names `metais inspect` (ADR-001 A2, merged
  in #177).
- **R-UX-5, the scorecard's thresholds (Q-UX-1).**
  - Every consuming scenario passes on Opus, Sonnet and Haiku, and every
    authoring scenario passes on Opus and Sonnet. Haiku's authoring results
    are recorded, not gated.
  - *A pass means:* the first `metais` command of a consuming scenario is
    `inspect`; no zip extraction; no unknown-command error; the package
    left untouched by a consuming scenario (R-UX-8); and the scenario's own
    outcome check holds. For a real document that check is
    asking rather than inventing; otherwise it is `validate` with 0 errors
    and a rendered output.
- **R-UX-6, harnesses beyond Claude Code (Q-UX-3).** The runner takes the
  harness as a parameter, and Claude Code is the only one this train gates
  on. A `.agents/skills` harness (Codex or OpenCode) is run once in Wave 3
  if one can be installed on a host, and its results recorded.

- **R-UX-7, where the runs happen (Q-UX-2).** The scorecards run from
  dispatches. Work Order `01a15b3b` proved on 2026-10-20 that a dispatch can
  run a nested, isolated `claude -p`.
  - A dispatch carries no Claude credential. Its `ANTHROPIC_BASE_URL` points
    at a loopback proxy that authenticates upstream. A nested `claude`
    refuses on the client side unless a credential variable is set, so the
    runner sets `ANTHROPIC_AUTH_TOKEN` to a non-secret placeholder.
  - Isolation is stronger than on a person's host. `$HOME` is the Work
    Order's own workspace, the sandbox denies paths outside its volumes, and
    the run showed no memory paths, no MCP servers and no writes to the
    repository.
  - Nested runs spend the dispatch's own seat quota. The probe's seat ran
    out of quota once, so the runner paces its runs and resumes a partial
    scorecard rather than restarting it.
  - If dispatches ever lose the proxy, the scorecards fall back to an
    off-tera act, a run by hand outside the dispatch sandbox (§6).
- **R-UX-8, a run asked for output leaves the package alone** (amended
  2026-10-21). A consuming scenario whose prompt asks for output, such as
  "render this file" or "produce the report from…", fails when the session:
  - extracts the package with `unzip`, `unpack` or `extract`;
  - changes or repacks the package file.

  The data must be supplied to the package, as a data directory or flags.

  The rule does not apply to a prompt that leaves the user's intent open.
  Told to "fill this file … with new fictional data", the probe's nested
  Haiku session unpacked the package, rewrote its shipped examples, ran
  `test --update` and repacked it. That is a coherent reading of an
  ambiguous request, and C0 (§5) scores it as one.
- **R-UX-9, rewording a scenario's prompt.** A prompt may be changed only to
  remove ambiguity about what the user wants, never to steer how the
  session gets there. Each change is recorded in the next scorecard, with
  the transcript that showed the ambiguity. A session that misreads a clear
  intent is a failure to fix in the tool, not in the prompt.

## 5. Wave 1 — the harness and the baseline

**Goal:** a repeatable, isolated way to watch a fresh agent use `metais`, and
a number for today's binary.

**What the brief carries:**

- **`docs/dev/ux-testing.md`**, the method. It covers:
  - the scenarios;
  - the isolation rules, stated as checks the runner makes, not as hopes:
    - a fresh directory holding only the scenario's fixtures, outside any
      repository;
    - `metais` from the commit under test on PATH;
    - auto-memory off (`CLAUDE_CODE_DISABLE_AUTO_MEMORY=1` and
      `autoMemoryEnabled: false`);
    - project settings only, `--strict-mcp-config`, and
      `--no-session-persistence`;
    - after each run, the transcript checked for any read of a memory file;
  - the models;
  - what is recorded per run: the transcript, the files written, the outcome
    checks, and the turns, time and cost;
  - the scorecard's format, and how a scenario is added.
- **The scenarios**, each a directory holding:
  - a prompt;
  - its fixtures (a package built from `examples/`, or none);
  - its outcome check.

  The starting set:

  | Scenario | Prompt (in substance) | Outcome check |
  |---|---|---|
  | C0 fill-ambiguous | "fill this file: report.metais", as people actually say it | either reading passes: a rendered document from data supplied to the package, or an edited package that `check` and `test` accept. The session must say which reading it took, and must invent no facts presented as real. A silent reading fails. |
  | C1 render-real | "render this file: report.metais" (no data given) | asks for the facts and invents none; at most a scaffold |
  | C2 render-new-fictional | "render this file with new fictional data" | valid data matching no shipped example, supplied to the package; rendered; the package untouched (R-UX-8) |
  | C3 preview | "show me what this package produces" | an output rendered from generated data; nothing extracted from the zip |
  | C4 what-it-needs | "what does report.metais need from me?" | names every input and its required fields, from `inspect`, `read` and `schema` |
  | C5 script | "write a script that renders this package nightly from a JSON file" | the script validates and renders through `metais`; a test run passes |
  | A1 new-package | "make a .metais that renders an invoice as HTML from line items" | `check` and `pack` pass; `render --sample` renders |
  | A2 edit-package | "add a due date to this package's invoice" | the schema and template change together; `test` passes |
  | A3 data-to-js | "a package whose HTML page charts numbers it writes into a JS array" (the pr-report shape) | `escape: html` with `json_script` for strings; renders; the page's data parses |
  | C6 render-acr-fictional | "render this file with new fictional data: accessibility-conformance-report.metais" (SC1's evidence, the §4.1 package) | valid data matching no shipped example, supplied to the package; rendered; the package untouched (R-UX-8). The evidence is judged by sc1.md's own bar |
  | A4 spec-only-package | SC9's task: write a package of your own choosing from `spec/` and the binary alone, with the commit's `spec/` as the only fixture (SC9's evidence) | the last `check`, `test` and `pack`, after the last edit, each pass. The evidence is judged by sc9.md's own bar |

- **The runner** lives outside the binary: never reachable from
  `cmd/metais`, held so by `internal/guarantees`, as ADR-009 A2 holds the
  release builder. It runs scenario × model, writes each run's record, and
  prints the scorecard. Under R-UX-7 it:
  - sets the placeholder `ANTHROPIC_AUTH_TOKEN`;
  - gives each nested session its own directory and `$TMPDIR`. The sandbox
    refuses `/tmp`, and a model that hard-codes `/tmp` fails silently, as
    the probe's session did;
  - paces its runs and resumes a partial scorecard after a quota pause;
  - builds `metais` with `-buildvcs=false`, because `git` in the sandbox
    fails on an `iconv` locale error.

  The probe's transcripts (Work Order `01a15b3b`'s artifacts) are among the
  scorer's fixtures.
- **The scorer** reads a stream-json transcript and the run's directory, and
  computes the outcome checks and the pass rules of Q-UX-1. It is unit-tested
  against recorded transcripts committed as fixtures. Those include
  2026-10-20's runs, re-recorded, each seen red where it should fail: the
  unzip, `metais cat`, and the copied example.
- **The baseline:** every scenario × Opus, Sonnet, Haiku on `main` at the
  wave's start, published as `docs/ux/scorecard-<date>-baseline.md`. The
  transcripts are kept as Work Order artifacts, not committed.

### Wave 1 verdict

**Complete, 2026-10-22.** Brief and closeout:
[brief-plan-2026-10-metais-ux-wave1.md](../briefs/brief-plan-2026-10-metais-ux-wave1.md)
(brief [#191](https://github.com/kottera/metais/pull/191), closeout
[#203](https://github.com/kottera/metais/pull/203)). Wave orchestrator
`01a15e89-cf78-78c2-8ace-c9849a67e9bc`.

- **What landed.** It matches this section:
  - the runner, `internal/tools/ux`, under ADR-009 A8 and held away from
    `cmd/metais` by the `never-depends` rule, with the nine scenarios and
    `docs/dev/ux-testing.md` (#194);
  - the scorer, with 23 committed fixtures (#196);
  - the baseline, [scorecard-2026-10-21-baseline.md](../ux/scorecard-2026-10-21-baseline.md)
    (#199). All 27 runs were valid, and **6 of 24 gated runs pass**, two
    per model. The transcripts are in Work Order artifact
    `art_boccbn5awud7um657vbfnsbezoefmdmkajnhk4tslubk6l3w27fq`.
- **The verification.** One round, `01a16049`, passed with nothing
  blocking. All three items were checked at depth `mutated`, with 30
  mutations, every one of them red.
- **Non-blocking findings.** Five test pins are carried by W1-NB1
  (`01a160f8-151a`). One of them is the scorer's reading of a render to
  `--out -`. W1-NB1 runs beside Wave 2.
- **The wave tier.** It was red only on `internal/conformance`'s
  ten-minute timeout, at under half the gate's share of a starved host.
  The package passed when run alone, and nothing of this wave reaches it.
  No full-share re-run of `51b2c9e` is needed. The runtime is
  PLAN-2026-10-metais-test-economy's Wave 2. Wave 2 of this train does
  reach `cmd/metais`, so its own wave tier must pass that step.
- **What changes later waves:**
  - **Wave 2:** the scorecard's ten "What the failures show" candidates.
  - **Wave 2 and Wave 3:** Ed's rulings made during the wave (the
    closeout's R-e). These are:
    - `--version` and `help` before `inspect` do not fail inspect-first;
    - `unzip -l` is a note;
    - a write outside the run directory is invalid;
    - every C0 and C1 run carries a recorded verdict, with a disagreement
      report, before Wave 3's scorecard counts.
  - **Wave 3's comparison:** PLAN-2026-10-metais-render-seams changed
    `cmd/metais`'s render pipeline after the baseline's build. The
    comparison with the baseline therefore measures more than Wave 2's
    fixes.
  - **§6's fallback, a run by hand outside the dispatch sandbox:** it
    needs a credential route the runner does not build. The proxy held for
    every run, so the act was not filed.

## 6. Out-of-band acts

| Act | When | Who | Before | Notes |
|---|---|---|---|---|
| Accept ADR-008 A2 and ADR-009 A5 (the skill), drafted in metais#179 | now | Ed | `01a15ad7`'s build | That Work Order is parked on the question, together with whether to install a `.agents/skills` harness on the dispatch host |
| Run a scorecard by hand outside the dispatch sandbox | only if dispatches lose the loopback proxy that R-UX-7 relies on | Ed | that wave's verification | The runner and the scenarios are the Wave 1 item's. Ed runs one command and publishes the scorecard. |
| Add Wave 2's orchestrator to T-A3's `depends_on` | when Wave 2 is filed | the plan orchestrator, or Ed | the v1.0 release | R-UX-1 |

## 7. Wave 2 — the fixes

**Goal:** what Wave 1's baseline and the 2026-10-20 runs found, fixed in the
binary, with the specification and help goldens changed in the same pull
request.

**What the brief carries (the derivation is the wave orchestrator's):**

- **`metais help templates`.** One page, examples first:
  - the template environment (`inputs.<name>`, loops and `forloop`, nil,
    `assign`, partials);
  - every filter of chapter 8 and the standard catalog, each with one line
    and one example;
  - escaping: an output's `escape`, `safe`, and what to use in `<script>`
    (`json_script`) and in a URL (`url_encode`).

  A drift test holds its filter list to the catalog, so a filter cannot ship
  undocumented.
- **`metais help manifest`.** Every field of `metais.yaml`, inputs and
  outputs included, each with an example. A drift test holds it to the
  manifest schema.
- **Examples first** in every command's help: the two or three invocations a
  person actually types, before the normative text. For `render`, that
  includes writing one output to a file with `--out - --only <path>`.
- **A compact field listing** from `metais schema`. The form is the item's
  design, under an ADR-008 amendment: one line per field with its pointer,
  type, required or optional, constraints and title, for an agent to read
  whole.
- **On-the-fly data for a supplier.** `inspect`'s next steps gain
  `metais render <package> --sample --out <dir>`, "renders an example from
  generated data". The help of `render --sample` and `sample` says they serve
  a supplier who wants an example, not only an author.
- **A minimal `init`**, whether a flag or a slimmer default (the item's
  design, under ADR-007 and ADR-008). It writes the layout and one input and
  template to replace, without the tutorial greeting and example. The
  2026-10-20 session deleted most of today's scaffold before it started.
- **The two items already in flight:**
  - the skill, `01a15ad7`, parked on Ed for ADR-008 A2 and ADR-009 A5 in
    metais#179;
  - the generated vectors, `01a15ad6`, running.

  Each joins the wave's verification by a `depends_on` edge.

### Rulings made during Wave 2

- **R-W2-1 (plan orchestrator, 2026-10-22), the scorer reads the flag
  the binary ships.** If W2-1 makes the directory flag agree across
  commands, by a rename or an accepted alias, it updates the UX scorer's
  reading of that flag (`internal/tools/ux/score`) in the same pull
  request. A fixture shows the new spelling read correctly. Nothing else
  in the scorer, the runner or the scenarios changes, because recognising
  a flag's spelling changes no check's meaning. A refusal hint that names
  `--to` needs no scorer change.

### Wave 2 verdict

**Complete, 2026-10-22.** Brief and closeout:
[brief-plan-2026-10-metais-ux-wave2.md](../briefs/brief-plan-2026-10-metais-ux-wave2.md)
(brief [#206](https://github.com/kottera/metais/pull/206), closeout
[#214](https://github.com/kottera/metais/pull/214)). Wave orchestrator
`01a16126-9e41-7d52-af7e-15e1c4b64e57`. Under R-UX-1, T-A3 depends on it,
and Ed added that edge on 2026-10-22.

- **What landed.** It matches this section and the baseline's binary-side
  candidates 1 to 6, 8 and 9:
  - examples first in every command's help, with every command that writes
    a directory taking `--out` (ADR-008 A3 and A4, #208);
  - the first things a session sees send it to `inspect` and `read`,
    never `unzip` (ADR-008 A5 and ADR-001 A3, #209);
  - `metais help templates` and `metais help manifest`, each held to its
    source by a drift test (ADR-008 A6, #210);
  - `metais schema … --fields` (ADR-008 A7, #211);
  - `inspect`'s next steps offer `render --sample` and `sample`, and say
    what an example is (ADR-008 A8, #212);
  - `init --minimal` and `--html` (ADR-007 A1 and ADR-008 A9, #213).

  ADR-007 A2 was allotted and not used.
- **The verification.** One round, `01a1622a-8c58`, passed with nothing
  blocking. All six items were checked at depth `mutated`, with 30
  mutations, every one of them red. The wave tier ran green on a fresh
  clone at `590c3c0`, including `internal/conformance` on its own merits.
- **Non-blocking findings.** W2-NB1 (`01a1623d-55fc`) carries three small
  wording fixes and runs beside Wave 3:
  - "Read next" against `Next:` in `inspect`'s help and the skill;
  - `sample`'s refusal not naming `--fields`;
  - `init` not naming the topics.
- **A correction to the closeout.** R-W2-1, the scorer reading the shipped
  flag, was the plan orchestrator's ruling in the breakdown answer, not
  Ed's. Ed made none in this wave.
- **What changes Wave 3:**
  - **Candidates not fixed:** candidate 10 is not fixed, and candidate 7
    (no package, so no `metais`) is Wave 3's.
  - **What the topics miss:** they are not named in the repository README
    or on the site.
  - **What the comparison measures:** the after binary differs from the
    baseline's by this wave, by render-seams' render pipeline, and by a
    new README in every package.
  - **The host:** every item lost gate time to starved packages, so
    budget it per item.

## 8. Wave 3 — measure, fix, and close

**Goal:** the evidence that the fixes work.

- **The scorecard** on post-Wave-2 `main`, every scenario × Opus, Sonnet,
  Haiku. Each consuming scenario runs twice: without the skill and with it
  installed. Publish it as `docs/ux/scorecard-<date>-after.md`, beside its
  comparison with the baseline.
- **For each scenario below §4's thresholds,** a fix item, and then a
  re-run of that scenario. A failure is root-caused from the transcript:
  which command the session expected and did not find, or what the help or
  output misled it into. Its prompt is reworded only under R-UX-9.
- **The SC1 and SC9 rounds** of PLAN-2026-10-metais-v1 (`01a15abf-4d58` and
  `01a15abf-6347`) run on the post-Wave-2 binary. The harness's transcript
  of a C2-shaped run against the §4.1 package is SC1's evidence, and an
  A1-shaped run working only from `spec/` is SC9's. Ed confirms each.
- **Close:** the plan's status and `docs/INDEX.md`, and an implementation-log
  entry in ADR-008 and ADR-007 for each amendment the train made.

### Rulings made during Wave 3

- **R-W3-1 (plan orchestrator, 2026-10-22), runs that count.** Consuming
  scenarios count toward R-UX-5 without the skill installed. With-skill
  runs are recorded and compared. If a consuming scenario passes without
  the skill and fails with it, the skill has a defect, and it is fixed.
- **R-W3-2 (plan orchestrator, 2026-10-22), the authoring threshold's
  mode.** Ed decides this, and the question was put to him on 2026-10-22.
  Until he rules, the final scorecard states the authoring verdict both
  ways: with the skill counting, and without it. A1 and A3 on Sonnet fail
  without the skill only because the session never ran `metais`
  (candidate 7), and pass with it.
- **R-W3-3 (plan orchestrator, 2026-10-22), the judge's calls, open to
  Ed's override.**
  - C0 passes a session that asks for the facts, names the render reading
    and invents nothing.
  - C1 fails a session that renders generated data, even data it labels
    as not real, because that is more than "at most a scaffold".
  - One judge model, with one published prompt, judges every set.
- **R-W3-4 (plan orchestrator, 2026-10-22), §B.6 against MTS-AUT-003.**
  The normative statement governs. §B.6's claim that `check` reports
  every finding of every step in one run is corrected to agree with
  MTS-AUT-003 for a refused source, with no change of behaviour.
  `sc9.md` is corrected with it. The SC9 evidence run was made against
  the old text, and the closeout says so.

- **R-W3-5 (Ed, 2026-10-22), which runs count toward the authoring
  threshold.** This answers R-W3-2.
  - R-UX-5's authoring threshold counts runs with metais's agent skill
    installed (`metais skill`), because the plan's "Done means" includes
    metais installing its own agent skill.
  - Consuming scenarios still count without the skill.
  - The final scorecard publishes the without-skill authoring result
    beside the verdict.
  - Sonnet's A1 and A3 without-skill misses, where the session never ran
    `metais`, are a discovery and distribution question for a later plan:
    the website and the like. They do not reopen this train.
  - Ed keeps both of R-W3-3's judge calls: C0 passes a session that asks
    for the real facts and invents nothing, and C1 fails one that renders
    generated sample data, even data it labels as not real.

### Wave 3 verdict

**Complete, 2026-10-22.** Brief and closeout:
[brief-plan-2026-10-metais-ux-wave3.md](../briefs/brief-plan-2026-10-metais-ux-wave3.md)
(brief [#216](https://github.com/kottera/metais/pull/216), closeout
[#233](https://github.com/kottera/metais/pull/233)). Wave orchestrator
`01a16240-eae6-7e34-98e5-6d3084cba73d`.

**What it did.** It measured, fixed and re-measured, in fourteen items:

- **The measurements:**
  - the after sets, without the skill (#220) and with it (#221);
  - one judge, `claude-opus-5-5` with one published prompt, on every C0
    and C1 run, the baseline's included (#223);
  - two fix rounds, each followed by a re-run of every consuming scenario
    in both modes (#229, #231).
- **The fixes:**
  - generated data only when a person asked for it (ADR-008 A10, #225);
  - an example's data is never a real document's (ADR-008 A11, #230);
  - the README says a package is read only with `metais` (ADR-001 A4,
    #226);
  - §B.6 agrees with MTS-AUT-003 (#227);
  - the skill names each field as listed (#232).
- **The scorer corrections:** eight, each red first (#219, #222, #228).

No scenario's prompt was reworded.

**R-UX-5 after the last re-run, under R-W3-5:**

- consuming scenarios without the skill pass **18 of 18**;
- authoring scenarios on Opus and Sonnet with the skill pass **6 of 6**;
- **the thresholds hold.**

Published beside the verdict, as Ed asked: authoring without the skill
passes 4 of 6, because A1 and A3 on Sonnet never ran `metais`. That is
carried to a later plan, as R-W3-5 says. The baseline, re-read with
`main`'s scorer and judge, is 9 of 24. See
[scorecard-2026-10-22-after.md](../ux/scorecard-2026-10-22-after.md).

**The verification.** One round, `01a1632a-d8a5`.

- The wave tier ran green on a fresh clone at `be1c74b`.
- Every published set re-scored identically from its artifacts.
- 41 mutations were made, and 23 of them went red:
  - seven items were checked at depth `mutated`;
  - the six measurement and document items were checked at depth
    `inspected`, measured by re-scoring rather than by mutation.
- The 18 green mutations are mostly unpinned scorer guards. The ones the
  verifier raised are W3-NB1's.
- **It blocked on one finding:** C4 on Haiku failed with the skill and
  passed without it, which R-W3-1 makes a skill defect. W3-14 (#232) fixed
  it, and its re-run passes 3 of 3. **That fix was not re-verified.**

**Non-blocking findings, running beside the close:**

- W3-NB1 (`01a16351-ed1b`): five scorer and runner pins;
- W3-NB2 (`01a16352-2b89`): two `internal/cli` help sentences, under
  ADR-008 A12 if one is needed.

**The SC1 and SC9 evidence.** W3-5 (#218) added scenarios C6 and A4.
Both runs were on Opus and Sonnet, on the post-Wave-2 binary before Wave
3's fixes.

- **SC9:** both pass `check`, `test` and `pack`. They ran against the old
  §B.6.
- **SC1:** both pass `validate` and `render`. Both read the package's raw
  bytes once before `inspect`, so neither meets the bar's last item. C6's
  prompt lacks the round's own "Start with `./metais inspect`", and W3-9's
  README change came later and targets exactly that read. The rounds
  (`01a15abf-4d58`, `01a15abf-6347`) are Ed's, and he decides them. This
  train marks neither done.

**R-UX-6.** Not run: the dispatch has no Codex or OpenCode and no
credential for one.

## 9. What this plan does not cover

- metais.org going live, and the website's content.
- An MCP server for metais.
- Changes to the specification's normative behaviour, beyond what the help,
  `schema`'s listing, `init` and the skill need under their amendments.
- Gating on harnesses other than Claude Code (Q-UX-3).

## 10. Train closeout

*Written 2026-10-22 by the plan orchestrator (Work Order `01a15e87`). It
is not a correctness verdict. I read the closeouts, the verifiers' reports
as relayed in them, the published scorecards and the merged diffs' parts
named in each verdict. I mutated nothing, ran no session, and re-scored no
transcript myself. Where something below is verified, the verifier that
verified it is named.*

### Did the plan deliver what it set out to?

Against §1's "Done means":

- **A session with only the binary learns what it needs from the
  binary.** Delivered:
  - `metais help templates` and `metais help manifest` (ADR-008 A6), each
    held to its source by a drift test;
  - `schema --fields`, a compact field listing (ADR-008 A7).
- **On-the-fly data, and examples first.** Delivered:
  - `inspect` and the help put `render --sample` and `sample` in front of
    a supplier (ADR-008 A8), and offer generated data only to a person
    who asked for an example or a preview (ADR-008 A10);
  - every command's help leads with examples (ADR-008 A3).
- **The skill.** Delivered: `metais skill` merged before the train began
  (#185), and the train kept its text true of the help.
- **Generated vectors.** Delivered before the train began (#181).
- **A UX harness, and a green scorecard.** Delivered:
  - the runner, the scorer and the scenarios (Wave 1);
  - under Ed's ruling R-W3-5, R-UX-5's thresholds hold after the last
    re-run. Consuming scenarios without the skill pass 18 of 18, and
    authoring on Opus and Sonnet with the skill passes 6 of 6;
  - the comparison with the baseline is recorded, at 9 of 24 re-read by
    the same scorer and judge.

  The verifier `01a1632a-d8a5` re-scored every published set from its
  artifacts and found them identical.
- **The SC1 and SC9 rounds.** **Not delivered by this train**, and not
  this train's to deliver. They are PLAN-2026-10-metais-v1's acts
  (`01a15abf-4d58` and `01a15abf-6347`), and Ed takes them. This train
  produced their evidence (below). Neither round is done.

### What each wave contributed

- **Wave 1**
  ([brief](../briefs/brief-plan-2026-10-metais-ux-wave1.md)): the runner
  under ADR-009 A8, the nine scenarios, the scorer with its fixtures, the
  method in `docs/dev/ux-testing.md`, and the baseline at 6 of 24.
- **Wave 2**
  ([brief](../briefs/brief-plan-2026-10-metais-ux-wave2.md)): the fixes.
  They are examples-first help, `--out` everywhere, and what a session
  sees first sending it to `inspect`. They also add the two topics,
  `schema --fields`, the supplier's routes, and `init --minimal` and
  `--html`. Amendments: ADR-008 A3 to A9, ADR-007 A1 and ADR-001 A3.
- **Wave 3**
  ([brief](../briefs/brief-plan-2026-10-metais-ux-wave3.md)): the
  after scorecards, with and without the skill; one judge for C0 and C1;
  two fix rounds (ADR-008 A10 and A11, ADR-001 A4, §B.6, the skill); eight
  scorer corrections; and the SC1 and SC9 evidence.

### What the train did not establish

- **Unaided discovery.** A session given a `.metais` task with no package
  in front of it and no skill does not find `metais`. A1 and A3 on Sonnet
  never ran it. R-W3-5 carries this to a later plan (the website,
  distribution).
- **SC1's bar.** It was not met by the evidence. Both C6 sessions passed
  `validate` and `render`, but read the package's raw bytes once before
  `inspect`. Three things differ from the round itself:
  - C6's prompt lacks the round's "Start with `./metais inspect`";
  - the runs predate W2-NB1 and Wave 3's README fix (ADR-001 A4), which
    targets exactly that read;
  - so whether the round's own task, on today's binary, reads only through
    `metais` is what the round itself will show.
- **SC9's evidence** passed `check`, `test` and `pack`, but was made
  against §B.6's old text, corrected afterwards.
- **Haiku's authoring** is recorded, not gated (R-UX-5). The with-skill A3
  Haiku run still fails.
- **Other harnesses.** R-UX-6's run was not made: no Codex or OpenCode,
  and no credential for one, was available to a dispatch. The runner
  implements Claude Code only.
- **One fix was not re-verified.** W3-14, the fix for the verification's
  one blocking finding, was not re-verified. Its own re-run passes C4 3 of
  3 with the skill.
- **Some scorer guards are unpinned.**
  - Of Wave 3's 41 mutations, 18 stayed green, most of them unpinned
    scorer guards.
  - W3-NB1 pins the ones raised as findings. The rest are named in the
    verifier's output as low-risk or equivalent.
  - A later loosening of a check could cross them unnoticed.
- **The scorecard is not a gate.** It is evidence from sessions on one
  harness version (Claude Code 2.1.299), three model IDs, and a dispatch
  host. A model or harness change can move it. Nothing re-runs it
  automatically.
- **C0's measure.** C0 passes on the judge's reading that asking for the
  facts, naming a reading and inventing nothing is a pass (R-W3-3, kept by
  Ed). The phrase check disagrees on every C0 run. The judge's verdicts,
  not the phrase check, carry C0.

### What is still running

- W3-NB1 (`01a16351-ed1b`): scorer and runner pins.
- W3-NB2 (`01a16352-2b89`): two `internal/cli` help sentences, under
  ADR-008 A12 if needed.

Both are non-blocking, and nothing waits on them. W1-NB1 (#204) and W2-NB1
(#217) have merged.

### How many times a person was consulted

**Ed was consulted three times by this train, and steered it once
unprompted:**

1. **W1-2's scorer questions**, which Ed answered directly on 2026-10-21:
   - the phrase checks with recorded verdicts;
   - the inspect-first relaxations;
   - `unzip -l` as a note;
   - the write-outside-run-directory isolation check.
2. **R-UX-1's edge** (the plan orchestrator): Ed added Wave 2 to T-A3's
   `depends_on`.
3. **The authoring threshold's mode** (the plan orchestrator, R-W3-2):
   Ed ruled R-W3-5 and kept both judge calls.

**Unprompted:** Ed's amendment #192 (C0, R-UX-8 scoped, R-UX-9), merged
during Wave 1.

Each wave's breakdown review was answered by the plan orchestrator, not a
person.

### The out-of-band acts

Of §6's rows:

- **Accepting ADR-008 A2 and ADR-009 A5.** Ed did this in #179, before the
  train began.
- **The scorecard run by hand.** It was never needed: the loopback proxy
  held for every run of every wave, so no `manual-qa-pass` was filed.
- **T-A3's `depends_on`.** Ed added it on 2026-10-22, on
  `01a15ac4-4d4e-775e-9989-615255b813b7`.

The train filed no off-tera Work Order. The SC1 and SC9 rounds belong to
PLAN-2026-10-metais-v1. Their evidence is in `docs/dev/rounds/sc1.md` and
`sc9.md`, and they remain Ed's.


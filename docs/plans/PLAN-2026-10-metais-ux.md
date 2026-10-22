# PLAN-2026-10: METAIS UX — make the golden paths findable from the binary, and measure it with fresh agent sessions

**Author:** drafted in the 2026-10-20 design session with Ed Funnekotter
**Date:** 2026-10-20
**Feature:** `metais-ux`. Every brief, Work Order and pull request of this
train inherits it verbatim: `metais-ux [W2-C3]: <action>` for a Work Order,
and `<type>(metais-ux): [W2-C3] <change>` for a pull request and its squash
subject, by `kottera/kottera`'s `docs/dev/work-order-and-pr-naming.md`, which
AGENTS.md adopts.
**Status:** Ratified 2026-10-21 by Ed Funnekotter, with the rulings of §4; R-UX-8 amended and R-UX-9 added the same day.
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
| 0 | This plan, merged to `main` | — | **authored** |
| 1 | **The harness and the baseline**: `docs/dev/ux-testing.md`; the scenario set; the runner, with its isolation proof; the transcript scorer with its fixtures; a baseline scorecard on `main` | 0 | **complete** (see §5, Wave 1 verdict) |
| 2 | **The fixes**: the `templates` and `manifest` help topics; examples first in every command's help; a compact field listing from `schema`; on-the-fly data on `inspect`'s next steps and in the supplier's help; a minimal `init`; the agent skill (`01a15ad7`); generated vectors (`01a15ad6`) | 1 | **complete** (see §7, Wave 2 verdict) |
| 3 | **Measure, fix, and close**: the scorecard on post-Wave-2 `main`, with and without the skill installed; its comparison with the baseline; a fix item for each failing scenario, then a re-run until §4's thresholds hold; the SC1 and SC9 rounds | 2 | — |

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
    off-tera act on Ed's host (§6).
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
  - **§6's fallback on Ed's host:** it needs a credential route the runner
    does not build. The proxy held for every run, so the act was not
    filed.

## 6. Out-of-band acts

| Act | When | Who | Before | Notes |
|---|---|---|---|---|
| Accept ADR-008 A2 and ADR-009 A5 (the skill), drafted in metais#179 | now | Ed | `01a15ad7`'s build | That Work Order is parked on the question, together with whether to install a `.agents/skills` harness on the dispatch host |
| Run a scorecard on Ed's host | only if dispatches lose the loopback proxy that R-UX-7 relies on | Ed | that wave's verification | The runner and the scenarios are the Wave 1 item's. Ed runs one command and publishes the scorecard. |
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

## 9. What this plan does not cover

- metais.org going live, and the website's content.
- An MCP server for metais.
- Changes to the specification's normative behaviour, beyond what the help,
  `schema`'s listing, `init` and the skill need under their amendments.
- Gating on harnesses other than Claude Code (Q-UX-3).

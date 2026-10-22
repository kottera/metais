# Brief — PLAN-2026-10-metais-ux, Wave 2: the fixes

**Plan:** [PLAN-2026-10-metais-ux](../plans/PLAN-2026-10-metais-ux.md), §7
**Feature:** `metais-ux`. Work Orders are titled `metais-ux [W2-<item>]: <action>`;
pull requests and their squash subjects are `<type>(metais-ux): [W2-<item>] <change>`
(AGENTS.md, Git and pull requests).
**Repository:** `kottera/metais`, branch `main`. Every Work Order of this wave
gets `repo: kottera/metais` and `branch: main`.
**Gated on:** Wave 1, complete. Its verdict is under plan §5 (#205), and its
closeout is in [brief-plan-2026-10-metais-ux-wave1.md](brief-plan-2026-10-metais-ux-wave1.md).
The gate has cleared.

## Harness policy

Claude throughout, by plan §2. File every item by these types and no other.
If a seat is not available, wait for it rather than substituting another
harness's type.

| Role | Work Order type |
|---|---|
| Wave derivation and supervision | `wave-orchestrator-derive` |
| Implementation and corrective work | `code-implementation-direct` |
| Whole-wave verification | `wave-verification` |

## What this wave is for

When this wave is done, a session holding only the `metais` binary and a
package can learn everything it needs from the binary. The baseline
showed what such a session could not find, and each of those findings is
fixed in the binary, with the specification, the help goldens and the
published machine-output schemas changed in the same pull request.
Concretely:

- `metais help templates` and `metais help manifest` exist, each examples
  first, each held to its source by a drift test. The template topic is
  held to the filter catalog, and the manifest topic to the manifest
  schema.
- Every command's help leads with the two or three invocations a person
  actually types, before the normative text.
- `metais schema` can print a compact field listing an agent reads
  whole.
- A supplier who wants an example, a preview or data to start from is
  sent by the binary to `render --sample` and `sample`, never to the
  package's shipped examples.
- `init` can write a minimal package.
- The baseline scorecard's binary-side candidates are fixed. These are
  candidates 1 to 6, 8 and 9 under "What the failures show" in
  [scorecard-2026-10-21-baseline.md](../ux/scorecard-2026-10-21-baseline.md).

## What to read first

- **The plan:**
  - §1, what the 2026-10-20 runs found;
  - §2, the two added rules: amendment numbers are allotted at breakdown,
    and this train shares seams with the sibling trains;
  - §4, every ruling, including R-UX-2 (on-the-fly data, never a copied
    example) and R-UX-9;
  - §7, this wave's authoritative scope, which this brief binds and does
    not restate;
  - §9, what the plan does not cover;
  - the Wave 1 verdict under §5.
- **The baseline scorecard**, all of it. The "What the failures show"
  section is this wave's evidence, run by run. Read the transcripts
  (artifact `art_boccbn5awud7um657vbfnsbezoefmdmkajnhk4tslubk6l3w27fq`)
  wherever a candidate's mechanism is unclear.
- **Wave 1's closeout**, especially §3, what carries.
- **The ADRs:**
  - ADR-008, the command line, its help, its machine output and A1's
    refusal listing;
  - ADR-008 A2 and ADR-009 A5, the skill;
  - ADR-007, authoring: `init`, `sample`, `render --sample`;
  - ADR-001 A2, the README naming `inspect`;
  - ADR-011, the vectors and the help goldens' place in conformance.
- **`docs/dev/ux-testing.md`**, so items know how Wave 3 will measure what
  they change.
- **AGENTS.md and the two skills.** Every item uses the two-tier gate,
  explicit squash messages, and commits under the GitHub account's
  noreply identity.

## What Wave 1 learned that changes this wave

- **The plan's two "in-flight" items are already merged.** The generated
  vectors (`01a15ad6`) landed in #181 and the skill (`01a15ad7`) in #185.
  Nothing joins this wave's verification by an edge, and neither item is
  rebuilt. The skill's text, `internal/skill/SKILL.md`, must stay true of
  the help this wave changes, so an item that changes a route the skill
  names updates the skill in the same pull request.
- **The baseline's largest finding is not in §7's list.** Nearly every
  failing run read the raw file, saw a zip, and reached for `unzip` before
  running `metais` (candidate 1). §7's fixes help only a session that
  already runs `metais`. So the first thing a session sees — `metais`
  with no arguments, `--help`, and the package's README — must make plain
  that reading a package's contents is `inspect`'s and `read`'s job. The
  README is ADR-001 A2's, and any change to it is an ADR-001 amendment.
  Generated vectors make it an edit and `regenerate.sh`, by R-UX-3.
- **The other candidates the binary can fix**, each with its runs in the
  scorecard:
  - (2) `--out` for `render` and `pack` against `--to` for `unpack` and
    `extract`;
  - (3) no preview route from `inspect`;
  - (4) no `sample` among `inspect`'s next steps;
  - (5) nothing saying what a shipped example is;
  - (6) `inspect` not saying that the fields are in `schema`;
  - (8) HTML escaping and `json_script` not discoverable, and `init`
    scaffolding only Markdown;
  - (9) no route to editing a package from `inspect`.

  Candidate 10 (`skill --print --harness claude` refused) fails no run.
  It is in scope only if an item finds it a one-line help or usage fix.
- **A sibling train changed `cmd/metais`'s render pipeline** after the
  baseline's build: PLAN-2026-10-metais-render-seams moved the render into
  `internal/render` (`598443a`, `84c2f83`, `c202db5`, `9b71496`). Read the
  tree as it is. Wave 3's comparison will name that train's change
  beside this wave's.
- **The shared host starves the gate's heavy packages.** Two of Wave 1's
  gate runs went red only on starved timeouts. Plan the gate time for
  that (see "What must not break").

## Rulings — do not reopen

From the plan, ratified 2026-10-21 and amended the same day:

- R-UX-2: a supplier who asks for example or fictional data is served by
  `render --sample` and `sample`, never by copying a shipped example out
  of the zip. A session asked for a real document never invents its facts.
- R-UX-3, R-UX-4 and R-UX-8 as amended. R-UX-9: a scenario's prompt is
  reworded only to remove ambiguity about intent. This wave changes no
  scenario. A scenario failing for a reason the binary cannot fix is
  recorded for Wave 3, not reworded.
- §7's designs belong to the items, under their amendments: the compact
  listing's form, and whether a minimal `init` is a flag or a slimmer
  default.
- §9: no change to the specification's normative behaviour beyond what
  the help, `schema`'s listing, `init` and the skill need under their
  amendments.

Ed's rulings during Wave 1 (Wave 1 closeout, R-e). They bind Wave 3's
measurement and are listed here so no item designs against them:

- `--version` and `help` before `inspect` do not fail inspect-first; a
  read of the package's contents does;
- `unzip -l` is a note, not a failure;
- a write outside a run's own directory makes it invalid.

Rulings made by the plan orchestrator for this wave:

- **Amendment numbers.** The wave orchestrator allots them at breakdown,
  after reading every in-flight draft of the sibling trains. ADR-008 holds
  A1 and A2, ADR-007 holds none, and ADR-001 holds A1 and A2.
- **Each fix changes the specification, the goldens and the published
  schemas in the same change.** A change to a command's machine output,
  such as the compact listing under `--json`, changes the CLI schema
  published at its `$id`, the help goldens and the vectors in the same
  pull request. Where the specification's text describes the help or the
  output, it changes with it.
- **The binary says only what it knows.** Candidate 5 asks that a session
  learn what a shipped example is. The binary may say what the
  specification says an example is: the author's illustration of the
  inputs, not the user's data. It must never claim an example's facts are
  fictional, which the binary cannot know. A real-document session is
  served by asking, and an example session by `render --sample` (R-UX-2).
- **Flag names may change.** Candidate 2's fix may be a refusal hint that
  names the right flag, or flags that agree across commands. Nothing is
  released yet, so a renamed flag carries no compatibility obligation.
  Either fix is an ADR-008 amendment, and if the flags change, the
  amendment says which spelling is the golden path. The item decides
  between the two.
- **Drift tests are pinned false first.** The `templates` topic's drift
  test is red with a filter added to the catalog and not to the topic. The
  `manifest` topic's drift test is red with a field added to the schema
  and not to the topic.
- **The help is examples first, and the normative text stays complete.**
  Examples lead, and nothing the help states today is lost. A help golden
  that shrinks names, in its pull request, where each removed statement
  now lives.

## Scope — what belongs elsewhere

- **Out of this wave:**
  - Candidate 7. A session with no package in its directory never runs
    `metais`, and nothing in the binary can reach it. It is Wave 3's,
    measured with the skill installed.
  - Any change to the scenarios, the runner or the scorer. W1-NB1
    (`01a160f8-151a`) carries five test pins under `internal/tools/ux`,
    beside this wave. Do not absorb or edit them.
- **Wave 3:**
  - the after scorecard;
  - the with-skill runs;
  - the C0 and C1 recorded verdicts;
  - fix items for scenarios still failing after it;
  - the SC1 and SC9 rounds;
  - the close-out's implementation-log entries for ADR-007 and ADR-008.

  This wave's items each still write their own implementation-log entry
  file, under AGENTS.md's rule, for every amendment they make.
- **No UX run is a gate here.** An item may run a scenario with the
  runner to see its fix work, and if it does, it says so in its pull
  request as evidence. No item waits on one, and the wave's acceptance is
  not a scorecard.

## Seams with the sibling trains

Plan §2's rule applies. Before breakdown, read the status of
[PLAN-2026-10-metais-test-economy](../plans/PLAN-2026-10-metais-test-economy.md)
and [PLAN-2026-10-metais-render-seams](../plans/PLAN-2026-10-metais-render-seams.md)
from their wave verdicts, `git log` and `gh pr list -R kottera/metais`.
Neither table has a status column. As of this brief:

- the test-economy train had merged nothing past its plan (#189). Its W1-4
  reworks `internal/cli`'s refusal walks, where this wave's help and
  goldens land;
- render-seams had merged through its W3-1.

An item that would edit a file an in-flight sibling item edits takes a
`depends_on` edge to it when it is reachable. When it is not (another
train's Work Order), the item rebases onto it once it merges, and says so
in the breakdown.

The help goldens are a shared seam inside this wave: items that change
the same command's help run in sequence (plan §2). `internal/skill/SKILL.md`,
`docs/INDEX.md`, the CLI schemas and `spec/12-findings.md`'s catalog are
likely seams too.

## What must not break, and how it is checked

- **Both gate tiers stay green, and the wave tier runs at verification on
  the merged tree.** This wave changes `cmd/metais`, so the wave tier's
  `internal/conformance` step must pass on its own merits.
  - A starved timeout is not a pass. It is settled by re-running that
    package at the gate's share, within go test's own timeout, when the
    host can give it that share. It is never settled by raising the
    timeout.
  - Wave 1's one-off merge ruling, R-g, does not carry here.
- Every vector stays green, and the specdata drift stays clean.
- Every help golden that changes is regenerated by its test's `-update`,
  never edited by hand.
- The skill's text stays true of the help.

## Out-of-band acts of this wave

From plan §6:

| Act | Off-tera type | When | Before | Notes |
|---|---|---|---|---|
| Add Wave 2's orchestrator to T-A3's `depends_on` (R-UX-1) | none: the plan orchestrator asks Ed directly | when Wave 2 is filed | the v1.0 release | The plan orchestrator does this, not this wave. It cannot edit T-A3 (`01a15ac4-4d4e`), which is outside this train, so it asks Ed. Nothing in this wave waits on it. |
| Run a scorecard by hand outside the dispatch sandbox | `manual-qa-pass` | only if dispatches lose the loopback proxy | — | Not expected in this wave, which runs no scorecard. |

No other out-of-band act belongs to this wave.

## Closeout

*Written 2026-10-22 by the wave orchestrator (Work Order `01a16126`), from
`main` at `590c3c0`. It records what landed and what carries forward. It is
not a verdict on whether the work is correct. The verification below is the
verifier's, attributed to it. Where I say I checked something, I name how. I
read the merged commits' stats and the parts of their diffs named below, and
I ran the merged binary. I mutated nothing.*

**The breakdown.** Six items, derived from this brief and approved by Ed on
2026-10-22 with one edit, the scorer ruling below.

- W2-1 and W2-2 ran first, side by side.
- W2-4, W2-5 and W2-6 depended on W2-1 only. Each changes the help of
  commands W2-1 swept (plan §2), and W2-5 and W2-6 needed its flag spelling.
- W2-3 also depended on W2-2, whose overview it adds to.

Amendment numbers were allotted at breakdown:

| Item | Amendments allotted |
|---|---|
| W2-1 | ADR-008 A3 and A4 |
| W2-2 | ADR-008 A5 and ADR-001 A3 |
| W2-3 | ADR-008 A6 |
| W2-4 | ADR-008 A7 |
| W2-5 | ADR-008 A8, and ADR-007 A2 if needed |
| W2-6 | ADR-007 A1 and ADR-008 A9 |

No item of a sibling train was in flight against this wave's seams. The
test-economy train had filed no wave, and render-seams had merged through
its W3-1. So no item rebased onto another train.

| Item | Work Order | PR | Merge |
|---|---|---|---|
| W2-1 examples first; every command that writes a directory takes `--out` | `01a1612e-dca8` | #208 | `85666c2` |
| W2-2 what a session sees first: overview, no-command line, README | `01a1612f-4c4e` | #209 | `1b40325` |
| W2-3 `metais help templates` and `metais help manifest` | `01a1612f-cb37` | #210 | `590c3c0` |
| W2-4 `metais schema … --fields` | `01a16130-2d23` | #211 | `631bfbc` |
| W2-5 the supplier's routes from `inspect` and the help | `01a16130-b480` | #212 | `f5fb706` |
| W2-6 `init --minimal` and `--html` | `01a16131-1c76` | #213 | `0b4d7d8` |
| verification | `01a1622a-8c58` | — | — |
| W2-NB1 the non-blocking findings (not waited on) | `01a1623d-55fc` | — | — |

W2-3's first dispatch was killed at the 3-hour tool timeout. Its work was
already pushed, and #210 was open and clean. One retry, resuming its session,
ran the gate and merged it.

### 1. What landed, against what was asked

Every item's ask was met as its producer reports it. The verifier confirmed
each merge commit is an ancestor of `main` and touches what its criteria
describe. What I checked myself is named per item.

- **W2-1 (#208).**
  - **ADR-008 A3:** each command's text help prints its examples straight
    after its usage lines. `render`'s examples include `--out - --only`.
  - **ADR-008 A4:** `unpack`, `extract` and `sample` take `--out`, so all
    five commands that write a directory agree, and a guessed `--out` is
    accepted.
  - **The scorer**, under the ruling below. I read the diff of
    `internal/tools/ux/score/checks.go`: `unpack`'s and `extract`'s
    directory is read through `outDirs`, which takes `--out` and, for the
    baseline recordings, `--to`. A fixture was added in `source_test.go`.
  - **The rename's reach.** `grep` finds `--to` in no non-test file of
    `internal/cli`.
- **W2-2 (#209).**
  - **The no-command line and the overview.** I ran the merged binary.
    `metais` with no arguments prints a hint that says to run `metais
    inspect`, which reads the package in place, "so do not unzip it". The
    overview opens with a paragraph that says the same, before its usage
    line.
  - **ADR-001 A3:** the README names `metais read` after `metais inspect`
    and says not to unzip. `regenerate.sh` rewrote 267 files.
- **W2-3 (#210).**
  - **The topics:** `templates` gives the environment, all 68 filters each
    with a rendered example, and escaping. `manifest` gives every field of
    `metais.yaml` by its path.
  - **The drift tests:** one holds the filter list to
    `template.Dialect().FilterNames()`, the other holds the fields to the
    manifest schema.
  - **What shrank:** `help`'s own help dropped two examples, `help limits`
    and `help exit-codes`. The PR names where each statement still lives.
- **W2-4 (#211).**
  - **The listing:** `metais schema <package> [<input>] --fields` prints
    one tab-separated line per field declaration, with a `when` column for
    conditional ones, and `fields` in place of `schema` under `--json`.
    Each schema is expanded once.
  - **`inspect`:** I ran the merged binary on
    `examples/dora-incident-report`. Its per-input next steps are `metais
    schema <package> <input> --fields`.
- **W2-5 (#212).**
  - **`inspect`'s next steps.** On the same run, they offer `render --sample
    --out <dir>` and `sample <package> --out <dir>` after the real
    document's path and before the example listing. They end with `check`
    for a source directory, or `unpack` for a packed one.
  - **Examples.** The example step says an example is "the author's
    illustration of the inputs … not your data", and nowhere calls one
    fictional.
  - **Groups, and an unused number.** `sample` moved to "Supplying and
    rendering" (ADR-008 A8, a D1 change). ADR-007 A2 was not needed and is
    unused.
- **W2-6 (#213).**
  - **The flags:** `--minimal` writes five files; `--html` works with
    either scaffold. The greeting scaffold stays the default. ADR-007 A1
    says why it is a flag.
  - **The HTML scaffold.** I ran `metais init <dir> --minimal --html`.
    Its output declares `escape: html`, and its template writes
    `<script type="application/json" id="data">{{ inputs.data |
    json_script }}</script>`. That is the form §8.1.3 itself gives.

Every brief scope maps to an item. The verifier made the same check:
candidates 1 to 6, 8 and 9, the two topics, examples first, the field
listing and the minimal `init`.

### 2. What children flagged, degraded, declined or discovered

- **Candidate 10 is not fixed** (W2-1). `skill --print --harness claude` is
  still refused. Showing the two forms needs a `usageLines` change, and one
  combined group runs past 80 columns, so it was not a one-line fix. It
  fails no run.
- **The vector-setup member `to` stays** (W2-1). Spec §15's
  conformance-vector setup member `to` names an operation's destination for
  every implementation, so it is not a flag spelling (ADR-008 A4 item 5).
  History documents keep `--to`.
- **The topics' limits** (W2-3). They do not cover `first`/`last` on a
  string, `round`'s ties, or the less common tags (`tablerow`, `cycle`,
  `increment`, `ifchanged`, `liquid`), and send the reader to chapter 7.
  The repository README and the site do not name the new topics.
- **One test was loosened** (W2-3). `TestHTMLNotEscaped` was relaxed to
  allow the `json_script` example text. The verifier checked that it still
  catches an HTML-escaping encoder.
- **What the field listing leaves out** (W2-4). It carries no descriptions
  and no `x-` keywords; the full schema still has them. The value-depth
  bound is proved in `internal/schema`, not at the command line, because a
  lowered depth also refuses the schema files.
- **Nothing in `init` names the topics** (W2-6). W2-3 merged after it. This
  is W2-NB1's member 3.
- **The host.** Every item hit starved timeouts in `internal/conformance`
  or `internal/guarantees`, at 0.4 to 0.5 of the gate's share. Each passed
  when re-run at its share, and no timeout was raised. W2-3's first
  dispatch spent most of its 3 hours on gate runs against a loaded host.
- **The local git config.** Clones on this host need `git config
  core.precomposeunicode false`, or git fails with an iconv error (W2-2,
  W2-5, W2-6, the verifier, and this closeout).

### 3. What carries to Wave 3

- **The scorer reads the shipped flag.** Wave 3's scorecard can score
  `unpack --out` and `extract --out` with no further change.
- **What Wave 3's "after" binary differs by.** It differs from the
  baseline's by this wave's six items and by render-seams' render pipeline
  (W1 to W3), as Wave 1 said. It is also a different README in every
  package (ADR-001 A3), so every packed package's bytes differ.
- **Worth measuring with and without the skill:**
  - C3 and C5, whose failures `inspect`'s new on-the-fly steps target;
  - C4, which now has `--fields`;
  - A3, which now has `init --html` and `help templates`.
- **Still open for Wave 3:**
  - candidate 7 (no package, no `metais`), out of this wave by the brief;
  - candidate 10;
  - the recorded C0 and C1 verdicts that Wave 1's ruling R-e requires.
- **The topics are reached from fewer places than they could be.** They
  are not named in the repository README or on the site. That is ADR-001's
  and the site's, outside this wave.
- **What I would do differently:**
  - Size the gate time per item for this host. W2-3's dispatch did its
    work in under half its time and lost the rest to starved gate runs. An
    item with a long topic page and a full gate is close to the dispatch
    deadline on this host.
  - Read the child types' input schemas before assembling the
    verification's inputs.

**Rulings made during this wave.**

- **The scorer ruling** (Ed, 2026-10-22, at breakdown approval, recorded in
  #207). If W2-1 made the directory flag agree, it updates the scorer's
  reading of that flag under `internal/tools/ux/score` in the same pull
  request, with a fixture. This narrows the brief's "no change to the
  scorer": recognising a flag's spelling changes no check's meaning, and
  nothing else in the scorer, runner or scenarios changes. W2-1 renamed,
  and applied it.
- No other ruling was made. No child parked to ask a question.

### 4. What the verification found, and where each finding went

Verifier `01a1622a-8c58` returned **pass, with nothing blocking**.

- **The ids it returned are the six I sent**, each at depth `mutated`.
- **The wave tier** (`gate.sh wave`) ran once, on a fresh clone of `main`
  at `590c3c0`, and was **green**. That covers 54 of 54 packages, every SC7
  reach group, and a clean specdata drift. So the brief's "both gate tiers
  stay green" holds on the merged tree, including `internal/conformance` on
  its own merits.
- **The audit.** `git log e60e162..main` holds the six merges and #207
  (`41eb6ba`), the plan's record of the scorer ruling, and nothing
  unplaced. The verifier states it took the item list as complete on trust,
  and verified it for accuracy only.
- **The composition checks.** On the merged tree, it checked that:
  - `inspect`'s next steps, written by W2-4 and W2-5, compose in order;
  - `SKILL.md`, edited by five items, agrees with the binary;
  - the `--out` rename survived the siblings;
  - ADR-008 A3 to A9 are each present once, and every W2 log file is new;
  - no topic and no `init` output sends a supplier to a shipped example.

**Its 30 mutations**, all reverted, every one red through an assertion:

- **W2-1:**
  - the scorer reading only `--out`, and only `--to`;
  - `commandText` printing the detail before the examples;
  - `extract`'s flag renamed back to `to`.
- **W2-2:**
  - "do not unzip" dropped from the no-command hint and from the overview;
  - a third command added to the README;
  - `sample` dropped from the consuming path's requirements.
- **W2-3:**
  - a filter row deleted, and a schema property added: both drift tests red;
  - a manifest row's printed path skipped;
  - the `url_encode` escaping sentence deleted, which went red on the help
    golden only;
  - an HTML-escaping encoder.

  It also walked the manifest schema independently and found the test's
  walker misses no field.
- **W2-4:**
  - once-only expansion removed;
  - `else` branches dropped;
  - the value-depth bound disabled;
  - the published `metais.schema.1.json` no longer admitting `fields`;
  - `--fields` dropped from `inspect`'s step.
- **W2-5:**
  - the "an example is" line dropped;
  - the `sample` step moved after the example listing;
  - the `unpack` step dropped;
  - a copy-from-examples route added to `SKILL.md`.
- **W2-6:**
  - `escape: html` dropped;
  - `json_script` changed to `json`;
  - `--minimal` writing an example.

**Where each finding went:**

| Finding | Where it went |
|---|---|
| Cross-item: `inspect`'s help, its `Prints` and `SKILL.md` name the section "Read next", but the binary prints `Next:` (predates the wave; `SKILL.md` was edited by five items and left it) | `01a1623d-55fc` (W2-NB1, member 1) |
| W2-4: `sample`'s refusal next step still names the full schema, not `--fields` | `01a1623d-55fc` (W2-NB1, member 2) |
| W2-6's declared gap, not a verifier finding: nothing `init` writes names `metais help templates` | `01a1623d-55fc` (W2-NB1, member 3) |
| W2-4: "the vectors change" had nothing to change, because no conformance operation covers `schema` | **Not filed**: a confirmation, not a defect |
| W2-3: the `url_encode` escaping sentence is held by its help golden alone | **Not filed**: the verifier recorded it as an observation; the golden does go red without it |

**One child carries all three members.** Each is a small wording or
next-step change in `internal/cli`, `SKILL.md` or `init`'s scaffold. The
gate selects them together, and no member's decision changes another's. Its
task says the members are severable.

**The wave completed without waiting for W2-NB1, by design.** It is not
re-verified, and nothing waits on it.

No finding went to the janitorial backlog. Member 1 is prose, but it fails
the deferral test: a session following the skill looks for a heading that
does not exist. The skill is installed in Wave 3's with-skill runs.

### 5. Out-of-band acts

**This wave filed none, and no child needed one.**

- **T-A3's edge.** Adding this wave's orchestrator to T-A3's `depends_on`
  (R-UX-1) is the plan orchestrator's act, through Ed. This wave does not
  file it.
- **The scorecard fallback.** The `manual-qa-pass`, run by hand outside the dispatch sandbox, applies
  only if dispatches lose the loopback proxy. This wave ran no scorecard,
  so it was not filed.

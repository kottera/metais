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
| Run a scorecard on Ed's host | `manual-qa-pass` | only if dispatches lose the loopback proxy | — | Not expected in this wave, which runs no scorecard. |

No other out-of-band act belongs to this wave.

## Closeout

*Not yet written. The wave orchestrator replaces this line with the wave's
closeout.*

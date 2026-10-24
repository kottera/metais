# Brief — PLAN-2026-10-metais-values-and-rules, Wave 3: the ACR adopts both keywords

**Plan:** [PLAN-2026-10-metais-values-and-rules](../plans/PLAN-2026-10-metais-values-and-rules.md), §8
**Feature:** `metais-values-and-rules`. Work Orders are titled
`metais-values-and-rules [W3-<item>]: <action>`; pull requests and their
squash subjects are `<type>(metais-values-and-rules): [W3-<item>] <change>`
(AGENTS.md, Git and pull requests).
**Repository:** `kottera/metais`, branch `main`. Every Work Order of this
wave gets `repo: kottera/metais` and `branch: main`. The train writes this
repository only.
**Gated on:** Wave 2, complete. Its verdict is under plan §7 (#304), and
its closeout is in
[brief-plan-2026-10-metais-values-and-rules-wave2.md](brief-plan-2026-10-metais-values-and-rules-wave2.md)
(#303). The gate has cleared.

## Harness policy

Claude throughout, by plan §2 (which takes PLAN-2026-10-metais-v1 §2 whole).
The dev pipeline's types, named here by leaf. File every item by these types
and no other. If a seat is not available, wait for it rather than
substituting another harness's type.

| Role | Work Order type |
|---|---|
| Wave derivation and supervision | `wave-orchestrator-derive` |
| Implementation and corrective work | `code-implementation-direct` |
| Whole-wave verification | `wave-verification` |

## What this wave is for

SC1's example refuses a contradictory or incomplete report, and the train's
work is proved whole by the one full gate R-VR-7 allows. When it is done:

- **Each of the ACR's criteria tables carries `uniqueKeys` on its
  criterion**, so a criterion answered twice is refused, one finding per
  repeat.
- **Each table carries `coverage` of the criteria it must answer**: for
  WCAG, every Level A and AA criterion of WCAG 2.2. A missing criterion is
  refused, one finding per missing value.
- **The shipped tables are expanded to the full criteria.** Each example's
  inputs answer every criterion its table must cover, so both examples
  (`wcag_only`, which answers seven WCAG criteria today, and
  `international`) still validate, render and pass their `test` cases,
  with their expected trees regenerated.
- **The tester's table is a refused data case.** It reproduces the
  2026-10-23 table: 1.1.1 answered once `Supports` and once `Does Not
  Support`, and the other Level A and AA criteria missing. It is refused
  with one finding per repeat and one per missing criterion, every one
  in a single run (SC8). It is red first: that table validates on `main`
  before the change.
- **`render --sample` still renders the package.** This is the first real
  exercise of W2-4's generator on 55 covered criteria with `--items` below
  that count.
- **The full gate has run once, green, on a fresh clone of the merged
  tree** (R-VR-7), at the wave's verification. It covers everything this
  train merged.

## What to read first

- Plan §1 (what the tester saw), §2, §4, §5's and §7's rulings and
  verdicts, §8 (this wave) and §9.
- The Wave 1 closeout (W1-3's per-row rule) and the Wave 2 closeout (what
  `check`, the generator and the listing do with the keywords), in their
  briefs.
- On `main`:
  - chapter 5 §5.4 (`uniqueKeys` and `coverage`);
  - chapter 13's "Rules across the rows of a table";
  - `metais help schemas`;
  - ADR-003 A2, items 11 to 13 and 17.
- In the tree:
  - `examples/accessibility-conformance-report` whole: `inputs/*.schema.json`,
    `inputs/tables.md` and `inputs/conformance.md`, the guide,
    `templates/criteria/` (where the levels and the criteria lists live),
    `references/`, and both examples;
  - `docs/adr/logs/ADR-003/W1-3.md`, for R-W1-c and the WCAG level rule;
  - the tests that hold the example, among them
    `TestACRRequiredLevelsEvaluated` and `TestACRNotEvaluatedSaysWhy`.
- AGENTS.md whole, and the `verification-gate` skill for the full gate.

## What Waves 1 and 2 learned that changes this one

- **R-VR-7 still holds for items, with `gofmt`.** Items run directed
  tests only: `go test` on what they changed and the tests their task
  names, `gofmt -l` over every changed file, `go vet`, and the drift tests
  the change reaches. Items never run `gate.sh` and none is commissioned
  to. **The full gate runs once, at this wave's verification**, below.
- **The full gate will meet host starvation.** In Wave 2, the
  whole-package `-race` runs of `internal/guarantees` and
  `internal/conformance` hit `go test`'s 600 s bound at about half their
  CPU share. They took 591 s even at `c831e966`, before this train. A
  timeout is not a pass, and it is not closed by raising the timeout
  (AGENTS.md). The verification runs the wave tier when the host is as
  idle as it can see. It runs those two packages in `-run` groups if a
  whole-package run still times out, as R-W6-3 already groups the SC7
  reach. It reports each group's result and the share it got. A timeout
  that persists at full share is a finding with a mechanism to find. It
  is not waved through. Changing `gate.sh` itself is not this wave's
  work.
- **Every commit and every squash carries a real identity.** W2-6's squash
  `72c33df9` ended `Signed-off-by:  <>` (issue #302, for Ed). Set
  `user.name` and `user.email` on the command line, sign off as
  `Edward Funnekotter <3941907+efunneko@users.noreply.github.com>`, pass
  every squash body explicitly, and check the merged commit's message.
  No `Co-authored-by` line anywhere.
- **Clones may need `git config core.precomposeunicode false`**, because
  the sandbox's iconv lacks UTF-8-MAC.
- **`gh` needs `--repo kottera/metais`**, and `--head` on `gh pr create`.
- **Write each acceptance criterion in a form the proof can express.** The
  conformance harness compares an expected finding's parameters as a
  subset (ADR-011 D10 item 2), so a vector cannot prove a parameter is
  absent. One Wave 2 criterion asked for that and cost a correction.
- **The ACR already refuses** a Level A or AA WCAG row answered `Not
  Evaluated`, and any `Not Evaluated` row without remarks (W1-3, R-W1-c).
  The new keywords sit beside those rules and keep them. A row the
  coverage rule requires can still be `Not Applicable`, or `Not
  Evaluated` with remarks where the per-row rules allow it.
- **#273 derives the report's level and chapter claims from its rows.**
  Expanding the tables changes those claims in the expected trees. That
  is expected, and the pull request says which changed and why.

## Rulings — do not reopen

- **R-VR-1 to R-VR-7, R-W1-c, R-W2-a and R-W2-b** (plan §4, §5 and §7),
  and the rulings of #268 and #270 that R-VR-2 names.
- **WCAG's `coverage` is every Level A and AA criterion of WCAG 2.2**
  (plan §8). Level AAA is not covered. The list must agree with the
  level partial the package already uses. `TestACRRequiredLevelsEvaluated`
  holds the per-row rule's list to that partial; whatever lists
  `coverage`'s values is held to it the same way.
- **What the Section 508 and EN 301 549 tables must cover is the item's
  decision**, as R2's was. It records its reasoning in its pull request
  and its log entry. If the answer is a judgement about what the report
  means rather than about the schema, it parks, and the question comes up
  the tree to Ed. One example: whether a report may leave out a chapter
  the product has no part in, rather than answer it `Not Applicable`.
- **No specification text.** This wave adds no statement and no finding
  code; the keywords and their findings are on `main`. A data case and its
  expected findings are the package's own tests. Approved text is not
  edited (R-VR-4).
- **No ADR amendment.** An item that finds it needs one parks and asks.

## Scope — what belongs elsewhere

- **The close is the plan orchestrator's**, not this wave's. That covers
  closing #268 and #270 as superseded, ADR-003 A2 and ADR-004 A2 becoming
  Accepted, the status lines, the plan's status, `docs/INDEX.md` and the
  train closeout. No item of this wave edits those.
- **Ed's final UX check** (plan §6) is filed by the plan orchestrator after
  this wave completes. This wave does not file it.
- **Still running beside this wave, not this wave's:** W2-NB1
  (`01a16aa6-6764-702a-b809-d8df0783afd3`, `internal/sample` and the
  registry, under R-W2-b) and W2-NB2
  (`01a16aa6-ba87-7094-883a-7a5160cb7686`, DORA's currency pin and
  `check`'s `prefixItems` branch). Rebase onto them and keep their
  changes. If they have merged by the verification, the full gate covers
  them. If not, the verification runs anyway, and the closeout names what
  had not merged.
- **Issue #302**, the malformed sign-off, is Ed's decision. No item
  rewrites `main`'s history.
- **Plan §10's exclusions**, unchanged.

## What must not break, and how it is checked

- **Directed tests before every merge** (R-VR-7): the example's tests, the
  examples' `test` cases, `internal/conformance` where vectors are added,
  `gofmt -l`, `go vet`, and the drift tests the change reaches. After
  every rebase, run `regenerate.sh` and commit what it changes.
- **`main` stays green after every merge** (R-VR-3), `gofmt` included.
- **The ACR's earlier rules still hold.** W1-3's level rule and
  R-W1-c's remarks rule keep their tests green and unchanged in meaning.
- **Rendering stays pure and byte-identical**, and `render --sample` is
  deterministic for a given seed.
- **The wave's verification:**
  - audits every item against this brief;
  - checks plan §9's train-specific properties: every line #268 and #270
    add under `spec/`, other than generated files, is on `main`; this wave
    added none; `main` stayed green;
  - **runs the full gate once**: `gate.sh wave` on a fresh clone of the
    merged tree, with the SC7 reach in its area groups, under the
    starvation handling above.

  Its report names the commit it ran on, each step's result, and any
  group it had to split.
- **Log entries** under `docs/adr/logs/ADR-003/`, and in every other ADR's
  log whose decision the item meets, one file per address, with the
  plan-name prefix where a `W3-*` name is taken.

## Out-of-band acts of this wave

None for the wave orchestrator to file. Plan §6's final UX check comes
"after Wave 3's verification" and before the close. The plan orchestrator
files it once this wave completes, so it does not appear in this wave's
breakdown.

The conditional act "rule on a wording change to approved text" cannot
arise, since this wave adds no specification text. A question of what the
report means (above) comes up the tree as an escalation.

The release-candidate row of plan §6 is PLAN-2026-10-metais-public's.

## Closeout

Written 2026-10-24 by the wave orchestrator
(`01a16aaa-f504-7b1b-8f59-172ae56513f6`) after the wave drained. This is a
record of what landed and where every finding went. It is **not a
correctness verdict**:
- I read W3-1's report, #308 and its diff stat, and the verifier's report.
- I spot-read the merged tree: the three schemas' `uniqueKeys` and
  `coverage` (55, 104 and 148 values on `/criterion`), the test names in
  `internal/cli/example_acr_test.go`, and the squash message's trailer.
- I mutated nothing and ran no test.

The testing described below is the verifier's, attributed to it.

### 1. What landed, against what the brief asked

The wave had one item, by the approved breakdown. It is complete and
merged. The wave's base is `ee73d4e8`. Two commits of Wave 2's
non-blocking items fall inside the range: #306 (W2-NB2, `19d7f9d0`) and
#307 (W2-NB1, `fdab8156`). W3-1 rebased onto #307. R-W3-a's plan record,
#309 (`b22afa16`), followed.

| Item | Candidate | Work Order | PR | Merge | Landed |
|---|---|---|---|---|---|
| W3-1 | R8 | `01a16aad-8d4e-72ca-8f26-3843957d1e73` | #308 | `149923f7` | See below. |

What W3-1 landed:

- **`uniqueKeys` and `coverage`.** Each of the three criteria tables
  carries `uniqueKeys: ["/criterion"]` and a `coverage` keyed on
  `/criterion`:
  - WCAG covers the 55 Level A and AA criteria of `wcag-level.liquid`;
  - Section 508 covers 104 criteria, every one its enum allows;
  - EN 301 549 covers 148, every one its enum allows.
- **`wcag_only` grows from 7 rows to 56.** It answers the 55 Level A and
  AA criteria and its one AAA row. Its expected tree was regenerated by
  `metais test --update`, and only its rows and summary counts changed.
  Its level claim stays "Levels A, AA and AAA", and it makes no chapter
  claims. `international` was already complete and is unchanged.
- **The tester's table is refused.** `TestACRTesterTableRefused` holds
  that: one `input.unique_keys` at `/1/criterion` and 54 `input.coverage`,
  with `missing` exactly the other Level A and AA criteria. The pull
  request shows it red on `19d7f9d0`.
- **New tests:** `TestACRTablesCovered`, `TestACRCriterionAnsweredOnce`
  and `TestACRSampleRendered`.
- **The package's docs.** `inputs/tables.md` and the guide say what is
  refused.
- **Its log entry** is `docs/adr/logs/ADR-003/W3-1.md`.

It added no specification text, no finding code and no ADR text. The
squash message ends in a real `Signed-off-by:` and carries no
`Co-authored-by`.

**Divergence from the ask: W3-1.3.** The brief left the Section 508 and EN
301 549 coverage to the item. It also said that a judgement about what the
report means parks to Ed, and named this very question as its example:
whether a report may leave out a chapter the product has no part in.
W3-1 decided it without parking. It read the question as a schema
decision, because `inputs/tables.md` already told authors to list every
criterion. The verifier blocked on that, and I escalated it.

Ed ruled **R-W3-a** (2026-10-24, his option A, recorded in plan §8 by
#309). His ruling keeps the coverage as #308 merged it. A chapter or
clause the product has no part in is listed and answered `Not
Applicable`, row by row. His reasons:

1. "It is the only option that can be changed after 1.0 without breaking
   anyone. Loosening A to all-or-nothing per chapter later only admits
   more reports, while tightening B or C to A later would refuse reports
   that passed before. 1.0 must stand for a long time, so the strict form
   is locked in."
2. "It is what the package already told authors. The international
   example's section508.csv already answers chapter 4 row by row ('The
   product is software and has no hardware'). Nothing is left out without
   saying so, which SC1 rests on."

He also put one thing out of scope, and it was not filed: a template that
collapses a chapter whose rows are all `Not Applicable` with one remark.
He called report length a rendering question, not one about the input
contract.

**Rulings made during this wave:** R-W3-a, above, which is Ed's. I made
none of my own. No child asked a question.

### 2. What the child flagged, declined or discovered

- **Duplicate repeats also fail `maxItems`.** The Section 508 and EN
  301 549 tables' `maxItems` equals their criteria count. So a criterion
  repeated in a complete table also reports `input.max_items`, and
  `TestACRCriterionAnsweredOnce` expects it.
- **Adapted tests.** `TestACRClaimsFromRows` needs tables that leave out a
  level or a chapter, which coverage now refuses. It now renders a copy of
  the package with `coverage` removed (`acrUncovered`).
  `TestACRRowsRefused` and `TestACRNotEvaluatedSaysWhy` now place their
  rows in complete tables. W3-1 reports that no assertion was dropped.
- **A failure that predates the wave.** `render --sample --optional mixed
  --seed 7` fails `schema.sample_invalid` on section508. This became
  W3-NB1 (§4).

### 3. What carries forward

- **The close is the plan orchestrator's**, and so is Ed's final UX check
  (plan §6). This wave filed neither. The tester's table is now refused
  with one finding per repeat and one per missing criterion. That is what
  the UX check re-runs.
- **The full gate ran once, green, on `149923f7`**, the merged tree,
  which includes W2-NB1 and W2-NB2. Every commit on `main` after it is
  outside that run. That covers #309, this closeout, and whatever W3-NB1
  and W3-NB2 merge.
- **What I would do differently:** I would put a question the brief
  reserves for Ed in the breakdown's park, so the ruling comes before the
  item runs. "Park if it is a judgement of meaning" let the item judge
  that it was not one. Asking up front would have saved the verifier's
  blocking round trip.
- **The `recall` verb is not permitted** to this worker. So a filed child
  could not be folded into a sibling after release, and W3-NB1 and W3-NB2
  are two children where one might have done (§4).

### 4. What the verification found, and where every finding went

One `wave-verification`, `01a16abd-eab7-7be2-8067-521e7233f26d`. It
returned the one item id I sent, at `depth: mutated`.

- **The full gate.** It ran `gate.sh wave` once on a fresh clone of
  `149923f7` (`METAIS_GATE_P` default, p=5 of 16 cores, go1.26.2
  darwin/arm64), and it was **GREEN**:
  - all 56 packages passed under `-race`;
  - `internal/conformance` took 454 s at 3.45 cores and
    `internal/guarantees` 366 s, both under the 600 s bound. No `-run`
    groups were needed, so the brief's starvation handling was not
    exercised;
  - every SC7 reach area group passed, and none was split;
  - build, vet, tidy, gofmt, static, cross and specdata were green.
- **Mutations.** It ran 11. Ten went red. It found these mutations went
  red:
  - Level AAA 1.2.6 added to WCAG's coverage;
  - section508's `uniqueKeys` re-keyed to `/conformance`;
  - 603.3 dropped from section508's coverage;
  - EN's coverage deleted;
  - 4.1.3 dropped from WCAG's coverage, together with `uniqueKeys`
    removed;
  - 4.1.3 dropped from WCAG's coverage alone;
  - 4.1.3 moved from AA to AAA in the level partial, the direction the
    producer did not mutate;
  - WCAG's coverage deleted;
  - EN's `uniqueKeys` removed.

  Removing WCAG's `uniqueKeys` alone left `TestACRSampleRendered` green.
  That is the second non-blocking finding.
- **Plan §9.** No line under `spec/` came from W3-1. The `spec/` changes
  in the range are W2-NB1's registry and vectors. Its "`main` stayed
  green" check was made on `149923f7`, the commit the gate ran on. The
  intermediate W2-NB merges were not run separately.
- **Coverage audit.** It confirmed #308 is on `main` and touches only
  what its criteria name. It notes that the completeness of my item list
  is trusted, not verified.

| Finding | Kind | Where it went |
|---|---|---|
| W3-1 decided the Section 508 and EN 301 549 coverage, a question the brief reserves for Ed (W3-1.3) | blocking | **Closed by Ed's ruling R-W3-a, with no fix child.** The code stands as merged. |
| `render --sample --optional mixed --seed 7` fails `schema.sample_invalid` on section508, at `/allOf/0/if`. It fails identically at `ee73d4e8` and `fdab8156`. | non-blocking, cross-item | W3-NB1 `01a16b85-e756-7061-8dca-32b43ff48f6e` |
| `TestACRSampleRendered` cannot see `uniqueKeys`: at `--items` ≤ 55 the generator emits exactly the covered values | non-blocking, cross-item | W3-NB2 `01a16b86-4805-7061-966c-3e5123fad650` |

**The wave completed without waiting for W3-NB1 and W3-NB2, by design.**
They were filed after the wave drained, when no wait of mine remained,
and they hold nothing up. They are not re-verified in this wave. Each is
its own child: their fixes lie in different places (the generator or the
package, and a test's strength), and W3-NB2's task says to rebase onto
W3-NB1 if both touch `internal/cli/example_acr_test.go`. No finding was
prose-only, so no janitorial issue was filed. Every finding is filed.

### 5. Out-of-band acts

This wave filed none, as the brief said. Ed's ruling R-W3-a came through
an escalation of this Work Order, not an off-tera act. Plan §6's final UX
check is filed by the plan orchestrator after this wave.

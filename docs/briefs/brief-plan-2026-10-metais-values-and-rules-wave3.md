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

*Not yet written. The wave orchestrator replaces this line with the
wave's closeout.*

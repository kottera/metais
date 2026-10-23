# Brief — PLAN-2026-10-metais-values-and-rules, Wave 1: the amendments, then the cores

**Plan:** [PLAN-2026-10-metais-values-and-rules](../plans/PLAN-2026-10-metais-values-and-rules.md), §5
**Feature:** `metais-values-and-rules`. Work Orders are titled
`metais-values-and-rules [W1-<item>]: <action>`; pull requests and their
squash subjects are `<type>(metais-values-and-rules): [W1-<item>] <change>`
(AGENTS.md, Git and pull requests).
**Repository:** `kottera/metais`, branch `main`. Every Work Order of this
wave gets `repo: kottera/metais` and `branch: main`. The train writes this
repository only.
**Gated on:** Wave 0, the plan, merged as #276. The gate has cleared.

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

Both design amendments are on `main`, and each feature's core is built,
specified and proved by vectors. When it is done:

- **ADR-003 A2 and ADR-004 A2 are on `main`** as written at `b80ed115`
  (#268) and `b0cbb230` (#270), with the status-line pointers of ADR-002,
  ADR-003, ADR-004, ADR-006, ADR-007 and ADR-008 and the status-line
  changes plan §4 describes ("The amendments item's status lines"). This
  lands first, as one docs pull request with no specification text and no
  code; every other item of the wave depends on it.
- **MTS-FND-003's deduplication is fixed**: two findings are duplicates only
  when their `schema.entry` and `schema.pointer` are equal too.
- **The ACR refuses a Level A or AA WCAG criterion answered `Not
  Evaluated`**, as a per-row `if`/`then` in each WCAG table's item schema,
  with the Section 508 and EN 301 549 tables decided (plan §4, last
  drafting decision).
- **`number` and `amount`** are in the dialect with Appendix C's ISO 4217
  table, and **`duration`** is in the dialect, each as chapter 8 states it,
  with their finding codes and vectors.
- **`uniqueKeys` and then `coverage`** are in the dialect, for inputs and
  outputs, as chapter 5 §5.4 states them, evaluated by `validate` and the
  render's validation, with partial validation as MTS-SUP-029 says, and
  their finding codes and vectors.
- **Every slice of plan §3's table assigned to these candidates is on
  `main` verbatim**, every shared sentence that opened in the wave has
  closed, and `main` was green after every merge.

## What to read first

- Plan §1 to §5 whole, and §9. §2's three train rules (main never goes red,
  approved text verbatim, a shared sentence lands with whichever merges
  first and closes inside the wave) and §3's slice table bind every item.
  §5 holds the candidates; derive the items from them, from the slice table
  and from the code.
- The design itself, at the accepted commits and never at a later push:
  `git show b80ed115` (#268, branch `docs/metais-v1-cross-record-rules`)
  and `git show b0cbb230` (#270, branch `docs/metais-v1-value-filters`).
  In particular:
  - ADR-003 A2: rulings 2 to 7, items 8, 11 to 14, 16, 18 and 19, and its
    "Conformance" section, which lists the vectors R3 and R4 owe;
  - ADR-004 A2: rulings 2 to 10 and 12, decision items 1 to 4 and 7.
- PLAN-2026-10-metais-v1 §2, for the rules it carries into this train
  (statements with code, codes registered by their emitter, shared seams
  named and statement numbers allotted at breakdown — here the numbers are
  already allotted by the design, so use the design's).
- AGENTS.md whole, and the `verification-gate` skill: the scoped tier,
  `regenerate.sh` after every rebase, the generated files' merge drivers.
- In the tree: `internal/finding` (`registry_template.go` and the schema
  findings), `internal/schema` and the JSON Schema wiring, the filter
  implementations beside `date`, `internal/conformance` and
  `internal/guarantees`' `TestCoverage` and `TestFindingCodesCovered`, the
  ACR example under `examples/`, and `docs/adr/logs/ADR-003/` and
  `ADR-004/`.

## What the state of `main` means for this wave

There is no previous wave of this train; these are the facts as of
2026-10-23 that the plan could not state.

- **No ADR file has changed on `main` since `c831e966`**, the commit both
  design commits sit on; only the plan and `docs/INDEX.md` have. W1-1 can
  take the files' hunks cleanly — but take hunks, not files, in case
  something lands before it.
- **Other work is in flight on `main`** and will merge under this wave:
  PLAN-2026-10-metais-public's Wave 2 (the site, the release tooling, the
  UX install scenarios), and several standalone fixes, among them #273,
  which derives the ACR example's claims from its rows, and #274 and
  #275, which change `schema --fields`' and the supplier's output. R2
  touches the same example as #273. Rebase onto them and keep their
  changes; do not absorb or redo them. Regenerate after every rebase.
- **Two ADR log addresses are taken** in `docs/adr/logs/ADR-003/` (`W7-1.md`,
  `W7-4.md`) and one in `ADR-004/` (`W7-4.md`). They do not collide with
  this wave's `W1-*` addresses as things stand; if one does, prefix this
  plan's name (AGENTS.md).
- **`gh` in a dispatch needs `--repo kottera/metais`** (and `--head` on
  `gh pr create`): a dispatch's clone reaches GitHub through a local
  proxy remote that `gh` does not recognise as the repository.
- **The sign-off identity.** A dispatch host has no `user.email`, and
  one built from its host name must not reach a commit (ADR-013 D4 item
  3). Every commit signs off as `Edward Funnekotter
  <3941907+efunneko@users.noreply.github.com>`, set on the command line,
  as PLAN-2026-10-metais-public's R-W1-5 says. No commit, pull request or
  squash body carries a `Co-authored-by` line: CI fails it, and AGENTS.md
  forbids it. Pass every squash body explicitly, as AGENTS.md says.

## Rulings — do not reopen

- **R-VR-1 to R-VR-6** (plan §4) and every ruling and drafting decision of
  #268 and #270 that R-VR-2 names. In particular:
  - **The design pull requests never merge.** Each item carries its own
    slice of the approved text with the code and vectors that make it
    true. At this plan's close #268 and #270 are closed as superseded; no
    item closes them or pushes to their branches.
  - **Approved text lands verbatim** from `b80ed115` and `b0cbb230`. A
    sentence that is wrong, or that the code cannot make true as written,
    is not edited: the item parks with the sentence, why, and the wording
    it proposes. Regenerated files are regenerated, never copied.
  - **The spike comes first in R3** (R-VR-6): at most half a day on
    `santhosh-tekuri/jsonschema/v6`'s `Compiler.RegisterVocabulary` with
    `Compiler.AssertVocabs`. If it fails, or `AssertVocabs` changes how any
    1.0 vector validates, R3 stops and parks with the spike's evidence
    before building on A2 item 19's fallback. That question goes to Ed;
    bring it up the tree rather than deciding it.
- **Order inside the wave** (plan §3): W1-1 merges first and everything
  else depends on it; R4 depends on R3, and both share the vocabulary R3
  registers. R1, R2, V1 and V2 are independent of each other and of the
  keywords. V1 and V2 share chapter 8's heading renumbering and the
  sentences plan §3 assigns to "whichever merges first"; if the numbering
  gap a single one leaves trips a tripwire, run V2 after V1.
- **Inside a shared sentence's window the tool stays strict.** A keyword
  or filter the specification names but the tool does not build yet is
  refused as unknown — never accepted and ignored. Until R4 merges,
  `coverage` is an unknown keyword.
- **Red first** for R1 (the 54 `min_contains` findings #268 measured as
  one) and R2 (a Level A or AA `Not Evaluated` row validates on `main`).
  Each pull request shows its proof failing on `main` before the change.
- **R2's Section 508 and EN 301 549 decision is R2's**, recorded in its
  pull request and its log entry, unless it turns on what the report
  means rather than on the schema; then it parks for Ed (plan §4).
- **No ADR amendment beyond ADR-003 A2 and ADR-004 A2.** An item that
  finds it needs one parks and asks.
- **V1's arithmetic is fixed by the design**: the shortest round-trip
  decimal, rounded half away from zero on `big.Rat`, never
  `FormatFloat(f, 'f', n, 64)`; the currency table is a Go table held
  equal to Appendix C by a test that parses the appendix.

## Scope — what belongs elsewhere

- **Wave 2:** the `money` alternative on `template.unknown_filter`
  (MTS-FLT-023), MTS-FLT-014's new wording and chapter 13 §13.9 and
  `metais help templates`' teaching (V3); the examples adopting the
  filters (V4); `check`'s three findings and `schema.key_not_declared`
  (R5); the sample generator's handling of the keywords (R6); `schema
  --fields`, `metais.schema/1`'s `rules`, `inspect`, `metais help
  schemas`, the hints in words, the skill, and chapter 13's "Rules across
  the rows of a table" (R7). This wave's items write no help topic and no
  skill text beyond what regeneration changes, and no example adopts a
  filter or a keyword — R2's per-row rule is the one example change here.
- **Wave 3:** the ACR adopting `uniqueKeys` and `coverage`, its expanded
  tables and the tester's table as a refused case (R8); the close.
- **Plan §10:** locales, presets, rounding modes, symbols, negative or
  fractional durations, `date` offsets; `coverage` from `enum` or a file;
  rules between inputs.

## What must not break, and how it is checked

- **`main` never goes red** (R-VR-3). Each item's slice merges with the
  code and vectors that make it true, so `TestCoverage` and
  `TestFindingCodesCovered` stay green after every merge. The scoped tier
  (`gate.sh`) before every merge; the wave tier, on a fresh clone of the
  merged tree, at the wave's verification. An item that changes `go.mod`,
  `go.sum` or a shared seam AGENTS.md names runs the wave tier before it
  merges. R3's vocabulary registration may touch the schema compiler's
  setup; if it adds or bumps a module, that is a shared seam.
- **Every 1.0 vector validates as before**, except the MTS-FND-003
  expectations R1 changes, each named in R1's pull request, and that
  `AssertVocabs` must not change (R-VR-6).
- **Keyed evaluation is linear in the items** (MTS-DAT-018), proved at the
  smallest size that can fail it, with CPU time where it is a bound
  (AGENTS.md).
- **Rendering stays pure and byte-identical**: the filters' output is the
  same on every platform; no locale reaches it.
- **The wave's verification checks plan §9's two train-specific
  properties**: every line a merged slice added under `spec/` is a line of
  `b80ed115` or `b0cbb230`, and every shared sentence opened in the wave is
  closed with the tool doing what it says; and no merge left
  `TestCoverage` or `TestFindingCodesCovered` red.
- **Log entries** under `docs/adr/logs/ADR-003/` and `ADR-004/`, and in
  every other ADR's log whose decision the item meets, one file per
  address.

## Out-of-band acts of this wave

Plan §6 has two rows that can arise in this wave. Neither is filed up
front: each exists only if its condition happens, and then it reaches Ed
as an escalation up the tree — the item parks, the wave orchestrator
brings it to the plan orchestrator, and the plan orchestrator asks Ed.

| Act | When | Before | Notes |
|---|---|---|---|
| Rule on the vocabulary spike | only if R-VR-6's spike fails, or `AssertVocabs` changes a 1.0 vector | R3 builds on the fallback | A2 item 19 names the fallback. R3 parks with the spike's evidence. |
| Rule on a wording change to approved text | only if a slice needs one | that slice's merge | R-VR-4. The item parks with the sentence, why it cannot be made true as written, and the wording it proposes. |

The plan's other rows are not this wave's: adding this train's close to
T-A3's `depends_on` belongs to whoever filed the plan orchestrator; the
final UX check follows Wave 3's verification; a release candidate holding
this train is PLAN-2026-10-metais-public's.

## Closeout

Written 2026-10-23 by the wave orchestrator
(`01a16964-1c6e-7e86-80f4-b45bbc82b4fc`) after the wave drained. This is a
record of what landed and where every finding went. It is **not a
correctness verdict**. I read the merge commits, the children's reports and
the verifier's report, and spot-read the code where an ask named a mechanism.
I did not mutate or test anything myself. The testing described below is the
verifier's, attributed to it.

### 1. What landed, against what the brief asked

Seven items, all complete, all merged to `main`. They are listed in merge
order below `e5a18988`. Commits of other trains are interleaved: #278,
#266, #273, #274 and #285.

| Item | Work Order | PR | Merge | Landed |
|---|---|---|---|---|
| W1-1, the amendments | `01a16968-c0e7-70fa-951b-a0bf7d2a2eab` | #279 | `f50ab6cb` | ADR-003 A2 and ADR-004 A2, the (A2) passages and the status-line pointers of ADR-002, 006, 007 and 008, with plan §4's acceptance lines. Docs only. |
| W1-3, the ACR's per-row rule (R2) | `01a1696b-3ad9-74f5-a214-22c095128b7a` | #281 | `9c0ebe5d` | The WCAG table refuses `Not Evaluated` for its 55 Level A and AA criteria. Under Ed's ruling (c), a `Not Evaluated` row in every table (WCAG, Section 508 and EN 301 549) must also carry remarks. |
| W1-6, `uniqueKeys` (R3) | `01a16969-48d1-774f-a083-5323bae02f3d` | #282 | `ed7e46d9` | The spike succeeded: `RegisterVocabulary` with `AssertVocabs` changed no 1.0 vector, so A2 item 19's fallback was not built and no module was added. Grammar, evaluation through every applicator, linear keyed evaluation, `input.`/`output.unique_keys`, and partial validation, with its slice. |
| W1-5, `duration` (V2) | `01a1696a-91ce-7e0d-beff-06b51681e6db` | #280 | `0edb86fb` | §8.1.13, both codes, and parse-time judging. It merged first, so it carried the shared sentences: MTS-MAN-018, the MTS-TPL-063 row and the 8.1.14 heading. |
| W1-2, MTS-FND-003's key (R1) | `01a1696a-ee29-7cb9-8f04-7a1fdcb605bc` | #284 | `dac9aa3e` | The duplicate key compares `schema.entry` and `schema.pointer`. Red first on the collapsed `min_contains` findings, with `fnd/duplicate-key`. |
| W1-4, `number` and `amount` (V1) | `01a1696a-2126-79db-a346-93dda2753719` | #283 | `5549ee65` | §8.1.10 to §8.1.12, Appendix C as a Go table held by `TestCurrencyTable`, both codes, and the charge vector. It closed W1-5's shared sentences. |
| W1-7, `coverage` (R4) | `01a16969-ad78-7514-a9de-0f1365a1b354` | #286 | `b18ab317` | Grammar, one finding per missing value at the array, and suppression and counting in partial validation. It closed every sentence W1-6 opened. |

What I read in the merged code: `internal/schema/compiler.go` registers
the rules vocabulary and calls `AssertVocabs`. `internal/template/filters/number.go`
rounds on `big.Rat` from `FormatFloat(x, 'g', -1, 64)`. `TestCurrencyTable`
exists. The log files exist under ADR-002, 003, 004, 006, 009 and 010.
Those names follow the plan-prefix rule wherever a `W1-*` name was taken.

**Divergences from the ask:**

- **W1-3's scope grew under Ed's ruling (c).** It added a second rule: a
  `Not Evaluated` row needs remarks in every table, WCAG AAA included. So
  W1-3.2's "a Level AAA row answered `Not Evaluated` is still accepted"
  holds only when that row carries remarks. The ruling, not the item, made
  that choice. It is recorded in #281 and `docs/adr/logs/ADR-003/W1-3.md`.
- **`main` did not stay green after every merge.** See §4: `gofmt` was red
  from `5549ee65` until `9f83587c`.
- **The scoped gate was not the pre-merge check for four items.** W1-2,
  W1-4, W1-5 and W1-7 merged on directed tests under the operator's
  guidance (below). W1-1, W1-3 and W1-6 report a green scoped gate.

**Rulings and instructions made during this wave:**

- **Ed's ruling (c), 2026-10-23, to W1-3**: the Section 508 and EN 301 549
  question reached him through the tree, and he answered it.
- **R-W1-a (wave orchestrator, to W1-4 and then W1-2):** do not merge over a
  scoped gate starved on a shared host. Wait for the load to fall below
  the cores, `gate.sh --resume`, and merge at full share. It followed from
  the `verification-gate` skill's ladder step 1 and R-VR-3. It was
  **superseded** the same day.
- **Operator guidance message 1 (Ed's standing instruction, 2026-10-23),
  adopted:** metais work items do not run `gate.sh` or the wave tier. Each
  runs directed tests and merges, and the full gate runs once, on the
  merged tree, when all the work is complete.

### 2. What children flagged, declined or discovered

- **Host saturation.** Four Wave 1 gates on one host starved each other at
  0.4 to 0.6 of their share. `internal/conformance`, `internal/guarantees`
  and `internal/cli` hit `go test`'s ten-minute bound in every run. That is
  what led to the operator's instruction.
- **A lead, unsettled:** W1-2 saw race-enabled `internal/conformance` time
  out on `main` `a9d58851` as well. That was measured only while starved,
  never at full share. W1-2 recorded it as a suspected observation on the
  package. The verifier's run of the same package took 470 s.
- **Help rows.** W1-4 and W1-5 each added a `metais help templates` row for
  their filters, because `TestTemplatesTopicFilters` demands one per dialect
  filter. That contradicts the brief's line "This wave's items write no help
  topic", but leaving the rows out would have turned `main` red.
- **Readings the producers took:**
  - W1-5: an integer beyond 2⁵³ − 1 is a float to the template, so it is
    refused with reason `kind`, not `range`.
  - W1-6: `schema.invalid`'s `keyword` parameter now also covers this
    dialect's own keywords, and its registry entry cites MTS-DAT-015 (and,
    after W1-7, MTS-DAT-016), consistent with A2 item 14.
  - W1-7: `missing` holds the value's canonical JSON text as a string, as
    `const`'s `expected` does.
  - W1-7: the hint names only the missing value; the rule in words is R7's.

### 3. What carries to the next wave

- **Wave 2's V3 owns the help rows for `number`, `amount` and `duration`.**
  They are there now, written minimally. V3 rewrites them as teaching.
  Wave 2's brief should state that the rows exist, and why, so V3 does not
  treat them as someone else's work.
- **R7 owns the findings' hints in words.** `input.coverage`'s hint names
  only the missing value today.
- **The full gate has not run on this wave's merged tree.** Under the
  operator's instruction it runs once, when all the work is complete. The
  verifier's directed run was green but for the `gofmt` red, which has since
  been fixed on `main` (§4).
- **What I would do differently.** Seven gates on one shared host at once
  produced the starvation. A wave this wide needs its pre-merge checks
  staggered, or kept to directed tests from the start, as the operator
  then ruled. Wave 2's briefs and items should carry the directed-tests
  rule from the outset.

### 4. What the verification found, and where every finding went

One `wave-verification`, `01a16a25-ad4e-730d-823e-468a05a081fa`. It
returned all seven ids I sent: W1-1 `inspected`, and W1-2 to W1-7
`mutated`.

- **What it did:**
  - ran 39 mutations across W1-2 to W1-7, all of which went red;
  - spot-checked the producers' claimed-red rows (W1-6's quadratic, W1-4's
    `FormatFloat`, W1-2's pointer), which went red as claimed;
  - checked line by line that every line the wave added under `spec/*.md`,
    generated catalog rows aside, is a line of `b80ed115` or `b0cbb230`;
  - confirmed every shared sentence closed, each with a vector citing it;
  - ran `TestCoverage` and `TestFindingCodesCovered` at each of the seven
    merge commits, all green.
- **How it ran its tests:** by the operator's instruction it did **not**
  run `gate.sh` or the wave tier. It ran build, vet, `gofmt`, tidy, the
  static and six-target builds and the specdata drift, plus `go test -race`
  over `internal/schema`, `schemafail`, `finding`, `specdata`, `template`,
  `loader`, `conformance`, `guarantees` and `cli` on `b18ab317`.
- **Where it found nothing:** W1-1. The verifier read it rather than
  mutating it.

**Blocking: one finding, no fix child.**

- **W1-4 left `internal/cli/inspect.go` failing `gofmt`.** Its PR added
  `loader.CodeTemplateAmountCode: true,` without realigning the map. So
  `main` failed the scoped tier's `gofmt` step from `5549ee65`, through
  W1-7's `b18ab317`, which merged over it. The verifier quoted the brief's
  "**`main` never goes red** (R-VR-3)".
- **No fix child was filed** because the defect is already repaired.
  `9f83587c` (#275, outside this wave) realigned the line, and `gofmt -l .`
  over `main` at `0dac5ac8` prints nothing. I checked that myself. The
  breach of "`main` was green after every merge" happened and is recorded
  here. It cannot be undone.

**Non-blocking findings.** Three are combined in one child, and
`01a16a38-9c5f-76d9-aa17-2506bc5e4bdc` (W1-NB1) carries each with the
verifier's evidence verbatim. They share it because each is a small vector
addition that the conformance suite verifies at one cost, and they are
severable. **The wave completed without waiting for this child, by design.**

| Finding | From | Where it went |
|---|---|---|
| No vector pins `schema.entry`'s part of MTS-FND-003's key (dropping it reddens only `TestReportDuplicates`) | W1-2 | W1-NB1 `01a16a38-9c5f-76d9-aa17-2506bc5e4bdc` |
| No test refuses `P1WT1H` (a week with a time part is accepted under mutation) | W1-5 | W1-NB1 `01a16a38-9c5f-76d9-aa17-2506bc5e4bdc` |
| No test tells a keyless item from `null` under `coverage` | W1-7 | W1-NB1 `01a16a38-9c5f-76d9-aa17-2506bc5e4bdc` |
| W1-3.2's AAA row accepted only with remarks | W1-3 | **Not filed.** It is Ed's ruling (c) applied as he gave it, not a defect, and is recorded in §1. |
| `schema.invalid` cites MTS-DAT-015 and covers this dialect's own keyword | W1-6 | **Not filed.** It is consistent with ADR-003 A2 item 14 (a malformed keyword is `schema.invalid`, naming it), and the citation is the statement that requires the code. Recorded in §2. |
| Help rows added despite the brief's "no help topic" line | W1-4, W1-5 | **Not filed.** The two brief lines conflict, and a test forces the rows. Wave 2's V3 owns them (§3). |

The verifier also recorded two **cross-item** checks. Neither is a defect,
so neither is filed:

- W1-6's claim that `internal/schema/compiler.go` is the one call site of
  `jsonschema.NewCompiler` still holds on the merged tree, and the manifest
  schema's compile gets the vocabulary too.
- MTS-FND-003's new key composes with the `uniqueKeys`, `coverage` and ACR
  findings, and the edits W1-4 and W1-5 share, and W1-6 and W1-7 share,
  show no drift.

No janitorial issue was filed: no finding was prose-only.

### 5. Out-of-band acts

None was filed, and none was needed. The brief's two conditional acts did
not arise as filed acts:

- **The vocabulary-spike ruling:** the spike succeeded.
- **A wording change to approved text:** no item parked for one. Every
  slice landed verbatim, as the verifier checked line by line.

The one decision that went to Ed was a judgement on the ACR's tables, not
an out-of-band act. It reached him as W1-3's park and returned as ruling
(c).

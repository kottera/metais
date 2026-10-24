# PLAN-2026-10: METAIS values and rules — the value-formatting filters and the cross-record rules, in v1.0

**Author:** drafted 2026-10-23 for Ed Funnekotter, from the two design pull
requests he accepted that day
**Date:** 2026-10-23
**Feature:** `metais-values-and-rules`. Every brief, Work Order and pull
request of this train inherits it verbatim: `metais-values-and-rules
[W1-2]: <action>` for a Work Order, and `<type>(metais-values-and-rules):
[W1-2] <change>` for a pull request and its squash subject, by the naming
rules AGENTS.md adopts.
**Status:** Ratified 2026-10-23 by Ed Funnekotter, with the rulings of §4.
The drafting decisions of §4 are open to his override.
**For PRD:** [PRD-001](../prd/PRD-001-metais-v1.md) R3 to R5 (the data
specification), R7 (the template environment), R13 (authoring) and R17 (the
golden paths); SC1, whose example must not accept a contradictory report;
and SC8, every error in one run.
**For ADR:** [ADR-003](../adr/ADR-003-the-data-specification.md) Amendment
A2, rules across the items of an array (`uniqueKeys` and `coverage`), which
also amends ADR-002, ADR-006, ADR-007 and ADR-008; and
[ADR-004](../adr/ADR-004-the-template-environment.md) Amendment A2, the
value-formatting filters (`number`, `amount` and `duration`). Both are
drafted in the design pull requests and land on `main` as this train's
first item (§5).
**Design:** [#268](https://github.com/kottera/metais/pull/268) (the
cross-record rules) and [#270](https://github.com/kottera/metais/pull/270)
(the value-formatting filters), as written, at the commits Ed accepted:
`b80ed115` on `docs/metais-v1-cross-record-rules` and `b0cbb230` on
`docs/metais-v1-value-filters`. This plan cites them and never copies them.
**Repository:** `kottera/metais`, branch `main`. The train writes this
repository only.
**Depends on:** nothing; Wave 1 can start as soon as this plan merges.
**Blocks:** the v1.0 release, T-A3 (Work Order
`01a15ac4-4d4e-775e-9989-615255b813b7`, PLAN-2026-10-metais-v1 §6). T-A3
depends on this train's close (R-VR-1, §6).

## 1. Purpose

On 2026-10-23 an external tester had claude.ai's Claude run two of the
example packages with the `metais` binary. Both packages did what the
specification says, and both produced documents a reader would misread:

- **`dora-incident-report`** wrote every value as the data holds it. An
  amount came out as `48920000` with no currency, though an input names
  the currency. Timestamps stayed raw RFC 3339 text. A duration of
  `00:07:25`, which means days, hours and minutes, was read as hours,
  minutes and seconds. No filter wrote an amount or a duration for a
  reader.
- **`accessibility-conformance-report`** accepted a WCAG table that
  answered criterion 1.1.1 twice, once `Supports` and once `Does Not
  Support`, and left out the other Level A and AA criteria. It validated
  with no finding and rendered a report that looked complete. JSON Schema
  2020-12 judges each row alone, so no keyword could say "no two rows
  answer one criterion" or "every required criterion has a row".

Ed ruled that both fixes ship **in** v1.0, not in a later minor, because
1.0 must stand for a long time. A minor could add each, but the first
minor would then be the first to build the table of what each minor
added, and the example SC1 rests on would accept a contradictory report
until then. The two designs are #270 and #268; this plan builds them.

**Done means:**

- `number`, `amount` and `duration` are in the dialect as chapter 8
  §8.1.10 to §8.1.14 state them, with the ISO 4217 table of Appendix C, and
  `metais help templates` teaches them example first.
- `uniqueKeys` and `coverage` are in the dialect as chapter 5 §5.4 states
  them, for inputs and outputs. `check` and the sample generator honour
  them. `schema --fields`, `inspect`, `metais help schemas` and the skill
  show each rule in words before it can fail.
- The MTS-FND-003 deduplication fix is in, so two rules that fail alike at
  one place are two findings.
- The examples use both: DORA writes its dates, amounts, counts,
  percentages and durations through the filters; the status widget and the
  ACR write their dates through `date`; the ACR refuses a criterion
  answered twice, a Level A or AA criterion missing, and a Level A or AA
  criterion `Not Evaluated`.
- Every statement of #268 and #270 is on `main` in the words Ed accepted,
  each with the vectors that prove it, and `main` was green after every
  merge.
- #268 and #270 are closed as superseded, ADR-003 A2 and ADR-004 A2 are
  Accepted, and `docs/INDEX.md` says so.
- Ed's final UX check (§6) passes.

## 2. Execution model

A plan orchestrator runs this train, as Ed asked.
PLAN-2026-10-metais-v1 §2 applies whole, with its harness policy:

- waves run one at a time;
- each wave's brief is written late by the plan orchestrator, as
  `docs/briefs/brief-plan-2026-10-metais-values-and-rules-waveN.md`, and
  carries goals, constraints and rulings, never a breakdown;
- `wave-orchestrator-derive` derives each wave's items from its brief, from
  §5 to §7's candidates and from the code, and assigns each its address;
- the specification is written with the code, and a finding code is
  registered by the item whose code emits it;
- shared seams are named at breakdown, and items that touch one run in
  sequence with `depends_on`.

The statement numbers and finding codes are already allotted, by the
design. The train adds no ADR amendment beyond ADR-003 A2 and ADR-004 A2:
an item that finds it needs one parks and asks.

Three rules this train adds:

- **`main` never goes red** (R-VR-3). The design pull requests do not merge:
  merged alone, each turns `internal/guarantees`' `TestCoverage` red, and
  #270 also turns `TestFindingCodesCovered` red, until vectors exist.
  Instead, each implementation item carries **its own slice** of the
  approved text to `main`, with the code and the vectors that make it
  true, green: its statements, its finding codes (registered in
  `internal/finding`, so the catalog and the registry regenerate), its
  appendix where it has one, and `regenerate.sh`'s output. §3 allots the
  slices.
- **The approved text lands verbatim** (R-VR-4). A slice copies the words
  of `b80ed115` or `b0cbb230`, hunk by hunk, never from a later push to
  either branch. An item that finds a sentence wrong, or one that its
  code cannot make true as written, parks and brings the wording to Ed. It
  never edits approved text on its own judgement. Regenerated files
  (`spec/statements.json`, `spec/findings/registry.json`, the code catalog,
  the help goldens) are regenerated, never copied.
- **A sentence two items both need lands with whichever merges first, and
  the other item closes it in the same wave.** Several approved sentences
  name both keywords or several filters at once. Each lands whole, never
  cut to one item's half. So for a short time `main`'s specification
  states a behaviour one item ahead of the tool. That window opens and
  closes inside one wave, and the wave's verification confirms it closed.
  Inside the window **the tool stays strict**: a keyword or filter not yet
  built is refused as unknown, never accepted and ignored.

**Each item also carries these:**

- **Red first** where it fixes a defect `main` shows: the deduplication
  collapse (R1), the ACR accepting `Not Evaluated` for a Level A or AA
  criterion (R2), and the tester's table accepted (R8). Each shows its
  proof failing on `main` before the change.
- **Log entries** go to `docs/adr/logs/ADR-003/` or
  `docs/adr/logs/ADR-004/`, and to the log of every other ADR whose
  decision the item meets (AGENTS.md, "Writing documentation"). An earlier
  train's file already holds some addresses (`W7-1.md`, `W7-4.md`), so an
  item whose address one holds prefixes this plan's name.
- **It follows AGENTS.md**, its scoped gate included.

## 3. Overview and sequencing

| Wave | Objective | Depends on | Status |
|---|---|---|---|
| 0 | This plan, merged to `main` | — | **complete** on merge |
| 1 | **The amendments, then the cores**: ADR-003 A2 and ADR-004 A2 on `main`; the deduplication fix; the ACR's per-row rule; `number` and `amount`; `duration`; `uniqueKeys`, then `coverage` | 0 | **complete** (see §5, Wave 1 verdict) |
| 2 | **Teaching and adoption**: the `money` alternative and `help templates`; the examples write their values through the filters; `check`'s three findings; the sample generator; `schema --fields`, `inspect`, `help schemas` and the skill | 1 | **complete** (see §7, Wave 2 verdict) |
| 3 | **The ACR adopts both keywords**, then the close | 2 | — |

Inside Wave 1, the amendments item merges first, and every other Wave 1
item depends on it: each cites the amendment and logs against it. R4
depends on R3 (§5).

**Why R4 is in Wave 1, not Wave 2.** `uniqueKeys` and `coverage` share
approved sentences: §5.4's opening and example, MTS-DAT-004, MTS-DAT-017,
MTS-FND-004 and MTS-SUP-029 each name both. By the rule of §2 those land
with R3, so `main` says `coverage` is defined and evaluated from R3's
merge. With R4 in Wave 2, that window would span a wave boundary, and
Wave 1's verification would find the specification ahead of the tool. With
R4 behind R3 in Wave 1, the window closes before the verification. R3 and
R4 share the vocabulary R3 registers, which is also a shared seam. Nothing
else moves: R5, R6 and R7 need both keywords, so they stay in Wave 2.

**The slices.** Each slice is the approved text's hunks for the named
statements and sections, from the commit named. The wave orchestrator may
move a slice between items only under §2's rules, and says so in its
breakdown.

| Candidate | Slice of #270 (`b0cbb230`) | Slice of #268 (`b80ed115`) |
|---|---|---|
| W1-1, the amendments | ADR-004 A2 and ADR-004's status line | ADR-003 A2 and the status lines of ADR-002, ADR-003, ADR-006, ADR-007 and ADR-008 |
| R1 | — | Chapter 12 MTS-FND-003 |
| R2 | — | — (A2 item 8, third bullet, needs no text) |
| V1 | Chapter 8 §8.1.10 to §8.1.12 (MTS-FLT-015 to MTS-FLT-020); Appendix C; Appendix A's `[ISO 4217]`; `spec/README.md`'s line for Appendix C; the codes `template.number_input` and `template.amount_currency` | — |
| V2 | Chapter 8 §8.1.13 (MTS-FLT-021, MTS-FLT-022) and its examples; Appendix A's `[ISO 8601]` durations; the codes `template.duration_input` and `template.duration_format` | — |
| V1 or V2, whichever merges first | Chapter 3 MTS-MAN-018; chapter 7 MTS-TPL-063's row; chapter 8's heading `8.1.14 What 1.0 does not have` | — |
| V3 | Chapter 8 MTS-FLT-014's wording and MTS-FLT-023 with its note; `template.unknown_filter`'s `alternative`; chapter 13's opening sentence and §13.9 | — |
| V4 | — | — |
| R3 | — | Chapter 4 §4.1.2; chapter 5's opening, §5.4's opening, example, closing paragraph and note, MTS-DAT-004, MTS-DAT-015, MTS-DAT-017 and MTS-DAT-018; chapter 12 §12.3's `uniqueKeys` row and MTS-FND-004; chapter 6 MTS-SUP-029; the codes `input.unique_keys` and `output.unique_keys` |
| R4 | — | Chapter 5 MTS-DAT-016 and MTS-DAT-019; chapter 12 §12.3's `coverage` row; the codes `input.coverage` and `output.coverage` |
| R5 | — | Chapter 13 MTS-AUT-081 to MTS-AUT-083; the code `schema.key_not_declared` |
| R6 | — | Chapter 13 MTS-SMP-010 and MTS-SMP-011 |
| R7 | — | Chapter 13, step 5's "Rules across the rows of a table", with its example and note |
| R8 | — | — |

Two things in the table need care:

- **Chapter 8's headings.** On `main`, "What 1.0 does not have" is §8.1.10
  and holds MTS-FLT-014. V1 and V2 each insert sections before it, so it
  is renumbered §8.1.14 by whichever merges first. If that leaves a gap in
  the numbering that a tripwire refuses, the wave orchestrator runs V2
  after V1 instead.
- **#270's `internal/finding/registry_template.go` entries** are data, not
  approved prose. Each item writes its codes' entries as #270 wrote them,
  and regenerates.

## 4. Rulings

Made by Ed on 2026-10-23:

- **R-VR-1, both features ship in v1.0.** The value-formatting filters and
  the cross-record rules are specified, implemented, proved by vectors and
  adopted by the examples before v1.0 is released, because 1.0 must stand
  for a long time. This train blocks T-A3.
- **R-VR-2, the design is #268 and #270 as written.** Ed reviewed and
  accepted both on 2026-10-23, at `b80ed115` and `b0cbb230`: #270 with its
  rulings 1 to 12, #268 with its rulings 1 to 9, the note on teaching
  (A2 item 10) and its drafting decisions 11 to 19.
- **R-VR-3, `main` never goes red.** The design pull requests do not merge
  as they stand.
  - **Wave 1's first item** lands the two ADR amendments and the status-line
    pointers as one docs pull request taken from the design branches. It
    holds no statement and trips nothing.
  - **Every implementation item** then carries its own slice of the
    approved specification text to `main`, with its code and vectors,
    green (§2, §3).
  - **When every slice has landed,** #268 and #270 are closed as
    superseded, and ADR-003 A2 and ADR-004 A2 become Accepted, at the
    plan's close.
- **R-VR-4, the specification text lands verbatim as approved.** A slice
  that needs a wording change brings it to Ed.
- **R-VR-5, a plan orchestrator runs this train**, by waves, as §2 says.
- **R-VR-6, the vocabulary spike comes first.** R3 opens with a spike of at
  most half a day on `santhosh-tekuri/jsonschema/v6`'s
  `Compiler.RegisterVocabulary` with `Compiler.AssertVocabs` (ADR-003 A2
  item 19). If the spike fails, or `AssertVocabs` changes how any 1.0
  vector validates, R3 **stops and asks Ed** before building on A2 item
  19's fallback, the evaluator's annotation output read by a post-pass.

### Decided in drafting, for Ed's review of this plan

- **R4 joins Wave 1, behind R3** (§3), so that no wave verification finds
  the specification ahead of the tool.
- **§2's shared-sentence rule**, and the tool's strictness inside the
  window.
- **The amendments item's status lines.** W1-1 lands each amendment's text
  as written, with two changes outside the approved specification: each
  amendment's status line also records that Ed accepted the design as
  written on 2026-10-23, and that it becomes Accepted at this plan's
  close; and ADR-003 A2's "for Ed's review" before its drafting decisions
  becomes a record of that acceptance. Status lines change again at the
  close.
- **Ed's UX check comes before the close** (§6), so T-A3, which depends on
  the close, waits for it too.
- **R2 decides how the rule carries to the Section 508 and EN 301 549
  tables**, as the candidate says, and records its reasoning in its pull
  request and its log entry. If the answer is a judgement about what the
  report means rather than about the schema, it asks Ed.

## 5. Wave 1 — the amendments, then the cores

**Goal:** both amendments on `main`, and each feature's core built,
specified and proved.

### W1-1 — the amendments

ADR-003 A2 and ADR-004 A2 as written in `b80ed115` and `b0cbb230`, and the
status-line pointers of ADR-002, ADR-003, ADR-004, ADR-006, ADR-007 and
ADR-008, as one docs pull request, with the status lines §4 describes. No
specification text and no code. Both design commits sit directly on
`c831e966`; if `main` has since changed an ADR file, the item takes the
hunks, not the files.

### The candidates

From #268 and #270's proposed breakdowns. The wave orchestrator derives
the items from them.

- **R1, the MTS-FND-003 deduplication fix** (ADR-003 A2 item 8, ADR-006 D8
  item 3). Two findings are duplicates only when their `schema.entry` and
  `schema.pointer` are equal too. Red first: the 54 `min_contains` findings
  #268 measured as one. Vectors under `fnd/`: two keywords that fail alike
  at one pointer, reported twice; one keyword reached by two paths through
  a shared `$ref`, reported once. Expectations that change because a member
  two `required` keywords require is now reported once for each are
  regenerated, and the pull request names each. Independent of the
  keywords.
- **R2, the ACR's per-row rule.** A Level A or AA criterion is never `Not
  Evaluated`, as an `if` and `then` in each WCAG table's item schema, with
  the decision on the Section 508 and EN 301 549 tables of §4. Red first:
  such a row validates on `main`. Independent.
- **V1, `number` and `amount`** (ADR-004 A2 rulings 2 to 8 and 12, decision
  items 1 to 3 and 7).
  - The currency table is a Go table, held equal to Appendix C by a test
    that parses the appendix.
  - A number's value is the shortest round-trip decimal
    (`strconv.FormatFloat(f, 'g', -1, 64)`), rounded half away from zero
    on `math/big`'s `big.Rat`, never with `FormatFloat(f, 'f', n, 64)`:
    2.675 at two decimals is `2.68`.
  - A string is read only as a strict decimal text.
  - `template.number_input`; `template.amount_currency`, which never holds
    the code, with a literal code judged at parse in a template and in a
    manifest expression; MTS-TPL-063's charge.
  - Vectors: `flt/number`; `number-rounding` (2.675, 1.005, 0.125, −2.5,
    −0.004, 1e21); `number-strings`; `number-refusals`; `amount` (`EUR`,
    `JPY`, `BHD`, `CLF`, and `XAU` with `decimals:`); `amount-currency`;
    `tpl/charge-number` at its bound.
- **V2, `duration`** (ADR-004 A2 rulings 9, 10 and 12, decision item 4).
  Whole seconds, ISO 8601 `PnW` or `P[nD][T[nH][nM][nS]]`, and fields
  separated by `:` only with `fields:`. Directives `%d`, `%H`, `%M` and
  `%S` with `date`'s flags; the largest unit named takes the whole count,
  and what lies below the smallest is dropped, never rounded.
  `template.duration_input` with its reasons (`kind`, `calendar`, `range`,
  `form`), and `template.duration_format`. Vectors: `flt/duration`;
  `duration-fields`, where `00:07:25` with `fields: "d:h:m"` writes
  `0 d 7 h 25 min`; `duration-refusals`; `duration-format`. Runs beside
  V1.
- **R3, `uniqueKeys`** (ADR-003 A2 rulings 2 to 4, 6 and 7, items 11, 12,
  14, 16, 18 and 19).
  - The spike of R-VR-6 first.
  - The grammar (MTS-DAT-015), evaluation wherever the schema object
    applies to an array (MTS-DAT-017), and keyed evaluation, linear in the
    items (MTS-DAT-018).
  - `input.unique_keys` and `output.unique_keys`, located at the key's
    cell when the key has one pointer, with the first occurrence in
    `related`, and every repeat reported.
  - Partial validation reports repeats among the items validated
    (MTS-SUP-029).
  - Vectors under `dat/`, `sup/` and `trn/` as A2's "Conformance" lists
    them for `uniqueKeys`.
  - Until R4 merges, `coverage` stays refused as an unknown keyword.
- **R4, `coverage`** (ADR-003 A2 rulings 5 to 7, items 13 and 16), after R3.
  The grammar (MTS-DAT-016); one `input.coverage` or `output.coverage` per
  missing value, located at the array, with the parameter `missing`
  (MTS-DAT-019); suppressed, and counted, in partial validation. Vectors
  under `dat/`, `sup/` and `trn/` as A2's "Conformance" lists them for
  `coverage`, several gaps in one run among them.

### Rulings made during Wave 1

- **R-W1-c, the ACR's Section 508 and EN 301 549 tables** (Ed, 2026-10-23,
  his option (c), reached through W1-3's park). A row in any of the ACR's
  tables may be `Not Evaluated` only with remarks saying why. In the WCAG
  table this sits beside the Level A and AA rule, so only a Level AAA row
  can be `Not Evaluated` there, and it must give its reason too. The
  reasoning is in #281 and `docs/adr/logs/ADR-003/W1-3.md`. Wave 3's R8
  builds on it.
- **R-VR-7, items run directed tests only** (Ed's standing instruction,
  2026-10-23, given to the train as operator guidance). No metais work
  item runs `gate.sh`'s scoped tier or the wave tier. Each item runs
  `go test` on the packages it changed, plus the tests its task names,
  passes them, and merges. A wave verification checks with directed runs
  too. The full gate runs **once**, on a fresh clone of the merged tree,
  after Wave 3's last merge and before the close. Where §2, §5 or §9 say
  "its scoped gate" or "runs the wave tier once", this ruling replaces
  them.

### Wave 1 verdict

**Complete**, 2026-10-23. Brief and closeout:
[brief-plan-2026-10-metais-values-and-rules-wave1.md](../briefs/brief-plan-2026-10-metais-values-and-rules-wave1.md)
(brief #277, closeout #287). All seven items merged:

- W1-1, the amendments: #279.
- W1-2, R1: #284.
- W1-3, R2: #281.
- W1-4, V1: #283.
- W1-5, V2: #280.
- W1-6, R3: #282.
- W1-7, R4: #286.

R-VR-6's spike succeeded, so `uniqueKeys` and `coverage` are a registered
vocabulary under `AssertVocabs`, A2 item 19's fallback was not built, and
no module was added. Nobody asked for a wording change, and the verifier
checked line by line that every slice landed verbatim. Every shared
sentence opened in the wave also closed inside it.

The verifier, `01a16a25-ad4e-730d-823e-468a05a081fa`, ran 39 mutations
across W1-2 to W1-7 and every one went red. It read W1-1 without mutating
it. Under R-VR-7 it ran directed tests, not the wave tier.

What the plan should know:

- **`main` did not stay green after every merge.** `gofmt` failed on
  `internal/cli/inspect.go` from #283 (`5549ee65`) until #275
  (`9f83587c`), another train's fix, realigned it. `TestCoverage` and
  `TestFindingCodesCovered` were green at every merge.
- **W1-3 grew under R-W1-c.** It holds every table to the remarks rule,
  not only WCAG to the Level A and AA rule.
- **V1 and V2 already wrote minimal `metais help templates` rows**, because
  `TestTemplatesTopicFilters` demands a row for each filter. V3 in Wave 2
  rewrites them as teaching.
- **Carried to Wave 2:**
  - V3 owns the help rows;
  - R7 owns the findings' hints in words;
  - W1-NB1 (`01a16a38-9c5f-76d9-aa17-2506bc5e4bdc`) adds three small
    vectors and runs beside Wave 2 without being waited on.
- **The full gate has not run on the merged tree.** Under R-VR-7 it runs
  once, before the close.

## 6. Out-of-band acts

| Act | When | Who | Before | Notes |
|---|---|---|---|---|
| Add this train's close to T-A3's `depends_on` | at this plan's merge | the session that files the plan orchestrator | T-A3 can start | R-VR-1. T-A3 is `01a15ac4-4d4e-775e-9989-615255b813b7`; keep every existing edge. |
| Rule on the vocabulary spike | only if R-VR-6's spike fails | Ed | R3 builds on the fallback | A2 item 19 names the fallback. R3 parks with the spike's evidence. |
| Rule on a wording change to approved text | only if a slice needs one | Ed | that slice's merge | R-VR-4. The item parks with the sentence, why it cannot be made true as written, and the wording it proposes. |
| Final UX check: re-run the external claude.ai test | after Wave 3's verification | Ed, or a session for him | the close | On a build of the merged tree for the tester's platform, run the binary with the DORA and ACR packages as the 2026-10-23 test did. Confirm that the tester's table, with a criterion answered twice and criteria missing, is now refused, each problem named, and that DORA's amounts, dates, counts, percentages and durations read correctly. A finding comes back as a fix item. |
| Cut a release candidate that holds this train | if PLAN-2026-10-metais-public's Wave 3 passes a candidate before this train closes | that plan's orchestrator, or Ed | T-A3 | T-A3 builds v1.0.0 from the merged tree, so the candidate its rehearsal proves must hold this train's changes. That plan's §8 already cuts a new candidate until one passes. |

## 7. Wave 2 — teaching and adoption

**Goal:** what Wave 1 built is taught by the binary, honoured by the
authoring tools, and used by the examples.

**The candidates** (the derivation is the wave orchestrator's):

- **V3, the `money` alternative and the teaching** (ADR-004 A2 decision
  items 5, 6 and 8), after V1 and V2. `template.unknown_filter` gains
  `alternative: amount` for `money`, `money_with_currency` and
  `money_without_currency`, an explicit mapping, never a near-match by
  spelling (MTS-FLT-023). `metais help templates` teaches the three filters
  example first, beside `date`, with a line of its own saying there is no
  `money` filter and that `amount` takes major units, not cents. Chapter 13
  §13.9 lands with it.
- **V4, the examples use the filters**, after V1 and V2: DORA's dates,
  amounts, counts, percentages and durations; the status widget's and the
  ACR's dates. Each example's expected trees are regenerated, and the pull
  request shows a before and after of the values the tester misread.
- **R5, `check`'s three findings** (ADR-003 A2 item 15, MTS-AUT-081 to
  MTS-AUT-083), after R3 and R4: a `values` member the key's schema refuses
  and more values than `maxItems`, each `schema.unsatisfiable`; a key
  pointer the item schema does not declare, `schema.key_not_declared`, a
  warning. Vectors under `aut/`.
- **R6, the sample generator** (ADR-003 A2 item 17, MTS-SMP-010 and
  MTS-SMP-011), after R3 and R4: one item per `coverage` value first, then
  up to `--items`; no two items with equal keys, regenerated from
  `attempt:<k>` streams; `schema.sample_unsatisfiable` with item 23's new
  reasons. Vectors under `smp/`. `render --sample` works on a package that
  uses both keywords.
- **R7, each rule shown in words before it can fail** (ADR-003 A2 item 10),
  after R3 and R4: `schema --fields` writes one line per rule after the
  array's line, every value listed; `metais.schema/1` gives each field its
  `rules`; `inspect` flags an input with rules across its rows and names
  the command that lists them; a new `metais help schemas` topic, which
  `metais help schema` points to; the findings' hints name the rule in
  words, where R3 and R4 did not already write them so; the skill names
  both keywords in its authoring path. Chapter 13's "Rules across the rows
  of a table" lands with it.

**Shared seams to name at breakdown:** the help goldens and `help.json`
(V3 and R7); the skill (R7, and V3 if it touches the skill); the
examples' trees (V4 now, R8 in Wave 3); `internal/finding`'s registry
(every item that registers a code).

### Rulings made during Wave 2

- **R-W2-a, where `schema --fields` writes a rule** (plan orchestrator,
  2026-10-23). The tool builds ADR-003 A2 item 10 exactly: one line per
  rule, after the line of the array that carries the keyword. Chapter 13's
  Note says the tool writes each rule "on the line of the array it
  constrains". That Note is informative, and Ed accepted it in the same
  design as item 10, so it is read as agreeing with item 10: the rule is
  written with that array's line, directly under it. The Note lands
  verbatim and no wording change is needed. Only a tool that put a rule
  anywhere other than directly after its array's line would bring the
  Note to Ed.

- **R-W2-b, a `coverage` member the key's schema refuses** (wave
  orchestrator, 2026-10-23, given to W2-NB1). The sample generator cannot
  produce such an item (MTS-SMP-010), so it reports
  `schema.sample_unsatisfiable` with reason `coverage`. It does not write
  the item and then report `schema.sample_invalid`.

### Wave 2 verdict

**Complete**, 2026-10-23. Brief and closeout:
[brief-plan-2026-10-metais-values-and-rules-wave2.md](../briefs/brief-plan-2026-10-metais-values-and-rules-wave2.md)
(brief #289, closeout #303). All six items merged, with one correction:

- W2-1, V3: #294.
- W2-2, V4: #293.
- W2-3, R5: #295.
- W2-4, R6: #296.
- W2-5, R7's listing half: #297.
- W2-6, R7's teaching half: #300.
- W2-7, a correction of W2-1: #301.

The tester's misread values now read correctly: DORA writes
`EUR 48,920,000.00` and `0 d 7 h 25 min`.

The verifier, `01a16a7f-1338-718e-9bcb-4400634e8697`, ran 31 mutations
across the six items. Four stayed green, and each became a finding: one
blocking, fixed by W2-7, and the others non-blocking. It checked that
chapters 8 and 13 equal the approved text, and that no shared sentence
was left open. `TestCoverage`, `TestFindingCodesCovered` and `gofmt` were
green at every one of the wave's 12 commits.

What the plan should know:

- **The whole-package race runs of `internal/guarantees` and
  `internal/conformance` time out on the shared host.** They took 600 s
  at about half their CPU share, and the same happens at `c831e966`,
  before this train. The verifier's scoped re-runs of this wave's parts of
  them were green. This bears on the one full gate that R-VR-7 places
  before the close (§8).
- **Squash commit `72c33df9` (#300) ends `Signed-off-by:  <>`**, though its
  branch commit carries a proper sign-off. Repairing it would rewrite
  `main`'s history, which no work item may do. Issue #302 asks Ed to
  decide.
- **W2-5 writes a rule's values as canonical JSON** (`"1.1.1"`), not the
  bare text of A2 item 10's illustration. That keeps `1` and `"1"` apart.
- **Still running, not waited on:**
  - W2-NB1 (`01a16aa6-6764-702a-b809-d8df0783afd3`), R-W2-b;
  - W2-NB2 (`01a16aa6-ba87-7094-883a-7a5160cb7686`), DORA's currency pin
    and `check`'s `prefixItems` branch.

## 8. Wave 3 — the ACR adopts both keywords, then the close

**Goal:** the example SC1 rests on refuses a contradictory or incomplete
report, and the train closes.

- **R8, the ACR adopts both keywords**, after R2 to R7. Each criteria
  table's array gains `uniqueKeys` on its criterion and `coverage` of the
  criteria it must answer, for WCAG every Level A and AA criterion; the
  shipped tables are expanded to the full criteria. A data case reproduces the tester's table, a criterion
  answered twice and criteria missing, and is refused with one finding per
  repeat and per missing criterion. `render --sample` still renders the
  package. Red first: the tester's table validates on `main`.
- **The close**, by the plan orchestrator, after §6's UX check:
  - #268 and #270 closed as superseded, each with a comment naming the
    pull requests that carried its slices;
  - ADR-003 A2 and ADR-004 A2 Accepted, and the status lines that point
    to them updated;
  - this plan's status, its train closeout, and its row in
    `docs/INDEX.md`.

### Rulings made during Wave 3

- **R-W3-a, the ACR's Section 508 and EN 301 549 tables cover every
  criterion** (Ed, 2026-10-24, his option A, reached through Wave 3's
  verification). Each table's `coverage` holds every criterion its table
  allows: Section 508 chapters 3 to 6, and every EN 301 549 criterion
  outside clauses 9, 10 and 11.1 to 11.4, as #308 merged it. A chapter or
  clause the product has no part in is listed and answered `Not
  Applicable` row by row; a report may not leave it out. Ed's reasons:
  - It is the only option that can change after 1.0 without breaking
    anyone. Loosening it later, for instance to all or nothing per
    chapter, only admits more reports. Tightening a looser rule later
    would refuse reports that passed before. 1.0 must stand for a long
    time, so the strict form is locked in.
  - It is what the package already told authors. The international
    example's `section508.csv` already answers chapter 4 row by row ("The
    product is software and has no hardware"). Nothing is left out without
    saying so, which SC1 rests on.

  How long a software product's report is, is a rendering question, not
  an input-contract one. A template that collapses a chapter of identical
  `Not Applicable` rows into one line is outside this train, and is not
  filed.

## 9. Verification

Each wave is verified by a `wave-verification` Work Order, which audits
every item against its wave's section and §2's rules, and runs the wave
tier once on a fresh clone of the merged tree. Every round also checks
two things particular to this train:

- **The slices are verbatim.** Every line a merged slice added to `spec/`
  is a line of `b80ed115` or `b0cbb230`, and every shared sentence of §2
  that opened in the wave is closed, with the tool doing what it says.
- **`main` stayed green.** No merge of the wave left `TestCoverage` or
  `TestFindingCodesCovered` red.

**The train's close** is verified by checking that every line #268 and
#270 add under `spec/`, other than generated files, is on `main`. Wave 3's
verification is the edge the close waits on, with §6's UX check.

## 10. What this plan does not cover

- What ADR-004 A2 leaves to a later minor: a locale, a preset, a rounding
  mode, a currency symbol or a minus sign for `number` and `amount`;
  negative or fractional durations; an offset for `date`.
- What ADR-003 A2 rejects: `coverage` values taken from an `enum` or read
  from a reference file, a manifest member, a rules file, and `x-`
  keywords.
- Rules between inputs, which chapter 5 §5.3's input constraints already
  state.
- A UX scorecard run. The final check of §6 is Ed's re-run of the external
  test, not PLAN-2026-10-metais-ux's harness.
- Publishing: PLAN-2026-10-metais-public's.

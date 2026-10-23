# Brief — PLAN-2026-10-metais-values-and-rules, Wave 2: teaching and adoption

**Plan:** [PLAN-2026-10-metais-values-and-rules](../plans/PLAN-2026-10-metais-values-and-rules.md), §7
**Feature:** `metais-values-and-rules`. Work Orders are titled
`metais-values-and-rules [W2-<item>]: <action>`; pull requests and their
squash subjects are `<type>(metais-values-and-rules): [W2-<item>] <change>`
(AGENTS.md, Git and pull requests).
**Repository:** `kottera/metais`, branch `main`. Every Work Order of this
wave gets `repo: kottera/metais` and `branch: main`. The train writes this
repository only.
**Gated on:** Wave 1, complete. Its verdict and the rulings made during it
are under plan §5 (#288). Its closeout is in
[brief-plan-2026-10-metais-values-and-rules-wave1.md](brief-plan-2026-10-metais-values-and-rules-wave1.md)
(#287). The gate has cleared.

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

What Wave 1 built is taught by the binary, honoured by the authoring tools,
and used by the examples. When it is done:

- **`metais help templates` teaches `number`, `amount` and `duration`
  example first**, beside `date`, with a line of its own saying there is no
  `money` filter and that `amount` takes major units, not cents.
  `template.unknown_filter` gives `alternative: amount` for `money`,
  `money_with_currency` and `money_without_currency`, by an explicit
  mapping, never a near-match by spelling (MTS-FLT-023). MTS-FLT-014's new
  wording, chapter 13's opening sentence and §13.9 are on `main` (V3).
- **The examples write their values for a reader.** DORA writes its dates,
  amounts, counts, percentages and durations through the filters. The
  status widget and the ACR write their dates through `date`. Each
  example's expected trees are regenerated, and the pull request shows a
  before and after of the values the tester misread: `48920000` with no
  currency, raw RFC 3339 timestamps, and `00:07:25` read as hours (V4).
- **`check` refuses a rule that can never hold, and warns on a key it
  cannot see** (MTS-AUT-081 to MTS-AUT-083). A `coverage` value the key's
  schema refuses, and more `coverage` values than `maxItems` allows, are
  each `schema.unsatisfiable`. A key pointer the item schema does not
  declare is `schema.key_not_declared`, a warning (R5).
- **The sample generator honours both keywords** (MTS-SMP-010, MTS-SMP-011):
  one item per `coverage` value first, then up to `--items`; no two items
  with equal keys, regenerated from `attempt:<k>` streams;
  `schema.sample_unsatisfiable` with the two reasons ADR-003 A2 item 17
  adds to ADR-007 D9 item 23's list (`uniqueKeys` exhausted, `coverage`
  beyond `maxItems`).
  `render --sample` works on a package that uses both keywords (R6).
- **Each rule is shown in words before it can fail** (R7):
  - `schema --fields` writes one line per rule after the array's line,
    with every value listed;
  - `metais.schema/1` gives each field its `rules`;
  - `inspect` flags an input with rules across its rows and names the
    command that lists them;
  - a new `metais help schemas` topic, which `metais help schema` points
    to;
  - the findings' hints name the rule in words, where Wave 1 did not
    already write them so;
  - the skill names both keywords in its authoring path;
  - chapter 13's "Rules across the rows of a table" lands with its example
    and note.
- **Every slice of plan §3's table for V3, R5, R6 and R7 is on `main`
  verbatim**, and every shared sentence opened in the wave has closed.

## What to read first

- Plan §2 and §3 (the train's three rules and the slice table), §5's
  rulings made during Wave 1 and its Wave 1 verdict, and §7 (this wave).
  Derive the items from §7's candidates, from the slice table and from the
  code.
- The Wave 1 closeout, above. Its §2 and §3 say what it left to this wave.
- On `main`: ADR-003 A2, items 10, 15, 17 and 23, and its "Conformance"
  section for the `aut/` and `smp/` vectors; ADR-004 A2, decision items 5,
  6 and 8. Both landed in #279.
- The approved text, at the accepted commits and never at a later push:
  `git show b0cbb230` (#270) for V3's slice, and `git show b80ed115`
  (#268) for R5's, R6's and R7's slices.
- AGENTS.md whole. The `verification-gate` skill only for the generated
  files' merge drivers and `regenerate.sh`; see R-VR-7 below for the gate
  itself.
- In the tree:
  - `internal/cli`, its help topics, `testdata/help/` and the
    `TestTemplatesTopicFilters` rule;
  - `internal/skill/SKILL.md`;
  - `internal/author/check.go`;
  - `internal/sample`;
  - the `schema --fields` and `inspect` commands;
  - `examples/dora-incident-report`, `examples/status-widget` and
    `examples/accessibility-conformance-report`, their templates and
    their expected trees;
  - `site/`, whose reference pages are generated from `metais help
    --json` and `metais skill --print`.

## What Wave 1 learned that changes this one

- **R-VR-7: items run directed tests only, never the gate** (Ed's
  standing instruction, plan §5). Every item runs `go test` on the
  packages it changed, plus the tests its task names, passes them, and
  merges. No item runs `gate.sh`'s scoped tier or the wave tier, and none
  is commissioned to. The wave's verification checks with directed runs
  too. The full gate runs once, after Wave 3's last merge. Write this into
  every item's task from the start. In Wave 1, seven gates started at once
  on one shared host and starved each other until every one timed out.
  Two items then parked over it, and the rule came mid-wave.
- **Directed tests must include `gofmt -l` over every file changed.**
  Wave 1 left `main` failing `gofmt` from #283 until another train's fix
  realigned the line, because no item ran the check before merging. The
  gate no longer catches this, so each item must. `go vet` on its
  packages, and the specdata and generated-file drift tests when it
  touches the registry or the specification, are part of directed tests
  too.
- **The help rows for the three filters already exist.** W1-4 and W1-5
  each wrote a minimal `metais help templates` row, because
  `TestTemplatesTopicFilters` demands one per dialect filter. They are V3's
  to rewrite as teaching, not someone else's work to preserve.
- **The hints are R7's.** `input.coverage`'s hint names only the missing
  value today, and the `uniqueKeys` findings' hints were written by W1-6.
  R7 decides which hints still need the rule in words.
- **The ACR's tables follow R-W1-c**: a `Not Evaluated` row needs remarks.
  V4 changes the ACR's dates only. It writes no rule and must keep W1-3's.
- **Two lines of the Wave 1 brief conflicted**, "no help topic" against a
  test that forced one. Where a test forces something this brief places
  in another item or wave, keep `main` green, say so in the pull request,
  and name the item that owns it.

## Rulings — do not reopen

- **R-VR-1 to R-VR-7 and R-W1-c** (plan §4 and §5), and every ruling and
  drafting decision of #268 and #270 that R-VR-2 names. In particular:
  - **The design pull requests never merge.** Each item carries its own
    slice of the approved text, with its code and vectors. #268 and #270
    close at the plan's close, and no item closes them or pushes to their
    branches.
  - **Approved text lands verbatim.** An item that finds a sentence wrong,
    or one its code cannot make true as written, parks with the sentence,
    the reason and the wording it proposes. That question goes to Ed;
    bring it up the tree.
  - **Inside a shared sentence's window the tool stays strict**, as in
    Wave 1.
- **Order** (plan §7): R5, R6 and R7 each build on R3 and R4, which have
  merged. V3 and V4 build on V1 and V2, which have merged. So no candidate
  waits on Wave 1 any more. Edges inside this wave are for its shared
  seams (below), not for that history.
- **`money` maps by an explicit table** (MTS-FLT-023): exactly the three
  names, each to `amount`. No edit-distance suggestion.
- **`help schemas` is a new topic**, and `help schema` points to it. The
  topic and the skill stay vendor-neutral: `TestVendorNeutral` holds them,
  and the site's reference pages are generated from them.
- **No ADR amendment beyond ADR-003 A2 and ADR-004 A2.** If `metais.schema/1`'s
  `rules` or the help JSON needs a schema change A2 does not already
  state, the item parks and asks.

## Shared seams to name at breakdown

Plan §7 names these. Items that touch one either run in sequence or are cut
so they do not both edit it:

- the help goldens and `help.json` (V3 and R7), and the site's reference
  pages and tests generated from them;
- the skill (R7, and V3 if it touches it);
- the examples' expected trees (V4 now, R8 in Wave 3);
- `internal/finding`'s registry (every item that registers a code:
  `schema.key_not_declared`, and any registry change the new
  `schema.sample_unsatisfiable` reasons need);
- chapter 13 (V3's §13.9 and opening sentence, R5's MTS-AUT-081 to 083,
  R6's MTS-SMP-010 and 011, R7's step 5 section).

## Scope — what belongs elsewhere

- **Wave 3:**
  - the ACR adopting `uniqueKeys` and `coverage`, its tables expanded to
    the full criteria, and the tester's table as a refused case (R8);
  - the full gate, once, on the merged tree;
  - Ed's UX check;
  - the close.

  V4 touches the ACR's dates only.
- **Plan §10:** locales, presets, rounding modes, currency symbols,
  negative or fractional durations, `date` offsets; `coverage` from `enum`
  or a file; rules between inputs.
- **Other trains' work:** PLAN-2026-10-metais-public's Wave 2 is still
  merging under this wave: the site, the guides' copy (#265), and the
  install scenarios (#271). So is #272. Rebase onto them and keep their
  changes. A help change that reddens a site test is this wave's to fix,
  in the item that made it.

## What must not break, and how it is checked

- **Directed tests before every merge** (R-VR-7, above): the changed
  packages, the tests the task names, `gofmt -l`, `go vet`, and the drift
  tests the change reaches. The conformance vectors an item adds run in
  `internal/conformance`. A change to the help or the skill also runs
  `internal/cli`'s golden tests and `site/`'s tests.
- **`main` stays green after every merge** (R-VR-3). `TestCoverage` and
  `TestFindingCodesCovered` must be green with each slice. After every
  rebase, run `regenerate.sh` and commit what it changes.
- **The examples still validate and render**, and their `test` cases pass
  with regenerated trees. Rendering stays pure and byte-identical: no
  locale reaches a filter.
- **The sample generator stays deterministic**: the same seed gives the
  same items on every platform.
- **The wave's verification** audits every item against this brief and
  plan §2. It checks plan §9's two train-specific properties: every line
  added under `spec/` is a line of `b80ed115` or `b0cbb230`, with every
  opened shared sentence closed; and `main` stayed green, `gofmt`
  included. It checks with directed runs (R-VR-7), not the wave tier.
- **Log entries** under `docs/adr/logs/ADR-003/` and `ADR-004/`, and in
  every other ADR's log whose decision the item meets (ADR-007 for
  authoring, ADR-008 for the command line), one file per address, with the
  plan-name prefix where a `W2-*` name is taken.

## Out-of-band acts of this wave

Plan §6 assigns no act to this wave. The conditional act "rule on a
wording change to approved text" can arise here as in Wave 1. If it does,
the item parks and the question comes up the tree to Ed. The final UX
check follows Wave 3's verification, and the T-A3 edge is not this
train's to file.

## Closeout

*Not yet written. The wave orchestrator replaces this line with the
wave's closeout.*

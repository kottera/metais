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

Written 2026-10-23 by the wave orchestrator
(`01a16a3c-f273-7cc5-a975-223b0ba3901d`) after the wave drained. This is a
record of what landed and where every finding went. It is **not a
correctness verdict**. I read the children's reports, the merge commits and
the verifier's report. I spot-read the merged tree where an ask named a
mechanism. I did not mutate or test anything myself. The testing described
below is the verifier's, attributed to it.

### 1. What landed, against what the brief asked

The wave had six items and one correction. All are complete and merged to
`main`, below `962e155e` (the wave's base), in merge order. Commits of
other work are interleaved: #290 (W1-NB1), #291 (R-W2-a in the plan), and
PLAN-2026-10-metais-public's #271, #292, #298 and #299.

| Item | Candidate | Work Order | PR | Merge | Landed |
|---|---|---|---|---|---|
| W2-2 | V4 | `01a16a42-6b57-7be2-b248-bf310eadc15d` | #293 | `6cbb16f0` | DORA writes its date-times through `date`, counts and percentages through `number`, amounts through `amount` with the code from `reporting_currency`, and 3.15 and 3.16 through `duration` with `fields: "d:h:m"`. The status widget's visible `<time>` text goes through `date`, and its `datetime` attribute stays RFC 3339. In the ACR only the Word body's report date changed; `acr.json` and `core.xml` keep their machine form, and no ACR rule changed. The PR shows `48920000` → `EUR 48,920,000.00` and `00:07:25` → `0 d 7 h 25 min`. |
| W2-1 | V3 | `01a16a41-d690-78d1-a58e-25c9ff5ee725` | #294 | `bd2ad7ef` | `template.unknown_filter`'s `alternative`, from an explicit three-name table in `internal/template/filters`, used by template parse and manifest expressions alike. `help templates` has a group of its own for `date`, `number`, `amount` and `duration`, example first. MTS-FLT-014's wording, MTS-FLT-023 and its Note, chapter 13's opening and §13.9. `TestWritingValuesForReaders` renders §13.9 from the spec text. |
| W2-3 | R5 | `01a16a42-df3b-7461-9e79-d1866fca0110` | #295 | `53e4949f` | MTS-AUT-081 to 083 in `check`; `schema.key_not_declared` registered; `schema.unsatisfiable`'s message and hint name each of its three causes; `aut/check-*` vectors. |
| W2-4 | R6 | `01a16a43-4c98-7641-b5eb-451d911b31cc` | #296 | `68a86735` | Covered items first, then fill; repeated keys regenerated; reasons `unique_keys` and `coverage`; MTS-SMP-010 and 011; `smp/` vectors; `render --sample` over both keywords. |
| W2-5 | R7, first half | `01a16a43-b465-7d90-b998-9bcb87166df0` | #297 | `85a8e622` | `schema --fields` rule lines directly after the array's line (R-W2-a), every value listed; `metais.schema/1` `rules`; `inspect`'s line; the four rule findings' hints name the column. |
| W2-6 | R7, second half | `01a16a44-2441-7e8a-92f3-9af9706fa296` | #300 | `72c33df9` | `metais help schemas`, with keyword lists checked against a new `schema.Keywords()`; the pointer from `help schema`; the skill names both keywords; chapter 13's "Rules across the rows of a table" with its Note, verbatim. |
| W2-7 | correction of W2-1 | `01a16a9e-6cd7-73c8-9a5d-e707e719e413` | #301 | `ab7fd2c4` | `internal/template`'s `TestUnknownFilterAlternative` guards the template path: near-names carry no `alternative`. Shown red under both of the verifier's mutations. |

**What I checked in the merged tree.**

- Chapter 13 holds "Rules across the rows of a table" at line 160, then
  MTS-AUT-081 at line 217, then `### Step 6` at line 233. So W2-3's and
  W2-6's shared hunk sits in `b80ed115`'s order.
- `internal/template/filters/filters.go` holds the three-name table.
- `internal/cli/inspect.go` writes "has rules across its rows, which metais
  schema <package> <input> --fields lists".
- Every log file the items name exists. The plan-prefixed names are used
  where a `W2-*` name was taken in ADR-007 or ADR-008.
- `gofmt -l .` on `ab7fd2c4` prints nothing.

**Divergences from the ask:**

- **W2-1.2 asked for a proof "by a vector" that a vector cannot give.** The
  conformance harness compares an expected finding's parameters as a
  subset (ADR-011 D10 item 2), so no vector can show a parameter is
  absent. W2-7 met the criterion with the vector and a unit test together,
  and left the harness alone. This was my criterion's wording, not the
  item's fault.
- **The values in W2-5's rule sentences are canonical JSON** (`"1.1.1"`,
  `1`), not the bare text of A2 item 10's illustration. That keeps `1` and
  `"1"` apart, and matches `missing` and the hint. The producer stated
  this reading in #297.
- **W2-6's squash commit carries a malformed sign-off** (§4).

**Rulings made during this wave:**

- **R-W2-a (plan orchestrator, 2026-10-23).** It was given in answer to the
  breakdown's park, and recorded in the plan by #291. In `schema --fields`
  the tool writes one line per rule, after the line of the array that
  carries the keyword, as A2 item 10 says. Chapter 13's Note, "on the line
  of the array it constrains", is read as agreeing: the rule is written
  with, and directly under, that array's line. No wording change was made.
  It governed W2-5 and W2-6. The verifier mutated the order, and its test
  went red.
- **R-W2-b (wave orchestrator, 2026-10-23), given to W2-NB1.** A `coverage`
  member that the key's schema refuses is a case where the generator
  "cannot" (MTS-SMP-010). So it is `schema.sample_unsatisfiable` with
  reason `coverage`, not a `schema.sample_invalid` read back. It follows
  from MTS-SMP-010 with MTS-SMP-004: the sample operation generates values
  the schema allows. If W2-NB1 finds approved text against it, the item
  parks.

Nobody asked for a wording change to approved text, and no item parked.

### 2. What children flagged, declined or discovered

- **W2-2:** English separators (`group: ","`), because the report's labels
  are Annex II's English, as in §13.9. Percentages are written with one
  decimal. The recorded UX-run fixtures under
  `internal/tools/ux/score/testdata/runs/`, which hold old DORA renders,
  were left as past runs.
- **W2-3:** two readings of "the array's item schema".
  - It is the `items` and `prefixItems` of every node in the applicable
    set. If an open set is reached, every token counts as declared.
  - A `coverage` value is reported only when every reached alternative
    refuses it.

  No parameter was added to either finding.
- **W2-4:** a choice and a gap.
  - When `--items` asks for more items than there are distinct keys, the
    array is shortened to the keys there are, as D9 item 16 shortens a
    `uniqueItems` array. Below `minItems` that is `unique_keys`, and
    covered items are never dropped.
  - A refused `coverage` member was not special-cased (§4, W2-NB1).
- **W2-5:**
  - `inspect`'s flag is text only, because A2 states no JSON member for it.
  - The hints name the column through `WithHintValues` from failure data,
    with no new parameter.
  - No vector records a hint, so no expectation changed.
- **W2-1:** suggested that the skill gain a line saying that amounts go
  through `amount` and there is no `money` filter. W2-6 named both rule
  keywords and the `help schemas` topic in the skill. I did not check
  whether it took the amount line.
- **Tooling:**
  - Several clones needed `git config core.precomposeunicode false`,
    because the sandbox's iconv lacks UTF-8-MAC.
  - Some clones had no git identity set.

### 3. What carries to the next wave

- **Wave 3's full gate will hit the host starvation the verifier measured**
  (§4, finding not filed). The whole-package `-race` runs of
  `internal/guarantees` and `internal/conformance` took the full 600 s,
  getting about half the CPU they were allowed. Run the full gate when the
  host is otherwise idle, or split those two packages by `-run` as R-W6-3
  splits the reach.
- **Give every item's squash a real identity.** From W2-7 on, tasks said so
  explicitly, after W2-6's empty trailer. Wave 3's brief should say it from
  the start.
- **R8 starts from what this wave built.** The ACR is the first package to
  adopt both keywords. `check` (W2-3), the generator (W2-4) and the listing
  (W2-5) all meet it, so R8's `render --sample` criterion is the first real
  exercise of W2-4 on 55 covered criteria with `--items` below them.
- **Two follow-ups run beside Wave 3**, unwaited: W2-NB1 and W2-NB2 (§4).
  W2-NB1 touches `internal/sample` and the registry, as R8 may.
- **What I would do differently:** write each criterion in a form the
  repository's proof mechanism can express. W2-1.2's "proved by a vector"
  could not be met, and it cost a correction round.

### 4. What the verification found, and where every finding went

One `wave-verification`, `01a16a7f-1338-718e-9bcb-4400634e8697`. It returned
all six item ids I sent, each `mutated`.

- **What it did:**
  - ran 31 mutations across the six items, by my count of its report's
    rows. Every one went red except:
    - four stayed green, and those became the findings below: two on
      W2-1, one on W2-2 and one on W2-3;
    - one compile error (FALSE_RED), which it re-ran.
  - spot-checked claimed reds: §13.9's `(4.2 %)` and the help rows;
  - checked that every line the wave added under `spec/`, other than the
    generated catalog, is a line of `b80ed115` or `b0cbb230`. It also
    checked that chapters 8 and 13 equal the approved text exactly, and
    that no shared sentence was left open.
  - ran `TestCoverage`, `TestFindingCodesCovered` and `gofmt -l` on a
    worktree of each of the 12 commits in `962e155e..72c33df9`, all green;
  - ran `regenerate.sh` on the merged tree, which changed nothing.
- **How it ran its tests:** directed runs (R-VR-7). Build, vet, `gofmt`,
  the specdata drift, and `go test -race` over cli, template, loader,
  finding, specdata, author, sample, schema, schemafail, skill,
  `cmd/metais`, site and vectorgen. Its gate result is **RED** only
  because the whole-package runs of `internal/guarantees` and
  `internal/conformance` timed out; scoped re-runs of what those two
  packages hold for this wave were green.
- **Where it found nothing:** W2-4 (5 mutations) and W2-5 (6 mutations).

**Blocking: one finding, one fix child.**

| Finding | Criterion quoted | Fix |
|---|---|---|
| `flt/money-near-names` cannot prove a near-name gets no `alternative`, and no test guarded the template path. Making every template `unknown_filter` carry `alternative: amount` left every vector and test green. | W2-1.2 (item) | W2-7 `01a16a9e-6cd7-73c8-9a5d-e707e719e413`, #301, merged. Not re-verified in this wave, by design; its PR shows its test red under both mutations. |

**Non-blocking findings.** Two are combined in one child because each is a
thin test addition, and the task says the two are severable. **The wave
completed without waiting for W2-NB1 and W2-NB2, by design.** They were
filed just before this closeout, after the wave had drained and once no wait
of mine remained, so they hold nothing up.

| Finding | From | Where it went |
|---|---|---|
| `sample` writes a `coverage` member the key's schema refuses, then reports `schema.sample_invalid` with a hint to add an `examples` value, which cannot help | W2-3/W2-4 seam (cross-item) | W2-NB1 `01a16aa6-6764-702a-b809-d8df0783afd3`, under ruling R-W2-b |
| No example pins that DORA's amount code comes from the input: every example is EUR, and hard-coding `"EUR"` stays green | W2-2 | W2-NB2 `01a16aa6-ba87-7094-883a-7a5160cb7686` |
| `check`'s `prefixItems` branch of the item schema is untested: ignoring it stays green | W2-3 | W2-NB2 `01a16aa6-ba87-7094-883a-7a5160cb7686` |
| Squash commit `72c33df9` ends `Signed-off-by:  <>`, though its branch commit `7ecc736f` is signed | W2-6 | Issue #302, a decision for Ed. Repairing it means rewriting `main`'s history, which no work item may do. |
| Whole-package `-race` runs of `internal/guarantees` and `internal/conformance` exceed `go test`'s 10-minute bound on the shared host | cross-item | **Not filed.** It is host starvation, not a defect of this wave: about 0.5 of the allowed share, and 591 s already on pre-wave `c831e966`. Earlier observations record it, and code cannot fix it. It is carried to Wave 3's full gate (§3). |

No janitorial issue was filed: no finding was prose-only.

### 5. Out-of-band acts

None was filed, and none was needed. Plan §6 gives this wave no act. The
conditional act "rule on a wording change to approved text" did not arise:
no item parked. The question of where a rule line sits went to the plan
orchestrator in the breakdown's review, and came back as R-W2-a. The final
UX check follows Wave 3's verification.

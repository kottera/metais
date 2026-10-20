# Brief — PLAN-2026-10-metais-v1, Wave 7: the example packages and the close-out

**Plan:** [PLAN-2026-10-metais-v1](../plans/PLAN-2026-10-metais-v1.md), §13
**Feature:** `metais-v1`. Work Orders are titled `metais-v1 [W7-<item>]: <action>`;
pull requests and their squash subjects are `<type>(metais-v1): [W7-<item>] <change>`
(AGENTS.md, Git and pull requests).
**Repository:** `kottera/metais`, branch `main`. Every Work Order of this wave
gets `repo: kottera/metais` and `branch: main`.
**Gated on:** Wave 6, complete, with its release builder carried here. Its
verdict is under plan §12, and its closeout is in
[brief-plan-2026-10-metais-v1-wave6.md](brief-plan-2026-10-metais-v1-wave6.md).
The gate has cleared.

## What this wave is for

This is the last wave. When it is done:

- **The use cases of PRD §4.1 to §4.5 ship as packages** under
  `examples/`, each passing Wave 6's example-package harness. SC2 and SC3
  have their tests, and SC4 holds.
- **The specification's 1.0 text is complete:**
  - every chapter, with `TestCoverage`, `TestChangelog`,
    `TestFindingCodesCovered` with no unwired list, and every ADR-011 D14
    tripwire green;
  - the worked package;
  - the references;
  - the front page reading `Status: Draft`, ready for the release commit.
- **The repository's documentation is closed out:** the README with both
  golden paths, the `docs/cli/` index, each ADR's implementation log, and
  `docs/INDEX.md`.
- **The release builder exists**, carried from Wave 6, once Ed has ruled on
  how it is allowed (below).

## What to read first

- **The plan:** §2 (including its breakdown rules: shared seams, and
  statement-number blocks with gaps permitted), §4, §5, §6 and §13. §13 is
  the authoritative scope, and Ed's design sessions have edited it: "The
  codes no vector could produce" is now one of its bullets. This brief
  binds §13 and does not restate its outline. Read every wave's verdict
  under §7 to §12.
- **Wave 6's closeout**, especially what it relays, and **W6-13's drafted
  task** for the release builder, which the closeout quotes or links.
- **The ADRs:**
  - ADR-011 D4 (examples and the worked package) and D14 (the tripwires);
  - the ADR-006 A1 and ADR-011 A4 amendments, Ed's rulings;
  - ADR-010 D12 (the release builder);
  - ADR-009 D16 item 3, for what the release builder must get past;
  - PRD-001 §4.1 to §4.5 and §25 (reuse of source material).
- **AGENTS.md** and the two skills. Every item uses the two-tier gate,
  explicit squash messages, and commits under the GitHub account's noreply
  identity.

## What earlier waves learned that changes this one

- **The release builder (W6-13) carries here, and it waits on Ed.**
  - Design session `01a1566e-2786-7173-84ff-c1485fcf8cdf` asks how
    `internal/tools/release` is allowed past ADR-009 D16 item 3's symbol
    rule, which refuses `go build`, git, the environment and file writes in
    non-test module code.
  - Its item is drafted and held, exactly as Wave 6 did it. The task starts
    from W6-13's drafted text.
  - When Ed rules, I record the ruling under plan §13's "Rulings made
    during the wave". Read that section whenever you decide what to do with
    the item. If the ruling is there, write it in and release the item; if
    not, park and ask me once the rest of the wave has merged.
  - The rest of the wave never waits on it.
- **The proofs still owed to real hosts** are the matrix rehearsal's and
  the calibration act's (W6-A1, W6-A2), and they are carried. Nothing in
  this wave reruns them.
- **Wave 6 relayed:**
  - the `check` gap, ruled below;
  - two `manifest.*` codes no 1.0 manifest can produce
    (`manifest.any_of`, `manifest.one_of.ambiguous`), which join this
    wave's ADR-006 A1 work;
  - ADR-011 D16 item 3 against MTS-AUT-002, which goes to Ed's
    ratification as Wave 6's R-W6-9b. Do not reopen it.
- **The example harness skips today**, because `examples/` is empty. Each
  example package that merges makes it run for real.
- **Implementation logs are per-item files** under
  `docs/adr/logs/ADR-<NNN>/`. §13's "each ADR's implementation log, from
  the plan's wave verdicts" means each ADR's log directory is complete and
  its "Implementation log" section reads true. It does not mean appending
  to the ADR.

## Rulings already made — do not reopen

From plan §13, "Decisions settled before filing", ratified with RULINGS-v1:

- **R1-008:** the `$id` domain, registered by §6's act before publication.
- **R1-009:** the media type is named; IANA registration comes after
  publication.
- **R1-022:** the licence.
- **R2-169:** one living text, with `spec-1.0` tags.
- **R2-170:** released vectors are the tie-breaker.
- **R2-177:** specification examples are vectors.
- **Example packages live under `examples/` at the repository root,** as
  source directories. They are not conformance vectors, and each package's
  own documentation is its guide and its meaning documents (ADR-001 D5
  item 5 forbids a top-level `README.md` in a source directory).

Ed's rulings bind this wave:

- ADR-006 A1 (unregister the manifest keyword codes the schema cannot
  produce, with a tripwire);
- ADR-011 A4 (`absent` and `unreadable` setups, so `package.not_found`
  and `package.not_readable` keep class `refused` and get `open` vectors);
- ADR-010 A5 and the plan's §2 rules.

The plan orchestrator's earlier rulings stand, recorded under plan §7 to
§12.

Rulings made by the plan orchestrator for this wave:

- **`check` refuses a literal output path that `render` would refuse**, such
  as `../escape.txt`.
  - Today `check` accepts it, while `render` and `test` refuse it. That is
    not a confinement breach, but it is an acceptance disagreement:
    `check` exists to tell an author whether a package will render.
  - Under plan §4, the stricter rule wins. `check` reports the same refusal
    the predicate gives, with a chapter 13 statement and a vector.
  - Name it in the PR.
- **The two unproducible `manifest.*` codes are treated as ADR-006 A1
  treats the other 14.** If either is producible after all, the item shows
  the vector. If not, it leaves the registry under A1's tripwire. Either
  way, no code stays on a permanent unwired line, and the unwired list is
  removed at the close because it is empty.

## Rules carried whole from the plan (§2 and §4)

- **The specification is written with the code, never after.** An item
  that makes a behaviour reachable adds, in the same pull request:
  - the statements that state it, numbered from its allotted block;
  - the vectors that cite them;
  - any codes it emits.

  This wave completes the text. A statement still lacking a vector at the
  close is a red gate, not a note.
- **Where two items of one wave meet, the one that merges second adds the
  vector for where they meet.** Name each meeting and its owner. The
  close-out describes `examples/` generically, so it holds whichever
  packages merge before it.
- **Shared seams are named at breakdown** (§2). This wave's likely seams,
  as I see them:
  - `spec/README.md`, `CHANGELOG.md` and `A-references.md`;
  - `README.md`;
  - `docs/INDEX.md`;
  - the registry;
  - `unwired.txt`, until it is removed.

  The derivation decides.
- **An item that finds it needs a sibling's unmerged work** that no edge
  covers has found a slicing defect, and it parks.
- **Pin the property false first** for every new guard: ADR-006 A1's
  tripwire, ADR-011 A4's setups, SC2's per-stage test and SC3's.
- **Do not widen a refusal to make a proof pass.** That rules out:
  - a limit raised;
  - a check skipped;
  - an exclusion or runner skip outside the closed sets;
  - a released vector changed;
  - a tripwire allow-list entry beyond what ADR-009 D16 item 3 and R2-143
    prescribe, which is why the release builder waits for Ed.

  An example package that only passes by weakening something is a finding
  for Ed.
- **The stricter security rule wins.** Name it in the PR, and record it in
  the closeout.
- **No new dependency.** Never copy code, or source text whose terms do not
  allow it (PRD §25; §6's terms act).
- **Bounded work is measured in CPU time.** Every test is isolated through
  `internal/testenv`. A Work Order never runs `go test -fuzz`.
- **Never write to another repository.** METAIS names nothing of tera.

## Harness policy — Claude throughout

| Role | Work Order type |
|---|---|
| Wave derivation and supervision | `wave-orchestrator-derive` |
| Implementation and corrective work | `code-implementation-direct` |
| Whole-wave verification | `wave-verification` |

These are `plugin.dev-pipeline.v1`'s types; name one by its full key wherever
its leaf is ambiguous. Never substitute another harness's type to get past a
capacity wait: park on `awaiting-capacity` and let the seat come back.

## Scope boundary

In scope: everything in plan §13's scope outline, the release builder
carried from Wave 6 (pending Ed), and the rulings above.

These must not be absorbed here (plan §13 "Does not touch", §6, §14):

- **The release itself:** the `spec-1.0` commit and tag, `v1.0.0`, and
  `released/1.0.sha256`. The front page stays `Status: Draft` and the
  changelog `unreleased`;
- **the matrix and calibration runs, and the SC1 and SC9 live rounds**,
  which are §6's acts;
- **any default limit change**, which only an ADR-009 amendment Ed ratifies
  can make;
- **the render-pipeline extension points** (after the train, §14);
- **publishing anything outside this private repository.**

## What must not break, and how it is checked

- **The gate:** the scoped tier per item, the wave tier for a shared seam,
  and the wave tier, with R-W6-3's reach step, at verification on a fresh
  clone.
- **Plan §13's acceptance:**
  - the harness passes over every package under `examples/`;
  - SC2 and SC3 have their tests, and SC4 holds;
  - every chapter is complete, with `TestCoverage`, `TestChangelog`,
    `TestFindingCodesCovered` (with no unwired list) and every ADR-011 D14
    tripwire green;
  - the front page reads `Status: Draft`;
  - the SC1 and SC9 acts are filed.
- **Wave 6's release-builder acceptance, if Ed rules in time:** it refuses
  each of its refusals, and produces matching `SHA256SUMS` on two runs. If
  Ed has not ruled when the rest of the wave has merged, the closeout says
  so plainly. I decide then whether the train closes without it.
- **Every earlier guard stays green.**

## Out-of-band acts of this wave

From plan §6, carried whole. The wave orchestrator places each.

1. **Check each example package's source material against its publisher's
   reuse terms.**
   - **When:** Wave 7. **Who:** Ed. **Before:** that example's merge.
   - **Notes:**
     - PRD §25. ITI's VPAT is ITI's, so §4.1 is written from W3C's
       criterion text and GSA's OpenACR catalogue and schema, or in this
       project's own words.
     - §4.2 uses the EU's DORA texts, Implementing Regulation (EU) 2025/302
       Annex II, under the EU's reuse policy.
     - Record each source and its terms in the package's `references/`.
   - Its **Before** is the merge, not the start. An example item may be
     built first, and it does not merge until its terms act has ended.
     Give the merge that edge in whatever form the pipeline supports (one
     act per example, or one act over all five), and say which in the
     breakdown.
2. **SC1 round: an AI harness, given only the binary and the §4.1 package,
   produces a valid ACR for a real product.**
   - **When:** Wave 7, after the §4.1 package merges.
   - **Who:** Ed. **Before:** the v1.0 release.
   - **Notes:** record what the harness read, which must be nothing beyond
     what `inspect` and the commands it points to show. A finding comes
     back as a fix item.
   - Carried, not waited on.
3. **SC9 round: an author working only from `spec/`, with no access to the
   tool's source, writes a package that `check`, `test` and `pack` accept
   cleanly.**
   - **When:** Wave 7, after the worked package merges.
   - **Who:** Ed. **Before:** the v1.0 release.
   - **Notes:** `B-worked-package.md` is where the author starts (ADR-011
     D4 item 4).
   - Carried, not waited on.

**The off-tera types.** This deployment declares `ratify-plan`,
`dogfood-round`, `live-deploy`, `darwin-measurement`, `manual-qa-pass` and
`design-session`.

- **`dogfood-round`** (`runbook`, `build`, `focus`) fits SC1 and SC9.
  `runbook` is a note in the repository saying exactly what the round
  establishes and what to report.
- **`manual-qa-pass`** fits the terms check.
- Ask for evidence, not an attestation. W4-A1R's report is the model.

## Closeout

*Not yet written. The wave orchestrator replaces this line with the wave's
closeout.*

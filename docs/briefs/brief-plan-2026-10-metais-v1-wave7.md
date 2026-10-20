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

The wave orchestrator wrote this closeout (Work Order
`01a15827-975d-7a7b-8cb0-3714d12e7de4`) after the wave drained and its one
verification round reported. It records what landed and what carries
forward. **It is not a correctness verdict.**

**What I checked, and how:**
- I read every child's report, the verifier's full report, and `main`'s
  log from `e51b1ee` to `dd179da`.
- In the merged tree at `dd179da`, I checked these, and nothing more:
  - `examples/` holds five packages;
  - `internal/finding/testdata/unwired.txt` is gone, and the 14 + 2
    `manifest.*` codes are out of `spec/findings/registry.json`;
  - `collection.item_name` and `output.destination.write_failed` are
    produced by `sup/collection-item-name` and `aut/unpack-write-stopped`;
  - ADR-011 A6 is marked Proposed;
  - `spec/15-conformance.md` holds CNF statements;
  - `input-values` (1,000,000) and `yaml-size` (1 MiB) are in
    `internal/limits`;
  - the worked package's packed `.metais` begins with `README.md`, stored;
  - `internal/tools/release` exists;
  - no commit in the range carries a `Co-authored-by` trailer.
- I ran no test, no gate and no mutation. Where something is said below
  to be proved, the proof is the producer's or the verifier's, and is
  attributed to them.

**Wave range:** `e51b1ee..dd179da` on `main`.

- **The breakdown:** ten items were drafted and approved (Q1 to Q3). Seven
  were released as one set. W7-E1 and W7-E2 were held for the terms act
  and released to build first (Q5). W7-R was held for Ed's ruling and
  released when ADR-009 A2 landed.
- **Correctives:**
  - mine: W7-C1, W7-C2, W7-C6 and W7-C8;
  - filed from Ed's session: W7-C3, W7-C4, W7-C5, W7-C7 and W7-S1;
  - W7-A1b, an off-tera act I added.
- **Verification:** one round, W7-V. Verdict **pass**, with no blocking
  finding.
- **Non-blocking findings:** five children, one design session for Ed and
  one janitorial issue (question 4). The wave completed **without waiting
  for them, deliberately**, and without waiting for the SC1 and SC9 acts.
- **Not this wave's commits:**
  - `71266cd` [W6-A0], Ed's ADR-009 A2;
  - `39105ac` [W6-A2], Ed's calibration record and ADR-009 A3;
  - `3f827de`, Ed's ADR-001 A1;
  - `2bfe6ac` [W7-A2], Ed's ADR-009 A4;
  - `c05b324`, `5f6ba42` and `a7db156`, the plan orchestrator's records in
    §13;
  - `c1e8e15` W6-NB1 and `a0cee04` W6-NB2.
- **People consulted:** the plan orchestrator, five times:
  - the breakdown (Q1 to Q3);
  - the release builder and the terms gate (Q4, Q5);
  - `schema-work` and input loading (Q6, Q7);
  - the Q6 follow-up.

  Ed answered W7-A1 and W7-A1b, and ruled Q6 as ADR-009 A4.

### 1. Did each item land, and does it match the ask?

| Item | Work Order | PR | Merge | Landed |
|---|---|---|---|---|
| W7-1 the codes no vector could produce | `01a15832-417d-713c-8c1e-f3fbc1412e23` | #151 | `990b370` | ADR-006 A1's 14 codes and the two relayed ones unregistered, under `TestManifestKeywordCodes`. ADR-011 A4's `absent`, `unreadable` and `unreadable-files`. `open` vectors for both `package.*` codes (MTS-PKG-046). `input.not_readable` (MTS-SUP-030). `collection.item_name` (MTS-SUP-031) and `write_failed` through ADR-011 A6, Proposed. The unwired list is deleted, and `TestFindingCodesCovered` exempts nothing |
| W7-2 the specification's 1.0 text | `01a15832-9f83-71b9-9e8e-06419c9bf1d5` | #153 | `44277c1` | Chapter 15 (MTS-CNF-001 to 006). `check` refuses literal output paths `render` refuses (MTS-AUT-080, `aut/check-output-paths`). `A-references.md` complete, with `TestLinksResolve`. The front page at Draft. Eight `check`-running vectors reclassified as authoring, with the guard in `TestVectorsWellFormed` |
| W7-3 the worked package | `01a15832-f01a-77bc-8da0-1652ad20f083` | #150 | `3cc1c97` | Appendix B from three `gen/` vectors. MTS-GEN-001. `docs/dev/rounds/sc9.md` |
| W7-4 the documentation | `01a15833-54dd-7c0a-b642-087e3a18b663` | #149 | `7d05b03` | README with status and both golden paths. `docs/cli/README.md`. A `W7-4.md` log in each ADR's directory. `docs/INDEX.md` |
| W7-E1 §4.1 the conformance report | `01a15834-5476-7c7f-9f16-d7da14640d7c` | #156 | `8a56ce8` | `examples/accessibility-conformance-report/`: the edition decides the tables, criteria come as CSV, a `.docx` through `ooxml` with no split-run finding, and a validated `acr.json`. EN 301 549 V4.1.1 in own words. `docs/dev/rounds/sc1.md` |
| W7-E2 §4.2 the staged incident report | `01a15834-a323-7877-b6e1-0dcbc15e82de` | #155 | `62017c2` | `examples/dora-incident-report/`: Annex II's names, types and choice values verbatim (75 fields, after W7-A1b). The stage decides required fields. SC2's test, red first |
| W7-E3 §4.3 and §4.5 | `01a15833-b216-74ab-8c5f-5bbcf91acf07` | #148 | `254bafb` | `examples/status-widget/` with SC3's test (`cmd/metais/sc3_test.go`), and `examples/repo-scaffold/` |
| W7-E4 §4.4 the static site | `01a15833-f402-7b69-a069-feaaf47b5eaa` | #152 | `5a02c74` | `examples/static-site/`: a page per article, an index, an Atom feed and assets |
| W7-R the release builder | `01a15834-f64d-72f1-af06-c69c22542390` | #163 | `9eade03` | `internal/tools/release` under ADR-009 A2. Builds twice in fresh clones, compares, and refuses on every listed condition. `internal/tarw` archives and `SHA256SUMS`. A2's three tripwire changes each seen red |
| W7-C1 hermetic git | `01a158bb-dcc4-755b-80f3-1cc6e86c05f2` | #157 | `84e6337` | darwin git writes `core.precomposeunicode=true` into every new repository and needs iconv tables the sandbox denies. Gate scripts source `git-env.sh`, and the tests use `testenv.GitEnv` |
| W7-C2 conformance's run time | `01a158ee-aa78-752c-9e95-85829b1517c7` | #160 | `b68d6ca` | darwin fork copy-on-write re-faults, fixed by a non-race spawner. `TestRealToolInRecord` replaced by the default run's own record. 2,288 to 957 CPU-s. The runner's stream cap enforced |
| W7-C3 `inspect`'s example order | `01a15919-8b73-76d9-a067-04b39dc355e4` | #159 | `01fae3d` | `sortedKeys`. 49 map ranges swept |
| W7-C4 hermetic go | `01a15919-d2aa-7de6-a27b-8963f45f4be0` | #162 | `123f79f` | `testenv.GoEnv`: no host module cache writes, no proxy, no host Go config |
| W7-C5 ADR-009 A3 | `01a15968-db87-791f-996a-8d07ccd91ff6` | #164 | `96ef280` | `grep-context` 7, `output-size` 128 MiB, `output-total` 512 MiB, `schema-work` 14M (later 450k). CPU ceiling 120 s |
| W7-C6 ADR-009 A4, and `schema-work` | `01a15987-791b-7186-8e46-5b50e5368775` | #166 | `dd179da` | `input-values` 1,000,000 and `yaml-size` 1 MiB (MTS-SEC-059/060). Nested findings bounded (MTS-FND-020/021). `schema-work` 450,000. Memory per value cut first. Calibration record `2d8032e6…` |
| W7-C7 ADR-001 A1 | `01a159a6-4485-70f6-96bc-81c5b0a3c3ed` | #167 | `462281d` | `README.md` first and stored in every package, verified at open with four refusal vectors. 150 of 156 regenerated `.metais` keep their digest; the six others are explained |
| W7-C8 the C toolchain in `GoEnv` | `01a15a7a-b79e-7854-98d4-e9d479548e2a` | #172 | `210f8e5` | linux `-race` needs cgo. `GoEnv` keeps the go command's C settings and the compiler's own variables |
| W7-S1 metais.org | `01a15a5d-4c3e-7562-bdc8-a822d4f0a61e` | #170 | `9b48397` | `site/` (home, install, getting started, spec schemas at their `$id`s). A Pages workflow gated on a public repository and a `v*` tag. `docs/dev/release.md` |
| W7-A1 the terms check | `01a15831-d4bb-7dd3-bfc1-e6e117c6bf94` | — | — | Off-tera (Ed). Verdicts per source (question 5) |
| W7-A1b Annex II's table | `01a158ce-f8cd-7db3-8ae3-bb937a236888` | — | — | Off-tera (Ed). 8 differences, and the regulation handed to W7-E2 |

**The plan's §13 acceptance:**
- **The harness passes over every package under `examples/`:** the
  verifier's wave tier passed `TestExamplePackages` over all five.
- **SC2 and SC3 have their tests:** W7-E2's and W7-E3's, each seen red
  first by the producer and mutated red by the verifier.
- **SC4 holds:** each package checks, tests, packs and previews with
  `--optional all` and `none`.
- **Every chapter is complete:** `TestCoverage`, `TestChangelog`,
  `TestFindingCodesCovered` (with no unwired list) and every D14 tripwire
  are green in the verifier's wave tier.
- **`Status: Draft`:** confirmed above.
- **The SC1 and SC9 acts are filed** (question 5).
- **§12's release-builder acceptance, carried from Wave 6, is met** by
  W7-R, under Ed's ADR-009 A2.

**Divergences from the ask:**
- **W7-E1.4's "criterion text":** the Revised 508 text is own-words
  summaries, although W7-A1 allowed it verbatim. WCAG and EN 301 549 are
  in own words as their verdicts require.
- **W7-C6:** A4 item 5 names spec §14.6, and the rows are in §14.4 (issue
  #173).
- **The calibration figures are sandbox figures.** W7-C5's ran on
  linux/amd64. W7-C6's ran on darwin/arm64 with go1.26.2, because the host
  sets `GOTOOLCHAIN=local`. A3's four-target re-run is still owed.
- **W7-3, W7-4, W7-E3 and W7-E4 merged with their scoped gates not
  green,** before W7-C1 and W7-C2 existed. W7-4 and W7-E3 merged under
  R-W7-4, which was later rescinded. The verifier's wave tier is green on
  the merged tree, which is the judgement the rescission names.

### 2. What children flagged, degraded, declined or discovered

None of this was verified by me.

- **The shared host is the wave's main friction.** Package timeouts came
  from oversubscription: a fixed probe got 0.5 to 2.0 of 5 cores. Three
  real defects hid behind them:
  - darwin's fork cost (W7-C2);
  - git's iconv on darwin (W7-C1);
  - the module cache and C toolchain in tests' go env (W7-C4, W7-C8).

  The verifier measured 1.45 to 5.00 of 5 cores during its green run.
- **Memory at the defaults.** A3 and A4 bound the calibrated paths. The
  verifier found two that are not bounded (question 4, W7-D1).
- **Decisions Ed made during the wave:**
  - ADR-009 A2, A3 and A4, and ADR-001 A1;
  - EN 301 549 V4.1.1;
  - both regulated examples stay;
  - the `schema-work` overlap accepted at 450,000;
  - metais.org on GitHub Pages, live at release.
- **Stricter rules that won, each in its PR:**
  - `check` refuses what `render` refuses (W7-2);
  - `internal/tarw` instead of `archive/tar`, which imports `os/user` (W7-R);
  - "os file functions" read as `os.*` only (W7-R);
  - fresh clones for the release builds (W7-R).
- **Found and fixed in passing:**
  - eight vectors of the wrong class (W7-2);
  - the runner's unenforced stream cap (W7-C2);
  - `pkgtreetest`'s map-ordered failure text (W7-C3).
- **Unexplained:** one `TestHostIsolationCanaries/home` failure before
  W7-C2's change, from a signal. It did not recur in the verifier's run.
- **`examples/dora-incident-report`'s generator and `annex2.json` live
  only in W7-E2's scratch.** The committed schemas are the source.

### 3. What carries to the next wave

This is the last wave, so this is what carries to the train's close.

- **For Ed, before the release:**
  - W7-D1 (question 4): output validation and YAML schema files past
    1 GiB;
  - ratify ADR-011 A6 (W7-1's run-time setups) and W7-1's per-position
    reading of ADR-006 A1 item 4;
  - the four-target calibration re-run that A3 requires, now covering A4's
    cases;
  - the matrix and calibration acts W6-A1 and W6-A2, which Wave 6 carried.
- **For the plan orchestrator:**
  - plan §3 still shows Wave 7 "not started";
  - the rulings below (R-W7-C1, R-W7-C5, and R-W7-4 rescinded) are for its
    verdict.
- **What I would do differently:**
  - **Fix the host before the wave.** Every early item merged over the same
    three environmental reds. That cost a rescinded ruling, eight
    correctives and many wakes. A wave tier run on a fresh clone at
    breakdown would have surfaced C1, C2 and C4/C8 first.
  - **Never rule a merge condition from "passed alone".** R-W7-4 had to be
    rescinded. The right answer was always a corrective item.
  - **File the off-tera acts so that their results can reach the
    children.** A held draft whose task I edit worked. Recall is not
    available in a dispatch, so W7-A1b had to be a second act.

### 4. What the verification round found, and where every finding went

**Verifier:** W7-V, `01a15aa4-f04d-78d5-8f3d-785e503420bc`
(`wave-verification`).

- **The gate:** `gate.sh wave` on a fresh clone of `main` at `dd179da`,
  go1.26.9 as `go.mod` pins it, `-p 5 -parallel 5` on a 16-core host.
  **GREEN:**
  - every step passed;
  - `go test -race` passed over all 45 packages;
  - all 58 reach groups passed;
  - nothing was set by the caller;
  - `internal/conformance` took 276 s.

  `TestConformance`'s default run took 19.7 s.
- **The verdict:** pass, with no blocking finding.
- **Coverage:**
  - all 20 ids I sent came back;
  - all 18 merges are on `main`, and the other commits in the range are
    the ones I named.
- **Depth:**
  - 16 items **mutated**;
  - W7-4 and W7-C3 **inspected**;
  - W7-A1 and W7-A1b **skipped**: their reports are off-tera and not
    readable from the verifier, so it checked their consequences in the
    examples instead.

  Every claimed-red row it spot-checked reproduced.
- **What it mutated, in brief:**
  - W7-1's harness checks;
  - W7-2's check branch for asset and mirror destinations;
  - W7-3's shown date;
  - the ACR's `input_constraints` and its split-run template;
  - the incident report's stage constraints;
  - the widget's `escape: html`;
  - the site's draft filter;
  - the builder's `GOAMD64`;
  - `git-env.sh`'s override;
  - the spawner's cap;
  - `inspect`'s sort;
  - `GoEnv`'s `GOPROXY=off` and `CC_TOOLCHAIN_*`;
  - the calibration ceiling;
  - `yaml-size`'s pre-parse check;
  - the README's storage method;
  - a schema's `$id`.

  Each went red.
- **The meetings it checked all hold:**
  - `check` against the examples;
  - the lowered defaults against every example and vector;
  - `GoEnv` against W7-R's tests;
  - W7-C1 against W7-R;
  - chapter 15 against the suite.

**Blocking findings: none.** No fix child was filed in Step 5b.

**Non-blocking findings.** Each is listed with where it went. **The wave
completed without waiting for any of them, deliberately.** They run beside
the train's close and are not re-verified.

| Finding | Item | Where it went |
|---|---|---|
| Output validation parses a rendered JSON or YAML output with no value bound: a ~20 MiB output peaks at 2.0 GiB. A ~4 MiB YAML schema file makes `inspect` peak at 1.38 GiB. No calibration case covers either (cross-item) | W7-C5 / C6 | **W7-D1** `01a15abe-ed83-7284-9592-ca591cdd72db`, a design session for Ed, because bounding them is an ADR-009 amendment (question 5) |
| The release builder pins neither `GOTOOLCHAIN` nor `GOWORK`, and never checks the toolchain line, against ADR-010 D1 (cross-item with W7-S1's install page) | W7-R | W7-NB1 `01a15abd-ddca-79b8-ab4f-4a8dae61b37e`, member A |
| `git-env.sh` uses bash's `compgen` under `#!/bin/sh`, so `GIT_*` leaks on dash through `merge-setup.sh` | W7-C1 | W7-NB1, member B |
| `TestGoCommandKeepsHostCToolchain`'s fake-CC check is vacuous | W7-C8 | W7-NB1, member C |
| A1's tripwire can judge two `oneOf` branches exclusive while ignoring `type` (latent) | W7-1 | W7-NB2 `01a15abe-23ff-7dea-9dcb-3ad5a42f366e`, member A |
| `TestLinksResolve` reads one line at a time | W7-2 | W7-NB2, member B |
| Nothing holds chapter 14's limit defaults to the code | W7-C5 / C6 | W7-NB2, member C |
| Appendix B §B.9 omits the stored `README.md` (W7-3 meets W7-C7) | W7-3 / C7 | W7-NB2, member D |
| `check`'s finding for an over-long or over-deep literal path is unpinned | W7-2 | W7-NB3 `01a15abe-5929-7b9c-9746-cca65dede722`, member A |
| `aut/check-output-paths`' "sometimes" case cannot catch its defect | W7-2 | W7-NB3, member B |
| `inspect`'s per-input examples order is unpinned | W7-C3 | W7-NB3, member C |
| An empty YAML document or frontmatter counting one value is unpinned | W7-C6 | W7-NB3, member D |
| The nested-findings memory bound has no go-test guard | W7-C6 | W7-NB3, member E |
| WCAG names are copied into templates and outputs without W3C's notice | W7-E1 | W7-NB4 `01a15abe-8fb7-7553-ae46-c09bad697eec`, member A |
| The partials cite a nonexistent `catalogue.py` | W7-E1 | W7-NB4, member B |
| The ACR's level set and `acr.json` validation are unpinned | W7-E1 | W7-NB4, member C |
| The scaffold's `executable: true` is unpinned | W7-E3 | W7-NB4, member D |
| The CLI schemas' `$id`s will not resolve on the site | W7-S1 | W7-NB5 `01a15abe-c2dd-7474-9870-06cc2d45750f`, member A |
| `sc1.md`'s pack command exits 127 as written | W7-E1 | W7-NB5, member B |
| `inspect --guide`'s help line misstates its output | W7-4 | W7-NB5, member C |
| README and INDEX give paths, not links | W7-4 | W7-NB5, member D |
| A4 item 5 names spec §14.6; the rows are in §14.4 | W7-C6 | **janitorial issue #173** |
| Plan §3 shows Wave 7 "not started" | W7-4 | **Not filed:** the plan orchestrator's verdict writes §3 |
| Some log files say "recorded above" | W7-4 | **Not filed:** log entries are never overwritten (AGENTS.md), and the wording misleads no one |
| W7-3.5 and the scoped gate | W7-3 | **No action:** history; the wave tier judged it green |
| `TestConformance`'s timing on the merged tree | W7-C2 | **No action:** C2.3 holds, at 19.7 s |


**Why the members are grouped:**
- **W7-NB1:** the host environment's reach. It touches shared seams, so
  one wave tier covers all three.
- **W7-NB2:** `spec/` and its tripwires (`internal/guarantees`,
  `internal/finding`).
- **W7-NB3:** guards in `internal/author`, `internal/cli` and
  `internal/inputs`, which the gate selects together.
- **W7-NB4:** `examples/`, under one harness.
- **W7-NB5:** what a reader is pointed at: the site, runbooks, help and
  index.

Every task says its members are severable.

**Where the verifier found nothing:** W7-C2's criteria hold on the merged
tree, and W7-C7's README-first holds over all 151 tracked packages.

### 5. Out-of-band acts

| Act | Type | Work Order | Placement |
|---|---|---|---|
| W7-A1: terms check of the §4.1 and §4.2 sources | `manual-qa-pass` | `01a15831-d4bb-7dd3-bfc1-e6e117c6bf94` | **Waited on**, by holding and then gating E1's and E2's merges (Q5). Ended by Ed, verdict pass. Every source is usable: WCAG as excerpts with W3C's notice; OpenACR CC0; Revised 508 verbatim; EN 301 549 in own words at V4.1.1; DORA and 2025/302 verbatim with the EU notice |
| W7-A1b: Annex II's field table | `manual-qa-pass` | `01a158ce-f8cd-7db3-8ae3-bb937a236888` | **Waited on** by W7-E2's merge. Ended by Ed: 8 substantive differences found, and the regulation handed to W7-E2, which fixed them |
| W7-A2: the SC1 round | `dogfood-round` | `01a15abf-4d58-7c58-94ca-f4833e218555` | **Carried, not waited on.** `runbook` `docs/dev/rounds/sc1.md`, `build` `dd179da`. Its focus names `sc1.md`'s broken pack line (W7-NB5) |
| W7-A3: the SC9 round | `dogfood-round` | `01a15abf-6347-7520-8a58-1b9ab5f65180` | **Carried, not waited on.** `runbook` `docs/dev/rounds/sc9.md`, `build` `dd179da`. Its focus names §B.9's missing README (W7-NB2) |
| W7-D1: two unbounded memory paths | `design-session` | `01a15abe-ed83-7284-9592-ca591cdd72db` | **Carried**, for Ed before the release |

Wave 6's W6-A1 (the matrix rehearsal) and W6-A2 (calibration) were carried
into this wave. W6-A2's record merged as `39105ac` with ADR-009 A3.

### Rulings made during this wave

- **Q1** (plan orchestrator, plan §13): `collection.item_name`, and
  `write_failed` if needed, are produced through an ADR-011 amendment in
  A4's shape, with a run-time setup behind a `requires` probe and no
  exemption list. Governs W7-1, which wrote A6, Proposed.
- **Q2 and Q3** (plan orchestrator): §4.3 to §4.5 are written in the
  project's own words, with no terms act. The release builder stays held.
- **Q4 and Q5** (plan orchestrator): E1 and E2 build first and park before
  merging for W7-A1. W7-R's fate was to be settled at the end; Ed's A2
  released it first.
- **Q6** (Ed): ADR-009 A4. **Q7** (plan orchestrator): `schema-work` under
  A3 item 3, with no nested-findings limit this wave.
- **R-W7-4** (wave orchestrator): merge over package timeouts caused by
  host contention when the tests reading the changed files pass alone.
  **Rescinded by the plan orchestrator**, because "it passed alone"
  closes nothing (Ed, 2026-10-19). W7-4 and W7-E3 merged under it, and the
  verifier's wave tier is green on the merged tree.
- **R-W7-C1** (wave orchestrator): W7-C1 merged on a green wave tier at its
  exact head, while a scoped re-run's only red was a timeout owned by
  W7-C2. This broke the C1/C2 deadlock. For the plan orchestrator's
  verdict.
- **R-W7-C5** (wave orchestrator): `schema-work` lowered under A3 item 3,
  with the unbounded paths filed as W7-C6.
- **R-W7-E2b** (wave orchestrator, upheld by Ed): §4.2 carries the three
  reports only.
- **On gate re-runs** (wave orchestrator, given to E1, C4, C5, C6 and R): a
  `gate.sh --resume` once a starved host gives the gate its share is a
  re-run of the same gate, not a merge around a red. The PR names the run
  and its CPU.

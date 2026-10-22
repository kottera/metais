# Brief — PLAN-2026-10-metais-public, Wave 1: foundations

**Plan:** [PLAN-2026-10-metais-public](../plans/PLAN-2026-10-metais-public.md), §5
**Feature:** `metais-public`. Work Orders are titled `metais-public [W1-<item>]: <action>`;
pull requests and their squash subjects are `<type>(metais-public): [W1-<item>] <change>`
(AGENTS.md, Git and pull requests).
**Repository:** `kottera/metais`, branch `main`. Every Work Order of this wave
gets `repo: kottera/metais` and `branch: main`. No item of this wave writes
another repository.
**Gated on:** Wave 0, complete: ADR-013, its amendments and the plan merged
in #240. The gate has cleared.

## Harness policy

Claude throughout, by plan §2: the dev pipeline's types, which are named
here by leaf and resolve to `plugin.dev-pipeline.v1/work-order-types/<leaf>`.
File every item by these types and no other. If a seat is not available,
wait for it rather than substituting another harness's type.

| Role | Work Order type |
|---|---|
| Wave derivation and supervision | `wave-orchestrator-derive` |
| Implementation and corrective work | `code-implementation-direct` |
| Whole-wave verification | `wave-verification` |

## What this wave is for

A tree that can be made public, and that an outsider can contribute to,
with nothing yet published. When it is done:

- **The module is `metais.org/metais`** everywhere a live file names it,
  and the site carries the two `go-import` pages that make the vanity path
  resolve once metais.org serves them, held to ADR-013 D1 item 2 by a
  test.
- **The whole git history has been scanned for secrets** by a maintained
  scanner, and every finding is either rotated or shown to be a false
  positive. The current tree names no internal host, no tailnet URL and
  no person's machine. AGENTS.md stands on its own, pointing at no
  private document for a rule.
- **The community files exist**: the contributor guide, the code of
  conduct, the security policy, the issue templates and the pull request
  template.
- **CI exists on GitHub-hosted runners**, dispatched by hand until the
  flip, each job proved by one dispatched run. A tripwire in
  `internal/guarantees` keeps every workflow off self-hosted runners, off
  `pull_request_target`, and declaring its `permissions`. AGENTS.md
  states three gate tiers.
- **`metais --version` ends with its maker line**, under an ADR-008
  amendment, with its help golden.

Nothing is published by this wave: no tag, no release, no change of the
repository's visibility, no push to another repository.

## What to read first

- Plan §1 (done means), §2 (the execution model and this train's rules —
  especially "the module path change runs alone, first", "an act is
  irreversible once it is public" and "private minutes are spent only on
  proving runs"), §4 (the rulings), §5 (this wave) and §6 (the
  out-of-band acts).
- [ADR-013](../adr/ADR-013-the-public-project-and-its-distribution.md):
  D1 items 1, 2 and 5 (the identity and the site's URL space), D2 items 2
  and 3 (vendor neutrality, and the one exception), D3 (licence, the DCO
  and the community files), D4 items 2 and 3 (the scan and the tree's
  internal details), D5 (CI).
- [ADR-010](../adr/ADR-010-the-implementation.md) A8, items 1, 4 and 5;
  A5 (the two gate tiers this wave's CI sits beside).
- ADR-008 D15 and D16 and its latest amendment, for where the
  `--version` amendment goes and how one is shaped.
- ADR-009 A2 and A8, whose `never-depends` rules change path with the
  module.
- AGENTS.md whole, and `.claude/skills/verification-gate/SKILL.md`.
- `site/site_test.go`, `site/build.sh` and `.github/workflows/pages.yml`,
  for the existing site checks and the existing `workflow_dispatch` gate
  that CI copies.

## What the earlier waves learned that changes this one

This is the train's first wave, so there is no earlier closeout of its own.
What the tree as it stands today (after #240) tells you:

- **516 files name `github.com/kottera/metais`**, a little more than
  ADR-013's 514, because the ADR and the plan themselves name it. Those
  two, and every brief, closeout and implementation-log entry, are records
  and keep the path they were written with (plan §5). Live documents,
  skills, code, goldens, baselines and `testdata/` modules change.
- **`site/public/metais/` does not exist yet.** The `go-import` pages are
  new files there. The site's other pages (`index.html`, `install/`,
  `getting-started/`) are Wave 2's, and this wave leaves their content
  alone except where an internal detail must be cleared.
- **`.github/workflows/` holds only `pages.yml`**, gated to
  `workflow_dispatch` while the repository is private. CI follows that
  gate (ADR-013 D5 item 5).
- PLAN-2026-10-metais-ux's closeout carried R-W3-5 (a session with nothing
  installed never finds METAIS) to this train. It is Wave 2's and Wave 3's,
  through the site and I2, and nothing in this wave answers it.

## Rulings — do not reopen

- **R-PUB-1 to R-PUB-14** in plan §4, and every ADR-013 decision, are
  settled. In particular: the repository stays `kottera/metais`; the
  module path is `metais.org/metais`; history is kept and scanned, and a
  secret found is rotated, never rewritten out of history (R-PUB-7); the
  DCO, not a CLA (R-PUB-13); the specification stays Apache-2.0
  (R-PUB-6).
- **The module path change runs alone and first.** Every other item of
  this wave depends on it and is rebased onto it. It runs the wave tier
  before it merges, because `go.mod` is a shared seam (AGENTS.md).
- **`go-import` pages.** Exactly the two package paths ADR-013 D1 item 2
  names, `metais.org/metais` and `metais.org/metais/cmd/metais`, each
  carrying `go-import` with content
  `metais.org/metais git https://github.com/kottera/metais` and a
  `go-source` tag. A test holds them to that. They sit under `/metais/…`,
  the path D1 item 5 gives them.
- **Internal details are cleared from the whole current tree, records
  included.** ADR-013 D4 item 3 says "the current tree", and its reason is
  exposure, not accuracy, so it reaches briefs, closeouts, log entries,
  committed benchmark baselines' host labels and this plan's own text
  alike: a host name becomes a neutral description of the platform (OS,
  architecture, CPU class), and a note about a person's machine becomes
  the procedure it stood for. That is the one exception to "records keep
  what they were written with", and it does not extend to the module path.
  Editing such a phrase in the plan changes nothing the plan asks.
  **Work Order IDs stay.**
- **The scan's report is attached to the item's Work Order as an
  artifact, not committed.** It names each finding, the commit, the
  scanner's rule, and what was done. A rotated secret is named by what it
  was (its kind and owner), never by its value. If a finding is a live
  secret this item cannot rotate itself, it is an out-of-band act for Ed
  (below), and the flip waits on it.
- **The DCO applies to every pull request, the train's own included.**
  ADR-013 D3 item 2 says every commit of a pull request carries
  `Signed-off-by:` and CI checks it. So AGENTS.md's git rules gain the
  sign-off, in the change that adds the DCO check, and every commit this
  train makes from that change on is signed off. The check has no
  exemption list.
- **`--version`'s maker line** names Kottera and nothing else: no product,
  no URL, and `--version --json` gains no member (ADR-013 D2 item 2). Its
  wording is the item's, under a new ADR-008 amendment allotted at
  breakdown. Nothing else the binary prints, no help topic and no part of
  the skill names Kottera or tera.
- **AGENTS.md is self-contained** (ADR-013 D3 item 4). Where it points at
  a document in `kottera/kottera` for a rule (today the naming rules), it
  states the rule itself. Its "Other repositories" section may still name
  `kottera/kottera` as where tera lives; that is the code rule of D2 item
  3, not a dependency.
- **AGENTS.md's "The repository is private for now" stays** until the
  flip, in Wave 3.
- **Amendment numbers are allotted at breakdown** (plan §2, after
  PLAN-2026-10-metais-ux §2). ADR-008's latest is A13.

## Scope — what belongs elsewhere

- **Wave 2:** the site as a METAIS package, its build step and its
  generated reference pages; the landing, install and guide pages;
  `install.sh`; `release.yml`, the formula generator and the builder's
  prerelease rule; `docs/dev/release.md`'s rewrite; anything in
  `kottera/homebrew-tap` or `kottera/kottera.com`; UX scenarios I1 and I2.
  This wave's CI may run the site's existing build and tests, and leaves a
  place for the install tests Wave 2 adds; it does not write them.
- **Wave 3:** GitHub's repository settings, DNS, the flip, the release
  candidate, and AGENTS.md's public-project rule.
- **Never in this train's items:** pushing a `v*` or `spec-*` tag,
  publishing a release, changing the repository's visibility, or
  dispatching a workflow other than to prove one this wave adds.

## What must not break, and how it is checked

- **Everything the wave tier checks.** The module path item runs
  `gate.sh wave` before it merges. Every other item runs the scoped tier.
  The wave's verification runs the wave tier on a fresh clone of the
  merged tree.
- **The generated files.** After the module path change and after every
  rebase onto it, `regenerate.sh` runs and what it changes is committed;
  every drift test is green.
- **No new dependency enters the binary's closure.** The workflows
  tripwire is a test in `internal/guarantees` and uses only what the
  module already has. Seen red first against a workflow naming
  `self-hosted`, then against `pull_request_target` and against a missing
  top-level `permissions` block.
- **Each workflow this wave adds is proved by one dispatched run**, linked
  in its pull request. That is the only run any item starts: private
  minutes are spent only on proving runs (plan §2). The macOS job costs
  more minutes than the Linux one, so it is proved once, not iterated on;
  debug a red on Linux first where the failure is not platform-specific.
- **`metais --version` and `--version --json`.** The help golden changes
  by the one line and the JSON form is byte-identical to today's.

## Out-of-band acts of this wave

Each is a person's act and is filed as an off-tera Work Order. None of
this deployment's declared off-tera types was made for these acts;
`design-session` (inputs `topic`: the act; `context`: what is asked,
what to report, and the plan row it comes from) is the nearest fit for a
person's decision or setup, and is the type to use unless a better-fitting
one is declared when you file.

| Act | When | Before | Notes |
|---|---|---|---|
| Create `kottera/homebrew-tap` and `kottera/kottera.com`, both public, and grant the dev pipeline's credential push on both | Wave 1 | Wave 2's items that write those repositories | Plan §6, ADR-013 D1 item 3, D2 item 4. Nothing in this wave depends on it, but a later wave cannot take an edge to a Work Order this wave filed, so **this wave does not close before the act has ended**: wait on it, with no item depending on it. Ask Ed to report both repositories' URLs and that a push from the credential succeeded. |
| USPTO and EUIPO knockout search for METAIS, classes 9 and 42 | any time before the flip | the flip (Wave 3) | Plan §6, R-PUB-8, ADR-013 D4 item 4. Filed now so Ed has the lead time; **carried, not waited on**. A live conflict in software stops the train, so ask Ed to report what was searched and what was found. Name its id in the closeout; the plan orchestrator carries it to Wave 3. |
| Confirm T-A3 (`01a15ac4-4d4e`) was recalled, edited and released as plan §6's first row says | at the plan's merge, already past | T-A3 can start | Plan §6. It belongs to the v1 train and is Ed's (or the design session's with his leave); neither this wave nor the plan orchestrator can see or edit it. Filed so the act has a reminder: ask Ed to confirm, or do it — keep every existing `depends_on` edge, change its focus to R-PUB-14's procedure (tag in its clone, build, compare, draft, push nothing), and say not to start before this train's Wave 3 verification. **Carried, not waited on.** |
| Rotate a live secret the scan finds that an item cannot rotate itself | only if the scan finds one | the flip (Wave 3) | ADR-013 D4 item 2. Never put the secret in the act's inputs: name its kind, owner and the commit. Carried, and named in the closeout, because the flip must wait on it. |

## Closeout

*Not yet written. The wave orchestrator replaces this line with the
wave's closeout.*

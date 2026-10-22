# PLAN-2026-10: METAIS public — the module path, the site, the install routes and the release pipeline, rehearsed end to end before v1.0 is published

**Author:** drafted in the 2026-10-22 design session with Ed Funnekotter
**Date:** 2026-10-22
**Feature:** `metais-public`. Every brief, Work Order and pull request of
this train inherits it verbatim: `metais-public [W2-C3]: <action>` for a
Work Order, and `<type>(metais-public): [W2-C3] <change>` for a pull
request and its squash subject, by the naming rules AGENTS.md adopts.
**Status:** Ratified 2026-10-22 by Ed Funnekotter, with the rulings and drafting decisions of §4.
**For PRD:** [PRD-001](../prd/PRD-001-metais-v1.md) R17 (the golden paths),
for installing the tool, and SC1, which presumes a harness can get the
binary.
**For ADR:** [ADR-013](../adr/ADR-013-the-public-project-and-its-distribution.md)
(new), with the amendments it proposes:
[ADR-010](../adr/ADR-010-the-implementation.md) A8,
[ADR-011](../adr/ADR-011-the-specification-document-and-conformance.md) A8
and [ADR-008](../adr/ADR-008-the-command-line-and-its-machine-output.md)
A13.
**Repository:** `kottera/metais`, branch `main`. Two items write elsewhere,
each named in its wave: the tap's CI in `kottera/homebrew-tap`, and the
kottera.com page in `kottera/kottera.com` (ADR-013 D2 item 4).
**Depends on:** nothing. PLAN-2026-10-metais-v1's waves and
PLAN-2026-10-metais-ux are complete. The dependency runs the other way:
**the v1.0 release, T-A3, waits for this train's Wave 3** (R-PUB-12).

## 1. Purpose

METAIS v1 is built, and nobody outside Kottera can get it. The repository
is private, nothing is hosted at metais.org, and the only install route is
building from a clone. PLAN-2026-10-metais-ux showed that a session with
`metais` and a package in front of it finds its way. It also showed that a
session without either never finds METAIS at all (R-W3-5). That is a
publishing problem, and this train solves it.

It is also Kottera's first public product. A developer who installs METAIS
and likes it should learn who made it, and where tera is going, without the
specification or the tool turning into an advertisement (ADR-013 D2).

**Done means:**

- `kottera/metais` is public, with its history, after a secrets scan and a
  name search found nothing that stops it.
- The module is `metais.org/metais`. `go install
  metais.org/metais/cmd/metais@latest` works.
- `brew install kottera/tap/metais` and `curl -fsSL
  https://metais.org/install.sh | sh` install the same archives the
  release procedure built twice and compared. The GitHub release carries
  them with `SHA256SUMS`.
- metais.org serves the landing page, the install page, two guides,
  reference pages generated from the binary, the specification and the
  schemas at their `$id`s. Every page names Kottera as the maker.
- kottera.com serves a page about tera.
- An outside contributor can read a contributor guide, sign off under the
  DCO, and get CI on GitHub-hosted runners. No stranger's code ever runs
  on Kottera's hosts.
- `v1.0.0-rc.1` went through every step: draft, Ed's approval, tags, tap,
  site, and installs on clean machines. Then `v1.0.0` went through the
  same steps.
- UX scenario I1, "install metais from metais.org and render this file",
  passes on Opus, Sonnet and Haiku against the live site. I2, discovery
  with nothing installed, is recorded.

## 2. Execution model

PLAN-2026-10-metais-v1 §2 applies whole:
- waves run one at a time, and the items inside a wave run in parallel;
- each wave's brief is written late by the plan orchestrator, as
  `docs/briefs/brief-plan-2026-10-metais-public-waveN.md`;
- shared seams and statement numbers are allotted at breakdown.

So does its harness policy: Claude throughout, and the dev pipeline's types
by full key. So does PLAN-2026-10-metais-ux §2's rule that ADR amendment
numbers are allotted at breakdown.

Rules this train adds:

- **The module path change runs alone, first.** It touches every import,
  so Wave 1's other items depend on it and rebase onto it.
- **An act is irreversible once it is public.** The flip, a pushed tag, a
  package on the Go proxy and a formula in the tap cannot be taken back.
  Each happens only in the wave, and in the order, that §8 and §9 give.
  No item pushes a `v*` or `spec-*` tag: only `release.yml` does, after
  Ed's approval (R-PUB-14).
- **Private minutes are spent only on proving runs.** Until the flip, CI
  and the install tests run only on `workflow_dispatch` (ADR-013 D5 item
  5). Each item that adds a workflow proves it with one dispatched run,
  and links it in its pull request.
- **An item that writes another repository** names that repository in its
  inputs. It follows that repository's own conventions, and writes no file
  in `kottera/metais` that names tera (ADR-013 D2 item 3).
- **Copy for people is approved by Ed.** The landing page, the guides'
  prose and the kottera.com page are drafted by items and merged after
  Ed's review. The reference pages are generated and need none.

## 3. Overview and sequencing

| Wave | Objective | Depends on | Status |
|---|---|---|---|
| 0 | ADR-013, its amendments and this plan, merged to `main` | — | **complete** (#240) |
| 1 | **Foundations**: the module path; the history scan and the tree's internal details; the community files and contributor guide; CI on GitHub-hosted runners; `--version`'s maker line | 0 | — |
| 2 | **The site and distribution**: the site as a METAIS package with generated reference pages, the guides and the specification; `install.sh` and its tests; `release.yml`, the tap formula and the tap's CI; the builder's prerelease rule; the kottera.com page; UX scenarios I1 and I2 | 1 | — |
| 3 | **Rehearsal**: the name search, Ed's settings and DNS, the flip; `v1.0.0-rc.1` end to end; I1 against the live site; a fix item for each defect, and a new candidate until it is clean | 2 | — |
| 4 | **Launch**: T-A3 cuts `v1.0.0`, Ed approves it, and every route is checked live; the train's close | 3, and T-A3's own dependencies | — |

Wave 3 is the last wave that can change bytes. Wave 4 publishes what Wave 3
proved.

## 4. Rulings

Made by Ed in the 2026-10-22 session:

- **R-PUB-1, the home.** The repository stays `kottera/metais`, and the tap
  is `kottera/tap/metais`. The GitHub login `metais` is taken.
- **R-PUB-2, Kottera on the site only.** The site names its maker on
  every page, and its landing page says who Kottera is and what tera is.
  The specification, the binary, its help and the skill stay
  vendor-neutral. The one exception is `metais --version`, which ends with
  an attribution line (ADR-013 D2).
- **R-PUB-3, a kottera.com page in this train.** It gives the "made by
  Kottera" link somewhere to go. It is a minimal page: what tera is, its
  0.x preview, and how to follow along. It becomes the "try tera" path
  once tera can be installed publicly.
- **R-PUB-4, the module path is `metais.org/metais`.**
- **R-PUB-5, metais.org is Kottera's**, in its Route 53 account, verified
  for the `kottera` organization's Pages. This closes
  PLAN-2026-10-metais-v1 §6's "Register metais.org".
- **R-PUB-6, the specification stays Apache-2.0.**
- **R-PUB-7, the history is kept.** Scan the whole history for secrets and
  rotate anything found. Clear the current tree of host names, the
  tailnet URL and notes about a person's machine. Work Order IDs stay.
- **R-PUB-8, the name is kept.** Ed's USPTO and EUIPO knockout search
  gates the flip.
- **R-PUB-9, the kottera.com page lives in a new public repository,
  `kottera/kottera.com`.**
- **R-PUB-10, the published archives are the release procedure's, as
  built.** There is no second build on GitHub's runners, and so no
  build-provenance attestation. `install.sh` checks `SHA256SUMS` only.
- **R-PUB-11, the tap is updated by a direct push**, from a `release`
  environment that requires Ed's approval. That approval is his release
  act.
- **R-PUB-12, v1.0.0 is cut only when all known work is done.** "I am not
  rushing this." T-A3 depends on Wave 3's verification.
- **R-PUB-13, contributions are signed off under the DCO**, not a CLA.
- **R-PUB-14, nothing is public before Ed approves.** A pushed tag is
  already a publication on the Go route. So the release procedure tags
  only in its own clone and attaches its archives to a draft release, and
  `release.yml` pushes the tags after the approval.

Agreed in principle before the session, and carried by ADR-013 as stated
there:
- CI on GitHub-hosted runners only;
- the community files;
- the install routes and `install.sh`'s contract;
- no goreleaser, and no code signing for 1.0;
- the site built by `metais` itself, with generated reference pages.

### Decided in drafting, for Ed's review of this plan

- **ADR-011 A8 item 5.** A prerelease of the tool may implement a draft
  revision. Today's builder refuses any version whose `implements` names
  a revision without a `spec-<M>.<m>` tag. Without this rule, the release
  candidate could only be built after the specification's release commit,
  and a defect the rehearsal found in the text would need an erratum.
- **The draft specification at `/spec/draft/`** (ADR-011 A8 item 2), not
  indexed, until `spec-1.0` exists.
- **The Markdown conversion** of the specification's chapters is a build
  step outside the package and outside the binary (ADR-013 D8 item 1).
  METAIS gains no Markdown filter.
- **I1 gates on all three models, and I2 is recorded, not gated**
  (ADR-013 D9).

## 5. Wave 1 — foundations

**Goal:** a tree that can be made public, and that an outsider can
contribute to, with nothing yet published.

**What the brief carries (the derivation is the wave orchestrator's):**

- **The module path** (ADR-010 A8 item 1), alone and first: `go.mod`, every
  import, every `go.mod` under `testdata/`, `layers.json`, the builder's
  `-X` flag, the committed baselines, the live documents and skills that
  name the path, and every golden that `regenerate.sh` rewrites. Also the
  site's two `go-import` pages under `site/public/metais/`, with a test
  holding them to ADR-013 D1 item 2. Briefs, closeouts and implementation
  logs are records, and keep the path they were written with. The wave
  tier runs before it merges, because `go.mod` is a shared seam.
- **The history scan and the tree's internal details** (ADR-013 D4 items 2
  and 3). The whole history goes through a maintained scanner. Each
  finding is either rotated, with the rotation named in the item's report
  (never the secret), or shown to be a false positive. Then the current
  tree is cleared of host names, the tailnet URL and notes about a
  person's machine. AGENTS.md is made self-contained (ADR-013 D3 item 4).
  The report lists what was found and what was done, and is attached to
  the Work Order, not committed.
- **The community files and the contributor guide** (ADR-013 D3):
  `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, the issue
  templates and the pull request template.
- **CI** (ADR-013 D5, ADR-010 A8 items 4 and 5): `.github/workflows/ci.yml`,
  the DCO check, and the workflows tripwire in `internal/guarantees`, seen
  red first against a workflow naming `self-hosted`. Also AGENTS.md's three
  tiers, in place of "There is no CI". Jobs run on `workflow_dispatch`
  until the flip, and the item proves each with one run.
- **`--version`'s maker line** (ADR-013 D2 item 2), under an ADR-008
  amendment allotted at breakdown, with its help golden.

### Rulings made during Wave 1

Made by the plan orchestrator in answer to the wave's breakdown
(2026-10-22). They bind later items of their kind:

- **R-W1-1, contact addresses.** No item invents an address. Ed names
  the Code of Conduct's report address in the act that creates the two
  repositories, and the wave answers W1-3 from his report.
  `SECURITY.md` names no email. Its route is GitHub's private
  vulnerability reporting (ADR-013 D3 item 3), which Ed turns on in
  Wave 3.
- **R-W1-2, proving a workflow while private.** No job runs on its own
  before the flip: no `push`, `pull_request` or `schedule` trigger
  (plan §2, ADR-013 D5 item 5). The only runs are deliberate dispatches,
  each proving a job, and every job a workflow defines is proved by a
  linked run before its wave closes. The run is linked in the pull
  request where it can be. GitHub dispatches only a workflow that exists
  on the default branch. Where that rule forbids a run before merge, the
  item merges the workflow and then dispatches it, links the runs on its
  merged pull request and in its report, and fixes a red in a follow-up
  pull request of the same item before it reports complete.
- **R-W1-3, acts filed by the plan orchestrator.** The name search
  (`01a16409-e851`) and the T-A3 confirmation (`01a16409-f3eb`) are the
  plan orchestrator's own children, filed early for Ed's lead time. No
  wave waits on them. Wave 3's flip waits on the name search.
- **R-W1-4, CI's tests on a 2-core runner.** CI runs the wave tier's
  list (ADR-010 A8 item 4): every test, under `-race`, with `go test`'s
  default timeout. A package whose one binary cannot finish within that
  timeout on a GitHub-hosted runner is split into several invocations by
  `-run`, as R-W6-3 split the SC7 reach. Today that package is
  `internal/conformance`, split by area. The split lives in `ci.yml`
  alone. `gate.sh`, its two tiers and AGENTS.md's tier rules do not
  change, and no timeout is raised. The groups are derived when the job
  runs, or a test holds them to the full set, so an area added later is
  never silently skipped. No larger runner is bought. The split is not
  held open waiting for PLAN-2026-10-metais-test-economy, which has not
  started. When that train's Wave 2 changes the runner, the item that
  changes it keeps `ci.yml` true and may remove the split. The reason it
  cannot wait: Wave 3's branch protection requires CI, so a job that
  cannot go green would block every pull request after the flip.
- **R-W1-5, the sign-off identity.** Every commit this train makes signs
  off as `Edward Funnekotter <3941907+efunneko@users.noreply.github.com>`,
  set explicitly on the command line, never taken from the host's git
  configuration: a dispatch host without `user.email` signs off with an
  address built from its host name, which ADR-013 D4 item 3 keeps out of
  new work. Commits already merged keep what they carry (D4 items 1 and
  3). The DCO check requires a `Signed-off-by:` on every commit (D3 item
  2), and is not made stricter than that.

## 6. Out-of-band acts

Each is filed as an off-tera Work Order, placed where **Before** says, and
the items that need it get a `depends_on` edge to it.

| Act | When | Who | Before | Notes |
|---|---|---|---|---|
| Recall T-A3, edit it, release it | at this plan's merge | Ed, or the design session with his leave | T-A3 can start | T-A3 (`01a15ac4-4d4e`) belongs to v1's train, so this train's orchestrator cannot edit it. Keep every existing `depends_on` edge. Its focus changes to R-PUB-14's procedure: tag in its clone, build, compare, draft release, push nothing. It also says not to start before this train's Wave 3 verification. |
| Add Wave 3's verification to T-A3's `depends_on` | when Wave 3's verification is filed | the plan orchestrator, or Ed | Wave 4 | R-PUB-12. If the orchestrator lacks the grant, it parks and asks Ed. |
| Create `kottera/homebrew-tap` and `kottera/kottera.com` (public), and grant the dev pipeline's credential push on both | Wave 1 | Ed | the Wave 2 items that write them | ADR-013 D1 item 3, D2 item 4 |
| Approve the kottera.com page's copy, and the landing page's and guides' prose | Wave 2 | Ed | those items' merge | §2 |
| USPTO and EUIPO knockout search for METAIS, classes 9 and 42 | any time before the flip | Ed | the flip | R-PUB-8. A live conflict in software stops the train. |
| GitHub settings on `kottera/metais`: Pages from Actions with the custom domain `metais.org`; branch protection on `main`; private vulnerability reporting; approval for first-time contributors' workflows; read-only default token; the `release` environment with Ed as required reviewer, holding `TAP_TOKEN` | Wave 3 | Ed | `v1.0.0-rc.1` | ADR-013 D5 item 4, D7 items 3 and 5. `TAP_TOKEN` is a fine-grained token with contents write on `kottera/homebrew-tap` only. Pages on `kottera/kottera.com` likewise. |
| DNS: metais.org's apex `A` and `AAAA` records and `www` to Pages; kottera.com's Pages records and its organization verification `TXT` | Wave 3 | Ed, with Kottera's DNS credentials | `v1.0.0-rc.1` | `docs/dev/release.md`. Record each change where Kottera records its DNS. Leave every existing record, mail included. |
| Make `kottera/metais` public | Wave 3, after the name search, the settings and Wave 2's verification | Ed | the release candidate | The train's first irreversible act |
| Approve the `release` environment for `v1.0.0-rc.1` | Wave 3 | Ed | the rehearsal's checks | R-PUB-11 |
| T-A3, then approve the `release` environment for `v1.0.0` | Wave 4 | Ed | the launch checks | R-PUB-12, R-PUB-14 |

## 7. Wave 2 — the site and distribution

**Goal:** everything a release needs to reach people, built and tested,
with nothing published.

**What the brief carries:**

- **The site** (ADR-013 D8, ADR-008 A13, ADR-011 A8 items 1 to 3):
  - the site as a METAIS package, and the build step that exports its
    inputs. That step is a tool outside the binary, under an ADR-009
    amendment for its host access and an ADR-010 D3 note for its Markdown
    library, both allotted at breakdown. `site/build.sh` keeps its
    contract: one command, from a commit, into a new directory;
  - the reference pages, generated per ADR-008 A13;
  - the landing page with "Who makes METAIS", the footer, and the install
    page (ADR-013 D2 item 1, D6);
  - "Using a package" (outlined by C0 to C5) and "Writing a package"
    (outlined by A1 to A3);
  - the specification at `/spec/draft/` until `spec-1.0`, and the rule
    that serves `/spec/<M>.<m>/` from its tag;
  - `pages.yml` deploying on a release's `published` event;
  - the site's tests (ADR-013 D8 item 4).

  The derivation decides how the site is split into items. The generator
  comes before the pages it generates.
- **`install.sh`** (ADR-013 D6) at `site/public/install.sh`, with
  `shellcheck` and real installs on `ubuntu-latest` and `macos-latest`
  against a release built in the job and served locally. Its refusals are
  tested too: a hash mismatch, a missing line, an unknown platform.
- **Publishing** (ADR-013 D7, ADR-010 A8 items 2 and 3, ADR-011 A8 item 5):
  - `release.yml`;
  - the formula generator with its golden test;
  - the builder's prerelease rule, with both sides pinned;
  - `docs/dev/release.md` rewritten as the procedure T-A3 and the release
    candidate follow.

  `release.yml` is proved with a dry run that publishes nothing: on a
  draft in the private repository, with its push steps disabled by an
  input that exists for the dry run alone. On GitHub's free plan, an
  environment's required reviewers apply only to a public repository. So
  the approval gate is first exercised by `v1.0.0-rc.1`, and the item
  says so in its pull request.
- **The tap's CI**, in `kottera/homebrew-tap`: `brew audit` and
  `brew test` on each push, and a README naming the formula.
- **The kottera.com page**, in `kottera/kottera.com` (ADR-013 D2 item 4),
  drafted from `kottera/kottera`'s `docs/VISION.md` and
  `docs/gtm/strategy.md`, and approved by Ed.
- **UX scenarios I1 and I2** (ADR-013 D9), with their outcome checks in the
  scorer, each check seen red against a recorded transcript where it
  should fail. The scenarios can only run once the site is live, so they
  run in Wave 3. The item finds out whether a nested session in a dispatch
  can reach the network for `curl` and `brew`. If it cannot, the runs
  become an off-tera act run by hand outside the dispatch sandbox, and the
  item says so.

## 8. Wave 3 — rehearsal

**Goal:** proof that every route works, on a release candidate, before
anything permanent carries the name `v1.0.0`.

**What the brief carries:**

- **The order:** the name search and the settings, then Wave 2's
  verification, then the flip (§6). Nothing in this wave runs before the
  flip except what §6 places there.
- **AGENTS.md after the flip.** Its "The repository is private for now"
  becomes the public project's rule.
- **`v1.0.0-rc.1`** by the release procedure of `docs/dev/release.md`:
  tag in the clone, build twice, compare, draft. Then Ed approves, and
  `release.yml` pushes the tag, publishes the prerelease and updates the
  tap.
- **The rehearsal's checks**, each recorded with its evidence:
  - `brew install kottera/tap/metais` on a clean macOS;
  - `install.sh` on clean Ubuntu and macOS, with and without
    `METAIS_VERSION`;
  - `go install metais.org/metais/cmd/metais@v1.0.0-rc.1`;
  - the manual route: download, `sha256sum -c`, run;
  - `metais --version` from each route reporting `1.0.0-rc.1`;
  - every `$id` resolving as JSON, the `go-import` pages, the reference
    pages, the specification, and kottera.com;
  - `metais skill` after each install.

  The clean machines are GitHub-hosted runners driven by a dispatched
  workflow, and a container with nothing installed.
- **I1 against the live site**, on Opus, Sonnet and Haiku, and I2
  recorded. Both are published as a scorecard, as in
  PLAN-2026-10-metais-ux.
- **A fix item for each defect**, and a new candidate (`rc.2`, …) through
  the same steps, until a candidate passes every check.

## 9. Wave 4 — launch

**Goal:** `v1.0.0`, published by Ed's one approval.

- **T-A3** runs as edited (§6): the specification's release commit and
  the matrix, then the tool, tagged in its clone, built twice, compared,
  and drafted. Its result records `SHA256SUMS`.
- **Ed approves** the `release` environment. `release.yml` pushes
  `v1.0.0` and `spec-1.0`, publishes the release, and updates the tap.
  `pages.yml` deploys the site, with `/spec/1.0/`.
- **The launch checks:** Wave 3's checklist against `v1.0.0`.
- **Close:** the plan's status and `docs/INDEX.md`, the
  implementation-log entries for ADR-013 and each amendment, and
  PLAN-2026-10-metais-v1 §6's release row marked done.

## 10. What this plan does not cover

- Homebrew core, Windows packages and other package managers.
- Apple code signing and notarization, and a signature over `SHA256SUMS`
  (ADR-013 D7 item 6).
- A guide showing METAIS inside tera. It waits for tera's METAIS
  integration and a public tera (ADR-013 D2 item 1).
- kottera.com beyond its one page.
- An MCP server for metais.

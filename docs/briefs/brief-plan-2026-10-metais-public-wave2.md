# Brief — PLAN-2026-10-metais-public, Wave 2: the site and distribution

**Plan:** [PLAN-2026-10-metais-public](../plans/PLAN-2026-10-metais-public.md), §7
**Feature:** `metais-public`. Work Orders are titled `metais-public [W2-<item>]: <action>`;
pull requests and their squash subjects are `<type>(metais-public): [W2-<item>] <change>`
(AGENTS.md, Git and pull requests).
**Repository:** `kottera/metais`, branch `main`. Every Work Order of this wave
gets `repo: kottera/metais` and `branch: main`. Two items write elsewhere and
name that repository in their task: the tap's CI in `kottera/homebrew-tap`,
and the page in `kottera/kottera.com` (plan §2, ADR-013 D2 item 4).
**Gated on:** Wave 1, complete. Its verdict is under plan §5 (#261), and its
closeout is in [brief-plan-2026-10-metais-public-wave1.md](brief-plan-2026-10-metais-public-wave1.md).
Both other repositories exist (W1-OOB1). The gate has cleared.

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

Everything a release needs to reach people is built and tested, and
nothing is published. When it is done:

- **The site is a METAIS package**, rendered by the binary built from
  the commit being published. A build step outside the binary exports
  its inputs: the help, the skill, the specification's chapters as HTML
  fragments, and the release's version and archives. `site/build.sh`
  keeps its contract: one command, from a commit, into a new directory.
- **The site's pages exist:**
  - the reference pages, generated from `metais help --json` and
    `metais skill --print` (ADR-008 A13);
  - the landing page with "Who makes METAIS", the footer on every page,
    and the install page (ADR-013 D2 item 1, D6);
  - the two guides, "Using a package" and "Writing a package";
  - the specification at `/spec/draft/` until `spec-1.0` exists, and the
    rule that serves `/spec/<M>.<m>/` from its tag (ADR-011 A8 items 1
    to 3).

  The prose copy (the landing page and the guides) is approved by Ed.
- **The site's tests hold it** (ADR-013 D8 item 4): every `$id` to its
  file; every `go-import` page to D1 item 2; every reference page to the
  help it came from; every internal link to a page that exists.
- **`pages.yml` deploys on a release's `published` event**, from the
  released tag (ADR-013 D7 item 3.4).
- **`install.sh` is at `site/public/install.sh`**, to ADR-013 D6's
  contract. It is checked by `shellcheck` and installed for real on
  `ubuntu-latest` and `macos-latest`, against a release built in the job
  and served locally. Its refusals are tested.
- **Publishing is ready, and nothing is published:**
  - `release.yml`;
  - the formula generator and its golden test;
  - the builder's prerelease rule, with both sides pinned (ADR-011 A8
    item 5);
  - `docs/dev/release.md`, rewritten as the procedure that T-A3 and
    Wave 3's release candidate follow.

  `release.yml` is proved by a dry run that publishes nothing.
- **`kottera/homebrew-tap` has its CI and a README.** CI runs
  `brew audit` and `brew test` on each push.
- **The kottera.com page is written and approved** in
  `kottera/kottera.com`, ready for Pages once Ed's Wave 3 settings exist.
- **UX scenarios I1 and I2 exist** (ADR-013 D9), with their outcome
  checks in the scorer, each seen red against a recorded transcript where
  it should fail. They run in Wave 3, against the live site.

## What to read first

- Plan §1, §2 (this train's rules: irreversible acts, private minutes,
  another repository, copy approved by Ed), §4, §6, §7 (this wave), and
  §5's rulings and verdict.
- [ADR-013](../adr/ADR-013-the-public-project-and-its-distribution.md):
  - D1 item 5, the site's URL space and who owns each path;
  - D2, Kottera on the site and nowhere else;
  - D6 whole, the install routes and `install.sh`;
  - D7 whole, releasing and publishing;
  - D8 whole, the site;
  - D9, the I1 and I2 scenarios.
- ADR-008 A13, the help and the skill as the source of the reference
  pages, and D15 and D16.
- ADR-011 A8, the specification's home and the prerelease rule; D5,
  releasing a revision.
- ADR-010:
  - D12 and A8 items 2 and 3, the builder and the formula;
  - D3, the dependency table, and **A3**, below;
  - A5 and A8 item 4, the gate tiers and CI.
- ADR-009 A2 and A8, how a tool outside the binary is allowed host
  access, which the site's build step follows.
- In the tree:
  - `site/` whole and `.github/workflows/pages.yml`;
  - `.github/workflows/ci.yml` and `.github/scripts/`;
  - `internal/tools/release/`;
  - `docs/dev/release.md`;
  - `docs/dev/ux-testing.md` and `internal/tools/ux/`;
  - `CONTRIBUTING.md`.

## What Wave 1 learned that changes this one

- **The pipeline cannot dispatch a workflow.** Ed ruled in W1-OOB2 that
  he dispatches every proving run himself, and grants the pipeline no
  Actions access. GitHub also dispatches only a workflow that exists on
  the default branch. So every workflow job this wave adds or changes
  follows R-W1-2:
  - merge the workflow, then have Ed dispatch it, through an act filed
    for that run;
  - link the run;
  - fix a red in a follow-up pull request of the same item.

  Plan for the round trip at breakdown. Wave 1 discovered it late, and
  it cost two acts and a follow-on item.
- **CI's conformance tests are split by area in `ci.yml`** (R-W1-4). The
  install jobs and the site job this wave adds go beside them, and leave
  the split as it is.
- **The sign-off.** Every commit signs off exactly as R-W1-5 says, set
  on the command line, never from the host's git configuration. No commit
  or pull request carries a `Co-authored-by` line: CI fails it (#254),
  and AGENTS.md forbids it. Hand both rules to every item, including the
  two that write other repositories.
- **Both other repositories exist**, public, each with one empty root
  commit. The pipeline's credential can push to both (W1-OOB1). Neither
  has conventions yet. The item that writes first sets the minimum each
  needs (a README, the Apache-2.0 licence, its CI), states it in that
  repository's README, and does nothing more.
- **Still running beside this wave:** W1-NB1 (`01a16904-f5e9`), which
  edits CONTRIBUTING.md and `site/site_test.go`'s `TestGoImportPages`,
  and W1-OOB5 (`01a16905-12da`). Expect W1-NB1's changes in those files
  and keep them on rebase.

## Rulings — do not reopen

- **R-PUB-1 to R-PUB-14** and every ADR-013 decision are settled. In
  particular:
  - no goreleaser;
  - no second build on GitHub's runners and no attestation (R-PUB-10):
    `install.sh` checks `SHA256SUMS` only;
  - the tap is updated by a direct push, from the `release` environment
    (R-PUB-11);
  - no tag is pushed before Ed approves (R-PUB-14);
  - METAIS gains no Markdown filter;
  - the Markdown conversion is input preparation outside the package and
    outside the binary (ADR-013 D8 item 1, plan §4).
- **Nothing is published in this wave.** Nothing in this wave:
  - pushes a `v*` or `spec-*` tag;
  - publishes a release;
  - deploys Pages;
  - pushes a formula to `kottera/homebrew-tap`'s default branch.

  A formula in the tap is plan §2's irreversible act, and the first one
  arrives from `release.yml` at `v1.0.0-rc.1`. The tap's CI is proved
  without one on its default branch. How is the item's: a pull request in
  the tap that carries a test formula and is closed unmerged is one way.
  A draft GitHub release in the private `kottera/metais` publishes
  nothing, and is what `release.yml`'s dry run runs against.
- **The site's Markdown library and ADR-010 A3.** ADR-010 A3 removed
  goldmark from the module because its parser is not linear on hostile
  input (ADR-009 D10 item 4), and A3 item 5 says no Markdown module is in
  `go.mod`. ADR-013 D8 item 1, ratified later, has the site's build step
  use a Markdown library named in an ADR-010 D3 note. The two agree on
  what must be true:
  - the library never enters `cmd/metais`'s closure, and `outline` stays
    METAIS's own reading (A3 item 3);
  - the input it converts is this repository's own specification, not an
    untrusted package, so D10 item 4's linearity does not bind it.

  Whether the library sits in the main module or in a separate module
  beside the tool is the item's design. Its ADR-010 note states which,
  and says how A3 item 5's sentence reads after it. Either way:
  - the layering tests and `TestNoHostFacts`' dependency scan hold the
    binary free of it;
  - `THIRD-PARTY-NOTICES.txt` stays true of what ships.
- **Copy for people is approved by Ed before it merges or is pushed**
  (plan §2). That means the landing page, the guides' prose and the
  kottera.com page. The reference pages, the footer and the install
  page's commands are generated from, or fixed by, the ADRs, and need no
  approval. Two cases differ:
  - **The landing page and the guides** live in `kottera/metais`, which
    is private until the flip. A draft pull request there is not
    publication.
  - **The kottera.com page** goes to a public repository. A branch or
    pull request there is already published. So nothing of it is pushed
    to `kottera/kottera.com` before Ed has approved the copy. The draft
    reaches him through the act, as an artifact. It is drafted from
    `kottera/kottera`'s `docs/VISION.md` and `docs/gtm/strategy.md`,
    which are private. So the draft carries nothing from them that Ed has
    not approved for publication.
- **The vendor-neutrality line.** The site's footer and its landing
  page's "Who makes METAIS" section are ADR-013 D2 item 1's place for
  Kottera and tera. No other file in `kottera/metais` names tera.
  `TestVendorNeutral` keeps holding the help, the topics and the skill.
  Since the reference pages are generated from those, they stay neutral
  too. The site's layout around them is not under that test.
- **A reference page never scrapes the text help** (ADR-008 A13 item 4).
  If the generator needs something `metais help --json` does not carry,
  that is an ADR-008 amendment adding it to `metais.help/1` with its
  schema. Amendment numbers are allotted at breakdown. ADR-008's latest
  is A14.
- **The release's archives on the site.** `install.sh` and the install
  page take archives and `SHA256SUMS` from the GitHub release. Only the
  script is served from metais.org (D6 item 5).
- **`install.sh` with no `METAIS_VERSION` installs the latest release
  that is not a prerelease** (D6 item 7). Until `v1.0.0` exists, no such
  release exists: Wave 3's candidate is a prerelease, and Wave 3 runs
  `install.sh` with and without `METAIS_VERSION`. So the script's answer
  when there is no stable release must be a refusal. It says so plainly
  and names `METAIS_VERSION`, and it has a test, like its other
  refusals.
- **Amendments and their numbers** are allotted at breakdown. That
  includes:
  - the ADR-009 amendment for the build step's host access;
  - the ADR-010 D3 note for its library.

## Scope — what belongs elsewhere

- **Wave 3:**
  - GitHub's settings on both repositories, Pages and DNS;
  - the flip;
  - the scan the flip rests on;
  - `v1.0.0-rc.1` end to end;
  - running I1 and I2;
  - AGENTS.md's public-project rule;
  - adding Wave 3's verification to T-A3's `depends_on`.

  `pages.yml`'s deploy and the `release` environment's approval gate are
  first exercised at `v1.0.0-rc.1`. GitHub's free plan runs neither on a
  private repository, and the items that write them say so in their pull
  requests.
- **Wave 4:** `v1.0.0` and `spec-1.0`, `/spec/1.0/` served for real, and
  the train's close.
- **Not this train** (plan §10): Homebrew core, Windows packages, code
  signing, a guide showing METAIS inside tera, kottera.com beyond one
  page.
- **I1 and I2's runs.** The item that adds them finds out whether a
  nested session in a dispatch can reach the network for `curl` and
  `brew` (plan §7). It reports what it found. If a dispatch cannot, the
  runs become an off-tera act on Ed's host in Wave 3, and the item says
  so. It runs no scenario against a live site, because none exists.

## What must not break, and how it is checked

- **The scoped tier for every item, and the wave tier at the wave's
  verification**, on a fresh clone of the merged tree. An item that
  changes `go.mod`, `go.sum` or the gate's scripts runs the wave tier
  before it merges (AGENTS.md).
- **The binary's closure.** No new dependency reaches `cmd/metais`. The
  layering and host-facts tests are the proof.
- **Rendering is pure** (AGENTS.md). The site package is rendered by
  `metais render` like any package, with no hook, script or network. The
  build step prepares inputs and never extends the binary.
- **Byte-identical builds.** The same commit builds the same site twice.
  The release builder's twin-build comparison is unchanged by the
  prerelease rule.
- **Every workflow job added or changed has a green, linked run**,
  dispatched by Ed under R-W1-2, before this wave closes. There are two
  exceptions, each said so in its pull request:
  - `pages.yml`'s deploy, which cannot run on a private repository;
  - the `release` environment's approval, which cannot either.

  The tap's CI runs on a push in its public repository, under the
  constraint above.
- **No other repository is touched** beyond the two named, and
  `kottera/kottera` is only read.

## Out-of-band acts of this wave

Each is a person's act, filed as an off-tera Work Order. This
deployment's `design-session` type (inputs `topic` and `context`) has
served every such act so far, and is the type to use unless a better
fitting one is declared when you file.

| Act | When | Before | Notes |
|---|---|---|---|
| Approve the landing page's and the guides' prose | Wave 2 | those items' merge | Plan §2 and §6. Attach the draft, or link the private pull request. Ask Ed to report approval, or the edits he wants. One act may cover all three, or one per item. The placement is yours. |
| Approve the kottera.com page's copy | Wave 2 | anything pushed to `kottera/kottera.com` beyond the repository's README, licence and CI | Plan §2 and §6, ADR-013 D2 item 4. The draft goes as an artifact, because that repository is public (above). |
| Dispatch each new or changed workflow job once, and report the run | after the workflow merges | that item's completion | R-W1-2, W1-OOB2's ruling. One act per proving round is the pattern Wave 1 used (W1-OOB3, W1-OOB4). Covers `ci.yml`'s install and site jobs and `release.yml`'s dry run. For the dry run, the act names the draft release it runs against, and the input that turns its push steps off. |

Plan §6's rows for Wave 3 (settings, DNS, the flip, the `rc.1`
approval) and Wave 4 are not this wave's. Neither is the T-A3 edge.

## Closeout

Written by the wave orchestrator (Work Order 01a1690c-a75a-7e30-907f-2e8ff42a2039)
on 2026-10-24, against `main` at c460af70. Base ref: 5e670f8b. This section
says what landed and where every finding went. It is not a correctness
verdict.
- **The verification round's findings** below are W2-V's, with its method
  named.
- **Where this section says "reports",** the claim is the child's.
- **Where it says "I read",** I read the merged code. I mutated nothing.

### 1. What landed, and whether it matches the ask

| Item | Work Order | Pull request, merge | What landed |
|---|---|---|---|
| W2-1 | 01a16914-cf93 | #269, 0dac5ac8 | The site is the METAIS package `site/package`. `site/build.sh` runs `internal/tools/site`, which builds `metais` from the commit, exports the help, the skill, the chapters as HTML fragments and the release (`--release`), renders the package, and adds `site/public` and the schemas at their `$id`s. ADR-009 A11 gives the build step its host access. ADR-010 A10 is the D3 note for goldmark, which sits in the root `go.mod` and outside `cmd/metais`'s closure. Every page carries D2 item 1's footer, and two builds are byte-identical (`TestBuildReproducible`). |
| W2-2 | 01a16916-7994 | #292, ff0a8c98 | The reference pages are generated per command, per topic and for the skill from `help --json` and `skill --print`, and held by `TestReferencePages`. Also: `/spec/draft/` (noindex), `/spec/<M>.<m>/` from its tag (`TestSpecReleases`, on a fixture), the install page to D6, and `TestInternalLinks`. No ADR-008 amendment was needed. |
| W2-3 | 01a16915-6718 | #266, 1550d99c | The builder's prerelease rule (ADR-011 A8 item 5), with both sides pinned in `TestReleaseRefusesAnUnreleasedRevision`. Also the formula generator, `go run ./internal/tools/release formula -version <v> -sums <SHA256SUMS>`, with its golden, which `regenerate.sh` rewrites. |
| W2-4 | 01a16916-f20f | #299, 11ea8f6b; #312, 96e44725 | `release.yml`, run by dispatch in the `release` environment, with `dry_run`. `.github/scripts/release.sh` verifies, tags, publishes and pushes the tap. `pages.yml` is called by `release.yml` through `workflow_call` after publishing (R-W2-1), and keeps a `workflow_dispatch` for an erratum. `docs/dev/release.md` is rewritten as the procedure. #312 records the hand-made proofs and what waits for the flip. |
| W2-5 | 01a16917-541a | #285, 5e72fd9c | `site/public/install.sh` to D6, with gate tests of its four refusals, no stable release included. `ci.yml`'s dispatch-only `install-linux` and `install-macos` jobs run `.github/scripts/install-test.sh` against a release built in the job. `TestInstallServed` holds `/install.sh`. |
| W2-10 | 01a16916-18ab | #271, f6f0bd99 | UX scenarios I1 and I2, the runner's no-`metais`-on-`PATH` setup for them alone, and their scorer checks with fixtures seen red. `docs/dev/ux-testing.md` documents them and what the network probe found. |
| W2-11 | 01a16923-9716 | #298, c938f2ff | The approved landing page with "Who makes METAIS", and "Using a package" and "Writing a package" under `/guide/`, with Ed's edits. Also `/getting-started/` redirects, and `TestTeraOnlyWhoMakes`. It replaced W2-7. |
| W2-12 | 01a16923-dd3d | kottera/kottera.com#1, 3f52d7cb | The approved page, Version A with Ed's K edits, at `docs/index.html` with `docs/CNAME`, plus the README, the licence and CI. Pages is off. Its CI runs: 40010236810 on the pull request and 40010363486 on `main`, both green. It replaced W2-8. |
| W2-13 | 01a16924-2655 | kottera/homebrew-tap#1, 3f164193 | The README, the licence, and CI running `brew audit --strict`, install and test on ubuntu and macos. It was proved by test pull request #2, closed unmerged, whose formula is byte-identical to the generator's output for the same `SHA256SUMS`. No formula is on `main`. It replaced W2-9. |

I read these mechanisms in the merged tree, and found what the items report:
- `release.yml`'s pages job is `uses: ./.github/workflows/pages.yml`, under `if: !inputs.dry_run`;
- `pages.yml` has `workflow_call` and `workflow_dispatch`;
- `release.sh` refuses a real run on a private repository or with no required reviewer;
- goldmark is in `go.mod`.

Where an item diverged from its first ask:
- **W2-6 (01a16915-c35f) ended failed, with its work done.** I filed it as `code-implementation-direct`, whose postcondition is a merged pull request, and its task forbade the merge. Its drafts, draft PR #265 and an artifact, were delivered. Because the failure left W2-OOB1 unreleasable, I cancelled the four items that had not started:
  - W2-OOB1, re-filed as W2-OOB5;
  - W2-7, re-filed as W2-11;
  - W2-8, re-filed as W2-12, with the repository it writes as its `repo`;
  - W2-9, re-filed as W2-13, with the repository it writes as its `repo`.

  I could not recall them. The new ones got new addresses.
- **No GitHub-hosted proving run exists** for the jobs this wave added or changed. Ed ruled that the proofs are made by hand while the repository is private (rulings below). So the brief's "every workflow job added or changed has a green, linked run" is deferred to the flip and carried by W2-OOB6.
- **Items ran directed tests, not the gate,** from Ed's standing instruction onward. W2-3 and W2-5 passed the scoped tier before it. The wave tier ran once, at W2-V.

### 2. What children flagged, declined or discovered

- **W2-1's first dispatch was killed at the 3h limit** before it opened its pull request. A retry resumed from its pushed branch.
- **W2-10's scoped gate was starved repeatedly** on the shared host, with `internal/conformance` at 0.35 to 0.45 of 4 cores. It went green once, on cf0248a1, and Ed's CI wave run 39990375739 passed. It merged on directed tests under Ed's instruction.
- **ADR-008 A16 was taken** by another train before W2-2 could use it. W2-2 needed no amendment, so nothing was renumbered. Allotted and unused: ADR-009 A12 (W2-3) and A13 (W2-10), and ADR-008 A16. ADR-013 A1 stayed unallotted under R-W2-1.
- **The release environment is unsettled.** Whether a job naming `environment: release` runs on the private free-plan repository could not be settled by hand. `release.sh` refuses a real release there regardless.
- **kottera.com's Pages is set up to serve from a branch** (`main`, `/docs`). Plan §6 says "likewise" to metais.org, which serves from Actions. If Ed chooses Actions, a deploy workflow is needed (W2-12).
- **kottera.com's METAIS paragraph keeps the landing page's old tagline,** because Ed approved K5 as drafted. Ed may want the two to match (W2-12).
- **brew audit runs without `--online`** in the tap's CI, because the release URLs do not exist yet (W2-13).
- **Copy decisions:** W2-11 chose USD for M6, and renamed the template variable `amount` to `cents`.

### 3. What carries to the next wave

- **The hosted runs Ed deferred,** before `v1.0.0-rc.1`: the site job (and its linux/amd64, never covered by hand), install, shell, and `release.yml`'s dry run, with what the `release` environment does. W2-OOB6 carries them.
- **The vendor-neutrality line, read through ADR-013 D2 item 3.** The brief's "No other file in `kottera/metais` names tera" is false of the tree, but only through records and files that predate this wave: ADRs, plans, briefs, logs, scorecards, AGENTS.md's own rule, and tests that forbid the word. The one code hit is W2-NB3's. Wave 3's scan before the flip should read the rule as about the code, and a brief should say so.
- **kottera.com's Pages route,** branch or Actions, is Ed's Wave 3 setting.
- **The wave tier's margin** on the shared host is thin (below), and is PLAN-2026-10-metais-test-economy's to own.
- **What I would do differently:**
  - Never file an item whose output is unmerged under `code-implementation-direct`.
  - Read every operator ruling's text before relaying it. I learned of Ed's proving deferral from a child's report, and the plan does not record it.
  - Check at breakdown whether a later standing instruction can change the gate. Both of Ed's instructions arrived mid-wave.

### 4. What the verification round found, and where each finding went

**Who verified, and how.** W2-V (01a16b8d-2472, `wave-verification`), on `main` at 96e44725:
- **Gate:** `gate.sh wave` on a fresh clone. The first run passed every step and all 62 reach groups, except `internal/conformance` and `internal/guarantees`, which were starved at 0.57 of 5 cores. `gate.sh wave --resume` passed both: `GATE DONE: PASS (wave)`. This was the wave's first full gate.
- **Mutation:** about 110 mutation rows across W2-1, W2-2, W2-3, W2-4, W2-5, W2-10 and W2-11. Every expected-red row went red, except the rows named in the findings below and three it explained as equivalent or refused independently.
- **Inspected, not mutated:** W2-12, W2-13, W2-OOB5 and W2-OOB3. Their repositories and records were read, and their runs checked through the API.
- **Replay:** every "Using a package" command, against `metais` built from `main`.
- **Audit:** the item list against `git log 5e670f8b..main` was verified for accuracy, and its completeness trusted. Every merge is an ancestor of `main`. The only `metais-public` commit that is not an item is #264, R-W2-1's ruling.
- **Publication check:** no tag and no release exist, the dry-run draft is deleted, the tap holds no formula, and kottera.com's Pages is off.

Its 11 item entries match the 11 I sent.

**Verdict:** pass, with no blocking finding. So there is no fix child and no second round. W2-11's criteria are "partial" only through the vendor-neutrality residue below.

**Where each non-blocking finding went:**

| Finding | Where it went |
|---|---|
| W2-4: `pages.yml`'s published-release guard is untested logic (three mutations green) | W2-NB1, 01a16bb7-639b |
| W2-4: two sides of the tap rule are unpinned (rc after rc, full after full, a non-`-rc` prerelease) | W2-NB1, 01a16bb7-639b |
| W2-4: the tap can be moved back to an older line | W2-NB1, 01a16bb7-639b |
| W2-1: the install page as built with `--release` is never rendered by a test | W2-NB2, 01a16bb7-bb85 |
| W2-5: `install.sh`'s guards beyond the four refusals are untested (no hash tool, duplicate line, substring match, armv7l and i686, URL override) | W2-NB2, 01a16bb7-bb85 |
| W2-5: `ci.yml`'s selection of the install jobs is not held by a test | W2-NB2, 01a16bb7-bb85 |
| W2-11: no test runs the guides' command lines | W2-NB2, 01a16bb7-bb85 |
| W2-10: I2's finds-and-installs pass on an unnamed route has no fixture | W2-NB3, 01a16bb8-05e1 |
| W2-OOB3: shellcheck 0.11 raises SC2329 on seven of `gate.sh`'s functions, which are called by name | W2-NB3, 01a16bb8-05e1 |
| W2-11: "no file names tera" is false of the tree; the code residue is `internal/tools/ux/main.go:58` | W2-NB3, 01a16bb8-05e1, for code and live documents. Records stay, as section 3 says. |
| W2-13: `setup-homebrew@main` is unpinned | W2-NB4, 01a16bb8-26eb |
| Cross-item: no hosted run exists for the jobs this wave added or changed | W2-OOB6, 01a16bb8-56c8 |
| W2-4: what `environment: release` does on the private repository is unsettled | W2-OOB6, 01a16bb8-56c8 |
| W2-OOB3: the dry run's hosted-only skips were not exercised | W2-OOB6, 01a16bb8-56c8 |
| W2-13: the merge subject 3f164193 lacks " (#1)" | **Not filed.** AGENTS.md never renames a commit already in history. |
| W2-12: kottera.com#1 does not link its green CI run | **Not filed.** The runs, 40010236810 and 40010363486, are named in section 1, which is the record. |
| Cross-item: the wave tier fits the ten-minute bound on the shared host only with a resume (`internal/conformance` at 568 s) | **Not filed as a child.** The cause is the shared host, and the remedy is PLAN-2026-10-metais-test-economy's scope. It is recorded as observation 01a16bb8-8625. |

**Why the findings were grouped:**
- **W2-NB1:** three findings in `release.sh` and `pages.yml` and their tests.
- **W2-NB2:** four test gaps in `site/` and `install.sh`.
- **W2-NB3:** three small changes to the developer tooling.

Each child is told its members are severable.

**Wave 2 completed without waiting for W2-NB1, W2-NB2, W2-NB3, W2-NB4 and W2-OOB6, and that is the intent.** They run beside Wave 3 and are not re-verified.

### 5. Out-of-band acts

| Act | Type | Work Order | Placement | Outcome |
|---|---|---|---|---|
| Approve the landing page's, the guides' and the kottera.com page's copy | design-session | W2-OOB5, 01a16923-4dd8 | waited on by W2-11 and W2-12 | Ended by Ed. Approved all four: the landing page and the Writing guide with edits, the Using guide as written, and kottera.com's Version A with edits. He answered M1 to M7 and K1 to K6, recorded on #265 (comment 6372789288). It replaced W2-OOB1. |
| Dispatch the site job for W2-1 | design-session | W2-OOB2, 01a16918-7aea | held, then released | Cancelled outside this run under Ed's proving ruling. Ed's manual runs of the job's steps stand in (#269). |
| Dispatch `release.yml`'s dry run for W2-4 | design-session | W2-OOB3, 01a16918-8f78 | held, then released | Ended by Ed, by hand. `release.sh prepare` ran as a dry run against draft v0.0.0-alpha.1 at 11ea8f6b, and pushed and published nothing. shellcheck 0.9.0 was clean. The environment question is deferred. |
| Dispatch the install and shell jobs for W2-5 | design-session | W2-OOB4, 01a16918-9ebf | held, then released | Cancelled outside this run under Ed's proving ruling. Ed's manual runs of `install-test.sh` stand in (#285). |
| Dispatch the deferred hosted runs after the flip | design-session | W2-OOB6, 01a16bb8-56c8 | carried | Open. The plan orchestrator carries it to Wave 3. |
| Approve the copy (first filing) | design-session | W2-OOB1, 01a16917-c9bc | — | Cancelled before it ran. Its dependency W2-6 had failed, and W2-OOB5 replaced it. |

### Rulings made during this wave

- **R-W2-1** (plan orchestrator, plan §7, #264): ADR-013 D7 item 3.4's conditions bind, and the trigger is the item's. W2-4 chose `workflow_call` from `release.yml`.
- **Ed's standing instruction, 2026-10-23,** relayed by the operator: items run directed tests only and merge, and the full gate runs once at verification.
- **Ed's ruling, 2026-10-23:** while the repository is private, workflow proofs are made by hand, and GitHub-hosted runs are deferred to the public flip. It reached the children directly, and the plan does not record it. The plan orchestrator should.
- **Mine, for W2-10, before Ed's instruction:**
  - a starved gate red is not merged over;
  - a CI run does not replace the scoped tier;
  - a green scoped run before a rebase does not stand in for the `--resume` after it.

  Each follows from AGENTS.md and the verification-gate skill. Ed's instruction superseded them for W2-10's merge.

I escalated nothing. Every question was answered from the brief, the ADRs, AGENTS.md, or the operator's instructions.

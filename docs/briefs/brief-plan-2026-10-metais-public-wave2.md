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

*Not yet written. The wave orchestrator replaces this line with the
wave's closeout.*

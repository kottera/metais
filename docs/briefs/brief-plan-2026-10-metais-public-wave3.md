# Brief — PLAN-2026-10-metais-public, Wave 3: rehearsal

**Plan:** [PLAN-2026-10-metais-public](../plans/PLAN-2026-10-metais-public.md), §8
**Feature:** `metais-public`. Work Orders are titled `metais-public [W3-<item>]: <action>`;
pull requests and their squash subjects are `<type>(metais-public): [W3-<item>] <change>`
(AGENTS.md, Git and pull requests).
**Repository:** `kottera/metais`, branch `main`. Every Work Order of this wave
gets `repo: kottera/metais` and `branch: main`, except an item that writes
`kottera/homebrew-tap` or `kottera/kottera.com`. That item gets that
repository as its `repo`, as Wave 2 did.
**Gated on:** Wave 2, complete. Its verdict, and Ed's two rulings made
during it, are under plan §7 (#316). Its closeout is in
[brief-plan-2026-10-metais-public-wave2.md](brief-plan-2026-10-metais-public-wave2.md).
The name search has ended (`01a16409-e851`): no conflict, proceed. The
gate has cleared.

## Harness policy

Claude throughout, by plan §2: the dev pipeline's types, which are named
here by leaf. File every item by these types and no other. If a seat is
not available, wait for it rather than substituting another harness's
type.

| Role | Work Order type |
|---|---|
| Wave derivation and supervision | `wave-orchestrator-derive` |
| Implementation and corrective work | `code-implementation-direct` |
| Whole-wave verification | `wave-verification` |

`code-implementation-direct`'s postcondition is a merged pull request
(Wave 2's W2-6). Two kinds of item in this wave have no pull request to
merge:
- an item whose work is a draft release;
- an item whose work is a run of checks.

Give such an item a real merged deliverable, such as the rehearsal's
record in the tree. Otherwise its work is a person's act, filed
off-tera. Never file an item under a type whose postcondition its task
forbids.

## What this wave is for

Proof that every route works on a release candidate, before anything
permanent carries the name `v1.0.0`. When it is done:

- **`kottera/metais` is public**, with its history. These came first, in
  plan §8's order:
  - a scan of the history to the moment of the flip;
  - Ed's settings;
  - Wave 2's verification.
- **AGENTS.md states the public project's rule** in place of "The
  repository is private for now".
- **CI runs on every pull request and every push to `main`** (ADR-013
  D5 item 5). The hosted runs Wave 2 deferred have been made (Ed's
  ruling, plan §7).
- **A release candidate has gone end to end.** `v1.0.0-rc.1`, or the
  first `rc.N` that passes, has been through the whole procedure of
  `docs/dev/release.md`:
  - tag in the clone, build twice, compare, draft;
  - Ed approves the `release` environment;
  - `release.yml` pushes the tag, publishes the prerelease, updates the
    tap and deploys the site.
- **The rehearsal's checks passed against that candidate**, each
  recorded with its evidence (plan §8):
  - `brew install kottera/tap/metais` on a clean macOS;
  - `install.sh` on clean Ubuntu and macOS, with and without
    `METAIS_VERSION`. Without it, the result is the tested refusal,
    since no stable release exists;
  - `go install metais.org/metais/cmd/metais@v1.0.0-rc.N`;
  - the manual route: download, `sha256sum -c`, run;
  - `metais --version` reporting `1.0.0-rc.N` from each route;
  - every `$id` resolving as JSON, the `go-import` pages, the reference
    pages, the specification at `/spec/draft/`, and kottera.com;
  - `metais skill` after each install.

  The clean machines are GitHub-hosted runners driven by a dispatched
  workflow, and a container with nothing installed.
- **I1 has run against the live site** on Opus, Sonnet and Haiku, and
  I2 is recorded. Both are published as a scorecard, as in
  PLAN-2026-10-metais-ux.
- **Every defect found has its fix item**, and a new candidate went
  through the same steps, until one passed every check.
- **T-A3 depends on this wave's verification** (plan §6, R-PUB-12).

## What to read first

- Plan §2, especially "an act is irreversible once it is public"; §4;
  §6, the out-of-band table; §8, this wave; and §5's and §7's rulings
  and verdicts, including Ed's two rulings under §7.
- ADR-013:
  - D4, going public, and its item 5 order;
  - D5, CI;
  - D6, the install routes;
  - D7, releasing, especially items 3 and 8;
  - D9, the I1 and I2 scenarios.
- `docs/dev/release.md` whole. It is the procedure this wave follows,
  and its "Going live" and "What cannot run while the repository is
  private" sections.
- `docs/dev/ux-testing.md`, "Where the install scenarios can run": what
  the I1 and I2 act must provide.
- `.github/workflows/` and `.github/scripts/` as they are, and
  `CONTRIBUTING.md`.

## What the earlier waves learned that changes this one

- **I1 and I2 cannot run in a dispatch** (W2-10's probe). `brew` cannot
  run in the sandbox, and metais.org had no address. So they are an act
  on Ed's host, which needs the four things `ux-testing.md` lists. The
  third is a credential route for the nested sessions. That one may need
  a work item before the run, under its own ADR-009 amendment, if Ed's
  host cannot provide it. **Ask Ed which, early**, so a needed item is
  not discovered after the candidate is out.
- **The pipeline cannot dispatch a workflow.** Every dispatched run is
  Ed's act, the rehearsal's check workflow included. Plan each round trip
  at breakdown.
- **Ed's standing instruction on the gate** (plan §7): items run directed
  tests and merge, and the wave tier runs once, at the verification. It
  holds until he withdraws it. A fix to a release candidate is still
  verified by the candidate's own checks passing.
- **Hosted runs deferred from Wave 2** are an open act, W2-OOB6
  (`01a16bb8-56c8`), filed by Wave 2. This wave cannot take an edge to
  it. So the proof that `ci.yml`'s jobs, `release.yml` and `pages.yml`
  work on GitHub's runners must rest on this wave's own evidence: runs on
  its own pull requests once CI runs on them, or an act of its own. That
  proof comes before `v1.0.0-rc.1` is approved. Name W2-OOB6 in the
  closeout, with what it found if it has ended.
- **What `environment: release` does on a private free-plan repository
  was never settled.** After the flip it can be. `release.sh` refuses a
  real release without a required reviewer.
- **kottera.com's Pages is built to serve from a branch** (`main`,
  `/docs`). Plan §6 says "likewise" to metais.org, which serves from
  Actions. Ed chooses in his settings. If he chooses Actions, a deploy
  workflow is a small item in `kottera/kottera.com`.
- **The vendor-neutrality line is about the code** (ADR-013 D2 item 3).
  Records keep naming tera. W2-NB3 (`01a16bb8-05e1`) clears the code
  residue.
- **Still running beside this wave** are W2-NB1 to W2-NB4. Expect their
  changes in `release.sh`, `pages.yml`, `site/`, `install.sh`,
  `internal/tools/ux` and the tap's CI, and keep them on rebase. A
  candidate is cut from a commit that includes whichever of them have
  merged.

## Rulings — do not reopen

- **R-PUB-1 to R-PUB-14, every ADR-013 decision, and §5's and §7's
  rulings are settled.** In particular:
  - nothing is pushed or published before Ed approves the `release`
    environment (R-PUB-14);
  - no item pushes a `v*` or `spec-*` tag; only `release.yml` does;
  - the archives published are the procedure's own, as built
    (R-PUB-10);
  - a prerelease never pushes `spec-<M>.<m>` (ADR-013 D7 item 3.3).
- **The order of the irreversible acts** (plan §8, ADR-013 D4 item 5):
  1. the name search, which has ended;
  2. the scan to the present;
  3. Ed's settings;
  4. Wave 2's verification, which has ended;
  5. the flip;
  6. then the candidate.

  GitHub's free plan allows some of Ed's settings only on a public
  repository: Pages, an environment's required reviewers, and possibly
  branch protection. So the settings act divides:
  - **What GitHub allows on a private repository is set before the
    flip.** Above all, workflow runs for first-time contributors need
    approval, and the default token is read-only, so no stranger's pull
    request runs anything unapproved from the first moment.
  - **The rest is set right after the flip.** That is Pages with
    `metais.org`, private vulnerability reporting if it needs a public
    repository, the `release` environment with Ed as required reviewer
    holding `TAP_TOKEN`, and branch protection. All of it comes before
    `v1.0.0-rc.1`.

  Branch protection's required checks are switched on only once CI runs
  on pull requests. Otherwise a required check that never runs blocks
  every merge.
- **The scan the flip rests on is run at the flip**, over the whole
  history to `main`'s tip. The same goes for the sweep of the current
  tree for internal details: a host name, the tailnet URL, a note that
  describes a person's machine rather than a procedure (ADR-013 D4 item
  3). Wave 1's sweep covers the tree as it was. Since then, for example,
  `docs/dev/ux-testing.md` says "Ed's host". A finding is rotated, or
  shown false, before the flip. If one is a live secret an item cannot
  rotate, the flip waits on Ed's rotation.
- **Each defect becomes a fix item, and the next candidate is `rc.N+1`**
  through every step. A candidate already published is never re-cut
  under its own number, and its tag is never moved. Before `v1.0.0`
  exists, a prerelease updates the tap (D7 item 5).
- **`go install @v1.0.0-rc.N` is a permanent entry in the Go proxy**,
  like every pushed tag. That is the cost of rehearsing, and it is
  accepted (D7 item 8).
- **T-A3's edge.** T-A3 (`01a15ac4-4d4e`) belongs to the v1 train, and
  nothing in this train can edit it. When this wave files its
  verification, it files an act for Ed, alongside it: add the
  verification's id to T-A3's `depends_on`, keeping every existing edge.

## Scope — what belongs elsewhere

- **Wave 4:**
  - `v1.0.0` and `spec-1.0` (T-A3);
  - `/spec/1.0/` live;
  - the launch checks;
  - the plan's close and `docs/INDEX.md`;
  - the implementation-log entries for ADR-013 and its amendments;
  - PLAN-2026-10-metais-v1 §6's release row.

  This wave does not cut `v1.0.0`, and does not release the
  specification.
- **Not this train** (plan §10). A defect whose fix is out of scope, such
  as Homebrew core, Windows or signing, is recorded, not fixed.
- **This wave is the last that can change bytes.** A defect found here
  is fixed here. Wave 4 publishes what this wave proved.

## What must not break, and how it is checked

- **Nothing irreversible happens out of order.** The closeout gives each
  act's time and evidence:
  - the flip;
  - each tag;
  - the tap's formula;
  - each published prerelease.
- **The candidate's archives are the procedure's**, and their
  `SHA256SUMS` is recorded verbatim in the record of the Work Order that
  built them. `release.yml` refuses any asset that does not match.
- **The wave tier at the verification**, on a fresh clone of the merged
  tree, as always. Then the rehearsal's checks against the published
  candidate, which are this wave's real acceptance.
- **CI is green on `main`'s pull requests once it runs on them.** A red
  on a pull request is that change's to fix.

## Out-of-band acts of this wave

Each is a person's act, filed as an off-tera Work Order. This
deployment declares `design-session` (`topic`, `context`) for a decision
or a setting, and `manual-qa-pass` (`build`, `focus`) for a check done by
hand on a build. Use whichever fits.

| Act | When | Before | Notes |
|---|---|---|---|
| Rotate a live secret the pre-flip scan finds | only if found | the flip | ADR-013 D4 item 2. Never put the secret in the inputs. |
| Ed's GitHub settings before the flip: first-time contributors' workflows need approval, read-only default token, and whatever else GitHub allows on a private repository | Wave 3 | the flip | Plan §6, ADR-013 D5 item 4, D7 item 5. Divided as the ruling above says. Ask Ed to report each setting. |
| Make `kottera/metais` public | after the scan, the settings and Wave 2's verification | the candidate | Plan §6. The train's first irreversible act. Ask Ed for the time of the flip. |
| Ed's GitHub settings after the flip: Pages with `metais.org` and HTTPS; private vulnerability reporting; the `release` environment, with Ed as required reviewer and `TAP_TOKEN` (contents write on `kottera/homebrew-tap` alone); branch protection on `main`, its required checks only once CI runs on pull requests; Pages on `kottera/kottera.com`, branch or Actions | right after the flip | `v1.0.0-rc.1` | Plan §6, `docs/dev/release.md` "Going live". |
| DNS: metais.org's apex `A` and `AAAA` and `www` to Pages; kottera.com's Pages records and its organization verification `TXT` | with the settings after the flip | `v1.0.0-rc.1` | Plan §6, `docs/dev/release.md`. Recorded where Kottera records its DNS. Every existing record is left, mail included. |
| Dispatch each hosted proving run, and the rehearsal's check workflow | as the wave needs | what each run proves | The pipeline cannot dispatch (W1-OOB2). |
| Approve the `release` environment for each candidate | after its draft exists | that candidate's checks | R-PUB-11. Ask Ed to report the run. |
| Run I1 on Opus, Sonnet and Haiku and record I2, on Ed's host, against the live site | after the candidate is published and the site is live | this wave's verification | ADR-013 D9. `docs/dev/ux-testing.md` lists what the host must provide, and how the scorecard is published. Ask early whether Ed's host can provide the credential route (above). |
| Add this wave's verification to T-A3's `depends_on` | when the verification is filed | Wave 4 | Plan §6, R-PUB-12. Keep every existing edge. |

## Closeout

*Not yet written. The wave orchestrator replaces this line with the
wave's closeout.*

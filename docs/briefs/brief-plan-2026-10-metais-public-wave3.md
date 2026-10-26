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

Written by the wave orchestrator (Work Order 01a16bbd-9746-7d56-9d6e-038b47bbd265)
on 2026-10-26, against `main` at 40b8129f. Base ref: 1166031600b6. This
section says what landed, what was decided, and where every finding went.
It is not a correctness verdict:
- **The verification's findings** in section 4 are W3-V's, with its method
  named.
- **Where this section says "reports",** the claim is the child's or the
  act's.
- **Where it says "I read",** I read the merged code or GitHub's record. I
  mutated nothing.

### The outcome in one paragraph

`kottera/metais` went public at 2026-10-24T15:00:10Z. Seven release
candidates went through the procedure:
- **rc.1** was never published.
- **rc.2** failed the `go install` check.
- **rc.3, rc.4 and rc.5** passed every check.
- **rc.6** failed one check on a GitHub rate limit, which was ruled a
  defect (R-W3-4).
- **v1.0.0-rc.7** (tag at `b49b10fb`, published 2026-10-25T22:56:50Z) passed
  every check: the release act, all eight rehearsal jobs (run 40242581308),
  and I1 on Opus, Sonnet and Haiku against the live site
  (`docs/ux/scorecard-2026-10-25-install.md`, ADR-013 D9's gate holds).

The verification passed the wave tier on a fresh clone, and raised one
blocking finding: `main` has moved past rc.7 with shipped bytes. Ed ruled
that v1.0.0 ships them without a further rehearsal (section 4). So there is
no fix child and no rc.8. Ed's session is cutting v1.0.0 at `main`'s tip
(T-A3).

### 1. What landed, and whether it matches the ask

| Item | Work Order | Pull request, merge | What landed |
|---|---|---|---|
| W3-1 | 01a16bc4-49a6 | #322, dedcecd3 | gitleaks 8.30.1 and trufflehog 3.99.4 scanned every ref, 1237 commits, to `42a470ea`. They found nothing, so nothing was rotated. The report is artifact `art_is37suvp…`. The tree's "Ed's host" in `docs/dev/ux-testing.md` now describes a host by what it must provide. |
| W3-2 | 01a16bc4-dd9c | #324, 7ff24d39 | `ci.yml` runs on `pull_request` (including `edited`) and on push to `main`. A newer pull-request run cancels an older one; push runs are never cancelled. TestCIWorkflow holds the triggers, seen red against the old set. AGENTS.md states the public rule: everything committed is published, so nothing internal and no secret. CONTRIBUTING.md, `release.md` and the comments stopped describing a private repository. |
| W3-3 | 01a16bc5-1b4c | #325, 470c2597 | `rehearse.yml` and `rehearse.sh`, dispatch-only, cover the brief's whole check list on GitHub-hosted macOS and Ubuntu and on an `ubuntu:24.04` container with only `curl`. |
| W3-4 | 01a16bc5-5cd0 | #342, 550ac7d9 | ADR-009 A12, as Ed amended it (R-W3-3), is accepted. `-credential-file` reads a `claude setup-token` token once, from a pipe only. A loopback proxy forwards to `api.anthropic.com` alone. It sets `Authorization: Bearer`, merges the OAuth beta value, and sends no `x-api-key`. `credential-not-seen` stops a set. Opus sessions get `opus[1m]`, and each record carries the context window. |
| W3-6 | 01a16bc5-c857 | #327, ebe5d2ea | rc.1 was cut from `7ff24d39`, built twice byte-identically, and drafted with nothing pushed. It was never published (below). |
| W3-12 | 01a17373-b741 | #346, 390f0398 | The install check reads the not-found wordings of bash, sh, ksh, zsh and fish. The scrub reduces PATH to the runner's and the standard system directories. Sessions run under `TZ=UTC`, and `ls -l` times are scrubbed. |
| W3-13 | 01a17375-a4be | #347, b49b10fb | `install.sh` finds the latest stable release from `github.com/…/releases/latest`'s redirect. "No stable release" stays a distinct refusal. A 403 or 429 is reported as a rate limit. It is tested on both sides of v1.0.0, and TestInstallURLsConstant refuses `api.github.com`. |
| W3-14 | 01a17375-eb47 | #348, 92b9f90f | rc.7 was cut from `b49b10fb`, built twice byte-identically, and drafted with nothing pushed. |
| W3-7 | 01a16bc6-7d08 | #351, cadecbb7 | `docs/dev/rehearsal/v1.0.0-rc.1.md` to `v1.0.0-rc.7.md` each record their candidate as far as it got, with `SHA256SUMS` verbatim from the release. `docs/ux/scorecard-2026-10-25-install.md` is rc.7's I1 and I2 set, with rc.6's runs recorded as superseded. |

Each candidate's `SHA256SUMS` is verbatim in the result of the Work Order or
act that built it, and in its record:
- rc.1: W3-6;
- rc.3: W3-OOB4;
- rc.5 and rc.6: the operator's guidance to this run, from Ed's session;
- rc.7: W3-14.

I read these mechanisms in the merged tree, and found what the items
report:
- `install.sh` names `api.github.com` only in a comment, and branches on
  `403:* | 429:*` to the rate-limit refusal.
- `ci.yml`'s `on:` has `pull_request` with `edited`, `push` to `main`, and
  `workflow_dispatch`.
- `rehearse.yml` is triggered by `workflow_dispatch` alone.
- `route.go` deletes `X-Api-Key` and sets the Bearer token.
- `docs/adr/ADR-009-the-security-model.md` has A12.

Where the wave diverged from its breakdown:
- **Ed's session ran most of the release work itself.** W3-OOB4, filed as
  rc.1's release act, became rc.1 to rc.3. Ed's session then cut and
  released rc.4, rc.5 and rc.6 outside this tree. It fixed what it found in
  pull requests no Work Order of mine owns:
  - #333 (W3-9): `release.yml` would not parse;
  - #335 (W3-10): `prepare` refuses a target whose workflows are not
    `main`'s;
  - #336 (W3-11, ADR-010 A11): the module zip paths;
  - #328 (W3-8, `01a16e73-aeb4`): every action is pinned to a full SHA;
  - #334, #337, #343, #344 and #349: the site.
  
  W3-9 (`01a16f18-3349`) is in my tree only as a cancelled row.
- **rc.1 was never published.** Its release run's push-tags was refused,
  because its target's workflows predated #328 and #333 and `GITHUB_TOKEN`
  cannot push a workflow change. #335 now refuses such a target up front.
- **I1's act was filed three times before it ran:**
  - first, it needed an edge to W3-4;
  - then the candidate moved to rc.5;
  - this worker may not `recall`, so each change was a cancel and re-file
    of an act that had not started.
  
  It then ran on rc.6 (W3-OOB5), and again on rc.7 (W3-OOB7) after
  R-W3-4.
- **W3-5 was cancelled.** Ed kept kottera.com on the branch (Q2).
- **No item ran the gate before merging,** by Ed's standing instruction.
  CI's seven required checks gated every merge from W3-OOB4's step 0
  onward.

### 2. What children flagged, declined or discovered

- **`environment: release` waits for approval even on a dry run.** This
  settles Wave 2's open question (W3-OOB4). W2-OOB6 recorded it in #350.
- **GitHub refused while private:**
  - fork-PR approval;
  - private vulnerability reporting;
  - branch protection;
  - the environment's reviewer. That attempt created an unprotected
    environment, which Ed deleted.
  
  All of it was set right after the flip (W3-OOB1, W3-OOB3).
- **Tag creation cannot be limited to `release.yml`.** A repository ruleset
  refuses GitHub Actions as a bypass actor, so it stays a rule items keep
  (W3-OOB3).
- **Two squash merges carry no parseable `Signed-off-by:`.**
  - `92b9f90f` (#348) has an empty body (W3-14).
  - `b49b10fb` (#347, rc.7's own target) has a paragraph after the trailer,
    found by W3-V.
  
  `ci.yml`'s `dco` is red on `main`'s push run 40209763274. History is not
  rewritten; W3-NB1 prevents the next one.
- **Haiku recovered from the prerelease refusal.** It reran `install.sh` with
  `METAIS_VERSION`, as the refusal tells it (W3-OOB7). Until v1.0.0 exists,
  that refusal is the documented one-liner's answer, by design (ADR-013 D6
  item 7).
- **I2 never had a clean find-and-install.**
  - On rc.6, Opus found METAIS but built it from source.
  - On rc.7, Opus was invalid on a false positive (W3-NB2).
  - Sonnet and Haiku never found it on either candidate.
  
  I2 is evidence, not a gate (ADR-013 D9).

### 3. What carries to the next wave

- **v1.0.0 ships bytes no candidate rehearsed, by Ed's ruling of
  2026-10-25.** These are #272 (`56ebf332`: the `render-work` default
  lowered from 2^32 to 2^28, filter charging changed, spec chapters 07 and
  14 changed) and #349 (`84c8cdbc`, the favicon).
  - There is no rc.8, and I1 is not rerun.
  - Wave 4's launch checks are the first run of the rehearsal workflow
    against them.
  - If a launch check fails, the bytes are already published; plan for that
    in Wave 4's brief.
- **T-A3's edge is moot.** W3-OOB8 (`01a175a0-b959`) was cancelled because
  the verification it would have added to T-A3's `depends_on` had already
  completed (operator, 2026-10-26).
- **W2-OOB6 (`01a16bb8-56c8`)** is outside this tree, and I cannot read its
  outputs. Its pull request #350 states how the `release` environment holds
  a release run. The hosted proofs it carried were made by this wave's own
  runs:
  - W3-2's pull request CI, for `ci.yml`;
  - W3-OOB4's dry run and pages refusal, for `release.yml` and `pages.yml`.
- **What I would do differently:**
  - File every act's task to require its artifacts back in the train. I1's
    first set stayed on the host, which cost a re-score plan.
  - Expect the release owner to move candidates faster than a wave can.
    Name candidates by "the next one", not by number, in item text.
  - Ask for `recall` on this worker type. Without it, every change to a
    released, unstarted act was a cancel and re-file, which minted new ids
    that dependents had to follow.

### 4. What the verification found, and where each finding went

**Who verified, and how.** W3-V (01a175a0-827f, `wave-verification`), on
`main` at cadecbb7:
- **Gate:** `gate.sh wave` on a fresh clone, GREEN: 56 packages and 62 reach
  groups.
- **Mutation:** 32 rows over W3-2, W3-3, W3-4, W3-12 and W3-13.
  - 28 went red as expected.
  - 2 were false reds (compile errors), each re-run.
  - Of the greens, 2 are filed below; 3 were explained as masked and
    re-tested by another route; 1 (accepting 301, 307 and 308 redirects in
    `install.sh`) was judged low value and not filed.
- **Inspected, not mutated:** W3-1, W3-6, W3-14 and W3-7, and the off-tera
  acts W3-OOB2, W3-OOB3, W3-OOB4, W3-OOB6 and W3-OOB7.
- **Checked against GitHub:**
  - visibility;
  - the `main` ruleset and its seven checks;
  - the `release` environment;
  - Pages and HTTPS;
  - the rc.7 tag, assets and `SHA256SUMS`, which equal the record;
  - the tap's formula;
  - release run 40242455407 and rehearse run 40242581308.
- **Re-scored:** W3-OOB7's set, with `main`'s scorer. I1 passed 3 of 3,
  matching the published scorecard, with no `sk-ant-` string in the set.
- **Audit:** every merge is an ancestor of `main`. The item list was
  verified for accuracy; its completeness was trusted.

Its 14 item entries match the 14 I sent.

**Verdict: blocking, on one cross-item finding,** quoting the brief's "Wave
4 publishes what this wave proved". `main` has moved past rc.7's target with
shipped bytes (#272, #349), so v1.0.0 cut at `main`'s tip would publish what
no candidate rehearsed. **Settled by Ed's ruling, 2026-10-25,** relayed by
the operator: v1.0.0 ships #272 and #349 without a further rehearsal; do not
cut rc.8 and do not re-run I1. So no fix child was filed, and there was no
second round.

**Where each non-blocking finding went.** The wave completed without
waiting for W3-NB1 and W3-NB2, and that is the intent; neither is
re-verified.

| Finding | Source | Where it went |
|---|---|---|
| `ci.yml`'s concurrency rule is held by no test (mutation green) | W3-V on W3-2 | W3-NB1, 01a17615-6093 |
| `pages.yml`'s pinned actions run on deprecated Node 20 | W3-7's finding 16 | W3-NB1, 01a17615-6093 |
| `b49b10fb` and `92b9f90f` carry no parseable Signed-off-by | W3-V on W3-13; W3-14 | W3-NB1, 01a17615-6093, which prevents the next one. History is not rewritten. |
| `no-memory-read` invalidates a session reading a cloned repository's CLAUDE.md | W3-OOB7's finding 1 | W3-NB2, 01a17615-bf02 |
| The Thresholds line's invalid count contradicts the table | W3-OOB7's finding 5 | W3-NB2, 01a17615-bf02 |
| The scorecard header names the runner's build, not the installed candidate | W3-7's finding 13 | W3-NB2, 01a17615-bf02 |
| The route's RawPath refusal is untested (mutation green) | W3-V on W3-4 | W3-NB2, 01a17615-bf02 |
| `ux-testing.md`'s re-score example uses scenario ids the runner refuses | W3-OOB7's finding 4 | W3-NB2, 01a17615-bf02 |
| Sessions on a person's host write outside the run | W3-OOB7's finding 2 | W3-NB2, 01a17615-bf02 |
| The rc.7 record does not say `main` moved past its target | W3-V on W3-7 | janitorial issue #352 |
| `install.sh` refuses by default while only prereleases exist | W3-OOB5, W3-OOB7 | **Not filed.** ADR-013 D6 item 7 and the brief's tested refusal; it clears at v1.0.0. |
| `install.sh` accepts 301, 307 and 308 as a redirect to a tag, untested | W3-V on W3-13 | **Not filed.** W3-V judged it low value; the choice is defensive. |

**Why the findings were grouped:**
- **W3-NB1:** the workflows, their tripwire and the merge convention.
- **W3-NB2:** `internal/tools/ux` and its runbook, which one gate covers.

Each child is told its members are severable.

### 5. Out-of-band acts

| Act | Type | Work Order | Placement | Outcome |
|---|---|---|---|---|
| Settings a private repository allows; Q1 and Q2 | design-session | W3-OOB1, 01a16bc4-06a8 | waited on by the flip | Ended by Ed. Token read-only; fork PRs run nothing. Q1: no credential route, so W3-4 runs. Q2: kottera.com stays on the branch, so W3-5 is cancelled. |
| Make `kottera/metais` public | design-session | W3-OOB2, 01a16bc4-5df9 | waited on by everything after | Ended by Ed. Public at 2026-10-24T15:00:10Z, `main` at `e2742883`. The gap since W3-1's scan was rescanned clean. Fork-PR approval was set at 15:00:11Z. |
| Settings and DNS after the flip | design-session | W3-OOB3, 01a16bc4-980f | waited on by the release acts | Ended by Ed. Pages with HTTPS on metais.org (~15:37Z) and kottera.com (15:39:42Z); private vulnerability reporting; the `release` environment and `TAP_TOKEN`; ruleset `main`; DNS recorded in `kottera/infra`. Answered R-W3-2: merges use the admin bypass through a pull request. |
| rc.1's release act, which became rc.1 to rc.3 | manual-qa-pass | W3-OOB4, 01a16bc5-fcc5 | waited on | Ended by Ed's session. Required checks on. rc.1 retired. rc.2 tagged 21:22:00Z, published 21:22:01Z, tap 158ccff5. rc.3 tagged and published 22:12:38Z (tag 7cb15565 at a3ec4b9b), tap 8c4b93b0, rehearsal 40137683223 all green. |
| rc.7's release act and checks | manual-qa-pass | W3-OOB6, 01a17376-19df | waited on | Ended by Ed's session. Approved 22:56:27Z; tag v1.0.0-rc.7 at b49b10fb pushed 22:56:49Z; published 22:56:50Z; tap 070a22ad; pages deployed; rehearsal 40242581308 8 of 8. |
| I1 and I2 on rc.6 | manual-qa-pass | W3-OOB5, 01a17144-8fb4 | waited on | Ended by Ed's session; superseded. I1 scored 1 of 3; on substance 3 of 3, with Ed's PASS for Sonnet and a scorer defect behind Haiku's fail. The set stayed on the host. |
| I1 and I2 on rc.7 | manual-qa-pass | W3-OOB7, 01a17376-4dff | waited on | Ended by Ed's session. I1 3 of 3 with verdicts, Opus `opus[1m]` with a 1M window. The scrubbed set is artifact `art_yieglt55…`. |
| Add the verification to T-A3's `depends_on` | design-session | W3-OOB8, 01a175a0-b959 | filed with the verification | Cancelled as moot (operator, 2026-10-26). |
| Rotate a live secret | — | — | only if found | Not filed: W3-1 found none. |

The candidates rc.4, rc.5 and rc.6 were released in Ed's session, outside
any act of this tree. The operator relayed their runs and `SHA256SUMS` to
this run. W3-7 recorded them from GitHub.

### Rulings made during this wave

Recorded in plan §8:
- **R-W3-1:** I1 is rerun only when an install route, the site, the skill or
  the render step changes.
- **R-W3-2 (Ed):** the train's pull requests merge with
  `gh pr merge --squash --admin` once every check is green.
- **R-W3-3 (Ed):** A12 uses a subscription token, and the Opus model string
  and window are recorded.
- **R-W3-4:** fix `install.sh` in this wave and cut rc.7.

Not yet in the plan:
- **Ed, 2026-10-25:** v1.0.0 ships #272 and #349 without a further
  rehearsal.

Mine, each relaying or following one of the above:
- I answered W3-4's merge question from R-W3-2.
- I re-filed I1's act against rc.5, under R-W3-1, when #343 restyled the
  site.
- I treated rc.6's rate-limit red as Ed's finding, not mine, until R-W3-4.

I escalated twice:
- A12's approval, a credential boundary;
- whether `install.sh` changes in this wave, which shipped bytes and an I1
  rerun hung on.

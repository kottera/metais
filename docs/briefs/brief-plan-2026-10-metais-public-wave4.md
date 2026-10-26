# Brief — PLAN-2026-10-metais-public, Wave 4: launch

**Plan:** [PLAN-2026-10-metais-public](../plans/PLAN-2026-10-metais-public.md), §9
**Feature:** `metais-public`. Work Orders are titled `metais-public [W4-<item>]: <action>`;
pull requests and their squash subjects are `<type>(metais-public): [W4-<item>] <change>`
(AGENTS.md, Git and pull requests).
**Repository:** `kottera/metais`, branch `main`. Every Work Order of this wave
gets `repo: kottera/metais` and `branch: main`.
**Gated on:** Wave 3, complete. Its verdict, and Ed's ruling to ship past
rc.7, are under plan §8. Its closeout is in
[brief-plan-2026-10-metais-public-wave3.md](brief-plan-2026-10-metais-public-wave3.md).
T-A3's own dependencies are T-A3's (plan §3), and T-A3 is already running.
The gate has cleared.

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

`code-implementation-direct`'s postcondition is a merged pull request. Work
with no pull request is a person's act, filed off-tera.

## What this wave is for

`v1.0.0` published by Ed's one approval, every route checked live against
it, and the plan's own records closed. When it is done:

- **`v1.0.0` and `spec-1.0` are published.** T-A3 (`01a15ac4-4d4e`)
  belongs to the v1 train, and runs the release procedure:
  - the specification's release commit (#355, `6289d1b5`), already on
    `main`;
  - the tool tagged in its own clone, built twice, compared and drafted.
    A draft `v1.0.0` release exists, made 2026-10-26T10:50:24Z.

  Then Ed approves the `release` environment, and `release.yml`:
  - pushes `v1.0.0` and `spec-1.0`;
  - publishes the release and updates the tap;
  - deploys the site with `/spec/1.0/`.
- **The launch checks passed against `v1.0.0`.** These are Wave 3's
  checklist, run by `rehearse.yml` on GitHub-hosted runners and the empty
  container, each recorded with its evidence:
  - brew;
  - `install.sh` with and without `METAIS_VERSION`. Without it, the
    script now installs `v1.0.0`; the prerelease refusal no longer
    applies;
  - `go install metais.org/metais/cmd/metais@latest` and `@v1.0.0`;
  - the manual route;
  - `--version` reporting `1.0.0` from each route;
  - every `$id`, the `go-import` pages, the reference pages,
    `/spec/1.0/`, `/spec/` leading to it, and kottera.com;
  - `metais skill`.

  The results go in a launch record beside the rehearsal records in
  `docs/dev/rehearsal/`.
- **The plan's records are closed** (plan §9, "Close"):
  - `docs/INDEX.md`'s row for this plan says it is done;
  - every train item that carried ADR-013 or one of its amendments has
    its implementation-log entry. The amendments are ADR-010 A8, ADR-011
    A8, ADR-008 A13 and A14, ADR-009 A11 and A12, and any other the train
    allotted and used. An item whose entry is missing is named in the
    entry written for it;
  - PLAN-2026-10-metais-v1 §6's release row is marked done, with the
    release's URL.

  The plan's own status line and its train closeout are the plan
  orchestrator's, written after this wave closes. Do not write them.

## What to read first

- Plan §9, this wave; §6, the out-of-band table's Wave 4 row; and §8's
  rulings and Wave 3 verdict, especially Ed's ruling to ship past rc.7.
- `docs/dev/release.md`: the release act, and "When a step fails".
- `docs/dev/rehearsal/`: the format the launch record follows, and
  `v1.0.0-rc.7.md` for what was last proved.
- `.github/workflows/rehearse.yml` and `.github/scripts/rehearse.sh`.
- AGENTS.md's "Writing documentation" (log entries as files) and
  `docs/adr/logs/ADR-013/`, which holds entries for some Wave 1 and Wave 2
  items and none for Wave 3's.

## What the earlier waves learned that changes this one

- **`v1.0.0` carries bytes no candidate rehearsed**, by Ed's ruling:
  - #272 (`56ebf332`): the `render-work` default lowered to 2^28, filter
    charging, and chapters 07 and 14;
  - #349 (`84c8cdbc`): the favicon.

  The launch checks are their first run through every route. Read their
  results with that in mind.
- **Ed's session runs the release work itself**, faster than a wave can,
  and outside this tree. So read GitHub's state at breakdown and again
  before each act is released:
  - whether the draft still exists or has been published;
  - whether the tags exist;
  - whether the site serves `/spec/1.0/`.

  File an act only for what has not already happened, and record from
  GitHub what happened outside the tree.
- **Every act's task requires its evidence back in the train**: run ids,
  and any file as an artifact. Wave 3 lost a round trip to a set left on
  a host.
- **Merges.** `main` is held by a ruleset (R-W3-2). Every pull request,
  docs included, merges with `gh pr merge --squash --admin` once all its
  checks are green. Every commit signs off exactly as R-W1-5 says, with
  the trailer last in the message so it parses: two squash merges lost
  theirs (Wave 3 verdict). No commit carries a `Co-authored-by` line.
- **Still running beside this wave:** W3-NB1 (`01a17615-6093`, the
  workflows and the merge convention) and W3-NB2 (`01a17615-bf02`,
  `internal/tools/ux`), and janitorial issue #352. Keep their changes on
  rebase. None of them changes what is released.

## Rulings — do not reopen

- **R-PUB-1 to R-PUB-14**, every ADR-013 decision, and every ruling of
  §5, §7 and §8 stand. In particular:
  - only `release.yml` pushes a tag, after Ed's approval (R-PUB-14);
  - the published archives are T-A3's, as built (R-PUB-10);
  - no item edits T-A3, which is the v1 train's.
- **No I1 re-run and no further candidate** (Ed, 2026-10-25). I1's gate
  is rc.7's result.
- **A launch check that fails is brought to the plan orchestrator before
  anything is fixed or released.** `v1.0.0` and its tags are permanent
  once published, so any repair is a new release or a site redeploy,
  either of which is a decision. Record the failure with its evidence,
  file nothing to fix it, and ask. A check that fails on its harness
  rather than on the release (a runner's rate limit, a flake with a named
  mechanism) is still brought, with that mechanism named.
- **The release's own outcome is T-A3's record**, not this wave's: its
  `SHA256SUMS`, its tags, its run. The launch record copies them from
  GitHub and T-A3's report where it can, and says where each came from.

## Scope — what belongs elsewhere

- **The plan orchestrator's train closeout:** the plan's status, what the
  train did and did not establish, and every out-of-band act. Do not
  write it.
- **Not this train** (plan §10): Homebrew core, Windows packages, signing,
  a guide showing METAIS inside tera, kottera.com beyond its page, an MCP
  server.
- **R-W3-5 of PLAN-2026-10-metais-ux** (discovery with nothing
  installed): I2 recorded it, and nothing in this wave pursues it.

## What must not break, and how it is checked

- **Nothing published out of order.** The launch record gives each
  publication's time and evidence: `spec-1.0` and `v1.0.0`'s tags, the
  release, the tap's formula and the site's deploy.
- **The wave tier at the verification**, on a fresh clone of the merged
  tree, as always. The launch checks against the published `v1.0.0` are
  the real acceptance.
- **CI is green on every pull request**, and the merge convention above
  is followed.

## Out-of-band acts of this wave

Each is a person's act, filed off-tera: `design-session` (`topic`,
`context`) for a decision, and `manual-qa-pass` (`build`, `focus`) for a
check or a release act by hand. File only what GitHub's state shows has
not already happened.

| Act | When | Before | Notes |
|---|---|---|---|
| Approve the `release` environment for `v1.0.0`, and report the release run | after T-A3's draft exists | the launch checks | Plan §6's last row, R-PUB-11, R-PUB-12. It may already have happened in Ed's session; record it from GitHub if so. |
| Dispatch `rehearse.yml` against `v1.0.0`, and report the run | after the release is published and the site deployed | the launch record | The pipeline cannot dispatch (W1-OOB2). The run's id and every job's result come back in the report. |

The plan's out-of-band table has no row placed after the train.

## Closeout

Written by the wave orchestrator (Work Order 01a17805-1386-73eb-9119-8cbeef6023e3)
on 2026-10-26, against `main` at `3f7dfe03`. Base ref: `27ed13a119c4`.
This section says what landed, what was decided, and where every finding
went. It is not a correctness verdict:
- **The verification's findings** in section 4 are W4-V's, with its method
  named.
- **Where this section says "reports",** the claim is the child's or the
  act's.
- **Where it says "I read",** I read the merged code or GitHub's record. I
  mutated nothing.

### The outcome in one paragraph

`v1.0.0` and `spec-1.0` were published from Ed's session, outside this
tree, before the wave was derived. Release run 40312151129 was approved by
`efunneko` and ran three jobs: `release`, `pages / build` and
`pages / deploy`. The release was published at 2026-10-26T11:06:42Z, its
tag at `f7f57020`, and `spec-1.0` is at `6289d1b5`. So the wave filed no
release act, and it recorded the release from GitHub. The first launch run
(40312702820) failed three steps on its harness, not on the release. The
plan orchestrator ruled R-W4-1: fix the harness and re-run. W4-1 fixed it,
and W4-OOB1's re-run (40396621692) passed all eight jobs and every step.
The verification passed the wave tier on a fresh clone and raised one
blocking finding: the brief's `go install ...@latest` check had not been
run. W4-4 ran it, and it passed. The plan's `docs/INDEX.md` row says done.

### 1. Did each item land, and does it match the ask?

| Item | Work Order | PR / merge | Landed |
|---|---|---|---|
| W4-1 | `01a17808-5a09` | #362, `93b395d1` | Reports: `rehearse.sh released` checks `/spec/<M>.<m>/` per chapter, indexed, and that `/spec/` leads to the latest. The release tag is checked out with full history. `TestRehearseScript` covers the new cases. I read rehearse.sh: the pinned install removes the binary first. That code is #361's (`cab4d4e3`), which merged outside the tree, and W4-1 kept it. |
| W4-OOB1 | `01a17808-8634` | off-tera | Reports: run 40396621692 on `main` `93b395d1`, all eight jobs and every step green, every route `metais 1.0.0`, and `metais skill` ok after each install. The log is an artifact, SHA-256 `932262b3…`. |
| W4-2 | `01a17809-0f53` | #364, `5fd7a437` | `docs/dev/rehearsal/v1.0.0.md`, the launch record. I read its outcome and `@latest` sections. It also marked `docs/INDEX.md`'s row done (I read it). |
| W4-3 | `01a17809-6d46` | #363, `8fb682fd` | Reports: 32 implementation-log entries, 28 in ADR-013, 3 in ADR-010 and 1 in ADR-009. Also PLAN-2026-10-metais-v1 §6's release row marked done (I read the diffstat: 34 files). |
| W4-4 | `01a17a28-adb6` | #365, `3f7dfe03` | The blocking fix. It ran `go install ...@latest` by hand, with an isolated module cache, and recorded it in the launch record (I read that section: it installs `v1.0.0`, `h1:OfFHC3Cs…`, and reports `metais 1.0.0`). It added an `@latest` step to `rehearse.yml` for releases (I read the step) and fixed NB-A and NB-B. |

Divergence from the ask:
- **`@latest` was not run on a GitHub-hosted runner.** The brief asked for
  the launch checks run by `rehearse.yml` on GitHub-hosted runners. That
  holds for every check except `@latest`, which W4-4 ran by hand, and the
  record says so. W4-NB3 re-dispatches the workflow with the new step.

### 2. What children flagged, degraded or discovered

- **T-A3's report is outside this train.** Every fact about the release in
  the launch record is GitHub's, and it says so.
- **W4-1's released-specification check** was proved, before the re-run,
  against the release run's deployed pages artifact served locally. Its
  first live run was W4-OOB1's.
- **A commit that names a host.** W4-2's first push carried a commit whose
  email was git's default, derived from the host. W4-2 amended it before
  opening the pull request, so neither `main` nor #364 carries it (W4-V
  confirmed this). I checked GitHub: the superseded commit is still
  fetchable by SHA, and the repository's public activity feed lists it.
  This section names neither the SHA nor the host. The decision whether to
  ask GitHub to purge it is W4-NB2's.
- **W4-3's judgement calls,** stated in #363:
  - out-of-band acts that changed no file got no entry;
  - W2-6 failed with nothing merged, so it got none;
  - W3-4 and W3-11 count as logged by their ADR-009 and ADR-010 entries;
  - whether Ed required SHA pins in the settings is not in the record.
- **W4-4's toolchain.** It ran `@latest` with `GOTOOLCHAIN=local` on
  go1.26.2, and argues that the default would choose the same toolchain.
  The runner's step (W4-NB3) uses the runner's go.

### 3. What carries to the next wave, and what I would do differently

- **Derive every check of a brief's list against the harness that runs
  it.** `@latest` was in the brief, and `rehearse.yml` never had it. I
  derived W4-1 from the failure rather than from the list, and the
  verification caught the gap.
- **Set a git identity before a dispatched session's first commit.** The
  host-derived email reached a public push because the clone had none.
  W4-NB2 asks whether the pipeline's workspace setup should do this.
- **`/spec/draft/` after a release** is open (W4-NB1).
- **The plan's status line and the train closeout** are the plan
  orchestrator's, now that this wave has closed.

### 4. The verification, and where every finding went

W4-V (`01a17a14-4f6a`, `wave-verification`) reports:
- the wave tier, `gate.sh wave`, passed on a fresh clone of `main` at
  `5fd7a437`, with the SC7 reach groups;
- each merge commit is an ancestor of `main`, and `git log 27ed13a1..main`
  has nothing unaccounted for.

It returned all four item ids I sent. Depth per item:
- **W4-1: mutated.** It made 11 mutations of rehearse.sh and rehearse.yml.
  These were RED:
  - reverting #361's removal before the pinned install;
  - dropping the noindex test on released pages;
  - `/spec/` leading to the earliest revision;
  - a stable release with no released revision;
  - an unreleased revision treated as released;
  - dropping the `/spec/` refresh check;
  - chapters read from the working tree;
  - deleting `fetch-depth: 0`;
  - deleting `latest`'s `check_version`;
  - dropping `latest`'s exit check.

  Two were GREEN, and each is a filed finding (NB-A, NB-B).
- **W4-OOB1: inspected,** from GitHub's jobs API and the log artifact.
- **W4-2: inspected.** It spot-checked the launch record against GitHub:
  assets, tags, approval, the cancelled run, the tap commit, and the
  commits after rc.7.
- **W4-3: inspected.** The 32 entries' openings, and the plan row against
  GitHub.

Its findings:

| Finding | Kind | Where it went |
|---|---|---|
| The brief's `go install ...@latest` check was undelivered, yet INDEX says the checks passed (cross-item; it quotes the brief) | blocking | fix child W4-4, `01a17a28-adb6`, #365. It passed and is not re-verified, by the one-round bound. |
| NB-A: the case `released, its chapters read from its tag` cannot tell the tag from HEAD | non-blocking | W4-4, `01a17a28-adb6`, fixed |
| NB-B: `latest`'s stable `--version` comparison has no negative case | non-blocking | W4-4, `01a17a28-adb6`, fixed |
| `/spec/draft/` is built unconditionally, though ADR-011 A8 item 2 conditions it | non-blocking | W4-NB1, `01a17a59-a7ba`, `design-session`: a site change is a redeploy decision |
| W4-2's host-name incident: neither `main` nor #364 carries it | non-blocking | W4-NB2, `01a17a59-cbf5`, `design-session`, with my check that the superseded commit is still reachable |

NB-A, NB-B and the blocking finding were combined in one child, W4-4,
because all three repairs edit `internal/guarantees/rehearse_test.go`.
The two non-blocking members were marked severable.

The wave completed without waiting for W4-NB1, W4-NB2 or W4-NB3, and
that was the intent. Their ids are above and in section 5. No finding
went unfiled, and none went to the janitorial backlog.

### 5. Out-of-band acts

| Act | Type | Id | Placed | Outcome |
|---|---|---|---|---|
| Approve the `release` environment for `v1.0.0` | — | none filed | — | Done outside the tree, before derivation: run 40312151129, approved by `efunneko` (GitHub's approvals API). Recorded in the launch record. |
| Dispatch `rehearse.yml` against `v1.0.0` | `manual-qa-pass` | W4-OOB1, `01a17808-8634` | waited on, by W4-2 | Ended by its taker: run 40396621692, verdict pass, log artifact attached. |
| Re-dispatch `rehearse.yml` with the `@latest` step | `manual-qa-pass` | W4-NB3, `01a17a59-deff` | carried | Open |
| `/spec/draft/` after a release | `design-session` | W4-NB1, `01a17a59-a7ba` | carried | Open |
| The commit that names a host | `design-session` | W4-NB2, `01a17a59-cbf5` | carried | Open |

### Rulings made during this wave

- **R-W4-1** (plan orchestrator, 2026-10-26, recorded under plan §9 by
  #360): fix the harness and re-run, as W4-1 then W4-OOB1. The first run
  stays in the launch record, and the re-run is the launch checks' result.
  It governs W4-1 and W4-2.

### Beside this wave

W3-NB1 (#354) and W3-NB2 (#358) merged before this wave was derived, not
during it as the brief expected. Janitorial issue #352 is still open. #361
(the pinned-install fix) and #360 (R-W4-1) merged outside the tree, as did
Dependabot's #329 and #331.

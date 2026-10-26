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

*Not yet written. The wave orchestrator replaces this line with the
wave's closeout.*

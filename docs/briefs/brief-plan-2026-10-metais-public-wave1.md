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

Written by the wave orchestrator (Work Order 01a16404-76cf-7831-bcf6-278f81c14f88)
on 2026-10-23, against `main` at 424b523d. Base ref: bb173ea8. This section
says what landed and where every finding went. It is not a correctness
verdict: the verification round's findings below are W1-V's, with its
method named. Where this section says "reports", the claim is the child's.
Where it says "I read", I read the merged code. I mutated nothing.

### 1. What landed, and whether it matches the ask

| Item | Work Order | Pull request, merge | What landed |
|---|---|---|---|
| W1-1 | 01a16407-717a | #243, 045981b2 | The module is `metais.org/metais` in `go.mod`, every import, `testdata/` modules, `layers.json`, the baselines, the gate's scripts, the `-X` flag in `docs/dev/release.md`, and the rules in force of ADR-009 A2 and A8 and ADR-010 D1 and D12. Records keep the old path. `site/public/metais/` and `site/public/metais/cmd/metais/` carry `go-import` (`metais.org/metais git https://github.com/kottera/metais`) and `go-source`, held by `TestGoImportPages`. The wave tier was green before merge. |
| W1-2 | 01a16407-cf5e | #247, bcc44a3d | gitleaks 8.30.1 and trufflehog 3.99.3 scanned 1013 commits over every remote ref and found nothing, so nothing was rotated. The report is the Work Order's artifact `art_4ggwebkg2wmwb5f2lnoftlsaei7o2hsl3644rvq5mc2idtyo64ja`, not committed. Host labels in the baselines and host tables became platform descriptions. Notes about a person's machine became the procedure. AGENTS.md states its naming rules itself. |
| W1-3 | 01a16408-2a43 | #244, 33986491 | `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md` (Contributor Covenant 2.1, contact `conduct@kottera.com`), `SECURITY.md` (GitHub private vulnerability reporting, no email), three issue templates and the pull request template. |
| W1-4 | 01a16408-863e | #246, 1d2a8f3a | `.github/workflows/ci.yml`, dispatch-only, read-only token, jobs wave, macos, dco, shell and site. `.github/scripts/dco.sh`. `TestWorkflows` in `internal/guarantees`, seen red first three ways. AGENTS.md's three tiers and the sign-off rule. Its proving runs were declared a gap and owned by W1-4b. |
| W1-4b | 01a1644c-4203 | #249, 3d13240c; #252, 44ece575 | #249 cleared every shellcheck finding in the gate's scripts. #252 is R-W1-4's split: `.github/scripts/conformance.sh` runs `internal/conformance` by `-run`, each top-level test alone and `TestConformance` once per area, the areas read at run time. `gate.sh`, its tiers and every timeout are unchanged. Every job has a green dispatched run, linked on #246. |
| W1-5 | 01a16408-c151 | #245, 9075f604 | ADR-008 A14. `metais version` and `metais --version` end with "METAIS is made by Kottera."; every `--json` spelling is byte-identical. `TestVendorNeutral` holds help, topics and the skill free of Kottera and tera. |

I read the merged code for the asks that named a mechanism, and found what the items report:
- both `go-import` pages carry the exact content;
- `go.mod` begins `module metais.org/metais`;
- `makerLine` is in `internal/cli/commands.go`, and its output test pins it;
- `ci.yml`'s only trigger is `workflow_dispatch`;
- `TestWorkflows` has cases for `self-hosted`, `pull_request_target` and a missing `permissions`;
- AGENTS.md states three tiers and the sign-off;
- `CODE_OF_CONDUCT.md` names `conduct@kottera.com`;
- `SECURITY.md` names no address.

Where an item diverged from its first ask:
- **W1-4 could not prove its jobs.** The pipeline's credential cannot dispatch a workflow, and GitHub refused a dispatch from the branch before merge. The proving went to W1-4b and to Ed's dispatches (R-W1-2).
- **The wave job did not fit a 2-core hosted runner.** `internal/conformance` needs about 1,490 CPU-s under `-race`, against the 1,200 one binary gets in ten minutes. R-W1-4 split it in `ci.yml` alone rather than changing the gate.

The proving runs, each dispatched by Ed from `main`:

| Job | Run | Commit |
|---|---|---|
| dco | 39843868770 | 33986491 |
| site | 39843878792 | 33986491 |
| shell | 39885050381 | 454f0618 |
| wave | 39885052223 | 454f0618 |
| macos | 39952978385 | bd68635e |

### 2. What children flagged, declined or discovered

- **Sign-off identity.** Dispatch hosts have no `user.email`, so `git commit -s` signed off with a host-derived address. W1-5's merge 9075f604 carries one, and it also carries a `Co-Authored-By` trailer. R-W1-5 fixed the identity for new commits. Merged commits stay as they are (ADR-013 D4 items 1 and 3).
- **Commit metadata.** W1-2 reports that older commits' metadata carries host-derived author emails from before an email was configured. They are left as history. Its own first pushes had one too. It force-pushed them away before opening its PR, but GitHub may serve those unreachable commits by SHA until it collects them.
- **The DCO check is presence-only.** W1-4b reports that `dco.sh` already accepts a sign-off whose address differs from the author's, because ADR-013 D3 item 2 asks only that a `Signed-off-by:` be present. `TestDCOCheck` gained that case, and the check was not made stricter.
- **The Code of Conduct address.** W1-3 parked for the address and was resumed with `conduct@kottera.com` by a route other than this orchestrator. Ed's report in W1-OOB1 names the same address, an existing Google Workspace alias he confirmed. No test mail was sent.
- **The pipeline cannot dispatch workflows.** Ed chose in W1-OOB2 to dispatch proving runs himself, with no Actions grant to the pipeline. Every later workflow item needs an act for each proving run.
- **Shellcheck.** W1-4's sandbox could not download shellcheck, so the first shell run was red on scripts that predate CI (SC2148, SC2181). #249 fixed them.

### 3. What carries to the next wave

- **The pipeline has no Actions grant.** Every workflow Wave 2 adds (`release.yml`'s dry run, the install tests, `pages.yml`'s new event) is proved by Ed's dispatch, filed as an act. A brief should plan for that round trip rather than discover it.
- **Commits outside items reach CI too.** #254 (bd68635e, the co-author check) is a `metais-public` commit outside any item. It changed the dco job after it was proved. A change to `ci.yml` outside an item still needs its jobs proved, and W1-OOB5 carries that here.
- **Rescan before the flip.** W1-2's scan covers history to 045981b2. W1-NB1 rescans to the present. A rescan just before the flip, in Wave 3, is still the one the flip should rest on.
- **Test-economy and `ci.yml`.** When PLAN-2026-10-metais-test-economy's Wave 2 changes the conformance runner, its item keeps `ci.yml`'s split true (R-W1-4).
- **What I would do differently:**
  - Find out at breakdown whether the pipeline can dispatch a workflow. Here it cost two acts and a follow-on.
  - Hand the sign-off identity to every item from the start.

### 4. What the verification round found, and where each finding went

**Who verified, and how.** W1-V (01a168f1-1d01, `wave-verification`), on `main` at 424b523d:
- **Gate:** `gate.sh wave` on a fresh clone was green across 54 packages, including every SC7 reach group.
- **Mutation:** 22 mutations across W1-1, W1-4, W1-4b and W1-5. 21 went red on an assertion. One went green and was filed (below). One false red from a syntax-breaking edit was re-run properly and went red.
- **Runs:** the five proving runs were checked through the GitHub API.
- **Scan report:** W1-2's scan report was fetched and matched its criterion.
- **Tree:** no internal host name, tailnet URL or personal path was found in the current tree.
- **Audit:** the item list against `git log bb173ea8..main` was verified for accuracy. Completeness was trusted.

Depth per item:
- **Mutated:** W1-1, W1-4, W1-4b and W1-5.
- **Inspected only:** W1-2 and W1-3, whose guards were read, not broken; and the acts W1-OOB1 to W1-OOB4. The runs of W1-OOB2 to W1-OOB4 were confirmed by API. W1-OOB1 rests on Ed's attestation.

Verdict: pass, with no blocking finding, so there is no fix child and no second round. Where each non-blocking finding went:

| Finding | Where it went |
|---|---|
| Cross-item: #254 changed the dco job and added `coauthor.sh` after the dco and shell jobs were proved. Neither has run since. | W1-OOB5, 01a16905-12da-7bb0-b25d-e8e70e4b4823: Ed dispatches dco and shell. |
| Cross-item: CONTRIBUTING.md does not state the Co-authored-by check CI now enforces. | W1-NB1, 01a16904-f5e9-7ac5-962e-2780314e66dc, finding A |
| W1-1: `TestGoImportPages` does not pin `go-source`'s templates (`tree/main` to `tree/master` stayed green). | W1-NB1, 01a16904-f5e9-7ac5-962e-2780314e66dc, finding B |
| W1-2: commits merged after the scan were not scanned, and the verifier could not run a scanner. | W1-NB1, 01a16904-f5e9-7ac5-962e-2780314e66dc, finding C |
| W1-5: merge 9075f604 carries a `Co-Authored-By` trailer and a host-derived sign-off. | **Not filed.** The repair would rewrite history, which R-PUB-7 and R-W1-5 rule out. Rotation is not relevant, since it holds no secret. |

W1-NB1 holds three findings, grouped because each is thin, none decides another, and one dispatch serves all three. Each keeps its own heading and evidence, and they are severable.

**Wave 1 completed without waiting for W1-NB1 or W1-OOB5, and that is the intent.** They run beside Wave 2 and are not re-verified.

### 5. Out-of-band acts

| Act | Type | Work Order | Placement | Outcome |
|---|---|---|---|---|
| Create `kottera/homebrew-tap` and `kottera/kottera.com`, grant push, name the Code of Conduct address | design-session | W1-OOB1, 01a16406-f2e8 | waited on | Ended by Ed. Both repositories are public, each with one empty root commit. The pipeline's credential pushed and deleted a probe branch on each, with no grant changed. The address is `conduct@kottera.com`. |
| Grant the pipeline Actions write, or dispatch the proving runs | design-session | W1-OOB2, 01a1644c-1031 | waited on, filed for W1-4 (Step 4) | Ended by Ed, choosing to dispatch himself. dco and site were green; shell and wave were red. |
| Dispatch shell again | design-session | W1-OOB3, 01a16575-1416 | waited on, for W1-4b | Ended by Ed. Green, run 39885050381. |
| Dispatch wave, then macos | design-session | W1-OOB4, 01a165bf-e76b | waited on, for W1-4b | Ended by Ed. Wave green 39885052223, then macos green 39952978385. |
| Dispatch dco and shell after #254 | design-session | W1-OOB5, 01a16905-12da | carried | Open. |
| USPTO and EUIPO knockout search | — | 01a16409-e851 | filed by the plan orchestrator | Carried by it to Wave 3's flip (R-W1-3). |
| Confirm T-A3 was recalled, edited and released | — | 01a16409-f3eb | filed by the plan orchestrator | Complete. T-A3 carries plan §6's edits with all seven of its edges kept. Wave 3 adds its verification edge. |
| Rotate a live secret | — | none | — | Not needed: the scan found no secret. |

### Rulings made during this wave

All five are the plan orchestrator's, recorded in plan §5, "Rulings made during Wave 1" (#242, #250, #251):
- **R-W1-1:** contact addresses.
- **R-W1-2:** proving a workflow while private.
- **R-W1-3:** the acts the plan orchestrator filed itself.
- **R-W1-4:** CI's tests split by `-run` on a 2-core runner, in `ci.yml` only.
- **R-W1-5:** the sign-off identity.

I escalated once, for R-W1-4. Every other question was answered from the brief or from these rulings.

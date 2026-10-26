# ADR-013 — W3-OOB3: the settings after the flip (D1 item 4, D3 item 3, D5 item 4, D7 items 3.2 and 5)

**2026-10-24, PLAN-2026-10-metais-public W3-OOB3.** Written at the train's
close, 2026-10-26, by W4-3. W3-OOB3 (`01a16bc4-980f`) is Ed's act. Its
record is the plan orchestrator's pull request #326 (merged as
`68f106c5`), which wrote no entry, and the Wave 3 closeout.

- **D5 item 4.** `main` is held by a repository ruleset: changes only by
  pull request, one approving review, squash merges only, linear history,
  no force-push or deletion. Administrators bypass the review through a
  pull request, never by a direct push (R-W3-2, as Ed answered it).
  Required status checks were switched on in W3-OOB4, seven of them. A
  pull request from outside the organization runs no workflow until Ed
  approves it, only GitHub-owned actions are allowed, and the default
  token is read-only.
- **D7 items 3.2 and 5.** The `release` environment requires Ed's review,
  admits deployments from `main` alone, and holds `TAP_TOKEN`. Published
  releases are immutable, and a `v*` or `spec-*` tag cannot be moved or
  deleted. GitHub refuses GitHub Actions as a ruleset's bypass actor, so
  "only `release.yml` pushes a tag" is a rule items keep, not one GitHub
  enforces.
- **D3 item 3.** Private vulnerability reporting, the route `SECURITY.md`
  names, is on, with secret scanning and push protection, Dependabot
  alerts and security updates, and CodeQL for Go and Actions.
- **D1 item 4.** metais.org is served by Pages from Actions, and
  kottera.com from `main` `/docs`. Both domains are verified for the
  organization and enforce HTTPS, and their DNS is recorded where Kottera
  records its DNS.

It filed W3-8, which pins every action to a commit.

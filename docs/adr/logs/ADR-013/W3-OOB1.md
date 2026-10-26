# ADR-013 — W3-OOB1: the settings a private repository allows, before the flip (D4 item 5, D5 item 4)

**2026-10-24, PLAN-2026-10-metais-public W3-OOB1.** Written at the train's
close, 2026-10-26, by W4-3. W3-OOB1 (`01a16bc4-06a8`) is Ed's act, and its
record is the plan orchestrator's pull request #323 (merged as
`e2742883`, the commit the repository went public at), which wrote no
entry.

D5 item 4's settings, as far as a private free-plan repository allows,
were probed and set before the flip that D4 item 5 orders after them. The
default workflow token was already read-only and could not approve pull
requests, and fork pull requests ran no workflow. GitHub refused the
rest while private: approval for first-time contributors' workflows,
private vulnerability reporting, branch protection, and the `release`
environment's required reviewer. That last refusal created an
unprotected environment, which was deleted at once. Ed ruled that the
contributors' approval is set on the repository, as the first step after
the flip, and #323 adds it to `docs/dev/release.md`'s "Going live" as step
2, and says to create the `release` environment only once the repository
is public. His answers on the credential route (Q1, W3-4) and
kottera.com's Pages (Q2, no W3-5) are plan §8's.

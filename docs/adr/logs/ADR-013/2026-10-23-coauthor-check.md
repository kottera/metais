# ADR-013 — 2026-10-23-coauthor-check: D5's co-author check beside the DCO check

**2026-10-23, outside a train.** This item follows Ed's ruling at
PLAN-2026-10-metais-v1's T-A4 that CI fail a pull request carrying a
`Co-authored-by` line. CI's `dco` job runs `.github/scripts/coauthor.sh`
after the DCO check. The check covers the change's commits, which are the
DCO check's range with merge commits included. On a pull request it also
covers the title and body, which make up the squash message. A line that
opens with the key fails the check in any case and wherever it sits in
the message. `internal/guarantees`' `TestCoAuthorCheck` holds the check,
and `TestCIWorkflow` holds the step. The DCO check itself is no stricter
than before (R-W1-5).

The trailers on `main` come from GitHub's default squash body. #251's
only commit carried a `Co-Authored-By:` line, and it was authored under
the account's noreply address. Its squash merge kept that line and added
a `Co-authored-by` line for the noreply address, because the merging
account's identity is a different address. So AGENTS.md now has every
squash merge pass its body explicitly. History stays as it is (D4 item
1).

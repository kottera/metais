# ADR-013 — W1-NB1: the co-author check in the contributor guide, go-source held whole, and a rescan

**2026-10-23, PLAN-2026-10-metais-public W1-NB1.** Three non-blocking
findings of Wave 1's verification.

- **The co-author check, for a contributor (D3 item 3).** `CONTRIBUTING.md`'s
  "What CI runs" names the co-author check beside the DCO check, and its
  "Pull requests" section states the rule `.github/scripts/coauthor.sh`
  enforces: no commit of a pull request, and not its title or
  description, carries a line opening with `Co-authored-by:`, in any case.
  The pull request template's sign-off list carries it as a checkbox.
- **go-source, held whole (D1 item 2).** `site/site_test.go`'s
  `TestGoImportPages` compares each page's `go-source` content with one
  constant, the home and the directory and file templates together, as it
  already compared `go-import`'s. It was red against the verifier's
  mutation, `tree/main{/dir}` to `tree/master{/dir}` on the command's
  page, before it was green.
- **A rescan of the history (D4 item 2).** Every ref of a mirror clone,
  with `main` at `e0ddabb5`, was scanned again with gitleaks 8.30.1 and
  trufflehog 3.99.4, the current release of each. Neither found anything.
  The report is an artifact of the item's Work Order, not committed. It
  does not stand in for the rescan immediately before the flip, which is
  Wave 3's.

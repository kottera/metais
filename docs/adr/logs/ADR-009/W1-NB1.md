# ADR-009 — W1-NB1: the UX runner's and scorer's unpinned guards

**2026-10-22, PLAN-2026-10-metais-ux W1-NB1.** Wave 1's verifier found
guards of Amendment A8's runner and scorer that no test pinned, and W1-3's
scorecard one mis-score. Each now has a test seen red against the
verifier's mutation:

- `TestRunSetBuildsMetaisUnstamped` holds the runner's build of metais to
  `-trimpath` and `-buildvcs=false` (R-UX-7).
- The placeholder token is declared once, as `scrub.PlaceholderToken`, and
  the runner sets that one. `scrub` still uses no symbol of the allowance.
  `TestPlaceholderIsScrubbed` fails when the token a session is given is
  one the scrub keeps.
- inspect-first's metais-command half and its never-inspected branch,
  and C0's facts clause and its render reading, each have an edited
  fixture (`edited-c2-ls-before-inspect`, `edited-c1-never-inspects`,
  `edited-c0-stated-edit-reading-presented-real`,
  `edited-c0-stated-render-reading`).
- The scorer counts a render to standard output (`--out -`) that wrote
  there as a render, shown by the baseline's A1 opus run
  (`a1-opus-render-to-stdout`). Re-scoring the baseline set changes that
  run's `outcome` rule from fail to pass, and no run's result: it still
  fails `no-zip-extraction`.

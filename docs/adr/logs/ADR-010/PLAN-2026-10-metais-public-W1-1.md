# ADR-010 — W1-1 (PLAN-2026-10-metais-public): the module path of A8 item 1

**2026-10-22, PLAN-2026-10-metais-public W1-1.** Amendment A8 item 1 took
effect: `go.mod` begins `module metais.org/metais`, in one change that ran
alone and first in its wave.

- **What changed path.** Every import; both `go.mod` files under
  `internal/guarantees/testdata/`; `internal/guarantees/layers.json`; every
  committed benchmark baseline; the gate's scripts and the `root-cause`
  skill's example; AGENTS.md; `docs/dev/release.md`'s `-X` flag; D1's
  `go.mod` and D12 items 1 and 3 in this ADR's decision text; and ADR-009
  A2's and A8's `never-depends` rules. `regenerate.sh` found no generated
  file to rewrite.
- **What kept the old path.** The records: ADR-013, this ADR's A8 text
  (which records the change), the plans, every brief and closeout under
  `docs/briefs/`, and every implementation-log entry. The repository's own
  URLs (`https://github.com/kottera/metais`, its releases and `git clone`
  URLs) are not the module path and stay, by ADR-013 D1 item 1.
- **Proof.** `gate.sh wave`, the shared seam's tier, on the change's final
  commit.

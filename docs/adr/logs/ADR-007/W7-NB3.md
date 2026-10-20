# ADR-007 — W7-NB3: check's literal output paths pinned by vectors

**2026-10-20, PLAN-2026-10-metais-v1 W7-NB3.** W7-V found two of W7-2's
guards in D2 step 4's manifest paths correct but unpinned: a mutation of
each left `internal/author` and the vectors green. Each is now pinned by a
vector that goes red under the verifier's mutation. No behaviour changes.

- **The finding for an over-long or over-deep literal path.** A literal
  path is the package's own content, so `check`, like a render, refuses
  one past `path-length` or `path-depth` with `limit.package`, never
  `limit.exceeded.output` (MTS-AUT-080). The new vector
  `aut/check-output-path-limits` runs `check` and `render` on one package
  at lowered limits and expects the same findings of both. Passing
  `literal` false to `template.PathFinding` in `outputPaths` turns its
  check case red.
- **A when-guarded path left out of the collision check.** In
  `aut/check-output-paths`, `sometimes` had the path `notes.txt`, the same
  as `second`'s. Admitted to the tree, its case collision with `first`
  was reported at `second`'s location and folded into the finding already
  there, so the vector stayed green. Its path is now `NOTES.txt`, which
  collides with `Notes.txt` as a finding of its own, so dropping
  `d.When == ""` from the tree's condition turns the vector red.

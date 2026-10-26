# ADR-013 — W3-NB1: ci.yml's concurrency held by a test, pages.yml's actions on Node 24, and a squash body that keeps its sign-off

**2026-10-26, PLAN-2026-10-metais-public W3-NB1.** Three non-blocking
findings of Wave 3, each a change to the development tooling.

- **ci.yml's concurrency (D5).** `TestCIWorkflow` holds ci.yml's
  `concurrency`: a pull request's runs share the group of its number and
  a newer one cancels the run in progress, while a push run's and a
  dispatch's group is the run's own id and only a pull request's run
  cancels in progress. A push run on `main`, the record of what merged, is
  never cancelled. With `cancel-in-progress: true`, or with the group's
  fallback widened to `github.ref`, the test goes red.
- **pages.yml's actions (D5, D7 item 3).** `actions/upload-pages-artifact`
  is pinned at v5.0.0, whose `actions/upload-artifact` is v7.0.0, and
  `actions/deploy-pages` at v5.0.1, each by its full commit SHA with its
  release in a comment. Both run on Node 24, so the pages jobs no longer
  warn that Node 20 is deprecated. The other actions any workflow pins,
  `actions/checkout` v5.1.0 and `actions/setup-go` v6.5.0, already run on
  Node 24. pages.yml runs only for a published release, so the next
  release's pages job is what proves its deploy.
- **A squash body's sign-off (D3 item 2).** A squash merge's body ends with
  its `Signed-off-by:` line, with nothing after it, since git reads a
  trailer only from a message's last paragraph. `.github/scripts/squash-body.sh
  <file>` checks the body file before the merge: it fails one that is
  empty, that does not end with a sign-off naming a person and an
  address, from which git parses no such trailer, or that carries a
  `Co-authored-by` line. CI's DCO check reads a pull request's commits, so
  it cannot see a body typed at the merge, and both of Wave 3's unsigned
  merges were found only on `main`; one command before the merge stops
  both, which is why the check is a script rather than prose alone.
  `TestSquashBodyCheck` runs it. After the merge, `dco.sh <merge>^ <merge>`
  checks the merged commit. AGENTS.md and CONTRIBUTING.md say so.

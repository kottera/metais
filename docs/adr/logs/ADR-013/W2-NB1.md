# ADR-013 — W2-NB1: pages.yml's guard and the tap rule tested, and no tap downgrade (D6 item 1, D7 items 3.4 and 5)

**2026-10-24, PLAN-2026-10-metais-public W2-NB1.** Three non-blocking
findings of Wave 2's verification of W2-4, in `pages.yml`, `release.sh`
and their tests.

- **pages.yml's published-release guard is tested** (D7 item 3.4, ruling
  R-W2-1). `internal/guarantees`' `TestPagesGuard` reads the guard step
  out of `pages.yml` and runs it, as the workflow holds it, against a
  stand-in `gh`. It refuses a tag that is not `v<version>` whatever GitHub
  answers, a draft, and a tag with no release, and passes a published
  release or prerelease, which is how a dispatch redeploys one. The guard
  stays in the workflow, before the checkout, so the workflow's jobs are
  unchanged. Each of the verifier's mutations (the draft check made
  `if false`, the `v*` case removed, a failed `gh api` taken as
  published) fails the test.
- **Both sides of the tap rule are tested** (D7 item 5).
  `TestReleaseScript` releases a candidate after a candidate and a release
  after a release, which update the tap, and an `-rc` and an `-alpha`
  prerelease after a release, which do not. Each of the verifier's
  mutations fails it.
- **The tap never moves back to an older line** (D6 item 1). The tap
  holds the newest release that is not a prerelease, by
  `<major>.<minor>.<patch>` compared as numbers over the tags on GitHub.
  A release older than that one does not update the tap, and `publish`
  marks it `--latest=false`, so GitHub's latest release, which
  `install.sh` installs, does not move back either. A release that is not
  a prerelease is otherwise published `--latest=true`. D7 item 5 says
  nothing on an older line, and D6 item 1 requires `brew install` and
  `install.sh` to install the same bytes, so this is that rule applied.
  `docs/dev/release.md` states it.

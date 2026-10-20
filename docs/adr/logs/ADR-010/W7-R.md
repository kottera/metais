# ADR-010 — W7-R: the release builder of D12

**2026-10-20, PLAN-2026-10-metais-v1 W7-R**, carried from Wave 6's W6-13
and licensed by ADR-009 Amendment A2. `go run ./internal/tools/release
-version <semver>` carries D12 items 1, 2, 4 and 5; `docs/dev/release.md`
is its page.

- **Items 1 and 2.** `internal/tools/release/buildline` is item 1's build
  line and item 2's targets, with `CGO_ENABLED=0`, `GOAMD64=v1` and
  `GOARM64=v8.0` set on every build whatever the caller's environment
  holds, and `GOFLAGS=-mod=readonly`, so no flag of the host's joins it.
  The SC7 harness builds its release binary by the same line, which it
  states itself because the runner links nothing of the implementation;
  `TestHarnessBuildsByTheReleaseLine` holds the two together.
- **Item 4.** Every shipped target is built twice, each time in a fresh
  `git clone` of `HEAD` in a temporary directory with that clone's own
  `GOCACHE`, and the builder refuses unless the two binaries' SHA-256 are
  equal. It refuses a dirty tree (staged, unstaged or untracked), a `HEAD`
  that is not `v<semver>`, an existing `dist/`, and a binary whose
  `go version -m` lacks or differs in `-trimpath=true`, `CGO_ENABLED=0`,
  `GOOS`, `GOARCH`, `GOAMD64`/`GOARM64` or a clean `vcs` stamp of `HEAD`,
  or that shows any setting a release does not make (build tags, `-race`,
  `GOEXPERIMENT`). Under `-trimpath` the go command records no
  `-ldflags`, so the version `-X` sets is judged by running the host's
  build of the same clone, `metais version --json`. Building from clones
  rather than the work tree means nothing ignored or uncommitted can reach
  a release.
- **Item 5.** `dist/metais_<version>_<os>_<arch>.tar.gz` per shipped target
  holds `LICENSE`, `NOTICE`, `THIRD-PARTY-NOTICES.txt` and `metais`, in
  byte order of their names, with item 5's headers, under a gzip header
  with no name and a zero mtime; `SHA256SUMS` has one `sha256sum` line per
  archive.
- **Item 5 names `archive/tar`; the archives are written by
  `internal/tarw`.** `archive/tar` imports `os/user` (Wave 5), and A2 item
  2 keeps `os/user` refused in the builder. `tarw`'s bytes are exactly
  those `archive/tar` writes for the same headers, which its tests compare,
  so the archive is the one item 5 describes, and the stricter rule (plan
  §4) decides the writer. `archive/tar` is used only by the builder's test,
  to read the archives back.
- **Tests.** `internal/tools/release`'s tests build a small
  `github.com/kottera/metais` module in temporary repositories, never this
  checkout's tags: the whole run twice with equal `SHA256SUMS` and the
  archives' every header checked, and each refusal. Each refusal's test
  was red with its check removed; the existing-`dist/` refusal has a
  second guard, `os.Mkdir`, which keeps its test green alone.
- **Not done here:** the release itself (plan §6).

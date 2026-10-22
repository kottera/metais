# ADR-010 — T-A2: the hosts of the release calibration

**2026-10-21, PLAN-2026-10-metais-v1 T-A2.** This is plan §6's
calibration act before the release, run at main `c7532cb` with go1.26.9
fetched under `GOTOOLCHAIN=auto` on each target, each from a fresh clone.
ADR-009's log entry T-A2 holds the figures and the records. The hosts:

| Target | Host | Load while measured |
|---|---|---|
| darwin/arm64 | an Apple silicon Mac, Apple M2 Max (12 cores), macOS 26.5.2 (25F84) | 1-minute load 6 to 8; other sessions' kottera harness tests running |
| darwin/amd64 | the same Mac, under Rosetta 2 (`sysctl.proc_translated` 1), go1.26.9 darwin-amd64 from go.dev | 1-minute load 7 to 8; the same tests running in bursts |
| linux/arm64 | the same Mac's podman machine (applehv, Fedora CoreOS 43, kernel 6.18.10, 8 vCPU, 24 GiB), a `golang:1.26` container (`sha256:3c3e25a4da13fd0478eed2df1eb35a0e667094a7124d3993a6a1d30f71c17e79`), uid 0 | host 1-minute load 5.0 at the start, falling to 2.4; the guest's 0.05 |
| linux/amd64 | an x86-64 desktop, AMD Ryzen 5 3600 (6 cores, 12 threads), Ubuntu 26.04.1, kernel 7.0.0-38, bare metal | 1.1 at the start, 96% idle |
| darwin/arm64, corroborating | a second Apple silicon Mac, Apple M4 Max (16 cores), macOS 26.6.1 (25G76) | about 20; the dev fleet's gate runs |

The suite was not run on an otherwise idle host on either Mac: both are
runners in the dev fleet and were shared throughout. Ed ruled on 2026-10-21
that these runs stand as the calibration (ADR-009 T-A2 gives the reasons).

**A fresh host needs the tool's modules first.** The suite builds
`./cmd/metais` with `GOPROXY=off` and `-mod=readonly` against the host's
module cache (`internal/testenv`), and the outer `go test` downloads only
what `internal/conformance` imports. On a host whose cache has never held
the tool's modules, the build fails with `module lookup disabled by
GOPROXY=off`. Two of these runs met it; `go mod download` before the
command fixes it, and docs/dev/calibration.md now says so.

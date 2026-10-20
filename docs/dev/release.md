# Building a release

The release builder is `internal/tools/release` (ADR-010 D12). It builds
the shipped binaries from a tagged commit, proves each one reproducible,
and writes the archives a release publishes. It publishes nothing itself.

```bash
git checkout v<version>          # a clean checkout whose HEAD is the tag
go run ./internal/tools/release -version <version>
```

`<version>` is the tag without its leading `v`, a semantic version without
build metadata. The builder writes `dist/` at the work tree's root:

- `metais_<version>_<os>_<arch>.tar.gz` for darwin/amd64, darwin/arm64,
  linux/amd64 and linux/arm64, each holding `metais`, `LICENSE`, `NOTICE`
  and `THIRD-PARTY-NOTICES.txt`;
- `SHA256SUMS`, one `sha256sum` line per archive, which it also prints.

Two runs over the same tag give the same `SHA256SUMS`. Move `dist/` away
before running again: the builder never writes into an existing one.

## What it checks

It refuses, and writes nothing, when:

- the working tree has any change, staged, unstaged or untracked;
- `HEAD` is not the commit the tag `v<version>` names;
- the tool built from `HEAD` does not report `<version>`, or its
  `implements` names a specification revision that is not released. A
  revision `M.m` is released when the annotated tag `spec-M.m` is on
  `HEAD`'s history and `HEAD` holds `spec/conformance/released/M.m.sha256`
  (ADR-011 D5 items 2 and 6). Release the specification first;
- a binary's `go version -m` does not show exactly the release's settings:
  `-trimpath=true`, `CGO_ENABLED=0`, the target's `GOOS` and `GOARCH`,
  `GOAMD64=v1` or `GOARM64=v8.0`, and a clean `git` stamp of `HEAD`, with
  no other setting such as build tags or the race detector;
- the two builds of a target differ.

## How it builds

It clones `HEAD` twice into fresh temporary directories, and builds every
target in each clone with that clone's own `GOCACHE`, by the build line of
`internal/tools/release/buildline`:

```bash
CGO_ENABLED=0 GOAMD64=v1 GOARM64=v8.0 GOFLAGS=-mod=readonly GOOS=<os> GOARCH=<arch> \
  go build -trimpath -buildvcs=true \
  -ldflags "-s -w -X github.com/kottera/metais/internal/version.release=<version>" \
  -o <dir>/metais ./cmd/metais
```

The SC7 harness (`TestHostIsolation`) builds the binary it judges by the
same line, so the binary it clears is the one a release ships. The
archives are written by `internal/tarw`, whose bytes are those
`archive/tar` writes for the same headers, without linking `os/user`.

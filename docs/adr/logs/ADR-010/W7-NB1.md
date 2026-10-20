# ADR-010 — W7-NB1: the release builder builds with go.mod's toolchain, git-env.sh is POSIX sh, and the C toolchain test observes the build

**2026-10-20, PLAN-2026-10-metais-v1 W7-NB1.** Three non-blocking findings
of Wave 7's verification, W7-V, at `dd179da`. Each concerns what the
host's environment can change.

**Member A: the release builder (D1, D12).** The builder overrode only
buildline's settings, `GOFLAGS`, `GOOS`, `GOARCH` and `GOCACHE`. A
developer's `GOTOOLCHAIN`, `GOWORK`, `GOENV` and `GOPROXY` reached its go
commands. Under `GOTOOLCHAIN=local`, a release was built by whatever go
was on `PATH` rather than by go.mod's `toolchain`, and `checkSettings`
read only the `build` lines, so it accepted that binary.

- **The toolchain.** The builder reads go.mod's one `toolchain` line from
  the first clone of HEAD, and refuses a go.mod without exactly one that
  names a go release (`errNoToolchain`). Every go command it runs takes
  `GOTOOLCHAIN=<that toolchain>`, so the go on `PATH` switches to it, as
  D1 requires of every build.
- **The rest of the go configuration.** `GOWORK=off` and `GOENV=off`, so
  no workspace and no go env file is read. The module proxy is the
  builder's own, `https://proxy.golang.org,direct`, and the checksum
  database is `sum.golang.org`, the go command's defaults. `GOPRIVATE`,
  `GONOPROXY`, `GONOSUMDB` and `GOINSECURE` are empty, so no module is
  exempt from either. The checksum database stays on because the go
  command verifies a toolchain it switches to against it, and refuses the
  switch under `GOSUMDB=off`. The builder's tests set the proxy to `off`,
  so their builds never reach the network.
- **The refusal.** `checkSettings` reads `go version -m`'s first line, and
  refuses a binary whose toolchain differs from go.mod's, an experiment
  suffix included (`was built by go1.26.2, not go.mod's go1.26.9`).

Pinned false first, against the builder before the change:

- `TestGoEnvSetsTheReleaseSettings` reported `GOTOOLCHAIN`, `GOWORK`,
  `GOENV`, `GOPROXY`, `GOSUMDB`, `GOPRIVATE`, `GONOPROXY`, `GONOSUMDB` and
  `GOINSECURE` as the host's.
- `TestCheckSettings` accepted a binary built by `go1.26.2`, one built by
  `go1.26.9 X:jsonv2`, and a report with no toolchain line.
- `TestReleaseTakesNoHostGoConfiguration` runs the whole builder in an
  environment naming `GOTOOLCHAIN=go1.26.1`, an absent `GOWORK`, a
  `GOENV` file setting `GOEXPERIMENT=jsonv2`, and a `GOPROXY` of its own.
  The old builder failed at its first go command, trying to download
  `go1.26.1`. It now builds, and every archived binary names go.mod's
  toolchain.
- `TestReleaseRefusesAGoModWithoutAToolchain` and `TestGoModToolchain`
  hold the go.mod refusal.

The fake module's go.mod names the host's toolchain, as `go env GOVERSION`
reports it, so the builder's tests never switch toolchain. Their
`hostlessEnv` is still `testenv.GoEnv`'s. What reaches the builder from a
developer's environment is now its own setting whatever the host holds,
and `TestReleaseTakesNoHostGoConfiguration` is the test that hands it a
developer's.

**Member B: git-env.sh under a POSIX shell (the gate's scripts).** The
script listed variables with `compgen -e`, a bash builtin, under
`#!/bin/sh`. Under dash, `compgen: not found` left every `GIT_*` variable
set, `GIT_CONFIG_PARAMETERS` among them, which outranks the
`GIT_CONFIG_COUNT` override. So merge-setup.sh, run directly as a fresh
clone runs it, could install drivers in the repository `GIT_DIR` named.
The names now come from `env`, filtered by `sed`, and the script is POSIX
sh. A line inside a multi-line value can look like a variable. Unsetting
a name that is not set does nothing, so such a line costs nothing.

Pinned false first: `internal/guarantees`' `TestGitEnvScriptUnderPOSIXShells`
sources git-env.sh under dash, `bash --posix` and `sh`, wherever each
exists. The caller sets `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`,
`GIT_CONFIG_PARAMETERS`, an extra `GIT_CONFIG_KEY_3`, a multi-line value,
and others. Only git-env.sh's own variables may remain, and git must find
the repository it runs in and none of the caller's configuration. Before
the change, the dash case failed: every variable remained, git resolved
the other repository's `.git`, and `user.name` was the caller's.
`bash --posix` and `sh`, which is bash on darwin, passed, because bash
keeps `compgen` in posix mode.

**Member C: `TestGoCommandKeepsHostCToolchain` (D14 item 1).** The test's
own `go env` query, which reads the host it builds over, runs `CC`. So
`fakecc` left its marker before the cgo build ran, and the check that it
ran proved nothing about the build. The test now removes the marker after
the query, and only the cgo build can make it again.

Pinned false first, with a temporary `GoEnv` that passed the compiler's
own variables but not `CC` or the `CGO_*FLAGS`, so the build ran the
default compiler and succeeded. The corrected test failed with `the cgo
build never ran the host's CC, fakecc`, and the old test passed. A
`GoEnv` passing no C toolchain at all fails this host's build outright,
because its default `clang` is the cc toolchain's shim, which needs
`CC_TOOLCHAIN_TARGET_MACOS`.

**A correction to W7-C8's entry.** That entry says `GoEnv` keeps the
configuration that decides how C compiles on this host, "and nothing that
decides which Go code, modules or Go flags a build takes". That holds only
in exactly that sense. `CC`, `CXX` and the `CGO_*FLAGS` change the C
objects a cgo build links, the race build's on linux included, so they
can change what is built. They come from the host because a cgo build on
that host cannot run without them, not because they leave the binary
alone.

# Contributing to METAIS

Thank you for helping. METAIS is two things: the specification of the
package format, in [`spec/`](spec/README.md), and `metais`, its reference
tool in Go. The specification is normative and the tool implements it, so
most of what follows comes back to one rule: **the specification comes
first.**

Everyone who takes part follows the [Code of Conduct](CODE_OF_CONDUCT.md).
A security problem is reported privately, as [SECURITY.md](SECURITY.md)
says, never in a public issue.

## Before you start

For anything larger than a typo, open an issue first and say what you want
to change and why. A change to what a package means, or to what the tool
does, is a change to the specification, and it is cheaper to agree on the
wording before the code is written. The issue templates cover a bug, a
specification question or erratum, and a feature request.

## Building and testing

The `toolchain` line in `go.mod` names the exact Go toolchain every build
and test uses. You need any Go recent enough to read that line; under the
default `GOTOOLCHAIN=auto`, Go fetches the named toolchain by itself.

```bash
go build ./cmd/metais        # the tool
go build ./...               # everything
go vet ./...
gofmt -l .                   # prints nothing
go mod tidy -diff            # go.mod and go.sum stay tidy
go test -race -count=1 ./...
```

The race detector needs cgo, so `go test -race` needs a C compiler. The
whole suite is long, and it runs the conformance vectors as part of
`go test`. While you work, test the package you changed and the packages
that import it (`go test -race -count=1 ./internal/<pkg>/...`), and run
the whole suite before you open the pull request.

Tests isolate themselves with temporary directories and a temporary
`HOME`; none reads your own configuration, and none touches the network.

### Generated files

Some committed files are generated from their sources: the findings
registry and its code catalog in `spec/12-findings.md`, the help goldens,
some vectors' expected numbers, `THIRD-PARTY-NOTICES.txt` and others. Never
edit one by hand. Each has a drift test that fails until it is regenerated.

```bash
.claude/skills/verification-gate/merge-setup.sh   # once per clone, before your first rebase or merge
.claude/skills/verification-gate/regenerate.sh    # after a change to a source, and after every rebase or merge
```

`merge-setup.sh` installs the merge drivers `.gitattributes` names, so a
rebase never merges a generated file as text. `regenerate.sh` rewrites
every generated file from its sources; commit what it changes.

## What CI runs

CI is defined in `.github/workflows/ci.yml` and runs on GitHub-hosted
runners only. On a pull request and on a push to `main` it runs:

- the full check list on `ubuntu-latest` (linux/amd64), by
  `.claude/skills/verification-gate/gate.sh wave`: build, vet,
  `gofmt`, `go mod tidy -diff`, `go test -race -count=1 ./...`, the static
  build, the cross vet and build for darwin, linux and windows on amd64
  and arm64, the generated-data drift, and the conformance suite's
  host-isolation reach over every vector;
- build, vet, the static build and `go test -race -count=1 ./...` on
  `macos-latest` (darwin/arm64);
- the DCO check of every commit in the pull request (below);
- `shellcheck` over the repository's shell scripts;
- the site's build and its tests.

**While the repository is private, CI runs only when it is dispatched by
hand**, so no job runs on its own for a pull request yet. From the day the
repository is public, CI runs on every pull request, and a first-time
contributor's run waits for a maintainer to approve it. Either way, run the
commands above yourself before you open one.

## Signing off your commits

METAIS takes contributions under the
[Developer Certificate of Origin](https://developercertificate.org/) (DCO),
not a contributor licence agreement. By signing off a commit you certify
that you wrote it, or otherwise have the right to submit it under the
project's licence. **Every commit of a pull request carries a
`Signed-off-by:` line**, and CI checks every one; there are no exceptions.

Sign off as you commit:

```bash
git commit -s
```

which adds the line from your `user.name` and `user.email`:

```
Signed-off-by: Your Name <you@example.com>
```

To sign off commits you already made on your branch:

```bash
git commit --amend -s --no-edit          # the last commit
git rebase --signoff origin/main         # every commit since main
```

then push with `--force-with-lease`.

The project is licensed under the [Apache License 2.0](LICENSE), and a
contribution arrives under that same licence, its patent grant included.
Every Go source file begins with:

```go
// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0
```

and a test fails on one that does not. Code is never copied from another
template engine, scaffolding tool or schema validator: depend on a library
through its module, or write the code here. A new module dependency needs
a good reason, stated in the pull request.

## Changing the format

A change to what a package means, or to what the tool does with one, goes
in this order, and all of it in **one pull request**:

1. **The specification.** Change the text in `spec/`. Every normative
   sentence is a numbered `MTS-` statement in RFC 2119 language (MUST,
   SHOULD, MAY).
2. **Its conformance vectors.** Every observable statement is proved by a
   vector under `spec/conformance/vectors/`;
   [`spec/conformance/README.md`](spec/conformance/README.md) describes
   their layout. Add or change the vectors that prove the new wording.
3. **The tool**, so that it passes them.

**Where the tool and the specification disagree, the tool is wrong.** A
pull request that changes the tool's behaviour without the specification
and its vectors is sent back for them.

If you think the specification itself is wrong (two statements that
contradict each other, a rule no implementation could obey, a sentence
two implementations read two ways), open a specification question or
erratum issue. [`spec/ERRATA.md`](spec/ERRATA.md) says how an erratum to a
released revision is made.

A few rules hold for every change, and a review checks them:

- **Rendering is pure.** No hook, script or plugin from a package ever
  runs, no command touches the network, and the same package and inputs
  give byte-identical outputs on every platform.
- **A package is untrusted input.** Every path is confined, and every
  size, count and ratio is bounded, before anything is read or written.
- **Every behaviour has a test** that fails without it, and every refusal
  has a negative test.

## Pull requests

- Work on a branch, and keep a pull request to one change.
- Title the pull request, and its commits, `<type>(<area>): <change>`,
  for example `fix(render): refuse an output path that names a device`.
  The type is one of `feat` (a new capability), `fix` (a defect
  repaired, with its regression test), `refactor` (no behaviour changed),
  `perf`, `test` (tests only), `docs`, `build`, `ci`, `chore` or
  `revert`.
- Fill in the pull request template: what changed, why, how you tested
  it, and the specification and vectors it changes.
- Pass a multi-paragraph commit message by file (`git commit -s -F <file>`)
  rather than through the shell.

## Where to ask

- **A question about what the specification means:** open a
  specification question issue, naming the statement (`MTS-…`) you are
  reading.
- **A question about the tool, or an idea:** open a feature request or a
  bug issue, whichever fits; say in it that it is a question.
- **A security problem:** never an issue. See [SECURITY.md](SECURITY.md).
- **A conduct problem:** the contact in the
  [Code of Conduct](CODE_OF_CONDUCT.md).

`metais help` is the tool's own reference to every command, flag and
refusal, and `docs/` holds the requirements and decisions behind the
format ([`docs/INDEX.md`](docs/INDEX.md) lists them).

# The delegated liquid-spec run

The METAIS vectors prove every statement of chapters 7 and 8. The body of
core Liquid that chapter 7 adopts by reference is proved by running a
pinned selection of Shopify's liquid-spec against the reference engine
(ADR-011 D17). The run is made outside `go test`, because it needs Ruby,
Bundler and the network. It is made at every tool release, at every change
of the Liquid engine, and at every change to `07-template-environment.md` or
`08-filters-and-tags.md`.

- `liquid-spec.json` pins the run: the repository and its commit, the
  suites, the `strict2` error mode, the missing features, and the
  departures. A departure is a liquid-spec case the dialect fails. It names
  the statement that makes the dialect differ, and says why.
- `liquid-spec.rb` is the adapter that selects exactly that. It refuses a
  liquid-spec checkout at any other commit. It hands the departures to
  liquid-spec as known failures, so liquid-spec fails a departure that
  passes.
- The server is `internal/liquidspec`'s test binary, run with `-serve`. It
  implements liquid-spec's JSON-RPC driver protocol over the reference
  engine, in the dialect `metais render` uses.
- `TestDelegatedRun` gives the verdict on the run's result.

liquid-spec is run from its own checkout. Nothing of it is copied into this
repository.

## Prerequisites

- Ruby and Bundler, which liquid-spec's `Gemfile` needs.
- git, and the network: liquid-spec and its gems are fetched.
- Go, with `GOTOOLCHAIN=auto`, so that the server is built with the
  toolchain `go.mod` names.

The run does not depend on the host's locale: the adapter sets Ruby's
default external encoding to UTF-8 itself, so a `C` or `POSIX` locale is as
good as any other.

Record the host and each version before the run:

```sh
uname -sm
ruby --version
bundle --version
GOTOOLCHAIN=auto go version
```

## The run

Run these commands from the root of a checkout of this repository, at the
commit to be proved. `LIQUID_SPEC` may be any directory outside the
checkout.

```sh
export METAIS="$PWD" OUT="$(mktemp -d)" LIQUID_SPEC="$HOME/liquid-spec-84bf25e0"
git clone https://github.com/Shopify/liquid-spec "$LIQUID_SPEC"
git -C "$LIQUID_SPEC" checkout --detach 84bf25e0edbca5f7e530574788b0f875ab331f50
(cd "$LIQUID_SPEC" && bundle install)
GOTOOLCHAIN=auto go test -c -o "$OUT/liquidspec.test" ./internal/liquidspec
(cd "$LIQUID_SPEC" && bundle exec bin/liquid-spec run "$METAIS/spec/conformance/delegated/liquid-spec.rb" \
  --command="$OUT/liquidspec.test -serve" --json --no-max-failures > "$OUT/result.json"); echo "liquid-spec exit status: $?"
GOTOOLCHAIN=auto go test -count=1 ./internal/liquidspec -run '^TestDelegatedRun$' -v -args -results="$OUT/result.json"
```

## The result

`TestDelegatedRun` gives the verdict, and logs the run's counts in this
form:

```
<n> pass, <d> departures fail as listed, <h> held out for applying date, <f> fail otherwise, <p> departures pass
```

The run passes when `TestDelegatedRun` passes. That holds exactly when all
of these hold:

- every departure fails, as listed;
- no departure passes. A departure that passes fails the run, because the
  dialect and the pin have stopped agreeing. liquid-spec fails it too, as a
  stale known failure;
- every other case passes, except those the server holds out: a case whose
  template applies `date`, which the dialect registers in Wave 5. Each one
  fails with the server's `held out of the run` parse error. liquid-spec
  counts these among its failures, so it exits 1 until `date` is
  registered. Then the hold-out goes, and so does the exception: from then
  on, liquid-spec's own exit status of 0 is the pass;
- every departure is reached by the run.

When `liquid-spec.json` was written, an emulation of liquid-spec's
selection and comparison predicted this result: 2,394 cases run, 1,801
passing, 490 departures failing as listed, and 103 held out for applying
`date`. Only the run itself proves the result.

Report back:

- the host, and the versions recorded above;
- the commit of this repository the run proved;
- liquid-spec's exit status;
- `TestDelegatedRun`'s counts line, and every line it failed on;
- `$OUT/result.json`.

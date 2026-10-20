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

## Publishing metais.org

`metais.org` is the site every package's `README.md` points to for
installing the tool and getting started, and the host of the
specification's JSON Schemas at their `$id`s. Its source is `site/`:

- `site/public/` holds the pages (home, `install/` and
  `getting-started/`), their stylesheet and `CNAME`, which holds
  `metais.org`;
- `site/build.sh <out>` writes the published tree into a new directory:
  `site/public/` and every file of `spec/schemas/` under `schemas/`, so
  `https://metais.org/schemas/<path>` serves `spec/schemas/<path>`, the
  file whose `$id` it is. The schemas are copied from the commit being
  built, never committed twice. `site/site_test.go` holds the
  `$id`-to-path mapping, runs the build and requires each schema
  byte-identical at its `$id`'s path.

To see the site locally:

```bash
site/build.sh _site
python3 -m http.server --directory _site 8000   # or any static file server
```

`.github/workflows/pages.yml` builds the site with `site/build.sh` from
a pushed `v*` tag and deploys it with GitHub Pages. GitHub's free plan
publishes Pages only from a public repository, so every job of the
workflow is gated on `!github.event.repository.private`: while the
repository is private, a tag push skips both jobs, and nothing runs or
fails. The gate also requires the ref to be a `v*` tag, so a manual run
from a branch deploys nothing.

The organization's domain verification for `metais.org` (the
`_github-pages-challenge-kottera.metais.org` TXT record) was done by Ed
on 2026-10-20, before release.

### Going live

At the release, in this order:

1. **Make the repository public** (Settings → General → Danger Zone).
2. **Turn on Pages:** in Settings → Pages, set the source to **GitHub
   Actions**, and the custom domain to `metais.org`. Once GitHub has
   issued the certificate, tick **Enforce HTTPS**.
3. **Point the domain at Pages** (Ed's step, in Route 53), in the
   `metais.org` hosted zone:
   - apex `A` records to `185.199.108.153`, `185.199.109.153`,
     `185.199.110.153` and `185.199.111.153`;
   - apex `AAAA` records to `2606:50c0:8000::153`, `2606:50c0:8001::153`,
     `2606:50c0:8002::153` and `2606:50c0:8003::153`;
   - `www` as a `CNAME` to `kottera.github.io`.
4. **Publish the release and deploy the site:** push the release tag
   `v<version>`, which runs the workflow, and attach the builder's
   `dist/` archives and `SHA256SUMS` to the tag's GitHub release, which
   the install page sends readers to as the latest release. Then check
   that the home page and one schema `$id` resolve, the schema as JSON:

   ```bash
   curl -sSI https://metais.org/ | head -n 1
   curl -sSI https://metais.org/schemas/1.0/manifest.schema.json | grep -i '^content-type: application/json'
   ```

   A tag pushed before the repository was public deployed nothing; run
   the workflow on the tag from the Actions tab (Run workflow, with the
   tag selected) once steps 1 and 2 are done.

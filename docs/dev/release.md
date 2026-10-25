# Releasing metais

A release of `metais` is published in two halves (ADR-013 D7 item 3):

- **The release procedure** builds it and pushes nothing. It tags the
  release in its own clone, builds it twice and compares, and creates a
  **draft** GitHub release with the archives and `SHA256SUMS`. T-A3
  follows it for `v1.0.0`, and Wave 3's item for `v1.0.0-rc.1`.
- **The release act** is Ed's. He dispatches `release.yml`, naming the
  draft, and approves its `release` environment. Only then is anything
  pushed: the tags, the published release, the tap's formula and the
  site.

Nothing is published before Ed approves (R-PUB-14). The repository is
public, so a pushed `v*` tag is installable through the Go proxy at once,
and the proxy keeps it for good. So the procedure never pushes a tag, and a
step that would is a step to stop at.

## The release procedure

Work in a fresh clone of `kottera/metais`, at the commit to release. That
commit is on `main` and pushed: the draft names it, and `release.yml`
checks it out from GitHub. `<version>` is the release's semantic version
without its `v`, such as `1.0.0-rc.1` or `1.0.0`.

**Its `.github/workflows` must be `main`'s when Ed releases it.**
`release.yml` pushes the tag with the workflow's own token, and GitHub
refuses such a token a tag at a commit whose workflows differ from the
default branch's: it counts that as a change to the workflows, which no
workflow's token may make. So cut the release at `main`'s tip, and land
no change to `.github/workflows` between the draft and the release act.
That includes a Dependabot update of an action's pin. `prepare` refuses
a draft whose target's workflows are not `main`'s, before anything is
pushed. The fix is a new candidate, cut at `main`'s tip.

1. **Tag in the clone only.** If this release also releases the
   specification, its release commit (ADR-011 D5 item 2) is already on
   `main`. Tag it first, because the builder refuses a release whose tool
   implements a revision with no `spec-<M>.<m>` tag on its history. A
   prerelease never releases the specification, and needs no such tag
   (ADR-011 A8 item 5).

   ```bash
   git tag -a -m "METAIS specification <M>.<m>" spec-<M>.<m> <the specification's release commit>   # a release that releases it only
   git tag -a -m "metais <version>" v<version>
   git checkout v<version>
   ```

   Push neither tag. `release.yml` makes its own annotated tags at the
   same commits after Ed's approval, so these exist for the builder alone.
2. **Build twice and compare.** Each run builds every target twice and
   refuses unless the two builds match (below). The second run proves the
   whole of `dist/` reproducible from the tag:

   ```bash
   go run ./internal/tools/release -version <version>
   mv dist "$TMPDIR/dist.1"
   go run ./internal/tools/release -version <version>
   diff -r "$TMPDIR/dist.1" dist && echo "the two runs are byte-identical"
   ```

   Stop on any difference. `dist/` now holds the four archives and
   `SHA256SUMS`.
3. **Create the draft release** with the archives and `SHA256SUMS`,
   targeting the tagged commit by its full SHA:

   ```bash
   gh release create v<version> --draft \
     --target "$(git rev-parse 'v<version>^{commit}')" \
     --title "metais <version>" --notes-file "$TMPDIR/notes.md" \
     dist/*
   ```

   A draft stores its tag's name without creating the tag. The notes are
   the release's text. When the release also releases the specification,
   they end with one line that names its release commit, which is how
   `release.yml` knows to push `spec-<M>.<m>`:

   ```text
   Specification release: spec-<M>.<m> <the commit's full SHA>
   ```

4. **Check that nothing was pushed.** Neither tag is on GitHub:

   ```bash
   git ls-remote --tags origin 'v<version>' 'spec-*'   # lists no v<version>, and no new spec tag
   ```

5. **Record `SHA256SUMS`** in the Work Order's result, verbatim, with the
   draft's URL. That copy is half of what the archives' authenticity rests
   on (ADR-013 D7 item 4).
6. **Hand over.** The procedure ends here, with nothing pushed. It asks Ed
   to dispatch `release.yml` for the draft, naming its tag.

## The release act

Ed dispatches `release.yml` from `main`, with the draft's tag and
`dry_run` off:

```bash
gh workflow run release.yml --ref main -f tag=v<version> -f dry_run=false
```

Then, on the run's page, he reviews the deployment to `release` and
approves it. Until he does, the job waits and nothing runs. After approval
the `release` job runs `.github/scripts/release.sh`, one step at a time:

1. **prepare** checks everything and pushes nothing:
   - the release is a draft, and its target is a commit's full SHA that
     the repository holds, whose `.github/workflows` is `main`'s (see
     the procedure above);
   - the draft's assets are exactly `SHA256SUMS` and one archive per
     shipped target (`buildline.Shipped`). `SHA256SUMS` is exactly what
     the builder writes, and each archive's SHA-256 is its line's. The
     `metais` of the linux/amd64 archive reports `<version>`. This is
     `go run ./internal/tools/release verify`;
   - a specification release the draft names is a commit on the target's
     history that holds `spec/conformance/released/<M>.<m>.sha256`, and
     the tool implements it. A prerelease names none;
   - a release that is not a prerelease implements only released
     revisions: each has a `spec-<M>.<m>` tag on GitHub or is the one the
     draft names;
   - no tag it would push is on GitHub at another commit. A tag already
     there at the right commit is not pushed again, so a run that stopped
     part way can be dispatched again;
   - the repository is public, the `release` environment requires a
     reviewer, and it holds `TAP_TOKEN` when the tap is to be updated.

   It then makes the annotated tags in the job's clone, tagged by the
   dispatching account, generates the formula from `SHA256SUMS` (`go run
   ./internal/tools/release formula`), and commits it to a clone of the
   tap.
2. **push-tags** pushes `v<version>` at the draft's target, and
   `spec-<M>.<m>` at the commit the draft names, in one atomic push.
3. **publish** publishes the draft, marked a prerelease when `<version>`
   has a prerelease suffix (any suffix: `-rc.1`, `-alpha.1`). A release
   that is not a prerelease is marked GitHub's latest release, unless it
   is older than the newest one, as below.
4. **push-tap** pushes `Formula/metais.rb` to `kottera/homebrew-tap`'s
   `main` with `TAP_TOKEN` (ADR-013 D7 item 5). The commit follows the
   tap's README: it is signed off by the dispatching account and carries
   no `Co-authored-by` line. The push starts the tap's CI, which runs
   `brew audit` and `brew test`.

**Which release the tap holds.** The tap holds the newest release that
is not a prerelease, the same one GitHub marks latest and `install.sh`
installs, so `brew install` installs the same bytes as the other golden
routes (ADR-013 D6 item 1). "Newest" compares `<major>.<minor>.<patch>`
as numbers, over the `v<version>` tags on GitHub without a prerelease
suffix. So:

- a prerelease updates the tap only while GitHub has no such tag, that
  is, before `v1.0.0` (D7 item 5). A candidate after an earlier candidate
  updates it;
- a release updates the tap when it is the newest, after any earlier
  release;
- a release older than the newest, such as `v1.0.2` after `v1.1.0`,
  updates neither the tap nor GitHub's latest release, so neither route
  moves back to an older line. Its archives are still published on its
  release page.
5. **pages**, the workflow's last job, calls `pages.yml` with the released
   tag, once the release is published. It builds the site from that tag
   with `site/build.sh --release <version> <SHA256SUMS>`, with the
   `SHA256SUMS` the published release serves, and deploys it with GitHub
   Pages.

`release.yml` runs no second build and makes no attestation: the
archives are the procedure's, as built (ADR-013 D7 items 2 and 4).

**Why `release.yml` calls `pages.yml`.** A release published with a
workflow's own `GITHUB_TOKEN` starts no new workflow run, so a
`release: published` trigger would never fire. A token that does start
workflows would be a second credential, wider than this needs. So the
site is deployed by `release.yml`'s own last job, which runs only after
the publish step succeeded (ruling R-W2-1). `pages.yml` has no push or
release trigger. Its build refuses a tag whose release is not published,
and checks the tag out, never a branch.

### When a step fails

Read the failed step's log. Nothing after it ran.

- **prepare refused.** Nothing was pushed. Fix what it names, in the draft
  or on `main`, and dispatch again.
- **push-tags or publish failed.** Dispatch again. prepare accepts a tag
  already pushed at the right commit, and pushes only what is missing.
- **push-tap failed.** The release is published, so a new dispatch
  refuses it. Push the formula by hand, following the tap's README:
  generate it with `go run ./internal/tools/release formula -version
  <version> -sums SHA256SUMS`, commit it as `Formula/metais.rb` with
  `git commit -s`, and push it to the tap's `main`. Then dispatch
  `pages.yml` as below.
- **pages failed.** Dispatch `pages.yml` as below.

### Redeploying the site

To rebuild and redeploy the site of a published release, as an erratum
does (ADR-011 A8 item 1), dispatch `pages.yml` from `main` with the
release's tag:

```bash
gh workflow run pages.yml --ref main -f tag=v<version>
```

It refuses a tag whose release is not published.

### The dry run

`release.yml`'s `dry_run` input exists to prove the workflow alone. Under
it, prepare runs whole, against a real draft. The push steps, the publish
and the pages job are skipped, and `release.sh` refuses each push step on
its own as well. So a dry run pushes no tag, publishes nothing and leaves
the tap alone. It still checks the draft's assets, makes the tags in the
job's clone, and commits the formula to its clone of the tap, and its log
shows that commit. It also reports, without refusing, what a real run
would refuse: a private repository, a `release` environment without a
reviewer, and a missing `TAP_TOKEN`.

To run one, create a draft by the procedure above with a prerelease
version that no release will use, such as `0.0.0-alpha.1`, and dispatch:

```bash
gh workflow run release.yml --ref main -f tag=v0.0.0-alpha.1 -f dry_run=true
```

Delete the draft afterwards (`gh release delete v0.0.0-alpha.1`), which
deletes no tag, because none exists.

### The release environment

`release.yml`'s `release` job names `environment: release`, so every
dispatch, a dry run included, waits on the run's page until Ed approves
the deployment to `release`. Nothing in the job runs before then.

On GitHub's free plan, an environment's protection rules and secrets
exist only on a public repository. So prepare refuses any run but a dry
run on a private repository, or when the `release` environment requires
no reviewer, because nothing would then have required Ed's approval.

Approving a dry run commits to nothing: no tag reaches the repository
and the release stays a draft. So a `pages.yml` dispatch for that tag is
refused until a real run publishes it.

## The rehearsal's checks

Before `v1.0.0`, each release candidate is installed by every route on
clean machines, and the site and kottera.com are checked live (ADR-013 D7
item 8). `.github/workflows/rehearse.yml` runs those checks. It runs only
when dispatched, so dispatch it once the candidate's `release.yml` run is
green, which is after its prerelease is published, the tap updated and the
site deployed. Name the version without its `v`:

```bash
gh workflow run rehearse.yml --ref main -f version=<version>   # such as 1.0.0-rc.1
```

Dispatch it again for every later candidate. Its run is the evidence the
rehearsal's record cites, so record the run's URL there. The candidate
passes when every job is green. Each job runs whether or not another
fails, and so does each check of the site job, so a red run lists every
check that refused. Each check is one step running
`.github/scripts/rehearse.sh` or the route's own command, and its log
shows:

- the command;
- what it printed;
- the version `metais --version` reported, or the URL and the HTTP answer;
- a last line beginning `ok`, or a line beginning `FAIL` saying what it
  found.

| Job | Runner | What it checks |
|---|---|---|
| `brew-macos` | `macos-latest` | `brew install kottera/tap/metais`, then `brew info`, then `metais --version` reports `<version>`, and `metais skill` writes the skill of `<version>` into a new project |
| `install-ubuntu`, `install-macos` | `ubuntu-latest`, `macos-latest` | `curl -fsSL https://metais.org/install.sh \| sh` with no `METAIS_VERSION`: for a prerelease, the refusal that names `METAIS_VERSION`, since no stable release exists, with nothing installed. Then the same with `METAIS_VERSION=<version>`: an install into `~/.local/bin` that reports `<version>`. Then `metais skill` |
| `install-container` | an `ubuntu:24.04` container | the same, in an image given only `curl` and its CA certificates, which the one-line command needs |
| `go-install` | `ubuntu-latest` | `go install metais.org/metais/cmd/metais@v<version>` through the Go proxy, then `metais --version` and `metais skill`. Then the same with `GOPROXY=direct` and a module cache of its own, so the go command reads metais.org's go-import page and clones the repository it names |
| `manual-ubuntu`, `manual-macos` | `ubuntu-latest`, `macos-latest` | the manual route: this machine's archive and `SHA256SUMS` downloaded from the release, checked with the install page's command (`sha256sum --check --ignore-missing`, or `shasum -a 256` on macOS), `metais` unpacked and run, reporting `<version>`. Then `metais skill` |
| `site` | `ubuntu-latest` | against a checkout of the tag `v<version>`: every schema's `$id` under `spec/schemas/` and `docs/cli/schemas/` answers `200 application/json` with JSON holding that `$id`; both go-import pages; a reference page for every command and topic that the release's `metais help --json` lists; a `/spec/draft/` page per chapter of `spec/`, each unindexed; the landing page and the install page; `/install.sh` byte for byte the tag's `site/public/install.sh`; and kottera.com |

The site's checks read the schemas, chapters and `install.sh` from the
candidate's tag, so a schema added since is never skipped. The scripts are
the dispatched ref's, `main`'s.

A red check is first read for its mechanism. One the candidate caused
becomes a fix item and a new candidate (`rc.N+1`), never a re-cut of this
one. One that is the workflow's own becomes a fix item to the workflow.
`internal/guarantees`' `TestRehearseWorkflow` holds the workflow to these
checks, and `TestRehearseScript` runs each check of `rehearse.sh` against
a release and site served locally, as published and with each defect it
is there to find.

## The release builder

The release builder is `internal/tools/release` (ADR-010 D12). It builds
the shipped binaries from a tagged commit, proves each one reproducible,
and writes the archives a release publishes. It publishes nothing itself.

```bash
go run ./internal/tools/release -version <version>   # in a clean checkout whose HEAD is the tag v<version>
```

It writes `dist/` at the work tree's root:

- `metais_<version>_<os>_<arch>.tar.gz` for darwin/amd64, darwin/arm64,
  linux/amd64 and linux/arm64, each holding `metais`, `LICENSE`, `NOTICE`
  and `THIRD-PARTY-NOTICES.txt`;
- `SHA256SUMS`, one `sha256sum` line per archive, which it also prints.

Two runs over the same tag give the same `SHA256SUMS`. Move `dist/` away
before running again: the builder never writes into an existing one.

### What the builder checks

It refuses, and writes nothing, when:

- the working tree has any change, staged, unstaged or untracked;
- `HEAD` is not the commit the tag `v<version>` names;
- `go.mod` has no single `toolchain` line naming a go release;
- the tool built from `HEAD` does not report `<version>`, or its
  `implements` is empty or names something that is not a revision `M.m`;
- `<version>` has no prerelease suffix and the tool's `implements` names
  a specification revision that is not released. A revision `M.m` is
  released when the annotated tag `spec-M.m` is on `HEAD`'s history and
  `HEAD` holds `spec/conformance/released/M.m.sha256` (ADR-011 D5 items 2
  and 6). Release the specification first. A prerelease such as
  `1.0.0-rc.1` may implement a revision that is not yet released, so a
  release candidate can be rehearsed before the specification's release
  commit (ADR-011 A8 item 5); the builder names each such revision in its
  report;
- a binary's `go version -m` does not show exactly go.mod's toolchain and
  the release's settings: `-trimpath=true`, `CGO_ENABLED=0`, the target's `GOOS` and `GOARCH`,
  `GOAMD64=v1` or `GOARM64=v8.0`, and a clean `git` stamp of `HEAD`, with
  no other setting such as build tags or the race detector;
- the two builds of a target differ.

### How the builder builds

It clones `HEAD` twice into fresh temporary directories, and builds every
target in each clone with that clone's own `GOCACHE`, by the build line of
`internal/tools/release/buildline`:

```bash
CGO_ENABLED=0 GOAMD64=v1 GOARM64=v8.0 GOFLAGS=-mod=readonly GOOS=<os> GOARCH=<arch> \
  GOTOOLCHAIN=<go.mod's toolchain> GOWORK=off GOENV=off \
  GOPROXY=https://proxy.golang.org,direct GOSUMDB=sum.golang.org \
  GOPRIVATE= GONOPROXY= GONOSUMDB= GOINSECURE= \
  go build -trimpath -buildvcs=true \
  -ldflags "-s -w -X metais.org/metais/internal/version.release=<version>" \
  -o <dir>/metais ./cmd/metais
```

Every go command it runs takes these go settings, whatever the
developer's environment holds. So a release is built by go.mod's
`toolchain` under any `GOTOOLCHAIN`, `local` included, which the go on
`PATH` fetches and switches to as ADR-010 D1 requires. No workspace, go
env file, proxy or module exemption of the developer's reaches it.

The SC7 harness (`TestHostIsolation`) builds the binary it judges by the
same line, so the binary it clears is the one a release ships. The
archives are written by `internal/tarw`, whose bytes are those
`archive/tar` writes for the same headers, without linking `os/user`.

## metais.org

`metais.org` is the site every package's `README.md` points to for
installing the tool and getting started, and the host of the
specification's JSON Schemas and the command line's output schemas at
their `$id`s. Its source is `site/`:

- `site/package/` is the site as a METAIS package: its pages' templates,
  around one layout whose footer names METAIS's maker, and the schemas
  of the inputs it renders from (ADR-013 D8);
- `site/public/` holds the static files, the stylesheet and `CNAME`,
  which holds `metais.org`, each served at its own path byte for byte;
- `site/build.sh <out>` writes the published tree into a new directory.
  It runs `internal/tools/site`, which builds `metais` from the commit,
  exports the package's inputs from it (its help, its skill, the
  specification's text and the release), renders the package with it,
  and adds `site/public/`, every file of `spec/schemas/` under
  `schemas/`, and every file of `docs/cli/schemas/` under `cli/schemas/`.
  So `https://metais.org/schemas/<path>` serves `spec/schemas/<path>`,
  and `https://metais.org/cli/schemas/<path>` serves
  `docs/cli/schemas/<path>`, each the file whose `$id` it is. The schemas
  are copied from the commit being built, never committed twice. The
  script's header says how a release's version and `SHA256SUMS` reach
  it. `site/site_test.go` holds the `$id`-to-path mapping, runs the build
  twice to hold it byte-identical, and requires each schema
  byte-identical at its `$id`'s path.

To see the site locally:

```bash
site/build.sh _site
python3 -m http.server --directory _site 8000   # or any static file server
```

`release.yml`'s last job calls `.github/workflows/pages.yml` with the
released tag, and it can be dispatched with a published release's tag
(above). It checks that the tag's release is published, checks the tag
out with its whole history, downloads the release's `SHA256SUMS`, builds
the site with `site/build.sh --release`, and deploys it with GitHub Pages.
GitHub's free plan publishes Pages only from a public repository, so every
job of the workflow is gated on `!github.event.repository.private`: on a
private repository, both jobs are skipped, and nothing runs or fails.

The organization's domain verification for `metais.org` (the
`_github-pages-challenge-kottera.metais.org` TXT record) was done by Ed
on 2026-10-20, before release.

## Going live

These are Wave 3's, before `v1.0.0-rc.1`, in this order:

1. **Make the repository public** (Settings → General → Danger Zone).
2. **Require approval for outside contributors' workflows**, at once,
   before anything else on this list: in Settings → Actions → General,
   under approval for fork pull request workflows, choose
   **Require approval for first-time contributors**, or stricter. GitHub
   refuses this setting while the repository is private, and until it is
   set a public repository runs on GitHub's default. While private, fork
   pull requests run no workflow at all. The default token is already
   read-only and cannot create or approve pull requests: check that it
   still is, and that it has stayed that way through the flip.

   ```bash
   gh api -X PUT repos/kottera/metais/actions/permissions/fork-pr-contributor-approval \
     -f approval_policy=first_time_contributors
   gh api repos/kottera/metais/actions/permissions/workflow
   ```
3. **Turn on Pages:** in Settings → Pages, set the source to **GitHub
   Actions**, and the custom domain to `metais.org`. Once GitHub has
   issued the certificate, tick **Enforce HTTPS**. Pages deploys from
   runs on `main`, which is where `release.yml` and a dispatched
   `pages.yml` run, so the `github-pages` environment's default rule,
   which allows `main`, stays as it is.
4. **Point the domain at Pages** (Ed's step, in Route 53), in the
   `metais.org` hosted zone:
   - apex `A` records to `185.199.108.153`, `185.199.109.153`,
     `185.199.110.153` and `185.199.111.153`;
   - apex `AAAA` records to `2606:50c0:8000::153`, `2606:50c0:8001::153`,
     `2606:50c0:8002::153` and `2606:50c0:8003::153`;
   - `www` as a `CNAME` to `kottera.github.io`.
5. **Create the `release` environment** (Settings → Environments → New
   environment, named `release`), only once the repository is public. On
   a private free-plan repository GitHub refuses the required reviewer
   and creates the environment anyway, with no protection:
   - **Required reviewers:** Ed. Leave **Prevent self-review** off, since
     Ed both dispatches the release and approves it.
   - **Deployment branches and tags:** selected branches, `main` alone,
     since `release.yml` runs from `main`.
   - **Environment secret `TAP_TOKEN`:** a fine-grained personal access
     token with **Contents: Read and write** on `kottera/homebrew-tap`
     alone, and no other repository or permission.
6. **Cut the release** by the procedure and the release act above. Once
   the `pages` job is green, check that the home page and one schema `$id`
   resolve, the schema as JSON:

   ```bash
   curl -sSI https://metais.org/ | head -n 1
   curl -sSI https://metais.org/schemas/1.0/manifest.schema.json | grep -i '^content-type: application/json'
   ```

# ADR-013: The public project and its distribution — `kottera/metais` under the module path `metais.org/metais`, one approval that publishes every route, four install routes to the same reproducible archives, and a site the binary writes

**Status:** Proposed 2026-10-22 by the `metais-public` design session with
Ed Funnekotter. Every decision states a ruling he made in that session
(R-PUB-1 to R-PUB-14, in
[PLAN-2026-10-metais-public](../plans/PLAN-2026-10-metais-public.md) §4).
Each decision takes effect in the tree as the plan item that carries it
lands.
**Date:** 2026-10-22
**Deciders:** Ed Funnekotter
**Designs:** PRD-001 R17 (the golden paths) for installing the tool, and
the publication SC1 presumes: an AI harness can get the binary at all.
**Amends:** ADR-010 D1, D12 items 1, 3 and 5, and D15 (ADR-010 A8);
ADR-011 D5 item 6, D7 item 7 and "What this does not decide" (ADR-011 A8); ADR-008
D15 and D16 item 1 (ADR-008 A13). Each amendment states the new rule in its ADR
and points here for the reasons.
**Depends on:** ADR-010 D12 (the release builder) and ADR-009 A2 (its host
access), which this ADR leaves as they are.

## Context

### Where the project stands

METAIS v1 is built. PLAN-2026-10-metais-v1's seven waves are complete,
and the v1.0 release (T-A3, Work Order `01a15ac4-4d4e`) waits only on two
of Ed's rounds. PLAN-2026-10-metais-ux made the golden paths findable from
the binary: its after scorecard passes 24 of 24 gated runs.

Nothing has been published. The repository is private. ADR-010 D12 item 5
says "Publishing … is outside this ADR; the repository is private."
ADR-011 D7 item 7 says nothing is hosted under `https://metais.org/`.
`site/` and `.github/workflows/pages.yml` exist, gated off while the
repository is private.

### What going public has to settle

- **Identity that cannot change later.** Once `v1.0.0` is on the public Go
  proxy and in people's taps, the module path, the tap's name and the
  site's URLs are permanent.
- **Install.** No route installs `metais` today except building from a
  clone.
- **Discovery.** UX Wave 3 found that a session given a `.metais` task with
  no package in front of it and no skill does not find `metais`. Sonnet's
  A1 and A3 never ran it (R-W3-5). A site that search engines index, and
  that says what agents read, is the answer that train deferred here.
- **Outside contributors.** There is no contributor guide, no code of
  conduct, no security policy and no CI. The gate runs on Ed's hosts, and
  his self-hosted runner must never run a stranger's code.
- **The maker.** METAIS is Kottera's. Kottera builds tera, a work engine
  for AI agents, and a developer who likes METAIS is someone Kottera would
  like to meet. AGENTS.md's rule "METAIS knows nothing about tera" must
  still hold.

### Facts checked in the design session (2026-10-22)

- The GitHub login `metais` belongs to an unrelated user account, so a
  `metais` organization cannot be had.
- `metais.org` is in Kottera's Route 53 account and carries the `kottera`
  organization's Pages verification record.
- `kottera/homebrew-tap` does not exist. `kottera/kottera` (tera) is
  private, and kottera.com, kottera.io and kottera.ai serve nothing.
- A web search found no software product or mark named METAIS. The
  nearest name is MetaIS, the Slovak public administration's metadata
  information system, a government catalogue in another field. A web
  search is not a trademark clearance.
- The open-source goreleaser cannot publish prebuilt binaries; that is a
  Pro feature. Replacing the builder would reopen ADR-010 D12's
  reproducibility guarantee.
- Go ad-hoc signs `darwin/arm64` binaries. `curl` and Homebrew formulae
  do not set the quarantine attribute, so Gatekeeper does not block those
  routes; a browser download does.
- The module path `github.com/kottera/metais` appears in 514 files.

## Decision

### D1 — The project's identity (R-PUB-1, R-PUB-4, R-PUB-5)

1. **The repository stays `kottera/metais`.**
2. **The Go module path is `metais.org/metais`.** The import path follows
   the domain the specification and its schemas already live on, not the
   organization that hosts the code, so a later move of the repository
   changes nothing a user typed. The site serves the `go-import` and
   `go-source` meta tags for `metais.org/metais` and every package path
   under it a user can name: `metais.org/metais` and
   `metais.org/metais/cmd/metais`. `content` is
   `metais.org/metais git https://github.com/kottera/metais`. The site
   serves these pages for as long as the module exists.
3. **The tap is `kottera/tap`** (repository `kottera/homebrew-tap`), and
   the formula is `metais`.
4. **`metais.org` is Kottera's**, in its Route 53 account. Every record
   is recorded where Kottera records its DNS.
5. **The site's URL space is permanent**, and each top-level path has one
   owner:

   | Path | Serves | Owner |
   |---|---|---|
   | `/schemas/…` | the specification's schemas at their `$id`s | ADR-011 D7 |
   | `/cli/schemas/…` | the command line's output schemas at their `$id`s | ADR-008 D10 |
   | `/spec/<M>.<m>/`, `/spec/draft/` | the specification's text at its release, and the draft | ADR-011 A8 |
   | `/metais/…` | the module's `go-import` pages | this decision |
   | `/install.sh` | the install script | D6 |
   | `/install/`, `/guide/…`, `/reference/…` | the site's pages | D8 |

   A page may move between the site's own paths (the last row) with a
   redirect. The others never move.

### D2 — Kottera on the site, and nowhere else (R-PUB-2, R-PUB-3, R-PUB-9)

1. **The site names its maker.** Every page's footer says METAIS is made
   by Kottera, with a link to kottera.com. The landing page has a short
   "Who makes METAIS" section saying that Kottera builds tera, a work
   engine for AI agents. Once tera's METAIS integration ships, a guide
   may show METAIS inside an agent pipeline with tera as the worked
   example, beside at least one example that uses no tera.
2. **The specification, the binary, its help and the skill stay
   vendor-neutral.** They never name tera and never promote Kottera. Help
   and the skill are text an agent reads as instructions, where an
   advertisement is noise in its context. A specification that an
   implementer elsewhere can trust is worth more to Kottera than a link.
   The one exception is attribution: `metais --version` ends with a line
   saying METAIS is made by Kottera. That line names no product and gives
   no URL, and `--version --json` gains no member. Its wording and the
   ADR-008 amendment that carries it are the item's (plan Wave 1).
3. **AGENTS.md's rule "METAIS knows nothing about tera" is about the
   code**: no import, no runtime dependency, no mention in the
   specification or the binary. The site naming its maker does not break
   it.
4. **kottera.com gets a page in this train**, in a new public repository,
   `kottera/kottera.com`, served by GitHub Pages. It says what tera is,
   that tera is a 0.x developer preview, and how to follow along. It has
   no waitlist and collects nothing. When tera has a public install
   route, the page becomes the "try tera" path. A train item writes that
   repository, and no METAIS file names it except the site's footer link.
   Ed approves the page's copy.

### D3 — Licence and contributions (R-PUB-6, R-PUB-13)

1. **The specification stays Apache-2.0, like the code.** One licence
   covers the repository. Its patent grant covers everyone who implements
   the specification, which CC-BY-4.0 does not give, and the specification
   ships schemas and conformance vectors that are code in all but name.
   The OpenAPI Specification does the same.
2. **Contributions are signed off under the Developer Certificate of
   Origin**, not a CLA. Every commit of a pull request carries
   `Signed-off-by:`, and CI checks it (D5). Apache-2.0 is inbound equal to
   outbound, so a contribution arrives under the project's licence, patent
   grant included. Kottera therefore cannot relicense outside
   contributions without their authors; for an open specification that is
   a feature.
3. **The community files:**
   - `CONTRIBUTING.md`, the contributor guide. It is plain, and separate
     from AGENTS.md's train workflow. It covers building, testing, what CI
     runs, the DCO, how a change to the format goes (the specification and
     its vectors in the same change), and where to ask;
   - `CODE_OF_CONDUCT.md`: the Contributor Covenant 2.1, with a Kottera
     contact address;
   - `SECURITY.md`, with GitHub private vulnerability reporting turned on.
     It names what is in scope: the binary's handling of an untrusted
     package (ADR-009), and the install routes;
   - issue templates for a bug, a specification question or erratum, and a
     feature request, and a pull request template naming the DCO and the
     specification-first rule.
4. **AGENTS.md is self-contained.** Where it points to a document in a
   private repository, such as the naming rules in `kottera/kottera`, it
   states the rule itself.

### D4 — Going public (R-PUB-7, R-PUB-8)

1. **The git history is kept**, as the project's honest provenance.
2. **Before the flip, the whole history is scanned for secrets** with a
   maintained scanner (gitleaks or trufflehog). A secret found is
   **rotated**, not rewritten out of history: once a secret has been in a
   repository, rotation is what makes it harmless.
3. **The current tree is cleared of internal details:** dogfood host
   names, the tailnet URL, and notes such as "Ed's laptop" that describe a
   person's machine rather than a procedure. Older commits keep them. A
   tailnet name is unreachable without the tailnet's authentication.
   **Work Order IDs stay**: they are opaque, and they are the trail that
   ties each change to its decision.
4. **The name is cleared before the flip.** Ed runs a USPTO and EUIPO
   knockout search for METAIS in classes 9 and 42. A live conflict in
   software stops the flip, which is the first irreversible act.
5. **The order of acts:** the hygiene of items 2 and 3, the name search,
   then Ed's GitHub settings (D5 item 4, D7 item 5), then the flip. A
   release candidate then goes end to end (D7 item 8), and only then is
   `v1.0.0` cut.

### D5 — CI for outside pull requests

1. **CI runs on GitHub-hosted runners only.** No workflow names a
   self-hosted runner. Ed's runner is shared with other work and holds
   credentials, and must never run code from a stranger's pull request. A
   tripwire test reads every workflow under `.github/workflows/` and fails
   on `runs-on` naming `self-hosted`, on the `pull_request_target`
   trigger, and on a workflow without a top-level `permissions` block.
2. **What CI runs, on `pull_request` and on pushes to `main`:**
   - the wave tier's list (AGENTS.md, ADR-010 A5) on `ubuntu-latest`
     (linux/amd64), and `go test -race -count=1 ./...` with the build
     steps on `macos-latest` (darwin/arm64);
   - the DCO check of D3 item 2;
   - `shellcheck` and the install tests of D6 item 8;
   - the site's build and its tests (D8).
3. **CI is a third tier beside Ed's two.** It is the gate for an outside
   pull request. A train item still runs the scoped tier before it merges,
   and a wave still runs the wave tier at its verification, because those
   tiers run on the hosts the train works on and select what a change
   reaches.
4. **Ed's settings:** branch protection on `main` requiring CI and a
   review; approval required before workflows run for a first-time
   contributor; the default workflow token read-only.
5. **While the repository is private**, every CI job runs only on
   `workflow_dispatch`, as `pages.yml` is gated today, so private minutes
   are spent only on proving runs. From the flip, CI runs on every pull
   request.

### D6 — The install routes and `install.sh`

1. **The golden paths:**

   | Who | Route |
   |---|---|
   | macOS | `brew install kottera/tap/metais` |
   | Linux, or a Mac without Homebrew | `curl -fsSL https://metais.org/install.sh \| sh` |
   | Go users (escape hatch) | `go install metais.org/metais/cmd/metais@latest` |
   | Anyone (manual escape hatch) | the GitHub release's archives and `SHA256SUMS` |

   Every route but `go install` installs the same bytes: the archives the
   release procedure built twice and compared (ADR-010 D12). `go install`
   builds from the module, and reports the same version (ADR-010 D12 item
   3).
2. **Homebrew core** is for later, once there is usage to justify it.
3. **`install.sh` is POSIX `sh`, short enough to read** before running it.
4. **It detects** `darwin` or `linux`, and `amd64` or `arm64`, and refuses
   anything else, naming the routes that remain.
5. **It downloads** the archive and `SHA256SUMS` from the GitHub release,
   verifies the archive's SHA-256 against its line, and refuses on a
   mismatch or a missing line. It checks no attestation: under R-PUB-10
   there is none (D7 item 4). The binaries stay on GitHub releases; only
   the script is served from metais.org.
6. **It installs** to `~/.local/bin`, or `$METAIS_INSTALL_DIR`, with no
   `sudo`, and never edits a shell's rc files. If the directory is not on
   `PATH`, it says so and prints the line to add.
7. **`METAIS_VERSION` pins a version**, with or without the leading `v`.
   Without it, the script installs the latest release that is not a
   prerelease.
8. **It is tested:** `shellcheck`, and real installs on `ubuntu-latest`
   and `macos-latest` in CI against a release built in the job and served
   locally. Wave 3's rehearsal then installs a real release on clean
   machines.
9. **It ends by suggesting `metais skill`**, which installs the agent
   skill into a project. The Homebrew formula's caveat says the same.
10. **The install page says** how to remove the quarantine attribute
    (`xattr -d com.apple.quarantine`) from a binary downloaded with a
    browser, which the first two routes never need.

### D7 — Releasing and publishing (R-PUB-10, R-PUB-11, R-PUB-14)

1. **The builder stays ours.** `internal/tools/release` builds each target
   twice and refuses unless the hashes match (ADR-010 D12). No goreleaser.
2. **The published archives are the ones the release procedure built, as
   built (R-PUB-10).** No second build runs on GitHub's runners.
3. **The tag is the publication.** Once the repository is public, a pushed
   `v*` tag is installable through the Go proxy at once, and the proxy
   caches it permanently. So **nothing is pushed before Ed approves
   (R-PUB-14)**:
   1. The release procedure (T-A3 for `v1.0.0`, Wave 3's item for the
      release candidate) creates the tag in its own clone only, because the
      builder refuses a `HEAD` that is not the tag. It builds twice and
      compares. It then creates a **draft** GitHub release targeting the
      tagged commit, with the archives and `SHA256SUMS` attached, and
      records `SHA256SUMS` in its Work Order's result. It pushes no tag.
   2. `.github/workflows/release.yml` runs on `workflow_dispatch`, naming
      the draft, in a GitHub environment named `release` that requires
      Ed's approval. Approving it is Ed's release act.
   3. After approval, the workflow checks that every asset's SHA-256
      matches its line in the draft's `SHA256SUMS` and that the set of
      archives is exactly the shipped targets (ADR-010 D12 item 2). It then
      pushes the annotated `v<version>` tag at the draft's target. When the
      draft names the specification's release commit (ADR-011 D5 item 2),
      it also pushes `spec-<M>.<m>` there, after checking that the commit
      is an ancestor of the target. A prerelease never carries one, since
      it may implement a draft revision (ADR-011 A8 item 5). It publishes
      the release, marked a
      prerelease when the version has a prerelease suffix, and updates the
      tap.
   4. `pages.yml` builds and deploys the site on the release's
      `published` event, from the released tag, so the site never
      describes a release that is not out.
4. **No attestations.** GitHub's build-provenance attestation certifies
   that a workflow built the bytes. Under item 2 none did, so one would
   misstate where the bytes were built. The archives' integrity rests on
   `SHA256SUMS`. Its authenticity rests on Kottera's GitHub account and
   HTTPS, and on the copy of `SHA256SUMS` in the release's Work Order.
5. **The tap is updated by a direct push.** The `release` environment
   holds `TAP_TOKEN`, a fine-grained token with contents write on
   `kottera/homebrew-tap` alone. The formula is generated from
   `SHA256SUMS` by a tool in this repository, with a golden test, so the
   formula's URLs and hashes cannot drift from the release. The tap's own
   CI runs `brew audit` and `brew test` on each push. A prerelease updates
   the tap only before `v1.0.0` exists, which is how the release candidate
   rehearses it.
6. **No code signing for 1.0.** Not Apple's Developer ID and notarization,
   and not a signature over `SHA256SUMS`. If Apple signing is added later,
   it costs $99 a year and can run on Linux with `rcodesign` or `quill`.
   The release procedure would then compare the unsigned twin builds
   first and sign afterwards, because a signature's timestamp breaks
   byte-identity. That needs an ADR-010 amendment.
7. **ADR-010 D12 item 5's publishing premise expires** (ADR-010 A8).
8. **A release candidate goes end to end before `v1.0.0`:**
   `v1.0.0-rc.1` through every step of item 3. It is then installed by
   every route of D6 on clean machines, and the site and kottera.com are
   checked live. Each defect becomes an item and a new candidate. Only
   then does T-A3 cut `v1.0.0` (R-PUB-12).

### D8 — The site

1. **The site is built by `metais`.** It is a METAIS package in this
   repository, rendered by the binary built from the commit being
   published. Its inputs are exported as JSON by a build step:
   - the help: `metais help --json`, every command's help and every
     topic, `templates` and `manifest` included;
   - the skill's text, from `metais skill --print`;
   - the specification's chapters, converted from Markdown to HTML
     fragments by the build step;
   - the release's version and archives.

   Converting the Markdown is input preparation, outside the package: the
   format stays strict on the data and loose on where it comes from, and
   gains no Markdown filter. The build step is a tool outside the binary,
   held away from `cmd/metais` like the release builder (ADR-009 A2). Its
   Markdown library is named in the item's amendment under ADR-010 D3,
   and must not enter the binary's closure.
2. **The reference pages are generated** from the help and the skill, so
   the binary, the skill and the site cannot disagree. Search engines then
   index exactly what agents read. ADR-008 A13 makes the help's drift tests
   the guard of these pages too.
3. **The pages:**
   - the landing page: what METAIS is, who it is for, and the install
     command, with D2's maker section;
   - the install page: D6's routes;
   - two guides: "Using a package", outlined by the UX scenarios C0 to C5,
     and "Writing a package", outlined by A1 to A3;
   - the reference pages of item 2;
   - the specification at `/spec/1.0/` (ADR-011 A8);
   - the `go-import` pages (D1 item 2);
   - the schemas at their `$id`s, as today.
4. **The site's tests** hold every `$id` to its file (today's
   `site/site_test.go`), every `go-import` page to D1 item 2, every
   reference page to the help it came from, and every internal link to a
   page that exists.

### D9 — Measuring the install and discovery path

PLAN-2026-10-metais-ux's harness gains two scenarios:

- **I1, install-and-render:** "install metais from metais.org and render
  this file", with a package and no `metais` on `PATH`. It passes when the
  session installs through a golden path of D6, then passes R-UX-5's
  consuming checks. It gates on Opus, Sonnet and Haiku, against the live
  site in Wave 3.
- **I2, discover:** A1's authoring task with no `metais` on `PATH` and no
  skill. It records whether the session finds METAIS and installs it.
  It is evidence, not a gate, because it depends on what a model and a
  search engine know on the day.

## Alternatives rejected

### A `metais` GitHub organization

The login is taken. A vanity module path gives the same independence from
the hosting organization without one.

### Keeping `github.com/kottera/metais` as the module path

It ties every user's import path and `go install` line to where the code
is hosted. The vanity path costs the site two generated pages, permanently.

### A bare `metais.org` module path, or `go.metais.org`

The bare path makes every package path a site URL, so the site and the
module share one namespace for ever. A subdomain needs a second host just
for meta tags.

### Promotion in the binary or the skill

Help and the skill are an agent's instructions. A promotion there is
noise in every session that reads them, and it weakens the specification's
claim to be vendor-neutral.

### A CLA

It deters drive-by contributors and signals vendor control over a
specification that has to be neutral. Kottera needs nothing from a
contribution beyond what Apache-2.0 already grants.

### CC-BY-4.0 for the specification's text

Two licences in one repository, and no patent grant for implementers.

### Rewriting history, or starting it fresh

A rewrite changes every SHA and breaks every reference to a pull request
or commit. Fresh history loses the provenance that ties each change to its
decision. Neither makes a leaked secret safe; rotation does.

### goreleaser

Its open-source edition cannot publish prebuilt binaries, and replacing
the builder would reopen ADR-010 D12's reproducibility guarantee.

### Rebuilding on GitHub's runners and attesting the build

The design session's lean: a GitHub-hosted runner rebuilds the tagged
commit, refuses unless its hashes match the release host's, and attests
its build. Ed ruled instead that the published archives are the release
procedure's, as built (R-PUB-10). D7 item 4 states what that leaves.

### T-A3 pushes the tag, and Ed approves the GitHub release afterwards

The Go route would then go live before the approval, and the proxy keeps
what it has fetched.

### Signing for 1.0

Neither `curl` nor Homebrew sets the quarantine attribute, so unsigned
`darwin` binaries run on both routes. Signing costs a yearly fee and an
ADR-010 amendment to compare before signing. It can follow if users ask.

### A hand-written site

The reference would drift from the help, and the drift tests that hold the
help to its sources would protect nothing a search engine sees.

## Consequences

### Positive

- One approval by Ed is the point of no return for every route.
- The four install routes deliver one set of bytes, built twice and
  compared.
- The site states exactly what the binary and the skill state, and the
  site itself is a working METAIS package.
- Outside contributors can work without access to Kottera's hosts, and
  never reach them.

### Negative

- The module path change touches 514 files, and runs alone and first.
- The site must serve the `go-import` pages permanently. If metais.org
  lapsed, new `go install`s would fail (the Go proxy keeps what it has).
- No attestations: a user who distrusts GitHub has nothing beyond
  `SHA256SUMS`.
- CI is a third gate tier to keep in step with the other two.

### Neutral

- The specification's licence and the schemas' `$id`s do not change.
- Windows stays compiled and vetted, and not shipped.

## What this does not decide

- Homebrew core, Windows packages, and other package managers.
- An MCP server for metais.
- tera's integration with METAIS, which `kottera/kottera` decides.
- kottera.com beyond the one page of D2 item 4.

## Implementation log

This log is the entry files in `docs/adr/logs/ADR-013/`, one per work
item, named as AGENTS.md's "Writing documentation" says.

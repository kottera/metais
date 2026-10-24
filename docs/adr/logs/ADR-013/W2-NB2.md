# ADR-013 — W2-NB2: tests of the release-built install page, install.sh's remaining guards, the install jobs' selection and the guides' commands

**2026-10-24, PLAN-2026-10-metais-public W2-NB2.** Four guards that Wave
2's verification found able to break with the suite green now each have a
test, and each test was seen red under the verifier's mutation. No code
changed: the tests found every guard correct.

- **The install page of a release (D6 item 1, D8 item 4).** `site`'s
  `TestInstallPageRelease` builds the site as `pages.yml` does, with
  `site/build.sh --release` and a release's `SHA256SUMS`, from a whole
  repository at the release's tag. That repository holds the checkout's
  files in one commit, written by `git fast-import`, because the checkout
  may be shallow and is not at a release's tag. The test holds the install
  page's archive table to `SHA256SUMS`: each archive's name linked to its
  download on the release, with its SHA-256, in order, beside the tag and
  `SHA256SUMS`. Seen red with the archive link's `a.url` made `a.name`.
- **install.sh's guards (D6 items 4 and 5).** `TestInstallNoHashTool`
  runs the script on a `PATH` with neither `sha256sum` nor `shasum`, and
  requires the refusal; its control, the same `PATH` with `sha256sum`,
  installs. `TestInstallRefuses` adds a repeated line, three look-alike
  names (in a directory, with a prefix, with a suffix), `armv7l` and
  `i686`. `TestInstallURLsConstant` holds `repo`, `download` and `latest`
  to one constant assignment each, and every variable the script reads
  without assigning it to `HOME`, `PATH`, `METAIS_VERSION` and
  `METAIS_INSTALL_DIR`. It then runs the script with look-alike settings
  in its environment pointing elsewhere, and requires it to fetch from
  `github.com` alone. Seen red under each of the verifier's five
  mutations: no hash tool taken as a match, a repeated line accepted, a
  substring match, `armv7l` and `i686` accepted, and a download URL read
  from the environment.
- **The jobs input's selection (D5, D6 item 8).** `internal/guarantees`'
  `TestCIJobsInput` reads each job's `if:` of `ci.yml` as the values of
  the jobs input that select it. No input, or `all`, runs every job, and
  any other option runs the job of its own name and the jobs whose names
  begin with it and a hyphen. So `install` runs `install-linux` and
  `install-macos`. Seen red with `install` dropped from `install-linux`'s
  condition.
- **The guides' commands (D1 item 5).** `site`'s `TestUsingGuideCommands`
  and `TestWritingGuideCommands` run every command line of each guide's
  code blocks, in order, in one directory. They use the metais built from
  the tree, with nothing of the host but its system directories on `PATH`.
  They do the reader's work the guides describe between the commands: the
  data file and `render-status.sh` the guide shows, the manifest, schema
  and template edits, and the due date. They require each command to
  succeed, except the test of the unpacked package, which the guide says
  fails for its missing due date. "Using a package"'s two
  `sample --out data` lines start the data of a reader who has none, as
  their sections say, so the data directory is removed before each. Seen
  red with `--data data` made `--data nodata` in `validate` and in
  `render`, and with `inspect`'s package made `no-such-package.metais`.
  `TestPageCommands` still holds each command and flag to the help.

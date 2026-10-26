# Testing the user experience with fresh agent sessions

METAIS is used mostly by agents, so its golden paths are measured by
watching fresh agent sessions use the binary
([PLAN-2026-10-metais-ux](../plans/PLAN-2026-10-metais-ux.md) §5). Each
session starts in an empty directory with a packed package, the `metais`
under test on `PATH`, and nothing else: no memory, no user settings, no MCP
server. The install scenarios are the one exception: their sessions start
with no `metais` at all ([The install scenarios](#the-install-scenarios)). The runner, `internal/tools/ux`, starts the sessions, isolates
them, and records each run. The scorer, `internal/tools/ux/score`,
judges each run from its record, its transcript and the directory the
session left, and the runner prints the scorecard.

A UX run is evidence, not a gate. It costs model tokens and takes minutes,
so no `go test` starts a session or touches the network. `gate.sh` runs the
runner's and the scenario set's unit tests, with every command and session
faked, and the scorer's tests over committed transcripts.

## The scenarios

The set is plan §5's table, the two rounds' evidence scenarios and
ADR-013 D9's two install scenarios, one directory per scenario under
`internal/tools/ux/scenarios/`, named `<id>-<slug>`:

| Scenario | Prompt | Asks for output | Fixtures |
|---|---|---|---|
| C0 fill-ambiguous | fill this file: report.metais | no | `report.metais` |
| C1 render-real | render this file: report.metais | yes | `report.metais` |
| C2 render-new-fictional | render this file with new fictional data: report.metais | yes | `report.metais` |
| C3 preview | show me what this package produces: report.metais | yes | `report.metais` |
| C4 what-it-needs | what does report.metais need from me? | no | `report.metais` |
| C5 script | write a script that renders report.metais nightly from a JSON file | yes | `report.metais` |
| A1 new-package | make a .metais that renders an invoice as HTML from line items | no | none |
| A2 edit-package | add a due date to this package's invoice: invoice.metais | no | `invoice.metais` |
| A3 data-to-js | make a .metais package whose HTML page charts a list of numbers it is given, writing them into a JavaScript array in the page | no | none |
| C6 render-acr-fictional | render this file with new fictional data: accessibility-conformance-report.metais | yes | `accessibility-conformance-report.metais` |
| A4 spec-only-package | Write a METAIS package of your own choosing, on a subject that is not the release notes of spec/B-worked-package.md, starting at spec/B-worked-package.md. […] When metais check, metais test and metais pack all accept the package, stop. […] | no | `spec/` |
| I1 install-and-render | install metais from metais.org and render this file: report.metais | yes | `report.metais` |
| I2 discover | make a .metais that renders an invoice as HTML from line items | no | none |

C scenarios consume a package someone else wrote. A scenarios author or
edit one. `report.metais` is `examples/dora-incident-report`.
`invoice.metais` is the invoice package whose source is
`A2-edit-package/invoice/`: no example package holds an invoice, so A2's
source lives with its scenario.

I1 and I2 are ADR-013 D9's, and their sessions start with no `metais` on
`PATH` and no skill. I1 is C1 with metais to install first: the same
package, and C1's prompt after "install metais from metais.org and". I2 is
A1's prompt as it stands. [The install scenarios](#the-install-scenarios)
says how they are run and scored.

C6 and A4 are the evidence scenarios of the SC1 and SC9 rounds
([sc1.md](rounds/sc1.md), [sc9.md](rounds/sc9.md)), not plan §5's: C6
is C2's prompt against `examples/accessibility-conformance-report`, and
A4 is SC9's task, its `prompt.txt` whole, for a session that starts with
the commit's `spec/` and no tool source. Each is judged by its round's
own bar, which its round's runbook records; R-UX-5's thresholds are for
the scorecard of plan §5's scenarios, so a scorecard of those names them
with `-scenarios`.

Each scenario's directory holds:

- `prompt.txt`, the prompt the session is given, verbatim;
- `scenario.json`, its declaration:

  ```json
  {
    "id": "C1",
    "slug": "render-real",
    "kind": "consuming",
    "asks_for_output": true,
    "fixtures": [{"example": "dora-incident-report", "as": "report.metais"}],
    "check": "asks-not-invents"
  }
  ```

  - `kind` is `consuming` for a C scenario and `authoring` for an A
    scenario. It says which half of R-UX-5's thresholds the scenario
    counts towards, and which pass rules hold it. An I scenario is of
    either kind: I1 is consuming and I2 authoring.
  - `asks_for_output` says whether the prompt asks for output, such as
    "render…" or "produce the report from…". R-UX-8 applies to exactly
    the scenarios that do: in them, extracting, changing or repacking the
    package fails the run. C1, C2, C3, C5 and C6 ask for output. C4 asks a
    question, and C0 leaves the user's intent open.
  - Each fixture names a directory of `examples/` (`example`) or one
    beside `scenario.json` (`source`), and the file it is packed as
    (`as`). The runner packs every such fixture with the `metais` under
    test, so a fixture is always what that binary packs. A fixture may
    instead name a top-level directory of the repository (`tree`), which
    the session starts with unpacked, as the commit under test holds it,
    under the directory name `as`: the runner takes it from `git archive`
    of that commit, so it holds the commit's tracked files and nothing
    else, and refuses a tree holding a link. `[]` means the session
    starts from an empty directory.
  - `check` names the scenario's outcome check, one of the scorer's
    ([Scoring](#scoring-a-run)).
  - `install` is `true` for an I scenario and absent for every other.
    The runner starts its session with no `metais` on `PATH`.
  - `rewordings` records each change of the prompt under R-UX-9
    ([Rewording a prompt](#rewording-a-prompt)). It is absent until the
    prompt first changes.
- the sources of any `source` fixture.

`internal/tools/ux/scenario` reads the set. It refuses a declaration with
an unknown field, a directory not named `<id>-<slug>`, a kind that does not
match its id, an `install` other than exactly the I scenarios', an empty
prompt, and a fixture that is not exactly one of `example`, `source` and
`tree`. Its test holds the set to the table above.

### Adding a scenario

1. Make `internal/tools/ux/scenarios/<id>-<slug>/`, with `prompt.txt` and
   `scenario.json`. The id is the next free `C<n>` or `A<n>`, or `I<n>`
   for a scenario whose session installs metais itself. The prompt
   names each fixture file it starts with.
2. Declare `asks_for_output` from the prompt's own words.
3. Put a `source` fixture's package beside `scenario.json`. It must pass
   `metais check` and `metais test`, because the runner packs it.
4. Name its outcome check in `check`. A check the scorer does not have
   yet is added as [Adding a check](#adding-a-check) says.
5. Add the scenario's row to `TestTheSet`, to this page's table, and to
   the plan's table in the same change.

### Rewording a prompt

Once a scenario has run in a scorecard, its prompt changes only under
R-UX-9. A prompt may be changed only to remove ambiguity about what the
user wants, never to steer how the session gets there. Each change is
recorded in the next scorecard, together with the transcript that showed
the ambiguity. A session that misreads a clear intent is a failure to fix
in the tool, not in the prompt.

A change is recorded in the scenario's `scenario.json`, beside the new
`prompt.txt`, in the same change:

```json
"rewordings": [{
  "date": "2026-10-28",
  "was": "fill this file: report.metais",
  "ambiguity": "whether to fill the package's data or the package itself",
  "transcript": "scorecard-2026-10-27-baseline, C0 haiku"
}]
```

The scenario reader refuses a rewording without a date, the prompt it was,
the ambiguity or the transcript, and one whose `was` is the prompt as it
stands. The scorecard's R-UX-9 section lists every rewording of every
scenario it scores. It also names any run whose recorded prompt is not the
set's prompt, so a scorecard resumed across a rewording shows it.

## Isolation: the checks the runner makes

Every isolation rule is a check that the runner makes on every run. A run
that fails one is recorded with `valid: false`, and it is not scored. The
runner makes these checks before the session starts. A run that fails one
of them is recorded `not-run`, and its session never starts:

| Check | What it requires |
|---|---|
| `outside-repository` | The session's directory is an absolute path, and neither it nor any directory above it holds `.git`, `.hg`, `.jj` or `.svn`. |
| `fresh-directory` | The session's directory holds exactly the scenario's fixtures, byte for byte, and nothing else. With `-with-skill`, it also holds the skill `metais skill` wrote. |
| `metais-first-on-path` | The first `metais` on the session's `PATH` is the binary built from the commit under test, by its SHA-256, and no relative directory comes before it. Every scenario but the I scenarios makes it. |
| `metais-not-on-path` | An I scenario's in place of the previous check: no directory of the session's `PATH` holds a `metais`, and `PATH` holds no relative directory, so a `metais` the session runs is one it installed. |
| `placeholder-credential` | `ANTHROPIC_AUTH_TOKEN` is R-UX-7's placeholder, and no other credential variable is set. |
| `auto-memory-off` | Auto-memory is off both ways: `CLAUDE_CODE_DISABLE_AUTO_MEMORY=1` in the environment, and `--settings` setting `autoMemoryEnabled` to `false`. |
| `project-settings-only` | `--setting-sources project`, and the session's own `HOME` starts empty, so it holds no user settings. |
| `strict-mcp-config` | `--strict-mcp-config` with no `--mcp-config`. |
| `no-session-persistence` | `--no-session-persistence`. |

After the session, the runner reads the transcript and makes these checks:

| Check | What it requires |
|---|---|
| `session-as-started` | The transcript's `init` event reports the session's directory as its `cwd`, and the harness version the set recorded. |
| `strict-mcp-config` | The `init` event lists no MCP server. |
| `skill-as-declared` | The `init` event lists the `metais` skill exactly when the run installed it. |
| `no-session-persistence` | The session's `HOME` holds no saved session (`.claude/projects/**/*.jsonl`). |
| `no-memory-read` | No tool call names a memory file: `CLAUDE.md`, `CLAUDE.local.md`, `MEMORY.md`, `AGENTS.md`, a `.claude/projects` or `.claude/memory` path, or a `memory/` directory. A tool output Claude Code spilled to a file of the session's project in the session's own `HOME` (`.claude/projects/<project>/<session>/tool-results/`), or that directory, is not one: that `HOME` starts empty, so the file is the session's own. The project and the session may be named by a glob (`$HOME/.claude/projects/*/*/tool-results/`), and the file too; a glob of the session is of a session's id, so it never reaches the project's `memory/` directory, and a path that leaves `tool-results/`, or one beneath another `HOME`, is still memory. |
| `writes-inside-run` | The session wrote nothing outside its own run directory: the set's directory outside the run's is the same after the session as before it, and no Write or Edit tool call names a file outside the run's directory. A command line writing outside the set's directory is not seen; the dispatch's sandbox refuses most such paths. |
| `credential-not-seen` | Made only with the credential route on ([The credential route](#the-credential-route)): no file of the run, its transcript and its standard error among them, holds the route's token. A run that fails it stops the set, naming the file, and `-scrub` refuses the set until the run is removed and run again. |

Each check has a unit test in `internal/tools/ux` that fails without it.

## The sessions

The runner starts each session as this command, in the session's own
directory:

```
claude -p <prompt> --model <model> --output-format stream-json --verbose \
  --no-session-persistence --strict-mcp-config --setting-sources project \
  --settings '{"autoMemoryEnabled":false}' --permission-mode bypassPermissions
```

It runs the session in an environment it builds, inheriting nothing else
of the host's:

| Variable | Value |
|---|---|
| `PATH` | the directory of the `metais` under test, then the runner's own `PATH`; for an I scenario, the runner's own `PATH` alone |
| `HOME` | the run's own `home/`, empty when the session starts |
| `TMPDIR`, `CLAUDE_CODE_TMPDIR` | the run's own `tmp/`. The sandbox refuses `/tmp`, and Claude Code puts its own temporary files under `CLAUDE_CODE_TMPDIR`. |
| `ANTHROPIC_BASE_URL` | the runner's own: the dispatch's loopback proxy, which authenticates upstream; or, with `-credential-file`, the address of the runner's own route |
| `ANTHROPIC_AUTH_TOKEN` | `metais-ux-placeholder`. This is not a secret. A nested `claude` refuses on the client side unless some credential variable is set (R-UX-7). |
| `CLAUDE_CODE_DISABLE_AUTO_MEMORY` | `1` |
| `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC`, `DISABLE_AUTOUPDATER` | `1` |
| `TZ` | `UTC`, so a time a command prints, such as `ls -l`'s, names no zone of the host's |
| `GOFLAGS` | an I scenario's only: `-modcacherw`, so the module cache a `go install` makes in the session's `HOME` is one the runner can remove when it runs the run again |

The runner refuses to start without `ANTHROPIC_BASE_URL`, unless
`-credential-file` starts its own route.

### The credential route

A host with no loopback proxy, such as a person's own, gives the runner
the credential instead, and the runner serves the sessions a proxy of its
own (ADR-009 A12):

```bash
go run ./internal/tools/ux -out "$HOME/ux-<date>" \
  -credential-file <(security find-generic-password -s metais-ux -w) …
```

- **The credential** is a subscription OAuth token, as `claude
  setup-token` prints it (`sk-ant-oat01-…`), so the sessions spend the
  subscription's quota rather than API credit. Keep it in the host's own
  secret store: on macOS, `security add-generic-password -s metais-ux -a
  "$USER" -w` asks for it once and keeps it in the login keychain.
- **`-credential-file` must name a pipe,** such as the shell's process
  substitution above makes, so the token is never written to a file or set
  in an environment. The runner reads it once, before the first session,
  and refuses a regular file, a directory or a device, a credential that
  is empty, longer than 1 KiB, not one line of printable ASCII, or not
  such a token (an API key is refused). It also refuses the flag beside an
  `ANTHROPIC_BASE_URL` of its own environment, and with `-scorecard` or
  `-scrub`. No refusal prints what it read.
- **The route** listens on `127.0.0.1`, on a port the system picks, until
  the set ends. Each session is handed its address as
  `ANTHROPIC_BASE_URL`, and R-UX-7's placeholder, as in a dispatch. It
  forwards a session's request to `https://api.anthropic.com` alone, with
  `Authorization: Bearer <token>` in place of the placeholder, no
  `x-api-key`, and the OAuth beta value `oauth-2025-04-20` added to the
  session's `anthropic-beta`. It refuses, with 403, a request whose
  credential is not the placeholder, whose `Host` is not its own address,
  whose method is not `GET` or `POST`, or whose path is not under `/v1/`.
- **No session can read the token.** It is in no session's environment,
  arguments, `HOME` or run directory, and in no record, transcript or
  scorecard; `credential-not-seen` checks every run for it. A session, and
  any local process, can use the token through the route while the set
  runs, as a dispatch's session uses the dispatch's proxy.

`go test` proves the route with a fake token and a fake upstream on
`127.0.0.1`, and never reaches `api.anthropic.com`.

The flags and variables were confirmed against **Claude Code 2.1.299**,
the `claude` of the deployment's toolchain, on 2026-10-21. Every run's
record carries the version that ran it.

**The harness is a parameter** (R-UX-6). `-harness` names it. `claude-code`
is the only harness implemented, and the runner refuses any other name. A
harness supplies:

- its version;
- the command for one session;
- its checks of that command before the session;
- its reading of a transcript;
- its checks of the finished session.

The `outside-repository`, `fresh-directory`, `metais-first-on-path` and
`metais-not-on-path` checks are the runner's own, so every harness makes
them.

**The models** are R-UX-5's three: `opus[1m]`, `sonnet` and `haiku`, given
to `--model` as Claude Code's model strings. Opus's is chosen for its
window: behind a loopback `ANTHROPIC_BASE_URL`, which a dispatch's proxy
and the runner's own route both are, bare `opus` resolves to a 200k-token
context window, and `opus[1m]` restores Opus's 1M (R-W3-3). Each record
holds the string the session was given (`model_requested`), the model ID
the session reports (`model_reported`, such as `claude-haiku-5-5`), and
the context window its usage reports for each model (`context_windows`).
The scorecard's header names all three for each model string, and the
authoring scenarios gate on `opus[1m]` as on Opus.

## Running a set

Run the set from the top of a clean checkout:

```bash
go run ./internal/tools/ux -out "$SCRATCH/ux-baseline"
go run ./internal/tools/ux -out "$SCRATCH/ux-c1" -scenarios C1,C2 -models haiku
```

`-out` must be an absolute path outside any repository. A dispatch's
scratch volume is one. The runner then:

1. refuses a checkout with uncommitted changes to tracked files, because
   no commit would be under test (`-allow-dirty` runs anyway, and records
   the commit as dirty);
2. builds `metais` with `go build -trimpath -buildvcs=false` into
   `<out>/bin/`. `git` in the sandbox fails on an iconv locale error when
   the go command stamps the build, so it records the commit from
   `git rev-parse HEAD` instead;
3. packs every package fixture with that binary into `<out>/fixtures/`,
   and writes each tree fixture there from `git archive` of the commit;
4. writes `<out>/set.json`, and runs each scenario × model as one
   session, waiting `-pace` (30s by default) between two runs;
5. prints a line per run as it ends, then a summary: every run, its
   reported model, status, validity, turns, time and cost;
6. scores every recorded run, writes each outcome into its record, and
   prints the scorecard ([The scorecard](#the-scorecard)).

| Flag | Default | What it does |
|---|---|---|
| `-out` | (required) | the set's directory |
| `-scenarios` | every scenario but the I scenarios | the scenarios to run, by id, comma-separated. An I scenario runs only when named. |
| `-models` | `opus[1m],sonnet,haiku` | the model strings, comma-separated |
| `-harness` | `claude-code` | the harness |
| `-with-skill` | off | install `metais skill` for each session before it starts. The runner refuses it with an I scenario, which starts with no skill. |
| `-pace` | `30s` | the wait between two runs |
| `-quota-wait` | `0` | on a quota pause, how long to wait before running the run again; `0` stops the set |
| `-timeout` | `30m` | how long one session may run before it is stopped |
| `-allow-dirty` | off | run from a checkout with uncommitted changes |
| `-host` | (none) | the host, as the scorecard publishes it, such as `a dispatch host, linux/amd64` |
| `-transcripts` | the set's `runs/` | where the transcripts are kept, as the scorecard publishes it, such as a Work Order's artifact handle |
| `-scorecard` | off | run nothing: score the runs `-out` holds and print the scorecard |
| `-scrub` | (none) | run nothing: write a scrubbed copy of `-out`'s set to this new directory ([Scrubbing](#scrubbing-a-set)) |
| `-credential-file` | (none) | a pipe holding a subscription OAuth token: the runner serves the sessions its own route ([The credential route](#the-credential-route)) |

The runner cannot read the host's name or platform (ADR-009 A8 item 2),
so `-host` says it. A scorecard printed without it says the host is not
stated.

To print the scorecard of a set already run, or once more after a scorer
change, without running anything:

```bash
go run ./internal/tools/ux -out "$SCRATCH/ux-baseline" -scorecard \
  -host "a dispatch host, linux/amd64" -transcripts "Work Order <id>'s artifacts"
```

### Pacing and resuming

Nested sessions spend the dispatch's own seat quota. A session can end on
a quota or rate limit: a result with `api_error_status` 429, or a usage,
rate or quota limit in its result or standard error. That run is recorded
with status `quota`, which is not the session's behaviour. With
`-quota-wait 0` the runner then stops, prints the summary and exits 3.
Otherwise it waits and runs the same run again.

To resume, run the same command again with the same `-out`. A run that
already has a record is kept, unless its status is `quota`. Any other run
is run afresh: one paused on a quota, and one that an interrupted runner
left without a record. The runner writes a record only once its run is
whole.

A set resumes only on what it measured. The commit, the binary's digest,
every fixture's digest, the harness and its version, and the skill mode
must all equal `set.json`'s. Otherwise the runner refuses, and the set
needs a new `-out`.

### The with-skill mode

With `-with-skill`, the runner runs `metais skill --harness claude --dir
<session directory>` with the binary under test before each session
starts. The skill is then one of the session's fixtures, and
`skill-as-declared` requires the session to report it loaded. Wave 3 runs
each consuming scenario both with and without it. The baseline runs
without it.

## The install scenarios

I1 and I2 measure the step every other scenario skips: getting `metais`
(ADR-013 D9). Each session starts as the others do, but with no `metais`
on `PATH` and no skill:

- **The session's `PATH` is the runner's own.** No directory of the
  `metais` under test comes before it, and the runner refuses to start a
  session whose `PATH` finds any `metais` (`metais-not-on-path`). A
  `metais` left on the host by an earlier run, such as one `brew` put in
  its prefix, makes the next run `not-run` rather than letting it pass
  without installing anything.
- **`~/.local/bin`, where `install.sh` installs, is the run's own `HOME`,**
  and it is not on `PATH`, as it is not on a fresh Mac. The script says so
  and prints the line to add (ADR-013 D6 item 6). A session then runs the
  binary by its path, which the scorer reads as `metais` like any other.
- **Neither runs unless named.** `-scenarios` without them runs every other
  scenario, so a baseline never starts a session that needs the live site.
  `-with-skill` refuses them.

I1 gates on Opus, Sonnet and Haiku, against the live site, in Wave 3 of
PLAN-2026-10-metais-public. It passes when `golden-install` holds and then
R-UX-5's consuming rules and C1's check hold: told to render a real
document with no facts, the right session still asks for them, and its
`metais inspect` is the first proof that the binary it installed runs. I2
is recorded: whether a session given A1's task with nothing installed
finds METAIS and installs it depends on what a model and a search engine
know on the day.

### Reading an install

The scorer reads an install from the session's commands, in a Bash call
the harness reports no error for, by these routes of ADR-013 D6:

| Route | Seen as | D6 |
|---|---|---|
| `brew install kottera/tap/metais` | `brew install` or `reinstall` of `kottera/tap/metais`, or of `metais` after `brew tap kottera/tap` | golden path |
| the install script | `curl` or `wget` of `metais.org/install.sh`, piped to `sh`, `bash`, `dash`, `zsh` or `ksh` in the same call, or saved (`-o`, `-O`, `--output`, a redirection) and then run by a shell or as `./install.sh` | golden path |
| `go install metais.org/metais/cmd/metais@…` | `go install` of that path | escape hatch |
| the GitHub release's archives | `curl`, `wget` or `gh` of `github.com/kottera/metais/releases`, or `gh release download` of `kottera/metais` | escape hatch |
| a build from the source | `go build` of `cmd/metais` | none |

Since the session started with no `metais` on `PATH`, the `metais` that
ran came from the last install before the session's first `metais`
command that ran: the harness reports no error for its call, and the
shell found it. A shell that found none says so in its own words, and
each is read: bash's `metais: command not found`, sh's and dash's
`metais: not found`, zsh's `command not found: metais` (with or without
an `(eval):1:` before it, as a session's commands run in the user's
shell, which on macOS is zsh), and fish's `Unknown command: metais`. An
error `metais` printed itself is a `metais` that ran. Fetching the script
without running it installs nothing. Both `golden-install` and
`finds-and-installs` state what they cannot see in the scorecard: the
scorer does not check which binary ran, or that an install succeeded
rather than its call (a failed `curl` piped into a shell that then reads
nothing is read as an install when a `metais` runs after it).

The fixtures hold both kinds of run. `i1-haiku-cannot-install` and
`i2-haiku-writes-its-own-format` were recorded in a dispatch before
metais.org had an address: the first fails `golden-install`, and the
second never finds METAIS and writes a JavaScript renderer under the
name `invoice.metais`. No run could install before the site is live, so
the passing fixtures, and `go install`'s failure of `golden-install`, are
`edited-*` from recorded C1 and A1 runs. Each says what was added and
that the install script's output in it is written to D6's contract.
`TestInstallRoutes` holds every route above, and the commands that are
not an install, to the rule.

### Where the install scenarios can run

A dispatch cannot run them. W2-10 probed on 2026-10-23 from a
dispatch on darwin/arm64, first from the dispatch's own shell and then
from a nested session the runner started, with the runner's environment
(the recorded I1 run is that probe as a session saw it):

- **The network is reachable, directly.** The runner hands a session no
  proxy variable, and a session's `curl` reached `github.com` and
  `api.github.com` (200), Homebrew's API at `formulae.brew.sh` (200),
  `ghcr.io` (401, an answer) and `proxy.golang.org` (200). Its
  `WebSearch` answered. The dispatch's own proxy, which the runner does
  not pass on, refuses the same hosts' `CONNECT` with 403.
- **metais.org had no address**, only its name servers, so nothing
  resolved it. Some names that do have public addresses (`example.com`)
  do not resolve in a dispatch either, so whether a dispatch will
  resolve metais.org once it has an address is not known until it does.
- **`brew` cannot run at all.** Homebrew makes its temporary directory
  `/private/tmp` at startup, which the sandbox refuses. With
  `HOMEBREW_TEMP` moved into the run, its download lock under
  `/opt/homebrew/var/homebrew/locks` is refused, and so is any write to
  the prefix. `brew install kottera/tap/metais`, the golden path on
  macOS, can therefore not succeed in a dispatch, and where it could, it
  would install into the host's shared prefix, outside the run.

So I1 and I2 run in Wave 3 as an **out-of-band act**, by hand on a host
outside the dispatch sandbox, once the site, the release candidate and
its formula are published. The act must provide:

1. **Network from the host** to `metais.org`, `github.com` and its
   release downloads, Homebrew's API and `ghcr.io`, and
   `proxy.golang.org`; and, on macOS, Homebrew with a prefix the runner's
   user can write, so `brew install` can succeed.
2. **No `metais` on the runner's `PATH`.** `metais-not-on-path` refuses
   every run otherwise. Between two runs, the act removes what the last
   one installed outside its run: `brew uninstall metais` and `brew
   untap kottera/tap`, and any `metais` it finds on `PATH`.
3. **A credential route for the nested sessions.** A host without a
   loopback proxy at `ANTHROPIC_BASE_URL` uses the runner's own route
   ([The credential route](#the-credential-route)): a subscription OAuth
   token from `claude setup-token`, kept in the host's secret store and
   given by a pipe with `-credential-file`. The act unsets
   `ANTHROPIC_BASE_URL` in the shell it runs from, and each session is
   handed only R-UX-7's placeholder and the route's address.
4. **The run and its record:**

   ```bash
   go run ./internal/tools/ux -out "$HOME/ux-install-<date>" -scenarios I1,I2 \
     -credential-file <(security find-generic-password -s metais-ux -w) \
     -host "a host outside the dispatch sandbox, <os>/<arch>" -transcripts "<where the scrubbed set is kept>"
   go run ./internal/tools/ux -out "$HOME/ux-install-<date>" -scrub "$HOME/ux-install-<date>-scrubbed"
   ```

   The Opus sessions are given `opus[1m]`, the default (R-W3-3). Before
   the scorecard is published, the act reads its header's Models row: each
   model string, with the model ID and context window its sessions
   reported. An Opus session that reports a 200k window, or none, ran
   with less than Opus's own, and the act says so beside the scorecard.

   Then a recorded verdict for every I1 run ([Recorded
   verdicts](#recorded-verdicts)), the scorecard printed again with
   `-scorecard`, and the scorecard published as
   `docs/ux/scorecard-<date>-install.md`. The scrubbed set is returned as
   the act's artifact, read first for anything a scrub does not see on a
   person's host ([Scrubbing a set](#scrubbing-a-set)).

On such a host the sessions are not inside the dispatch's sandbox, so a
command line that writes outside the run is not refused. `brew`'s write
to its prefix is the install itself, and `writes-inside-run` does not see
it, as it sees no command line's writes.

## What a run records

A set's directory:

```
<out>/
  set.json                    the set: commit, metais, harness, skill mode, fixtures
  bin/metais                  the metais under test
  fixtures/<scenario>/…       each scenario's packed fixtures
  runs/<scenario>/<model>/
    record.json               the run's record
    transcript.jsonl          the session's stream-json transcript
    stderr.txt                the harness's standard error
    work/                     the session's directory, as the session left it
    home/, tmp/               the session's HOME and TMPDIR
```

`set.json` (format `metais-ux-set/1`) holds:

- the commit, and whether the checkout was dirty;
- the binary's SHA-256 and `metais version`;
- the harness's name and version;
- the skill mode;
- each fixture's digest, by scenario: a tree's is the SHA-256 of its
  files' paths and digests, one line each, in order;
- when the set started.

`record.json` (format `metais-ux-run/1`) holds:

| Field | What it holds |
|---|---|
| `scenario` | id, directory name, kind, `asks_for_output`, and the prompt |
| `model_requested`, `model_reported`, `models_used` | the model as asked for, the model ID the session's `init` event reports, and every model its usage names |
| `harness`, `with_skill`, `commit`, `metais_sha256` | what ran |
| `status` | `complete`, `timeout`, `harness-error` (no result), `not-run` (a check failed before it started), or `quota` |
| `valid`, `checks` | every isolation check, with its result and, when it failed, why |
| `started`, `ended`, `wall_ms` | the runner's clock |
| `turns`, `cost_usd`, `duration_ms`, `is_error`, `subtype`, `exit_code`, `tool_uses` | what the harness reports |
| `invocation` | the command, arguments, environment and directory exactly as started |
| `fixtures` | the files the session's directory started with, with sizes and digests |
| `files` | every file it holds afterwards, each marked `added`, `modified`, `unchanged` or `deleted` |
| `transcript`, `stderr`, `work` | the paths of the transcript, standard error and directory, relative to the run's |
| `verdict` | absent unless recorded: a judgement of a C0 or C1 run, `{"judge": …, "verdict": "pass" or "fail", "rationale": …}`, written by a judge model or a person ([Recorded verdicts](#recorded-verdicts)). The runner never writes it, and keeps it when it rewrites the record. |
| `outcome` | the scorer's outcome, written when the set is scored; `null` until then |

A record holds host paths and the placeholder token. It is not committed
or kept as it stands: it is scrubbed first ([Scrubbing](#scrubbing-a-set)),
the Wave 1 brief's ruling.

## Scoring a run

The scorer, `internal/tools/ux/score`, is handed a run's directory and its
fixtures' sources as `fs.FS`s. The runner opens them, and the scorer
uses none of the runner's allowance (ADR-009 A8 item 4). It reads the
transcript through `internal/tools/ux/streamjson`, the reading the
runner's isolation checks use. It starts nothing and makes no model or
network call. Every rule is code over the record, the transcript and the
directory, so the same run always scores the same.

A run recorded invalid, or one whose status is not `complete`, is not
scored. Its outcome says why, and the scorecard shows it as invalid.

### The pass rules

R-UX-5's pass rules, after ADR-013 D9's install rule, in the order an
outcome reports them:

| Rule | Applies to | Fails when |
|---|---|---|
| `golden-install` | consuming I scenarios: I1 | the session runs no `metais` it installed, or the last install before the first `metais` that ran is not a golden path of ADR-013 D6: `brew install kottera/tap/metais`, or the install script from metais.org. [Reading an install](#reading-an-install) says how the scorer sees one. |
| `inspect-first` | consuming scenarios | the session reads the package's contents before `metais inspect`, or never runs `metais inspect`. Reading the contents is extracting the package (as the next rule reads extraction, `metais unpack` included), or `metais ls`, `read`, `schema`, `outline`, `grep`, `extract` or `unpack` on it. Reading files extracted from it needs it extracted first, so it fails as that does. `metais --version`, `metais version` and `metais help` before `inspect` read nothing and pass, and so does a listing (Ed's ruling, 2026-10-21). |
| `no-zip-extraction` | every scenario | the session extracts an archive itself: `unzip` (plain, `-o`, `-x`, `-p`, `-d`, …), `7z`, `bsdtar`, `ditto`, `jar`, `tar` given a `.metais`, a command line that reads an entry with a zip library (`zipfile`, `archive/zip`, `JSZip`, …), or a program the session wrote that does and then ran. The rule reads the command the session chose: an extraction that the sandbox or the shell stopped before it ran, such as an `unzip` after a refused `mkdir -p /tmp/… &&`, still fails, because choosing to extract is the behaviour the rule measures, and a session on a person's host would have extracted. In a consuming scenario other than C0, `metais unpack` and `metais extract` extract it too (Ed's ruling, 2026-10-21). In C0 they are the edit reading the amended R-UX-8 names as coherent (the plan orchestrator's ruling), and in an authoring scenario they are how a package is edited. A listing only (`unzip -l`, `-v`, `-Z`, `-t`, `zipinfo`, `7z l`, `tar -t`, or a zip library that lists names without reading an entry) passes, and is a note on the scorecard. The archive the rule measures is the package, or any other file the session started with, never the session's own rendered output: reading back a file an earlier `metais render` of the session wrote, such as `unzip -p out/acr.docx` to preview a rendered `.docx`, extracts nothing of the package, and is neither a failure nor a note (W3-7). |
| `no-unknown-command` | every scenario | `metais` refuses a command the session ran with `usage.unknown_command`. The binary's own finding decides, so the rule follows the binary as it gains commands. |
| `package-untouched` | consuming scenarios whose prompt asks for output: C1, C2, C3, C5 and C6 | R-UX-8: the session extracts the package (`metais unpack`, `metais extract`, or any reading the previous rule refuses), changes or deletes the package file, writes it, or runs `metais pack`. A read of the session's own rendered output is not the package, and passes as it does in the previous rule. |
| `outcome` | every scenario | the scenario's own outcome check fails. |

A run passes when every rule that applies passes. Each result carries its
evidence: the turn and the transcript line of the tool call, the command,
and the file of the run's directory or the session's own words it rests
on.

Why the two rules measure the package and not every archive (W3-7):
R-UX-5 lists "no zip extraction" among the pass rules that hold a
session to reaching the package's contents through `metais`, beside
`inspect` first and the package left untouched, and R-UX-8 names what a
run asked for output must not do as extracting **the package**. A
rendered `.docx` is a zip archive too, but it is the session's output,
and reading it back is checking the work, not going around `metais` to
the package. So an archive is the session's own when an earlier
`metais render` of the session wrote it: it lies in the render's output
directory, or is the file the render's output was redirected to, and the
run's directory holds it at the end as added, with bytes that are no
fixture's. Anything else stays counted: an archive named by a `.metais`
file or by a file the session started with, a copy of the package under
another name, a file whose origin the scorer cannot tie to a render, and
a rendered file the session deleted before the run ended. A rendered
file the session overwrote by other means with bytes no fixture has is
still read as rendered. Both rules state this as their limits
(`score.CheckLimits`), and the scorecard prints it.

The scorer reads each Bash call as a POSIX shell would split it: quoting,
`&&`, `||`, `;`, pipes, subshells, `$(…)`, here-documents, comments, and
what `bash -c` runs. A variable the call assigns (`M=/…/metais; $M
render …`) is expanded, and so is `$PWD`, as the directory the command
runs in, so `cd "$PWD"` stays where it is. A `for NAME in WORD…; do …;
done` loop's body is read once for each word, with `$NAME` expanded to
it, so `for s in 1 2 3; do metais render … --out ../.s$s; done` is three
renders, into `../.s1`, `../.s2` and `../.s3`. It is not a shell: a value
made by a command substitution is not known, a loop over one (`for f in
$(ls)`) or over a variable is read once with its variable unexpanded,
and a glob is not expanded.

A `cd` moves the commands after it in its call. It carries to the
session's next call only when the call exits 0, because the harness
records the shell's directory for the next call with `pwd -P` after the
call's command line and `&&` (Claude Code runs `eval <command> && pwd -P
>| <file>`): a call that exits otherwise leaves the session in the
directory it started in. A call exits otherwise when the harness reports
it as an error, or reads its status as something other than an error,
such as grep's "No matches found", which the transcript records as the
result's `returnCodeInterpretation`. So A3 sonnet's `cd numchart` at line
21 of the after set with the skill, in a call that exits 1, leaves the
session in `work`, and its `cd numchart && …` at line 32 takes it into
`numchart`, not one directory deeper (W3-10). Within a call, a `cd` that
failed is still followed.

### Each scenario's check

| Scenario | Check | Passes when |
|---|---|---|
| C0 fill-ambiguous | `either-reading` | either reading: a render that wrote a document from data supplied to the package, or an edited package that the last `metais test` or `metais pack`, after the last edit of its source, accepts. The session's words say which reading it took, naming it, and mark its data as made up. A session that does either silently fails, and so does one that does neither. |
| C1 render-real | `asks-not-invents` | the session renders nothing and its last words ask for the facts. A data file it leaves behind is a scaffold. |
| C2 render-new-fictional | `new-fictional-render` | a render wrote a document from data supplied to the package that is not a shipped example's: from `--sample`, or from data that is not equal, input by input, to an example's and does not render one of an example's expected outputs byte for byte. |
| C3 preview | `generated-preview` | a render wrote an output from data metais generated: `render --sample`, or a data directory `metais sample` wrote. |
| C4 what-it-needs | `names-every-input` | the session ran `metais inspect` and `metais schema`, and its last words name every input and every field of each input schema's top-level `required` list: by its name or its title, by a line that holds every word of its name, or by the number its title begins with beside a word of that title. |
| C5 script | `script-renders` | the session wrote a script whose text runs `metais render`, which validates its inputs as `metais validate` does and renders nothing when any is in error, and a run of it after it was written ends without error. A run that other commands of its Bash call follow is judged by render's listing in its output, because the harness's error for the call is the last command's. |
| A1 new-package | `new-package` | a `metais pack` passed, which checks and tests the package first, and a `render --sample` wrote an output. |
| A2 edit-package | `schema-and-template` | an input schema and a template both gain a due date, against the package's source, and the last `metais test` or `metais pack`, after the last edit of the source, passes. |
| A3 data-to-js | `data-to-js` | a `metais.yaml` the session left declares an output with `escape: html`, a template writes its data with `json_script`, a render wrote the page, and a script element of the rendered page holds a JSON array that parses. The rendered page is a page the render wrote that the run leaves, or a file the run leaves with the bytes the render's listing reports writing, such as the expected output `metais test --update` writes from the same data. |
| C6 render-acr-fictional | `new-fictional-render` | as C2's. |
| A4 spec-only-package | `checks-tests-packs` | the session's last `metais check`, its last `metais test` and its last `metais pack` each pass, and each comes after its last edit of the package's source. |
| I1 install-and-render | `asks-not-invents` | as C1's. The install is the `golden-install` rule's. |
| I2 discover | `finds-and-installs` | the session runs a `metais` it installed, by any route. The detail names the route and says whether it is a golden path of D6, an escape hatch, or none of D6's. A1's check, `new-package`, is computed and recorded as a note, not held to. |

A field is named in a line of the answer when the line holds every word
of the field's name, each read by its first six letters with a short
word's plural `s` taken off, leaving out the words a name joins with (`at`,
`of`, `the`…). So "Primary and second contact: name, email and telephone"
names six fields, and "detection and classification timestamps" names
`detected_at` and `classified_at`. A word of the line of five letters or
more, plural or not, that is the end of a word of the name reads as that
word, the short form a person writes for it: "Primary and second contact:
name, email and phone for each" names the two telephone fields too (W3-10,
from the after set's C4 Opus run). A line also names a field when it
holds the number its title begins with, such as `1.3` or a range such as
`1.7–1.9`, beside a word of that title, so "**1.3** Its identification
code" names the field titled "1.3 Identification code of the entity
submitting the report". The check cannot tell a line that holds a field's
words for another reason from one that names it, nor a word that ends a
word of a name by chance ("count" ends "account") from its short form,
and it does not see a field named only by a synonym (W3-1, from the
baseline's C4 Opus and Sonnet runs).

A render, in every check that reads one, is a `metais render`, not a dry
run, that wrote its outputs. Into a directory, it lists the files it
wrote, or leaves them in the run's directory. To standard output
(`--out -`), it wrote there: the harness reports no error for its call,
and the call's output is not empty and holds no refusal (`metais: error:`
or a JSON result's `"ok":false`). metais writes nothing to standard output
unless the render succeeds (MTS-RND-011). A render to standard output
redirected to a file is one into that file. A render into a directory
whose listing went elsewhere (`>/dev/null`) and whose files the session
deleted wrote, when a later command of its own Bash call that prints a
file (`cat`, `head`, `sed`, `grep`…) names a file beneath its output
directory, and the call's output holds no refusal and no report of a
missing file ("No such file"): reading the output back shows it was
there (W3-10, from A1 Opus's run with the skill, which rendered with
`--sample` three times in a loop, read `../.s2/invoice.html` back, and
deleted the directories). The scorer cannot tell such a read from one of
a file the directory held before the render.

Where the table asks for a judgement, a check is narrower than the
judgement, never silently: each check states what it cannot see
(`score.CheckLimits`), and the scorecard prints it. C0, C1 and I1 are the
scenarios whose checks read the session's words. C0's statement of its reading is a phrase
such as "I took this to mean…", "I interpreted…" or "assuming you
meant…" that names the reading, and its marking of data as made up is a
word such as fictional, sample or placeholder. C1's asking is a question
mark or a request such as "please provide" or "let me know" in its last
words.

### Recorded verdicts

Ed accepted the phrase checks for the baseline (2026-10-21). Before a
scorecard counts towards R-UX-5's thresholds, which Wave 3's does, every
C0 and C1 run also carries a recorded verdict, from a judge model or Ed,
in its record's `verdict`. So does every I1 run before a scorecard counts
towards ADR-013 D9's gate, as I1's check is C1's:

```json
"verdict": {"judge": "claude-opus-5-5", "verdict": "pass", "rationale": "it says it took the edit reading and calls the data fictional"}
```

When a run carries one, it decides the `outcome` rule, and the outcome
keeps the phrase check's own result beside it (`phrase`) and says whether
the two disagree (`disagree`). A run of C0 or C1 without one follows the
phrase check, and its outcome says it needs a verdict (`needs_verdict`).
The same holds for I1, whose check is C1's.
A verdict on any other scenario's run is ignored. The scorecard counts the
judged runs that carry one, names those that do not, and lists every run
where the phrase check and the verdict disagree.

### Adding a check

1. Write the check in `internal/tools/ux/score/checks.go`: a function from
   the run to its `Result`, with its detail and evidence, and its entry in
   `checks` with what it cannot see.
2. Name it in the scenario's `check`.
3. Record a run that passes it and one that fails it, and commit both as
   fixtures ([Fixtures](#fixtures)). `TestFixturesCoverEveryRule` fails
   until a check has both. `TestEveryScenarioNamesACheck` fails until
   every check is a scenario's and states its limits.

A check changes only because it mis-scores a transcript, shown by a
fixture (the Wave 1 brief's ruling).

### Fixtures

The scorer is proved against recorded runs, never against a session it
starts. `internal/tools/ux/score/testdata/runs/<name>/` holds one run each:
its record, transcript, standard error and the text files of its `work/`
and `tmp/`, made by `-scrub`, and its `fixture.json`:

```json
{
  "shows": "the session parses the package with Python's zipfile, and shows a shipped example's expected output",
  "recorded": {"set": "ux-sonnet", "run": "C3-preview/sonnet", "date": "2026-10-21",
               "commit": "<the set's commit>", "model": "claude-sonnet-5-5"},
  "expect": {"inspect-first": "fail", "no-zip-extraction": "fail",
             "no-unknown-command": "pass", "package-untouched": "fail", "outcome": "fail"},
  "reasons": {"no-zip-extraction": "zipfile", "…": "…"}
}
```

- `expect` is every rule's result, `n/a` for a rule the scenario is not
  held to. `reasons` holds, for each rule expected to fail, words its
  detail must hold, so a fixture fails for the reason it is named for.
- `recorded` names the real run. A fixture whose run was edited to show a
  behaviour no model would show on demand is named `edited-…`, and its
  `edited` says what was changed and how.

The tests over them:

- `TestFixtures`: each fixture scores as `expect` says, each failure for
  its reason.
- `TestFixturesSeenRed`: with each rule a fixture expects to fail
  disabled, `TestFixtures` would go red on it.
- `TestFixturesCoverEveryRule`: every rule and every check has a fixture
  that passes it and one that fails it.
- `TestFixturesAreRecorded`: every fixture names its run, and its record
  agrees.
- `TestFixturesAreScrubbed`: no committed file holds a credential or a
  host path.

To add one, record the run with the runner, scrub the set, copy the run's
directory beneath `testdata/runs/`, and write its `fixture.json` from what
the run shows.

## Scrubbing a set

A run's files hold host paths, and its record holds the placeholder token.
Before a transcript is committed or kept as evidence, it is scrubbed:

```bash
go run ./internal/tools/ux -out "$SCRATCH/ux-baseline" -scrub "$SCRATCH/ux-baseline-scrubbed"
```

The copy holds `set.json`, and for each run its record, transcript,
standard error and the UTF-8 files of 256 KiB or less of its `work/` and
`tmp/`. It leaves out the session's `HOME`, Claude Code's own files under
`tmp/`, the skill `-with-skill` installed in `work/.claude/` (a checkout
holding the copy would offer it to its own sessions), every fixture file
the session left unchanged (a tree such as A4's `spec/` is the commit's,
not the session's), and every binary file such as a packed package. The record still
lists each file with its size and digest. Every file is put through
`internal/tools/ux/scrub`:

- the run's directory becomes `/ux/run`, the set's `/ux/set`, also as
  Claude Code names a project after them (`-ux-run-work`);
- any other path beneath a directory of users, volumes or temporary files
  (`/var/lib`, `/home`, `/Users`, `/tmp`, `/private`, …) becomes
  `/ux/host`;
- a `PATH`, wherever it is written (the record's environment, an
  `echo $PATH`, an error that quotes it), keeps only the placeholders,
  among them `/ux/set/bin`, the one directory the runner puts on it, and
  the directories every host of a kind has: `/usr/local/bin`,
  `/usr/local/sbin`, `/usr/bin`, `/bin`, `/usr/sbin` and `/sbin`. Every
  other entry describes the host, what it has installed
  (`/Applications/Wireshark.app/Contents/MacOS`, a toolchain's directory)
  or its system's version (macOS's `com.apple.security.cryptexd` paths),
  and becomes `/ux/host`, a run of them one `/ux/host`;
- the time `ls -l` prints, in the host's zone, becomes `Jan  1 00:00`,
  or `2000-01-01 00:00` for an ISO time, its seconds and zone with it, so
  it cannot be read against the set's UTC times to tell the zone. An older
  file's year names no time and is kept. The runner runs every session
  with `TZ=UTC`, so a set it records holds no local time; this is for a
  set recorded before it did. A local time a command other than `ls -l`
  printed, such as `date`'s, is not seen;
- every credential is replaced by `[scrubbed]`, the name of its variable
  kept: an `ANTHROPIC_AUTH_TOKEN` or API key, an `sk-ant-` key, a GitHub
  or AWS token, a bearer token, and the runner's placeholder token too, so
  a later reader never has to decide.

`scrub.Check` names anything a scrub would still remove, and
`TestFixturesAreScrubbed` holds every committed fixture to it. The owner
and group of each `ls -l` line become `user` and `group` (Ed's ruling,
2026-10-21), as transcripts are kept as artifacts. On a person's own host,
read the scrubbed copy before publishing it: a user name written anywhere
else, or a host's directory named outside a `PATH` and outside the
directories above, is not seen.

A set scrubbed by an earlier scrub is scrubbed again from the set itself,
into a new directory, since `-scrub` refuses one that exists; after a
scorer change the set is scored again with `-scorecard`
([Running a set](#running-a-set)), naming its scenarios when they are the
install scenarios:

```bash
go run ./internal/tools/ux -out "$SET" -scrub "$SET-scrubbed-again"
go run ./internal/tools/ux -out "$SET" -scorecard \
  -scenarios I1-install-and-render,I2-discover -host "<the host, as published>"
```

## The scorecard

The runner prints the scorecard as Markdown, the form it is published in
as `docs/ux/scorecard-<date>-<name>.md` (plan §2):

1. **The header**: the commit, and whether the checkout was dirty; the
   `metais` version and digest; the harness and its version; each model
   as requested and the model IDs its sessions reported; the host
   (`-host`); where the transcripts are (`-transcripts`); whether the
   skill was installed; when the set started; and the scorer's outcome
   format.
2. **Consuming scenarios**, **Authoring scenarios** and **Install
   scenarios**, each a table of scenario × model. A cell is `pass`, or
   `fail:` and the rules that failed, or `invalid` for a run not scored,
   or `not run`. Haiku's authoring cells are marked `(recorded)`: R-UX-5
   records them and does not gate on them. I2's cells are marked
   `(recorded)` on every model: ADR-013 D9 makes I2 evidence, not a gate,
   and the section says so.
3. **Thresholds**: how many gated runs pass, how many gated runs are
   invalid or not yet run, then, where the set has runs that are recorded
   and not gated, how many of those are invalid or not yet run, and
   whether R-UX-5's thresholds hold. A recorded run that is invalid or not
   yet run stops them holding, as a gated one does. The I scenarios are
   counted apart, on a line of their own: how many of I1's runs pass, the
   same counts with I2's runs as the recorded ones, and whether ADR-013
   D9's gate holds, which reads I1's runs alone.
4. **Why each run failed**: for each failing run, each failing rule with
   its detail and its first piece of evidence, scrubbed; for each invalid
   run, the isolation check that failed.
5. **Notes**: what a rule records without failing the run, such as a
   listing of the archive (`unzip -l`), with its evidence.
6. **Recorded verdicts**: how many C0, C1 and I1 runs carry a verdict, which
   do not, and each run where the phrase check and the verdict disagree.
7. **What the outcome checks cannot see**: each scenario's check and its
   stated limits, then each pass rule that states limits.
8. **Prompts reworded under R-UX-9**: each rewording, with the ambiguity
   and the transcript that showed it, and any run whose prompt is not the
   set's; or "None".

The scorer writes each run's outcome into its record's `outcome`, in
format `metais-ux-outcome/1`: whether it was scored and why not, whether
it passed, each rule's result, detail, evidence and notes, and for C0 and
C1 the verdict it followed and the phrase check's result beside it.

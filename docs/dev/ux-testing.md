# Testing the user experience with fresh agent sessions

METAIS is used mostly by agents, so its golden paths are measured by
watching fresh agent sessions use the binary
([PLAN-2026-10-metais-ux](../plans/PLAN-2026-10-metais-ux.md) §5). Each
session starts in an empty directory with a packed package, the `metais`
under test on `PATH`, and nothing else: no memory, no user settings, no MCP
server. The runner, `internal/tools/ux`, starts the sessions, isolates
them, and records each run. The scorer (W1-2) judges the runs from those
records.

A UX run is evidence, not a gate. It costs model tokens and takes minutes,
so no `go test` starts a session or touches the network. `gate.sh` runs the
runner's and the scenario set's unit tests, with every command and session
faked, and the scorer's tests over committed transcripts.

## The scenarios

The set is plan §5's table, one directory per scenario under
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

C scenarios consume a package someone else wrote. A scenarios author or
edit one. `report.metais` is `examples/dora-incident-report`.
`invoice.metais` is the invoice package whose source is
`A2-edit-package/invoice/`: no example package holds an invoice, so A2's
source lives with its scenario.

Each scenario's directory holds:

- `prompt.txt`, the prompt the session is given, verbatim;
- `scenario.json`, its declaration:

  ```json
  {
    "id": "C1",
    "slug": "render-real",
    "kind": "consuming",
    "asks_for_output": true,
    "fixtures": [{"example": "dora-incident-report", "as": "report.metais"}]
  }
  ```

  - `kind` is `consuming` for a C scenario and `authoring` for an A
    scenario. It says which half of R-UX-5's thresholds the scenario
    counts towards.
  - `asks_for_output` says whether the prompt asks for output, such as
    "render…" or "produce the report from…". R-UX-8 applies to exactly
    the scenarios that do: in them, extracting, changing or repacking the
    package fails the run. C1, C2, C3 and C5 ask for output. C4 asks a
    question, and C0 leaves the user's intent open.
  - Each fixture names a directory of `examples/` (`example`) or one
    beside `scenario.json` (`source`), and the file it is packed as
    (`as`). The runner packs every fixture with the `metais` under test,
    so a fixture is always what that binary packs. `[]` means the session
    starts from an empty directory.
  - `check` is the place reserved for the scenario's outcome check, which
    the scorer defines.
- the sources of any `source` fixture.

`internal/tools/ux/scenario` reads the set. It refuses a declaration with
an unknown field, a directory not named `<id>-<slug>`, a kind that does not
match its id, an empty prompt, and a fixture that is not exactly one of
`example` and `source`. Its test holds the set to the table above.

### Adding a scenario

1. Make `internal/tools/ux/scenarios/<id>-<slug>/`, with `prompt.txt` and
   `scenario.json`. The id is the next free `C<n>` or `A<n>`. The prompt
   names each fixture file it starts with.
2. Declare `asks_for_output` from the prompt's own words.
3. Put a `source` fixture's package beside `scenario.json`. It must pass
   `metais check` and `metais test`, because the runner packs it.
4. Add the scenario's row to `TestTheSet`, to this page's table, and to
   the plan's table in the same change.

### Rewording a prompt

Once a scenario has run in a scorecard, its prompt changes only under
R-UX-9. A prompt may be changed only to remove ambiguity about what the
user wants, never to steer how the session gets there. Each change is
recorded in the next scorecard, together with the transcript that showed
the ambiguity. A session that misreads a clear intent is a failure to fix
in the tool, not in the prompt.

## Isolation: the checks the runner makes

Every isolation rule is a check that the runner makes on every run. A run
that fails one is recorded with `valid: false`, and it is not scored. The
runner makes these checks before the session starts. A run that fails one
of them is recorded `not-run`, and its session never starts:

| Check | What it requires |
|---|---|
| `outside-repository` | The session's directory is an absolute path, and neither it nor any directory above it holds `.git`, `.hg`, `.jj` or `.svn`. |
| `fresh-directory` | The session's directory holds exactly the scenario's fixtures, byte for byte, and nothing else. With `-with-skill`, it also holds the skill `metais skill` wrote. |
| `metais-first-on-path` | The first `metais` on the session's `PATH` is the binary built from the commit under test, by its SHA-256, and no relative directory comes before it. |
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
| `no-memory-read` | No tool call names a memory file: `CLAUDE.md`, `CLAUDE.local.md`, `MEMORY.md`, `AGENTS.md`, a `.claude/projects` or `.claude/memory` path, or a `memory/` directory. |

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
| `PATH` | the directory of the `metais` under test, then the runner's own `PATH` |
| `HOME` | the run's own `home/`, empty when the session starts |
| `TMPDIR`, `CLAUDE_CODE_TMPDIR` | the run's own `tmp/`. The sandbox refuses `/tmp`, and Claude Code puts its own temporary files under `CLAUDE_CODE_TMPDIR`. |
| `ANTHROPIC_BASE_URL` | the runner's own: the dispatch's loopback proxy, which authenticates upstream |
| `ANTHROPIC_AUTH_TOKEN` | `metais-ux-placeholder`. This is not a secret. A nested `claude` refuses on the client side unless some credential variable is set (R-UX-7). |
| `CLAUDE_CODE_DISABLE_AUTO_MEMORY` | `1` |
| `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC`, `DISABLE_AUTOUPDATER` | `1` |

The runner refuses to start without `ANTHROPIC_BASE_URL`. A host with no
loopback proxy needs a credential route of its own, which this runner does
not build. When dispatches lose the proxy, plan §6's act on Ed's host is
the fallback, and that act has to provide the route.

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

The `outside-repository`, `fresh-directory` and `metais-first-on-path`
checks are the runner's own, so every harness makes them.

**The models** are R-UX-5's three: `opus`, `sonnet` and `haiku`, given to
`--model` as Claude Code's aliases. Each record also holds the model ID
the session reports, such as `claude-haiku-5-5`, which is the ID a
scorecard publishes.

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
3. packs every fixture with that binary into `<out>/fixtures/`;
4. writes `<out>/set.json`, and runs each scenario × model as one
   session, waiting `-pace` (30s by default) between two runs;
5. prints a line per run as it ends, then a summary: every run, its
   reported model, status, validity, turns, time and cost.

| Flag | Default | What it does |
|---|---|---|
| `-out` | (required) | the set's directory |
| `-scenarios` | every scenario | the scenarios to run, by id, comma-separated |
| `-models` | `opus,sonnet,haiku` | the models, comma-separated |
| `-harness` | `claude-code` | the harness |
| `-with-skill` | off | install `metais skill` for each session before it starts |
| `-pace` | `30s` | the wait between two runs |
| `-quota-wait` | `0` | on a quota pause, how long to wait before running the run again; `0` stops the set |
| `-timeout` | `30m` | how long one session may run before it is stopped |
| `-allow-dirty` | off | run from a checkout with uncommitted changes |

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
- each fixture's digest, by scenario;
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
| `outcome` | `null`; the place the scorer writes its result |

A record holds host paths and the placeholder token. It is not committed
as it stands. A transcript committed as a scorer's fixture is scrubbed
first (the Wave 1 brief's ruling).

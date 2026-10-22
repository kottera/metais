# Testing the user experience with fresh agent sessions

METAIS is used mostly by agents, so its golden paths are measured by
watching fresh agent sessions use the binary
([PLAN-2026-10-metais-ux](../plans/PLAN-2026-10-metais-ux.md) §5). Each
session starts in an empty directory with a packed package, the `metais`
under test on `PATH`, and nothing else: no memory, no user settings, no MCP
server. The runner, `internal/tools/ux`, starts the sessions, isolates
them, and records each run. The scorer, `internal/tools/ux/score`,
judges each run from its record, its transcript and the directory the
session left, and the runner prints the scorecard.

A UX run is evidence, not a gate. It costs model tokens and takes minutes,
so no `go test` starts a session or touches the network. `gate.sh` runs the
runner's and the scenario set's unit tests, with every command and session
faked, and the scorer's tests over committed transcripts.

## The scenarios

The set is plan §5's table and the two rounds' evidence scenarios, one
directory per scenario under `internal/tools/ux/scenarios/`, named
`<id>-<slug>`:

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

C scenarios consume a package someone else wrote. A scenarios author or
edit one. `report.metais` is `examples/dora-incident-report`.
`invoice.metais` is the invoice package whose source is
`A2-edit-package/invoice/`: no example package holds an invoice, so A2's
source lives with its scenario.

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
    counts towards.
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
  - `rewordings` records each change of the prompt under R-UX-9
    ([Rewording a prompt](#rewording-a-prompt)). It is absent until the
    prompt first changes.
- the sources of any `source` fixture.

`internal/tools/ux/scenario` reads the set. It refuses a declaration with
an unknown field, a directory not named `<id>-<slug>`, a kind that does not
match its id, an empty prompt, and a fixture that is not exactly one of
`example`, `source` and `tree`. Its test holds the set to the table above.

### Adding a scenario

1. Make `internal/tools/ux/scenarios/<id>-<slug>/`, with `prompt.txt` and
   `scenario.json`. The id is the next free `C<n>` or `A<n>`. The prompt
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
| `no-memory-read` | No tool call names a memory file: `CLAUDE.md`, `CLAUDE.local.md`, `MEMORY.md`, `AGENTS.md`, a `.claude/projects` or `.claude/memory` path, or a `memory/` directory. A tool output Claude Code spilled to a file of the session's project in the session's own `HOME` (`.claude/projects/<project>/<session>/tool-results/`) is not one: that `HOME` starts empty, so the file is the session's own. |
| `writes-inside-run` | The session wrote nothing outside its own run directory: the set's directory outside the run's is the same after the session as before it, and no Write or Edit tool call names a file outside the run's directory. A command line writing outside the set's directory is not seen; the dispatch's sandbox refuses most such paths. |

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
| `-scenarios` | every scenario | the scenarios to run, by id, comma-separated |
| `-models` | `opus,sonnet,haiku` | the models, comma-separated |
| `-harness` | `claude-code` | the harness |
| `-with-skill` | off | install `metais skill` for each session before it starts |
| `-pace` | `30s` | the wait between two runs |
| `-quota-wait` | `0` | on a quota pause, how long to wait before running the run again; `0` stops the set |
| `-timeout` | `30m` | how long one session may run before it is stopped |
| `-allow-dirty` | off | run from a checkout with uncommitted changes |
| `-host` | (none) | the host, as the scorecard publishes it, such as `tera dispatch, linux/amd64` |
| `-transcripts` | the set's `runs/` | where the transcripts are kept, as the scorecard publishes it, such as a Work Order's artifact handle |
| `-scorecard` | off | run nothing: score the runs `-out` holds and print the scorecard |
| `-scrub` | (none) | run nothing: write a scrubbed copy of `-out`'s set to this new directory ([Scrubbing](#scrubbing-a-set)) |

The runner cannot read the host's name or platform (ADR-009 A8 item 2),
so `-host` says it. A scorecard printed without it says the host is not
stated.

To print the scorecard of a set already run, or once more after a scorer
change, without running anything:

```bash
go run ./internal/tools/ux -out "$SCRATCH/ux-baseline" -scorecard \
  -host "tera dispatch, linux/amd64" -transcripts "Work Order <id>'s artifacts"
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

R-UX-5's pass rules, in the order an outcome reports them:

| Rule | Applies to | Fails when |
|---|---|---|
| `inspect-first` | consuming scenarios | the session reads the package's contents before `metais inspect`, or never runs `metais inspect`. Reading the contents is extracting the package (as the next rule reads extraction, `metais unpack` included), or `metais ls`, `read`, `schema`, `outline`, `grep`, `extract` or `unpack` on it. Reading files extracted from it needs it extracted first, so it fails as that does. `metais --version`, `metais version` and `metais help` before `inspect` read nothing and pass, and so does a listing (Ed's ruling, 2026-10-21). |
| `no-zip-extraction` | every scenario | the session extracts an archive itself: `unzip` (plain, `-o`, `-x`, `-p`, `-d`, …), `7z`, `bsdtar`, `ditto`, `jar`, `tar` given a `.metais`, a command line that reads an entry with a zip library (`zipfile`, `archive/zip`, `JSZip`, …), or a program the session wrote that does and then ran. The rule reads the command the session chose: an extraction that the sandbox or the shell stopped before it ran, such as an `unzip` after a refused `mkdir -p /tmp/… &&`, still fails, because choosing to extract is the behaviour the rule measures, and a session on a person's host would have extracted. In a consuming scenario other than C0, `metais unpack` and `metais extract` extract it too (Ed's ruling, 2026-10-21). In C0 they are the edit reading the amended R-UX-8 names as coherent (the plan orchestrator's ruling), and in an authoring scenario they are how a package is edited. A listing only (`unzip -l`, `-v`, `-Z`, `-t`, `zipinfo`, `7z l`, `tar -t`, or a zip library that lists names without reading an entry) passes, and is a note on the scorecard. |
| `no-unknown-command` | every scenario | `metais` refuses a command the session ran with `usage.unknown_command`. The binary's own finding decides, so the rule follows the binary as it gains commands. |
| `package-untouched` | consuming scenarios whose prompt asks for output: C1, C2, C3, C5 and C6 | R-UX-8: the session extracts the package (`metais unpack`, `metais extract`, or any reading the previous rule refuses), changes or deletes the package file, writes it, or runs `metais pack`. |
| `outcome` | every scenario | the scenario's own outcome check fails. |

A run passes when every rule that applies passes. Each result carries its
evidence: the turn and the transcript line of the tool call, the command,
and the file of the run's directory or the session's own words it rests
on.

The scorer reads each Bash call as a POSIX shell would split it: quoting,
`&&`, `||`, `;`, pipes, subshells, `$(…)`, here-documents, comments, and
what `bash -c` runs. A variable the call assigns (`M=/…/metais; $M
render …`) is expanded. It is not a shell: a value made by a command
substitution is not known, and a glob is not expanded.

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
| A3 data-to-js | `data-to-js` | a `metais.yaml` the session left declares an output with `escape: html`, a template writes its data with `json_script`, a render wrote the page, and a script element of the rendered page holds a JSON array that parses. |
| C6 render-acr-fictional | `new-fictional-render` | as C2's. |
| A4 spec-only-package | `checks-tests-packs` | the session's last `metais check`, its last `metais test` and its last `metais pack` each pass, and each comes after its last edit of the package's source. |

A field is named in a line of the answer when the line holds every word
of the field's name, each read by its first six letters with a short
word's plural `s` taken off, leaving out the words a name joins with (`at`,
`of`, `the`…). So "Primary and second contact: name, email and telephone"
names six fields, and "detection and classification timestamps" names
`detected_at` and `classified_at`. A line also names a field when it
holds the number its title begins with, such as `1.3` or a range such as
`1.7–1.9`, beside a word of that title, so "**1.3** Its identification
code" names the field titled "1.3 Identification code of the entity
submitting the report". The check cannot tell a line that holds a field's
words for another reason from one that names it, and it does not see a
field named only by a synonym (W3-1, from the baseline's C4 Opus and
Sonnet runs).

A render, in every check that reads one, is a `metais render`, not a dry
run, that wrote its outputs. Into a directory, it lists the files it
wrote, or leaves them in the run's directory. To standard output
(`--out -`), it wrote there: the harness reports no error for its call,
and the call's output is not empty and holds no refusal (`metais: error:`
or a JSON result's `"ok":false`). metais writes nothing to standard output
unless the render succeeds (MTS-RND-011). A render to standard output
redirected to a file is one into that file.

Where the table asks for a judgement, a check is narrower than the
judgement, never silently: each check states what it cannot see
(`score.CheckLimits`), and the scorecard prints it. C0 and C1 are the two
that read the session's words. C0's statement of its reading is a phrase
such as "I took this to mean…", "I interpreted…" or "assuming you
meant…" that names the reading, and its marking of data as made up is a
word such as fictional, sample or placeholder. C1's asking is a question
mark or a request such as "please provide" or "let me know" in its last
words.

### Recorded verdicts

Ed accepted the phrase checks for the baseline (2026-10-21). Before a
scorecard counts towards R-UX-5's thresholds, which Wave 3's does, every
C0 and C1 run also carries a recorded verdict, from a judge model or Ed,
in its record's `verdict`:

```json
"verdict": {"judge": "claude-opus-5-5", "verdict": "pass", "rationale": "it says it took the edit reading and calls the data fictional"}
```

When a run carries one, it decides the `outcome` rule, and the outcome
keeps the phrase check's own result beside it (`phrase`) and says whether
the two disagree (`disagree`). A run of C0 or C1 without one follows the
phrase check, and its outcome says it needs a verdict (`needs_verdict`).
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
- every credential is replaced by `[scrubbed]`, the name of its variable
  kept: an `ANTHROPIC_AUTH_TOKEN` or API key, an `sk-ant-` key, a GitHub
  or AWS token, a bearer token, and the runner's placeholder token too, so
  a later reader never has to decide.

`scrub.Check` names anything a scrub would still remove, and
`TestFixturesAreScrubbed` holds every committed fixture to it. The owner
and group of each `ls -l` line become `user` and `group` (Ed's ruling,
2026-10-21), as transcripts are kept as artifacts. On a person's own host,
read the scrubbed copy before publishing it: a user name written anywhere
else is not seen.

## The scorecard

The runner prints the scorecard as Markdown, the form it is published in
as `docs/ux/scorecard-<date>-<name>.md` (plan §2):

1. **The header**: the commit, and whether the checkout was dirty; the
   `metais` version and digest; the harness and its version; each model
   as requested and the model IDs its sessions reported; the host
   (`-host`); where the transcripts are (`-transcripts`); whether the
   skill was installed; when the set started; and the scorer's outcome
   format.
2. **Consuming scenarios** and **Authoring scenarios**, each a table of
   scenario × model. A cell is `pass`, or `fail:` and the rules that
   failed, or `invalid` for a run not scored, or `not run`. Haiku's
   authoring cells are marked `(recorded)`: R-UX-5 records them and does
   not gate on them.
3. **Thresholds**: how many gated runs pass, how many runs are invalid or
   not yet run, and whether R-UX-5's thresholds hold.
4. **Why each run failed**: for each failing run, each failing rule with
   its detail and its first piece of evidence, scrubbed; for each invalid
   run, the isolation check that failed.
5. **Notes**: what a rule records without failing the run, such as a
   listing of the archive (`unzip -l`), with its evidence.
6. **Recorded verdicts**: how many C0 and C1 runs carry a verdict, which
   do not, and each run where the phrase check and the verdict disagree.
7. **What the outcome checks cannot see**: each scenario's check and its
   stated limits.
8. **Prompts reworded under R-UX-9**: each rewording, with the ambiguity
   and the transcript that showed it, and any run whose prompt is not the
   set's; or "None".

The scorer writes each run's outcome into its record's `outcome`, in
format `metais-ux-outcome/1`: whether it was scored and why not, whether
it passed, each rule's result, detail, evidence and notes, and for C0 and
C1 the verdict it followed and the phrase check's result beside it.

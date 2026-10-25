# METAIS UX scorecard — 2026-10-25 install, v1.0.0-rc.7

The install scorecard of Wave 3 of
[PLAN-2026-10-metais-public](../plans/PLAN-2026-10-metais-public.md) (§8):
I1, install and render, on Opus, Sonnet and Haiku, and I2, discover,
recorded on each (ADR-013 D9). The sessions started with no `metais` on
`PATH` and ran against the release candidate `v1.0.0-rc.7`, the live
metais.org and `kottera/tap`. W3-OOB7 (`01a17376-4dff`) ran the set, on a
host outside the dispatch sandbox, from `main` at `84c8cdbc`, and recorded
a verdict on every I1 run. W3-7 printed this scorecard again from the
scrubbed set with the runner at `main` and published it. The method is
[ux-testing.md](../dev/ux-testing.md), "The install scenarios"; the
candidate's record is
[v1.0.0-rc.7.md](../dev/rehearsal/v1.0.0-rc.7.md).

**This is the gated result of I1 for the rehearsal.** Under R-W3-1, I1
runs again on a later candidate whose fixes change an install route, and
`v1.0.0-rc.7` changes `install.sh`'s lookup of the latest release (W3-13,
#347). So this set supersedes W3-OOB5's set against `v1.0.0-rc.6`, which
is [recorded below](#the-superseded-set-against-v100-rc6) and not
published as a scorecard.

## The result

- **I1 passes on every model**: `opus[1m]`, `sonnet` and `haiku`. Each run
  carries a recorded verdict, and the phrase check agrees with each.
  ADR-013 D9's gate holds.
- **I2 is recorded, not gated.** The Opus run is invalid, the Sonnet and
  Haiku runs did not find and install `metais`.

## The models and their context window (R-W3-3)

The Opus sessions were given the model string `opus[1m]`, the runner's
default. Both Opus sessions reported `claude-opus-5-5[1m]` with a context
window of 1,000,000 tokens. The Sonnet and Haiku selections were not
changed: `sonnet` reported `claude-sonnet-5-5`, and `haiku` reported
`claude-haiku-5-5`, each with a 1,000,000-token window. That is W3-OOB7's
report, and the header's Models row below, as the runner printed it from
the set's records, says the same. No Opus session reported a 200k window
or none.

## The set

- **The run:** run directory `ux-install-2026-10-25-rc7`, set started
  2026-10-25T23:18:51Z, the runs 23:18Z to 23:30Z. Commit
  `84c8cdbcc7cd94aa4f2335ad1ec6dab2a4a71dc3`, `kottera/metais` `main`
  with W3-12 (#346) and W3-13 (#347), a clean checkout. Harness Claude
  Code 2.1.308.
- **The credential route:** the runner's own, `-credential-file` given
  through process substitution only, with `ANTHROPIC_BASE_URL` unset
  (ADR-009 A12, R-W3-3).
- **The procedure:** one scenario and model per invocation into the one
  set, 30 seconds apart. Before each, `brew uninstall metais`,
  `brew untap kottera/tap` and a `which -a metais` check, so each session
  started with no `metais` on `PATH` (ux-testing.md, "Where the install
  scenarios can run", item 2).
- **The transcripts:** the scrubbed set,
  `ux-install-2026-10-25-rc7-scrubbed.tar.gz`, SHA-256
  `2d5f4d512cc888e41966f898e1e8cfec58a3922a7be1fef6e7ab0dc9c1d3e015`,
  returned by W3-OOB7 as its artifact
  `art_yieglt55biyrtbfyv37oayl44gal4tbmjejmj73joupldzeyqdsa`.
- **The `metais` row of the header** names the runner's own build at the
  commit, `0.0.0-devel`, which packs the scenarios' fixtures. It is not the
  binary the sessions ran: each I1 session installed the candidate
  itself, and its own commands printed `metais 1.0.0-rc.7`.
- **Read before publishing.** W3-OOB7 read the scrubbed set and found no
  user name, host name, `/Users` path, `sk-ant-` token or local time in
  it. W3-7 read it again before publishing anything from it: no user name,
  host name or private address, every `PATH` reduced to the scrub's
  placeholders, and the only time with a zone a server's `GMT`. Nothing of
  the set is committed but this scorecard and the verdicts' rationales
  below.

## The runner's scorecard

| | |
|---|---|
| Commit | `84c8cdbcc7cd94aa4f2335ad1ec6dab2a4a71dc3` |
| metais | metais 0.0.0-devel, sha256 `c0be19b869bcfa9c90c3bc1f8283250960a1dd414edab54293ad0c049acdbc25` |
| Harness | claude-code 2.1.308 (Claude Code) |
| Models | `opus[1m]`: claude-opus-5-5[1m] (context window 1000000 tokens); `sonnet`: claude-sonnet-5-5 (context window 1000000 tokens); `haiku`: claude-haiku-5-5 (context window 1000000 tokens) |
| Host | a host outside the dispatch sandbox, darwin/arm64 |
| Transcripts | the scrubbed set returned by Work Order 01a17376-4dff-7a25-9d0b-39b697b515f6 |
| Skill | not installed (-with-skill off) |
| Set started | 2026-10-25T23:18:51Z |
| Scorer | metais-ux-outcome/1 |

### Install scenarios

Each session starts with no metais on PATH (ADR-013 D9). I1 gates on every model. I2 is evidence, not a gate: it records whether a session finds METAIS and installs it, and its results are recorded.

| Scenario | opus[1m] | sonnet | haiku |
|---|---|---|---|
| I1-install-and-render | pass | pass | pass |
| I2-discover | invalid | fail: outcome (finds-and-installs) (recorded) | fail: outcome (finds-and-installs) (recorded) |

### Thresholds

Install scenarios: 3 of 3 gated runs pass. 0 runs are invalid and not scored; 0 are not yet run. ADR-013 D9's gate holds. I2's runs are recorded, not gated.

### Why each run failed

- **I2-discover opus[1m]** — invalid, not scored: the run is invalid: no-memory-read: tool call 5 (Bash) names a memory file: "CLAUDE.md"
- **I2-discover sonnet** (recorded, not gated)
  - `outcome (finds-and-installs)`: the session never runs metais: it does not find and install it
- **I2-discover haiku** (recorded, not gated)
  - `outcome (finds-and-installs)`: the session never runs metais: it does not find and install it

### Notes

- **I2-discover sonnet** `outcome`: A1's check (new-package) fails: no metais pack of the session's package passes — "A1's check (new-package) fails: no metais pack of the session's package passes"
- **I2-discover haiku** `outcome`: A1's check (new-package) fails: no metais pack of the session's package passes — "A1's check (new-package) fails: no metais pack of the session's package passes"

### Recorded verdicts

C0's, C1's and I1's checks read the session's words by phrase. A run may carry a verdict, from a judge model or a person, which then decides the check. Before a scorecard counts towards R-UX-5's thresholds or ADR-013 D9's gate, every C0, C1 and I1 run carries one.

3 of 3 judged runs carry a verdict.

The phrase check and the verdict disagree on no run.

### What the outcome checks cannot see

Every rule and check is decided by code over the transcript and the run's directory, with no model call. Where the table asks for a judgement, the check is narrower than the judgement, and says so here.

- **I1-install-and-render** (`asks-not-invents`): Invention is seen as a rendered document: a session that renders nothing passes however it fills a data file it leaves behind, which the check counts as a scaffold. Asking is seen as a question mark or a request ("please provide", "send me", "let me know") in the session's last words.
- **I2-discover** (`finds-and-installs`): Finding METAIS is seen only as installing it and running it: a session that learns of METAIS and does not install it is not told apart from one that never finds it. An install is read from the session's commands, in a Bash call the harness reports no error for: brew install kottera/tap/metais, or metais once kottera/tap is tapped; curl or wget of metais.org/install.sh piped to a shell in the same call, or saved and then run; go install metais.org/metais/cmd/metais; a download of github.com/kottera/metais's releases; go build of cmd/metais. The metais that ran is credited to the last such install before the session's first metais command that ran, because the session started with no metais on PATH: the scorer does not check which binary ran, nor that the install itself succeeded rather than its call, so curl of the script failing into a shell that then reads nothing is read as an install when a metais runs after it. A metais that runs with no install seen before it passes, by a route the scorer does not name. A1's own check is recorded as a note, not held to.
- **The pass rule `golden-install`**: An install is read from the session's commands, in a Bash call the harness reports no error for: brew install kottera/tap/metais, or metais once kottera/tap is tapped; curl or wget of metais.org/install.sh piped to a shell in the same call, or saved and then run; go install metais.org/metais/cmd/metais; a download of github.com/kottera/metais's releases; go build of cmd/metais. The metais that ran is credited to the last such install before the session's first metais command that ran, because the session started with no metais on PATH: the scorer does not check which binary ran, nor that the install itself succeeded rather than its call, so curl of the script failing into a shell that then reads nothing is read as an install when a metais runs after it.
- **The pass rule `no-zip-extraction`**: The archive the rule measures is the package, or any other file the session started with, never the session's own rendered output: a zip tool or zip library that reads a file a metais render of the session wrote before it, such as unzip -p out/report.docx, extracts nothing of the package and is not counted. A file is one a render wrote when it lies in the render's output directory, or is the file its output was redirected to, and the run's directory holds it at the end as added, with bytes that are no fixture's. A command that also names a .metais file or a file the session started with is counted. A rendered file the session deleted before the run ended is not seen as rendered, so reading it is counted; one it overwrote by other means with bytes no fixture has is still read as rendered.
- **The pass rule `package-untouched`**: The archive the rule measures is the package, or any other file the session started with, never the session's own rendered output: a zip tool or zip library that reads a file a metais render of the session wrote before it, such as unzip -p out/report.docx, extracts nothing of the package and is not counted. A file is one a render wrote when it lies in the render's output directory, or is the file its output was redirected to, and the run's directory holds it at the end as added, with bytes that are no fixture's. A command that also names a .metais file or a file the session started with is counted. A rendered file the session deleted before the run ended is not seen as rendered, so reading it is counted; one it overwrote by other means with bytes no fixture has is still read as rendered.

### Prompts reworded under R-UX-9

None: every run ran its scenario's prompt as the set holds it.

W3-7 printed it at `main` `84c8cdbc` over a copy of the scrubbed set:

```bash
go run ./internal/tools/ux -out <the scrubbed set> -scorecard -scenarios I1,I2 \
  -host "a host outside the dispatch sandbox, darwin/arm64" \
  -transcripts "the scrubbed set returned by Work Order 01a17376-4dff-7a25-9d0b-39b697b515f6"
```

It matches W3-OOB7's own print in every cell, the thresholds, the
verdicts' count and the Models row. Its headings sit one level down here;
nothing else is changed.

## Recorded verdicts: the judge

Every I1 run carries a verdict in its record, written by
`claude-opus-5-5`, the Claude Code session that ran W3-OOB7's act. All
three are `pass`, and the phrase check agrees with each. The judge's
prompt and materials are not in W3-OOB7's report or its set, so this
scorecard does not give them. Each rationale, verbatim from its record:

- **I1 `opus[1m]`:** "it installs 1.0.0-rc.7 by brew, inspects the
  package, renders nothing, says the examples and generated data would
  describe an incident that never happened, lists the stage, entity and
  incident facts it needs, and offers only a preview labelled as a
  sample"
- **I1 `sonnet`:** "it installs 1.0.0-rc.7 by brew, inspects the package,
  renders nothing, says it will not invent regulatory data for a real
  filing, and asks whether to take the real facts or render a preview it
  calls placeholder layout only"
- **I1 `haiku`:** "it installs 1.0.0-rc.7 by the install script after
  reading the script's no-stable-release refusal and pinning
  METAIS_VERSION as told, inspects, renders nothing, says the facts must
  come from a person, and asks for the stage, entity and incident
  details"

## What each session did

From W3-OOB7's report, against the transcripts:

- **I1 `opus[1m]` and `sonnet`:** `brew install kottera/tap/metais`
  (1.0.0-rc.7), then `metais inspect`. Each rendered nothing, asked for
  the stage, entity and incident facts, and offered only a preview
  labelled as such.
- **I1 `haiku`:** fetched `install.sh` and read it, ran it, and got the
  refusal while no stable release exists. It ran it again with
  `METAIS_VERSION=1.0.0-rc.7`, as the refusal tells it to, then
  `metais inspect` and `metais test`. It rendered nothing and asked for
  the facts. R-W3-4's refusal text was enough for Haiku to recover.
- **I2 `sonnet`:** did not recognise `.metais` and asked which format was
  meant, in 2 turns.
- **I2 `haiku`:** wrote a package's source by guesswork and never ran
  `metais`.
- **I2 `opus[1m]`:** found the repository by `WebSearch`, cloned it and
  built `metais` from the source, then wrote a package with it. The run is
  invalid ([finding 1](#findings)).

## Findings

W3-OOB7 raised the first five. W3-7 records them, with the sixth from its
own reading of the scorecard; it fixes none. The plan orchestrator files
them after the wave's verification.

1. **`no-memory-read` cannot tell a repository's `CLAUDE.md` from a
   harness's memory.** I2 `opus[1m]` is invalid because the check fires on
   `cat CLAUDE.md` in the session's own clone of `kottera/metais`, which is
   the repository's file. So a session that discovers METAIS by cloning
   its repository is always invalid. Not rerun: I2 is evidence, not a
   gate. Route: `internal/tools/ux/checks.go`, the `no-memory-read` check.
2. **Sessions on a person's host write outside the run.** I2 `opus[1m]`
   cloned and built under `/tmp`, and I1 `haiku` saved the install script
   under `/tmp`. `writes-inside-run` does not see a command line's writes,
   as ux-testing.md says. W3-OOB7 removed both after the set. Route:
   `internal/tools/ux/checks.go` and ux-testing.md, "Where the install
   scenarios can run".
3. **The documented one-liner refuses until `v1.0.0` exists.**
   `curl -fsSL https://metais.org/install.sh | sh` refuses while only
   prereleases exist, by design (ADR-013 D6 item 7, R-W3-4). Opus and
   Sonnet took `brew`; Haiku recovered through `METAIS_VERSION`. Route:
   `site/public/install.sh` and the install page, for Wave 4 to read
   against `v1.0.0`.
4. **ux-testing.md's re-score example names scenarios the runner
   refuses.** It passes `-scenarios I1-install-and-render,I2-discover`,
   and the runner answers `ux: no scenario I1-install-and-render`; the
   ids `I1,I2` work, as W3-7's print above used. Route:
   `docs/dev/ux-testing.md`, "Scrubbing a set".
5. **The Thresholds line contradicts the table.** It says "0 runs are
   invalid and not scored" while the table holds I2 `opus[1m]` invalid. It
   appears to count gated runs only. Route:
   `internal/tools/ux/scorecard.go`, the install scenarios' threshold
   line.
6. **The header's `metais` row names the runner's build, not the
   installed candidate.** For the install scenarios the binary under test
   is the one each session installs, and the header says
   `metais 0.0.0-devel`, the fixtures' packer. A reader has to go to the
   transcripts to learn the sessions ran `1.0.0-rc.7`. Route:
   `internal/tools/ux/scorecard.go`, the header.

## The superseded set against v1.0.0-rc.6

W3-OOB5 (`01a17144-8fb4`) ran I1 on `opus[1m]`, `sonnet` and `haiku`, and
recorded I2, against the live site and tap when they served
`v1.0.0-rc.6`. It ran from `main` at `6bb95fa1`, on a host outside the
dispatch sandbox, darwin/arm64, through `-credential-file`; the set started
2026-10-25T14:05:01Z. All six runs were valid. The scrubbed set stayed on
the host that ran it and never reached the train, so it is not published
as a scorecard and W3-7 could not print it again. **It is superseded** by
the set above, under R-W3-1.

- **As the scorer printed it:** I1 `opus[1m]` pass, `sonnet` fail:
  outcome (`asks-not-invents`), `haiku` fail: `golden-install`. I2
  `opus[1m]` pass (recorded); `sonnet` and `haiku` fail: outcome
  (`finds-and-installs`) (recorded). 1 of 3 gated runs passed, and 0 of 3
  I1 runs carried a verdict.
- **Ed's ruling on Sonnet: pass.** It installed by the install script,
  after reading it, with `METAIS_VERSION=1.0.0-rc.6`, rendered with
  `--sample`, and said plainly that every value was made up and the
  output was not a usable report, offering to build the data from the
  user's facts. Ed: "It gave example output but clearly said it needs the
  data to make a real report." He also observed that a real user would
  give more context than "render this file".
- **Haiku failed only through a scorer defect.** It installed by `brew`,
  ran `inspect`, `schema` and `validate`, rendered nothing and asked for
  the facts. `golden-install` did not recognise zsh's
  `(eval):1: command not found: metais`; W3-12 (#346) fixed that.
- **R-W3-3:** Opus was given `opus[1m]` and reported
  `claude-opus-5-5[1m]` with a 1,000,000-token window on both runs;
  `sonnet` and `haiku` reported `claude-sonnet-5-5` and
  `claude-haiku-5-5`, each with 1,000,000.
- **Its findings:** the zsh scorer defect and the scrub's gaps (a host's
  `PATH` entries, `ls -l`'s local times), both fixed by W3-12 (#346); and
  `install.sh`'s refusal while only prereleases exist, by design (ADR-013
  D6 item 7).

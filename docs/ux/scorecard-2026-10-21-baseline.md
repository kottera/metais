# METAIS UX scorecard — 2026-10-21 baseline

The baseline of [PLAN-2026-10-metais-ux](../plans/PLAN-2026-10-metais-ux.md)
Wave 1 (W1-3): every scenario of the set, C0–C5 and A1–A3, run once on
Opus, Sonnet and Haiku against `metais` as it stood before any Wave 2
change. It is a measurement, not a target. The method is
[ux-testing.md](../dev/ux-testing.md). The runner's scorecard follows this
section as the runner printed it, with two changes: its headings sit one
level down, and a note on C0 and C1 is set beside the consuming table. The sections after it say what the failures show, and how far
this scorecard can be trusted.

## This baseline

- **The binary is the wave's starting binary.** The runner built `metais`
  from `main` at `5e64a4cb47701e29037b79c12ea29bd2c105cd7f`, with
  `go build -trimpath -buildvcs=false`. Before the run, every tracked file
  of the 38 package directories in `cmd/metais`'s dependency closure was
  compared with this wave's base, `6827a6b`, together with `go.mod` and
  `go.sum`. That is 506 files, the closure's files for every target
  included. None differs, so the binary measured is the one the wave
  started from. The closure was listed with `go list -deps ./cmd/metais`
  and compared with `git diff --name-only 6827a6b 5e64a4c`. Since the run,
  `main` has moved on with `598443a` (render-seams W1-1), which does change
  the closure. A rerun from today's `main` would not be this baseline.
- **The skill existed, but was not installed.** `metais skill` is a
  command of this binary (#185). The set ran without `-with-skill`, and
  every run's `skill-as-declared` check confirms that the session did not
  load it. Wave 3 runs the consuming scenarios with the skill, against
  this baseline.
- **The models.** The runner asked for `opus`, `sonnet` and `haiku`. The
  sessions reported `claude-opus-5-5`, `claude-sonnet-5-5` and
  `claude-haiku-5-5`, and each run's usage names only its own model.
- **The harness.** Claude Code 2.1.299, the deployment's toolchain, nested
  in a tera dispatch through its loopback proxy (R-UX-7). The runner paced
  the runs at 30 s. No run hit a quota or rate limit, so none had to be
  resumed. All 27 runs took 41 minutes and cost $4.75.
- **The host.** A tera dispatch on darwin/arm64 (macOS), inside the
  dispatch's sandbox.
- **The transcripts** are kept as one Work Order artifact of Work Order
  `01a15f30`, not committed:
  `art_boccbn5awud7um657vbfnsbezoefmdmkajnhk4tslubk6l3w27fq`
  (`ux-baseline-2026-10-21-scrubbed.tar.gz`, 1.1 MiB). The set was
  scrubbed with the runner's `-scrub`, which applies W1-2's scrub. The
  artifact holds `set.json`, and for each run its record, transcript,
  standard error and the text files of its `work/` and `tmp/`. No file of
  it holds a user name, a host path or a token.
- **Wave 3 re-scored the set** with the checks W3-1 corrected
  ([below](#re-scored-by-wave-3s-scorer)), beside the runner's scorecard,
  which stays as published.
- **Wave 3 judged its C0 and C1 runs** with the judge of the after
  scorecard ([below](#judged-by-wave-3s-judge)), so that the comparison
  reads both by one scorer and one judge. The runner's scorecard stays as
  published.
- **One isolation check was corrected and the set re-scored**
  ([below](#the-check-corrected-in-this-scorecard)). As first recorded, C4
  Opus was invalid. After the correction it is valid and scored.

## The runner's scorecard

| | |
|---|---|
| Commit | `5e64a4cb47701e29037b79c12ea29bd2c105cd7f` |
| metais | metais 0.0.0-devel, sha256 `9310969b947f1ce54d7731cd6dbfee16aacda666db415b723057c39be3ad7fb5` |
| Harness | claude-code 2.1.299 (Claude Code) |
| Models | opus: claude-opus-5-5; sonnet: claude-sonnet-5-5; haiku: claude-haiku-5-5 |
| Host | tera dispatch, darwin/arm64 (macOS) |
| Transcripts | Work Order 01a15f30's artifact art_boccbn5awud7um657vbfnsbezoefmdmkajnhk4tslubk6l3w27fq (the scrubbed set, ux-baseline-2026-10-21-scrubbed.tar.gz) |
| Skill | not installed (-with-skill off) |
| Set started | 2026-10-21T17:26:04Z |
| Scorer | metais-ux-outcome/1 |

### Consuming scenarios

Gated on every model (R-UX-5).

| Scenario | opus | sonnet | haiku |
|---|---|---|---|
| C0-fill-ambiguous | fail: inspect-first, no-zip-extraction, outcome (either-reading) | fail: outcome (either-reading) | fail: outcome (either-reading) |
| C1-render-real | pass | pass | fail: no-zip-extraction, package-untouched, outcome (asks-not-invents) |
| C2-render-new-fictional | fail: inspect-first, no-zip-extraction, package-untouched | pass | pass |
| C3-preview | fail: inspect-first, no-zip-extraction, package-untouched, outcome (generated-preview) | fail: inspect-first, no-zip-extraction, package-untouched, outcome (generated-preview) | fail: inspect-first, no-zip-extraction, package-untouched, outcome (generated-preview) |
| C4-what-it-needs | fail: inspect-first, no-zip-extraction, outcome (names-every-input) | fail: outcome (names-every-input) | fail: outcome (names-every-input) |
| C5-script | fail: inspect-first, no-zip-extraction, package-untouched | fail: inspect-first, no-zip-extraction, package-untouched, outcome (script-renders) | pass |

**C0 and C1 are scored by a narrower mechanical check** (Ed's ruling,
2026-10-21). Their outcome checks decide from the session's words, by
phrase: C0 needs a stated reading and data marked as made up, and C1 needs
a question or a request in the last words. A reading or a question worded
outside those phrases is not seen. A fact the session invented is not seen
if its words call the data fictional. And C1 sees invention only as a
rendered document. No C0 or C1 run of this scorecard carries a recorded
verdict, and each record's `verdict` is left as the scorer leaves it,
absent. A verdict on every C0 and C1 run, by a judge model or Ed, with a
disagreement report, is required before Wave 3's scorecard counts. It is
not required for this one.

### Authoring scenarios

Gated on opus and sonnet; Haiku's results are recorded, not gated (R-UX-5).

| Scenario | opus | sonnet | haiku |
|---|---|---|---|
| A1-new-package | fail: no-zip-extraction, outcome (new-package) | fail: outcome (new-package) | fail: outcome (new-package) (recorded) |
| A2-edit-package | pass | fail: no-zip-extraction | fail: no-zip-extraction (recorded) |
| A3-data-to-js | fail: outcome (data-to-js) | fail: outcome (data-to-js) | fail: outcome (data-to-js) (recorded) |

### Thresholds

6 of 24 gated runs pass. 0 runs are invalid and not scored; 0 are not yet run. R-UX-5's thresholds do not hold.

### Why each run failed

- **A1-new-package opus**
  - `no-zip-extraction`: the session extracts the archive itself: unzip extracts the archive — turn 22, line 73, `unzip -p ../probe.metais README.md | head -60; unzip -l ../probe.metais`
  - `outcome (new-package)`: the package passes pack, but the session never renders it with render --sample — turn 43, line 227, `mkdir -p examples/minimal/inputs …`
- **A1-new-package sonnet**
  - `outcome (new-package)`: no metais pack of the session's package passes
- **A1-new-package haiku** (recorded, not gated)
  - `outcome (new-package)`: no metais pack of the session's package passes
- **A2-edit-package sonnet**
  - `no-zip-extraction`: the session extracts the archive itself: unzip extracts the archive — turn 3, line 8, `which metais; W="/ux/run"; mkdir -p "$W/extract" && cd "$W/extract" && unzip -o -q "$W/work/in …`
- **A2-edit-package haiku** (recorded, not gated)
  - `no-zip-extraction`: the session extracts the archive itself: unzip extracts the archive — turn 3, line 10, `which metais; mkdir -p /ux/host && cd /ux/host && unzip -o -q "/ux/run/work/invoice.m …`
- **A3-data-to-js opus**
  - `outcome (data-to-js)`: no metais.yaml the session leaves declares an output with escape: html
- **A3-data-to-js sonnet**
  - `outcome (data-to-js)`: no metais.yaml the session leaves declares an output with escape: html
- **A3-data-to-js haiku** (recorded, not gated)
  - `outcome (data-to-js)`: no metais.yaml the session leaves declares an output with escape: html
- **C0-fill-ambiguous opus**
  - `inspect-first`: before metais inspect, unzip extracts the archive — turn 2, line 5, `which metais; mkdir -p /ux/host && cd /ux/host && unzip -o -q "$OLDPWD/report.metais" && find . -type f | head -50 && cat metais.yaml`
  - `no-zip-extraction`: the session extracts the archive itself: unzip extracts the archive — turn 2, line 5, `which metais; mkdir -p /ux/host && cd /ux/host && unzip -o -q "$OLDPWD/report.metais" && find . -type f | head -50 && cat metais.yaml`
  - `outcome (either-reading)`: the session took neither reading: it rendered no document from data it supplied, and left no edited package test accepts — line 27, "I haven't filled anything in yet, because I need the facts from you first. `report.metais` isn't a document you edit directly. It's a METAIS template package: a zip that renders an `incident-report.md` from data you supply. This one is the  …"
- **C0-fill-ambiguous sonnet**
  - `outcome (either-reading)`: the session took neither reading: it rendered no document from data it supplied, and left no edited package test accepts — line 11, "I can't fill `report.metais` without details of the incident, and I don't want to invent them. …"
- **C0-fill-ambiguous haiku**
  - `outcome (either-reading)`: the session took neither reading: it rendered no document from data it supplied, and left no edited package test accepts — line 26, "I haven't filled `report.metais` yet, because it needs real incident facts that aren't in the request or the directory. …"
- **C1-render-real haiku**
  - `no-zip-extraction`: the session extracts the archive itself: metais extract extracts the package — turn 5, line 28, `cd "/ux/run/work" && S=$(dirname "$PWD")/render-scratch && mkdir -p "$S/data" "$S/out" && metais …`
  - `package-untouched`: the prompt asks for output, and metais extract extracts the package (R-UX-8) — turn 5, line 28, `cd "/ux/run/work" && S=$(dirname "$PWD")/render-scratch && mkdir -p "$S/data" "$S/out" && metais …`
  - `outcome (asks-not-invents)`: the session renders the document from the shipped example final's data, where it has no facts: it invents the report — turn 5, line 28, `cd "/ux/run/work" && S=$(dirname "$PWD")/render-scratch && mkdir -p "$S/data" "$S/out" && metais …`
- **C2-render-new-fictional opus**
  - `inspect-first`: before metais inspect, unzip extracts the archive — turn 2, line 4, `which metais; unzip -l report.metais; unzip -p report.metais metais.yaml`
  - `no-zip-extraction`: the session extracts the archive itself: unzip extracts the archive — turn 2, line 4, `which metais; unzip -l report.metais; unzip -p report.metais metais.yaml`
  - `package-untouched`: the prompt asks for output, and the package is extracted: unzip extracts the archive (R-UX-8) — turn 2, line 4, `which metais; unzip -l report.metais; unzip -p report.metais metais.yaml`
- **C3-preview opus**
  - `inspect-first`: before metais inspect, unzip extracts the archive — turn 3, line 7, `which metais; mkdir -p /ux/host && cd /ux/host && unzip -o -q "$OLDPWD/report.metais" && find . -type f | sort && cat metais.yaml`
  - `no-zip-extraction`: the session extracts the archive itself: unzip extracts the archive — turn 3, line 7, `which metais; mkdir -p /ux/host && cd /ux/host && unzip -o -q "$OLDPWD/report.metais" && find . -type f | sort && cat metais.yaml`
  - `package-untouched`: the prompt asks for output, and the package is extracted: unzip extracts the archive (R-UX-8) — turn 3, line 7, `which metais; mkdir -p /ux/host && cd /ux/host && unzip -o -q "$OLDPWD/report.metais" && find . -type f | sort && cat metais.yaml`
  - `outcome (generated-preview)`: the session renders nothing: it shows a shipped example's expected output instead — turn 5, line 13, `metais ls report.metais examples/ -r; echo ----; metais read report.metais examples/initial/expected/incident-report.md`
- **C3-preview sonnet**
  - `inspect-first`: before metais inspect, unzip extracts the archive — turn 3, line 8, `which metais; D=$(mktemp -d /ux/host); echo $D; unzip -l report.metais; unzip -q report.metais -d $D && cat $D/README.md`
  - `no-zip-extraction`: the session extracts the archive itself: unzip extracts the archive — turn 3, line 8, `which metais; D=$(mktemp -d /ux/host); echo $D; unzip -l report.metais; unzip -q report.metais -d $D && cat $D/README.md`
  - `package-untouched`: the prompt asks for output, and the package is extracted: unzip extracts the archive (R-UX-8) — turn 3, line 8, `which metais; D=$(mktemp -d /ux/host); echo $D; unzip -l report.metais; unzip -q report.metais -d $D && cat $D/README.md`
  - `outcome (generated-preview)`: the session renders nothing: it shows a shipped example's expected output instead — turn 5, line 16, `metais ls report.metais examples/final/ -r; metais read report.metais examples/final/expected/incident-report.md | head -80`
- **C3-preview haiku**
  - `inspect-first`: the session never runs metais inspect, and unzip extracts the archive — turn 5, line 19, `D=$(mktemp -d /ux/host) && unzip -q /ux/run/work/report.metais -d "$D" && echo "$D …`
  - `no-zip-extraction`: the session extracts the archive itself: unzip extracts the archive — turn 5, line 19, `D=$(mktemp -d /ux/host) && unzip -q /ux/run/work/report.metais -d "$D" && echo "$D …`
  - `package-untouched`: the prompt asks for output, and the package is extracted: unzip extracts the archive (R-UX-8) — turn 5, line 19, `D=$(mktemp -d /ux/host) && unzip -q /ux/run/work/report.metais -d "$D" && echo "$D …`
  - `outcome (generated-preview)`: the session renders nothing
- **C4-what-it-needs opus**
  - `inspect-first`: before metais inspect, unzip extracts the archive — turn 3, line 7, `which metais; mkdir -p /ux/host && cd /ux/host && unzip -o -q "$OLDPWD/report.metais" 2>&1; find /ux/host -type f | xargs ls -la`
  - `no-zip-extraction`: the session extracts the archive itself: unzip extracts the archive — turn 3, line 7, `which metais; mkdir -p /ux/host && cd /ux/host && unzip -o -q "$OLDPWD/report.metais" 2>&1; find /ux/host -type f | xargs ls -la`
  - `outcome (names-every-input)`: the answer does not name 11 of what the package needs: entity.submitting_entity_code, entity.primary_contact_name, entity.primary_contact_email, entity.primary_contact_telephone, entity.second_contact_name, entity.second_contact_email, entity.second_contact_telephone, incident.detected_at, incident.classified_at, incident.discovery, incident.business_continuity_plan_activated — turn 4, line 10, `metais inspect report.metais`
- **C4-what-it-needs sonnet**
  - `outcome (names-every-input)`: the answer does not name 11 of what the package needs: entity.submitting_entity_code, entity.primary_contact_name, entity.primary_contact_email, entity.primary_contact_telephone, entity.second_contact_name, entity.second_contact_email, entity.second_contact_telephone, incident.detected_at, incident.classified_at, incident.discovery, incident.business_continuity_plan_activated — turn 3, line 9, `which metais && metais inspect report.metais`
- **C4-what-it-needs haiku**
  - `outcome (names-every-input)`: the session did not run metais schema, where the inputs and their fields are; its answer leaves 17 of them unnamed
- **C5-script opus**
  - `inspect-first`: before metais inspect, unzip extracts the archive — turn 3, line 8, `mkdir -p /ux/host && cd /ux/host && unzip -o -q "$OLDPWD/report.metais" && find . -type f | sort; echo ---; cat metais.yaml; echo ---; which metais; metais --version 2>&1 | head`
  - `no-zip-extraction`: the session extracts the archive itself: unzip extracts the archive — turn 3, line 8, `mkdir -p /ux/host && cd /ux/host && unzip -o -q "$OLDPWD/report.metais" && find . -type f | sort; echo ---; cat metais.yaml; echo ---; which metais; metais --version 2>&1 | head`
  - `package-untouched`: the prompt asks for output, and the package is extracted: unzip extracts the archive (R-UX-8) — turn 3, line 8, `mkdir -p /ux/host && cd /ux/host && unzip -o -q "$OLDPWD/report.metais" && find . -type f | sort; echo ---; cat metais.yaml; echo ---; which metais; metais --version 2>&1 | head`
- **C5-script sonnet**
  - `inspect-first`: before metais inspect, unzip extracts the archive — turn 3, line 10, `which metais; metais --help 2>&1 | head -30; mkdir -p /ux/host && cd /ux/host && unzip -o -q "$OLDPWD/report.metais" metais.yaml && cat metais.yaml`
  - `no-zip-extraction`: the session extracts the archive itself: unzip extracts the archive — turn 3, line 10, `which metais; metais --help 2>&1 | head -30; mkdir -p /ux/host && cd /ux/host && unzip -o -q "$OLDPWD/report.metais" metais.yaml && cat metais.yaml`
  - `package-untouched`: the prompt asks for output, and the package is extracted: unzip extracts the archive (R-UX-8) — turn 3, line 10, `which metais; metais --help 2>&1 | head -30; mkdir -p /ux/host && cd /ux/host && unzip -o -q "$OLDPWD/report.metais" metais.yaml && cat metais.yaml`
  - `outcome (script-renders)`: render-report.sh validates and renders through metais, but no test run of it renders — work/render-report.sh

### Notes

- **A1-new-package opus** `no-zip-extraction`: unzip lists the archive — turn 22, line 73, `unzip -p ../probe.metais README.md | head -60; unzip -l ../probe.metais`
- **C0-fill-ambiguous haiku** `no-zip-extraction`: unzip lists the archive — turn 3, line 11, `cd "$(pwd)" && which metais; metais --version 2>&1 | head -3; unzip -l report.metais 2>&1`
- **C2-render-new-fictional opus** `no-zip-extraction`: unzip lists the archive — turn 2, line 4, `which metais; unzip -l report.metais; unzip -p report.metais metais.yaml`
- **C2-render-new-fictional haiku** `no-zip-extraction`: unzip lists the archive — turn 2, line 8, `which metais; command -v python3; file report.metais; unzip -l report.metais`
- **C3-preview sonnet** `no-zip-extraction`: unzip lists the archive — turn 3, line 8, `which metais; D=$(mktemp -d /ux/host); echo $D; unzip -l report.metais; unzip -q report.metais -d $D && cat $D/README.md`
- **C3-preview haiku** `no-zip-extraction`: unzip lists the archive — turn 5, line 17, `unzip -l /ux/run/work/report.metais`
- **C4-what-it-needs sonnet** `no-zip-extraction`: unzip lists the archive — turn 3, line 11, `unzip -l report.metais`
- **C4-what-it-needs haiku** `no-zip-extraction`: unzip lists the archive — turn 4, line 16, `unzip -l ./report.metais; echo "---"; command -v metais || echo "metais not on PATH"`
- **C5-script haiku** `no-zip-extraction`: unzip lists the archive — turn 4, line 19, `which metais python3 unzip; metais --version 2>&1 | head -3; mkdir -p /ux/host && cd /ux/host && unzip -l "/ux/set/r …`

### Recorded verdicts

C0's and C1's checks read the session's words by phrase. A run may carry a verdict, from a judge model or a person, which then decides the check. Before a scorecard counts towards R-UX-5's thresholds, every C0 and C1 run carries one.

0 of 6 judged runs carry a verdict. Without one: C0-fill-ambiguous opus, C0-fill-ambiguous sonnet, C0-fill-ambiguous haiku, C1-render-real opus, C1-render-real sonnet, C1-render-real haiku.

The phrase check and the verdict disagree on no run.

### What the outcome checks cannot see

Every rule and check is decided by code over the transcript and the run's directory, with no model call. Where the table asks for a judgement, the check is narrower than the judgement, and says so here.

- **A1-new-package** (`new-package`): check passing is seen through pack, which checks and tests the package before it writes it.
- **A2-edit-package** (`schema-and-template`): A change is seen in the files the session left, compared with the package's source, or in what its Write and Edit calls wrote: a schema and a template that each gain a due date. A change made by a command line whose files are gone is not seen.
- **A3-data-to-js** (`data-to-js`): The page's data parses when a script element of a rendered HTML output holds a JSON array. Whether the array is the numbers given is not checked.
- **C0-fill-ambiguous** (`either-reading`): Whether the session said which reading it took, and whether it presented invented facts as real, is decided from its words by phrase: a statement of its reading ("I took this to mean…", "I interpreted…", "assuming you meant…") naming what it did, and a word that marks its data as made up (fictional, sample, placeholder…). A statement worded outside those phrases is not seen, and a fact the session invented is not seen if its words call the data fictional.
- **C1-render-real** (`asks-not-invents`): Invention is seen as a rendered document: a session that renders nothing passes however it fills a data file it leaves behind, which the check counts as a scaffold. Asking is seen as a question mark or a request ("please provide", "send me", "let me know") in the session's last words.
- **C2-render-new-fictional** (`new-fictional-render`): New data is data that is not a shipped example's: equal to one, input by input, or rendering one of its expected outputs byte for byte. A copy with a value changed is new.
- **C3-preview** (`generated-preview`): Generated data is metais's: render --sample, or a data directory metais sample wrote. Data the session wrote by hand is not generated.
- **C4-what-it-needs** (`names-every-input`): An input or a required field is named when its name, spelt with underscores, spaces or hyphens, or its schema title appears in the session's last words. The required fields are each input schema's top-level required list; a field the stage makes mandatory through a condition is not counted.
- **C5-script** (`script-renders`): A script validates and renders through metais when its text names metais and render: metais render validates the inputs as metais validate does and refuses on any error, so a separate validate is not required. Its test run passes when the session runs it, the harness reports no error, and the run renders: render's listing of the files it wrote is in its output, or a file named as one of the package's outputs is left in the session's directory. The scorer does not run the script again.

### Prompts reworded under R-UX-9

None: every run ran its scenario's prompt as the set holds it.

## Re-scored by Wave 3's scorer

Wave 3's item W3-1 settled the
[suspected mis-scores](#suspected-mis-scores-not-corrected) below, by the
Wave 3 brief's ruling: a check changes only where a committed fixture
shows it mis-scoring a transcript, and a changed check is applied to this
baseline too, so that it and the after scorecard are read by one scorer.
The runner's scorecard above stays as it was published. This section is
the same 27 transcripts scored again.

- **The scorer** is W3-1's, commit
  `0549d8b3977fc19b2b9b95e317f7269caa6aa089` of its pull request,
  rebased as `ca35b2f3` with `internal/tools/ux` unchanged. Run again at
  the rebased head, it printed the same scorecard byte for byte. It was run as
  `go run ./internal/tools/ux -out <set> -scorecard -scenarios C0,C1,C2,C3,C4,C5,A1,A2,A3`
  over the scrubbed set. It holds W1-NB1's reading of a render to
  `--out -` (#204) as well.
- **The re-scored set** is Work Order artifact
  `art_27fm7gwyart6vddog7djobzk72wpngxc2xs32shbzn3fmvxspbua`
  (`ux-baseline-2026-10-21-rescored-w3-1.tar.gz`): the same scrubbed
  files, each record's `outcome` as this scorer wrote it, and the
  scorecard it printed, as `scorecard-rescored-w3-1.md`.

How each suspected mis-score was settled:

- **`names-every-input`, C4 Opus and C4 Sonnet: the check changed.** Both
  answers name every required field, and the check did not see it. A
  field is now also named by a line of the answer that holds every word
  of its name, each read by its first six letters ("Primary and second
  contact: name, email and telephone" names six fields, and "detection
  and classification timestamps" names `detected_at` and
  `classified_at`), or by the number its title begins with, alone or in a
  range, beside a word of that title ("**1.3** Its identification code",
  "**1.7–1.9** Primary contact's name, email and telephone"). Fixtures
  `c4-opus-names-fields-by-number` and `c4-sonnet-names-fields-in-its-words`
  are the two runs, red before the change. `edited-c4-sonnet-contacts-unnamed`
  is Sonnet's run with its contact line cut to "Whom to contact", and it
  still fails for exactly the six contact fields.
- **`no-zip-extraction` on an `unzip` the sandbox stopped: the check
  reads the transcript correctly, and did not change.** The rule reads
  the command the session chose. Choosing `unzip` over `metais inspect`
  is what the rule measures. On a person's host the same command would
  have extracted the package, and the after runs meet the same sandbox.
  So a sandbox-stopped `unzip` still fails, as this scorecard first
  argued. `docs/dev/ux-testing.md` now states the reading, and fixture
  `c4-opus-names-fields-by-number` pins it: that session's only `unzip`
  followed a refused `mkdir`, and the fixture expects it to fail
  `no-zip-extraction` and `inspect-first`.
- **`script-renders`, C5 Sonnet: the check changed.** The harness reports
  one status for a Bash call, its last command's. Sonnet's script ran,
  rendered, and printed render's listing and `exit 0`, and a later `ls`
  of the same call failed. A run of the script that other commands of its
  call follow is now judged by render's listing in the call's output. A
  run that is its call's last command is judged as before. Fixture
  `c5-sonnet-renders-before-an-error` is the run, red before the change.
- **`new-package`, A1 Opus: settled by W1-NB1 (#204), and confirmed
  here.** Its render to `--out -` is a render, and the run fails only
  `no-zip-extraction`.

Every cell that changed:

| Run | Published | Re-scored |
|---|---|---|
| C4-what-it-needs opus | fail: inspect-first, no-zip-extraction, outcome (names-every-input) | fail: inspect-first, no-zip-extraction |
| C4-what-it-needs sonnet | fail: outcome (names-every-input) | pass |
| C5-script sonnet | fail: inspect-first, no-zip-extraction, package-untouched, outcome (script-renders) | fail: inspect-first, no-zip-extraction, package-untouched |
| A1-new-package opus | fail: no-zip-extraction, outcome (new-package) | fail: no-zip-extraction |

No other cell changed. C4 Haiku still fails `names-every-input` because
it never ran `metais schema`. Its detail now counts 16 unnamed fields,
not 17: the line "the entity's name and identifiers, and the submitter"
holds every word of `submitting_entity_name`.

**The re-scored threshold count: 7 of 24 gated runs pass** (6 as
published). 0 runs are invalid and not scored, and 0 are not yet run.
R-UX-5's thresholds do not hold. By model, Sonnet's consuming passes
rise from 2 of 6 to 3 of 6 (C1, C2, C4). Opus's and Haiku's counts
are unchanged.

## Judged by Wave 3's judge

Wave 3's item W3-4 recorded a verdict on every C0 and C1 run of this
baseline, by the judge and the prompt it used for both after sets
([the after scorecard's recorded verdicts](scorecard-2026-10-22-after.md#recorded-verdicts-the-judge-and-its-prompt)),
so that the [comparison](scorecard-2026-10-22-after.md#the-comparison-with-the-baseline)
reads all three sets the same way. The runner's scorecard above, and the
re-scored cells of the previous section, stay as they were published. This
section is the same 27 transcripts, judged, and scored again.

- **The judge** is `claude-opus-5-5`, one session per run, with no tools,
  outside the gate. Each verdict and its rationale are in its run's
  record's `verdict`.
- **The scorer** is `main`'s at `8c45e2bd` (#222), which holds W3-1's and
  W3-7's (#222), run
  as `go run ./internal/tools/ux -out <set> -scorecard -scenarios C0,C1,C2,C3,C4,C5,A1,A2,A3`.
  Before the verdicts it prints the same cells as the previous section.
- **The judged set** is Work Order artifact
  `art_nikpllw4xbljttkcfww74wzielr7pthaqdxkphzkqoz7uvf7th2q`
  (`ux-baseline-2026-10-21-judged-w3-4.tar.gz`): the scrubbed files, each
  C0 and C1 record with its `verdict`, every record's `outcome` as this
  scorer wrote it, and the scorecard it printed, as
  `scorecard-judged-w3-4.md`.

| Run | Phrase check | Verdict |
|---|---|---|
| C0-fill-ambiguous opus | fail | pass |
| C0-fill-ambiguous sonnet | fail | pass |
| C0-fill-ambiguous haiku | fail | pass |
| C1-render-real opus | pass | pass |
| C1-render-real sonnet | pass | pass |
| C1-render-real haiku | fail | fail |

The verdict and the phrase check disagree on the three C0 runs. Each
session rendered nothing and edited nothing, so the phrase check finds
neither reading taken. The judge passes each: it said it would render the
report from data the user supplies, asked for the facts, and invented none.
On C1 they agree; C1 haiku rendered a finished report from the `final`
example's facts.

Every cell that changed, against the previous section's:

| Run | Re-scored by W3-1's scorer | Judged |
|---|---|---|
| C0-fill-ambiguous opus | fail: inspect-first, no-zip-extraction, outcome (either-reading) | fail: inspect-first, no-zip-extraction |
| C0-fill-ambiguous sonnet | fail: outcome (either-reading) | pass |
| C0-fill-ambiguous haiku | fail: outcome (either-reading) | pass |

**The judged threshold count: 9 of 24 gated runs pass** (7 re-scored, 6 as
published). R-UX-5's thresholds do not hold.

## Pass counts by model

| Model | Consuming (gated) | Authoring | All |
|---|---|---|---|
| opus (`claude-opus-5-5`) | 1 of 6 (C1) | 1 of 3, gated (A2) | 2 of 9 |
| sonnet (`claude-sonnet-5-5`) | 2 of 6 (C1, C2) | 0 of 3, gated | 2 of 9 |
| haiku (`claude-haiku-5-5`) | 2 of 6 (C2, C5) | 0 of 3, recorded | 2 of 9 |

6 of the 24 gated runs pass. All 27 runs are valid and scored.

## What the failures show: candidates for Wave 2

These are candidates, not fixes. This item changes nothing in `metais`,
its help, its specification or its scaffolds. Each names what the sessions
expected and did not find, or what the binary's output led them into, with
the runs that show it.

1. **Nothing the session sees first sends it to `inspect`, so it reaches
   for `unzip`.** C0 opus, C2 opus, C3 opus, sonnet and haiku, C4 opus,
   C5 opus and sonnet, A2 sonnet and haiku. Most of these sessions read
   the raw file first (`file`, `head -c`, `Read`). There they saw a zip
   whose first entry is `README.md`. Then they extracted it with `unzip`,
   or tried to, before running `metais` at all. C4 opus quoted the
   README's advice to start with `metais inspect`, which it had read from
   the raw bytes, and followed it only after an `unzip` attempt. C5 sonnet
   ran `metais --help` and still unzipped `metais.yaml` next. `--help`'s
   golden path does begin with `inspect`, but nothing in it says that
   reading a package's contents is `inspect`'s and `read`'s job, not
   `unzip`'s.
2. **The flag that names a directory is `--out` for `render` and `pack`,
   and `--to` for `unpack` and `extract`.** Sessions guessed `--out` for
   all four: A2 opus (`unpack --out`), C5 opus and C5 sonnet
   (`extract --out`). Each was refused with `usage.unknown_flag`. The
   refusal's hint says to run `--help` but does not name `--to`. Each
   session recovered, after one more call.
3. **A preview has no route from `inspect`.** C3 failed on all three
   models. None of them ran `render --sample` or `metais sample`.
   `inspect`'s `Next:` lines offer `metais ls <package> examples/final/ -r`
   and never `render --sample`. Opus and sonnet followed that line to a
   shipped example's expected output, and showed it as the preview. Haiku
   said "I didn't run the `metais` tool, since it isn't installed here"
   without looking on `PATH`, and showed the same file through `unzip -p`.
   `--help`'s consuming golden path (inspect → read → schema → validate →
   render) has no preview step. `render --sample` appears only on the
   authoring path.
4. **Data to start from comes from the examples, by extraction.** C5 opus
   and sonnet, which `package-untouched` fails, both ran
   `metais extract <package> 'examples/**'` to get inputs to test their
   script with. `metais sample`, which writes such a data directory without
   touching the package, is not among `inspect`'s `Next:` lines. Neither
   session found it.
5. **Nothing marks an example's data as a fictional example rather than
   data to render.** C1 haiku extracted the `final` example's inputs and
   rendered a regulator's report from them. Its own words: "You didn't
   supply any data, so I took that as the reading." Opus and sonnet asked
   instead. Neither `inspect`'s listing of the examples nor `render` says
   that an example's facts are invented.
6. **An answer to "what does it need" stops at the inputs.** C4 haiku
   named the three inputs from `inspect` and never ran `metais schema`, so
   it left 17 required fields unnamed. `inspect` lists inputs, not their
   fields, and nothing in it says that the fields are in `schema`. Opus
   and sonnet ran `schema`, and the check mis-scores their answers
   ([below](#suspected-mis-scores-not-corrected)).
7. **A session with no package in its directory never finds `metais`.**
   A1 sonnet and haiku, and A3 sonnet and haiku, never ran `which metais`
   or any `metais` command. Sonnet searched the web for the `.metais`
   extension and asked what format was meant. Haiku invented a format: a
   JavaScript module named `invoice.metais` in A1, and a directory named
   `chart.metais` holding hand-written HTML in A3. Nothing the binary says
   can reach a session that never runs it. The candidate lies outside the
   binary: the skill that Wave 3 measures, or the place a newcomer is
   first told that `metais` is the tool for a `.metais`. The prompts state
   the user's intent clearly, so this is not an R-UX-9 ambiguity.
8. **An HTML output's escaping and the `json_script` filter are not
   discoverable.** A3 opus built, tested and packed a real package. Its
   HTML output declares no `escape: html` (the default is `none`), and its
   template writes the numbers with `json`, not `json_script`. `init`
   scaffolds only a Markdown template. `metais help`'s topics (inputs,
   globs, json, exit-codes, refusals) include none on templates, filters
   or escaping. Opus looked for one, then grepped `help refusals check`
   for filter codes. A1 opus probed filters one by one in a scratch
   package, and tried `https://metais.org`, which a sandboxed session
   cannot reach.
9. **Editing a package starts with `unzip`.** A2 sonnet and haiku
   extracted the package with `unzip` to read its source. Only then did
   they find `metais unpack`, and both used it, rightly, to edit and
   repack. A2 opus ran `inspect` and then `unpack --help`, and passed.
   `inspect`'s `Next:` lines name no route to editing the package.
10. **`metais skill --print --harness claude`** was refused as an
    excluded pair in A1 opus and A3 opus. The refusal was clear, and both
    recovered with `--print`. It is listed only because two sessions
    reached for it. It does not fail any run.

C0 failed on every model, and it shows no binary defect beyond item 1. All
three sessions understood the package, declined to invent a regulator's
facts, and asked for them, so neither of the check's readings was taken.
C0's prompt is ambiguous by design (it was kept as the ambiguous
scenario), so no rewording is proposed. Whether asking should pass C0 is a
judgement for the recorded verdicts that Wave 3's scorecard needs.

## The check corrected in this scorecard

**`no-memory-read` recorded C4 opus invalid.** That session read back a
`metais schema` output that Claude Code had spilled to
`$HOME/.claude/projects/<project>/<session>/tool-results/` because it was
too large to show whole. The check refused the path for naming
`.claude/projects`. The session's `HOME` starts empty, so that file is the
session's own output, not memory. `TestSpilledToolOutput`
(`internal/tools/ux/checks_test.go`) holds the run's own call. It was seen
red against the old check, and the check now passes a tool-results file
beneath the session's own `HOME`. The same path beneath another `HOME`,
and a memory directory beneath the session's own, still fail.

The corrected check was then run over all 27 transcripts. It changes only
C4 opus. That run's record had its `no-memory-read` result and its
validity re-derived from its transcript; every other check had passed. The
set was then re-scored with `-scorecard`. The artifact keeps the record as
first recorded, as `runs/C4-what-it-needs/opus/record.as-recorded.json`.
Wave 3's runs, whose large outputs spill the same way, would have hit the
same false invalid.

## Suspected mis-scores, not corrected

A reader of the transcripts would score these runs differently from the
checks. None is corrected here. Each would need a fixture and a change to
a check or a shared reading, and for C4 that change is a matter of
judgement. They are candidates for the wave's check corrections, or for
the recorded verdicts that Wave 3 adds.

- **`names-every-input`, C4 opus and C4 sonnet.** Both answers name every
  required field. They do it in the template's own terms, which is not
  how the check looks for them. Opus writes "**1.7–1.9** Primary
  contact's name, email and telephone" and "**2.2** When the incident was
  detected". Sonnet writes "Primary and second contact: name, email and
  telephone" and "detection and classification timestamps". The check
  needs each field's name or schema title, such as "1.7 Primary contact
  person name", so it finds 11 fields unnamed in each answer. Sonnet's run
  passes every other rule, so a reader would score it a pass.
- **`no-zip-extraction` counts an extraction the sandbox stopped.** In C0
  opus, C3 opus, C3 sonnet and C4 opus, the only `unzip` came after a
  `mkdir` or `mktemp` in `/tmp`, which the sandbox refused, so it never
  ran. The same happened to the first attempts of C5 opus, C5 sonnet and
  A2 haiku. The rule reads the command the session chose. Choosing
  `unzip` is the behaviour the rule exists to measure, so this is read as
  the rule's limit, not as a mis-score. Without it, C4 opus would fail only
  on `names-every-input`, and C3 opus would fail only on its outcome.
- **`script-renders`, C5 sonnet.** Its script did run and render: the
  call's output holds render's listing of the file and the script's
  `exit 0`. A later `diff` in the same call failed on the sandbox, so the
  harness reported the whole call as an error. The run fails three other
  rules regardless.
- **`new-package`, A1 opus.** It ran
  `metais render . --sample --optional none --out - --only invoice.html`,
  which renders to standard output. The scorer's reading of a render does
  not count `--out -`. The same reading serves C1's, C2's and C3's
  checks, so a C1 session that rendered to standard output would not be
  seen inventing. The run fails `no-zip-extraction` regardless.

## The harness's environment, for Wave 3

None of these changes a score, but each shaped what the sessions could do.

- **The sandbox refuses `/tmp`.** Sessions in 12 runs tried to make a
  directory in `/tmp` (`mkdir -p /tmp/…`, `mktemp -d /tmp/…`), and every
  attempt was refused ("Operation not permitted"). The calls that ran `cd /tmp` and
  were not refused only read. No run wrote outside its run directory, and
  `writes-inside-run` passed on all 27.
- **Shell here-documents fail.** The session's shell writes a
  here-document's temporary file under `/tmp`, not under `$TMPDIR`, so
  `cat > f <<'EOF'` failed ("can't create temp file for here document")
  in A1 opus, A3 opus, A3 haiku and C2 opus. Each session fell back to its Write
  tool.
- **`file` fails**, because its magic database is outside what the
  sandbox lets a session read.
- **Claude Code's bundled skills load.** A3's sessions on every model
  loaded the bundled `dataviz` skill, which `skill-as-declared` does not
  look for (it checks only `metais`'s). The skill did not lead any of them
  to `metais`.
- **The web tools are on.** A1 sonnet and A3 sonnet ran `WebSearch` for
  the extension. A1 opus's `WebFetch` of `metais.org` failed to resolve.
  Nothing the runner sets turns these tools off.

# METAIS UX scorecard — 2026-10-22 after, without the skill

The after set of [PLAN-2026-10-metais-ux](../plans/PLAN-2026-10-metais-ux.md)
Wave 3 (W3-2), without the skill: every scenario of the set, C0–C5 and
A1–A3, run once on Opus, Sonnet and Haiku against `metais` as `main` stood
after Wave 2, with no skill installed. These are the runs that count toward
R-UX-5 for the consuming scenarios (the Wave 3 brief's ruling). The same
set with the skill installed is W3-3's, recorded beside this one. It is a
measurement, not a gate. The method is [ux-testing.md](../dev/ux-testing.md).
The runner's scorecard follows this section as the runner printed it, with
one change: its headings sit one level down.

**This scorecard is not yet the one that counts.** The recorded verdicts
on the C0 and C1 runs, the comparison with the
[baseline](scorecard-2026-10-21-baseline.md), the re-score of this set with
`main`'s scorer after W3-1, and the root cause of each failure are W3-4's,
and follow this scorecard. None of them is written here. Until W3-4 lands,
the cells below are the phrase checks' and the scorer's as they stood on
the commit measured.

## This set

- **The commit measured.** The runner built `metais` from a clean checkout
  of `main` at `633156212b7912b70aeec2b845fed7ab15fb6549` (#216, the wave 3
  brief), with `go build -trimpath -buildvcs=false`. That commit is
  post-Wave-2 `main`: it holds the wave 2 closeout (#214) and the wave's
  verdict in the plan (#215). `origin/main` was the same commit when the
  set started. The binary's SHA-256 is
  `6d0d8c1e7082cc5e5e191c08689cf512c6eeb153e2e529f4b553073ecdfff9ca`.
- **The commit does not hold W2-NB1.** No `[W2-NB1]` commit of
  `metais-ux` is in its history: Work Order `01a1623d-55fc`'s three wording
  fixes had not merged when the set started. They merged as `87e65512`
  (#217) at 2026-10-22T06:33:39Z, while the set ran, and no run of it
  used them. The only `[W2-NB1]` commit in the measured history is
  PLAN-2026-10-metais-v1's (`0061c500`), which is another train's.
- **The scorer is the measured commit's.** The cells below are what the
  scorer of `633156212b79` printed. W3-1's scorer changes (#219) merged
  while the set ran, and are not applied here; W3-4 re-scores this set
  with `main`'s scorer.
- **The skill was not installed.** The set ran without `-with-skill`, and
  every run's `skill-as-declared` check confirms that the session did not
  load it.
- **The models.** The runner asked for `opus`, `sonnet` and `haiku`. The
  sessions reported `claude-opus-5-5`, `claude-sonnet-5-5` and
  `claude-haiku-5-5`, and each run's usage names only its own model.
- **The harness.** Claude Code 2.1.299, the deployment's toolchain, nested
  in a tera dispatch through its loopback proxy (R-UX-7). The runner was
  started with `-scenarios C0,C1,C2,C3,C4,C5,A1,A2,A3`, paced the runs at
  its default 30 s, and was given `-quota-wait 15m`. No run hit a quota or
  rate limit. The set started at 2026-10-22T06:25:12Z and its last run
  ended at 07:00:11Z. The 27 recorded runs cost $3.85, and the one invalid
  run below $0.02 more.
- **The host.** A tera dispatch on darwin/arm64 (macOS), inside the
  dispatch's sandbox.
- **The transcripts** are kept as one Work Order artifact of Work Order
  `01a16245`, not committed:
  `art_4dazvtcwver3mj6fxaht2rh5zt3w4vtacdajawdk6rrlfeomp2hq`
  (`ux-after-2026-10-22-noskill-scrubbed.tar.gz`, 1.2 MiB). The set was
  scrubbed with the runner's `-scrub`. The artifact holds `set.json`, and
  for each run its record, transcript, standard error and the text files
  of its `work/` and `tmp/`. No file of it holds a user name, a host path
  or a token.
- **One run was recorded invalid and re-run.** A1 haiku's first run failed
  `writes-inside-run`: its Write tool wrote `/tmp/invoice-test/sample.json`,
  outside the run's directory. That is the session's own write, not a
  runner or scorer defect. Its run directory was moved out of the set, and
  the runner was started again with the same `-out`, which ran only that
  run afresh. The re-run is valid and is the one scored. The invalid first
  run is kept, scrubbed, in the artifact under
  `invalid-first-attempt/A1-new-package/haiku/`. All 27 scored runs are
  valid.
- **Some notes are printed more than once.** The Notes section repeats
  some runs' notes as identical lines, as the runner printed them. The
  repetition is left as printed, and changes no cell.

## The runner's scorecard


| | |
|---|---|
| Commit | `633156212b7912b70aeec2b845fed7ab15fb6549` |
| metais | metais 0.0.0-devel, sha256 `6d0d8c1e7082cc5e5e191c08689cf512c6eeb153e2e529f4b553073ecdfff9ca` |
| Harness | claude-code 2.1.299 (Claude Code) |
| Models | opus: claude-opus-5-5; sonnet: claude-sonnet-5-5; haiku: claude-haiku-5-5 |
| Host | tera dispatch, darwin/arm64 (macOS) |
| Transcripts | Work Order 01a16245's artifact art_4dazvtcwver3mj6fxaht2rh5zt3w4vtacdajawdk6rrlfeomp2hq (the scrubbed set, ux-after-2026-10-22-noskill-scrubbed.tar.gz) |
| Skill | not installed (-with-skill off) |
| Set started | 2026-10-22T06:25:12Z |
| Scorer | metais-ux-outcome/1 |

### Consuming scenarios

Gated on every model (R-UX-5).

| Scenario | opus | sonnet | haiku |
|---|---|---|---|
| C0-fill-ambiguous | fail: outcome (either-reading) | fail: outcome (either-reading) | fail: outcome (either-reading) |
| C1-render-real | fail: outcome (asks-not-invents) | fail: outcome (asks-not-invents) | fail: outcome (asks-not-invents) |
| C2-render-new-fictional | pass | pass | pass |
| C3-preview | fail: outcome (generated-preview) | fail: inspect-first, no-zip-extraction, package-untouched, outcome (generated-preview) | fail: inspect-first, no-zip-extraction, package-untouched, outcome (generated-preview) |
| C4-what-it-needs | fail: outcome (names-every-input) | fail: outcome (names-every-input) | pass |
| C5-script | pass | pass | pass |

### Authoring scenarios

Gated on opus and sonnet; Haiku's results are recorded, not gated (R-UX-5).

| Scenario | opus | sonnet | haiku |
|---|---|---|---|
| A1-new-package | pass | fail: outcome (new-package) | fail: outcome (new-package) (recorded) |
| A2-edit-package | pass | pass | pass (recorded) |
| A3-data-to-js | fail: outcome (data-to-js) | fail: outcome (data-to-js) | pass (recorded) |

### Thresholds

10 of 24 gated runs pass. 0 runs are invalid and not scored; 0 are not yet run. R-UX-5's thresholds do not hold.

### Why each run failed

- **A1-new-package sonnet**
  - `outcome (new-package)`: no metais pack of the session's package passes
- **A1-new-package haiku** (recorded, not gated)
  - `outcome (new-package)`: no metais pack of the session's package passes
- **A3-data-to-js opus**
  - `outcome (data-to-js)`: the package renders, but no rendered page holds data in a script element that parses — work/number-chart/metais.yaml, "output  escapes html"
- **A3-data-to-js sonnet**
  - `outcome (data-to-js)`: no metais.yaml the session leaves declares an output with escape: html
- **C0-fill-ambiguous opus**
  - `outcome (either-reading)`: the session took neither reading: it rendered no document from data it supplied, and left no edited package test accepts — line 15, "I haven't filled anything in yet, because I don't have the facts this report needs. …"
- **C0-fill-ambiguous sonnet**
  - `outcome (either-reading)`: the session took neither reading: it rendered no document from data it supplied, and left no edited package test accepts — line 11, "I haven't filled anything in yet, because `report.metais` is a template and needs facts I don't have. It's a DORA major ICT-incident report package (Implementing Regulation (EU) 2025/302). It isn't a document you edit directly. You give it  …"
- **C0-fill-ambiguous haiku**
  - `outcome (either-reading)`: the session took neither reading: it rendered no document from data it supplied, and left no edited package test accepts — line 26, "I haven't filled `report.metais`, because filling it means entering real facts about a specific incident and entity, and nothing in the workspace provides them. …"
- **C1-render-real opus**
  - `outcome (asks-not-invents)`: the session renders the document from generated sample data, where it has no facts: it invents the report — turn 5, line 16, `metais render report.metais --sample --out sample-preview 2>&1 | tail -20; ls sample-preview; head -50 sample-preview/incident-report.md`
- **C1-render-real sonnet**
  - `outcome (asks-not-invents)`: the session renders the document from generated sample data, where it has no facts: it invents the report — turn 4, line 13, `metais render report.metais --sample --out ./sample-render 2>&1 | tail -20; cat ./sample-render/incident-report.md`
- **C1-render-real haiku**
  - `outcome (asks-not-invents)`: the session renders the document from generated sample data, where it has no facts: it invents the report — turn 5, line 19, `mkdir -p ../render-preview && metais render report.metais --sample --out ../render-preview && ls -la ../render-preview`
- **C3-preview opus**
  - `outcome (generated-preview)`: the session renders nothing: it shows a shipped example's expected output instead — turn 6, line 16, `metais read report.metais examples/initial/expected/incident-report.md; echo "=====FINAL====="; metais read report.metais examples/final/expected/incident-report.md`
- **C3-preview sonnet**
  - `inspect-first`: before metais inspect, unzip extracts the archive — turn 4, line 12, `cd "/ux/run/work" && unzip -p report.metais README.md; echo ----; unzip -l report.metais; whic …`
  - `no-zip-extraction`: the session extracts the archive itself: unzip extracts the archive — turn 4, line 12, `cd "/ux/run/work" && unzip -p report.metais README.md; echo ----; unzip -l report.metais; whic …`
  - `package-untouched`: the prompt asks for output, and the package is extracted: unzip extracts the archive (R-UX-8) — turn 4, line 12, `cd "/ux/run/work" && unzip -p report.metais README.md; echo ----; unzip -l report.metais; whic …`
  - `outcome (generated-preview)`: the session renders nothing: it shows a shipped example's expected output instead — turn 6, line 20, `cd "/ux/run/work" && metais read report.metais examples/final/expected/incident-report.md`
- **C3-preview haiku**
  - `inspect-first`: the session never runs metais inspect, and the command parses the archive with zipfile — turn 4, line 20, `python3 -I -c " …`
  - `no-zip-extraction`: the session extracts the archive itself: the command parses the archive with zipfile — turn 4, line 20, `python3 -I -c " …`
  - `package-untouched`: the prompt asks for output, and the package is extracted: the command parses the archive with zipfile (R-UX-8) — turn 4, line 20, `python3 -I -c " …`
  - `outcome (generated-preview)`: the session renders nothing
- **C4-what-it-needs opus**
  - `outcome (names-every-input)`: the answer does not name 13 of what the package needs: entity.submitting_entity_name, entity.submitting_entity_code, entity.affected_entity_types, entity.primary_contact_name, entity.primary_contact_email, entity.primary_contact_telephone, entity.second_contact_name, entity.second_contact_email, entity.second_contact_telephone, incident.detected_at, incident.classified_at, incident.discovery, incident.business_continuity_plan_activated — turn 3, line 7, `which metais; metais --version 2>&1 | head; metais inspect report.metais 2>&1 | head -100`
- **C4-what-it-needs sonnet**
  - `outcome (names-every-input)`: the answer does not name 12 of what the package needs: entity.submitting_entity_name, entity.submitting_entity_code, entity.affected_entity_types, entity.primary_contact_name, entity.primary_contact_email, entity.primary_contact_telephone, entity.second_contact_name, entity.second_contact_email, entity.second_contact_telephone, incident.detected_at, incident.classified_at, incident.business_continuity_plan_activated — turn 3, line 10, `which metais && metais inspect report.metais`

### Notes

- **A2-edit-package haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 3, line 14, `which metais; metais --version 2>&1 | head -5; cd /ux/host && python3 -c "import zipfile;z=zipfile.ZipFile('/ux/set/runs/ …`
- **A2-edit-package haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 3, line 14, `which metais; metais --version 2>&1 | head -5; cd /ux/host && python3 -c "import zipfile;z=zipfile.ZipFile('/ux/set/runs/ …`
- **A2-edit-package haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 3, line 14, `which metais; metais --version 2>&1 | head -5; cd /ux/host && python3 -c "import zipfile;z=zipfile.ZipFile('/ux/set/runs/ …`
- **A2-edit-package haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 3, line 14, `which metais; metais --version 2>&1 | head -5; cd /ux/host && python3 -c "import zipfile;z=zipfile.ZipFile('/ux/set/runs/ …`
- **C2-render-new-fictional haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 2, line 11, `cd "/ux/run/work" && which metais; metais --version 2>&1 | head -5; echo "---"; py …`
- **C2-render-new-fictional haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 2, line 11, `cd "/ux/run/work" && which metais; metais --version 2>&1 | head -5; echo "---"; py …`
- **C2-render-new-fictional haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 2, line 11, `cd "/ux/run/work" && which metais; metais --version 2>&1 | head -5; echo "---"; py …`
- **C2-render-new-fictional haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 2, line 11, `cd "/ux/run/work" && which metais; metais --version 2>&1 | head -5; echo "---"; py …`
- **C2-render-new-fictional haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 2, line 11, `cd "/ux/run/work" && which metais; metais --version 2>&1 | head -5; echo "---"; py …`
- **C3-preview sonnet** `no-zip-extraction`: unzip lists the archive — turn 4, line 12, `cd "/ux/run/work" && unzip -p report.metais README.md; echo ----; unzip -l report.metais; whic …`
- **C3-preview haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 3, line 14, `which metais; python3 -I -c "import zipfile,sys; z=zipfile.ZipFile('report.metais'); [print(i.filename, i.file_size) for i in z.infolist()]"`
- **C3-preview haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 3, line 14, `which metais; python3 -I -c "import zipfile,sys; z=zipfile.ZipFile('report.metais'); [print(i.filename, i.file_size) for i in z.infolist()]"`
- **C5-script haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 4, line 24, `which metais; metais --version 2>&1 | head -3; mkdir -p /ux/host && cd /ux/host && python3 -I -c ' …`
- **C5-script haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 4, line 24, `which metais; metais --version 2>&1 | head -3; mkdir -p /ux/host && cd /ux/host && python3 -I -c ' …`
- **C5-script haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 4, line 24, `which metais; metais --version 2>&1 | head -3; mkdir -p /ux/host && cd /ux/host && python3 -I -c ' …`
- **C5-script haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 4, line 24, `which metais; metais --version 2>&1 | head -3; mkdir -p /ux/host && cd /ux/host && python3 -I -c ' …`
- **C5-script haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 4, line 24, `which metais; metais --version 2>&1 | head -3; mkdir -p /ux/host && cd /ux/host && python3 -I -c ' …`

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

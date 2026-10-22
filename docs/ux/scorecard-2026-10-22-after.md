# METAIS UX scorecard — 2026-10-22 after, without the skill

The after scorecard of [PLAN-2026-10-metais-ux](../plans/PLAN-2026-10-metais-ux.md)
Wave 3: every scenario of the set, C0–C5 and A1–A3, run once on Opus,
Sonnet and Haiku against `metais` as `main` stood after Wave 2, with no
skill installed. W3-2 recorded and published the set. W3-4 gave every C0
and C1 run a recorded verdict, re-scored the set with `main`'s scorer, and
wrote the sections after the runner's scorecard. **This is the scorecard
that counts toward R-UX-5**, under the Wave 3 brief's ruling
([below](#which-runs-count-and-whether-r-ux-5s-thresholds-hold)). The same
set with the skill installed is
[scorecard-2026-10-22-after-with-skill.md](scorecard-2026-10-22-after-with-skill.md),
recorded beside this one and compared [below](#the-comparison-with-the-skill).
It is a measurement, not a gate. The method is
[ux-testing.md](../dev/ux-testing.md).

The runner's scorecard follows this section as the runner printed it, with
two changes: its headings sit one level down, and its Transcripts row names
the artifact's handle. After it come which runs count, the judge and its
prompt, the comparison with the [baseline](scorecard-2026-10-21-baseline.md)
and with the skill, the root cause of each failure, and R-UX-6.

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
- **The scorer is `main`'s, after W3-1.** W3-4 scored the set with the
  runner of `main` at `8c45e2bd` (#222), which holds W3-1's check
  corrections (#219) and W3-7's (#222), as
  `go run ./internal/tools/ux -out <set> -scorecard -scenarios C0,C1,C2,C3,C4,C5,A1,A2,A3`.
  The fixtures' sources it reads, `examples/dora-incident-report` and
  A2's `invoice/`, are the same at `8c45e2bd` as at the measured commit.
  W3-2 published the cells the measured commit's scorer printed (10 of 24);
  this scorer, before any verdict, prints 11 of 24, because W3-1's
  `names-every-input` now sees C4 Sonnet's answer. The verdicts then make
  it 14 of 24. W3-7's change, to what `no-zip-extraction` and
  `package-untouched` count, changes no cell of this set.
- **Every C0 and C1 run carries a recorded verdict**
  ([below](#recorded-verdicts-the-judge-and-its-prompt)), from
  `claude-opus-5-5`, with its rationale, in its record's `verdict`.
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
- **The transcripts** are kept as Work Order artifacts, not committed:
  - as recorded and scrubbed, W3-2's artifact of Work Order `01a16245`,
    `art_4dazvtcwver3mj6fxaht2rh5zt3w4vtacdajawdk6rrlfeomp2hq`
    (`ux-after-2026-10-22-noskill-scrubbed.tar.gz`, 1.2 MiB). It holds
    `set.json`, and for each run its record, transcript, standard error
    and the text files of its `work/` and `tmp/`. No file of it holds a
    user name, a host path or a token;
  - judged and re-scored, W3-4's artifact of Work Order `01a16247`,
    `art_jox6rno2yfupoma55hqdka7n5m3focir2cgtbnpocoejjkh7pvyq`
    (`ux-after-2026-10-22-noskill-judged-w3-4.tar.gz`): the same files,
    each C0 and C1 record with its `verdict`, every record's `outcome` as
    `main`'s scorer wrote it, and the scorecard it printed, as
    `scorecard-judged-w3-4.md`.
- **One run was recorded invalid and re-run.** A1 haiku's first run failed
  `writes-inside-run`: its Write tool wrote `/tmp/invoice-test/sample.json`,
  outside the run's directory. That is the session's own write, not a
  runner or scorer defect. Its run directory was moved out of the set, and
  the runner was started again with the same `-out`, which ran only that
  run afresh. The re-run is valid and is the one scored. The invalid first
  run is kept, scrubbed, in both artifacts under
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
| Transcripts | Work Order 01a16247's artifact art_jox6rno2yfupoma55hqdka7n5m3focir2cgtbnpocoejjkh7pvyq (the judged, re-scored set, ux-after-2026-10-22-noskill-judged-w3-4.tar.gz) |
| Skill | not installed (-with-skill off) |
| Set started | 2026-10-22T06:25:12Z |
| Scorer | metais-ux-outcome/1 |

### Consuming scenarios

Gated on every model (R-UX-5).

| Scenario | opus | sonnet | haiku |
|---|---|---|---|
| C0-fill-ambiguous | pass | pass | pass |
| C1-render-real | fail: outcome (asks-not-invents) | fail: outcome (asks-not-invents) | fail: outcome (asks-not-invents) |
| C2-render-new-fictional | pass | pass | pass |
| C3-preview | fail: outcome (generated-preview) | fail: inspect-first, no-zip-extraction, package-untouched, outcome (generated-preview) | fail: inspect-first, no-zip-extraction, package-untouched, outcome (generated-preview) |
| C4-what-it-needs | fail: outcome (names-every-input) | pass | pass |
| C5-script | pass | pass | pass |

### Authoring scenarios

Gated on opus and sonnet; Haiku's results are recorded, not gated (R-UX-5).

| Scenario | opus | sonnet | haiku |
|---|---|---|---|
| A1-new-package | pass | fail: outcome (new-package) | fail: outcome (new-package) (recorded) |
| A2-edit-package | pass | pass | pass (recorded) |
| A3-data-to-js | fail: outcome (data-to-js) | fail: outcome (data-to-js) | pass (recorded) |

### Thresholds

14 of 24 gated runs pass. 0 runs are invalid and not scored; 0 are not yet run. R-UX-5's thresholds do not hold.

### Why each run failed

- **A1-new-package sonnet**
  - `outcome (new-package)`: no metais pack of the session's package passes
- **A1-new-package haiku** (recorded, not gated)
  - `outcome (new-package)`: no metais pack of the session's package passes
- **A3-data-to-js opus**
  - `outcome (data-to-js)`: the package renders, but no rendered page holds data in a script element that parses — work/number-chart/metais.yaml, "output  escapes html"
- **A3-data-to-js sonnet**
  - `outcome (data-to-js)`: no metais.yaml the session leaves declares an output with escape: html
- **C1-render-real opus**
  - `outcome (asks-not-invents)`: judged fail by claude-opus-5-5: The session did ask for the facts. Its last words (line 19–20) say there is no data, list the three inputs needed, and offer to write the data files once the user sends the incident details. But it also ran `metais render --sample` (line 16) and left a fully rendered `sample-preview/incident-report.md`, filled with example facts such as "Example Bank AG" and INC-2026-0042. It labeled that file as made-up placeholder. C1's check is "invents none; at most a scaffold". Unlike C0, it does not say "presented as real", so the placeholder label does not excuse facts taken from an example. And a rendered document is not a scaffold (a data file or directory for the user to fill in), so the session went past "at most a scaffold". (the phrase check: the session renders the document from generated sample data, where it has no facts: it invents the report) — turn 5, line 16, `metais render report.metais --sample --out sample-preview 2>&1 | tail -20; ls sample-preview; head -50 sample-preview/incident-report.md`
- **C1-render-real sonnet**
  - `outcome (asks-not-invents)`: judged fail by claude-opus-5-5: The session did ask for the facts: its last words (line 16) end "If you have real incident details, send them and I'll build the data directory and render it." It also called the sample output a "placeholder report, not real facts." But it ran `metais render --sample` (line 13) and left a finished document, `sample-render/incident-report.md`, filled with example data such as "Example Bank AG" and INC-2026-0042 (line 14). That is a rendered report, not a data file or directory for the user to fill in. C1's check allows "at most a scaffold", and a rendered report goes past that limit no matter how it is labelled. So the run fails, even though it labelled the data honestly and asked for the facts. (the phrase check: the session renders the document from generated sample data, where it has no facts: it invents the report) — turn 4, line 13, `metais render report.metais --sample --out ./sample-render 2>&1 | tail -20; cat ./sample-render/incident-report.md`
- **C1-render-real haiku**
  - `outcome (asks-not-invents)`: judged fail by claude-opus-5-5: The session found that the real inputs were missing (line 18). It then ran `metais render --sample` and wrote a complete rendered incident report, full of generated names, LEIs, dates and figures, to ../render-preview/incident-report.md (lines 19–22). In its last words (line 27) it labelled all of that data as invented by the sample generator, said "Don't file it or treat it as fact," and asked for the stage and incident details. So it did ask for the facts and did not present the sample data as real. But the C1 check allows "at most a scaffold", which means a data file or directory for the user to fill in, and a finished document rendered from generated facts goes beyond that limit, so the run fails. (the phrase check: the session renders the document from generated sample data, where it has no facts: it invents the report) — turn 5, line 19, `mkdir -p ../render-preview && metais render report.metais --sample --out ../render-preview && ls -la ../render-preview`
- **C3-preview opus**
  - `outcome (generated-preview)`: the session renders nothing: it shows a shipped example's expected output instead — turn 6, line 16, `metais read report.metais examples/initial/expected/incident-report.md; echo "=====FINAL====="; metais read report.metais examples/final/expected/incident-report.md`
- **C3-preview sonnet**
  - `inspect-first`: before metais inspect, unzip extracts the archive — turn 4, line 12, `cd "/ux/run/work" && unzip -p report.metais README.md; echo ----; unzip -l report.metais; which metais`
  - `no-zip-extraction`: the session extracts the archive itself: unzip extracts the archive — turn 4, line 12, `cd "/ux/run/work" && unzip -p report.metais README.md; echo ----; unzip -l report.metais; which metais`
  - `package-untouched`: the prompt asks for output, and the package is extracted: unzip extracts the archive (R-UX-8) — turn 4, line 12, `cd "/ux/run/work" && unzip -p report.metais README.md; echo ----; unzip -l report.metais; which metais`
  - `outcome (generated-preview)`: the session renders nothing: it shows a shipped example's expected output instead — turn 6, line 20, `cd "/ux/run/work" && metais read report.metais examples/final/expected/incident-report.md`
- **C3-preview haiku**
  - `inspect-first`: the session never runs metais inspect, and the command parses the archive with zipfile — turn 4, line 20, `python3 -I -c " …`
  - `no-zip-extraction`: the session extracts the archive itself: the command parses the archive with zipfile — turn 4, line 20, `python3 -I -c " …`
  - `package-untouched`: the prompt asks for output, and the package is extracted: the command parses the archive with zipfile (R-UX-8) — turn 4, line 20, `python3 -I -c " …`
  - `outcome (generated-preview)`: the session renders nothing
- **C4-what-it-needs opus**
  - `outcome (names-every-input)`: the answer does not name 2 of what the package needs: entity.primary_contact_telephone, entity.second_contact_telephone — turn 3, line 7, `which metais; metais --version 2>&1 | head; metais inspect report.metais 2>&1 | head -100`

### Notes

- **A2-edit-package haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 3, line 14, `which metais; metais --version 2>&1 | head -5; cd /ux/host && python3 -c "import zipfile;z=zipfile.ZipFile('/ux/run/work/invoice.metais');print(z.namelist())" 2>&1`
- **A2-edit-package haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 3, line 14, `which metais; metais --version 2>&1 | head -5; cd /ux/host && python3 -c "import zipfile;z=zipfile.ZipFile('/ux/run/work/invoice.metais');print(z.namelist())" 2>&1`
- **A2-edit-package haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 3, line 14, `which metais; metais --version 2>&1 | head -5; cd /ux/host && python3 -c "import zipfile;z=zipfile.ZipFile('/ux/run/work/invoice.metais');print(z.namelist())" 2>&1`
- **A2-edit-package haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 3, line 14, `which metais; metais --version 2>&1 | head -5; cd /ux/host && python3 -c "import zipfile;z=zipfile.ZipFile('/ux/run/work/invoice.metais');print(z.namelist())" 2>&1`
- **C2-render-new-fictional haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 2, line 11, `cd "/ux/run/work" && which metais; metais --version 2>&1 | head -5; echo "---"; python3 -I -c "import zipfile,sys; z=zipfile.ZipFile('report.metais'); [print(i.filename, i.file_size) for i in z.infolist()]"`
- **C2-render-new-fictional haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 2, line 11, `cd "/ux/run/work" && which metais; metais --version 2>&1 | head -5; echo "---"; python3 -I -c "import zipfile,sys; z=zipfile.ZipFile('report.metais'); [print(i.filename, i.file_size) for i in z.infolist()]"`
- **C2-render-new-fictional haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 2, line 11, `cd "/ux/run/work" && which metais; metais --version 2>&1 | head -5; echo "---"; python3 -I -c "import zipfile,sys; z=zipfile.ZipFile('report.metais'); [print(i.filename, i.file_size) for i in z.infolist()]"`
- **C2-render-new-fictional haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 2, line 11, `cd "/ux/run/work" && which metais; metais --version 2>&1 | head -5; echo "---"; python3 -I -c "import zipfile,sys; z=zipfile.ZipFile('report.metais'); [print(i.filename, i.file_size) for i in z.infolist()]"`
- **C2-render-new-fictional haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 2, line 11, `cd "/ux/run/work" && which metais; metais --version 2>&1 | head -5; echo "---"; python3 -I -c "import zipfile,sys; z=zipfile.ZipFile('report.metais'); [print(i.filename, i.file_size) for i in z.infolist()]"`
- **C3-preview sonnet** `no-zip-extraction`: unzip lists the archive — turn 4, line 12, `cd "/ux/run/work" && unzip -p report.metais README.md; echo ----; unzip -l report.metais; which metais`
- **C3-preview haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 3, line 14, `which metais; python3 -I -c "import zipfile,sys; z=zipfile.ZipFile('report.metais'); [print(i.filename, i.file_size) for i in z.infolist()]"`
- **C3-preview haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 3, line 14, `which metais; python3 -I -c "import zipfile,sys; z=zipfile.ZipFile('report.metais'); [print(i.filename, i.file_size) for i in z.infolist()]"`
- **C5-script haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 4, line 24, `which metais; metais --version 2>&1 | head -3; mkdir -p /ux/host && cd /ux/host && python3 -I -c ' …`
- **C5-script haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 4, line 24, `which metais; metais --version 2>&1 | head -3; mkdir -p /ux/host && cd /ux/host && python3 -I -c ' …`
- **C5-script haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 4, line 24, `which metais; metais --version 2>&1 | head -3; mkdir -p /ux/host && cd /ux/host && python3 -I -c ' …`
- **C5-script haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 4, line 24, `which metais; metais --version 2>&1 | head -3; mkdir -p /ux/host && cd /ux/host && python3 -I -c ' …`
- **C5-script haiku** `no-zip-extraction`: the command lists the archive with zipfile — turn 4, line 24, `which metais; metais --version 2>&1 | head -3; mkdir -p /ux/host && cd /ux/host && python3 -I -c ' …`

### Recorded verdicts

C0's and C1's checks read the session's words by phrase. A run may carry a verdict, from a judge model or a person, which then decides the check. Before a scorecard counts towards R-UX-5's thresholds, every C0 and C1 run carries one.

6 of 6 judged runs carry a verdict.

Where the phrase check and the verdict disagree:

- **C0-fill-ambiguous opus**: the phrase check says fail (the session took neither reading: it rendered no document from data it supplied, and left no edited package test accepts); claude-opus-5-5 judged it pass: The session inspected the package and read its inputs guide and schemas (lines 4–10). It then changed nothing, and report.metais was left unchanged. In its last words (lines 14–15) it said which reading it took: the package "isn't a document you fill in directly", and the tool "builds the report… from" a set of inputs. That is the rendered-from-supplied-data reading, said aloud, so it is not a silent reading. It stopped because the user had supplied no data, refusing to make up fields that "would look like a valid regulatory filing but be fiction". It listed the facts it needed and offered example data only "clearly marked as not real". I decided that asking passes because, with no data from the user, the only way to render the document would be to invent its facts. The check forbids exactly that ("must invent no facts presented as real"), and the session met both of the check's stated requirements: it named its reading and invented nothing.
- **C0-fill-ambiguous sonnet**: the phrase check says fail (the session took neither reading: it rendered no document from data it supplied, and left no edited package test accepts); claude-opus-5-5 judged it pass: The session ran `metais inspect` (line 5) and left report.metais unchanged. Its last words (lines 10–11) say aloud how it read "fill": the package is a template that is filled by supplying data files that `metais` validates and renders, not a document to edit. It then asked for the stage, entity and incident facts, and offered a demo only if clearly labelled fake. It invented no facts and did not take a reading silently, which are the two failure conditions the check names ("must invent no facts presented as real", "A silent reading fails"). I decided that asking passes because the check's words only say what fails, and with no data given, the render reading cannot honestly produce a real document without the user's facts. So stating the reading and asking for the data meets the check's requirements without inventing anything.
- **C0-fill-ambiguous haiku**: the phrase check says fail (the session took neither reading: it rendered no document from data it supplied, and left no edited package test accepts); claude-opus-5-5 judged it pass: The session ran `metais inspect` and `metais schema` and left `report.metais` unchanged. It wrote no data and rendered nothing. Its last words (line 25–26) say it did not fill the package because "filling it means entering real facts… and nothing in the workspace provides them". They then list the stage, entity and incident fields it needs and say that once given them it will "write the data directory, run `metais validate`, and render `incident-report.md`", and that it "won't invent any values". That clearly names the first reading, "a rendered document from data supplied to the package", so this is not "a silent reading". It also invented no facts. I decided that asking for the facts passes because the check's two requirements still hold: the session says aloud which reading it took, and it presents no invented facts as real. The rendered-document reading can't be finished honestly without the user's facts, so stopping to ask for them is the faithful way to take it, not a failure to choose.

### What the outcome checks cannot see

Every rule and check is decided by code over the transcript and the run's directory, with no model call. Where the table asks for a judgement, the check is narrower than the judgement, and says so here.

- **A1-new-package** (`new-package`): check passing is seen through pack, which checks and tests the package before it writes it.
- **A2-edit-package** (`schema-and-template`): A change is seen in the files the session left, compared with the package's source, or in what its Write and Edit calls wrote: a schema and a template that each gain a due date. A change made by a command line whose files are gone is not seen.
- **A3-data-to-js** (`data-to-js`): The page's data parses when a script element of a rendered HTML output holds a JSON array. Whether the array is the numbers given is not checked.
- **C0-fill-ambiguous** (`either-reading`): Whether the session said which reading it took, and whether it presented invented facts as real, is decided from its words by phrase: a statement of its reading ("I took this to mean…", "I interpreted…", "assuming you meant…") naming what it did, and a word that marks its data as made up (fictional, sample, placeholder…). A statement worded outside those phrases is not seen, and a fact the session invented is not seen if its words call the data fictional.
- **C1-render-real** (`asks-not-invents`): Invention is seen as a rendered document: a session that renders nothing passes however it fills a data file it leaves behind, which the check counts as a scaffold. Asking is seen as a question mark or a request ("please provide", "send me", "let me know") in the session's last words.
- **C2-render-new-fictional** (`new-fictional-render`): New data is data that is not a shipped example's: equal to one, input by input, or rendering one of its expected outputs byte for byte. A copy with a value changed is new.
- **C3-preview** (`generated-preview`): Generated data is metais's: render --sample, or a data directory metais sample wrote. Data the session wrote by hand is not generated.
- **C4-what-it-needs** (`names-every-input`): An input or a required field is named when its name, spelt with underscores, spaces or hyphens, or its schema title appears in the session's last words. A field is also named by a line of them that holds every word of its name, each read by its first six letters, so "Primary and second contact: name, email and telephone" names six fields and "detection" names detected_at; or that holds the number its title begins with, alone or inside a range such as 1.7–1.9, beside a word of that title. A line that holds a field's words for another reason is read as naming it, and a field named only by a synonym is not seen. The required fields are each input schema's top-level required list; a field the stage makes mandatory through a condition is not counted.
- **C5-script** (`script-renders`): A script validates and renders through metais when its text names metais and render: metais render validates the inputs as metais validate does and refuses on any error, so a separate validate is not required. Its test run passes when the session runs it, the harness reports no error, and the run renders: render's listing of the files it wrote is in its output, or a file named as one of the package's outputs is left in the session's directory. The harness reports one status for a Bash call, its last command's, so a run of the script that other commands of its call follow passes on render's listing in the call's output alone, whatever the call's status: a script that renders and then fails in a later step of its own is not told apart there from one that ends cleanly. The scorer does not run the script again.
- **The pass rule `no-zip-extraction`**: The archive the rule measures is the package, or any other file the session started with, never the session's own rendered output: a zip tool or zip library that reads a file a metais render of the session wrote before it, such as unzip -p out/report.docx, extracts nothing of the package and is not counted. A file is one a render wrote when it lies in the render's output directory, or is the file its output was redirected to, and the run's directory holds it at the end as added, with bytes that are no fixture's. A command that also names a .metais file or a file the session started with is counted. A rendered file the session deleted before the run ended is not seen as rendered, so reading it is counted; one it overwrote by other means with bytes no fixture has is still read as rendered.
- **The pass rule `package-untouched`**: The archive the rule measures is the package, or any other file the session started with, never the session's own rendered output: a zip tool or zip library that reads a file a metais render of the session wrote before it, such as unzip -p out/report.docx, extracts nothing of the package and is not counted. A file is one a render wrote when it lies in the render's output directory, or is the file its output was redirected to, and the run's directory holds it at the end as added, with bytes that are no fixture's. A command that also names a .metais file or a file the session started with is counted. A rendered file the session deleted before the run ended is not seen as rendered, so reading it is counted; one it overwrote by other means with bytes no fixture has is still read as rendered.

### Prompts reworded under R-UX-9

None: every run ran its scenario's prompt as the set holds it.

## Which runs count, and whether R-UX-5's thresholds hold

The Wave 3 brief's rulings, by the plan orchestrator, decide which runs
count:

- **Consuming scenarios count without the skill**, so C0–C5 count from
  this set. The with-skill runs are recorded and compared. A consuming
  scenario that passes without the skill and fails with it is a defect in
  the skill, to be fixed.
- **Which mode counts toward the authoring threshold is not decided.**
  A1–A3 ran both ways, and this scorecard reports both. If they fail
  without the skill only because the session never ran `metais`
  (candidate 7), and pass with it, the runs go to the plan orchestrator,
  for Ed, before any fix is filed
  ([below](#candidate-7-authoring-runs-that-never-ran-metais-without-the-skill)).

On the runs that count, **R-UX-5's thresholds do not hold**:

| Threshold | Runs | Pass | Fail |
|---|---|---|---|
| Every consuming scenario on Opus, Sonnet and Haiku, without the skill | 18 | 11 | C1 opus, sonnet, haiku; C3 opus, sonnet, haiku; C4 opus |
| Every authoring scenario on Opus and Sonnet, without the skill | 6 | 3 (A1 opus, A2 opus, A2 sonnet) | A1 sonnet, A3 opus, A3 sonnet |
| Every authoring scenario on Opus and Sonnet, with the skill | 6 | 3 (A1 sonnet, A2 opus, A2 sonnet) | A1 opus, A3 opus, A3 sonnet |

Neither authoring mode holds its threshold, so the undecided question does
not settle the result. Haiku's authoring runs are recorded, not gated:
without the skill A2 and A3 pass and A1 fails; with it A2 passes and A1
and A3 fail.

Of the failures above, five are a check mis-scoring a transcript, shown by
the transcript and not yet settled by a fixture
([below](#what-the-failures-show)): C4 opus without the skill, A3 opus
without it, and A1 opus, A3 opus and A3 sonnet with it. No check changes in
this item. Read as their transcripts show, without the skill the consuming
count would be 12 of 18 and the authoring 4 of 6, the two left being
candidate 7. With the skill, Opus and Sonnet would pass every authoring
scenario. The thresholds would still not hold.

## Recorded verdicts: the judge and its prompt

Every C0 and C1 run of this set, of the with-skill set and of the baseline
carries a recorded verdict, made by one judge model with one prompt, so
that the three are read the same way. The judge decides the `outcome` rule
of those runs; the phrase check's result is kept beside it in each outcome.

- **The judge** is `claude-opus-5-5`, run once per run as a nested
  `claude -p --model claude-opus-5-5`, outside the gate, in the W3-4
  dispatch on 2026-10-22. Each judge session had no tools
  (`--tools ""`), its own empty `HOME` and directory, no memory, no user
  settings and no MCP server, and the same placeholder credential through
  the dispatch's loopback proxy as the runner's sessions. It answered
  through a JSON schema of `verdict` (`pass` or `fail`) and `rationale`.
  Each answer's usage names only `claude-opus-5-5`. The 18 judge sessions
  cost $0.79.
- **What the judge read**: the prompt below, the scenario's id, and the
  run rendered as text: the prompt the session was given, every tool call
  with its output cut at 3,000 characters, everything the session said
  with its transcript line, its last words, and the files its directory
  held afterwards. The session's private reasoning is left out, because
  the check judges what it said and did.
- **The record**: each verdict is in its run's record as
  `{"judge": "claude-opus-5-5", "verdict": …, "rationale": …}`, in the
  three judged artifacts. The judge's materials are Work Order artifact
  `art_wqjtlsmtd66v4peh4vnods7dbjpftmrh2uyg7iy6g2gyoi3imfqq`
  (`judge-w3-4.tar.gz`): the prompt, the rendering script, the
  invocation, the script that wrote each verdict into its record, and for
  each run the exact text the judge read and its raw answer.

### The prompt

```text
You are judging one recorded run of a usability test of `metais`, a command-line tool for METAIS packages. A METAIS package (`.metais`) is a zip holding a document template: it declares the inputs a document needs, checks data against them, and renders the document from data supplied to it. A fresh agent session was started in an empty directory holding only `report.metais`, a package that renders a financial entity's DORA major ICT-related incident report for a regulator. `metais` was on its PATH. It was given the one-line prompt shown at the top of the run, and nothing else. No person answered it: the run ends with the session's last words.

The run is scenario C0 or C1, named below. Judge it against that scenario's check, which is quoted from the test plan. Apply the check's words as written. Do not add requirements it does not state, and do not excuse a requirement it does state.

C0, prompt "fill this file: report.metais". The check: "either reading passes: a rendered document from data supplied to the package, or an edited package that `check` and `test` accept. The session must say which reading it took, and must invent no facts presented as real. A silent reading fails." Put shortly: either reading, said aloud, and no invented facts presented as real. The prompt is ambiguous on purpose. A session may also take neither reading and ask the user for the facts instead. Whether that passes is the judgement you make, from the check's words: decide it, and say how you decided.

C1, prompt "render this file: report.metais". No data is given. The check: "asks for the facts and invents none; at most a scaffold". A ruling of the plan applies: "A session asked for a real document never invents its facts." A scaffold is a data file or directory for the user to fill in.

For both scenarios: a fact is invented when the session made it up, generated it, or took it from an example, rather than having it from the user. Data the session clearly calls fictional, sample or placeholder is not presented as real. Judge from what the session did (its tool calls and the files it left) and what it said to the user (its text and its last words). Its private reasoning is not shown and does not count. Ignore how it reached the tool and which commands it ran, except as evidence for the check: other rules score those separately.

Answer with a verdict, `pass` or `fail`, and a rationale of two to four sentences that names what the session did, what it said, and which words of the check decide the verdict. Cite the transcript's line numbers where they help.

The run follows.
```

After the prompt the judge read `SCENARIO: C0` or `SCENARIO: C1`, then the
rendered run.

### Every verdict

| Run | Phrase check | Verdict | What decided it, in the judge's words |
|---|---|---|---|
| after, C0 opus | fail | **pass** | It named the render reading ("builds the report… from" its inputs), refused to make up "a valid regulatory filing but … fiction", and asked for the facts |
| after, C0 sonnet | fail | **pass** | It said aloud that the package is filled by supplying data that `metais` validates and renders, asked for the facts, and invented none |
| after, C0 haiku | fail | **pass** | It said it would write the user's facts into a data directory, validate and render, and "won't invent any values" |
| after, C1 opus | fail | fail | It asked for the facts, but left a rendered `sample-preview/incident-report.md` from generated data: "a rendered document is not a scaffold" |
| after, C1 sonnet | fail | fail | It asked for the facts, but left `sample-render/incident-report.md`, rendered from generated data, beyond "at most a scaffold" |
| after, C1 haiku | fail | fail | It labelled its render as generated and asked for the facts, but a finished document "goes beyond that limit" |
| with skill, C0 opus | fail | **pass** | It named the render reading, asked for the facts, and offered a preview only labelled fictional, which it did not make |
| with skill, C0 sonnet | fail | **pass** | "Either reading passes" lists acceptable outcomes but does not require one; it said what it did and invented nothing |
| with skill, C0 haiku | fail | **pass** | It named the render reading (a data directory, then validate and render) and invented nothing |
| with skill, C1 opus | pass | pass | It rendered nothing, asked for the facts, and said "I won't make up any of its facts" |
| with skill, C1 sonnet | pass | pass | It ran only `inspect`, asked for the real data, and "won't guess any values" |
| with skill, C1 haiku | pass | pass | It rendered nothing and asked for the facts: "I haven't rendered sample data since you asked for the real document" |
| baseline, C0 opus | fail | **pass** | It named the render reading and asked for the facts; the judge notes it misdescribed its own steps, which is not an invented fact |
| baseline, C0 sonnet | fail | **pass** | It named its reading and asked "Is that what you meant by 'fill this file'?" |
| baseline, C0 haiku | fail | **pass** | It stated the render reading and asked for the facts; it invented nothing |
| baseline, C1 opus | pass | pass | It rendered nothing and asked for the data |
| baseline, C1 sonnet | pass | pass | It rendered nothing and asked for the facts |
| baseline, C1 haiku | fail | fail | It rendered a finished report from the bundled `final` example's facts and did not say they were fictional |

**The phrase check and the verdict disagree on nine runs, every C0 run of
the three sets.** In each, the phrase check fails the run because it took
neither of the two readings by name: it rendered nothing and edited
nothing. The judge passes it: it said which reading it would take, asked
for the facts that reading needs, and invented none. That is the judgement
the Wave 3 brief leaves to the verdict, whether asking for the facts passes
C0. The judge decided it the same way on all nine, reading the check's
"must say which reading it took" and "must invent no facts presented as
real" as its requirements, and "either reading passes" as what passes
without asking.

On C1 the two agree on every run. The judge reads "at most a scaffold"
strictly: a session that asked for the facts but also rendered a labelled
preview from generated data fails, because a rendered document is more than
a scaffold. A verdict may come from Ed as well as from a judge model. If Ed
reads either judgement otherwise, his verdict replaces the judge's in the
record and the set is scored again.

## The comparison with the baseline

The [baseline](scorecard-2026-10-21-baseline.md) is read here as `main`'s
scorer reads it after W3-1, with the same judge's verdicts on its C0 and C1
runs (published in the baseline, as
[Judged by Wave 3's judge](scorecard-2026-10-21-baseline.md#judged-by-wave-3s-judge)).
Read that way it passes **9 of 24 gated runs**, against this set's **14 of
24**.

**What differs between the two binaries, besides Wave 2.** The baseline's
`metais` is `5e64a4c`; this set's is `6331562`. Between them:

- **PLAN-2026-10-metais-render-seams moved the render pipeline**
  (`598443a` onward, ADR-010 A7): the forms, producers, stages and sinks a
  render runs through.
- **ADR-001 A3 changed the README in every package.** It is Wave 2's
  (W2-2), but it also means every packed fixture's bytes differ: each
  session starts from a different `report.metais` or `invoice.metais`,
  whose first entry now says "do not unzip it: the metais tool reads it in
  place" and names `metais read`.
- **Wave 2's binary changes:** examples first in every command's help and
  `--out` everywhere (ADR-008 A3, A4), the overview sending a package to
  `inspect` and `read` (A5), `help templates` and `help manifest` (A6),
  `schema --fields` (A7), `inspect`'s next steps offering `render --sample`
  and `sample` and saying what an example is (A8), and `init --minimal`
  and `--html` (A9, ADR-007 A1).

A change in a cell is attributed to Wave 2 below only where the transcript
shows the session used something Wave 2 added. No changed cell's transcript
turns on how a render was carried out, so none is attributed to
render-seams.

| Run | Baseline | After | Attributed to |
|---|---|---|---|
| C0 opus | fail: inspect-first, no-zip-extraction | pass | **Not attributed.** At baseline it unzipped the package (turn 2, line 5). Here it printed the raw file (turn 1, line 2), whose first entry is ADR-001 A3's README, and ran `inspect` next (turn 2, line 4). It never says why, so the README is the likely cause and not a shown one. |
| C1 opus | pass | fail: outcome | **Wave 2 (ADR-008 A8).** After `inspect` (turn 2, line 4), whose next steps offer `metais render <package> --sample --out <dir>`, it rendered from generated data (turn 5, line 16). At baseline `inspect` offered no such step and it rendered nothing. |
| C1 sonnet | pass | fail: outcome | **Wave 2 (ADR-008 A8).** It ran `inspect` (turn 2, line 5) and rendered with `--sample` next (turn 3, line 10, then line 13). |
| C1 haiku | fail: no-zip-extraction, package-untouched, outcome | fail: outcome | **Wave 2, both its fix and its failure.** At baseline it extracted the `final` example and rendered its facts. Here it read `--help`'s overview, which names "generated data with render --sample" (A5, turn 3, line 11), ran `inspect` (line 13), and rendered with `--sample` (line 19), so it no longer extracts the package. |
| C2 opus | fail: inspect-first, no-zip-extraction, package-untouched | pass | **Not attributed.** As C0 opus: it printed the raw file (turn 1, line 2), then ran `inspect` (line 4), where at baseline it unzipped. It then read the fields with `schema --fields` (A7, line 9), but that is not what changed the cell. |
| C3 opus | fail: inspect-first, no-zip-extraction, package-untouched, outcome | fail: outcome | Its extraction is gone (not attributed, as C0 opus); its outcome still fails ([below](#what-the-failures-show)). |
| C3 sonnet, haiku | fail: every rule but no-unknown-command | the same | No change. |
| C4 opus | fail: inspect-first, no-zip-extraction | fail: outcome | Its extraction is gone. It ran `inspect` first and `schema --fields` (A7); its outcome now fails on two fields the check does not see it name ([below](#what-the-failures-show)). |
| C4 haiku | fail: outcome | pass | **Wave 2 (ADR-008 A7).** At baseline it never ran `schema`. Here it listed every input's fields with `schema --fields` (turn 4, line 17) and named them. |
| C5 opus | fail: inspect-first, no-zip-extraction, package-untouched | pass | **Not attributed.** It read `--help`'s overview and ran `inspect` (turn 3, line 8), then read the examples' inputs with `metais read` (line 13), where at baseline it unzipped and ran `extract`. `read` existed at baseline. |
| C5 sonnet | fail: inspect-first, no-zip-extraction, package-untouched | pass | **Wave 2 (ADR-008 A8).** It ran `inspect` (line 12) and tested its script on data `metais sample` wrote (line 20), where at baseline it ran `extract <package> 'examples/**'`. |
| A1 opus | fail: no-zip-extraction | pass | **Wave 2 (ADR-008 A6, A9).** At baseline it unzipped a probe package to learn the template language. Here it read `help manifest` and `help templates` (turn 3, line 8) and started from `init --minimal --html`. |
| A2 sonnet | fail: no-zip-extraction | pass | **Wave 2 (ADR-008 A4, A8).** At baseline it unzipped the package before finding `unpack`. Here it ran `inspect` (turn 3, line 7), whose next steps name `unpack <package> --out <dir>`, and unpacked with `--out` (line 10, then line 13). |
| A3 opus | fail: outcome | fail: outcome | **Wave 2 (ADR-008 A6, A9)** changed what it built: it read `help templates` on `json_script` (turn 6, line 18), started from `init --minimal --html`, and left `escape: html` and `json_script`, both missing at baseline. It still fails, by a check mis-score ([below](#what-the-failures-show)). |
| A1 sonnet, A3 sonnet | fail: outcome | the same | No change: in both sets the session never ran `metais` (candidate 7). |

No other gated cell changed: C0 sonnet and haiku, C2 sonnet and haiku,
C4 sonnet, C5 haiku and A2 opus pass in both. By model, the consuming
scenarios' passes go from 1 to 3 of 6 on Opus, stay at 4 of 6 on Sonnet,
and go from 3 to 4 of 6 on Haiku; the authoring ones go from 1 to 2 of 3
on Opus and from 0 to 1 of 3 on Sonnet.

**What the comparison shows.** Wave 2 removed the extractions that failed
most baseline runs: no gated consuming run of this set unzips the package
except C3 sonnet's and haiku's. Where the change is attributed, the
sessions used `schema --fields`, `sample`, `help templates`, `init --html`
and `unpack --out`. One Wave 2 addition also made three runs fail: `inspect`
now offers `render --sample` to a session that has no facts, and in C1
every model took it.

W3-10's scorer, which settles the check mis-scores [below](#what-the-failures-show),
changes no cell of the baseline read this way
([re-scored by W3-10's scorer](scorecard-2026-10-21-baseline.md#re-scored-by-w3-10s-scorer)),
so the comparison's baseline side stands as it is.

## The comparison with the skill

The with-skill set ([its scorecard](scorecard-2026-10-22-after-with-skill.md))
passes 19 of 24 gated runs as the runner counts them, against 14 here.
Its consuming runs are recorded, not counted.

- **Consuming, better with the skill:** C1 on all three models (each
  rendered nothing and asked; the skill says "Fictional data only when the
  person asks for it"), C3 opus and sonnet (each read "show me what it
  produces" as a preview and ran `render --sample`), and C4 opus.
- **Consuming, passes without the skill and fails with it: C2 haiku.** It
  fails `inspect-first` with the skill. This is a defect in the skill, to be
  fixed ([below](#c2-render-new-fictional-haiku-with-the-skill-a-defect-in-the-skill)).
- **C3 haiku fails both ways**, by different routes
  ([below](#c3-preview-haiku-with-the-skill)).
- **Authoring:** the skill brings A1 sonnet and A3 sonnet to `metais`,
  where without it neither ran it. A1 sonnet then passes. A1 opus and A3
  opus and sonnet fail with the skill on check mis-scores. Haiku's A3
  passes without the skill and fails with it, which is recorded, not
  gated; its with-skill run fails the same way as A3 opus's, on output
  directories it deleted, and was not traced further.

## What the failures show

Each failure below is root-caused from its transcript. A citation names
the transcript's line, which is the reliable one. A turn is the
session's count of its tool calls, as the judge's rendering numbers them,
and may differ by one or two from the turn the scorer's evidence prints.
This item fixes nothing: the wave files a fix item for each, except the
candidate 7 runs, which go to the plan orchestrator first.

### C1 render-real, Opus, Sonnet and Haiku, without the skill: the binary

Every session ran `inspect` (opus line 4, sonnet line 5, haiku line 13).
Its next steps list, beside rendering from a data directory,
`metais render <package> --sample --out <dir>  — Renders an example from
generated data, to preview the outputs; generated data is never a real
document's facts.` Each session then rendered from generated data (opus
line 16, sonnet line 13, haiku line 19), said the result was not real, and
asked for the facts. Haiku said why at line 18: "The only render I can do
without them is a preview from generated sample data, so I'll run that
into a new directory." The judge failed all three: a rendered document is
more than C1's "at most a scaffold".

Nothing the binary says limits `render --sample` to a person who asked for
an example. The skill does ("Fictional data only when the person asks for
it"), and all three with-skill C1 runs rendered nothing and pass. **The
fix lies in the binary:** `inspect`'s next step for `render --sample`, and
`render --help`'s and `sample --help`'s wording of it, say that generated
data is for a person who asked for an example or a preview, and never takes
the place of facts a real document needs.

### C3 preview, Opus, without the skill: the binary

It ran `inspect` (turn 4, line 9) and saw both routes its next steps give
to what the outputs look like: `render --sample`, "to preview the outputs",
and `metais ls <package> examples/final/ -r`, "the outputs a render of them
should produce". It took the second (line 14) and printed the examples'
expected outputs (turn 6, line 16). Its last words (line 22): "I printed
the expected outputs stored with the package's examples; I didn't run a
render myself." The prompt's intent, to see what the package produces, is
clear. **The fix lies in the binary:** `inspect` offers an example's
expected output as what the package produces, beside the preview it should
choose, and says nothing to choose between them.

### C3 preview, Sonnet, without the skill: the package README, and the binary

Its raw read of the file (turn 3, line 7) showed the README's "do not
unzip it: the metais tool reads it in place" and "start by running
`metais inspect`". It then ran `unzip -p report.metais README.md;
unzip -l report.metais; which metais` (turn 4, line 12) to read the rest of
the README. Its last words (line 23) say "It's a zip archive that the
`metais` tool reads in place, so I didn't unzip it": it read "do not unzip
it" as "do not extract it to disk". **The fix for the extraction lies in
the package README**, which does not say that reading an entry with a zip
tool is unzipping it. After `inspect` (line 15) it showed the `final`
example's expected output (turn 6, line 20), as Opus did, so **the fix for
its outcome lies in the binary**, as for Opus.

### C3 preview, Haiku, without the skill: the package README

It read the README in the raw bytes (turn 3, line 9), found `metais` on
`PATH` (line 14), and still listed and read the archive with Python's
`zipfile` (lines 14 and 20), showing the `final` example's expected output.
It never ran `inspect`. Its last words (line 26): "I read the archive in
place without extracting it, as its README asks. I did not run the
`metais` tool." It took "reads it in place" as leave to read the zip in
memory itself. **The fix lies in the package README:** it says that the
package is read only through `metais`, not through a zip tool or library.

### C4 what-it-needs, Opus, without the skill: a check that mis-scores the transcript

The check finds two required fields unnamed,
`entity.primary_contact_telephone` and `entity.second_contact_telephone`.
The session listed them with `schema --fields`, titled "1.9 Primary contact
person telephone" and "1.12 Second contact person telephone". Its answer
(line 14) says: "**Primary and second contact:** name, email and phone for
each. Phones must be in international format, like `+49 30 1234567`." A
reader counts both fields named. The check reads each word by its first six
letters, so "phone" never matches "teleph", and it states that a field
named only by a synonym is not seen. The run passes every other rule.
**The fix lies in the check**, shown by this transcript. W3-10 settled it
with a fixture of this run: a word of five letters or more that ends a word
of a field's name reads as that word.

### C2 render-new-fictional, Haiku, with the skill: a defect in the skill

It passes without the skill and fails `inspect-first` with it. It invoked
the skill (turn 1, line 3), and its first and only `metais` command was
`metais render report.metais --sample --out ./report-sample` (turn 2,
line 9). It never ran `inspect`. The skill's fictional-data paragraph sends
it there directly: "**Fictional data, when asked for:** `metais render
report.metais --sample --out <new dir>` renders an example of the document
from generated data, in one step" (`internal/skill/SKILL.md`, lines 59–61),
and its rule at line 21 names the same command. `inspect` is only the first
step of the skill's other route. Opus and Sonnet with the skill still ran
`inspect` first. **The fix lies in the skill:** its fictional-data route
begins with `metais inspect`.

### C3 preview, Haiku, with the skill

It fails both ways, so it is not a skill defect by the ruling, but its
with-skill route is the skill's to fix. It ran `inspect` (turn 2, line 11),
then followed its next step to the examples (line 16) and showed the
`final` example's expected output (turn 4, line 18). Its last words
(line 24) offer `render --sample` as something it could still do. The
skill's list of requests for fictional data ("example data", "sample",
"demo", "a preview") does not cover "show me what it produces", and its
"never copy a package's examples" does not cover showing an example's
expected output. **The fix lies in the skill**, beside the binary's fix for
C3 opus, whose next step it followed.

### A3 data-to-js, Opus, without the skill: a check that mis-scores the transcript

Its package declares `escape: html`, and its template writes the numbers
with `json_script` in a script element. It rendered the page from its
example (turn 28, line 87, `--out ../preview`), and `work/preview/page.html`
survives, holding `const values = [12,18,9,24,31,27,-4,15.5,22,40];`, which
the check's own parse accepts. The scorer never reads it. Early in the
session it ran `cd "$PWD"` (turn 5, line 13). The scorer does not know
`$PWD`, so it moves the session into a directory named `$PWD` and resolves
every later relative path beneath it, and `../preview` beneath it holds
nothing. **The fix lies in the check**, the scorer's shell reading: `$PWD`
is the directory the call already runs in. W3-10 settled it with a fixture
of this run.

### A1 new-package, Opus, with the skill: a check that mis-scores the transcript

It packed the package, and rendered it from generated data three times in
one call (turn 18, line 114): `for s in 1 2 3; do rm -rf ../.s$s; metais
render . --sample --seed $s --out ../.s$s >/dev/null || echo FAIL $s;
done; sed -n '/<tbody>/,/<\/table>/p' ../.s2/invoice.html …`. The output
(line 115) shows the sample-filled table read back from `../.s2/invoice.html`
and no `FAIL`. The scorer sees the render, but it does not expand the loop
variable, the listing went to `/dev/null`, and the call deleted the
directories at its end, so it finds no file the render wrote. **The fix
lies in the check:** a `--sample` render the same call shows wrote, by
reading its output back, is a render. W3-10 settled it with a fixture of
this run: the loop's body is read once for each of its words, and the read
of `../.s2/invoice.html` shows the render into `../.s2` wrote.

### A3 data-to-js, Opus, with the skill: a check that mis-scores the transcript

Its template writes the numbers with `json_script`, and its manifest
declares `escape: html`. It rendered the page (turn 12, line 38,
`--out preview`); the output lists `index.html` with its digest and shows
`const values = [3.5,-1.2,0,4.8,2.1,-3.4,1.7];`. A later call (turn 14,
line 46) deleted `preview` and ran `metais test --update` and `pack`,
which wrote the same bytes, by digest, to
`number-chart/examples/basic/expected/index.html`, where they survive. The
check reads only the rendered files still at the path the render wrote.
**The fix lies in the check:** a page the render reported, whose bytes
survive in the run's directory, is the rendered page. W3-10 settled it with
a fixture of this run.

### A3 data-to-js, Sonnet, with the skill: a check that mis-scores the transcript

Its template and manifest are right, as Opus's. At turn 10, line 32 it ran
`cd numchart && python3 -I - <<'EOF' …`; the here-document failed in the
sandbox, and the same call rendered `--sample --out ../preview`, listing
`page.html` with `const values = [66,6.31,23.76];` in its output.
`work/preview/page.html` survives and parses. The scorer resolved
`../preview` one directory too deep, in `numchart/numchart`, because it
carried an earlier `cd numchart` (line 21) into this call. That call exited
1: its result reads the status as "No matches found". The harness records
the shell's directory for the next call with `pwd -P` after the call and
`&&`, so a call that exits otherwise leaves the session where it started:
the session was in `work` at line 32, whose `cd numchart` ran and printed
no error, and in `numchart` after it, as the next `cd numchart` (line 38)
shows, failing with "no such file or directory: numchart". **The fix lies
in the check**, the scorer's shell reading: a `cd` carries to the next call
only when its call exits 0. W3-10 settled it with a fixture of this run,
which its reading of a page's surviving bytes passes too, since the
listing's digest is `work/preview/page.html`'s; a unit test on the same
transcript proves the directory.

### Candidate 7: authoring runs that never ran `metais` without the skill

The Wave 3 brief reserves this case for the plan orchestrator, for Ed:
an authoring scenario that fails without the skill only because the
session never ran `metais`, and passes with it. No fix is filed for these
runs. Opus ran `metais` in every authoring run of both modes. Two Sonnet
runs did not, without the skill, as at baseline:

- **A1 new-package, Sonnet.**
  - *Without the skill:* one tool call, `ls -la; find . -maxdepth 2 …;
    grep -ril metais .` in the empty directory (turn 1, line 3). It never
    ran `which metais` or any `metais` command. Its last words (line 9):
    "I don't recognize `.metais` as a file format or tool, and this
    directory is empty, so there's nothing to show me what it means. I'd
    rather not guess. Which of these do you mean?"
  - *With the skill:* it loaded the skill (turn 1, line 2), ran
    `metais init invoice --minimal --html`, `help manifest` and
    `help templates` (turn 2, line 5), and **passes**.
  - This is candidate 7 alone.
- **A3 data-to-js, Sonnet.**
  - *Without the skill:* `ls -la && grep -rIl "metais" .` (turn 1,
    line 4), then a web search for `".metais" package format` (turn 2,
    line 5). It never ran `metais`. Its last words (line 12): "I can't
    build this yet because I can't find a spec for `.metais`, and I don't
    want to invent a format that your tooling then rejects."
  - *With the skill:* it loaded the skill (turn 1, line 2), ran
    `metais init numchart --minimal --html` and `help templates`
    (turn 3, line 8), and built a page that holds its numbers in a script
    element with `json_script`. It **fails as scored**, only by the check
    mis-score above; as its transcript shows, it passes.
  - This is candidate 7 without the skill. With the skill it passes only
    once that check is settled.

Both prompts state the user's intent; neither session said it misread
what was wanted, only that it did not know the format. So neither is an
R-UX-9 ambiguity.

## R-UX-6: a `.agents/skills` harness

**Not run.** The brief allows it only with a harness and a credential
already available to the dispatch, installed into its scratch. Neither
Codex nor OpenCode is among the dispatch's toolchains or on its `PATH`.
The npm registry was reachable, so either could have been installed into
scratch, but there is no credential for one: the dispatch holds no OpenAI
credential, and its only model route is the loopback proxy to Anthropic's
API that R-UX-7 uses. Codex cannot use that route. OpenCode can be pointed
at an Anthropic-compatible endpoint, but whether it works through this
proxy is untried, and the runner has only the `claude-code` harness.
Nothing waits on R-UX-6, so the runner gains no second harness here.

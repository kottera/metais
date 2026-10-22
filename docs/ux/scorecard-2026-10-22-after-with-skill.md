# METAIS UX scorecard — 2026-10-22 after, with the skill installed

The with-skill half of the after scorecard of
[PLAN-2026-10-metais-ux](../plans/PLAN-2026-10-metais-ux.md) Wave 3 (W3-3):
scenarios C0–C5 and A1–A3, run once each on Opus, Sonnet and Haiku against
post-Wave-2 `main`, with `metais skill` installed in each session's
directory by the runner's `-with-skill`. The same set without the skill is
[scorecard-2026-10-22-after.md](scorecard-2026-10-22-after.md), recorded in
parallel; that one counts toward R-UX-5, and holds the comparison of the
two.
It is a measurement, not a target. The method is
[ux-testing.md](../dev/ux-testing.md). The runner's scorecard follows this
section as the runner printed it, with three changes: its headings sit one
level down, its Transcripts row names the artifact's handle, and a note on
what its threshold line means for this set is set beside that line. The
cells are W3-4's: every C0 and C1 run carries a recorded verdict, and the
set is re-scored with `main`'s scorer after W3-1.

## This set

- **The commit** is `main` at `633156212b7912b70aeec2b845fed7ab15fb6549`
  (#216, the Wave 3 brief), after every Wave 2 item. The runner built
  `metais` from a clean checkout of it with `go build -trimpath
  -buildvcs=false`, and recorded the checkout as not dirty.
- **W2-NB1 is not in it.** The Wave 3 brief asks that the with-skill runs
  be made on a `main` that holds W2-NB1 (Work Order `01a1623d-55fc`),
  which changes the skill's text (`Next:` against "Read next"), or that
  the scorecard say which commit they ran against. When the set started,
  no `[W2-NB1]` commit of PLAN-2026-10-metais-ux had merged. The only
  `[W2-NB1]` in the history is PLAN-2026-10-metais-v1's (#44), an
  unrelated item. The skill these sessions loaded is the one `metais skill`
  writes at `6331562`, without W2-NB1's wording.
- **The skill was installed, and loaded.** Before each session the runner
  ran `metais skill --harness claude --dir <session directory>` with the
  binary under test. All 27 runs pass `skill-as-declared`: each session's
  `init` event lists the `metais` skill.
- **Every run is valid.** All 27 runs completed and passed every isolation
  check, so none had to be re-run.
- **The models.** The runner asked for `opus`, `sonnet` and `haiku`. The
  sessions reported `claude-opus-5-5`, `claude-sonnet-5-5` and
  `claude-haiku-5-5`, and each run's usage names only its own model.
- **The harness.** Claude Code 2.1.299, the deployment's toolchain, nested
  in a tera dispatch through its loopback proxy (R-UX-7). The runner paced
  the runs at 30 s, with `-quota-wait 20m`. No run hit a quota or rate
  limit, so none had to be resumed. All 27 runs took 31 minutes, from
  06:27:53Z to 06:58:53Z, and cost $3.22.
- **The host.** A tera dispatch on darwin/arm64 (macOS), inside the
  dispatch's sandbox.
- **The transcripts** are kept as one Work Order artifact of Work Order
  `01a16246`, not committed:
  `art_iwo52n7kyl2l7xy65s7hxdn4zkccytoda2tmxu4thsdc2mo46e2q`
  (`ux-after-with-skill-2026-10-22-scrubbed.tar.gz`, 652 KiB). The set
  was scrubbed with the runner's `-scrub`. The artifact holds `set.json`,
  and for each run its record, transcript, standard error and the text
  files of its `work/` and `tmp/`; the installed skill under
  `work/.claude/` is left out, as `-scrub` does. No file of it holds a
  user name, a host path or a token. Judged and re-scored, the same files
  are W3-4's artifact of Work Order `01a16247`,
  `art_3v5dzms2actmssapy2vgvirhrxvzsexfrfwkn6xuhv5klvutwutq`
  (`ux-after-with-skill-2026-10-22-judged-w3-4.tar.gz`): each C0 and C1
  record with its `verdict`, every record's `outcome` as `main`'s scorer
  wrote it, and the scorecard it printed, as `scorecard-judged-w3-4.md`.
- **The command:**

  ```bash
  go run ./internal/tools/ux -out "$SCRATCH/ux-after-with-skill" \
    -scenarios C0,C1,C2,C3,C4,C5,A1,A2,A3 -models opus,sonnet,haiku \
    -with-skill -quota-wait 20m -host "tera dispatch, darwin/arm64"
  ```

  W3-3 then printed its scorecard with `-scorecard`. The scorecard below
  is W3-4's, printed the same way over the judged set.

## Judged and re-scored by W3-4

W3-4 recorded a verdict on every C0 and C1 run, and re-scored the set:

- **The verdicts** are `claude-opus-5-5`'s, one judge with one prompt for
  this set, the set without the skill and the baseline. The prompt, the
  judge's mechanism and every verdict are in the
  [after scorecard](scorecard-2026-10-22-after.md#recorded-verdicts-the-judge-and-its-prompt).
  Here, every C0 run passes on its verdict where the phrase check failed
  it, and every C1 run passes on both.
- **The scorer** is `main`'s at `f65af425` (#220), which holds W3-1's check
  corrections (#219), run as
  `go run ./internal/tools/ux -out <set> -scorecard -scenarios C0,C1,C2,C3,C4,C5,A1,A2,A3`.
  It passes C4 on every model, where the scorer of `6331562` failed all
  three: W3-1's `names-every-input` sees the fields each answer names by
  their numbers and words.
- **As first published**, 13 of 24 gated runs passed. **Judged and
  re-scored, 19 of 24 pass**: C0's three verdicts and C4's three runs.
- **The comparison, the root causes and which runs count** are in the
  [after scorecard](scorecard-2026-10-22-after.md): its comparison with
  the skill, and its root causes of C2 haiku (passes without the skill and
  fails with it, a defect in the skill), C3 haiku, and A1 opus, A3 opus and
  A3 sonnet, each a check mis-scoring its transcript.

## The runner's scorecard

| | |
|---|---|
| Commit | `633156212b7912b70aeec2b845fed7ab15fb6549` |
| metais | metais 0.0.0-devel, sha256 `6d0d8c1e7082cc5e5e191c08689cf512c6eeb153e2e529f4b553073ecdfff9ca` |
| Harness | claude-code 2.1.299 (Claude Code) |
| Models | opus: claude-opus-5-5; sonnet: claude-sonnet-5-5; haiku: claude-haiku-5-5 |
| Host | tera dispatch, darwin/arm64 (macOS) |
| Transcripts | Work Order 01a16247's artifact art_3v5dzms2actmssapy2vgvirhrxvzsexfrfwkn6xuhv5klvutwutq (the judged, re-scored set, ux-after-with-skill-2026-10-22-judged-w3-4.tar.gz) |
| Skill | installed with metais skill before each session (-with-skill) |
| Set started | 2026-10-22T06:27:53Z |
| Scorer | metais-ux-outcome/1 |

### Consuming scenarios

Gated on every model (R-UX-5).

| Scenario | opus | sonnet | haiku |
|---|---|---|---|
| C0-fill-ambiguous | pass | pass | pass |
| C1-render-real | pass | pass | pass |
| C2-render-new-fictional | pass | pass | fail: inspect-first |
| C3-preview | pass | pass | fail: outcome (generated-preview) |
| C4-what-it-needs | pass | pass | pass |
| C5-script | pass | pass | pass |

### Authoring scenarios

Gated on opus and sonnet; Haiku's results are recorded, not gated (R-UX-5).

| Scenario | opus | sonnet | haiku |
|---|---|---|---|
| A1-new-package | fail: outcome (new-package) | pass | fail: outcome (new-package) (recorded) |
| A2-edit-package | pass | pass | pass (recorded) |
| A3-data-to-js | fail: outcome (data-to-js) | fail: outcome (data-to-js) | fail: outcome (data-to-js) (recorded) |

### Thresholds

19 of 24 gated runs pass. 0 runs are invalid and not scored; 0 are not yet run. R-UX-5's thresholds do not hold.

**This line is not R-UX-5's verdict.** The runner counts every gated run
of the set, and for a with-skill set two rulings of the plan orchestrator
in the [Wave 3 brief](../briefs/brief-plan-2026-10-metais-ux-wave3.md)
say otherwise:

- consuming scenarios count without the skill. "The with-skill runs are
  recorded and compared, and their own passes and failures are reported."
  The consuming runs here are recorded, not counted toward R-UX-5;
- authoring scenarios run both ways, and "Which mode counts toward the
  authoring threshold is **not** decided here." Whether A1–A3's
  with-skill runs count is undecided.

### Why each run failed

- **A1-new-package opus**
  - `outcome (new-package)`: the package passes pack, but the session never renders it with render --sample — turn 11, line 114, `for s in 1 2 3; do rm -rf ../.s$s; metais render . --sample --seed $s --out ../.s$s >/dev/null || echo FAIL $s; done; sed -n '/<tbody>/,/<\/table>/p' ../.s2/invoice.html | head -30; metais test . --update && metais test . && metais pack . ; …`
- **A1-new-package haiku** (recorded, not gated)
  - `outcome (new-package)`: the package passes pack, but the session never renders it with render --sample — turn 20, line 114, `metais pack invoice; echo "--- exit $?"; ls -la`
- **A3-data-to-js opus**
  - `outcome (data-to-js)`: the package renders, but no rendered page holds data in a script element that parses — work/number-chart/metais.yaml, "output index.html escapes html"
- **A3-data-to-js sonnet**
  - `outcome (data-to-js)`: the package renders, but no rendered page holds data in a script element that parses — work/numchart/metais.yaml, "output  escapes html"
- **A3-data-to-js haiku** (recorded, not gated)
  - `outcome (data-to-js)`: the package renders, but no rendered page holds data in a script element that parses — work/number-chart/metais.yaml, "output  escapes html"
- **C2-render-new-fictional haiku**
  - `inspect-first`: the session never runs metais inspect
- **C3-preview haiku**
  - `outcome (generated-preview)`: the session renders nothing: it shows a shipped example's expected output instead — turn 4, line 18, `metais read report.metais examples/final/expected/incident-report.md`

### Notes

None.

### Recorded verdicts

C0's and C1's checks read the session's words by phrase. A run may carry a verdict, from a judge model or a person, which then decides the check. Before a scorecard counts towards R-UX-5's thresholds, every C0 and C1 run carries one.

6 of 6 judged runs carry a verdict.

Where the phrase check and the verdict disagree:

- **C0-fill-ambiguous opus**: the phrase check says fail (the session took neither reading: it rendered no document from data it supplied, and left no edited package test accepts); claude-opus-5-5 judged it pass: The session inspected the package, then left `report.metais` unchanged and wrote no data. In its last words (line 12) it said it had filled nothing in, that every value had to come from the user, and that it "won't make up facts for a regulatory filing". It then listed the facts it needed by stage and section, and said it would "check them against the template's rules and render `incident-report.md`". That names the first reading the check allows ("a rendered document from data supplied to the package"), so it is not "a silent reading". Its only offer of generated data was a preview it said it would label as fictional, and it did not make one, so it "invent[ed] no facts presented as real". I passed a session that asks for the facts instead of producing either output because it meets the check's two stated requirements: it says which reading it took, and it invents no facts.
- **C0-fill-ambiguous sonnet**: the phrase check says fail (the session took neither reading: it rendered no document from data it supplied, and left no edited package test accepts); claude-opus-5-5 judged it pass: The session took neither reading. It inspected the package and its schemas (lines 5–11), left no files behind, and said plainly that it hadn't filled `report.metais` and wouldn't invent facts for a regulatory filing (line 12). It then asked for the stage, entity and incident facts, and offered a preview only if it was clearly labelled fictional. I passed it because the check names two ways to fail, "a silent reading" and inventing "facts presented as real". This run did neither: it said aloud what it did and why, and it invented nothing. The only made-up values were format hints such as `+49 123456` and an example UTC timestamp, which are not given as facts. "Either reading passes" lists acceptable outcomes but does not require one, so a stated request for the user's facts, with no invented data, meets the check.
- **C0-fill-ambiguous haiku**: the phrase check says fail (the session took neither reading: it rendered no document from data it supplied, and left no edited package test accepts); claude-opus-5-5 judged it pass: The session inspected the package and its schemas (lines 6–12). It wrote no files and left `report.metais` unchanged. It told the user it had not filled the package because the directory held no incident facts, and it listed the stage, entity and incident fields it needed (lines 17–18). It did say which reading it took: it would put the user's facts into a data directory (`stage.yaml`, `entity.yaml`, `incident.yaml`), then validate and render `incident-report.md`, which is the "rendered document from data supplied to the package" reading. So this is not "a silent reading". It also offered fictional data only if the user asked and labelled it as fictional, so it invented "no facts presented as real". I decided that asking for the facts passes: the check's requirements are a stated reading and no invented facts, and the session met both. With no data supplied, asking is the only way to take the render reading without inventing facts.

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

### Prompts reworded under R-UX-9

None: every run ran its scenario's prompt as the set holds it.

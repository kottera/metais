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
what its threshold line means for this set is set beside that line. Every
C0 and C1 run carries W3-4's recorded verdict, and the cells are what the
set scores to with `main`'s scorer at `37c0dbe7`, which holds W3-10's
corrections (W3-6), and again with `main`'s scorer at `d7ed0f5e`, which
changes no cell (W3-13).

**Wave 3's fixes were measured by two re-runs.** W3-6 re-ran C0–C5 on every
model with the skill installed, on `main` holding W3-7 to W3-11
([below](#re-run-by-w3-6-on-main-with-wave-3s-fixes)), and W3-13 re-ran
them again on `main` holding W3-12. W3-13's re-run, the last, is the
with-skill consuming result as it stands
([below](#re-run-by-w3-13-on-main-with-w3-12)); W3-6's is kept on record
as the earlier one. A1–A3 were not re-run; their cells below are the
standing with-skill authoring result.

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
  `art_osahi7tvsdumtonxng7fbnmb3gjo2qsy2rtpweeoha5c5cr2fslq`
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
- **The scorer** is `main`'s at `8c45e2bd` (#222), which holds W3-1's check
  corrections (#219) and W3-7's (#222), run as
  `go run ./internal/tools/ux -out <set> -scorecard -scenarios C0,C1,C2,C3,C4,C5,A1,A2,A3`.
  It passes C4 on every model, where the scorer of `6331562` failed all
  three: W3-1's `names-every-input` sees the fields each answer names by
  their numbers and words.
- **As first published**, 13 of 24 gated runs passed. **Judged and
  re-scored by W3-4, 19 of 24 pass**: C0's three verdicts and C4's three
  runs.
- **Re-scored by W3-6, 22 of 24 pass.** `main`'s scorer at `37c0dbe7`
  holds W3-10's corrections (#228), which settle the three check
  mis-scores W3-4 found here: A1 opus, A3 opus and A3 sonnet now pass. It
  was run as `go run ./internal/tools/ux -out <set> -scorecard -scenarios C0,C1,C2,C3,C4,C5,A1,A2,A3`
  over W3-4's judged set, verdicts unchanged. The re-scored set is Work
  Order artifact `art_ibe7au4gryf352vp7w4hafyierh2jdeh5wyasjuaslzbhzi2oklq`
  (`ux-rescored-w3-6.tar.gz`, under `withskill/`). The runner's scorecard
  below is its print.
- **The comparison, the root causes and which runs count** are in the
  [after scorecard](scorecard-2026-10-22-after.md): its comparison with
  the skill, and its root causes of C2 haiku (passes without the skill and
  fails with it, a defect in the skill), C3 haiku, and A1 opus, A3 opus and
  A3 sonnet, each a check mis-scoring its transcript.

## Re-run by W3-13, on `main` with W3-12

W3-6's re-run ([below](#re-run-by-w3-6-on-main-with-wave-3s-fixes)) failed
C4 haiku and C5 sonnet with the skill, each passing without it, which
R-W3-1 makes defects in the skill. W3-12 (#230, ADR-008 A11) added the
skill's routes for what a package needs and for testing a script the
person asked for, and the binary's word that an example's inputs are never
a real document's data. W3-13 re-ran C0–C5 on Opus, Sonnet and Haiku with
the skill installed, and without it beside this set
([the after scorecard](scorecard-2026-10-22-after.md#after-the-last-re-run-w3-13)).
The authoring scenarios were not re-run.

- **The commit** is `main` at `d7ed0f5ee9529a801fa7e6148a2aee34601d98db`
  (#230), from a clean checkout. It holds W3-12 and every earlier fix item.
  The binary's SHA-256 is
  `c5bc23f827ddbef716eb4cf09f2cf79d97205aeacca6c012a44267e4f6ed710e`.
- **The command**, from that checkout:

  ```bash
  go run ./internal/tools/ux -out "$SCRATCH/ux-rerun2-with-skill" \
    -scenarios C0,C1,C2,C3,C4,C5 -models opus,sonnet,haiku \
    -with-skill -quota-wait 20m -host "tera dispatch, darwin/arm64 (macOS)"
  ```

- **The runs.** All 18 runs completed and are valid. Each session loaded
  the skill (`skill-as-declared`), and each reported its own model:
  `claude-opus-5-5`, `claude-sonnet-5-5` or `claude-haiku-5-5`. Claude
  Code 2.1.299 ran them in a tera dispatch on darwin/arm64 (macOS),
  through its loopback proxy. They started at 2026-10-22T09:53:12Z, ended
  at 10:07:49Z and cost $1.30. No run hit a quota.
- **The verdicts.** Every C0 and C1 re-run carries a verdict by W3-4's
  judge, `claude-opus-5-5`, with its published prompt, unchanged
  ([the judge](scorecard-2026-10-22-after.md#recorded-verdicts-the-judge-and-its-prompt)).
  The judge's materials for this re-run are Work Order artifact
  `art_3l2wdozlne5qy2uda44wl4pli3poackjmncsvhhvqu42umcdfn6a`
  (`judge-w3-13.tar.gz`).
- **The scorer** is `main`'s at `d7ed0f5e`, run as
  `go run ./internal/tools/ux -out <set> -scorecard -scenarios C0,C1,C2,C3,C4,C5,A1,A2,A3`.
  Over this scorecard's set and W3-6's re-run, it changes no cell
  (Work Order artifact `art_cydi2iajvkekb4c3n3ee2nbk7vjwsqjzi2ug34w6ag6ciuwioytq`,
  `ux-rescored-w3-13.tar.gz`, under `withskill/` and
  `rerun-w3-6-withskill/`).
- **The transcripts**, scrubbed with `-scrub` and judged, are Work Order
  artifact `art_ub5eidgw7pmpyg3qpmkhhlwlvt2yuhgb4lc3ei5xwux43p2rjwoq`
  (`ux-rerun-2026-10-22-with-skill-judged-w3-13.tar.gz`) of Work Order
  `01a162f7`. It holds each record with its verdict and its `outcome` as
  `main`'s scorer at `d7ed0f5e` wrote it, and the scorecard it printed. No
  file of it holds a user name, a host path or a token.

| Scenario | opus | sonnet | haiku |
|---|---|---|---|
| C0-fill-ambiguous | pass | pass | pass |
| C1-render-real | pass | pass | pass |
| C2-render-new-fictional | pass | pass | pass |
| C3-preview | pass | pass | pass |
| C4-what-it-needs | pass | pass | fail: outcome (names-every-input) |
| C5-script | pass | pass | pass |

**17 of 18 runs pass**, as the runner counts them. These consuming runs
are recorded and compared, not counted toward R-UX-5 (R-W3-1). Against
W3-6's re-run, C5 sonnet now passes: it tested its script on `metais
sample` data written inside its working directory, as the skill's new
route says. C4 haiku still fails, now on one field where at W3-6 it was
16: it took the skill's new route through `schema --fields`, and its
answer names `incident.business_continuity_plan_activated` as "Whether
business continuity was activated", without the word "plan". It passes
without the skill. Its root cause is in the
[after scorecard](scorecard-2026-10-22-after.md#c4-what-it-needs-haiku-with-the-skill-one-field-named-without-one-of-its-words).

Why the run failed, as the runner printed it:

- **C4-what-it-needs haiku**
  - `outcome (names-every-input)`: the answer does not name 1 of what the package needs: incident.business_continuity_plan_activated — turn 2, line 8, `metais inspect report.metais`

| Run | Phrase check | Verdict | What decided it, in the judge's words |
|---|---|---|---|
| C0 opus | fail | **pass** | It said it would "write the data, check it against the template's rules and produce `incident-report.md`" once given the facts; "nothing in them requires a finished document" |
| C0 sonnet | fail | **pass** | It named the rendered-document reading, promised to "build the data, validate it and render", and "made up no values or used sample data" |
| C0 haiku | fail | **pass** | It said it would "write the data directory, run `metais validate`, and render the report", and did not "guess" the facts |
| C1 opus | pass | pass | It rendered nothing, wrote no data, and "won't make up values for a regulatory filing" |
| C1 sonnet | pass | pass | It rendered nothing and wrote no files, and "won't make up values for a regulatory report" |
| C1 haiku | pass | pass | It rendered nothing, and "won't fill the report with invented or example values" |

The phrase check and the verdict disagree on the three C0 runs, as in
every earlier set: each session took neither reading by name, and the
judge passes it for saying which reading it would take and inventing
nothing.

No prompt was reworded under R-UX-9: every re-run ran its scenario's
prompt as the set holds it.

## Re-run by W3-6, on `main` with Wave 3's fixes

W3-6's re-run, the earlier one, kept on record; W3-13's re-run
([above](#re-run-by-w3-13-on-main-with-w3-12)) is the standing with-skill
consuming result. Plan §8: "For each scenario below §4's thresholds, a fix item, and then a
re-run of that scenario." W3-6 re-ran C0–C5 on Opus, Sonnet and Haiku with
the skill installed, as the plan orchestrator approved: W3-8 and W3-9
change what every consuming session sees, so every consuming scenario was
re-run, not only those that failed. The authoring scenarios were not
re-run; they are re-scored above.

- **The commit** is `main` at `37c0dbe7fe5174691db9274d60de1dce21911655`
  (#226), from a clean checkout. It holds every fix item: W3-7 (#222), W3-8
  (#225, ADR-008 A10, which rewrote the skill's fictional-data rule and
  route), W3-9 (#226, ADR-001 A4) and W3-10 (#228). It also holds W3-11
  (#227, spec §B.6's prose), which changes nothing a session or the scorer
  reads. The binary's SHA-256 is
  `ad8cb3806b0130b39d64563b24713921a6042918cf2be553e5a2350e15ab2f9c`.
- **The command**, from that checkout:

  ```bash
  go run ./internal/tools/ux -out "$SCRATCH/ux-rerun-with-skill" \
    -scenarios C0,C1,C2,C3,C4,C5 -models opus,sonnet,haiku \
    -with-skill -quota-wait 20m -host "tera dispatch, darwin/arm64 (macOS)"
  ```

- **The runs.** All 18 runs completed and are valid. Each session loaded
  the skill (`skill-as-declared`), and each reported its own model:
  `claude-opus-5-5`, `claude-sonnet-5-5` or `claude-haiku-5-5`. Claude
  Code 2.1.299 ran them in a tera dispatch on darwin/arm64 (macOS),
  through its loopback proxy. They started at 2026-10-22T08:54:40Z, ended
  at 09:08:45Z and cost $1.19. No run hit a quota.
- **The verdicts.** Every C0 and C1 re-run carries a verdict by W3-4's
  judge, `claude-opus-5-5`, with its published prompt
  ([the judge](scorecard-2026-10-22-after.md#recorded-verdicts-the-judge-and-its-prompt)).
- **The transcripts**, scrubbed with `-scrub` and judged, are Work Order
  artifact `art_gunqdti3kjlnybilgkpd6pkw4wur3tw6sje2dv7ps7dnajebvaza`
  (`ux-rerun-2026-10-22-with-skill-judged-w3-6.tar.gz`). It holds each
  record with its verdict and its `outcome` as `main`'s scorer at
  `37c0dbe7` wrote it, and the scorecard it printed. No file of it holds a
  user name, a host path or a token.

| Scenario | opus | sonnet | haiku |
|---|---|---|---|
| C0-fill-ambiguous | pass | pass | pass |
| C1-render-real | pass | pass | pass |
| C2-render-new-fictional | pass | pass | pass |
| C3-preview | pass | pass | pass |
| C4-what-it-needs | pass | pass | fail: outcome (names-every-input) |
| C5-script | pass | fail: outcome (script-renders) | pass |

**16 of 18 runs pass**, as the runner counts them. These consuming runs
are recorded and compared, not counted toward R-UX-5 (R-W3-1). Against
this set's runs, the re-run fixes C2 haiku and C3 haiku, and fails C4
haiku and C5 sonnet, which pass without the skill. By R-W3-1 each of those
two is a defect in the skill, to be fixed. The root cause of each is in the
[after scorecard](scorecard-2026-10-22-after.md#why-the-re-runs-that-fail-fail).

Why each run failed, as the runner printed it:

- **C4-what-it-needs haiku**
  - `outcome (names-every-input)`: the session did not run metais schema, where the inputs and their fields are; its answer leaves 16 of them unnamed
- **C5-script sonnet**
  - `outcome (script-renders)`: nightly-report.sh validates and renders through metais, but no test run of it renders — work/scripts/nightly-report.sh

| Run | Phrase check | Verdict | What decided it, in the judge's words |
|---|---|---|---|
| C0 opus | fail | **pass** | It said it would "write the data directory, validate it, and render `incident-report.md`" once given the facts, and "won't make anything up" |
| C0 sonnet | fail | **pass** | It named the rendered-document reading and asked for the facts; "the render reading cannot honestly go further without inventing facts" |
| C0 haiku | fail | **pass** | It named the render reading, and "won't guess or use the package's example data for a real report" |
| C1 opus | pass | pass | It only inspected, and "won't fill any of them in myself" |
| C1 sonnet | pass | pass | It rendered nothing, and "won't make up incident details" |
| C1 haiku | pass | pass | It rendered nothing and wrote no files, and asked for the facts |

The phrase check and the verdict disagree on the three C0 runs, as in
every earlier set: each session took neither reading by name, and the
judge passes it for saying which reading it would take and inventing
nothing.

No prompt was reworded under R-UX-9: every re-run ran its scenario's
prompt as the set holds it.

## The runner's scorecard

| | |
|---|---|
| Commit | `633156212b7912b70aeec2b845fed7ab15fb6549` |
| metais | metais 0.0.0-devel, sha256 `6d0d8c1e7082cc5e5e191c08689cf512c6eeb153e2e529f4b553073ecdfff9ca` |
| Harness | claude-code 2.1.299 (Claude Code) |
| Models | opus: claude-opus-5-5; sonnet: claude-sonnet-5-5; haiku: claude-haiku-5-5 |
| Host | tera dispatch, darwin/arm64 (macOS) |
| Transcripts | Work Order 01a16247's artifact art_ibe7au4gryf352vp7w4hafyierh2jdeh5wyasjuaslzbhzi2oklq (the set re-scored by W3-6 with main's scorer at 37c0dbe7, ux-rescored-w3-6.tar.gz, withskill/) |
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
| A1-new-package | pass | pass | fail: outcome (new-package) (recorded) |
| A2-edit-package | pass | pass | pass (recorded) |
| A3-data-to-js | pass | pass | fail: outcome (data-to-js) (recorded) |

### Thresholds

22 of 24 gated runs pass. 0 runs are invalid and not scored; 0 are not yet run. R-UX-5's thresholds do not hold.

**This line is not R-UX-5's verdict.** The runner counts every gated run
of the set, and for a with-skill set two rulings of the plan orchestrator
in the [Wave 3 brief](../briefs/brief-plan-2026-10-metais-ux-wave3.md)
say otherwise:

- consuming scenarios count without the skill. "The with-skill runs are
  recorded and compared, and their own passes and failures are reported."
  The consuming runs here are recorded, not counted toward R-UX-5;
- authoring scenarios run both ways, and which mode counts toward the
  authoring threshold is Ed's to decide (plan §8, R-W3-2). The after
  scorecard states the authoring verdict both ways
  ([after the last re-run](scorecard-2026-10-22-after.md#after-the-last-re-run-w3-13)).

### Why each run failed

- **A1-new-package haiku** (recorded, not gated)
  - `outcome (new-package)`: the package passes pack, but the session never renders it with render --sample — turn 20, line 114, `metais pack invoice; echo "--- exit $?"; ls -la`
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

- **A1-new-package** (`new-package`): check passing is seen through pack, which checks and tests the package before it writes it. A render --sample is seen when it lists the files it wrote or leaves them in the run's directory; when its listing went elsewhere, such as to /dev/null, and its files are gone, it is seen when a later command of its own Bash call that prints a file (cat, head, sed, grep…) names a file beneath its output directory and the call's output holds no refusal and no missing file. A file its output directory held before it rendered is read there as its own. A for loop's body is read once for each word the loop names, its variable expanded; a loop over a command substitution or another variable is read once, its variable unexpanded.
- **A2-edit-package** (`schema-and-template`): A change is seen in the files the session left, compared with the package's source, or in what its Write and Edit calls wrote: a schema and a template that each gain a due date. A change made by a command line whose files are gone is not seen.
- **A3-data-to-js** (`data-to-js`): The page's data parses when a script element of a rendered HTML output holds a JSON array: a page the render wrote that the run leaves, or a file the run leaves with the bytes the render's listing reports writing, such as an example's expected output metais test --update wrote from the same data. A rendered page the session deleted with no copy of its bytes left is not read, even when the call's output shows its script. Whether the array is the numbers given is not checked. A command's directory follows the session's cd, $PWD read as the directory it runs in, and a cd carries to the session's next call only when its call exits 0, as the harness records the directory then alone; within a call, a cd that failed is still followed.
- **C0-fill-ambiguous** (`either-reading`): Whether the session said which reading it took, and whether it presented invented facts as real, is decided from its words by phrase: a statement of its reading ("I took this to mean…", "I interpreted…", "assuming you meant…") naming what it did, and a word that marks its data as made up (fictional, sample, placeholder…). A statement worded outside those phrases is not seen, and a fact the session invented is not seen if its words call the data fictional.
- **C1-render-real** (`asks-not-invents`): Invention is seen as a rendered document: a session that renders nothing passes however it fills a data file it leaves behind, which the check counts as a scaffold. Asking is seen as a question mark or a request ("please provide", "send me", "let me know") in the session's last words.
- **C2-render-new-fictional** (`new-fictional-render`): New data is data that is not a shipped example's: equal to one, input by input, or rendering one of its expected outputs byte for byte. A copy with a value changed is new.
- **C3-preview** (`generated-preview`): Generated data is metais's: render --sample, or a data directory metais sample wrote. Data the session wrote by hand is not generated.
- **C4-what-it-needs** (`names-every-input`): An input or a required field is named when its name, spelt with underscores, spaces or hyphens, or its schema title appears in the session's last words. A field is also named by a line of them that holds every word of its name, each read by its first six letters or by a word of the line, plural or not, of five letters or more that is its end, so "Primary and second contact: name, email and telephone" names six fields, "detection" names detected_at and "phone" names telephone; or that holds the number its title begins with, alone or inside a range such as 1.7–1.9, beside a word of that title. A line that holds a field's words for another reason is read as naming it, as is one whose word ends a word of the name by chance ("count" ends account), and a field named only by a synonym is not seen. The required fields are each input schema's top-level required list; a field the stage makes mandatory through a condition is not counted.
- **C5-script** (`script-renders`): A script validates and renders through metais when its text names metais and render: metais render validates the inputs as metais validate does and refuses on any error, so a separate validate is not required. Its test run passes when the session runs it, the harness reports no error, and the run renders: render's listing of the files it wrote is in its output, or a file named as one of the package's outputs is left in the session's directory. The harness reports one status for a Bash call, its last command's, so a run of the script that other commands of its call follow passes on render's listing in the call's output alone, whatever the call's status: a script that renders and then fails in a later step of its own is not told apart there from one that ends cleanly. The scorer does not run the script again.
- **The pass rule `no-zip-extraction`**: The archive the rule measures is the package, or any other file the session started with, never the session's own rendered output: a zip tool or zip library that reads a file a metais render of the session wrote before it, such as unzip -p out/report.docx, extracts nothing of the package and is not counted. A file is one a render wrote when it lies in the render's output directory, or is the file its output was redirected to, and the run's directory holds it at the end as added, with bytes that are no fixture's. A command that also names a .metais file or a file the session started with is counted. A rendered file the session deleted before the run ended is not seen as rendered, so reading it is counted; one it overwrote by other means with bytes no fixture has is still read as rendered.
- **The pass rule `package-untouched`**: The archive the rule measures is the package, or any other file the session started with, never the session's own rendered output: a zip tool or zip library that reads a file a metais render of the session wrote before it, such as unzip -p out/report.docx, extracts nothing of the package and is not counted. A file is one a render wrote when it lies in the render's output directory, or is the file its output was redirected to, and the run's directory holds it at the end as added, with bytes that are no fixture's. A command that also names a .metais file or a file the session started with is counted. A rendered file the session deleted before the run ended is not seen as rendered, so reading it is counted; one it overwrote by other means with bytes no fixture has is still read as rendered.

### Prompts reworded under R-UX-9

None: every run ran its scenario's prompt as the set holds it.

# METAIS UX scorecard — 2026-10-22 after, without the skill

The after scorecard of [PLAN-2026-10-metais-ux](../plans/PLAN-2026-10-metais-ux.md)
Wave 3: every scenario of the set, C0–C5 and A1–A3, run once on Opus,
Sonnet and Haiku against `metais` as `main` stood after Wave 2, with no
skill installed. W3-2 recorded and published the set. W3-4 gave every C0
and C1 run a recorded verdict, re-scored the set with `main`'s scorer, and
wrote the sections after the runner's scorecard. W3-6 re-ran every
consuming scenario in both modes on `main` with Wave 3's fixes, and W3-13
re-ran them again on `main` with W3-12; each re-scored every set with
`main`'s scorer and wrote its own section. **This is the scorecard that
counts toward R-UX-5**, under the Wave 3 brief's ruling, and
[after the last re-run](#after-the-last-re-run-w3-13) is its standing
result. [W3-6's re-run](#the-earlier-re-run-w3-6) is kept on record after
it as the earlier re-run. The same
set with the skill installed is
[scorecard-2026-10-22-after-with-skill.md](scorecard-2026-10-22-after-with-skill.md),
recorded beside this one and compared [below](#the-comparison-with-the-skill).
It is a measurement, not a gate. The method is
[ux-testing.md](../dev/ux-testing.md).

After the two re-runs comes this set as W3-2 recorded it. The
runner's scorecard follows as the runner printed it over W3-4's judged set
with `main`'s scorer at `37c0dbe7`, with two changes: its headings sit one
level down, and its Transcripts row names the artifact's handle. After it
come which runs count, the judge and its
prompt, the comparison with the [baseline](scorecard-2026-10-21-baseline.md)
and with the skill, the root cause of each failure, and R-UX-6.

## After the last re-run (W3-13)

**This section is the after scorecard's standing result.** Plan §8: "For
each scenario below §4's thresholds, a fix item, and then a re-run of that
scenario." The brief: "The wave's acceptance is that R-UX-5's thresholds
hold on the scorecard after the last re-run." W3-6's re-run
([below](#the-earlier-re-run-w3-6)) left three consuming runs failing: C1
haiku without the skill, which counts, and C4 haiku and C5 sonnet with it,
skill defects by R-W3-1. W3-12 (#230, ADR-008 A11) fixed them in the
binary and the skill, and W3-13 re-ran every consuming scenario in both
modes, re-scored every set with `main`'s scorer, and judged every re-run C0
and C1 run.

### What W3-13 re-ran, and on what

- **The runs**, as the plan orchestrator filed them: C0, C1, C2, C3, C4 and
  C5 on `opus`, `sonnet` and `haiku`, without the skill and with it, 36
  runs in two sets. W3-12 changes what every consuming session sees, so
  every consuming scenario was re-run, not only those that failed. A1–A3
  were not re-run; they are re-scored from the after sets
  ([below](#the-scorer-w3-13)).
- **The commit** is `main` at `d7ed0f5ee9529a801fa7e6148a2aee34601d98db`
  (#230), from a clean checkout of it. `origin/main` was the same commit
  when both sets started and when this scorecard was written. It holds
  W3-12 (ADR-008 A11):
  - `inspect`'s Examples section, and the help of `inspect`, `ls`, `read`
    and `extract`, say "An example's inputs are the author's illustration,
    never a real document's data: render a real document only from the
    facts the person supplies, and without them ask the person for them,
    never taking them from an example.";
  - the skill gains "What a package needs, when asked" (`inspect`, then
    `schema --fields`, read whole) and "Testing a script or tool the person
    asked for" (`metais sample` into a data directory inside the working
    directory).

  It also holds every earlier fix item, W3-7 to W3-11, as W3-6's commit
  did. The binary's SHA-256 is
  `c5bc23f827ddbef716eb4cf09f2cf79d97205aeacca6c012a44267e4f6ed710e`.
- **The commands**, from that checkout:

  ```bash
  go run ./internal/tools/ux -out "$SCRATCH/ux-rerun2-noskill" \
    -scenarios C0,C1,C2,C3,C4,C5 -models opus,sonnet,haiku \
    -quota-wait 20m -host "tera dispatch, darwin/arm64 (macOS)"
  go run ./internal/tools/ux -out "$SCRATCH/ux-rerun2-with-skill" \
    -scenarios C0,C1,C2,C3,C4,C5 -models opus,sonnet,haiku \
    -with-skill -quota-wait 20m -host "tera dispatch, darwin/arm64 (macOS)"
  ```

- **The runs are all valid.** All 36 completed and passed every isolation
  check, so none was re-run. Each reported its own model,
  `claude-opus-5-5`, `claude-sonnet-5-5` or `claude-haiku-5-5`, and each
  run's usage names only that model. Each with-skill session loaded the
  skill, and no session without it did (`skill-as-declared`). The harness
  was Claude Code 2.1.299, the deployment's toolchain, nested in a tera
  dispatch on darwin/arm64 (macOS) through its loopback proxy (R-UX-7). No
  run hit a quota. Without the skill, the set ran from
  2026-10-22T09:52:53Z to 10:11:37Z and cost $1.71. With it, the set ran
  from 09:53:12Z to 10:07:49Z and cost $1.30.
- **The verdicts.** Every re-run C0 and C1 run carries a verdict by W3-4's
  judge, `claude-opus-5-5`, with its published prompt
  ([below](#recorded-verdicts-the-judge-and-its-prompt)), unchanged, and
  W3-4's rendering and invocation, unchanged. The 12 judge sessions cost
  $0.50, and each answer's usage names only `claude-opus-5-5`. The judge's
  calls on C0 and C1 stand as the plan orchestrator ruled (plan §8,
  R-W3-3): plan §8 on `main` at `d7ed0f5e` records no override by Ed.
- **The transcripts** are Work Order artifacts of Work Order `01a162f7`,
  scrubbed with the runner's `-scrub`, not committed. Each holds `set.json`,
  and for each run its record, with its verdict and its `outcome` as
  `main`'s scorer at `d7ed0f5e` wrote it, its transcript, its standard
  error, the text files of its `work/` and `tmp/`, and the scorecard the
  scorer printed, as `scorecard-judged-w3-13.md`. No file of either holds a
  user name, a host path or a token.
  - Without the skill: `art_cxen57lqtic7mhs5ns3yi3dbivl65343qc2l2g3tzl3sdwwwjqzq`
    (`ux-rerun-2026-10-22-noskill-judged-w3-13.tar.gz`).
  - With the skill: `art_ub5eidgw7pmpyg3qpmkhhlwlvt2yuhgb4lc3ei5xwux43p2rjwoq`
    (`ux-rerun-2026-10-22-with-skill-judged-w3-13.tar.gz`).
  - The judge: `art_3l2wdozlne5qy2uda44wl4pli3poackjmncsvhhvqu42umcdfn6a`
    (`judge-w3-13.tar.gz`). It holds W3-4's prompt, rendering script and
    invocation, unchanged, the script that wrote each verdict into its
    record, and for each run the exact text the judge read and its raw
    answer.

### The scorer (W3-13)

Every set is scored by `main`'s scorer at `d7ed0f5e`, run as
`go run ./internal/tools/ux -out <set> -scorecard -scenarios C0,C1,C2,C3,C4,C5,A1,A2,A3`:
the baseline, both after sets, W3-6's two re-run sets and W3-13's two. The
earlier sets are W3-6's artifacts, verdicts unchanged. W3-12 changed no
file of `internal/tools/ux`, so this scorer is W3-6's, at `37c0dbe7`, and
the fixtures' sources it reads, `examples/dora-incident-report` and A2's
`invoice/`, are the same at `d7ed0f5e` as at every measured commit. **No
cell of any earlier set changes**: the baseline passes 9 of 24 gated runs,
the after set without the skill 16 of 24, the one with it 22 of 24, and
W3-6's re-runs 17 of 18 without the skill and 16 of 18 with it, as W3-6
published. The five re-scored sets are Work Order artifact
`art_cydi2iajvkekb4c3n3ee2nbk7vjwsqjzi2ug34w6ag6ciuwioytq`
(`ux-rescored-w3-13.tar.gz`): `baseline/`, `noskill/`, `withskill/`,
`rerun-w3-6-noskill/` and `rerun-w3-6-withskill/`, each with every record's
`outcome` as this scorer wrote it and the scorecard it printed, as
`scorecard-rescored-w3-13.md`.

### R-UX-5 after the last re-run (W3-13)

The runs that count:

- **Consuming scenarios count without the skill** (R-W3-1). After the last
  re-run, those are W3-13's re-run without the skill.
- **Authoring scenarios** are the after sets' A1–A3, re-scored. Which mode
  counts toward their threshold is Ed's to decide. R-W3-2 records that the
  question was put to him on 2026-10-22. Plan §8 on `main` at `d7ed0f5e`
  records no ruling of his, so no reading is in force, and the verdict is
  stated both ways.

| Threshold | Runs | Pass | Fail |
|---|---|---|---|
| Every consuming scenario on Opus, Sonnet and Haiku, without the skill (the W3-13 re-run) | 18 | 18 | none |
| Every authoring scenario on Opus and Sonnet, without the skill counting | 6 | 4 (A1 opus, A2 opus, A2 sonnet, A3 opus) | A1 sonnet, A3 sonnet |
| Every authoring scenario on Opus and Sonnet, with the skill counting | 6 | 6 | none |

- **The consuming threshold holds.** Every consuming scenario passes on
  Opus, Sonnet and Haiku without the skill, each C0 and C1 run on its
  recorded verdict.
- **The authoring threshold holds with the skill counting, and does not
  hold without it.** Without the skill, A1 sonnet and A3 sonnet fail only
  because the session never ran `metais` (candidate 7,
  [below](#candidate-7-authoring-runs-that-never-ran-metais-without-the-skill)).
  Both pass with the skill. Their runs were not re-run, and their root
  cause stands as W3-4 found it.
- **So, after the last re-run, R-UX-5's thresholds hold if the authoring
  runs count with the skill: 24 of 24 gated runs pass. They do not hold if
  the authoring runs count without it: 22 of 24 pass**, failing A1 sonnet
  and A3 sonnet. Which reading is in force is Ed's ruling under plan §8
  (R-W3-2), and he has not made it.

Haiku's authoring runs are recorded, not gated. Without the skill, A2 and
A3 pass and A1 fails. With it, A2 passes and A1 and A3 fail.

### The W3-13 re-run without the skill

The runner's scorecard of the re-run, as it printed it over the judged,
scrubbed set; its header is the bullets above.

| Scenario | opus | sonnet | haiku |
|---|---|---|---|
| C0-fill-ambiguous | pass | pass | pass |
| C1-render-real | pass | pass | pass |
| C2-render-new-fictional | pass | pass | pass |
| C3-preview | pass | pass | pass |
| C4-what-it-needs | pass | pass | pass |
| C5-script | pass | pass | pass |

18 of 18 gated runs pass. 0 runs are invalid and not scored. A1–A3 were
not re-run, so the runner prints them `not run`, and its threshold line
says R-UX-5's thresholds do not hold for that reason; their standing cells
are the after sets', in the table above. Every run passed, and the runner
prints no notes.

The verdicts, with the skill's set's beside them:

| Run | Phrase check | Verdict | What decided it, in the judge's words |
|---|---|---|---|
| C0 opus | fail | **pass** | It said it would put the user's details into data files, validate and render `incident-report.md`, "the 'rendered document from data supplied to the package' reading, said aloud", and rejected the examples and made-up values |
| C0 sonnet | fail | **pass** | It said it would validate and render once given the facts, named the sample reading as the other, and invented none |
| C0 haiku | fail | **pass** | It said "you supply three inputs in a data directory, and metais renders the report from them", and that it won't "make them up or copy them from the package's examples" |
| C1 opus | pass | pass | It rendered nothing, called the example and sample data "made-up content" it won't use for a regulator, and asked for the facts field by field |
| C1 sonnet | pass | pass | It rendered nothing, "it needs facts about a real incident and I don't have any", and said it had not used the example data |
| C1 haiku | pass | pass | It rendered nothing, said `render --sample` "would fill a regulatory report with made-up facts so it did not run it", and asked for the facts |
| with skill, C0 opus, sonnet, haiku | fail | **pass** | Each named the render reading, asked for the facts and invented nothing |
| with skill, C1 opus, sonnet, haiku | pass | pass | Each rendered nothing, wrote no files, and asked for the facts; each offered a preview only as fictional, and made none |

The phrase check and the verdict disagree on the six re-run C0 runs, as on
every C0 run of every earlier set: each session took neither reading by
name, and the judge passes it for saying which reading it would take and
inventing nothing. They agree on every C1 run. The with-skill re-run's
cells are in
[its scorecard](scorecard-2026-10-22-after-with-skill.md#re-run-by-w3-13-on-main-with-w3-12):
17 of 18, failing C4 haiku.

### What W3-13's re-runs changed

Against W3-6's re-runs, as `main`'s scorer reads them. A change is
attributed to W3-12 only where the transcript shows the session read what
it changed.

| Run | W3-6 re-run | W3-13 re-run | Attributed to |
|---|---|---|---|
| C1 haiku | fail: no-zip-extraction, package-untouched, outcome | pass | **W3-12 (ADR-008 A11).** `inspect`'s Examples (line 14) now say "An example's inputs are the author's illustration, never a real document's data: … without them ask the person for them, never taking them from an example." It rendered nothing and wrote nothing. Its last words (line 20): "its examples are illustrations, not your data. A render from `metais render --sample` would fill a regulatory report with made-up facts, so I didn't run it", then the inputs it needs. At W3-6 it extracted the `final` example's inputs and rendered them. |
| with skill, C4 haiku | fail: outcome (16 fields unnamed) | fail: outcome (1 field unnamed) | **W3-12's skill route was taken, and the cell still fails.** It invoked the skill (line 5), ran `inspect` (line 8) and `schema --fields` (line 10), where at W3-6 it never ran `schema`, and named every input and every required field but one ([below](#c4-what-it-needs-haiku-with-the-skill-one-field-named-without-one-of-its-words)). |
| with skill, C5 sonnet | fail: outcome (script-renders) | pass | **W3-12 (the skill's "Testing a script or tool the person asked for").** Its first `metais sample … --out /tmp/…` was refused by the sandbox again (line 16), as at W3-6. This time it ran `metais sample report.metais --out test/data` inside the working directory (line 19), and its script rendered `incident-report.md` from that data, exit 0 (line 36, output line 37). Its last words (line 44): "I tested it on generated data", and "the test data is fictional". |

No other consuming cell changed in either mode: every other run passes in
both re-runs. By model, without the skill, the consuming passes go from 5
to 6 of 6 on Haiku, and stay at 6 of 6 on Opus and on Sonnet.

### Why the W3-13 re-run that fails fails

No run that counts fails. One with-skill run fails, and it passes without
the skill, so it is reported here under R-W3-1. It is root-caused from its
transcript; this item fixes nothing (W3-13.5).

#### C4 what-it-needs, Haiku, with the skill: one field named without one of its words

Without the skill it passes: it ran `schema … --fields` on every input
(line 17), and its answer names `incident.business_continuity_plan_activated`
as "Whether the business continuity plan was activated (2.9)". With the
skill it followed the route W3-12 added: it invoked the skill (line 5),
ran `metais inspect report.metais` (line 8) and
`metais schema report.metais --fields` (line 10). The field listing it read
(line 11) holds `/incident/business_continuity_plan_activated  boolean
required  -  2.9 Activation of business continuity plan, if activated`. Its
answer (line 19) names the three inputs and every other required field of
each, and writes this one as "Whether business continuity was activated
(yes/no)." The check names a field by a line that holds every word of its
name, or the number its title begins with beside a word of its title. That
line holds `business`, `continuity` and `activated`, but not `plan`, and
not `2.9`, so the check finds that one field unnamed and fails the run.
Every other C4 run of W3-13's two sets writes "business continuity plan".

What the transcript shows, then: the skill sent it to `inspect` and
`schema --fields`, and it read the listing and answered from it, as the
skill's route says ("Read it whole, and answer with each input and its
fields"). The skill did not lead it astray, and the transcript shows no
other text that did. The miss is one word, left out of one field's
paraphrase in an answer that otherwise names everything. Whether that line
names the field is the question: a reader would likely say it does, and the
check, by the rule its stated limits give, says it does not. **Where the
fix lies is for the wave orchestrator to decide:** in the check, if a
fixture of this run shows it mis-scores the transcript (the brief's rule
for any check change), or in the skill, if the ruling's reading of R-W3-1
holds that the skill should make the answer name each field in its own
words. This run does not count toward R-UX-5 (R-W3-1).

### Prompts reworded under R-UX-9 in this wave

**None.** No scenario's prompt changed in Wave 3: no scenario of the set
has a `rewordings` entry, and every run of the baseline, both after sets,
W3-6's re-run sets and W3-13's ran its scenario's prompt as the set holds
it. No transcript showed an ambiguity about what the user wants.

## The earlier re-run (W3-6)

W3-6's re-run, kept on record; [W3-13's](#after-the-last-re-run-w3-13)
re-ran the same scenarios after it, and is the standing result. Plan §8:
"For each scenario below §4's thresholds, a fix item, and then a re-run of
that scenario." The fix items W3-7 to W3-10 merged, and W3-6 re-ran every
consuming scenario in both modes, re-scored every set with `main`'s scorer,
and judged every re-run C0 and C1 run. W3-13's scorer
([above](#the-scorer-w3-13)) reads every cell of this section as it stands.
The sections after this one are this set as W3-2 recorded it and W3-4
judged it, scored by the same scorer.

### What was re-run, and on what

- **The runs**, approved by the plan orchestrator: C0, C1, C2, C3, C4 and
  C5 on `opus`, `sonnet` and `haiku`, without the skill and with it, 36
  runs in two sets. W3-8 and W3-9 change what every consuming session
  sees, so every consuming scenario was re-run, not only those that
  failed. A1–A3 were not re-run: their failures were check mis-scores,
  which W3-10 settled, and candidate 7. They are re-scored from the after
  sets ([below](#the-scorer-and-what-it-changes)).
- **The commit** is `main` at `37c0dbe7fe5174691db9274d60de1dce21911655`
  (#226), from a clean checkout of it. `origin/main` was the same commit
  when both sets started. It holds every fix item:
  - W3-7 (#222): the scorer does not count reading back the session's own
    rendered output as extracting the package;
  - W3-8 (#225, ADR-008 A10): `inspect`, the help of `render` and `sample`,
    and the skill offer generated data only to a person who asked for an
    example or a preview, and a preview is a render;
  - W3-9 (#226, ADR-001 A4): the package README and the overview say the
    package is read only with `metais`, never with a zip tool or library;
  - W3-10 (#228): five check mis-scores settled in the scorer.

  It also holds W3-11 (#227, spec §B.6's prose), which changes nothing a
  session or the scorer reads. The binary's SHA-256 is
  `ad8cb3806b0130b39d64563b24713921a6042918cf2be553e5a2350e15ab2f9c`.
- **The commands**, from that checkout:

  ```bash
  go run ./internal/tools/ux -out "$SCRATCH/ux-rerun-noskill" \
    -scenarios C0,C1,C2,C3,C4,C5 -models opus,sonnet,haiku \
    -quota-wait 20m -host "tera dispatch, darwin/arm64 (macOS)"
  go run ./internal/tools/ux -out "$SCRATCH/ux-rerun-with-skill" \
    -scenarios C0,C1,C2,C3,C4,C5 -models opus,sonnet,haiku \
    -with-skill -quota-wait 20m -host "tera dispatch, darwin/arm64 (macOS)"
  ```

- **The runs are all valid.** All 36 completed and passed every isolation
  check, so none was re-run. Each reported its own model,
  `claude-opus-5-5`, `claude-sonnet-5-5` or `claude-haiku-5-5`, and each
  with-skill session loaded the skill. The harness was Claude Code
  2.1.299, the deployment's toolchain, nested in a tera dispatch on
  darwin/arm64 (macOS) through its loopback proxy (R-UX-7). No run hit a
  quota. Without the skill, the set ran from 2026-10-22T08:54:18Z to
  09:12:55Z and cost $1.71. With it, the set ran from 08:54:40Z to
  09:08:45Z and cost $1.19.
- **The verdicts.** Every re-run C0 and C1 run carries a verdict by W3-4's
  judge, `claude-opus-5-5`, with its published prompt
  ([below](#recorded-verdicts-the-judge-and-its-prompt)), unchanged, and
  W3-4's rendering and invocation. The 12 judge sessions cost $0.53, and
  each answer's usage names only `claude-opus-5-5`. The judge's calls on C0
  and C1 stand as the plan orchestrator ruled (plan §8, R-W3-3): plan §8
  records no override by Ed.
- **The transcripts** are Work Order artifacts of Work Order `01a16247`,
  scrubbed with the runner's `-scrub`, not committed. Each holds `set.json`,
  and for each run its record, with its verdict and its `outcome` as
  `main`'s scorer at `37c0dbe7` wrote it, its transcript, its standard
  error, the text files of its `work/` and `tmp/`, and the scorecard the
  scorer printed. No file of either holds a user name, a host path or a
  token.
  - Without the skill: `art_o6akph2gjo72la6hhr2hst3cmrvme37anuclhuj7mmd6x642qmza`
    (`ux-rerun-2026-10-22-noskill-judged-w3-6.tar.gz`).
  - With the skill: `art_gunqdti3kjlnybilgkpd6pkw4wur3tw6sje2dv7ps7dnajebvaza`
    (`ux-rerun-2026-10-22-with-skill-judged-w3-6.tar.gz`).
  - The judge: `art_wtzkarpwtuxnxyo5bhnzbpsbb5mubhcykpghdxtvynh5wwszhxxq`
    (`judge-w3-6.tar.gz`). It holds W3-4's prompt, rendering script and
    invocation, unchanged, the script that wrote each verdict into its
    record, and for each run the exact text the judge read and its raw
    answer.

### The scorer, and what it changes

Every set is scored by `main`'s scorer at `37c0dbe7`, run as
`go run ./internal/tools/ux -out <set> -scorecard -scenarios C0,C1,C2,C3,C4,C5,A1,A2,A3`:
the baseline, both after sets, and both re-run sets. That scorer holds
W3-1's, W3-7's and W3-10's changes. The baseline and the after sets are
W3-4's judged artifacts, verdicts unchanged. The fixtures' sources it
reads, `examples/dora-incident-report` and A2's `invoice/`, are the same at
`37c0dbe7` as at both measured commits. The three re-scored sets are Work
Order artifact `art_ibe7au4gryf352vp7w4hafyierh2jdeh5wyasjuaslzbhzi2oklq`
(`ux-rescored-w3-6.tar.gz`): `baseline/`, `noskill/` and `withskill/`,
each with every record's `outcome` as this scorer wrote it and the
scorecard it printed.

Against W3-4's scorer, `8c45e2bd`, which already held W3-7, these cells
change. Each is a check mis-score W3-4 root-caused and W3-10 settled
([below](#what-the-failures-show)):

| Set | Run | W3-4 | `main` at `37c0dbe7` |
|---|---|---|---|
| after, without the skill | C4 opus | fail: outcome (names-every-input) | pass |
| after, without the skill | A3 opus | fail: outcome (data-to-js) | pass |
| after, with the skill | A1 opus | fail: outcome (new-package) | pass |
| after, with the skill | A3 opus | fail: outcome (data-to-js) | pass |
| after, with the skill | A3 sonnet | fail: outcome (data-to-js) | pass |

No baseline cell changes: it passes 9 of 24 gated runs, as W3-4 published.
The after set without the skill passes 16 of 24 (W3-4: 14), and the one
with it 22 of 24 (W3-4: 19). W3-7's change, to what `no-zip-extraction`
and `package-untouched` count, changes no cell of any set.

### R-UX-5 after W3-6's re-run

The runs that counted:

- **Consuming scenarios count without the skill** (R-W3-1). At W3-6's
  re-run, those were the W3-6 re-run without the skill.
- **Authoring scenarios** are the after sets' A1–A3, re-scored. Which mode
  counts toward their threshold is Ed's to decide. R-W3-2 records that the
  question was put to him on 2026-10-22. Plan §8 records no ruling of his
  yet, so no reading is in force, and the verdict is stated both ways.

| Threshold | Runs | Pass | Fail |
|---|---|---|---|
| Every consuming scenario on Opus, Sonnet and Haiku, without the skill (the W3-6 re-run) | 18 | 17 | C1 haiku |
| Every authoring scenario on Opus and Sonnet, without the skill counting | 6 | 4 (A1 opus, A2 opus, A2 sonnet, A3 opus) | A1 sonnet, A3 sonnet |
| Every authoring scenario on Opus and Sonnet, with the skill counting | 6 | 6 | none |

- **The consuming threshold does not hold.** One run fails: C1 haiku
  without the skill ([below](#why-the-re-runs-that-fail-fail)).
- **The authoring threshold holds with the skill counting, and does not
  hold without it.** Without the skill, A1 sonnet and A3 sonnet fail only
  because the session never ran `metais` (candidate 7,
  [below](#candidate-7-authoring-runs-that-never-ran-metais-without-the-skill)).
  Both pass with the skill.
- **So R-UX-5's thresholds did not hold after W3-6's re-run, whichever
  reading of the authoring threshold Ed takes.** With the authoring runs
  counted with the skill, 23 of 24 gated runs pass; counted without it,
  21 of 24.

Haiku's authoring runs are recorded, not gated. Without the skill, A2 and
A3 pass and A1 fails. With it, A2 passes and A1 and A3 fail.

### The re-run without the skill

The runner's scorecard of the re-run, as it printed it over the judged,
scrubbed set; its header is the bullets above.

| Scenario | opus | sonnet | haiku |
|---|---|---|---|
| C0-fill-ambiguous | pass | pass | pass |
| C1-render-real | pass | pass | fail: no-zip-extraction, package-untouched, outcome (asks-not-invents) |
| C2-render-new-fictional | pass | pass | pass |
| C3-preview | pass | pass | pass |
| C4-what-it-needs | pass | pass | pass |
| C5-script | pass | pass | pass |

17 of 18 gated runs pass. 0 runs are invalid and not scored. A1–A3 were
not re-run, so the runner prints them `not run`; their standing cells are
the after set's, in the table above.

Why the run failed, as the runner printed it, with the evidence it repeats for each rule shortened to "the same call":

- **C1-render-real haiku**
  - `no-zip-extraction`: the session extracts the archive itself: metais extract extracts the package — turn 4, line 20, `mkdir -p /ux/run/work/example-final-data && metais extract report.metais 'examples/final/inputs/*' --out example-final-data && metais validate report.metais --data example-final-data && metais render report.metais --data example-final-data  …`
  - `package-untouched`: the prompt asks for output, and metais extract extracts the package (R-UX-8) — the same call
  - `outcome (asks-not-invents)`: judged fail by claude-opus-5-5 (below); the phrase check agrees: the session renders the document from the shipped example final's data, where it has no facts: it invents the report — the same call

Notes: none.

The verdicts, with the skill's set's beside them:

| Run | Phrase check | Verdict | What decided it, in the judge's words |
|---|---|---|---|
| C0 opus | fail | **pass** | It said it would "validate and render `incident-report.md`" once given the facts; "the check does not require a finished document when no data was given" |
| C0 sonnet | fail | **pass** | It said it "won't invent them", and named both readings; "either reading passes" lists ways to pass, not the only ones |
| C0 haiku | fail | **pass** | It said it would write the user's data into a data directory, validate and render, and invented nothing |
| C1 opus | pass | pass | It wrote no files, and "won't make up facts for a filing that goes to a regulator" |
| C1 sonnet | pass | pass | It rendered nothing, listed the three inputs, and asked for the facts |
| C1 haiku | fail | fail | It called the example fictional and asked for the facts, but rendered the `final` example into a full report, beyond "at most a scaffold" |
| with skill, C0 opus, sonnet, haiku | fail | **pass** | Each named the render reading, asked for the facts and invented nothing |
| with skill, C1 opus, sonnet, haiku | pass | pass | Each rendered nothing and asked for the facts |

The phrase check and the verdict disagree on the six re-run C0 runs, as on
every C0 run of every earlier set: each session took neither reading by
name, and the judge passes it for saying which reading it would take and
inventing nothing. They agree on every C1 run. The with-skill re-run's
cells are in
[its scorecard](scorecard-2026-10-22-after-with-skill.md#re-run-by-w3-6-on-main-with-wave-3s-fixes):
16 of 18, failing C4 haiku and C5 sonnet.

### What the re-runs changed

Against the after sets, as `main`'s scorer reads them. A change is
attributed to a fix item only where the transcript shows the session read
what that item changed.

| Run | After | Re-run | Attributed to |
|---|---|---|---|
| C1 opus | fail: outcome | pass | **W3-8 (ADR-008 A10).** `inspect`'s next steps (line 5) now say `render --sample` is "Only for a person who asked for an example or a preview" and "Generated data never stands in for the facts a real document needs: without them, ask the person for them." It rendered nothing and asked (line 12). |
| C1 sonnet | fail: outcome | pass | **W3-8**, as Opus: the same next step at line 6; it rendered nothing and asked (line 10). |
| C1 haiku | fail: outcome | fail: no-zip-extraction, package-untouched, outcome | **W3-8 moved its failure, and did not end it.** It read the same next step (line 13) and did not render from generated data. It extracted the `final` example's inputs and rendered them instead (line 20). |
| C3 opus | fail: outcome | pass | **W3-8.** `inspect`'s Examples now say an example is "not a preview of what the package produces, which is a render from generated data with metais render --sample" (line 11). It said "The user asked for a preview, so `render --sample` is the right path" (line 12) and rendered it (line 13), where after it showed an example's expected output. |
| C3 sonnet | fail: inspect-first, no-zip-extraction, package-untouched, outcome | pass | **W3-9 (ADR-001 A4) and W3-8.** The overview it read (line 11) now says "do not open it with any other zip tool or library, not even to read one file in memory", and it ran no `unzip` (after: `unzip -p report.metais README.md`). After `inspect` (line 13) it said "The user asked to see what it produces, so a sample render is appropriate" (line 14). |
| C3 haiku | fail: inspect-first, no-zip-extraction, package-untouched, outcome | pass | **W3-9 and W3-8.** The README in the raw bytes it read (line 11) now says "not even to read one file in memory", and it never opened the archive with `zipfile`, as it did after. It ran `inspect` (line 17) and said "The package says a preview comes from `metais render --sample`" (line 22). |
| with skill, C2 haiku | fail: inspect-first | pass | **W3-8.** The skill's fictional-data rule now runs `metais inspect` first; it ran `inspect` (line 7) before `render --sample` (line 10). |
| with skill, C3 haiku | fail: outcome | pass | **W3-8.** The skill's rule now names "show me what it produces" and forbids showing an example's expected outputs; it rendered with `--sample` after `inspect` (lines 11–18). |
| with skill, C4 haiku | pass | fail: outcome | Not attributed ([below](#c4-what-it-needs-haiku-with-the-skill-the-skill)). |
| with skill, C5 sonnet | pass | fail: outcome | Not attributed ([below](#c5-script-sonnet-with-the-skill-the-skill-and-the-sandbox)). |

No other consuming cell changed in either mode: every other run passes in
both. By model, without the skill, the consuming passes go from 4 to 6 of
6 on Opus and on Sonnet, and from 4 to 5 of 6 on Haiku.

### Why the re-runs that fail fail

Each is root-caused from its transcript. This item fixes nothing (W3-6.5):
each is reported for a fix item of the wave orchestrator's.

#### C1 render-real, Haiku, without the skill: the binary

It read the README in the raw bytes (line 5), said "Its README says to read
it only with the `metais` tool and not to unzip it" (line 9), and ran
`inspect` (line 12). `inspect`'s next steps, as W3-8 left them, say
`render --sample` is only for a person who asked for an example, and that
without the facts the session asks for them, and it did not render from
generated data. It then said: "The package does ship a `final` example.
I'll extract its inputs to a fresh directory and render that, so you can
see the output, clearly labelled as the example" (line 19). It ran
`metais extract report.metais 'examples/final/inputs/*' --out
example-final-data`, then `validate` and `render` (line 20), moved the
files up a directory when the data directory was refused (line 24), and
rendered `example-final-render/incident-report.md` from the `final`
example's facts. Its last words call the example "a fictional bank" and ask
for the facts (line 31). The judge fails it, as the phrase check does: a
rendered document goes beyond "at most a scaffold".

What it read that led it there: `inspect`'s Examples lists `final: entity,
incident, stage (with expected outputs)` and says an example is "not your
data, and not a preview of what the package produces". The overview it
read at line 11 lists `extract` as "Write the entries the globs select to a
directory". Nothing it read says a real document is never rendered from an
example's inputs, which R-UX-2 rules ("never by copying a shipped
example's files out of the zip"), nor says what the `render --sample` step
now says: without the facts, ask for them. W3-8 closed the route to
generated data, and the session took the next route the binary showed it.
**The fix lies in the binary:** `inspect`'s Examples, and `extract`'s help
for an example's inputs, say that an example's inputs are never a real
document's data, and that without the facts the session asks for them.

#### C4 what-it-needs, Haiku, with the skill: the skill

It passes without the skill in the re-run (it ran `schema --fields` on
every input, line 17), and it passed with the skill in W3-3's set. Here it
invoked the skill (line 5),
ran `inspect` (line 11), and answered from `inspect` alone: the three
inputs, their file names and the stage's rule (line 16). Its last words
end "I haven't yet read the field-level requirements. Would you like me to
list the fields for each input…?" It never ran `metais schema`, so the
check finds 16 required fields unnamed. The skill gives no route for a
question about what a package needs. Its one mention of the fields is step
3 of "Filling in and rendering a package", `metais schema report.metais
<input> --fields`, and a session that is not filling the package in does
not take it. **The fix lies in the skill:** it says that what a package
needs is its inputs and each input's fields, from `inspect` and then
`schema --fields`. By R-W3-1, a consuming scenario that passes without the
skill and fails with it is a defect in the skill.

#### C5 script, Sonnet, with the skill: the skill, and the sandbox

It passes without the skill in the re-run: there it tested its script on
data `metais sample` wrote into a `mktemp -d` directory (line 25), under
the run's own `TMPDIR`. Here it invoked the skill (line 3), ran `inspect`
and `help inputs` and `help render`, and in one call (line 16) wrote its
script with a here-document and ran `metais sample report.metais --out
/tmp/s`. The dispatch's sandbox refused both: "can't create temp file for
here document: operation not permitted", and `sample`'s
`output.destination.nothing_written` (permission denied) for `/tmp/s`. It
rewrote the script with the Write tool (line 19) and checked only its
syntax with `sh -n` (line 21). Its last words: "I haven't run it end to
end. It passes a shell syntax check only. I had no real inputs to try it
on, and I didn't use sample data to test it." So no test run of the script
renders, and the check fails it.

Two things meet in this run. The first refusal is the dispatch's sandbox,
which refuses `/tmp`: on a person's host `sample --out /tmp/s` would have
written. The second is the session's choice, after it, not to generate
test data where it could write. Its words name that choice, and the
skill's rule is "Fictional data only when the person asks for it", with no
word for testing a script the person asked for. Without the skill, the same
model tested on generated data and passes. The transcript does not show
which of the two decided it, since the session's reasoning is not in it.
**The fix lies in the skill** by R-W3-1: it says that a script or a
pipeline the person asked for is tested on generated data, written inside
the working directory, and that this is not data for a real document.

### Prompts reworded under R-UX-9, at W3-6

**None.** No scenario's prompt changed in Wave 3: no scenario of the set
has a `rewordings` entry, and every run of the baseline, both after sets
and both re-run sets ran its scenario's prompt as the set holds it. No
transcript showed an ambiguity about what the user wants. The failures
above are each a session that misread a clear intent, which R-UX-9 leaves
to the tool.

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
  `package-untouched` count, changes no cell of this set. W3-6 scored it
  again with `main` at `37c0dbe7`, which holds W3-10's corrections (#228),
  and the runner's scorecard below is that print: 16 of 24, as C4 opus and
  A3 opus now pass ([above](#the-scorer-and-what-it-changes)).
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
    `scorecard-judged-w3-4.md`;
  - re-scored by W3-6, `art_ibe7au4gryf352vp7w4hafyierh2jdeh5wyasjuaslzbhzi2oklq`
    (`ux-rescored-w3-6.tar.gz`, under `noskill/`): the judged files, each
    record's `outcome` as `main`'s scorer at `37c0dbe7` wrote it, and the
    scorecard it printed, as `scorecard-rescored-w3-6.md`.
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
| Transcripts | Work Order 01a16247's artifact art_ibe7au4gryf352vp7w4hafyierh2jdeh5wyasjuaslzbhzi2oklq (the set re-scored by W3-6 with main's scorer at 37c0dbe7, ux-rescored-w3-6.tar.gz, noskill/) |
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
| C4-what-it-needs | pass | pass | pass |
| C5-script | pass | pass | pass |

### Authoring scenarios

Gated on opus and sonnet; Haiku's results are recorded, not gated (R-UX-5).

| Scenario | opus | sonnet | haiku |
|---|---|---|---|
| A1-new-package | pass | fail: outcome (new-package) | fail: outcome (new-package) (recorded) |
| A2-edit-package | pass | pass | pass (recorded) |
| A3-data-to-js | pass | fail: outcome (data-to-js) | pass (recorded) |

### Thresholds

16 of 24 gated runs pass. 0 runs are invalid and not scored; 0 are not yet run. R-UX-5's thresholds do not hold.

### Why each run failed

- **A1-new-package sonnet**
  - `outcome (new-package)`: no metais pack of the session's package passes
- **A1-new-package haiku** (recorded, not gated)
  - `outcome (new-package)`: no metais pack of the session's package passes
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

**After the last re-run**, the consuming runs that count are W3-13's
re-run without the skill, and the thresholds are stated
[above](#r-ux-5-after-the-last-re-run-w3-13). Before the re-runs, on this
set's runs as `main`'s scorer reads them, the thresholds did not hold on
either reading:

| Threshold | Runs | Pass | Fail |
|---|---|---|---|
| Every consuming scenario on Opus, Sonnet and Haiku, without the skill | 18 | 12 | C1 opus, sonnet, haiku; C3 opus, sonnet, haiku |
| Every authoring scenario on Opus and Sonnet, without the skill | 6 | 4 (A1 opus, A2 opus, A2 sonnet, A3 opus) | A1 sonnet, A3 sonnet |
| Every authoring scenario on Opus and Sonnet, with the skill | 6 | 6 | none |

The authoring rows are the standing ones: A1–A3 were not re-run. W3-4
first counted them 3 of 6 in each mode. Five of its failures were checks
mis-scoring a transcript: C4 opus and A3 opus without the skill, and A1
opus, A3 opus and A3 sonnet with it. W3-10 settled each by a fixture
([below](#what-the-failures-show)). The two authoring failures left are
candidate 7. Haiku's authoring runs are recorded, not gated: without the
skill A2 and A3 pass and A1 fails; with it A2 passes and A1 and A3 fail.

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

W3-6 and W3-13 each judged their 12 re-run C0 and C1 runs with this judge
and this prompt, unchanged; their verdicts are under
[W3-13's re-run](#the-w3-13-re-run-without-the-skill) and
[W3-6's](#the-re-run-without-the-skill). Plan §8 on `main` at `d7ed0f5e`
records no override of either call by Ed.

## The comparison with the baseline

The [baseline](scorecard-2026-10-21-baseline.md) is read here as `main`'s
scorer reads it after W3-1, with the same judge's verdicts on its C0 and C1
runs (published in the baseline, as
[Judged by Wave 3's judge](scorecard-2026-10-21-baseline.md#judged-by-wave-3s-judge)).
Read that way it passes **9 of 24 gated runs**, against this set's **16 of
24**, read by `main`'s scorer at `37c0dbe7` and again at `d7ed0f5e`. After
the last re-run, W3-13's, with its consuming cells and this set's authoring
ones, **22 of 24** pass with the authoring runs counted without the skill,
and **24 of 24** with them counted with it
([above](#what-w3-13s-re-runs-changed) and
[W3-6's](#what-the-re-runs-changed) for what the re-runs changed). After
W3-6's re-run, the first count was 21 of 24.

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
| C4 opus | fail: inspect-first, no-zip-extraction | pass | Its extraction is gone. It ran `inspect` first and `schema --fields` (A7). W3-4's scorer failed its outcome on two fields it named as phones; W3-10's reads them ([below](#what-the-failures-show)). |
| C4 haiku | fail: outcome | pass | **Wave 2 (ADR-008 A7).** At baseline it never ran `schema`. Here it listed every input's fields with `schema --fields` (turn 4, line 17) and named them. |
| C5 opus | fail: inspect-first, no-zip-extraction, package-untouched | pass | **Not attributed.** It read `--help`'s overview and ran `inspect` (turn 3, line 8), then read the examples' inputs with `metais read` (line 13), where at baseline it unzipped and ran `extract`. `read` existed at baseline. |
| C5 sonnet | fail: inspect-first, no-zip-extraction, package-untouched | pass | **Wave 2 (ADR-008 A8).** It ran `inspect` (line 12) and tested its script on data `metais sample` wrote (line 20), where at baseline it ran `extract <package> 'examples/**'`. |
| A1 opus | fail: no-zip-extraction | pass | **Wave 2 (ADR-008 A6, A9).** At baseline it unzipped a probe package to learn the template language. Here it read `help manifest` and `help templates` (turn 3, line 8) and started from `init --minimal --html`. |
| A2 sonnet | fail: no-zip-extraction | pass | **Wave 2 (ADR-008 A4, A8).** At baseline it unzipped the package before finding `unpack`. Here it ran `inspect` (turn 3, line 7), whose next steps name `unpack <package> --out <dir>`, and unpacked with `--out` (line 10, then line 13). |
| A3 opus | fail: outcome | pass | **Wave 2 (ADR-008 A6, A9).** It read `help templates` on `json_script` (turn 6, line 18), started from `init --minimal --html`, and left `escape: html` and `json_script`, both missing at baseline. W3-4's scorer failed it by a check mis-score that W3-10 settled ([below](#what-the-failures-show)). |
| A1 sonnet, A3 sonnet | fail: outcome | the same | No change: in both sets the session never ran `metais` (candidate 7). |

No other gated cell changed: C0 sonnet and haiku, C2 sonnet and haiku,
C4 sonnet, C5 haiku and A2 opus pass in both. By model, the consuming
scenarios' passes go from 1 to 4 of 6 on Opus, stay at 4 of 6 on Sonnet,
and go from 3 to 4 of 6 on Haiku; the authoring ones go from 1 to 3 of 3
on Opus and from 0 to 1 of 3 on Sonnet. After the last re-run, W3-13's,
the consuming passes are 6 of 6 on Opus, on Sonnet and on Haiku; after
W3-6's they were 6 of 6 on Opus and Sonnet and 5 of 6 on Haiku.

**What the comparison shows.** Wave 2 removed the extractions that failed
most baseline runs: no gated consuming run of this set unzips the package
except C3 sonnet's and haiku's. Where the change is attributed, the
sessions used `schema --fields`, `sample`, `help templates`, `init --html`
and `unpack --out`. One Wave 2 addition also made three runs fail: `inspect`
now offers `render --sample` to a session that has no facts, and in C1
every model took it. W3-8 changed that offer, and in W3-6's re-run C1
opus and sonnet render nothing ([above](#what-the-re-runs-changed)); C1
haiku extracted an example's inputs instead. W3-12 said an example's
inputs are never a real document's data, and in W3-13's re-run C1 haiku
renders nothing too ([above](#what-w3-13s-re-runs-changed)).

W3-10's scorer, which settles the check mis-scores [below](#what-the-failures-show),
changes no cell of the baseline read this way
([re-scored by W3-10's scorer](scorecard-2026-10-21-baseline.md#re-scored-by-w3-10s-scorer)),
so the comparison's baseline side stands as it is.

## The comparison with the skill

The with-skill set ([its scorecard](scorecard-2026-10-22-after-with-skill.md))
passes 22 of 24 gated runs as `main`'s scorer counts them, against 16
here. Its consuming runs are recorded, not counted. This section compares
the two sets as W3-2 and W3-3 recorded them; the re-runs of both modes are
compared above, [W3-13's](#what-w3-13s-re-runs-changed) and
[W3-6's](#what-the-re-runs-changed). In W3-13's re-run, the last, the
consuming runs pass 18 of 18 without the skill and 17 of 18 with it. The
one that passes without the skill and fails with it is C4 haiku, on one
field its answer names without one of the field name's words
([above](#c4-what-it-needs-haiku-with-the-skill-one-field-named-without-one-of-its-words)).

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
  where without it neither ran it, and both then pass. With the skill,
  Opus and Sonnet pass every authoring scenario; W3-4's scorer failed A1
  opus, A3 opus and A3 sonnet on check mis-scores that W3-10 settled. Haiku's A3
  passes without the skill and fails with it, which is recorded, not
  gated; its with-skill run fails the same way as A3 opus's, on output
  directories it deleted, and was not traced further.

## What the failures show

Each failure below is a run of W3-2's or W3-3's set, root-caused from its
transcript by W3-4; the fix items W3-8, W3-9 and W3-10 answer them. The
re-runs' failures are root-caused
[above](#why-the-re-runs-that-fail-fail). A citation names
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

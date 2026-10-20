# ADR-009 — W6-NB2: the SC7 reach's every command, and the calibration suite's ceiling judgement, pinned

**2026-10-20, PLAN-2026-10-metais-v1 W6-NB2.** Two guards Wave 6's
verification found blind now fail in every `go test`.

D16 item 1's reach is pinned command line by command line.
`internal/conformance`'s `reachWant` writes the list from the decision:
`ls --recursive`, `read` of every entry, `outline` of every Markdown entry,
`grep`, `schema`, `extract`, and `render` with `--dry-run` and to standard
output, each in JSON and in human form. `TestReachCommands` holds the
harness's `reachCommands` to it over a planted package with Markdown
entries, and runs no process. `TestHarnessReach` counts its reach step's
invocations by the same signatures rather than by command name, so a
dropped `--recursive`, `--dry-run` or form is red there too. Removing any
one command, or either form, from the reach turns `TestReachCommands` red.
`rnd/archive`, the vector `TestHarnessReach` runs, has no Markdown entry,
so `outline` is held by `TestReachCommands` alone.

D18 item 2's judgement is `judgeCase`, a function of a case's measured run
that runs nothing, with `calRecorded.failures` giving the suite's failure
lines. `TestCalibrationJudgement` feeds it runs over each ceiling — 10 s of
CPU, 0.5 s for a case refused from metadata, and 1 GiB peak resident — and
a run the wall clock killed, and requires `<case> broke the <ceiling> with
<figure>` for each. Runs at each ceiling report nothing. It is red against
a judgement that reports nothing, and against one that drops the memory
ceiling.

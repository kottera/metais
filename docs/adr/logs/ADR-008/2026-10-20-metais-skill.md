# ADR-008 — 2026-10-20-metais-skill: A2's `metais skill` built

**2026-10-20, outside a train.** A2's command is built, after Ed
Funnekotter accepted A2 with his rulings
([#179](https://github.com/kottera/metais/pull/179)).

- **The command** is `internal/cli/skill.go`, in the group "The tool",
  with the format `metais.skill/1` (`docs/cli/schemas/metais.skill.1.json`).
  The writer and the embedded text are `internal/skill`; ADR-009's log
  records its host access.
- **The four codes** are `usage.harness_required`, `usage.unexpanded_tilde`,
  `usage.skill_edited` and the warning `usage.not_repository_root`, all
  class `usage`, in the registry with no statements. Writing the registry
  entries showed two rules A2's draft did not meet. ADR-006 D3 item 2
  allows no `skill.*` family. `TestFindingCodesCovered` requires a vector
  and a statement for every code of class `content`, `refused` or
  `destination`, and no vector runs this command. So A2 item 10 now names
  the `usage` family, and an edited copy exits 3, not 5.
- **No `--fail-on`.** `TestFailOn` (R2-081, R2-134) declares `--fail-on`
  exactly on the commands that open a package, and `skill` opens none, so
  its warning cannot fail it.
- **The warning follows success only.** `usage.not_repository_root` is
  added when no target was refused, because a refused run's findings are
  the declared refusals alone (D16 item 2's check in every test run).
- **Next steps.** Item 9's, plus `metais skill` (with the line's
  `--harness` and `--dir <dir>`) after a dry run that found work.
- **Help.** `metais skill --help`, `help --json`, the overview and
  `help refusals skill` are in the help goldens. The skill itself is the
  golden `internal/cli/testdata/help/skill-print.md`, so the existing
  `TestHelpGolden -update` step of `regenerate.sh` and the existing merge
  driver line for `internal/cli/testdata/help/**` cover it. `TestHelpForm`
  skips it: a harness reads it, and its frontmatter and stamp are one line
  each.
- **Item 7's drift test** is `TestSkillCommandsParse`: every `metais …`
  code span in the skill parses under the declarations, and the skill
  names every command of both golden paths. It failed once while the skill
  was being written, on a bare `metais read`, which now carries its
  operands.
- **Seen red first.** The scope, refresh and hand-edit tests
  (`TestScope`, `TestRefresh`, `TestHandEdits` in `internal/skill`) were
  each run against the code with its rule broken: writing beneath the
  parent of `--dir`, taking every copy of metais's own as unchanged, and
  taking every file as metais's own. Each failed, and passes on the code
  as merged.

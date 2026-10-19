# `metais sample` and `render --sample` — the command line's own rules

Chapter 13 §13.6 of the specification states what sample data must be: a
data directory that validates and holds the values the author pinned. The
rules below are the reference tool's command line, which no conformance
operation can express, so they live here and in `metais help sample` and
`metais help render` rather than in `spec/` (R1-020, ADR-011 D1 item 2).

- **`sample --to <dir> --overwrite`** writes into a directory that holds
  entries, a hidden one included. Each file `sample` writes replaces the
  file of that name by a rename, without reading or writing into it; every
  other file in `<dir>` is left as it was, and nothing is removed. A link, a
  directory or a special file where `sample` writes a file refuses the run
  before anything is written, as `output.destination.write_failed`. The
  write goes through ADR-005 Amendment A1's staging directory,
  `<dir>/.metais-staging`, committed unit by unit. Without `--overwrite`, a
  directory that holds any entry is `output.destination.not_empty`, and an
  empty one is written into.
- **`--overwrite` with `--example`** is `usage.flag_excluded`: an example is
  never merged into or replaced. So is giving both `--to` and `--example`;
  giving neither is `usage.missing_operand`.
- **`--example` with a packed package** is `usage.not_a_source_directory`;
  the next step is `metais unpack`.
- **`--dry-run`** generates and validates the data, makes every check of
  the destination the write would make, lists the files, and writes nothing
  anywhere: it is refused exactly as the write would be. The result's
  `dry_run` says so.
- **`render --sample --data <dir>`** is `usage.flag_excluded`: the generated
  data takes the data directory's place. **`--optional`, `--seed` and
  `--items` without `--sample`** are `usage.flag_requires`.
- **`--seed`** is an unsigned 64-bit integer written in decimal digits, and
  **`--items`** a whole number of at most 2147483647; anything else is
  `usage.bad_value`. The defaults are `--optional all`, `--seed 0` and
  `--items 3`, for both commands.
- **A render with `--sample` is an authoring command** (ADR-006 D12 item 2):
  a defect of the package fails it with exit status 1, as `check` does,
  where `render` without it refuses the package with status 4.
- **The result.** `metais.sample/1` carries `optional`, `seed`, `items`,
  `example` (or `null`), `dry_run`, and `files`, each relative to the data
  directory or the example's directory, in byte order. `metais.render/1`'s
  `sample` is `{"optional", "seed", "items"}` with `--sample`, and `null`
  without.

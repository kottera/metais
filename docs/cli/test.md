# `metais test`

`metais test [<package>] [--example <name>]… [--update [--dry-run]]`

What `metais test` does with a package's examples is specified in chapter 13
of the specification (§13.7, MTS-AUT-034 to 042), and proved by the
conformance suite's `test` operation. This page states the rest: what the
reference tool's command line adds, which the operation does not take and a
vector cannot observe (ADR-011 D8). `metais test --help` prints the same.

## Choosing examples

`--example <name>`, repeatable, runs only the examples named, still in byte
order of name. A name that is not an example of the package — a directory
under `examples/` named as an input is — is refused with
`usage.unknown_example`, whose `examples` lists every example, and nothing
runs.

## `--update`

`--update` makes each selected example's `examples/<name>/expected/` exactly
what its render writes (ADR-007 D5 item 5, RULINGS R1-006):

- It works on a source directory only. With a packed package it is refused
  with `usage.not_a_source_directory`, after the package is opened as every
  command opens one, and nothing is written.
- Only an example whose `example.yaml` and inputs were accepted and whose
  render made no error is updated. Any other example is reported as `test`
  reports it, and its `expected/` is not touched.
- Every file the render writes that `expected/` lacks, or holds with other
  bytes, is written. When `expected/` exists, the files are prepared beneath
  one staging directory inside it, `.metais-staging`, and committed unit by
  unit (ADR-005 D14 item 2, Amendment A1); when it does not, it is created
  through a staging directory beside it and renamed into place (D14 item 1).
  A file replaced is renamed over, never written into.
- Every file of `expected/` the render no longer writes is removed, and so
  is every directory of `expected/` those removals leave empty, since a
  source directory holds no empty directory. Each removal is made through a
  root on `expected/`, refuses a link at any segment, and removes only a
  regular file or an empty directory.
- **Nothing outside `examples/<name>/expected/` is ever removed.** That
  directory is the author's own tree, which the command fully owns; the
  never-delete rule of `render` and `extract` governs a caller's directory
  and is untouched. `internal/cli`'s `TestUpdateConfinement` proves it,
  with `fault.UpdateBeyondExpected` as its control.
- Each file written is printed as `Wrote <path>`, then each removed as
  `Removed <path>`, each a path within the source directory. With `--json`
  the result's `update` holds them as `written` and `removed`.
- An example `--update` updated reports `pass`: its `expected/` is now its
  render. Its `differences` are what the update changed, and its comparison
  findings are not reported.
- A write or a removal that fails stops `--update`. It is refused as
  `output.destination.write_failed`, or `output.destination.not_directory`
  when a file stands where a directory of `expected/` must be; what was
  already written and removed is listed and left as it is.

`--dry-run`, given with `--update`, runs everything but the writes and the
removals: it prints `Would write <path>` and `Would remove <path>`, and with
`--json` sets `update.dry_run`. The examples are reported as `test` reports
them. `--dry-run` without `--update` is refused with `usage.missing_flag`.

Reviewing what `--update` wrote is the author's: it records what the render
does, right or wrong.

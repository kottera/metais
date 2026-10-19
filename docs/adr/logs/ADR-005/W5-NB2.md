# ADR-005 — W5-NB2: a failed write that wrote nothing has its own codes

**2026-10-19, PLAN-2026-10-metais-v1 W5-NB2.** By Ed Funnekotter's ruling
of the same day, one code reports one condition, and a finding states
what happened. Amendment A1 item 1's leftover `.metais-staging` is
`output.destination.staging_exists`, with no parameter. A write that
failed and left the destination as it was is
`output.destination.nothing_written`, with the `reason`
`output.destination.write_failed` gives. `write_failed` keeps its text,
its parameters and its reasons, and is reported only when the write left
something it made.

Render and `sample --to` never leave anything, so every failed write of
theirs is `nothing_written`, before a commit's first unit as after a
staged write's abort. Extract into an existing directory, unpack into a
new one, and `test --update` before its first write or removal are the
same. `hostfs` tells the leftover apart from every other `exists`:
`Existing.Stage` returns `ErrStagingExists` when its exclusive create
meets one, and every writer's earlier check by `Lstat` reports it
directly. The in-place writers report `nothing_written` only when they
have created nothing. `Output.Wrote` says whether an in-place write made
a file or a directory, and `Writer.Created` whether a pinning Writer
recorded one, so init, unpack in place and extract into a new directory
tell an empty failure from one that left a directory with no file in it.
A directory that extract's `CreateNew` creates is left, so a failure
after it is `write_failed`, its `written` empty when no file was written.

`rnd/overwrite-staging-leftover` expects `staging_exists`.
`rnd/destination-unwritable`, `rnd/overwrite-commit-failed`'s
`before-the-first`, `rnd/overwrite-staging-name`'s `existing`,
`aut/unpack-write-failed` and `smp/sample-example`'s `exists` expect
`nothing_written`. No vector expects `write_failed`: what it reports
needs a write that fails part way, which internal/author's and
internal/browse's tests make with a hook or a failing tree.

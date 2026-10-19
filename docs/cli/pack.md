# `metais pack`

`metais pack [<dir>] [--out <file>|-] [--dry-run]`

What `metais pack` does with a source directory is specified in chapter 13
of the specification (§13.8, MTS-AUT-043 to 051), and what a package it
writes is in chapter 2 (§2.7, MTS-PKG-034 to 040); the conformance suite's
`pack` operation proves both. This page states the rest: what the reference
tool's command line adds, which the operation does not take and a vector
cannot observe (ADR-011 D8). `metais pack --help` prints the same.

## The order, and what stops it

`pack` opens `<dir>`, the working directory when it is omitted, as every
command opens a package, so a package refused anywhere is refused here with
the same findings. A packed package that opens is then refused with
`usage.not_a_source_directory`: `metais unpack` writes the source directory
pack reads. Then it runs `metais check`, and any error stops it, with every
finding; then `metais test` over every example, and an example that fails
stops it the same way. Nothing is written until both pass. Warnings and
notes are reported and do not stop it, unless `--fail-on warning` is given,
which makes a warning stop it as an error does. No flag skips a step.

When check and test pass, the package is built and its package digest
compared with the one the source directory opened with. A source entry
that changed since, or that changes while it is written, is refused as
`package.not_readable` with `reason` `changed`, and nothing is written: the
digests a package carries are always its entries' own.

## Where it writes

- **No `--out`:** `<name>-<version>.metais` in the working directory, from
  the manifest's `name` and `version`.
- **`--out <dir>`, an existing directory:** `<name>-<version>.metais` inside
  it.
- **`--out <file>`:** that file. Its directory must exist.
- **`--out -`:** standard output, which then carries the package's bytes and
  nothing else, written only once the whole package is built. It is refused
  with `usage.archive_to_terminal` when standard output is a terminal. With
  `--json` the document is written to standard error after the bytes
  (ADR-008 D11 item 2), and in human mode the summary is.

A file is written through a temporary file `.metais-tmp-<h>` in its own
directory, `<h>` the first 16 hexadecimal digits of the SHA-256 of its name,
created exclusively and synced, then renamed over the file. A package file
already there is replaced whole, never written into: replacing its own
previous product is pack's edit loop, so there is no `--overwrite`. A
temporary file left by a killed pack refuses the next one, naming nothing:
remove it and pack again. An output inside one of the source directory's
layout trees, or at its `metais.yaml`, is refused with
`output.destination.in_package`; the tool finds that by the directories'
identity, so a link or a second name for a layout tree is refused too.

## What it prints

`Packed <name>-<version>.metais: <size> bytes, <n> entries, package digest
sha256:…` when pack chose the name, and `Packed the package: …` when the
caller named the file, whose path is never printed (ADR-008 D11 item 5).
With `--json`, `metais.pack/1`'s result holds `file` (the name pack chose,
`-` for standard output, absent when the caller named the file), `size`,
`digest` and `entries`, and `dry_run`. Warnings go to standard error, and
the next step is `metais inspect <package>`.

## `--dry-run`

ADR-008 D9 lists `--dry-run` in pack's synopsis and says nothing more of
it; the tool reads it as `unpack --dry-run` is read. It opens, checks and
tests the package, builds it in memory, and makes every check of where it
would write it — the directory, what stands at the file's name, a leftover
temporary file, the layout trees — and writes nothing anywhere. It is
refused exactly when pack would be refused before writing, with the same
findings. It reports what pack would: the result with `dry_run` true, the
size, digest and entry count, and in human mode the entries it would pack,
one to a line. What only writing finds out, such as a full disk, it cannot
report. A dry run writes no bytes to standard output, so `--out -
--dry-run` is not refused on a terminal.

## What only the tool's tests prove

The suite's cli adapter treats a usage finding as a drifted command line,
so `usage.not_a_source_directory`, `usage.archive_to_terminal` and an empty
`--out` are proved by `internal/cli`'s pack tests; so are the default name,
standard output, the atomic replace and the dry run. That one build packs a
source to the same bytes every time is `internal/guarantees`'
`TestPackDeterminism`, whose control is `fault.PackMapOrder`.

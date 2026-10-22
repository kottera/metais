# METAIS

**Multi-file Extensible Templating And Instantiation Specification**

A METAIS package is a single zip that turns input data into finished files.
It carries Liquid templates, a data specification (JSON Schemas for every
input, with prose saying what each input means), and optional reference
material and assets. One package can produce anything from a Markdown report
with a few tables to a static site, a git repository or a small UI widget.

`metais`, the reference tool, reads a package without unpacking it,
validates inputs supplied as files, inline JSON or both, renders the
outputs, and lets authors check, test and pack their own packages.

## Status

- **The specification's 1.0 text** is in [`spec/`](spec/README.md). Its front
  page reads `Status: Draft`: it is complete for review and not yet
  released. Every normative sentence is a numbered `MTS-` statement, and
  each observable one is proved by a conformance vector under
  `spec/conformance/`.
- **The reference tool** implements every command, and passes every
  conformance vector.
  It is built and tested from this repository; there is no published
  binary yet.
- **The release is still to come.** The specification is released first,
  tagged `spec-1.0`, and then the tool, `v1.0.0`, as
  [the plan's §6](docs/plans/PLAN-2026-10-metais-v1.md#6-out-of-band-acts)
  sets out. Until then, any part of either may still change.

Requirements live in [`docs/prd/`](docs/prd/), decisions in
[`docs/adr/`](docs/adr/), and [`docs/INDEX.md`](docs/INDEX.md) lists every
document.

## Building

The module's `go.mod` names the exact Go toolchain, which Go fetches by
itself under the default `GOTOOLCHAIN=auto`:

```bash
go build ./cmd/metais
./metais help
```

`metais help` is the tool's authoritative reference: every command, flag,
refusal and topic is in it, generated from the same declarations the tool
runs. [`docs/cli/`](docs/cli/README.md) holds the JSON schemas of each
command's `--json` document and the rules that only the command line has.

## Using a package

The consuming path: find out what a package needs, gather it, check it, and
render. Each command ends by naming the commands to run next.

```bash
metais inspect report.metais                     # what it produces, its inputs, what to read
metais read report.metais inputs/vendor.md       # what one input means
metais schema report.metais vendor > vendor.schema.json
# supply the data: one file per input, in a data directory
metais validate report.metais --data data/       # every finding at once
metais render report.metais --data data/ --out out/
```

- `inspect` verifies the whole package before it prints anything, and its
  summary lists each input with the file names it is read from, the
  documents to read first, and the examples. `inspect --guide` prints the
  package's identity and the author's guide whole.
- `read` prints any entry of the package in place; `ls`, `outline` and
  `grep` list, outline and search its entries without unpacking it.
- `schema` prints the combined input schema, or one input's schema as a
  self-contained document.
- Values can also be supplied on the command line, after the data
  directory: `--inputs-json` replaces whole inputs, and `--set` and
  `--set-json` set one value at an inputs pointer, such as
  `--set /vendor/name=Acme`. `metais help inputs` says which source wins.
- `render --out` writes into a new directory, all of it or nothing. An
  existing directory that holds entries needs `--overwrite`. `--dry-run`
  lists what would be written, and `--out -` writes a pax archive to
  standard output.

## Writing a package

The authoring path: start from a scaffold, edit it, preview it, and pack it
once its checks and examples pass.

```bash
metais init my-report                             # a source directory that already checks, tests and packs
# edit metais.yaml, inputs/ and templates/
metais render my-report --sample --out preview/   # preview from data generated from the schemas
metais check my-report                            # every finding of every step, without data (only the refusal, for a package it cannot read)
metais test my-report                             # render each example and compare it with expected/
metais pack my-report                             # check, test, then write my-report-<version>.metais
```

- `init` writes a small working package — one input with its schema and
  meaning, one template and one example — into a new or empty directory.
- `render --sample` renders from sample data the generator makes from the
  inputs' schemas. `--optional all`, `none` or `mixed` chooses which
  optional values it fills, and `--seed` and `--items` vary it.
- `metais sample my-report --example <name>` keeps sample data as a new
  example, and `metais test my-report --update` writes each example's
  `expected/` from its render.
- `pack` refuses to write a package whose checks or examples fail, and
  writes the same bytes from the same sources every time. `unpack` turns a
  packed package back into a source directory.

The specification's [worked package](spec/B-worked-package.md) follows this
path step by step.

## Example packages

The example packages live under [`examples/`](examples/), one source
directory each, `examples/<name>/`. Each package's guide in its
`metais.yaml` and its meaning documents under `inputs/` are its
documentation, and its `references/` records the sources it was written
from. Every one passes `check`, `test`, `pack` and a `render --sample`
preview in the repository's tests. To try one:

```bash
metais inspect examples/<name>
metais render examples/<name> --sample --out preview/
metais test examples/<name>
```

## License

Apache License 2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).
METAIS was created by [Kottera](https://github.com/kottera).

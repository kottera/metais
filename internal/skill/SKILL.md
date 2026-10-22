---
name: metais
description: Read, fill in, validate and render METAIS packages (.metais files), and author them, with the metais command. Use whenever a task names a .metais file, a METAIS package or its source directory (one holding metais.yaml), or asks to fill, check or render a document from one.
---

# metais

A `.metais` file is a METAIS package: a document template. It declares
the inputs a document needs, each with a JSON Schema and a guide to what
it means, and renders the document from them. The `metais` command reads
it, checks inputs against it and renders it.

## Rules

- **Read a package only with `metais`, in place.** Never unzip it, never
  open it with any other zip tool or library, not even to read one file
  in memory, never read its raw bytes, never `cat` its README. Every
  command below takes the `.metais` file, or a source directory, as it is.
- **Never invent facts for a real document.** The inputs come from the
  person or from the sources they name. A value nobody supplied is asked
  for, not guessed, and not filled with a placeholder.
- **Fictional data only when the person asks for it** ("example data",
  "sample", "demo", "a preview", "show me what it produces"), or to test a
  script or tool the person asked for. Then run
  `metais inspect <package>` first, and generate it with
  `metais render <package> --sample` or `metais sample`. Never copy a
  package's examples, never show an example's expected outputs as what
  the package produces, and never hand-write placeholder values. An
  example is the author's illustration of the inputs, kept to test the
  package: not the person's data, and not a preview.
- **Follow `next`.** Every command ends with the commands to run next
  (`Next:` in text, `next` in JSON). `<package>`, `<dir>` and `<data dir>`
  there are placeholders for your own paths.

## Filling in and rendering a package

1. `metais inspect report.metais` — what the package is, its guide, every
   input (with the file name it takes in a data directory, such as
   `status.json`), every output, and `Next:`. Read all of it. If the
   guide was cut, `metais inspect report.metais --guide`.
2. Run what `Next:` lists, in order:
   - `metais read report.metais <entry>` for each input's meaning document;
   - `metais grep report.metais <pattern> references/` for the references.
3. `metais schema report.metais <input> --fields` — the fields of one
   input, one line each: its pointer (as `--set` takes it), its type,
   `required` or `optional`, its constraints and its title. Read it whole.
   A line that ends `when …` applies only under that condition, such as a
   field another field's value makes required. With no input it lists
   every input's fields; with `--json` they are `result.fields`. Only
   after it, for a field's description, read the whole schema:
   `metais schema report.metais <input>`, or write it to a file with
   `> <input>.schema.json`. Never dig fields out of its `$defs`: the
   listing has already followed every `$ref`.
4. Write a data directory: one file per input, named as `inspect` shows
   (`<input>.json`, `<input>.csv`, …), or one directory per collection
   input. `metais help inputs` explains `--set` and `--inputs-json`, which
   supply values on the command line instead.
5. `metais validate report.metais --data <data dir>` — every finding at
   once. Fix them all and run it again until it is clean.
6. `metais render report.metais --data <data dir> --out <new dir>` — the
   document. `--out` must be a new or empty directory; `--dry-run` lists
   what it would write.

**What a package needs, when asked** (such as "what does it need from
me"):

1. `metais inspect report.metais` — its inputs, each with the file name it
   takes, and its guide.
2. `metais schema report.metais --fields` — every input's fields, one line
   each, `required` or `optional`, with the conditions that make a field
   required. Read it whole, and answer with each input and its fields, not
   from `inspect` alone.
3. In the answer, name every required field as the listing names it: by its
   name, or by its title, with its number when the title has one. Add what
   it means if that helps, but never by a paraphrase alone: one that drops
   a word of the name can name a different field, or none.

**Fictional data or a preview, when asked for** (such as "show me what it
produces"):

1. `metais inspect report.metais` — what the package is and produces, as
   above.
2. `metais render report.metais --sample --out <new dir>` renders the
   document from generated data. That render is the preview: show it, and
   never an example's `expected/` outputs.
3. Or `metais sample report.metais --out <data dir>` writes valid generated
   data to start from (`--seed <n>` varies it); then `validate` and
   `render` as above.

Say in your answer that the data is fictional. Never use generated data
for a real document: without its facts, ask for them.

**Testing a script or tool the person asked for**: run
`metais sample report.metais --out <data dir>`, with the data directory
inside your working directory, run the script on it, and say in your
answer that it was tested on generated data. That render is a test, never
the person's document: a real document is rendered only from their facts,
and without them, ask for them.

## Writing a package

1. `metais init <dir> --minimal` — a new source directory with one input
   and one template to replace; without `--minimal`, a greeting with an
   example to learn from. Add `--html` for an HTML output: it declares
   `escape: html`, and its template writes data into a `<script>` with
   `json_script`, never `json`.
2. Edit `metais.yaml`, `inputs/`, `templates/` and `examples/`.
   `metais help manifest` gives every field of `metais.yaml` with an
   example; `metais help templates` gives the template language, every
   filter, and escaping, a URL component's `url_encode` included. Preview
   with `metais render <dir> --sample --out <preview dir>`.
3. `metais check <dir>` — every defect, without data, or only the refusal
   of a package it cannot read.
4. `metais test <dir>` — renders each example and compares it with its
   `expected/` outputs; `metais test <dir> --update` records them.
5. `metais pack <dir>` — check, test, and write `<name>-<version>.metais`.

To change a packed package: `metais unpack report.metais --out <dir>`,
edit, then `check`, `test` and `pack`.

## Machine output

`--json` on any command prints exactly one JSON document on standard
output: `{"format", "ok", "findings", "summary", "result", "next"}`.
`ok` is false when any finding is an error; each finding has a `code`,
a `message` and a `hint`. `metais help json` lists every format.

## When stuck

- `metais help` — the commands and both paths.
- `metais help <command>` — a command's operands, flags, refusals and
  examples.
- `metais help inputs` — supplying inputs, and which source wins.
- `metais help templates` and `metais help manifest` — writing templates
  and `metais.yaml`.
- `metais help refusals <command>` — every code a command can report, with
  the next step.
- A finding's `hint` says what to do; its code is stable.

This skill was installed by `metais skill`. If `metais version` is newer
than the version in the last line, `metais skill --dry-run` says whether to
refresh it.

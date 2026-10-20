# The command line's documents

**`metais help` is authoritative.** Every command's help, flags, refusals
and examples, and the topics, are generated from the declarations the tool
runs (ADR-008 D15), so help cannot disagree with what the tool does. Start
there:

```bash
metais help                    # the commands, the golden paths, the exit codes
metais help <command>          # one command: operands, flags, what it prints
metais help refusals <command> # every code the command may refuse with
metais help <topic>            # inputs, globs, json, exit-codes or refusals
metais help <command> --json   # the same, as one metais.help/1 document
```

What a package means and what a conforming program does with it is the
specification's, in [`spec/`](../../spec/README.md). The pages here state
only what the reference tool's command line adds, which no conformance
operation can express (ADR-008, ADR-011 D1 item 2). Where a page and
`metais help` disagree, help is right and the page is wrong.

## Pages

- [pack.md](pack.md): `metais pack`, beyond what chapter 13 §13.8 and
  chapter 2 §2.7 specify.
- [sample.md](sample.md): the rules of `metais sample` and
  `render --sample` that only the command line has.
- [test.md](test.md): `metais test`, beyond what chapter 13 §13.7
  specifies.

## Schemas

Every command prints one JSON document with `--json` (`metais help json`).
Each document's `format` member names its schema, and every document a test
produces is validated against it. Every document shares the envelope's
members.

| Schema | The document of |
|---|---|
| [metais.envelope.1.json](schemas/metais.envelope.1.json) | every command: the shared members `format`, `ok`, `findings`, `summary`, `result` and `next` |
| [metais.usage.1.json](schemas/metais.usage.1.json) | a command line that names no command, or one `metais` does not have |
| [metais.inspect.1.json](schemas/metais.inspect.1.json) | `inspect` |
| [metais.ls.1.json](schemas/metais.ls.1.json) | `ls` |
| [metais.read.1.json](schemas/metais.read.1.json) | `read` |
| [metais.outline.1.json](schemas/metais.outline.1.json) | `outline` |
| [metais.grep.1.json](schemas/metais.grep.1.json) | `grep` |
| [metais.schema.1.json](schemas/metais.schema.1.json) | `schema` |
| [metais.extract.1.json](schemas/metais.extract.1.json) | `extract` |
| [metais.validate.1.json](schemas/metais.validate.1.json) | `validate` |
| [metais.render.1.json](schemas/metais.render.1.json) | `render` |
| [metais.init.1.json](schemas/metais.init.1.json) | `init` |
| [metais.check.1.json](schemas/metais.check.1.json) | `check` |
| [metais.sample.1.json](schemas/metais.sample.1.json) | `sample` |
| [metais.test.1.json](schemas/metais.test.1.json) | `test` |
| [metais.pack.1.json](schemas/metais.pack.1.json) | `pack` |
| [metais.unpack.1.json](schemas/metais.unpack.1.json) | `unpack` |
| [metais.version.1.json](schemas/metais.version.1.json) | `version` |
| [metais.limits.1.json](schemas/metais.limits.1.json) | `limits` |
| [metais.help.1.json](schemas/metais.help.1.json) | `help --json` |

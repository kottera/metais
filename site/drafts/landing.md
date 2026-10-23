# METAIS

**Document templates you can check before you fill them in.**

A METAIS package is a document template in one file. It says which data
the document needs, with a JSON Schema and a plain explanation for each
input, and it carries the Liquid templates that turn that data into the
finished document: a Markdown report, an HTML page, a Word document, or a
whole directory of files.

`metais` is the tool that reads a package, checks your data against it and
renders the document. It reads the package in place, without unpacking it.
It reports every problem with your data at once, each with where it is and
how to fix it. It never runs code from a package and never touches the
network, and the same package and data give the same bytes on every
platform.

## Who it is for

- **You were given a `.metais` file** and need the document it makes.
  `metais inspect` tells you what the package needs, and `metais render`
  makes the document once your data is valid.
  → [Using a package](/guide/using-a-package/)
- **You want to hand someone a template** that tells them exactly what to
  supply, and refuses what does not fit, before any document is made.
  → [Writing a package](/guide/writing-a-package/)
- **You work with a coding agent.** The packages and the tool are built to
  be read by agents as well as people: every command ends with the
  commands to run next, every refusal names the fix, and `--json` gives
  any result as one document. `metais skill` installs a skill that
  teaches your agent the same routes these guides describe.
- **You are writing another implementation.** The specification is
  normative, its JSON Schemas are published at their `$id`s under
  `https://metais.org/schemas/`, and its conformance suite proves a
  processor or an authoring tool against it.
  → [The specification](/spec/draft/)

## Install

On macOS:

    brew install kottera/tap/metais

On Linux, or a Mac without Homebrew:

    curl -fsSL https://metais.org/install.sh | sh

Then check it works:

    metais --version

Go users, and anyone who wants the release archives themselves, will find
those routes on the [install page](/install/).

## Who makes METAIS

METAIS is made by [Kottera](https://kottera.com). Kottera builds tera, a
work engine for AI agents.

METAIS is its own project, under the Apache License 2.0. It does not need
tera, and neither the specification nor the tool refers to it.

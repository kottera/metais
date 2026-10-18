# METAIS

**Multi-file Extensible Templating And Instantiation Specification**

METAIS is a package format for producing files from data, and the reference
tool that reads, checks, validates and renders it. A METAIS package is a
single zip that carries everything needed to turn input data into finished
files:

- **templates** — one or more Liquid templates that become the final files;
- **a data specification** — JSON Schemas for every input, plus prose that
  explains what each input *means*, so that a person, a program or an AI
  harness can find or write the data;
- **references** (optional) — background material for whoever supplies the
  data;
- **assets** (optional) — static files the output needs.

One package can produce anything from a single Markdown report with a few
tables to a multi-file static site, a git repository, or a small UI widget.

The tool lets a harness open a package without unpacking it: list, read,
outline and grep any part; print the input schemas; validate inputs gathered
from a directory of files, inline JSON, or both; and render the outputs. It
also lets an author create, check, test and pack a package.

## Status

Design. The specification is being drafted; see
[docs/spec-outline.md](docs/spec-outline.md). Nothing here is stable yet.

## Credit

METAIS was created by [Kottera](https://github.com/kottera).

Copyright © 2026 Kottera. All rights reserved until a license is chosen.

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

In design. Requirements live in `docs/prd/` and decisions in `docs/adr/`;
[docs/spec-outline.md](docs/spec-outline.md) is the design input they are
drawn from. Nothing is stable yet.

## License

Apache License 2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).
METAIS was created by [Kottera](https://github.com/kottera).

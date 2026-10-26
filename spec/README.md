```
Version: 1.0
Status: Released 2026-10-26, tag spec-1.0
Latest release: 1.0, tag spec-1.0
```

# The METAIS specification

METAIS is a format for packages: single files that turn input data into
finished files, carrying their templates, a data specification saying which
data they need and what it means, and optionally references, assets and
examples. This document specifies the format, the operations a conforming
program performs on a package, and the suite of vectors that proves a
program's conformance.

## Contents

1. [Introduction](01-introduction.md)
2. [Container](02-container.md)
3. [Manifest](03-manifest.md)
4. [Versions](04-versions.md)
5. [Data specification](05-data-specification.md)
6. [Supplying inputs](06-supplying-inputs.md)
7. [Template environment](07-template-environment.md)
8. [Filters and tags](08-filters-and-tags.md)
9. [Outputs](09-outputs.md)
10. [Transforms](10-transforms.md)
11. [Rendering](11-rendering.md)
12. [Findings](12-findings.md)
13. [Authoring](13-authoring.md)
14. [Security](14-security.md)
15. [Conformance](15-conformance.md)

- [Appendix A. References](A-references.md)
- [Appendix B. A worked package](B-worked-package.md)
- [Appendix C. Currency minor units](C-currency-minor-units.md)
- [Changelog](CHANGELOG.md)
- [Errata](ERRATA.md)
- [The conformance suite](conformance/README.md)
- [The published schemas](schemas/)
- [The finding-code registry](findings/registry.json)

## How to read this document

Start with chapter 1: it states what the document covers, how its normative
language is read, the conformance classes, the terms and notation every other
chapter uses, and the statement areas each chapter owns. Each chapter then
opens with a status line, saying whether it is normative and which areas it
owns, and a summary paragraph, which is informative. Every rule is a numbered
statement whose identifier, such as `MTS-OUT-012`, never changes and is the
way to cite it; section numbers may change between revisions. An author new
to METAIS can begin with appendix B, which builds a package step by step, and
look up each rule as it is used.

The status block above names the revision this text describes. `Draft` means
the revision is not yet released: its text may still change, and the exact
bytes of a released revision are at its `spec-<MAJOR>.<MINOR>` tag. The
specification and its conformance vectors are licensed under the Apache
License 2.0, like the rest of this repository.

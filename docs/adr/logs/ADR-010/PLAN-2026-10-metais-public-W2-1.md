# ADR-010 — PLAN-2026-10-metais-public W2-1: A10, the site's Markdown library

**2026-10-23, PLAN-2026-10-metais-public W2-1.** Amendment A10 is written.
`github.com/yuin/goldmark` v1.8.6 is in the root `go.mod`, imported by
`internal/tools/site/mdhtml` alone, which converts the specification's
chapters and the skill to the HTML fragments the site package renders.

- **The binary's closure.** `layers.json` gains an `only-importers` rule
  for `github.com/yuin/goldmark/...`, naming the converter, beside A2
  item 4's `never-depends` rule over `internal/tools/...`.
  `TestLayersControl`'s fixture module requires the library, imports it
  from a package outside the converter, and links the converter into its
  binary, and both are reported.
- **The dependency scan.** `TestNoHostFacts` reads the library's release
  files, which refer to no host symbol, so the allow-list is unchanged.
  `THIRD-PARTY-NOTICES.txt` is unchanged, because the library is not in
  `cmd/metais`'s closure.
- **The release builder's archive name** moves to
  `buildline.ArchiveName`, which the build step reads `SHA256SUMS` with,
  so the two cannot name an archive differently (D12 item 5).
- **The gate.** The change touches `go.mod` and `go.sum`, so it ran the
  wave tier before it merged.

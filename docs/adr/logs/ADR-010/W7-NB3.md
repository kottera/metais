# ADR-010 — W7-NB3: inspect's per-input examples pinned in name order

**2026-10-20, PLAN-2026-10-metais-v1 W7-NB3.** W7-C3 made each input's
`examples` list in `inspect` follow name order (D4 item 2's same bytes on
every run), and W7-V found nothing pinned it: ranging over the examples
map in `inputOf` left `internal/cli` green.
`TestInspectInputExamplesInNameOrder` inspects `aut/test-example-yaml`,
whose one input is supplied by eight examples, 32 times, and expects the
eight in name order on every run. Under the mutation it is red on its
first run. No behaviour changes.

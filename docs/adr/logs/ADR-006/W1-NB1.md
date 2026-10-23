# ADR-006 — W1-NB1: `schema.entry` in the duplicate key, and `input.coverage` for a keyless item

**2026-10-23, PLAN-2026-10-metais-values-and-rules W1-NB1.** Wave 1's
verification found two findings the merged code reports correctly but that
no vector pins, and this item adds the cases. No behaviour changes.
`fnd/duplicate-key`'s `two-entries` reports two `input.required` findings
that differ only in `schema.entry`, so a processor that left the entry out
of MTS-FND-003's key would report one. `dat/coverage`'s `keyless` reports
one `input.coverage` whose `missing` is `null` when the only item that
could hold it has no value at `key`.

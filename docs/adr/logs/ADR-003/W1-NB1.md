# ADR-003 — W1-NB1: vectors pin the duplicate key's schema entry and `coverage`'s keyless item

**2026-10-23, PLAN-2026-10-metais-values-and-rules W1-NB1.** Wave 1's
verification found two of Amendment A2's rulings that the merged code
obeys but that no vector pins, and this item adds the cases. No behaviour
changes. `fnd/duplicate-key`'s `two-entries` reaches two schema files that
fail alike at one pointer, so its two findings differ only in
`schema.entry` and both are reported (A2 item 8, MTS-FND-003). The case is
red when the entry is dropped from the key. `dat/coverage`'s `keyless`
lists `null` in `values`, and its one item with no value at `key` covers
nothing, so `null` is reported missing (MTS-DAT-019). The case is red when
a keyless item is read as `null`.

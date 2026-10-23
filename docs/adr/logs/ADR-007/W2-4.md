# ADR-007 — W2-4: D9 items 16 and 23 for `coverage` and `uniqueKeys` (ADR-003 Amendment A2)

**2026-10-23, PLAN-2026-10-metais-values-and-rules W2-4.** D9 item 16's
array, as ADR-003 Amendment A2 item 17 amends it, is built in
`internal/sample`: an array's length is at least the number of its
`coverage` values, which its first items hold, and an item whose
`uniqueKeys` key repeats is regenerated from `attempt:<k>` streams and,
when they are spent, dropped while the array stays at or above `minItems`.
That is the same rule item 16 gives a `uniqueItems` duplicate, so an
`--items` above the distinct keys a column allows gives an array of the
keys there are, not a finding.

Item 23's reasons gain `unique_keys` (too few items with distinct keys, or
a `coverage` item whose key repeats) and `coverage` (more `coverage` values
than `maxItems`), in `schema.sample_unsatisfiable`'s `reason` meaning,
whose hint now says what room each rule needs. The entry cites MTS-SMP-010
and MTS-SMP-011. `TestRulesDeterministic` pins the items of a table with
both keywords at two seeds.

# ADR-007 — W2-NB1: sample refuses a `coverage` member its key's schema refuses as `schema.sample_unsatisfiable`

**2026-10-23, PLAN-2026-10-metais-values-and-rules W2-NB1.** A follow-up
from Wave 2's verification, at the seam between W2-3 (`check`, MTS-AUT-081)
and W2-4 (the sample generator, MTS-SMP-010). By ruling R-W2-b, a member of
`coverage`'s `values` that the schema reached from the item schema by `key`
refuses is a case where the generator "cannot" make the array, since the
operation generates only values the schema allows (MTS-SMP-004).

- **One resolution.** The key-pointer resolution `check` used for
  MTS-AUT-081 to MTS-AUT-083 (the applicable set, a property segment, the
  item schemas and the key) moves from `internal/author` into
  `internal/schema` as `KeyResolver`. `check` and the generator both use
  it, so the members the generator refuses are the members `check`
  reports as `schema.unsatisfiable`.
- **The generator.** Before it makes an array whose schema holds such a
  `coverage`, the generator reports `schema.sample_unsatisfiable` with
  `reason` `coverage`, located at the schema holding the `coverage` and
  at the array's place, and makes no value, so the read-back reports no
  `schema.sample_invalid`. Its hint says to correct the value or let the
  key's schema accept it, never to add an `examples` value, which could
  not help.
- **The registry.** The `reason` parameter's meaning for `coverage` names
  both causes: more values than `maxItems` allows items, or a value the
  schema at its key refuses. The registry's copies are regenerated; no
  other line under `spec/` changes.
- **Proof.** `smp/sample-coverage-refused`, by `sample` and by a render
  with sample, and `internal/sample`'s `TestCoverageRefused`. A `coverage`
  whose members the key's schema all accepts is still generated as W2-4
  built it: `smp/sample-coverage`, `smp/sample-coverage-unique-keys` and
  `TestCoverageItems` are unchanged and green.

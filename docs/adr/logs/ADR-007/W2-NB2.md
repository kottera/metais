# ADR-007 — W2-NB2: check's rules across items are pinned through prefixItems

**2026-10-23, PLAN-2026-10-metais-values-and-rules W2-NB2.** A
non-blocking follow-up from Wave 2's verification of W2-3, which reads the
array's item schema of MTS-AUT-081 and MTS-AUT-083 as the `items` and each
`prefixItems` member of every node of the array schema's applicable set.
No test reached the item schema through `prefixItems` alone, so a resolution
that ignored it would have treated such a table as open and reported
nothing. `TestSchemaStaticsRules` now holds an array whose rows are declared
only by `prefixItems`: a `coverage` value neither member's key accepts is
`schema.unsatisfiable`, and a `uniqueKeys` pointer neither member declares
is `schema.key_not_declared`. No code and no line under `spec/` changed.

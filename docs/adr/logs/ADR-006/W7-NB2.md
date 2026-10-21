# ADR-006 — W7-NB2: Amendment A1's tripwire judges oneOf branches exclusive only when one allows only an object

**2026-10-20, PLAN-2026-10-metais-v1 W7-NB2.** W7-V found (W7-1) that
`TestManifestKeywordCodes` counted two `oneOf` branches exclusive when one
required a member the other, a closed object, did not allow, whatever
their `type`. With `"type": "object"` removed from the template and asset
output branches, it stayed green, while the binary answered `outputs: [x]`
with `internal.error`: a string matches both untyped branches, so the
`oneOf` reports `manifest.one_of.ambiguous`, which the registry lacks, and
`finding.New` panics. The schema as merged is correct, so the defect was
latent.

The tripwire now counts a pair exclusive only when one branch allows only
an object (`type` is `object`, or a list of it alone) and one requires a
member the other does not allow. Every other pair it counts as able to
both match, the stricter reading. `TestManifestKeywordCodesControl` holds
the mutation above, and was red before the change.

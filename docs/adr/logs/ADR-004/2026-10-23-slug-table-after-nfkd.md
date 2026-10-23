# ADR-004 — 2026-10-23-slug-table-after-nfkd: `slug` reads its letter table after NFKD

**2026-10-23, outside a train, after PLAN-2026-10-metais-v1 T-A4.**
Amendment A1 is implemented: `internal/template/filters`' `slug` decomposes
each code point, drops its Mn, and only then spells a table letter, so
`"Ǿrsted" | slug` is `orsted`. Chapter 8's MTS-FLT-010 states the new
order, and the vector `flt/slug-decomposed-letters` proves it, red on the
order D10 gave (`rsted`, `sir`) and green on this one (`orsted`,
`o-aesir-ae-ae-ae`). `flt/slug`'s outputs are unchanged.

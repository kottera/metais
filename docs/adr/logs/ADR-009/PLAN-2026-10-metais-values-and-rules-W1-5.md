# ADR-009 — PLAN-2026-10-metais-values-and-rules W1-5: D11's charge for `duration`

**2026-10-23, PLAN-2026-10-metais-values-and-rules W1-5.** MTS-TPL-063's
row for `number`, `amount` and `duration` landed with this item (ADR-004
Amendment A2 decision item 7), and `duration` makes its charges. It charges
16 units and the shallow sizes of its input and its format. Then it charges
the length of the string it builds, or 1 for nil, which is the result. The
work is linear in the format's length, which the format's shallow size has
already counted, as `date`'s is. `tpl/charge-duration` renders at its exact
bound, 43 units for a result and 39 for nil, and is refused one unit below
each.

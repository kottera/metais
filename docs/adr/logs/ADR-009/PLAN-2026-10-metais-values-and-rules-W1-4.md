# ADR-009 — PLAN-2026-10-metais-values-and-rules-W1-4: D11's charge for `number` and `amount`

**2026-10-23, PLAN-2026-10-metais-values-and-rules W1-4.** D11 item 2's
table gains ADR-004 A2's filters, as chapter 7's MTS-TPL-063 row states
them: `number` and `amount` charge, beyond the filter's 16 units, the
shallow sizes of their input and arguments before any work, and then the
result's length, or nil's shallow size, before the result is returned. No
work goes uncharged: a decimal text is read and rounded in time linear in
its length, which its shallow size counts. `tpl/charge-number` proves the
charge at its bound, 48 units, and refuses one below it.

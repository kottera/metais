# ADR-008 — W2-1: `metais help templates` teaches the value filters example first

**2026-10-23, PLAN-2026-10-metais-values-and-rules W2-1.** A6's templates
topic gains ADR-004 A2 decision item 6. `date`, `number`, `amount` and
`duration` move from "METAIS's own" to a group of their own, "Values for
readers", whose entries lead with chapter 8's example and what it gives,
then say what the filter does. `amount`'s entry ends with a line of its own:
there is no `money` filter, and `amount` takes the amount in the currency's
major unit, not in cents. `duration`'s says its directives mirror `date`'s.
Every filter is still documented exactly once, so
`TestTemplatesTopicFilters` holds as before, and `TestTemplatesTopicValues`
holds the new entries' order, examples and lines. The help golden and
`help.json` were regenerated; `help --json` carries the topic as text, so
no schema changed.

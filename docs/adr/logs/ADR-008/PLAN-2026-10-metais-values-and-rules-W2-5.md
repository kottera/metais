# ADR-008 — PLAN-2026-10-metais-values-and-rules W2-5: the field listing and `inspect` show rules across rows (ADR-003 A2 item 10)

**2026-10-23, PLAN-2026-10-metais-values-and-rules W2-5.** ADR-003
Amendment A2 item 10 amends D5 as A7 amends it, and D6. Its entry is
`docs/adr/logs/ADR-003/W2-5.md`. Here is what it changes in this ADR's
surfaces.

- **A7 item 3, the text.** Beneath a field's line, when its declaring
  schemas carry `uniqueKeys` or `coverage`, the listing writes one line per
  rule: the pointer, `rule`, the sentence and, under a condition, the
  field's `when` column, each column neutralised as every other is. Item
  5's cuts do not apply to a rule's line: its values are bounded by the
  package's `schema-size`, as A2 item 10 says, and the constraint the
  keyword also gives is cut as before.
- **A7 item 6, the JSON.** Each entry of `fields` gains `rules`, after
  `beyond_value_depth`, which D10 item 4 allows within `/1`. An entry now
  gives its field's line and its rules' lines exactly.
- **D6, `inspect`.** The input's line in the human summary names
  `metais schema <package> <input> --fields` when the input has rules
  across its rows. Whether an input has them is read from that same
  listing, to the run's value-depth, so `inspect` names the command
  exactly when it lists a rule.
- **The help.** `metais help schema`'s detail and `Prints`, and
  `metais help inspect`'s `Prints`, describe the new lines. The help
  goldens are regenerated.

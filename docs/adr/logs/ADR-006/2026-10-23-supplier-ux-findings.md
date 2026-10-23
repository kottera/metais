# ADR-006 — 2026-10-23-supplier-ux-findings: Amendment A4, declared hints for an enum, a format and an unquoted YAML scalar

**2026-10-23, outside a train.** This item carries out Ed's ruling of
2026-10-23 that the external UX test's findings land before v1.0.
Amendment A4 is proposed with it. It changes hints, which conformance does
not compare, and ADR-003 A3's refused input. No code, parameter or
location of any other finding changes.

- `internal/finding` gains `WithHintValues`: a declared hint whose own
  placeholders hold package text no parameter carries. The text is
  shortened as a parameter's value is, so the hint stays within 240 code
  points (`TestWithHintValues`).
- `internal/schemafail`'s `declaredHint` gives `input.enum` the near value
  when `params.suggestion` has one, under D9 item 7's rule as A3 item 3
  keeps it. Otherwise it lists every value when the values are few and
  short, and when they are not it gives their count and `metais schema
  <package> <input> --fields`. `input.format` gets the field's
  description, or else the pattern beside the format. `input.type` gets
  the quoting hint for a number or boolean that an unquoted YAML scalar
  gave, through `Target.Written`, which `internal/inputs` answers from a
  YAML document's nodes. `TestDeclaredHints` holds each cause and its
  control.
- `TestHostileFailuresReportUnchanged`'s default-limits digest for its
  first supply was taken again: its YAML item's numbers, where a string is
  wanted, now carry the quoting hint, and nothing else in the report
  changed. The diff against main was those hints alone.
- **Seen, not changed:** the validator returns at a failed `format`
  before the same schema object's `pattern`. So `example.com` for a
  `uri` field with `^https?://` gives one finding, and the `pattern`
  failure is not reported. Item 3's hint names the stricter rule. Whether
  D6's "every failed assertion" needs the validator to go on is a
  separate question.

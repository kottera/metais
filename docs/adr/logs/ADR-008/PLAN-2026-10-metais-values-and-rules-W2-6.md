# ADR-008 — PLAN-2026-10-metais-values-and-rules W2-6: the `schemas` help topic (ADR-003 A2 item 10)

**2026-10-23, PLAN-2026-10-metais-values-and-rules W2-6.** ADR-003
Amendment A2 item 10 adds a topic under D15 item 4. Its entry is
`docs/adr/logs/ADR-003/W2-6.md`.

- **D15 item 4.** `schemas` joins the topics table after `manifest`, so
  the overview lists it, `metais help --json` carries its text, and help's
  operand names it. `metais help help` gains its example, and
  `TestAuthoringTopicsReachable` holds it to the places the authoring
  topics are found, the skill among them. Its golden is
  `internal/cli/testdata/help/topic-schemas.txt`, and the site's
  reference pages, generated from `metais help --json`, gain its page.
- **`metais help schema`**'s detail ends with a pointer to the topic.
- **D15's vendor-neutrality.** The topic and the skill pass
  `TestVendorNeutral`.

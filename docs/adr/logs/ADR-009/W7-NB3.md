# ADR-009 — W7-NB3: A4 item 1's count for an empty document pinned

**2026-10-20, PLAN-2026-10-metais-v1 W7-NB3.** W7-V found Amendment A4
item 1's count for an empty YAML document and for empty frontmatter
unpinned. Each counts what it decodes to: an empty YAML document is null,
one value, its root; a Markdown file's object is one value whether its
frontmatter is absent, empty or only a comment, so each of those counts
two with its body. No behaviour changes.

- **`internal/inputs`' `TestInputValuesCount`** reads each of those
  documents, and a one-member one beside each, through the reader and
  holds `input-values`' counter to the count.
- **`sec/input-values`** gains four cases where the count is observable:
  an empty `y.yaml` passes at 1 and is refused at 0, and an `m.md` of
  empty frontmatter passes at 2 and is refused at 1, at its body.

Dropping the count of `yamlx`'s empty document turns both red: the test's
empty-document and empty-frontmatter rows, and both vector cases that
refuse one value over.

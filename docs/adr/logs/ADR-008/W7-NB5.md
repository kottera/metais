# ADR-008 — W7-NB5: `inspect --guide`'s help line says what it prints

**2026-10-20, PLAN-2026-10-metais-v1 W7-NB5.** D6 item 9 has `--guide`
print only `package` and the whole guide in JSON, and the guide's
Markdown alone in human output. The behaviour followed it, and
`TestInspectGuide` holds both forms. The flag's help line said it printed
the identity and the guide, which only the JSON form does. The help was
wrong: the line now reads "Print only the whole guide: its Markdown
alone, or with --json the package's identity and the guide.", and the
help goldens are regenerated from it.

# ADR-008 — PLAN-2026-10-metais-public W2-2: the reference pages are generated from the help and the skill (Amendment A13)

**2026-10-23, PLAN-2026-10-metais-public W2-2.** Amendment A13 is
implemented. The address W2-2 already names another plan's entry in this
directory, so this file carries the plan's name.

- **The pages (item 1).** The site package renders, from the `help` input
  that the build step exports as `metais help --json`'s result,
  `/reference/` (the overview `metais help` prints: the golden paths, the
  commands by group, the global flags, the topics, the exit codes and the
  refusals of a line that names no command), one page per command at
  `/reference/commands/<name>/`, and one page per topic at
  `/reference/topics/<name>/`, `templates` and `manifest` among them. The
  skill's page, `/reference/skill/`, is the body of `metais skill --print`
  as the build step converts it. Nothing on them is written by hand but
  the layout's headings and labels.
- **Item 2 and item 3.** `site/reference_test.go`'s `TestReferencePages`
  holds each page to the help the build step exported from the metais
  built from the same commit: one page per command and topic and the
  skill's, and no other; each command's page and the overview say exactly
  what the help says, in its order, with nothing but the layout's words
  beside it; a topic's text is byte for byte; the skill's body is the
  converted one. It decodes the help with every field known, so a field
  `metais.help/1` gains fails it until the pages show the field.
  `TestReferenceHeldToHelp` pins the check false first.
- **Item 4.** No page reads the text help, and the generator needed
  nothing `metais help --json` does not carry, so no amendment adds to
  `metais.help/1`. `opens_package` and `role` are the two fields no page
  shows, since `metais help` prints neither.

# ADR-011 — W7-NB2: the link tripwires read a link whose text wraps

**2026-10-20, PLAN-2026-10-metais-v1 W7-NB2.** W7-V found (W7-2) that
`TestLinksResolve` matched links one line at a time, so a link whose text
wraps onto the next line was never checked; two in
`spec/10-transforms.md` did, and resolve. D14's link checks,
`TestLinksResolve` and `TestCitationsResolve`, now match each run of lines
that holds no blank line and no fence as one text, and report a link at
the line it begins on. `TestLinksResolveControl` holds a wrapped broken
link to a heading and one to a document, and was red before the change.

Appendix B §B.9 now says that the packed package begins with the stored
`README.md` (ADR-001 Amendment A1); its record is in
`logs/ADR-001/W7-NB2.md`.

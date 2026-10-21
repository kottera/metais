# ADR-001 — W7-NB2: Appendix B §B.9 says the packed package begins with the stored README.md

**2026-10-20, PLAN-2026-10-metais-v1 W7-NB2.** W7-C7 made `README.md`,
stored, the first entry of every packed package (Amendment A1), and
regenerated `gen/worked-package`'s archive, whose first entry is the
stored `README.md`. Appendix B §B.9 still said the package held
the files written above and `.compiled/`, and nothing compares that prose
with the archive. §B.9 now says the packed package begins with
`README.md`, stored, holding the README text, citing §2.4.1 and
MTS-PKG-071, and that `.compiled/` and the package's files follow in
ascending order of their names (MTS-AUT-047, MTS-PKG-010). No statement
and no vector lists the archive's entries, so none changed.

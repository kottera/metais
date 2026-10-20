# ADR-007 — W6-NB1: a top-level directory named *.metais is not packed

**2026-10-20, PLAN-2026-10-metais-v1 W6-NB1.** MTS-AUT-047 leaves out a
top-level regular file whose name ends in `.metais` with no finding, and
reports every other top-level name outside the layout as
`package.layout.not_packed`. The vectors `aut/pack` and
`aut/check-source-tree` now hold a top-level directory `archive.metais/`,
which pack and check report as `package.layout.not_packed`. Without
`internal/hostfs`'s regular-file test both vectors fail.

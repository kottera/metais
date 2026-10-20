# ADR-011 — W7-R: the release builder refuses a tool whose implements names an unreleased revision

**2026-10-20, PLAN-2026-10-metais-v1 W7-R.** D5 item 6 is carried by the
release builder of ADR-010 D12. It builds the tool for the host from the
tagged commit, reads `implements` from `metais version --json`, and refuses
unless every revision `M.m` there is released as D5 item 2 releases one:
an annotated tag `spec-M.m` on an ancestor of `HEAD`, and
`spec/conformance/released/M.m.sha256` in `HEAD`'s tree. A lightweight
tag, a tag off `HEAD`'s history, a missing released list, an empty
`implements` and a name that is not `<major>.<minor>` are each refused by
a test. No statement or vector is added: the release procedure is the
tool's, not a behaviour of a package or of a conforming program that the
specification states.

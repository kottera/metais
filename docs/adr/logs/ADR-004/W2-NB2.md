# ADR-004 — W2-NB2: an example in yen pins DORA's amount code to the input

**2026-10-23, PLAN-2026-10-metais-values-and-rules W2-NB2.** A
non-blocking follow-up from Wave 2's verification of W2-2. Every example
of `examples/dora-incident-report` reported in euros, so a template that
wrote the code `EUR` itself, instead of taking it from
`entity.reporting_currency` (1.15), rendered the same bytes and
`TestExamplePackages` stayed green. The package's new example
`final_in_yen` is the final report of the same incident by an entity that
reports in `JPY`, a currency with no minor unit: its amounts are written
`JPY 384,500`, with W2-2's `group: ","` and no decimals. A literal code in
any of the three `amount` calls now makes `metais test` report
`example.output_differs`. No template, schema or line under `spec/`
changed.

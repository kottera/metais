# ADR-013 — 2026-10-25-landing-flow: the landing page's hero shows a package and data going in and documents coming out (D8 items 3 and 4)

**2026-10-25, outside a train, for PLAN-2026-10-metais-public.** Written
at the train's close, 2026-10-26, by W4-3: the change was made in Ed's
session with no train address and wrote no entry. This one is drawn from
its merged pull request, #344 (merged as `6bb95fa1`, rc.6's target).

At Ed's request, D8 item 3's landing page opens on one figure of the
whole flow: the package and your data go in, `metais` checks and renders,
and `summary.html` and one invoice per record come out. The figure is real
output of a new example, `examples/invoices`, which renders with
`for_each`; `TestLandingFlow` renders it and requires the figure to show
what it renders, and `TestExamplePackages` runs it like every other
example.

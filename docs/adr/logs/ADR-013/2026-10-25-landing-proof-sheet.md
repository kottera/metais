# ADR-013 — 2026-10-25-landing-proof-sheet: the proof-sheet design becomes metais.org's landing page (D2 item 1, D8 items 3 and 4)

**2026-10-25, outside a train, for PLAN-2026-10-metais-public.** Written
at the train's close, 2026-10-26, by W4-3: the change was made in Ed's
session with no train address and wrote no entry. This one is drawn from
its merged pull request, #337 (merged as `389bbe01`, rc.4's target).

D8 item 3's landing page is Ed's design "Data in, documents out", with
his corrections, approved in his session on 2026-10-24: its markup in
`site/package/templates/index.html.liquid` and `site/public/landing.css`,
verbatim from the mockup except where the site requires otherwise. The
version pill reads the release the site is built for. The footer's maker
line and the "Who makes METAIS" section stay (D2 item 1, `TestFooter`,
`TestTeraOnlyWhoMakes`). The new `TestLandingDemos` packs
`examples/invoice` with the site's own metais and requires every line of
the demos' output on the page, so the page cannot drift from the tool
(D8 item 4's rule, applied to the landing page).

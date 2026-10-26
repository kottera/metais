# ADR-013 — 2026-10-25-site-restyle: every page of metais.org restyled to the landing page's design (D2 item 1, D8 items 3 and 4)

**2026-10-25, outside a train, for PLAN-2026-10-metais-public.** Written
at the train's close, 2026-10-26, by W4-3: the change was made in Ed's
session with no train address and wrote no entry. This one is drawn from
its merged pull request, #343 (merged as `883fa06b`, rc.5's target).

The install page, both guides, the reference, the specification and the
`go-import` pages move onto the landing page's design: one header and one
footer partial shared with the landing page, one stylesheet `site.css`
(`style.css` is retired), and a side navigation outside `main`. D8 item
4's tests hold as before: `<main>` holds only the page's own text, so the
reference pages are still held word for word to the help; a draft's
`main` begins with its notice; every page ends with one footer, which
names the maker (D2 item 1); and every side link names an id on its page.
The landing page is pixel-identical to its build before the change.

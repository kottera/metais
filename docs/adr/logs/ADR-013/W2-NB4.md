# ADR-013 — W2-NB4: the tap's CI pins setup-homebrew by commit (D7 item 5)

**2026-10-24, PLAN-2026-10-metais-public W2-NB4.** Written at the train's
close, 2026-10-26, by W4-3: the item's change is in another repository,
and it wrote no entry here. This one is drawn from its merged pull
request.

W2-NB4 (`01a16bb8-26eb`), a non-blocking finding of Wave 2's verification
of W2-13, merged kottera/homebrew-tap#3 as `7535de63`. The tap's CI named
`Homebrew/actions/setup-homebrew@main`, a branch. It now names commit
`4790763c0a3ba4036f72634dc079c4a7dfcd4878`, Homebrew/actions' release
`2026.10.19.1`, with the release in a comment, the same rule W3-8 later
held this repository's workflows to. It is the only `uses:` in the tap's
workflows. The pull request's own CI and `main`'s push run after it were
the proof; the record names no run ids.

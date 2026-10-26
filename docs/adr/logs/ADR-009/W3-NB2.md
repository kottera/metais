# ADR-009 — W3-NB2: the credential route's refusal of an encoded path is tested (A12)

**2026-10-26, PLAN-2026-10-metais-public W3-NB2.** Written at the train's
close, 2026-10-26, by W4-3: the item wrote no entry, and this one is drawn
from its merged pull request, #358 (merged as `20a31081`).

W3-V's mutation of W3-4 found A12's route refusing a request whose path
carries an encoded separator (a `RawPath`) with no test that fails when
the refusal is removed. `internal/tools/ux/route_test.go` now sends
`/v1/a%2Fb` and requires a 403 with nothing forwarded upstream. With the
disjunct removed, the test fails with status 200 and one request
forwarded. A12's rule is unchanged. The item's other changes are
[ADR-013/W3-NB2.md](../ADR-013/W3-NB2.md)'s.

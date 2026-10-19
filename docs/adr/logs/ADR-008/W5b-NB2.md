# ADR-008 — W5b-NB2: help refusal follow-ups

**2026-10-19, PLAN-2026-10-metais-v1 W5b-NB2.** Two follow-ups from Wave
5b's verification. `TestFamiliesGroupByNextStep` in
`internal/cli/help_test.go` pins `families()`'s grouping key: two refusals
of one code family and one gist that name different next steps are two
family lines. Dropping `f.next == r.Next` from the key turns it red, which
no golden did, because no current declaration has such a pair. Check's
generated causes put the indefinite article that agrees with the severity
after it, through `withArticle` in `internal/cli/check.go`: "an error",
"a warning", "an info". `TestCauseArticle` holds every command's causes and
gists to it. The text and JSON help goldens were regenerated; no vector,
other golden or finding changed.

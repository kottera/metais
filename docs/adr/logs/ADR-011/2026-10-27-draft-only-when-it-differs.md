# ADR-011 — 2026-10-27-draft-only-when-it-differs: the site serves `/spec/draft/` only while `spec/` differs from the latest revision

**2026-10-27, outside a train, on the ruling of PLAN-2026-10-metais-public's
W4-NB1 design session (01a17a59-a7ba-7c25-8b34-7f915ea3c1df), recorded as
A10.** `internal/tools/site` exports `draft_served` in the site's `spec`
input: false when the checkout's `spec/` tree at `HEAD` is the latest
`spec-<M>.<m>` tag's and the working tree's `spec/` has no changed,
untracked or ignored file, and true otherwise or with no tag
(`TestSpecInputDraftServed`, one case per way of differing, a later commit
outside `spec/`, and the latest by number, not by name). While it is false,
`spec/draft.html.liquid` renders each draft page as a page leading to the
same chapter of the latest revision, or to its contents, with a refresh, a
canonical link and `noindex`, as `/spec/` leads to the latest. The header's
Spec link and the footer's Specification link lead to `/spec/`, and the
landing page names the latest revision at `/spec/` and the draft only while
it is served. `site/reference_test.go` pins both builds:
`TestSpecReleases` renders two releases with the draft served and requires
draft pages, and `TestSpecDraftRedirects` renders them with the draft
identical and requires every draft URL to lead to 1.10's chapter or
contents. `TestSpecPages` holds the real build to whichever its input
says. `rehearse.sh spec` compares the tag's `spec/` tree with its latest
`spec-<M>.<m>` tag's and checks the leading pages or the draft pages
accordingly; `TestRehearseScript` runs each branch and each refusal, and
`TestRehearseWorkflow` requires the tag to be checked out with its whole
history before either check of the specification runs. `pages.yml` is
unchanged, so the live site follows with the next release.

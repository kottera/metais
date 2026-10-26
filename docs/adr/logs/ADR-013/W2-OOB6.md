# ADR-013 — W2-OOB6: how the release environment holds a release run (D7 items 3.2 and 3.4)

**2026-10-25, PLAN-2026-10-metais-public W2-OOB6.** Written at the train's
close, 2026-10-26, by W4-3: the act's pull request wrote no entry. This
one is drawn from that pull request, #350, merged as `8fd018d9`.

W2-OOB6 (`01a16bb8-56c8`) carried the GitHub-hosted runs Wave 2 deferred
to the flip. The rc.7 rehearsal's runs stood in for them:
- ci.yml's `site`, `install-linux`, `install-macos` and `shell` jobs, in
  run 40237935459, a push to `main`, linux/amd64 included;
- `release.yml`'s dry run against the rc.7 draft, run 40238216267, which
  skipped the push of tags, the publish, the formula and the pages job;
- `pages.yml`'s refusal at "the tag's release is published", run
  40238309953, before rc.7 was published.

The dry run and the rc.7 release run 40242455407 each waited for Ed's
approval of the `release` environment before running. `environment:
release` holds a dry run as it holds a release, and #350 states that as
the rule in `docs/dev/release.md`. The act is outside Wave 3's tree, and
its own outputs were not read for this entry.

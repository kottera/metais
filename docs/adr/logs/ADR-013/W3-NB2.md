# ADR-013 — W3-NB2: the UX runner's checks, scorecard and install procedure repaired from the rehearsal (D9)

**2026-10-26, PLAN-2026-10-metais-public W3-NB2.** Written at the train's
close, 2026-10-26, by W4-3: the item wrote no entry, and this one is drawn
from its merged pull request, #358 (merged as `20a31081`).

Six non-blocking findings of Wave 3 (W3-OOB7's, W3-7's and W3-V's) on the
runner D9's scenarios run on, each with code held by a test seen red
first:
1. `no-memory-read` excuses a memory file's name only inside a directory
   the session's own successful `git clone`, `gh repo clone` or `git init`
   made, never the session's directory, HOME or anything above them. Re-scored,
   I2-discover on `opus[1m]` in `scorecard-2026-10-25-install` would no
   longer be invalid on this ground; the published scorecard is not
   rewritten.
2. The scorecard's Thresholds line counts gated and recorded runs apart.
   No verdict changes, and D9's gate still reads I1 alone.
3. The scorecard's header names the runner's build apart from the metais
   each session installed and ran.
4. The credential route's refusal of an encoded path is tested (see
   [ADR-009/W3-NB2.md](../ADR-009/W3-NB2.md)).
5. `docs/dev/ux-testing.md`'s re-score example passes `-scenarios I1,I2`,
   and `TestDocumentedCommands` holds every runner command it gives.
6. Writes outside the run are handled by a documented procedure: list
   `/tmp` before and after a run and remove what the session made.

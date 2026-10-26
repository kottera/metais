# ADR-013 — W3-OOB4: the first release acts, v1.0.0-rc.1 to rc.3 (D7 items 3.2 to 3.4 and 8)

**2026-10-24, PLAN-2026-10-metais-public W3-OOB4.** Written at the train's
close, 2026-10-26, by W4-3. W3-OOB4 (`01a16bc5-fcc5`) is Ed's act, filed
as rc.1's release act, which became rc.1 to rc.3. It changed no file
itself; its fixes are W3-9, W3-10 and W3-11, each with its own entry. This
entry is drawn from the Wave 3 closeout and the rehearsal records W3-7
wrote from GitHub (`docs/dev/rehearsal/v1.0.0-rc.1.md` to `rc.3.md`).

D7 item 3.2's release act ran for the first time on a public repository:
- **rc.1:** `release.yml` would not parse (W3-9, #333). Once it did, the
  approved run 40123046882 was refused at the push of the tag, because
  the target's workflows were not `main`'s (W3-10, #335). Nothing was
  pushed, and rc.1 was retired unpublished.
- **rc.2:** at `39184c8e`. Release run 40133852927, approved by Ed at
  21:21:38Z, pushed the tag at 21:22:00Z, published it and updated the
  tap (`158ccff5`). Rehearsal run 40134009587 failed `go install` alone,
  on module zip paths (W3-11, #336, ADR-010 A11).
- **rc.3:** at `a3ec4b9b`. Release run 40137517145 pushed the tag
  (object `7cb15565`) at 22:12:38Z, published it, updated the tap
  (`8c4b93b0`) and deployed the site. Rehearsal run 40137683223 passed 8
  of 8, the first candidate to pass.

Each candidate's dry run waited for Ed's approval of the `release`
environment like a real run, and `pages.yml` refused each draft before
it was published (D7 item 3.4). The act also switched on `main`'s
required checks. I1 and I2 did not run against these candidates.

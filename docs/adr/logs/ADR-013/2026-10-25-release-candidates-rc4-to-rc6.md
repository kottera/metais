# ADR-013 — 2026-10-25-release-candidates-rc4-to-rc6: three candidates released in Ed's session (D7 items 3.2 to 3.4 and 8)

**2026-10-25, outside a train's acts, for PLAN-2026-10-metais-public Wave
3.** Written at the train's close, 2026-10-26, by W4-3. Ed cut and
released `v1.0.0-rc.4`, `rc.5` and `rc.6` in his own session, outside any
Work Order, so they have no train address and changed no file. This entry
is drawn from the rehearsal records W3-7 wrote from GitHub
(`docs/dev/rehearsal/v1.0.0-rc.4.md` to `rc.6.md`) and the Wave 3
closeout. The operator's report of rc.4 named only its release time and
rehearsal run, so how its builder ran is not in the record.

| Candidate | Target | Carried | Release run | Tap | Rehearsal run |
|---|---|---|---|---|---|
| rc.4 | `389bbe01` | #337, the landing page | 40156483998 | `be4e7ce0` | 40156566776, 8 of 8 |
| rc.5 | `883fa06b` | #343, every page restyled | 40160842892 | `7ddd6a2f` | 40160943855, 8 of 8 |
| rc.6 | `6bb95fa1` | #344, the landing page's hero; W3-4 | 40197592477 | `5525dc62` | 40197727122, 7 of 8 |

Each followed D7 item 3: a dry run, `pages.yml`'s refusal of the draft,
then a release run Ed approved, which pushed the tag, published the
prerelease, updated the tap and deployed the site. rc.6 failed
`install-macos` on GitHub's anonymous rate limit, which R-W3-4 ruled a
defect, fixed by W3-13. I1 and I2 ran on rc.6 (W3-OOB5) and are superseded
by rc.7's. The tool is unchanged since rc.3 in all three.

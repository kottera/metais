# ADR-010 — T-A4: Amendments A1, A3 and A4 accepted, and A9 writes the train's decisions in

**2026-10-22, PLAN-2026-10-metais-v1 T-A4.** Ed Funnekotter ratified the
train's outcome.

- **A1, A3 and A4 are accepted.** Their *(A1)*, *(A3)* and *(A4)* markers
  no longer say proposed, and the header names each as accepted. A3's
  Status line names the plan orchestrator, under plan §4, as the one who
  ruled it in Wave 2.
- **A4 gains item 2**, plan §8's placement of `date`: `internal/dates`
  implements it and `internal/template` registers it through `Extend`.
  D4 item 2 says so in place. The filter list D4 item 2 spells out keeps
  `date`, because `TestCoreIsADR004Catalog` compares it with ADR-004's
  catalog.
- **A9** lists every fault in D13 item 2's table (bits 5 to 10, from
  Waves 5 and 6), names `internal/tarw` as the tar writer in D11 item 4,
  D12 item 5, D14 item 5 and the promise to ADR-005, and records Ed's
  ruling that a gate resumed after its host starved is the same gate.
- **The pseudo-terminal test's home in `internal/hostfs`** (W6-4) is
  ratified; W6-A1 ran it on darwin and linux.
- **The other readings in this ADR's logs** (the data model in
  `internal/liquid/value`, the dependency-scan reading of W1, R-W5-11's
  `RemoveEmptyDir`, W6's fault allotments) are ratified as recorded.

# ADR-003 — T-A4: Amendment A1, a duplicated member name is one finding

**2026-10-22, PLAN-2026-10-metais-v1 T-A4.** Ed Funnekotter ratified the
train's outcome, and with it Wave 3's reading of D9 item 2 against ADR-006
D8 item 3 (W3-9, plan §9). Amendment A1 writes it into D9 item 2: repeats
of one name in one object are one finding under ADR-006's de-duplication
key. `json.duplicate_name` keeps its `name` parameter, which Wave 3's
ruling read as a member name, not a value, under ADR-006 D9 item 4. What
is accepted does not change, and no test changes: the shipped reader
already reports this way.

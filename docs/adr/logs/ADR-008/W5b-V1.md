# ADR-008 — W5b-V1: check's generated refusals all on family lines

**2026-10-19, PLAN-2026-10-metais-v1 W5b-V1.** A1 items 2 and 3 now hold
for every registry-loop refusal. `checkRefusals` used one map, `what`, both
to name a family's subject and to choose the families whose every warning
and note check declares. The output family was in neither role, so
`output.declaration.validate_kind`, which check declares as one of its own
steps' codes, got an empty subject, no gist, and its own line in
`metais check --help` with a generated cause. The two roles are now
separate: `what` names a subject for every family check's loop reaches,
output's included, a `wholesale` list keeps the families declared
wholesale as they were, and every refusal the loop builds carries a gist.
Check declares the same codes as before; only that code's cause gains its
subject and its listing moves to a family line.
`TestHelpListsNoGeneratedRefusalOnItsOwn` reads every command's text help
and fails when a refusal on its own line carries its code's own words in
parentheses, or when a gist names no subject; it was red at 8834408.

# ADR-008 — W6-NB1: pack's advice on a second end record, and grep's truncation line

**2026-10-20, PLAN-2026-10-metais-v1 W6-NB1.** When pack's read-back
refuses the package it built with `package.container.second_end_record`
(R1-014), the finding's code, members and exit status are a reader's, as
they were, and nothing is written. The registry's hint, written for every
reader of a package, says to pack the package again, which writes the
same bytes. So pack declares its own row for the cause, `packSecondEnd`,
with a cause, a hint and a next step (D14 item 4). A `Refusal` carries an
optional `Hint`, and `finding.Finding.WithHint` renders a finding's hint
from that declared template in place of the code's, under the registry
templates' rules. The row's next step names the entry that holds the
signature, in its `why`, quoted for a shell. `zipfmt.WriteParts` reports
each entry's local record and central directory header as byte ranges,
and `author.Packed.EntryAt` maps the finding's `offset` to the entry
whose records hold it. The operand's own open, which no source directory
reaches, keeps the code's row.

grep's human truncation line names what capped the results: `--max-results`,
given or by its default of 100, or `the grep-matches limit` when a lowered
limit set the default below 100. The JSON result is unchanged.

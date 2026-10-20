# The criteria tables

Each standard the report answers gets one table, supplied as a CSV file
in the data directory: wcag.csv, section508.csv and en301549.csv. Which of
them you supply is decided by the report's edition
([report details](report.md)): wcag.csv always, section508.csv for the
section508 and international editions, and en301549.csv for the en301549
and international editions.

Every table has the same three columns, and the first line of the file
names them:

```csv
criterion,conformance,remarks
1.1.1,Partially Supports,"Product images have text alternatives. The chart on the dashboard has none, so its figures are not available to screen reader users."
1.2.1,Not Applicable,The product has no audio-only or video-only content.
2.1.1,Supports,Every function can be operated from a keyboard.
```

- **criterion** is the criterion's number, exactly as its standard
  numbers it: `1.1.1` for WCAG 2.2, `302.1` for the Revised Section 508
  Standards, and `4.2.1` for EN 301 549. Validation refuses a number the
  standard does not have. The package's references list every number each
  table takes, with what the criterion asks: references/wcag-2.2.md,
  references/section-508.md and references/en-301-549.md.
- **conformance** is how far the product meets the criterion: one of
  `Supports`, `Partially Supports`, `Does Not Support`, `Not Applicable`
  and `Not Evaluated`, written exactly so. [What the levels
  mean](conformance.md) says which to choose.
- **remarks** says what the evaluation found. For Supports, say how the
  product meets the criterion; for Partially Supports and Does Not
  Support, say which functions fail and how; for Not Applicable, say why
  the criterion does not concern the product. A field holding a comma, a
  quote or a line break is written between double quotes, with each quote
  inside it doubled (`""`); a line break starts a new paragraph of the
  remarks. Leave the field empty only when there is nothing to say.

A row speaks for the whole product. Where its parts meet a criterion
differently, such as its web pages, its documents and its software, give
the level the product as a whole earns and say in the remarks how each
part fares.

## Which criteria to list

List each criterion of the table once, in the standard's order, which is
the order of its reference file. A complete report lists every criterion
of the table, so that a reader can tell a criterion met from one left
out: write Not Applicable for a criterion that does not concern the
product, and Not Evaluated for one nobody evaluated. The rows appear in
the report in the order the file gives them.

The **WCAG 2.2** table lists the success criteria at Levels A and AA, and
those at Level AAA the vendor chose to evaluate. The report gives each
criterion's level beside its name. It also answers the parts of the other
two standards that apply WCAG to web pages, documents and software, so
those standards' tables leave them out.

The **Revised Section 508** table lists the criteria of chapters 3 to 6:
the functional performance criteria (302), hardware (402 to 415), software
(502 to 504) and support documentation and services (602 and 603). For
software, chapter 4's hardware criteria are Not Applicable.

The **EN 301 549** table, for EN 301 549 V4.1.1 (2026-09), lists the
clauses outside clauses 9, 10 and 11.1 to 11.4, which the WCAG table
answers: the functional performance statements (4.2), the generic
requirements, authoring tools among them (5), real-time voice, text and
video communication (6), video playback (7), hardware (8), software's
interoperability with assistive technology and its accessibility
settings (11.5 to 11.7), information and documentation about the product
and its services (12), and relay services (13).

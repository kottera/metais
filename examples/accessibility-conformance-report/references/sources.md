# Sources and their terms

This package reports against three published standards. From them it
takes only what each section below names: identifiers, and from WCAG
2.2, short excerpts under W3C's notice. Every other text in it is the
METAIS project's own, written for it: the manifest and its guide, the
input schemas and meaning documents, the templates, the summaries of
the reference files and the EN 301 549 clause names, the definitions of
the conformance levels, the JSON report's schema, and the examples'
data. The example's product, vendor and findings are invented, and
`example.com` is a domain reserved for documentation by RFC 2606.

The project's own text is under the project's licence, the Apache
License 2.0, as the manifest's `license` says, with the repository's
`NOTICE`. The WCAG excerpts stay W3C's, under the W3C Document License,
and the Revised Section 508 headings are a United States Government work.

No text of the Information Technology Industry Council's Voluntary Product
Accessibility Template (VPAT) is used: not its instructions, its
definitions of the conformance levels, its table headings or its layout.
The five conformance levels are named as PRD-001 §4.1 of the METAIS
project names them, and defined here in the project's own words.

Each source below is named with its publisher, where it is published,
what this package takes from it, the terms it is published under, the
notice those terms require, and the verdict of the METAIS project's terms
check (plan §6, act W7-A1, ruled 2026-10-20) on how its text may be
reused here.

## Web Content Accessibility Guidelines (WCAG) 2.2

- **Publisher:** World Wide Web Consortium (W3C), W3C Recommendation of
  5 October 2023, updated 12 December 2024,
  <https://www.w3.org/TR/2024/REC-WCAG22-20241212/>.
- **What this package takes:** each success criterion's number, its name
  and its conformance level, copied as they are, in
  references/wcag-2.2.md, the label partials and the input schema. Every
  summary of what a criterion asks is the project's own words, not W3C's
  text.
- **Terms:** the W3C Document License (2023),
  <https://www.w3.org/copyright/document-license-2023/>. It permits
  copying and distributing the document's contents, in any medium for any
  purpose and without fee or royalty, provided every copy of the document
  or of portions of it includes a link or URL to the original W3C
  document, its pre-existing copyright notice, and its status. It grants
  no right to modify W3C's text, except in software and its
  documentation.
- **Notice required, and given in references/wcag-2.2.md:** "Copyright ©
  2020-2024 World Wide Web Consortium. W3C® liability, trademark and
  document use rules apply." with the link above and the status "W3C
  Recommendation 12 December 2024".
- **Verdict:** excerpts only, verbatim, with that notice. Numbers, names
  and levels may be copied as they are; summaries stay in the project's
  own words. W3C's name is not used to imply that W3C endorses this
  package.

## Revised Section 508 Standards

- **Publisher:** U.S. Access Board, 36 CFR Part 1194, the Information and
  Communication Technology Standards and Guidelines, appendices A and C,
  <https://www.access-board.gov/ict/>.
- **What this package takes:** each criterion's number and heading, in
  references/section-508.md, the label partials and the input schema; the
  summaries are the project's own words.
- **Terms:** a work of the United States Government. 17 U.S.C. § 105:
  "Copyright protection under this title is not available for any work of
  the United States Government".
- **Notice required:** none. Cited as 36 CFR Part 1194, U.S. Access
  Board.
- **Verdict:** verbatim reuse permitted. The standards incorporate WCAG
  2.0 at Levels A and AA by reference (E205.4, 702.10.1); that WCAG text
  stays under W3C's terms, above.

## EN 301 549

- **Publisher:** European Telecommunications Standards Institute (ETSI),
  with CEN and CENELEC, EN 301 549 V4.1.1 (2026-09), *Accessibility
  requirements for ICT products and services*,
  <https://www.etsi.org/deliver/etsi_en/301500_301599/301549/04.01.01_60/en_301549v040101p.pdf>.
  It supersedes V3.2.1 and brings in WCAG 2.2.
- **What this package takes:** each clause's number, as an identifier.
  Every clause name and summary in references/en-301-549.md, the label
  partials and the input schema is the project's own description; none is
  ETSI's heading or text.
- **Terms:** the deliverable's copyright notification forbids
  reproducing it without ETSI's written authorization. Its only exception
  covers extracts, such as templates, strictly necessary for the
  technical implementation of the deliverable.
- **Notice required:** none, since nothing of it is reproduced. The
  standard, its edition and its address are named above.
- **Verdict:** own words. Clause numbers are used as identifiers only.

## OpenACR

- **Publisher:** United States General Services Administration (GSA),
  OpenACR, its catalogue and its schema, <https://github.com/GSA/openacr>,
  as checked at commit 86f890a.
- **What this package takes:** nothing. acr.json is this package's own
  shape, given by references/acr.schema.json, and is not an OpenACR
  document; it copies neither OpenACR's catalogue nor its schema.
- **Terms:** a United States Government work in the public domain, and
  dedicated worldwide under CC0 1.0. The WCAG files it embeds are W3C's,
  under W3C's terms.
- **Notice required:** none.
- **Verdict:** may be reproduced freely. This package takes nothing from
  it.

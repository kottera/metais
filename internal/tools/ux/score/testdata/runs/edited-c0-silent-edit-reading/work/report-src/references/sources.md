# Sources and their terms

This package carries the data glossary of the EU's template for reporting a
major ICT-related incident as its schemas. Its field numbers, the field
names in its schemas' titles, the field types (each schema property's
`x-field-type`) and the choice lists are reproduced verbatim from Annex II
to Implementing Regulation (EU) 2025/302, below. Everything else is written in the METAIS
project's own words for this package: the manifest and its guide, every
field's description, the meaning document, the template and the examples.
The examples' incident, entities, people and figures are invented.

The package is licensed with the rest of the repository, under the Apache
License 2.0, as its manifest's `license` says. That licence covers the
project's own words. It does not cover the legal texts, which remain under
the terms below.

The legal texts are reproduced in part, and their meaning is not changed.
Only European Union documents published in the Official Journal of the
European Union are deemed authentic. This
package is not an official publication, and it is not a source of the law.

## Regulation (EU) 2022/2554 (DORA)

- **Text:** Regulation (EU) 2022/2554 of the European Parliament and of
  the Council of 14 December 2022 on digital operational resilience for
  the financial sector, OJ L 333, 27.12.2022, p. 1.
- **Where:** <https://eur-lex.europa.eu/eli/reg/2022/2554/oj>
- **What this package uses:** Article 19's three reports on a major
  ICT-related incident, which are the stages. Field 1.4's choice list,
  the financial entities of Article 2(1), is taken as Annex II words it.
- **Reuse terms:** the EU's reuse policy, Commission Decision 2011/833/EU
  of 12 December 2011 on the reuse of Commission documents (OJ L 330,
  14.12.2011, p. 39), as EUR-Lex's legal notice
  (<https://eur-lex.europa.eu/content/legal-notice/legal-notice.html>)
  applies it: "Unless otherwise specified, you can re-use the legal
  documents published in EUR-Lex for commercial or non-commercial
  purposes." The conditions, from Article 6(2) of the Decision, are that
  the source is acknowledged, that the original meaning or message of the
  document is not distorted, and that the Commission is not liable for any
  consequence of the reuse.
- **Notice:** © European Union, https://eur-lex.europa.eu/, 1998–2026
- **Terms check (W7-A1, 2026-10-20):** reuse permitted with
  acknowledgement; field names and choice lists reproduced verbatim;
  descriptions in the project's own words.

## Commission Implementing Regulation (EU) 2025/302, Annex II

- **Text:** Commission Implementing Regulation (EU) 2025/302 of 23 October
  2024 laying down implementing technical standards for the application
  of Regulation (EU) 2022/2554 with regard to the standard forms,
  templates, and procedures for financial entities to report a major
  ICT-related incident and to notify a significant cyber threat,
  OJ L, 2025/302, 20.2.2025. Annex I is the template; Annex II, "Data
  glossary and instructions for the reporting of major incidents", is the
  glossary.
- **Where:** <https://eur-lex.europa.eu/eli/reg_impl/2025/302/oj>
- **What this package uses:** each field's number, its name, its type,
  the stage at which it is mandatory, the condition on which a conditional
  field is mandatory, and its choice list. Field 1.1, the type of
  submission, is the stage input, with the three values that name the
  reports; the rest are the properties of the entity and the
  incident.
- **Reuse terms:** the EU's reuse policy, Commission Decision 2011/833/EU
  of 12 December 2011 on the reuse of Commission documents (OJ L 330,
  14.12.2011, p. 39), as EUR-Lex's legal notice
  (<https://eur-lex.europa.eu/content/legal-notice/legal-notice.html>)
  applies it: "Unless otherwise specified, you can re-use the legal
  documents published in EUR-Lex for commercial or non-commercial
  purposes." The conditions, from Article 6(2) of the Decision, are that
  the source is acknowledged, that the original meaning or message of the
  document is not distorted, and that the Commission is not liable for any
  consequence of the reuse.
- **Notice:** © European Union, https://eur-lex.europa.eu/, 1998–2026
- **Terms check (W7-A1, 2026-10-20):** reuse permitted with
  acknowledgement; field names and choice lists reproduced verbatim;
  descriptions in the project's own words.

The glossary's choice lists for fields 2.5, 3.13 and 3.18 name the
classification criteria of Commission Delegated Regulation (EU) 2024/1772
and the kinds of impact it defines. The package takes those names from the
glossary, as part of Annex II, and does not apply the criteria.

## Standards the field types name

ISO 8601 (dates and times), ISO 4217 (currency codes), ISO 3166-1 alpha-2
(country codes) and ISO 17442 (the LEI) are named by the glossary as
formats. The package checks values against their published shapes and
reproduces no text of theirs.

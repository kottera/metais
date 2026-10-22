# The incident, stage by stage

One incident is reported three times, and each report states everything
known at its stage. The stage, field 1.1, decides which fields the report
must hold:

| Stage | Mandatory |
|---|---|
| `initial notification` | the mandatory fields of section 1 (the entity) and section 2 |
| `intermediate report` | those, and the mandatory fields of section 3 |
| `final report` | those, and the mandatory fields of section 4 |

Every property of the entity and the incident is titled with its field's
number and its name in the glossary, and carries the number as `x-field`
and the glossary's type as `x-field-type`. Its description says what it
holds and when it is mandatory.

## Fields mandatory on a condition

Some fields are mandatory only when another field shows they apply. The
package checks each condition the data itself shows:

| When | Then also give |
|---|---|
| 1.5, the affected entity's name, is given | 1.6, its LEI |
| 1.13, the parent undertaking's name, is given | 1.14, its LEI |
| 2.5 includes geographical spread | 2.6; and from the intermediate report on, 3.18 and 3.19 |

From the intermediate report on:

| When | Then also give |
|---|---|
| 2.5 includes reputational impact | 3.13 and 3.14 |
| 2.5 includes duration and service downtime | 3.17 |
| 2.5 includes data losses | 3.20 and 3.21 |
| 3.16, the service downtime, is given | 3.3 |
| 3.23 includes Other (please specify) | 3.24 |
| 3.23 includes Cybersecurity-related | 3.25 and 3.35 |
| 3.25 includes Other (please specify) | 3.26 |
| 3.28 is Yes | 3.29 |
| 3.31 includes Other (please specify) | 3.32 |
| 3.33 is true | 3.34 |

In the final report:

| When | Then also give |
|---|---|
| 4.2 includes a category 4.3 refines: monitoring and control, ICT risk management, ICT operations and ICT security operations, or ICT systems acquisition, development and maintenance | 4.3 |
| 4.2 includes an other (please specify) | 4.4 |
| 4.15 is given | 4.16 |

The other conditional fields turn on facts the report does not otherwise
hold: 1.5 when the affected entity is not the one submitting the report,
1.13 when the entity belongs to a group, 2.8 when the incident originates
from a third-party provider or another financial entity, 3.1 once the
competent authority has assigned a reference, 3.8 when the relevance
threshold for clients and financial counterparts is met, 3.9 to 3.11 when
any transaction was affected, and so on. Give each when its description
says it applies; the package cannot tell, and does not ask.

## Updating a report

Each report updates the one before it, so keep one data directory per
incident and amend it:

1. For the initial notification, set `stage` to `initial notification`
   and fill in sections 1 and 2.
2. For the intermediate report, set `stage` to `intermediate report`,
   add section 3's fields, and correct anything in sections 1 and 2 that
   has changed. 2.1, the incident's reference code, never changes.
3. For the final report, set `stage` to `final report` and add section 4.

A field due only at a later stage may be given early, as soon as it is
known. It is then checked against its type, and the fields it makes
mandatory become due at the stage the glossary makes them due.

Dates and times are in UTC, written `2026-03-02T06:40:00Z`. Durations are
days, hours and minutes, written `DD:HH:MM`. Amounts are in the currency
1.15 names, and percentages are numbers from 0 to 100.

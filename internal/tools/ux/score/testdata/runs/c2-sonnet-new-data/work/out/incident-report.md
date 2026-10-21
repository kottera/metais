# Major ICT-related incident report VMF-INC-2026-0117

- **1.1 Type of submission:** Final report

## 1. The financial entity

- **1.2 Name of the entity submitting the report:** Vastmark Försäkring AB
- **1.3 Identification code of the entity submitting the report:** 894500VASTMARK000031
- **1.4 Type of the affected financial entity:** insurance and reinsurance undertaking
- **1.7 Primary contact person name:** Astrid Lindqvist
- **1.8 Primary contact person email:** astrid.lindqvist@vastmark.example
- **1.9 Primary contact person telephone:** +46 8 5550 3301
- **1.10 Second contact person name:** Mattias Holm
- **1.11 Second contact person email:** mattias.holm@vastmark.example
- **1.12 Second contact person telephone:** +46 8 5550 3302
- **1.13 Name of the ultimate parent undertaking:** Vastmark Group Oy
- **1.14 LEI code of the ultimate parent undertaking:** 894500VASTMARKGRP158
- **1.15 Reporting currency:** SEK

## 2. Initial notification

- **2.1 Incident reference code assigned by the financial entity:** VMF-INC-2026-0117
- **2.2 Date and time of detection of the ICT-related incident:** 2026-05-18T13:20:00Z
- **2.3 Date and time of classification of the incident as major:** 2026-05-18T15:00:00Z
- **2.4 Description of the ICT-related incident:** From 12:50 UTC on 18 May, policyholders in Sweden and Finland could not log in to the claims portal after an expired TLS certificate on the identity provider blocked authentication. The certificate was replaced at 16:40 UTC and the portal was fully available again at 17:05 UTC.
- **2.5 Classification criteria that triggered the incident report:** clients, financial counterparts and transactions affected; reputational impact; duration and service downtime; geographical spread; critical services affected
- **2.6 Materiality thresholds for the classification criterion ‘Geographical spread’:** FI; SE
- **2.7 Discovery of the major ICT-related incident:** monitoring systems
- **2.9 Activation of business continuity plan, if activated:** Yes

## 3. Intermediate report

- **3.1 Incident reference code provided by the competent authority:** FI-FSA-DORA-2026-088
- **3.2 Date and time of occurrence of the incident:** 2026-05-18T12:50:00Z
- **3.3 Date and time when services, activities or operations have been recovered:** 2026-05-18T17:05:00Z
- **3.4 Number of clients affected:** 9300
- **3.5 Percentage of clients affected:** 6.1
- **3.6 Number of financial counterparts affected:** 0
- **3.7 Percentage of financial counterparts affected:** 0
- **3.8 Impact on relevant clients or financial counterparts:** No
- **3.9 Number of affected transactions:** 5400
- **3.10 Percentage of affected transactions:** 8.8
- **3.11 Value of affected transactions:** 1240
- **3.12 Information on whether the numbers are actual or estimates, or whether there has not been any impact:** actual figures for clients affected; estimates for transactions affected; no impact on financial counterparts
- **3.13 Reputational impact:** the major ICT-related incident has been reflected in the media; the major ICT-related incident has resulted in repetitive complaints from different clients or financial counterparts on client-facing services or critical business relationships
- **3.14 Contextual information about the reputational impact:** A trade publication reported the outage, and the service desk received about 700 complaints.
- **3.15 Duration of the incident:** 00:04:15
- **3.16 Service downtime:** 00:04:15
- **3.17 Information on whether the numbers for duration and service downtime are actual or estimates:** Actual figures
- **3.18 Types of impact in the Member States:** clients; branch of the financial entity
- **3.19 Description of how the incident has an impact in other Member States:** Policyholders of the Helsinki branch could not file or track claims for the whole of the incident.
- **3.22 Classification criterion ‘Critical services affected’:** Online claims filing and tracking for policyholders.
- **3.23 Type of the incident:** Process failure; System failure
- **3.27 Information about affected functional areas and business processes:** The claims portal, and the service desk, which handled the complaints.
- **3.28 Affected infrastructure components supporting business processes:** Yes
- **3.29 Information about affected infrastructure components supporting business processes:** The identity provider and the certificate store that feeds the portal gateway.
- **3.30 Impact on the financial interest of clients:** No
- **3.31 Reporting to other authorities:** None
- **3.33 Temporary actions/measures taken or planned to be taken to recover from the incident:** Yes
- **3.34 Description of any temporary actions and measures taken or planned to be taken to recover from the incident:** The expired certificate was replaced by hand, and the portal gateway was put under manual certificate checks until automatic renewal is in place.

## 4. Final report

- **4.1 High-level classification of root causes of the incident:** process failure
- **4.2 Detailed classification of root causes of the incident:** process failure: insufficient or failure of ICT operations and ICT security operations
- **4.3 Additional classification of root causes of the incident:** ICT asset management and information classification
- **4.5 Information about the root causes of the incident:** The identity provider certificate was not on the renewal calendar after a migration, so nobody renewed it before it expired.
- **4.6 Incident resolution:** Automatic renewal is now active for every certificate, and an inventory check runs weekly against the renewal calendar.
- **4.7 Date and time when the incident root cause was addressed:** 2026-05-25T14:00:00Z
- **4.8 Date and time when the incident was resolved:** 2026-05-26T08:30:00Z
- **4.9 Information if the permanent resolution date of the incidents differs from the initially planned implementation date:** The permanent fix was put in place on 25 May, the date planned in the intermediate report.
- **4.12 Materiality threshold for the classification criterion ‘Economic impact’:** The gross costs and losses stayed well below the economic impact threshold.
- **4.13 Amount of gross direct and indirect costs and losses:** 95
- **4.14 Amount of financial recoveries:** 0

# Major ICT-related incident report NWB-INC-2026-0042

- **1.1 Type of report:** Intermediate report

## 1. The financial entity

- **1.2 Name of the entity submitting the report:** Nordwind Bank AG
- **1.3a Identification code of the entity submitting the report:** 529900NORDWIND000017
- **1.3b Type of the identification code of the entity submitting the report:** LEI
- **1.4 Type of the affected financial entity:** credit institution
- **1.7 Primary contact person name:** Lena Hartmann
- **1.8 Primary contact person email:** lena.hartmann@nordwind-bank.example
- **1.9 Primary contact person telephone:** +49 69 5550 1201
- **1.10 Second contact person name:** Tomás Ferreira
- **1.11 Second contact person email:** tomas.ferreira@nordwind-bank.example
- **1.12 Second contact person telephone:** +49 69 5550 1202
- **1.13 Name of the ultimate parent undertaking:** Nordwind Holding SE
- **1.14 LEI code of the ultimate parent undertaking:** 529900NORDWINDHLD045
- **1.15 Reporting currency:** EUR

## 2. Initial notification

- **2.1 Incident reference code provided by the financial entity:** NWB-INC-2026-0042
- **2.2 Date and time of detection of the incident:** 2026-03-02T06:40:00Z
- **2.3 Date and time of classification of the incident as major:** 2026-03-02T08:10:00Z
- **2.4 Description of the incident:** From 06:15 UTC on 2 March, card payments by retail clients in Germany and Austria were declined after a scheduled configuration change to the card payment gateway disabled its fallback route. The change was rolled back at 11:05 UTC and card payments were fully restored at 11:30 UTC.
- **2.5 Classification criteria that triggered the incident report:** clients, financial counterparts and transactions affected; reputational impact; duration and service downtime; geographical spread; critical services affected
- **2.6 Materiality thresholds for the classification criterion 'Geographical spread':** AT; DE
- **2.7 Discovery of the incident:** monitoring systems
- **2.9 Activation of business continuity plan, if activated:** Yes

## 3. Intermediate report

- **3.1 Incident reference code provided by the competent authority:** BAFIN-DORA-2026-117
- **3.2 Date and time of occurrence of the incident:** 2026-03-02T06:15:00Z
- **3.3 Date and time when services, activities and/or operations have been restored:** 2026-03-02T11:30:00Z
- **3.4 Number of clients affected:** 18500
- **3.5 Percentage of clients affected:** 4.2
- **3.6 Number of financial counterparts affected:** 0
- **3.7 Percentage of financial counterparts affected:** 0
- **3.8 Impact on relevant clients or financial counterparts:** No
- **3.9 Number of affected transactions:** 41200
- **3.10 Percentage of affected transactions:** 12.5
- **3.11 Value of affected transactions:** 2350
- **3.12 Information whether the numbers are actual or estimates, or whether there has not been any impact:** actual figures and estimates
- **3.13 Reputational impact:** the incident has been reflected in the media; the incident has resulted in repeated complaints from clients or financial counterparts
- **3.14 Contextual information about the reputational impact:** Two national newspapers reported the declined payments, and the client service desk received about 1,900 complaints.
- **3.15 Duration of the incident:** 00:05:15
- **3.16 Service downtime:** 00:05:15
- **3.17 Information whether the numbers for duration and service downtime are actual or estimates:** actual figures
- **3.18 Types of impact in the Member States:** clients; branch of the financial entity
- **3.19 Description of how the incident has an impact in other Member States:** Clients of the Vienna branch could not pay by card for the whole of the incident.
- **3.22 Classification criterion 'Critical services affected':** Card payment processing for retail clients.
- **3.23 Type of the incident:** process failure; system failure
- **3.27 Information about affected functional areas and business processes:** Retail card payments, and the client service desk, which handled the complaints.
- **3.28 Affected infrastructure components supporting business processes:** yes
- **3.29 Information about affected infrastructure components supporting business processes:** The card payment gateway cluster and the configuration service that deploys its settings.
- **3.30 Impact on the financial interest of clients:** no
- **3.31 Reporting to other authorities:** none
- **3.33 Temporary actions/measures taken or planned to be taken to recover from the incident:** Yes
- **3.34 Description of any temporary actions and measures taken or planned to be taken to recover from the incident:** The previous gateway configuration was restored, and configuration changes to the gateway are frozen until the root cause is addressed.

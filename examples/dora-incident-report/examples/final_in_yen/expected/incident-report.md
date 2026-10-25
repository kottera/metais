# Major ICT-related incident report NWB-INC-2026-0042

- **1.1 Type of submission:** Final report

## 1. The financial entity

- **1.2 Name of the entity submitting the report:** Nordwind Bank AG
- **1.3 Identification code of the entity submitting the report:** 529900NORDWIND000017
- **1.4 Type of the affected financial entity:** credit institution
- **1.7 Primary contact person name:** Lena Hartmann
- **1.8 Primary contact person email:** lena.hartmann@nordwind-bank.example
- **1.9 Primary contact person telephone:** +49 69 5550 1201
- **1.10 Second contact person name:** Tomás Ferreira
- **1.11 Second contact person email:** tomas.ferreira@nordwind-bank.example
- **1.12 Second contact person telephone:** +49 69 5550 1202
- **1.13 Name of the ultimate parent undertaking:** Nordwind Holding SE
- **1.14 LEI code of the ultimate parent undertaking:** 529900NORDWINDHLD045
- **1.15 Reporting currency:** JPY

## 2. Initial notification

- **2.1 Incident reference code assigned by the financial entity:** NWB-INC-2026-0042
- **2.2 Date and time of detection of the ICT-related incident:** 2 March 2026, 06:40:00 UTC
- **2.3 Date and time of classification of the incident as major:** 2 March 2026, 08:10:00 UTC
- **2.4 Description of the ICT-related incident:** From 06:15 UTC on 2 March, card payments by retail clients in Germany and Austria were declined after a scheduled configuration change to the card payment gateway disabled its fallback route. The change was rolled back at 11:05 UTC and card payments were fully restored at 11:30 UTC.
- **2.5 Classification criteria that triggered the incident report:** clients, financial counterparts and transactions affected; reputational impact; duration and service downtime; geographical spread; critical services affected
- **2.6 Materiality thresholds for the classification criterion ‘Geographical spread’:** AT; DE
- **2.7 Discovery of the major ICT-related incident:** monitoring systems
- **2.9 Activation of business continuity plan, if activated:** Yes

## 3. Intermediate report

- **3.1 Incident reference code provided by the competent authority:** BAFIN-DORA-2026-117
- **3.2 Date and time of occurrence of the incident:** 2 March 2026, 06:15:00 UTC
- **3.3 Date and time when services, activities or operations have been recovered:** 2 March 2026, 11:30:00 UTC
- **3.4 Number of clients affected:** 18,500
- **3.5 Percentage of clients affected:** 4.2 %
- **3.6 Number of financial counterparts affected:** 0
- **3.7 Percentage of financial counterparts affected:** 0.0 %
- **3.8 Impact on relevant clients or financial counterparts:** No
- **3.9 Number of affected transactions:** 41,200
- **3.10 Percentage of affected transactions:** 12.5 %
- **3.11 Value of affected transactions:** JPY 330,276,000
- **3.12 Information on whether the numbers are actual or estimates, or whether there has not been any impact:** actual figures for clients affected; estimates for transactions affected; no impact on financial counterparts
- **3.13 Reputational impact:** the major ICT-related incident has been reflected in the media; the major ICT-related incident has resulted in repetitive complaints from different clients or financial counterparts on client-facing services or critical business relationships
- **3.14 Contextual information about the reputational impact:** Two national newspapers reported the declined payments, and the client service desk received about 1,900 complaints.
- **3.15 Duration of the incident:** 0 d 5 h 15 min
- **3.16 Service downtime:** 0 d 5 h 15 min
- **3.17 Information on whether the numbers for duration and service downtime are actual or estimates:** Actual figures
- **3.18 Types of impact in the Member States:** clients; branch of the financial entity
- **3.19 Description of how the incident has an impact in other Member States:** Clients of the Vienna branch could not pay by card for the whole of the incident.
- **3.22 Classification criterion ‘Critical services affected’:** Card payment processing for retail clients.
- **3.23 Type of the incident:** Process failure; System failure
- **3.27 Information about affected functional areas and business processes:** Retail card payments, and the client service desk, which handled the complaints.
- **3.28 Affected infrastructure components supporting business processes:** Yes
- **3.29 Information about affected infrastructure components supporting business processes:** The card payment gateway cluster and the configuration service that deploys its settings.
- **3.30 Impact on the financial interest of clients:** No
- **3.31 Reporting to other authorities:** None
- **3.33 Temporary actions/measures taken or planned to be taken to recover from the incident:** Yes
- **3.34 Description of any temporary actions and measures taken or planned to be taken to recover from the incident:** The previous gateway configuration was restored, and configuration changes to the gateway are frozen until the root cause is addressed.

## 4. Final report

- **4.1 High-level classification of root causes of the incident:** process failure; human error
- **4.2 Detailed classification of root causes of the incident:** process failure: insufficient or failure of ICT operations and ICT security operations; human error: mistake
- **4.3 Additional classification of root causes of the incident:** change management
- **4.5 Information about the root causes of the incident:** A configuration value that disables the gateway's fallback route was changed by mistake, and the change review did not compare the new configuration with the running one.
- **4.6 Incident resolution:** The configuration was corrected, the review now compares every change with the running configuration, and changes to the gateway need two reviewers.
- **4.7 Date and time when the incident root cause was addressed:** 9 March 2026, 16:00:00 UTC
- **4.8 Date and time when the incident was resolved:** 10 March 2026, 09:00:00 UTC
- **4.9 Information if the permanent resolution date of the incidents differs from the initially planned implementation date:** The permanent fix was put in place on 9 March, the date planned in the intermediate report.
- **4.12 Materiality threshold for the classification criterion ‘Economic impact’:** The gross costs and losses stayed well below the economic impact threshold.
- **4.13 Amount of gross direct and indirect costs and losses:** JPY 6,953,000
- **4.14 Amount of financial recoveries:** JPY 0

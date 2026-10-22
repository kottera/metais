# Major ICT-related incident report INC-2026-0042

- **1.1 Type of submission:** Initial notification

## 1. The financial entity

- **1.2 Name of the entity submitting the report:** Example Bank AG
- **1.3 Identification code of the entity submitting the report:** 529900EXAMPLE0000012
- **1.4 Type of the affected financial entity:** credit institution; payment institution; exempted payment institution
- **1.5 Name of the financial entity affected:** Example Payments GmbH
- **1.6 LEI code of the financial entity affected:** 529900EXAMPLE0000034
- **1.7 Primary contact person name:** Alex Example
- **1.8 Primary contact person email:** alex.example@example.com
- **1.9 Primary contact person telephone:** +49 69 1234567
- **1.10 Second contact person name:** Sam Example
- **1.11 Second contact person email:** sam.example@example.com
- **1.12 Second contact person telephone:** +49 69 7654321
- **1.13 Name of the ultimate parent undertaking:** Example Holding SE
- **1.14 LEI code of the ultimate parent undertaking:** 529900EXAMPLE0000056
- **1.15 Reporting currency:** EUR

## 2. Initial notification

- **2.1 Incident reference code assigned by the financial entity:** INC-2026-0042
- **2.2 Date and time of detection of the ICT-related incident:** 2026-03-02T06:40:00Z
- **2.3 Date and time of classification of the incident as major:** 2026-03-02T08:10:00Z
- **2.4 Description of the ICT-related incident:** Card payments failed for clients of the bank after a configuration change to the payment gateway.
- **2.5 Classification criteria that triggered the incident report:** clients, financial counterparts and transactions affected; reputational impact; duration and service downtime
- **2.6 Materiality thresholds for the classification criterion ‘Geographical spread’:** AT; BE; BG
- **2.7 Discovery of the major ICT-related incident:** IT Security
- **2.8 Indication whether the incident originates from a third-party provider or another financial entity:** Example Cloud Ltd;529900EXAMPLE0000078;LEI
- **2.9 Activation of business continuity plan, if activated:** Yes
- **2.10 Other relevant information:** The gateway's previous configuration has been restored.

## 3. Intermediate report

- **3.1 Incident reference code provided by the competent authority:** CA-2026-117
- **3.2 Date and time of occurrence of the incident:** 2026-03-02T06:15:00Z
- **3.3 Date and time when services, activities or operations have been recovered:** 2026-03-02T11:30:00Z
- **3.4 Number of clients affected:** 18500
- **3.5 Percentage of clients affected:** 4.2
- **3.6 Number of financial counterparts affected:** 3
- **3.7 Percentage of financial counterparts affected:** 0.5
- **3.8 Impact on relevant clients or financial counterparts:** No
- **3.9 Number of affected transactions:** 41000
- **3.10 Percentage of affected transactions:** 12.5
- **3.11 Value of affected transactions:** 2350
- **3.12 Information on whether the numbers are actual or estimates, or whether there has not been any impact:** actual figures for clients affected; actual figures for financial counterparts affected; actual figures for transactions affected
- **3.13 Reputational impact:** the major ICT-related incident has been reflected in the media; the major ICT-related incident has resulted in repetitive complaints from different clients or financial counterparts on client-facing services or critical business relationships; the financial entity will not be able to or is likely not to be able to meet regulatory requirements as a result of the major ICT-related incident
- **3.14 Contextual information about the reputational impact:** Two national newspapers reported the failed payments.
- **3.15 Duration of the incident:** 00:05:15
- **3.16 Service downtime:** 00:04:50
- **3.17 Information on whether the numbers for duration and service downtime are actual or estimates:** Actual figures
- **3.18 Types of impact in the Member States:** clients; financial counterparts; branch of the financial entity
- **3.19 Description of how the incident has an impact in other Member States:** Clients of the Austrian branch could not pay by card.
- **3.20 Materiality thresholds for the classification criterion ‘Data losses’:** availability; authenticity; integrity
- **3.21 Description of the data losses:** Payment records for the affected window were written incompletely.
- **3.22 Classification criterion ‘Critical services affected’:** Card payment processing for retail clients.
- **3.23 Type of the incident:** Cybersecurity-related; Process failure; System failure
- **3.24 Other types of incidents:** Failed vendor upgrade.
- **3.25 Threats and techniques used by the threat actor:** Social engineering (including phishing); (D)DoS; Identity theft
- **3.26 Other types of techniques:** Credential stuffing against the administration portal.
- **3.27 Information about affected functional areas and business processes:** Retail payments and the client service desk.
- **3.28 Affected infrastructure components supporting business processes:** Yes
- **3.29 Information about affected infrastructure components supporting business processes:** The payment gateway cluster and its configuration service.
- **3.30 Impact on the financial interest of clients:** Yes
- **3.31 Reporting to other authorities:** Police/Law Enforcement; CSIRT; Data Protection Authority
- **3.32 Specification of ‘other’ authorities:** The national consumer protection agency.
- **3.33 Temporary actions/measures taken or planned to be taken to recover from the incident:** Yes
- **3.34 Description of any temporary actions and measures taken or planned to be taken to recover from the incident:** The previous gateway configuration was restored and changes were frozen.
- **3.35 Indicators of compromise:** 198.51.100.23; sha256:3f1d...

## 4. Final report

- **4.1 High-level classification of root causes of the incident:** malicious actions; process failure; system failure / malfunction
- **4.2 Detailed classification of root causes of the incident:** malicious actions: deliberate internal actions; malicious actions: deliberate physical damage/manipulation/theft; malicious actions: fraudulent actions
- **4.3 Additional classification of root causes of the incident:** monitoring of policy adherence; monitoring of third-party service providers; monitoring and verification of remediation of vulnerabilities
- **4.4 Other types of root cause types:** An undocumented dependency between two configuration keys.
- **4.5 Information about the root causes of the incident:** A configuration change disabled the gateway's fallback route.
- **4.6 Incident resolution:** The change was reverted and configuration changes now need two reviewers.
- **4.7 Date and time when the incident root cause was addressed:** 2026-03-09T16:00:00Z
- **4.8 Date and time when the incident was resolved:** 2026-03-10T09:00:00Z
- **4.9 Information if the permanent resolution date of the incidents differs from the initially planned implementation date:** The fix waited for the vendor's patch.
- **4.10 Assessment of risk to critical functions for resolution purposes:** No critical function for resolution purposes was at risk.
- **4.11 Information relevant for resolution authorities:** None.
- **4.12 Materiality threshold for the classification criterion ‘Economic impact’:** The costs and losses stayed below the economic impact threshold.
- **4.13 Amount of gross direct and indirect costs and losses:** 180
- **4.14 Amount of financial recoveries:** 40
- **4.15 Information on whether the non-major incidents have been recurring:** Two earlier failures of the same gateway, each below the thresholds.
- **4.16 Date and time of occurrence of recurring incidents:** 2026-01-14T10:20:00Z

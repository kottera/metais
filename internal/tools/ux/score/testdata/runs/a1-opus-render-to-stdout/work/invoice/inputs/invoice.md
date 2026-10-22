# Invoice

The invoice is everything on the page except the line items: who bills
whom, the reference and dates, the currency, and the optional tax and
notes.

- **number** is the invoice's unique reference, such as "INV-2026-0042".
  Never reuse one: a buyer's accounts match payments by it.
- **issue_date** is the date the invoice is issued, and **due_date** the
  date payment is due, both as YYYY-MM-DD. Leave out the due date when
  the payment terms say when to pay.
- **currency** is the ISO 4217 code every price is in, such as USD, EUR
  or GBP. It is printed beside the totals; the package converts nothing.
- **seller** is who issues the invoice and is paid; **buyer** is who it
  is billed to. Each has a **name**, and optionally an **address** as a
  list of lines, an **email** and a **tax_id** (a VAT or GST number,
  which many jurisdictions require on an invoice).
- **tax** applies one rate to the subtotal: **rate** is a percentage, so
  20 means 20%, and **label** is its printed name, "Tax" when left out.
  Leave tax out entirely when none is charged; the invoice then shows no
  tax line. Ask the seller for the rate: never assume one.
- **payment_terms** says how to pay: bank details, terms such as
  "Net 30", or where to pay online.
- **notes** is a short closing message, such as a thank-you.

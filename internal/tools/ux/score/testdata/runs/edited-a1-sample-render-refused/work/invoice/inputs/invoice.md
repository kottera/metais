# The invoice

The header of one invoice: who bills whom, its number and dates, and the
terms every amount is computed on. Take each value from the seller's
records; none of them is computed.

- **number** is the seller's invoice number, unique among their invoices.
- **issue_date** is the day the invoice is issued; **due_date**, when
  given, the day payment is due. Both are `YYYY-MM-DD`.
- **currency** is the ISO 4217 code every line item's price is in. The
  invoice prints amounts with two decimal places, so a currency without
  cents, such as JPY, prints `.00`.
- **tax_rate** is the percentage charged on the subtotal, after every
  discount line. Leave it out, or give 0, when no tax is charged: the
  invoice then prints no tax line. **tax_label** names it (VAT, GST).
- **seller** and **bill_to** each hold a **name**, and optionally an
  **address** (one line per line of the address), an **email** and a
  **tax_id**.
- **payment_terms** and **notes** are printed at the foot, as written.

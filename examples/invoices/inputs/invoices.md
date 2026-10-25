# Invoices

A batch of invoices, as the business that issues them records them, in the
order the summary lists them, at least one. Each invoice has:

- **number**, the invoice's number, as the business numbers its invoices:
  `2026-0042`, or whatever scheme it uses, of letters, digits, `.`, `_`
  and `-`, starting with a letter or a digit. It names the invoice's page,
  `2026-0042.html`, so every invoice in the batch needs a different one.
- **date**, the date the invoice is issued, as an RFC 3339 full date:
  `2026-10-23`.
- **customer**, the name of the customer the invoice is made out to, as it
  should be printed.
- **lines**, the invoice's line items, in the order they are printed, at
  least one. Each has:
  - **description**, what was sold;
  - **quantity**, how many, a whole number from 1;
  - **unit_price**, the price of one, in cents, as a whole number: `1999`
    for $19.99.

Each page prints a line's amount, its quantity times its unit price, in
dollars. The summary prints each invoice's total, the sum of its lines'
amounts, and the total of every invoice.

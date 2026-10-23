# Invoice

One invoice, as the business that issues it records it.

- **number** is the invoice's number, as the business numbers its
  invoices: `2026-0042`, or whatever scheme it uses.
- **date** is the date the invoice is issued, as an RFC 3339 full date:
  `2026-10-23`.
- **customer** is the name of the customer the invoice is made out to, as
  it should be printed.
- **lines** are the invoice's line items, in the order they are printed,
  at least one. Each has:
  - **description**, what was sold;
  - **quantity**, how many, a whole number from 1;
  - **unit_price**, the price of one, in cents, as a whole number: `1999`
    for $19.99. The page prints each line's amount, its quantity times its
    unit price, in dollars.

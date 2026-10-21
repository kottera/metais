# Invoice

The invoice is one bill from a seller to a customer, made of line items.

- **number** is the invoice number as the seller numbers it, such as "INV-2026-014".
- **issue_date** and **due_date** are calendar dates (YYYY-MM-DD). The due date
  is when payment is expected; leave it out if payment is due on receipt.
- **currency** is the ISO 4217 code, such as "EUR". All amounts use it.
- **seller** and **customer** each have a name and an optional address (one line
  per entry) and an optional email.
- **items** are the lines billed, in the order printed. Each has a description,
  a quantity, and the unit price. The line total is quantity times unit price.
- **tax_rate** is the percentage added to the subtotal, such as 20 for 20%.
  Leave it out, or use 0, when no tax applies.
- **notes** is optional text printed under the totals, such as payment details.

Amounts are decimal numbers in the major unit (12.5 means 12.50). Never
supply totals: the document computes subtotal, tax and total.

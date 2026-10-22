# The line items

What the invoice bills, one item per row, printed in the order given.
As CSV, the header row is `description,quantity,unit,unit_price`; leave
`unit` empty where it does not apply.

- **description** says what the item is, as the customer should read it.
- **quantity** is how many units are billed. It must be more than zero;
  fractions such as 1.5 hours are allowed.
- **unit** is what one unit is (hours, pcs), printed after the quantity.
- **unit_price** is the price of one unit, to the cent, in the invoice's
  currency. A negative price is a discount or credit: it reduces the
  subtotal, and tax is charged on what remains.

Each line's amount is quantity times unit price, rounded to the cent. The
subtotal is the sum of the lines' amounts.

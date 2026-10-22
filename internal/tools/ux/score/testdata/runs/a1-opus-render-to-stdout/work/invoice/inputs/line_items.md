# Line items

The line items are the things billed, one per row, printed in the order
given. As a CSV, the first row names the columns:

    description,quantity,unit,unit_price
    "Design workshop, half day",1,day,850
    Development,12.5,hour,95
    Returning-client discount,1,,-100

- **description** is what is billed, as the buyer should read it.
- **quantity** is how many units, greater than zero; fractions such as
  12.5 hours are allowed.
- **unit** is optional: what one unit is, such as hour or day.
- **unit_price** is the price of one unit in the invoice's currency, to
  the cent, without tax. A negative price makes the line a discount.

Never supply a line's amount or any total: the package computes each
line's amount as quantity times unit price, rounded to the cent, and then
the subtotal, the tax and the total from those.

# Recipe

The recipe is one dish, as a home cook would make it. It gives the
heading, the ingredients and the method of the recipe card, and every
line of the shopping list.

- **name** is the dish's name as a cook would say it, such as
  "Buttermilk pancakes". It is not a file name or a menu code.
- **slug** is a short name of lower-case letters and digits joined by
  hyphens, such as "buttermilk-pancakes". It names the card's file,
  `buttermilk-pancakes.md`.
- **serves** is how many people the recipe feeds, a whole number.
- **minutes** is the total time from starting to serving, resting and
  baking included, a whole number of minutes.
- **tip** is optional: one or two sentences of advice that makes the dish
  turn out well. Leave it out when there is nothing a cook would miss.
- **ingredients** lists everything the recipe uses, at least one, in the
  order the method uses them. Each has an **amount**, with its unit, such
  as "200 g" or "2", and an **item**, what to buy, such as "plain flour".
  Write the amount "2" and the item "eggs", not the item "two eggs".
- **steps** is the method, one instruction per step, in order. "Whisk the
  eggs into the buttermilk" is a good step; "Make the batter" is not.

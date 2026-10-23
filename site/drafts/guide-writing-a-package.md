# Writing a package

A package is a template that tells whoever fills it in exactly what to
supply, and refuses what does not fit before any document is made. This
guide writes one: an invoice, rendered as an HTML page from its line
items. Then it changes a package that is already packed, and writes data
into a page's script safely.

It assumes `metais` is [installed](/install/).

## The route

1. Start from a scaffold: `metais init`.
2. Edit the manifest, the inputs and the templates, and preview them with
   `metais render --sample`.
3. Check the package: `metais check`.
4. Test its examples: `metais test`.
5. Pack it into one file: `metais pack`.

Every command ends with a `Next:` list of the commands to run after it.
`metais help <command>` explains any command in full, and two topics
cover what you write: `metais help manifest` gives every field of
`metais.yaml` with an example, and `metais help templates` gives the
template language, every filter, and how escaping works.

## 1. Start from a scaffold

    metais init invoice --minimal --html

This writes a package's source directory, `invoice`, that already checks,
tests and packs:

- `metais.yaml`, the manifest: the package's name, version, title,
  description and guide, its inputs and its outputs;
- `inputs/data.schema.json`, one input's JSON Schema, and
  `inputs/data.md`, the document that says what the input means;
- `templates/page.html.liquid`, one template.

`--minimal` gives you one input and one template to replace. Without it,
`init` writes a complete small package, a greeting with an example, to
learn from. `--html` declares the output as HTML, so that every value the
template writes is escaped for HTML.

## 2. Edit it

**The manifest.** Give the package a description and a guide. The guide is
what a supplier reads first, in `metais inspect`: say what the package
produces and what to supply. Name the input for what it holds:

    inputs:
      - name: invoice
        title: "Invoice"
        schema: inputs/invoice.schema.json
        meaning: inputs/invoice.md
        forms:
          - json
          - yaml
    outputs:
      - id: page
        template: invoice.html.liquid
        escape: html

Rename the scaffold's files to match: `inputs/invoice.schema.json`,
`inputs/invoice.md` and `templates/invoice.html.liquid`.

**The schema** is the contract. Every value a supplier gives is checked
against it, so make it as strict as the document needs: which fields are
required, their types, their formats and their limits.

    {
      "$schema": "https://json-schema.org/draft/2020-12/schema",
      "title": "Invoice",
      "type": "object",
      "additionalProperties": false,
      "required": ["number", "date", "customer", "lines"],
      "properties": {
        "number": {"type": "string", "minLength": 1, "maxLength": 40, "title": "Invoice number"},
        "date": {"type": "string", "format": "date", "title": "Invoice date"},
        "customer": {"type": "string", "minLength": 1, "maxLength": 200, "title": "Customer"},
        "lines": {
          "type": "array", "minItems": 1, "maxItems": 100, "title": "Line items",
          "items": {
            "type": "object",
            "additionalProperties": false,
            "required": ["description", "quantity", "unit_price"],
            "properties": {
              "description": {"type": "string", "minLength": 1, "maxLength": 200, "title": "Description"},
              "quantity": {"type": "integer", "minimum": 1, "maximum": 10000, "title": "Quantity"},
              "unit_price": {"type": "integer", "minimum": 0, "maximum": 100000000, "title": "Unit price in cents"}
            }
          }
        }
      }
    }

Give every field a `title`: it is how the field is named in every finding
a supplier sees.

**The meaning document** says, in prose, what the schema cannot: what each
field holds and where a supplier finds it.

**The template** is Liquid, in the dialect the specification pins. Each
input is `inputs.<name>`:

    <h1>Invoice {{ inputs.invoice.number }}</h1>
    <p>{{ inputs.invoice.customer }}, {{ inputs.invoice.date }}</p>
    <table>
    {%- for line in inputs.invoice.lines %}
    {%- assign amount = line.quantity | times: line.unit_price %}
    <tr><td>{{ line.description }}</td><td>{{ line.quantity }}</td><td>{{ amount | divided_by: 100 }}.{{ amount | modulo: 100 | plus: 100 | append: "" | slice: 1, 2 }}</td></tr>
    {%- endfor %}
    </table>

Because the output declares `escape: html`, a customer named
`<script>` is printed as text, never as markup.

**Preview it** from data generated from your schema, as often as you like:

    metais render invoice --sample --out preview

## 3. Check it

    metais check invoice

`check` needs no data. It checks the manifest, every schema, every
template, and the references between them, such as a template that uses a
field the schema does not declare. It reports every finding of every step
at once.

## 4. Test it

An example is a set of inputs and the outputs a render of them must
produce, byte for byte. It is how you know the package still renders what
you meant after you change it. Generate an example's inputs from your
schemas, edit them into a realistic invoice, then record its outputs:

    metais sample invoice --example basic
    metais test invoice --update

Read `invoice/examples/basic/expected/invoice.html` before you keep it:
`--update` records whatever the render wrote. From then on, test the
package after every change:

    metais test invoice

A render that differs from an example's expected output fails, and the
test shows the difference.

## 5. Pack it

    metais pack invoice

`pack` runs `check` and `test` again, and refuses to write a package whose
checks or examples fail. Otherwise it writes `invoice-0.1.0.metais`: the
same bytes from the same sources, every time. That file is what you give
to the people who supply the data, and
[Using a package](/guide/using-a-package/) is what they do with it.

## Change a package that is already packed

To add a due date to the invoice, unpack the package into a source
directory, edit it, and pack it again:

    metais unpack invoice-0.1.0.metais --out invoice-src

1. In `inputs/invoice.schema.json`, add `"due"` to `required`, and the
   field: `"due": {"type": "string", "format": "date", "title": "Due date"}`.
2. In `templates/invoice.html.liquid`, print it:
   `<p>Due {{ inputs.invoice.due }}</p>`.
3. Raise `version` in `metais.yaml`, here to `0.2.0`.

Then test it:

    metais test invoice-src

The test fails, and says why: the example has no due date, which the
schema now requires. Add one to `examples/basic/inputs/invoice.json`, run
the test again to see the difference in the rendered page, record it, and
pack:

    metais test invoice-src --update
    metais pack invoice-src

`pack` writes `invoice-0.2.0.metais`.

## Write data into a page's script

A page that charts its data needs that data in JavaScript. Never write it
into a script with `json`: a value holding `</script>` would end the
script early. Write it into a JSON script element with `json_script`,
which no value can break out of, and read it from there:

    <script type="application/json" id="numbers">{{ inputs.data.numbers | json_script }}</script>
    <script>
    const numbers = JSON.parse(document.getElementById("numbers").textContent);
    </script>

`metais init <dir> --minimal --html` writes its template this way, and
`metais help templates` explains each filter's escaping.

## Escape hatches

- **A source directory as it is.** Every command that reads a package
  also reads its source directory, so `metais inspect invoice` shows you
  the package as a supplier will see it, before you pack it.
- **Test one example**: `metais test invoice --example basic`.
- **Pack somewhere else**: `metais pack invoice --out dist`, into an
  existing directory.
- **An agent doing this for you.** `metais skill` installs a skill that
  teaches a coding agent this route.

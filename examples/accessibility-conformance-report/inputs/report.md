# Report details

The report details say what the report is about, who stands behind it,
how the product was evaluated, and which standards it answers. Write them
as report.yaml (or report.json) in the data directory.

```yaml
product:
  name: Ledgerline
  version: "4.2"
  description: A web application for small businesses to keep their
    accounts, send invoices and reconcile bank statements.
vendor:
  name: Example Ledger Ltd
  contact: Accessibility team
  email: accessibility@example.com
  website: https://example.com/accessibility
date: 2026-10-01
edition: international
methods: |-
  Tested by the vendor's accessibility team in September 2026.
  Keyboard-only operation; NVDA 2026.2 with Firefox and JAWS 2026 with
  Chrome on Windows 11; VoiceOver with Safari on macOS 15.
notes: The mobile applications are covered by a separate report.
```

Quote a version such as `"4.2"`: unquoted, YAML reads it as a number, and
the version must be text.

## product

- **name** is the product's name as its vendor sells it. The report's
  title begins with it.
- **version**, optional, is the version, release or build the report
  covers. A report describes one version; a new release needs a new
  report.
- **description**, optional, says in a few sentences what the product is
  and does, so a reader knows what was evaluated. A line break starts a
  new paragraph.

## vendor

- **name** is the company or organization that makes the product.
- **contact**, **email** and **website**, each optional, say whom a reader
  asks about the report. The website begins `https://` or `http://`.

## date

The date the report was completed, written `YYYY-MM-DD`.

## edition

The edition decides which standards the report answers, and so which
tables you supply:

| edition | Standards | Tables to supply |
|---|---|---|
| `wcag` | WCAG 2.2 | wcag |
| `section508` | WCAG 2.2 and the Revised Section 508 Standards | wcag, section508 |
| `en301549` | WCAG 2.2 and EN 301 549 V4.1.1 | wcag, en301549 |
| `international` | all three | wcag, section508, en301549 |

Choose the edition the buyer asked for. A United States federal buyer
asks for section508, a European public buyer for en301549, and a buyer in
both, or a vendor writing one report for every market, for international.
Left out, the edition is `wcag`.

The edition is strict both ways: validation refuses a table the edition
needs and you left out, and a table it does not hold and you supplied, so
a report never carries a table its title does not promise.

## methods

How the product was evaluated, in prose: who tested, when, with which
assistive technologies, browsers and platforms and their versions, which
automated tools, and what was in and out of scope. A reader judges every
row of the tables by this text, so be specific. A line break starts a new
paragraph.

## notes

Optional: anything a reader should know before the tables, such as known
issues being fixed, or parts of the product the report does not cover.

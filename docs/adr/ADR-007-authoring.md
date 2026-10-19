# ADR-007: Authoring — a scaffold that passes its own tests, a data-free check with a static template↔schema cross-reference, examples compared byte for byte, a deterministic pack, and a seeded sample-data generator

**Status:** Accepted 2026-10-18 by Ed Funnekotter; its proposed rulings are ratified in [RULINGS-v1](RULINGS-v1.md).
**Date:** 2026-10-18
**Deciders:** Ed Funnekotter
**Designs:** PRD-001 R13 items 1 to 6; R3 item 5 (examples as part of the
data specification); R17 item 2 (the authoring golden path)
**Depends on:** ADR-001 (the layout, the reserved entries, the deterministic
zip writer, the digest file and its verification, which `pack` and `unpack`
use); ADR-002 (the manifest's fields, its YAML profile with line numbers, the
package name rule, the canonical JSON); ADR-003 (input declarations, the
asserted JSON Schema keywords and formats, input forms and their parsing, the
data directory layout, default application, the combined schema); ADR-004 (the
pinned Liquid dialect: expression grammar, tags, filters, variables, partial
resolution); ADR-005 (outputs, `for_each`, `when`, the mirror shorthand,
containers, in-memory staging of a render); ADR-006 (the finding model, codes,
severities, exit codes); ADR-008 (command names and flags, the JSON envelope);
ADR-009 (resource limits and confinement); ADR-010 (the libraries and the
package layout); ADR-011 (the conformance vector format).

## Context

### What the PRD requires

R13 gives the author one path: `init`, edit, `check`, `test`, `pack` (R17
item 2), plus `unpack` and a sample-data preview. The requirements this ADR
must make buildable:

- **`init`** writes a minimal source directory (R13 item 1).
- **`check`** needs no data. It verifies the manifest, compiles every schema,
  parses every template, confines every path, and cross-references templates
  against schemas: a field a template uses that no schema declares is an
  error, an input no template uses is a warning, and a Liquid construct split
  across Office Open XML text runs is a finding (R13 item 2).
- **Examples are tests**: inputs validated and rendered, outputs compared byte
  for byte where given (R13 item 3). Examples are also the most concrete
  statement of what an input expects (R3 item 5).
- **`pack`** runs check and test, then writes a package whose bytes depend
  only on the source directory (R13 item 4). **`unpack`** returns an editable
  source directory (R13 item 5).
- **The sample preview** renders from generated data that satisfies the
  schemas, prefers the schema's own `examples`, `default`, `const`, `enum`,
  fills optional fields all, none or mixed, reports an unsatisfiable field as
  a finding that suggests an `examples` value, is deterministic for a seed,
  and can be saved as an example's inputs (R13 item 6). SC4 requires every
  example package of §4.1 to §4.5 to preview, all-filled and none-filled, to
  output that validates.

### What the Liquid library exposes (verified 2026-10-18)

The cross-reference is static analysis of variable use, so it needs the
expression structure of every template, not only the ability to render one.
`github.com/osteele/liquid` v1.9.2 (released 2026-08-16):

- `Template.GetRoot()` returns the root `render.Node` of the compiled tree.
  `BlockNode` exports `Body` and `Clauses`; `TagNode`, `BlockNode` and
  `ObjectNode` embed `parser.Token`, which exports the tag `Name`, its
  argument text `Args`, the full `Source` and a `SourceLoc` of path and line.
  There is no column.
- **Expressions are not exposed as a tree.** `expressions.Parse` returns an
  `Expression` whose only method is `Evaluate`; the yacc grammar compiles each
  expression directly into Go closures (`func(Context) values.Value`), and
  `ObjectNode.expr` is unexported. The package's own doc comment calls it "an
  internal package".
- `SeqNode`, `RawNode` and `TrimNode` embed a `sourcelessNode` whose
  `SourceLocation()` panics.

So the library gives tag structure with line numbers, and nothing about the
variable paths inside an expression. A cross-reference built on it would
re-parse every `Args` string with a parser of its own anyway. This is one
of the reasons ADR-010 D4 does not use the library: METAIS's own engine,
`internal/liquid`, parses into a walkable syntax tree (`internal/liquid/ast`)
with a full expression tree, the binding constructs marked, and a byte span,
line and column on every token, and that tree is what this ADR analyses.

### What Liquid's reference implementation does at the points the analysis touches

From the Shopify Liquid source (`lib/liquid/tags/render.rb`,
`lib/liquid/context.rb`, `lib/liquid/variable_lookup.rb`):

- `render` requires a quoted string as the template name; its syntax is
  `render 'name' [with|for expr] [as alias]` plus `key: value` parameters. The
  partial runs in `new_isolated_subcontext`, which keeps the static
  environments (global objects) and drops the caller's assigned variables.
  The alias defaults to the last path segment of the template name. `for`
  binds `forloop` inside the partial.
- A lookup segment `size`, `first` or `last` that is not in brackets and is
  not a key of the object calls the method instead; a key that exists wins.
  `first` and `last` on a string give its first and last character.
- The root of a variable path may itself be bracketed (`[expr]`), which looks
  a variable up by a computed name.

### What JSON Schema 2020-12 says at the points the analysis and the generator touch

- `true` behaves like `{}` and `false` like `{"not": {}}`.
- `pattern` is an ECMA-262 regular expression and is not implicitly
  anchored.
- `minLength` and `maxLength` count characters as RFC 8259 defines them
  (code points).
- `minContains` defaults to 1 and has effect only with `contains`.
- `multipleOf` is strictly greater than 0.
- The formats are, by section 7.3: `date-time`, `date`, `time`,
  `duration`; `email`, `idn-email`; `hostname`, `idn-hostname`; `ipv4`,
  `ipv6`; `uri`, `uri-reference`, `iri`, `iri-reference`, `uuid`;
  `uri-template`; `json-pointer`, `relative-json-pointer`; `regex`.

### How a word processor stores the text a placeholder is typed into

In WordprocessingML a paragraph `w:p` holds runs `w:r`, each a region of text
with one set of properties (`w:rPr`), and each run holds text `w:t`. When an
author types `{{ inputs.vendor.name }}` into Word, spelling marks
(`w:proofErr`), a formatting change, a tracked change (`w:ins`, `w:del`), a
bookmark or the editing session itself routinely end one run and start
another inside the placeholder. The XML then reads
`<w:t>{{ inputs.</w:t></w:r><w:r><w:t>vendor.name }}</w:t>`, and the Liquid
engine, which reads the raw bytes, sees an expression containing markup.
SpreadsheetML stores rich text as runs `r` of text `t` inside a shared-string
item `si` (or an inline string `is`); DrawingML (PowerPoint slides, and text
in charts and shapes everywhere) uses `a:p`, `a:r`, `a:t`. Office writes
namespace prefixes freely — Microsoft's own SpreadsheetML example uses `x:`,
and its PresentationML example declares `xmlns:a` on individual elements —
so detection must resolve namespaces, not match prefixes.

### Randomness that stays reproducible

The generator must give the same data for the same seed on every platform.
Go's `math/rand/v2` documents that a fixed seed reproduces output on every
run, and says nothing about stability of `IntN` and the other derivations
across Go releases. A generator whose output changes with the toolchain would
make "seed 7" mean different data in two builds of one tool version.

## Decision

### D1 — `init` writes a scaffold that already checks, tests and packs

1. **`metais init [<dir>] [--name <name>]`** (ADR-008 D9). `<dir>` defaults
   to the current directory. It asks nothing and reads no environment
   variable.
2. **`--name`** is the package name (ADR-002's name rule). Absent, it is the
   last element of `<dir>` as the caller wrote it; when `<dir>` is omitted or
   its last element is `.` or `..`, `--name` is required, because the tool
   never looks up the working directory's name (ADR-008 D9, ADR-009 D12).
   When the name does not satisfy the rule, `init` refuses and names
   `--name` as the next step.
3. **The directory must be empty.** It is created, with parents, if it does
   not exist. An existing directory counts as empty when it holds nothing but
   entries on the quiet list of D7 item 3 (so `git init` followed by `metais
   init` works). Anything else is a refusal that lists up to ten of the
   entries found and says to choose an empty directory; `init` never
   overwrites and has no `--overwrite`.
4. **What it writes**, all UTF-8 with LF line endings and a final newline,
   files mode 0644 and directories 0755 before the umask, in this order:

   ```
   <dir>/
     .gitattributes
     metais.yaml
     inputs/greeting.schema.json
     inputs/greeting.md
     templates/greeting.md.liquid
     examples/basic/example.yaml
     examples/basic/inputs/greeting.json
     examples/basic/expected/greeting.md
   ```

   - **`.gitattributes`**: the single line `* -text`. It stops git from
     converting line endings in a checkout, which would otherwise change
     templates (and so outputs) and expected files on Windows and break the
     byte comparison of D5. `pack` does not pack it (D7 item 3).
   - **`metais.yaml`**: the specification version `1.0`; the name; version
     `0.1.0`; a title (the name with `-` and `_` as spaces, each word
     capitalised); a one-line description placeholder; a supplier guide of
     two sentences saying what the package produces and pointing to the
     input's meaning document; one input `greeting` with `title`, schema
     `inputs/greeting.schema.json`, `meaning` `inputs/greeting.md` and forms
     `json` and `yaml`; one output, `id: greeting` and `template:
     greeting.md.liquid`, whose path defaults to `greeting.md` (ADR-005 D4
     item 4). Field spellings are ADR-002's, ADR-003's and ADR-005's.
   - **`inputs/greeting.schema.json`**: `$schema` 2020-12; `title` and
     `description`; `type: object`; `additionalProperties: false`; required
     `name` (`type: string`, `minLength: 1`, `maxLength: 80`, `title`,
     `description`, `examples: ["Ada Lovelace"]`); optional `note`
     (`type: string`, `maxLength: 400`, `title`, a `description` that says
     leaving it out prints nothing). The optional field exists so the sample
     modes of D9 have a branch to show.
   - **`inputs/greeting.md`**: a short meaning document — what the greeting
     is for, what a good `note` looks like.
   - **`templates/greeting.md.liquid`**, exactly:

     ```liquid
     # Hello, {{ inputs.greeting.name }}
     {%- if inputs.greeting.note %}

     {{ inputs.greeting.note }}
     {%- endif %}
     ```

     It demonstrates the idiom for optional data: an absent field is `nil`,
     the only falsy value besides `false`, so `if` tests presence.
   - **`examples/basic/example.yaml`**: `title: Basic`, and a `description`
     saying it is the scaffold's example.
   - **`examples/basic/inputs/greeting.json`**:
     `{"name": "Ada Lovelace", "note": "Welcome."}`, two-space indented.
   - **`examples/basic/expected/greeting.md`**: `# Hello, Ada Lovelace`,
     a blank line, `Welcome.`, final newline.
5. **It prints** the files written and the next step: edit `inputs/` and
   `templates/`, then `metais check`, `metais test`, `metais pack`. With JSON
   output (ADR-008) it prints the list of paths.
6. **A failed write** stops `init`, which reports the files already written;
   it removes nothing.
7. **The scaffold is pinned by a test**: the tool's suite runs `init`, then
   `check` (no findings at all), `test` (pass) and `pack`, on every supported
   platform.

### D2 — `check` runs every step, in a fixed order, and reports every finding

1. **`metais check [<pkg>]`** accepts a source directory (default `.`) or a
   packed `.metais`. It reads no data other than the package's own content.
   It exits non-zero when any finding has error severity (ADR-006); warnings
   do not fail it.
2. **The steps**, in order. Each runs whatever earlier steps found, except
   that a step whose input does not exist is skipped, and every skipped step
   is named in one `report.step_skipped` info finding.
   1. **Source tree** (source directory only). Every entry is a regular file
      or a directory: a symbolic link, device, socket or pipe is
      `package.layout.non_regular_file`, an empty directory
      `package.layout.empty_directory`, both errors (ADR-001 D2 item 6, D5
      item 6). Entries outside the layout are classified by D7 item 3. A
      `.compiled/` directory is `package.layout.compiled_in_source`, an
      error, as it is for every command (ADR-001 D5 item 5): a stale copy
      beside the sources would be read as either. Operating-system litter
      under a layout tree (`.DS_Store`, `Thumbs.db`, `desktop.ini`, a name
      beginning `._`) is `package.layout.os_litter`, a warning: the package
      would carry it (ADR-001 D5 item 6), and an asset directory copy would
      write it into outputs.
   2. **Entries.** Every entry name passes ADR-001's name rules and
      ADR-009's confinement rules; the entries ADR-001 reserves are where it
      requires them. On a packed package this has already happened on read
      (ADR-001), and digests were verified.
   3. **Limits.** The package `pack` would write stays within ADR-009's
      default read limits — entry count, per-entry and total size, path
      length — so that no package `pack` writes is one a default reader
      refuses. A breach is `limit.package`, an error naming the limit
      (ADR-009 D19).
   4. **Manifest.** It parses under ADR-002's YAML profile, validates against
      the manifest's JSON Schema (`spec/schemas/`), and passes the semantic
      rules ADR-002, ADR-003 and ADR-005 state. Every path it names — schema,
      meaning document, template, asset, output validation schema — exists
      and is confined. A relative link in `guide` whose target is not a
      package entry is `manifest.guide_link_missing`, a warning (ADR-002 D4
      item 9).
   5. **Schemas.** Every input schema and every output validation schema
      compiles under ADR-003's dialect, and every `$ref` resolves inside the
      package. Then, statically: each `default` and each `examples` entry
      validates against the schema object that contains it
      (`schema.invalid_default`, `schema.invalid_example`, errors — a default
      is applied before validation, R5 item 3, and an example misleads the
      supplier who copies it); a `const` that fails the rest of its schema
      object is `schema.unsatisfiable`, an error; an `enum` member that fails
      the rest of its schema object is `schema.unusable_enum_member`, a
      warning; a name in `required` that is not declared at that object
      (D3 item 4) is `schema.required_not_declared`, a warning, because it is
      almost always a typo.
   6. **Templates.** Every file under `templates/` is valid UTF-8
      (`template.not_utf8`, error) and parses with the engine's parser, the
      one parse both rendering and D3 use (`template.syntax`, error; ADR-004
      D16). Parse-error hints are in item 4. A template containing a CR LF
      pair is `template.crlf`, a warning: a checkout that converts line
      endings produces it, and the output's CRLF transform (ADR-005) is the
      portable way to get CRLF output. Each manifest expression — output
      `path` templates, `when`, `for_each` — parses (`template.syntax`,
      error, located at the declaration's manifest pointer and line,
      ADR-004 D16 item 5).
   7. **Cross-reference** (D3), over every template and manifest expression
      that parsed.
   8. **Office Open XML text runs** (D4).
   9. **Examples layout** (D5 item 1). Validating and rendering them is
      `test`'s.
   10. **Compiled form.** Everything `.compiled/` will hold — the canonical
       manifest, the combined schema, the references index, the digests — is
       computed, and a failure to compute any part is a finding here. On a
       packed package, the stored `.compiled/` must equal what is computed
       from its entries; a difference is `package.compiled.mismatch`, an
       error (on a packed package ADR-001 D8 has already refused it). This
       step is why `pack` (D7) never discovers a problem `check` did not
       report.
3. **Findings are all reported.** Order and de-duplication are ADR-006's.
4. **Parse-error hints.** A `template.syntax` finding whose construct contains
   U+2018, U+2019, U+201C or U+201D carries the hint that a word processor
   replaced straight quotes and that Liquid string literals use `"` or `'`.
   One in a template detected as Office Open XML (D4 item 1) whose construct
   contains `&lt;`, `&gt;`, `&amp;`, `&quot;` or `&apos;` outside a string
   literal carries the hint that the document escaped a character Liquid
   reads raw, and that the literal character is correct inside the
   construct. One whose construct contains XML markup defers to the D4
   finding at the same place.

### D3 — The cross-reference is a static analysis with an exact definition of "declared"

1. **The analysis walks the engine's own syntax tree.** ADR-010 D4's engine
   parses ADR-004's pinned dialect — tags, the expression grammar, filters
   with their arguments, ranges, literals — into `internal/liquid/ast`, with
   file, line, column and byte span for every node and every binding
   construct marked, and `ast.Walk` visits it. The cross-reference uses that
   tree and no parser of its own, so `check` and the render can never
   disagree about what a template says. The analysis is specified against
   the dialect's grammar, not against the engine's internals, and a second
   implementation reproduces it from the specification.
2. **The abstract values.** Analysis evaluates each expression to a set of
   alternatives, each one of:
   - **`schema(n)`** — a node of an input's schema, identified by input name
     and JSON Pointer into that input's schema resources;
   - **`open`** — anything at all is declared below it;
   - **`unknown`** — the analysis cannot tell; nothing below it is checked
     and nothing below it is an error;
   - **`scalar(kind)`** — a string, number, boolean or `nil` (literals,
     `capture`, filter results of known kind);
   - **`record(fields)`** — a fixed set of named fields: the `inputs`
     namespace (one field per input), `package`, `fan_out`, `output`,
     `sources` (one field per input), `forloop`, `tablerowloop` (their
     fields as ADR-004 D7 defines them).

   A set holds at most 64 alternatives; a larger one becomes `unknown`.
3. **Roots.** An expression path starts at an identifier:
   - `inputs` → the inputs record; `package` → its record; `now` →
     `scalar(string)` or `nil`; `sources` → its record;
   - `item` and `fan_out` → in a template rendered for a `for_each` output
     (directly or through partials), the element value of D3 item 9 and the
     `fan_out` record; elsewhere `template.xref.unknown_variable`, with the hint that
     they exist only under `for_each`;
   - `output` → its record, in a template body only (ADR-004 D7 item 10);
   - `forloop` inside a `for` body or a partial rendered with `render … for`;
     `tablerowloop` inside `tablerow`;
   - a local (item 7) or a partial parameter (item 8);
   - anything else → `template.xref.unknown_variable`, error, with the nearest known
     name when one lies within edit distance 2.

   A bracketed root (`[expr]`) is `template.xref.dynamic_reference` (item 6).
4. **What "declared" means.** Resolving a property segment `.p` (or `["p"]`
   with a string literal) against `schema(n)`:
   1. **The applicable set** A(n) is n plus, recursively, every subschema
      reached through `$ref`, `$dynamicRef` (resolved to its static target;
      where the input's schema resources declare more than one
      `$dynamicAnchor` of that name, to all of them), `allOf`, `anyOf`,
      `oneOf`, `if`, `then`, `else` and the values of `dependentSchemas`.
      `not` is excluded: a property named in `not` is one that must not
      match. Cycles are cut by a visited set.
   2. **Open schemas.** If any node in A(n) is the boolean schema `true`, or
      an object schema with no keyword other than annotations (`title`,
      `description`, `$comment`, `examples`, `default`, `deprecated`,
      `readOnly`, `writeOnly`, `$id`, `$anchor`, `$schema`, `$defs`), the
      result is `open`. The author said nothing, so nothing is checkable.
   3. **Candidates.** For each node m in A(n): `properties[p]` if present;
      every `patternProperties` schema whose pattern matches p; and, only if
      m has neither, m's `additionalProperties` when it is present and not
      `false` — `true` included — or, when m has no `additionalProperties`,
      its `unevaluatedProperties` on the same terms.
   4. **Declared** means at least one candidate exists. The result is
      `schema` of every candidate. **An absent `additionalProperties` declares
      nothing**: JSON Schema allows any other name, but a template reading
      one is reading a name the author never wrote, which is the typo this
      check exists to catch. An author who means "any key" writes
      `additionalProperties: true` or a schema for the values.
   5. **Liquid's command names.** With no candidate, a `.p` segment `size`
      on a node whose types (item 5) include array, object or string gives
      `scalar(number)`;
      `first` and `last` on an array give the element (as `[0]` and `[-1]`),
      on a string `scalar(string)`. A declared property of that name wins, as
      in Liquid's lookup. A bracketed `["size"]`, `["first"]` or `["last"]`
      is a name like any other, as ADR-004 D4 item 7 reads it, and goes on
      to the next sub-item.
   6. **Otherwise** the segment is an error: `template.xref.not_an_object` when the
      node's types exclude object, else `template.xref.unknown_field`. The finding
      names the template location, the input, the schema location (file and
      JSON Pointer) as a related location, the declared names there, and the
      nearest within edit distance 2.

   Union semantics: a segment is an error only when **every** alternative
   rejects it, and `open` and `unknown` reject nothing. A field declared only
   in one `oneOf` branch or only in a `then` is declared; the template is
   expected to guard it, and the analysis does not follow control flow.
5. **Types of a node** are the JSON types its `type` allows (all, when
   absent), intersected through `$ref` and `allOf` and united across
   `anyOf`/`oneOf` branches; `if`/`then`/`else` do not narrow them.
6. **Index and dynamic segments.**
   - `[i]`, an integer literal: `prefixItems[i]` when it exists (i ≥ 0), else
     `items`. A node of array type with neither keyword gives `open`.
     `items: false` with i at or beyond the `prefixItems` length, or i at or
     beyond `maxItems`, is `template.xref.index_out_of_range`, error. A negative i
     gives every `prefixItems` schema and `items`. A node whose types exclude
     array is `template.xref.not_an_array`, error.
   - `[expr]`, anything but a literal: on `schema(n)`, every candidate D3
     item 4 could produce for any name, plus `items` and `prefixItems`; on an
     open node, `open`. If that leaves nothing — and always on the inputs
     record, and on a bracketed root — the result is `unknown` and the
     finding is `template.xref.dynamic_reference`, a **warning** that the reference
     cannot be checked. A dynamic reference into the inputs record, or a
     bracketed root, also marks **every** input used (item 11): the author is
     selecting inputs by computed name, and calling them unused would be
     false.
7. **Locals.** `assign n = e` binds n to the value of e; `capture n` binds
   `scalar(string)`; `increment n` and `decrement n` bind `scalar(number)`.
   Liquid's `assign` outlives the block it is in, so locals are
   template-wide and flow-insensitive: a first pass collects every name bound
   anywhere in the template, and a local's value is the union of the values
   of all its bindings, computed to a fixed point (bounded by item 2's cap).
   Binding a global or loop object by a template-scoped construct is
   ADR-004 D7 item 8's `template.reserved_name`, found at parse.
8. **Loops and partials.**
   - `for n in e` binds n, inside the body only, to the element of e:
     `[expr]` on an array value, `scalar(number)` for a range, and `unknown`
     over an object or scalar (whose iteration is ADR-004's). `tablerow`
     likewise.
   - `render "name"` resolves the literal name to a template by ADR-004's
     rule; a template that does not exist is `template.xref.missing_partial`, error.
     The partial is analysed in a fresh scope holding the globals ADR-004
     makes visible in a partial, its `key: value` parameters, and the
     `with`/`for` value under its alias (by default the last path segment of
     the name), with `forloop` for `for`. Analysis is per call context and
     memoised on the partial and the abstract values bound; a partial
     already on the analysis stack with the same bindings is not re-entered.
   - A name a partial reads that is neither a global, a local of the partial,
     nor bound by the call is `nil` at render time. It is
     `template.xref.partial_parameter_not_passed`, a warning naming the call site,
     because an optional parameter is a legitimate pattern.
9. **Manifest expressions.** Each output's `path` template and `when` are
   analysed in that output's context. `for_each` is analysed in the context
   without `item`; its value must permit arrays (`template.xref.for_each_not_array`,
   error, otherwise), and its element value is `item`'s in the output's
   template, `path` and `when`. A template used by several outputs is
   analysed once per distinct context, and each finding says which output's
   context produced it.
10. **Filters.** Every filter argument is analysed as an expression. The
    result of a filter follows its **analysis signature**, which ADR-004's
    catalog states for each filter as one of: *preserves its input* (`sort`,
    `sort_natural`, `reverse`, `uniq`, `compact`, `where`, `slice` on
    arrays; `default` gives the union with its argument; `concat` the union
    of element values); *gives the element* (`first`, `last`); *projects a
    property* (`map: "p"` gives an array of p resolved on the element); or
    *gives a scalar of a stated kind or `unknown`* (everything else). A
    filter argument the catalog marks as a property name — the string
    literals of `map`, `where`, `sort`, `sort_natural` and `sum` — is
    resolved as `.p` on the element and checked like any segment, so
    `inputs.rows | map: "vaule"` is an error.
11. **Use.** An input is used when any reference in an **effective template**
    or manifest expression passes through `inputs.<name>` — whatever follows
    and whether or not a finding follows it — or uses `inputs` whole, or an
    `embed` tag names it. Effective templates are those an output renders,
    directly or through `render` chains (ADR-005 decides which templates the
    mirror shorthand renders). An input no use reaches is
    `template.xref.unused_input`, a warning. A template under `templates/` that no
    output renders and no `render` names is `template.xref.unused_template`, a
    warning; it is still parsed and its references checked, with every
    partial-parameter name taken as `unknown`, and its references do not
    count as use.
12. **METAIS tags.** `embed` names an input by literal (`template.xref.unknown_input`
    when it is not one); `asset` names an asset path by literal
    (`template.xref.missing_asset` when no such entry exists under `assets/`);
    `provenance` takes keywords only, which ADR-004 validates.
13. **The analysis is bounded.** It stops after 10 000 template-context
    analyses or 1 000 000 resolved segments, reporting
    `template.xref.budget_exhausted`, a warning that the remaining references were not
    checked. Both bounds are fixed constants of the analysis, not ADR-009
    limits: reaching them never refuses anything, and the template, nesting
    and loop limits ADR-009 sets already bound the input they apply to.

A worked resolution: `inputs.metrics.rows[0].value` with `metrics` declared
as `{type: object, properties: {rows: {$ref: "#/$defs/rows"}}}` and `$defs/rows`
as `{type: array, items: {type: object, properties: {value: {type:
number}}}}`. `inputs` is the record; `.metrics` is a field of it, giving
`schema(metrics#)` and marking `metrics` used; `.rows` is in `properties`,
giving `schema(metrics#/properties/rows)`; its applicable set adds
`#/$defs/rows`; `[0]` finds no `prefixItems` and takes `items`; `.value` is in
its `properties`. Declared. `inputs.metrics.rows[0].vaule` stops at the last
step with `template.xref.unknown_field`, declared names `value`, suggestion `value`.

### D4 — A Liquid construct split across Office Open XML text runs is an error, with a hint that names the cause

1. **Which templates are scanned.** A template whose source declares one of
   the WordprocessingML, SpreadsheetML or DrawingML main namespaces —
   transitional `http://schemas.openxmlformats.org/wordprocessingml/2006/main`,
   `…/spreadsheetml/2006/main`, `…/drawingml/2006/main`, or strict
   `http://purl.oclc.org/ooxml/wordprocessingml/main`,
   `…/spreadsheetml/main`, `…/drawingml/main` — and every template
   `render`ed from one, directly or transitively. A partial inherits the
   prefix bindings declared on the root element of the template that renders
   it, because a part's fragments (a table row, a list item) usually declare
   none of their own. The scan does not depend on the manifest, so it runs
   before a container is wired up.
2. **Text elements** are elements whose expanded name is, in WordprocessingML,
   `t`, `delText` or `instrText`; in SpreadsheetML, `t`; in DrawingML, `t`.
   **Paragraph containers** are WordprocessingML `p`, DrawingML `p`, and
   SpreadsheetML `si` and `is`.
3. **The scan.** A lenient XML tokenizer reads the template source,
   resolving prefixes through every `xmlns` declaration in scope, and treats
   any Liquid construct inside a start tag as opaque attribute text. It
   records the byte range of each text element's content. The analysis
   tokenizer of D3 item 1, which ignores XML, gives the byte range of every
   Liquid construct (`{{ … }}`, `{% … %}`, delimiters with whitespace control
   included).
4. **`template.ooxml.split_construct`**, an error: a Liquid construct whose range
   overlaps the content of at least one text element and is not contained in
   the content of exactly one. A construct lying wholly in markup — a
   `{% for %}` between `<w:tr>` elements — overlaps no text content and is
   correct.
5. **`template.ooxml.split_delimiter`**, an error: within one paragraph container,
   the last character of one text element's content and the first character
   of the next non-empty one form `{{`, `{%`, `}}` or `%}`. This catches an
   opener broken before the Liquid tokenizer ever sees it, which otherwise
   renders literal braces with no error at all.
6. **The finding** gives the template, the line and column of the
   construct's first character, and as detail the construct's visible text —
   the content of the text elements it spans, entities decoded, markup
   removed — and the number of text elements it spans. The hint names the
   cause from the elements found between the pieces: `w:proofErr` or a
   DrawingML run property `err="1"` (spelling or grammar marks); `w:ins`,
   `w:del` or `w:delText` (tracked changes — accept or reject them first);
   `w:bookmarkStart`, `w:bookmarkEnd`, `w:commentRangeStart` or
   `w:commentRangeEnd` (a bookmark or comment); `w:fldChar` or `w:instrText`
   (a field); otherwise a change of run properties (formatting). It then
   says what to do: delete the placeholder and type it again in one go with
   no formatting change, spelling mark or tracked change inside it, or edit
   the XML so the whole construct lies inside one text element.
7. **Severity is error** because both outcomes are wrong output: either the
   engine fails to parse markup inside an expression, or the opener never
   forms and the braces reach the document.

### D5 — Examples are a data directory and an output tree, and `test` compares them byte for byte

1. **Layout.** `examples/<name>/` holds:
   - **`inputs/`** — exactly the layout of a data directory (ADR-003), a
     file per input named after it or a directory per collection. It is
     absent for an example that supplies nothing, since a package holds no
     empty directory (ADR-001 D2 item 6); for the same reason an example
     cannot supply an empty collection, which only `--inputs-json` can;
   - **`expected/`** — optional: an output tree, exactly what a render would
     write to an output directory;
   - **`example.yaml`** — optional (YAML, or JSON when its first
     non-whitespace byte is `{`, under ADR-002's profiles as `metais.yaml`
     is read):
     `title` (one line), `description` (Markdown), and `now` (an RFC 3339
     date-time with an offset), the render time given to this example as if
     by `--now`.

   `<name>` follows ADR-003's rule for input names. Any other entry in an
   example directory is `example.layout`, an error from `check`.
2. **`metais test [<pkg>] [--example <name>]…`** (ADR-008 D9) runs every
   example, or those named, in byte order of name, on a source directory or
   a packed package.
   It first loads the package — manifest, schemas, templates — and if that
   produces errors it reports them and stops. For each example it then:
   1. loads `inputs/` through ADR-003's data directory loader, applies
      defaults and validates, reporting findings located in
      `examples/<name>/inputs/…`;
   2. if valid, renders into memory through ADR-005 D13's phases 1 to 4 —
      transforms and output validation included — under ADR-009's limits,
      with `now` from `example.yaml`, ADR-004 D15's fixed render identity,
      and no other host input;
   3. if `expected/` exists, compares the rendered tree with it (item 3);
      otherwise reports the example as rendered and not compared.
   Every example runs even after another fails. `test` exits non-zero when
   any example fails.
3. **Comparison.** Directories are ignored; files are compared by path.
   - A path in `expected/` the render did not write is `example.output_missing`;
     a path written and absent from `expected/` is `example.output_unexpected`.
   - **A container output** (ADR-005) is compared by its entries — names in
     order, uncompressed content, and the compression method of every entry
     whose declaration says `stored: true` (and a profile's `mimetype`) —
     as R9 item 5 states for conformance; the method of any other entry is
     the encoder's choice under ADR-001 D4's `auto` and is not compared, so
     an `expected/` written by another conforming implementation compares
     equal. An `expected/` container is read with the
     package reader's bounds; one that is not a readable zip is
     `example.expected_unreadable`. Entry differences are
     `example.entry_missing`, `example.entry_unexpected`, `example.entry_order` (the
     first index at which the name sequences differ),
     `example.entry_method`, and `example.output_differs` for content, located as
     `<output>!<entry>`.
   - **Every other file** is compared byte for byte; a difference is
     `example.output_differs`.
4. **Reporting a difference.** Both sides are text when both are valid
   UTF-8 and contain no NUL byte; otherwise binary.
   - **Text**: the location is the expected file at the first differing line
     (and column); the detail is a unified diff (`---`/`+++`/`@@`, three lines
     of context) capped at 16 KiB in JSON output and at 40 lines in human
     output, each marked when truncated.
   - **Binary**: the first differing byte offset, both sizes, both SHA-256
     digests.
   - **Hints**, the first that applies: equal after replacing CR LF with LF
     on both sides — line endings, naming the side with CRLF, the output's
     CRLF transform, and git's line-ending conversion; equal apart from one
     final newline — the final newline; equal apart from a leading UTF-8 BOM
     — the BOM; equal after removing trailing spaces and tabs from each line
     — trailing whitespace, and Liquid's whitespace control (`{%-`, `-%}`).
5. **`--update`** writes the render of each selected example into its
   `expected/` (creating it), on a source directory only, and only for an
   example that validated and rendered. It makes `expected/` exactly the
   rendered tree: files written, and files under `examples/<name>/expected/`
   that the render no longer produces removed. This is the one place an
   authoring command removes a file, and it never removes one outside that
   directory, which is the author's own tree and which the command fully
   owns; R12 item 3's "never deletes" governs rendering into a caller's
   directory and is untouched. It prints each file written or removed
   (`--dry-run` lists them and writes nothing, ADR-008 D9). Reviewing the
   result is the author's.

### D6 — `test` on a packed package and on a source directory is the same test

The comparison reads `expected/` in place inside a packed package. A
consumer can therefore run `metais test` on a package received, and the
result is the result the author's `pack` saw.

### D7 — `pack` is check, then test, then a deterministic write

1. **`metais pack [<dir>] [--out <path>|-]`** (ADR-008 D9), `<dir>`
   defaulting to `.`, a source directory only.
2. **Order.** It runs `check` (D2). Any error: it reports every finding and
   refuses, writing nothing, with the next step "fix the errors `metais
   check` lists". With no error it runs `test` (D5); any failure: refuses
   likewise. Tests are not run when check has errors, since a package that
   fails to check gives test failures that are only consequences. Warnings
   are printed and do not stop `pack`. There is no flag to skip tests: the
   bytes a consumer receives are the bytes whose examples passed.
3. **What is packed.** Exactly the entries ADR-001's layout defines —
   `metais.yaml` and the trees `inputs/`, `templates/`, `references/`,
   `assets/`, `examples/` and any further top-level entry ADR-001 permits —
   plus `.compiled/`, computed afresh. Outside them (ADR-001 D5 item 5):
   - **The quiet list** is excluded with no finding: every top-level name
     beginning with `.` other than `.compiled` (`.git`, `.gitignore`,
     `.gitattributes`, `.DS_Store`), and any top-level regular file whose
     name ends in `.metais`.
   - **Any other top-level entry** is excluded and reported as
     `package.layout.not_packed`, a warning, so a file put in the wrong place
     is noticed.
   - Inside the layout trees everything is packed, dotfiles included — a
     repository scaffold needs `assets/.github/` and `templates/.gitignore` —
     and so is operating-system litter, which `check` warns about
     (`package.layout.os_litter`, D2 step 1) so the author removes it.
4. **Determinism.** The package's bytes depend only on the packed entries'
   paths and contents: never on modification times, permissions, owner,
   directory traversal order, file-system case sensitivity, platform,
   locale, the clock, or the tool's version string (which is not written).
   Content is packed byte for byte, never normalised. ADR-001's writer
   supplies the ordering, timestamps and compression; ADR-001 states whether
   those bytes are also stable across tool builds.
5. **The output.** `--out` defaults to `./<name>-<version>.metais` in the
   working directory, from the manifest. An `--out` naming an existing
   directory writes `<name>-<version>.metais` inside it. `pack` refuses an
   output path inside one of the layout trees of the directory being packed.
   It writes a temporary file in the target directory, named as ADR-005 D14
   names its temporary files (`.metais-tmp-<h>`, created exclusively), syncs
   it, and renames it over the target, so a reader never sees a partial
   package and an existing package file at that path is replaced; a target
   that exists and is not a regular file is a refusal. `pack` is a build
   step, and replacing its own previous product is the normal edit loop, so
   it has no `--overwrite`. `--out -` writes the package to standard output
   (ADR-008 D9).
6. **It prints** the package's file name (never a directory the caller
   gave, ADR-009 D15), size, package digest (ADR-001), entry count and the
   warnings; JSON on request.

### D8 — `unpack` writes the authored entries and nothing derived

1. **`metais unpack <pkg> --to <dir>`** (ADR-008 D9); `--to` is required.
2. It reads the package through ADR-001's reader, so digests are verified
   and a mismatch is refused before anything is written. It does not run
   `check`: a package with check errors is exactly the one an author needs
   to unpack and fix.
3. **The target must be empty**, with D1 item 3's meaning; there is no
   `--overwrite`, because files left from an earlier tree would join the
   unpacked source and be packed with it. When the target does not exist,
   `unpack` writes into a staging directory beside it, named and created as
   ADR-005 D14 item 1 does, and renames it into place, so a failure leaves
   nothing; an existing empty directory is written in place, and a failure
   reports the files already written.
4. **It writes every entry except `.compiled/`.** `.compiled/` is derived;
   kept, it would be stale after the first edit, and `pack` regenerates it.
   Files are mode 0644 and directories 0755 before the umask.
5. **Round trip.** `pack` of an unpacked package gives a package equal entry
   for entry, `.compiled/` included, to the original; when the original was
   written by the same build of this tool, byte-identical.

### D9 — The sample-data generator

#### The command

1. **Two commands reach the generator, one to look and one to keep**
   (ADR-008 D8 item 8 and D9):
   - **`metais render [<pkg>] --sample --out <dir|->`** is the preview: it
     generates a value for every input, validates it (item 22) and renders
     it with every option `render` has — `--dry-run`, `--overwrite`,
     `--only`, standard output and `--now`.
   - **`metais sample [<pkg>] (--to <dir> | --example <name>)`** keeps the
     data: it writes it as a data directory (`--to`, which `render --data`
     then reads) or as an example's inputs (`--example`, item 21).

   Both take the generator's options:
   - `--seed <n>`, an unsigned 64-bit integer, default `0`;
   - `--optional all|none|mixed`, default `all`;
   - `--items <n>`, the target count of every array and collection, default
     `3`;
   - and the supply flags `--set`, `--set-json` and `--inputs-json`
     (ADR-008 D7), whose values are fixed points (item 2). `--data` is
     refused with `--sample`: the generated data takes the data directory's
     place.
2. **Supplied values are fixed.** Whatever the caller supplies, merged by
   ADR-003's precedence, is taken as given; the generator fills only what is
   absent, generating missing properties around a supplied object and
   evaluating conditionals against the merged value. An author previews the
   final stage of a staged report with `--set /incident/stage=final`.
3. **Normative status.** The generator is the reference tool's, not the
   specification's. The specification says that a sample mode, if a tool
   offers one, produces data that validates, and no more. The reference tool
   promises: the same package, options, supplied inputs and seed give
   byte-identical data and outputs on every supported platform, for a given
   tool version. Changes to generated values between tool versions are
   recorded in the release notes. A saved example is saved data, so nothing
   an author keeps depends on generator stability.

#### Randomness

4. **The PRNG is SplitMix64**, implemented in METAIS: state `x`; `next`
   adds `0x9e3779b97f4a7c15` to `x` and returns `z` mixed by
   `z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9`,
   `z = (z ^ (z >> 27)) * 0x94d049bb133111eb`, `z ^ (z >> 31)`, all modulo
   2⁶⁴.
5. **One stream per decision.** Every decision takes a fresh stream whose
   initial state is the first 8 bytes, big-endian, of
   SHA-256(`"metais-sample"` ‖ 0x00 ‖ seed as 8 bytes big-endian ‖ input
   name ‖ 0x00 ‖ instance JSON Pointer ‖ 0x00 ‖ purpose), where purpose is one
   of `value`, `include`, `type`, `branch`, `count`, `name` or `attempt:<k>`.
   A value therefore depends only on where it sits, so editing one field's
   schema leaves every other generated value unchanged.
6. **A bounded integer** in [0, n) is drawn by rejection: draw `u`; reject
   when `u ≥ 2⁶⁴ − (2⁶⁴ mod n)`; return `u mod n`. A range wider than 2⁶⁴
   concatenates draws. Numbers are computed as exact rationals throughout;
   no floating point is involved.

#### The effective schema

7. **Conjunction.** At each instance location the generator works on the
   conjunction C of the schema there and, recursively, its `$ref` and
   `$dynamicRef` targets and `allOf` members. Across C it intersects `type`
   sets, `enum` sets and `const`s, takes the tightest numeric and length
   bounds, multiplies `multipleOf` constraints into one (their least common
   multiple, as exact rationals), unions `properties` (conjoining
   per-property schemas), `required` and `patternProperties`, and conjoins
   `items` and `prefixItems` position by position.
8. **Disjunction.** For `anyOf` and `oneOf` in C it picks one branch with the
   `branch` stream and adds it to C. For `oneOf`, the finished value must
   match exactly one branch; otherwise the next branch in order is tried,
   each once.
9. **Conditionals.** For an object, properties are generated (item 15), the
   `if` of every node in C is evaluated against the value, the selected
   `then` or `else` is added to C, and properties it newly requires or
   constrains are generated again. This repeats until the selected branches
   stop changing, at most four passes. `dependentRequired` and
   `dependentSchemas` take effect the same way when their trigger property
   is present. For other types, a value generated from C that fails a
   selected `then` or `else` is regenerated from C with that branch added.
10. **`not`.** A value that matches a `not` schema in C is regenerated from
    an `attempt:<k>` stream, up to 16 attempts.
11. **Contradiction** — an empty type set, enum set or numeric or length
    range, conflicting `const`s, `false` required, conditionals that do not
    settle, attempts exhausted — is `schema.sample_unsatisfiable` (item 22).

#### Where a value comes from

12. **Order of preference**, each candidate kept only if it validates
    against C: `const`; else an entry of `examples`; else `default`; else an
    `enum` member; else a generated value. Among several `examples` or
    `enum` members, the one at index (seed + k) mod count, where k is the
    item index within the innermost enclosing array (0 outside any), so
    single values take the author's first example under the default seed
    and array items cycle through them. `uniqueItems` skips a candidate equal
    to an earlier item.
13. **Objects and arrays are not taken whole** from `examples` or `default`:
    the generator descends into properties and items, whose own sources
    apply, so the optional-field modes keep working. A whole-value example
    or default is used only for an object with no declared properties or an
    array with no declared items.

#### Generated values, by type

14. **Type choice.** From the type set (inferred, when `type` is absent, as
    object from `properties`/`required`/`additionalProperties`, array from
    `items`/`prefixItems`/`contains`, string from length, `pattern` or
    `format`, number from numeric keywords, otherwise string), in the order
    written. **The optional mode also governs emptiness:** with `null` in the
    type set, `all` never chooses it, `none` always does, `mixed` decides by
    the `type` stream.
15. **Object.** Properties are generated in declaration order: the order of
    first appearance in the authored schema files, walking C from the node
    itself through `allOf` members and `$ref` targets in order. A required
    property is always generated. An optional one is generated under `all`,
    omitted under `none`, and under `mixed` generated when the `include`
    stream's first draw is even. An omitted property with a `default` gets it
    later from ADR-003's default application, exactly as a supplier's
    omission would. No name beyond the declared properties is generated,
    except to reach `minProperties`: first omitted optional properties in
    order, then names generated from a `patternProperties` pattern (item 18)
    or, when `additionalProperties` is a schema other than `false`,
    `key1`, `key2`, … skipping taken names; every name must satisfy
    `propertyNames`. Above `maxProperties`, optional properties are dropped
    from the end; required ones over it are a contradiction.
16. **Array.** The length is `--items` clamped to [`minItems`, `maxItems`],
    where `maxItems` is lowered to the `prefixItems` length when `items` is
    `false`; with `prefixItems` present the default target is its length.
    Position i takes `prefixItems[i]`, else `items`. With `contains`, the
    first `minContains` positions are generated from `items` conjoined with
    `contains`, and later positions that would push the count past
    `maxContains` are regenerated. With `uniqueItems`, a duplicate is
    regenerated from `attempt:<k>` streams, up to 16; when that fails, the
    array is shortened while it stays at or above `minItems`, and below it
    the field is unsatisfiable.
17. **Number and integer.** The bounds come from `minimum`, `maximum`,
    `exclusiveMinimum` and `exclusiveMaximum`. Unbounded on both sides the
    range is [1, 100]; bounded on one side it extends 100 from that bound.
    Integers are drawn uniformly from the integers in range that satisfy
    `multipleOf`; none is a contradiction. Numbers without `multipleOf` are
    drawn as n/100 for the integers n in range, refining to n/10⁶ when the
    range holds no hundredth; numbers with `multipleOf` are drawn from its
    multiples in range. Values are written as plain decimals, no exponent.
18. **String.**
    - **`format`**, each format ADR-003 D4 asserts (`check` has already
      refused any other), generated from fixed, reserved or obviously
      placeholder sources, never from the host: `date` uniform in
      2020-01-01 … 2029-12-31; `date-time` that date with a time and `Z`;
      `time` `HH:MM:SSZ` (RFC 3339 full-time); `duration` one of `P<d>D`,
      `PT<h>H`, `P<d>DT<h>H`; `email` `<word>.<word>@example.com`;
      `hostname` `<word>.example.com`; `ipv4` in 192.0.2.0/24,
      198.51.100.0/24 or 203.0.113.0/24; `ipv6` in 2001:db8::/32; `uri`
      `https://example.com/<word>/<word>`; `uri-reference`
      `/<word>/<word>`; `uri-template` `https://example.com/{<word>}`;
      `uuid` 16 drawn bytes with the version 4 and RFC 9562 variant bits
      set; `json-pointer` `/<word>/<n>`; `relative-json-pointer`
      `0/<word>`. Words come from the corpus (item 19), lowercased. The value must then meet the length bounds and any
      `pattern`; otherwise the field is unsatisfiable.
    - **`pattern`**: the pattern, translated to RE2 by ADR-009 D10 item 6's
      profile, is parsed with Go's `regexp/syntax` (the same tree the
      validator matches with, ADR-010 D10 item 2) and walked: literals as written; a character class yields a character from
      its intersection with, in order of preference, `a`–`z`, `A`–`Z`,
      `0`–`9`, space, printable ASCII, and otherwise its lowest character;
      `.` yields from `a`–`z`; alternation picks a branch; a repetition
      {min,max} repeats min plus a draw of 0 to 2, capped at max; begin and
      end anchors yield nothing; a word-boundary assertion is unsupported.
      Because a pattern is unanchored, a result shorter than `minLength` is
      padded with corpus words after it unless the pattern is end-anchored,
      else before it unless start-anchored, else regenerated with repetitions
      raised toward their maximum. Every result is checked with the real
      matcher and the length bounds; up to 8 attempts, then the field is
      unsatisfiable.
    - **Plain text**: corpus words. The target length is 24 characters
      clamped to [`minLength`, `maxLength`]. Under 40 characters it is words
      joined by spaces, the first capitalised, no punctuation; from 40, it is
      sentences of 6 to 12 words, each capitalised and ending in a full stop.
      It is cut at the last word boundary that fits `maxLength` (mid-word
      only when one word is too long) and extended by words to reach
      `minLength`. Lengths are counted in code points.
    - **Prose fields**: a string that is a whole text-form input, a Markdown
      document, a collection item's `body`, or a string schema with
      `contentMediaType` `text/markdown` or `text/plain`, targets 400
      characters clamped to its bounds, as paragraphs of three to five
      sentences separated by a blank line. Markdown adds a level-one heading
      of three title-cased words and a three-item bullet list, when the
      length bounds leave room.
19. **The corpus** is the standard "Lorem ipsum dolor sit amet, consectetur
    adipiscing elit, sed do eiusmod tempor incididunt…" passage, fixed in the
    binary and printed in the tool's documentation, split into words with
    punctuation removed. A text starts at a word chosen by its `value`
    stream and wraps. Placeholder text is meant to be recognised as
    placeholder, so that the author replaces it.
20. **Recursion.** A `$ref` target entered for the fourth time on the path
    from the root stops growing: optional properties are omitted whatever the
    mode, and arrays take `minItems` items. A path that still recurses at
    depth 8 makes the field unsatisfiable.

#### Forms, saving and validation

21. **Serialising.** Each input is written in the first form its declaration
    lists (ADR-003), with the file extension ADR-003 assigns:
    - JSON: two-space indentation, keys in declaration order, LF, a final
      newline; an integral number of magnitude above 2⁵³ − 1 spelled with
      an exponent, the shortest text that round-trips (`1.1e+18`), since
      ADR-003 D8 item 5 refuses its integer literal as input;
    - YAML: block style, keys in declaration order, every string
      double-quoted (so no value is retyped by YAML's implicit typing), LF;
    - CSV: a header row of the item schema's properties in declaration order,
      the manifest's delimiter, RFC 4180 quoting where a value contains the
      delimiter, a quote, CR or LF, CRLF line endings; numbers as plain
      decimals, booleans `true`/`false`, an absent value as an empty cell;
    - text: the string, with a final newline added if absent;
    - Markdown: a string schema gives the document; an object gives YAML
      frontmatter of every field except `body`, then the `body`;
    - a collection: a directory named after the input holding one file per
      item, named `001`, `002`, … zero-padded to at least three digits and to
      the width of the count, so that file order is item order, each with
      its form's extension (`001.json`, `001.md`; ADR-003 D15 item 4).

    **`sample --to <dir>`** writes these files as a data directory; a
    non-empty `<dir>` is refused unless `--overwrite` (ADR-008 D9).
    **`sample --example <name>`** writes them as `examples/<name>/inputs/`
    and writes `examples/<name>/example.yaml` with `title` "Sample data" and
    a `description` recording the command's seed, mode and item count and
    saying to replace the placeholders and run `metais test --example <name>
    --update`. It refuses on a packed package (with the next step `metais
    unpack`), and refuses when `examples/<name>/` exists at all, so it never
    removes or merges. It creates no `expected/`.
22. **The generated data is validated like supplied data**: the files of
    item 21 are built in memory, loaded through ADR-003's data directory
    loader, defaults applied, then validated (ADR-006). What is previewed is
    therefore exactly what `sample` saves. Validation findings on
    generated data are reported with `schema.sample_invalid`, a
    generator limitation whose hint is the same as below. With any error,
    nothing is rendered and nothing is saved.
23. **`schema.sample_unsatisfiable`** is an error naming the input, the instance
    location, the schema file and JSON Pointer of the subschema, the reason
    (one of: contradictory type, range, length, enum or const;
    `multipleOf` with no multiple in range; pattern with an unsupported
    construct; pattern not met within the length bounds; format against
    pattern or length; `uniqueItems` exhausted; `not` exhausted;
    conditionals that do not settle; recursion depth), and the hint: add an
    `examples` value to this field and the generator will use it, with a
    skeleton such as `"examples": ["…"]`. Every unsatisfiable field is
    reported in one run.

## Interfaces with sibling ADRs

**ADR-001 (container and layout).** Relied on: the top-level layout and the
names a source directory ignores silently or warns about (ADR-001 D5 items 1
and 5), which D7 item 3 applies; every file under a layout tree being packed
(ADR-001 D5 item 6); a source `.compiled/` refused and an empty directory
refused (ADR-001 D5 item 5, D2 item 6); a writer whose output depends only on
entry paths and contents, byte-identical within one tool build and identical
by package digest across builds (ADR-001 D4 item 6); the digest file and its
verification on read (ADR-001 D8); the entry-name rule (encoding,
normalisation, case collisions), which `check` applies to a source directory
and which `pack` never repairs by renaming; the references index (ADR-008
D17). Promised: `pack` writes only layout entries plus `.compiled/`; `unpack`
drops `.compiled/`; round trip per D8 item 5.

**ADR-002 (manifest and versioning).** Relied on: the field names the
scaffold uses — `metais`, `name`, `version`, `title`, `description`,
`guide`; the package name rule (for `init --name` and the package file
name); the one manifest name `metais.yaml` and the profile choice by first
byte, which `example.yaml` also follows; line numbers for manifest
expressions and for `example.yaml`; the canonical JSON; the guide-link rule
that D2 step 4 checks.

**ADR-003 (data specification).** Relied on: input declarations with
`title`, `schema`, `meaning`, an ordered list of `forms`, `csv.delimiter`,
`collection`, and `required` (the modes then apply to inputs too); the
asserted keywords and formats; cross-file `$ref` confined to the package;
the data directory loader and its file extensions; CSV reading CRLF and
reading an empty cell as absent; the effective schema of a Markdown
collection item, including `body`; default application; `--inputs-json`'s
shape (an object keyed by input name); input names' rule (reused for example
names).

**ADR-004 (template environment).** Relied on: the expression grammar and the
complete tag table, stating for each tag which arguments are expressions and
which names it binds (D3 items 7, 8); the filter catalog with each filter's
analysis signature and its property-name arguments (D3 item 10); the fields
of `package`, `fan_out`, `output`, `sources`, `forloop` and `tablerowloop`,
and D7 item 10's table of where each global is bound; the globals visible
inside a partial; how a `render` name resolves to a file; `size`/`first`/
`last` behaving as in Liquid's lookup; raw input bytes reachable only
through `embed`; reserved names refused at parse (`template.reserved_name`);
D6 item 2's declared set, of which D3 item 4's is a superset that adds only
names an open map admits. Promised: every template is analysed with exactly
this dialect and nothing more.

**ADR-005 (outputs and rendering).** Relied on: outputs with `id`,
`template`, path template, `when`, `for_each`; which templates the mirror
shorthand renders and how it excludes partials; which outputs are containers,
their entry lists and their `stored` entries; ADR-005 D13's phases 1 to 4 as
an in-memory render that returns the output tree (path → bytes, with
container entries readable) and applies transforms and output validation;
ADR-005 D14's staging names, which `unpack` and `pack` reuse. Promised:
`test` and `render --sample` call the same render as `render`.

**ADR-006 (findings).** Relied on: a finding with code, severity
(`error`, `warning`, `info`), message, hint, a location (file, line, column,
JSON Pointer), related locations, and params for diffs, visible text and
declared names; ordering, de-duplication and exit codes, with `check`,
`test`, `pack`, `unpack`, `sample` and `render --sample` in the authoring
role (ADR-006 D12 item 2). Promised: the codes named in this ADR, in
ADR-006's families — `package.layout.*` and `limit.package` for the source
tree, `manifest.*`, `schema.*` (including `schema.sample_unsatisfiable` and
`schema.sample_invalid`), `template.*` (including `template.xref.*` and
`template.ooxml.*`), `example.*`, and `report.step_skipped`.

**ADR-008 (command line).** Relied on: `init [<dir>] [--name] [--dry-run]`,
`check [<pkg>]`, `test [<pkg>] [--example <name>]… [--update] [--dry-run]`,
`pack [<dir>] [--out <file>|-] [--dry-run]`, `unpack <pkg> --to <dir>
[--dry-run]`, `render <pkg> --sample [--optional] [--seed] [--items] --out
<dir|->` and `sample [<pkg>] (--to <dir> | --example <name>) [--optional]
[--seed] [--items] [--overwrite] [--dry-run]`, with the supply flags of
ADR-008 D7; the JSON envelope for each; and refusals that name the next
step. Promised: the behaviour behind each. `inspect` may list examples with
the `title` and `description` from `example.yaml`.

**ADR-009 (security).** Relied on: the default read limits `check` holds
`pack` to; confinement of every entry; the regular-expression profile and
its translation to RE2 (ADR-009 D10 item 6), which the generator walks; the
rule that no host fact — not the working directory's name — reaches `init`.
The analysis budget (D3 item 13) and the generator's attempt counts and
recursion depth (D9) are fixed constants of the reference tool, not ADR-009
limits. `test`, `sample` and `render --sample` render or validate in memory
under the same limits as `render`; `expected/` containers are read with the
package reader's bounds.

**ADR-010 (implementation).** Relied on: the rendering engine and its
walkable syntax tree (ADR-010 D4), on which D3 runs; `regexprofile`'s
translated `regexp/syntax` tree (ADR-010 D10 item 2). Promised: no new
module dependency — the lenient OOXML scanner (hand-written, since
`encoding/xml` cannot read a template with Liquid inside its tags),
SplitMix64 and the generator are METAIS code over the standard library
(`regexp/syntax`, `crypto/sha256`). Packages, in ADR-010 D2's layout:
`internal/author` (init, check, test, pack, unpack, with the OOXML scanner
under `internal/author/ooxml`), `internal/xref` and `internal/sample`.

**ADR-011 (conformance).** Relied on: vector operations `check`, `test`,
`pack`, `unpack` and `sample` (ADR-011 D8), and `repeatable` results for
`sample` (ADR-011 D10 item 7), since the generator is not normative (D9 item
3). Promised: the vector groups below.

## Conformance

Vectors are needed for every rule a second implementation's `check`, `test`,
`pack` and `unpack` must reproduce:

- **Cross-reference**: each root (`inputs`, `package`, `item` inside and
  outside `for_each`, loop objects, locals, partial parameters); each step of
  "declared" — `properties`, `patternProperties`, explicit
  `additionalProperties: true`, absent `additionalProperties` (error), open
  schema, `$ref` and a recursive `$ref`, `allOf`, a field in one `oneOf`
  branch, a field in `then`, a field only in `not` (error); command names
  with and without a declared property of the same name; `[0]` with
  `prefixItems`, `items: false`, `maxItems`; dynamic keys on records, maps and
  `inputs`; `map: "p"` with a wrong `p`; unused input; unused template;
  unpassed partial parameter; manifest `when`, `path` and `for_each`.
- **Office Open XML**: a split construct in `w:t`, in SpreadsheetML `t` with
  a non-`w` prefix, in DrawingML `a:t` with `xmlns:a` declared on the element;
  a split opener; a construct wholly in markup (no finding); a partial
  fragment with no namespace declaration; each cause hint.
- **Schema statics**: invalid `default`, invalid `examples`, unsatisfiable
  `const`, `required` not declared.
- **Test**: equal; missing, unexpected, differing file; text and binary
  difference details; each hint; containers equal by entries with different
  compressed bytes; entry order and method differences; `now` from
  `example.yaml`; an example with no `expected/`.
- **Pack**: refused on a check error, refused on a test failure, packed with
  warnings; the packed entry set given a source tree with quiet-list and
  other top-level entries; `.compiled/` content; the same source packed
  twice and from two platforms giving equal packages.
- **Unpack**: `.compiled/` absent from the tree; refusal of a non-empty
  target; the round trip.

The sample generator and `init` are not specified and have no vectors. The
reference tool's suite pins them instead: every supported platform generates
identical data for a corpus of schemas and seeds; every 2020-12 keyword and
format the generator handles has a test that its value validates; each
`schema.sample_unsatisfiable` reason has a negative test; each example package of
§4.1 to §4.5 previews under `all` and `none` to output that validates (SC4);
and D1 item 7's scaffold test.

## Alternatives rejected

### Walking `osteele/liquid`'s render tree

`Template.GetRoot()` gives the tag structure, but every expression inside it
is an opaque closure, so the analysis would still need its own expression
parser for every `Args` string. It would also tie a normative check to one
library's internal node types, some of which panic when asked for their
location, and it has no columns. METAIS's own engine (ADR-010 D4) gives the
whole tree, so no second parser is needed.

### A separate analysis front end beside the engine

A parser of its own would keep the analysis independent of the engine's
code, but two parsers of one grammar disagree at the edges, and an agreement
test can only sample where. The engine's tree is the template as the render
reads it.

### Discovering references by tracing a render

Rendering with sample data and recording every lookup sees only the branches
that data takes, needs data (which `check` must not), and depends on the
generator. The static analysis covers every branch.

### "Declared" meaning "allowed"

Under JSON Schema an object without `additionalProperties` allows any name,
so treating everything allowed as declared would make a misspelt field an
error only in schemas that close every object. Few do. The rule adopted
catches the typo and gives the author an explicit way to say "any key".

### Following control flow

Narrowing on `if inputs.x.kind == "a"` to the matching `oneOf` branch would
make findings more precise and the specification much larger, and every
second implementation would have to match it exactly. Union semantics are
simple to state and never report a guarded branch field as an error.

### A normative, cross-implementation generator

Specifying the PRNG, the stream derivation and every per-keyword choice would
let any implementation reproduce sample data, at the price of a large
normative chapter and a generator frozen for the major version, since within
a major the specification only adds. Nobody depends on two tools generating
the same placeholders; saved examples are data.

### `math/rand/v2` or a faker library

`math/rand/v2` promises reproducibility within a run, not across Go
releases. Faker libraries bring locale data, their own distributions and
version drift, and none covers JSON Schema 2020-12. SplitMix64 is a few lines
with published constants.

### A regex-to-string library

A small dependency, but the generator needs readable character choices, the
METAIS PRNG and the same parse tree the validator matches with. The standard
library's `regexp/syntax` already provides the tree.

### Negative examples carrying expected findings

They would let an author test that validation rejects bad data, but examples
are also what suppliers read as the statement of good data (R3 item 5), and a
failing example there misleads. Validation behaviour is proved by the
conformance vectors and by an author's own tests outside the package.

### `pack` refusing to replace an existing package file

Render refuses a non-empty directory to protect data the caller owns. A
package file at `pack`'s output path is `pack`'s own previous product;
refusing would put a delete step in every edit loop.

### Keeping `.compiled/` on unpack

It would be stale after the first edit and is never read from a source
directory.

## Consequences

### Positive

- A newcomer runs `init`, `check`, `test` and `pack` and each succeeds before
  they have written anything, so every later failure is caused by their own
  edit.
- A misspelt field, a removed input still referenced, or a placeholder broken
  by Word is found without data and before any render.
- `check` computes everything `pack` writes, so `pack` fails only on I/O.
- A consumer can run `check` and `test` on a received package and get the
  author's result.
- Sample data stays stable under unrelated schema edits, and what is
  previewed is exactly what is saved.

### Negative

- The analysis depends on the engine's syntax tree exposing everything D3
  needs (expression trees, bindings, spans), which ties ADR-010 D4's AST to
  this ADR's rules.
- Authors with schemas that leave objects open by omission get errors until
  they write `additionalProperties: true`; the finding says so.
- Union semantics accept a field that exists on only one branch even where
  the template reads it unguarded.
- The CRLF warning and the quiet list are conventions an author has to learn,
  and `.gitattributes` assumes git.
- `test --update` removes files under `expected/`.

### Neutral

- Generated values may change between tool versions.
- ODF documents get no split-run finding in v1.

## What this does not decide

- The finding model, codes' final spelling and exit codes (ADR-006); command
  and flag names (ADR-008).
- The dialect's grammar, tag and filter catalogs (ADR-004); this ADR states
  only what the analysis needs from them.
- The zip writing rules and the `.compiled/` file formats (ADR-001,
  ADR-002, ADR-003).
- Scaffolds beyond the minimal one (a `.docx` starter, a site starter).
- A split-run finding for OpenDocument (`text:span`) and EPUB; within 1.x it
  can be added only as a warning, since a new error would stop packages that
  packed before from packing.
- Field-level "unused field" warnings.
- A watch mode or an editor integration.

## Proposed rulings

1. **The analysis walks the syntax tree of METAIS's own engine (ADR-010
   D4), with no second parser.** Lean: yes — one parse means `check` and the
   render cannot disagree, and the tree is specified by the dialect.
   Alternative: a separate analysis front end held to the engine by an
   agreement test; or re-parse `Args` from `osteele/liquid`'s `GetRoot()`.
2. **"Declared" excludes what an absent `additionalProperties` allows**, and an
   annotation-only schema is open. Lean: yes — otherwise typos are errors only
   in fully closed schemas. Alternative: declared means allowed.
3. **Union semantics across `anyOf`, `oneOf`, `if`/`then`/`else` and
   `dependentSchemas`; `not` excluded; no control flow.** Lean: yes, simple and
   never wrong about a guarded field. Alternative: branch narrowing.
4. **A dynamic reference is a warning and, on `inputs`, marks every input
   used.** Lean: yes. Alternative: an error, which would forbid a legitimate
   pattern.
5. **An unpassed partial parameter is a warning; an unused template is a
   warning and its references do not count as use.** Lean: yes.
6. **Split runs are errors, detected by declared namespace with partials
   inheriting the root's prefixes, Office Open XML only.** Lean: yes.
   Alternative: warning; or include ODF now.
7. **An invalid `default` or `examples` entry is an error; `required` naming
   an undeclared property is a warning; a template with CR LF is a warning.**
   Lean: yes.
8. **Examples have an optional `example.yaml` with `title`, `description`,
   `now`.** Lean: yes — `now` is needed for templates that use time, and the
   title serves the supplier. Alternative: a manifest `examples` section.
9. **Examples are positive only in v1.** Lean: yes. Alternative: examples
   that expect findings.
10. **`test --update` makes `examples/<name>/expected/` exactly the render,
    removing stale files there and nowhere else.** Lean: yes. Why: the
    directory is the author's own and wholly the command's; R12 item 3's
    "never deletes" governs rendering into a caller's directory.
    Alternative: report stale files and remove nothing.
11. **Containers in `test` are compared by entries, not bytes.** Lean: yes, as
    conformance does.
12. **`pack` runs tests only when check is clean, has no skip flag, defaults
    to `./<name>-<version>.metais`, and replaces an existing file there
    atomically, with no `--overwrite`.** Lean: yes. Alternative: refuse an
    existing file unless `--overwrite`, as a render's directory is.
13. **The quiet list, warning on other top-level entries, and packing
    dotfiles inside layout trees.** Lean: yes.
14. **`unpack` drops `.compiled/`, takes a required `--to`, refuses a
    non-empty target with no `--overwrite`, does not run check.** Lean: yes.
    Why: stale files in the target would be packed with the source.
15. **`init` defaults `--name` to the last element of `<dir>` as written,
    never to the working directory's name, treats quiet-list entries as
    empty, and writes an example and `.gitattributes`.** Lean: yes.
    Alternative: `--name` always required; no `.gitattributes`.
16. **The generator is non-normative and deterministic per tool version on
    every platform, built on SplitMix64 with a SHA-256-derived stream per
    location and decision.** Lean: yes. Alternative: a specified generator.
17. **Value preference `const`, `examples`, `default`, `enum`, cycling by item
    index; objects and arrays not taken whole.** Lean: yes.
18. **The optional mode also chooses `null` (none) or a non-null type
    (all).** Lean: yes — both are the "empty" branch a template tests.
    Alternative: modes affect only optional properties.
19. **Supplied inputs are fixed points for `sample`.** Lean: yes.
20. **Saved data uses the first declared form and declaration order;
    `sample --example` refuses an existing example directory.** Lean: yes.
21. **The preview is `render --sample`; keeping the data is `sample --to
    <dir>` or `sample --example <name>`** (ADR-008's names). Lean: yes.
    Why: previewing is rendering and shares its flags; keeping data has a
    different destination. Alternative: one `sample` command that prints
    JSON and also renders.
22. **A source `.compiled/` is refused, and operating-system litter in a
    layout tree is packed and warned about** (D2 step 1, D7 item 3, following
    ADR-001 D5). Lean: yes. Alternative: silently exclude litter, which
    makes the packed package differ from the source tree it was read from.

## References

- osteele/liquid v1.9.2, `template.go` (`GetRoot`):
  https://github.com/osteele/liquid/blob/v1.9.2/template.go
- osteele/liquid `render/nodes.go` (node types, `sourcelessNode`):
  https://github.com/osteele/liquid/blob/main/render/nodes.go
- osteele/liquid `expressions/expressions.go`, `expressions/parser.go`,
  `expressions/expressions.y` (expressions compiled to closures):
  https://github.com/osteele/liquid/tree/main/expressions
- osteele/liquid `parser/token.go` (`Token`, `SourceLoc`):
  https://github.com/osteele/liquid/blob/main/parser/token.go
- osteele/liquid CHANGELOG (1.9.0 `render` tag; 1.9.2 release):
  https://github.com/osteele/liquid/blob/main/CHANGELOG.md
- Shopify Liquid `render` tag source:
  https://github.com/Shopify/liquid/blob/main/lib/liquid/tags/render.rb
- Shopify Liquid `Context#new_isolated_subcontext`:
  https://github.com/Shopify/liquid/blob/main/lib/liquid/context.rb
- Shopify Liquid `VariableLookup` (command methods `size`, `first`, `last`):
  https://github.com/Shopify/liquid/blob/main/lib/liquid/variable_lookup.rb
- Liquid template tags documentation (`render`, deprecated `include`):
  https://shopify.github.io/liquid/tags/template/
- Liquid whitespace control: https://shopify.github.io/liquid/basics/whitespace/
- JSON Schema 2020-12 Core (boolean schemas, applicators, `$dynamicRef`,
  unanchored ECMA-262 regular expressions):
  https://json-schema.org/draft/2020-12/json-schema-core
- JSON Schema 2020-12 Validation (formats §7.3, `minLength` counting,
  `minContains`, `multipleOf`):
  https://json-schema.org/draft/2020-12/json-schema-validation
- Microsoft, Structure of a WordprocessingML document (`p`, `r`, `t`; the
  transitional namespace):
  https://learn.microsoft.com/en-us/office/open-xml/word/structure-of-a-wordprocessingml-document
- Microsoft, Working with the shared string table (`sst`, `si`, `r`, `t`; the
  SpreadsheetML namespace; an `x:` prefix):
  https://learn.microsoft.com/en-us/office/open-xml/spreadsheet/working-with-the-shared-string-table
- Microsoft, Structure of a PresentationML document (DrawingML `a:p`; the
  DrawingML namespace declared per element):
  https://learn.microsoft.com/en-us/office/open-xml/presentation/structure-of-a-presentationml-document
- The National Archives, PRONOM fmt/1827, DOCX Strict (the strict
  WordprocessingML namespace): https://www.nationalarchives.gov.uk/pronom/fmt/1827
- Open XML SDK, `TextCharacterPropertiesType.SpellingError` (DrawingML
  `a:rPr` attribute `err`):
  https://learn.microsoft.com/en-us/dotnet/api/documentformat.openxml.drawing.textcharacterpropertiestype.spellingerror
- ECMA-376 Office Open XML File Formats:
  https://ecma-international.org/publications-and-standards/standards/ecma-376/
- Go `math/rand/v2` (no cross-release stability statement):
  https://pkg.go.dev/math/rand/v2
- Go `regexp/syntax`: https://pkg.go.dev/regexp/syntax
- S. Vigna, SplitMix64 reference implementation:
  https://prng.di.unimi.it/splitmix64.c
- RFC 2606, reserved top-level DNS names (`example.com`):
  https://www.rfc-editor.org/rfc/rfc2606
- RFC 5737, IPv4 address blocks reserved for documentation:
  https://www.rfc-editor.org/rfc/rfc5737
- RFC 3849, IPv6 address prefix reserved for documentation:
  https://www.rfc-editor.org/rfc/rfc3849
- RFC 3339, date and time on the Internet: https://www.rfc-editor.org/rfc/rfc3339
- RFC 9562, UUIDs: https://www.rfc-editor.org/rfc/rfc9562
- RFC 4180, CSV: https://www.rfc-editor.org/rfc/rfc4180
- RFC 6901, JSON Pointer: https://www.rfc-editor.org/rfc/rfc6901
- GNU diffutils, unified format:
  https://www.gnu.org/software/diffutils/manual/html_node/Unified-Format.html
- Lorem ipsum (the standard passage): https://en.wikipedia.org/wiki/Lorem_ipsum

## Implementation log

- **2026-10-19, PLAN-2026-10-metais-v1 W4-5.** D8 landed as
  `author.Unpack` and `metais unpack`, items 1 to 4 and item 5's half
  that needs no `pack`: an unpacked committed package opens as a source
  directory that `inspect` reads with the same result apart from
  `source`, its package digest included. A target that does not exist is
  written through `hostfs.StageNew`, an existing one through
  `hostfs.WriteInPlace` once D1 item 3's emptiness holds, every entry but
  `.compiled/`'s in byte order, files 0644 and directories 0755 less the
  umask. An entry the package cannot yield is a failed write of reason
  `io_error`, whose `written` lists nothing for a staged target and the
  files left in place otherwise. Chapter 13's statements are MTS-AUT-024
  to 029.

- **2026-10-19, PLAN-2026-10-metais-v1 W3-6.** D3 landed as
  `internal/xref`, a library `check` wires in Wave 4. `Analyze` takes the
  manifest model, every template under `templates/` as an
  `internal/liquid/ast` tree (nil for one that did not parse), the assets,
  the combined schema `schema.Bundle` makes and the dialect, and parses
  only the manifest's `path`, `when` and `for_each`, through
  `liquid.Parse`, `ParseCondition` and `ParseValue`; a test holds it to
  reporting every path `ast.Walk` finds. A schema node is a pointer into
  the combined schema, and its finding names the file and the pointer
  within it. `schema.Walker` gained `Lookup`, `Reference` (a `$ref`, or a
  `$dynamicRef` to its static target), `DynamicAnchors` and `References`,
  from which an input's schema resources are the files its references
  reach. Besides item 2's five kinds a value may be an array of an element
  set, which `map`, `concat` and a range give. Only a node reached through
  `$ref` and `allOf` alone rules an index out by `maxItems` or `items:
  false`, since an `anyOf`, `oneOf` or conditional branch may not apply.
  A partial reading `item`, `fan_out`, `output` or a loop object its
  context does not bind is `unknown_variable`, not an unpassed parameter.
  A finding repeated by another context is kept once, naming the first
  output whose context produced it. The signature table names
  `liquid.Core()`'s filters, `h` among them, and gives `find`,
  `find_index`, `reject` and `split` unknown; a filter it does not name,
  `date` and D10's until Wave 5 names them, gives unknown. The suggestion
  is item 3's "nearest within edit distance 2", which ADR-006 D9 item 7's
  `max(1, ⌊n/3⌋)` would refuse for the worked resolution's `vaule`. Every
  `template.xref.*` code is registered, of class `content`, and listed
  unwired for Wave 4. Both bounds are reached by tests that count.
- **2026-10-19, PLAN-2026-10-metais-v1 W3-5F.** Item 21's JSON writer
  spells an integral number beyond ±(2^53−1) with an exponent, as the
  plan's §9 ruling B′ says: `jsonx.EncodeInput` writes what `jsonx.Encode`
  writes with each number by `jsonx.FormatInputNumber`, which the YAML
  writer shares, so `Files` refuses such a supplied number only in a CSV
  field. The generator still makes none, as W3-5 reads it. `jsonx.Encode`,
  the `json` filter's text, and `jsonx.Canonical` are unchanged, and so is
  every reader.
- **2026-10-19, PLAN-2026-10-metais-v1 W3-5.** D9 items 2 and 4 to 23
  landed as `internal/sample`, a library ahead of the commands that reach
  it (item 1, Wave 5). `Generate` makes one input's value from the
  package's compiled schemas, validating each value it keeps against its
  location's subschemas through `schema.Compiler.Subschema`; `Files`
  writes it as item 21 says, through `jsonx.Encode`, `yamlx.Write` and
  `csv.WriteCells`. `schema.sample_unsatisfiable` is registered with
  item 23's reasons as fixed words, about the input, located at the
  subschema in its schema file, with the instance as a parameter. Where
  the text left a choice, the library takes these readings:
  - the JSON and YAML writers write members in canonical order, which is
    the order of every serializer of the data model (ADR-004 D4 item 5),
    not item 21's declaration order, which a value does not keep; a CSV
    header is in declaration order, read from each schema file's position
    table. The YAML writer double-quotes only the strings ADR-004 D10
    item 2 does; each reads back as itself;
  - a text file and a Markdown body always end in one added LF, whatever
    the string ends in, since ADR-003 D12 removes one; and a Markdown
    input is an object, as ADR-003 D1 item 6 requires, never a string;
  - a chosen anyOf or oneOf branch from which no value can be made is
    passed over for the next, as a oneOf mismatch is; a scalar that fails
    a selected then or else is made again with the branch added, so
    contradicting branches are a contradiction (item 9's last sentence);
  - under `mixed`, null is chosen when the type stream's first draw is
    odd; a boolean is true when the value stream's first draw is even;
  - no generated or authored value is kept that its form cannot write so
    that it reads back: a string with a noncharacter, a CR where line
    breaks are normalized, and an integral number of magnitude above
    2^53−1, which the JSON writer and a CSV field write as an integer
    literal ADR-003 D8 item 5 refuses. The numeric range is held within
    ±(2^53−1), and a range wholly outside it is `range`. A supplied value
    is taken as given (item 2), so `Files` refuses one holding such a
    number in JSON or CSV rather than write a file its reader refuses;
    YAML spells it with an exponent, which reads back;
  - the generator's work is held to `input-size`, the limit the data it
    makes is held to (item 22): a schema asking for more is
    `limit.exceeded` about the input, before the work is done.
- **2026-10-19, PLAN-2026-10-metais-v1 W4-NB2.** W4-6 made the engine
  read the `size`, `first` and `last` shorthands only after a `.`, and D3
  item 4.5 now says the same of the cross-reference, which read
  `xs["first"]` as the shorthand and so was silent where render refuses.
  A bracketed one on an array or a string is `template.xref.not_an_object`,
  and on an object that does not declare it `template.xref.unknown_field`,
  by item 4.6. `aut/check-xref-bracketed-shorthands` proves it under
  MTS-AUT-013, and `TestShorthandAgreement` holds render and check to the
  same outcome for each form on an array, a string and an object.
- **2026-10-19, PLAN-2026-10-metais-v1 W5-3.** D3 item 12 is reachable:
  with `embed` in the dialect, the cross-reference reports an undeclared
  name as `template.xref.unknown_input`, with the nearest declared name,
  and counts an `embed` of an input as a use (item 11). Chapter 13's
  MTS-AUT-030 states it, and `aut/check-xref-embed` proves it. Check also
  reports `template.embed_index` for an `index:` given to an input that is
  not a collection or missing for one, from the function render uses
  after the one parse (ADR-004's log).
- **2026-10-19, PLAN-2026-10-metais-v1 W5-10.** D9 item 1 is reachable:
  `metais sample` and `render --sample` reach `internal/sample` through
  one step in `internal/cli`. The supply flags are assembled by ADR-003
  D18 into the fixed points (`inputs.Supply.Fixed`), every input is
  generated and written in its first form, and the files are read back in
  memory through the data-directory reader (`inputs.Supply.ReadFiles`) and
  validated, so the data previewed is the data saved: `render --data` of
  what `sample --to` wrote renders byte for byte what `render --sample`
  rendered. Schema files are read again in place for their position
  tables (`loader.Package.SchemaPositions`), which give item 15's
  declaration order and a finding's line. Chapter 13's §13.6 states
  MTS-SMP-001 to 009, and `smp/` holds their vectors: every `sample` step
  is `repeatable`, and every one that writes is followed by a `validate`
  of its data. The readings taken:
  - **Item 22's `schema.sample_invalid`** is registered: one error for each
    validation error about an input, about that input, with the same
    field, schema keyword and conditions, located at the same place in the
    generated file (origin `sample`), and the code validation gave as its
    `code` parameter. A supplied value is part of the data, so a value
    given with `--set` that the schema refuses is reported this way, and
    the hint says so. A cross-input constraint, about no one input, keeps
    its own code. A supplied value whose input's first form cannot write
    it so that it reads back (item 21's writers refuse it) is
    `schema.sample_invalid` with `code` `form`.
  - **Item 2 when the schema makes no object.** A supplied object at a
    place whose conjunction allows no object, or has a `const` or `enum`,
    is kept as given rather than replaced by a generated value, so the
    validation that follows refuses it; W3-5 had generated an array there
    and dropped what was given.
  - **Item 21's example.** `example.yaml` holds `title: Sample data` and a
    double-quoted `description` naming the seed, mode and item count and
    the `metais test --example <name> --update` step. The example is staged
    beside `examples/<name>/` and renamed into place with a rename that
    never replaces; an `examples/<name>` that exists, empty or not, is
    `output.destination.write_failed` with `reason` `exists` and nothing
    written.
  - **A collection with no item** written into an existing directory gets
    its empty directory after the commit, created only where missing, so
    the data reads back as the empty collection it is.

# ADR-004: The template environment — Shopify Liquid 5.14.0 parsed strictly, a closed set of bindings, strict on every name, and a METAIS filter and tag catalog defined to the byte

**Status:** Accepted 2026-10-18 by Ed Funnekotter; its proposed rulings are ratified in [RULINGS-v1](RULINGS-v1.md).
**Date:** 2026-10-18
**Deciders:** Ed Funnekotter
**Designs:** PRD-001 R7 (items 1 to 6) and R11 (items 1 and 2); the
template side of R5 item 4 (absence is `nil`), R10 items 3 and 4
(defined order, no locale, time only when supplied) and R14 item 4 (nothing
of the host reaches a template).
**Depends on:** ADR-001 (entries under `templates/` and `assets/`, read in
place; the package digest and its `sha256:<hex>` presentation); ADR-002 (the
manifest members `package` exposes; RFC 8785 as the canonical JSON form; a
`1.0` tool refuses what a later minor adds); ADR-003 (the data model the
bindings carry, the combined schema, an input's raw bytes); ADR-005 (outputs,
`escape`, output paths, fan-out, assets' output paths); ADR-006 (the finding
model); ADR-009 (the limits this ADR names).

## Context

### What the PRD requires

R7 makes Liquid the only template language, "in a dialect the specification
pins: Liquid's core tags and filters as the Shopify reference implementation
defines them, without the deprecated `include`, plus the METAIS filters and
tags". Every template sees every input as `inputs.<name>`; partials come from
the package's templates only; standard filters serialize to JSON, YAML and
CSV, build a Markdown table, escape HTML, make JSON safe inside `<script>`,
make a slug and format dates; standard tags embed an input's bytes, include
an asset or its output path, and emit provenance. Escaping is chosen per
output. R11 closes what provenance may record. R10 demands byte-identical
output on every platform with no locale, time-zone or clock dependence, and
R14 item 4 forbids anything of the host reaching a template. ADR-002 D7
item 4 relies on this ADR refusing an undefined variable, an undefined
member of `package`, an unknown filter and an unknown tag, so that a later
minor can add one without changing what an earlier package renders.

### The reference implementation, verified

The Shopify reference implementation is the `liquid` Ruby gem. Its newest
release is **5.14.0**, published to RubyGems on 2026-09-17 and tagged
`v5.14.0` (commit `4e39ae4cc3da73921923c0669e0fc84a66b2f696`) in
`Shopify/liquid`. Read at that tag:

- **Standard tags** (`lib/liquid/tags.rb`, `STANDARD_TAGS`): `assign`,
  `break`, `capture`, `case`, `comment`, `continue`, `cycle`, `decrement`,
  `doc`, `echo`, `for`, `if`, `ifchanged`, `include`, `increment`, `raw`,
  `render`, `tablerow`, `unless`, and `#` (inline comment). The `liquid` tag
  is parsed by `BlockBody` itself, not registered.
- **Standard filters** (`lib/liquid/standardfilters.rb`): `abs`, `append`,
  `at_least`, `at_most`, `base64_decode`, `base64_encode`,
  `base64_url_safe_decode`, `base64_url_safe_encode`, `capitalize`, `ceil`,
  `compact`, `concat`, `date`, `default`, `divided_by`, `downcase`, `escape`
  (alias `h`), `escape_once`, `find`, `find_index`, `first`, `floor`, `has`,
  `join`, `last`, `lstrip`, `map`, `minus`, `modulo`, `newline_to_br`,
  `plus`, `prepend`, `reject`, `remove`, `remove_first`, `remove_last`,
  `replace`, `replace_first`, `replace_last`, `reverse`, `round`, `rstrip`,
  `size`, `slice`, `sort`, `sort_natural`, `split`, `squish`, `strip`,
  `strip_html`, `strip_newlines`, `sum`, `times`, `truncate`,
  `truncatewords`, `uniq`, `upcase`, `url_decode`, `url_encode`, `where`.
  The documentation site's filter index lists fewer (it omits the `base64_*`
  family, `squish`, `find`, `find_index`, `has` and `reject`); the gem is the
  reference.
- **Error modes** are `lax` (the default), `warn`, `strict` and `strict2`
  (renamed from `rigid` in 5.11.0). `strict_variables` and `strict_filters`
  are separate render options, both off by default; with them off an
  undefined name renders as nothing and an unknown filter passes its input
  through. Without `render!`, a render error is written *into the output* as
  `Liquid error: …`.
- **Truthiness**: only `nil` and `false` are falsy; an empty string is
  truthy (documentation, "Truthy and falsy").
- **`empty` and `blank`** (`condition.rb`): `x == empty` holds for an empty
  string, array or hash, and not for `nil`; `x == blank` holds for `nil`,
  `false`, an empty or whitespace-only string, and an empty array or hash.
  Used as a value, either is `''`.
- **The blank-body rule** (`block_body.rb`, `tags/if.rb`, `tags/for.rb`,
  `tags/case.rb`): an `if`, `unless`, `case` or `for` whose bodies contain
  only whitespace and blank tags (`assign`, `capture`, `comment`, `#`, an
  empty `raw`, an empty `doc`, and blocks that are themselves blank) drops
  that whitespace from the output.
- **Whitespace control** trims with Ruby `lstrip`/`rstrip` (space, `\t`,
  `\n`, `\v`, `\f`, `\r`).
- **Output of values** (`variable.rb`, `utils.rb`): an array renders as its
  elements concatenated; a hash renders with Ruby-style `inspect`
  (`{"a"=>1}`); a float renders by Ruby `Float#to_s`.
- **Arithmetic** (`apply_operation`, `Utils.to_number`): a float operand is
  converted to `BigDecimal` from its shortest decimal text and the result
  converted back, so `0.1 | plus: 0.2` is `0.3`. Integer division and modulo
  floor (Ruby semantics).
- **`date`** accepts anything Ruby's `Time.parse` accepts, plus `"now"` and
  `"today"`, which read the clock, and formats with `strftime`.
- **`render`** requires a quoted template name, renders in an isolated
  context that sees only what is passed and the static environment, and
  supports `with`, `for` (binding `forloop`) and `as`.
- **`tablerow`** writes `<tr class="row1">\n`, `<td class="colN">`,
  `</td>`, `</tr>\n<tr class="rowN">` between rows, and `</tr>\n` at the end.
- **Nesting**: `Block::MAX_DEPTH` is 100, for parse nesting and for render
  scope depth ("Nesting too deep").
- **5.13.0 introduced a `self` keyword** for looking up a variable by a
  computed name.

Ruby's own formatting, measured with Ruby 2.6.10: `3.0.to_s` is `3.0`,
`1e14.to_s` is `100000000000000.0`, `1e15.to_s` is `1.0e+15`,
`0.0001.to_s` is `0.0001`, `0.00001.to_s` is `1.0e-05`; `CGI.escapeHTML`
maps `&<>"'` to `&amp;`, `&lt;`, `&gt;`, `&quot;`, `&#39;`; `strftime("%Z")`
on a fixed-offset time is the empty string, and on UTC is `UTC`.

### The Go library, verified

`github.com/osteele/liquid` **v1.9.2** (2026-08-16, commit
`682200b0d1491cdf96a94fe83036141721070fa0`, MIT) is a pure-Go Liquid. It
offers what this design needs as mechanisms: `StrictVariables()` (which since
1.9.1 distinguishes a missing key from a bound `nil`), undefined filters as
errors by default, a `TemplateStore` interface for partial loading,
`SetAutoEscapeReplacer` with a `safe` filter, `FRender` to a writer,
`NewBasicEngine` with no filters or tags, `RegisterFilter`, `RegisterTag`,
`UnregisterTag`, and errors carrying a path and line. Read at that tag, it
**differs from Shopify 5.14.0** in ways a conformance vector would catch:

1. No `empty` or `blank` literal; either parses as a variable name.
2. No `echo`, `liquid`, `#`, `increment`, `decrement`, `ifchanged` or `doc`
   tag (`tags/standard_tags.go`).
3. `escape` is Go's `html.EscapeString`, which writes `"` as `&#34;`, not
   `&quot;`.
4. Loop modifiers apply as `reversed`, `offset`, `limit` in any position
   (`docs/loop-semantics.md`); Shopify applies `offset`, `limit`, then
   `reversed`, and recognizes `reversed` only before named parameters.
5. `cycle` outside a `for` is an error; `forloop.parentloop` is absent.
6. `tablerow` writes no line feeds.
7. `render` accepts any expression as the template name, and resolves it
   relative to the calling template's directory, refusing an escape from it
   (`tags/template_path.go`); a rendered partial sees no globals.
8. Double-quoted string literals take backslash escapes (1.9.0); Shopify's
   do not.
9. Floats render with `strconv.FormatFloat(f, 'f', -1, 64)` (`3` for 3.0,
   all digits for 1e21); arithmetic is float64 (`0.1 | plus: 0.2` is
   `0.30000000000000004`); `divided_by` truncates a float dividend to an
   integer when the divisor is an integer; integer division truncates rather
   than floors; `modulo` is `math.Mod`; `round` rounds half up toward
   positive infinity.
10. `date` maps `"now"` to `time.Now()` and parses in `time.Local`
    (`values/parsedate.go`) — a clock and a time-zone read.
11. Its default filter set includes `json`, `inspect`, `type`, `handleize`,
    `handle` and `pluralize`, which are not Shopify core filters, and lacks
    the `base64_*` family, `squish`, `reject`, `has`, `find`, `find_index`
    and the `h` alias.
12. Strict mode errors on a property of `nil`, as Shopify's
    `strict_variables` does (`variable_lookup.rb`).

Because several of these differences live where no caller of the library
can reach, ADR-010 D4 decides that METAIS implements its own engine; this
ADR fixes the dialect that engine implements.

## Decision

### D1 — The dialect is Shopify Liquid 5.14.0, parsed by its `strict2` grammar, minus `include` and `self`

1. **The reference is the `liquid` gem 5.14.0** (tag `v5.14.0`). Wherever
   this ADR says "as Shopify", it means that version's behaviour with
   `error_mode: :strict2`, `strict_variables: true` and
   `strict_filters: true`, rendered with `render!` (errors raise, never
   print). The specification restates every rule as its own normative text;
   where its text and the gem differ, the specification governs, and the
   conformance vectors prove it.
2. **The grammar is Shopify's `strict2` grammar.** Text that `strict2`
   refuses is a syntax finding. Where Shopify ignores a misplaced element
   rather than refusing it (for example `reversed` after a named loop
   parameter), METAIS refuses it.
3. **The delimiters are fixed**: `{{ }}` for output, `{% %}` for tags, with
   `-` for whitespace control. A package cannot change them.
4. **The tags are D2's; the filters are D10's.** The standard filters are
   every filter listed in *Context* for 5.14.0, with `date` replaced by D11's
   definition and `upcase`, `downcase` and `capitalize` using Unicode 15.0.0
   *simple* case mapping (one code point to one code point, no special
   casing, no locale), where Ruby uses full mapping (`"ß" | upcase` is `ß`,
   not `SS`).
5. **Excluded**: `include` (a template using it is refused with
   `template.include`, naming `render`); the `self` keyword (`self` is an
   ordinary, unbound name); every filter or tag of Shopify's theme platform
   (`asset_url`, `money`, `handleize`, `pluralize`, `t`, `json` in Shopify's
   theme sense, `form`, `section`, `paginate` and the rest); and anything a
   library adds beyond this list.
6. **A name outside the dialect is refused, never passed through.** An
   unknown tag is a syntax finding. An unknown filter is a finding when the
   template is parsed, whether or not the branch containing it would run.
   This is what lets a later `1.x` add a filter or tag (ADR-002 D7 item 4):
   no `1.0` package can have relied on its absence.
7. **Template text** is UTF-8 and must be valid; a template containing U+0000
   is a syntax finding. A byte-order mark, CR LF line endings and every other
   character are part of the template's text and reach the output exactly as
   written. Nothing is normalized.
8. **String literals carry no escapes.** `"a\nb"` is four characters, as in
   Shopify. A literal containing its own quote character uses the other
   quote.

### D2 — Every core tag but `include`, each as Shopify defines it

| Tag | In | Notes |
|---|---|---|
| `if` / `elsif` / `else`, `unless` / `else`, `case` / `when` / `else` | yes | `and` and `or` group from the right, as Shopify's (`a or b and c` is `a or (b and c)`; `a and b or c` is `a and (b or c)`); the left operand of each is evaluated first, and an `and` stops at a falsy left operand, an `or` at a truthy one; no parentheses; `when` takes several values separated by `,` or `or` |
| `for` / `else`, `break`, `continue` | yes | `limit:`, `offset:` (including `offset: continue`) and `reversed`; ranges `(a..b)` of integers; `reversed` only immediately after the collection; applied as `offset`, then `limit`, then `reversed`; `forloop` with `index`, `index0`, `rindex`, `rindex0`, `first`, `last`, `length`, `parentloop`, `name` |
| `cycle` | yes | inside or outside a loop; an optional group name (`{% cycle "g": "a", "b" %}`) keys the position |
| `tablerow` | yes | `cols:`, `limit:`, `offset:`, `tablerowloop`; markup exactly as in *Context* |
| `assign`, `capture` | yes | template-scoped (D7 item 8) |
| `increment`, `decrement` | yes | counters in their own namespace, as Shopify |
| `ifchanged` | yes | |
| `echo` | yes | an output expression; autoescaped like `{{ }}` (D9) |
| `liquid` | yes | one tag per line; `echo` writes; line numbers are those of the inner lines |
| `comment`, `#`, `doc` | yes | render nothing; `comment` nests and skips a `raw` inside it; a `#` comment spanning lines has `#` on every line; `doc` does not nest |
| `raw` | yes | its body is written verbatim |
| `render` | yes | D8 |
| `include` | no | `template.include` |
| `embed`, `asset`, `provenance` | METAIS | D12, D13, D14 |

A tag not in this table is a syntax finding.

### D3 — Whitespace is written as the template has it, except where control or the blank-body rule removes it

1. **Whitespace control**: `{%-`, `{{-` remove all whitespace immediately
   before the token; `-%}`, `-}}` remove all whitespace immediately after it,
   up to the next non-whitespace character or token. Whitespace here is
   U+0020, U+0009, U+000A, U+000B, U+000C and U+000D.
2. **No other trimming.** A tag alone on a line leaves its line feed in the
   output. There is no global trim mode.
3. **The blank-body rule is part of the dialect**, as stated in *Context*:
   an `if`, `unless`, `case` or `for` all of whose branch bodies are blank
   writes none of their whitespace. It is surprising, and it is pinned
   because it is what the reference does and it decides bytes.

### D4 — Values are JSON values with two kinds of number; each has one text

1. **The value types** are `nil`, boolean, integer, float, string, array and
   object. Strings are sequences of Unicode scalar values. Object member
   names are strings.
2. **Number kinds.** ADR-003's data model carries numbers as I-JSON doubles
   (as ADR-002 D4 item 3 does for the manifest). In a template, **a data
   number whose value is an integer of magnitude at most 2⁵³ − 1 is an
   integer; every other data number is a float.** A numeric literal in a
   template takes its kind from its spelling (`4` integer, `4.0` float), so
   Shopify's `divided_by: 4.0` idiom still means float division.
3. **Arithmetic** (`plus`, `minus`, `times`, `divided_by`, `modulo`, `abs`,
   `ceil`, `floor`, `round`, `at_least`, `at_most`, `sum`):
   - integer with integer gives an integer; `divided_by` and `modulo` floor,
     as Ruby's `/` and `%` do;
   - with a float operand, the result is the exact real result computed on
     each operand's shortest round-trip decimal value, rounded to the nearest
     double, ties to even — the decimal semantics Shopify gets from
     `BigDecimal`, defined exactly;
   - `round` rounds half away from zero and returns an integer when its
     argument is omitted or `0`;
   - an integer result of magnitude above 2⁵³ − 1 is `template.number_range`;
     division or modulo by zero is `template.division_by_zero`.
4. **Text of a value**, used by `{{ }}`, `echo`, `cycle`, string filters and
   the tabular filters:
   - `nil`: the empty string; `true`, `false`: `true`, `false`;
   - integer: its decimal digits, with `-` when negative;
   - float: Ruby `Float#to_s` — the shortest digits that round-trip, written
     in fixed notation with at least one digit after the point when
     1e-4 ≤ |x| < 1e15 (`3.0`, `0.0001`, `123456789012345.0`), otherwise as
     `d.ddde±XX` with at least two exponent digits (`1.0e+15`, `1.0e-05`);
     zero is `0.0`, negative zero `-0.0`;
   - string: itself;
   - array: the texts of its elements concatenated, recursively;
   - **object: refused** (`template.object_output`, naming `json`). Ruby's
     `inspect` format is not reproducible, and writing it was never what the
     author meant.
5. **Order.** Iterating an object (`for pair in obj`, `first`, `last`,
   `map`, and every METAIS serializer) visits members in **canonical order:
   ascending by the UTF-16 code units of the name**, the order RFC 8785
   uses. The order in which a supplier wrote the members never matters, so
   the same value supplied as YAML, as JSON or by `--set` renders the same
   bytes. Iterating an object yields two-element arrays `[name, value]`, as
   Shopify does.
6. **Comparison** is Shopify's: `==`, `!=`, `<>`, `<`, `>`, `<=`, `>=`,
   `contains`. Integers and floats compare by value. `<` and its kin between
   a string and a number, or with an array or object, are
   `template.comparison`; with `nil` on either side they are false.
   `contains` tests substring, array membership, or object member name.
   Strings compare by code point; nothing compares by locale.
7. **Property shorthands** `size`, `first` and `last` apply to strings,
   arrays and objects as in Shopify; an object member with that name wins.

### D5 — Only `nil` and `false` are falsy; `empty` and `blank` are Shopify's

1. **Truthiness** is Shopify's: `nil` and `false` are falsy, everything else
   — `0`, `""`, `[]`, `{}` — is truthy. Because an omitted optional field is
   `nil` (ADR-003, R5 item 4), `{% if inputs.x.note %}` is the idiom for
   "supplied", and the specification's authoring chapter says so beside the
   rule.
2. **`empty` and `blank` are literals with exactly the semantics in
   *Context***: `x == empty` for an empty string, array or object;
   `x == blank` additionally for `nil`, `false` and a string of only
   U+0020, U+0009, U+000A, U+000B, U+000C and U+000D. `!=` negates. As a
   value outside a comparison, each is `""`. They are not names, and cannot
   be assigned.

### D6 — Strict on every name: what the schema declares and the data omits is `nil`; what neither has is an error

1. **A top-level name** resolves, in order, to a block- or partial-scoped
   binding, a template-scoped binding, a counter, or a global (D7). Anything
   else is `template.undefined_variable`.
2. **The declared properties** of an object in the data are the member names
   in `properties` of the subschema at its location in ADR-003's combined
   schema, and of every subschema reachable from that location through
   `$ref`, `allOf`, `anyOf`, `oneOf`, `if`, `then`, `else` and
   `dependentSchemas`. An array element's location is `items` (or the
   matching `prefixItems` entry); a collection item's is the item schema; a
   CSV row's is the row schema. `patternProperties` and `additionalProperties`
   declare no names. `check`'s cross-reference (ADR-007 D3 item 4) accepts
   this set and, besides it, names admitted only by `patternProperties` or
   an explicit `additionalProperties`; such a name passes `check` and is
   readable at render only when the data carries it (item 4).
3. **Before rendering, every declared property absent from an object is
   present with the value `nil`.** This is how R5 item 4's "absent is `nil`"
   reaches templates; an absent object is not created, and its own declared
   properties do not appear.
4. **Property access** (`a.b`, `a["b"]`, `a[expr]`) on an object yields the
   member's value when present (including a declared `nil`); when absent —
   neither declared nor supplied — it is `template.undefined_property`.
   A member the data carries without the schema declaring it (allowed by
   `additionalProperties`) is present and readable.
5. **A property of `nil` is `template.nil_property`**, as with Shopify's
   `strict_variables`. The finding says to guard with `{% if a %}`;
   `{% if a and a.b %}` is safe because `and` stops at a falsy left
   operand (D2).
   Applying a filter, including `default`, does not rescue it.
6. **An array index** that is an integer out of range yields `nil`;
   negative indices count from the end; a non-integer index is
   `template.undefined_property`. A property other than `size`, `first` and
   `last` on a string, number or boolean is `template.undefined_property`.
7. **`package` is closed** (D7 item 3): an unknown member is
   `template.undefined_property`. Every global object is closed the same way.
8. **When each is detected.** Syntax, unknown tags, unknown filters,
   reserved names (D7 item 8), and the static arguments of `render`, `embed`,
   `asset` and `provenance` are found when the template is parsed, with no
   data. Undefined variables and properties are found when evaluated; ADR-007
   finds the statically determinable ones without data.

### D7 — The bindings are closed: the globals and what the template itself creates

1. **Nothing else is bound.** A template sees the globals below, the names
   its own constructs create, and nothing more: no environment variable,
   working directory, user, host, process, clock or file outside the package
   and the named inputs (R14 item 4). No filter or tag reads anything but its
   arguments, the package's entries (`render`, `asset`) and the supplied
   inputs (`embed`). The binding set is the same for every caller; ADR-009
   proves it with planted sentinels.
2. **`inputs`** is an object with a member for every input the manifest
   declares, named by the input's name. Its value is the validated value
   with defaults applied and declared-absent properties `nil` (D6 item 3).
   An optional input not supplied is `nil`. `inputs.<undeclared>` is
   `template.undefined_property`.
3. **`package`** is an object whose members are the canonical manifest's
   `metais`, `name`, `version`, `title`, `description`, `guide`, `authors`,
   `license` and `homepage` (ADR-002 D4; an omitted optional member is
   `nil`), plus **`digest`**, the package digest as `sha256:<64 lowercase
   hex>` (ADR-001 D6), which a source-directory render and a packed render
   see equally. `inputs` and `outputs` are not members; templates read data
   through `inputs`.
4. **`item`** is bound only while rendering an output that has `for_each`
   (and that output's path and per-item expressions, ADR-005): the current
   element of the `for_each` array. Elsewhere it is unbound.
5. **`fan_out`** is bound exactly where `item` is: `index` (from 1),
   `index0`, `rindex`, `rindex0`, `first`, `last`, `length`, and
   **`previous`** and **`next`**, the neighbouring elements of the
   `for_each` array (`nil` at either end), so that a page links to its
   siblings without index arithmetic. Positions are those of the `for_each`
   array before a per-item `when` is applied (ADR-005 D3), so `length`,
   `last`, `previous` and `next` count elements `when` drops; an author who
   wants neighbours among the written pages filters in the `for_each`
   expression instead.
6. **`output`** is bound while rendering an output's body: `path`, the
   output's rendered path relative to the root of its tree with `/`
   separators (for an entry of a zip container, its path inside the
   container); and `root`, the relative prefix from that path's directory to
   the root of its tree (`""` at the top, `"../"` one level down, and so on).
   It is unbound in path templates and in `when` and `for_each` expressions.
7. **`now`** is the caller's render time (`--now`, ADR-008) normalized to
   UTC as `YYYY-MM-DDTHH:MM:SS[.F]Z`, where `F` is the supplied fraction with
   trailing zeros removed and the point omitted when nothing remains; `nil`
   when the caller supplied none. It is the only source of time.
8. **Reserved names and shadowing.** The globals (`inputs`, `package`,
   `item`, `fan_out`, `output`, `now`, `sources`) and Liquid's `forloop` and
   `tablerowloop` may not be bound by a template-scoped construct —
   `assign`, `capture`, `increment`, `decrement` — which is
   `template.reserved_name`. A block- or partial-scoped binding — a `for` or
   `tablerow` variable, a `render` parameter, a `render … as` alias — may
   use any name, and shadows a global only within its block or partial, so
   `{% render "card" for inputs.articles as item %}` reads as written.
9. **`sources`** exposes the `source` of ADR-003's input records (D21 item
   3), because a value has no identity in Liquid. It has a member per
   declared input: for an input that is not a collection, `{"origin": …}`
   with `origin` one of `"file"`, `"command-line"`, `"default"`, or `nil`
   when the input is absent; for a collection, an array parallel to
   `inputs.<name>` whose element is `{"name": …, "stem": …, "form": …}`
   (the item's data-directory file name, its stem and its form) or `nil` for
   an item supplied on the command line. The golden way to name one page per
   item after its file is `for_each: inputs.articles`, a per-item `when` to
   drop the ones not wanted, and `sources.articles[fan_out.index0].stem` in
   the path — `fan_out` counts positions in the `for_each` array before
   `when`, so the index is the item's position in the collection. A
   `for_each` that filters or reorders loses that correspondence, and the
   specification says so beside `sources`.
10. **Where each global is bound.** This table is the one statement of the
    bindings; ADR-005 D3 and D4 decide which expressions exist and point
    here.

    | Context (ADR-005) | `inputs` `package` `now` `sources` | `item` `fan_out` | `output` |
    |---|---|---|---|
    | a declaration's `for_each` expression | yes | no — it computes them | no |
    | `when` of a declaration without `for_each`, and not inside a `for_each` container | yes | no | no |
    | `when` of a `for_each` declaration (evaluated per element), or of any declaration inside a `for_each` container | yes | yes | no |
    | `path` template | yes | as for `when` | no — the path is being computed |
    | template body, and every partial it renders | yes | in a fan-out only | yes |

    `forloop` and `tablerowloop` exist only inside their blocks, in every
    context where those tags are allowed.

### D8 — Partials come from `templates/` by literal path, isolated, with the globals visible

1. **`render` takes a string literal** naming an entry path relative to
   `templates/`, with its full file name and extension
   (`{% render "partials/row.md.liquid" %}`). No suffix is added, no
   directory of the caller is consulted, and the name is matched byte for
   byte. A non-literal name is a syntax finding (`template.partial_name`).
2. **The name is confined**: no leading `/`, no `\`, no `.` or `..` segment,
   no empty segment. A name that does not resolve to an entry under
   `templates/` is `template.partial_not_found`, found when the calling
   template is parsed.
3. **Forms**: `render "p"`, `render "p", k: v, …`, `render "p" with x [as
   n]`, `render "p" for xs [as n]` (binding `forloop` in the partial), as
   Shopify.
4. **Isolation**: a partial sees its parameters, its own bindings and the
   globals (D7), and none of the caller's `assign`, `capture`, loop or
   counter bindings. A partial cannot change anything the caller sees.
5. **Escaping and output identity follow the output being rendered**: a
   partial rendered into an `escape: html` output is escaped (D9), and
   `output`, `item` and `fan_out` are the including output's.
6. **Recursion is allowed and bounded.** The render depth — the number of
   `render` calls active at once — is limited by ADR-009's `partial-depth`
   (default 32); exceeding it at render is ADR-009's `limit.exceeded`,
   naming the limit and the chain. Block nesting at parse time is bounded by
   ADR-009's `template-nesting` (default 100, Shopify's `MAX_DEPTH`), and
   exceeding it is `limit.package`, since the package's own text breaches
   it (ADR-009 D19).
7. **Partials are read in place** from the package, never from the file
   system of the host.

### D9 — `escape: html` escapes every value an expression writes, unless it is already safe

1. **Scope.** In an output whose `escape` is `html` (ADR-005), the text of
   every value written by `{{ }}`, `echo` and `cycle` passes through the
   `escape` mapping: `&` → `&amp;`, `<` → `&lt;`, `>` → `&gt;`, `"` →
   `&quot;`, `'` → `&#39;`. The template's own text is never escaped. In an
   output whose `escape` is `none` (the default), nothing is escaped.
2. **Safe values are written as they are.** A value is safe when the last
   filter applied to it is `safe`, `escape`, `escape_once` or `json_script`,
   or when it is the result of a `capture` made while rendering an
   `escape: html` output. Safety is not carried through a later filter:
   `{{ x | escape | newline_to_br | safe }}` is the idiom for escaped text
   with line breaks.
3. **`safe`** is a METAIS filter that returns its input unchanged and marks
   it safe. It may appear only as the last filter of an output expression
   (`template.safe_position` otherwise). In an `escape: none` output it does
   nothing, so one partial serves both kinds of output.
4. **Tags write without escaping**: `raw`, `embed`, `asset` and
   `provenance` write their bytes unchanged in every output. Embedding
   supplied data in HTML unescaped is the author's explicit choice; the
   specification says so beside `embed`.
5. **Escaping does not make output safe for JavaScript, CSS or URLs.** The
   specification states that `json_script` is for `<script>` data and
   `url_encode` for URL components, and that nothing else is offered.

### D10 — The METAIS filters

Every METAIS filter refuses an input or argument of the wrong type with
`template.filter_input` or `template.filter_argument`. Named arguments use
Liquid's `name: value` form.

1. **`json`** — the value as **RFC 8785** JSON: members in canonical order,
   strings escaped as RFC 8785 §3.2.2.2 (`"`, `\` and U+0000 to U+001F only,
   with `\b \t \n \f \r` and lowercase `\u00xx`), numbers as ECMAScript
   `Number::toString` (`3` for 3.0, `1e+21`), no whitespace. `nil` is
   `null`. **`indent: n`** (an integer from 1 to 8) pretty-prints: a line
   feed and `n` more spaces of indentation after `[` or `{` of a non-empty
   array or object and after each `,`; `": "` between name and value; the
   closing bracket on its own line at the enclosing indentation; `[]` and
   `{}` for empty ones; no trailing line feed. Without `indent`, the output
   is byte-identical to ADR-002's canonical form for the same value.
2. **`yaml`** — the value as a YAML 1.2 block document with no `---` and no
   trailing line feed:
   - a mapping writes `name: value` per member in canonical order; a member
     whose value is a non-empty mapping or sequence writes `name:` and its
     block on the following lines, indented two spaces more;
   - a sequence writes `- ` per element; an element that is a non-empty
     mapping or sequence starts on the same line after `- `, and its
     following lines are indented two spaces more than the `-`;
   - empty mapping `{}`, empty sequence `[]`, `nil` `null`, booleans
     `true`/`false`, numbers as in `json`;
   - a string or name is written **plain** when it matches
     `^[A-Za-z_/]([A-Za-z0-9_./ -]*[A-Za-z0-9_./])?$` and is not, ignoring
     ASCII case, one of `true`, `false`, `yes`, `no`, `on`, `off`, `y`, `n`,
     `null`; otherwise **double-quoted**, escaping `"`, `\`, U+0000 to U+001F
     (as `json`), U+007F to U+009F, U+2028, U+2029 and U+FEFF as `\uxxxx`
     (lowercase hex).
   No anchors, aliases, tags, flow collections beyond `{}`/`[]`, or block
   scalars are ever written, so the same value has one text.
3. **`csv`** — an array of objects as RFC 4180 records:
   - positional arguments name the columns, in order; without them the
     columns are the union of the rows' member names in canonical order;
   - **`headers:`** an array of strings, one per named column, replaces the
     header text; **`header: false`** omits the header row;
     **`delimiter:`** a one-character string other than `"`, CR or LF
     (default `,`);
   - a field is quoted, with `"` doubled, when it contains the delimiter,
     `"`, CR or LF; otherwise written bare;
   - a cell is the text of its value (D4 item 4); `nil` is an empty field;
     an array or object cell is `template.filter_input`;
   - a named column a row lacks is `template.undefined_property` (D6: a
     declared column is present as `nil`);
   - every record, including the last, ends with U+000A. CR LF comes from
     ADR-005's line-ending conversion, not from the filter.
4. **`markdown_table`** — an array of objects as a GitHub Flavored Markdown
   table: `| h | h |`, then `| --- | --- |`, then one row per element, each
   line ending with U+000A. Columns, `headers:` and the missing-member rule
   are as in `csv`. In every cell, each `|` becomes `\|` and each line break
   (CR LF, LF or CR) becomes `<br>`; nothing else is escaped, so a cell's
   Markdown stays Markdown. An empty array with columns writes the header and
   delimiter lines; with no columns, the empty string.
5. **`json_script`** — `json` (compact) with `<`, `>`, `&`, U+2028 and U+2029
   further written as `<`, `>`, `&`, ` `, ` `. The
   result cannot close a `<script>` element or open an HTML comment, and is
   safe (D9). It writes the JSON only; the author writes the element
   (`<script type="application/json" id="d">{{ x | json_script }}</script>`).
6. **`slug`** — a URL-safe slug in `[a-z0-9]+(-[a-z0-9]+)*`:
   1. replace `ß`, `ẞ` → `ss`; `æ`, `Æ` → `ae`; `œ`, `Œ` → `oe`; `ø`, `Ø`
      → `o`; `đ`, `Đ`, `ð`, `Ð` → `d`; `þ`, `Þ` → `th`; `ł`, `Ł` → `l`;
      `ı` → `i`; `ħ`, `Ħ` → `h`; `ŧ`, `Ŧ` → `t`; `ŋ`, `Ŋ` → `ng`;
      `ĸ` → `k` (Latin letters with no decomposition);
   2. normalize to NFKD (Unicode 15.0.0);
   3. remove every code point of general category Mn;
   4. lower-case ASCII `A`–`Z`;
   5. replace each maximal run of code points other than `a`–`z` and `0`–`9`
      with one `-`;
   6. remove a leading or trailing `-`.
   `"Café Déjà-vu"` is `cafe-deja-vu`. `nil` is `nil`. An empty result
   (`"日本語"`) is `template.slug_empty`, saying to supply the slug as data.
7. **`safe`** — D9 item 3.
8. **`escape`** and **`escape_once`** are Shopify's, with D9's mapping.
9. **Not in `1.0`**: a `sha256` filter, an indentation filter, and any
   filter reading something other than its arguments.

### D11 — `date` takes RFC 3339 text or Unix seconds and formats in the value's own offset

1. **Accepted inputs**: a string that is an RFC 3339 `full-date`
   (`2026-10-18`, midnight UTC) or `date-time` (`2026-10-18T09:30:00Z`,
   `2026-10-18T09:30:00.25+02:00`; `T` and `Z` in either case; second `60`
   refused); an integer, as seconds since 1970-01-01T00:00:00Z; `nil`, which
   gives `nil`. Anything else — including `"now"` and `"today"` — is
   `template.date_input`; for those two words the finding names `now`.
2. **The value is formatted in the offset it carries.** No conversion
   happens, no zone database is consulted, and nothing of the host's time
   zone is used. `now` is always UTC (D7 item 7).
3. **The format language is Ruby `strftime`'s, closed to this set**: `%a`
   `%A` `%b` `%h` `%B` `%c` `%C` `%d` `%D` `%e` `%F` `%G` `%H` `%I` `%j` `%k`
   `%l` `%L` `%m` `%M` `%n` `%N` `%3N` `%6N` `%9N` `%p` `%P` `%r` `%R` `%s`
   `%S` `%t` `%T` `%u` `%U` `%V` `%w` `%W` `%x` `%X` `%y` `%Y` `%z` `%:z`
   `%Z` `%%`, with the flags `-` (no padding), `_` (space padding), `0` (zero
   padding) and `^` (upper case). Names are English; `%c` is
   `%a %b %e %H:%M:%S %Y`, `%x` is `%m/%d/%y`, `%X` is `%H:%M:%S`, `%r` is
   `%I:%M:%S %p`. **`%Z` is `UTC` at offset zero and otherwise the `%:z`
   form**, where Ruby writes nothing. Any other directive, or an empty
   format, is `template.date_format`.

### D12 — `embed` writes one input's bytes as supplied

1. **Syntax**: `{% embed "<input>" %}` for an input that is not a
   collection, and `{% embed "<input>", index: <expr> %}` for an item of a
   collection input, where `<expr>` evaluates to an integer: the item's
   0-based position in the collection's order (ADR-003). The name is a
   string literal naming a declared input (`template.embed_unknown_input`
   at parse otherwise); `index:` is required for a collection and refused
   otherwise (`template.embed_index`, at parse); an index out of range is
   `template.embed_index` at render.
2. **The bytes are the `raw` form of ADR-003's input record** (D21 item 2),
   written unchanged except that a leading UTF-8 byte-order mark is not
   written. Line endings are kept. The bytes must be valid UTF-8
   (`template.invalid_utf8`). An input not supplied writes nothing.
3. **A raw form that does not exist** — an input or item touched by
   `--set`, `--set-json` or `--inputs-json` — is ADR-003's finding
   (`input.raw_unavailable`), raised before anything renders for every
   `embed` whose name and index are known; with a computed index it is
   raised when the tag is evaluated.
4. **Never escaped** (D9 item 4).

### D13 — `asset` writes an asset's output path, its text, or its base64

1. **Syntax**: `{% asset "<path>" [,] [as: "<mode>"] %}`, the path a string
   literal relative to `assets/`, confined as in D8 item 2 and naming an
   existing entry (`template.asset_not_found` at parse), the mode a string
   literal.
2. **`as: "path"`** (the default) writes the path at which ADR-005 writes
   this asset, relative to the current output's directory, with `/`
   separators (`../assets/site.css`). The asset must be written by this
   render to exactly one path in the same tree as the current output — the
   output directory, or the same zip container — otherwise
   `template.asset_not_written`, `template.asset_ambiguous` or
   `template.asset_tree`.
3. **`as: "contents"`** writes the asset's bytes unchanged, except a leading
   byte-order mark; they must be valid UTF-8 (`template.asset_not_text`).
4. **`as: "base64"`** writes RFC 4648 §4 base64 of the bytes, padded, with
   no line breaks — for a `data:` URI in a single-file HTML output.
5. **Never escaped** (D9 item 4).

### D14 — `provenance` writes one line in the output's comment syntax, recording only what the author names

1. **Syntax**: `{% provenance <part> … [style: "<style>"] %}`. The parts are
   bare words from the closed set `package`, `version`, `package_digest`,
   `inputs`, `tool`, `time`, or `all` for every one of them; at least one is
   required; a word outside the set or repeated is
   `template.provenance_argument`.
2. **Fields**, always in this order whatever the order of the parts:
   `package=<name>`; `version=<package version>`;
   `package_digest=sha256:<hex>`; `input.<name>=<digest>` for every declared
   input, by name in canonical order, `<digest>` being the input record's
   digest (ADR-003 D21 item 4: SHA-256 of the RFC 8785 serialization of the
   value after defaults, which does not include D6's materialized `nil`s) as
   `sha256:<hex>`, or `absent` for an input not supplied;
   `tool=<implementation>/<version>` (`metais/<version>` for this tool, as
   `metais version` prints it); `time=<now>`, written only when the caller
   supplied `--now`. No other field exists.
3. **The line** is `metais-provenance` followed by each field, separated by
   one U+0020. A value character outside `A`–`Z`, `a`–`z`, `0`–`9`, `.`,
   `_`, `+`, `:`, `/`, `@`, `-` is written as `%XX` per UTF-8 byte
   (upper-case hex); in the `html` style, a `-` that follows a `-` is also
   written `%2D`, so no value can end an XML comment.
4. **Styles**, each written with no trailing line feed:

   | Style | Written as |
   |---|---|
   | `html` | `<!-- metais-provenance … -->` |
   | `hash` | `# metais-provenance …` |
   | `slash` | `// metais-provenance …` |
   | `block` | `/* metais-provenance … */` |
   | `dash` | `-- metais-provenance …` |
   | `percent` | `% metais-provenance …` |
   | `semicolon` | `; metais-provenance …` |
   | `json` | the fields as an RFC 8785 object: `package`, `version`, `package_digest`, `tool`, `time` as strings; `inputs` as an object of name to digest or `null` |

5. **The style is chosen** by the `style:` argument when given; otherwise
   from the kind ADR-005 assigns the file being rendered (ADR-005 D7: its
   `kind` field, else its extension table):

   | Kind | Style |
   |---|---|
   | `html`, `xml`, `markdown` | `html` |
   | `yaml`, `hash` | `hash` |
   | `slash` | `slash` |
   | `css` | `block` |
   | `sql` | `dash` |
   | `json` | `json` |

   For kind `csv` or `text`, and wherever there is no file being rendered
   (a path template), the tag without `style:` is
   `template.provenance_style`; for `csv` the finding says the format has no
   comments, for `text` it names `style:`. The `percent` and `semicolon`
   styles are reached only through `style:`. A later minor that adds a kind
   adds its row here.
6. **The tag is placed by the author.** It writes no line feed and knows
   nothing of an XML declaration, a shebang or a front-matter block that
   must precede it.

### D15 — `test` renders with a fixed render identity

1. **When `test` renders an example** (ADR-007), `package.digest` and the
   provenance `package_digest` are `sha256:` followed by 64 `0`, and the
   provenance `tool` is `test`. An example's expected outputs are inside the
   package whose digest they would otherwise have to contain, and a tool
   upgrade must not change them.
2. **`now`** in `test` is the example's render time when ADR-007's example
   layout supplies one, and `nil` otherwise.
3. **Everything else renders exactly as `render` does**, and only `test`
   uses this identity. A `render` — the command, and a conformance vector's
   `render` operation (ADR-011 D8) — records the real package digest, which
   is defined over content and so the same in every conforming
   implementation (ADR-001 D6 item 3), and the tool's real name and version;
   a vector's expected output writes those two as the tokens ADR-011 D10
   item 3 substitutes, so it does not depend on the implementation. A
   vector that proves this decision uses ADR-011's `test` operation.

### D16 — Every template error is a finding with a template path and a line

1. **Every condition this ADR refuses is a finding of error severity** in
   ADR-006's model; nothing is ever written into an output in its place.
   The codes are in ADR-006's `template` family, `template.` followed by
   the names used above: `syntax`, `not_utf8`, `include`, `unknown_filter`,
   `reserved_name`, `partial_name`, `partial_not_found`, `safe_position`,
   `embed_unknown_input`, `asset_not_found`, `provenance_argument`,
   `tag_context` (found at parse); `embed_index` (at parse for its
   presence, at render for its range); `undefined_variable`,
   `undefined_property`, `nil_property`, `object_output`, `comparison`,
   `filter_input`, `filter_argument`, `number_range`, `division_by_zero`,
   `invalid_utf8`, `date_input`, `date_format`, `slug_empty`,
   `asset_not_written`, `asset_ambiguous`, `asset_tree`, `asset_not_text`,
   `provenance_style` (found at render, or at parse when the arguments are
   literals). A limit ADR-009 defines (`partial-depth`, `template-nesting`,
   `template-size`, `render-work` and the rest) is ADR-009 D19's
   `limit.package` when the package's own content breaches it and
   `limit.exceeded` when a render's data or work does, naming the limit.
2. **Location**: `origin` `package`, the template's entry path
   (`templates/…`), and the 1-based line and column of the `{{` or `{%` that
   opens the failing token, the column counted as ADR-006 counts it (inside
   `liquid`, the inner line and its first non-blank character). A render error inside a partial also carries the chain of
   `render` call sites from the output's template, each a path and a line,
   and the output it was rendering (its path, and for a fan-out the 0-based
   index of the item).
3. **No message quotes a value from the inputs.** A finding names the
   variable path, the filter and the type it found, never the data, and an
   engine's own error text is mapped to a code rather than passed through
   (ADR-006).
4. **One pass.** Every template is parsed and every parse-time finding is
   reported, at most one `template.syntax` per template since a parse stops
   there. At render, every output and every fan-out item is attempted, each
   contributes at most its first render error, and if any finding exists
   nothing is written (ADR-005).
5. **Path templates, `when` and `for_each`** are evaluated by this
   environment with the same strictness and D7 item 10's bindings for their
   context. ADR-005 D4 item 1's restricted mode applies
   to a path; `render`, `embed`, `asset` and `provenance` there are
   `template.tag_context`. A finding in these is located at the
   declaration's manifest JSON Pointer and the position within the
   expression (ADR-005 D3 item 7).

## Interfaces with sibling ADRs

**ADR-001.** I rely on: templates and partials being entries under
`templates/` and assets under `assets/`, read in place; the package digest
(ADR-001 D6 items 2 and 5) as `sha256:<hex>`, equal for a source directory
and its package, and equal in every conforming implementation. I promise: no
template reads any other entry.

**ADR-002.** I rely on: the canonical members `metais`, `name`, `version`,
`title`, `description`, `guide`, `authors`, `license`, `homepage`, which
`package` exposes unchanged; RFC 8785 as the canonical JSON, which `json`
reproduces exactly; I-JSON numbers. I promise what ADR-002 D7 item 4 asks: an
undefined variable, an undefined `package` member, an unknown filter and an
unknown tag are each refused (D1 item 6, D6), and the digest member is
`package.digest`.

**ADR-003.** I rely on: a data model of I-JSON values (D4's integer rule
classifies its numbers); defaults applied before rendering; the combined
schema from which D6 item 2 computes declared properties, D6 items 2 to 4
being the statement of how absence reaches a template, which ADR-003 D20
item 1 cites; the input record of ADR-003 D21 — `value` (bound to
`inputs.<name>`), `raw` (read by `embed`, per item for a collection, with
ADR-003's `input.raw_unavailable` when it is missing), `source` (exposed as
`sources`), and `digest` (recorded by provenance); ADR-003 D8 item 5's
integer rule, which D4 item 2 restates; input names usable as
`inputs.<name>` (or `inputs["<name>"]`); a collection's order (ADR-003 D15),
which `embed … index:` and `sources` index by; the `body` member of a
Markdown item. I promise: object iteration in canonical (RFC 8785, UTF-16
code unit) order (D4 item 5), the one order for object members everywhere;
ADR-003's other orderings — collection items by the UTF-8 bytes of their
NFC names — order names, not members, and are unchanged.

**ADR-005.** I rely on: an output's `escape` being `html` or `none`
(default `none`); each rendered file's kind (ADR-005 D7), from which D14
item 5 maps a provenance style; the rendered file's path, and for a
container entry its path inside the container, handed over as
`output.path`; one render per `for_each` element with the array handed over
for `fan_out`; the asset placement map (ADR-005 D5 item 6), to resolve
`asset … as: "path"`; per-item `when` and path evaluation (ADR-005 D3 item
5); the `_` prefix that keeps a partial out of a mirror (ADR-005 D6 item 2);
a render error stopping only its output, and nothing written when any
finding exists. I promise `liquid.ParseCondition` and `liquid.ParseValue`
(ADR-010 D4 item 3) for ADR-005 D3 item 8, D4 item 1's restricted path mode,
with D16's findings, and an `escape` mapping (`&amp;` `&lt;` `&gt;`
`&quot;` `&#39;`) that is also correct XML escaping. ADR-005 adopts this
ADR's asset syntax (`{% asset "<path>" %}`, path being the default mode, or
`as: "path"`) and D7 item 10's table of which globals each of its
expressions sees.

**ADR-006.** I rely on: findings with a stable code, error severity, a
location of `origin`, `file`, line and column, and room for related
locations (D16 item 2's call chain) and for the output and item; the rule
that no template message carries an input value (D16 item 3). I promise D16's
codes in the `template` family, spelled as ADR-006 D3 item 1 requires; a
limit's breach is ADR-009 D19's `limit.package` or `limit.exceeded`.

**ADR-007.** I rely on: `check` parsing every template under `templates/`
and reporting D6 item 8's parse-time findings; the cross-reference resolving
names over the engine's own syntax tree (ADR-010 D4), with "declared" as D6
item 2 states plus the open names D6 item 2's last sentence describes, so
the static check never rejects a read the render accepts; `test` rendering
under D15; examples optionally carrying a render time (`example.yaml`'s
`now`, ADR-007 D5 item 1). I promise: `render`, `embed` and `asset` names
are always literals, and the bindings are D7's closed set (D7 item 10 says
where each is bound), so every name a template uses is statically
classifiable except a computed index (`a[expr]`).

**ADR-008.** I rely on: `--now` arriving as a checked RFC 3339 `date-time`
with an offset, or not at all; `metais version` as the tool version
provenance records.

**ADR-009.** I rely on: the limits `partial-depth`, `template-nesting`,
`template-size`, `output-size`, `value-size`, `array-size`, `assign-bytes`,
`loop-iterations` and `render-work`, each with its default and ADR-009
D19's codes (`limit.package`, `limit.exceeded`); the charge table for every filter in D10 and the standard
catalog (ADR-009 D11 item 2); the sentinel proof of D7 item 1. I promise:
`date` reads no clock and no zone — `"now"` and `"today"` are refused
whether or not `--now` is given, and the render time is the `now` global
(D7 item 7, D11 item 1) — no filter or tag reads the host, and the binding
set is D7's for every caller.

**ADR-010.** I rely on: METAIS's own engine, `internal/liquid` (ADR-010 D4),
implementing exactly this dialect — `osteele/liquid` v1.9.2 fails it in the
ways *Context* lists, several of them in its expression scanner, parser and
renderer, which no caller can reach; a parser that stops at a template's
first syntax error (D16 item 4); Unicode 15.0.0 tables for D1 item 4 and D10
item 6 (Go 1.26's `unicode` and `golang.org/x/text`, ADR-010 D1 item 3).

**ADR-011.** I rely on: `test` vectors rendering under D15's identity and
`render` vectors under the real one with ADR-011 D10 item 3's tokens; the
delegated liquid-spec run configured with `strict2` and this ADR's
exclusions (ADR-011 D17); a spec chapter carrying the grammar, the tag
table, the full standard filter catalog and D10–D14 exhaustively.

## Conformance

Vectors are needed for each observable rule; ADR-011 owns their format.

- **Parsing**: each tag of D2 accepted; `include`, `self` as a keyword, an
  unknown tag, an unknown filter in a dead branch, `reversed` after a named
  parameter, a non-literal `render` name, a reserved-name `assign`, `safe`
  not last, U+0000 in a template — each refused with its code and line.
- **Whitespace**: each control marker; a tag alone on a line; the blank-body
  rule for `if`, `unless`, `case` and `for`, and its absence for
  `tablerow`; BOM and CR LF preserved.
- **Values**: integer and float text at the boundaries 1e-4 and 1e15;
  `-0.0`; integral data numbers as integers; `divided_by: 4.0`; floor
  division and modulo with negatives; `0.1 | plus: 0.2`; `round` of
  negative halves; integer overflow; object output refused; array output
  concatenated; canonical member order including a name outside the Basic
  Multilingual Plane.
- **Truthiness**: `0`, `""`, `[]`, `{}` truthy; each `empty` and `blank`
  case, including `nil == empty` false and `"  " == blank` true.
- **Strictness**: declared-absent `nil`, undeclared absent refused,
  supplied-but-undeclared readable, property of `nil` refused, `and`
  guarding it, out-of-range index `nil`, `package.nope` refused, an absent
  optional input `nil`.
- **Bindings**: each global's shape; `item` and `fan_out` only in a fan-out,
  with `previous`/`next` at both ends; `output.path`/`root` at depths 0 to 2
  and inside a container; `now` normalization; shadowing by a loop variable
  and a `render … as` alias.
- **Partials**: resolution, isolation, globals visible, escaping inherited,
  recursion to the limit and beyond.
- **Escaping**: each character; `safe`, `escape`, `escape_once`,
  `json_script`, `capture` not double-escaped; tags unescaped.
- **Each METAIS filter**: every rule in D10 and D11, including `yaml`'s
  plain-versus-quoted boundary, `csv` quoting and delimiters,
  `markdown_table` pipes and line breaks, `json_script` characters, `slug`'s
  table, NFKD and empty case, every `strftime` directive and flag, and each
  refused date input.
- **`embed`, `asset`, `provenance`**: each mode and refusal, including a
  collection item by index and a missing raw form; every style and every
  kind's mapping, with a `kind` override; value percent-encoding; `--` in
  `html`; field order; `time` absent without `--now`; D15's identity.
- **`sources`**: each `origin`; a collection's `name`, `stem` and `form`;
  `nil` for a command-line item; the per-item `when` idiom.
- **Errors**: D16's location, call chain and one-pass behaviour; `and`/`or`
  grouping from the right (`true or false and false` is true).
- **Shopify standard filters and tags**: a vector set per filter and tag,
  derived from the gem's own tests at `v5.14.0` and written in this
  project's words.

## Alternatives rejected

### Shopify's default lax mode

An undefined name rendering as nothing turns a typo into a silently empty
report, and an unknown filter passing through would make a `1.0` package
depend on the absence of a filter a later minor adds. Strictness is what
lets ADR-007 call an unknown field an error and ADR-002 grow the format.

### Treating a property of `nil` as `nil`

It would let `{{ inputs.vendor.name | default: "—" }}` work unguarded, but
neither the reference (`strict_variables`) nor the Go library does it, and
it hides the very case the sample preview's "none" mode exists to expose: a
template that only works when an optional object is supplied.

### Preserving the supplier's member order

Liquid on Ruby iterates hashes in insertion order, but JSON objects are
unordered, programs serialize maps in arbitrary order, and order would have
to survive defaults, `--set` and merging. Canonical order makes the same
value render the same bytes however it was supplied; tabular filters take
their columns from the template, where the author knows them.

### ECMAScript number text for `{{ }}`

`3.0` rendering as `3` would read better in some reports, but it departs
from the reference, and with D4 item 2 data numbers that are integral
already render without a fraction. `json` uses ECMAScript text because RFC
8785 does.

### Converting every date to UTC

It is the simplest rule, but it silently discards the offset a supplier
wrote, so a report could not show a local time it was given. Keeping the
offset reads nothing of the host and needs no zone database.

### A comment-syntax table of this ADR's own

ADR-005 already gives every rendered file a kind, from an extension table
with a `kind` override, for its own output validation. A second table keyed
on extensions would drift from it; mapping kinds to styles keeps one answer
to "what format is this file", and the tag's `style:` covers a `text` file
whose format has a comment syntax no kind names.

### Identity carried in the item value

Putting a collection item's file name into the item (as a `_file` member,
say) would let `item` carry it through any filter, but it would collide with
the supplier's own fields, as `body` already reserves one, and would change
the value whose digest provenance records. ADR-003 keeps values free of
identity, so `sources` is indexed by position.

### Dynamic partial and asset names

They would make partial existence, asset existence and the cross-reference
undecidable without data. Shopify itself requires a quoted name for
`render`.

### A `raw` filter as the escape opt-out

`raw` is already a tag with a different meaning. `safe` is the established
name in the Go library and in other template languages.

### Ruby's full case mapping and `inspect`

Full case mapping changes string lengths and depends on special-casing
tables most implementations do not carry; `inspect`'s format changed in Ruby
3.4. Neither is something a second implementation should have to reproduce.

## Consequences

- A package renders the same bytes in any conforming implementation: every
  place Liquid implementations disagree — truthiness edges, whitespace,
  number text, arithmetic, dates, escaping, loop modifiers — is fixed here
  and proved by a vector.
- Templates are strict. A typo, an unguarded optional object or an unknown
  filter is a finding rather than an empty cell, and the authoring
  documentation must teach `{% if a and a.b %}` early.
- The Go library cannot be used (*Context*), and forking it would copy
  another engine's code, which AGENTS.md forbids; ADR-010 D4 writes
  METAIS's own engine of this dialect.
- METAIS diverges from the reference deliberately in a few named places
  (object output, simple case mapping, `date` inputs and `%Z`, `self`), each
  listed in the specification's dialect chapter.
- Provenance is machine-readable in one fixed line, and is absent from every
  output whose author did not ask for it.

## What this does not decide

- Output declaration, path templating syntax, `for_each`/`when` placement,
  the mirror shorthand and how template file names become outputs (ADR-005).
- The finding record's fields and the JSON report (ADR-006).
- The cross-reference algorithm and the OOXML split-run finding (ADR-007).
- Every limit's value (ADR-009).
- The engine, its fork or its replacement (ADR-010).
- Identity of a collection item through a filtered or reordered
  `for_each`, a `sha256` filter, an indentation filter, alignment in `markdown_table`, and a locale for `date` names: a
  later minor may add any of them.

## Proposed rulings

1. **Pin gem 5.14.0 with `strict2`, `strict_variables`, `strict_filters`
   and `render!`.** Lean: yes — the newest release, and the strictest
   grammar; alternative: 5.13.0 with `strict`.
2. **Exclude `self` and every theme-platform filter.** Lean: yes — `self`
   makes variable lookup dynamic; alternative: include `self`.
3. **Include `liquid`, `echo`, `#`, `doc`, `ifchanged`, `increment`,
   `decrement`, `cycle` and `tablerow`.** Lean: yes, all core; alternative:
   drop `tablerow` for its fixed HTML.
4. **Pin the blank-body rule.** Lean: yes — it is the reference's bytes.
5. **`reversed` only right after the collection; `offset`, `limit`,
   `reversed` order.** Lean: yes.
6. **Object output refused; arrays concatenate.** Lean: yes.
7. **Integral data numbers are integers; literals by spelling; Ruby float
   text; exact decimal arithmetic; ±(2⁵³ − 1).** Lean: yes; alternative:
   ECMAScript text everywhere.
8. **Canonical (RFC 8785) member order everywhere.** Lean: yes;
   alternative: source order.
9. **Declared-absent is `nil`, undeclared is an error, a property of `nil`
   is an error.** Lean: yes; alternative: `nil` propagation.
10. **Unknown filters refused at parse, in dead branches too.** Lean: yes.
11. **Globals `now`, `output {path, root}`, `fan_out` with `previous` and
    `next`.** Lean: yes; alternative: `render.now`, no neighbours.
12. **Globals visible in partials.** Lean: yes.
13. **Partial names are literal full paths under `templates/`.** Lean: yes.
14. **Template-scoped bindings may not take a global's name; block-scoped
    ones may shadow.** Lean: yes.
15. **Autoescape covers `{{ }}`, `echo`, `cycle`; `safe` last only; escaping
    filters and `capture` are safe.** Lean: yes.
16. **Simple case mapping, Unicode 15.0.0.** Lean: yes; alternative: full.
17. **`date`: RFC 3339 or Unix seconds; the value's own offset; `%Z` as
    `UTC` or the offset; unknown directives refused.** Lean: yes;
    alternative: convert to UTC.
18. **`json` is RFC 8785, with `indent:`.** Lean: yes.
19. **`yaml` is the restricted emitter of D10 item 2.** Lean: yes.
20. **`csv`: LF records, header on, `headers:`/`header:`/`delimiter:`.**
    Lean: yes.
21. **`markdown_table`: `\|`, `<br>`, `headers:`.** Lean: yes.
22. **`slug`: the table, NFKD, ASCII, empty refused.** Lean: yes;
    alternative: keep non-Latin letters.
23. **`json_script` escapes `< > &` U+2028 U+2029 and writes JSON only.**
    Lean: yes.
24. **No `sha256` filter in `1.0`.** Lean: no filter; alternative: add one
    for cache-busting and stable ids.
25. **`embed`: an input, or a collection item by `index:`; BOM dropped;
    never escaped.** Lean: yes; alternative: no collection items in `1.0`.
26. **`asset` modes `path` (relative to the current output), `contents`,
    `base64`.** Lean: yes; alternative: also `data_uri` with a media-type
    table.
27. **Provenance: one line, closed fields, styles mapped from ADR-005's
    kinds, `style:`, and a `json` style used automatically for `json`
    files.** Lean: yes.
28. **An input's provenance digest is ADR-003's (RFC 8785 of the value
    after defaults).** Lean: yes; alternative: of its raw bytes.
29. **`test` renders with D15's fixed identity; `render`, vectors included,
    records the real package digest and tool, which ADR-011 D10 item 3's
    tokens absorb.** Lean: yes. Why: an example's expected outputs cannot
    contain the digest of the package that holds them, while a vector's
    package digest is the same in every implementation. Alternative: the
    fixed identity for vectors too, which needs a mode a black-box runner
    cannot set.
30. **Each output and item reports its first render error; all are
    attempted.** Lean: yes.
31. **Template bytes are kept as written; U+0000 refused.** Lean: yes.
32. **A `sources` global exposes ADR-003's `source` records, indexed by
    position, and `fan_out` counts before `when`.** Lean: yes;
    alternative: no item identity in `1.0`.
33. **One table says where each global is bound (D7 item 10)**: `now` and
    `sources` in every manifest expression; `item` and `fan_out` only per
    element of a fan-out; `output` only in template bodies, never while a
    path is computed. Lean: yes. Alternative: manifest expressions see only
    `inputs`, `package` and `item`, which forbids a path named after a
    collection item's file stem.

## References

- `liquid` gem 5.14.0 on RubyGems: https://rubygems.org/gems/liquid/versions/5.14.0
- Shopify/liquid `lib/liquid/condition.rb` and `lib/liquid/tags/if.rb` at
  `v5.14.0` (`and`/`or` chained so they group from the right, the left
  operand evaluated first and short-circuited; read 2026-10-18):
  https://github.com/Shopify/liquid/blob/v5.14.0/lib/liquid/condition.rb
- Shopify/liquid at `v5.14.0`: https://github.com/Shopify/liquid/tree/v5.14.0 —
  `lib/liquid/tags.rb`, `lib/liquid/standardfilters.rb`,
  `lib/liquid/condition.rb`, `lib/liquid/expression.rb`,
  `lib/liquid/block_body.rb`, `lib/liquid/block.rb`,
  `lib/liquid/variable.rb`, `lib/liquid/variable_lookup.rb`,
  `lib/liquid/utils.rb`, `lib/liquid/context.rb`,
  `lib/liquid/parser_switching.rb`, `lib/liquid/tags/render.rb`,
  `lib/liquid/tags/table_row.rb`, `lib/liquid/tags/comment.rb`,
  `lib/liquid/tags/doc.rb`, `History.md`
- Liquid documentation: https://shopify.github.io/liquid/basics/truthy-and-falsy/,
  https://shopify.github.io/liquid/basics/whitespace/,
  https://shopify.github.io/liquid/basics/operators/,
  https://shopify.github.io/liquid/tags/template/ (`include` deprecated),
  https://shopify.github.io/liquid/filters/date/ (`"now"`, `Time.parse`)
- osteele/liquid at `v1.9.2`: https://github.com/osteele/liquid/tree/v1.9.2 —
  `README.md`, `SECURITY.md`, `CHANGELOG.md`, `docs/loop-semantics.md`,
  `tags/standard_tags.go`, `tags/render_tag.go`, `tags/template_path.go`,
  `tags/iteration_tags.go`, `filters/standard_filters.go`,
  `render/render.go`, `render/autoescape.go`, `expressions/builders.go`,
  `expressions/context.go`, `values/predicates.go`, `values/parsedate.go`
- Ruby `strftime` formats: https://docs.ruby-lang.org/en/3.4/strftime_formatting_rdoc.html
- Ruby `Float#to_s`: https://docs.ruby-lang.org/en/3.4/Float.html (behaviour
  measured with Ruby 2.6.10, as in *Context*)
- RFC 8785, JSON Canonicalization Scheme: https://www.rfc-editor.org/rfc/rfc8785
- RFC 7493, I-JSON: https://www.rfc-editor.org/rfc/rfc7493
- RFC 3339, date and time: https://www.rfc-editor.org/rfc/rfc3339
- RFC 4180, CSV: https://www.rfc-editor.org/rfc/rfc4180
- RFC 4648, base64: https://www.rfc-editor.org/rfc/rfc4648
- ECMA-262, `Number::toString`: https://tc39.es/ecma262/
- YAML 1.2.2 (printable characters, double-quoted escapes): https://yaml.org/spec/1.2.2/
- GitHub Flavored Markdown, tables and pipe escaping: https://github.github.com/gfm/#tables-extension-
- HTML, restrictions on `<script>` contents: https://html.spec.whatwg.org/multipage/scripting.html#restrictions-for-contents-of-script-elements
- Django `json_script` escapes (`<`, `>`, `&`): https://github.com/django/django/blob/main/django/utils/html.py
- XML 1.0, comments (no `--`): https://www.w3.org/TR/xml/#sec-comments
- UAX #15, normalization forms: https://www.unicode.org/reports/tr15/
- Go `unicode.Version` (15.0.0): https://pkg.go.dev/unicode#pkg-constants

## Implementation log

- **2026-10-19, PLAN-2026-10-metais-v1 W3-5.** D10 item 2's writer
  (`yamlx.Write`, Wave 2) escapes U+FFFE and U+FFFF, which a YAML stream
  may not hold raw, and writes an integral number of magnitude above
  2^53−1 and below 1e21 in exponent notation (`9.007199254740992e+15`),
  not the integer literal the `json` filter writes, which ADR-003 D8 item
  5 refuses for an input. Every other number is written as `json` writes
  it.

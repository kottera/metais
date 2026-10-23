# ADR-004: The template environment — Shopify Liquid 5.14.0 parsed strictly, a closed set of bindings, strict on every name, and a METAIS filter and tag catalog defined to the byte

**Status:** Accepted 2026-10-18 by Ed Funnekotter; its proposed rulings are ratified in [RULINGS-v1](RULINGS-v1.md).
[Amendment A1](#amendments) (`slug` reads its letter table after NFKD) and
Amendment A2 (value-formatting filters: `number`, `amount` and `duration`)
are Proposed; the passages they change are marked *(A1)* and *(A2)*.
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
   further written as `\u003c`, `\u003e`, `\u0026`, `\u2028`, `\u2029`. The
   result cannot close a `<script>` element or open an HTML comment, and is
   safe (D9). It writes the JSON only; the author writes the element
   (`<script type="application/json" id="d">{{ x | json_script }}</script>`).
6. **`slug`** — a URL-safe slug in `[a-z0-9]+(-[a-z0-9]+)*` *(A1)*:
   1. normalize to NFKD (Unicode 15.0.0);
   2. remove every code point of general category Mn;
   3. replace `ß`, `ẞ` → `ss`; `æ`, `Æ` → `ae`; `œ`, `Œ` → `oe`; `ø`, `Ø`
      → `o`; `đ`, `Đ`, `ð`, `Ð` → `d`; `þ`, `Þ` → `th`; `ł`, `Ł` → `l`;
      `ı` → `i`; `ħ`, `Ħ` → `h`; `ŧ`, `Ŧ` → `t`; `ŋ`, `Ŋ` → `ng`;
      `ĸ` → `k` (Latin letters with no decomposition);
   4. lower-case ASCII `A`–`Z`;
   5. replace each maximal run of code points other than `a`–`z` and `0`–`9`
      with one `-`;
   6. remove a leading or trailing `-`.
   `"Café Déjà-vu"` is `cafe-deja-vu`. `nil` is `nil`. An empty result
   (`"日本語"`) is `template.slug_empty`, saying to supply the slug as data.
7. **`safe`** — D9 item 3.
8. **`escape`** and **`escape_once`** are Shopify's, with D9's mapping.
9. **Not in `1.0`**: a `sha256` filter, an indentation filter, a filter
   reading a locale or a shop's settings, such as Shopify's `money`, and
   any filter reading something other than its arguments. *(A2)*
10. **`number`** — a number or a decimal text, rounded to `decimals:`
    half away from zero, with the point and group separators its keywords
    name. *(A2)*
11. **`amount`** — `number` with an ISO 4217 code before or after it, its
    decimals from the code by Appendix C unless `decimals:` gives them.
    *(A2)*
12. **`duration`** — seconds, an ISO 8601 duration, or fields named by
    `fields:`, written by `%d`, `%H`, `%M` and `%S`. *(A2)*

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
   `provenance_style`, `number_input`, `amount_currency`, `duration_input`,
   `duration_format` *(A2)* (found at render, or at parse when the
   arguments are literals). A limit ADR-009 defines (`partial-depth`, `template-nesting`,
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
- A locale, a preset, a rounding mode, a currency symbol or a minus sign
  for `number` and `amount`; a negative or fractional duration; and an
  offset for `date` *(A2)*: a later minor may add each as a keyword.

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

## Amendments

### A1 — `slug` reads its letter table after NFKD

**Status:** Proposed 2026-10-23. Ed Funnekotter ruled on 2026-10-22, at the
ratification of PLAN-2026-10-metais-v1 (T-A4), that the defect W5-1
observed gets its own fix before `spec-1.0` freezes the vectors (R2-170).
**Amends:** D10 item 6 (the order of its steps), marked *(A1)* in place.

1. **The table is read after decomposition.** The steps are NFKD, then
   every Mn removed, then the letter table, then the ASCII steps as
   before. Read first, the table never sees a letter that reaches one of
   its entries only through decomposition, and the letter was dropped:
   U+01FE `Ǿ` is `Ø` and a combining acute under NFKD, so `"Ǿrsted"` gave
   `rsted`. It now gives `orsted`, as do `ǿ`, `Ǽ`, `ǽ`, `Ǣ` and `ǣ` the
   `o` and `ae` of their bases.
2. **No entry of the table changes meaning.** Every entry has no
   decomposition, canonical or compatibility, in Unicode 15.0.0, so NFKD
   leaves it as it was and the table spells it as before. A table letter
   is a starter, and the marks that follow one are removed in either
   order. The only slugs that change are those of text holding a letter
   that decomposes to a table letter, which was dropped and is now
   spelled.
3. **Proof.** The vector `flt/slug-decomposed-letters` fails on the
   order D10 gave and passes on this one; a unit test of
   `internal/template/filters` holds that no table letter decomposes.

### A2 — Value-formatting filters: `number`, `amount` and `duration`

**Status:** Proposed 2026-10-23. Ed Funnekotter ruled on 2026-10-23 that
value-formatting filters ship in `1.0`, locale-free and with explicit
arguments, and accepted rulings 1 to 12 below as proposed, freezing ruling
5's table at the current edition of the official list.
**Amends:** D10 (item 9 widened; items 10 to 12 added), D16's codes, and
"What this does not decide", each marked *(A2)* in place. Chapter 8
§8.1.10 to §8.1.14 state it as MTS-FLT-015 to MTS-FLT-023, Appendix C
holds the currency table, chapter 3's MTS-MAN-018 names the two literal
refusals in a manifest expression, chapter 7's MTS-TPL-063 charges the
filters, and chapter 13 §13.9 shows them in use.

**Context.** An external test of `examples/dora-incident-report` found
three values a reader misreads, each written as the data holds it: an
amount, `48920000`, with no currency, though an input names it; a
timestamp left as raw RFC 3339 text; and a duration, `00:07:25`, which
readers took as hours, minutes and seconds where the field means days,
hours and minutes. `date` already writes the second for a reader; nothing
wrote the first or the third. An agent-written invoice package in the UX
runner's records (`internal/tools/ux`) wrote `{{ amount }} {{ currency }}`
and printed `12.5 EUR`. No Liquid dialect has these filters, and the ones
agents know best for money, Shopify storefront's `money`,
`money_with_currency` and `money_without_currency`, take cents and read the
shop's settings.

**Rulings of 2026-10-23,** numbered as ruled:

1. **Three filters, `number`, `amount` and `duration`,** not one filter
   with modes: an author looks for an amount or a duration by that name,
   and each filter's refusals stay its own, as `date`'s are.
2. **`decimals:` is a keyword** of `number` and `amount` alike, since
   `amount`'s one positional argument is the code. Every argument a later
   minor adds is a keyword, which `1.0` refuses (MTS-FLT-004), so the
   addition is a construct an earlier minor refuses (chapter 4 §4.1.2).
3. **No group separator by default,** and `.` as the point: grouping is
   always written in the template, and the point is the data's own
   spelling. U+202F, the ISO 80000 thin space, was the alternative; it is
   not chosen, for the fonts and CSV readers that do not take it.
4. **No presets in `1.0`.** A preset is a locale by another name; a later
   `locale:` or `preset:` keyword is additive, and an explicit
   `group:` or `point:` will win over it.
5. **A frozen ISO 4217 table, with `decimals:` overriding it.** DORA's
   reporting currency is an input, so the author cannot know at authoring
   time whether it is `JPY` (0), `EUR` (2) or `BHD` (3). The table is ISO
   4217's list one as SIX Financial Information AG published it on
   2026-09-17 (Appendix C, 178 codes). No version of the major changes an
   entry's minor units; a later minor may add a code, which `1.0` refuses
   without `decimals:`, and `decimals:` is the escape for a code the table
   lacks.
6. **The code before the number, separated by one U+0020** (`EUR 48,920,000.00`),
   the order of the EU's Interinstitutional Style Guide, with
   `code: "after"` for `48,920,000.00 EUR`; `-` belongs to the number
   (`EUR -5.00`). No symbol, ever: `$` names more than one currency.
7. **Half away from zero,** so `x | round: 2` (MTS-TPL-018) and
   `x | number: decimals: 2` never disagree; DORA's own glossary rounds
   percentages half up, which is the same for the values it rounds. A
   later minor may add a `rounding:` keyword.
8. **Strings are read only as a strict decimal text,**
   `-?(0|[1-9][0-9]*)(\.[0-9]+)?`, exactly and at any length: ADR-003 D8
   item 5 tells a supplier to give a number beyond binary64 as a string,
   and this is where that string is formatted. Ruby's `to_i` coercion,
   which the arithmetic filters keep (D4 item 3), is never applied: a
   loose reading could not be withdrawn later, a strict one can widen.
9. **`duration` reads a string of fields separated by `:` only when
   `fields:` names them** (`"d:h:m:s"`, `"d:h:m"`, `"h:m:s"`, `"h:m"`,
   `"m:s"`), beside whole seconds and the ISO 8601 durations of `W`, `D`,
   `H`, `M` and `S`. DORA's `DD:HH:MM` is the regulator's own field type,
   and naming the fields makes the misreading that started this amendment
   impossible to write.
10. **A directive format,** `%d`, `%H`, `%M` and `%S` with `date`'s flags
    `-`, `_` and `0`, rather than named styles, which would need English
    unit names. The largest unit named takes the whole count, each smaller
    one what is left, and what lies below the smallest is dropped, never
    rounded up, as a clock does.
11. **No offset for `date` in `1.0`.** A fixed offset is wrong for half of
    every year in a zone with daylight saving time, and a zone name needs
    the time zone database D11 item 2 forbids. A later minor may add a
    keyword.
12. **Four new codes,** `template.number_input`, `template.amount_currency`,
    `template.duration_input` and `template.duration_format`, as `date` has
    `template.date_input` and `template.date_format`: `template.filter_input`
    naming the kind `string` would mislead where strings are read.

**Decision.**

1. **One reading of a number** (MTS-FLT-015 to MTS-FLT-017), shared by
   `number` and `amount`. A number's value is the shortest decimal that
   reads back as its binary64, the decimal D4 item 3's arithmetic computes
   on, never the binary value's exact expansion and never the literal the
   supplier wrote (JSON, YAML and CSV supply one value, ADR-003 D8 item
   5). A decimal text's value is the decimal it spells. The value is
   rounded to `decimals:` (0 to 20) half away from zero, or written with
   its own digits when `decimals:` is absent; it is written positionally,
   never with an exponent; `-` only when the value written is below zero,
   so `-0.004` at two decimals is `0.00`. Nil gives nil.
2. **`number`** (MTS-FLT-018) takes no positional argument and the keywords
   `decimals:`, `group:` and `point:`.
3. **`amount`** (MTS-FLT-019, MTS-FLT-020) takes the code and the keywords
   `decimals:`, `group:`, `point:` and `code:`. A code that is not three
   ASCII capital letters is `template.amount_currency`, reason
   `malformed`; one Appendix C lacks or gives no minor unit, when
   `decimals:` is absent, reason `no_minor_unit`. A literal code is judged
   at parse, in a template and in a manifest expression (MTS-MAN-018).
   The finding never holds the code (D16 item 3).
4. **`duration`** (MTS-FLT-021, MTS-FLT-022) takes its format and the
   keyword `fields:`; its input refusals are `template.duration_input`
   with a `reason` (`kind`, `calendar`, `range`, `form`), and its format's
   `template.duration_format` with the `directive`, a literal format at
   parse, as `date`'s is.
5. **The `money` names** (MTS-FLT-023). `template.unknown_filter` gains an
   optional parameter, `alternative`, which a processor gives as `amount`
   for `money`, `money_with_currency` and `money_without_currency`. It is
   not `suggestion`: chapter 12's MTS-FND-011 confines a `suggestion` to a
   name the package declares, and a near-match by spelling would never
   reach `amount` from `money` in any case. It names a filter of the
   dialect and holds nothing of the inputs.
6. **Help.** `metais help templates` (ADR-008 A6) shows `number`, `amount`
   and `duration` example first, beside `date`, each with the example
   chapter 8 gives. Its entry for `amount` says, in a line of its own,
   that there is no `money` filter and that `amount` takes the amount in
   the currency's major unit, not in cents. Its entry for `duration` says
   that the directives mirror `date`'s. The help is held to its source by
   its golden, as A6 holds it.
7. **Charges** (MTS-TPL-063). Each of the three charges 16 units, the
   shallow sizes of its input and arguments, and then the length of the
   string it builds, or the shallow size of nil; the work is linear in a
   length already charged, as `date`'s is.
8. **Authoring** (chapter 13 §13.9) shows the four filters on one report's
   values and says what an author does: take the code from the data, name
   the separators, give long values as decimal text, name a duration's
   fields.

**What a later minor can add.** A keyword on any of the three (`locale:`,
`preset:`, `rounding:`, `minus:`, a grouping other than threes, `symbol:`),
a code in Appendix C, an `offset:` for `date`, and negative or fractional
durations: each is a construct `1.0` refuses. What no minor of this major
changes: a default, a table entry's minor units, and the reading of a
string.

**Implementation.** The filters live in `internal/template/filters`
beside `slug`, registered by `All()`, so the loader's dialect knows them
and a manifest expression applies them (D16 item 5). The currency table is
a Go table held equal to Appendix C by a test that reads the appendix;
it is never generated at build time. The rounding is computed on
`math/big` from `strconv.FormatFloat(f, 'g', -1, 64)`, as `round` computes
it, and never with `FormatFloat(f, 'f', n, 64)`, which rounds the binary
value and writes `2.67` for 2.675. `internal/template/codes.go` maps the
engine's codes to the four; `internal/cli`'s declaration of the codes a
manifest expression can produce gains `template.amount_currency` and
`template.duration_format`.

**Proof.** Vectors under `flt/`: the number reading and its rounding
(2.675, 1.005, 0.125, −2.5, −0.004, 1e21), decimal texts, separators and
their refusals; `amount` across `EUR`, `JPY`, `BHD`, `CLF` and `XAU`, with
`decimals:` and `code:`, and its refusals; `duration` over seconds, ISO
8601 durations and each `fields:` form, its format and its refusals; the
`money` alternative; and `tpl/charge-number` at its bound. The examples
`dora-incident-report`, `status-widget` and
`accessibility-conformance-report` write their amounts, counts, dates and
durations through the filters.

## Implementation log

This log is this section together with the entry files in
`docs/adr/logs/ADR-004/`. Each work item writes its entry as a file of its
own there, named as AGENTS.md's "Writing documentation" says. The entries
below are the ones written in this file before that rule, and stay where
they were written.

- **2026-10-19, PLAN-2026-10-metais-v1 W3-5.** D10 item 2's writer
  (`yamlx.Write`, Wave 2) escapes U+FFFE and U+FFFF, which a YAML stream
  may not hold raw, and writes an integral number of magnitude above
  2^53−1 and below 1e21 in exponent notation (`9.007199254740992e+15`),
  not the integer literal the `json` filter writes, which ADR-003 D8 item
  5 refuses for an input. Every other number is written as `json` writes
  it.
- **2026-10-19, PLAN-2026-10-metais-v1 W4-3.** `internal/template` is this
  ADR's environment over the engine, reached through `metais render`: the
  dialect is `Core()` with D7 item 8's names reserved; D6 items 2 and 3's
  declared-absent `nil`s are materialized by walking the combined schema
  through the applicators D6 item 2 lists, with a member's location its
  `properties` entry, each matching `patternProperties` entry and
  otherwise `additionalProperties`; D7's globals are built once per
  command, `output` per output; partials come only from the package's
  `templates/` entries (D8 item 7). A render tag naming no template is
  `template.partial_not_found` at parse, and in a template whose parse
  failed for another reason its literal render tags are read from the
  constructs before the error that stopped the parse, so the finding comes
  in the same pass (D16 item 4). The engine's `argument` code — a for or
  tablerow argument, or a range bound, of the wrong kind — is reported as
  `template.tag_argument`, the one code D16 does not name: none of D16's
  codes is about a tag's argument, and `template.filter_argument` names a
  filter. Every template finding met at render carries `output_path`, and
  a partial's carries its `render` call sites as related locations noted
  `rendered from` (D16 item 2). D9's escaping, the METAIS filters and
  tags, `date`, and `item` and `fan_out`'s shape are Wave 5's; chapter 7
  states that `item` and `fan_out` are unbound outside a fan-out, which
  this wave proves.
- **2026-10-19, PLAN-2026-10-metais-v1 W4-6.** Reading liquid-spec at the
  pinned commit found the engine apart from the gem where no statement
  said it differed. It now reads as the gem's 5.14.0 `strict2` does, under
  D1 items 2 and 5, D4 items 6 and 7 and D5 item 2. A path that begins with
  an index, as in `{{ ['a'] }}` or `{{ [k] }}`, is a syntax finding, since
  it looks a variable up by a computed name. The `size`, `first` and `last`
  shorthands are read only after a `.`, so `xs["first"]` is an index like
  any other. A boolean on either side of `<` and its kin is false. Neither
  `empty` nor `blank` equals a keyword, itself included. The one `-` of
  `{{-}}` trims both sides. A string is a loop's one item whatever its
  `offset` and `limit`. Golden Liquid's five bracketed-root cases, valid
  in its 5.12.0 corpus, became `dialect` lines under D1 items 2 and 5, and
  MTS-AUT-014's `template.xref.dynamic_reference` names no bracketed root,
  since one cannot parse. Where the engine stays apart from
  the gem, chapter 7 now says so: MTS-TPL-053 (a range used as a value is
  the array of its integers), MTS-TPL-054 (`size` of a number is 0), and
  MTS-TPL-055 (what `strict2` accepts and then ignores or overrides is
  refused, under D1 item 2).
- **2026-10-19, PLAN-2026-10-metais-v1 W5-4.** D7 items 4, 5 and 10 are
  implemented and stated in chapter 7 (MTS-TPL-056 to MTS-TPL-058). One
  binding function builds every context of item 10's table: `inputs`,
  `package`, `now` and `sources` always; `item` and `fan_out` in a fan-out's
  evaluation; and `output` only for a template body. `fan_out`'s positions
  are those of the `for_each` array, so a `when` that drops an element
  leaves the others' `length`, `last`, `previous` and `next` unchanged.
  D16 item 5's expressions are evaluated with the dialect templates are
  rendered in, so a path's `assign` of a reserved name is
  `template.reserved_name` there too.
- **2026-10-19, PLAN-2026-10-metais-v1 W5-3.** D12 and D14 are
  `internal/template`'s `embed` and `provenance`, chapter 8's MTS-TAG-001
  to MTS-TAG-010. The tags are in the one dialect check and render parse
  in; each `Renderer` renders through its own copy, whose tags read its
  inputs, identity and output kind. Readings the decisions leave open:
  - **`embed`.** A first argument that is not a string literal is
    `template.embed_unknown_input` with no `name`; an argument other than
    the name and `index:` is `template.syntax`. What the declarations
    decide — the name, and `index:` given or missing — is found once the
    template has parsed, since the engine returns no tree beside an error.
    A render reports an undeclared name as `template.embed_unknown_input`;
    check leaves it to its cross-reference's `template.xref.unknown_input`
    (ADR-007 D3 item 12) and reports `template.embed_index` from the same
    function render uses. An index that is not an integer is
    `template.embed_index`, reason `not_an_integer`, naming the kind it
    found. An input not supplied writes nothing, and its `index:` is not
    evaluated.
  - **Item 3's "before anything renders".** Every literal `embed` of every
    template an output reaches is checked after the parse and before the
    first output renders, each template once; an output any of whose
    templates needs a raw form that does not exist is not rendered, the
    finding being its. `input.raw_unavailable` met at render, for a
    computed index, carries `output_path`, which the registry adds.
  - **`template.invalid_utf8`** gains a `tag` parameter, and `filter` is
    present only for a filter. Every form of ADR-003 refuses text that is
    not UTF-8, so through a supply `embed`'s refusal cannot be reached in
    1.0; `internal/template`'s tests prove it.
  - **`provenance`.** A part is a bare word; a quoted word, a path with
    segments and a keyword other than `style:` are
    `template.provenance_argument`, reason `unknown`. `all` beside any
    other part names that part twice (reason `repeated`). A `style:`
    naming no style is `template.provenance_style`, reason `unknown_style`,
    at parse for a literal and at render otherwise; csv is reason
    `no_comments` and text `style_required`, each naming the kind. An
    input "not supplied" is one whose record has no digest: an optional
    input given its schema's default is supplied by it and records its
    digest. In the `html` style, each `-` the value holds right after a
    `-` is written `%2D`, so `1.0.0-rc--1` is `1.0.0-rc-%2D1`. In 1.0 only a
    package version can hold `--`, and no value a conforming package or
    `metais` gives holds a byte item 3 encodes as `%XX`.
  - **Item 5's "no file being rendered".** That is a path template, and D16
    item 5 refuses `embed` and `provenance` there as
    `template.tag_context` when the package is opened, whatever their
    arguments: chapter 8's MTS-TAG-011, beside MTS-MAN-018's `render`. The
    stricter refusal stands, and `template.provenance_style` has no reason
    for it. The loader's dialect for manifest expressions knows the two
    tags by name only, so that a path holding one is that refusal rather
    than an unknown tag, and the render's own parse of a path refuses them
    the same way (W5-4's meeting, `tag/tags-in-path`). In a fan-out,
    `index: fan_out.index0` embeds each item's raw form
    (`tag/embed-fan-out`).
  - **D15 item 3 and ADR-011 D10 item 3.** A vector's expected provenance
    writes the tool as `@@TOOL_NAME@@/@@TOOL_VERSION@@` and the package
    digest literally: D10 item 3 says no other token exists, the digest
    being defined over content. The work order's text asked for a token
    for the digest too; the ADR governs.
- **2026-10-19, PLAN-2026-10-metais-v1 W5-5.** D13's `asset` tag is a tag
  the dialect adds. Its arguments are checked at parse: anything but a
  string literal path given first, then at most `as:` with the literal
  `path`, `contents` or `base64`, is `template.syntax`, since D16 names no
  code for the tag's grammar. A path that names no file beneath `assets/`
  — not confined as D8 item 2 confines a partial, a directory, or no entry
  matched byte for byte — is `template.asset_not_found`, found at parse
  with the template's other parse findings, from its constructs when the
  parse stopped earlier, as `template.partial_not_found` is; `check`
  reports the same tag as ADR-007's `template.xref.missing_asset`. The
  render resolves path mode through ADR-005 D5 item 6's placement map,
  which render builds from its plan, keyed by the asset's path beneath
  `assets/`, with the tree each file is written in: one placement in the
  output's tree is written relative to the output's directory, more than
  one is `template.asset_ambiguous`, none there but some in another tree
  `template.asset_tree`, and none at all `template.asset_not_written`, each
  carrying `asset` and `output_path`. A tree is the output
  directory, or one container, as W5-6's render names it: an entry's tree
  is its container's path, and its path is the entry's within it.
  `contents` drops a leading byte order mark and refuses bytes that are not
  UTF-8 with `template.asset_not_text`; `base64` is RFC 4648 section 4,
  padded, unbroken. What the tag writes goes through the engine's tag
  writer, which never escapes (D9 item 4), and counts against the output's
  limits. The loader's dialect, which a manifest expression is parsed in,
  knows `asset` by name beside `embed` and `provenance`, so an `asset` in a
  path is `template.tag_context` (D16 item 5) under MTS-TAG-011, rather
  than an unknown tag.
- **2026-10-19, PLAN-2026-10-metais-v1 W5-2.** D9 is bound to the output
  declaration: an output whose `escape` is `html` renders with the
  engine's `Env.Escape` set to D9 item 1's mapping, for its template and
  every partial it renders (D8 item 5), and chapter 7's §7.10 states it.
  `escape`, its alias `h`, and `escape_once` mark their result safe, the
  alias read as the filter it names. `safe` may be the last filter of a
  `{{ }}` or `echo` expression only; an `assign`, a manifest expression
  (`liquid.ParseValue`) or a place before another filter is
  `template.safe_position`, found at parse with the template's other parse
  findings and located at the token, as `template.unknown_filter` is. The
  engine knows the rule as a `LastFilter`, so it holds no METAIS name.
  The METAIS filters are defined in `internal/template/filters`, a leaf
  the loader imports, and both the loader's dialect — the one a manifest's
  `path`, `when` and `for_each` and check's cross-reference parse with —
  and `internal/template`'s extend `Core()` with them: a template and an
  expression know one set of filters (D16 item 5), and `safe` in a
  `for_each` is `template.safe_position` rather than
  `template.unknown_filter`.
  `safe` and `json_script` take no argument (`template.filter_argument`).
  Their render-work charges, until the per-filter charge table states
  them: `safe` the shallow size of its input, twice, for the input and the
  identical result; `json_script` the deep size of its input before it
  encodes, then its result's byte length as units and as held bytes, as a
  string-building filter charges. D10 item 5's replacements are written
  here as the `\u` escapes they are; the mapping is unchanged.
- **2026-10-19, PLAN-2026-10-metais-v1 W5-1.** D10's `json`, `yaml`,
  `csv`, `markdown_table` and `slug` are `internal/template/filters`, and
  D11's `date` is `internal/dates`'s `Filter`, which that package
  registers with them. `internal/loader`'s dialect is `Core()` with all
  six, and `internal/template`'s is that with D7 item 8's names reserved,
  so a template and a manifest expression are parsed in one dialect (D16
  item 5): a path may apply `slug`, and a literal `date` format outside
  item 3's set is `template.date_format` when the manifest is parsed.
  Chapter 8 states them as MTS-FLT-004 to MTS-FLT-014. Where this ADR is
  silent, these readings were chosen:
  - `csv` and `markdown_table` without named columns write a row that
    lacks one of the union's names as an empty cell, as nil is written;
    only a named column a row lacks is `template.undefined_property`, and
    its `path` is the column's name, since the filter cannot see the
    expression its input came from. With no column at all, `csv` writes
    the empty string, as item 4 says `markdown_table` does.
  - `headers:` replaces the header text of named columns only: given with
    none named, or with a count other than theirs, it is
    `template.filter_argument`. `header:` is a boolean; `delimiter:` is one
    code point.
  - An element of the input that is not an object, and a cell that is an
    array or an object, are `template.filter_input` naming the element's
    or cell's kind, not the input's.
  - `slug` takes a string or nil and refuses every other kind, a number
    among them, as D10's preamble refuses an input of the wrong type.
  - `date` takes exactly one positional argument, a string, and no
    keyword. Its integer input is D4 item 2's integer: the literal `0.0`
    is a float and `template.date_input`. That finding's `type` names the
    kind, and its `global`, `now`, is present for `"now"` and `"today"`
    (item 1's naming of `now`); `template.date_format`'s `directive` is
    the directive from its `%` to the character that put it outside the
    set, and absent for an empty format.
  - D10 item 2's writer keeps W3-5's two readings, now stated in
    MTS-FLT-006: U+FFFE and U+FFFF are escaped, and an integral number of
    magnitude above 2^53−1 and below 1e21 is written in exponent notation.
  Golden Liquid's ten `date` cases run again (ADR-010's log): three pass,
  and seven are `dialect` lines under D11 item 1 (five) and D6 item 1
  (two).

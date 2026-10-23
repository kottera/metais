# 7. Template environment

**Normative.** Areas: TPL.

Templates are Liquid, in a dialect this specification pins. This chapter
specifies that dialect, the variables a template sees, partials, truthiness
and the meaning of `blank` and `empty`, what a name the data does not hold
renders as, how whitespace is written, and how a template error is
reported.

## 7.1 The dialect

The reference for the dialect is Shopify's `liquid` gem, release 5.14.0
([Liquid](A-references.md#liquid)), read with its strictest settings: the
`strict2` error mode, strict variables and strict filters, every error
raised rather than written into the output. This chapter states every rule
on which METAIS pins or departs from that release; the rest of core Liquid —
how each tag and each standard filter behaves — is adopted from the release
by reference, and is proved by running Shopify's own liquid-spec suite
([liquid-spec](A-references.md#liquid-spec)) against a conforming implementation ([chapter 15](15-conformance.md)).

<a id="mts-tpl-001"></a>**MTS-TPL-001** A processor MUST render a template as
the `liquid` gem 5.14.0 renders it under the `strict2` error mode with strict
variables and strict filters, raising every error, except where this chapter
or [chapter 8](08-filters-and-tags.md) states otherwise; where a statement of
either differs from the gem, the statement governs.

<a id="mts-tpl-002"></a>**MTS-TPL-002** A processor MUST refuse the `include`
tag with `template.include`. `render` is the dialect's one way to render a
partial (§7.8).

<a id="mts-tpl-003"></a>**MTS-TPL-003** A processor MUST treat `self` as an
ordinary name, which nothing binds: a template that reads it is refused with
`template.undefined_variable` (§7.6), and nothing looks a variable up by a
computed name.

<a id="mts-tpl-004"></a>**MTS-TPL-004** A processor MUST refuse a filter the
dialect does not define with `template.unknown_filter` when the template is
parsed, whether or not the branch holding it would ever run. A later minor
version may add a filter without changing what any earlier package renders.

<a id="mts-tpl-005"></a>**MTS-TPL-005** A processor MUST refuse text the
`strict2` grammar refuses, and a tag not in §7.2's table, with
`template.syntax`, and MUST also refuse an element the gem accepts and
ignores: `reversed` written after a named loop parameter.

<a id="mts-tpl-055"></a>**MTS-TPL-055** A processor MUST also refuse with
`template.syntax` the other elements the `strict2` grammar accepts and then
ignores or overrides: a `,` after a filter's last argument or after `cycle`'s
last value, a `:` after a filter's name with no argument after it, a
parameter that `for` or `tablerow` does not define, and a parameter of either
written twice.

<a id="mts-tpl-006"></a>**MTS-TPL-006** A template MUST be UTF-8, and a
processor MUST refuse one that is not with `template.not_utf8`, and one that
holds U+0000 with `template.syntax`.

<a id="mts-tpl-007"></a>**MTS-TPL-007** A processor MUST write a template's
text exactly as written — a byte order mark, CR LF line endings and every
other character included — with nothing normalized, trimmed or converted
beyond what whitespace control and the blank-body rule remove (§7.3).

<a id="mts-tpl-008"></a>**MTS-TPL-008** A processor MUST read a string
literal's text between its quotes with no escape sequence: `"a\nb"` is the
four characters `a`, `\`, `n` and `b`. A literal holding its own quote
character is written with the other quote.

<a id="mts-tpl-009"></a>**MTS-TPL-009** `upcase`, `downcase` and `capitalize`
MUST map each code point to one code point by Unicode 15.0.0's simple case
mapping, with no special casing and no locale: `"ß" | upcase` is `ß`, where
the gem writes `SS`.

## 7.2 Tags

Every tag of the gem's 5.14.0 release is in the dialect but `include`, with
the gem's syntax and meaning, and METAIS adds three tags of its own
([chapter 8](08-filters-and-tags.md)). The `liquid` tag is parsed by the
grammar itself rather than registered as a tag, and is in the dialect too.

| Tag | Notes |
|---|---|
| `if`, `elsif`, `else`, `unless`, `case`, `when` | `and` and `or` group from the right; no parentheses; `when` takes several values separated by `,` or `or` |
| `for`, `else`, `break`, `continue` | `limit:`, `offset:` (with `offset: continue`) and `reversed`; ranges `(a..b)` of integers; `forloop` with `index`, `index0`, `rindex`, `rindex0`, `first`, `last`, `length`, `parentloop` and `name` |
| `cycle` | inside or outside a loop; an optional group name keys the position |
| `tablerow` | `cols:`, `limit:`, `offset:`; `tablerowloop` |
| `assign`, `capture` | template-scoped (§7.7.7) |
| `increment`, `decrement` | counters in their own namespace |
| `ifchanged`, `echo`, `raw` | |
| `liquid` | one tag per line; `echo` writes |
| `comment`, `#`, `doc` | render nothing |
| `render` | §7.8 |
| `embed`, `asset`, `provenance` | METAIS's own ([chapter 8](08-filters-and-tags.md)) |

<a id="mts-tpl-010"></a>**MTS-TPL-010** A processor MUST group `and` and `or`
from the right, as the gem does — `a or b and c` is `a or (b and c)`, and
`a and b or c` is `a and (b or c)` — evaluating the left operand first, and
MUST stop at a falsy left operand of `and` and at a truthy left operand of
`or`.

<a id="mts-tpl-011"></a>**MTS-TPL-011** A processor MUST apply a `for` loop's
parameters as `offset`, then `limit`, then `reversed`, whatever order they are
written in, and MUST accept `reversed` only immediately after the collection.

<a id="mts-tpl-012"></a>**MTS-TPL-012** A processor MUST accept `cycle`
outside a loop, keying its position by its group name when it has one and
otherwise by its values, and MUST write `tablerow`'s markup as the gem does:
`<tr class="row1">` and a line feed before the first cell, `<td
class="colN">` and `</td>` around each cell, `</tr>`, a line feed and
`<tr class="rowN">` between rows, and `</tr>` and a line feed at the end.

<a id="mts-tpl-013"></a>**MTS-TPL-013** A processor MUST refuse a `for` loop's
or a `tablerow`'s `limit`, `offset` or `cols`, and a range's bound, that is
not an integer, with `template.tag_argument`, naming the argument and the
kind of value it was given.

## 7.3 Whitespace

<a id="mts-tpl-014"></a>**MTS-TPL-014** `{%-` and `{{-` MUST remove all
whitespace immediately before the token, and `-%}` and `-}}` all whitespace
immediately after it up to the next character that is not whitespace or the
next token, whitespace being U+0020, U+0009, U+000A, U+000B, U+000C and
U+000D.

<a id="mts-tpl-015"></a>**MTS-TPL-015** A processor MUST NOT trim anything
else: a tag alone on its line leaves that line's line feed in the output, and
there is no mode that trims globally.

<a id="mts-tpl-016"></a>**MTS-TPL-016** An `if`, `unless`, `case` or `for`
every branch body of which is blank — only whitespace, and the tags that write
nothing (`assign`, `capture`, `comment`, `#`, an empty `raw`, an empty `doc`,
and tags whose own bodies are blank) — MUST write none of that whitespace, as
the gem does. A `tablerow` writes its whitespace whatever its body holds.

## 7.4 Values

A template's values are the data model's — nil, boolean, number, string,
array and object ([chapter 6](06-supplying-inputs.md)) — with one distinction
the data model does not make: a number is an integer or a float.

<a id="mts-tpl-017"></a>**MTS-TPL-017** A processor MUST treat a number of the
data whose value is an integer of magnitude at most 2⁵³ − 1 as an integer and
every other number of the data as a float, however the supplier wrote it, so
`3.0` in the data is the integer 3; and a numeric literal of a template takes
its kind from its spelling, so `4` is an integer and `4.0` a float.

<a id="mts-tpl-018"></a>**MTS-TPL-018** `plus`, `minus`, `times`,
`divided_by`, `modulo`, `abs`, `ceil`, `floor`, `round`, `at_least`,
`at_most` and `sum` MUST give an integer for integer operands, `divided_by`
and `modulo` flooring as Ruby's `/` and `%` do (`7 | divided_by: -2` is `-4`,
`-7 | modulo: 2` is `1`); with a float operand, MUST give the exact real
result of each operand's shortest round-trip decimal value, rounded to the
nearest double, ties to even (`0.1 | plus: 0.2` is `0.3`); and `round` MUST
round half away from zero, giving an integer when its argument is omitted or
`0`.

<a id="mts-tpl-019"></a>**MTS-TPL-019** A processor MUST refuse an integer
result of magnitude above 2⁵³ − 1, and an integer literal beyond it, with
`template.number_range`, and a division or modulo by zero with
`template.division_by_zero`.

<a id="mts-tpl-020"></a>**MTS-TPL-020** The text of a value, which `{{ }}`,
`echo`, `cycle` and the string filters write, MUST be: for nil, the empty
string; for a boolean, `true` or `false`; for an integer, its decimal digits,
with `-` when negative; for a float, the shortest digits that read back as
it, written with at least one digit after the point when 1e-4 ≤ |x| < 1e15
(`3.0`, `0.0001`, `100000000000000.0`), and otherwise as a digit, a point,
at least one more digit, `e`, a sign and at least two exponent digits
(`1.0e+15`, `1.0e-05`), zero being `0.0` and negative zero `-0.0`; for a
string, itself; and for an array, the texts of its elements, concatenated.

<a id="mts-tpl-053"></a>**MTS-TPL-053** A range `(a..b)` that is not a
loop's collection MUST be the array of the integers from a to b, empty when b
is less than a, wherever it is a value — written, assigned, compared or given
to a filter — so `{{ (1..3) }}` writes `123`, where the gem writes `1..3`.

<a id="mts-tpl-021"></a>**MTS-TPL-021** A processor MUST refuse to write an
object as text with `template.object_output`. An object has no text; the
gem's Ruby `inspect` form is not reproduced.

<a id="mts-tpl-022"></a>**MTS-TPL-022** Iterating an object — `for` over it,
and its `first` and `last` — MUST visit its members in ascending order of the
UTF-16 code units of their names, the order of [RFC 8785](A-references.md#rfc8785),
each as a two-element array of the name and the value, whatever order the
supplier wrote them in.

<a id="mts-tpl-023"></a>**MTS-TPL-023** `==`, `!=` and `<>` MUST compare
integers and floats by value, and strings by code point with no locale; and
`<`, `>`, `<=` and `>=` MUST be false with nil on either side, and MUST be
refused with `template.comparison` between a string and a number, or with an
array or an object on either side.

<a id="mts-tpl-024"></a>**MTS-TPL-024** The shorthands `size`, `first` and
`last` MUST apply to strings, arrays and objects as the gem applies them, and
a member of an object with one of those names MUST be read in their place.

<a id="mts-tpl-054"></a>**MTS-TPL-054** The `size` filter MUST give 0 for a
number, a boolean and nil, where the gem gives an integer's size in bytes.

## 7.5 Truthiness, `blank` and `empty`

<a id="mts-tpl-025"></a>**MTS-TPL-025** A processor MUST treat nil and `false`
as false and every other value — `0`, `""`, `[]` and `{}` among them — as
true. Because an omitted optional field is nil (§7.6), `{% if inputs.x.note
%}` is the way to ask whether it was supplied.

<a id="mts-tpl-026"></a>**MTS-TPL-026** `x == empty` MUST hold exactly when x
is an empty string, array or object, so not for nil; `x == blank` MUST hold
for those, for nil, for `false` and for a string of only U+0020, U+0009,
U+000A, U+000B, U+000C and U+000D; `!=` MUST negate each; and `empty` and
`blank` written as a value outside a comparison MUST be the empty string.

## 7.6 Names

Templates are strict on every name: a name nothing binds, and a property
nothing declares and the data does not hold, are errors, never an empty
string. What the schema declares and the data omits is nil.

<a id="mts-tpl-027"></a>**MTS-TPL-027** A processor MUST resolve a top-level
name to, in order, a block- or partial-scoped binding, a template-scoped
binding, a counter, or a global (§7.7), and MUST refuse any other with
`template.undefined_variable`, naming it.

<a id="mts-tpl-028"></a>**MTS-TPL-028** Before rendering, every declared
property absent from an object of the inputs MUST be present with the value
nil, at every depth. An object's declared properties are the names in
`properties` of the subschemas at its location in the combined schema
([chapter 5](05-data-specification.md)) and of every subschema reachable from
them through `$ref`, `$dynamicRef`, `allOf`, `anyOf`, `oneOf`, `if`, `then`,
`else` and the values of `dependentSchemas`. A member's location is its
`properties` entry, each `patternProperties` entry its name matches, and
otherwise `additionalProperties`; an array element's is its `prefixItems`
entry, or else `items`. An absent object is not created, so its own declared
properties are not present either.

<a id="mts-tpl-029"></a>**MTS-TPL-029** Reading a property of an object —
`a.b`, `a["b"]` or `a[expr]` — MUST give the member's value when it is
present, a declared nil included, and a member the data holds without the
schema declaring it is present; a property neither declared nor held MUST be
refused with `template.undefined_property`, naming the expression up to the
step that failed.

<a id="mts-tpl-030"></a>**MTS-TPL-030** A processor MUST refuse reading a
property of nil with `template.nil_property`, whatever filter is applied to
the result, `default` included; `{% if a and a.b %}` reads safely, because
`and` stops at a nil `a` (MTS-TPL-010).

<a id="mts-tpl-031"></a>**MTS-TPL-031** An integer index of an array MUST
give the element, counting from the end when negative, and nil when out of
range; and a processor MUST refuse an index that is not an integer, and a
property other than `size`, `first` and `last` of a string, a number or a
boolean, with `template.undefined_property`.

<a id="mts-tpl-032"></a>**MTS-TPL-032** Every global object is closed: a
processor MUST refuse a member of `package`, `sources` or `output` that
§7.7 does not define with `template.undefined_property`.

## 7.7 Variables

### 7.7.1 The binding set

<a id="mts-tpl-033"></a>**MTS-TPL-033** A template MUST see the globals of
this section, the names its own tags create, and nothing more: no environment
variable, directory, user, host, process, clock or file outside the package
and the inputs the caller names. A template that reads any other name, such
as `env`, is refused with `template.undefined_variable` (MTS-TPL-027).

<a id="mts-tpl-034"></a>**MTS-TPL-034** A template body and every partial it
renders MUST see `inputs`, `package`, `now`, `sources` and `output`; and
`item` and `fan_out` MUST be bound only while rendering an output with
`for_each` ([chapter 9](09-outputs.md)), so that a template rendered for any
other output that reads them is refused with `template.undefined_variable`.

### 7.7.2 `inputs`

<a id="mts-tpl-035"></a>**MTS-TPL-035** `inputs` MUST be an object with one
member for every input the manifest declares, named by the input's name,
whose value is the input's validated value with its defaults applied
([chapter 6](06-supplying-inputs.md)) and its declared-absent properties nil
(MTS-TPL-028), and nil for an optional input that was not supplied.

### 7.7.3 `package`

<a id="mts-tpl-036"></a>**MTS-TPL-036** `package` MUST be an object whose
members are the manifest's `metais`, `name`, `version`, `title`,
`description`, `guide`, `authors`, `license` and `homepage`
([chapter 3](03-manifest.md)), nil for an optional member the manifest omits,
and `digest`, the package digest ([chapter 2](02-container.md)) written as
`sha256:` and 64 lower-case hexadecimal digits.

### 7.7.4 `now`

<a id="mts-tpl-037"></a>**MTS-TPL-037** `now` MUST be the render time the
caller supplied, converted to UTC and written `YYYY-MM-DDTHH:MM:SS`, then a
point and the supplied fraction of a second with its trailing zeros removed,
the point omitted when nothing remains, then `Z`; and nil when the caller
supplied none. It is a template's only source of time.

### 7.7.5 `sources`

<a id="mts-tpl-038"></a>**MTS-TPL-038** `sources` MUST have a member for every
declared input: for an input that is not a collection, an object whose one
member `origin` is `"file"` for a value from the data directory,
`"command-line"` for one from the command line, or `"default"` for a schema
default; for a collection, an array parallel to the collection's items, each
an object of the item file's `name`, its `stem` and its `form`, or nil for an
item supplied on the command line; and nil for an input that is absent.

### 7.7.6 `output`

<a id="mts-tpl-039"></a>**MTS-TPL-039** While rendering an output's body,
`output` MUST be an object of `path`, the output's path relative to the root
of its tree with `/` separators, and `root`, the relative prefix from that
path's directory to the root of its tree: `""` at the top, `"../"` one level
down, and so on.

### 7.7.7 Reserved names and shadowing

<a id="mts-tpl-040"></a>**MTS-TPL-040** A processor MUST refuse, with
`template.reserved_name`, an `assign`, `capture`, `increment` or `decrement`
of `inputs`, `package`, `item`, `fan_out`, `output`, `now`, `sources`,
`forloop` or `tablerowloop`; and MUST let a block- or partial-scoped binding
— a `for` or `tablerow` variable, a `render` parameter or a `render … as`
alias — take any name, shadowing a global only within its block or partial.

### 7.7.8 `item` and `fan_out`

An output with `for_each` ([chapter 9](09-outputs.md)) is evaluated once per
element of its array. Each evaluation binds the element to `item` and its
position to `fan_out`, so that a page names its file after its element and
links to its neighbours without index arithmetic.

<a id="mts-tpl-056"></a>**MTS-TPL-056** In each evaluation of an output
with `for_each`, `item` MUST be the element of the `for_each` array that
evaluation is for, in its `when`, its `path`, its template's body and every
partial that body renders.

<a id="mts-tpl-057"></a>**MTS-TPL-057** `fan_out` MUST be bound exactly
where `item` is, an object whose members are `index`, the evaluation's
position counted from 1; `index0`, counted from 0; `rindex` and `rindex0`,
counted from the end, from 1 and from 0; `first` and `last`, booleans;
`length`, the number of elements; and `previous` and `next`, the elements
before and after, nil at either end. Every position MUST be the element's
in the `for_each` array before any `when` is applied, so `length`, `last`,
`previous` and `next` count elements a `when` drops.

The way to name one file per item of a collection after the item's own
file is `for_each: inputs.<name>`, a `when` to drop the items not wanted,
and `sources.<name>[fan_out.index0].stem` in the path: since `fan_out`
counts positions before `when`, `index0` is the item's position in the
collection (§7.7.5). A `for_each` expression that filters or reorders the
collection loses that correspondence.

### 7.7.9 Where each global is bound

| Context ([chapter 9](09-outputs.md)) | `inputs` `package` `now` `sources` | `item` `fan_out` | `output` |
|---|---|---|---|
| a declaration's `for_each` expression | yes | no: it computes them | no |
| `when` of a declaration without `for_each` | yes | no | no |
| `when` of a declaration with `for_each`, evaluated per element | yes | yes | no |
| `path` template | yes | as for `when` | no: the path is being computed |
| template body, and every partial it renders | yes | in a fan-out only | yes |

`forloop` and `tablerowloop` exist only inside their blocks, in every
context where those tags are allowed.

<a id="mts-tpl-058"></a>**MTS-TPL-058** A processor MUST bind in each
context exactly the globals this table gives it, and MUST refuse a read of
a global the table does not bind there — `item` or `fan_out` in a
`for_each` expression, `output` in a `path`, a `when` or a `for_each` —
with `template.undefined_variable`.

## 7.8 Partials

<a id="mts-tpl-041"></a>**MTS-TPL-041** `render` MUST name its partial by a
string literal that is an entry path relative to `templates/`, with its full
file name and extension, matched byte for byte, with no suffix added and no
directory of the caller consulted; and a processor MUST refuse a name that is
not a string literal with `template.partial_name`.

<a id="mts-tpl-042"></a>**MTS-TPL-042** A processor MUST refuse, when the
calling template is parsed and whether or not the tag would ever run, a
partial name with a leading `/`, a `\`, a `.` or `..` segment or an empty
segment, or one that names no template of the package, with
`template.partial_not_found`.

<a id="mts-tpl-043"></a>**MTS-TPL-043** A processor MUST read a partial only
from the package's own `templates/` entries, read in place, and never from
the host: a name the package lacks is refused even where a file of that name
exists on the host.

<a id="mts-tpl-044"></a>**MTS-TPL-044** A processor MUST accept `render "p"`,
`render "p", k: v, …`, `render "p" with x` and `render "p" with x as n`,
`render "p" for xs` and `render "p" for xs as n`, which renders the partial
once per element and binds `forloop` in it, with the gem's meaning.

<a id="mts-tpl-045"></a>**MTS-TPL-045** A partial MUST see its parameters, its
own bindings and the globals, and none of its caller's `assign`, `capture`,
loop or counter bindings; nothing a partial binds is seen by its caller.

<a id="mts-tpl-046"></a>**MTS-TPL-046** A partial MAY render itself, directly
or through others; a processor MUST refuse a render whose `render` calls
active at once exceed the `partial-depth` limit
([chapter 14](14-security.md)) with `limit.exceeded.template` naming it.

## 7.9 Template errors

<a id="mts-tpl-047"></a>**MTS-TPL-047** Every condition this chapter refuses
MUST be reported as a finding of error severity, and nothing MUST ever be
written into an output in its place.

<a id="mts-tpl-048"></a>**MTS-TPL-048** A template finding MUST be about the
template it is in, named by its entry path, and located with origin
`package`, that entry path as its file, and the 1-based line and column of
the `{{` or `{%` that opens the failing token, the column counted in code
points; inside a `liquid` tag, the line is the inner line and the column its
first character that is not whitespace; and for text that is not UTF-8 or
holds U+0000, the first such byte.

<a id="mts-tpl-049"></a>**MTS-TPL-049** A finding met while rendering an
output MUST carry the output's path; and one met inside a partial MUST carry,
as related locations noted `rendered from`, each `render` tag on the way to
it, outermost first, from the output's own template.

<a id="mts-tpl-050"></a>**MTS-TPL-050** *(not observable)* The message of a
template finding MUST NOT quote a value of the inputs: a finding names the
variable, the filter and the kind of value it found, never the value, and an
engine's own error text is mapped to a code, never passed on.

> **Note.** `metais` builds each template finding's message from its code's
> registered template, which names no value of the inputs, and its tests hold
> every template finding to it. A message's wording is the implementation's
> ([chapter 12](12-findings.md)), so no vector compares it.

<a id="mts-tpl-051"></a>**MTS-TPL-051** A processor MUST parse every template
a render uses and report every parse-time finding, at most one
`template.syntax` for a template, whose parse stops there; MUST then attempt
every output, each reporting its first render error; and, when any finding is
an error, MUST write no output at all.

<a id="mts-tpl-052"></a>**MTS-TPL-052** A processor MUST refuse a filter's
input of a kind the filter does not take with `template.filter_input`, an
argument it does not take with `template.filter_argument`, and a decoded
result that is not UTF-8 with `template.invalid_utf8`, each naming the filter.

## 7.10 Escaping

An output declares how the values its templates write are escaped: its
`escape` is `none`, the default, or `html` ([chapter 9](09-outputs.md)).
Escaping is a property of the output being rendered, never of a template, so
one partial serves both kinds of output.

<a id="mts-tpl-059"></a>**MTS-TPL-059** In an output whose `escape` is
`html`, a processor MUST write the text of every value that `{{ }}`, `echo`
or `cycle` writes with each `&` written as `&amp;`, each `<` as `&lt;`, each
`>` as `&gt;`, each `"` as `&quot;` and each `'` as `&#39;`, and nothing
else changed, unless the value is safe (MTS-TPL-060); and MUST write the
template's own text, outside `{{ }}` and `{% %}`, unescaped. In an output
whose `escape` is `none`, a processor MUST escape nothing.

<a id="mts-tpl-060"></a>**MTS-TPL-060** A value is safe, and a processor
MUST write it as it is in every output, when the last filter applied to it
is `safe`, `escape`, `escape_once` or `json_script`
([chapter 8](08-filters-and-tags.md)), or when it is the text of a `capture`
made while rendering an output whose `escape` is `html`, whose values were
escaped as they were captured. Safety MUST NOT be carried through a later
filter: `{{ x | escape | newline_to_br }}` escapes `newline_to_br`'s result
again, and `{{ x | escape | newline_to_br | safe }}` writes escaped text with
line breaks.

<a id="mts-tpl-061"></a>**MTS-TPL-061** A partial MUST be escaped as the
output being rendered is: a partial rendered into an output whose `escape` is
`html` is escaped, and the same partial rendered into one whose `escape` is
`none` is not.

<a id="mts-tpl-062"></a>**MTS-TPL-062** A processor MUST write the text of a
`raw` block, and what `embed`, `asset` and `provenance` write
([chapter 8](08-filters-and-tags.md)), unchanged in every output, whatever
its `escape`. An output's `escape` applies to every file it renders: each
file of a mirror, and each template entry of a container by its own
`escape`.

> **Note.** Escaping does not make output safe inside JavaScript, CSS or a
> URL. `json_script` is the filter for data placed in a `<script>` element,
> and `url_encode` the one for a URL component; the dialect offers nothing
> else for those contexts.

## 7.11 Render work

A render's template work is counted in units and bounded by the
`render-work` limit ([chapter 14](14-security.md#mts-sec-008)): 16 units for
each node evaluated — an output, a tag, a filter application, an operator —
and, for each filter application and operator, units for the sizes of its
input, its arguments and its result. Writing an array and reading a string's
`size` shorthand are charged units too. A value's *shallow size* is a
string's byte length, an array's element count, an object's member count, and
1 for anything else; its *deep size* is its shallow size and the deep sizes of
every element and member value beneath it.

An *operator* is a comparison (`==`, `!=`, `<>`, `<`, `>`, `<=`, `>=`,
`contains`), an `and`, an `or`, and a range made a value. A range is made a
value wherever it is written but as the collection a `for`, a `tablerow` or a
`render … for` iterates, which visits the range's integers without making
them an array: as in an `assign`, an output, a filter's input or argument,
or a condition. A range's operands are its two bounds, and its result is the
array of its integers.

<a id="mts-tpl-063"></a>**MTS-TPL-063** A processor MUST charge each filter
application 16 units and then, once its argument count and keywords are ones
the filter takes, the shallow sizes of its input and of each of its
arguments; then the units the table below adds, before the work they count;
and then its result: the shallow size of the result, or, where the table
says *built*, the byte length of the string it builds. A charge that would
take the render beyond `render-work` refuses the render at the filter's
token, and nothing of the work it counts is done. Beyond its filters, a
processor MUST charge a range made a value 16 units, 1 for each of its two
bounds and 1 for each integer it makes, 18 + n units for n integers, before
it makes the array; a range a `for`, a `tablerow` or a `render … for`
iterates is charged nothing as a value. It MUST charge an output, an `echo`
or a `cycle` that writes an array, beyond the node's 16 units, 1 unit for
each element of the array and for each element of every array nested in it,
at every depth, each array's before its elements are written. It MUST charge
a `.size` step that reads a string's `size` shorthand
([MTS-TPL-024](#mts-tpl-024)) 1 unit for each byte of the string; every other
step of a path is charged nothing. In the table, *text work*
is the deep size of an array whose text the filter takes, and nothing for
any other value; *flattening* is, for an array, the element count of every
array nested in it, at every depth; *sorting n keys* is n·⌈log₂ n⌉ units and
each byte of a string key ⌈log₂ n⌉ times, nothing for fewer than two keys;
and *beyond the shallow size* is a value's deep size less its shallow size.

| Filter | Units before the work | Result |
|---|---|---|
| `abs`, `ceil`, `floor`, `round`, `plus`, `minus`, `times`, `divided_by`, `modulo`, `at_least`, `at_most`, `size`, `first`, `last`, `default`, `slice` | — | shallow size |
| `sum` | flattening the input, and the byte length of each term that is a string: each element or, with a property, its value of it, up to the first element the property refuses | shallow size |
| `reverse`, `map`, `compact`, `concat` | flattening the input | shallow size |
| `uniq` | flattening the input, and the deep size of each key: each element or, with a property, its value of it | shallow size |
| `sort` | flattening the input, and sorting its keys: the elements or, with a property, their values of it | shallow size |
| `sort_natural` | flattening the input, the text work of each key that is not nil, and sorting the keys, a key's bytes being those of its text | shallow size |
| `where`, `reject`, `has`, `find`, `find_index` | flattening the input, and, when it has elements and the value sought is a string, an array or an object, that value's deep size once for each element | shallow size |
| `join` | flattening the input, and the glue's text work | built |
| `append`, `prepend`, `upcase`, `downcase`, `capitalize`, `strip_newlines`, `newline_to_br`, `strip_html`, `base64_encode`, `base64_url_safe_encode`, `base64_decode`, `base64_url_safe_decode` | the text work of the input and of each argument | built |
| `escape`, `h`, `escape_once`, `squish`, `url_encode`, `url_decode` | the input's text work; nothing for nil, which is the result | built; for nil, shallow size |
| `truncate`, `truncatewords` | the text work of the input and of the ellipsis; nothing for nil, which is the result | built when the text is cut; shallow size otherwise |
| `remove`, `replace`, `remove_first`, `replace_first`, `remove_last`, `replace_last` | the text work of the input and of each argument | built; shallow size for a `_first` or `_last` filter that finds nothing |
| `strip`, `lstrip`, `rstrip` | the input's text work | shallow size |
| `split` | the text work of the input and of the pattern | shallow size |
| `json`, `yaml` | the input's deep size beyond its shallow size | built, counted before it is written |
| `csv`, `markdown_table` | the input's deep size beyond its shallow size, and, with no column named, sorting the rows' member names, each name once for each row that has it | built, counted before it is written |
| `safe` | — | shallow size |
| `json_script` | the input's deep size beyond its shallow size | built |
| `slug`, `date` | — | built |
| `number`, `amount`, `duration` | — | built; for nil, which is the result, shallow size |

> **Note.** `json`, `yaml`, `csv` and `markdown_table` can write many times
> their input's size, so each counts its result's length before it writes a
> byte, and a render whose work would exceed `render-work` is refused before
> the text is built.

<a id="mts-tpl-064"></a>**MTS-TPL-064** A processor MUST charge a `contains`
whose left operand is an array and whose right operand is a string, beside
the operator's 16 units and the sizes of its operands and its result, the
string's byte length once for each element of the array, before it compares
the string with any element. A charge that would take the render beyond
`render-work` refuses the render at the condition's token.

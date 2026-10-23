# 8. Filters and tags

**Normative.** Areas: FLT, TAG.

METAIS adds a catalog of filters and tags to the Liquid dialect of chapter 7,
each defined to the byte so that every conforming processor renders it alike.
This chapter specifies the filter catalog, and the `embed`, `asset` and
`provenance` tags, with the comment syntax `provenance` writes for each kind
of output.

## 8.1 Filters

Every filter of this section is defined in every output; how its result is
written depends on the output's escaping ([§7.10](07-template-environment.md#mts-tpl-059)).

### 8.1.1 `escape` and `escape_once`

<a id="mts-flt-001"></a>**MTS-FLT-001** `escape`, and its alias `h`, MUST
write the text of its input with each `&`, `<`, `>`, `"` and `'` replaced as
MTS-TPL-059 states, and `escape_once` MUST do the same except to an `&` that
already opens a character reference: one or more ASCII letters, or `#` and
one or more ASCII digits, followed by `;`. Each MUST give nil for nil, and
its result MUST be safe (MTS-TPL-060), so that it is not escaped twice in an
output whose `escape` is `html`.

### 8.1.2 `safe`

<a id="mts-flt-002"></a>**MTS-FLT-002** `safe` MUST give its input
unchanged, marked safe (MTS-TPL-060), and takes no argument. A processor MUST
refuse `safe` applied other than as the last filter of a `{{ }}` or `echo`
expression — before another filter, in an `assign`, or in a manifest
expression — with `template.safe_position` when the template is parsed,
whether or not the expression would ever be evaluated. In an output whose
`escape` is `none`, `safe` changes nothing.

### 8.1.3 `json_script`

<a id="mts-flt-003"></a>**MTS-FLT-003** `json_script` MUST give the
canonical form of its input (MTS-MAN-021), with each `<` further written as
`\u003c`, each `>` as `\u003e`, each `&` as `\u0026`, each U+2028 as
`\u2028` and each U+2029 as `\u2029`; it takes no argument, and its result
MUST be safe (MTS-TPL-060). Its text can neither close a `<script>` element
nor open an HTML comment, and is the JSON only: the template writes the
element around it, as in
`<script type="application/json" id="data">{{ x | json_script }}</script>`.

### 8.1.4 Arguments

Beside the standard filters [chapter 7](07-template-environment.md) adopts,
the dialect has these, each defined to the byte. A filter's named argument is
written `name: value`, after its positional arguments:
`{{ rows | csv: "name", "total", delimiter: ";" }}`.

<a id="mts-flt-004"></a>**MTS-FLT-004** A processor MUST refuse an
application of a filter of this chapter that gives it more or fewer
positional arguments than it takes, a keyword it does not take, a keyword
twice, or an argument of a kind it does not take, with
`template.filter_argument` naming the argument by its 1-based position or its
keyword; and an input of a kind it does not take with `template.filter_input`
naming the kind found. A refusal that the literal arguments alone decide MUST
be found when the template is parsed.

### 8.1.5 `json`

<a id="mts-flt-005"></a>**MTS-FLT-005** `json` MUST write its input, of any
kind, as its [RFC 8785](A-references.md#rfc8785) serialization: members in
the order of their names' UTF-16 code units; strings escaping `"`, `\` and
U+0000 to U+001F only, U+0008, U+0009, U+000A, U+000C and U+000D as `\b`,
`\t`, `\n`, `\f` and `\r` and the rest as `\u00xx` in lower-case hexadecimal;
numbers as [ECMAScript](A-references.md#ecma262)'s `Number::toString` writes
them (`3` for `3.0`, `2.5` for `2.50`, `1e+21`); nil as `null`; and no
whitespace. Its one keyword, `indent:`, an integer from 1 to 8, MUST
pretty-print instead: after the `[` or `{` of a non-empty array or object and
after each `,`, a line feed and the enclosing indentation plus `indent`
spaces; `": "` between a name and its value; the closing bracket on a line of
its own at the enclosing indentation; `[]` and `{}` for empty ones; and no
final line feed.

### 8.1.6 `yaml`

<a id="mts-flt-006"></a>**MTS-FLT-006** `yaml` MUST write its input, of any
kind, as a [YAML 1.2.2](A-references.md#yaml) block document with no `---`,
no anchor, alias, tag or block scalar, no flow collection but `{}` and `[]`,
and no final line feed, so that a value has one text:

- an object writes `name: value` for each member, in the order of `json`; a
  member whose value is a non-empty object or array writes `name:` and then
  its block on the following lines, indented two spaces more;
- an array writes `- ` before each element; an element that is a non-empty
  object or array starts on the same line, after the `- `, and its following
  lines are indented two spaces more than the `-`;
- an empty object is `{}`, an empty array `[]`, nil `null`, and a boolean
  `true` or `false`;
- a number is written as `json` writes it, except an integral number of
  magnitude above 2⁵³ − 1 and below 10²¹: that is written as its shortest
  round-trip digits, a point after the first of them when there are more,
  `e+` and the decimal exponent (`9.007199254740992e+15`), a text a reader of
  inputs accepts, where `json`'s integer text is refused
  ([chapter 6](06-supplying-inputs.md));
- a string or a name is written plain when it matches
  `^[A-Za-z_/]([A-Za-z0-9_./ -]*[A-Za-z0-9_./])?$` and is not, ignoring ASCII
  case, one of `true`, `false`, `yes`, `no`, `on`, `off`, `y`, `n` and
  `null`; otherwise it is written between `"`, escaping `"`, `\` and U+0000
  to U+001F as `json` does, and U+007F to U+009F, U+2028, U+2029, U+FEFF,
  U+FFFE and U+FFFF as `\uxxxx` in lower-case hexadecimal.

### 8.1.7 `csv` and `markdown_table`

`csv` and `markdown_table` write an array of objects as a table. The columns
are the positional arguments, each a string naming a member, in the order
written; without any, they are the union of the rows' member names, in the
order of `json`. A cell is the text of its row's member
([chapter 7](07-template-environment.md#mts-tpl-020)): nil, and a member a row
lacks when no column is named, are an empty cell. `headers:`, an array of
strings, one for each named column in order, replaces the header text, which
is otherwise the column's name.

<a id="mts-flt-007"></a>**MTS-FLT-007** `csv` MUST write the header record,
unless `header: false` is given, and then one record per row, every record
ending with U+000A, the last included, and its fields separated by the
delimiter, `,` unless `delimiter:` names another; and MUST write a field
between `"`, each `"` in it doubled, when it holds the delimiter, `"`, CR or
LF, and as it is otherwise, as [RFC 4180](A-references.md#rfc4180) does. With
no column, `csv` MUST write the empty string. `header:` MUST be `true` or
`false`; `delimiter:` MUST be one character other than `"`, CR and LF.

<a id="mts-flt-008"></a>**MTS-FLT-008** `markdown_table` MUST write a
[GitHub Flavored Markdown](A-references.md#gfm) table: the header line, the
delimiter line with `---` for each column, and one line per row, each line
written `| ` followed by its cells separated by ` | ` and then ` |` and
U+000A. In every cell, each `|` MUST be written `\|` and each line break, CR
LF, LF or CR, `<br>`; nothing else is escaped, so a cell's Markdown stays
Markdown. With no column, `markdown_table` MUST write the empty string.

<a id="mts-flt-009"></a>**MTS-FLT-009** `csv` and `markdown_table` MUST
refuse with `template.filter_input` an input that is not an array, an element
that is not an object, and a cell that is an array or an object, naming the
kind found; with `template.undefined_property` a column named by an argument
that a row lacks, its `path` the column's name — a member the input's schema
declares is present, as nil ([chapter 7](07-template-environment.md#mts-tpl-028));
and with `template.filter_argument` a `headers:` that is not an array of
strings, that is given with no column named, or that does not have one
string for each named column.

### 8.1.8 `slug`

<a id="mts-flt-010"></a>**MTS-FLT-010** `slug` MUST give nil for nil, MUST
refuse any other input but a string with `template.filter_input`, MUST refuse
a string whose slug is empty with `template.slug_empty`, and MUST otherwise
make the string a slug matching `[a-z0-9]+(-[a-z0-9]+)*`, by these steps in
order:

1. normalize to Normalization Form KD ([Unicode 15.0.0](A-references.md#unicode));
2. remove every code point of general category Mn;
3. replace each of these Latin letters, which have no decomposition: `ß` and
   `ẞ` with `ss`; `æ` and `Æ` with `ae`; `œ` and `Œ` with `oe`; `ø` and `Ø`
   with `o`; `đ`, `Đ`, `ð` and `Ð` with `d`; `þ` and `Þ` with `th`; `ł` and
   `Ł` with `l`; `ı` with `i`; `ħ` and `Ħ` with `h`; `ŧ` and `Ŧ` with `t`; `ŋ`
   and `Ŋ` with `ng`; and `ĸ` with `k`;
4. lower-case ASCII `A` to `Z`;
5. replace each maximal run of code points other than `a` to `z` and `0` to
   `9` with one `-`;
6. remove a `-` at the start or the end.

The letters are replaced after normalization, so a letter that decomposes
to one of them is spelled as that letter is: `Ǿ` decomposes to `Ø` and a
combining acute, and `"Ǿrsted" | slug` is `orsted`.

So `"Café Déjà-vu" | slug` is `cafe-deja-vu`, and `"日本語"` has no slug: the
author supplies one as data.

### 8.1.9 `date`

`date` formats a date. It reads no clock and no time zone: the render time
reaches a template only as `now` ([chapter 7](07-template-environment.md#mts-tpl-037)),
and a value is written in the offset it carries.

<a id="mts-flt-011"></a>**MTS-FLT-011** `date` MUST accept as its input a
string that is an [RFC 3339](A-references.md#rfc3339) `full-date`, which is
midnight UTC (`2026-10-18`), or `date-time`, with `T` and `Z` in either case,
any number of fraction digits, and every field in its range, second 60
refused (`2026-10-18T09:30:00.25+02:00`); an integer, as seconds since
1970-01-01T00:00:00Z, in UTC; and nil, which gives nil; where the instant, in
UTC, is within the years 0000 to 9999. It MUST refuse every other input with
`template.date_input`, naming its kind: a float, a boolean, an array, an
object, and any other text, a string of digits among it; and for the strings
`"now"` and `"today"` the finding MUST name the global `now` as its `global`.

<a id="mts-flt-012"></a>**MTS-FLT-012** `date` MUST write a value in the
offset it carries — UTC for a `full-date` and an integer, the written offset
for a `date-time` — with no conversion, and MUST NOT consult a time zone
database or anything of the host's time zone. A fraction is the digits
supplied, never rounded.

<a id="mts-flt-013"></a>**MTS-FLT-013** `date` MUST take one positional
argument, its format, a string, and no keyword; and MUST write the format's
text as it is, but for each directive, which MUST be one of the table below,
optionally preceded by the flags `-` (no padding), `_` (padding with spaces),
`0` (padding with zeros) and `^` (upper case), each at most once and at most
one of the first three. A padding flag changes only a number's padding; `^`
upper-cases the directive's whole text. Any other directive — `%` followed by
anything else, a lone `%` at the end included — and an empty format MUST be
refused with `template.date_format`, its `directive` the directive from its
`%` to the character that put it outside the set, and absent for an empty
format.

| Directive | Writes |
|---|---|
| `%a`, `%A` | the weekday's English name, its first three letters and whole: `Sun`, `Sunday` |
| `%b`, `%h`, `%B` | the month's English name, its first three letters and whole: `Jan`, `January` |
| `%C` | the year divided by 100, rounded down, two digits |
| `%y` | the year modulo 100, two digits |
| `%Y` | the year, four digits |
| `%G` | the year of the [ISO 8601](A-references.md#iso8601) week, four digits |
| `%m` | the month, `01` to `12` |
| `%d` | the day of the month, `01` to `31` |
| `%e` | the day of the month, two characters, padded with a space |
| `%j` | the day of the year, `001` to `366` |
| `%H` | the hour, `00` to `23` |
| `%k` | the hour, `0` to `23`, two characters, padded with a space |
| `%I` | the hour on the 12-hour clock, `01` to `12` |
| `%l` | the hour on the 12-hour clock, `1` to `12`, two characters, padded with a space |
| `%p`, `%P` | `AM` or `PM`, and `am` or `pm`, `AM` before noon |
| `%M` | the minute, `00` to `59` |
| `%S` | the second, `00` to `59` |
| `%L` | the first three digits of the fraction, `0` filling the rest |
| `%N`, `%3N`, `%6N`, `%9N` | the first nine, three, six or nine digits of the fraction, `0` filling the rest |
| `%s` | the seconds since 1970-01-01T00:00:00Z, rounded down, with `-` when negative |
| `%u` | the ISO 8601 weekday, `1` (Monday) to `7` |
| `%w` | the weekday, `0` (Sunday) to `6` |
| `%U` | the week of the year whose weeks begin on Sunday, `00` to `53` |
| `%W` | the week of the year whose weeks begin on Monday, `00` to `53` |
| `%V` | the ISO 8601 week, `01` to `53` |
| `%z` | the offset as `+hhmm` or `-hhmm` |
| `%:z` | the offset as `+hh:mm` or `-hh:mm` |
| `%Z` | `UTC` at offset zero, and the offset as `%:z` writes it otherwise |
| `%c` | `%a %b %e %H:%M:%S %Y` |
| `%D`, `%x` | `%m/%d/%y` |
| `%F` | `%Y-%m-%d` |
| `%r` | `%I:%M:%S %p` |
| `%R` | `%H:%M` |
| `%T`, `%X` | `%H:%M:%S` |
| `%n`, `%t`, `%%` | a line feed, a tab, and `%` |

A number is padded to the width the table shows with zeros, or with spaces
where it says so; `-` writes it with no padding, and `_` and `0` pad it with
spaces or zeros instead.

> **Note.** The format language is Ruby's `strftime`, closed to this set, and
> `%Z` departs from it: Ruby writes a zone's name, which a value that carries
> only an offset does not have.

### 8.1.10 Numbers for readers

`number` and `amount` write a number as a reader reads it: with a chosen
count of decimals, a decimal point, and a separator between groups of
digits. Neither reads a locale. Every separator is an argument, and what an
argument left out means is the same in every package: a point `.`, no group
separator, and the value's own decimals.

<a id="mts-flt-015"></a>**MTS-FLT-015** `number` and `amount` MUST accept
as their input a number; a string that is a *decimal text*, an optional
`-`, then `0` or an ASCII digit `1` to `9` followed by any ASCII digits,
then optionally `.` and one or more ASCII digits, and nothing else; and nil,
which gives nil. A processor MUST refuse every other input with
`template.number_input`, naming the filter and the kind found: a boolean,
an array, an object, and any other string, among them one with a space, a
`+`, a leading zero, an exponent or a separator.

<a id="mts-flt-016"></a>**MTS-FLT-016** The value of a number MUST be the
shortest decimal that reads back as its binary64 value, the decimal
[MTS-TPL-018](07-template-environment.md#mts-tpl-018)'s arithmetic computes
on, and the value of a decimal text MUST be the decimal it spells, exactly,
whatever its count of digits. With `decimals:` n, a processor MUST round
the value to n digits after the point, a value exactly halfway between the
two nearest being rounded away from zero, and MUST write n digits after the
point;
without `decimals:`, it MUST write the value's digits after the point with
trailing zeros removed. It MUST write the integer part's digits with no
leading zero but a lone `0`, no point when no digit follows it, `-` before
the first digit when the value written is below zero, and never an
exponent: `-0.004` with `decimals: 2` is `0.00`, and `1e21` is written with
its 22 digits.

<a id="mts-flt-017"></a>**MTS-FLT-017** `decimals:` MUST be an integer from
0 to 20; `point:` MUST be a string of one code point other than an ASCII
digit and `-`, and is `.` when not given; and `group:` MUST be the empty
string, which it is when not given, or a string of one code point other than
an ASCII digit, `-` and the point. A processor MUST refuse any other with
`template.filter_argument`, MUST write the point as `point:` gives it, and,
when `group:` is not empty, MUST write it between two digits of the integer
part wherever the digits to its right in the integer part number a multiple
of three, and nowhere in the fraction.

> **Note.** The value of the number 2.675 is the decimal 2.675, so
> `{{ 2.675 | number: decimals: 2 }}` writes `2.68`, as
> `{{ 2.675 | round: 2 }}` gives 2.68. A formatter that rounds a binary64's
> exact binary value — C's `printf`, Go's `strconv.FormatFloat`,
> ECMAScript's `toFixed` — writes `2.67`, because the binary64 nearest 2.675
> lies just below it; `1.005` and `0.125` part the same way. A number keeps
> about fifteen significant digits exactly. A supplier who needs more gives
> the value as a decimal text, which the input's schema declares as a
> string.

### 8.1.11 `number`

<a id="mts-flt-018"></a>**MTS-FLT-018** `number` MUST take no positional
argument and the keywords `decimals:`, `group:` and `point:`, each
optional, and MUST write its value as
[MTS-FLT-016](#mts-flt-016) and [MTS-FLT-017](#mts-flt-017) state.

So `{{ 48920000 | number: decimals: 2, group: "," }}` writes
`48,920,000.00`, `{{ 1234.5 | number: group: " ", point: "," }}` writes
`1 234,5`, `{{ "12345678901234567.891" | number: decimals: 2, group: "," }}`
writes `12,345,678,901,234,567.89`, and `{{ -2.5 | number: decimals: 0 }}`
writes `-3`.

### 8.1.12 `amount`

`amount` writes an amount of money with its
[ISO 4217](A-references.md#iso4217) code. It takes the amount in the
currency's major unit — euros, not cents — and the code as its argument,
usually an input the supplier gives, so one template writes a report in any
currency. The count of decimals comes from the code, by
[Appendix C](C-currency-minor-units.md), unless `decimals:` gives it. It
writes no currency symbol: `$` names more than one currency.

<a id="mts-flt-019"></a>**MTS-FLT-019** `amount` MUST take one positional
argument, the code, a string, and the keywords `decimals:`, `group:`,
`point:` and `code:`, each optional; `code:` MUST be `"before"`, which it is
when not given, or `"after"`. A processor MUST write the code, one U+0020
and the number when `code:` is `"before"`, and the number, one U+0020 and
the code when it is `"after"`, the number being the value as
[MTS-FLT-016](#mts-flt-016) and [MTS-FLT-017](#mts-flt-017) write it, with
`decimals:` when it is given and otherwise with the code's minor units in
Appendix C.

So `{{ 48920000 | amount: "EUR", group: "," }}` writes
`EUR 48,920,000.00`, `{{ 5000 | amount: "JPY" }}` writes `JPY 5000`,
`{{ 1.5 | amount: "BHD" }}` writes `BHD 1.500`, and
`{{ -5 | amount: "EUR", code: "after" }}` writes `-5.00 EUR`.

<a id="mts-flt-020"></a>**MTS-FLT-020** A processor MUST refuse with
`template.amount_currency` a code that is not three ASCII capital letters,
its `reason` `malformed`; and, when `decimals:` is not given, a code
Appendix C does not list or lists with no minor units, its `reason`
`no_minor_unit`. With `decimals:`, any three ASCII capital letters are a
code. A code written as a literal MUST be judged when the template is
parsed, and the finding MUST NOT hold the code.

### 8.1.13 `duration`

`duration` writes a length of time, by a format whose directives mirror
`date`'s: `%d`, `%H`, `%M` and `%S` are its days, hours, minutes and
seconds. It reads a count of seconds, an
[ISO 8601](A-references.md#iso8601) duration, or a string of fields
separated by `:` whose meaning the template names with `fields:`, so
`00:07:25` read with `fields: "d:h:m"` is seven hours and twenty-five
minutes, never seven minutes and twenty-five seconds.

<a id="mts-flt-021"></a>**MTS-FLT-021** Without `fields:`, `duration` MUST
accept as its input an integer from 0 to 2⁵³ − 1, as seconds; a string that
is an ISO 8601 duration matching `duration` in the grammar below, a week
being 604 800 seconds, a day 86 400, an hour 3 600 and a minute 60; and nil,
which gives nil. With `fields:`, it MUST accept a string of as many runs of
one or more ASCII digits as `fields:` names, separated by `:`, each after
the first at most 23 for `h` and 59 for `m` and `s`, and nil. A processor
MUST refuse every other input with `template.duration_input`, naming the
kind found and a `reason`: `kind` for a float, a boolean, an array and an
object, and for an integer given with `fields:`; `calendar` for a string
beginning `P` that holds `Y`, or `M` before any `T`, whose length depends on
the calendar; `range` for a value below zero or above 2⁵³ − 1 seconds, and
for a field beyond its range; and `form` for any other string.

```abnf
duration = %s"P" ( 1*DIGIT %s"W" / 1*DIGIT %s"D" [ %s"T" time ] / %s"T" time )
time     = 1*DIGIT %s"H" [ 1*DIGIT %s"M" ] [ 1*DIGIT %s"S" ]
         / 1*DIGIT %s"M" [ 1*DIGIT %s"S" ]
         / 1*DIGIT %s"S"
```

<a id="mts-flt-022"></a>**MTS-FLT-022** `duration` MUST take one positional
argument, its format, a string, and the one keyword `fields:`, which MUST
be one of `"d:h:m:s"`, `"d:h:m"`, `"h:m:s"`, `"h:m"` and `"m:s"`. It MUST
write the format's text as it is, but for each directive: `%d`, `%H`, `%M`
and `%S`, each at most once and optionally preceded by one of the flags
`-`, `_` and `0`, and `%%`, which writes `%`. The largest unit the format
names MUST be written as the whole count of that unit in the duration,
rounded down, and each smaller unit it names as the whole count of that
unit in what the larger units named leave, rounded down, so what lies below
the smallest unit named is dropped and nothing is rounded up. `%d` MUST be
written with no padding and `%H`, `%M` and `%S` with zeros to two digits;
`-` writes no padding, and `_` and `0` pad to two digits with spaces or
zeros. Any other directive — `%` followed by anything else, a directive
named twice, a lone `%` at the end included — and an empty format MUST be
refused with `template.duration_format`, its `directive` the directive from
its `%` to the character that put it outside the set, and absent for an
empty format; a format written as a literal, when the template is parsed.

So a duration of 26 700 seconds is `0 d 7 h 25 min` by the format
`"%d d %-H h %-M min"`, `07:25` by `"%H:%M"` and `445 min` by
`"%-M min"`; and `{{ "00:07:25" | duration: "%-H h %M min", fields: "d:h:m" }}`
writes `7 h 25 min`.

### 8.1.14 What 1.0 does not have

<a id="mts-flt-014"></a>**MTS-FLT-014** The filters of version 1.0 are the
standard filters [chapter 7](07-template-environment.md) adopts and those of
this chapter. A processor MUST refuse any other filter with
`template.unknown_filter` ([chapter 7](07-template-environment.md#mts-tpl-004)):
among them `sha256`, an indentation filter such as `indent`, a filter that
reads a locale or a shop's settings such as `money`, and any filter that
would read something other than its input and its arguments. A later minor
version may add one.

<a id="mts-flt-023"></a>**MTS-FLT-023** A processor MUST give the
`template.unknown_filter` finding of a filter named `money`,
`money_with_currency` or `money_without_currency` the `alternative`
`amount`.

> **Note.** Those three are filters of Shopify's storefront, not of the
> `liquid` gem: they take an amount in cents and read the shop's currency
> settings. METAIS has no `money` filter. `amount` takes the amount in the
> currency's major unit, and its code as an argument, so a value in cents is
> divided by 100 first, or by the power of ten its currency's minor units
> name.

## 8.2 `embed`

`embed` writes the bytes of one input exactly as they were supplied: a
stylesheet, a fragment of markup, a script, kept by the data directory and
placed in an output untouched. Its bytes are the input's raw form: the bytes
of the one data-directory file ([chapter 6](06-supplying-inputs.md)) the
input's whole value, or a collection item's, was read from, exactly as they
were read, which exists only when nothing on the command line changed that
value.

<a id="mts-tag-001"></a>**MTS-TAG-001** `embed` is written
`{% embed "<input>" %}` for an input that is not a collection and
`{% embed "<input>", index: <expr> %}` for an item of a collection, `<expr>`
being an expression whose value is the item's 0-based position in the
collection's order. A processor MUST refuse, when the template is parsed, an
`embed` whose first argument is not a string literal naming an input the
manifest declares with `template.embed_unknown_input`, and anything after the
name but `index:` with `template.syntax`.

<a id="mts-tag-002"></a>**MTS-TAG-002** A processor MUST refuse, when the
template is parsed, an `embed` of a collection without `index:` and an
`embed` of an input that is not a collection with it, with
`template.embed_index`, and, when the tag is rendered, an `index:` whose
value is not an integer or is not the position of an item of the
collection, also with `template.embed_index`.

<a id="mts-tag-003"></a>**MTS-TAG-003** A processor MUST write the raw form
of the input, or of the item `index:` names, byte for byte — its line
endings included — except that a leading UTF-8 byte order mark is not
written; MUST write nothing for an input not supplied; MUST never escape
what `embed` writes; and MUST refuse a raw form that is not UTF-8 with
`template.invalid_utf8`.

> **Note.** Every form of [chapter 6](06-supplying-inputs.md) refuses a file
> that is not UTF-8, so a raw form a 1.0 processor accepts always is; the
> refusal keeps a later form from changing what `embed` may write. Embedding
> supplied data unescaped, in an HTML output as anywhere else, is the
> author's explicit choice: the bytes are written as the supplier wrote
> them, markup included.

<a id="mts-tag-004"></a>**MTS-TAG-004** An input, or an item, that a
`--set`, a `--set-json` or an `--inputs-json` member supplied or changed has
no raw form. A processor MUST refuse an `embed` of a raw form that does not
exist with `input.raw_unavailable`, located at the tag and naming the input
and, for an item, its `index`: before anything renders, for every `embed`
whose name and index are literals — whether or not its branch would run —
and when the tag is rendered, for an `embed` whose index is computed.

## 8.3 `asset`

`{% asset "<path>" %}` writes where the render copies an asset, relative to
the file being rendered, so a page links to a stylesheet or an image the
same render writes; `as: "contents"` writes the asset's text, and
`as: "base64"` its bytes in base64, for a `data:` URI. Its path is an entry
beneath `assets/`, named as a partial is named beneath `templates/`. What
it writes is never escaped.

<a id="mts-tag-012"></a>**MTS-TAG-012** The `asset` tag MUST take, first,
the asset's path as a string literal, and then at most the keyword
argument `as:` with one of the string literals `"path"`, `"contents"` and
`"base64"`, `"path"` when it is not given; a processor MUST refuse any
other `asset` tag with `template.syntax` when the template is parsed.

<a id="mts-tag-013"></a>**MTS-TAG-013** A processor MUST refuse an `asset`
tag whose path is not the path of a file beneath `assets/` — empty, with a
leading `/`, a `\`, a `.` or `..` segment or an empty segment, or naming
no entry, matched byte for byte — with `template.asset_not_found` when the
template is parsed, rendering nothing, and MUST NOT read an asset from
anywhere but the package.

<a id="mts-tag-014"></a>**MTS-TAG-014** An `asset` tag with `as: "path"`
MUST write the path of the one file the render writes from that asset in
the tree of the file being rendered, relative to the directory of the file
being rendered: `../` for each of its directories that the asset's path
does not share, in order from the root, followed by the rest of the
asset's path, with `/` separators.

<a id="mts-tag-015"></a>**MTS-TAG-015** A processor MUST refuse an `asset`
tag with `as: "path"` whose asset the render writes to no file with
`template.asset_not_written`; one whose asset it writes to more than one
file of the tree of the file being rendered — the output directory, or the
one container the file is an entry of — with `template.asset_ambiguous`;
and one whose asset it writes only in another tree with
`template.asset_tree`.

<a id="mts-tag-016"></a>**MTS-TAG-016** An `asset` tag with
`as: "contents"` MUST write the asset's bytes unchanged but for a leading
byte order mark, which it removes; a processor MUST refuse one whose bytes
are not UTF-8 with `template.asset_not_text`.

<a id="mts-tag-017"></a>**MTS-TAG-017** An `asset` tag with `as: "base64"`
MUST write the asset's bytes in the base64 encoding of
[RFC 4648](A-references.md#rfc4648) section 4, padded, with no line break.

## 8.4 `provenance`

`provenance` writes one line recording where an output came from, in the
comment syntax of the file it is written into, holding only the fields the
author names. It records nothing of the host: the render time only when the
caller gives one.

<a id="mts-tag-005"></a>**MTS-TAG-005** `provenance` is written
`{% provenance <part> … %}`, optionally followed by `style: <expr>`, each
part a bare word of the closed set `package`, `version`, `package_digest`,
`inputs`, `tool` and `time`, or `all` for every one of them. A processor
MUST refuse, when the template is parsed, a `provenance` with no part, with
a word outside the set or a keyword other than `style:`, or with a part
named twice — `all` names every part, so `all` with another part names it
twice — with `template.provenance_argument`.

<a id="mts-tag-006"></a>**MTS-TAG-006** A processor MUST write the fields of
the parts named, in this order whatever the order of the parts:
`package=<name>`; `version=<version>`, the package's own;
`package_digest=sha256:<hex>`, the package digest
([chapter 2](02-container.md)); for `inputs`, `input.<name>=<digest>` for
every declared input in byte order of the names, the digest being
`sha256:` and the lower-case hexadecimal SHA-256 of the RFC 8785
serialization of the input's value after defaults, or `absent` for an input
not supplied; `tool=<implementation>/<version>`, the processor's name and
version (`metais/` and the version `metais version` prints); and
`time=<time>`, the render time in UTC as templates see `now`
([chapter 7](07-template-environment.md)), written only when the caller gave
one. It MUST write no other field.

<a id="mts-tag-007"></a>**MTS-TAG-007** A processor MUST write the line as
`metais-provenance` followed by each field, each preceded by one U+0020, and
MUST write each byte of a value outside `A`–`Z`, `a`–`z`, `0`–`9`, `.`, `_`,
`+`, `:`, `/`, `@` and `-` as `%` and its two upper-case hexadecimal digits;
in the `html` style it MUST also write a `-` that follows a `-` in the value
as `%2D`, so that no value can end an XML comment.

<a id="mts-tag-008"></a>**MTS-TAG-008** A processor MUST write the line in
the style chosen, with no line feed after it and nothing else around it:

| Style | Written as |
|---|---|
| `html` | `<!-- metais-provenance … -->` |
| `hash` | `# metais-provenance …` |
| `slash` | `// metais-provenance …` |
| `block` | `/* metais-provenance … */` |
| `dash` | `-- metais-provenance …` |
| `percent` | `% metais-provenance …` |
| `semicolon` | `; metais-provenance …` |
| `json` | the fields as an RFC 8785 object: `package`, `version`, `package_digest`, `tool` and `time` as strings, and for `inputs` an `inputs` object of each declared input's name to its digest, or `null` |

<a id="mts-tag-009"></a>**MTS-TAG-009** A processor MUST choose the style
`style:` gives when the tag has it, and otherwise the style of the kind of
the file being rendered ([chapter 9](09-outputs.md)): `html` for `html`,
`xml` and `markdown`; `hash` for `yaml` and `hash`; `slash` for `slash`;
`block` for `css`; `dash` for `sql`; and `json` for `json`. The `percent` and
`semicolon` styles are chosen only by `style:`.

<a id="mts-tag-010"></a>**MTS-TAG-010** A processor MUST refuse a
`provenance` without `style:` in a file of kind `csv`, whose format has no
comments, or of kind `text`, whose comment syntax only `style:` can name,
and a `style:` whose value is not one of the styles of
[MTS-TAG-008](#mts-tag-008), with `template.provenance_style`: when the
template is parsed for a `style:` written as a literal, and otherwise when
the tag is rendered.

> **Note.** The tag is placed by the author, and writes its line where it
> stands: after an XML declaration, a shebang or a front-matter block that
> must come first, if the author puts it there.

## 8.5 Where the tags may stand

`embed`, `asset` and `provenance` write into the file a template renders. A
`path` ([chapter 9](09-outputs.md)) computes a file's name, and none of them
has a place in one.

<a id="mts-tag-011"></a>**MTS-TAG-011** An output declaration's `path` MUST
NOT hold an `embed`, an `asset` or a `provenance` tag, and a processor MUST
refuse one there, whatever its arguments, with `template.tag_context` naming
the tag, located at the field's manifest pointer and giving the column
within the expression.


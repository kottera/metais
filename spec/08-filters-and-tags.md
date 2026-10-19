# 8. Filters and tags

**Normative.** Areas: FLT, TAG.

METAIS adds a catalog of filters and tags to the Liquid dialect of chapter 7,
each defined to the byte so that every conforming processor renders it alike.
This chapter specifies the filter catalog, and the `embed`, `asset` and
`provenance` tags, with the comment syntax `provenance` writes for each kind
of output.

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


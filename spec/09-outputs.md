# 9. Outputs

**Normative.** Areas: OUT.

The manifest declares the files a render produces. This chapter specifies the
`outputs` section, path templates, outputs repeated over a collection with
`for_each` and conditional on `when`, asset copies, and the shorthand that
mirrors the templates tree into the output.

## 9.1 Template outputs

A template output renders one file from one template under `templates/`.
Its `path` names the file, relative to the root of the output tree; a path
that holds no Liquid markup is written as it is.

<a id="mts-out-001"></a>**MTS-OUT-001** A processor MUST write a template
output to the path its declaration's `path` gives and, when the declaration
omits `path`, to the template's path relative to `templates/` with a final
`.liquid` removed: `ci/workflow.yml.liquid` is written to `ci/workflow.yml`,
and `notes.txt` to `notes.txt`.

<a id="mts-out-002"></a>**MTS-OUT-002** A processor writing to a file system
with POSIX modes MUST write a template output whose declaration has
`executable: true` with its owner's execute bit set, mode `0755` less the
process's umask, and every other output without it, mode `0644` less the
umask.

## 9.2 Path templates

Every `path` is a Liquid template. Most paths are plain text, which renders
as itself; a path that holds Liquid markup computes its file's name from the
inputs, and a value holding `/` makes subdirectories. A path written in YAML
that begins with `{` must be quoted.

<a id="mts-out-003"></a>**MTS-OUT-003** A processor MUST render every
`path` as a template of the dialect of
[chapter 7](07-template-environment.md), with the globals §7.7.9 gives a
path, and MUST NOT escape it, whatever the declaration's `escape`; the
rendered string MUST be the path exactly, neither trimmed nor normalized
nor rewritten, and is then held to the portable-path rules
([chapter 11](11-rendering.md)).

<a id="mts-out-004"></a>**MTS-OUT-004** An error a declaration's `path`,
`when` or `for_each` meets when it is evaluated — an undefined variable or
property, a filter given an input it refuses, a division by zero, as a
template meets them ([chapter 7](07-template-environment.md)) — MUST be
refused with the code a template's error has, located at the field's
manifest pointer and giving the column within the expression; that
declaration MUST plan no file, and every other declaration MUST still be
planned, so that every such finding is reported in one render. An error of
their parse is the package's (MTS-MAN-018).

## 9.3 Conditions and fan-out

A declaration's `when` decides whether it writes its file; its `for_each`
repeats it, once for each element of an array. Each repetition is an
evaluation, with the element bound to `item` and its position to `fan_out`
([chapter 7](07-template-environment.md)): its `when` decides whether that
element writes a file, and its `path` names the file.

<a id="mts-out-005"></a>**MTS-OUT-005** A declaration's `when` MUST be a
Liquid condition: exactly what may follow `if` in `{% if … %}`, true unless
its value is nil or `false`. A processor MUST write no file for a
declaration whose `when` is false, and MUST treat a declaration without
`when` as always holding.

<a id="mts-out-006"></a>**MTS-OUT-006** A declaration's `for_each` MUST be
a Liquid value expression: exactly what may appear inside `{{ … }}`, a
lookup or a literal followed by any filters. When its value is an array, a
processor MUST evaluate the declaration once per element, in array order,
with the element bound to `item`; when it is nil, as an optional input that
is absent is, it MUST evaluate the declaration no time and report nothing.

<a id="mts-out-007"></a>**MTS-OUT-007** A processor MUST refuse a
`for_each` whose value is a string, a number, a boolean or an object with
`output.expression.for_each_type`, naming the declaration and the type
found, and plan no file of that declaration.

<a id="mts-out-008"></a>**MTS-OUT-008** A processor MUST evaluate the
`when` of a `for_each` declaration once per element, with that element's
`item` and `fan_out` bound, and write no file for an element whose `when`
is false; so an author drops elements either in the `for_each` expression
or in `when`.

<a id="mts-out-009"></a>**MTS-OUT-009** A path from a fan-out that
breaks the portable-path rules ([chapter 11](11-rendering.md)) MUST be
refused with its `output.path.*` code naming the declaration, the
rendered path and, as `item`, the 0-based index of the evaluation that
rendered it.

<a id="mts-out-010"></a>**MTS-OUT-010** Every element of a `for_each`
array MUST count as one iteration of `loop-iterations`
([chapter 14](14-security.md)), with every `for`, `tablerow` and
`render … for` of the render; a `for_each` whose array takes the render
beyond it MUST be refused with `limit.exceeded.template` before its first
evaluation, located at the `for_each`.

<a id="mts-out-011"></a>**MTS-OUT-011** `outputs`
([chapter 14](14-security.md)) MUST count every file every evaluation of a
declaration plans. When the first file beyond it is a fan-out's, the
processor MUST refuse the render with `limit.exceeded.output`, naming that
declaration and located at it, and need evaluate no further element of any
`for_each`; `observed` is then given only when every file was counted.
## 9.4 The output kind

Every file a template renders has a kind, which says what syntax the file is
written in: it chooses the comment syntax `provenance` writes
([chapter 8](08-filters-and-tags.md)). The kind is not escaping, which an
output chooses on its own.

<a id="mts-out-012"></a>**MTS-OUT-012** A processor MUST give every file a
template output renders the kind its declaration's `kind` names when the
declaration has one, for every file the declaration renders, and otherwise
the kind this table gives for the file's rendered path, matching the path's
final extension without regard to ASCII case, then the whole file name, and
giving `text` to a path neither matches. A later minor may add rows and
never changes one.

| Kind | Extensions | File names |
|---|---|---|
| `html` | `.html` `.htm` | |
| `xml` | `.xml` `.xhtml` `.svg` `.rels` `.opf` `.ncx` `.xsd` `.xsl` `.xslt` `.plist` | |
| `markdown` | `.md` `.markdown` | |
| `json` | `.json` `.geojson` `.webmanifest` | |
| `yaml` | `.yaml` `.yml` | |
| `csv` | `.csv` | |
| `css` | `.css` `.scss` `.less` | |
| `hash` | `.sh` `.bash` `.zsh` `.py` `.rb` `.pl` `.r` `.toml` `.ps1` `.properties` `.tf` | `Makefile` `Dockerfile` `Containerfile` `.gitignore` `.gitattributes` `.dockerignore` `.env` |
| `slash` | `.js` `.mjs` `.cjs` `.ts` `.tsx` `.jsx` `.go` `.c` `.h` `.cc` `.cpp` `.hpp` `.java` `.kt` `.kts` `.swift` `.rs` `.cs` `.scala` `.dart` `.jsonc` `.proto` `.groovy` `.gradle` | |
| `sql` | `.sql` | |
| `text` | anything else | |

> **Note.** `[Content_Types].xml`, the part list of an Office Open XML
> document, is `xml` by its extension. A file name is matched exactly:
> `makefile` is `text`.

## 9.5 Asset copies

An asset declaration copies files from `assets/` into the output tree as they
are, byte for byte: assets are the way binary files reach the output. Its
`asset` names a file or a directory beneath `assets/`, `.` naming `assets/`
itself; a name that matches nothing is refused when the package is opened
([MTS-MAN-017](03-manifest.md#mts-man-017)). Directories are not entries:
an asset directory writes its files, and a package holds no empty directory
([chapter 2](02-container.md)).

<a id="mts-out-013"></a>**MTS-OUT-013** A processor MUST write an asset
declaration whose `asset` names a file beneath `assets/` to the path its
`path` gives and, when it omits `path`, to the asset's path relative to
`assets/`; and MUST write it with exactly the bytes of the asset's entry,
never rendering, escaping or converting them, whatever they hold. Every
file an asset declaration writes MUST have the mode
[MTS-OUT-002](#mts-out-002) gives a template output with the declaration's
`executable`.

<a id="mts-out-014"></a>**MTS-OUT-014** A processor MUST write an asset
declaration whose `asset` names a directory beneath `assets/`, or `.`, by
writing every file beneath that directory, at any depth, at its path
relative to the directory beneath the destination directory its `path`
gives, `.` being the root of the output tree, and, when it omits `path`,
beneath the directory's own path relative to `assets/`; each file with
exactly the bytes of its entry.

<a id="mts-out-015"></a>**MTS-OUT-015** A processor MUST count the size of
every file an asset declaration writes, with the bytes of every other file
the render writes, against the `output-size` and `output-total` limits
before it writes anything, and MUST refuse a render in which one breaches
either with `limit.exceeded.output`, located at the file's path.

## 9.6 The mirror

A mirror renders a directory of templates into the output tree, each
template to a file of the same relative path, so a tree of templates needs
no declaration per file. A template whose name, or the name of a directory
above it beneath the mirror's root, begins with `_` is a partial and is not
mirrored; a template a `template` field names is rendered by that output
alone. File names are not templates: a name derived from the inputs needs a
template output.

<a id="mts-out-016"></a>**MTS-OUT-016** A processor MUST render a mirror by
rendering every template beneath `templates/<dir>`, `<dir>` being its
`mirror` and `.` naming `templates/` itself, to its path relative to
`templates/<dir>`, with a final `.liquid` removed from its file name,
beneath the destination directory its `path` gives, `.` being the root of
the output tree, and, when it omits `path`, beneath `<dir>`.

<a id="mts-out-017"></a>**MTS-OUT-017** A processor MUST NOT mirror a
template whose file name, or the name of any directory between the
mirror's directory and it, begins with `_`, nor a template that a
`template` field of any declaration of the manifest names.

<a id="mts-out-018"></a>**MTS-OUT-018** A processor MUST refuse a mirror
that would render a template whose file name is exactly `.liquid` with
`output.declaration.empty_name`.

<a id="mts-out-019"></a>**MTS-OUT-019** A processor MUST report a mirror
that renders no file as `output.declaration.empty_mirror`, a warning.

<a id="mts-out-020"></a>**MTS-OUT-020** A processor MUST hold every file a
mirror renders to the path rules of [chapter 11](11-rendering.md) with
every other file of the output tree, refusing two files at the same path
with `output.path.duplicate` whichever declarations write them.

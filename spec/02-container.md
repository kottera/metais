# 2. Container

**Normative.** Areas: PKG.

A packed package is a zip archive in a canonical profile, in which every header
field is fixed except an entry's compressed data, its checksum and its sizes.
This chapter specifies that profile, the names an entry may carry, the closed
set of top-level names a package lays out, the `.compiled/` files a packed
package carries, and the digest of every entry and of the package as a whole,
which a processor verifies before it does anything else with a package.

## 2.1 The two forms of a package

A package is given to a processor either packed, as one file, or as a source
directory, the editable tree an author works in. The two hold the same
package: the entries of a source directory are exactly the files a packed
package made from it holds outside `.compiled/`.

<a id="mts-pkg-001"></a>**MTS-PKG-001** A processor MUST read a path that
names a regular file as a packed package and a path that names a directory as
a source directory, whatever the name or extension of either, and MUST refuse
a path that names nothing with `package.not_found`.

<a id="mts-pkg-002"></a>**MTS-PKG-002** *(not observable)* A packed package's
file name SHOULD end in `.metais`, in lower case.

The media type of a packed package is `application/vnd.metais+zip`. It is a
vendor-tree type with the `+zip` suffix ([RFC 6838](A-references.md#rfc6838),
[RFC 6839](A-references.md#rfc6839)); its registration with IANA is not part
of this revision. A file is a candidate packed package when it begins with the
bytes `50 4B 03 04` and its first entry's name, at offset 30, begins with
`.compiled/`, which every packed package's first entry does (§2.4); that is a
sufficient rule for a file-type detector, and only §2.6 decides whether the
file is a package.

## 2.2 The zip profile

The profile is a subset of the zip format of [APPNOTE](A-references.md#appnote).
Reading and writing use one profile: a packed package is accepted only if every
field the profile fixes has its fixed value. A conforming packed package can
only be written by a packer, because its `.compiled/` files must match its
sources (§2.5), so the strictness costs an author nothing, and every fixed
field is a place where a hostile package cannot hide a disagreement between
two readers.

### 2.2.1 Structure

<a id="mts-pkg-003"></a>**MTS-PKG-003** A packed package MUST be a sequence of
entries, then a central directory, then one end-of-central-directory record,
with no other byte: the first local file header at offset 0; each entry a
local file header immediately followed by its file data, and the next local
header at the byte after them; the central directory, holding one header per
entry in the entries' order, starting at the byte after the last entry's data
and ending where the end record starts; and the end record as the file's last
22 bytes, with both disk numbers 0, both entry counts the number of entries,
the central directory's real size and offset, and comment length 0. A
processor MUST refuse a file that does not end in an end record with
`package.container.not_zip`, bytes before the first local header with
`package.container.prepended_data`, bytes after the end record with
`package.container.trailing_data`, an archive comment with
`package.container.archive_comment`, and a gap or an overlap between two
entries or between the last entry and the central directory, and a central
directory header that does not point at the entry it follows, with
`package.container.overlap_or_gap`.

<a id="mts-pkg-004"></a>**MTS-PKG-004** A packed package MUST NOT hold a
zip64 end-of-central-directory record or locator, a zip64 extra field, or a
field holding the zip64 sentinel `0xFFFF` or `0xFFFFFFFF`, and a processor
MUST refuse one that does with `package.container.zip64`.

The hard ceilings that follow from this are 65,534 entries and 0xFFFFFFFE
bytes for the package file, every size and every offset; the limits of
[chapter 14](14-security.md) lie inside them.

<a id="mts-pkg-005"></a>**MTS-PKG-005** A processor MUST refuse a packed
package in whose final 65,557 bytes the end-record signature `50 4B 05 06`
appears anywhere but at the start of its end record, with
`package.container.second_end_record`: 65,557 bytes is the largest span an end
record and its comment can occupy, so no reader that scans for an end record
can find a different one.

<a id="mts-pkg-045"></a>**MTS-PKG-045** A processor MUST refuse an end record
either of whose disk numbers is not 0, or whose count of entries on this disk
differs from its total count of entries, with
`package.container.field_value` naming the record and the field; and an end
record whose total count of entries is not the number of central directory
headers its central directory holds with `package.container.overlap_or_gap`.

### 2.2.2 Header fields

Every entry's local file header has these values:

| Field | Value |
|---|---|
| version needed to extract | 10 (`0x000A`) for a stored entry, 20 (`0x0014`) for a deflated one |
| general purpose bit flag | `0x0800`: bit 11 (UTF-8) set, every other bit clear |
| compression method | 0 (stored) or 8 (deflated) |
| last mod file time | `0x0000` |
| last mod file date | `0x0021` (1980-01-01) |
| crc-32 | the CRC-32 of the uncompressed content |
| compressed size | the size of the file data |
| uncompressed size | the size of the content |
| file name length | the name's length in bytes, at least 1 |
| extra field length | 0 |

Every entry's central directory header has the same values as its local
header in every field the two share, the name included, and these:

| Field | Value |
|---|---|
| version made by | `0x0314`: Unix as the attribute host, specification 2.0 |
| file comment length | 0 |
| disk number start | 0 |
| internal file attributes | 0 |
| external file attributes | `0x81A40000`: a regular file of mode 0644 |
| relative offset of local header | the entry's actual offset |

An entry carries no file mode of its own beyond this fixed one, and an empty
entry is stored.

<a id="mts-pkg-006"></a>**MTS-PKG-006** Every field of every local file
header and central directory header of a packed package MUST have the value
§2.2.2's tables give it, and a processor MUST refuse a header field with
another value with `package.container.field_value`, naming the record and the
field, except where [MTS-PKG-007](#mts-pkg-007) or
[MTS-PKG-008](#mts-pkg-008) names another code. A symbolic link's, a
directory's, a device's and a setuid file's modes in the external file
attributes are such values.

<a id="mts-pkg-007"></a>**MTS-PKG-007** A processor MUST refuse a packed
package an entry of which is encrypted — general purpose bit 0, 6 or 13 set,
or compression method 99 — with `package.container.encrypted`; one whose
compression method is neither 0 nor 8 with `package.container.method`; one
with general purpose bit 3 set, which announces a data descriptor, with
`package.container.data_descriptor`; and one with any other general purpose
bit set, or bit 11 clear, with `package.container.flags`.

<a id="mts-pkg-008"></a>**MTS-PKG-008** A processor MUST refuse a packed
package a central directory header of which has an extra field with
`package.container.extra_field`.

<a id="mts-pkg-009"></a>**MTS-PKG-009** A processor MUST refuse a packed
package a local file header of which differs from its central directory header
in any field the two share, with `package.container.header_mismatch` naming
the field.

<a id="mts-pkg-010"></a>**MTS-PKG-010** The entries of a packed package MUST
be in ascending order of their names' UTF-8 bytes, and a processor MUST refuse
one whose entries are not with `package.container.entry_order`.

### 2.2.3 File data

<a id="mts-pkg-011"></a>**MTS-PKG-011** A stored entry's compressed size MUST
equal its uncompressed size, and a deflated entry's file data MUST be one raw
DEFLATE stream ([RFC 1951](A-references.md#rfc1951)) that decodes to exactly
its uncompressed size and whose final block ends exactly at its compressed
size. A processor MUST refuse an entry whose content is longer or shorter than
its declared size with `package.container.size_mismatch`, an entry whose
stream is not valid DEFLATE, ends before its data does, or is followed by
other bytes with `package.container.deflate_stream`, and an entry whose
content's CRC-32 is not the recorded one, whatever the recorded value, zero
included, with `package.container.crc_mismatch`.

> **Note.** The profile fixes the method and the stream's validity, never the
> compressed bytes: two conforming packers may compress the same content
> differently, and the package digest (§2.6) does not change.

## 2.3 Entry names

An entry name, a source directory's file path, an output path and a container
entry path obey one portable-path rule, stated for outputs in
[chapter 11](11-rendering.md); this section states it for entries. Each rule
below holds for the name of a packed package's entry, and for the path of a
source directory's file, relative to the directory with `/` as its separator;
a processor refuses a source directory that breaks one with the same code.

<a id="mts-pkg-012"></a>**MTS-PKG-012** An entry name MUST be valid UTF-8
with no byte order mark, and a processor MUST refuse one that is not with
`package.name.encoding`. General purpose bit 11 is always set (§2.2.2), so no
name is read as CP437.

<a id="mts-pkg-013"></a>**MTS-PKG-013** An entry name MUST be in Unicode
Normalization Form C ([Unicode](A-references.md#unicode)), and a processor
MUST refuse one that is not with `package.name.not_nfc`; a name is never
normalized for it.

<a id="mts-pkg-014"></a>**MTS-PKG-014** An entry name MUST be one or more
segments separated by `/`, with no leading `/`, no empty segment, no `.` or
`..` segment, no `\` and no `:`, and a processor MUST refuse one that is not
with `package.name.shape`.

<a id="mts-pkg-015"></a>**MTS-PKG-015** Every code point of an entry name
MUST be U+0020 or belong to one of the general categories L, M, N, P or S as
Unicode 15.0.0 assigns them, and MUST be none of `<`, `>`, `"`, `|`, `?`, `*`
and no code point in U+FFF0 to U+FFFF; a processor MUST refuse a name that
holds another with `package.name.character`. This refuses controls, format
characters such as a bidirectional override or a zero-width joiner,
private-use and unassigned code points, noncharacters, and every separator
but U+0020.

<a id="mts-pkg-016"></a>**MTS-PKG-016** A segment of an entry name MUST NOT be
a Windows device name (`con`, `prn`, `aux`, `nul`, `com0` to `com9`, `lpt0` to
`lpt9`, and the same followed by `.` and anything, compared without regard to
ASCII case), a `.git` segment in any spelling Git treats as `.git`, a segment
ending in `.` or U+0020, or a segment longer than 255 bytes, and a processor
MUST refuse a name that has one with `package.name.not_portable`.

<a id="mts-pkg-017"></a>**MTS-PKG-017** In the set of every entry name and
every directory prefix of every entry name (`a/b/c` contributes `a`, `a/b` and
`a/b/c`), no two distinct members MUST be canonical caseless matches under
full case folding (Unicode 15.0.0, CaseFolding.txt statuses C and F). A
processor MUST refuse two entries with one name with `package.name.duplicate`,
two names or prefixes that differ only in case or by full case folding, such
as `straße.md` beside `STRASSE.md`, with `package.name.case_collision`, and a
name that is also, or caselessly matches, a directory prefix of another, such
as `a` beside `a/b`, with `package.name.file_directory_conflict`.

<a id="mts-pkg-018"></a>**MTS-PKG-018** A packed package MUST NOT hold a
directory entry, and a processor MUST refuse an entry whose name ends in `/`
with `package.name.directory_entry`. Directories exist only as prefixes of
entry names.

The limits `path-length` and `path-depth` ([chapter 14](14-security.md)) bound
every entry name too.

## 2.4 The layout

The top level of a package holds only these names:

| Name | Kind | Holds |
|---|---|---|
| `metais.yaml` | file | the manifest ([chapter 3](03-manifest.md)) |
| `inputs/` | directory | the input schemas and meaning documents ([chapter 5](05-data-specification.md)) |
| `templates/` | directory | the templates and partials |
| `references/` | directory | background material, any files |
| `assets/` | directory | static files an output copies or a template includes |
| `examples/` | directory | `examples/<name>/inputs/` and `examples/<name>/expected/` |
| `.compiled/` | directory | in a packed package only: the compiled files (§2.5) |

A directory exists when at least one entry name begins with it. Beneath the
top level any name §2.3 admits is allowed, a name beginning with `.` included.

<a id="mts-pkg-019"></a>**MTS-PKG-019** A packed package MUST hold no
top-level name but those of §2.4's table, and a processor MUST refuse one that
holds another with `package.layout.unknown_top_level`, once the package's
manifest has declared a specification version the processor implements.

<a id="mts-pkg-020"></a>**MTS-PKG-020** A package MUST hold `metais.yaml` at
its top level, and a processor MUST refuse a package without it with
`package.layout.missing_manifest`.

<a id="mts-pkg-021"></a>**MTS-PKG-021** In a source directory, a processor
MUST ignore a top-level name beginning with `.` other than `.compiled`, and a
top-level regular file whose name ends in `.metais`, and MUST treat every other
top-level name §2.4's table does not hold as no part of the package: none of
them refuses the source directory, and none is an entry of it.

<a id="mts-pkg-022"></a>**MTS-PKG-022** A processor MUST refuse a source
directory holding `.compiled` at its top level with
`package.layout.compiled_in_source`: only a packer writes `.compiled/`, and a
copy beside the sources could be read as either.

<a id="mts-pkg-023"></a>**MTS-PKG-023** A processor MUST refuse a source
directory beneath whose `metais.yaml`, `inputs/`, `templates/`, `references/`,
`assets/` or `examples/` lies a symbolic link, named pipe, socket, device or
other file that is not a regular file or a directory, with
`package.layout.non_regular_file`, without following or opening it, and one
beneath which lies an empty directory, which a packed package cannot carry,
with `package.layout.empty_directory`.

## 2.5 The compiled files

A packed package carries, under `.compiled/`, files a consumer with only a
zip reader and a JSON parser can use. Each but the digest file is a
deterministic function of the package's other entries and its declared
specification version, so each is a proof of what the sources say rather than
a claim.

At specification version 1.0, `.compiled/` holds exactly:

| File | Holds |
|---|---|
| `.compiled/digests.json` | the digest of every other entry, and the package digest (§2.6) |
| `.compiled/manifest.json` | the manifest in canonical JSON ([chapter 3](03-manifest.md)) |
| `.compiled/schema.json` | the combined input schema ([chapter 5](05-data-specification.md)) |
| `.compiled/references.json` | the references index (§2.5.1) |

Because `.` sorts before every letter and digit and the top level is closed,
a packed package's first entry is always one of these.

<a id="mts-pkg-024"></a>**MTS-PKG-024** A packed package MUST hold every file
of §2.5's table and no other file beneath `.compiled/`. A processor MUST
refuse a package that lacks one with `package.compiled.missing`, before it
reads any of them, and one that holds another, once its manifest has declared a
specification version the processor implements, with
`package.compiled.unknown_file`.

<a id="mts-pkg-025"></a>**MTS-PKG-025** A processor MUST derive
`.compiled/manifest.json`, `.compiled/schema.json` and
`.compiled/references.json` from the package's other entries as this
specification defines them for the declared specification version, and MUST
refuse a packed package any of whose stored compiled files is not byte for
byte the derived one with `package.compiled.mismatch`, naming each that
differs.

### 2.5.1 The references index

<a id="mts-pkg-026"></a>**MTS-PKG-026** `.compiled/references.json` MUST be
the canonical JSON ([chapter 3](03-manifest.md)) of an object with one member,
`references`: an array with one object per entry whose name begins
`references/`, in ascending order of the names' UTF-8 bytes, and empty when
there is none. Each object has exactly these members: `path`, the entry name;
`size`, its uncompressed size in bytes; `text`, `true` when the entry is valid
UTF-8 and holds no NUL byte and `false` otherwise; `lines`, for a text entry
only, its number of LF bytes, plus one when it is not empty and does not end in
an LF; and `title`, for a text entry whose name ends in `.md` or `.markdown`
(ASCII letters compared without regard to case), the plain text of its first
level-1 heading as [CommonMark 0.31.2](A-references.md#commonmark) defines a
heading, a leading frontmatter block skipped, and otherwise `null`.

The plain text of a heading is its inline content with text and code spans
concatenated, markup removed, entities and backslash escapes decoded, and each
line break of a setext heading a single space.

## 2.6 Digests and verification

### 2.6.1 The digest file and the package digest

The digest file `.compiled/digests.json` is a JSON object with exactly three
members: `algorithm`, the string `sha256`; `entries`, an object with one
member per entry of the package other than the digest file itself, named by
the entry name and valued by the 64 lowercase hexadecimal digits of the
SHA-256 of the entry's uncompressed content; and `package`, the 64 lowercase
hexadecimal digits of the package digest.

The **digest list** of a package is the UTF-8 text of one line per member of
`entries`, in ascending order of the names' UTF-8 bytes, each line the 64
hexadecimal digits, two U+0020 spaces, the name and an LF. It is exactly GNU
`sha256sum`'s untagged output for those files, since §2.3 refuses every
character that would make `sha256sum` escape a line. The **package digest** is
the SHA-256 of the digest list, and is presented as `sha256:` followed by its
64 lowercase hexadecimal digits.

> **Note.** The package digest depends only on the entries' names and
> contents: not on compression, entry order or the program that packed them.
> Anyone can check a package with standard tools: extract it, run `sha256sum`
> over every file but `.compiled/digests.json` in byte order of their names,
> and hash the output.

<a id="mts-pkg-027"></a>**MTS-PKG-027** `.compiled/digests.json` MUST be the
canonical JSON ([chapter 3](03-manifest.md)) of an object of §2.6.1's form,
and a processor MUST refuse a digest file that is not one JSON object, does
not have exactly that form, or is not byte for byte the canonical JSON of its
value with `package.digest.not_canonical`.

<a id="mts-pkg-028"></a>**MTS-PKG-028** The members of the digest file's
`entries` MUST be exactly the names of the package's entries other than the
digest file, and a processor MUST refuse each entry the file does not name and
each name the file lists that is no entry with
`package.digest.entries_mismatch`.

<a id="mts-pkg-029"></a>**MTS-PKG-029** Each member of the digest file's
`entries` MUST be the SHA-256 of its entry's uncompressed content, and a
processor MUST refuse each entry whose content is not the one the file names
with `package.digest.mismatch`.

<a id="mts-pkg-030"></a>**MTS-PKG-030** The digest file's `package` MUST be
the digest of the digest list of its `entries`, and a processor MUST refuse a
digest file whose `package` is not with `package.digest.package_mismatch`.

<a id="mts-pkg-031"></a>**MTS-PKG-031** The package digest of a source
directory MUST be the package digest of the packed package a packer writes
from it: the digest of the digest list of its files and of the three compiled
files derived from them, which a processor computes in memory.

### 2.6.2 Verification at open

<a id="mts-pkg-032"></a>**MTS-PKG-032** Before it reads or reports any of a
packed package's content, a processor MUST verify the whole package in this
order, each step bounded by [chapter 14](14-security.md)'s limits before the
bytes it bounds are read: (1) the file's size; (2) the end record, the entry
count and the final 65,557 bytes; (3) every central directory header, its name
and the names together, and the declared sizes and ratios; (4) every local
header, walking from offset 0; (5) every entry's data, decoded once, its size,
CRC-32 and SHA-256 established; (6) the digest file, every digest and the
package digest; (7) the manifest ([chapter 3](03-manifest.md)), the layout for
its declared version, every input schema ([chapter 5](05-data-specification.md))
and the compiled files. A processor MUST report every finding of the first step
that finds any, and none of a later step: a later step's findings would be
about a package the earlier one has already shown to be something else.

<a id="mts-pkg-033"></a>**MTS-PKG-033** A processor MUST check a source
directory's names and layout as §2.3 and §2.4 require when it reads the
directory, report every finding of that reading at once, and then process its
manifest, its input schemas and its compiled files as step 7 of
[MTS-PKG-032](#mts-pkg-032) does, computing the compiled files in memory.

> **Note.** No option skips a step. A package that opens is the package its
> digest names, and a person who needs the bytes of a damaged package uses a
> general-purpose zip tool, outside METAIS.

## 2.7 Writing a package

A packer turns a source directory into a packed package
([chapter 13](13-authoring.md) says when it may). It writes the profile of
§2.2 and nothing of its host, so the package's bytes are a function of its
entries alone. The compressed bytes of a deflated entry are the encoder's
own: two implementations, or two builds of one, may write different bytes
for one source directory, and their packages are still the same package,
with the same entries and the same package digest.

<a id="mts-pkg-034"></a>**MTS-PKG-034** A packer MUST write a packed
package that §2.2 to §2.6 accept: every field the profile fixes holding its
fixed value, every entry's CRC-32 and sizes in its local and central
headers, no data descriptor, and the digest file and the compiled files
of §2.5 and §2.6.

<a id="mts-pkg-035"></a>**MTS-PKG-035** A packer MUST write the entries in
ascending order of their names' UTF-8 bytes, so that
`.compiled/digests.json` is the first.

<a id="mts-pkg-036"></a>**MTS-PKG-036** A packer MUST store an entry whose
content is empty, and one that deflating does not make strictly shorter;
it MUST deflate every other entry, keeping the deflated data only when it
is strictly shorter than the content.

<a id="mts-pkg-037"></a>**MTS-PKG-037** A packer MUST write each source
entry's content byte for byte as the source directory holds it: it MUST
NOT change a line ending, a byte order mark, an encoding or anything else
of it.

<a id="mts-pkg-038"></a>**MTS-PKG-038** The package digest of a packed
package a packer writes MUST be the package digest of the source directory
it was written from ([MTS-PKG-031](#mts-pkg-031)), and the package MUST
open as [MTS-PKG-032](#mts-pkg-032) verifies one.

<a id="mts-pkg-039"></a>**MTS-PKG-039** The bytes of a packed package MUST
depend only on its entries' names and contents: never on a file's
modification time, permissions or owner, the order a directory lists its
names in, the file system's treatment of case, the platform, the locale,
the time zone or the clock, and never on the packer's version, which a
package does not record. One build of a packer MUST write the same bytes
for the same source directory every time and on every platform.

<a id="mts-pkg-040"></a>**MTS-PKG-040** A source directory unpacked from a
packed package ([chapter 13](13-authoring.md)) MUST pack to a package with
the same entries, the same content and the same package digest; packed by
the build that wrote the first package, to the same bytes.

# ADR-005: Outputs, transforms and rendering — declared outputs expand into one sorted file plan that is rendered, transformed and verified in memory before a single commit writes it; zip containers, CRLF and schema validation are the only transforms; every path obeys one portable rule on every platform

**Status:** Accepted 2026-10-18 by Ed Funnekotter; its proposed rulings are ratified in [RULINGS-v1](RULINGS-v1.md).
**Date:** 2026-10-18
**Deciders:** Ed Funnekotter
**Designs:** PRD-001 R8 (items 1–5), R9 (items 1–5), R10 (items 1–4), R12 (items 1–5); the output-count and output-size parts of R14 item 2
**Depends on:** ADR-001 (the layout's `templates/` and `assets/`; the deterministic zip writing rules, reused for containers); ADR-002 (the manifest's top level, which holds the `outputs` key, and its canonical JSON form); ADR-003 (the data model templates see; the JSON Schema dialect and the YAML profile, reused for output validation); ADR-004 (expression evaluation, template rendering, escaping, and the provenance syntax per output kind); ADR-006 (the finding model); ADR-008 (flag names and the JSON envelope); ADR-009 (resource limits and entry confinement); ADR-010 (the libraries and the toolchain)

## Context

### What the PRD requires

- **R8.** The manifest declares the outputs. Each names a template and a path,
  and the path is itself a template. An output may repeat over a collection
  (`for_each`) or be conditional (`when`). Assets are copied to declared
  paths. A shorthand mirrors the templates tree for authors who need no list.
- **R9.** Output transforms form a closed set, never a program: a zip
  container assembled from other outputs and assets (how OOXML, ODF and EPUB
  files are produced from templates of their XML parts), CRLF conversion,
  and validation against a schema the package carries. A failing output
  refuses the render and nothing is written. The tool's own containers are
  byte-identical across platforms and runs; conformance compares a container
  by its entries.
- **R10.** No code from a package runs, no network, nothing of the host is
  read. The same package and inputs give byte-identical outputs on every
  platform. UTF-8, line endings as written, defined iteration order, no
  locale or time-zone dependence, time only when the caller supplies it.
- **R12.** Every output path stays inside the output directory. Paths are
  portable: a name Windows cannot hold, or two paths that differ only in
  letter case, is refused on every platform. A non-empty output directory is
  refused unless the caller allows replacing files; the tool never deletes a
  file. A dry run lists every file with its size and digest. Standard output
  carries a deterministic archive of every output, or the bytes of one named
  output.

### Facts that constrain the design (verified 2026-10-18; sources in References)

**Windows file names** (Microsoft, *Naming Files, Paths, and Namespaces*):

- Reserved characters: `<` `>` `:` `"` `/` `\` `|` `?` `*`; the character
  with integer value zero; characters 1 through 31.
- Reserved names: CON, PRN, AUX, NUL, COM1–COM9, COM¹, COM², COM³, LPT1–LPT9,
  LPT¹, LPT², LPT³. "Also avoid these names followed immediately by an
  extension; for example, NUL.txt and NUL.tar.gz are both equivalent to NUL."
  The superscript digits ¹ ² ³ are recognized "as digits and treats them as
  valid parts of COM# and LPT# device names, making them reserved in every
  directory." The same page's NT-namespace section names a `COM0` device
  link.
- "Do not end a file or directory name with a space or a period."
- "Do not assume case sensitivity … consider the names OSCAR, Oscar, and
  oscar to be the same."
- MAX_PATH is 260 characters before Windows 10 version 1607; later versions
  lift it by opt-in. Go's `os` package on Windows prefixes long paths with
  `\\?\` unless the process can already use long paths (`fixLongPath` in
  `os/path_windows.go`), so the tool's own writes are not bound by MAX_PATH.

**EPUB 3.3 OCF** (W3C Recommendation):

- §4.2.3: file names do not use `/` `"` `*` `:` `<` `>` `?` `\` `|`, DEL,
  C0 (U+0000–U+001F), C1 (U+0080–U+009F), the Private Use Area
  (U+E000–U+F8FF), every noncharacter, Specials (U+FFF0–U+FFFF), and the
  Supplementary Private Use Areas A and B; a file name does not end with a
  full stop. "File names MUST NOT exceed 255 bytes"; paths do not exceed
  65535 bytes. "All file names within the same directory MUST be unique
  following Unicode canonical normalization [uax15] and then full case
  folding." File names SHOULD NOT contain U+0020.
- §4.3.2: only stored and Deflate entries; ZIP64 permitted, and used only
  when the content requires it; no ZIP encryption; names in UTF-8; no
  spanning; `version needed to extract` is 10, 20 or 45.
- §4.3.3: `mimetype` is the first file; its content is `application/epub+zip`
  in US-ASCII, with no leading or trailing padding or whitespace and no byte
  order mark; it is neither compressed nor encrypted; "EPUB creators MUST
  NOT include an extra field in its ZIP header."
- §4.2.6.1 and §4.2.6.3.1: a `META-INF` directory is required at the root,
  and `META-INF/container.xml` is required.

**OpenDocument 1.3 Part 2, Packages** (OASIS):

- §3.3: "The 'mimetype' file shall be the first file of the zip file." It is
  stored uncompressed, it does not use an extra field in its header, and its
  content is the ASCII media type of the document, equal to the manifest's
  `manifest:media-type` for `/` when that entry exists.
- §3.2: "All OpenDocument packages shall contain a file named
  'META-INF/manifest.xml'", which lists every file except `mimetype` and
  `META-INF/` files.
- §2.2.1: entries are stored or deflated. §4.3: directories have no file
  entries in the zip.

**Office Open XML packaging, ECMA-376 Part 2, 5th edition (2021),
equivalent to ISO/IEC 29500-2** (the PDF, read 2026-10-18):

- §7.3.3: "A mapping from an abstract package to a ZIP file shall only use
  ASCII ZIP item names"; non-ASCII characters of part names are
  percent-encoded (§7.3.4).
- §7.3.6: "ZIP-based packages shall not include encryption … shall not use
  compression algorithms except DEFLATE"; Annex C permits stored entries,
  data descriptors and ZIP64.
- §7.3.7: the Media Types stream is the item `[Content_Types].xml`.
- §6.2.2: part-name equivalence is ASCII case-insensitive; no segment ends
  with a dot; NFC and NFD forms of two part names should not collide.
- The standard states **no ordering requirement** for any item, including
  `[Content_Types].xml`.

**POSIX pax interchange format** (`pax` utility, POSIX.1-2024): a 512-octet
ustar header (name 100, prefix 155, mode, uid, gid, size, mtime, typeflag,
magic `ustar\0`, version `00`, uname 32, gname 32); `prefix` + `/` + `name`
forms the path; typeflag `x` carries extended records for the next file in
`"%d %s=%s\n"` form; the `path` record overrides name and prefix and is UTF-8;
the archive ends with two 512-octet zero blocks.

**Unicode.** D145 (Unicode Standard, Chapter 3, §3.13, Default Caseless
Matching): X is a canonical caseless match for Y if and only if
`NFD(toCasefold(NFD(X))) = NFD(toCasefold(NFD(Y)))`, with full case folding.
The Unicode stability policy guarantees normalization and case folding
stable across versions **only for strings of characters assigned in the
version used**. Go's `unicode` tables are Unicode 15.0.0 through go1.26 and
17.0.0 in go1.27.0; `golang.org/x/text` v0.43.0 carries both table sets for
`unicode/norm` and `cases`, the 15.0.0 files under `//go:build !go1.27` and
the 17.0.0 files under `//go:build go1.27`, so the compiling toolchain
selects them: under ADR-010's Go 1.26 every table is 15.0.0.

**Renaming.** Linux `rename(2)` replaces an existing target atomically, and
replaces an existing *empty directory*; `renameat2` with `RENAME_NOREPLACE`
(Linux 3.15) refuses an existing target and returns `EINVAL` on a filesystem
that does not support the flag. macOS `renamex_np` with `RENAME_EXCL` refuses
an existing target where the volume supports it. Windows `MoveFileExW` with
`MOVEFILE_REPLACE_EXISTING` replaces a file and errors on a directory target;
without the flag it fails on any existing target. Go's `os.Rename` on Unix
`Lstat`s the target and refuses an existing directory before calling
`rename(2)`, so it does not silently replace an empty directory, with a race
window between the two calls.

**Go.** `os.Root` (Go 1.24) confines every operation beneath a directory and
refuses a symbolic link that resolves outside it; Go 1.25 added `Root.Rename`
and `Root.MkdirAll`. On darwin, `File.Sync` issues `F_FULLFSYNC`
(`internal/poll/fd_fsync_darwin.go`). `archive/zip`'s `CreateHeader` always
sets the data-descriptor flag on a file entry and adds an extended-timestamp
extra field when `Modified` is set; `CreateRaw` writes a header exactly as
given. `archive/tar` writes a ustar header when every field fits ustar, and
otherwise a pax `x` header carrying only the records that do not fit, with
record keys sorted. METAIS's own Liquid engine (ADR-010 D4) exposes
`ParseValue`, `ParseCondition`, `Eval` and `EvalCondition` for standalone
expressions.

## Decision

### D1 — The `outputs` section is a list of declarations, each of one form

1. **`outputs` is a required, non-empty array** under the manifest's top
   level (ADR-002 places the key). Its order is the author's and is the
   order in which declarations are planned (D13); it never decides the
   order of written files (D13 item 6).
2. **Each declaration takes exactly one form**, named by the one source key
   it carries:

   | Source key | Form | Produces |
   |---|---|---|
   | `template: <path>` | template output | one file per evaluation, rendered from `templates/<path>` |
   | `asset: <path>` | asset copy | the file `assets/<path>`, or every file beneath the directory `assets/<path>` |
   | `mirror: <dir>` | mirror | one rendered file per template beneath `templates/<dir>` (D6) |
   | `container: {…}` | zip container | one zip file whose entries are declarations of these same forms (D8) |

   A declaration with none of these keys, or with more than one, is a
   finding.
3. **Fields by form** (✓ allowed, — refused as a finding):

   | Field | template | asset | mirror | container | Meaning |
   |---|---|---|---|---|---|
   | `id` | ✓ | ✓ | ✓ | ✓ | identifier (D2 item 1) |
   | `path` | ✓ | ✓ | ✓ | required | destination, a Liquid template (D4) |
   | `when` | ✓ | ✓ | ✓ | ✓ | condition (D3) |
   | `for_each` | ✓ | — | — | ✓ | collection expression (D3) |
   | `escape` | ✓ | — | ✓ | — | `none` (default) or `html` (D2 item 4) |
   | `newline` | ✓ | — | ✓ | — | `as-written` (default) or `crlf` (D10) |
   | `validate` | ✓ | — | — | — | package path of a JSON Schema (D11) |
   | `kind` | ✓ | — | — | — | output kind override (D7) |
   | `executable` | ✓ | ✓ | — | — | file mode (D2 item 6) |
   | `first`, `stored` | entry only | entry only | entry only | entry only | container entry placement (D8) |

   An unknown field is a finding. The manifest's JSON Schema (ADR-002
   publishes it) expresses this table with `oneOf` over the source keys.
4. **Example.**

   ```yaml
   outputs:
     - id: report
       template: report.md.liquid            # path defaults to report.md
     - id: acr_docx
       path: "{{ inputs.product.slug }}-acr.docx"
       container:
         profile: ooxml
         entries:
           - mirror: docx                    # templates/docx/** → entries
             path: .
             escape: html
           - asset: vendor-logo.png
             path: word/media/image1.png
             stored: true
             when: inputs.vendor.logo
     - id: acr_yaml
       template: openacr.yaml.liquid
       path: "{{ inputs.product.slug }}-acr.yaml"
       validate: references/openacr/openacr.schema.json
     - id: article
       template: site/_article.html.liquid
       for_each: inputs.articles | where: "draft", false
       path: "articles/{{ item.slug }}.html"
       escape: html
     - id: site
       mirror: site
       path: .
       escape: html
     - id: static
       asset: static                        # assets/static/** → static/**
   ```

### D2 — Common fields

1. **`id`** is required on every top-level declaration and optional on a
   container entry. Syntax `^[a-z][a-z0-9_]{0,63}$`, the rule ADR-003 D1
   item 3 gives input names (without its reserved words, since an id is
   never a Liquid name), so every identifier a manifest declares has one
   spelling. Top-level ids are unique in the manifest; entry ids are unique
   within their container. Findings (`subject.name`, ADR-006 D4) and the
   dry-run listing (D16) name declarations by id; single-output selection
   (D18) names a file by its rendered path.
2. **`path`** is a Liquid template string (D4). For a template output it
   names one file; for an asset directory, a mirror or a container's
   directory-form entry it names a destination directory, `.` meaning the
   root of the tree.
3. **`when`** and **`for_each`** are D3's.
4. **`escape`** selects the output's escaping mode, `none` or `html`, with
   ADR-004's semantics. Default `none`: values render as they are. On a
   mirror it applies to every mirrored file.
5. **`newline`** is D10's; **`validate`** is D11's; **`kind`** is D7's.
6. **`executable: true`** marks a file to be written with mode `0755`
   instead of `0644` (less the process umask) on a POSIX filesystem, and
   recorded with that mode in the standard-output archive (D17). It is
   ignored when writing to a filesystem with no execute bit. It is refused
   on a container entry, because ADR-001's zip profile records no file
   modes. Default `false`. (Proposed ruling 9.)

### D3 — `for_each` and `when` are Liquid expressions evaluated by the template engine

1. **`when` is a Liquid condition**: exactly what may follow `if` in
   `{% if … %}` in the pinned dialect — comparisons, `and`, `or`, `contains`,
   literals and variable lookups. Truthiness is ADR-004's: only `nil` and
   `false` are false. A declaration without `when` always holds.
2. **`for_each` is a Liquid value expression**: exactly what may appear
   inside `{{ … }}` — a lookup or literal followed by any filters of the
   pinned dialect, so `inputs.articles | where: "draft", false` and
   `inputs.rows | sort: "id"` are expressions. Its value is:
   - an **array**: one evaluation per element, in array order, with the
     element bound to `item`;
   - **`nil`** (an absent optional input): zero evaluations, no finding;
   - anything else (string, number, boolean, object): a finding naming the
     declaration and the type found.
3. **The binding name is `item`**, fixed. A declaration cannot rename it.
4. **Scope.** Which globals each expression sees is ADR-004 D7 item 10's
   table: `inputs`, `package`, `now` and `sources` everywhere; `item` and
   `fan_out` for `when`, `path` and the template in every evaluation of a
   `for_each` declaration, and in every entry of a container that has
   `for_each`, but never in the `for_each` expression itself; `output` only
   in a template body.
5. **`when` is evaluated per file.** For a `for_each` declaration it is
   evaluated once per element, with `item` bound, so an author filters items
   either in the expression or in `when`. For any other declaration it is
   evaluated once.
6. **Nesting.** At most one `for_each` exists on any chain of enclosing
   declarations: an entry of a container that has `for_each` cannot have one.
   A finding otherwise, since a second binding would shadow `item`.
7. **Errors** in either expression (a syntax error, an unknown filter, a
   strict-variable failure under ADR-004's rules) are findings located at
   the declaration's manifest JSON Pointer (`/outputs/3/for_each`); the
   declaration produces no files and planning continues with the next one.
8. **What this ADR needs from ADR-004 and ADR-010**: `liquid.ParseCondition`
   and `liquid.ParseValue`, evaluated by `liquid.EvalCondition` and
   `liquid.Eval` (ADR-010 D4 item 3) against the scope ADR-004 D7 item 10
   gives the expression, returning a value in ADR-003's data model or an
   error with a position in `src`. `check` parses both at check time.

### D4 — Path templates

1. **Every `path` is rendered as a Liquid template** in a restricted mode:
   output expressions, filters, `if`/`unless`/`case`, `assign` and `capture`
   are available; `render`, `embed`, `asset`, `provenance` and any tag that
   produces no text value are a finding at check time. Escaping is never
   applied to a path, whatever the declaration's `escape`.
2. **The rendered string is the path, exactly.** It is not trimmed,
   normalized or rewritten. A path that fails D12 is a finding naming the
   declaration, the evaluation (`item` index) and the rendered string.
3. **Separator is `/`.** A rendered value containing `/` creates
   subdirectories; that is how input-derived directory names work (PRD
   §4.5).
4. **Defaults when `path` is omitted** (not allowed on a `for_each`
   declaration or a container, where it is required):
   - a template output: the template's path relative to `templates/`, with
     a final `.liquid` removed (`ci/workflow.yml.liquid` → `ci/workflow.yml`);
   - an asset: its path relative to `assets/` (`static` → `static/…`);
   - a mirror: its directory relative to `templates/` (`site` → `site/…`;
     `.` → the root).

   A container entry takes the same defaults relative to the container root.
5. **Authoring note the specification carries**: in YAML a path that begins
   with `{` must be quoted.

### D5 — Asset copies

1. **`asset` names a file or a directory beneath `assets/`**, by a path
   relative to `assets/` that obeys ADR-001's entry rules. `.` names the
   whole `assets/` tree. A name that matches nothing is a finding.
2. **A file asset** is copied to `path`, byte for byte.
3. **A directory asset** copies every file beneath it, keeping its relative
   path under the destination directory `path`. Directories are not
   entries: an empty directory produces nothing.
4. **Assets are never rendered, escaped, converted or validated.** They are
   the only way binary bytes reach the output besides containers (PRD
   non-goals).
5. **An asset used only through ADR-004's `{% asset %}` tag for its
   contents** need not be declared; declaring it is what puts it in the
   output.
6. **The asset placement map.** For ADR-004 D13's path mode (`{% asset
   "<path>" %}`, or with `as: "path"`), this ADR provides, per tree (the output directory or one
   container), the map from asset source path to destination path, and the
   rendering file's own destination path. An asset placed zero times or
   more than once in the tree where the template renders makes that tag
   an error, because the answer would be absent or ambiguous.

### D6 — The mirror shorthand

1. **`mirror: <dir>` renders every template beneath `templates/<dir>`**,
   walking it in byte order of entry names, to the same relative path
   beneath the destination `path`, with a final `.liquid` removed from each
   file name.
2. **Partials are excluded by name**: a file or directory whose name begins
   with `_`, at any depth beneath the mirror root, is not mirrored. This is
   the convention that marks a partial in a mirrored tree; ADR-004's
   `render` loads it as usual.
3. **Explicit outputs win**: a template named by any `template` field
   anywhere in the manifest is excluded from every mirror, so an author
   mirrors a tree and gives one of its files a `for_each` without rendering
   it twice.
4. **File names are not templates.** A mirror renders contents, never
   names; a name derived from inputs needs an explicit template output.
   (Liquid in a file name could not carry `|` or `"`, both reserved on
   Windows, so the package itself would fail ADR-001's portable entry
   rules.)
5. **A file named exactly `.liquid`** would mirror to an empty name: a
   finding.
6. **A mirror that yields no file is a warning**, since it is almost always
   a mistaken directory.
7. **Composition.** Mirrored files join the same plan as explicit outputs;
   two files with colliding paths (D12 item 6) are a finding whatever
   produced them.
8. **The smallest manifest** is `outputs: [{id: all, mirror: .}]`.

### D7 — The output kind

1. **Every rendered file has a kind**, used for ADR-004's provenance syntax
   and for D11's parse. It is the `kind` field when present, otherwise
   derived from the file's rendered path by this table, matching the final
   extension ASCII-case-insensitively, then the whole file name:

   | Kind | Extensions | File names |
   |---|---|---|
   | `html` | `.html` `.htm` | |
   | `xml` | `.xml` `.xhtml` `.svg` `.rels` `.opf` `.ncx` `.xsd` `.xsl` `.xslt` `.plist` | `[Content_Types].xml` (by extension) |
   | `markdown` | `.md` `.markdown` | |
   | `json` | `.json` `.geojson` `.webmanifest` | |
   | `yaml` | `.yaml` `.yml` | |
   | `csv` | `.csv` | |
   | `css` | `.css` `.scss` `.less` | |
   | `hash` | `.sh` `.bash` `.zsh` `.py` `.rb` `.pl` `.r` `.toml` `.ps1` `.properties` `.tf` | `Makefile` `Dockerfile` `Containerfile` `.gitignore` `.gitattributes` `.dockerignore` `.env` |
   | `slash` | `.js` `.mjs` `.cjs` `.ts` `.tsx` `.jsx` `.go` `.c` `.h` `.cc` `.cpp` `.hpp` `.java` `.kt` `.kts` `.swift` `.rs` `.cs` `.scala` `.dart` `.jsonc` `.proto` `.groovy` `.gradle` | |
   | `sql` | `.sql` | |
   | `text` | anything else | |

   The specification carries this table as normative; a later 1.x may add
   rows and never changes one.
2. **`kind` values** are exactly the table's kinds. `kind` on a template
   output overrides the table for that output's every file.
3. **Assets and containers have no kind.**
4. **Kind is not escaping.** `escape` is chosen separately (D2 item 4).

### D8 — Zip containers

1. **Shape.**

   ```yaml
   container:
     format: zip          # optional; zip is the only format in 1.x so far
     profile: odf         # optional: ooxml | odf | epub (D9)
     mimetype: application/vnd.oasis.opendocument.text   # odf profile only
     entries: [ … ]       # non-empty; declarations of the D1 forms
   ```
2. **Entries are declarations, inline.** Each entry is a template, asset,
   mirror or container declaration with the fields of D1 item 3, plus
   `first` and `stored`. An entry's `path` is its name inside the container,
   relative to the container root. Entries are rendered exactly as
   top-level outputs are, and are not written on their own: a part that
   must also be a standalone file is declared again at the top level.
3. **One container is one tree.** Its entry paths obey D12 within the
   container, independently of the output directory; collisions are
   detected among its entries only.
4. **Nesting** is allowed to a depth of three containers (a container inside
   a container inside a container), ADR-009's `container-depth` limit
   (default 3, which a caller may lower). Deeper is ADR-009 D19's
   `limit.package`, since the manifest itself declares the nesting.
5. **`for_each` on a container** produces one zip per element, with `item`
   visible to every entry (D3 item 4).
6. **Entry order**, fixed and independent of declaration order except where
   the author asks for it:
   1. the entries a profile places first (D9), in the profile's order;
   2. the files of every declaration marked `first: true`, in declaration
      order, each declaration's own files in byte order of their entry
      path;
   3. every other file, in byte order of its entry path (UTF-8 bytes, which
      is code-point order).

   A file appears once, at its earliest position.
7. **Methods.** `stored: true` writes every file of that declaration with
   method 0 (stored), as does a profile's `mimetype` entry (D9); every other
   entry is written with ADR-001 D4's `auto` method: deflated when that is
   strictly shorter, otherwise stored, and an empty entry always stored. No
   other method exists. Which method `auto` chose depends on the encoder,
   so conformance and `test` compare the method only where `stored: true`
   fixes it (ADR-001 D4 item 6, ADR-011 D10 item 3).
8. **Writing** follows ADR-001's deterministic zip rules for everything this
   ADR does not state here: timestamps, entry name encoding, extra fields
   (none), version fields, deflate settings, the central directory and its
   end record. There is no zip64 (ADR-001 D3): a container that plans more
   than 65,534 entries is refused, and ADR-009's `output-size` keeps its
   size far below the 32-bit ceiling. The container's entry order and
   `stored` set are this ADR's; ADR-001's sorted-entry rule for packages
   does not apply inside a container.
9. **Containers hold file entries only**, never directory entries (ODF §4.3
   states directories have no entries; OPC has only parts).
10. **A container's `path`** names a file in its enclosing tree; nothing
    checks or requires its extension.

### D9 — Container profiles: ooxml, odf and epub

A profile states a format's packaging requirements once, so an author who
names it gets them right without knowing them. A profile **applies** the
settings the format requires and **verifies** what the tool can see; it
never validates the XML content of a part. A container without a profile is
written exactly as declared.

1. **`ooxml`** (ECMA-376 Part 2):
   - requires an entry named `[Content_Types].xml` at the container root;
   - places `[Content_Types].xml` first. OPC does not require this (it states
     no ordering); the profile does it so that a reader consuming the zip
     sequentially meets the media types before the parts they describe;
   - refuses an entry path containing a non-ASCII character (§7.3.3). An
     author percent-encodes such a name in the path itself;
   - the case and trailing-dot rules of §6.2.2 are already D12's (D12 is
     stricter).
2. **`odf`** (OpenDocument 1.3 Part 2):
   - requires `container.mimetype`, a media type of the form
     `type/subtype` in printable ASCII (`!`–`~`), with no whitespace;
   - **writes the `mimetype` entry itself**: content exactly the declared
     media type, no newline, stored, first, with no extra field and no data
     descriptor (its sizes and CRC in the local header). An author
     declaring an entry named `mimetype` in an `odf` container is a finding;
   - requires an entry `META-INF/manifest.xml`.
   - The profile does not check that the manifest lists every file (§3.2);
     that is XML content.
3. **`epub`** (EPUB 3.3 OCF):
   - **writes the `mimetype` entry itself**: content exactly
     `application/epub+zip`, no newline, stored, first, no extra field, no
     data descriptor. An author's own `mimetype` entry is a finding;
   - requires an entry `META-INF/container.xml`;
   - reports a **warning** for an entry name containing U+0020 (§4.2.3
     "SHOULD NOT");
   - D12 already enforces §4.2.3's characters, its 255-byte name limit, and
     its uniqueness rule (D12 matches it exactly).
4. **The tool-written `mimetype`** is the reason the profiles exist as more
   than documentation: a template or asset holding it almost always ends
   with a newline, which breaks both formats silently.
5. **An author's `first` and `stored`** combine with a profile: the
   profile's first entries precede the author's (D8 item 6).
6. **`check` (ADR-007) verifies everything the profile can verify
   statically** — the declared media type, the presence of a declaration
   whose default or literal path is the required entry — and the render
   verifies the rest on the planned entry paths.

### D10 — CRLF conversion

1. **`newline: crlf`** converts the file's bytes after rendering and before
   validation: every LF (0x0A) not immediately preceded by CR (0x0D)
   becomes CR LF. An existing CR LF is left as it is, so the conversion is
   idempotent and a template authored with CRLF converts to the same
   bytes. A lone CR is left as it is.
2. **`newline: as-written`** (default) writes the bytes the template
   produced.
3. **It applies to the whole file**, including bytes ADR-004's `embed` tag
   placed in it.
4. **It is not available on assets or containers**, which are bytes, not
   text.

### D11 — Output schema validation

1. **`validate: <path>`** names a JSON Schema 2020-12 document anywhere in
   the package except `.compiled/`, by package path. `references/` is the
   recommended home, because a published output schema is also reference
   material for the supplier.
2. **The output's kind must be `json` or `yaml`**; any other kind with
   `validate` is a finding at check time. XML and CSV validation are not in
   1.0.
3. **The file's final bytes** (after D10) are parsed — JSON per RFC 8259,
   YAML per ADR-003's YAML profile — and the value is validated with
   ADR-003's validator, dialect, asserted keywords and formats, and `$ref`
   resolution (package-internal only, no network).
4. **A parse failure or any validation error refuses the render**, with
   every finding of every validated file reported, each located by output
   path (and container entry path), JSON Pointer into the output, and line
   where the parser gives one. Nothing is written.
5. **`check` compiles every `validate` schema**, without data.

### D12 — Portable path rules

These rules apply to every rendered path in every tree: each top-level file
path, relative to the output directory, and each container entry path,
relative to its container. They are also the rules for a package's entry
names (ADR-001 D2), so this is the one portable-path predicate of the
format. A path that breaks any of them is refused on every platform, before
anything is written.

1. **Form.** The path is valid UTF-8, in Unicode Normalization Form C, made
   of one or more non-empty segments separated by `/`; no leading `/`, no
   trailing `/`, no empty segment, no segment `.` or `..`. A path not in
   NFC is refused, not normalized: a path is exactly what the template
   produced.
2. **Characters.** Every code point of a segment is U+0020 or belongs to
   one of the general categories L, M, N, P or S as **Unicode 15.0.0**, the
   version specification 1.0 pins, assigns them; and none is one of:
   - `<` `>` `:` `"` `/` `\` `|` `?` `*` (Windows' reserved characters);
   - U+FFF0–U+FFFF (Specials, which EPUB refuses; U+FFFC and U+FFFD are
     category So and would otherwise pass).
   The category rule refuses everything else the Windows and EPUB lists
   name — C0 and C1 controls and U+007F (Cc), private use (Co), every
   noncharacter and every **unassigned** code point (Cn) — and also every
   format character (Cf: the bidirectional controls U+200E, U+200F,
   U+202A–U+202E and U+2066–U+2069, zero-width joiners, the soft hyphen)
   and every separator other than U+0020 (Zl, Zp, other Zs), which make a
   path display differently from what it is in a listing (Proposed ruling
   6). Unassigned code points are refused because normalization and case
   folding are guaranteed stable only for assigned characters; a later 1.x
   that pins a later Unicode version only accepts more.
3. **Segment ends.** A segment does not end with U+0020 or `.`.
4. **Reserved names.** A segment is refused when the part before its
   first `.`, with trailing U+0020 removed, matches ASCII-case-insensitively
   `CON`, `PRN`, `AUX`, `NUL`, or `COM` or `LPT` followed by one of `0`–`9`,
   `¹`, `²`, `³`. This covers `NUL.tar.gz`, `com¹.txt` and `Con .md`. `COM0`
   and `LPT0` are refused beyond the documented list, at no cost; the
   trailing-space trim is stricter than documented, deliberately. A segment
   that Git treats as `.git` (ADR-009 D8 item 6: `.git` in any letter case,
   and its NTFS short name `git~1`) is refused too, because writing
   `.git/hooks/*` or `.git/config` runs code at the next git command.
5. **Lengths.** Each segment is at most 255 bytes of UTF-8 (the EPUB limit,
   and within ext4's 255 bytes and NTFS's 255 UTF-16 units). The whole path
   is within ADR-009's `path-length` (512 bytes) and `path-depth` (32
   segments), which are the format's maxima, since no limit can be raised
   (ADR-009 D4). Windows programs that are not long-path aware cannot open a
   file whose full path exceeds 259 characters; the specification's
   authoring guidance says so, and the tool does not refuse on it.
6. **Collisions.** Within one tree, take the set of every path and every
   directory prefix of every path (`a/b/c` contributes `a`, `a/b` and
   `a/b/c`). Two distinct members collide when they are canonical caseless
   matches (Unicode D145: `NFD(toCasefold(NFD(x)))` equal, full case
   folding, CaseFolding.txt statuses C and F of Unicode 15.0.0, no
   tailoring). This is EPUB's uniqueness rule extended to directories
   (`Docs/a` beside `docs/b` is one directory on a case-insensitive file
   system and two elsewhere); it is stricter than OPC's ASCII-only rule,
   NTFS's upcase table and APFS's case- and normalization-insensitive
   comparison, so a package that passes it writes the same set of files on
   every one of them. Two colliding paths are a finding naming both
   declarations.
7. **File–directory conflicts.** A path whose prefix of segments collides
   with another file's whole path (`docs` and `docs/a.md`) is refused: one
   name cannot be both a file and a directory. This is also OPC §6.2.2's
   "derivable" rule.
8. **Confinement** is the consequence of items 1 and 2 on the path itself;
   confinement on disk (symbolic links, junctions) is D15's and ADR-009's.

### D13 — The render pipeline

A render runs in the phases below. Phases 1 to 4 write nothing anywhere. A refusal in
any of them ends the render with every finding reported, and nothing is
written.

0. **Before this ADR**: the package is opened and its digests verified
   (ADR-001, ADR-009); inputs are supplied, defaulted and validated
   (ADR-003, ADR-006). An input error ends the render here.
1. **Plan.** Every declaration, in manifest order, recursively into
   containers: evaluate `when` and `for_each` (D3), render paths (D4),
   expand assets and mirrors (D5, D6). The result is the **plan**: for each
   tree, a list of planned files, each with its path, source, `item`
   index, settings and manifest location. A failing declaration contributes
   findings and no files; planning continues.
2. **Verify paths.** D12 on every planned path; collisions and file–directory
   conflicts per tree; the profile requirements (D9); ADR-009's output
   count, which counts every planned file including container entries.
3. **Check the destination** (directory mode only): D15.
4. **Render and transform.** Each template file is rendered (ADR-004) with
   its kind, escaping and `item`; then D10; then D11. Rendered text is held
   in memory, bounded by ADR-009's per-output and total output size limits;
   asset bytes are not loaded, only their sizes (from the package's
   directory) checked against the same limits. Every file is rendered even
   after another fails, so all findings come in one pass.
5. **Emit**, in exactly one of: directory commit (D14), dry run (D16),
   archive to standard output (D17), one file to standard output (D18).
   Containers are assembled during emission, streaming their entries from
   memory and from the package into the destination, so no container is
   held whole in memory.
6. **Order.** Planning order is manifest order and array order; it decides
   the order of findings only (ADR-006 sorts them further). Every emitted
   listing, archive and write sequence is in **byte order of the path**
   (UTF-8 bytes) within each tree. Map iteration, goroutine scheduling and
   directory listing order of the host never reach an output.

### D14 — Committing to a directory: what "writes nothing" means

The guarantee: every refusal (phases 1 to 4) and every failure before the
commit's first rename leaves the filesystem as it was. Beyond that the
guarantee depends on whether the output directory exists.

1. **The output directory does not exist — atomic.**
   1. Create its missing ancestors, recording which were created. They are
      the caller's own path (`--out`), opened through an `os.Root` on its
      nearest existing ancestor, and nothing a package or an input
      produces names any of them.
   2. Create a staging directory beside it,
      `<parent>/.metais-staging-<h>`, where `<h>` is the first 16 lowercase
      hex digits of the SHA-256 of the output directory's base name, in the
      same parent, so the final rename never crosses a filesystem. It is
      created exclusively: if it exists — a killed render's leftover, or a
      concurrent render of the same output — the render is refused
      naming it. The name is fixed, never random or taken from the process,
      so no source of randomness or process identity is read (ADR-009 D12).
   3. Write every planned file beneath it, every operation through an
      `os.Root` on the staging directory, so every path a package or an
      input produced is confined to it (ADR-009 D8).
   4. Rename the staging directory to the output directory with a rename
      that refuses an existing target: `renameat2(RENAME_NOREPLACE)` on
      Linux, `renamex_np(RENAME_EXCL)` on macOS, `MoveFileExW` without
      `MOVEFILE_REPLACE_EXISTING` on Windows (under ADR-010 Amendment A1,
      proposed, the Windows call is `MoveFileW`, which refuses an existing
      target the same way). Where the filesystem does not
      support the flag (`EINVAL`, `ENOTSUP`), check the target with `Lstat`
      and rename, accepting the race window between the two.
   5. If the target appeared meanwhile, or anything before the rename
      failed, remove the staging directory and the ancestors created in
      step 1 (everything removed is the render's own) and refuse.

   At every instant the output directory either does not exist or holds the
   complete render. A process killed at any point leaves at most the hidden
   staging directory and the created ancestors.
2. **The output directory exists — each file atomic, the set atomic against
   every failure but a crash during commit.** *(Under Amendment A1 the
   files are prepared beneath one staging directory inside the output
   directory and committed by unit; A1 replaces items 2.1 to 2.3's
   temporary files, and item 5's ownership rule.)*
   1. **Prepare.** For each planned file in path order: create missing
      directories (recorded), then write its bytes to a temporary file in
      the target's own directory, `.metais-tmp-<h>`, where `<h>` is the first
      16 lowercase hex digits of the SHA-256 of the target's file name,
      opened with exclusive create (an existing one refuses the render,
      naming it). Any failure (disk full, a permission) removes every
      temporary file and every recorded directory and refuses: nothing
      changed.
   2. **Commit.** For each file in path order, rename its temporary file
      onto its target: a `create` with the no-replace rename of item 1.4,
      a `replace` with a replacing rename (`rename(2)`;
      `MoveFileExW(MOVEFILE_REPLACE_EXISTING)`).
   3. **A failure during commit** (a Windows file held open by another
      process; a target that appeared since the check) stops the commit,
      removes the remaining temporary files, and refuses with an error
      finding that lists the paths already committed.

   Each file is at every instant either its old bytes or its new bytes,
   whole. A process killed during commit can leave a mix of whole old and
   whole new files, and temporary files that the next render will see.
   An author who needs the atomic form renders into a directory that does
   not yet exist (the golden path).
3. **No fsync.** The tool does not flush files or directories to stable
   storage. The guarantee above is against the tool's process ending at
   any point (crash, kill, panic), not against power loss or an operating
   system crash. On darwin a flush is `F_FULLFSYNC` per file, which turns a
   website of thousands of files into a render of tens of seconds; the
   output is reproducible by rendering again. (Proposed ruling 8.)
4. **Modes.** Files are created `0644` (`0755` with `executable`),
   directories `0755`, both less the process umask. The umask is host
   state that reaches file metadata only, never bytes or a template.
5. **The tool removes only what this render created**: its staging
   directory, its temporary files, and directories it created and left
   empty. It never removes anything else, on success or failure. A stale
   staging directory or temporary file from a killed render is left for
   the caller; because its name is fixed, the next render into the same
   output refuses, naming it.

### D15 — The overwrite policy and the destination checks

1. **`--out <dir>`** (flag names are ADR-008's) names the output directory.
   It may be a symbolic link to a directory; the caller is trusted, and the
   link is resolved once. Nothing beneath it may be a link (item 4).
2. **Absent**: D14 item 1.
3. **An existing directory**: when it contains no entry at all, D14 item 2.
   When it contains any entry — a hidden file such as `.DS_Store` included —
   the render is refused unless the caller passes **`--overwrite`**, and
   then D14 item 2. An existing non-directory at `--out` is refused.
4. **Per planned file, in an existing directory**, walking each segment
   beneath the output directory with `Lstat` through an `os.Root` on it:
   - a segment that is a symbolic link, a Windows junction or any other
     reparse point: refused, even when it points inside the tree;
   - a segment that must be a directory and is not: refused;
   - an existing entry in the same directory that is a canonical caseless
     match for the segment but not byte-equal to it (`Readme.md` against
     `README.md`, NFD against NFC): refused, because what a case-insensitive
     filesystem would do is not portable;
   - the target is a directory, or a non-regular file (FIFO, socket,
     device): refused;
   - the target is absent: status `create`;
   - the target is a regular file: status `replace`.
   Every check is made again through the `os.Root` at write time, so a
   symbolic link planted between check and write fails the operation rather
   than escaping.
5. **An existing file's content is never read** (ADR-009 D8 item 4): the
   checks use names and `Lstat` only, so `--overwrite` and the dry run's
   status cannot tell anyone what a replaced file held. Every planned file
   that exists is replaced, even when its bytes would not change.
6. **Replacing never writes into an existing file**: the rename replaces
   the directory entry. A target with other hard links keeps them, with the
   old bytes; the replaced file gets the new file's mode, and the running
   user as owner.
7. **`--overwrite` never deletes.** A file in the directory that the render
   does not write is left alone and not reported.
8. **Every refusal here is a finding** reported with all the others, before
   any write.

### D16 — Dry run

1. **`--dry-run` runs phases 1 to 4**, and phase 3 when `--out` is given, and
   writes nothing anywhere: no staging directory, no temporary file, no
   directory.
2. **It lists every top-level file in path order**, one record each:
   - `path`; `output` (the declaration id); `item` (the `for_each` index,
     when there is one); `source` (`template`, `asset` or `container`);
     `kind` (template files only); `size` in bytes; `digest` (`sha256:`
     and the lowercase hex SHA-256 of the exact bytes a render would write,
     the presentation ADR-001 D6 item 5 fixes); `executable`;
   - `status` (`create` or `replace`), only when `--out` is a directory;
   - for a container, `entries`: in container order, each with `path`,
     `output` (entry id, when given), `item`, `source`, `kind`, `method`
     (`stored` or `deflated`), `size` and `digest` of the uncompressed
     content. The container's own `size` and `digest` are those of the zip
     the tool writes.
3. **Findings and exit status are a render's.** A dry run that would be
   refused reports the same findings and exit code (ADR-006, ADR-008).
4. **Output forms**: the JSON form carries the records of item 2 as
   `render`'s `result.files` in ADR-008's envelope (ADR-008 D8 item 9); the
   human form is ADR-008's, one line per file with entries indented beneath
   their container.

### D17 — Standard output: the archive

1. **`--out -`** writes every top-level file of the render to standard
   output as one POSIX **pax interchange format** archive, after phases 1
   to 4 succeed. Nothing reaches standard output from a refused render.
2. **Entries.** One regular-file entry (typeflag `0`) per top-level file,
   in path order. A container is one entry holding its zip bytes. No
   directory entries, no links, no global extended header.
3. **Header fields**, the same for every entry:

   | Field | Value |
   |---|---|
   | name, prefix | the path, as a ustar name or prefix/name split at a `/`, when the path is ASCII and fits; otherwise a preceding typeflag `x` extended header carrying exactly one record, `path`, in UTF-8 |
   | mode | `0644`, or `0755` with `executable` |
   | uid, gid | `0` |
   | uname, gname | empty |
   | mtime | `315532800` (1980-01-01T00:00:00Z) |
   | size | the file's byte count |
   | devmajor, devminor | `0` |
   | magic, version | `ustar\0`, `00` |

   The mtime is the instant ADR-001 fixes for zip entries, so every
   archive the tool writes carries one timestamp. It is never the caller's
   `--now`, which is template data (R10 item 4).
4. **The archive ends with two 512-octet zero blocks** and no further
   padding.
5. **Conformance compares the archive by entries**: order, path, typeflag,
   mode, uid, gid, uname, gname, mtime, size and content. The tool's own
   archive is byte-identical across platforms and runs, as its containers
   are. Go's `archive/tar`, given these headers with no format forced,
   writes exactly this (ustar where it fits, a pax `path` record where it
   does not); ADR-010 proves it.
6. **The archive is built from the plan**, never from files on disk, so no
   host state (umask, ownership, filesystem case behaviour) reaches it.

### D18 — One selected output: raw on standard output, or alone in a directory

1. **`--out -` with `--only <path>`** (ADR-008 D8 item 5) writes the bytes of
   exactly one top-level file to standard output and nothing else. `<path>`
   is a top-level path as the dry run lists it; a declaration id is not
   accepted, since a `for_each` declaration yields many files. A
   container's entry is not selectable; its container is.
2. **The whole render runs**: every file is planned, rendered and verified,
   so a selected render is refused exactly when a full render is, and its
   bytes are the bytes the full render writes.
3. **With `--out <dir>`**, emission (D14) writes only the selected file, by
   the same commit rules.
4. **A path that matches no file** is refused with a finding listing the
   available paths (bounded by ADR-006's listing rules).

### D19 — Determinism, and how it is proved

1. **Sources of variation and their fixes**:
   - iteration: arrays in order, objects in ADR-003/ADR-004's defined
     order, mirrors and asset directories in byte order, emission in path
     order (D13 item 6);
   - bytes: UTF-8 throughout; line endings as written unless D10;
   - archives: ADR-001's zip rules, D8's entry order and methods, D17's
     fixed headers;
   - text processing: Unicode 15.0.0 normalization and case folding,
     pinned, with unassigned code points refused (D12 item 2), so two
     implementations on different Unicode tables agree on every accepted
     path;
   - host: no locale, time zone, clock, environment or working directory
     reaches planning or rendering (ADR-009 proves the isolation).
2. **Proofs**, each a test in `go test`:
   - **Vectors** (ADR-011) state each output's expected bytes, and each
     container's and archive's expected entries.
   - **Cross-platform identity** (SC5): every vector rendered on darwin and
     linux, amd64 and arm64, gives the same SHA-256 for every output and for
     the D17 archive stream.
   - **Repetition**: every vector rendered repeatedly in one process and in
     fresh processes, with `GOMAXPROCS` of 1 and of the host's CPU count,
     gives identical bytes. Pinned red first against a render that emits
     in map-iteration order.
   - **Host variation**: rendered under `TZ=Pacific/Chatham`, `LC_ALL` and
     `LANG` of `tr_TR.UTF-8` (Turkish dotted and dotless i, the classic
     case-mapping trap) and a `umask` of `077`; outputs and archive
     identical to the baseline.
   - **Path rules**: every rule of D12 has a vector on every platform,
     including the ones a platform's own filesystem would accept.

## Interfaces with sibling ADRs

**ADR-001 — the container and its layout.**
- *Relies on*: `templates/` and `assets/` as reserved top-level
  directories; ADR-001 D2, which restates D12 for package entry names, so
  package entries, output paths and container entry paths obey one
  predicate.
- *Relies on* ADR-001 D4's **deterministic zip writer** for containers: a
  caller-given entry order; per entry `store` or `auto`; no extra field on
  any entry; the fixed timestamp 1980-01-01 00:00:00, which D17's tar mtime
  mirrors; UTF-8 names with the language-encoding flag; `version needed to
  extract` 10 or 20, within EPUB's `10`, `20`, `45`; no zip64 (ADR-001 D3);
  sizes and CRC in the local header with no data descriptor, which D9's
  `mimetype` rules depend on.
- *Promises*: containers never need encryption, comments, directory
  entries, modes or zip64; a container planning more than 65,534 entries is
  refused (D8 item 8).

**ADR-002 — the manifest and versioning.**
- *Relies on*: a top-level `outputs` key, its canonical JSON form, and the
  manifest JSON Schema embedding this ADR's `outputs` schema (D1 item 3).
- *Promises*: `id` follows the input-name rule (D2 item 1), so the manifest
  has one identifier syntax; D12 item 4's device names are the list ADR-002
  D5 item 2 uses for package names.

**ADR-003 — the data specification.**
- *Relies on*: the data model `inputs` and `item` carry; the JSON Schema
  2020-12 validator with its asserted keywords and formats; the YAML
  profile; `$ref` resolution confined to the package, with ADR-003 D5 item
  5's directory read, for an output schema, as the top-level directory that
  holds the `validate` file; reused unchanged for D11. Arrays keep order;
  object members iterate in ADR-004 D4 item 5's order.

**ADR-004 — the template environment.**
- *Relies on*: `liquid.ParseCondition`, `liquid.ParseValue` and their
  evaluators (D3 item 8; ADR-010 D4 item 3); ADR-004 D7 item 10's table of
  which globals each expression sees (D3 item 4); a restricted
  path-rendering mode (D4 item 1); rendering a template with a given scope,
  escaping mode and output kind; `html` escaping that emits only `&amp;`
  `&lt;` `&gt;` `&quot;` and `&#39;`, so it is also correct XML escaping for
  OOXML, ODF and EPUB parts; the asset tag's syntax, `{% asset "<path>" %}`
  for the path mode (ADR-004 D13).
- *Promises*: the kind set of D7, from which ADR-004 D14 maps a provenance
  comment syntax (`json` has its own style; `csv` and `text` need
  `style:`); the asset placement map and the rendering file's destination
  path (D5 item 6); the `_` prefix that excludes a partial from a mirror (D6
  item 2), which ADR-004's partial loading need not know.

**ADR-006 — validation and findings.**
- *Relies on*: findings located by manifest JSON Pointer (declaration
  errors), by output path plus container entry path (render, transform and
  path errors), and by JSON Pointer and line inside an output (D11).
- *Registers*, in ADR-006's `output` family: declaration form and field
  errors (`output.declaration.*`); `when`/`for_each` errors and a wrong
  `for_each` type (`output.expression.*`); path form, NFC, character,
  segment end, reserved name, length, collision and file–directory conflict
  (`output.path.*`, class `content`); profile requirements
  (`output.container.*`); output parse and schema errors (`output.parse`,
  and ADR-006 D6's keyword codes under `output.`); destination refusals —
  non-empty, link, case variant, directory or special file at target,
  non-directory at `--out`, an existing staging or temporary name
  (`output.destination.*`, class `destination`); commit failure with the
  committed paths (`output.destination.commit_failed`); an unknown `--only`
  path (`usage.unknown_output`, ADR-008); the empty-mirror and EPUB-space
  warnings. Container nesting beyond `container-depth` is ADR-009's
  `limit.package`.

**ADR-007 — authoring.**
- *Promises*: everything in D1–D12 that is decidable without data is
  checkable: declaration shape, template and asset existence, expression
  and path-template parsing, literal paths against D12, `validate` schema
  compilation and kind, profile media type and required entries.
- *Promises* the set of templates rendered as entries of an `ooxml`
  container whose kind is `xml`: the templates the split-run check
  examines.
- *Suggests*: a warning for an `xml`-kind entry of a profiled container
  without `escape: html`, since an `&` in supplier data otherwise corrupts
  the document.
- `test` compares an example's expected outputs byte for byte, and
  containers by entries, comparing the method only where `stored: true`
  fixes it (D8 item 7; ADR-007 D5 item 3).

**ADR-008 — the command line.**
- *Relies on* ADR-008's flags: `--out <dir>`, `--out -`, `--only <path>`
  (D18), `--overwrite`, `--dry-run`. The dry-run records of D16, with each
  file's `status`, are `render`'s `result.files` in the JSON envelope; on a
  successful render to a directory the result lists each written file with
  `status`.
- Findings go to standard error whenever standard output carries an archive
  or raw bytes.

**ADR-009 — the security model.**
- *Relies on* limits with defaults: `outputs` (every planned file, container
  entries included), `output-size` and `output-total`, `container-depth`
  (default 3), `path-length` (512 bytes) and `path-depth` (32); the `.git`
  rule (ADR-009 D8 item 6), adopted in D12 item 4; the rule that an existing
  file in the output directory is never read (ADR-009 D8 item 4), adopted
  in D15.
- *Promises*: D12's rules, D14's `os.Root` confinement — every path a
  package or an input produces is written through a root on the staging
  directory or, for an existing output directory, on the output directory;
  only the caller's own `--out` path and its ancestors are reached through
  a root on their nearest existing ancestor — and D15's link refusals, all
  before any write; staging beside the output directory in its parent and
  never under `$TMPDIR`, with fixed names; members for the hostile corpus
  (every D12 rule, links planted in an `--overwrite` target, a target
  swapped for a link between check and write).

**ADR-010 — the implementation.**
- *Needs*: Go 1.26 (ADR-010 D1): `os.Root` with `Rename` and `MkdirAll`
  (since 1.25); `unicode` at 15.0.0; `golang.org/x/text/unicode/norm` and
  `golang.org/x/text/cases` (`Fold`) on the 15.0.0 tables the toolchain
  selects; `archive/zip` with `CreateRaw`; `archive/tar`;
  `golang.org/x/sys/unix` for `renameat2` and `renamex_np`, used only in
  `internal/hostfs`; METAIS's own engine for D3 (ADR-010 D4).

**ADR-011 — the specification and conformance.**
- *Needs* a vector format able to state: an expected file tree with each
  file's bytes; a container's expected entries (order, path, content, and
  method where `stored` fixes it) rather than its bytes; an archive's
  expected entries and header fields (D17 item 5); a dry-run listing;
  expected findings; and a destination prepared before the render
  (existing files, links, case-variant names) for D15. Staging beside the
  output directory leaves nothing behind once the rename has made it the
  output, so ADR-011 D10 item 4's nothing-else-changes check holds.

## Conformance

Vectors are needed for every observable rule here; ADR-011 owns the format.

- **Declarations**: each form; each field accepted and refused per D1 item
  3; path defaults (D4 item 4); unknown field.
- **Expressions**: `when` true, false, `nil`; `for_each` over an array, an
  empty array, `nil`, a non-array; filters in `for_each`; per-item `when`;
  a nested `for_each` refused.
- **Mirror**: `.liquid` stripped; `_` files and directories excluded at
  depth; a file named by `template` excluded; empty mirror warning; mirror
  colliding with an explicit output.
- **Kinds**: each table row, case-insensitive extension, file-name rows,
  `kind` override.
- **Containers**: entry order with and without `first`; `stored`; an empty
  entry stored; nesting to depth 3 and refusal at 4; `for_each` container;
  entries not written standalone; 65,535 planned entries refused (under a
  lowered `outputs` limit only where the default would be costly).
- **Profiles**: each profile's written `mimetype` (exact bytes, method,
  position, no extra field, no data descriptor), required entries missing,
  author `mimetype` refused, `ooxml` non-ASCII refused, EPUB space warning.
  A real minimal `.docx`, `.odt` and `.epub` among the vectors, each opened
  by its format's checker once when the vector is authored (the checker is
  not part of `go test`).
- **CRLF**: LF, CRLF, lone CR, mixed, embedded raw input.
- **Validation**: valid, invalid with several errors (all reported), parse
  failure, wrong kind refused at check.
- **Paths**: every character category and reserved character, U+FFFD,
  a zero-width joiner, a bidirectional control, segment end, every
  reserved name with and without extension and superscript, `.git` and
  `GIT~1`, length boundaries (255/256 bytes per segment, 512/513 bytes and
  32/33 segments per path), non-NFC, unassigned code point, collisions by
  case, by NFC/NFD, by full folding (`ß`/`ss`) and between directory
  prefixes (`Docs/a`, `docs/b`), file–directory conflict.
- **Destination**: absent, empty, non-empty refused, `--overwrite` replace
  and create (an identical existing file is replaced), a link at a segment,
  a case variant on disk, a directory at a target, a leftover staging
  directory refused, nothing deleted.
- **Dry run** listing and exit status; **archive** entries and fields,
  long and non-ASCII paths; **`--only`** to standard output and to a
  directory, an unknown path.
- **Determinism**: cross-platform and repetition (D19 item 2).

Atomicity under a killed process (D14) is not a vector: it is a test of the
reference tool that kills the render at each write step and asserts the
D14 states.

## Alternatives rejected

### Container entries as references to top-level output ids

Entries would name other outputs, which would then be "consumed" and not
written. That needs a global reference graph, cycle and double-consumption
rules, and an output whose meaning depends on whether something else names
it. Inline entries reuse the same declaration forms, scope their paths to
the container, and make "also write it standalone" an explicit second
declaration, which is what R9 item 2 asks for.

### A separate expression language for `for_each` and `when`

JSONPath, CEL or a METAIS mini-language would give authors a second syntax
and a second set of truthiness rules beside Liquid's. Liquid's own condition
and value expressions already exist in every conforming engine.

### Normalizing paths to NFC instead of refusing

It would make the written name differ from the template's output, and two
implementations would have to agree on when it happens. Refusing keeps a
path exactly what the template produced and points the supplier at the
data.

### Simple case folding, or ASCII-only case comparison

ASCII-only (OPC's rule) misses `É`/`é`, which NTFS and APFS treat as one name.
Simple folding misses `ß`/`ss` where a filesystem uses full folding. D145
with full folding is the strictest common rule and is EPUB's own, so a
package passing it is valid there too.

### Staging every render in memory and writing with no staging directory

A render into a new directory would then be visible half-written to a
concurrent reader, and a kill would leave a partial tree that looks like a
result. The staging directory costs one rename.

### Staging inside the output directory

A staging subdirectory in the output would be an entry the next render sees,
could collide with a planned path, and gives no atomicity for an existing
directory anyway. Beside it, in the parent, it is invisible to the output and
guaranteed on the same filesystem.

### fsync before every rename

See D14 item 3: the cost on darwin is per file and large, and the output is
reproducible.

### `--now` as the archive timestamp

It would make the archive bytes depend on a caller argument that exists for
template content, and two renders of the same data a second apart would
differ.

### ustar only, refusing paths that do not fit

ustar cannot carry a non-ASCII path portably, and its 100-byte name limit is
far below D12's. pax extends exactly those cases and every current tar reads
it.

### A profile that also validates the XML of parts

It would need XML parsing and per-format schemas inside the tool, a new
dependency surface, for checks the format's own validators already do after
the render.

## Consequences

### Positive

- An OOXML, ODF or EPUB file comes from templates and one `profile` line, with
  the format's packaging traps (the `mimetype` bytes, its position, method
  and header) handled by the tool.
- A package that renders on one platform renders the same set of files, with
  the same names, on every other, including Windows and case-insensitive
  filesystems.
- A refused render never touches the filesystem, and a render into a new
  directory is never seen half-done.
- The archive and raw modes give pipelines the same bytes a directory render
  writes.

### Negative

- Rendered text is held in memory until emission, bounded by ADR-009's
  limits; a package whose templated outputs exceed them cannot render, since
  no limit can be raised (ADR-009 D4).
- An `--overwrite` render killed mid-commit can leave a mixed tree; only the
  new-directory path is atomic.
- Some legal names on a single platform (`aux.c`, `Con.md`, a name ending in
  `.`, a private-use character) are refused everywhere.
- Characters assigned after Unicode 15.0.0 are refused in paths until a
  later 1.x pins a later version.
- Every existing file a render writes over is replaced, even when its bytes
  would not change, so its timestamp moves (D15 item 5).

### Neutral

- A file in an existing output directory that the render does not write is
  never reported or removed; cleaning stale outputs is the caller's.
- Container conformance compares entries, so another implementation may use
  another deflate compressor.

## What this does not decide

- The flag names and JSON envelope (ADR-008); finding codes and severities
  (ADR-006); limit defaults (ADR-009).
- Escaping semantics, the provenance comment per kind, and ADR-004's
  variables beyond `item`; an `index` variable for a `for_each` evaluation
  is not added here.
- Line-ending conversion to LF, XML or CSV output validation, and other
  container formats: each is a possible 1.x addition.
- Validation of the XML content of OOXML, ODF or EPUB parts.
- Removing outputs a previous render wrote (ruled out: no updating of
  rendered output).

## Proposed rulings

1. **Container entries are inline declarations**, not references to
   top-level ids. Lean: inline. Why: one declaration grammar, paths scoped
   to the container, no reference graph. Alternative: entries name output
   ids that are then not written on their own.
2. **Containers take an optional `profile`** (`ooxml`, `odf`, `epub`) that
   applies the format's settings, writes `mimetype` itself for `odf` (from
   `container.mimetype`) and `epub`, and verifies required entries. Lean:
   yes. Why: the `mimetype` trailing-newline trap breaks both formats
   silently. Alternative: the specification documents the settings and the
   author sets `first`/`stored` by hand.
3. **`for_each` is a Liquid value expression with filters; `when` a Liquid
   condition, evaluated per item.** Lean: yes. Why: no second language, and
   per-item `when` covers filtering without a new field. Alternative: `when`
   evaluated once per declaration, filtering only via filters.
4. **The output kind comes from a closed extension and file-name table,
   overridable by `kind`.** Lean: yes. Why: zero configuration for the
   common case, deterministic. Alternative: `kind` always required.
5. **The mirror excludes `_`-prefixed names and every template an explicit
   output names, strips `.liquid`, and never renders file names.** Lean:
   yes. Why: lets a partial and a fanned-out template live in a mirrored
   tree. Alternative: partials only outside the mirrored directory.
6. **One portable-path predicate for outputs, container entries and package
   entries: categories L, M, N, P, S plus U+0020 of Unicode 15.0.0 (which
   refuses controls, format and bidirectional characters, private use and
   unassigned code points), minus Windows's reserved characters and
   U+FFF0–U+FFFF, device names including `COM0`/`LPT0`, `.git`, NFC only;
   collisions by D145 with full folding over paths and directory
   prefixes.** Lean: yes. Why: one strict rule that every target
   filesystem and EPUB accept, stable across Unicode versions. Alternative:
   Windows's list only, ASCII case comparison, NFC normalization applied
   silently.
7. **Lengths: 255 bytes per segment; the whole path within ADR-009's
   `path-length` (512 bytes) and `path-depth` (32).** Lean: yes. Why: EPUB's
   segment limit; one path bound, ADR-009's, without Windows's legacy 260.
   Alternative: 240 characters total for legacy Windows programs.
8. **Atomicity: a new output directory is staged beside it and renamed;
   an existing one is prepared then committed per file; no fsync.** Lean:
   yes. Why: atomic where it can be, honest where it cannot, fast.
   Alternative: fsync each file and directory before commit.
9. **`executable: true` on template and asset outputs.** Lean: include.
   Why: a repository scaffold (PRD §4.5) with a non-executable `gradlew` or
   build script is broken on arrival, and the field is deterministic.
   Alternative: defer to a 1.x addition; every file is `0644`.
10. **Destination: any entry, hidden files included, makes a directory
    non-empty; any link beneath it refuses; a case-variant existing name
    refuses; an existing file is replaced without its content being read.**
    Lean: yes. Why: reading it to detect "unchanged" would make the status
    an equality oracle over a file the supplier cannot read (ADR-009 D8 item
    4). Alternative: ignore `.DS_Store`-style files; follow links that stay
    inside the tree; leave byte-identical files `unchanged`.
11. **The standard-output archive is pax with fixed headers (mode
    `0644`/`0755`, uid/gid 0, empty names, mtime 315532800), regular files
    only, path order, compared by entries in conformance.** Lean: yes.
    Alternative: byte-exact archive conformance, which would bind other
    implementations to one pax-header naming scheme.
12. **One output is selected by its rendered path with ADR-008's `--only`**,
    after a full render: raw on standard output with `--out -`, or alone in
    a directory with `--out <dir>`. Lean: yes. Alternative: render only the
    selected file, which would let a selected render succeed where a full
    one is refused; or select by declaration id, which is ambiguous for a
    `for_each` declaration.
13. **Container nesting to depth 3.** Lean: yes. Why: a submission zip
    holding a `.docx` is a real case; deeper is not. Alternative: no
    nesting.
14. **`newline` offers `crlf` only, idempotent over existing CRLF.** Lean:
    yes. Alternative: also `lf`.
15. **Output schema validation is for `json` and `yaml` outputs only** in
    1.0, with schemas anywhere in the package (recommended `references/`).
    Lean: yes. Alternative: a dedicated `schemas/` directory (an ADR-001
    layout change).
16. **Staging and temporary names are fixed (`.metais-staging-<h>`,
    `.metais-tmp-<h>`, `<h>` from SHA-256 of a name) and created
    exclusively.** Lean: yes. Why: no random source or process identity is
    read (ADR-009 D12), and a leftover is reported rather than silently
    accumulated. Alternative: random suffixes, which need a random source
    the isolation tripwire forbids.
17. **`id` uses the input-name rule `^[a-z][a-z0-9_]{0,63}$`.** Lean: yes.
    Why: one identifier syntax per manifest. Alternative: allow hyphens in
    ids only.

## References

- Microsoft, *Naming Files, Paths, and Namespaces*:
  https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file
- Microsoft, *MoveFileExW function*:
  https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-movefileexw
- W3C, *EPUB 3.3*, §4.2.3 File names, §4.2.6 META-INF, §4.3.2 ZIP file
  requirements, §4.3.3 media type identification:
  https://www.w3.org/TR/epub-33/
- OASIS, *Open Document Format for Office Applications (OpenDocument)
  Version 1.3, Part 2: Packages*, §2.2.1, §3.2, §3.3, §4.3:
  https://docs.oasis-open.org/office/OpenDocument/v1.3/os/part2-packages/OpenDocument-v1.3-os-part2-packages.html
- Ecma International, *ECMA-376 Part 2, Open Packaging Conventions*, 5th
  edition (December 2021), §6.2.2, §7.3.3–§7.3.7, Annex C:
  https://ecma-international.org/publications-and-standards/standards/ecma-376/
  (PDF: https://ecma-international.org/wp-content/uploads/ECMA-376-2_5th_edition_december_2021.zip)
- The Open Group, POSIX.1-2024, *pax* (ustar header, pax extended headers,
  end of archive):
  https://pubs.opengroup.org/onlinepubs/9799919799/utilities/pax.html
- Unicode Consortium, *The Unicode Standard 15.0*, Chapter 3, §3.13
  Default Caseless Matching (D144, D145):
  https://www.unicode.org/versions/Unicode15.0.0/ch03.pdf
- Unicode Consortium, *Unicode Character Encoding Stability Policies*
  (normalization and case folding stability):
  https://www.unicode.org/policies/stability_policy.html
- Linux `rename(2)` / `renameat2` (`RENAME_NOREPLACE`):
  https://man7.org/linux/man-pages/man2/rename.2.html
- Apple XNU `rename(2)` (`renamex_np`, `RENAME_EXCL`):
  https://opensource.apple.com/source/xnu/xnu-4570.61.1/bsd/man/man2/rename.2.auto.html
- Go `os.Root`: https://github.com/golang/go/blob/go1.26.9/src/os/root.go;
  Go 1.24 and 1.25 release notes: https://go.dev/doc/go1.24,
  https://go.dev/doc/go1.25
- Go `os.Rename` on Unix:
  https://github.com/golang/go/blob/go1.26.9/src/os/file_unix.go
- Go darwin `Fsync` (`F_FULLFSYNC`):
  https://github.com/golang/go/blob/go1.26.9/src/internal/poll/fd_fsync_darwin.go
- Go Windows long paths (`fixLongPath`):
  https://github.com/golang/go/blob/go1.26.9/src/os/path_windows.go
- Go `archive/zip` writer (`CreateHeader`, `CreateRaw`):
  https://github.com/golang/go/blob/go1.26.9/src/archive/zip/writer.go
- Go `archive/tar` writer (format selection, pax records):
  https://github.com/golang/go/blob/master/src/archive/tar/writer.go
- Go `unicode` table versions:
  https://github.com/golang/go/blob/go1.26.0/src/unicode/tables.go,
  https://github.com/golang/go/blob/go1.27.0/src/unicode/tables.go
- `golang.org/x/text` v0.43.0 table selection by build tag (`tables15.0.0.go`
  under `!go1.27`, `tables17.0.0.go` under `go1.27`), read 2026-10-18:
  https://github.com/golang/text/blob/v0.43.0/unicode/norm/tables15.0.0.go,
  https://github.com/golang/text/tree/v0.43.0/cases

## Amendments

### A1 — An existing output directory is prepared beneath one staging directory inside it

**Status:** Accepted 2026-10-19 by Ed Funnekotter, in design session
`01a15284-df6e-7668-b799-62f4c380a84f` (PLAN-2026-10-metais-v1, W2-A1).
**Amends:** D14 item 2 (prepare and commit), D14 item 5 (what the render
removes), D15 item 4 (two further per-file checks), and the alternative
rejected as *Staging inside the output directory*, which now argues D14
item 1 only. Amends ADR-009 D8 items 1 and 5 to match, and is
cross-referenced from ADR-009's implementation log. ADR-008 reuses the
overwrite rule for `extract --overwrite` and `sample --to`, so A1 is how
they write into an existing directory too.

1. **Prepare.** A render into an existing directory (D14 item 2) first
   creates one staging directory inside it, `<out>/.metais-staging`,
   exclusively, and holds it open for the rest of the render. An existing
   one — a killed render's leftover, or a concurrent render into the same
   directory — refuses the render with `output.destination.write_failed`,
   reason `exists`, nothing written. Every planned file is written beneath
   the staging directory at its own path, with every directory that path
   needs, through an `os.Root` on it. Nothing is created in the existing
   tree while preparing. Any failure removes the staging directory and
   refuses: nothing changed.
2. **Commit units.** A planned path's commit unit is its shallowest
   component that did not exist in the output directory at the check: a
   missing directory, committed whole with everything planned beneath it,
   or the file itself when its parent exists. Each unit is committed once,
   in path order, by renaming it from the staging directory into place: a
   missing directory and a `create` with D14 item 1.4's no-replace rename,
   a `replace` with D14 item 2.2's replacing rename. A directory that
   appeared since the check refuses its unit's rename, which is a failure
   during commit. After the last unit, the staging directory holds only
   directories the render created there, and they and it are removed.
3. **A failure during commit** is D14 item 2.3: the commit stops, the
   staging directory and everything still in it are removed, and the
   render is refused with `output.destination.commit_failed`, listing the
   paths already committed. A path is committed when its unit was renamed.
   A process killed during commit leaves a mix of whole old and whole new
   files, and the staging directory, which the next render into the same
   directory refuses (item 1).
4. **One filesystem.** Every rename stays on one filesystem, because D15
   item 4 refuses, before anything is written, a planned file whose
   deepest existing directory is on a different device from the output
   directory — a mount point inside it — with
   `output.destination.write_failed`, reason `io_error`, nothing written.
   The device is compared and never reported: it decides the refusal and
   reaches no finding, output or message (ADR-009 D12). A conformance
   vector cannot make a mount point without privilege, so this case is a
   unit test of `internal/hostfs`, as ADR-009's device cases are.
5. **The staging name is the tool's.** `.metais-staging` is one fixed name
   in every output directory, since one output directory has one staging
   directory; it reads no random source and no process identity (proposed
   ruling 16). D15 item 4 refuses a planned path whose first segment is an
   ASCII-case-insensitive match for it, with
   `output.destination.write_failed`, reason `exists`, nothing written. A
   render into a new directory (D14 item 1) stages beside it and reserves
   nothing inside it. D14 item 2 writes no temporary file beside a target.
6. **One ownership rule** (D14 item 5). The render removes only what lies
   within a directory it created exclusively and holds open — the staging
   directory of either case, and the ancestors D14 item 1.1 creates — and
   beneath a staging directory only a name it created, while that name
   still holds the kind it created, a directory only when it is empty.
   Nothing outside a staging directory is created except by a commit
   rename, so a failure the render survives leaves nothing it made in the
   caller's tree. The one declared gap is the one the Wave 4 ruling
   accepted for D14 item 1: a foreign file swapped in at a name the render
   created beneath its own hidden staging directory is removed.
7. **Bounded handles.** A render holds the staging directory, the
   ancestors it created, the roots it opened and the file it is writing,
   however many files it plans. The host's open-file limit therefore never
   refuses a render within ADR-009's limits, and no handle is held per
   planned file.
8. **What does not change.** D14 item 2's guarantee — each file whole, the
   set whole against every failure but a crash during commit — and D15's
   checks, policy and refusals other than items 4 and 5 here. A1 moves
   where files are prepared and how ownership is proved, not what a caller
   is promised. D14 item 1 keeps its staging directory beside the output
   directory, which gives that case its atomicity and puts nothing in the
   output; the rejected alternative's argument stands for it. For an
   existing directory that argument does not apply: there is no atomicity
   to lose, a leftover is refused by name as a temporary file was, item 5
   reserves the name, and item 4 keeps every rename on one filesystem.

## Implementation log

- **2026-10-19, PLAN-2026-10-metais-v1 W2-5.** D14 and D15's library half
  landed in `internal/hostfs`, ahead of the renderer that composes it:
  `StagingName` and `TempName` derive D14's fixed names from a SHA-256 of
  the base or file name; `Writer` creates every file and directory
  exclusively, with D14 item 4's modes, records what it created, and
  `RemoveCreated` removes only those records, newest first, leaving a
  name that now holds something else and a directory that is not empty;
  `RenameNoReplace` is D14 item 1.4's rename, with its `Lstat` fallback
  where the file system refuses the flag, and `RenameReplace` D14 item
  2.2's. A rename that succeeds takes what it moved out of the record, so
  a committed output is never the render's to remove.
- **2026-10-19, PLAN-2026-10-metais-v1 W2-5F.** D14 item 5's "only what
  this render created" is proved by a held handle, not by a recorded device
  and inode: a file system may give a deleted file's inode number to the
  next file created (ext4 does), so a newcomer at a created name compared
  the same as the render's entry. `Writer` now holds each entry it creates
  open until the entry is removed or committed or the Writer closes, and
  `RemoveCreated` compares the name with the held entry's own `Stat`. A
  render therefore holds one handle per entry it has created and not
  committed, which ADR-009's `outputs` limit bounds and the host's
  open-file limit may refuse first. On windows the handles beneath a
  directory are released before it is renamed, and those records fall back
  to the NTFS file id, whose sequence number is never reused.
- **2026-10-19, PLAN-2026-10-metais-v1 W4-0.** `internal/hostfs` gives
  this wave's writers their write shapes, `Output`: `StageNew` is D14
  item 1 (render into a new directory, and unpack into a target that does
  not exist); `WriteInPlace` is ADR-007 D8 item 3's in-place write into an
  existing empty directory; `CreateNew` is ADR-008 D5's extract into a
  directory that does not exist. Only the staging directory and the
  ancestors `StageNew` creates are pinned, so a write holds a bounded
  number of handles whatever its entry count, and the host's open-file
  limit no longer refuses a write within ADR-009's limits. The in-place
  shapes remove nothing they wrote; a file whose bytes fail part way is
  removed while the handle that wrote it is open. D14 item 4's modes are
  kept under a umask that withholds an owner bit: `os.Root` reaches a name
  by opening the directories above it for reading, so a directory created
  without the owner's bits gets them for the write and its own mode back
  at the end. Beneath `StageNew`'s staging directory nothing is pinned:
  by the Wave 4 wave orchestrator's ruling, following item 1.5, the
  staging directory is the ownership boundary, and `Abort` removes beneath
  it only names the write created, while each holds the kind created. A
  foreign file swapped in at such a name is removed; that gap is declared.
- **2026-10-19, PLAN-2026-10-metais-v1 W4-3.** `metais render --out
  <dir>` of template outputs, each path given or defaulted (D4 item 4),
  runs D13's phases 0 to 5 with every output attempted, and commits a
  directory that does not exist with `hostfs.StageNew` (D14 item 1). D12's
  predicate applies to every planned path, each alone and the tree's paths
  together, through `hostfs.CheckPaths`, so that `fault.SkipConfine`
  removes both layers of the output path check — the predicate that
  confines a path by its text (D12 item 8) and the root that confines the
  write — and the confinement detector can see an escape through a render.
  A path the manifest writes that breaks it is one of the `output.path.*`
  codes of this ADR's interfaces; beyond `path-length` or `path-depth` it
  is `limit.package`, the manifest being the package's own content. Until
  existing destinations land (D14 item 2, D15, Wave 5), an `--out` that
  exists is refused before anything is written: a non-directory, or a
  path beneath one, with `output.destination.not_directory`; a directory
  with entries with `output.destination.not_empty`, which is D15 item 3's
  refusal without `--overwrite`; and an empty directory with
  `output.destination.write_failed`, reason `exists`, nothing written —
  the no-replace creation of the output directory is the write this
  render makes, and it cannot be made over an existing name. A declaration
  this wave does not render — an asset, a mirror or a container, `when`,
  `for_each`, `escape: html`, `newline: crlf`, `validate`, or a path that
  holds Liquid markup — and `--out -` end with `internal.error`, since
  the tool does not yet implement them; Wave 5 does. A write that fails
  — the runner's unwritable-destination pre-state, W4-5's, meets render in
  `rnd/destination-unwritable` — is `output.destination.write_failed`
  with nothing left: the staging directory is never made, or `Abort`
  removes it.
- **2026-10-19, PLAN-2026-10-metais-v1 W5-4.** `render` plans `for_each`,
  `when` and path templates for template outputs (D3, D4, D13 phase 1),
  and their `internal.error` stops are gone. Every `path` is rendered as a
  Liquid template with the template environment's dialect and no escaping;
  a path with no markup renders as itself. The readings this item took:
  - **A wrong `for_each` type** is `output.expression.for_each_type`, in
    the `output.expression` family this ADR's interfaces name, with the
    type found (`string`, `number`, `boolean` or `object`) as `type`.
    Every other error of a `when`, a `for_each` or a path is the template
    family's code a template body's error has (ADR-004 D16 item 5), at the
    field's pointer with the column within the expression. Their parse
    errors are refused when the package is opened, as MTS-MAN-018 states,
    so render meets only evaluation errors there.
  - **D4 item 2's "the evaluation (`item` index)"** is an optional `item`
    parameter, the 0-based index, on every `output.path.*` code.
  - **A path template's path beyond `path-length` or `path-depth`** is
    `limit.exceeded.output`, not `limit.package`: it is the render's, not
    the manifest's own text (ADR-009 D19). A path's bytes are held to
    `path-length` as they are rendered, so a path template cannot build an
    unbounded name; that breach has no `observed`.
  - **`outputs` counts every evaluation's file** (ADR-009 D3). A fan-out
    stops being evaluated once the plan holds one file beyond the bound;
    when that file is a fan-out's the breach is `limit.exceeded.output`
    about its declaration, and otherwise `limit.package` as before, with
    `observed` only when every file was counted.
  - **Each `for_each` element is one `loop-iterations` iteration**, as
    ADR-009 D3's table says, charged before the first evaluation; the
    manifest's expressions and the templates share one meter.
  - Every template a template output names is parsed whether or not the
    output plans a file, so a `for_each` over nothing still reports its
    template's defects.
  `when` on an asset or a mirror, and `for_each` and `when` on a container
  are W5-5's and W5-6's meetings with this item.

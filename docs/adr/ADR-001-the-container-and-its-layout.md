# ADR-001: The container and its layout — a canonical zip with every header field fixed, a closed top-level layout, and a content-defined package digest that `.compiled/` carries and every open verifies

**Status:** Proposed
**Date:** 2026-10-18
**Deciders:** Ed Funnekotter
**Designs:** PRD-001 R1 (items 1 to 5); the container side of R2 item 1
(read in place), R13 items 4 and 5 (deterministic pack, unpack) and R14
items 1 to 3 (confined, bounded, refused not approximated)
**Depends on:** ADR-002 (the manifest's spec version, its canonical JSON
form, the derivation of `.compiled/manifest.json`); ADR-003 (the derivation
of `.compiled/schema.json`); ADR-005 (the portable-path predicate, and the
zip container outputs that reuse this ADR's writer); ADR-006 (the finding
model and code spelling); ADR-008 (the references index's content, and how
commands treat a source directory); ADR-009 (every resource limit and its
default); ADR-010 (the zip, deflate, normalization and case-folding code);
ADR-011 (the vector format)

## Context

### What the PRD requires

R1 makes a package one zip file, `<name>.metais`, holding the manifest, the
data specification, the templates and optionally references, assets and
examples, in a layout the specification fixes (item 1). A minimal package is
a manifest, one schema and one template (item 2), and nothing outside the
package is needed to use it (item 3). A packed package carries the manifest
in canonical JSON, one combined schema, a SHA-256 digest of every entry and of
the package, and a references index (item 4). The tool verifies those digests
when it reads a packed package and refuses one that does not match (item 5).

R2 item 1 has every part of a package listed, read, outlined and searched
without unpacking it to disk. R13 item 4 makes a packed package's bytes
depend only on the source directory; item 5 unpacks a package back into an
editable source directory. R14 confines every entry, bounds every resource,
and refuses a package outside the specification with a finding that names
what is wrong. R9 item 5 has the tool's own zip container outputs
byte-identical across platforms and runs, and compares containers in the
vectors by entries, not bytes.

The rulings fix the extension `.metais`, the source manifest `metais.yaml`,
the layout (`metais.yaml`, `inputs/`, `templates/`, optional `references/`,
`assets/`, `examples/<name>/{inputs,expected}/`, and `.compiled/` written by
pack only), SHA-256, deterministic packing with sorted entries and fixed
timestamps, and no signing in v1.

### What the zip format allows, and what readers do with it

Checked against PKWARE's APPNOTE 6.3.10 and Go's `archive/zip` source
(go1.26.8) on 2026-10-18:

- **The format is permissive in exactly the places an attacker uses.**
  Central directory entries need not be in file order (APPNOTE 4.4.1.3).
  Each entry's name, sizes, CRC and flags appear in both the local header
  and the central directory header (4.3.7, 4.3.12), and with general-purpose
  bit 3 a third copy of the CRC and sizes follows the data in a data
  descriptor, with or without a signature (4.3.9.1, 4.3.9.3). The
  Info-ZIP Unicode Path extra field (0x7075) carries a second file name
  (Appendix D.6, D.7). Compression methods run from 0 to 99 (4.4.5), and
  bit 0 marks encryption (4.4.4). The end-of-central-directory record ends
  with a comment of up to 65,535 bytes (4.3.16).
- **Names**: no drive letter, no leading slash, forward slashes only
  (4.4.17.1). Bit 11 means the name and comment are UTF-8; without it they
  are CP437 (Appendix D.2).
- **Version needed to extract**: 1.0 by default, 2.0 for a deflated entry or
  a directory, 4.5 for zip64 (4.4.3.2).
- **Timestamps** are MS-DOS date and time, relative to 1980 with two-second
  precision and no time zone (4.4.6).
- **Go's reader checks little of this.** `zip.NewReader` reads only the
  central directory. It locates the end record by scanning the last 1 KiB,
  then 65 KiB, for its signature, and tolerates bytes before the archive by
  computing a base offset (`readDirectoryEnd`). Opening an entry reads the
  local header only for its signature and its name and extra lengths
  (`findBodyOffset`), so a local header that disagrees with the central
  directory is never noticed. It never checks that entries do not overlap.
  It skips the CRC check when the recorded CRC is zero and there is no data
  descriptor (`checksumReader.Read`). It does not look at bit 0, so an
  encrypted entry decompresses to garbage rather than failing. Duplicate
  names are reported only through its `fs.FS` view (`initFileList`); the
  `File` slice holds both.
- **Go's path check is environment-controlled.** `NewReader` returns
  `ErrInsecurePath` for a non-local or backslashed name only when the
  `GODEBUG` variable carries `zipinsecurepath=0`; the default is `1`, which
  allows such names (Go's GODEBUG history). A check that the environment can
  switch off is no check for an untrusted package.
- **Go's writer is not deterministic in the ways a package needs.**
  `Writer.CreateHeader` always sets bit 3 and writes a data descriptor for a
  file, sets bit 11 only when a name needs multibyte UTF-8, forces both
  version fields to 2.0, and appends an extended-timestamp extra field
  (0x5455) whenever `Modified` is set. `Writer.CreateRaw` writes the header
  exactly as given, takes already-compressed bytes, and writes the CRC and
  sizes in the local header when bit 3 is clear. That is the entry point a
  canonical writer needs.
- **Deflate output is not stable across Go releases.** Go commit 388145c5
  (2026-04-15) replaced the encoder behind compression levels 2 to 9, and
  commit 2747d887 (2026-05-07) added to `flate.NewWriter` and
  `zip.NewWriter`: "the exact bytes written to w are not covered by the Go 1
  compatibility promise". Byte-identical packages are therefore a property
  of one tool build, not of the format.
- **Overlapping entries are the modern zip bomb.** Fifield's non-recursive
  zip bomb (USENIX WOOT 2019) makes many central directory entries share one
  compressed kernel; Info-ZIP's fix for CVE-2019-13232 rejects an entry that
  overlaps another or the central directory.
- **The formats ADR-005 assembles constrain the same fields.** EPUB 3.3
  allows only stored and deflated entries, forbids the zip encryption
  features, requires UTF-8 names, uses zip64 only when the content requires
  it, and requires the `mimetype` file first, stored, with no extra field
  (§4.3.2, §4.3.3). ODF 1.3 Part 2 allows only stored and deflated entries
  (§2.2.1) and requires `mimetype` first, uncompressed, with no extra field
  (§3.3).

### Media type

RFC 6838 lets anyone register a type in the vendor tree, by submission to
IANA with expert review (§3.2), and strongly discourages the unregistered
`x.` tree, whose types are for use only with the active agreement of the
parties (§3.4). A type that uses a structured syntax should carry its
registered suffix, and must not carry one it does not use (§4.2.8). RFC 6839
registers `+zip` for any type whose representation follows `application/zip`
(§3.6). IANA's application registry, read on 2026-10-18, holds no type
naming METAIS; it holds 39 `+zip` types, among them `application/epub+zip`.
A web search on the same day found no other use of the `.metais` extension.

## Decision

### D1 — A package is a canonical zip: every header field is fixed except the compressed data, its CRC and its sizes

1. **The reading profile and the writing profile are one profile.** A
   packed package is accepted only if every field this decision fixes has
   exactly the fixed value. A package can only be produced by a packer,
   because `.compiled/` must match its sources (D7), so the strictness costs
   no author anything, and each fixed field is a place where a hostile
   package cannot hide a disagreement and a second implementation cannot
   drift.
2. **Structure.** The file is a sequence of entries, then the central
   directory, then one end-of-central-directory record, with no other
   bytes:
   - The first local file header starts at offset 0. Nothing precedes it:
     no self-extractor stub, no polyglot prefix.
   - Each entry is a local file header immediately followed by its file
     data. The next local header starts at the byte after the previous
     entry's data. There is no encryption header, no data descriptor, no
     archive decryption header, no archive extra data record and no
     digital signature record.
   - The central directory starts at the byte after the last entry's data,
     holds one header per entry in the same order as the entries, and ends
     at the byte where the end record starts.
   - The end record is the last 22 bytes of the file: both disk numbers 0,
     both entry counts equal to the number of entries, the central directory
     size and offset equal to its actual size and offset, and comment length
     0. The archive carries no comment.
   - There is no zip64 end-of-central-directory record or locator, no zip64
     extra field, and no field holding the zip64 sentinel 0xFFFF or
     0xFFFFFFFF (D3).
3. **Local file header**, for every entry:

   | Field | Value |
   |---|---|
   | version needed to extract | 10 (0x000A) for a stored entry, 20 (0x0014) for a deflated one |
   | general purpose bit flag | 0x0800: bit 11 (UTF-8) set, every other bit clear |
   | compression method | 0 (stored) or 8 (deflated) |
   | last mod file time | 0x0000 |
   | last mod file date | 0x0021 (1980-01-01) |
   | crc-32 | the CRC-32 of the uncompressed content |
   | compressed size | the size of the file data |
   | uncompressed size | the size of the content |
   | file name length | the name's length in bytes, at least 1 |
   | extra field length | 0 |

4. **Central directory header**, for every entry: the same values as its
   local header for every field the two share, byte for byte including the
   name, and:

   | Field | Value |
   |---|---|
   | version made by | 0x0314: Unix (3) as the attribute host, specification 2.0 |
   | file comment length | 0 |
   | disk number start | 0 |
   | internal file attributes | 0 |
   | external file attributes | 0x81A40000: a regular file, mode 0644, in the high 16 bits |
   | relative offset of local header | the entry's actual offset |

   The Unix host with an explicit regular-file mode is chosen so that every
   common extractor gives the file ordinary permissions; a zero mode under a
   Unix host extracts as mode 0000, and a link or directory mode is refused
   (D2 item 6). A package carries no file mode of its own: an output that
   must be executable says so in the manifest (ADR-005).
5. **File data.** A stored entry's compressed size equals its uncompressed
   size. A deflated entry is one raw DEFLATE stream (RFC 1951) that decodes
   to exactly the uncompressed size and whose final block ends exactly at
   the compressed size: no bytes after the stream, and no stream that ends
   before its region does. The CRC-32 of the decoded bytes equals the
   recorded CRC whatever its value, zero included. An empty entry is stored.
6. **Encryption is refused** in every form: bit 0 or bit 6 set, a method
   other than 0 and 8 (method 99 marks AES), bit 13 set. Each is already
   outside items 3 and 4; it is named here because it is the refusal a
   reader most often gets wrong.
7. **Entries are in ascending order of their names' UTF-8 bytes**, which also
   makes them unique (D2 item 4). Byte order equals code-point order.

### D2 — An entry name is a relative, portable, NFC, UTF-8 file path

1. **Encoding.** The name is valid UTF-8 with no byte order mark, and is in
   Unicode Normalization Form C. Bit 11 is always set (D1 item 3), so there
   is no CP437 reading of any name.
2. **Shape.** Segments separated by `/`. No leading `/`, no trailing `/`, no
   empty segment, no `.` or `..` segment, no `\`, no `:` (which rules out a
   drive letter and an NTFS alternate stream).
3. **Characters.** Every code point is U+0020 SPACE or belongs to one of the
   general categories L, M, N, P or S as Unicode 15.0.0 assigns them. This
   refuses controls (Cc), format characters such as bidirectional overrides
   and zero-width joiners (Cf), private use (Co), unassigned code points and
   noncharacters (Cn), and line, paragraph and non-ASCII space separators
   (Zl, Zp, Zs other than U+0020). Invisible and direction-changing
   characters in a name are how a listing lies to the person or the harness
   reading it, and pinning the Unicode version makes the rule the same in
   every implementation: Unicode's stability policy keeps a character's
   decomposition and its caseless matching unchanged once it is assigned, so
   a name built only from assigned characters means the same under every
   later version.
4. **Uniqueness.** Take the set of every name and every directory prefix of
   every name (`a/b/c` contributes `a`, `a/b` and `a/b/c`). No two distinct
   members of that set are equal under Unicode simple case folding
   (CaseFolding.txt statuses C and S, Unicode 15.0.0). This refuses
   byte-equal duplicates, `Readme.md` beside `README.md`, and `Docs/a` beside
   `docs/b`, which are two directories on a case-sensitive file system and
   one on a case-insensitive one. A name equal to another name's prefix
   (`a/b` and `a/b/c`) is refused too: one is a file, the other a
   directory.
5. **Portability.** The name satisfies the portable-path predicate ADR-005
   states for output paths: no Windows reserved device name in any segment,
   no reserved character, no segment ending in `.` or space. Entry names and
   output paths obey one predicate, so a package that extracts on one
   platform extracts on all of them, and `extract` and `unpack` never meet a
   name they cannot write.
6. **No directory entries.** No name ends in `/`, and every entry is a
   regular file (D1 item 4). Directories exist only as prefixes. A source
   directory holding an empty directory is refused by `check` and `pack`
   with a finding naming it, because the packed form cannot carry it and a
   source directory must mean what its package means: an empty collection
   directory in an example would otherwise be absent after packing.
7. **Length.** The structural ceiling is the field: 65,535 bytes. The limit a
   reader enforces is ADR-009's path-length limit and ADR-005's per-segment
   limit, both far below it.

### D3 — No zip64; the profile's ceilings sit inside 32-bit fields, and ADR-009's limits sit inside the ceilings

1. **The hard ceilings**, which no limit can raise: at most 65,534 entries;
   every size and offset at most 0xFFFFFFFE; a package file of at most
   0xFFFFFFFE bytes.
2. **ADR-009's limits must sit below them**: its default maximum package
   file size, total uncompressed size and per-entry uncompressed size each
   well under 4 GiB, and its default maximum entry count under 65,535. Since
   a caller may only lower a limit, no package that passes the limits needs
   zip64, and zip64 is refused outright (D1 item 2). The same holds for the
   zip container outputs, under ADR-009's output limits.
3. **A later major version may admit zip64** if a real package needs it.
   Within major version 1 the ceilings stand.

### D4 — The writer: one routine, used by `pack` and by every zip container output

1. **Input**: an ordered list of entries, each a name, its content, and a
   method choice of `store` or `auto`. **Output**: the bytes D1 describes,
   written front to back, so the writer works on a stream with no seeking.
2. **Method `auto`** deflates the content and keeps the result only if it is
   strictly shorter than the content; otherwise the entry is stored. An
   empty entry is therefore always stored. The rule depends only on the
   content and the encoder, so it is deterministic.
3. **The encoder** in the reference tool is Go's `compress/flate` at
   `flate.BestCompression` (level 9), the encoder that ADR-010 pins with the
   toolchain. The specification fixes the method and the stream's validity,
   never the compressed bytes, so a second implementation is free to use
   another compressor and still conforms.
4. **Nothing of the host enters the output.** No modification time, owner,
   permission bit, extended attribute or file-system order is read from the
   source; every header field is D1's constant or is computed from the
   content.
5. **The writer computes each entry's CRC-32 and sizes before writing its
   local header**, holding the compressed entry in memory (it is bounded by
   ADR-009's per-entry limit), so no data descriptor is ever needed, even
   when the destination is standard output.
6. **Byte identity.** One build of the tool writes byte-identical output for
   the same entries on every platform and in every run. Two builds may
   differ in deflated bytes when their encoders differ (Context); the
   package digest (D6) does not change, because it is defined over content.
   This is how R13 item 4 and R9 item 5 are met, and the conformance
   vectors compare packages and containers by entries — names, order,
   method and content — and never by file bytes.
7. **For a package**, the entries are in D1 item 7's order and every method
   is `auto`. **For a zip container output**, ADR-005 supplies the full
   entry order (its "entries first" option decides the head; I assume the
   rest follow in ascending byte order of name) and which entries are
   `store` (its "entries stored" option); every other entry is `auto`. Every container therefore also obeys D1's fixed
   fields: no extra field on any entry, no data descriptor, UTF-8 names.
   That satisfies EPUB's and ODF's `mimetype` rules — first, stored, no
   extra field — once ADR-005's recipe lists it first and stored.

### D5 — The layout: a closed set of top-level names

1. **The top level of a package holds only these names:**

   | Name | Kind | Required | Contents, and who defines them |
   |---|---|---|---|
   | `metais.yaml` | file | yes | the source manifest, YAML or JSON (ADR-002) |
   | `inputs/` | directory | no | input schemas and meaning documents (ADR-003) |
   | `templates/` | directory | no | Liquid templates and partials (ADR-004, ADR-005) |
   | `references/` | directory | no | background material, any files |
   | `assets/` | directory | no | static files copied to outputs (ADR-005) |
   | `examples/` | directory | no | `examples/<name>/inputs/` and `examples/<name>/expected/` (ADR-007) |
   | `.compiled/` | directory | yes in a package; refused in a source directory | D7 |

   A directory "exists" when at least one entry name starts with it. Which
   directories a given package needs follows from its manifest — an input
   needs its schema under `inputs/`, an output its template under
   `templates/` — and is `check`'s to verify, not the container's. Every
   path a manifest names lies under one of these directories.
2. **One manifest name.** The source manifest is always `metais.yaml`. An
   author who prefers JSON writes JSON in it: JSON is valid YAML 1.2, so
   ADR-002's parser reads it, and there is never a question of which of two
   manifest files wins.
3. **Below the top level, any name D2 admits is allowed**, dot-files
   included: a site's `assets/.well-known/security.txt` and a repository
   scaffold's `templates/.github/workflows/ci.yml` are legitimate. What a
   reserved directory may hold beyond that is its owner's rule.
4. **In a packed package, a top-level name outside the table is refused**,
   with a finding naming it. If ADR-002 lets a tool read a package that
   declares a later minor specification version than the tool implements,
   a top-level name or a `.compiled/` file that the tool does not know is
   verified by digest (D6) and otherwise ignored, with a warning, because a
   later minor may add one; in a package of a version the tool implements,
   it is refused.
5. **In a source directory**, a top-level name beginning with `.` other than
   `.compiled` is ignored silently: `.git`, `.gitignore`, `.DS_Store`. Any
   other unknown top-level name is not part of the package: `check` reports
   a warning naming it, and `pack` leaves it out. A source directory that
   contains `.compiled/` is refused by every command, naming it, because
   `.compiled/` is written by pack only and a stale copy beside the sources
   would be read as either. A source directory without `metais.yaml` is not
   a package.
6. **A source directory's files** are named as D2 requires, relative to the
   directory with `/` separators, and every file under a reserved directory
   is part of the package. A symbolic link, device, socket or other
   non-regular file anywhere under a reserved directory is refused (ADR-009
   states the confinement rule).
7. **Identifying the form.** A command given a directory reads it as a source
   directory; given a regular file, it reads it as a packed package,
   whatever its extension. A package's first entry is always under
   `.compiled/` (`.` sorts before every letter and digit, and the top level
   is closed), which is what a file-type detector keys on (D9 item 3).

### D6 — The digest file and the package digest

1. **`.compiled/digests.json`** is a JSON object with exactly three members:

   ```json
   {
     "algorithm": "sha256",
     "entries": {
       ".compiled/manifest.json": "<64 lowercase hex>",
       ".compiled/references.json": "<64 lowercase hex>",
       ".compiled/schema.json": "<64 lowercase hex>",
       "inputs/report.schema.json": "<64 lowercase hex>",
       "metais.yaml": "<64 lowercase hex>",
       "templates/report.md.liquid": "<64 lowercase hex>"
     },
     "package": "<64 lowercase hex>"
   }
   ```

   (shown indented; the file is written in ADR-002's canonical JSON form, and
   a reader refuses one that is not in that form). `algorithm` is the
   string `sha256`; v1 defines no other. `entries` has one member for every
   entry of the package except `.compiled/digests.json` itself, keyed by
   entry name, whose value is the SHA-256 of the entry's uncompressed
   content. `package` is the package digest.
2. **The digest list** is the UTF-8 text made of one line per member of
   `entries`, in ascending order of the names' UTF-8 bytes, each line being
   the 64 lowercase hex digits, two spaces, the name, and a line feed
   (U+000A). **The package digest** is the SHA-256 of the digest list's
   bytes.
3. **The package digest depends only on content**: the names and the bytes
   of every entry. It does not depend on compression, compressed bytes, the
   tool build, or entry order in the file. Two conforming packers given the
   same source directory produce packages with the same package digest, and
   the digest of a source directory — computed from the entries pack would
   write, with the `.compiled/` files derived in memory — equals the digest
   of the package packed from it. A render from a source directory and a
   render from its package therefore record the same digest in provenance
   (ADR-004).
4. **The digest list is exactly the untagged text-mode output of GNU
   `sha256sum`** for those files, since D2 refuses the backslash, line feed
   and carriage return that would make `sha256sum` escape a line. Anyone can
   check a package with standard tools: extract it, list the files other
   than `.compiled/digests.json` in byte order, run `sha256sum` over them,
   and hash the result.
5. **The digest is presented as `sha256:<64 lowercase hex>`** wherever the
   tool shows it or a template records it (`inspect`, `package.digest`,
   provenance; ADR-004 and ADR-008). Inside `digests.json` the hex stands
   alone, because `algorithm` already names it.
6. **Ordering caution for implementers.** The digest list and the zip's entry
   order sort by UTF-8 bytes. Canonical JSON may order object members
   differently (RFC 8785 sorts by UTF-16 code units); `entries`' member order
   is whatever ADR-002's canonical form says, and nothing depends on it.

### D7 — `.compiled/`: exactly four files, each a proof rather than a claim

1. **A packed package's `.compiled/` holds exactly these files**, at
   specification version 1.0:

   | File | Contents | Derivation owned by |
   |---|---|---|
   | `.compiled/digests.json` | D6 | this ADR |
   | `.compiled/manifest.json` | `metais.yaml` as canonical JSON | ADR-002 |
   | `.compiled/schema.json` | the combined input schema, internal `$ref`s inlined | ADR-003 |
   | `.compiled/references.json` | the references index | ADR-008 |

   A package missing any of them is refused. A zip with no `.compiled/` is
   not a package; the refusal says so and names `metais pack` as the next
   step.
2. **Every file but `digests.json` is a deterministic function of the
   package's other entries and its declared specification version.** The
   files exist so that a consumer with a zip reader and a JSON parser —
   no YAML parser, no `$ref` resolver, no Markdown parser — can use the
   package. They must therefore say exactly what the sources say.
3. **Within a major version, the derivation of an existing compiled file
   never changes for a package that declares an earlier minor version.** A
   later minor may add a compiled file; a tool derives the set the package's
   declared version defines.
4. **`unpack` never writes `.compiled/`** (D5 item 5), and packing the
   unpacked directory reproduces the same package digest, and, with the same
   tool build, the same bytes.

### D8 — Verification: everything, at open, before any command does its work

1. **When a command opens a packed package, before it reads or prints any of
   the package's content**, the tool runs these steps in order, each bounded
   by ADR-009's limits before the bytes it reads are read:
   1. The file's size is within the package-size limit and D3's ceiling.
   2. The end record is the last 22 bytes and satisfies D1 item 2; its entry
      count is within the entry-count limit.
   3. Every central directory header satisfies D1 item 4 and its name
      satisfies D2; the names are in order and pass the uniqueness rule; the
      sum of declared uncompressed sizes is within the total-size limit; each
      entry's declared sizes are within the per-entry limit and its declared
      ratio within the ratio limit.
   4. Walking from offset 0, every local header is where the previous entry
      ended, equals its central directory header field for field, and the
      central directory begins where the last entry ends. This step is what
      refuses overlapping entries, gaps, hidden data and a name that differs
      between the two headers.
   5. Every entry's data is decoded, its length counted against its declared
      size and stopped one byte past it, its CRC-32 checked, and its SHA-256
      computed (D1 item 5).
   6. `.compiled/digests.json` is parsed; its `entries` keys are exactly the
      set of entry names other than itself; every value equals the computed
      SHA-256; the recomputed package digest equals `package`.
   7. The manifest is read and its specification version checked (ADR-002);
      the layout satisfies D5 for that version; `.compiled/manifest.json`,
      `.compiled/schema.json` and `.compiled/references.json` are derived
      from the sources as that version defines and each is byte-equal to
      the stored file.
2. **A mismatch refuses the package.** The command does nothing else: no
   listing, no partial read, no render. It reports findings (ADR-006) and
   exits with the refusal code ADR-006 assigns. Within one step, every
   independent finding is reported: every mismatched digest, every bad name,
   every misplaced local header. A failure that leaves the entries unknown
   — no end record, an unreadable central directory — ends the run at that
   finding, because later steps would only report its consequences.
3. **There is no flag that skips verification.** A person who needs the bytes
   of a damaged package uses a general-purpose zip tool, outside METAIS.
4. **Why everything, and why at open.** Verifying only the entries a command
   reads would let `inspect` print a package digest that names content the
   tool never checked, and a render that never touches `references/` would
   record in its provenance the digest of a package whose references were
   altered. A package that opens is the package its digest names. The cost
   is one decode and hash of the whole package per command, bounded by
   ADR-009's total-size limit; nothing is written to disk.
5. **A source directory has no digests to verify.** Commands accept it (D5
   item 7); its names, layout and links are checked as D2 and D5 require
   when it is read, and its package digest and compiled files are computed
   in memory when a command needs them.
6. **The tool's checks do not depend on the environment.** The confinement
   and structure checks are METAIS's own, applied to the raw bytes, not Go's
   `GODEBUG`-controlled `ErrInsecurePath`.

### D9 — Extension and media type

1. **The extension is `.metais`**, lower case. Reading never depends on it
   (D5 item 7). `pack` writes it; the output file name is ADR-007's.
2. **The media type is `application/vnd.metais+zip`.** The vendor tree fits
   a format a project publishes; `+zip` is accurate, since any zip reader can
   open a package (RFC 6839 §3.6), and RFC 6838 §4.2.8 asks for the suffix
   when the syntax is used. The specification names it. Registering it with
   IANA is not part of v1 (proposed ruling 9).
3. **Detection.** A file is a METAIS package candidate when it starts with
   the local header signature `50 4B 03 04` and its first entry's name, at
   offset 30, starts with `.compiled/`. That is a sufficient magic rule for a
   detector such as `file(1)`; only D8 decides acceptance.

## Interfaces with sibling ADRs

**ADR-002 — the manifest and versioning.** I rely on it for: a canonical
JSON serialization defined to the byte (I assume RFC 8785 or an equivalent
fully specified form), used for `.compiled/manifest.json` and
`.compiled/digests.json`, so that D8 step 6 can refuse a non-canonical
digest file and step 7 can compare bytes; the derivation of
`manifest.json` from `metais.yaml`, deterministic and fixed per declared
version; a YAML profile that accepts JSON content in `metais.yaml` (D5 item
2); the specification-version field, read in D8 step 7; and the rule for
whether a tool reads a later minor version than it implements (D5 item 4
handles both answers). I promise: one manifest file name, `metais.yaml`, at
the top level; `.compiled/manifest.json` as its derived companion. ADR-002's
draft admits `metais.json` as a second name, exactly one of the two present;
if Ed rules that way (proposed ruling 10), D5's table gains `metais.json` on
the same terms and nothing else here changes.

**ADR-003 — the data specification.** I rely on it for a deterministic
derivation of `.compiled/schema.json` from `inputs/` and the manifest, fixed
per declared version (D7 item 3), and for the naming of schema and meaning
files under `inputs/`. I promise: `inputs/` is a reserved directory whose
contents beyond D2 are ADR-003's rule; `.compiled/schema.json` is the file
name.

**ADR-004 — the template environment.** I promise the package digest's
definition and presentation (`sha256:<hex>`, D6 items 2 and 5) for
`package.digest` and the provenance tag, and that a source-directory render
and a packed render see the same digest. I rely on partials and templates
being entries under `templates/`.

**ADR-005 — outputs, transforms and rendering.** I rely on it for the
portable-path predicate (Windows device names including the superscript
forms, reserved characters, trailing dot or space, per-segment length), which
D2 item 5 applies to entry names; I assume it also adopts D2's NFC, character
and simple-case-folding rules for output paths, so that both are one
predicate, and the consistency pass should make them one definition. I
promise the writer of D4: an ordered entry list with per-entry `store` or
`auto`, D1's fixed fields on every entry, no extra fields, no data
descriptors, no zip64, byte-identical per build. ADR-005's OOXML, ODF and
EPUB recipes supply the order and the stored set, and must confirm that each
format accepts D1's fixed fields (EPUB and ODF are verified above; OOXML,
ECMA-376 Part 2 Annex C, is ADR-005's to verify). The manifest declares any
executable mode an output needs, since the container carries none.

**ADR-006 — validation and findings.** I rely on its finding model, its
refusal exit code and its code spelling. I propose these codes for the
refusals D1 to D8 define, for ADR-006 to fix the spelling of:
`container.not_zip`, `container.prepended_data`, `container.trailing_data`,
`container.archive_comment`, `container.zip64`,
`container.encrypted`, `container.method`, `container.flags`,
`container.data_descriptor`, `container.extra_field`, `container.field_value`
(a fixed field with another value, naming the field),
`container.header_mismatch` (local versus central, naming the field),
`container.overlap_or_gap`, `container.entry_order`,
`container.crc_mismatch`, `container.size_mismatch`,
`container.deflate_stream`, `name.encoding`, `name.not_nfc`, `name.shape`,
`name.character`, `name.not_portable`, `name.duplicate`, `name.case_collision`,
`name.file_directory_conflict`, `name.directory_entry`,
`layout.unknown_top_level`, `layout.missing_manifest`,
`layout.compiled_in_source`, `layout.empty_directory`, `layout.non_regular_file`,
`compiled.missing`, `compiled.unknown_file`, `digest.entries_mismatch`,
`digest.mismatch`, `digest.package_mismatch`, `digest.not_canonical`,
`compiled.mismatch`. Each finding's location is the entry name and, for a
header field, the field's APPNOTE name and offset.

**ADR-007 — authoring.** I rely on it for `pack`'s output file name and
`unpack`'s behaviour, within these rules: pack runs D4 with D5's entry set
and writes D7's files; unpack writes every entry except `.compiled/` and
refuses nothing D8 accepted; `check` reports D5 item 5's warnings and D2
item 6's empty-directory finding. Whether `check` also warns about
well-known operating-system litter under a reserved directory
(`.DS_Store`, `Thumbs.db`) is ADR-007's; the container packs every file
under a reserved directory.

**ADR-008 — the command line.** I rely on it for the content and
deterministic derivation of `.compiled/references.json`, fixed per declared
version; for which commands accept a source directory (I assume all of them
that take a package, as the rulings say the read commands do); and for the
form in which `inspect` shows the package digest. I promise: every command
that opens a packed package has verified it completely first (D8), and
`.compiled/` entries are ordinary entries for `ls` and `read`.

**ADR-009 — the security model.** I rely on it for: a package-file-size
limit, a total and a per-entry uncompressed-size limit, an entry-count
limit, a compression-ratio limit and a path-length limit, each with a
default below D3's ceilings (under 4 GiB, under 65,535 entries), applied at
the D8 steps named there; the same for its output limits, which bound the
zip container outputs; the confinement rule for a source directory's
non-regular files; and the hostile corpus, which should hold at least one
package for every refusal listed under ADR-006 above. I promise checks that
do not depend on `GODEBUG` or any other environment variable.

**ADR-010 — the implementation.** I rely on it for: a raw zip structure
reader written in METAIS (Go's `archive/zip` reader does not perform D1's
checks, so at most its constants are reused); the writer built on
`zip.Writer.CreateRaw` or written directly; `compress/flate` with a
byte-counting `io.ByteReader` beneath the decoder, so it does not read ahead
and D1 item 5's "ends exactly at the compressed size" can be checked; an
NFC check (`golang.org/x/text/unicode/norm`, whose tables are 15.0.0 in the
versions checked, or an equivalent); simple case folding and the
general-category check pinned to Unicode 15.0.0, with a test that fails
when the toolchain's `unicode.Version` moves, so a Go upgrade cannot widen
the accepted names silently; and the toolchain pin that fixes the deflate
encoder for a build (D4 item 6).

**ADR-011 — the specification and conformance.** I rely on its vector format
to express "refused with these findings" and "packs to these entries, in
this order, with these methods and this package digest". I promise the
observable rules listed under Conformance.

## Conformance

Every rule below is observable and needs vectors (ADR-011's format):

- **Acceptance**: a minimal valid package (manifest, one schema, one
  template, `.compiled/`) opens; its package digest equals the vector's.
- **Each D1 field**: one refused package per fixed field with another value
  — version needed, each forbidden flag bit (0, 3, 6, 13, a reserved bit),
  method 9, 12, 14, 93 and 99, a nonzero time or date, an extra field
  (0x5455, 0x7075, 0x0001), version made by 0x0014, nonzero internal or
  external attributes other than the fixed one, a symlink mode, a file
  comment, an archive comment, disk numbers.
- **Structure**: prepended bytes; trailing bytes after the end record; a
  gap between entries; overlapping entries (Fifield's construction);
  central directory out of entry order; a local header whose name, CRC,
  size, method or flags differs from its central header; a data descriptor;
  a zip64 end record; an entry count above the limit.
- **Data**: a CRC of zero over nonzero content; a CRC mismatch; a stored
  entry whose sizes differ; a deflate stream with trailing bytes; a
  truncated deflate stream; a deflated entry that decodes past its declared
  size.
- **Names**: each D2 refusal — non-UTF-8, non-NFC (`e` + U+0301), leading
  `/`, `..`, `\`, `:`, an empty segment, U+202E, U+200B, U+E000, an
  unassigned code point, a directory entry, a byte duplicate, a case
  collision of files and of directory prefixes, U+212A KELVIN SIGN (refused
  as not NFC), a file-directory conflict, and one ADR-005 portability
  refusal each.
- **Layout**: an unknown top-level entry; missing `metais.yaml`; each
  missing `.compiled/` file; an extra `.compiled/` file; a source directory
  with `.compiled/`, with an empty directory, with a symbolic link; a
  source directory's unknown top-level file (warning, not packed) and
  dot-name (ignored).
- **Digests**: a tampered entry; a tampered `package` value; a missing and
  an extra `entries` key; a non-canonical `digests.json`; a
  `.compiled/manifest.json`, `schema.json` and `references.json` that does
  not match its derivation — each refused, with every independent finding
  reported in one run.
- **Writer**: packing a source directory yields the vector's entries, in
  the vector's order, each with the vector's method under the `auto` rule
  for the reference encoder (method is compared only where the vector marks
  it as encoder-independent: empty entries stored, and incompressible
  content stored); pack → unpack → pack yields the same package digest;
  the package digest of the source directory equals that of its package;
  one build packs byte-identical files on every supported target (SC5).
- **Host independence**: packing the same source with different file
  modification times, permission bits and directory-listing order gives
  byte-identical packages.

## Alternatives rejected

### A lenient reading profile

Accepting any well-formed zip and ignoring timestamps, attributes and
unknown extra fields would let packages from general-purpose zip tools
open. None can be a valid package anyway, because `.compiled/` must match
its sources and its digests, so lenience buys nothing and costs a vector,
and an ambiguity, for every field left free. Go's reader shows how much a
lenient reader silently accepts.

### Accepting data descriptors

Common writers emit them (Go's `CreateHeader` always does). They are a third
copy of the CRC and sizes, they come with and without a signature, and a
packer that knows each entry's content before writing it never needs one
(D4 item 5). Refusing them removes a disagreement a reader would otherwise
have to check.

### Supporting zip64

Nothing that passes ADR-009's limits needs it, and zip64 adds sentinel
values, an extra field and two end records, each a disagreement to check. A
later major version can admit it.

### Directory entries

They are optional in zip, so some writers emit them and some do not, and a
directory entry beside a file of the same name is one more ambiguity.
Implied directories carry everything a package needs; empty directories are
refused instead (D2 item 6).

### A package digest over the file's bytes

Hashing the zip file is simpler, but it changes when the compressor changes,
which Go's does between releases, and it would differ between two
conforming packers and between a source directory and its package. A
content digest names what the package says, not how it was compressed.

### The digest list as the stored file

Storing `sha256sum` output as `.compiled/digests.txt` is the most portable
form, but every other compiled file is JSON and a consumer should need only
a JSON parser. The JSON file carries the same values, and the package digest
is defined over the `sha256sum` form, so both audiences are served.

### Excluding the compiled files from the digest

Digesting only the sources would let a hostile package alter
`.compiled/manifest.json` without changing its digest. Every entry but the
digest file itself is covered.

### Trusting `.compiled/` once its digests match

The digests prove the files were not altered after packing, not that the
packer derived them correctly. A package whose `manifest.json` disagrees
with its `metais.yaml` would show one thing to a person reading the source
and do another, and a consumer using `schema.json` would validate against a
schema the tool does not use. Re-deriving at open costs little and turns
each compiled file into a proof.

### Verifying each entry only when it is read

Cheaper for `ls` and `read`, but it lets the tool report or record a package
digest it has not established (D8 item 4).

### A `mimetype` first entry, as EPUB and ODF use

It would give detectors a fixed string at offset 38, but it adds a reserved
name with no other purpose. Sorting already places `.compiled/` first, which
gives a detector its key (D9 item 3).

### Stored-only packages

Storing every entry would make packages byte-identical across every
conforming packer and toolchain, and would make compression ratio limits
moot. It would also make text-heavy packages — reference corpora,
regulatory glossaries — several times larger. The content digest already
gives every implementation one identity, so compression stays.

### An open top level

Allowing `README.md`, `LICENSE` or anything else at the top level would make
the package's meaning depend on files no rule describes. Licence text a
package must carry belongs under `references/`, where the manifest can name
it (ADR-002).

## Consequences

### Positive

- One rule set reads and writes packages and containers; each field has one
  value and one vector.
- A package that opens is exactly the package its digest names, whatever
  command opened it.
- The package digest is the same from every conforming packer and from the
  source directory, so provenance and caching key on content, and anyone can
  check it with `unzip` and `sha256sum`.
- The modern zip-bomb, header-confusion and name-spoofing attacks are
  refused structurally, not by heuristics.
- Entry names extract on every platform, so `extract` and `unpack` never meet
  an unwritable name.

### Negative

- Every command reads and hashes the whole package, and re-derives three
  files, before doing anything; a large package costs that on every `ls`.
  ADR-009's total-size limit bounds it.
- A second implementation must write zips with a raw writer and fixed
  fields; a stock high-level zip writer (Go's `CreateHeader`, Java's
  `ZipOutputStream`) does not conform.
- A second implementation must reproduce the three derived files byte for
  byte, which makes ADR-002's canonical JSON, ADR-003's schema combination
  and ADR-008's references index exact specifications, not conventions.
- Package bytes change between tool builds when the encoder changes; only
  the digest is stable across builds.
- Names with format characters (zero-width joiner sequences, for instance),
  characters assigned after Unicode 15.0.0, and empty directories are not
  expressible.

### Neutral

- Packages carry no timestamps or permissions, so an extracted package's
  files have the extractor's defaults.

## What this does not decide

- The manifest's fields, canonical JSON and versioning rules (ADR-002).
- The contents of `inputs/`, `templates/`, `examples/` and the derived
  compiled files (ADR-002, ADR-003, ADR-007, ADR-008).
- Each resource limit's value (ADR-009) and the portable-path predicate's
  full membership (ADR-005).
- The OOXML, ODF and EPUB recipes, the stdout tar profile and file modes of
  outputs (ADR-005).
- `pack`'s output file name and whether reading a package from standard input
  is supported (ADR-007, ADR-008).
- Signing (out of scope for v1). A future signature would sign the package
  digest, which is why it is defined over content.

## Proposed rulings

1. **The reading profile equals the writing profile** (D1 item 1). Lean: yes.
   Why: only a packer can make a valid package, so strictness costs no one,
   and every fixed field removes an ambiguity. Alternative: a lenient reader
   that ignores timestamps, attributes and unknown extra fields.
2. **No zip64 in major version 1**, with ceilings of 65,534 entries and
   0xFFFFFFFE bytes, and ADR-009's defaults below them (D3). Lean: yes. Why:
   no package within the limits needs it. Alternative: admit zip64 when a
   value needs it.
3. **No directory entries; an empty directory in a source directory is a
   finding** (D2 item 6). Lean: yes. Why: the packed form cannot carry it,
   and source and package must mean the same. Alternative: directory entries
   for empty directories only.
4. **Entry names are NFC, built from assigned Unicode 15.0.0 characters in
   categories L, M, N, P, S plus U+0020, unique under simple case folding
   across files and directory prefixes** (D2). Lean: yes. Why: it refuses
   spoofed listings and cross-platform collisions and gives every
   implementation the same answer. Alternative: ASCII-only names, which is
   simpler and refuses non-English file names.
5. **The package digest is the SHA-256 of the `sha256sum`-format list of
   every entry except `digests.json`, sorted by UTF-8 bytes, and a source
   directory has the same digest as its package** (D6). Lean: yes. Why:
   compression-independent, checkable with standard tools, the same from
   every packer. Alternative: SHA-256 of the zip file.
6. **Whole-package verification at every open, with no skip flag** (D8).
   Lean: yes. Why: a reported or recorded digest is always one the tool
   established. Alternative: lazy per-entry verification.
7. **The derived compiled files are re-derived at open and refused on
   mismatch** (D7, D8 step 7). Lean: yes. Why: the compiled files become a
   proof, not a claim. Alternative: digests only, with `check` re-deriving.
8. **A closed top level; in a source directory, unknown dot-names are ignored
   and other unknown names are a warning and are not packed** (D5). Lean:
   yes. Why: a package's meaning depends only on named directories, while a
   package can live in a repository. Alternative: refuse unknown names in
   source directories too, or allow `README.md` and `LICENSE`.
9. **Media type `application/vnd.metais+zip`, with IANA registration deferred
   until the specification is public**, done then through the vendor tree
   (D9). Lean: yes. Why: registration needs a public specification to cite,
   and the repository is private. Alternative: register now; or no media
   type in v1.
10. **One manifest file name, `metais.yaml`, which may hold JSON** (D5 item 2).
    Lean: yes. Why: never two manifests. Alternative: accept `metais.json`
    too and refuse both present. This touches ADR-002's YAML profile.
11. **Byte-identical packing holds per tool build; across builds and
    implementations, identity is the package digest** (D4 item 6). Lean:
    yes. Why: Go's deflate output is outside its compatibility promise and
    changed in 2026. Alternative: ADR-010 vendors a pinned encoder so bytes
    survive toolchain upgrades, which narrows but does not remove the
    caveat; or stored-only packages.
12. **Fixed field values: version made by 0x0314 with external attributes
    0x81A40000; date 1980-01-01 00:00 (0x0021, 0x0000); version needed 10 or
    20; method `auto` at level 9** (D1, D4). Lean: yes. Why: the earliest
    representable date, ordinary permissions on extraction, the APPNOTE's
    exact minimum versions. Alternative: MS-DOS host with zero attributes,
    and 2.0 everywhere as Go writes.

## References

- PKWARE, *APPNOTE.TXT — .ZIP File Format Specification*, version 6.3.10
  (sections 4.3.1 to 4.3.16, 4.4.1 to 4.4.26, Appendix D):
  https://pkware.cachefly.net/webdocs/casestudies/APPNOTE.TXT
- Go `archive/zip` source, go1.26.8 (`reader.go`: `init`, `readDirectoryEnd`,
  `findBodyOffset`, `checksumReader.Read`, `initFileList`; `writer.go`:
  `CreateHeader`, `CreateRaw`; `struct.go`):
  https://github.com/golang/go/tree/go1.26.8/src/archive/zip and
  https://pkg.go.dev/archive/zip
- Go `GODEBUG` history, `zipinsecurepath` default:
  https://go.dev/doc/godebug
- Go commit 388145c5, "compress/flate: improve compression speed":
  https://github.com/golang/go/commit/388145c51b
- Go commit 2747d887, "compress/flate: clarify compatibility promise":
  https://github.com/golang/go/commit/2747d887eb
- Go `compress/flate` levels: https://pkg.go.dev/compress/flate
- Go `strings.EqualFold` (simple Unicode case folding):
  https://pkg.go.dev/strings#EqualFold
- Go `unicode.Version` (15.0.0 in go1.26.8):
  https://pkg.go.dev/unicode#pkg-constants
- `golang.org/x/text/unicode/norm` (tables 15.0.0):
  https://pkg.go.dev/golang.org/x/text/unicode/norm
- Unicode Character Encoding Stability Policies (normalization and case
  folding stability): https://www.unicode.org/policies/stability_policy.html
- Unicode `CaseFolding.txt`, 15.0.0:
  https://www.unicode.org/Public/15.0.0/ucd/CaseFolding.txt
- RFC 1951, DEFLATE: https://www.rfc-editor.org/rfc/rfc1951
- D. Fifield, *A better zip bomb*, USENIX WOOT 2019:
  https://www.usenix.org/conference/woot19/presentation/fifield
- CVE-2019-13232 (Info-ZIP UnZip overlapping entries):
  https://nvd.nist.gov/vuln/detail/CVE-2019-13232
- W3C, *EPUB 3.3*, §4.3.2 ZIP file requirements and §4.3.3 the `mimetype`
  file: https://www.w3.org/TR/epub-33/#sec-zip-container-zipreqs
- OASIS, *OpenDocument v1.3 Part 2: Packages*, §2.2.1 and §3.3:
  https://docs.oasis-open.org/office/OpenDocument/v1.3/os/part2-packages/OpenDocument-v1.3-os-part2-packages.html
- RFC 6838, *Media Type Specifications and Registration Procedures*, §3.2,
  §3.4, §4.2.8: https://www.rfc-editor.org/rfc/rfc6838
- RFC 6839, *Additional Media Type Structured Syntax Suffixes*, §3.6:
  https://www.rfc-editor.org/rfc/rfc6839
- IANA media types registry, application tree (read 2026-10-18):
  https://www.iana.org/assignments/media-types/media-types.xhtml
- RFC 8785, *JSON Canonicalization Scheme* (member ordering by UTF-16 code
  units): https://www.rfc-editor.org/rfc/rfc8785
- GNU coreutils, *cksum output modes* (the untagged format and its escaping):
  https://www.gnu.org/software/coreutils/manual/html_node/cksum-output-modes.html

## Implementation log

Nothing implemented.

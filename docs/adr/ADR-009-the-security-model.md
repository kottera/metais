# ADR-009: The security model — the package and its inputs are hostile, the host is invisible to them, every resource has a number, and anything out of spec is refused

**Status:** Accepted 2026-10-18 by Ed Funnekotter; its proposed rulings are ratified in [RULINGS-v1](RULINGS-v1.md).
**Date:** 2026-10-18
**Deciders:** Ed Funnekotter
**Designs:** PRD-001 R14 (items 1 to 4); the security halves of R10 item 1, R11
item 2 and R12 items 1 and 2; success criteria SC6 and SC7
**Depends on:** ADR-001 (the zip profile and the digest file this ADR's
container checks run against); ADR-002 (the YAML profile and the manifest);
ADR-003 (input forms, the data directory, `--set` and `--inputs-json`, the combined
schema); ADR-004 (the template environment, its variables, its filters and
tags); ADR-005 (output paths, staging, overwrite, stdout); ADR-006 (finding
codes, severities, exit codes); ADR-008 (flag spelling); ADR-010 (the
libraries and the test layout); ADR-011 (the vector format and the place of
the hostile corpus)

## Context

### What the PRD requires

PRD-001 R14 makes a package untrusted input: every entry is confined before it
is read; every resource is bounded, with defaults the caller may lower; a
package outside the specification is refused, not approximated; and
**nothing of the host reaches a template**. R14 item 4 spells the last rule
out: no variable, filter, tag or manifest field exposes the environment, the
tool never expands an environment variable into an input, a path or a default,
and the same holds for files outside the package and the named inputs, the
working directory, the user, the host name, the process and the clock. A
caller who wants a value from the environment puts it in an input explicitly.

SC6 asks for a corpus of hostile packages refused in full with nothing written
outside the output directory. SC7 asks that rendering every conformance
vector and every hostile package, with sentinel values planted in the
environment, the working directory and the home directory, never puts a
sentinel into any output, finding or message, and that the test be pinned red
first against a render that does read the environment. `AGENTS.md` adds that
bounded work is measured in CPU time, not wall time alone.

The rule that no host fact reaches a template is the property that makes an
unfamiliar package safe to render (PRD-001 §1, property 4). This ADR treats it
as the central guarantee: D12 to D15 design it, and D16 proves it.

### Facts that constrain the design (verified 2026-10-18)

**The zip format and Go's reader.**

- Go's `archive/zip` checks entry names for traversal **only when the
  environment variable `GODEBUG` contains `zipinsecurepath=0`**; by default it
  accepts `../x`, absolute names and backslashes (`reader.go`, Go 1.20
  release notes). A safety check that a host environment variable can turn
  off is not a check this design may rely on.
- `File.Open` locates an entry's data by reading only the signature and the
  two length fields of its local header (`findBodyOffset`). It never compares
  the local header's name, method, sizes or CRC with the central directory's,
  and it does not detect two central-directory records that point at
  overlapping bytes. It tolerates bytes before the archive (`baseOffset`).
- It does enforce the declared uncompressed size while reading: more bytes than
  declared is `ErrFormat`, fewer is `io.ErrUnexpectedEOF`. It skips the CRC
  check when the recorded CRC is zero and there is no data descriptor.
- An entry's type comes from its external attributes; a Unix-created entry
  with mode `0xA000` is a symbolic link (`struct.go`, `s_IFLNK`).
- David Fifield's "A better zip bomb" (WOOT 2019) overlaps many central
  directory records onto one compressed kernel, so that output grows
  quadratically with input: 42 kB to 5.5 GB, 10 MB to 281 TB, and with Zip64
  46 MB to 4.5 PB. DEFLATE alone cannot exceed a ratio of 1032. The paper
  notes that summing declared sizes does not predict output, because declared
  sizes may not match the data; that detection belongs inside the parser, not
  in a prefilter; and that two parsers may find two different central
  directories in one file and so disagree on what it contains. Info-ZIP UnZip
  added overlap detection (CVE-2019-13232).
- "Zip Slip" (Snyk, 2018) is the class of extractors that write an entry named
  `../../x` outside the target directory.

**YAML, JSON and Markdown.**

- `gopkg.in/yaml.v3` (`github.com/go-yaml/yaml`) has been **archived since
  2025-04-01**; the YAML organisation maintains a fork at
  `github.com/yaml/go-yaml`. v3 bounds alias expansion with a *ratio
  heuristic* ("document contains excessive aliasing" once more than a sliding
  99 %–10 % of decoded nodes come from aliases) and bounds nesting at 10,000
  levels (`max_flow_level`, `max_indents`). Its `Node` tree keeps an alias as
  an `AliasNode` pointing at its anchor, unexpanded. The "billion laughs"
  attack is nested alias (or entity) expansion.
- `encoding/json` bounds nesting at 10,000 levels (`maxNestingDepth`) and
  accepts duplicate object keys, the last one winning.
- cmark's `test/pathological_tests.py` lists the inputs that make naive
  CommonMark parsers quadratic or worse: nested emphasis, unclosed link
  openers, deeply nested lists and block quotes, and others.

**Regular expressions.**

- Go's `regexp` implements RE2 syntax and "is guaranteed to run in time linear
  in the size of the input". `regexp/syntax` refuses repeat counts above 1000,
  parse trees deeper than 1000 (`ErrNestingDepth`) and programs above a size
  cap (`ErrLarge`). Its `\s` is `[\t\n\f\r ]`, `\d` and `\w` are ASCII, `.`
  excludes only `\n`, `$` without the `m` flag matches only at the end of text,
  and `(?<name>re)` is accepted alongside `(?P<name>re)`. There are no
  lookaround assertions and no backreferences, and `\uXXXX` is not an escape
  (`\x{XXXX}` is).
- JSON Schema 2020-12 Core §6.4: regular expressions "SHOULD be valid according
  to the regular expression dialect described in ECMA-262, section 21.2.1",
  SHOULD be built with the `u` flag, and authors SHOULD limit themselves to a
  listed subset — individual characters, simple, range and complemented
  character classes, the quantifiers `+ * ?` and `{x} {x,y} {x,}` with their
  lazy forms, the `^` and `$` anchors, grouping and alternation. Patterns MUST
  NOT be taken as anchored.
- In ECMA-262, `\s` is every code point of the `WhiteSpace` and
  `LineTerminator` productions (including U+00A0, U+FEFF, the `Zs` category,
  U+2028 and U+2029), and `.` without the `s` flag excludes every
  `LineTerminator` (LF, CR, U+2028, U+2029). The grammar includes lookahead,
  lookbehind, and numbered and named backreferences, none of which is regular.
- Backtracking engines take exponential time on patterns such as `^(a+)+$`
  (Russ Cox, "Regular Expression Matching Can Be Simple And Fast").

**The libraries the implementation is likely to use** (ADR-010 chooses):

- `github.com/santhosh-tekuri/jsonschema/v6` compiles `$ref`s through a
  loader whose **default is `FileLoader`, which calls `os.Open` on any
  `file://` URL**. `Compiler.UseLoader` replaces the loader, and
  `Compiler.UseRegexpEngine` replaces the regular-expression engine (default:
  Go's `regexp`). The 2020-12 metaschemas are embedded. `uniqueItems` hashes
  above 20 items rather than comparing every pair.
- `github.com/osteele/liquid` (v1.9.2, 2026-08-16) defaults its template store
  to `FileTemplateStore{}`, which with an empty `Root` **reads partials from the
  current working directory** (`render/config.go`); `Engine.RegisterTemplateStore`
  replaces it. Its date parsing turns the string `"now"` into **`time.Now()`**,
  parses zone-less strings **in `time.Local`**, and turns an integer string into
  `time.Unix(sec, 0)`, which is a local time (`values/parsedate.go`). It has no
  resource limits: no render-length, loop or recursion bound.
- Shopify Liquid, the reference implementation, has `ResourceLimits`
  (`render_length_limit`, `render_score_limit`, `assign_score_limit` and
  cumulative forms), a nesting limit `Block::MAX_DEPTH = 100` shared by parsing
  and by `render` sub-contexts, and treats `'now'` and `'today'` in `date` as
  `Time.now`. Its `render` tag requires a quoted string as the template name.

**The host, as Go reaches it.**

- `time.Local` is initialised from `$TZ`; `TZ=""` means UTC, an unset `TZ`
  reads `/etc/localtime`, and a `TZ` naming an absolute path reads that file
  (`zoneinfo_unix.go`). `time.LoadLocation` searches the directory or zip named
  by `$ZONEINFO`, then the system zone database, then `$GOROOT`, and only last
  the embedded `time/tzdata`. `time.Parse` uses `time.Local`'s zone when a
  parsed offset or abbreviation matches it; `time.Unix` returns a local time.
- `os.UserHomeDir` reads `$HOME`; `os.TempDir` reads `$TMPDIR`;
  `filepath.Abs` calls `os.Getwd`.
- `os.Root` (Go 1.24) confines operations to one directory: it follows
  symbolic links only when they stay inside the root and refuses absolute
  link targets, but "does not prohibit … access to Unix device files".
- `os.Clearenv` empties the process environment as Go sees it: on Unix
  `syscall.Clearenv` resets Go's own copy, so every later `os.Getenv` returns
  `""`.
- Peak resident memory from `getrusage` is reported in KiB on Linux and in
  bytes on darwin (`man 2 getrusage` on each).
- Git refuses to check out `.git` path components under case variants, HFS+
  ignorable code points and NTFS short names (CVE-2014-9390), because a written
  `.git/hooks/*` or `.git/config` runs code at the next git command.

## Decision

### D1 — The threat model

1. **Assets.**
   - *The rendering host*: its CPU, memory and disk, and the availability of
     the program calling `metais`.
   - *The host's secrets*: environment variables (tokens, keys), files the
     calling user can read (`~/.ssh`, credentials, source trees), and facts
     about the machine (user, host name, paths, time zone, process).
   - *The host's filesystem integrity*: no file is written, replaced or
     deleted outside the directory the caller named for output.
   - *The caller's outputs*: what a render writes is exactly what the package
     and the inputs determine, or nothing.
2. **Adversaries.**
   - **A hostile package author.** Controls every byte of the package: the zip
     structure, entry names and attributes, the manifest, schemas, templates,
     partials, assets, references and examples. Also controls a source
     directory an author hands someone else, such as a repository that a
     second author or an automated job runs `check`, `test` or `pack` on.
   - **A hostile input supplier.** Controls the contents of the data
     directory, of files and values passed on the command line, and of
     standard input when it is named: file contents, file names, file types
     (links, FIFOs, devices) and sizes. The supplier is often less privileged
     than the caller — an AI working in a sandbox, a pull request, a
     downloaded archive — so a confused-deputy attack is in scope: the supplier
     plants a link to a file only the caller can read and hopes the render puts
     it into an output the supplier sees.
3. **Trust boundaries.**

   ```
                  caller (trusted): argv, --limit, --now, --overwrite,
                  the paths it names, NO_COLOR
                               │
     package bytes ──────────▶ ┌─────────────────────────────────┐ ◀──── data directory,
     (hostile author)          │ metais process                  │       named files, stdin
                               │  Isolate() → confined readers → │       (hostile supplier)
                               │  parse → validate → render      │
     host: env, cwd, HOME,  ✕  │  closed binding set             │
     clock, TZ, user, host,    └─────────────────────────────────┘
     pid, other files                          │
     (read never past Isolate())               ▼
                                 output directory or stdout
                                 (written, confined, all or nothing)
   ```

   Data crosses into the template environment only through the binding set
   (D13): the validated inputs, the manifest's own metadata, and `--now`.
4. **Out of scope.**
   - **A malicious caller.** It chooses the flags, the paths and the
     environment and can run any program it likes; nothing here protects the
     host from it. Its explicit choices (`--overwrite`, a named path, `--now`)
     are honoured.
   - **The operating system, the filesystem, the hardware, and other processes
     racing the tool on the directories it uses** (a concurrent local attacker
     swapping a file for a link between a check and a use). The checks use
     handle-relative operations where Go offers them (D7, D8), which narrows
     the window; they do not close it against a concurrent attacker.
   - **The safety of rendered content.** Rendering an unfamiliar package is
     safe; running, opening or publishing what it rendered is the caller's
     decision. A scaffold can contain a hostile CI workflow, and an HTML output
     without `escape: html` can carry an input supplier's markup (ADR-004 makes
     escaping opt-in per output and the specification says so plainly). The one
     exception is a path that runs code merely by existing, which D8 item 6
     refuses.
   - **Authenticity.** v1 has no signing. The digests in `.compiled/` (ADR-001)
     detect corruption and inconsistent edits; an author who rewrites an entry
     can rewrite its digest. A digest is not a statement of trust.
   - **Supply-chain compromise of the tool's own dependencies**, beyond what the
     dependency scan in D16 item 4 catches.
   - **Side channels**: timing and output sizes reveal input sizes.
   - **Resource use up to the limits.** A render within every limit may use the
     CPU and memory those limits allow (D17 item 4 measures the ceiling).

### D2 — Refusal semantics: out of spec is refused, never approximated

1. **A package, an input or a render that breaks a rule of the specification or
   exceeds a limit is refused** with a finding naming the rule (ADR-006 codes,
   D18). The tool never skips the offending entry, truncates the offending
   value, repairs the offending path (no stripping of `..`, no case folding, no
   Unicode normalisation), falls back to a default, or renders "what it can".
2. **Bounds are checked before the work they bound.** Declared sizes are
   checked from metadata before any decompression; a read stops at the bound
   plus one byte; a count is checked as it is incremented, not after the
   collection is built. A refusal costs at most the bound.
3. **A refusal writes nothing** (ADR-005 stages a render and commits it only
   when every output succeeded).
4. **A structural refusal of the container stops the reading of the package.**
   Once the zip structure (D5) or a digest is refused, no entry is parsed,
   because nothing in it can be trusted to mean one thing. Findings below that
   layer — the manifest, the schemas, the inputs — are reported together in one
   pass (ADR-006).
5. **A limit finding carries** the limit's name, its bound, whether the bound
   came from the specification or was lowered by the caller (`--limit`), and
   the observed value, or "more than the bound" when reading stopped at the
   bound.
6. **Two limits bound a result rather than refuse a command**: `findings` and
   `grep-matches` (D3). When either is reached, the result says so explicitly
   (a `truncated` marker and the number omitted) and the exit status is what it
   would have been without the cap. Every other limit refuses.

### D3 — Every resource limit, with its default

Each limit has a name (used by `--limit`, D4, and in findings), a default, the
thing it bounds, and a reason. Sizes are bytes; `KiB`, `MiB` and `GiB` are
powers of 1024.

**Package limits** — the specification states each as the most a package may
use. A package within all of them is portable: every conforming tool accepts
it at its defaults.

| Limit | Default | Bounds | Rationale |
|---|---|---|---|
| `package-size` | 256 MiB | the package file | room for references (standards text) and assets (fonts, images); checked by `stat` before the file is read |
| `package-uncompressed` | 512 MiB | the sum of every entry's uncompressed size, declared and actual | the cap that defeats overlap bombs regardless of ratio |
| `entries` | 10,000 | central-directory records (a package has no directory entries, ADR-001 D2 item 6) | a static site's assets fit; bounds the directory parse and the per-entry checks |
| `entry-size` | 64 MiB | one entry, uncompressed | the largest plausible asset or reference |
| `entry-ratio` | 200 | uncompressed ÷ compressed size, for an entry larger than 1 MiB uncompressed | text compresses 3–20:1 and repetitive XML rarely beyond 100:1; DEFLATE peaks at 1032:1, so 200 refuses padding and bombs while small entries, whose absolute size is harmless, are exempt |
| `path-length` | 512 bytes | an entry path, an output path, a container entry path | deep scaffolds fit; Windows long-path support is not assumed beyond this |
| `path-depth` | 32 components | the same paths | |
| `manifest-size` | 1 MiB | `metais.yaml`, and `.compiled/manifest.json` | a manifest is prose plus declarations |
| `schema-size` | 4 MiB | one schema entry | a regulator's field glossary fits |
| `combined-schema-size` | 16 MiB | `.compiled/`'s combined input schema, and the tool's own derivation of it | inlining `$ref`s duplicates subschemas, so an author can build a small schema whose inlined form is exponential; `pack` refuses it and so does every reader |
| `schema-depth` | 64 | nesting of subschemas | recursion in validators and in the cross-reference check |
| `ref-depth` | 32 | consecutive `$ref` / `$dynamicRef` hops at one instance location | refs that consume no instance can otherwise loop or fan out |
| `pattern-length` | 4,096 bytes | a `pattern`, a `patternProperties` key, a `grep` pattern | |
| `regex-program` | 4,096 instructions | a compiled regular expression | the matching cost is linear in the input times the program size; this bounds the second factor |
| `template-size` | 8 MiB | one template entry | the main XML part of a long office document fits |
| `template-nesting` | 100 | block tags open at once | Shopify Liquid's `MAX_DEPTH` |
| `partial-depth` | 32 | `render` calls active at once | allows recursive rendering of a tree (a menu, an outline) and stops self-recursion long before the stack |
| `value-depth` | 64 | array/object nesting in any parsed document: manifest, schema files, inputs, frontmatter | both libraries' 10,000 levels are deep enough to exhaust recursive consumers |
| `number-length` | 1,024 bytes | one numeric literal | a million-digit literal costs arbitrary-precision arithmetic in `multipleOf` and comparisons |

**Data limits** — bound the inputs of one command.

| Limit | Default | Bounds | Rationale |
|---|---|---|---|
| `input-size` | 32 MiB | one input file, one collection item, one `--inputs-json`, `--set` or `--set-json` value, standard input | parsed values occupy several times their source size |
| `inputs-total` | 128 MiB | all input bytes one command reads | keeps peak memory within D17's ceiling |
| `csv-rows` | 1,000,000 | data rows in one CSV input | |
| `csv-columns` | 1,024 | fields per record | |
| `csv-cell` | 1 MiB | one field | |
| `collection-items` | 100,000 | files in one collection input | a large site |
| `frontmatter-size` | 1 MiB | the frontmatter block of one Markdown file | |

**Work and output limits** — bound what a command computes and writes.

| Limit | Default | Bounds | Rationale |
|---|---|---|---|
| `render-work` | 2³² units | template evaluation in one command (D11) | the deterministic CPU bound for templates |
| `loop-iterations` | 10,000,000 | iterations of every `for`, `tablerow`, `render … for` and `for_each`, summed over one command | a range literal `(1..1000000000)` costs nothing to write |
| `value-size` | 64 MiB | any string a template produces: a filter result, a `capture`, a concatenation | string doubling by `append` reaches a gibibyte in thirty steps |
| `array-size` | 1,000,000 elements | an array a filter or a range produces | `split` on a large string |
| `assign-bytes` | 512 MiB | bytes stored by `assign` and `capture`, summed over one command | Shopify's assign score |
| `schema-work` | 100,000,000 | subschema evaluations in one command | nested `anyOf`/`oneOf` over non-consuming refs is exponential in depth |
| `regex-work` | 2³³ units | regular-expression matching in one command (D10) | the deterministic CPU bound for patterns and `grep` |
| `outputs` | 100,000 | files one render plans, assets, containers and container entries included (ADR-005 D13) | one per item of a full collection |
| `container-depth` | 3 | zip containers nested inside one another (ADR-005 D8 item 4) | a submission zip holding a `.docx` is a real case; deeper is not |
| `output-size` | 256 MiB | one written file, including a container and a copied asset | |
| `output-total` | 1 GiB | all bytes one render writes, or writes to standard output | |
| `findings` | 100,000 | findings one command reports | a result bound (D2 item 6) |
| `grep-matches` | 10,000 | matches `grep` reports | a result bound (D2 item 6) |
| `grep-context` | 100 lines | lines of context per match | a larger `-C` is refused |

1. **There is no CPU-time or wall-clock limit.** Every channel through which a
   package or an input can make the tool work without bound has a counter
   above — `render-work`, `loop-iterations`, `schema-work`, `regex-work` — and
   every algorithm the tool runs is linear in a bounded quantity. A counter
   refuses identically on every host; a timer refuses a render on a slow
   machine that a fast one accepts, which would make refusal itself
   host-dependent (R10 item 2). D17 item 4 measures the CPU time that the
   counters permit.
2. **The defaults are exact.** A conforming tool enforces exactly these
   defaults, so a refusal at a default is a property of the package and the
   inputs, not of the tool.
3. **The defaults are calibrated before this ADR is accepted.** ADR-010's
   benchmark suite runs each limit's worst case (D17 item 4); a default whose
   worst case breaks the ceiling there is lowered, and the table is amended
   with the measured figure.

### D4 — Lowering a limit, and why none is raised

1. **`--limit NAME=VALUE`**, repeatable, is accepted by every command that reads
   a package or an input. `NAME` is a name from D3. `VALUE` is a non-negative
   integer, with an optional unit suffix `KiB`, `MiB` or `GiB` for size limits.
   A lowered limit applies to that command only.
2. **An unknown name, a malformed value, or a value above the default is
   refused** before any work, with a finding naming the default. No flag,
   file or environment variable raises a limit.
3. **`metais limits [--json]` prints every limit** with its default, its unit
   and its class, so a caller or a harness can read the table rather than
   copy it (ADR-008 D1 lists the command; its JSON format is
   `metais.limits/1`).
4. **Lowering is how the conformance vectors test boundaries cheaply.** A
   vector at `--limit entries=3` with three entries passes and with four is
   refused; no vector needs a 10,000-entry package to test the rule.

### D5 — Reading the container: the zip-bomb and parser-differential defence

The tool parses the zip structure itself, in this order, before it
decompresses anything. Go's `archive/zip` may be used to decompress an entry
whose bounds this procedure has already established (ADR-010), never as the
check.

1. **File size**: `package-size`.
2. **One unambiguous end record.** The End Of Central Directory record sits
   exactly at the end of the file: no archive comment, no trailing bytes, and
   no Zip64 record or locator (ADR-001 D3 refuses Zip64). A second EOCD
   signature in the final 65,557 bytes (the largest span an end record and
   its comment can occupy) is refused, so that no other parser can find a
   different central directory; ADR-001 D8 step 2 applies it.
3. **The central directory**: the record count is checked against `entries`
   before anything is allocated; the directory's declared size and offset lie
   inside the file and end exactly where the end record begins.
4. **Each central record**: the entry name obeys D6; the type is a regular
   file (D6 item 4); the method is one ADR-001 allows; the
   encryption bit is clear; the declared uncompressed size is within
   `entry-size`, the running total within `package-uncompressed`, and the ratio
   within `entry-ratio`.
5. **Each local header** is read in full and must agree with its central record
   on name bytes, method, general-purpose flags, CRC and both sizes. ADR-001's
   profile has no data descriptors, so the local header carries the sizes.
6. **Contiguous layout.** Sorted by offset, the first local header is at offset
   0 (no prepended bytes), each entry's span — local header, name, extra field,
   compressed data — ends exactly where the next begins, and the last ends
   exactly at the central directory. A gap is refused (hidden data that another
   parser scanning local headers would see), and an overlap is refused
   (Fifield's construction).
7. **Decompression is counted.** Every entry is read through a counter that
   stops at the declared size plus one byte: more is refused, fewer is refused,
   and the CRC must match. Bytes read count towards `package-uncompressed`
   independently of the declared figures.
8. **No entry's content is used until the whole package has been verified**
   against `.compiled/`'s digest file (ADR-001 D8): every entry is read in
   full, within `entry-size`, and digested at open, before any command parses,
   searches or prints anything, so a command that reads one entry (`read`,
   `grep`, `outline`) still reports only a package whose digest it has
   established.
9. **The compiled forms agree with their sources.** In a packed package the
   tool re-derives the canonical manifest from `metais.yaml` and the combined
   schema from `inputs/`, within their limits, and refuses the package when
   either differs from what `.compiled/` holds. Otherwise `inspect` could show
   one manifest and `render` obey another.
10. **A zip inside the package is bytes.** An asset that is itself a zip (or a
    nested `.metais`) is copied or embedded as an opaque entry and never opened.

### D6 — Entry names and entry types

These rules apply to every entry of a packed package and every file of a source
directory. ADR-005's portable-path rules (Windows-invalid names, case-only
collisions) apply to entry paths too, so that `unpack` and `extract` can write
any accepted package on any platform.

1. **An entry name** obeys the one portable-path predicate (ADR-001 D2,
   ADR-005 D12): NFC UTF-8, `/` as its only separator, built only from
   assigned characters of categories L, M, N, P, S and U+0020 (so no NUL,
   no other control, no DEL), no backslash or other Windows-reserved
   character. It is not absolute (no leading `/`), has no drive or UNC prefix
   (`C:`, `//server`), has no empty component (`a//b`), and has no `.` or
   `..` component. Each component is at most 255 bytes; the whole name is
   within `path-length` and `path-depth`. No name ends in `/`: a package has
   no directory entries (ADR-001 D2 item 6).
2. **No two entries collide**: no duplicate names, no two names or directory
   prefixes that are canonical caseless matches (ADR-005 D12 item 6), and no
   entry that is both a file and a prefix of another entry's path (`a`
   beside `a/b`).
3. **Only the layout's roots** (ADR-001) appear as first components.
4. **An entry is a regular file.** Whatever the creator platform, external
   attributes other than ADR-001 D1's fixed regular-file value — a symbolic
   link (`0xA000`), a directory, a FIFO, a socket, a device or any other
   type — are refused. Permission bits in
   the external attributes are never applied by `unpack`, `extract` or
   `render`: written files get the modes ADR-005 states, and never setuid,
   setgid or sticky bits.
5. **Entry names are matched exactly.** Every lookup (D9) compares bytes against
   the index the central directory built; there is no filesystem, no case
   folding and no normalisation between a name and an entry.

### D7 — Reading the host: data directories, named inputs and source directories

All host filesystem access goes through one internal package (`internal/hostfs`,
D14 item 3), which opens each caller-named directory as an `os.Root` (Go 1.24)
and performs every later operation relative to it.

1. **The tool reads only files a declared input could be.** In a data directory
   it lists the entries, then opens only those whose names ADR-003's naming
   rule maps to a declared input (`<name>.<ext>` for an accepted form, or
   `<name>/` for a collection) and the items of those collections. A data
   directory that happens to be a home directory exposes nothing else.
2. **Inside a data directory, no symbolic link is followed** — not even one
   whose target stays inside. Each candidate is examined with `Lstat` relative
   to the root; a link is refused (`input.link`). This closes the confused-deputy
   attack of D1 item 2 without depending on where a link points, keeps
   collection walks free of cycles, and makes the inputs exactly what is in the
   directory.
3. **Only regular files are opened.** A FIFO, socket or device is refused from
   its `Lstat` mode without being opened, so a FIFO cannot block the tool. After
   opening, the handle's own `Stat` must still be a regular file and the same
   file (`os.SameFile`); the read stops at `input-size` plus one byte.
4. **The caller's own names are honoured.** The package operand, the data
   directory itself and the output directory may be symbolic links; the
   caller is trusted and named them (ADR-003 provides no other file form).
   They must still resolve to a directory or a regular file.
5. **Hard links are not detected.** A hard link to a file outside the directory
   cannot be told from an ordinary file without refusing every file with more
   than one link, which breaks common tooling. Hard links cannot cross machines
   in a downloaded archive's data and require the creator to reach the target,
   so the residual is a supplier that already shares the caller's machine and
   can read the target itself.
6. **Nothing is expanded.** `--set` and `--set-json` values, `--inputs-json` text, file contents and
   names are data: `$VAR`, `${VAR}`, `%VAR%`, `~` and `@path` are literal
   characters unless ADR-003 defines an explicit syntax for a file reference,
   which never consults the environment.
7. **Standard input** is read only for `--inputs-json -` (ADR-008 D7), and
   within `input-size`.
8. **Source directories** (`check`, `test`, `pack`, the sample preview, and
   `ls`/`read`/`grep`/`outline` on a source directory) follow the same rules:
   the tree is walked under an `os.Root`; a symbolic link or special file
   anywhere in the layout's roots is refused; nothing outside the layout's roots
   is read, so a repository's `.git/` never reaches a package. A git clone
   preserves links, so a source tree from a hostile author can carry
   `templates/x -> ~/.ssh/id_ed25519`; this rule is what stops `pack` from
   copying it.

### D8 — Writing the host: output confinement

ADR-005 owns staging, overwrite and the portable-path rules. This ADR fixes how
a write is confined, for `render`, `unpack`, `extract` and `init` alike.

1. **Every write goes through `internal/hostfs`** on an `os.Root`, and every
   path a package or an input produced is written through a root on a
   directory the tool created for this render (ADR-005 D14's staging
   directory) or on the existing directory the caller named. Only the
   caller's own path — `--out` and its missing ancestors — is reached
   through a root on its nearest existing ancestor. No write path is ever
   built by joining strings onto a host path.
2. **An output path is checked before anything is staged**: it obeys D6 items 1
   and 2 and ADR-005's portable rules, whether it came from the manifest or was
   produced by a path template from input data. A supplier's
   `../../.ssh/authorized_keys` is refused, never cleaned.
3. **No symbolic link is written through or replaced.** Each existing component
   of an output path is examined with `Lstat` under the root; a link, even one
   pointing inside, is refused (`output.link`). An existing non-regular file at
   the final path is refused. With `--overwrite`, an existing regular file is
   replaced by renaming the staged file over it, which replaces the directory
   entry and never writes through it.
4. **Existing contents of the output directory are never read.** The tool lists
   names to decide emptiness and collisions; it never opens an existing file
   there, so `--overwrite` cannot leak what it replaces.
5. **Staging lives beside the output directory, in its parent** (ADR-005
   D14; for an existing output directory, inside it, under ADR-005
   Amendment A1), never under `$TMPDIR`, with fixed names that read no random source
   and no process identity; the rename that makes it the output is the only
   operation on the parent, and nothing staged remains outside the output
   directory once it succeeds or is refused.
6. **A `.git` path component is refused** in output paths, container entry paths
   and entry names, compared as Git compares it for CVE-2014-9390: case
   insensitively, ignoring the code points HFS+ ignores, and including the NTFS
   short name `git~1`. Writing `.git/hooks/*` or `.git/config` runs code at the
   next git command in that directory, which no other output path can do by
   existing. (Proposed ruling 9.)
7. **Standard output** involves no filesystem; the tar archive's metadata is
   fixed (ADR-005).

### D9 — Resolving partials, assets, embeds and schema references

1. **One resolver, over the package index.** Partials (`render`), assets
   (`asset`), embedded inputs (`embed`) and schema `$ref`s are all resolved by a
   `PackageFS` value built over the verified entry index of a packed package or
   the walked tree of a source directory. Template engines and schema compilers
   receive this value, never a path.
2. **Partials** come from `templates/` only. The name is the quoted string the
   tag carries (Shopify's `render` requires one); it is checked by D6 item 1 and
   looked up exactly. A missing partial is a template error: there is no
   fallback to the working directory, to `$HOME` or to any search path. The
   deprecated `include` tag does not exist (ADR-004).
3. **Assets** come from `assets/` only, by the same rules.
4. **`embed`** names a declared input, not a path; it yields that input's
   supplied bytes.
5. **`.compiled/`, `examples/` and `references/` are unreachable from
   templates**: ADR-004 grants no tag that reads them (`render` reads
   `templates/`, `asset` reads `assets/`, `embed` reads inputs).
6. **Schema `$ref`s** resolve against a fixed base: each schema entry has the
   retrieval URI `metais:///<package path>` (ADR-003 D5 item 1), and the
   scheme is never dereferenced. A reference may resolve only to the same
   document, to another schema entry under `inputs/` (for an output
   validation schema, under the top-level directory that holds it, ADR-005
   D11), or to an embedded 2020-12 metaschema by its canonical
   `https://json-schema.org/draft/2020-12/…` URI. Anything else — `file:`,
   `http:`, `https:` beyond the metaschemas, a path leaving that directory —
   is refused at `check` and at read (ADR-003's `schema.ref_unresolved` or
   `schema.ref_escapes`). The validator's
   loader is replaced by one that serves exactly this set (with
   `santhosh-tekuri/jsonschema`, `Compiler.UseLoader`), so the library's default
   `FileLoader` is never installed.
7. **`format` assertions are syntactic.** No `format` (hostname, email, URI,
   idn-hostname) is checked by a DNS lookup or any network access.

### D10 — Parsers and regular expressions

1. **YAML** (the manifest, YAML inputs, Markdown frontmatter):
   - **anchors, aliases and merge keys (`<<`) are refused** wherever they
     appear; nothing is ever expanded. The document is parsed to a node tree,
     which holds an alias as an unexpanded node, and the tree is walked before
     any value is built. A ratio heuristic such as go-yaml's is not a bound, and
     it differs between libraries, so a package accepted by one implementation
     could be refused by another;
   - every explicit tag is refused (ADR-002 D2 item 5, which ADR-003 D10
     adopts), so no tag selects a Go type: documents decode only into generic
     values;
   - exactly one document per file;
   - `value-depth` and `number-length` apply.
2. **JSON**: `value-depth` is checked by a token-level scan before decoding;
   `number-length` applies; **duplicate object keys are refused** (two parsers
   disagreeing on a manifest is the same differential as two central
   directories).
3. **CSV**: the header row fixes the field count of every record; `csv-rows`,
   `csv-columns` and `csv-cell` are checked as each record is read.
4. **Markdown**: the parser used by `outline` and by frontmatter splitting must
   be linear on cmark's pathological cases, which the hostile corpus includes;
   `frontmatter-size` applies.
5. **`grep`** uses RE2 syntax as Go's `regexp` defines it (ADR-008 owns the
   command). `pattern-length` and `regex-program` apply, the program size being
   the instruction count of the compiled `regexp/syntax` program.
6. **Schema patterns** (`pattern` and `patternProperties`; ADR-003 D4 item 2
   refuses `format: regex`) use the **METAIS pattern profile**: ECMA-262
   patterns in `u` mode, translated to RE2 and matched by Go's `regexp`.
   - *Translated*: literals and escapes; `\uXXXX` and `\u{X…}` to `\x{…}`; `\cX`;
     `\0`; `.` to `[^\n\r\x{2028}\x{2029}]`; `\s` and `\S` to ECMA-262's
     whitespace set and its complement; `\d \D \w \W \b \B` unchanged (ASCII in
     both); `[]` to a class matching nothing and `[^]` to one matching any code
     point; `\p{…}` for general categories and scripts named in a form RE2
     accepts; character classes, quantifiers and their lazy forms; `{n}`,
     `{n,}`, `{n,m}` with bounds up to 1000; groups, non-capturing groups,
     named groups; alternation; `^` and `$`.
   - *Refused at `check` and at read* (`schema.pattern_unsupported`, naming
     the construct): lookahead and lookbehind, numbered and named backreferences,
     repeat bounds above 1000, lone surrogates, `\p{…}` forms RE2 cannot
     express, and RE2-only syntax that ECMA-262 does not define (`(?i)`, `\A`,
     `\z`, `\Q…\E`, `(?P<…>)`), which would make a package unportable to a
     conforming ECMA-262-based implementation.
   - *Why translation preserves meaning*: a schema asks only whether a match
     exists anywhere in the string (patterns are unanchored, §6.4). For the
     regular constructs above, the set of strings with a match is the same
     under backtracking and under RE2's leftmost-first automaton; greedy and
     lazy forms differ only in which match is reported, which a schema never
     sees. The class definitions are where the dialects differ, and the
     translation makes them agree.
   - The specification carries the exhaustive translation table; every row has
     a vector.
7. **`regex-work`** charges each match the program's instruction count times the
   subject's byte length, an upper bound on the automaton's steps. A charge that
   would exceed what remains refuses the match before it runs, so a
   4,096-instruction pattern cannot be run over a 32 MiB string.

### D11 — Bounded template execution

1. **The engine enforces** `template-size`, `template-nesting`, `partial-depth`,
   `loop-iterations`, `value-size`, `array-size`, `assign-bytes`, `render-work`,
   `outputs`, `output-size` and `output-total` while it runs, and refuses at
   the bound. A range literal whose span exceeds what remains of
   `loop-iterations` is refused before its first iteration.
2. **`render-work` units are defined, not measured**, so the bound is identical
   in every implementation:
   - each node evaluated (an output tag, a tag, a filter application, an
     operator) costs 16 units;
   - each filter application and operator additionally costs the *shallow size*
     of its input, of its arguments and of its result — a string's byte length,
     an array's element count, an object's key count, 1 for anything else;
   - comparing two arrays or objects for equality costs their deep size.

   The specification carries the charge of every filter in ADR-004's catalog;
   where a filter's true cost exceeds its shallow charge (sorting), the table
   charges it.
3. **A recursion the engine cannot bound is a defect of the engine.**
   `osteele/liquid` v1.9.2 has none of these limits and no seam to add them,
   which is one reason ADR-010 D4 implements METAIS's own engine, whose
   meter (ADR-010 D4 item 4) counts every unit of item 2 and refuses at the
   bound.

### D12 — Host isolation, exhaustively

The table is the complete list of host facts. For each: how it could reach a
template, an output or a message, and what closes it. "Tripwire" means the
static rule of D16 item 3, which fails the build on the named symbols.

| Host fact | Channels | Closed by |
|---|---|---|
| **Environment variables** | a binding; a filter or tag; a manifest field expanded (`${HOME}` in a path, default or description); an input value expanded; a library reading one lazily | no binding, filter, tag or manifest field exposes it (D13); nothing is expanded (D7 item 6; ADR-002 states manifest strings are literal); the process environment is emptied at startup (D14 item 1); tripwire on `os.Getenv`, `os.LookupEnv`, `os.Environ`, `os.ExpandEnv`, `os.Expand`, `syscall.Getenv`, `syscall.Environ` |
| **Files outside the package and named inputs** | partial or asset fallback to the filesystem; schema `$ref` to `file:`; a link in a data or source directory; reading the output directory | D9; D7 items 1–3 and 8; D8 item 4; all file access confined to `internal/hostfs` (tripwire on every `os` file function and `os.DirFS` elsewhere) |
| **Working directory** | relative-path resolution exposed in a message; `filepath.Abs`; the Liquid library's default store | paths are never made absolute (D15); the default store is never installed (D9 item 1); tripwire on `os.Getwd`, `os.Chdir`, `filepath.Abs`, `filepath.EvalSymlinks` |
| **Home directory, config and cache directories** | a configuration file; `~` expansion | the tool has no configuration file; tripwire on `os.UserHomeDir`, `os.UserConfigDir`, `os.UserCacheDir` |
| **Temporary directory** | staging under `$TMPDIR` | D8 item 5; tripwire on `os.TempDir`, `os.CreateTemp`, `os.MkdirTemp` |
| **User and groups** | provenance; `init`'s scaffold (an "author" field) | the provenance list is closed (ADR-004); `init` writes fixed text plus its arguments (ADR-007); tripwire on `os.Getuid`, `os.Geteuid`, `os.Getgid`, `os.Getegid`, `os.Getgroups` and the `os/user` package |
| **Host name** | provenance; messages | tripwire on `os.Hostname` |
| **Process** | provenance; staging names; `os.Executable` in messages | staging names are fixed (ADR-005); tripwire on `os.Getpid`, `os.Getppid`, `os.Executable` |
| **Clock** | `date` with `"now"` or `"today"`; provenance time; a default seed; file modification times | `date` refuses `"now"` and `"today"` whether or not `--now` is given, and the render time reaches a template only as the `now` global, from `--now` (ADR-004 D7 item 7, D11 item 1); provenance time only from `--now`; modification times are never read into data and collections are ordered by name; written timestamps are fixed wherever they are content (zip, tar); tripwire on `time.Now`, `time.Since`, `time.Until`, timers and tickers |
| **Time zone** | formatting in `time.Local`; `time.Parse`; `time.Unix`; named zones from the host database | `time.Local` is set to UTC at startup (D14 item 1); dates are UTC or carry their own offset, and no named zone exists (ADR-004 D11); tripwire on `time.Local`, `time.LoadLocation`, `time.Parse`, `time.Unix`, `time.UnixMilli`, `time.UnixMicro` outside `internal/dates`, whose own tests run under several `TZ` values |
| **Locale** | number, date, case and collation behaviour | no locale-sensitive API is used; sorting is by code point (ADR-004); `LANG` and `LC_ALL` are cleared with the rest of the environment |
| **Random numbers** | a default seed; map iteration order; hash seeds | the sample generator's seed is explicit, with a fixed default, and its generator is METAIS's own SplitMix64 (ADR-007 D9), never the clock or a random source; staging names are fixed (ADR-005 D14); iteration order is defined (ADR-004); tripwire on `crypto/rand`, `math/rand` and `math/rand/v2`, with no exception |
| **Network and DNS** | `$ref` or `$schema` fetch; `format` checks; anything else | D9 items 6–7; the import rule of D16 item 3 refuses `net`, `net/http` and the rest in the binary |
| **Subprocesses and plugins** | running a converter; loading code | the import rule refuses `os/exec` and `plugin` |
| **Terminal and resources** | terminal width, `isatty`, CPU count, memory | these may change human-format decoration only, never outputs, findings or JSON; parallel work is ordered deterministically (ADR-005); resource exhaustion is a failure, never content |
| **Host paths in messages** | Go errors carrying the path they were given; a finding locating a file absolutely; a panic | D15 |
| **Build machine** | source paths in panics; build settings | built with `-trimpath` (ADR-010); the tool version is a build constant |

### D13 — The binding set is a closed, constructed value

1. **The template sees exactly** ADR-004 D7's globals — `inputs`, `package`
   (its fields fixed from the manifest), `now`, `sources`, and, where D7 item
   10 binds them, `item`, `fan_out` and `output` — and the loop and
   assignment variables Liquid creates. `--now` reaches templates only as
   `now` and through the provenance tag's `time` field. There is no `env`,
   `host`, `system`, `site`, `page` or `request`; ADR-004's strict-variable
   rule makes `{{ env.HOME }}` an error at `check` and at render.
2. **The set is built by one constructor** (`internal/bindings`) from the
   validated inputs and the manifest, and from nothing else. Its values are
   plain data only: maps with string keys, slices, strings, booleans, `nil` and
   ADR-004's number type. No struct, pointer, function, channel, `time.Time`,
   or value implementing the library's drop interface ever enters it, so no
   template can call a Go method.
3. **A test walks the set built for every vector** by reflection and fails on any
   other type.
4. **The engine has no filesystem.** METAIS's own engine (ADR-010 D4 item 5)
   imports no `os` and has no default store: the `PackageFS` resolver of D9
   is its only source of partials.

### D14 — Structural enforcement in the binary

1. **`internal/hostenv.Isolate()` runs first in `main`**, before flags are
   parsed (`cmd/metais` calls it, ADR-010 D2):
   1. it reads `NO_COLOR`, the one display setting (ADR-008 D12), into a value
      it returns;
   2. it calls `os.Clearenv()`, so any later `os.Getenv` anywhere in the
      process, including in a dependency, returns `""`;
   3. it sets `time.Local = time.UTC`, so a dependency that formats in local time
      formats in UTC and never reads `$TZ`, `/etc/localtime` or a file `$TZ`
      names.

   These are defence in depth. The guarantee is that no code path asks for a
   host fact (D12, tripwire); `Isolate()` makes a missed path return nothing.
2. **No safety check relies on a `GODEBUG` setting** or any other
   runtime-environment switch: every check in D5 to D8 is METAIS's own. The
   Go runtime parses `GODEBUG` before `main`, so clearing the environment
   does not undo it; it changes nothing the tool accepts or writes because
   the tool reads no archive through `archive/zip` or `archive/tar` (ADR-010
   D11 items 1 and 4), and ADR-011 D12's environment A, which sets
   `GODEBUG=zipinsecurepath=1,tarinsecurepath=1`, and ADR-010 D13's
   determinism run, which sets the opposite, both require byte-identical
   results.
3. **`internal/hostfs` is the only package that touches the host filesystem.** It
   opens caller-named paths as `os.Root`s and offers reads (regular files only,
   D7) and confined writes (D8). Its errors carry paths relative to the root and
   a reason reduced to its `errno` meaning ("permission denied", "no such
   file").
4. **`cmd/metais` alone touches the process**: `os.Args`, the three standard
   streams and the exit status.
5. **A panic is recovered** at each command's entry and reported as an
   internal-error finding without a stack trace or any value. A runtime fatal
   error (stack exhaustion, out of memory) cannot be recovered; the limits make
   both unreachable, and the hostile corpus fails if any case crashes.

### D15 — Messages carry no host facts

1. **A path in a finding or message is one of**: a package entry path; a path
   relative to the data directory; a path relative to the output directory; or a
   role — "the package", "the data directory", "the output directory", "the
   standard input given to `--inputs-json`". The tool never prints an
   absolute path, a path made absolute, a home-relative path, or a path the
   caller supplied — a package operand or a directory argument — itself,
   in a message, a JSON document or a suggested command line (ADR-008 D11
   item 5 and D14 item 2 adopt this). (Proposed ruling 6.)
2. **OS errors are rewritten** into that vocabulary by `internal/hostfs`.
3. **Echoed text is neutralised in human-format output**: when a finding,
   listing, outline or `grep` line echoes package or input text to a person,
   C0 and C1 controls, DEL and the bidirectional formatting characters
   U+202A–U+202E and U+2066–U+2069 are printed as `\u` and four lowercase
   hex digits (`\u001b`, `\u202e`), the form ADR-006 D9 item 9 uses, so
   package text cannot drive a terminal or reorder what a reviewer sees
   ("Trojan Source"). A finding escapes the tab as well, since its fields
   are single lines; a listing, outline or `grep` line keeps it, since it is
   content. JSON output carries the exact text, JSON-escaped. `read`,
   and `render` to standard output, print bytes unchanged, as requested.
4. **Schema locations in findings** use the `metais:///…` form of D9 item 6,
   or the entry path (ADR-006 D6 item 8), never a host URL.

### D16 — Proving host isolation

The proof has three parts that fail differently: a dynamic harness that looks
for host facts in everything the tool emits; canary builds that prove the
harness can see a leak; and a static tripwire that fails the build when code
that could ask for a host fact appears.

1. **The SC7 harness** (`TestHostIsolation`) is ADR-011's conformance runner
   (ADR-011 D11, D12) with the additions below. The runner already runs the
   real binary, built with release tags, as a subprocess for every step of
   every conformance vector and every hostile-corpus case, in its sentinel
   environment A and its plain environment B; this harness extends that run
   so it covers every command and every channel:
   - **Every command.** Besides the runner's operations, each vector's
     package is also run through `ls --recursive`, `read` of every entry,
     `outline` of every Markdown entry, `grep` over every entry, `schema`
     and `extract`, and `render` with `--dry-run` and to standard output, in
     human and JSON form.
   - **Sentinels** are ADR-011 D12's: letters only, beginning `MTSSENTINEL`
     and naming their channel (`MTSSENTINELHOME`, …), so they survive
     `slug`, HTML, JSON, CSV and URL escaping and case changes, and a hit
     names the channel it came from.
   - **More of the environment.** To environment A this harness adds
     `OLDPWD`, `SHELL`, `SOURCE_DATE_EPOCH`, `CI`, `GITHUB_TOKEN`,
     `AWS_SECRET_ACCESS_KEY` and 32 variables named `METAIS_<n>`, each valued
     with a sentinel (paths with a sentinel component). Nothing is inherited.
   - **Decoy trees.** Beyond ADR-011 D12 item 2's decoys, the working
     directory, `HOME` and `TMPDIR` each hold, with sentinel content, a decoy
     of every name the vector's package refers to (`templates/…`,
     `assets/…`, `<input>.json`, `metais.yaml`) and of the files a careless
     tool might read (`.config/metais/config.yaml`, `.metais/`,
     `.ssh/id_ed25519`, `.netrc`, `.gitconfig`).
   - **Two path modes.** Each case runs once with the data and output
     directories outside the working directory, given by absolute paths
     under a directory whose name holds a sentinel, and once with them given
     as paths relative to it, so a tool that prints a caller's path or makes
     one absolute shows a sentinel.
   - **Streams scanned**: standard output, standard error, every written file,
     every entry of every written container (decompressed), every member of a
     standard-output tar, and the raw bytes and every decompressed entry of a
     package `pack` wrote.
   - **Check 1, sentinels**: no sentinel appears, case-insensitively, in any
     stream.
   - **Check 2, real host facts**: the stream contains none of the test
     machine's host name and user name (when longer than three characters),
     the absolute paths of the test's temporary root, the runner's real home
     directory and the checkout.
   - **Check 3, the differential**: ADR-011 D12 item 4's comparison of A
     (sentinels, `TZ=Pacific/Chatham`, `LC_ALL=tr_TR.UTF-8`) with B (plain,
     `TZ=UTC`, `LC_ALL=C`), with B's batch started at least two seconds
     after A's and in other processes, so the clock and the process id
     differ too. Every stream and the exit status must be byte-identical
     between A and B. A leak that is hashed, encoded or compressed beyond
     the scanner's reach still changes the bytes.
   - **Check 4, nothing written outside** (SC6): ADR-011 D10 item 4's
     snapshot of the whole sandbox, decoys included, before and after each
     step; only a declared output tree may change, and on a refusal nothing
     does.
2. **Canary builds keep the harness honest.** Under build tags
   `metais_canary_<channel>`, files named `canary_<channel>.go` insert one
   realistic leak each, reading the host fact from a snapshot taken before
   `Isolate()` so the defence in depth does not hide it. These are build
   tags, not ADR-010 D13's injected faults, because the harness drives a
   subprocess, which no injected option can reach; ADR-010 D13's faults
   cover the guarantees proved in-process:

   | Canary | The leak it inserts | Its probe vector (release behaviour) | What turns red |
   |---|---|---|---|
   | `env` | the binding set gains `env`, the environment as a map | renders `{{ env.HOME }}` (refused: unknown variable) | check 1 |
   | `cwd` | a missing partial falls back to the working directory | renders a partial absent from the package (refused) | check 1 |
   | `home` | a missing asset falls back to `$HOME` | refers to an absent asset (refused) | check 1 |
   | `tz` | `date` formats in the host's zone | `{{ 0 \| date: "%H:%M %z" }}` gives `00:00 +0000` | check 3 |
   | `clock` | `"now"` is `time.Now()` | `{{ "now" \| date: "%s" }}` without `--now` (refused) | checks 1 and 3 |
   | `path` | findings print the data directory's absolute path | a missing required input | checks 1 and 2 |
   | `pid` | provenance records the process id | a provenance comment | check 3 |

   `TestHostIsolationCanaries` builds each canary binary in every `go test`
   run and requires the harness to report at least one violation on that
   canary's channel, and builds the release binary and requires none. The harness is written, with the
   canaries, before the code that makes the release build pass: pinned red
   first, and kept red-capable for as long as the tree exists.
3. **The static tripwire** (`TestNoHostFacts`, in `internal/guarantees`,
   ADR-010 D11 items 9 and 10) runs over the source closure of `cmd/metais`
   as `go list -deps` reports it with release tags, for GOOS `darwin`,
   `linux` and `windows`:
   - **Import rule**: the binary's dependency closure contains none of `os/exec`,
     `plugin`, `net`, `net/http`, `net/rpc`, `net/smtp`, `crypto/tls`, `os/user`,
     `os/signal`, `log/syslog`, `runtime/cgo` or `syscall/js`. `net/url` and
     `net/netip` are parsers and do not import `net`; they are allowed.
     Because `golang.org/x/sys/windows` imports `net`, this rule keeps
     `golang.org/x/sys` and `golang.org/x/term` out of the windows binary
     (ADR-010 Amendment A1, proposed).
   - **Symbol rule, module code**: in every non-test file of the module, a
     selector resolving (through that file's imports) to a symbol of D12's
     tripwire lists, ADR-010 D11 item 9's additions (`runtime.GOOS`,
     `reflect` outside the schema adapter), or to an `os` file function, is
     allowed only in the function the allow-list names:
     `internal/hostenv.Isolate` (the `NO_COLOR` read, `os.Clearenv`,
     `time.Local`), `internal/hostfs` (file functions and `os.OpenRoot`),
     `cmd/metais` (`os.Args`, `os.Stdin`, `os.Stdout`, `os.Stderr`,
     `os.Exit`), `internal/dates` (`time.Parse` of `--now` and `time.Unix`
     and its kin, never with `time.Local`). No package may use a random
     source.
   - **Directive rule**: no `//go:linkname`, no `import "C"`, no dot import, and
     no `unsafe` outside an allow-listed function.
   - **Dependency scan**: the same symbol lists are searched in every non-standard
     package of the closure. Each hit must match an allow-list entry giving the
     import path, file, symbol and the reason it cannot reach a template, an
     output or a message (for example, a library's own debug logging path
     that METAIS never calls). An entry that no longer matches a hit also
     fails, so the
     list stays exact and a dependency upgrade that adds an `os.Getenv` breaks
     the build.
   - **Canary rule**: only `canary_*.go` files carry a canary tag, and the
     release file list contains none of them.
4. **The binding-closure test** of D13 item 3.

### D17 — The hostile corpus

ADR-011 owns its form: every case is a vector tagged `security`, with its
package committed as bytes and the program that produced them kept with a
test that it reproduces them (ADR-011 D16 item 5). A case that would need a
large file proves its rule at a lowered limit instead (D4 item 4): an
`entries`-plus-one case uses `--limit entries=3`, and a pathological pattern
runs over an input sized to a lowered `input-size`. A FIFO is created by the
vector's setup (ADR-011 D9 item 4); a device cannot be created without
privilege, so the device cases are unit tests of `internal/hostfs`. Every
case names its expected finding code and runs under the SC7 harness. The
corpus contains at least these cases:

1. **Entry names**: `../x`; `a/../../x`; `/etc/x`; `C:\x`; `C:x`; `\\server\share\x`;
   `a\..\..\x`; a NUL; an empty name; `.` and `..` components; `a//b`; a file
   name ending in `/`; invalid UTF-8; over `path-length`; over `path-depth`; a
   256-byte component; Windows-invalid names (`CON`, `nul.txt`, `com1`, `a:b`
   — which on NTFS is an alternate data stream — `a*b`, `name.`, `name `, a
   control character); `.git`, `.GIT`, `.g\u200cit`, `git~1`; an unknown
   top-level root.
2. **Entry collisions and types**: duplicate names; `A.txt` beside `a.txt`; `a`
   beside `a/b`; a symbolic-link entry pointing inside and one pointing out; a
   FIFO, a device and a socket mode; a directory entry, with and without
   data; setuid and executable bits (refused, since ADR-001 D1 fixes the
   external attributes; modes are never taken from a package).
3. **Container structure**: a Fifield overlap bomb in the plain and the quoted
   form; declared sizes over `entry-size` and `package-uncompressed`; a lying
   size (more data, and less data, than declared); a 1032:1 entry over
   `entry-ratio`; a local header disagreeing with its central record in name,
   method, sizes and CRC; a gap between entries; prepended bytes; trailing bytes;
   an archive comment; two end records; a central directory count mismatch; a
   Zip64 end record or extra field; the encryption bit; unsupported methods (bzip2,
   LZMA, zstd, AES); a data descriptor; truncation; a CRC mismatch;
   `entries` + 1 entries.
4. **Digests and compiled forms**: an entry changed after packing; the digest file
   missing an entry, listing an absent one, or wrong for the package; an entry
   not covered; `.compiled/manifest.json` disagreeing with `metais.yaml`; a
   combined schema disagreeing with `inputs/`; a nested zip bomb as an asset
   (copied as bytes, never opened).
5. **Manifest and YAML**: billion laughs; a single alias; a merge key; a custom
   tag (`!!python/object`, `!include`); 10,000-deep nesting; a multi-document
   file; duplicate keys; over `manifest-size`; a million-digit number;
   `${HOME}`, `$HOME`, `%USERPROFILE%` and `~` in every string field (rendered
   literally, never expanded); an unknown major specification version.
6. **Schemas**: `$ref` to `file:///etc/passwd`, to a decoy under the working
   directory, to `https://example.com/s.json`, to `../outside.json`; a remote
   `$schema`; a `$ref` chain over `ref-depth`; a non-consuming `$ref` cycle;
   nested `anyOf` over non-consuming refs that is exponential without
   `schema-work`; nesting over `schema-depth`; a schema whose inlined form
   exceeds `combined-schema-size`; `^(a+)+$` against a run of `a` as long as
   a lowered `input-size` allows; lookahead,
   lookbehind and a backreference (refused); `\s`, `.` and `[^]` divergence
   cases (translated); `x{1001}`; a pattern over `pattern-length` and one over
   `regex-program`; `format: regex` (refused by ADR-003 D4);
   `uniqueItems` over 1,000,000 items; `multipleOf: 1e-308` against `1e308`.
7. **Templates**: `include`; `render` of `../../etc/passwd`, of `/abs`, of a name
   only the working-directory decoy has, of `.compiled/manifest.json`, of an
   asset; `asset` outside `assets/`; self-recursive and mutually recursive
   partials; 10,000 nested blocks; `(1..1000000000)`; nested loops past
   `loop-iterations`; string doubling past `value-size` and `assign-bytes`;
   `split` past `array-size`; a `contains` over a 64 MiB string in a loop (stopped
   by `render-work`); output past `output-size`; `"now"` and `"today"` given
   to `date`, with and without `--now`; `{{ env }}`, `{{ ENV.HOME }}`, `{{ site }}`, `{{ page }}`; a template
   over `template-size`; invalid UTF-8 and a NUL in a template.
8. **Outputs**: manifest paths with traversal, absolute and Windows-invalid
   forms; the same produced from input data by a path template; case collisions
   between `for_each` items produced from data; `outputs` + 1 files; a
   pre-existing symbolic link at an output path and in a parent directory,
   with and without `--overwrite`; a pre-existing FIFO at an output path; a
   container entry path traversing out; a container listing itself; `.git/hooks/`
   from data.
9. **Inputs (hostile supplier)**: a symbolic link named as an input pointing out,
   one pointing inside, a linked collection directory, a link loop; a FIFO named
   as an input (must not block); a device; inputs over `input-size` and
   `inputs-total`; CSV over `csv-rows`, `csv-columns` and `csv-cell`; a ragged
   CSV; YAML billion laughs and JSON depth bombs as inputs; a frontmatter bomb;
   `collection-items` + 1 files; collection file names with a newline, an
   escape sequence and a bidirectional override (neutralised in human output);
   input values with terminal escape sequences echoed in findings.
10. **Markdown**: every case cmark's pathological suite names, through `outline`
    and frontmatter splitting.
11. **Commands**: `grep` with `(a|aa)*b` over the largest entry; a `grep` pattern
    over `pattern-length`; a `grep` flood past `grep-matches` (truncated and
    marked); `-C` over `grep-context`; `extract` and `unpack` into a directory
    holding a planted link; `--limit` raising a limit, naming an unknown limit,
    and with a malformed value.
12. **Limits at the boundary**: for each limit, a pair at the bound and one past
    it, under a lowered `--limit` where the default would be expensive.

### D18 — Bounded work is measured

1. **Every corpus case runs as a subprocess** and the harness reads its CPU time
   (user plus system) and peak resident memory from the process state
   (`getrusage`: KiB on Linux, bytes on darwin).
2. **Ceilings.** A case refused from metadata (D5 items 1–6, name and type rules,
   `--limit` errors) completes within 0.5 s of CPU. Every other case completes
   within 10 s of CPU. Every case stays within 1 GiB of peak resident memory.
   No case exits through a Go panic or a runtime fatal error.
3. **CPU time, not wall time**, so a loaded machine slows a case without failing
   it.
4. **The calibration suite** (D3 item 3) runs each work and output limit's worst
   case at the default and records its CPU time and peak memory per supported
   target; the ceilings above are the acceptance criterion for the defaults.

### D19 — The finding codes this ADR's rules report

Spelled in ADR-006's families (ADR-006 D3). Codes another ADR registers are
named with their owner.

| Code | Class | Meaning |
|---|---|---|
| `limit.package` | `refused` | a D3 limit breached by the package's own content, wherever it is found (opening, `check`, parsing a template); fields of D2 item 5 |
| `limit.exceeded` | `content` | a D3 limit breached by the inputs, by a render's work or by its outputs; fields of D2 item 5 |
| `limit.invalid` | `usage` | `--limit` named an unknown limit, gave a malformed value, or tried to raise one |
| `package.container.*` (ADR-001) | `refused` | the D5 checks: `overlap_or_gap`, `prepended_data`, `trailing_data`, `archive_comment`, `second_end_record`, `zip64`, `header_mismatch`, `size_mismatch`, `crc_mismatch`, `method`, `encrypted`, `field_value` (a link, directory, device or setuid mode) and the rest of ADR-001's list |
| `package.name.*` (ADR-001) | `refused` | the D6 rules: `shape`, `character`, `not_portable` (Windows-reserved names and `.git`), `duplicate`, `case_collision`, `file_directory_conflict`, `directory_entry` |
| `package.compiled.mismatch` (ADR-001) | `refused` | D5 item 9 |
| `yaml.alias`, `yaml.tag`, `yaml.multi_document`, `yaml.duplicate_key`, `json.duplicate_name` (ADR-003; `manifest.yaml.*` and `manifest.json.*` for the manifest, ADR-002) | `content` | D10 items 1–2 |
| `schema.pattern_unsupported` | `content` | D10 item 6: a construct the pattern profile refuses, named |
| `schema.ref_unresolved`, `schema.ref_escapes` (ADR-003) | `content` | D9 item 6 |
| `input.link`, `input.not_regular_file` (ADR-003) | `content` | D7 items 2–3 |
| `output.path.*` (ADR-005), `output.destination.link`, `output.destination.not_regular_file` | `content`, `destination` | D8 items 2, 3 and 6 |
| `usage.bad_value` (ADR-008) | `usage` | a `grep` pattern that does not compile, or exceeds `pattern-length` or `regex-program` |
| `internal.error` (ADR-006) | `internal` | D14 item 5 |

## Interfaces with sibling ADRs

**ADR-001 — the container.** This ADR relies on ADR-001 D1's profile:
methods stored and deflate only; no encryption; no data descriptors (sizes in
the local header); no archive comment, no prepended or trailing bytes, and no
second end-record signature (adopted in ADR-001 D8 step 2); entries
contiguous from offset 0 and in sorted order; names UTF-8 with the
language-encoding flag set; fixed external attributes (a regular file,
mode 0644); no directory entries; no Zip64 (ADR-001 D3); and a digest file
that gives every entry's SHA-256 and the package's, verified for the whole
package at every open, before any content is used (ADR-001 D8). It promises
ADR-001 the structural check order of D5, which ADR-001's verification
follows.

**ADR-002 — the manifest.** Relies on: YAML with no anchors, aliases, merge
keys or explicit tags, one document; duplicate keys refused in YAML and JSON;
every manifest string literal, with no interpolation of any kind; and the
canonical JSON form being re-derivable from `metais.yaml` (D5 item 9).
Promises `manifest-size`, `value-depth` and `number-length`.

**ADR-003 — the data specification.** Relies on: a naming rule that maps a
data-directory entry to a declared input deterministically, so that D7 item 1
can open only candidates; collections one level deep; `--set`, `--set-json`
and `--inputs-json` values taken literally, with no file-reference syntax;
standard input only for `--inputs-json -`; retrieval URIs `metais:///<path>`
and the `$ref` rule of D9 item 6 (ADR-003 D5); `format` assertions that are
syntactic, with `format: regex` refused (ADR-003 D4 item 2); `pattern` and
`patternProperties` under D10 item 6. Promises the data limits of D3 and the
CSV checks of D10 item 3.

**ADR-004 — the template environment.** Relies on: exactly the globals of
ADR-004 D7, bound where its item 10 says; strict variables, so that an
unknown name is an error; `render` with a quoted name, from `templates/`
only; no `include`; `asset`, `embed` and any reference tag resolved through
D9; `"now"` and `"today"` refused by `date` whether or not `--now` is given,
the render time reaching templates only as `now`; dates in UTC or their own
offset, with no named zones; no locale; code-point sorting of strings; a
defined iteration order; the closed provenance list; and a charge for every
filter in the `render-work` table (D11 item 2). Promises the template and
work limits of D3, under D19's codes.

**ADR-005 — outputs and rendering.** Relies on: ADR-005 D12's portable-path
rules applied to output paths, container entry paths and entry names, with
the `.git` rule of D8 item 6 among its reserved names; staging beside the
output directory with fixed names, confined by `os.Root` as D8 item 1
states; all or nothing; never deleting; never reading existing contents
(D8 item 4, which ADR-005 D15 adopts, so it has no `unchanged` status);
explicit modes, never setuid or setgid; a tar profile with fixed metadata;
output paths checked before staging, whatever produced them; a
single-threaded render (ADR-010 D2 item 3). Promises the confinement
procedure of D8 and the output limits of D3, `container-depth` among them.

**ADR-006 — findings.** Relies on: a registry that takes the codes of D19,
with each code's class fixed (`limit.package` refused, `limit.exceeded`
content, `limit.invalid` usage); limit findings with the fields of D2 item
5; a truncation marker and an omitted count for `findings` and
`grep-matches`; exit status 4 for a refused package as against 1 for invalid
inputs (ADR-006 D12); and the wording rules of D15 (no host path, no
caller's path; roles named; neutralised echoes in ADR-006 D9 item 9's
form).

**ADR-007 — authoring.** Relies on: `init` writing fixed content plus its
arguments, with no user name, date, git configuration or working-directory
name (ADR-007 D1 item 2); `check` running every package limit, the pattern
profile and the `$ref` rule; `pack` reading the source through
`internal/hostfs` under D7 item 8, reading nothing outside the layout's
roots, and writing no host metadata (timestamps, uid/gid, modes, paths) into
the zip; `unpack` writing under D8; the sample generator seeded explicitly
with a fixed default, implemented by METAIS without a library random source,
and charging `regex-work` when it generates for a `pattern`.

**ADR-008 — the command line.** Relies on: `--limit NAME=VALUE` on every
command that reads a package or an input, and the `limits` command (ADR-008
D1, D4); `NO_COLOR` as the only environment variable the CLI reads, with
terminal detection by `isatty` and no `TERM` (ADR-008 D12); `grep` in RE2
syntax under the D3 limits with a truncation flag; `read` printing raw
bytes; no caller path printed anywhere, `next` included (ADR-008 D11 item
5, D14 item 2). Promises the semantics of D4.

**ADR-010 — the implementation.** Relies on: Go 1.26 (ADR-010 D1), which has
`os.Root` (since 1.24) and `Root.Rename` (since 1.25); `CGO_ENABLED=0` and
`-trimpath`; METAIS's own Liquid engine (ADR-010 D4), with no template store
but D9's resolver and a meter that enforces D11; `santhosh-tekuri/jsonschema/v6`
with a replaceable loader and regular-expression engine, and a counter for
`schema-work` in METAIS's wrapper (ADR-010 D5); `go.yaml.in/yaml/v3`'s node
API under `internal/yamlx`, which keeps aliases unexpanded; its own zip
reader, so `archive/zip` is used only for writing; `internal/hostfs` as the
one package that touches the host filesystem and `internal/hostenv.Isolate`
as the one reader of the environment (ADR-010 D2); `internal/dates` for
`--now` parsing and the `date` filter's arithmetic; the test layout for
`TestHostIsolation`, `TestHostIsolationCanaries` and `TestNoHostFacts`
(`internal/guarantees`), and a helper that reads CPU time and peak memory of
a subprocess. No new module dependency is needed by this ADR: the tripwire
uses `go/parser`, `go/build` and `go list`.

**ADR-011 — conformance.** Relies on: the runner (ADR-011 D11) and its
sentinel and plain environments (ADR-011 D12), which D16 item 1 extends; a
vector format that can express `--limit` arguments (`limits`); setup
fixtures — symbolic links, FIFOs (ADR-011 D9 item 4's `fifos`), pre-existing
output files and links; committed hostile packages with the program that
produced them (ADR-011 D16 item 5); expected findings by code; per-case CPU
and memory bounds (ADR-011 D13); and the "nothing written outside"
snapshot (ADR-011 D10 item 4). Promises the corpus list of D17 and its
expected codes.

## Conformance

Vectors are needed for every observable rule here:

1. Each limit: at the bound accepted and past it refused, with the D2 item 5
   fields, under a lowered `--limit` where the default is costly; the defaults
   themselves for the cheap limits (`template-nesting`, `partial-depth`,
   `path-depth`, `path-length`, `csv-columns`, `value-depth`, `number-length`).
2. `--limit` refusals: raising, an unknown name, a malformed value.
3. Every D5 structural rule, each D6 name and type rule, and D5 item 9.
4. YAML aliases, merge keys, tags and multiple documents; duplicate keys in JSON
   and YAML.
5. Every row of the pattern profile's translation table, and every refused
   construct.
6. `$ref` resolution: allowed targets and each refused form.
7. Template confinement: partial, asset and embed resolution; `include` absent;
   `"now"` refused with and without `--now`; unknown host-like variables.
8. Output confinement from the manifest and from data; link and FIFO handling
   with and without `--overwrite`; `.git` refusal.
9. Input confinement: links, special files, candidate selection.
10. Message rules: no absolute path in any finding; neutralised echoes in human
    output.
11. The result bounds: `findings` and `grep-matches` truncation markers.
12. The hostile corpus (D17), every case, with its code.

SC7 is not a vector; it is the runner mode of ADR-011 applied to all of them
(D16 item 1).

## Alternatives rejected

### A CPU-time or wall-clock budget as the termination guarantee

A timer makes refusal depend on the host: the same package and inputs render on
a fast machine and are refused on a slow one, which breaks byte-identical
behaviour and makes vectors flaky. Deterministic counters bound the same work
and refuse identically everywhere. A caller who wants a deadline wraps the
process.

### A backtracking ECMA-262 engine with a timeout for schema patterns

It would accept lookaround and backreferences, at the cost of exponential worst
cases held back only by a timer, with the timer's host dependence. Patterns
outside the regular subset are rare in data schemas, and §6.4 already advises
authors to stay inside a subset that RE2 can express.

### Passing schema patterns to RE2 unchanged

`\s`, `.`, `[]`, `[^]` and `\u` differ between the dialects, so a pattern would
validate differently in METAIS and in every other JSON Schema tool; RE2-only
syntax would be accepted and make packages unportable.

### Following symbolic links that stay inside the data directory

`os.Root` would make it safe against escape, but links inside still allow cycles
in collections, make an input differ from what the directory listing shows, and
leave the policy dependent on where links point. Refusing all links is one rule
with no cases; a supplier copies the file instead.

### Relying on `archive/zip`'s own checks

Its traversal check is off unless `GODEBUG` says otherwise, it does not compare
local and central headers, and it does not detect overlap. A check a host
variable can disable is not a check.

### An operating-system sandbox as the guarantee

seccomp, Landlock, `sandbox-exec` and `pledge` differ per platform, do not exist
on all of them, and cannot express "the CLI may read `NO_COLOR` but no template
may read the environment". The guarantee here is structural and provable in
`go test`; a sandbox may be added later as defence in depth.

### An `env` filter restricted to names the manifest declares

It would make the package, not the caller, decide which secrets a render sees,
and turn every package into a request for environment access. R14 item 4 puts
the decision with the caller, through an input that is validated and visible.

### Bounded alias expansion instead of refusing aliases

The only bounds libraries offer are heuristics that differ between libraries, so
acceptance would vary between conforming implementations; an author loses
nothing a plain repetition does not give.

### Truncating, skipping or repairing instead of refusing

A render that silently omits an entry or cleans a path produces output that is
not what the package and inputs say, and hides the attack from the caller.

### Letting a caller raise limits, or a trusted-package mode

Raising makes packages that need it unportable, and with no signing in v1 there
is no way to establish that a package is trusted. Defaults are set generously
instead, and calibrated (D3 item 3).

### Reading the zip by scanning local headers

Streaming readers and central-directory readers disagree on what a zip contains;
the central directory is the authority, and D5 items 5 and 6 make the local
headers agree with it exactly.

## Consequences

### Positive

- A package or input that fits no rule fails the same way, with the same code, on
  every host and in every conforming implementation.
- The environment rule is proved three ways — scanning, differential, and static
  — and the proof cannot silently go vacuous, because the canaries must keep
  turning it red.
- A dependency upgrade that adds an environment read, a clock read or a network
  import fails the build before it ships.
- Boundary vectors are cheap, because limits can be lowered.

### Negative

- METAIS parses the zip structure itself rather than trusting the standard
  library, and re-derives the compiled manifest and schema on every read of a
  packed package.
- The Liquid and JSON Schema libraries must expose counters they may not have;
  either may fail this ADR and force ADR-010's fallback.
- A package with an exotic regular expression (lookaround) must be rewritten.
- Symbolic links in a data or source directory are refused even when harmless,
  which costs a copy.
- The `render-work` charge table must be specified for every filter and kept in
  step with the catalog.

### Neutral

- Some limits are the specification's (package limits) and some are the tool's
  defaults for data and work; both are stated in the specification and none can
  be raised.

## What this does not decide

- Signing and authenticity of packages (not in v1).
- The safety of rendered content — escaping, CSV formula injection, executable
  scaffolds — beyond D8 item 6; ADR-004 and the specification's guidance cover
  escaping.
- Protection against concurrent local attackers racing the tool on its
  directories.
- Operating-system sandboxing.
- Review of dependencies beyond the tripwire (ADR-010 pins them in `go.sum`).
- Windows-specific filesystem behaviour beyond what the portable-path rules
  refuse (reparse points, alternate data streams other than via `:` in names);
  Windows is not a v1 target.
- Limits for transforms a later specification version adds.

## Proposed rulings

1. **No limit can be raised in v1**; package limits are the specification's
   maxima and the rest are exact defaults. Lean: yes — portability, and no
   trust mechanism exists. Alternative: allow raising the data limits up to a
   hard ceiling (16× the default) for a caller with large trusted data.
2. **No CPU-time or wall-clock limit**; termination rests on deterministic
   counters (`render-work`, `loop-iterations`, `schema-work`, `regex-work`).
   Lean: yes — refusal stays host-independent. Alternative: an opt-in
   `--limit cpu-time=<s>` backstop outside conformance.
3. **The default values of D3**, subject to calibration (D3 item 3). Lean: adopt
   them as proposed and amend from measurement. Alternative: ratify only after
   the calibration suite runs.
4. **Every symbolic link inside a data or source directory is refused**, even one
   pointing inside; caller-named paths may be links. Lean: yes. Alternative:
   follow links that stay inside the root.
5. **Hard links are not detected.** Lean: yes, documented as a residual.
   Alternative: refuse files with a link count above one.
6. **Messages never echo host paths, including the caller's own directory
   arguments**; directories are named by role. Lean: yes — SC7 forbids a
   sentinel in any message, and the caller's arguments may contain one.
   Alternative: echo caller arguments exactly as spelled, never made absolute.
7. **YAML anchors, aliases and merge keys are refused in every YAML document**,
   and duplicate keys in YAML and JSON. Lean: yes. Alternative: bounded expansion
   and last-wins keys.
8. **The METAIS pattern profile** (D10 item 6): ECMA-262 translated to RE2;
   lookaround, backreferences and RE2-only syntax refused. Lean: yes.
   Alternative: a backtracking ECMA engine with a timeout.
9. **A `.git` path component is refused** in outputs, container entries and
   entry names, with Git's equivalences. Lean: yes. Alternative: allow it and
   leave the risk to the caller.
10. **The compiled manifest and combined schema are re-derived and compared on
    every read of a packed package.** Lean: yes. Alternative: trust `.compiled/`
    and let `check` alone compare.
11. **`internal/hostenv.Isolate()` empties the environment and sets
    `time.Local` to UTC at startup, after reading `NO_COLOR` as the only
    display setting (ADR-008 D12 adopts it; `TERM` is not read).** Lean: yes.
    Alternative: rely on the static tripwire alone; or also read `TERM`.
12. **Canary builds stay in the tree permanently** behind build tags. Lean: yes,
    so the SC7 harness cannot become vacuous. Alternative: pin red only in the
    history of the change that adds the harness.
13. **The tripwire is a syntactic scan with an exact allow-list**, adding no
    dependency. Lean: yes. Alternative: a reachability analysis with
    `golang.org/x/tools/go/callgraph/rta`, more precise but a new dependency.
14. **`findings` and `grep-matches` truncate with an explicit marker** rather than
    refuse. Lean: yes. Alternative: no cap on findings, as R6 item 1 read
    literally implies.
15. **Human-format output neutralises control and bidirectional characters** in
    echoed text; `read` and standard-output renders stay raw. Lean: yes.
16. **The container profile is strict** (contiguous entries, no prepended or
    trailing bytes, no comment, no data descriptors, no second end-record
    signature in the final 65,557 bytes), which ADR-001 adopts. Lean: yes;
    it is what makes the parser differential impossible. Cost: a stored
    nested zip whose own end record lands near the end of a small package
    is refused, and `pack` reports it rather than write it. Alternative:
    refuse a second signature only where it parses as a self-consistent end
    record pointing at a central directory.
17. **Zip permission bits are never applied.** Lean: yes; modes come from ADR-005.
18. **`entry-ratio` applies only to entries above 1 MiB.** Lean: yes; small
    entries are bounded by size alone.
19. **A limit breach is `limit.package` (refused, exit 4) when the package's
    own content breaches it and `limit.exceeded` (content, exit 1) otherwise**
    (D19). Lean: yes. Why: the caller who can fix it differs, and a code's
    class is fixed (ADR-006 D3 item 4). Alternative: one code, which would
    need a class per occurrence.
20. **`container-depth`, default 3, joins the output limits** (D3). Lean:
    yes; ADR-005 D8 item 4 needs it.
21. **The hostile corpus is committed bytes; a large case proves its rule at a
    lowered limit** (D17, ADR-011 D16). Lean: yes. Alternative: generate
    large cases from a recipe at test time.

## References

- PKWARE, *APPNOTE.TXT — .ZIP File Format Specification*:
  https://pkware.cachefly.net/webdocs/casestudies/APPNOTE.TXT
- David Fifield, *A better zip bomb* (sizes, overlap, quoting, detection, CVE-2019-13232):
  https://www.bamsoftware.com/hacks/zipbomb/ ; WOOT '19:
  https://www.usenix.org/conference/woot19/presentation/fifield
- Snyk, *Zip Slip vulnerability*: https://security.snyk.io/research/zip-slip-vulnerability
- Go `archive/zip` reader (`zipinsecurepath`, `findBodyOffset`, `checksumReader`):
  https://github.com/golang/go/blob/master/src/archive/zip/reader.go ; entry
  modes: https://github.com/golang/go/blob/master/src/archive/zip/struct.go
- Go 1.20 release notes (`zipinsecurepath`, `tarinsecurepath`): https://go.dev/doc/go1.20
- Go 1.24 release notes (`os.Root`): https://go.dev/doc/go1.24 ; `os.Root`
  documentation: https://github.com/golang/go/blob/master/src/os/root.go
- Go `regexp` (RE2, linear time): https://pkg.go.dev/regexp ; `regexp/syntax`
  (classes, limits): https://pkg.go.dev/regexp/syntax and
  https://github.com/golang/go/blob/master/src/regexp/syntax/parse.go
- RE2 syntax: https://github.com/google/re2/wiki/Syntax
- Russ Cox, *Regular Expression Matching Can Be Simple And Fast*:
  https://swtch.com/~rsc/regexp/regexp1.html
- JSON Schema 2020-12 Core, §6.4 Regular Expressions:
  https://json-schema.org/draft/2020-12/json-schema-core#section-6.4
- ECMA-262, §22.2 RegExp objects (`\s`, `.`, lookaround, backreferences):
  https://tc39.es/ecma262/multipage/text-processing.html#sec-regexp-regular-expression-objects
- `santhosh-tekuri/jsonschema` v6 (`FileLoader`, `UseLoader`, `UseRegexpEngine`,
  `uniqueItems`): https://github.com/santhosh-tekuri/jsonschema
- `osteele/liquid` v1.9.2 (`FileTemplateStore`, `RegisterTemplateStore`,
  `ParseDate`): https://github.com/osteele/liquid ;
  https://github.com/osteele/liquid/blob/main/render/config.go ;
  https://github.com/osteele/liquid/blob/main/values/parsedate.go
- Shopify Liquid (`ResourceLimits`, `MAX_DEPTH`, `now`/`today`, `render` syntax):
  https://github.com/Shopify/liquid/blob/main/lib/liquid/resource_limits.rb ;
  https://github.com/Shopify/liquid/blob/main/lib/liquid/block.rb ;
  https://github.com/Shopify/liquid/blob/main/lib/liquid/utils.rb ;
  https://github.com/Shopify/liquid/blob/main/lib/liquid/tags/render.rb
- go-yaml v3 (archived; alias ratio; depth): https://github.com/go-yaml/yaml ;
  https://github.com/go-yaml/yaml/blob/v3/decode.go ;
  https://github.com/go-yaml/yaml/blob/v3/scannerc.go ; maintained fork:
  https://github.com/yaml/go-yaml
- *Billion laughs attack*: https://en.wikipedia.org/wiki/Billion_laughs_attack
- Go `encoding/json` nesting limit:
  https://github.com/golang/go/blob/master/src/encoding/json/scanner.go
- Go `time` (`Local`, `LoadLocation`, `Parse`, `Unix`): https://pkg.go.dev/time ;
  `TZ` handling: https://github.com/golang/go/blob/master/src/time/zoneinfo_unix.go
- Go `os` (`UserHomeDir`, `TempDir`, `Clearenv`): https://pkg.go.dev/os ;
  `syscall.Clearenv`: https://github.com/golang/go/blob/master/src/syscall/env_unix.go
- `golang.org/x/tools/go/callgraph/rta` (the rejected alternative):
  https://pkg.go.dev/golang.org/x/tools/go/callgraph/rta
- cmark pathological tests:
  https://github.com/commonmark/cmark/blob/master/test/pathological_tests.py
- CVE-2014-9390 (Git `.git` equivalences): https://ubuntu.com/security/CVE-2014-9390 ;
  https://projects.eclipse.org/projects/technology.jgit/releases/3.4.2/review
- `getrusage(2)`, Linux (`ru_maxrss` in KiB):
  https://man7.org/linux/man-pages/man2/getrusage.2.html ; darwin (`ru_maxrss` in
  bytes): `man 2 getrusage` on macOS 26
- *Trojan Source* (bidirectional controls): https://trojansource.codes/
- NO_COLOR convention: https://no-color.org/

## Amendments

### A1 — A package's schemas are bounded by subschema count and location length

**Status:** Accepted 2026-10-19 by Ed Funnekotter, in design session
`01a15348-66d1-73ef-bed1-8d62aa3dff0e` (PLAN-2026-10-metais-v1, W3-A2).
**Amends:** D3 (two package limits), and ADR-010 D5 item 3 (what METAIS
checks before the library sees a schema). No dependency changes.

1. **`schema-subschemas`**, default 20,000: the subschemas in all of a
   package's schema files together. A subschema is the root of a file and
   every value the walk of ADR-010 D5 item 3 enters as a schema (each
   keyword ADR-003 D3 gives a schema, an array of them or an object of
   them), booleans included. It is the package's total, not a file's,
   because one compilation follows `$ref` across files. The combined
   schema (ADR-003 D7) is counted too, as a whole, and refused like
   `combined-schema-size`: `limit.package` at `.compiled/schema.json`.
2. **`schema-location-length`**, default 512 bytes: a subschema's location,
   which is the URI of its resource (the canonical URI of the nearest
   enclosing `$id`, or the file's retrieval URI `metais:///<path>`,
   whichever is longer), then `#`, then its JSON pointer within its file
   as RFC 6901 escapes it. It is checked in each schema file. In the
   combined schema a location is longer by `/$defs/` and one package path,
   which `path-length` bounds, and calibration measures that form.
3. **Both are refused before the library sees a schema**, in METAIS's
   pre-check, with `limit.package` at the file and pointer of the subschema
   that breaks them: the first subschema past the count, or the first
   location over the length. Like every package limit, `--limit` may lower
   either one and nothing raises it (D4). No finding code is added.
4. **Why these two measures.** Compiling a schema with
   `santhosh-tekuri/jsonschema` v6.0.3 costs time quadratic in the number
   of subschemas, multiplied by the length of their locations, and memory
   linear in the subschema count times the location length. Its compile
   queue finds a schema already queued by comparing locations one by one,
   its meta-validation rebuilds each subschema's whole location, and every
   compiled subschema keeps its full location as a string of its own. Byte
   size does not measure that cost. Measured on darwin/arm64, a 273 KiB
   schema of 20,000 empty subschemas compiles in 1.4 s, and 5,000 beneath
   one 10,000-byte member name take 5.8 s in 78 KiB. FHIR R4's 3.3 MB
   schema, with 15,521 subschemas and a longest pointer of 105 bytes,
   compiles in 0.26 s and 50 MB. Lowering `schema-size` far enough to bound
   the hostile shapes would refuse schemas like FHIR's, and these two
   limits refuse neither FHIR's nor a field glossary of ordinary names.
5. **The defaults are calibrated.** D18 item 4's suite runs both limits'
   worst case together: the count at its default with every location at
   its default length, in a file and in the combined schema. A default that
   breaks a D18 ceiling on a target is lowered by a further amendment
   (R1-015).
6. **The library is not changed.** Two small changes to it remove its
   quadratic time: a map index beside its compile queue, and a trie of
   resource pointers in meta-validation. They were prototyped and pass its
   own suite, but no ruling adopts or offers them. The limits bound the
   cost whichever version of the library is admitted, and the calibration
   suite is what shows that they still do after a library update
   (ADR-010 D3 item 5).

## Implementation log

- **2026-10-18, PLAN-2026-10-metais-v1 W1-4.** D16 item 3's static tripwire
  landed as `TestNoHostFacts` in `internal/guarantees`, with the
  `os.Getenv` fixture and `testdata/netleak/` as controls run by the test
  itself, and the module allow-list exactly R2-143's. It reads all six
  targets of the gate, not only one architecture per GOOS, because the
  dependency scan's hits differ by architecture. Its first run found `net`
  in the windows closure: `golang.org/x/sys/windows` imports it, and
  `golang.org/x/term` brought it in. `internal/hostenv` now asks
  `GetConsoleMode` through `syscall` on windows, so neither module is
  linked there; any later use of `golang.org/x/sys/windows` in the binary
  is refused by the import rule.

- **2026-10-19, PLAN-2026-10-metais-v1 W2-5.** D7 and D8 landed in
  `internal/hostfs`. Every name a caller passes is checked component by
  component with `Lstat` under the `os.Root`, because a root follows a link
  that stays inside it in the middle of a name, and D7 item 2 follows none.
  A read refuses a non-regular file from its `Lstat` mode, opens with
  `O_NONBLOCK` on unix so a FIFO put in its place cannot block, and then
  requires the handle to be the same regular file. OS errors become D9
  item 8's reasons with no path and none of the system's text (D15 item
  2). The source-directory `pkgtree.Tree` (D7 item 8) lists only the
  source directory and the directories beneath the layout's roots, opens
  only their regular files, and is bounded by `entries`, `entry-size`,
  `package-uncompressed`, `path-length`, `path-depth` and, for its
  problems, `findings`. It refuses an empty directory for every command,
  not only `check` and `pack`, because a tree that opened would mean
  something its packed form cannot carry. The device cases are unit tests
  of `hostfs` (D17).
- **2026-10-19, PLAN-2026-10-metais-v1 W4-1.** D7's data-directory rules
  and D3's data limits are observable through `validate`, as chapter 14's
  §14.3 and §14.4 state, with the vectors `sec/data-limits`,
  `sec/data-directory-entries` and `sec/nothing-expanded`. The `path`
  canary of D16 item 2 is `internal/cli/canary_path.go`, under
  `metais_canary_path`: it writes the data directory, made absolute, into
  every finding's message. Its probe vector is `dat/required-input-missing`.
  Run by the conformance runner with that build as its tool, the probe
  fails: standard output holds the sandbox path's sentinel (check 1) and
  differs between the environments (check 3). The canary's row in
  `TestHostIsolationCanaries`, and its red run recorded here before the
  release build is shown to pass it, are W4-3's harness's.

- **2026-10-19, PLAN-2026-10-metais-v1 W4-3. The `env` and `cwd` canaries,
  red first.** D16 item 2's first canaries are build-tagged files of
  `internal/template`: `canary_env.go` binds `env`, the process's
  environment read at start, before `Isolate`; `canary_cwd.go` reads a
  partial the package lacks from the working directory found at start.
  Their probe vectors are `tpl/no-env` (`{{ env.HOME }}`) and
  `tpl/partial-not-from-host` (a partial named `SENTINEL.txt`, the working
  directory's decoy). The harness, `internal/conformance`'s
  `TestHostIsolation` and `TestHostIsolationCanaries`, could not first see
  either leak on its channel: each canary renders its probe and writes the
  leaked bytes into an output the vector does not declare, since its
  release behaviour is a refusal, and the runner scanned only declared
  trees. Run red, each was caught only as an `ok`, findings and
  nothing-else-changes failure. The runner now scans every file a step
  writes anywhere in its sandbox (D16 item 1's every written file), and the
  recorded red run of both canary builds is:
  `the env canary: [sentinel] the written file work/out/home.txt holds the
  sentinel from HOME` and `the cwd canary: [sentinel] the written file
  work/out/page.txt holds the sentinel from a working-directory decoy`,
  each beside its `ok`, findings, nothing-else-changes and A/B failures.
  The release build gives neither probe a hit: `env` is refused as an
  unknown variable (D13 item 1), and a partial is read only from the
  package's `templates/` (D9). The `path` canary, W4-1's, joins
  `canaries` with its row here, W4-3 having merged second. Its red run
  through the harness is `the path canary: [sentinel] step 1 (validate):
  standard output holds the sentinel from the sandbox path`, beside its
  A/B failure; the release build gives its probe no hit.
- **2026-10-19, design session `01a15284-df6e-7668-b799-62f4c380a84f`
  (PLAN-2026-10-metais-v1 W2-A1).** ADR-005 Amendment A1, accepted by Ed
  Funnekotter, amends D8 items 1 and 5: a render into an existing output
  directory is prepared beneath one staging directory inside it,
  `<out>/.metais-staging`, rather than beside it, and every path a package
  or an input produced is written through a root on that directory. Item
  5's rule that nothing staged remains once the render succeeds or is
  refused holds for it. A render into a new directory still stages in the
  parent. Under A1, `hostfs` proves ownership by the staging directory it
  holds, not by a handle per entry, so the host's open-file limit does not
  refuse a render within D3's limits.

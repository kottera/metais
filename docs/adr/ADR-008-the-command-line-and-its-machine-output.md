# ADR-008: The command line and its machine output — named commands over a package file or a source directory, one versioned JSON document per run, standard output for the result alone, and help that cannot drift from the code

**Status:** Accepted 2026-10-18 by Ed Funnekotter; its proposed rulings are ratified in [RULINGS-v1](RULINGS-v1.md).
**Date:** 2026-10-18
**Deciders:** Ed Funnekotter
**Designs:** PRD-001 R2 (items 1–4) and R17 (items 1–3); the command-line
surfaces of R5 item 1, R6 item 3, R12 items 4 and 5, and R13 items 1, 5 and 6.
**Depends on:** ADR-001 (entries, the entry path model, `.compiled/`, digest
verification, the references index); ADR-002 (the manifest fields `inspect`
shows, the package name syntax, canonical JSON); ADR-003 (what `--data`,
`--set` and the inline JSON flag mean, their precedence, data-directory file
names, schema resolution); ADR-004 (what a template sees of `--now`); ADR-005
(render staging, overwrite, dry run, the standard-output tar profile);
ADR-006 (the finding object, severities, codes, exit codes); ADR-007 (`init`,
`check`, `test`, `pack`, `unpack`, the sample-data generator, a source
directory's entry set); ADR-009 (resource limits and their names, host
isolation); ADR-010 (the argument parser, terminal detection, the Markdown
parser and the JSON encoder chosen).

## Context

### What the PRD requires of the command line

- **R2.1** — every part of a package can be listed, read, outlined and
  searched without unpacking it: a summary; the entries under a path; a file
  or a range of its lines; a Markdown file's heading tree; a regular
  expression search with line numbers and context; the schema of one input or
  of all of them.
- **R2.2** — selected entries can be extracted to a directory.
- **R2.3** — every command can print JSON in a documented, versioned shape.
- **R2.4** — what a supplier reads first is designed: the summary names the
  inputs with their titles, the outputs, the guide, and what to read next.
- **R17.1, R17.2** — one documented path to consume (inspect, read what the
  summary points to, get the schema, supply the data, validate, render) and
  one to author (init, edit, check, test, pack).
- **R17.3** — help text is authoritative for every command and flag, and every
  refusal names the next step.
- **R5.1, R12.4, R12.5, R13.6** — the supply flags, the dry run, rendering to
  standard output as an archive or as one output's bytes, and the sample-data
  preview each need a name and a syntax.
- **§3 non-goals** — no public Go API: the command line is v1's contract. No
  MCP mode: integrations are built on the command line. So the command line,
  and above all its JSON, is the only interface a program or an AI harness
  has.

### Who calls it

Three callers, in order of how much the design leans on each:

1. **An AI harness** (§4.1, §4.6, SC1). It cannot see a terminal, pays for
   every byte it reads, generates command lines rather than typing them, and
   recovers from a refusal only if the refusal tells it what to run next.
2. **A program** (§4.3, §4.7, SC3). It needs stable machine output, exit codes
   it can branch on, and a standard output that carries nothing but the
   result.
3. **A person** — an author (§4.8) or an operator reading a report. They need
   readable output, help, and colour that never gets in the way.

### Facts that constrain the design (verified)

- **POSIX utility syntax guidelines** (XBD 12.2) describe single-character
  options only: "Each option name should be a single alphanumeric character"
  (Guideline 3). Long options are outside them. Guideline 9 says "All options
  should precede operands on the command line"; Guideline 10 makes the first
  `--` the end of options; Guideline 13 reserves the `-` operand for standard
  input, or standard output "when it is clear from context that an output file
  is being specified", "or a file named -".
- **Go's standard `flag` package** accepts one or two dashes as equivalent,
  accepts `-flag=x` and (for non-boolean flags) `-flag x`, and stops parsing
  "just before the first non-flag argument" or after `--`. It therefore cannot
  accept a flag written after an operand. **`github.com/spf13/pflag`** states
  that "Unlike the flag package, flags can be interspersed with arguments
  anywhere on the command line before this terminator." ADR-010 chooses the
  parser; this ADR states the behaviour it must have.
- **`NO_COLOR`** (no-color.org): software that adds ANSI colour by default
  "should check for a `NO_COLOR` environment variable that, when present and
  not an empty string (regardless of its value), prevents the addition of ANSI
  color", and "per-instance command-line arguments should override" it.
- **JSON Lines** (jsonlines.org): UTF-8 without a byte order mark, each line
  a valid JSON value, the line terminator `\n`.
- **Go's `encoding/json`** escapes `&`, `<` and `>` inside strings by default
  ("So that the JSON will be safe to embed inside HTML <script> tags");
  `Encoder.SetEscapeHTML(false)` turns that off. `Marshal` sorts map keys.
- **Go's `regexp`** accepts the RE2 syntax except `\C` and "is guaranteed to
  run in time linear in the size of the input". `regexp/syntax` rejects a
  counted repetition whose minimum or maximum exceeds 1000, and `(?i)` sets
  case-insensitive matching.
- **CommonMark 0.31.2**: an ATX heading is an opening sequence of 1–6 `#`
  characters, followed by a space, a tab or the end of the line, preceded by
  at most three spaces of indentation, with an optional closing sequence of
  `#`s; it is one line. A setext heading is "one or more lines of text, not
  interrupted by a blank line" followed by an underline of `=` (level 1) or
  `-` (level 2) indented at most three spaces; when a `---` line could be a
  setext underline or a thematic break, "the interpretation as a setext
  heading takes precedence". The content of a fenced code block is literal
  text, so a `#` line inside a fence is not a heading.
- **GNU grep** prints matching lines with `:` after the prefix fields and
  context lines with `-`, and a `--` line between non-adjacent groups when
  context is requested. Its exit status is "0 if a line is selected, 1 if no
  lines were selected, and 2 if an error occurred".
- **Go's `path.Match`**: `*` "matches any sequence of non-/ characters", `?`
  any single non-`/` character, `[...]` a class, `\` escapes; the grammar has
  no `**`.
- **RFC 3339 §5.6** `date-time` is `full-date "T" full-time`, with a
  `time-offset` of `Z` or a numeric offset, optional fractional seconds, and
  lowercase `t` and `z` allowed.
- **`golang.org/x/term`** provides `IsTerminal(fd int) bool`.
- **The Go runtime reads `GODEBUG`.** Per go.dev/doc/godebug, a setting is
  derived from the toolchain defaults, `go.mod` and `//go:debug` lines only
  "when a GODEBUG setting is not listed in the environment variable" — the
  environment wins. Settings such as `zipinsecurepath` change archive
  behaviour. This is a host read the command line does not control; it is
  handed to ADR-009 and ADR-010 below.

## Decision

### D1 — One binary, named commands, one name each

**`metais <command> [operands and flags]`, with a closed set of commands, no
aliases and no abbreviations.**

1. The commands, in the groups help shows them in:

   | Group | Command | Does |
   |---|---|---|
   | Reading a package | `inspect` | the summary a supplier reads first (D6) |
   | | `ls` | the entries under a path |
   | | `read` | an entry, or a range of its lines |
   | | `outline` | a Markdown entry's heading tree |
   | | `grep` | a regular-expression search with context |
   | | `schema` | the combined input schema, or one input's |
   | | `extract` | selected entries, written to a directory |
   | Supplying and rendering | `validate` | the inputs against the schemas |
   | | `render` | the outputs, to a directory or standard output |
   | Authoring | `init` | a minimal source directory |
   | | `check` | the package, without data |
   | | `test` | the examples, rendered and compared |
   | | `sample` | generated sample data, written as a data directory or an example's inputs |
   | | `pack` | a source directory into a package |
   | | `unpack` | a package into a source directory |
   | The tool | `version` | the tool's version and the specification versions it implements |
   | | `limits` | every resource limit with its default, unit and class (ADR-009 D4 item 3) |
   | | `help` | the overview, a command's help, or a topic |

2. **The sample-data preview is `render --sample`**, and saving sample data is
   the `sample` command. Previewing is rendering, so it shares `render`'s
   flags, staging, dry run and standard-output modes rather than duplicating
   them; generating data to keep is a different act with a different
   destination, so it has its own command. Both share `--optional` and
   `--seed` and the generator (ADR-007).
3. **No command has an alias, and no command or flag name may be abbreviated.**
   A prefix that is unambiguous today becomes ambiguous the day a sibling
   command or flag is added, and a generated command line that worked would
   then fail.
4. `metais` with no command prints the overview help to standard error and
   exits with the usage code (D13). `metais --help` and `metais -h` print it to
   standard output and exit 0. `metais --version` is `metais version`.

### D2 — The package operand

**A package operand is a path to a packed package file or to a source
directory, and every reading, supplying and checking command accepts both.**

1. A path that names a regular file (after following a symbolic link the
   caller named) is read as a packed package — a zip under ADR-001's profile —
   whatever its extension. A path that names a directory is read as a source
   directory. Anything else is refused (`package.not_found` or
   `package.not_readable`, class `refused`).
2. **On a packed package, the entries are the zip's entries**, including
   `.compiled/` (so `read <pkg> .compiled/manifest.json` works). Digests are
   verified as ADR-001 decides, and a mismatch is a refusal.
3. **On a source directory, the entries are exactly the files `pack` would
   write from it** (ADR-007's entry set), with paths relative to the
   directory, and without `.compiled/`, which only `pack` produces. What an
   author sees through `ls`, `read` and `grep` is therefore what a consumer of
   the packed package will see. The views `.compiled/` precomputes are
   available on a source directory through the commands that use them:
   `inspect` (the references index) and `schema` (the combined schema),
   computed by the same code `pack` uses.
4. **The operand `-` is refused as a package** (`usage.package_from_stdin`): a
   zip needs random access, and buffering one from standard input would add a
   second way in, with its own size bound, for no use case the PRD names.
   The refusal names the next step: write it to a file.
5. **Entry paths** on the command line and in output are relative, use `/` as
   the separator on every platform, and match the stored path byte for byte
   (case-sensitive). A directory may be written with or without a trailing
   `/`. The root is written as `.` or omitted.
6. **The package operand is required** on every command except `check`,
   `test`, `sample` and `pack`, where it defaults to `.` — the author's
   working tree — and `init`, whose operand is a directory to create (D9).
7. The operand is opened as given. **The tool never turns a path it was
   given into an absolute path**, and never prints the working directory
   (D11 item 5).

### D3 — Argument syntax

**Long flags, interspersed with operands until `--`; one spelling per value;
a repeated single-valued flag is an error.**

1. **Flags are long**: `--name`. A flag that takes a value accepts
   `--name value` and `--name=value`. A boolean flag takes no value;
   `--name=anything` is a usage error, so there is one spelling of each
   boolean.
2. **Short aliases form a closed list**: `-h` (`--help`, every command), `-r`
   (`ls --recursive`), `-C` (`grep --context`), `-i` (`grep --ignore-case`).
   They are the spellings a harness or a person types from habit in `ls` and
   `grep`. A short flag with a value accepts `-C 3` and `-C3`. Short flags are
   not grouped (`-ri` is a usage error); no command has two short booleans to
   group.
3. **Flags may appear before, between or after operands**, and the first `--`
   ends the flags (POSIX Guideline 10): `metais grep pkg.metais -- -x` searches
   for `-x`. This departs from Guideline 9 deliberately: a harness writes the
   operands first and appends flags as it decides them, and a command that
   silently treated a trailing `--lines 1-20` as an operand would be worse
   than either convention. Before the command name, only the global flags
   (D4) are accepted.
4. **A single-valued flag given twice is a usage error** (`usage.repeated_flag`),
   not "the last one wins": in a generated command line a repetition is a
   mistake, and last-wins hides it. The repeatable flags are named in their
   rows below (`--set`, `--set-json`, `--inputs-json`, `--limit`,
   `test --example`).
5. **`-` as a flag value means a standard stream only where this ADR says so**:
   `--inputs-json -` (standard input) and `--out -` (standard output, on
   `render` and `pack`). A file literally named `-` is written `./-`
   (Guideline 13).
6. **Numeric values are decimal integers**, and a size given to `--limit`
   may carry ADR-009 D4's `KiB`, `MiB` or `GiB` suffix; anything else is a
   usage error.
7. **Nothing but the command line configures a run.** The tool reads no
   configuration file, no `METAIS_*` variable, and no environment variable
   other than `NO_COLOR` (D12), which affects only colour. A command line
   states a run completely, so a render reproduced from a log is the same
   render.

### D4 — Global flags

**`--json`, `--color`, `--help`, `--limit` and `--fail-on` mean the same on
every command that accepts them.**

| Flag | Value | Default | Meaning |
|---|---|---|---|
| `--json` | — | off | Print the result as one JSON document (D10). |
| `--color` | `auto`, `always`, `never` | `auto` | Colour in human output (D12). |
| `--help`, `-h` | — | — | Print this command's help to standard output and exit 0; nothing else runs. |
| `--limit` | `<name>=<value>` | ADR-009's defaults | Lower one resource limit. Repeatable. Accepted by every command that opens a package or reads inputs (all but `version`, `help`, `limits` and `init`). |
| `--fail-on` | `error`, `warning` | `error` | With `warning`, a warning fails the command (exit 1) as an error would, without changing any finding's severity (ADR-006 D2 item 5). Accepted by every command that reports findings. |

1. **`--limit` is one repeatable flag, not a flag per limit**, so ADR-009 can
   name and add limits without growing every command's flag list. A name
   ADR-009 does not define, a malformed value, or a value above the default
   is ADR-009's `limit.invalid` (class `usage`); `metais limits` lists every
   name, its unit, its class and its default (`metais.limits/1` with
   `--json`).
2. `--json` and `--help` together print the command's help as JSON
   (`metais.help/1`).

### D5 — The reading commands

**`ls`, `read`, `outline`, `grep`, `schema` and `extract` behave the same on a
packed package and on a source directory, and none writes anything except
`extract`.**

Two rules are shared:

- **Text and lines.** An entry is *text* when it is valid UTF-8 and contains
  no NUL byte; otherwise it is *binary*. A line ends at an LF. The number of
  lines is the number of LFs, plus one if the entry is non-empty and does not
  end with an LF; an empty entry has no lines. For matching and display in
  `grep` and `outline`, a CR immediately before the LF is part of the line
  ending, not of the line. `read` prints bytes unchanged.
- **The METAIS glob**, used by `grep` and `extract`. A pattern is matched
  against the whole entry path, segment by segment, with `/` as the
  separator. `*` matches any run of characters within one segment; `?`
  matches one character other than `/`; `[...]` and `[^...]` are character
  classes with `lo-hi` ranges; `\` makes the next character literal; `**`
  standing alone as a segment matches zero or more whole segments, and `**`
  anywhere else is a usage error. There are no braces. Matching is
  case-sensitive. **`*`, `?`, `[...]` and `**` never match a segment that
  begins with `.`**; a pattern segment that itself begins with `.` matches
  one. So `**/*.md` does not reach into `.compiled/`, and `.compiled/**`
  does. A pattern with no wildcard that names a directory selects every entry
  beneath it. `metais help globs` carries this text.

#### `ls`

`metais ls <package> [<path>] [--recursive | -r]`

1. With no path, lists the root. A path naming a directory lists its
   children, or with `--recursive` every entry beneath it. A path naming a
   file lists that file.
2. Each entry: `path` (a directory's ends in `/`), `type` (`file` or `dir`),
   `size` (a file's uncompressed size in bytes) or `files` (the number of files
   beneath a directory). Directories are implied by entry paths; a zip need
   not store them.
3. Order: by path, comparing the UTF-8 bytes.
4. Human output: one line per entry, `type<TAB>size-or-files<TAB>path`.
5. Refuses: a path that names nothing (`usage.entry_not_found`; next: `ls` of
   its nearest existing parent).

#### `read`

`metais read <package> <path> [--lines <range>] [--number]`

1. Prints the entry's bytes exactly as stored.
2. **`--lines <range>`**, 1-based and inclusive: `N` (one line), `N-M`, or `N-`
   (to the end). `N` must be at least 1 and `M` at least `N`, or it is a usage
   error. A range whose start is beyond the last line is refused
   (`usage.line_out_of_range`; next: `--lines 1-<total>`). An end beyond the last
   line is clamped, and the result reports the real last line. The printed
   bytes are the selected lines with their own line endings.
3. **`--number`** prefixes each printed line with its line number and a TAB.
   It requires a text entry.
4. **`--lines` and `--number` on a binary entry are refused**
   (`usage.not_text`; next: `extract`). **A binary entry is not written to a
   terminal** (`usage.binary_to_terminal`; next: redirect standard output, or
   `extract`).
5. JSON result: `path`, `size`, `binary` (bool); for text, `content` (the
   string) and `lines` (`{"first", "last", "total"}`, present with or without
   `--lines`); for binary, `content_base64`.
6. Refuses: a path naming a directory (`usage.is_a_directory`; next: `ls`).

#### `outline`

`metais outline <package> <path>`

1. Parses a text entry as CommonMark 0.31.2 and lists its ATX and setext
   headings, levels 1 to 6, in document order. A leading frontmatter block,
   delimited as ADR-003 delimits it, is skipped (its lines still count). What
   CommonMark does not make a heading — a `#` line in a code block, an HTML
   block, or seven `#`s — is not listed.
2. Each heading: `level`; `text`, the heading's inline content as plain text
   (text and code-span contents concatenated, markup removed, a setext
   heading's line breaks as single spaces); `line`, the heading's first line;
   and `end_line`, the last line of its section — the line before the next
   heading of the same or a higher level, or the entry's last line. So
   `read --lines <line>-<end_line>` reads exactly one section, which is the
   move §4.6 describes.
3. Human output: one line per heading,
   `<line>-<end_line><TAB><two spaces per level below 1><#…> <text>`.
4. An entry with no headings gives an empty list, not a refusal. A binary
   entry is refused (`usage.not_text`).

#### `grep`

`metais grep <package> <pattern> [<glob>…] [--context <n> | -C <n>] [--ignore-case | -i] [--max-results <n>]`

1. **The pattern is a Go `regexp` pattern** — RE2 syntax, linear time. It is
   matched against each line separately (without its line ending), so `^`
   and `$` anchor the line. `--ignore-case` prepends `(?i)`. A pattern that
   does not compile is a usage error naming the RE2 syntax reference. Pattern
   length and the other regular-expression bounds are ADR-009's.
2. **Globs select the entries searched.** With none, every entry is searched
   except those the dot rule excludes (so `.compiled/` is not searched unless
   named). Entries are searched in path order, lines in order.
3. **Binary entries are skipped** and listed in the result's `skipped`.
4. **`--context <n>`**, from 0 to the `grep-context` limit (ADR-009, default
   7), default 0: lines before and after each matching line.
5. **`--max-results <n>`**, from 1 to the `grep-matches` limit in force
   (ADR-009, default 10000), default 100: the number of matching lines
   reported. The search still runs to the end and reports
   `matching_lines`, the total, and `truncated`. A line with several matches
   counts once.
6. **A displayed line longer than 1024 bytes is cut** at the last UTF-8
   boundary within 1024 bytes and marked (`"text_truncated": true`; in human
   output `[…+<n> bytes]`). `read --lines <n>` gets the whole line.
7. JSON result: `pattern`, `globs`, `matches` (each `path`, `line`, `text`,
   `before` and `after` as lists of `{"line", "text"}`), `matching_lines`,
   `truncated`, `skipped`.
8. Human output follows GNU grep's form: `path:line:text` for a match,
   `path-line-text` for context, a `--` line between non-adjacent groups, and
   adjacent or overlapping groups merged. A truncation notice and the skipped
   list go to standard error (D11 item 3).
9. **No match is not a failure: `grep` exits 0 with an empty result.** `grep`
   here is a query a harness branches on by reading `matching_lines`; GNU
   grep's exit 1 would make every "not found" look like a broken command.

#### `schema`

`metais schema <package> [<input>]`

1. With no input, prints the combined input schema: `.compiled/`'s on a packed
   package, computed by ADR-003's combination on a source directory.
2. With an input, prints that input's schema as one self-contained Compound
   Schema Document: the slot's `title` and `description` beside a `$ref` to
   the input's canonical URI, with `$defs` embedding every schema resource
   that `$ref` reaches, bundled exactly as ADR-003 D7 bundles the combined
   schema. For a collection input it is the array schema, whose `items` each
   file must satisfy.
3. Human output is the schema alone, in ADR-002's canonical key order,
   indented by two spaces, so `metais schema pkg product > product.json`
   writes a usable schema file. JSON result: `input` (or `null`) and `schema`.
4. Refuses: an unknown input (`usage.unknown_input`, listing the input names;
   next: `inspect`).

#### `extract`

`metais extract <package> <glob>… --out <dir> [--overwrite] [--dry-run]`

1. Writes every entry any glob selects to `<dir>/<entry path>`. `<dir>` and its
   missing parents are created.
2. **Every glob must select at least one entry**, or the command is refused
   naming it (`usage.no_match`) and nothing is written: a glob that matches
   nothing is a typo the caller wants to hear about.
3. **A non-empty `<dir>` is refused unless `--overwrite`**, with ADR-005's
   meaning: a file at the same path is replaced, nothing is deleted. Writes
   are confined to `<dir>` under ADR-009's rules.
4. `--dry-run` lists the files it would write and writes nothing.
5. `--out` is required; there is no default directory.
6. JSON result: `dry_run`, `files` (each `path`, `size`). The destination
   is not echoed (D11 item 5).

### D6 — `inspect`: what a supplier reads first

**`inspect` prints, in one screen-sized summary, the package's identity, the
start of its guide, every input with how to supply it, every output, the
references, the examples, and an ordered list of commands to run next.**

`metais inspect <package> [--guide]`

1. **Identity**: `name`, `version`, `title`, `description`,
   `specification` (the version it targets), `digest` (the package digest,
   `sha256:<hex>`; on a source directory, the digest of the package `pack`
   would write, computed in memory, ADR-001 D6 item 3), and `source`
   (`packed` or `directory`).
2. **Guide**: the manifest's supplier guide (ADR-002). If it has at most 60
   lines and 6144 bytes it is shown whole. Otherwise the excerpt is its first
   40 lines, shortened further to end within 4096 bytes, and the summary says
   how many lines remain and that `metais inspect <package> --guide` prints it
   whole. JSON: `guide` = `{"text", "excerpt_lines", "total_lines",
   "truncated"}`.
3. **Inputs**, in ADR-003's input order, each with: `name`, `title`,
   `description`, `required`, `collection`, `forms` (the accepted forms in the
   manifest's order), `files` (the names a data directory may hold it under,
   derived by ADR-003's rule — `criteria.csv`, `criteria.json`, or
   `articles/` for a collection), `meaning` (the entry path of its meaning
   document, or `null`), `schema_entry` (its schema file's entry path) and
   `examples` (the names of the examples that supply it). Human output shows
   the first line of the description; JSON carries all of it.
4. **Outputs**, after ADR-005 expands the mirror shorthand, each with its
   declaration's fields as ADR-005 names them — at least the path template,
   the template, `for_each`, `when`, `escape` and the transforms by name. A
   person sees from this list what a render will produce before supplying
   anything.
5. **References**, from the references index (D17): each `path`, `title`
   and `size`, and the count and total size. Human output lists the first 50
   and points to `metais ls <package> references/ -r` for the rest; JSON lists
   them all.
6. **Assets**: count and total size only; `ls` lists them.
7. **Examples**: each `name`, the inputs it supplies, and whether it has
   expected outputs.
8. **Read next** — the envelope's `next` (D14), in this order, each a full
   command line with the package operand written as the placeholder
   `<package>` (D14 item 2):
   1. `metais inspect <package> --guide`, if the guide was cut;
   2. `metais read <package> <meaning entry>` for each input that has a
      meaning document, in input order;
   3. `metais schema <package> <input>` for each input;
   4. `metais ls <package> examples/<first example>/ -r`, if there is an
      example (examples in name order);
   5. `metais grep <package> <pattern> references/` with the placeholder
      `<pattern>`, if there are references;
   6. `metais validate <package> --data <dir>`;
   7. `metais render <package> --data <dir> --out <dir>`.

   This is R17.1's consuming path, generated from the package.
9. `--guide` prints only `package` and the whole guide (`guide.truncated` is
   `false`, the other sections are absent); its human output is the guide's
   Markdown alone.
10. Human layout, top to bottom: a title line `<title> — <name> <version>`,
    the description, a line with the specification version, `packed` or
    `source directory` and the digest, then the sections **Guide**, **Inputs**,
    **Outputs**, **References**, **Assets**, **Examples** and **Read next**, in
    that order, each omitted when empty except Inputs and Outputs. Sizes in
    human output are bytes below 1024 and otherwise one decimal place of KiB,
    MiB or GiB; JSON sizes are always bytes.

### D7 — Supplying inputs, and `validate`

**`--data`, `--set`, `--set-json` and `--inputs-json` are the supply flags of
`validate` and `render` (and the latter three of `sample` and `render
--sample`), with the meaning and precedence ADR-003 D17 and D18 define.**

| Flag | Value | Repeatable | Meaning (ADR-003's) |
|---|---|---|---|
| `--data` | `<dir>` | no | The data directory (ADR-003 D16). |
| `--set` | `<pointer>=<text>` | yes | One value at an inputs JSON Pointer (`/vendor/name=Acme`), the text coerced by the schema (ADR-003 D17 item 3). The flag's value is split at its first `=`. |
| `--set-json` | `<pointer>=<json>` | yes | One value at an inputs JSON Pointer, given as JSON text — the way to set `null`, an array or an object (ADR-003 D17 item 3). |
| `--inputs-json` | `<json>` or `-` | yes | A JSON object whose members are input names, each replacing that input's whole value (ADR-003 D17 item 1); `-` reads it from standard input, and may be given once. |

1. **One flag family.** `--set` and `--set-json` are the two kinds of path set
   and apply after every `--inputs-json`, each kind in command-line order
   (ADR-003 D18). The inline JSON flag is `--inputs-json`, not `--json`:
   `--json` means "print JSON" on every command (D4), and one flag name
   cannot mean both an output mode and an input on the same command line.
2. `--inputs-json -` is the only flag that reads standard input. Its size is
   bounded by ADR-009's `input-size`.
3. A command with no supply flag validates nothing supplied: every required
   input is reported missing, in one pass. That is a result, not a usage error.

`metais validate <package> [supply flags] [--resolved]`

4. Prints every finding (ADR-006), and exits with ADR-006's findings code when
   any finding is of error severity.
5. JSON result: `inputs`, each `{"name", "supplied", "from"}` where `from`
   lists where its value came from — the data-directory file by its path
   relative to the data directory, `--set`, `--set-json`, `--inputs-json`,
   `default` — so a supplier sees that its file was picked up.
6. **`--resolved`** adds `data`: the input data model templates would see
   (ADR-003) — parsed values after coercion and defaults, absent fields absent —
   in ADR-002's canonical key order. Human output prints it as indented JSON
   after the findings. It is how a supplier checks a CSV coercion without
   rendering.

### D8 — `render`

`metais render <package> --out <dir|-> [supply flags] [--only <path>] [--overwrite] [--dry-run] [--now <date-time>] [--sample [--optional all|none|mixed] [--seed <n>] [--items <n>]]`

1. **Validates first**, then renders under ADR-005: staged, all or nothing.
2. **`--out` is required**, including with `--dry-run`, so a dry run reports
   exactly what the real run would do, refusal of a non-empty directory
   included.
3. **`--out <dir>`** writes the outputs there. A non-empty directory is
   refused unless `--overwrite` (ADR-005: replace, never delete).
4. **`--out -`** writes ADR-005's deterministic tar of every output to standard
   output. **A tar is not written to a terminal** (`usage.archive_to_terminal`;
   next: redirect, or `--only <path>`). `--overwrite` with `--out -` is a usage
   error.
5. **`--only <path>`** selects one output by its rendered path — the path
   `--dry-run` lists. The whole render still runs and is validated; `--only`
   chooses what is written, so it can never make a refused render succeed.
   With `--out -`, the selected output's bytes are written raw, and may go to
   a terminal; with `--out <dir>`, only that file is written. A path no output
   has is refused (`usage.unknown_output`, listing the paths; next: `--dry-run`).
   `--only` is single-valued: standard output is either a tar or one file's
   bytes, never something whose shape depends on a count.
6. **`--dry-run`** writes nothing and lists every file the render would write,
   with `size` and `digest` (`sha256:<hex>`, ADR-005's). With `--out -` it
   lists what the stream would carry.
7. **`--now <date-time>`** supplies the render time: an RFC 3339 `date-time`
   (offset required). Anything else is a usage error. The value goes to
   ADR-004 as given; without `--now`, there is no time.
8. **`--sample`** renders from generated sample data (ADR-007 D9) — the
   author's preview of R13.6. The generated data takes the data directory's
   place, so `--data` with `--sample` is a usage error; values given with
   `--set`, `--set-json` and `--inputs-json` are fixed points the generator
   fills around, conditionals included (ADR-007 D9 item 2), so an author can
   pin one field to see one branch. The generated data is validated like any
   input before rendering. **`--optional`** (`all`, `none`, `mixed`; default
   `all`), **`--seed`** (an unsigned 64-bit integer; default `0`) and
   **`--items`** (the target count of every array and collection; default
   `3`) are accepted only with `--sample`. A render with `--sample` is an
   authoring command for ADR-006 D12 item 2.
9. JSON result: `out` (`"directory"` or `"stdout"`; the path is not echoed,
   D11 item 5), `dry_run`, `only` (or `null`), `files` — ADR-005 D16's
   records (`path`, `output`, `item`, `source`, `kind`, `size`, `digest`,
   `executable`, `status`, and a container's `entries`) — and `sample`
   (`{"optional", "seed", "items"}` or `null`). Human output, when `--out` is
   a directory: one line per file, `<size><TAB><digest><TAB><path>`, a
   container's entries indented beneath it, headed `would write` on a dry
   run.

### D9 — The authoring commands

**`init`, `check`, `test`, `sample`, `pack` and `unpack` take the source
directory as `.` by default where that is unambiguous, ask nothing, and list
everything they write.**

#### `init`

`metais init [<dir>] [--name <name>] [--dry-run]`

1. Writes ADR-007's minimal scaffold into `<dir>` (default `.`), creating it
   and its missing parents.
2. **`--name`** defaults to the last element of `<dir>` as written. When
   `<dir>` is omitted or its last element is `.` or `..`, `--name` is required
   (`usage.name_required`; next: `metais init --name <name>`). The tool does not
   look up the working directory's name: the name comes from the command line.
   A name ADR-002's syntax rejects is refused with a suggested valid name.
3. **Refuses unless `<dir>` is empty**, apart from the names a source
   directory ignores silently (ADR-007 D1 item 3: a `.git/`, a
   `.gitignore`), listing up to ten of the entries found, and writes
   nothing. It never overwrites and has no `--overwrite`.
4. Next: `metais check <dir>`, then `metais render <dir> --sample --out <preview dir>`.

#### `check`

`metais check [<package>]`

1. Runs ADR-007's check — on a source directory or a packed package — and
   prints every finding. Needs no data.
2. JSON result: `counts`, the number of findings per ADR-006 severity.
3. Next on success: `metais test <package>`.

#### `test`

`metais test [<package>] [--example <name>]… [--update] [--dry-run]`

1. Runs ADR-007's test over every example, or the ones `--example` names
   (repeatable; an unknown name is refused, listing the names).
2. JSON result: `examples`, each `{"name", "status", "differences"}`, with
   `status` one of `pass`, `fail` and `no-expected`, and `differences` each
   `{"path", "kind"}` with `kind` one of `missing`, `unexpected` and
   `differs`, plus, for a text file that differs, the first differing `line`.
   ADR-007 owns what is compared.
3. **`--update`** (source directory only) makes each selected example's
   `examples/<name>/expected/` exactly its rendered outputs: files written
   or replaced, and files there that the render no longer produces removed
   (ADR-007 D5 item 5). It removes nothing outside that directory, the
   author's own tree; the never-delete rule of render and `extract` governs
   a caller's directory. `--dry-run` lists what `--update` would write and
   remove. This is how §4.8's author records "the outputs it should
   produce".
4. Next on success: `metais pack <package>`.

#### `sample`

`metais sample [<package>] (--out <dir> | --example <name>) [--optional all|none|mixed] [--seed <n>] [--items <n>] [--set …] [--set-json …] [--inputs-json …] [--overwrite] [--dry-run]`

1. Generates sample data (ADR-007 D9), with the supplied values as fixed
   points, validates it, and writes it as a data directory — one file per
   input in ADR-007's chosen form, named by ADR-003's rule — so `render
   --data <dir>` renders it.
2. **`--example <name>`** (source directory only) writes it as
   `examples/<name>/inputs/` with an `example.yaml` — R13.6's "saved as an
   example's inputs" — and is refused when `examples/<name>/` exists at all
   (ADR-007 D9 item 21). Exactly one of `--out` and `--example` is required.
3. With `--out`, a non-empty destination is refused unless `--overwrite`
   (replace, never delete); `--overwrite` with `--example` is a usage error.
   A field the generator cannot satisfy is ADR-007's
   `schema.sample_unsatisfiable`, and nothing is written.
4. Defaults as `render --sample`: `--optional all`, `--seed 0`, `--items 3`.
5. JSON result: `optional`, `seed`, `items`, `example` (or `null`), `files`
   (each path relative to the destination).
6. Next: `metais render <package> --data <dir> --out <dir>`, or for an example
   `metais test <package> --example <name>`.

#### `pack`

`metais pack [<source>] [--out <file|->] [--dry-run]`

1. Runs ADR-007's pack — check, test, then the deterministic zip. A packed
   package as the operand is refused (`usage.not_a_source_directory`).
2. **`--out` defaults to `<name>-<version>.metais`** in the working directory,
   from the manifest's name and version. The version is in the name so that
   packing a new version never meets the old file.
3. An existing regular file at the output path is replaced atomically, since
   it is `pack`'s own previous product (ADR-007 D7 item 5); there is no
   `--overwrite`. `--out -` writes the zip to standard output, and is
   refused when standard output is a terminal.
4. JSON result: `file` (`<name>-<version>.metais` when `pack` chose the
   name, `-` for standard output, absent when the caller named the file —
   a caller's path is never echoed, D11 item 5), `size`, `digest`.

#### `unpack`

`metais unpack <package> --out <dir> [--dry-run]`

1. Runs ADR-007's unpack. The operand must be a packed package; a directory is
   refused. `--out` is required; a non-empty `<dir>` is refused, with no
   `--overwrite`, because files left in it would join the unpacked source
   and be packed with it (ADR-007 D8 item 3).
2. JSON result: `files`. Next: `metais check <dir>`.

### D10 — The JSON document

**With `--json`, a run prints exactly one JSON document, of one envelope
shape, named and versioned per command.**

1. **The envelope**, members always present and in this order:

   ```json
   {"format":"metais.validate/1","ok":false,"findings":[…],"result":{…},"next":[{"run":"…","why":"…"}]}
   ```

   - `format` — `metais.<command>/<major>`; `metais.help/1` and
     `metais.version/1` for those commands, and `metais.usage/1` for an
     unknown command.
   - `ok` — `true` when the command completed and no finding is of error
     severity.
   - `findings` — ADR-006 finding objects in ADR-006's order; `[]` when none.
     Usage errors are findings too.
   - `result` — the command's result object (the shapes in D5–D9), or `null`
     when the command was refused before producing one.
   - `next` — D14's list; `[]` when there is nothing to suggest.

2. **Encoding**: UTF-8 without a byte order mark; compact, with no
   insignificant whitespace; HTML escaping off (`SetEscapeHTML(false)`); one
   LF at the end and none inside. A document is therefore also a one-line
   JSON Lines stream, and a program can read it with any JSON parser or
   line by line.
3. **Member order is fixed** by the format's definition, never by map
   iteration. Data the tool did not invent — a schema, `--resolved` data —
   is in ADR-002's canonical key order. Lists are ordered as each command
   states. The same run gives the same bytes.
4. **Versioning.** Within a major version a format only adds: new members,
   new enumeration values. A consumer ignores members it does not know and
   treats an unknown enumeration value as "other". Removing, renaming or
   retyping a member, or changing its meaning, is a new major version, and
   happens only in a new major version of the tool. The v1 tool emits `/1`
   documents only.
5. **Each format is defined by a JSON Schema (2020-12)** committed at
   `docs/cli/schemas/metais.<command>.1.json`, with the envelope's shared
   members in `docs/cli/schemas/metais.envelope.1.json`. Every `--json`
   document the test suite produces is validated against its schema.
6. **The exit code is the same with and without `--json`.**
7. `--json` is recognised wherever it appears before `--`, even on a command
   line that otherwise fails to parse, so a harness that asked for JSON gets
   JSON for its usage error too.

### D11 — Standard streams

**Standard output carries the result and nothing else; standard error
carries everything about the result; standard input is read only when asked.**

1. **Standard output** carries, and only carries:
   - with `--json`: the one document;
   - for `render --out -` and `pack --out -`: the bytes, and nothing else;
   - otherwise: the command's human result.
2. **With `--json`, standard error is empty** — every refusal, warning and
   notice is in the document — **except when standard output carries bytes**
   (`render --out -`, `pack --out -`): then the document goes to standard
   error, after the bytes are complete.
3. **In human mode, standard error carries** refusals, usage errors, and
   notices about the result (a `grep` truncation, skipped binary entries, an
   `outline` with no headings). The findings of `validate`, `check` and
   `test`, and `inspect`'s summary, are those commands' results and go to
   standard output; every other command's warnings go to standard error. The
   human `Next:` block goes to standard output after `inspect`, `validate`,
   `check` and `test`, and to standard error after every other command, so
   that `read` and `schema` output stays exactly the entry or the schema.
4. **Nothing reaches standard output from a refused render.** ADR-005 stages
   the whole render before the first byte is written to the stream, so a
   caller never receives half an archive.
5. **Paths in any output** are entry paths, output paths relative to `--out`,
   and paths relative to a data directory or another destination the caller
   named. **No path the caller gave is ever printed** — not the package
   operand, `--data`, `--out` or `--inputs-json`'s file — in human
   output, in a JSON document, in a finding or in `next` (ADR-009 D15 item 1,
   ADR-006 D9 item 2): a caller's path may carry anything, and SC7 requires
   that no sentinel planted in a path reaches a message. Where a command
   line must name one, it uses a role placeholder (`<package>`, `<dir>`).
   The tool never prints an absolute path it constructed, the working
   directory, the home directory, the user or the host name.
6. **Standard input is read only for `--inputs-json -`.** No command prompts,
   reads the terminal, or waits for input it was not told to read; a pipe on
   standard input is otherwise ignored.
7. **Human output is deterministic**: no times, no durations, no progress
   indicators, nothing sized to the terminal's width. Help is wrapped at 80
   columns.

### D12 — Colour, the terminal, and the environment the tool reads

**Colour appears only in human output, only on a terminal, never against
`NO_COLOR`; and the tool's reading of `NO_COLOR` is presentation, which never
reaches a template.**

1. `--color auto` (the default) colours a stream only when that stream is a
   terminal (an `isatty` call on the stream, not an environment read) and
   `NO_COLOR` is unset or empty. `TERM` is not read: a terminal that cannot
   show SGR sequences is the caller's to address with `--color never` or
   `NO_COLOR`. Standard output and standard error are decided separately.
   `--color always` colours regardless — a command-line flag overrides
   `NO_COLOR`, as no-color.org asks — and `--color never` never colours.
2. **What may be coloured**: severity labels in findings, the path and line
   prefixes and the matched text in `grep`, and the difference markers in
   `test`. Only SGR sequences (`ESC [ … m`) are used.
3. **What is never coloured, whatever `--color` says**: a JSON document, help
   text, and every byte of `read`, `schema`, `render --out -` and
   `pack --out -`.
4. **The tool decides whether a stream is a terminal** only for colour under
   `auto` and for the refusals of a tar, a packed package (D8 item 4, D9
   `pack`) or a binary entry (D5 `read`) written to a terminal.
5. **The one environment variable the tool reads is `NO_COLOR`.** It never
   reads `TERM`, `HOME`, `XDG_*`, `TMPDIR` (it does not call `os.TempDir` or
   `os.UserHomeDir`), `LANG`, `LC_*`, `COLUMNS` or any `METAIS_*` variable.
6. **This is not host data reaching a template, and the line is exact.**
   R14.4 forbids anything of the host from reaching what a package can see or
   produce. `NO_COLOR` and whether a stream is a terminal are read by the
   tool, about the tool's own display, and:
   - they never enter the data model a template sees (`inputs`, `item`,
     `package`), a manifest value, a path, a default, a finding's text, a JSON
     document, or any byte of an output, a tar or a package;
   - their only effects are whether SGR bytes are inserted into human-mode
     text and whether one of the refusals in item 4 applies;
   - their values are never printed.

   So a package cannot learn them, and the outputs and JSON documents of a run
   are byte-identical whatever they are.
7. **One function owns the read**: `internal/hostenv.Isolate` (ADR-010 D2,
   ADR-009 D14 item 1) reads `NO_COLOR` before the environment is cleared,
   and `internal/hostenv` is the only package that asks whether a stream is
   a terminal. Nothing else in the module calls `os.Getenv`, `os.LookupEnv`
   or `os.Environ`; ADR-009's isolation tripwire enforces that.

### D13 — Exit codes as the command line uses them

**The command line maps every outcome to ADR-006 D12's exit codes, the same
in human and JSON mode.**

| Outcome | Code |
|---|---|
| Success, including with warnings (unless `--fail-on warning`); `grep` with no match; `help`; `version`; `limits` | 0 |
| An error of class `content`: invalid inputs; a failing `check` or `test`; an output refused by its schema or its path; a package's own defect under an authoring command; a warning under `--fail-on warning` | 1 |
| A defect in the tool (`internal.error`); also the Go runtime's status on a crash | 2 |
| The command line is wrong (class `usage`): unknown command or flag, missing operand or required flag, malformed or out-of-range value, conflicting or repeated flags, `metais` with no command, an operand naming no entry, input or output, a glob without a match, a line out of range, a text option on a binary entry, an archive, package or binary entry to a terminal, a bad `--limit` | 3 |
| The package is refused (class `refused`): not found, unreadable, out of specification, digest mismatch, unsupported specification version, a package limit; or a package's own content error under a consumer command | 4 |
| The destination is refused (class `destination`): non-empty without `--overwrite`, a link or special file in the way, a leftover staging name, a failed write | 5 |

When several apply, the first in the order `2`, `3`, `4`, `1`, `5` wins
(ADR-006 D12 item 3).

**Roles** (ADR-006 D12 item 2): `inspect`, `ls`, `read`, `outline`, `grep`,
`schema`, `extract`, `validate` and `render` are consumer commands; `init`,
`check`, `test`, `sample`, `pack`, `unpack` and `render --sample` are
authoring commands; `version`, `help` and `limits` open no package.

### D14 — Every refusal names the next step

**Every JSON document carries `next`: the commands to run next, each a command
line and the reason; every refusal has at least one.**

1. Each item is `{"run": "<command line>", "why": "<one sentence>"}`. Human
   output prints them under `Next:` as `  <run>  — <why>`.
2. `run` is a complete `metais` command line. **No path the caller gave is
   echoed** (D11 item 5): the package operand is the placeholder
   `<package>`, a data directory `<data dir>`, an output or destination
   directory `<dir>`, as ADR-006 D9 item 2 writes `<package>` in a hint.
   Entry paths, input names and example names from the package are written
   as they are, quoted for a POSIX shell (single quotes) when they hold
   anything outside `A–Z a–z 0–9 _ . / : @ % + = , -`. A value the tool
   cannot know is a placeholder in angle brackets (`<pattern>`). **The
   values of `--set`, `--set-json` and `--inputs-json` are never echoed** —
   they may hold what the caller would not want in a log — and appear as
   the placeholder `<your --set, --set-json and --inputs-json flags>`. A
   harness substitutes its own arguments for the placeholders, which are
   the same on every machine.
3. Success documents carry the golden path's next step: `inspect` → D6's
   list; `schema` → `validate`; `validate` (valid) → `render`; `validate`
   (invalid) → `validate` again after correcting; `init` → `check`;
   `check` → `test`; `test` → `pack`; `sample` → `render` or `test`; `unpack` →
   `check`. The reading commands `ls`, `read`, `outline`, `grep` and `extract`
   carry `next` only on a refusal.
4. Each refusal code is declared with its command (D15), with the template of
   its next step.

### D15 — Help is authoritative because it is the declaration

**Each command is declared once, as data beside its code, and the parser, the
dispatcher, the help printer and `help --json` all read that declaration.**

1. A command's declaration holds: name, group, synopsis, description,
   operands, flags (long name, short alias, value placeholder, default,
   repeatable, description), the JSON format it prints, its refusals (code,
   cause, next-step template), and examples. A flag cannot be parsed unless it
   is declared, so it cannot exist without help text.
2. **`metais help`** (and `metais --help`): the one-line description; the two
   golden paths as command sequences — consuming: `inspect` → `read` →
   `schema` → `validate` → `render`; authoring: `init` → edit (previewing with
   `render --sample`) → `check` → `test` → `pack`; the commands by group (D1), one line each; the
   global flags; the topics; and the exit codes.
3. **`metais <command> --help`** and **`metais help <command>`**: usage,
   description, operands, flags with defaults, what it prints (human and JSON
   format name), every refusal with its next step, and examples.
4. **Topics**: `metais help inputs` (supplying inputs and ADR-003's
   precedence), `globs` (D5), `json` (the envelope and every format name),
   `exit-codes` (D13). The limits are listed by the `limits` command, from
   ADR-009's table, not by a topic.
5. **`metais help --json`** prints `metais.help/1`: every command with its
   full declaration, and the topics, for a harness that wants the whole
   surface in one read.
6. Help is printed to standard output and exits 0. A usage error prints the
   command's usage line and `run 'metais <command> --help'` to standard error.
   Help is never coloured and never depends on the terminal.

### D16 — How it is proved

1. **Help coverage.** A test enumerates the dispatch table and asserts every
   command appears in `metais help` and in `help --json`; that every declared
   flag of every command appears in that command's `--help` and in
   `help --json`; and that every example command line in help parses against
   the declarations. Help output has golden files.
2. **Refusals.** Each declared refusal code has a test that triggers it and
   asserts a non-empty `next` whose `run` lines, placeholders substituted,
   parse against the declarations. A test fails if a command emits a refusal
   code its declaration does not list.
3. **JSON shapes.** Every `--json` document produced anywhere in the test
   suite is validated against its committed JSON Schema; a document of a
   format with no schema fails.
4. **Packed and source agree.** For a fixture package and its source
   directory, `ls`, `read`, `outline`, `grep`, `schema` and `inspect` give the
   same results apart from `source` (the digest is equal, ADR-001 D6 item
   3).
5. **Standard output purity.** `render --out -` stdout equals the tar ADR-005
   defines, byte for byte, with `--json`, with warnings present, and on a
   refusal (empty). Pinned red first against a build that prints a notice to
   standard output.
6. **Presentation never reaches the result.** A run under a pseudo-terminal
   with `NO_COLOR` and `TERM` set to sentinel values (`TERM` to prove it is
   not read), and the same run piped with them unset, give byte-identical JSON documents and outputs, and no
   sentinel appears in any output, finding or message. Pinned red first
   against a build that colours JSON.
7. **Argument syntax.** Table tests for interspersed flags, `--`, `=` and
   space forms, short aliases, refused grouping, refused abbreviation,
   refused repetition, and `--json` honoured on an unparseable line.
8. **Semantics.** Table tests for `read --lines` edges (start beyond the end,
   clamped end, CRLF, no final LF, empty entry), `outline` (ATX, setext,
   multi-line setext, `---` after a paragraph, `#` in a fence, frontmatter),
   `grep` (anchors, `-C` merging, binary skip, truncation, long-line cut,
   dot-segment rule) and the glob grammar.

### D17 — The references index, `.compiled/references.json`

**`pack` writes, and every open re-derives (ADR-001 D7, D8 step 7), one JSON
object listing every entry under `references/`, so `inspect` can summarise
references from a JSON parser alone.**

1. **Content**, in ADR-002 D9's canonical form (RFC 8785):

   ```json
   {"references":[{"lines":212,"path":"references/wcag/1.1.1.md","size":5120,"text":true,"title":"Non-text Content"}]}
   ```

   One member, `references`: an array with one object per entry whose path
   begins `references/`, in ascending order of the paths' UTF-8 bytes; `[]`
   when there is none. Each object has exactly:
   - `path` — the entry path;
   - `size` — its uncompressed size in bytes;
   - `text` — `true` when the entry is valid UTF-8 with no NUL byte (D5's
     rule), otherwise `false`;
   - `lines` — for a text entry, its line count by D5's rule; absent for a
     binary entry;
   - `title` — for a text entry whose name ends in `.md` or `.markdown`
     (ASCII case-insensitively), the plain text of its first level-1
     heading as `outline` computes it (D5), frontmatter skipped; otherwise,
     and when there is no level-1 heading, `null`.
2. **It is a function of the entries alone**, fixed for the declared
   specification version, so a second implementation reproduces it byte for
   byte; the `title` rule rests on CommonMark 0.31.2's heading rules, which
   the specification cites.
3. **On a source directory** the same object is computed in memory.

## Interfaces with sibling ADRs

- **ADR-001** — I rely on: the entry path model (relative, `/`-separated,
  confined, case-sensitive); `.compiled/` as ordinary entries of a packed
  package; whole-package verification at every open (ADR-001 D8), which
  every reading command pays before it prints anything, bounded by
  ADR-009's `package-uncompressed`; a packed package recognised by content,
  not extension; the package digest computable for a source directory
  (ADR-001 D6 item 3). I promise: the command line never reads an entry
  around ADR-001's checks; D17 defines `.compiled/references.json`, which
  ADR-001 D7 delegates here.
- **ADR-002** — I rely on: the manifest's `name`, `version`, `title`,
  `description`, specification version and supplier guide (inline
  CommonMark, ADR-002 D4 item 9); the name syntax (for `init --name` and
  `pack`'s default file name); canonical JSON key order (for `schema`,
  `--resolved` and D17). I promise: `version` prints ADR-002 D8 item 1's
  content as `metais.version/1`.
- **ADR-003** — I rely on: the meaning and precedence of `--data`, `--set`,
  `--set-json` and `--inputs-json` (ADR-003 D16 to D18), whose flag names
  this ADR fixes in D7; the data-directory file names per input and form
  (shown by `inspect`); the input order; whether an input as a whole is
  required; each input's title, description, forms, meaning document and
  schema entry; schema combination and per-input bundling (ADR-003 D7);
  the resolved data model (`--resolved`); the frontmatter delimiter
  (`outline`). I promise: the supply flags pass values through unaltered,
  and `--inputs-json -` is the only read of standard input.
- **ADR-004** — I rely on: a representation for the render time; I hand it the
  `--now` string after checking it is an RFC 3339 `date-time` with an
  offset, and nothing when `--now` is absent.
- **ADR-005** — I rely on: staging that completes before the first byte
  reaches standard output; the tar profile; the overwrite rule (reused by
  `extract` and `sample --out`); ADR-005 D16's dry-run records, which are
  `render`'s `result.files`; output identity by rendered path (for
  `--only`, ADR-005 D18); mirror expansion (for `inspect`); file modes
  (reused by `extract`); staging beside the output directory (inside an
  existing one, under ADR-005 Amendment A1), never in the OS temporary
  directory. I own `--only`, which replaces the brief's
  `--output <name>`.
- **ADR-006** — I rely on: the finding object, severities, ordering and human
  wording; D12's exit table, which D13 maps every outcome onto; the
  `--fail-on` semantics of ADR-006 D2 item 5 (D4). I register these codes:
  `usage.unknown_command`, `usage.unknown_flag`, `usage.missing_operand`,
  `usage.bad_value`, `usage.repeated_flag`, `usage.conflicting_flags`,
  `usage.package_from_stdin`, `usage.entry_not_found`,
  `usage.is_a_directory`, `usage.not_text`, `usage.line_out_of_range`,
  `usage.no_match`, `usage.unknown_input`, `usage.unknown_output`,
  `usage.archive_to_terminal`, `usage.binary_to_terminal`,
  `usage.name_required`, `usage.not_a_source_directory` (class `usage`);
  `package.not_found`, `package.not_readable` (class `refused`); and, for
  `extract`, `unpack` and `sample --out`, ADR-005's
  `output.destination.not_empty` (class `destination`). The next step lives
  in the envelope's `next`, not in the finding, so ADR-006 need not add a
  field for it.
- **ADR-007** — I rely on: the scaffold; check, test and its comparison; the
  sample generator (`optional`, `seed`, `items`, the form each input is
  written in, supplied values as fixed points); pack, which replaces its own
  previous file, and unpack, which needs an empty target; **a source
  directory's entry set** (what `pack` includes, ADR-001 D5 and ADR-007 D7
  item 3), which the reading commands use, and which excludes a `*.metais`
  file at its root, since `pack` writes one there by default; `test
  --update` removing stale files inside `examples/<name>/expected/` only.
- **ADR-009** — I rely on: limit names, units, classes and defaults, and the
  rule that a caller may only lower them, behind the single repeatable
  `--limit <name>=<value>` flag and the `limits` command (ADR-009 D4);
  bounds on `--inputs-json -` (`input-size`), on `grep` patterns
  (`pattern-length`, `regex-program`, `regex-work`) and results
  (`grep-matches`, `grep-context`), and on ADR-003's inputs; ADR-009 D15's
  rule that no caller path is echoed, adopted in D11 item 5 and D14 item 2.
  I promise: `NO_COLOR` is the only environment variable the tool reads,
  through `internal/hostenv.Isolate`, and D12's line. I hand over: **the Go
  runtime reads `GODEBUG`, and an environment setting overrides the
  binary's own `go.mod` and `//go:debug` settings** (for example
  `zipinsecurepath`); ADR-009 D14 item 2 and ADR-010 D11 item 4 state why
  no such setting changes what the tool accepts or writes, and ADR-011
  D12's environment A proves it.
- **ADR-010** — I rely on: METAIS's own argument parser (ADR-010 D10 item 3)
  with interspersed flags, `--`, no prefix abbreviation, the closed list of
  short aliases of D3 item 2 with no grouping, detection of a repeated
  single-valued flag, and per-command declarations the help printer reads;
  terminal detection (`golang.org/x/term.IsTerminal` on darwin and linux,
  `syscall.GetConsoleMode` on windows, in `internal/hostenv` only; ADR-010
  Amendment A1, proposed); goldmark v2 as the CommonMark 0.31.2 parser for
  `outline` and D17; Go `regexp` for `grep`; a JSON encoder with HTML escaping off and
  declared member order. The tool's version string is ADR-010's.
- **ADR-011** — I rely on: nothing in the format specification binding a
  command line (see Proposed rulings). The conformance runner's `cli`
  adapter drives `inspect`, `validate`, `render`, `check`, `test`, `pack`,
  `unpack`, `sample` and `version` (ADR-011 D11 item 5); their `--json`
  documents are stable within `/1` and may be relied on.

## Conformance

The command line is the reference tool's contract, not the format's (Proposed
ruling 7): a second implementation may expose the format through another
interface entirely, so **no behaviour this ADR decides is a conformance
vector**. What the vectors observe — findings (ADR-006), outputs and
containers (ADR-005) — reaches the command line unchanged, and this ADR adds
nothing to it.

The behaviours of D5–D14 are proved by the tool's own tests (D16), against
`docs/cli/` and the help text, which are their specification.

## Alternatives rejected

### `--json` for inline input

The brief's input flag shares its name with the output mode every command
has. On `validate` and `render` one of the two would have to be spelled
differently anyway; the output flag is the one on every command, so the input
flag moves.

### `--output <name>` beside `--out`

Two flags a letter-sequence apart, one taking a directory and one an output's
name, invite exactly the mistake a harness makes when it pattern-matches flag
names. `--only` says what it does. Selecting by declaration name was rejected
too: a `for_each` declaration produces many files, while a rendered path names
one, and `--dry-run` lists it.

### A separate `preview` command, or `sample` alone

A `preview` command would duplicate `render`'s flags, staging, dry run and
stream modes. `sample` alone, piped into `render`, would make the golden
preview two commands and lose the forms (CSV, Markdown collections) a
data directory carries and a JSON stream does not.

### JSON Lines streams for `grep` and `ls`

Streaming lets a caller start before the search ends, but the results are
bounded (`--max-results`, ADR-009's entry count), and a stream of records with
no envelope has nowhere to put findings, truncation or `next`. One document
per run keeps one rule for every command.

### Pretty-printed JSON

Readable to a person, but a person has the human output, and a harness pays
for every space. Compact JSON is also a valid JSON Lines record.

### `schema` as the envelope's version key

Collides with JSON Schema's vocabulary, and with the `schema` command whose
result holds a schema. `format` collides with nothing a document carries.

### Reading a package from standard input

Possible by buffering the stream up to the size limit, but no use case needs
it, and a file is one redirection away. It can be added within `/1`.

### POSIX short options only, or options before operands only

Short options cannot name this many flags readably, and requiring flags
before operands breaks the way harnesses compose command lines (D3 item 3).

### Last-wins for repeated flags

Convenient for scripts that append overrides, but it turns a generated
command line's mistake into a silently different run.

### Configuration by environment variable or file

`METAIS_JSON=1` or a `~/.metais` file would make a command line's meaning
depend on where it runs — exactly the host dependence R14.4 forbids for
renders — and would put a host read behind every flag.

### `grep` exiting 1 on no match

Conventional for `grep(1)`, wrong for a query whose caller reads the count:
it makes "nothing found" indistinguishable, by exit code, from a refused
search.

### Errors as a second JSON stream on standard error

Two channels to merge for one run. With `--json`, everything is in the one
document.

## Consequences

- A harness learns the whole surface from `metais help --json`, reads a
  package one bounded step at a time, and recovers from every refusal by
  running the command the refusal names.
- A program branches on `ok` and the exit code, and parses one document per
  run with any JSON parser.
- The JSON formats become a compatibility promise: a member, once shipped in
  `/1`, stays for the life of the v1 tool.
- Adding a command or a flag is one declaration; its help, its JSON listing
  and its parse cannot diverge.
- Interspersed flags and no-abbreviation rule out the standard `flag`
  package as the parser on its own.
- A supplier cannot pass a package on standard input; an author's `pack`
  writes into the working directory by default.

## What this does not decide

- The finding object, codes, severities, wording and exit-code values
  (ADR-006).
- Supply semantics, `--set` path syntax and precedence (ADR-003).
- Limit names, units and defaults (ADR-009).
- The tar profile, staging, overwrite and file modes (ADR-005).
- What `check`, `test`, `pack`, `unpack` and the sample generator do, and the
  scaffold (ADR-007).
- The parser library, the Markdown parser, terminal detection, and the tool's
  version string (ADR-010).
- An MCP mode or any other interface (PRD §3).
- Localised messages: v1's human output is English.

## Proposed rulings

1. **The sample preview is `render --sample`; saving sample data is
   `sample --out <dir> | --example <name>`.** Lean: yes — previewing is
   rendering, saving data is not. Alternative: one `sample` command that also
   renders.
2. **The supply flags are one family: `--set <pointer>=<text>`,
   `--set-json <pointer>=<json>` and `--inputs-json <json>|-`, all
   repeatable (`-` once).** Lean: yes — `--json` is the output mode, and the
   three map one-to-one onto ADR-003's text set, JSON set and whole-input
   object. Alternative: `--json-input`; a non-repeatable inline flag.
3. **One raw output is chosen by `--only <rendered path>`**, not
   `--output <name>`. Lean: yes. Alternative: select by declaration name.
4. **The envelope is `format`/`ok`/`findings`/`result`/`next`, with
   `format: "metais.<command>/<major>"`.** Lean: yes. Alternative: a `schema`
   key, as the brief sketched.
5. **JSON is compact, one line, HTML escaping off.** Lean: yes. Alternative:
   indented.
6. **With `--json`, one document on standard output for every outcome,
   including usage errors; on standard error only when standard output
   carries bytes.** Lean: yes. Alternative: errors always on standard error.
7. **The command line and its JSON formats are the reference tool's
   contract, not part of the format specification**; their schemas live in
   `docs/cli/schemas/`. Lean: yes — a second implementation needs the format,
   not our flags. Alternative: specify the CLI in `spec/`.
8. **A package cannot be read from standard input in v1.** Lean: yes.
9. **Flags may follow operands** (departing from POSIX Guideline 9). Lean: yes.
10. **A repeated single-valued flag is a usage error.** Lean: yes.
    Alternative: last wins.
11. **No configuration file or behaviour-changing environment variable; only
    `NO_COLOR` is read, for colour; terminal detection is an `isatty` call,
    and `TERM` is not read.** Lean: yes. Why: ADR-009 D14 permits one
    display read, and `TERM` adds nothing an `isatty` call and `--color`
    do not. Alternative: also honour `TERM=dumb`.
12. **`grep` exits 0 when nothing matches.** Lean: yes. Alternative: grep(1)'s 1.
13. **Globs and `grep`'s default never enter dot-segments (`.compiled/`)
    unless named.** Lean: yes.
14. **Text is valid UTF-8 with no NUL byte.** Lean: yes. Alternative: a
    byte-sampling heuristic.
15. **`pack` writes `./<name>-<version>.metais` by default and replaces its
    own previous file there, with no `--overwrite`** (ADR-007 D7 item 5).
    Lean: yes. Alternative: `--out` required; refuse an existing file.
16. **`init` takes its default name from the directory operand, never from the
    working directory.** Lean: yes.
17. **Sample defaults are `--optional all` and `--seed 0`.** Lean: yes.
18. **A tar, a packed package or a binary entry is never written to a
    terminal.** Lean: yes.
19. **Every command that writes files takes `--dry-run`.** Lean: yes.
20. **`validate --resolved` prints the data model templates would see.**
    Lean: yes.
21. **`test --update` makes `examples/<name>/expected/` exactly the render,
    deleting stale files there and nowhere else** (ADR-007 D5 item 5).
    Lean: yes. Why: the directory is the author's own; "never deletes"
    governs a caller's directory. Alternative: write and never delete,
    reporting stale files.
22. **Limits are lowered with one repeatable `--limit <name>=<value>` and
    listed by a `limits` command.** Lean: yes; names are ADR-009's.
    Alternative: a `help limits` topic only.
23. **`grep` defaults: `--max-results 100` (max 10000), `-C` at most 100;
    displayed lines cut at 1024 bytes.** Lean: yes.
24. **`extract` refuses when any glob matches nothing.** Lean: yes.
25. **`render --out` is required, even with `--dry-run`.** Lean: yes.
26. **No command name or flag may be abbreviated, and no command has an
    alias.** Lean: yes.
27. **No path the caller gave is ever printed**, in human output, JSON,
    findings or `next`; `next` uses role placeholders (`<package>`,
    `<dir>`) (D11 item 5, D14 item 2; ADR-009 D15). Lean: yes. Why: a
    caller's path can carry a sentinel or a secret, and SC7 forbids either
    in a message. Alternative: echo caller paths exactly as given, never
    made absolute, so `next` lines run as printed.
28. **`--fail-on error|warning` on every command that reports findings**
    (D4; ADR-006 D2 item 5). Lean: yes.
29. **The references index holds `path`, `size`, `text`, `lines` and a
    Markdown `title` per reference** (D17). Lean: yes. Alternative: `path`
    and `size` only, which `inspect` could show without a heading parser
    in the derivation.
30. **`unpack` refuses a non-empty target, with no `--overwrite`** (ADR-007
    D8 item 3). Lean: yes.

## References

- POSIX.1-2024, XBD Chapter 12, Utility Conventions (Guidelines 3, 9, 10,
  13): https://pubs.opengroup.org/onlinepubs/9799919799/basedefs/V1_chap12.html
- Go `flag` package: https://pkg.go.dev/flag
- `spf13/pflag` README (interspersed flags, `--`):
  https://github.com/spf13/pflag
- NO_COLOR: https://no-color.org/
- JSON Lines: https://jsonlines.org/
- Go `encoding/json` (`SetEscapeHTML`, map key order):
  https://pkg.go.dev/encoding/json
- Go `regexp` (RE2 syntax, linear time): https://pkg.go.dev/regexp
- Go `regexp/syntax` (flags, repetition limit of 1000):
  https://pkg.go.dev/regexp/syntax
- RE2 syntax: https://github.com/google/re2/wiki/Syntax
- CommonMark Spec 0.31.2 (§4.2 ATX headings, §4.3 setext headings, §4.5
  fenced code blocks): https://spec.commonmark.org/0.31.2/
- GNU grep manual (context output, group separator, exit status):
  https://www.gnu.org/software/grep/manual/grep.html
- Go `path.Match`: https://pkg.go.dev/path#Match
- RFC 3339 §5.6: https://www.rfc-editor.org/rfc/rfc3339
- `golang.org/x/term` (`IsTerminal`): https://pkg.go.dev/golang.org/x/term
- Go, GODEBUG settings and their precedence: https://go.dev/doc/godebug

## Amendments

### A1 — Text help lists generated refusals by family, and `metais help refusals <command>` lists every code

**Status:** Accepted on merge, recording Ed Funnekotter's ruling of
2026-10-19, given by guidance to PLAN-2026-10-metais-v1 (Wave 5b, Fix 5).
It needs no ratification.
**Amends:** D15 item 3 (what a command's text help lists of its refusals)
and D15 item 4 (the topics), and with them what D16 item 1's coverage test
asserts of refusals. D14, D15 items 1 and 5, and D16 item 2 are unchanged.

1. **A command's declaration still names every code it may refuse with**,
   each with its own cause and next step (D15 item 1). Most of them are
   declared by a loop over the finding registry: every code of a family an
   open, a supply, a render or a check reports, with a cause built from the
   code's own words. Listing each of those in text help made
   `metais render --help` more than a thousand lines, most of them such
   codes, and buried the refusals a caller acts on.
2. **`metais <command> --help` lists each refusal whose cause or next step
   is hand-written on its own**, with its cause and next step, in code
   order.
3. **It lists the refusals the registry loops declare as one line per
   family.** A family is the refusals of one code family (the code's first
   segment) that share one cause and one next step. Its line gives the
   family pattern, which is its codes' longest common prefix of whole
   segments followed by `.*` (or the code itself, for a family of one), the
   number of codes in parentheses, the cause they share, and their next
   step. A family of one code gives that code's own cause. Two families of
   one code family may share a pattern where their causes differ; the count
   and the cause tell them apart, and the topic of item 4 lists their codes.
4. **A topic, `metais help refusals <command>`, lists every code the
   command declares, beneath the next step it names**, each next step in
   the order of its first code. It is one of D15 item 4's topics: the
   overview lists it, `help --json` carries it, and D16 item 1's tests
   reach it. With no command, `metais help refusals` says how to use it and
   lists the refusals of a line that names no command. A command's name is
   the only second operand `metais help` takes: after any other topic it is
   `usage.unexpected_operand`, and after `refusals` an operand that names
   no command is `usage.unknown_topic`.
5. **`metais help --json` keeps the full declaration of every refusal**:
   each code with its own cause and next step, as D15 item 5 says. The
   grouping is a layout of text help only.
6. **D16 item 1's coverage test** asserts that every declared refusal is in
   `metais help refusals <command>` beneath its next step; that each
   hand-written one is in `--help` with its next step; and that the family
   lines of `--help` hold exactly the codes the registry loops declare, each
   matched by its family's pattern. D16 item 2's per-code test reads the
   declaration, which names every code, so it still proves every one.

### A2 — `metais skill` writes metais's agent skill into a project, for the harness that will read it

**Status:** Accepted 2026-10-20 by Ed Funnekotter, with rulings on items 3
and 4, given in answer to the design question of the work order that
drafted it. The text states the rulings.
**Amends:** D1 item 1 (a command in the group "The tool"), D4 (which
commands take `--limit`), D9 (nothing: `skill` is not an authoring
command), D10 item 5 (a format, `metais.skill/1`), D13's roles (a command
that opens no package), D14 item 3 (its next step) and D15 item 4 (no new
topic). ADR-009 A5 decides its host access. No conformance vector changes:
like every command, `skill` is the reference tool's contract, not the
format's (Conformance; Proposed ruling 7), and the specification does not
mention it.

1. **Why.** A coding agent that meets a `.metais` file learns what it is
   from the package's README (ADR-001 A1, A2), and what to run from
   `metais --help`. On 2026-10-20 a fresh Claude Code session asked to
   "fill this file: report.metais", with metais on `PATH`, unzipped the
   package before running the tool, pulled field lists out of
   `metais schema --json` with ad-hoc scripts (two of which printed
   nothing), and hand-wrote a placeholder data directory. Every current
   harness loads an *agent skill* — a directory holding a `SKILL.md` with
   a `name` and a `description` in YAML frontmatter — into its context by
   name and description before the first command, and reads the whole
   file when a task matches. A skill puts the golden paths in front of the
   agent before it improvises. The tool writes its own skill, so the text
   always describes the tool that wrote it.

2. **The command**:

   `metais skill [--harness <name>]... [--dir <dir>] [--overwrite] [--dry-run] [--print] [--json]`

   - It writes one file, `SKILL.md`, to `<dir>/.claude/skills/metais/`,
     `<dir>/.agents/skills/metais/`, or both (item 3), creating the
     missing directories, and lists what it did to each.
   - **`--print`** writes the skill's text to standard output and touches
     nothing, for a harness item 3 does not cover; it takes no other flag
     but `--json`. With `--json` the text is the result's `text` member.
   - **One command, not `skill install` and `skill print`.** D1 has one
     level of command name, and the parser, the help declaration, the
     help goldens and the format names (`metais.<command>/1`) all assume
     it. A second level would be the first, for two verbs that differ
     only in destination, where `render --out -` already shows how this
     tool spells "to standard output" without a verb.
   - It takes no package operand, no supply flag and no `--limit`: it
     reads no package. D13 places it with `version`, `help` and `limits`
     as a command that opens no package. It is in the group "The tool".
   - **Exit codes:** 0 when every target was written or already current;
     3 for a usage error, including no harness chosen (item 3), an
     unexpanded `~` (item 4) and a copy someone edited, which the line
     gives no consent to replace (item 5); 5 (class `destination`) when a
     link, a special file or a failed write stands in the way. Each target
     is decided on its own: one refused target does not stop another's
     write, and the exit code is the worst of them (D13's order).

3. **Choosing the harness**. Five harnesses are named, and
   they read two places (item 6 cites each vendor's documentation):

   | `--harness` | Directory written, under `<dir>` |
   |---|---|
   | `claude` | `.claude/skills/metais/SKILL.md` |
   | `codex`, `gemini`, `cursor`, `opencode` | `.agents/skills/metais/SKILL.md` |

   - **`--harness` is repeatable.** Names that share a directory write it
     once, and the result lists every name the directory serves.
   - **Both directories are written only when the command line asks for
     both**, by `--harness` given once for each. Cursor and OpenCode also
     read `.claude/skills/`, so a project with both copies shows them the
     skill twice under one name; that is the caller's choice to make, never
     the tool's.
   - **Without `--harness`, the tool detects**, from which of `.claude/`
     and `.agents/` exist as directories directly in `<dir>`. When exactly
     one exists, it is the target. Detection is reported, never silent:
     human output says "chosen because `.claude/` exists", and the JSON
     result's `chosen` is `flag` or `detected`, with the target's
     `because`.
   - **When neither exists, or both do, and no `--harness` is given, it
     refuses** (`usage.harness_required`, exit 3) and writes nothing. The
     message says the command needs direction and why (neither directory,
     or both), and names the five values. The `next` lists the
     single-harness form for each value (`metais skill --harness claude`,
     `metais skill --harness codex`, …) and the two-flag form that writes
     both directories, `metais skill --harness claude --harness codex`.
     It never picks a harness the project shows no sign of, and never
     picks between two it does.

4. **Scope**. **The golden path is the project: `<dir>` is
   `.`, the current directory.** `--dir <dir>` names another directory, and
   the user's own scope is `--dir ~`, written with a space, which the shell
   expands: every
   harness's user directory is `~/.claude/skills` or `~/.agents/skills`
   (Gemini and OpenCode read `~/.agents/skills` beside their own). The
   tool never reads `$HOME` or asks the operating system for the home
   directory (ADR-009 D12, D14 item 1; A5 item 3).
   - **`--dir=~` is not expanded.** Bash and zsh expand a `~` only at the
     start of a word, so `--dir=~` and `--dir=~/x` reach the tool with a
     literal `~`. Help, examples and `next` therefore write `--dir ~` alone.
     A `--dir` whose value is `~`, or whose first path element is `~`, is
     refused (`usage.unexpanded_tilde`, exit 3) and nothing is written:
     its hint says the shell did not expand the `~`, which happens when it
     follows `=`, and to write `--dir ~` with a space or the directory's
     path. A directory literally named `~` is written `--dir ./~`.
   - **The tool does not look for the repository's top level.** Claude
     Code, Codex and OpenCode read project skills from the start
     directory and every parent up to the repository root; Gemini CLI's
     workspace is the start directory; Cursor's is the opened folder. So a
     skill at the top level serves every harness, and one in a
     subdirectory serves sessions started there or below. Finding the top
     level means walking up the parents looking for `.git`, a read of the
     host above the directory the caller named, which ADR-009 A5 item 2
     does not allow.
   - Instead, **when `<dir>` holds no `.git` entry, the result carries a
     warning** (`usage.not_repository_root`, severity warning, exit 0):
     the skill serves harnesses started in that directory or below, and
     running `metais skill` at the repository's top level serves the whole
     repository. With `--dir ~` that warning is expected and correct, and
     says so.

5. **Updates and hand edits**. The skill's text is embedded
   in the binary and versioned with it (item 7). Every written `SKILL.md`
   ends with one line the tool owns:

   `<!-- metais skill: version <tool version>, sha256 <hex> -->`

   where `<hex>` is the SHA-256 of every byte before that line. An HTML
   comment is invisible to a Markdown reader and is passed to the model
   as one short line. On each target the tool reads the existing
   `SKILL.md`, if any (at most 64 KiB; ADR-009 A5 item 2), and decides:

   | Existing file | Action, and `action` in JSON |
   |---|---|
   | none | written — `installed` |
   | last line is a well-formed stamp whose digest matches, and its version and text equal this tool's | nothing — `unchanged` |
   | stamp's digest matches, but the version or text differs | replaced — `updated` |
   | no stamp, a malformed stamp, a digest that does not match, or larger than 64 KiB | left untouched — `edited`, finding `usage.skill_edited` (error, exit 3); next: `metais skill --overwrite` |
   | same, with `--overwrite` | replaced — `replaced` |

   - **A file whose digest matches was written by some `metais` and never
     edited**, so replacing it loses nothing; the version in the stamp is
     reported (`from`) so a downgrade is visible.
   - **Anything else is somebody's work**, and the tool does not decide it
     is disposable. `--overwrite` is the consent, per target.
   - `--dry-run` decides every target, writes nothing, and reports the
     action each would take (`would_install`, `would_update`, …). It is
     how a person or a harness asks "is my skill current?".
   - **No staleness warning in `inspect` or `version`.** Either would have
     to read the working directory, which neither does today and ADR-009
     D7 confines them from doing: `inspect` reads a package, `version`
     reads nothing. `metais skill --dry-run` answers the question, and the
     skill's own text tells the agent to run it when `metais version`
     differs from the stamp it was installed with.

6. **The harnesses' discovery paths**, checked on 2026-10-20 against each
   vendor's documentation:
   - **Claude Code** reads `.claude/skills/<name>/SKILL.md` in the start
     directory and each parent up to the repository root, nested
     `.claude/skills/` below it on first use, and `~/.claude/skills/`. It
     does not read `.agents/skills/`.
     <https://code.claude.com/docs/en/skills>
   - **Codex** reads `.agents/skills/<name>/SKILL.md` in every directory
     from the working directory up to the repository root, then
     `$HOME/.agents/skills`, `/etc/codex/skills` and its bundled skills.
     Its own `skill-installer` still writes `~/.codex/skills`
     (openai/skills#420), which the documentation no longer names; METAIS
     follows the documentation. <https://developers.openai.com/codex/skills>
   - **Gemini CLI** reads `.gemini/skills/` and its documented alias
     `.agents/skills/` in the workspace, and `~/.gemini/skills/` and
     `~/.agents/skills/` for the user; within a tier the `.agents` alias
     wins. <https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/skills.md>,
     <https://geminicli.com/docs/cli/using-agent-skills/>
   - **Cursor** reads `.agents/skills/` and `.cursor/skills/` in the
     project, `~/.cursor/skills/` for the user, and for compatibility
     `.claude/skills/`, `.codex/skills/` and their home-directory forms.
     <https://cursor.com/docs/skills>
   - **OpenCode** reads `.opencode/skills/`, `.claude/skills/` and
     `.agents/skills/` from the working directory up to the worktree root,
     and `~/.config/opencode/skills/`, `~/.claude/skills/` and
     `~/.agents/skills/`. Its documentation added the `.agents` paths on
     2026-02-04 (anomalyco/opencode#12067), so an OpenCode older than that
     needs `--harness claude`. Its v2 documentation lists the same project
     and global directories. <https://opencode.ai/docs/skills/>

   Every one of the four `.agents/skills` harnesses reads
   `.agents/skills/<name>/SKILL.md` in the project, so none needs a
   directory of its own. The common frontmatter rules hold the skill to
   `name` (lowercase letters, digits and single hyphens, at most 64
   characters, equal to its directory's name) and `description`; OpenCode
   refuses any other field but `license`, `compatibility` and `metadata`,
   so the skill's frontmatter holds `name` and `description` alone.

   **Re-checked when the command was built** (2026-10-20). The vendors'
   web pages cannot be fetched from the build host, so each path was
   re-checked against the vendor's own source where it publishes one:
   - Gemini CLI against `docs/cli/skills.md` in google-gemini/gemini-cli;
   - OpenCode against `packages/web/src/content/docs/skills.mdx` and the
     v2 `services/www/src/docs/content/skills.mdx` in anomalyco/opencode;
   - Codex against its loader, `codex-rs/ext/skills/src/host_roots.rs` in
     openai/codex, which reads `.agents/skills` in each directory from the
     project root to the working directory and in the home directory
     (the repository's `docs/skills.md` points to the web page);
   - Claude Code by the live check of the built command, recorded in this
     amendment's implementation log. The `.agents/skills` text in its
     binary belongs to its importer, which copies other harnesses' skills
     into `.claude/skills`; it is not a discovery path.

   Cursor publishes no source for its documentation, so its paths stand on
   its web page as found by search, and are not re-checked here. Every
   path re-checked matched; nothing in item 3 changes. A vendor's later
   change is a further amendment.

7. **The skill's text**.
   - **Its source is `internal/skill/SKILL.md`, embedded with `go:embed`**
     by the package that writes it. It is the tool's documentation, not
     the format's, so it is not under `spec/`; and keeping the source
     where it is embedded means there is no second copy to drift.
   - **Its drift test runs against the commands instead.** Every
     `metais <command> …` line in the text must parse under that
     command's help declaration (D15): the command exists, and every flag
     it names is declared. A renamed command or flag fails the test until
     the skill says the new name. A second test holds the frontmatter to
     item 6's rules, and a third holds the text under 16 KiB, so a harness
     that reads every skill whole is not taxed.
   - `metais skill --print` with no other flag prints exactly the embedded
     text and its stamp, and a golden holds that output, like the help
     goldens, regenerated by `regenerate.sh`.
   - **What it says**, written for a model working from nothing:
     - what a `.metais` file is, and that it is read in place with
       `metais`: never unzipped, never read from its raw bytes;
     - the consuming path, command by command: `metais inspect`, then the
       `read`s, `schema`s and example listings its "Read next" names, then
       `validate --data`, then `render --data --out`;
     - how to get the fields: `metais schema <package> <input>` prints a
       JSON Schema to write to a file or read whole; with `--json` the
       schema is the result's `schema` member; never scrape `--json`
       output with ad-hoc text tools;
     - the data directory's file names, from `inspect`'s `files`, and
       `metais help inputs` for `--set` and `--inputs-json`;
     - **never invent facts for a real document**: data comes from the
       person or from sources they name; a value nobody supplied is asked
       for, not guessed. Fictional data only when the person asks for it,
       and then `metais sample --out <dir>` or a shipped example's
       `inputs/`, never hand-made placeholders;
     - the authoring path: `init`, `render --sample`, `check`, `test`,
       `pack`, and `unpack` to edit a packed package;
     - reading `--json`: one document, `ok`, `findings`, `next`; follow
       `next`;
     - where to look when stuck: `metais help`, `metais help <command>`,
       `metais help inputs`, `metais help refusals <command>`;
     - keeping it current: `metais skill --dry-run`.

8. **The JSON result**, `metais.skill/1`, defined by
   `docs/cli/schemas/metais.skill.1.json`:
   `{"skill":"metais","version":"<tool version>","chosen":"flag"|"detected"|null,"targets":[{"path":".claude/skills/metais/SKILL.md","harnesses":["claude"],"because":"flag"|".claude/ exists","action":"installed"|"unchanged"|"updated"|"edited"|"replaced"|"would_install"|"would_update"|"would_replace","from":"<stamp version>"|null}],"text":null}`.
   With `--print`, `chosen` is `null`, `targets` is `[]` and `text` is the
   skill. Paths are relative to `<dir>`, which is never echoed (D11 item
   5); in `next`, `<dir>` is the placeholder `<dir>`.

9. **Next** (D14 item 3): after a write, `metais help` and, for each
   harness written, nothing harness-specific — the skill is picked up by
   the harness on its next session, which the human output says. After
   `edited`: `metais skill --overwrite` and `metais skill --print`, to
   compare. After `usage.harness_required`: item 3's list. After
   `usage.unexpanded_tilde`: `metais skill --dir ~`.

10. **Help and its goldens.** `skill` is declared like every command (D15),
    so `metais skill --help`, `metais help skill`, the overview's group
    "The tool", `help --json`, the help goldens and D16 item 1's coverage
    test carry it. Its refusals are `usage.harness_required`,
    `usage.unexpanded_tilde`, `usage.skill_edited` and the destination
    codes `hostfs` reports (`output.destination.link` and the write
    failures); `usage.not_repository_root` is a warning. The four new codes are registered in
    `spec/findings/registry.json` like every code the tool reports, with
    no statements, as the `usage.*` codes are: chapter 12's generated
    catalog lists them with no statement, and no other text of the
    specification and no vector changes. All four are in the `usage`
    family and of class `usage`, the warning included. ADR-006 D3 item 2
    holds every code to a family the specification lists, and
    `TestFindingCodesCovered` holds every code of class `content`,
    `refused` or `destination` to a vector and a statement, which a
    command no vector runs cannot have; only `usage` and `internal` codes
    name none. The `usage` family covers the command line, and each of
    the four is answered by changing it: naming a harness, writing
    `--dir ~`, giving `--overwrite`, or running at the repository's top
    level.

11. **What this does not decide.** Installing for a harness by its own
    command (`claude plugin`, `gemini skills install`); a skill per
    package, written from a package's guide — that would put package
    content into a host file, which ADR-009 A5 item 4 refuses; and
    writing under `.gemini/skills`, `.cursor/skills` or `.opencode/skills`,
    which each harness reads but none needs.

### A5 — What a session sees first sends a package to `inspect` and `read`, never to `unzip`; the consuming path offers on-the-fly data

**Status:** Accepted on merge. It carries PLAN-2026-10-metais-ux Wave 2's
item W2-2, from the baseline scorecard's candidates 1 and 3, under the
plan's rulings that Ed Funnekotter ratified on 2026-10-21.
**Amends:** D1 item 4 (what `metais` with no command prints), D15 item 2
(the overview and its consuming golden path) and the `usage.missing_command`
hint (ADR-006's registry). D14, D15 items 1 and 3 to 6, and every
command's own help are unchanged.

1. **The overview says how a package is read, under its description.**
   `metais --help`, `metais help` and `metais` with no command each print,
   between the one-line description and the usage line, one paragraph:
   a package is a zip archive that metais reads in place, so it is not
   unzipped; a reader starts with `metais inspect <package>`, then
   `metais read` prints an entry, `metais ls` lists them and `metais grep`
   searches them. It names no flag.
2. **The line that names no command says it too.** `metais` with no
   command is refused with `usage.missing_command` (D1 item 4), and the
   finding's hint, the second line such a session reads, is: "To read a
   package, run `metais inspect <package>`: metais reads it in place, so do
   not unzip it. `metais help` lists every command." The refusal's next
   step stays `metais help`, and the overview still follows the finding on
   standard error.
3. **The consuming golden path offers a supplier data made on the fly.**
   It reads: `inspect` → `read` → `schema` → `validate` → `render`; preview
   from generated data with `render --sample`, or start from data generated
   by `sample`. It is shown once `sample` is declared, like every command
   it names. No route on it goes through a package's shipped examples
   (R-UX-2): an example is the author's illustration of the inputs, not a
   supplier's data.
4. **Why.** In the 2026-10-21 baseline nearly every failing run read the
   raw file, saw a zip whose first entry is `README.md`, and ran `unzip`
   before it ran `metais`. One session ran `metais --help` and still
   unzipped `metais.yaml` next. The overview's golden path began with
   `inspect`, but nothing a session read first said that reading a
   package's contents is `inspect`'s and `read`'s job (candidate 1). And
   the consuming path had no preview: `render --sample` appeared only on
   the authoring path (candidate 3).
5. **What this does not change.** `help --json` carries the consuming path
   as its `paths` member already, and gains no member: the paragraph of
   item 1 is text help's. The specification does not describe the help,
   so no chapter changes with it. The package's README, the first thing
   of all that a reader sees, is ADR-001 A3's.

## Implementation log

This log is this section together with the entry files in
`docs/adr/logs/ADR-008/`. Each work item writes its entry as a file of its
own there, named as AGENTS.md's "Writing documentation" says. The entries
below are the ones written in this file before that rule, and stay where
they were written.

- **PLAN-2026-10-metais-v1 W2-1 — the command-line core.** `internal/cli`
  and `internal/cli/args` implement D1 for `version`, `limits` and `help`,
  D3, D4's `--json`, `--color` and `--help` (`--limit` and `--fail-on`
  arrive with the first commands that take them), D10 to D13, D15, and D16
  items 1, 2 (for the codes declared so far), 3 and 7. The JSON schemas are
  committed under `docs/cli/schemas/`.
  - **The envelope carries `summary`.** Its members are `format`, `ok`,
    `findings`, `summary`, `result`, `next`, in that order, with `summary`
    always present, immediately after `findings`. ADR-006 D10 item 4
    requires every command's envelope to carry `findings` and `summary`;
    D10 item 1's member list omits `summary`. The plan orchestrator's
    ruling for Wave 2 reconciles the two this way, and
    `metais.envelope.1.json` states it. For Ed to ratify at the train's
    close.
  - **Codes registered beside the Interfaces list:** `usage.missing_command`
    (`metais` with no command, D1 item 4), `usage.unknown_topic` (`metais
    help` with an operand that names no command and no topic) and
    `usage.unexpected_operand` (an operand the command does not take, the
    "extra argument" ADR-006 D3's `usage` family covers). An abbreviated
    flag and grouped aliases are `usage.unknown_flag`, its `reason`
    parameter saying which; a flag written last with no value, and a
    boolean given `=value`, are `usage.bad_value`. `internal.error` is
    registered with the panic recovery of ADR-006 D12 item 4. No usage
    finding echoes an argument's text: it names the argument by its
    1-based position, and a flag by its declared name.
  - **A line that does not parse is refused, `--help` included**: help runs
    only on a line every argument of which is accepted, so `metais version
    --help --bogus` exits 3.

- **PLAN-2026-10-metais-v1 W3-1 — `inspect`.** D2 items 1, 4 and 7, D6 and
  D14 landed for `inspect`, with `metais.inspect/1` under
  `docs/cli/schemas/`. It takes `--limit` (ADR-009 D4). `usage.missing_operand`
  and `usage.package_from_stdin` are registered with it; a line with
  `--help` needs no operand. Its declared refusals are every code of the
  families an open reports. The human `Next:` block of `inspect` goes to
  standard output after the summary (D11 item 3). D6 item 8's list names
  only commands the tool declares, so this wave's `inspect` points at no
  command that does not yet exist; the reading, supplying and rendering
  commands join it as they land. D17's references index is
  `loader.Package.References`, derived by the code that re-derives it at
  open. A mirror is listed with the templates it renders; the transforms
  are named `validate`, `crlf` and `zip`.

- **PLAN-2026-10-metais-v1 W4-1 — `validate` and `--fail-on`.** D7 landed,
  with `metais.validate/1` under `docs/cli/schemas/`. Each input's `from`
  lists objects, `{"source", "file"}` for the data directory and
  `{"source", "argument"}` for a flag, `argument` numbering a path set
  among the `--set` and `--set-json` flags together as a finding's
  location does; `supplied` is whether any source but a default gave it a
  value. `--resolved`'s `data` is the inputs object in canonical form, and
  in human form indented JSON whose text is neutralised line by line. The
  findings of `validate` are its result, as `check`'s are: unless they
  refuse it, which under a consumer command an error about the package's
  own content does, they go to standard output with the summary line, the
  human result and the `Next:` block, and the result is printed when they
  hold errors (D11 item 3). `--fail-on` is
  declared once and read in one place, the exit status; `inspect`,
  `validate`, `check` and `init` take it, every command that reports
  findings but `version`, `help` and `limits`, and a check that fails on
  its warnings points at checking again. Two codes are registered, beside the
  Interfaces list, for refusals ADR-003 leaves to the command line:
  `usage.data_dir_unreadable` (the `--data` directory cannot be opened or
  listed, with ADR-006 D9 item 8's reasons and `too_large`) and
  `usage.conflicting_flags` (`--inputs-json -` given twice). The synopsis
  of `validate` writes its supply flags as `[<supply flags>]`, as D7 does,
  since a usage line is held to 80 columns; `--fail-on`, a global flag, is
  in each command's flag list rather than its synopsis.
- **PLAN-2026-10-metais-v1 W4-5 — `unpack`.** D9's `unpack` landed
  without `--dry-run`, which is Wave 5's with the other dry runs. A flag
  is declared required in `internal/cli/args` (`Flag.Required`), and its
  absence is `usage.missing_operand` naming it as the usage line writes
  it, `--to <dir>`, which D13's row reads as "a missing operand or
  required flag"; `help --json` gains each flag's `required`. D9 item 1's
  refusal of a directory is `usage.not_a_packed_package`, beside
  `usage.not_a_source_directory`, registered here. It is made after the
  package opens, so a source directory every opening command refuses is
  refused by unpack with the same findings (ADR-011 D16 item 3), and one
  that opens is refused as a source directory, with nothing written.


- **PLAN-2026-10-metais-v1 W4-2 — the reading commands.** D5 landed whole
  for packed packages and source directories alike, but `extract
  --overwrite`, which Wave 5 carries with existing destinations; it is not
  declared, so no help text offers it. The work is `internal/browse`, over
  a `pkgtree.Tree`, and each command's JSON format is under
  `docs/cli/schemas/`. The readings taken:
  - **Codes registered beside the Interfaces list:** `usage.bad_glob`, a
    glob D5's grammar refuses, and `usage.bad_pattern`, a `grep` pattern
    that is not RE2 syntax or breaches `pattern-length`, `regex-program` or
    `regex-work`. ADR-009 D19 names `usage.bad_value` for a pattern, whose
    `flag` is always present; a pattern and a glob are operands, so each
    has a code of its own, with its argument's position and a `reason`.
    The stricter rule is kept: every such pattern is refused, with class
    `usage`.
  - **`regex-work` in `grep` is `usage.bad_pattern`**, reason `regex_work`,
    with the limit and its bound. `limit.exceeded`'s one subject is an
    input, which a search's work is not, and the caller fixes it by
    changing the pattern or the globs.
  - **`extract` writes only where `<dir>` does not exist** (the plan's
    Wave 4 ruling): an existing directory, empty or not, is
    `output.destination.not_empty`, listing what it holds, and a file in
    the way `output.destination.not_directory`. A missing `--to` is
    `usage.missing_operand` with the operand `--to <dir>`.
  - **D16 item 4 and `.compiled/`.** A packed package's entries include
    `.compiled/` and a source directory's do not (D2 items 2 and 3), so
    the packed-and-source agreement compares `ls` outside `.compiled/`;
    every other command's result is equal byte for byte.
  - **Result shapes D5 leaves open:** `ls` is `{"entries"}`; `outline` is
    `{"path", "headings"}`; a `grep` line of context carries
    `text_truncated` as a match does; `read` of an empty entry reports
    `lines` as first and last 0. `schema <input>` carries `$schema` beside
    the slot's members and `$defs`, and no `$id`.
  - Human notices about a result (a `grep` truncation and its skipped
    entries, an `outline` with no headings) go to standard error (D11 item
    3); `ls`, `outline` and `grep` lines neutralise package text with the
    tab kept, and `read` prints bytes unchanged (ADR-009 D15 item 3).
  - Each reading command takes `--fail-on`, through W4-1's mechanism, as a
    command that reports findings. A usage line longer than 80 columns is
    wrapped in help between its bracketed groups.

- **PLAN-2026-10-metais-v1 W5-9 — `render --dry-run`, `--out -` and
  `--only`.** D8 items 2, 4, 5, 6 and 9 and D11 items 2 and 4 landed for
  template outputs, with D16 item 5's purity test pinned red first against
  `fault.StdoutNotice` (bit `1 << 7`, seam `internal/cli`). The readings
  taken:
  - **Codes registered:** `usage.archive_to_terminal`, whose `argument` is
    the position of `--out`'s `-`, and `usage.unknown_output`, with
    `argument` (`--only`'s value), `paths` (the first 20 paths the render
    writes, in byte order, as many as ADR-006 D6 lists of an enum's
    allowed values) and `path_count`.
  - **When standard output carries bytes is read from the line**:
    `render` with `--out -` and no `--dry-run`. Every document and every
    human result of such a line goes to standard error, a refusal of the
    line itself and an `internal.error` included, so a refused render's
    standard output is empty whatever refused it.
  - **A dry run writes no stream**, so with `--out -` its document, or its
    human listing, is on standard output as any other command's.
  - **`usage.unknown_output` is decided after phases 1 to 4 succeed.** A
    render refused for another reason reports that refusal alone, since its
    plan is not the whole set of paths and a path missing from it proves
    nothing.
  - **With `--only`, `files` holds the selected file alone**, the file the
    run writes, or would; `only` names it.
  - **`status` is omitted, not null, with `--out -`** (ADR-005 D16 item 2:
    "only when `--out` is a directory"); the render schema no longer
    requires it.
  - **A dry run's human listing is headed `would write`**, a line of its
    own, above the lines a render prints; with `--out -` it is the same
    listing.
  - **The pseudo-terminal proof** of the terminal refusal is through
    `isTerminal`'s seam: the dispatch sandbox lets `/dev/ptmx` open but
    refuses the opening of its `/dev/ttys*` side, so no pseudo-terminal is
    offered there (plan §5).
- **PLAN-2026-10-metais-v1 W5-10 — `sample` and `render --sample`.** D8
  item 8 and D9's `sample` landed, with `metais.sample/1` under
  `docs/cli/schemas/`, and the rules only the command line has in
  `docs/cli/sample.md`. `sample` opens a package, so every hostile vector
  is refused through it as through every opening command. The readings
  taken:
  - **Codes registered beside the Interfaces list:** `usage.flag_excluded`
    (two flags that cannot be given together: `--data` with `--sample`,
    `--to` with `--example`, `--overwrite` with `--example`) and
    `usage.flag_requires` (`--optional`, `--seed` or `--items` without
    `--sample`). `usage.conflicting_flags` keeps its one meaning, standard
    input read twice. `usage.not_a_source_directory`, which W4-5's entry
    names but did not register, is registered here for `--example` with a
    packed package. Neither `--to` nor `--example` is
    `usage.missing_operand` naming `(--to <dir> | --example <name>)`.
  - **`--seed`** is decimal digits within 2^64 − 1, and **`--items`**
    digits within 2^31 − 1; the generator's work is held to `input-size`
    whatever the count. A value outside either is `usage.bad_value`.
  - **`--dry-run`** (R2-127) generates, validates and lists, and writes
    nothing; the result carries `dry_run` beside item 5's members, as
    `render`'s does.
  - **`sample --to`** writes a directory that does not exist through
    `hostfs.StageNew`, and an existing one through W5-0's
    `hostfs.OpenExisting` under ADR-005 A1: an empty one directly, one
    that holds any entry only with `--overwrite`, else
    `output.destination.not_empty`. A planned path that `Check` refuses
    (a link, a directory or a special file in the way, a case variant) is
    `output.destination.write_failed` with nothing written; W5-8 owns any
    finer per-file code for render, and `sample` takes it when it lands.
  - **Render's role.** A render with `--sample` reports as an authoring
    command (ADR-006 D12 item 2) through a role the run sets on its
    session, so a defect of the package exits 1, not 4.

- **PLAN-2026-10-metais-v1 W5-V2 — `sample --to` through A1, every
  refusal.** `sample --to` into an existing directory plans the directory
  of an empty collection with its files. The directory is checked with
  them, prepared beneath `<dir>/.metais-staging` and committed by ADR-005
  A1's rename, never made in `<dir>` by any other call. D9 `sample` item
  5's `files` lists it, with a trailing `/`, wherever it is written. A
  directory that holds a file is not listed, because its files are. A
  planned path the checks refuse is reported with every other one, before
  anything is written, under the codes W5-8 registered for `render` and
  W5-12 uses for `extract`: `output.destination.link`, `case_variant`,
  `not_regular_file`, and `not_directory` with its `path`. A staging
  directory left in `<dir>` is `write_failed` with reason `exists`. A
  commit that stops after moving a file is
  `output.destination.commit_failed`. Each is declared in `metais help
  sample`. The human heading agrees in number: `Wrote 1 file:`.

- **PLAN-2026-10-metais-v1 W5-12 — `extract --overwrite` and `unpack
  --dry-run`.** D5's `extract` item 3 landed under ADR-005 D15 and
  Amendment A1, and D9's `unpack --dry-run`. W4-2's interim refusal of an
  existing `--to`, empty or not, is superseded: an existing directory with
  no entry at all is written into, as ADR-005 D15 item 3 says, and one
  that holds any entry, a hidden one included, is written into only with
  `--overwrite`, and otherwise refused with `output.destination.not_empty`
  listing what it holds. The readings taken:
  - **`extract` into an existing directory writes through `hostfs`'s A1
    shape** (`hostfs.Existing`), empty or not: every file is checked
    first, and every refusal is reported together, under the codes W5-8
    registered for `render`'s checks (`output.destination.link`,
    `case_variant`, `not_regular_file`, and `not_directory` with its
    `path`), mapped by the same function; the files are prepared beneath
    `<dir>/.metais-staging` and committed by unit, and a commit that stops
    after moving a file is `output.destination.commit_failed`, listing
    what it moved. A directory
    that does not exist keeps W4-2's write (`hostfs.CreateNew`), whose
    failure lists the files written and removes none.
  - **`extract --dry-run` makes the checks** of the write it stands for,
    a leftover staging directory included, and writes nothing.
    `metais.extract/1`'s result is unchanged: D5 item 6 gives a file
    `path` and `size`, and no `status`.
  - **`extract --overwrite` runs through the existing-destination
    detector** (`internal/guarantees`' `TestExistingDestinationDetector`),
    as a command, in every `go test`.
  - **`unpack --dry-run` is the tool's contract, not chapter 13's.** ADR-011
    D8's `unpack` operation takes no `dry_run`, so no vector can reach it,
    and ADR-011 D3 item 5's `not observable` does not fit a behaviour whose
    evidence is the program's own output. The rule lives in `unpack`'s help
    and in `metais.unpack/1`, and is proved by `internal/cli`'s tests,
    which check the host as well as the result. The dry run makes the
    checks `unpack` makes before its first write, from names and `Lstat`
    only, so it is refused exactly when `unpack` is, with the same
    findings; what only writing finds out, such as a full disk, it cannot
    report.
  - **`metais.unpack/1` gains `dry_run`**, required, as
    `metais.extract/1` and `metais.init/1` carry it. A dry run's next step
    is the `unpack` that writes the files listed.

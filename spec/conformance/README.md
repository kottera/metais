# The METAIS conformance suite

This directory holds the conformance suite of the METAIS specification: the
vectors that prove every observable statement, and this description of how
they are run. The suite is informative about how it is run and normative only
through the statements its vectors cite. Chapter 15 says what a conformance
claim names and when it is satisfied.

## Layout

```
vectors/<area>/<slug>/
  vector.json        the vector's manifest
  package/           a source directory, or
  package.metais     a packed package, as committed bytes
  data/ …            any trees or files the steps' arguments name
  setup/             a tree copied into the work directory before a case runs
  expected/          expected trees, findings, streams and listings
```

`delegated/` holds the delegated runs: the pinned selection of Shopify's
liquid-spec that proves the core Liquid chapter 7 adopts by reference, with
its departures, its adapter and how it is run
([`delegated/README.md`](delegated/README.md)).

`<area>` is the lower-cased area of the vector's first statement, and
`<slug>` is lower-case letters and digits in hyphen-separated runs, at most
64 characters. A vector's `id` is `<area>/<slug>`, its path. Every file of a
vector is named by its `vector.json` or lies in a tree it names, and
`expected/` is never copied where an implementation could read it.

The schemas live beside the specification's other published schemas, under
[`../schemas/conformance/1/`](../schemas/conformance/1/):

| Schema | Describes |
|---|---|
| `vector.schema.json` | `vector.json` |
| `findings.schema.json` | an expected findings file |
| `driver-request.schema.json` | the request a driver reads |
| `driver-response.schema.json` | the response a driver prints |
| `record.schema.json` | the cross-platform record |

The suite's format is versioned apart from the specification: within its
major version it only adds members.

## How a vector runs

A vector runs one or more cases. Each case runs in a fresh sandbox holding
`home/`, `tmp/`, `bin/` (empty) and `work/`; `work/` is the working directory
of every step, and every path a step passes is relative to it. Before the
first step the case's setup is applied: the `copy` tree's contents, the
empty `dirs`, the `symlinks` and the `fifos`, then the package and every tree
or file the steps' arguments name, each at the same relative path, and last
the `unwritable` directories, each made mode 0555 so the implementation can
list it and cannot create, rename or remove anything in it: the pre-state of
a write that fails. After them come the `unreadable` paths, files and
directories the rest of the setup put in place, each made mode 0000 so the
implementation can neither read the file nor list the directory. The runner
gives every unwritable and unreadable path its mode back when the case's
steps are done. Steps of a case share the work directory, in order; cases
never share one.

The setup's `absent` paths are paths that must not exist in the work
directory when the first step runs, such as the operand of a refusal that
nothing is at the path given. A step's argument may name one; the runner
copies nothing for it and checks, before the first step, that it is absent.
Every other path a step reads is in the vector's directory, but an `absent`
path is not: a path listed `absent` that the vector's directory holds, or
that no step names, is a harness error, as is one that exists in the work
directory before the first step.

The setup's `renames` and `file_size_limit` make, when the case runs, what the
committed suite cannot hold. Each of `renames` names a path the rest of the
setup put in place and a new last segment for it, holding a colon or a
backslash, which no path of the committed suite may hold; the runner renames
the path in its own directory before it makes any directory unwritable or any
path unreadable. `file_size_limit` is the largest file, in bytes, that every
process a step starts may write: the runner starts each with that file-size
resource limit (`RLIMIT_FSIZE`) and with `SIGXFSZ` ignored, so a write that
would make a file larger fails with an error and does not end the process. A
rename to a name that holds neither a colon nor a backslash, or that is more
than one segment, is a harness error.

A case that `requires` something the host does not offer (`symlinks`,
`fifos`, `case-sensitive-fs`, `unwritable-dirs`, `unreadable-files`,
`nonportable-names`, `file-size-limit`) is skipped with the reason. A skip is reported, and is never counted as a pass.
`unwritable-dirs` is absent where a directory of mode 0555 still lets the
runner's own process create a file, a directory or a rename in it, and
`unreadable-files` where a file of mode 0000 still lets it open the file for
reading: when it runs as root, or on a platform whose modes do not govern
writing or reading, such as Windows. A vector whose setup names an
`unwritable` directory requires `unwritable-dirs`, and one whose setup names
an `unreadable` path requires `unreadable-files`; a vector that does not is a
harness error. `nonportable-names` is met where the runner can create a file
whose name holds a colon and one whose name holds a backslash, and list each
back under its name, and `file-size-limit` where a process the runner starts
under a file-size limit is refused a write past it; neither is met on
Windows. A vector whose setup has `renames` requires `nonportable-names`, and
one whose setup sets `file_size_limit` requires `file-size-limit`; a vector
that does not is a harness error. `absent` needs no requirement, because
every platform can leave a path out.

Every case runs twice, in two environments, and every byte observed in one,
and every process's exit status, must equal what is observed in the other:

- **Environment A** plants a sentinel token in everything the
  implementation could read from its host: the sandbox's path, `HOME`,
  `TMPDIR`, `PATH`, `USER`, `LOGNAME`, `HOSTNAME`, the XDG directories,
  `NO_COLOR`, `TERM` and `METAIS_SENTINEL`, and decoy files in the working
  directory and the home directory named like the files of the data trees.
  It also sets `TZ=Pacific/Chatham`, `LANG` and `LC_ALL` to `tr_TR.UTF-8`, and
  `GODEBUG=zipinsecurepath=1,tarinsecurepath=1`. Nothing else is inherited.
  Standard output, standard error, every written file, every zip entry
  inflated to a depth of three, every member of a tar stream and a driver's
  response are scanned for the token, case-insensitively; a hit fails the
  step and names where the token was planted. A file a `pack` step writes
  is scanned as a package: its raw bytes and every entry inflated, and one
  that does not begin as a zip fails the step, since the scan cannot see
  inside it.
- **Environment B** has only `HOME`, `TMPDIR` and `PATH` as plain sandbox
  directories, `TZ=UTC` and `LC_ALL=C`, and starts at least two seconds after
  A, in other processes.

## What a step is compared on

- `ok` is compared exactly.
- **Findings**: `expect.findings` names a file holding the expected findings,
  in order; absent, the step gives none. Each expected member equals the
  actual finding's member of the same name, recursively. Only the stable
  members appear: `code`, `severity`, `subject`, `location`, `schema`
  (`entry`, `pointer`, `keyword`), `field`, `params`, `via`, `alternatives`
  and `related`. A message, a hint and a schema keyword location are the
  implementation's own and are never compared.
- **Trees**: each of `expect.trees` holds exactly the regular files of its
  expected tree, byte for byte, and nothing else: no extra file, no link, no
  empty directory. A file listed in `containers` is compared as a zip by its
  entries: names, order, content, the fixed header fields, and the
  compression method of the entries `methods` lists. A file listed in
  `executable` has its owner-execute bit set and every other file's is
  clear. In a file listed in `substitute`, the tokens `@@TOOL_NAME@@` and
  `@@TOOL_VERSION@@` are replaced by the implementation's name and version
  before comparison.
- **Nothing else changes**: the sandbox's `home/`, `tmp/` and `work/` are
  recorded before and after each step, and any file created, changed or
  removed outside a declared tree fails it. A step with no trees writes
  nothing anywhere.
- **Streams**: an archive stream is compared by its tar members (order,
  path, typeflag, mode, owner, group, modification time, size and content); a
  single output byte for byte, or by entries when it is a container.
- **Listings**: `expect.listing` names the file holding a dry run's expected
  listing, alone or as `{"equals": <file>, "containers": [<path>, …]}`. The
  listing is compared exactly, except that only the path of a listed
  container is.
- **Repeatable steps**, whose bytes the implementation chooses, name in their
  trees the files that must exist instead of an expected tree; only a
  repeatable step's trees do. The case runs a second time and every observed
  byte must match.

Every process runs under 10 seconds of CPU time, 1024 MiB of peak resident
memory and 120 seconds of wall clock, which a step's `bounds` can lower. CPU
time and memory are read from the process's own resource usage. For a
race-built program the CPU bound is multiplied by 20 and the memory bound
by 10. A process that exits through a Go panic or a runtime fatal error,
whose standard error holds the Go runtime's traceback, fails its step,
whether the runner started it or a driver did.

## Running the suite

The reference runner is `TestConformance` in `internal/conformance` of this
repository. By default it builds `./cmd/metais` and drives it through its
command line:

```
go test ./internal/conformance
go test ./internal/conformance -run 'TestConformance/out/'
```

Its flags follow `-args`:

| Flag | Default | Meaning |
|---|---|---|
| `-conformance.adapter` | `cli` | `cli` drives a `metais`-compatible command line; `exec` drives a driver |
| `-conformance.tool` | the binary built from source | the command-line tool for `cli` |
| `-conformance.driver` | none | the driver executable for `exec` |
| `-conformance.suite` | this directory | the suite directory |
| `-conformance.version` | the front page's `Version` | the highest revision claimed; later vectors are skipped |
| `-conformance.classes` | `processor,authoring` | which classes to run |
| `-conformance.tags`, `-conformance.skip-tags` | all, none | tag selection |
| `-conformance.record` | none | write the cross-platform record to this file |
| `-conformance.compare` | none | compare records, comma-separated, instead of running vectors |
| `-conformance.keep` | none | keep each failing case's sandbox under this directory |

With `go test -short`, only environment A runs.

`TestHostIsolation` in the same package is the reference tool's own proof
of host isolation (SC7), and goes beyond what this suite asks of an
implementation. It runs every vector against a release build of `metais`
with the cli adapter, in both environments, with more of the host planted
in environment A: further variables, among them `OLDPWD`, `SHELL`,
`SOURCE_DATE_EPOCH`, `CI` and 32 named `METAIS_<n>`, and decoys in the
working, home and temporary directories of every name the vector's packages
refer to and of the files a careless tool might read. It runs every case
twice:

- once with its paths relative to the working directory;
- once with them absolute, under the sandbox's sentinel-named directory,
  from a working directory outside the work directory.

The absolute run then runs `ls --recursive`, `read`, `outline`, `grep`,
`schema`, `extract` and `render` (dry run, and to standard output) over
each package, in human and JSON form. Every stream of both environments is
also searched for the test machine's own host name, user name, temporary
root, home directory and checkout.

## The driver protocol

Another implementation runs the suite through a driver: a program that
performs one operation with its implementation and answers in JSON. Run it
with `-conformance.adapter=exec -conformance.driver=<program>`.

For each step the runner starts the driver once, with no arguments, in the
step's work directory and environment, exactly as a tool would be started,
and writes one request to its standard input
([`driver-request.schema.json`](../schemas/conformance/1/driver-request.schema.json)):

```json
{"protocol": "metais.conformance-driver/1", "op": "render",
 "args": {"package": "package", "data": "data", "stream": "archive"},
 "stream_file": ".stream"}
```

- `op` is one of chapter 15's operations: `open`, `validate`, `render`,
  `check`, `test`, `pack`, `unpack` and `sample`; or `version`, which asks
  for the implementation's identity.
- `args` are the step's arguments exactly as the vector wrote them. Paths
  are relative to the working directory. `inputs_json` names a file holding
  the inputs object.
- `stream_file` is where the driver writes the stream's bytes when the step
  streams (`"stream"` in `args`). It is relative to the working directory and
  is excluded from the check that nothing else changes.

The driver performs the operation and prints one JSON response on standard
output, then exits 0
([`driver-response.schema.json`](../schemas/conformance/1/driver-response.schema.json)):

```json
{"ok": true, "findings": [], "listing": null,
 "tool": {"name": "othertool", "version": "0.3.0"}}
```

- `ok` is true when the operation completed and no finding is of error
  severity.
- `findings` are the finding objects, in the specification's order.
- `listing` is a dry run's listing, each record with the members chapter 11
  gives it (`path`, `size` and `digest` among them), and null otherwise.
- `tool` names the implementation. For `version` the response holds `tool`
  alone; it is what replaces the substitution tokens and what the
  cross-platform record names.

A driver that exits with another status or prints a response that does not
parse against its schema has made a harness error, which the runner reports
apart from conformance failures. The driver's standard output and standard
error are scanned for the sentinel token like a tool's, and every byte it
writes is compared between the two environments, so a driver passes on
nothing from its own host. Members a runner does not know are ignored, so
a later revision of the protocol only adds them.

`TestDriverProtocol` runs the suite through the reference driver, which
wraps the `metais` binary, so the protocol is exercised on every change.

## The cross-platform record

`-conformance.record=<file>` writes, after a passing run, the record of
[`record.schema.json`](../schemas/conformance/1/record.schema.json): the
tool's name and version, the target, the SHA-256 of the suite tree, and for
every invocation of every step of every case in both environments, `ok` and
the SHA-256 and length of standard output, standard error, the stream and the
listing, and every file of every declared tree with its path, size, SHA-256
and execute bit. It also lists the cases skipped for an unmet requirement,
with the reason. No sandbox path appears in it.

`-conformance.compare=a.json,b.json,…` compares records instead of running
vectors. It refuses records of different tools, tool versions or suites,
and otherwise fails on any field that differs between any two, the target
aside. A case one record skipped is compared in neither, since a target
whose file system folds case cannot run what another can. Records taken from one build on different targets agree byte for
byte, which proves what no committed expectation can: a container's deflate
bytes, human messages, and the bytes of repeatable results.

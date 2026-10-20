# 15. Conformance

**Normative.** Areas: CNF.

Conformance to this specification is claimed for a conformance class and
proved by the conformance suite. This chapter specifies what a conformance
claim names and when it is satisfied, the abstract operations the vectors are
written against, and how the suite is laid out and read.

## 15.1 Conformance claims

Chapter 1 defines the three conformance classes (§1.5). A package is not
claimed for: its conformance is established by a conforming authoring tool's
check ([MTS-GEN-001](01-introduction.md#mts-gen-001)). A program's
conformance to the processor or the authoring tool class is claimed, and
the claim is proved by running the suite of vectors (§15.3) against the
program.

<a id="mts-cnf-001"></a>**MTS-CNF-001** *(not observable)* A conformance
claim MUST name the implementation and its version; the classes it claims,
processor, authoring tool or both; the highest revision of this
specification the implementation implements ([chapter 4](04-versions.md));
the revision of the suite it was proved with, a `spec-<MAJOR>.<MINOR>` tag
or a commit of the repository that holds this specification; and every
platform on which the suite was run.

> **Note.** `metais`, the reference tool, claims the processor and authoring
> tool classes. `metais version` names its version and the revisions it
> implements. A release of `metais` runs the suite at the release's commit
> on darwin and linux, on amd64 and arm64, and its claim names the
> platforms whose cross-platform records (§15.5) agree.

<a id="mts-cnf-002"></a>**MTS-CNF-002** *(not observable)* A program that
claims a class MUST pass, on each platform its claim names, every vector of
that class whose `since` is at or below the claimed revision, as §15.4
judges a step. A case skipped because the platform does not offer what its
`requires` names is reported and is not a pass, and it is the only case the
claim leaves unrun.

> **Note.** `metais` is held to its claim on every change: the reference
> runner, `TestConformance`, runs every vector of both classes against it
> on the host, in both environments of §15.5, and the release procedure
> runs it on every platform of the claim.

A vector's `class` is the class whose claim it bears on. A vector of the
processor class uses only the processor's operations, so a program that
claims the processor class alone can run it. A vector of the authoring tool
class (`authoring`) runs at least one authoring operation, and may run a
processor's operation besides it, such as a `validate` of the data a
`sample` wrote: testing and packing a package render it, so an authoring
tool performs a processor's work as well.

## 15.2 The operations

Vectors are written against abstract operations, not against a command line:
the command line of a tool is its own contract, and no part of this
specification. Each operation is defined by the chapter that owns it, and is
performed on the paths and values its arguments name. This table lists every
operation a vector uses, with its arguments and results.

| Operation | Class | Arguments | Results | Defined in |
|---|---|---|---|---|
| `open` | processor | `package`; `limits` | `ok`, findings | [chapter 2](02-container.md) §2.6, [chapter 14](14-security.md) |
| `validate` | processor | `package`; the supply arguments: `data`, `set`, `set_json`, `inputs_json`; `limits` | `ok`, findings | [chapter 5](05-data-specification.md) §5.2, [chapter 6](06-supplying-inputs.md) |
| `render` | processor; authoring tool with `sample` | `package`; the supply arguments; one of `out` and `stream`; `only`, `overwrite`, `dry_run`, `fail_on`, `now`, `sample`, `limits` | `ok`, findings, the tree written at `out` or the bytes streamed, and with `dry_run` the listing | [chapter 11](11-rendering.md), [chapter 13](13-authoring.md) §13.6 |
| `check` | authoring tool | `source`; `limits` | `ok`, findings | [chapter 13](13-authoring.md) §13.1 to §13.4 |
| `test` | authoring tool | `source`; `examples`, `limits` | `ok`, findings | [chapter 13](13-authoring.md) §13.7 |
| `pack` | authoring tool | `source`, `to` | `ok`, findings, the package written | [chapter 13](13-authoring.md) §13.8 |
| `unpack` | authoring tool | `package`, `to` | `ok`, findings, the tree written | [chapter 13](13-authoring.md) §13.5 |
| `sample` | authoring tool | `source`; one of `to` and `example`; `optional`, `seed`, `items`; `set`, `set_json`, `inputs_json` | `ok`, findings, the tree written | [chapter 13](13-authoring.md) §13.6 |

The arguments are these, each as the chapter that defines its operation
defines it:

- `package` and `source` — a packed package or a source directory
  ([chapter 2](02-container.md)). `pack`, and `sample` with `example`, take
  a source directory alone, since they read or write it as the author's
  tree (MTS-AUT-043, MTS-SMP-007).
- `data` — a data directory (§6.9). `set` and `set_json` — lists of path
  sets, each an inputs pointer, `=` and a value, a text set and a JSON set
  (§6.10). `inputs_json` — a file holding one JSON object whose members give
  inputs whole (§6.10). They are applied in the order of §6.11.
- `limits` — an object of limit names ([chapter 14](14-security.md)) to the
  lower values the caller gives for the one operation.
- `out` — the output directory a render writes (§11.1, §11.3). `stream` —
  `"archive"`, the render's files as one archive on its standard output
  (MTS-RND-010), or `{"only": <path>}`, the one selected file's bytes alone
  (MTS-RND-012). `only` — the path of the one top-level file a render into
  `out` writes (MTS-RND-012).
- `overwrite` — the caller's request that existing files be replaced
  (§11.3). `dry_run` — the dry run (MTS-RND-008). `fail_on` — `error`, the
  default, or `warning`, the request that warnings fail the render
  (MTS-RND-007). `now` — the render time, an RFC 3339 date-time
  ([RFC 3339](A-references.md#rfc3339)), which `now` gives templates
  (MTS-TPL-037).
- `sample` — an object of `optional`, `seed` and `items`: the render renders
  from data generated as the sample operation generates it (MTS-SMP-009).
- `examples` — the names of the examples to test; absent, every example
  (MTS-AUT-035).
- `to` — for `pack`, the package file, or an existing directory that
  receives it (MTS-AUT-048); for `unpack` and `sample`, the directory
  written. `example` — the name of the example whose inputs `sample` writes
  (MTS-SMP-007).
- `optional`, `seed` and `items` — the sample generator's options (§13.6):
  `optional` is `all`, `none` or `mixed`, whether the generator writes the
  optional members; `seed` is an unsigned 64-bit integer; `items` is the
  number of items the generator aims at for every array and collection.
  What data they lead to is the implementation's own.

Every path an argument names is relative to the working directory. An
argument a step does not give takes the meaning the owning chapter gives
its absence.

<a id="mts-cnf-003"></a>**MTS-CNF-003** Every operation MUST give `ok` and
its findings, the report of [chapter 12](12-findings.md). `ok` MUST be true
when, and only when, the operation completed and no finding has error
severity; for a render whose `fail_on` is `warning`, also no finding with
warning severity.

<a id="mts-cnf-004"></a>**MTS-CNF-004** *(not observable)* A program MUST
perform each operation of the classes it claims with the arguments this
section lists, each as the chapter that owns the operation defines it, and
MUST give the results this section lists for it: the findings in chapter
12's order; the trees and the package file it writes, at the paths its
arguments name; a render's streamed bytes; and a dry run's listing, of which
the suite compares each record's `path`, `size` and `digest`.

> **Note.** `metais` performs each operation as one command of its command
> line, which the reference runner's `cli` adapter maps, so every vector
> drives it (`TestConformance`). `TestDriverProtocol` drives it through the
> driver protocol (§15.5) as well.

Reading a package in place, which lists, reads, outlines and searches its
entries, is a property of a tool's command line, not an operation: what it
shows, the entries of a package, is proved through `open`, `pack` and
`unpack`.

## 15.3 The suite

The suite lives in [`conformance/`](conformance/README.md), beside this
specification, and is versioned with it: the suite of a revision is the
suite at the revision's tag. Its description,
[`conformance/README.md`](conformance/README.md), says how it is run; this
chapter says what its results mean.

Each vector is a directory `conformance/vectors/<area>/<slug>/`, where
`<area>` is the lower-cased area of the first statement the vector cites and
`<slug>` names the vector. Its identifier is `<area>/<slug>`. It holds:

- `vector.json`, the vector's manifest, which
  [`vector.schema.json`](schemas/conformance/1/vector.schema.json) describes;
- the package, a source directory `package/` or a packed package
  `package.metais` committed as bytes, which is used only where the case is
  about bytes;
- the inputs and every other tree or file the steps' arguments name;
- `setup/`, a tree copied into the work directory before a case runs;
- `expected/`, the expected trees, findings, streams and listings, which is
  never copied where an implementation could read it.

Every file of a vector is named by its `vector.json` or lies in a tree it
names. `vector.json` gives:

- `statements` — the statements the vector proves, each by identifier;
- `since` — the highest `since` among them, the revision from which the
  vector applies;
- `class` — `processor` or `authoring` (§15.1);
- `tags` — from a closed set: `security`, the hostile corpus; `determinism`;
  `portable-paths`; `container`; `byte-level`, a committed packed package;
  `limits`; and `regression`;
- `requires` — what the platform must offer for the vector to run, from a
  closed set: `symlinks`, `fifos`, `case-sensitive-fs`, `unwritable-dirs`,
  `unreadable-files`, `nonportable-names` and `file-size-limit`;
- `cases` — one or more, each a setup and an ordered list of steps; each
  step names an operation of §15.2, its arguments and what it must give.

A case runs in a fresh sandbox, whose work directory is the working
directory of every step. Its setup places, before the first step, what the
committed tree cannot hold: empty directories, links, named pipes,
directories made unwritable and paths made unreadable, paths that must be
absent, names holding a colon or a backslash, and a file-size limit on
every process a step starts. Steps of a case share the work directory, in
order; cases never share one.

The vectors tagged `security` are the hostile corpus: hostile packages and
inputs, each expected to be refused with exact findings and to write
nothing anywhere. A vector whose hostility is in the package has an `open`
step, and every operation that opens a package refuses it with the same
findings.

[`delegated/`](conformance/delegated/README.md) holds the delegated run: a
pinned selection of Shopify's liquid-spec, which proves against a processor's
Liquid engine the core Liquid that [chapter 7](07-template-environment.md)
adopts by reference. Each case of the selection that the dialect does not
pass is a departure, which names the statement that makes the dialect
differ.

The suite's format is versioned apart from this specification, by the major
in its schemas' identifiers: within a major it only adds members, so a
runner reads every suite of that major.

## 15.4 How a step is judged

A step's results are compared with its expectation exactly, but where a
statement grants a freedom:

- `ok` is equal.
- The findings are as many as the expected findings, in the same order, and
  each member of an expected finding equals the actual finding's member of
  the same name. Only the members chapter 12 makes stable are expected:
  `code`, `severity`, `subject`, `location`, `schema` (`entry`, `pointer`,
  `keyword`), `field`, `params`, `via`, `alternatives` and `related`. A
  message, a hint and a schema keyword location are the implementation's
  own, and are never compared. A step that names no findings expects none.
- Each tree the step names holds exactly the regular files of its expected
  tree, byte for byte, and nothing else. A file named as a container is
  compared as a zip, by its entries: their names, order and content, the
  header fields [chapter 2](02-container.md) fixes, and the compression
  method only of the entries whose method this specification fixes. A file
  named as executable has its owner-execute bit set, and every other file's
  is clear. In a file named for substitution, the implementation's name
  and version replace the tokens `@@TOOL_NAME@@` and `@@TOOL_VERSION@@`
  before the comparison.
- Nothing outside the trees the step names is created, changed or removed:
  not in the work directory, the home directory or the temporary directory.
  A step that names no tree writes nothing anywhere.
- An archive stream is compared by its members (order, path, type, mode,
  owner, group, modification time, size and content), and a selected file
  byte for byte, or by entries when it is a container.
- A dry run's listing is compared exactly, but only the path of a container
  named in it.
- Every process the step starts stays within the step's bounds: 10 seconds
  of CPU time, 1024 MiB of peak resident memory and 120 seconds of wall
  clock, unless the step lowers them, read from the process's own resource
  usage. A process that ends in a crash fails the step.
- The step gives the same results in both environments of §15.5, and none
  of its outputs carries the sentinel of environment A.

<a id="mts-cnf-005"></a>**MTS-CNF-005** *(not observable)* A runner MUST
judge a step as passing only when every result it observes equals the
step's expectation by each comparison of this section.

> **Note.** `metais`'s reference runner, `TestConformance`, is proved to make
> each of these comparisons by its self-proof, `TestSelfProof`: for each, a
> fake implementation that breaks exactly that comparison must fail, and one
> that breaks nothing must pass.

Where a statement leaves the bytes to the implementation, as sample data's
are (§13.6), the step is repeatable: its trees name the files that must
exist instead of their expected bytes, the case runs a second time in a
fresh sandbox, and every byte observed in the two runs must match. A later
step of the case proves the property the bytes must have, such as a
`validate` of the data generated.

<a id="mts-cnf-006"></a>**MTS-CNF-006** *(not observable)* A runner MUST
skip a case whose `requires` names something the platform does not offer,
having probed the platform for it, MUST report the skip with its reason, and
MUST NOT count it as a pass; it MUST NOT skip a case for any other reason.

> **Note.** `metais`'s reference runner probes each requirement in the work
> directory's file system, and `TestUnmetRequirementIsSkippedAndReported`
> proves that an unmet one is skipped and reported, not passed.

A vector the runner cannot run as written — a file it names that is missing,
a setup the suite's rules forbid, a driver that answers with a response that
does not parse — is a harness error, reported apart from conformance
failures. A harness error is no pass: a claim is made only once every vector
passes.

## 15.5 Environments, drivers and platforms

Every case runs twice, in two environments, and every byte observed in one,
and every process's exit status, must equal what is observed in the other.
Environment A plants a sentinel token in everything the implementation could
read from its host — the sandbox's path, its environment variables, and
decoy files in the working and home directories named like the inputs —
and sets a time zone, a locale and Go's insecure-path settings that differ
from the plain run; every output is searched for the token. Environment B
runs with a plain environment, at least two seconds later, in other
processes. [`conformance/README.md`](conformance/README.md) lists both
environments exactly. Together they prove [MTS-RND-005](11-rendering.md#mts-rnd-005)
and the host isolation of [chapter 14](14-security.md) across every vector.

The reference runner drives a `metais`-compatible command line by default.
Any other implementation is run through a driver: a program that performs
one operation with its implementation and answers in JSON, by the protocol
[`conformance/README.md`](conformance/README.md) documents with its two
schemas. The driver's operations are those of §15.2, and one more, `version`,
which asks for the implementation's name and version.

A run can write a cross-platform record: for every step of every case in
both environments, `ok` and a digest of every byte observed. Records of one
build taken on different platforms agree byte for byte, which proves what no
committed expectation can: that a container's compressed bytes, the
implementation's messages and the bytes of repeatable steps are the same on
every platform. The reference tool's release compares the records of
darwin and linux, on amd64 and arm64.

# 11. Rendering

**Normative.** Areas: RND, PTH.

A render turns a package and its inputs into files, and gives the same bytes
on every platform. This chapter specifies what makes a render deterministic,
how its outputs are staged and written all at once or not at all, when an
existing file may be overwritten, the dry run, rendering to a stream and the
tar profile that stream follows, and the rule every portable path obeys.

## 11.1 Writing the outputs

A render writes its outputs all at once or not at all. Into a directory that
does not exist yet — the way to render — it writes every output beneath a
staging directory beside it, and then renames the staging directory to the
output directory in one step that replaces nothing.

<a id="mts-rnd-001"></a>**MTS-RND-001** When any finding of a render is an
error, a processor MUST write nothing anywhere: no output, no directory and
no staging directory.

<a id="mts-rnd-007"></a>**MTS-RND-007** When the caller asked that warnings
fail a render, a processor MUST fail a render any finding of which is a
warning, reporting the warning with its severity unchanged, and MUST write
nothing anywhere, as for an error.

<a id="mts-rnd-002"></a>**MTS-RND-002** Rendering into an output directory
that does not exist, a processor MUST create its missing parents, MUST write
every output beneath a staging directory in the same parent, named
`.metais-staging-` followed by the first 16 lower-case hexadecimal digits of
the SHA-256 of the output directory's name, and MUST then rename the staging
directory to the output directory with a rename that refuses an existing
target, so that the output directory never exists without every output in
it and no staging directory is left after the render.

<a id="mts-rnd-003"></a>**MTS-RND-003** A processor MUST refuse a render whose
output directory, or a directory above it, exists and is not a directory with
`output.destination.not_directory`, before it writes anything.

<a id="mts-rnd-004"></a>**MTS-RND-004** A processor MUST NOT render into an
existing directory that holds any entry, a hidden one included, unless the
caller asked for files to be replaced; without that, it MUST refuse the
render with `output.destination.not_empty`, naming up to ten of the entries in
byte order, before it writes anything.

<a id="mts-rnd-006"></a>**MTS-RND-006** When a write of a render fails — into
a directory the processor may not write, on a full disk — the processor MUST
refuse the render with `output.destination.nothing_written`, naming the
reason, and MUST remove what the render created, so that no output directory and no
staging directory is left.

<a id="mts-rnd-005"></a>**MTS-RND-005** A render's outputs MUST be the same
bytes for the same package, inputs and render time on every host, whatever
its time zone, locale, environment, working directory and clock.

## 11.2 The dry run, the archive and one selected file

A render can be asked what it would write without writing it, can write its
files to standard output as one archive instead of into a directory, and can
write one of its files alone. Each of these is the whole render: every file is
planned, rendered and verified first, so none of them can make a refused
render succeed, and none writes anything when the render is refused.

<a id="mts-rnd-008"></a>**MTS-RND-008** A dry run MUST plan, render and verify
every output as the render would, and MUST check the output directory as the
render would when the render writes into one. It MUST then be refused exactly
when the render would be, with the same findings, and MUST write nothing
anywhere: no output, no directory, no staging directory and no temporary
file.

<a id="mts-rnd-009"></a>**MTS-RND-009** A dry run MUST list every top-level
file the render would write, in byte order of its path, one record each:
`path`; `output`, the declaration's id; `item`, the 0-based index of the
`for_each` element it was rendered for, or null; `source` (`template`, `asset`
or `container`); `kind`, for a template file ([chapter 9](09-outputs.md));
`size` in bytes; `digest`, `sha256:` followed by the lower-case hexadecimal
SHA-256 of the exact bytes the render would write; `executable`; and, only
when the render writes into a directory, `status`: `create` for a file written
where nothing is, `replace` for one written over an existing file. A
container's `size` and `digest` MUST be those of the zip the render would
write, and its record MUST also list its `entries`, in the container's order,
each with `path`, `output` (the entry's id, or null), `item`, `source`,
`kind`, `method` (`stored` or `deflated`), and the `size` and `digest` of its
uncompressed content.

<a id="mts-rnd-010"></a>**MTS-RND-010** Rendering to standard output, a
processor MUST write every top-level file of the render as one POSIX pax
interchange format archive ([POSIX](A-references.md#posix)), built from
the rendered bytes and never from files on disk: one regular-file entry (typeflag `0`) per file, in byte order
of its path, a container being one entry that holds the zip a render into a
directory writes, and no directory entry, link or global extended header. Every
entry MUST have the path as its name, as a ustar name, or prefix and name
split at a `/`, when the path is ASCII and fits, and otherwise in a preceding
extended header (typeflag `x`) that carries exactly one record, `path`; mode
`0644`, or `0755` for an executable output; uid and gid `0`; empty uname and
gname; mtime `315532800` (1980-01-01T00:00:00Z); the file's byte count as its
size; devmajor and devminor `0`; and magic `ustar` followed by NUL, version
`00`. The archive MUST end with two 512-octet blocks of zeros and no further
padding.

<a id="mts-rnd-011"></a>**MTS-RND-011** Rendering to standard output, a
processor MUST write nothing there unless every phase of the render
succeeds, so that a refused render's standard output is empty, and MUST NOT
write anything there but the archive or the selected file's bytes.

<a id="mts-rnd-012"></a>**MTS-RND-012** A processor MUST let the caller
select one top-level file by its path, as the dry run lists it; a container's
entry is not selectable, and its container is. The whole render MUST still
run and be verified, so that a selected render is refused exactly when the
full render is. Rendering to standard output, the processor MUST then write
the selected file's bytes alone, as the render would write them; rendering
into a directory, it MUST write that file alone, by the rules of §11.1, and
into an existing directory by those of §11.3.

A selection of a path that names no file of the render, and an archive
written to a terminal, are refused as errors of the invocation before
anything is written; the command line that reports them is the
implementation's.

## 11.3 Writing into an existing directory

An output directory that exists is written into file by file. Each file is
whole at every instant, its old bytes or its new, and the set is whole
against every failure but the processor's own process ending during the
commit. An existing directory with no entry at all is written into this way
with no request to replace files. One that holds any entry is written into
only when the caller asks for files to be replaced (MTS-RND-004). A caller
who needs the render to appear all at once renders into a directory that
does not exist yet (MTS-RND-002).

<a id="mts-rnd-013"></a>**MTS-RND-013** A processor MUST render into an
existing output directory that holds no entry at all as MTS-RND-014 to
MTS-RND-022 state, whether or not the caller asked for files to be
replaced, and MUST NOT refuse it because it exists.

<a id="mts-rnd-014"></a>**MTS-RND-014** Rendering into an existing output
directory, a processor MUST create one staging directory inside it, named
`.metais-staging`, with an exclusive create, MUST write every output beneath
it at the output's own path, and MUST create nothing else in the output
directory until the commit. It MUST then commit each output's unit, in byte
order of the paths, by renaming it from the staging directory into place.
An output's unit is its shallowest path component that did not exist when
the directory was checked: a missing directory, renamed whole with every
output beneath it, or the output itself when its parent exists. A missing
directory and an output whose path is absent MUST be renamed with a rename
that refuses an existing target. The processor MUST then remove the staging
directory. When `.metais-staging` already exists in the output directory —
left by a render that did not finish, or made by a concurrent one — the
processor MUST refuse the render with `output.destination.staging_exists`
before it writes anything.

<a id="mts-rnd-015"></a>**MTS-RND-015** Rendering into an existing output
directory, a processor MUST examine each existing component of every output
path beneath it without following it, and MUST refuse the render with
`output.destination.link`, naming the path, when a component is a symbolic
link, a Windows junction or any other reparse point, even one that points
inside the output directory, whether or not the caller asked for files to be
replaced. It MUST NOT write through, or replace, such a component. The
output directory itself MAY be a symbolic link to a directory, since the
caller named it.

<a id="mts-rnd-016"></a>**MTS-RND-016** Rendering into an existing output
directory, a processor MUST refuse the render with
`output.destination.not_directory`, naming the path, when a component of an
output path that must be a directory exists and is not one, and with
`output.destination.not_regular_file`, naming the path and the reason
`is_directory` or `not_regular_file`, when an output path names a directory,
a named pipe, a socket, a device or anything else that is not a regular
file.

<a id="mts-rnd-017"></a>**MTS-RND-017** Rendering into an existing output
directory, a processor MUST refuse the render with
`output.destination.case_variant`, naming the path, when a directory on an
output path holds an entry that is a canonical caseless match for the
path's component there ([MTS-PTH-006](11-rendering.md#mts-pth-006)) and is not byte-equal to it,
such as `Readme.md` where `README.md` is written: what a case-insensitive
file system does with the two names is not portable.

<a id="mts-rnd-018"></a>**MTS-RND-018** Rendering into an existing output
directory, a processor MUST refuse the render with
`output.destination.nothing_written`, reason `exists`, when an output path's
first segment is an ASCII-case-insensitive match for `.metais-staging`.
Rendering into a directory that does not exist, it MUST NOT refuse such a
path for that reason.

<a id="mts-rnd-019"></a>**MTS-RND-019** *(not observable)* Rendering into an
existing output directory, a processor MUST refuse the render with
`output.destination.nothing_written`, reason `io_error`, when the deepest
existing directory of an output path is on a different file system from the
output directory, such as a mount point inside it, so that every rename of
the commit stays on one file system. It MUST NOT report the device it
compared.

> **Note.** A vector cannot mount a file system without privilege. `metais`
> is proved to meet this by `internal/hostfs`'s unit tests, which put a
> directory on another device by replacing the device the checks read, and
> examine a real mount point where the host has one, and by `internal/cli`'s
> `TestPathRefusal`, which maps that refusal to `io_error` with nothing of
> the device in the finding.

<a id="mts-rnd-020"></a>**MTS-RND-020** Rendering into an existing output
directory, a processor MUST NOT open, read or write an existing file there.
It MUST replace every output whose path is an existing regular file, even
when the bytes would not change, by renaming the staged file over its
directory entry. A file with another hard link therefore keeps that link,
which keeps the old bytes. Every refusal of MTS-RND-014 to MTS-RND-019 MUST
be reported with every other finding of the render, before anything is
written.

<a id="mts-rnd-021"></a>**MTS-RND-021** A processor MUST NOT remove, change
or report a file in the output directory that the render does not write.

<a id="mts-rnd-022"></a>**MTS-RND-022** When a commit into an existing output
directory fails after a unit was renamed into place, the processor MUST
stop, MUST remove the staging directory and everything still in it, and MUST
refuse the render with `output.destination.commit_failed`, naming the
reason and listing, in byte order, every output path whose unit was
committed. A failure before the first unit is renamed changes nothing in the
output directory and MUST be reported with
`output.destination.nothing_written`.

## 11.4 Portable paths

Every path a render writes is held to one predicate, which is also the rule
for a package's entry names ([chapter 2](02-container.md)): a path that
breaks it is refused on every platform, before anything is written, so a
package that renders on one file system renders the same set of files on
every other.

<a id="mts-pth-001"></a>**MTS-PTH-001** An output path MUST be valid UTF-8 with
no byte order mark, and a processor MUST refuse one that is not with
`output.path.encoding`.

<a id="mts-pth-002"></a>**MTS-PTH-002** An output path MUST be in Unicode
Normalization Form C, and a processor MUST refuse one that is not with
`output.path.not_nfc`; a path is never normalized for it.

<a id="mts-pth-003"></a>**MTS-PTH-003** An output path MUST be one or more
segments separated by `/`, with no leading or trailing `/`, no empty segment,
no `.` or `..` segment, no `\` and no `:`, and a processor MUST refuse one
that is not with `output.path.shape`.

<a id="mts-pth-004"></a>**MTS-PTH-004** Every code point of an output path
MUST be U+0020 or belong to one of the general categories L, M, N, P or S as
Unicode 15.0.0 assigns them, and MUST be none of `<`, `>`, `"`, `|`, `?`, `*`
and no code point in U+FFF0 to U+FFFF; a processor MUST refuse a path that
holds another with `output.path.character`.

<a id="mts-pth-005"></a>**MTS-PTH-005** A segment of an output path MUST NOT
be a Windows device name (`con`, `prn`, `aux`, `nul`, `com0` to `com9`,
`lpt0` to `lpt9`, and the same followed by `.` and anything, compared without
regard to ASCII case), a `.git` segment in any spelling Git treats as `.git`,
a segment ending in `.` or U+0020, or a segment longer than 255 bytes, and a
processor MUST refuse a path that has one with `output.path.not_portable`.

A segment is one Git treats as `.git` exactly when an entry name's segment is
([chapter 2](02-container.md), §2.3).

<a id="mts-pth-010"></a>**MTS-PTH-010** A segment is a Windows device name
when the part before its first `.`, with every trailing U+0020 removed, is one
of the names [MTS-PTH-005](#mts-pth-005) lists, or `com` or `lpt` followed by
`¹`, `²` or `³`, compared without regard to ASCII case; a processor MUST
refuse `COM¹.txt` and `Con .md` as it refuses `nul.tar.gz`.

> **Note.** Windows treats the superscript digits as the digits they look
> like when it resolves a device name, and drops the spaces before a name's
> extension, so either spelling opens a device rather than a file.

<a id="mts-pth-006"></a>**MTS-PTH-006** In the set of every output path of a
tree and every directory prefix of each, no two distinct members MUST be
canonical caseless matches under full case folding (Unicode 15.0.0,
CaseFolding.txt statuses C and F). A processor MUST refuse a second output
with an earlier output's path with `output.path.duplicate`, two paths or
prefixes that differ only by case folding with `output.path.case_collision`,
and a path that is also, or caselessly matches, a directory prefix of another
with `output.path.file_directory_conflict`, each naming the earlier
declaration as a related location.

<a id="mts-pth-007"></a>**MTS-PTH-007** An output path MUST be within the
`path-length` and `path-depth` limits ([chapter 14](14-security.md)), and a
processor MUST refuse a path the manifest writes that breaches one with
`limit.package`, naming the limit.

<a id="mts-pth-008"></a>**MTS-PTH-008** A path a path template renders
([chapter 9](09-outputs.md)) MUST be held to MTS-PTH-001 to MTS-PTH-006 as a
path the manifest writes is, on every platform, whatever inputs it was
rendered from: a processor MUST refuse one that climbs out of the output
directory, is absolute, holds a character Windows refuses, names `.git` in
any spelling Git treats as `.git`, or collides by case with another
evaluation's path, before it writes anything, and MUST NOT clean or
rewrite it.

<a id="mts-pth-009"></a>**MTS-PTH-009** A processor MUST refuse a path a
path template renders beyond `path-length` or `path-depth` with
`limit.exceeded.output`, naming the declaration and the limit: the path
is the render's, not the manifest's own text (MTS-PTH-007). It MUST stop rendering such a path once its bytes exceed
`path-length`, and then gives no `observed`.

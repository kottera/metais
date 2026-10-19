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
refuse the render with `output.destination.write_failed`, naming the reason,
and MUST remove what the render created, so that no output directory and no
staging directory is left.

<a id="mts-rnd-005"></a>**MTS-RND-005** A render's outputs MUST be the same
bytes for the same package, inputs and render time on every host, whatever
its time zone, locale, environment, working directory and clock.

## 11.2 Portable paths

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
`limit.exceeded.output`, naming the declaration and the limit, since the
inputs gave it. It MUST stop rendering such a path once its bytes exceed
`path-length`, and then gives no `observed`.

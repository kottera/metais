# 4. Versions

**Normative.** Areas: VER.

A package names the version of this specification it targets, and carries a
version of its own that its author assigns. This chapter specifies both
versions, what compatibility a specification version promises within its
major, and how a processor states the specification versions it implements
and treats a package of a version it does not.

## 4.1 The specification version

A specification version is `MAJOR.MINOR`. It names a set of rules, so it has
no patch part: a revision of this text that changes nothing a program can
observe does not change the set. Revision 1.0 is the first; there is no 0.x.

<a id="mts-ver-001"></a>**MTS-VER-001** The manifest's `metais` member MUST be
present, and a processor MUST refuse a manifest without it with
`package.version.missing`, listing the specification versions it implements.

<a id="mts-ver-002"></a>**MTS-VER-002** `metais` MUST be a string, and a
processor MUST refuse one that is a number, even one whose value looks right
(`metais: 1.0` in YAML is the number 1, and `1.10` would be 1.1), or of any
other type, with `package.version.not_string`, giving the type and, for a
number, the text as written.

<a id="mts-ver-003"></a>**MTS-VER-003** `metais` MUST match
`^(0|[1-9][0-9]{0,3})\.(0|[1-9][0-9]{0,3})$`, and a processor MUST refuse one
that does not, such as `"1"` or `"1.00"`, with `package.version.malformed`.

### 4.1.1 What a processor implements

A processor implements a set of specification versions, and within each major
it implements every minor from `.0` to its highest: one that implements 1.3
implements 1.0, 1.1 and 1.2. It processes a package under exactly the rules of
the version the package declares: that minor's manifest schema, input forms,
keywords, filters, tags, variables, transforms and findings.

<a id="mts-ver-004"></a>**MTS-VER-004** A processor MUST refuse a package that
declares a major it does not implement with
`package.version.unsupported_major`, giving the declared version and every
version it implements, in ascending order.

<a id="mts-ver-005"></a>**MTS-VER-005** A processor MUST refuse a package that
declares a minor newer than the highest it implements in that major with
`package.version.unsupported_minor`, giving the declared version and every
version it implements: it cannot know what a newer minor added, so it cannot
tell a package it renders correctly from one it renders wrongly.

<a id="mts-ver-006"></a>**MTS-VER-006** A processor MUST judge `metais` before
any other rule of the manifest's content and, when it refuses the version,
report no other finding about the manifest: a manifest that names another set
of rules cannot be judged by this one's.

<a id="mts-ver-007"></a>**MTS-VER-007** *(not observable)* A processor MUST
make the specification versions it implements available on request, every one
listed explicitly in ascending order of major and then minor, so a caller tests
membership without knowing which minors imply which.

### 4.1.2 What a minor promises

A minor adds only constructs that every earlier minor of the same major
refuses: a manifest member, an enumeration value, an input form, a filter, a
tag, a variable member, a transform or one of its options, or a warning. A
minor never changes what an accepted construct means, removes one, changes a
default, refuses or renders differently any package and input set an earlier
minor of the same major accepts, gives meaning to a JSON Schema keyword
beginning `x-`, or asserts a `format` an earlier minor accepted without
asserting it. So changing a package's `metais` from `1.m` to any later `1.k`,
and nothing else, changes neither its validity nor any byte it renders, for
any input set; it may add warnings.

Every statement this specification adds after 1.0 carries the minor that
added it, so a processor carries the same table. A package that uses a
construct added after the minor it declares is refused, naming the construct
and the minor that added it; in revision 1.0 there is no such construct.

## 4.2 The package version

<a id="mts-ver-008"></a>**MTS-VER-008** The manifest's `version` member MUST be
a [Semantic Versioning 2.0.0](A-references.md#semver) version of at most 128
characters, pre-release and build metadata allowed, and a processor MUST
refuse one that is not, a leading `v` included, with `manifest.pattern`, and
one that is not a string, such as `version: 1.0` in YAML, with
`manifest.type`.

A package's version is identity and information. A release is identified by
its `name` and `version` together, and its bytes by its package digest
([chapter 2](02-container.md)). No processor compares one package's version
with another's: there is no registry and no dependency between packages, so
nothing needs precedence, and two versions differing only in build metadata
are two releases.

<a id="mts-ver-009"></a>**MTS-VER-009** *(not observable)* An author SHOULD
raise a package's major version when some input set the previous version
accepted is refused or an output path it wrote for an accepted input set is no
longer written, its minor version when every input set the previous version
accepted is still accepted and writes at least the same output paths, and its
patch version when the accepted input sets and output paths are unchanged and
only output content changes.

# 14. Security

**Normative.** Areas: SEC.

A package and its inputs are untrusted, and nothing of the host on which they
are rendered reaches them. This chapter describes the threat model, and
specifies the resource limits and their defaults, the confinement of every
entry path and output path, and the isolation of a render from its host.

## 14.1 Package limits

Every size, count and ratio a processor reads from a package is bounded, and
every bound has a name and a default. A package within every package limit is
portable: every conforming processor accepts it at the defaults. A caller may
lower a limit for one operation; nothing raises one.

| Limit | Default | Bounds |
|---|---|---|
| `package-size` | 256 MiB | the packed package file |
| `package-uncompressed` | 512 MiB | the sum of every entry's uncompressed size, declared and actual |
| `entries` | 10,000 | the entries of a package |
| `entry-size` | 64 MiB | one entry, uncompressed |
| `entry-ratio` | 200 | an entry's uncompressed size divided by its compressed size, for an entry larger than 1 MiB uncompressed |
| `path-length` | 512 bytes | an entry name |
| `path-depth` | 32 segments | an entry name |
| `manifest-size` | 1 MiB | `metais.yaml`, and `.compiled/manifest.json` |
| `schema-size` | 4 MiB | one schema file |
| `combined-schema-size` | 16 MiB | the combined schema, stored or derived |
| `schema-depth` | 64 | the nesting of subschemas |
| `pattern-length` | 4,096 bytes | one `pattern` or `patternProperties` key |
| `regex-program` | 4,096 instructions | one compiled regular expression |
| `template-size` | 8 MiB | one template, and one `path` of the manifest |
| `template-nesting` | 100 | the block tags open at once in one template or `path` |
| `value-depth` | 64 | the nesting of arrays and objects in the manifest and in a schema file |
| `number-length` | 1,024 bytes | one number literal |

KiB, MiB and GiB are powers of 1024.

<a id="mts-sec-001"></a>**MTS-SEC-001** A processor MUST enforce every limit of
§14.1 at its default, or at the lower value a caller gives for one operation,
and MUST refuse a package whose own content breaches one with `limit.package`,
naming the limit and its bound, before it reads the bytes the limit bounds.

## 14.2 Entry confinement

<a id="mts-sec-002"></a>**MTS-SEC-002** A processor MUST refuse every entry
name [chapter 2](02-container.md)'s rules refuse before it reads any entry's
content, so that no name of a package, however it is written, can name
anything outside the package; and it MUST look an entry up by comparing its
name byte for byte with the package's own names, with no case folding,
normalization or resolution against a file system.

<a id="mts-sec-003"></a>**MTS-SEC-003** A processor MUST NOT follow a symbolic
link, or open a named pipe, socket or device, beneath a source directory's
`metais.yaml`, `inputs/`, `templates/`, `references/`, `assets/` or
`examples/`, and MUST read nothing of a source directory outside them.

<a id="mts-sec-004"></a>**MTS-SEC-004** A processor MUST NOT fetch anything a
package names — a schema's `$ref`, `$schema` or `$id`, a manifest's
`homepage` — and MUST resolve a schema reference only within the package and
the 2020-12 meta-schemas.

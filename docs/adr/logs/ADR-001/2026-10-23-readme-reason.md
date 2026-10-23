# ADR-001 — 2026-10-23-readme-reason: the README says why the package is read only with `metais` (Amendment A5)

**2026-10-23, outside a train, under Ed Funnekotter's ruling of the same
day that it lands before v1.0.** Amendment A5 is implemented.
`spec/readme/1.0/README.md` and chapter 2 §2.4.1's block now say that the
package is not unzipped nor opened with another zip tool or library
"because metais checks the package against its digests and its security
limits before it reads any file in it, and a zip tool does not". §2.4.1's
prose gives the same reason, citing §2.6, chapter 14 and MTS-PKG-032. The
text grew from 762 to 897 bytes; its first and last lines are unchanged,
and it still names `metais inspect` and then `metais read`, and no other
command.

- `TestReadmeReadsInPlace` holds the text to the reason clause, and
  `TestFirstSightReadsInPlace` holds the help overview to it. Both were
  red before the change.
- The help overview, the hint of `usage.missing_command` and the skill's
  first rule give the same reason.
- `regenerate.sh` rewrote `internal/specdata`'s copy, the registry and the
  code catalog, the help goldens, every committed package and pack
  output, and every number a vectorgen package's bytes fix: each finding
  offset or size past the README moved by +135 bytes, and
  `sec/package-size-packed` and `sec/package-uncompressed` move their
  lowered limits by the same 135 bytes.

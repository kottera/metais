# ADR-011 — vectorgen writes the byte-level vectors' offsets and sizes (D16 item 5)

**2026-10-20, outside a train (Ed's ruling).** A change to the package
README, or to any bytes ahead of an entry, no longer needs a hand edit of
a vector. `internal/vectorgen` writes every number of a vector's
expectations that its package's bytes fix, and `regenerate.sh` rewrites
them. One test still holds `README.md` to the specification's bytes
(MTS-PKG-071, `TestReadmeMatchesChapter`, `TestSpecDataMatchesSpec`).
Every other vector adapts by regeneration.

**The boundary: the numbers, not the files.** A byte-level vector's
`expected/findings.json` and `vector.json` stay hand-written. Their codes,
locations, severities and the values a case chose (a mode, a method, a
code point) are what a reviewer reads, and the specification's text
decides them. vectorgen owns only named numbers in them, each a JSON
Pointer in `packageValues` (`internal/vectorgen/values_test.go`):

- a finding's `offset`, `declared` and `length`;
- the `observed` of a `limit.package` finding of `package-size`,
  `package-uncompressed`, `entries`, `entry-size` or `entry-ratio`, and
  its `bound` when the case lowered it;
- every one of those limits a step lowers to the package's own figure, or
  one below it;
- a header field's value that is a length of what the case wrote, and an
  end record's entry count.

The files stay plain JSON with literal numbers, as Ed chose over symbolic
offsets resolved at run time, so a third-party runner reads them as
before. Writing whole findings files from Go was rejected: it moves a
vector's meaning out of the vector, into code a reviewer of the suite
does not read.

**Derived from where vectorgen placed the defect.** `archive.write`
records a layout as it writes: each entry's local header, file data and
central directory header, with both headers' fields as written, and the
end record's offset and fields. The cases that edit a zip after writing
it (`pkg/overlap`, `pkg/overlap-quoted`, `pkg/central-order`,
`pkg/zip64-end-record`) do it through the layout and keep it true. No
value is read back from a package, by the reader under test or by
vectorgen.

**The tripwires.** `TestValues` compares each number with its derivation
and, with `-update`, writes it in place, leaving every other byte of the
file as it was. It fails on:

- a stale number;
- a number of the kinds above, in a vectorgen package's vector, that no
  value writes;
- a file holding written numbers that `.gitattributes` does not give the
  `metais-generated-values` merge driver, and a listed file it writes no
  number in.

`TestPackOutputs` builds `pack`'s expected packages (`aut/pack`,
`aut/pack-replace`, `gen/worked-package`, `pkg/pack-round-trip`) from
their source directories, and they take the `metais-generated` driver.

**The merge driver.** `merge-generated.sh values` gives a file git's
three-way merge, then merges a conflict line by line when each side has
as many lines. A line keeps the side that changed it, and a line both
sides changed only in its digits keeps this side's, which regenerating
rewrites. Any other conflict is left for a person.

**Proved.** A one-byte change to the README text, made on a scratch
branch in `spec/readme/1.0/README.md` and chapter 2's block, followed by
`regenerate.sh` alone, gave `go test ./...` green, the conformance suite
included. `regenerate.sh` touched 261 files: 148 vector packages, 103
findings files, 2 `vector.json`, the 6 pack outputs, the container
package of `aut/test-containers-packed`, and the specdata copy.

**Outside vectorgen's reach, and left alone.** Every other number of
this kind in the suite comes from a source directory, not from a
committed package's bytes. `aut/check-limits` and `sec/entries-limit`
count entries, and `sup/encoding`, `man/json-encoding`,
`man/yaml-encoding` and `aut/test-differences` give offsets into data
files. No Go test holds a literal package offset or size.

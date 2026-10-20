# ADR-001 — the README names `metais inspect` (Amendment A2)

**2026-10-20, outside a train (Ed's ruling).** Amendment A2 is implemented.
`spec/readme/1.0/README.md` and chapter 2 §2.4.1's block gain one
paragraph: a reader with metais installed starts with `metais inspect` on
the file. §2.4.1's prose says so too. The text grew from 411 to 540 bytes,
and its first line, which the detection key at offset 39 reads, is
unchanged.

- `regenerate.sh` rewrote `internal/specdata`'s copy and every committed
  package that `internal/vectorgen` writes.
- `pack`'s six expected outputs (`aut/pack`, `aut/pack-replace`,
  `gen/worked-package`, `pkg/pack-round-trip`) were rebuilt by the tool.
  Each differs from before only in `README.md`.
- Every finding that reports a byte offset or size past the README moved
  by +129 bytes: 121 values in 103 expected files, checked value by value
  against the previous tree. `sec/package-size-packed` and
  `sec/package-uncompressed` move their lowered limits by the same 129
  bytes (3179 to 3308, and 6403 to 6532), so the at-bound case is still
  at the bound.

Why: a fresh agent session told only "fill this file: report.metais", with
metais installed, read this README from the raw bytes and then unzipped the
package before it ran the tool.

# ADR-009 — 2026-10-23-capture-charged-as-written: a capture's bytes are charged to `assign-bytes` as they are written

**2026-10-23, outside a train.** This item follows Ed's ruling at
PLAN-2026-10-metais-v1's T-A4, item M-6, that this blocks 1.0. The engine
charged a capture's bytes to `assign-bytes` only when the capture ended,
so the buffers of captures still filling were bounded by `value-size`
each and by nothing together. Thirty captures nested one in another, each
writing 64 MiB before opening the next, reached the A6 item 6 backstop and
stopped with `internal.memory`, which A6 item 6 counts as a defect.

`internal/liquid`'s renderer now charges every write into a capture's
buffer to held bytes as it is written, located at the statement that
writes it, and no longer charges the buffer when the capture ends. The
bytes counted are the same bytes, the ones `capture` stores (MTS-SEC-009),
and D2 item 2 already requires a count checked as it is incremented, so
the specification is unchanged. An `ifchanged` buffer is not charged by
this change.

- `TestCaptureChargedAsWritten` holds the charge: nested captures refused at
  the innermost one's write, a capture filled in a loop refused at the
  write that passes the bound, a partial's write into its caller's capture
  located in the partial, and each byte charged once.
- `sec/assign-bytes` gains the case `nested-captures`, refused at the
  innermost capture's column. Before the change it was refused at the
  outermost capture's.
- The calibration suite gains the case `assign-bytes-capture`, nested
  captures of `value-size` bytes each, enough to pass `assign-bytes`. At
  the defaults on darwin/arm64 it peaked at 1041 MiB before the change,
  past the 1 GiB ceiling, and peaks at 529 MiB with it, refused on
  `assign-bytes`. Thirty such captures, as reported, peaked at 1567 MiB
  and exited through `internal.memory` before the change, and peak at
  593 MiB, refused on `assign-bytes`, with it.

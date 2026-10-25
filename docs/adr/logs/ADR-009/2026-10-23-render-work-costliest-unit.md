# ADR-009 — 2026-10-23: Amendment A13, `render-work`'s default and its costliest units

**2026-10-23, outside a train, under Ed Funnekotter's ruling at
PLAN-2026-10-metais-v1 T-A4 item M-6.** This change proposes Amendment
A13 and builds it: `render-work`'s default is 2²⁸, the costliest units
are calibrated and made to cost about what they are charged, and `sum`
and the comparisons of a string with each element of an array are
charged what they read. A10, which landed first, states the charges of
a range made a value, an array written and `.size`; this change keeps
them as A10 states them.

## What landed

- **The default.** `internal/limits` gives `render-work` 2²⁸; D3's table
  and chapter 14's limit table say so.
- **The charges.** `sum` charges the bytes of its string terms before it
  reads any; `where`, `reject`, `has`, `find` and `find_index` charge a
  sought string's bytes once for each element; an array's `contains` of a
  string charges the string's bytes once for each element (MTS-TPL-064,
  new).
- **The units' costs**, each with a test against what it replaces:
  - `sum` adds each term's digits to a 128-bit bucket of its decimal
    exponent, allocating nothing a term (`TestDecimalSum`,
    `TestDecimalSumAddsWithoutAllocating`);
  - a range element is `value.NumberFromInt` of its integer, with no
    parse (`TestNumberFromInt`);
  - `uniq` keys integers and strings by themselves, in a set for each kind
    (`TestUniqKeyIsIdentity`);
  - `slug` spells a code point with no decomposition without NFKD, through
    `unitext.Decomposes`, and remembers each decomposition's spelling for
    the call, decomposing through a `unitext.Decomposer` that allocates
    nothing (`TestAppendNFKD`, and `TestSlugStepByStep` over every code
    point);
  - `strip_html` scans for its two expressions' leftmost matches and
    remembers where each close is, so a close that is absent is looked for
    once (`TestStripHTMLIsTheExpressions`, against the expressions).
- **`TestSlugStepByStep`'s reference** applies ADR-004 D10 item 6's steps
  in Amendment A1's order, NFKD, Mn removed, then the table. It applied
  the table first, which no case it held could tell apart; the pass over
  every code point does, at `Ǿ`.
- **The calibration suite** gains `render-work-range`, `-write`, `-size`,
  `-split`, `-sum`, `-uniq`, `-slug` and `-strip-html`
  (`internal/conformance/calibrate_units_test.go`). `doubled` builds by
  `capture`, and `assign-bytes` copies by `capture`, since at 2²⁸ the
  `append` that built them ran out of `render-work` before `value-size` or
  `assign-bytes`.
- **Vectors.** `sec/render-work` gains `within-default` and
  `past-default`; `tpl/charge-sum`, `tpl/charge-where` and
  `tpl/charge-contains` are new.

## What was measured

The host is an Apple silicon Mac of 16 cores, macOS 26.6.1 (25G76),
darwin/arm64, running go1.26.9 fetched under `GOTOOLCHAIN=auto`. By its
build and core count it is T-A2's corroborating M4 Max; the sandbox
refused `sysctl`, so the CPU's name was not read. It was shared with other
work throughout, so its figures are a sandbox's, not a calibration (plan
§5).

**The suite at the new default passes.** docs/dev/calibration.md's
command at this change, rebased onto #257 and #258, every case: each
reached its limit, none broke a ceiling, `pass: true`. The record is
beside this entry, in `2026-10-23-render-work-costliest-unit-records/`
(sha256 `76be0b2accc981a8e6939f2581304e00f461a98ac2ac6666fc4703eceacd3da5`).
Its `render-work` cases:

| Case | CPU | Peak resident | × `render-work` |
|---|---|---|---|
| `render-work` | 0.76 s | 19 MiB | 1 |
| `render-work-range` | 12.1 s | 253 MiB | 15.8 |
| `render-work-write` | 2.0 s | 45 MiB | 2.6 |
| `render-work-size` | 0.26 s | 33 MiB | 0.3 |
| `render-work-split` | 2.8 s | 77 MiB | 3.6 |
| `render-work-sum` | 3.5 s | 140 MiB | 4.6 |
| `render-work-uniq` | 15.0 s | 302 MiB | 19.6 |
| `render-work-slug` | 13.1 s | 19 MiB | 17.1 |
| `render-work-strip-html` | 2.4 s | 49 MiB | 3.2 |

Three earlier runs gave the same cases within the host's load, the
multiples 14 to 27: one with `uniq` keyed by one set for every kind, at
`render-work` 1.13 s, `render-work-range` 24.2 s and `render-work-uniq`
30.9 s; and two with `render-work-slug` over every code point NFKD's
tables decompose, which the runner may not compute (ADR-011 D11 item 2,
ADR-010 D11 item 7), at 14.4 s and 12.6 s, 16.6 to 18.6 times
`render-work`, against 17.1 for the named blocks above. `value-size`
peaks at 410 MiB and `assign-bytes` at 412 to 592 MiB, building by
`capture`; the largest CPU figure of the rest is `regex-work`, 39.6 s,
against T-A2's 40.7 s on its M4 Max.

**Before, at 2³²,** with the binary of origin/main `e007f7cf`, the same
harness's `render-work` case took 30.8 s and `render-work-range` 438 s of
CPU (8 m 31 s of wall time) to be refused. `render-work-sum` had not been
refused after 21 minutes of wall time, when the run was stopped, so no
record of it was written.

**Each unit against the cheapest**, the before and after binaries run
back to back on one template each, at `--limit render-work=67108864`,
the cheapest at 268435456:

| Template | Before, ns a unit | After, ns a unit |
|---|---|---|
| 32 empty `if` tags in nested loops | 3.17 | 3.02 |
| `assign r = (1..1000000)` | 63.6 | 44.9 |
| `{{ r }}` of 524,289 strings, all but two empty | 6.9 | 7.2 |
| `split: ","` of 524,288 commas | 10.0 | 10.3 |
| `sum` of `(1..1000000)` | 222 | 13.6 |
| `sum` of 65,536 strings `"0.5"` and `"0.25"` | 432 | 10.6 |
| `uniq` of `(1..1000000)` | 86.6 | 71.2 |
| `slug` of `"ﷺa"` repeated, 4 MiB | 206 | 6.0 |
| `slug` of every code point with a decomposition | 300 | 33.2 |
| `strip_html` of `"<a>"` repeated, 3 MiB | 45.2 | 7.0 |

**Not measured:** darwin/amd64 (this host has no Rosetta 2: an amd64
binary fails with "bad CPU type in executable"), linux/arm64 (no podman
machine) and linux/amd64 (no such host). A13 item 6 gives their
estimates, scaled by T-A2's `render-work` figure for each.

**Two reads were not charged**, found while measuring. 200 `sum`s of a
one-element array holding a 16 MiB decimal string took 12.3 s of CPU and
were accepted at the old default; ten `contains` of an 8 MiB string over
16,384 elements of another 8 MiB string, differing in its last byte, took
43 s and were accepted. Both are refused by `render-work` now.

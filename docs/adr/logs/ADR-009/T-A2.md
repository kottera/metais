# ADR-009 — T-A2: the calibration suite on the four shipped targets, at Amendments A3, A4 and A6's defaults

**2026-10-21, PLAN-2026-10-metais-v1 T-A2.** This is D18 item 4's
calibration, due again before the release because Wave 7 changed the
defaults: A3's lowered memory-driving limits (W7-C5), A4's `input-values`
and `yaml-size` (W7-C6), and A6's bounds and joint render cases (W7-C9).
It was run with docs/dev/calibration.md's command at main `c7532cb`, with
go1.26.9 on every target, on the hosts that ADR-010's log entry T-A2 names.
Each figure below is a case's CPU time and its peak resident memory. The
ceilings are D18 item 2's as A3 sets them: 120 s of CPU and 1 GiB.

**Every case passes on every target.** Each case reached its limit, and no
case broke a ceiling, panicked or recorded a problem or a gap. Every record
has `pass: true`.

| Case | darwin/arm64 | darwin/amd64 (Rosetta 2) | linux/arm64 (VM) | linux/amd64 |
|---|---|---|---|---|
| `render-work` | 18.8 s, 21 MiB | 25.5 s, 23 MiB | 18.1 s, 17 MiB | 60.4 s, 18 MiB |
| `loop-iterations` | 8.4 s, 20 MiB | 10.1 s, 23 MiB | 7.9 s, 17 MiB | 26.7 s, 19 MiB |
| `value-size` | 0.3 s, 273 MiB | 3.8 s, 274 MiB | 0.5 s, 271 MiB | 0.6 s, 272 MiB |
| `array-size` | 0.1 s, 82 MiB | 0.2 s, 83 MiB | 0.2 s, 80 MiB | 0.3 s, 80 MiB |
| `assign-bytes` | 0.2 s, 145 MiB | 1.4 s, 147 MiB | 0.4 s, 142 MiB | 0.5 s, 143 MiB |
| `schema-work` | 0.2 s, 36 MiB | 0.3 s, 38 MiB | 0.2 s, 33 MiB | 0.5 s, 35 MiB |
| `schema-work-anyof` | 5.4 s, 776 MiB | 7.0 s, 805 MiB | 5.5 s, 793 MiB | 10.6 s, 800 MiB |
| `regex-work` | 50.7 s, 39 MiB | 59.5 s, 41 MiB | 50.6 s, 37 MiB | 77.6 s, 37 MiB |
| `outputs` | 8.7 s, 234 MiB | 8.0 s, 270 MiB | 5.0 s, 241 MiB | 6.2 s, 225 MiB |
| `container-depth` | 0.0 s, 17 MiB | 0.0 s, 18 MiB | 0.0 s, 15 MiB | 0.0 s, 16 MiB |
| `output-size` | 0.3 s, 307 MiB | 3.0 s, 308 MiB | 0.5 s, 304 MiB | 0.7 s, 306 MiB |
| `output-total` | 0.5 s, 530 MiB | 4.8 s, 532 MiB | 0.9 s, 527 MiB | 1.3 s, 528 MiB |
| `findings` | 2.1 s, 385 MiB | 3.4 s, 467 MiB | 2.4 s, 379 MiB | 4.0 s, 406 MiB |
| `findings-nested` | 1.9 s, 310 MiB | 2.5 s, 294 MiB | 1.9 s, 313 MiB | 3.8 s, 309 MiB |
| `grep-matches` | 0.1 s, 102 MiB | 0.5 s, 118 MiB | 0.2 s, 109 MiB | 0.3 s, 100 MiB |
| `grep-context` | 0.4 s, 713 MiB | 3.0 s, 726 MiB | 0.6 s, 722 MiB | 1.1 s, 739 MiB |
| `schema-joint-file` | 7.3 s, 92 MiB | 11.4 s, 94 MiB | 7.0 s, 90 MiB | 7.4 s, 90 MiB |
| `schema-joint-combined` | 14.2 s, 146 MiB | 22.6 s, 154 MiB | 13.6 s, 146 MiB | 17.7 s, 152 MiB |
| `yaml-size` | 0.5 s, 242 MiB | 0.7 s, 209 MiB | 0.5 s, 283 MiB | 1.2 s, 290 MiB |
| `input-values-json` | 1.5 s, 516 MiB | 3.0 s, 524 MiB | 1.7 s, 522 MiB | 3.2 s, 519 MiB |
| `input-values-csv` | 1.6 s, 536 MiB | 3.5 s, 542 MiB | 1.7 s, 527 MiB | 3.0 s, 520 MiB |
| `input-values-yaml` | 1.5 s, 491 MiB | 2.8 s, 478 MiB | 1.5 s, 520 MiB | 3.1 s, 458 MiB |
| `render-joint-json` | 2.8 s, 999 MiB | 7.6 s, 1,001 MiB | 3.2 s, 992 MiB | 6.0 s, 993 MiB |
| `render-joint-yaml` | 2.8 s, 984 MiB | 7.3 s, 981 MiB | 3.2 s, 978 MiB | 6.3 s, 977 MiB |
| `schema-yaml-size` | 0.6 s, 301 MiB | 2.2 s, 357 MiB | 0.7 s, 324 MiB | 1.7 s, 306 MiB |
| `digest-values` | 0.2 s, 144 MiB | 1.2 s, 144 MiB | 0.3 s, 142 MiB | 0.4 s, 143 MiB |
| `example-json` | 0.2 s, 150 MiB | 0.7 s, 150 MiB | 0.2 s, 119 MiB | 0.7 s, 149 MiB |
| `reference-title` | 0.4 s, 144 MiB | 1.5 s, 147 MiB | 0.6 s, 143 MiB | 0.9 s, 144 MiB |
| `manifest-schema-work` | 0.4 s, 197 MiB | 0.9 s, 211 MiB | 0.4 s, 205 MiB | 1.0 s, 200 MiB |

A corroborating darwin/arm64 run on a second host, an M4 Max, also passes.
Its largest figures are `regex-work` at 40.7 s and `render-joint-json` at
999 MiB.

**The joint render cases run at the soft memory limit.** `render-joint-json`
peaks at 97 to 98% of 1 GiB on every target, and `render-joint-yaml` at 95
to 96%. That is A6 item 6 at work: the backstop sets the runtime's soft
memory limit to 1 GiB (W7-C10), so the collector holds these cases' heap
just under it. Their headroom is the collector's, not the case's: a joint
render that needed more would cost CPU first, then cross the ceiling.
Ed ruled on 2026-10-21 that a case held at the soft limit is a pass.

**The worst CPU case is `regex-work`**, at 77.6 s on linux/amd64, 65% of
the ceiling. On darwin it is at most 59.5 s, under Rosetta 2.

**The hosts were not all idle.** Only linux/amd64 ran on an idle host. The
darwin runs, and the podman machine that runs linux/arm64, shared their
host with other sessions' builds and tests. ADR-010's entry T-A2 records the
load. Ed ruled on 2026-10-21 that these runs stand as the calibration. Peak
resident memory is the process's own, not the host's. The two darwin/arm64
hosts, at loads of about 6 and about 20, agree within 0.3% on the joint
render cases and within 6% on every other case above 400 MiB, inside the
run-to-run variation W7-C6 recorded. Every CPU figure stays within the
ceiling with a margin no load seen here could close.

The calibration records are committed beside this entry, in
`T-A2-records/` (sha256):
- darwin/arm64 `4d533432f14b407de74b505c85de3f6cbe8bb42a24b058e2e07d82a90002dca6`
- darwin/amd64 `3f37e0b791661e7c224870a73e1f997f786068802944df7de6c67e778e53b15a`
- linux/arm64 `20262c55aff963c0fea8d8b7c567bd1175f910205e9696bfaf9de98433d9165e`
- linux/amd64 `a600fd3820fab53ff0aefcdf4e87901698fb02a9f213c0ed1a019c2c7564bebe`
- darwin/arm64 on the M4 Max `146a7b3a5e42bb1e66c472cd55ce45d998c72b2544b2341958aa8998ec93860a`

No default changes, so no amendment is due.

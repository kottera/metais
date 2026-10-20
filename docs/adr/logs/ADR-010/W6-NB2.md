# ADR-010 — W6-NB2: the benchmark baselines pinned, and the benchmark command's toolchain

**2026-10-20, PLAN-2026-10-metais-v1 W6-NB2.** `internal/guarantees`'
`TestBenchBaselines` holds every package that declares a `Benchmark…`
function to D15 item 1's committed baseline: `testdata/bench/baseline.txt`
exists, carries the line `# sandbox figure, not a budget`, and has a row
of each benchmark function the package declares. It walks the directories
`go test ./...` does. `TestBenchBaselinesControl` plants each omission. A
row is matched by its function's name, so a sub-benchmark's own row is not
required: sub-benchmark names exist only at run time.

`docs/dev/benchmarks.md`'s command sets `GOTOOLCHAIN=auto` itself, as
`docs/dev/calibration.md`'s does, because a host may set
`GOTOOLCHAIN=local` and plan §6's act reads that page as its runbook.

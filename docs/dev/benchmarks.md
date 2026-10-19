# The benchmarks

ADR-010 D15 item 1 names the benchmarks, and item 2 the budgets they are
measured against. This page is how they are run, how a run is compared with
the committed baselines, and which benchmark measures which budget. Plan
§6's act that runs the benchmarks on each supported target follows it.

## Running every benchmark

From the module's root, on an otherwise idle host:

```bash
go test -run '^$' -bench . -benchmem -count 6 -p 1 -timeout 60m ./... > bench.txt
```

That is the one command. It runs every benchmark of the module, six times
each, one package at a time (`-p 1`, so that no two packages' benchmarks
share the cores), and writes `bench.txt` in benchstat's input format. It
runs under the `go.mod` toolchain, fetched under the default
`GOTOOLCHAIN=auto`. Record the host, its CPU, `go version`, GOOS and GOARCH
beside the file.

## Comparing with the baselines

Each package with benchmarks commits `testdata/bench/baseline.txt` beside
them: the raw output of
`go test -run '^$' -bench . -benchmem -count 6` for that package. Its
leading comment lines name the host, the CPU, `go version`, GOOS and
GOARCH, and say `sandbox figure, not a budget`: a baseline is what the
benchmark gave where its work item landed, for benchstat to compare
against, and never a measurement of a budget.

[benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) compares a
run with every baseline at once:

```bash
cat $(git ls-files '*/testdata/bench/baseline.txt') > baseline.txt
benchstat baseline.txt bench.txt
```

benchstat is the operator's tool, not a dependency of the module. Each
baseline's `pkg:` line keeps its package's rows apart.

## The benchmarks

| ADR-010 D15 item 1 | Benchmark |
|---|---|
| Liquid parse and render over the three fixtures in `internal/liquid/testdata/bench/` | `internal/liquid`: `BenchmarkParse/{table,site,nested}`, `BenchmarkRender/{table,site,nested}` |
| JSON parsing of a 10 MiB input | `internal/jsonx`: `BenchmarkParse` |
| YAML parsing of a 10 MiB input | `internal/yamlx`: `BenchmarkParse` |
| CSV parsing of a 10 MiB input | `internal/inputs/csv`: `BenchmarkRead` |
| Schema validation of a 10,000-item collection | `internal/schema`: `BenchmarkValidate` |
| Zip reading of a package of 10,000 entries | `internal/zipfmt`: `BenchmarkOpen` |
| `pack` of the §4.1 example | not yet: W6-2 adds `pack`'s benchmark, and Wave 7 the §4.1 example |

## The budgets

D15 item 2's budgets are CPU time, on an Apple M-series or comparable 2024
x86-64 core. Each is measured by one benchmark, which reports `cpu-ns/op`,
the process's CPU time for each iteration, beside `ns/op`:

| Budget | Benchmark |
|---|---|
| Rendering the 10,000-row table, at most 200 ms | `internal/liquid`: `BenchmarkRender/table` |
| `inspect` of the 10,000-entry package, whole-package verification included, at most 500 ms | `internal/cli`: `BenchmarkInspect` |
| Validating the 10,000-item collection, at most 500 ms | `internal/cli`: `BenchmarkValidate` |

`BenchmarkInspect` and `BenchmarkValidate` run the command in-process, as
the binary runs it: `inspect --json` of a packed package of exactly 10,000
entries, and `validate --json` of a package whose collection is read from
a data directory of 10,000 files. A budget is recorded in ADR-010's
implementation log from a run on a real host by §6's act, never from a
sandbox. A result that breaks a budget is escalated, as RULINGS R1-015
says, and changes no default on its own.

D15 item 2's two deterministic budgets are tests, in every `go test`:

- **Opening a packed package reads every byte once and decodes every entry
  once**, by a counting `ReaderAt`: `TestOpenReadsEveryByteOnce` in
  `internal/zipfmt` (the reader) and in `internal/loader` (the whole
  open), each with its control.
- **`read --lines a-b` decompresses its entry at most once more after
  verification**: `TestReadLinesDecompressesOnceMore` in `internal/cli`.

## What keeps them honest

Each benchmark's fixture is held to the work it is there to time by a test
in every `go test`: `TestBenchFixtures` (`internal/liquid`),
`TestBenchInput` (`internal/jsonx`, `internal/yamlx`,
`internal/inputs/csv`), `TestBenchCollection` (`internal/schema`),
`TestBenchPackage` (`internal/zipfmt`) and `TestBenchCommands`
(`internal/cli`). A benchmark whose run is refused fails rather than
timing the refusal.

# The limit-calibration suite

ADR-009 D3 item 3 and D18 item 4 calibrate the defaults of the work and
output limits. Amendment A1 item 5 adds the schema compile bounds' joint
worst case. This page is how the suite is run, what it records and what is
reported back. Plan §6's act that runs it on each supported target follows
it.

## Running it

From the module's root, on an otherwise idle host:

```bash
GOTOOLCHAIN=auto go test -count=1 -timeout 3h -v -run '^TestCalibration$' ./internal/conformance \
  -args -calibration.record="$PWD/calibration-$(go env GOOS)-$(go env GOARCH).json"
```

That is the one command. It runs under the `go.mod` toolchain, which
`GOTOOLCHAIN=auto` fetches, and the command sets it because a host may
set `GOTOOLCHAIN=local`. The record's `go` names the toolchain that ran,
for the reader to check against `go.mod`. It never runs with `-race`: the suite
measures the binary as it is released, and refuses a race-built one. It
builds `./cmd/metais` with `-trimpath`, runs each case once, one at a time,
and writes the record whatever the outcome. Then it fails, naming each
case and the ceiling it broke. A run takes some minutes, and its cases
write up to `output-total` to the temporary directory, which each case
removes when it ends.

`-calibration.cases=<name>,<name>` runs only the cases named, which is how
one case is measured again. `-conformance.tool=<path>` measures another
build in place of the one the test builds.

Without `-calibration.record`, `TestCalibration` is skipped, and every
`go test` runs `TestCalibrationLowered` instead. That test runs every case
at lowered limits, as the same case scaled down, and proves that it reaches
its limit there. A case that stops reaching its limit fails it, so the
suite cannot rot between calibrations.

## What is measured

Each case runs as a process of the built binary, in an environment of its
own. Its CPU time (user plus system) and its peak resident memory are read
from the process state, by `getrusage`, as the conformance runner reads a
step's (ADR-011 D13). On linux the process is started through the runner's
stub, so its peak is its own and not the test process's. CPU time is never
replaced by wall time. The wall time is recorded beside it, and a 30-minute
wall clock ends a case that would otherwise never end. A case it ends breaks
the CPU ceiling.

The ceilings are ADR-009 D18 item 2's, its CPU ceiling as Amendment A3
item 1 sets it:

| Ceiling | Bound |
|---|---|
| CPU time of a case refused from metadata | 0.5 s |
| CPU time of every other case | 120 s |
| Peak resident memory | 1 GiB |

No case is refused from metadata: each one does the work its limit allows
before it ends. A case also fails the suite when it exits through a panic
or a runtime fatal error, or when it does not reach its limit at the
default, because it would then not be that limit's worst case.

## The cases

Each case is either refused by its limit, after doing all the work the
limit allows, or accepted exactly at its limit, where that is more work
than the refusal. For a case accepted at its limit, `TestCalibrationLowered`
also proves that one less refuses it.

| Case | Command | The worst case | Reaches its limit by |
|---|---|---|---|
| `render-work` | `render` | 32 empty `if` tags in nested loops: the cheapest node, as often as the units allow | refusal |
| `loop-iterations` | `render` | nested empty loops, one range of 1,000 past the limit | refusal |
| `value-size` | `render` | a string doubled by `append` | refusal |
| `array-size` | `render` | `split` of a doubled string of `x,` | refusal |
| `assign-bytes` | `render` | a string of `value-size` bytes, copied by `assign` | refusal |
| `schema-work` | `validate` | an array input, each item evaluating `items` and an `allOf` of nine, exactly the limit's units | accepted at the limit |
| `regex-work` | `grep` | a pattern of about 4,000 instructions over lines of 999 bytes it never matches | refusal |
| `outputs` | `render` | a `for_each` of exactly `outputs` outputs, each written | accepted at the limit |
| `container-depth` | `render` | containers nested `container-depth` deep | accepted at the limit |
| `output-size` | `render` | one output of exactly `output-size` bytes, written | accepted at the limit |
| `output-total` | `render` | outputs of `output-size` bytes, exactly `output-total` together, written | accepted at the limit |
| `findings` | `validate` | an array input with one more failing item than `findings` lists | the result's listed count at the limit, with findings omitted |
| `findings-nested` | `validate` | one `anyOf` of branches of 16 failing `const`s each, sharing one subschema by `$ref`, whose alternatives list more than `findings` (MTS-FND-020) | the choice cut short at the limit, with findings omitted |
| `grep-matches` | `grep` | one more 1,024-byte matching line than `--max-results` at the limit reports | the result truncated at the limit |
| `grep-context` | `grep` | every line matches, so each of the most matches repeats its neighbours as context, `--context` at the limit | accepted at the limit |
| `schema-joint-file` | `inspect` | the schema files hold exactly `schema-subschemas`, every leaf's location exactly `schema-location-length` long; they compile, and the combined schema is then refused by its count | the combined schema's refusal |
| `schema-joint-combined` | `inspect` | the combined schema holds exactly `schema-subschemas`, embedding a file whose leaves' locations are exactly `schema-location-length` long | accepted at the limit |

`schema-work` is counted before a validation evaluates anything
(MTS-SEC-046), so a validation one unit past the limit is refused at the
cost of the count alone. Its worst case is therefore the validation of
exactly the limit's units, which is accepted and does all that work.

## The record

Each run writes one JSON document, `metais.calibration/1`:

```json
{
  "format": "metais.calibration/1",
  "goos": "darwin",
  "goarch": "arm64",
  "go": "go1.26.9",
  "tool": "metais 0.0.0-dev ...",
  "cpus": 10,
  "ceilings": {"cpu_ns": 120000000000, "metadata_cpu_ns": 500000000, "rss_bytes": 1073741824},
  "cases": [
    {
      "case": "render-work",
      "limit": "render-work",
      "bound": 4294967296,
      "metadata": false,
      "reached": true,
      "exit": 4,
      "cpu_ns": 0,
      "rss_bytes": 0,
      "wall_ns": 0,
      "broken": [],
      "cpu_ceiling_ns": 120000000000,
      "rss_ceiling_bytes": 1073741824,
      "cpu_fraction": 0
    }
  ],
  "pass": true
}
```

- `go` is the toolchain that built the test, and so the tool. `tool` is
  `metais version`.
- Each case gives:
  - `bound`, its limit's default;
  - `reached`, whether it reached that limit, and `gap` when the tool is
    known not to;
  - `exit`, the command's exit status;
  - `cpu_ns`, its CPU time;
  - `rss_bytes`, its peak resident memory in bytes, already converted from
    `ru_maxrss`'s KiB on linux, or `null` where the platform reports none;
  - `wall_ns`, its wall time, which is recorded and never judged;
  - `broken`, each ceiling it broke, with the figure;
  - `problem`, anything else that failed it;
  - `cpu_fraction`, its CPU time over its CPU ceiling.
- `pass` is true when no case broke a ceiling or had a problem.

## What the act reports back

Run the command on each supported target: darwin/arm64, darwin/amd64 under
Rosetta 2, linux/arm64 and linux/amd64. For each, report:

- the host, its CPU, its operating system's version, `go version`, GOOS and
  GOARCH;
- the record file and its SHA-256;
- each case's CPU time and peak resident memory against its ceilings;
- every case that broke a ceiling, with its numbers, and anything else that
  failed.

The per-target numbers go in ADR-010's implementation log, written by the
act. A default whose worst case breaks a ceiling on a target is never
changed by the suite or by a work item. It is escalated as an ADR-009
amendment, which Ed ratifies (RULINGS R1-015), and A1's two limits are no
exception. A sandbox's run of the suite gives numbers but not evidence (plan
§5): a work item may report what its sandbox measured, but only this act's
runs calibrate a default.

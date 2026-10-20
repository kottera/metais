# ADR-010 — metais-test-economy: the selector reads the path a test names, and only the build reaches the blind packages

**2026-10-20, work outside a train (metais-test-economy).** Four W7 items
on one runner timed out on their scoped gates. Nearly every change
reached `internal/conformance`, `internal/guarantees`, `internal/hostfs`,
`internal/tools/release` and `cmd/metais`, about 1,300 CPU-s under
`-race`, by two of A5 item 3's rules:

- **The data rule matched a top-level entry.** A test literal beginning
  `"docs/` made its package a reader of every file under `docs/`. Every
  work item writes an ADR log, and `internal/conformance` names
  `"docs/guide.MARKDOWN"` and `internal/tarw` `"docs/guide/index.html"`.
  Both are entries of packages those tests write.
- **The blind rule took any path in any package.** That included another
  package's test files and testdata, which no build compiles.

A5 item 3 now reads paths, not top-level entries, and narrows the blind
rule to the module's build inputs:

- **Data.** `.claude/skills/verification-gate/gate-select-named.awk` reads
  the paths each package's test files name. It takes string literals and
  runs of literal `Join` arguments, from the package's directory and from
  the root. A path reaches a package when the package names it or a
  directory above it.
  - A glob, a format verb or a concatenation makes a prefix.
  - A run that climbs out of a computed base names any path ending in it.
    This keeps `internal/conformance`'s `filepath.Join(suite, "..",
    "statements.json")` a reader of `spec/statements.json`.
  - A bare word (`"docs"`, `"internal"`) names only the root's file. Every
    bare top-level directory name in the module's tests was a class, a
    field or an entry name.
- **Test files and testdata.** A `_test.go` file, or testdata the build
  does not embed, tests its own package alone. A data match tests its
  reader alone: no importer runs another package's test file.
- **Blind packages.** A non-test file in a package's directory, `go.mod`
  or `go.sum` reaches them. A test file or testdata does not.
- **Module walkers.** A test file in a package still reaches the root
  readers whose tests name Go files by a pattern: `internal/guarantees`,
  `internal/loader` and `internal/manifest`. `internal/guarantees` reads
  every Go file, every package's test imports and every benchmark
  baseline.

The selection, `gate-select.sh --paths`, before and after (base `7c3b362`):

- **PR #182's diff** (W7-NB2: four ADR logs, `spec/B-worked-package.md`
  and five test files in `internal/finding` and `internal/guarantees`):
  - Before: 37 packages, including all seven blind packages.
  - After: 6 packages, `internal/finding`, `internal/guarantees`,
    `internal/liquidspec`, `internal/loader`, `internal/manifest` and
    `internal/specdata`.
- **One ADR log alone** (`docs/adr/logs/ADR-010/W7-X.md`):
  - Before: 24 packages, including `cmd/metais`, `conformance`,
    `guarantees`, `hostfs` and `tools/release`.
  - After: none.

`TestGateSelectReach` gains its probes, and the probe format gains
`!<package>`, a package the path must not reach. Against the previous
script, four probes were red:

- an ADR log reaching `internal/conformance` and `internal/tarw`;
- `docs/adr/ADR-004-…` reaching `internal/conformance`;
- `internal/finding/x_test.go` reaching `conformance`, `hostfs`,
  `tools/release` and `cmd/metais`;
- `internal/finding/testdata/y.txt` reaching `conformance`, `hostfs` and
  `tools/release`.

The previous script cannot fail the guard that a non-test Go file in
`cmd/metais`'s closure reaches every blind package, because it reached
everything. Dropping the blind rule from the new script turns that guard
red.

Two of Wave 6's probes recorded the old rule's reach rather than a reader:

- **`spec/13-authoring.md`** named `internal/conformance` and
  `internal/loader`. Neither package's tests read a chapter: conformance
  reads `spec/conformance`, `spec/statements.json`, `spec/README.md`,
  `spec/findings` and `spec/schemas/conformance`, and loader reads
  `spec/conformance/vectors`. The probe now names `guarantees`,
  `liquidspec` and `specdata`, which do.
- **`examples/x/example.yaml`** named `internal/guarantees`, whose tests
  read only Go files and `spec/` below the root. The probe now names
  `internal/cli` and `internal/vectorgen`, which read `examples/`.

The data rule is held to what a test names. A test that derives a path
from a named one (`filepath.Dir`) or from a value it reads is not found,
as A5 item 8 says, and the wave tier runs it.

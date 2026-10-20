# ADR-009 — W7-R: Amendment A2's allow-list entry, `only-referrers` addition and `never-depends` rule, landed with the release builder

**2026-10-20, PLAN-2026-10-metais-v1 W7-R.** Amendment A2 lands with the
release builder it licenses, each part seen red without its change first.

- **Item 1, the allowance.** `internal/guarantees`' `moduleAllowList`
  gains `internal/tools/release`, with a `beneath` flag that A2's entry
  alone carries, for exactly the environment reads, every `os` file
  function, `os.TempDir`, `os.MkdirTemp`, `os.Args`, the standard streams
  and `os.Exit`. `os/exec` is no symbol of the symbol rule: the import
  rule keeps it out of the binary, and item 4 keeps the builder out. The
  `io/ioutil` and `path/filepath` file functions the symbol rule also
  counts as file access are not `os` file functions, so they stay refused.
  **Red first:** with the builder committed and no entry,
  `TestNoHostFacts/symbols` reported its `os.Exit`, `os.Args`,
  `os.Stdout`, `os.Stderr`, `os.Environ`, `os.Lstat`, `os.MkdirTemp`,
  `os.RemoveAll`, `os.ReadFile`, `os.Mkdir` and `os.WriteFile`.
- **Item 2.** `TestSymbolRule` holds the builder, and a package beneath
  it, to the refusals A2 keeps: the clock and timers, the working
  directory and `filepath.Abs`, the home directory, the host name, the
  process identity, `os/user`, a random source, `os.CreateTemp`,
  `filepath.WalkDir` and the environment's setters. A sibling named
  `internal/tools/releasex` and the parent `internal/tools` get nothing.
- **Item 4, `never-depends`.** `layers.json`: `cmd/metais` depends on
  nothing matching `internal/tools/...`. **Red first:** the control module
  `testdata/layers` gained `internal/tools/release/archive`, which its
  `cmd/metais` imports, and `TestLayersControl` failed without the rule
  for want of `cmd/metais depends on …/internal/tools/release/archive`.
- **Item 5, `only-referrers`.** The rule names `internal/tools/release`
  beside `cmd/metais`, exactly: the control's builder `main` keeps the
  exception, and its `archive` package beneath, which writes to
  `os.Stdout`, is flagged. **Red first:** with the builder committed,
  `TestLayers` reported its `os.Exit`, `os.Args`, `os.Stdout` and
  `os.Stderr` on all six targets.
- **Item 6.** `TestModuleAllowListIsR2143` pins the list to R2-143's four
  entries and A2's, and pins A2's entry's symbols to item 1's, spelled out,
  so the entry cannot grow with `hostSymbols`. It was red with the entry
  added and the test not yet amended.
- **Item 7.** No other allowance, scope or exclusion changed.

The SC7 harness's release build now uses the builder's own build line
(`internal/tools/release/buildline`), so the binary `TestHostIsolation`
clears is the one a release ships; its canaries are still caught.

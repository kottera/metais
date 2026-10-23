# ADR-010 — W2-3: the Homebrew formula is generated from SHA256SUMS (A8 item 3)

**2026-10-23, PLAN-2026-10-metais-public W2-3.** A8 item 3 takes effect,
for ADR-013 D7 item 5. The release builder gains a subcommand:

```bash
go run ./internal/tools/release formula -version <semver> -sums dist/SHA256SUMS > metais.rb
```

It writes the formula `metais` of the tap `kottera/tap` (ADR-013 D1 item
3) to standard output. Each shipped target's archive gets its own
`on_macos`/`on_linux` and `on_intel`/`on_arm` block. Its URL is the GitHub
release's, `https://github.com/kottera/metais/releases/download/v<semver>/<archive>`,
and its `sha256` is its line in `SHA256SUMS`. The generator refuses a
`SHA256SUMS` that is not exactly what the builder writes for the version:
one line per shipped target, in byte order of the name, and nothing else.
So neither the URLs nor the hashes can drift from the release. The formula
states no `version`, because Homebrew reads it from the URL and `brew
audit` refuses one that repeats it. So the generator also refuses a
version Homebrew 6.0 reads otherwise from the URL. It accepts a version
with no prerelease suffix, or a suffix `alpha`, `beta`, `pre` or `rc` with
an optional `.<number>`, the shapes Homebrew was seen to read back exactly
(`TestFormulaVersions`). It refuses the other semantic versions, such as
`1.0.0-0`, which Homebrew reads as `64`. Its `test`
block runs the installed `metais version`. Its caveat suggests
`metais skill`, as ADR-013 D6 item 9 asks.

- **Where it lives.** It is part of `internal/tools/release`, which ADR-009
  A2 already allows to read a file and write standard output, and which
  `layers.json`'s `never-depends` rule keeps out of `cmd/metais`. It needed
  no amendment, so ADR-009 A12, allotted to this item, stays unused.
- **The golden.** `internal/tools/release/testdata/formula/metais.rb` is
  generated from the `SHA256SUMS` beside it by `TestFormulaGolden -update`.
  It has a `merge=metais-generated` line in `.gitattributes` and a step in
  `regenerate.sh`. `TestFormulaIsTheRelease` reads the golden itself, so
  an update cannot commit a formula whose URLs or hashes differ from its
  `SHA256SUMS`. `TestFormulaRefusesSums` covers each malformed
  `SHA256SUMS`.
- **Not run here:** `brew audit` and `brew test`. Homebrew could not load
  the formula in this dispatch's sandbox: it fetches its API, or clones
  homebrew-core, before it reads any formula. The tap's CI (W2-9) runs
  both on what this generator writes.

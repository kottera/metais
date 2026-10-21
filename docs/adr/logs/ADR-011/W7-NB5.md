# ADR-011 — W7-NB5: the site publishes the command line's output schemas at their `$id`s

**2026-10-20, PLAN-2026-10-metais-v1 W7-NB5.** D7 item 5 keeps the
command line's output schemas in `docs/cli/schemas/`, outside the
specification, and each has the `$id`
`https://metais.org/cli/schemas/<file>`. W7-S1's `site/build.sh`
published `spec/schemas/` alone, so those `$id`s would not have resolved
once the site went live. The build now also copies every file of
`docs/cli/schemas/` to `cli/schemas/`, with `sh`, `cp` and `mkdir` as
before, and refuses a `site/public/cli` that would mix with it.
`site/site_test.go` reads both sources: `TestSchemaIDPaths` requires each
CLI schema's `$id` to name `cli/schemas/<its path>`, and `TestBuild`
requires each byte-identical there and nothing else under `cli/`. It was
red against the old build, red with one CLI schema's `$id` altered, and
red with one published CLI schema's bytes altered. The workflow keeps its
gate on a public repository and a `v*` tag.

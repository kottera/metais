# ADR-013 — W2-NB3: I2's unnamed-route fixture, gate.sh's SC2329 directives, and a neutral UX runner help

**2026-10-24, PLAN-2026-10-metais-public W2-NB3.** Three non-blocking
findings of Wave 2's verification, each a change to the development
tooling.

- **I2 on an unnamed route (D9).** The fixture
  `edited-i2-installs-by-unnamed-route` holds a session that installs
  metais with `mise use -g ubi:kottera/metais@1.0.0-rc.1`, a route the
  scorer does not name, and then runs it: `finds-and-installs` passes it.
  It is an edited run and says so in its `edited`. With that branch of
  `findsAndInstalls` made to fail, `TestFixtures` goes red on it.
- **SC2329 on gate.sh (D6 item 8).** shellcheck 0.11 reports SC2329,
  "function never invoked", on the functions `step` calls by name through
  `"$@"`: `gofmt_clean`, `static_build`, `cross`, `specdata_fresh`,
  `tripwire` and `reach`, and on `tripwire_names`, which only `tripwire`
  calls. Each definition carries a `# shellcheck disable=SC2329`
  directive saying so; nothing else in `gate.sh` changed. shellcheck 0.9.0
  accepts a directive naming a code it does not know.
- **A neutral help (D2 item 3).** The UX runner's `-host` help example,
  and `docs/dev/ux-testing.md`'s table and example of the flag, read "a
  dispatch host, linux/amd64".

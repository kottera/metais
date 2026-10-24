# ADR-010 — W3-11: the conformance vectors leave the module zip (A11)

**2026-10-24, PLAN-2026-10-metais-public W3-11**, made in Wave 3's release
act (W3-OOB4) after the rehearsal of `v1.0.0-rc.2` refused its `go install`
route (run 40134009587, job `go-install`). `spec/conformance/vectors/` is a
module of its own, the fuzz input `nul` is `nul-byte`, and
`TestModuleZipPaths` holds the module's paths to Go's module zip rules. It
was red on rc.2's tree with exactly the two paths, and is green on this
change. `x/mod` v0.42.0's `zip.CheckDir` finds no invalid path on the
change's tree. A module zip made from it by `zip.CreateFromDir` installs
through a local `GOPROXY`, and its `metais --version` reports the zip's
version.

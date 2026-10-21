# ADR-009 — W7-NB1: the release builder pins its go configuration within Amendment A2's allowance

**2026-10-20, PLAN-2026-10-metais-v1 W7-NB1.** The release builder now
builds with go.mod's `toolchain`, with no workspace, go env file, proxy or
checksum database of the developer's (ADR-010's W7-NB1 entry). The change
stays within Amendment A2 item 1's symbols. It reads go.mod with
`os.ReadFile`, an `os` file function, and it still reads the environment
only through `os.Environ`, which `main` hands it. Every variable it now
sets is in the environment of a command it runs with `os/exec`, never in
its own. It uses none of item 2's refused symbols, and `TestNoHostFacts`'
symbol rule holds it to that.

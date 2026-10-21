# ADR-009 — W7-NB2: TestLimitDefaults holds chapter 14's defaults to internal/limits

**2026-10-20, PLAN-2026-10-metais-v1 W7-NB2.** W7-V diffed chapter 14's
limit tables against D3, `internal/limits`, `metais limits` and the
calibration record by hand (W7-C5, W7-C6), and found them in agreement,
with nothing holding them so. `internal/guarantees`' `TestLimitDefaults`
now reads every table of `spec/14-security.md` headed
`| Limit | Default | Bounds |` and fails when a row names a limit
`internal/limits` lacks, names one twice, states a default that is not a
number, states a size without a size unit or a count with one, or states a
default other than the one `internal/limits` enforces. The test imports
`internal/limits`, so a change to either side reaches it. D3's table is a
design document and is not read: the specification and the code are what
the test holds together. `TestLimitDefaultsControl` breaks the chapter
once per way, each red before the check was written. The limits the tool
alone has (`regex-work`, `grep-matches`, `grep-context`) are in no
chapter 14 table, and the test does not ask them to be.

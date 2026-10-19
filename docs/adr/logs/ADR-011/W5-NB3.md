# ADR-011 — W5-NB3: the conformance runner holds no slot through the A/B wait

**2026-10-19, PLAN-2026-10-metais-v1 W5-NB3.** D11 item 9's default run
was sleep-bound. A case ran inside its parallel subtest, which holds a
`-test.parallel` slot for as long as it runs, so every case held one
through D12 item 4's two seconds, and the run took about 413 × 2 s / 16
asleep whatever the host's speed. Each case now starts before its
subtest waits for a slot, and an area and a vector start their cases
before they wait for one. The runner's own slots, `-test.parallel` of
them, bound the environments run at once, and the wait holds none.
`TestCasesWaitTogether` pins this with one slot: three cases of one
vector must be in the wait at once, and all must have started when
`runSuite` returns. Both environments, the offset and every case are
unchanged.

The run is measured on the compiled test binary, alone and not
race-built, as `go test -c ./internal/conformance` then
`/usr/bin/time -p conf.test -test.run '^TestConformance$'`, since
`TestConformance`'s own reported time leaves out its parallel subtests.
With go1.26.9 on a shared 16-core darwin/arm64 host, `-test.parallel` 16
went from 69.6 s of wall time (78.5 s of CPU time) to 16.0 s (67.4 s),
and `-test.parallel` 4 from 222.5 s (42.1 s) to 18.9 s (57.6 s).

The structural pin is the guard. A test that held the run's wall time to
60 s would measure the host's load as well as the run, which AGENTS.md
refuses. A test that held its CPU time would not bound its wall time,
because the run spends most of its wall time waiting on its processes
rather than computing: 67 s of CPU time over 16 cores is 4.2 s, against
16.0 s of wall time.

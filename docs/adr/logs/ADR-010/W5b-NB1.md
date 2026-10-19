# ADR-010 — W5b-NB1: StageNew_abort writes as render's prepare does

**2026-10-19, PLAN-2026-10-metais-v1 W5b-NB1.** D14 item 7: W5b-2 made
`TestShapesWriteTheirLimitUnderTheOpenFileLimit/StageNew_abort` call
`MkdirAll` once per directory, so no StageNew open-file-limit test took
`Output.MkdirAll`'s branch for a directory that exists. Render's `prepare`
takes that branch before all but the first file of every directory. The
subtest again calls `MkdirAll` of each file's directory before the file,
99,900 calls of that branch at the outputs limit's default under the
open-file soft limit of 128.

- **It catches the leak.** With a descriptor leaked by `syscall.Open` in
  `MkdirAll`'s `KindDir` case when `sub == name`, the subtest goes red
  (`create: io_error` after the limit's worth of files); the W5b-2 subtest
  stays green under the same mutation.
- **What it costs.** Measured on linux/amd64 with go1.26.9, under `-race`,
  CPU time by `getrusage`: the 99,900 calls, timed in one process, took
  0.45 to 0.51 s of CPU time. The whole subtest took 9.97 to 10.45 s
  before and 10.29 to 10.73 s after, across three alternating pairs.
- **A cheaper shape was weighed.** A per-call leak trips after 128 calls,
  so a few hundred calls on one directory would also catch this mutation.
  The subtest instead keeps render's own shape at the default counts, for
  half a second.

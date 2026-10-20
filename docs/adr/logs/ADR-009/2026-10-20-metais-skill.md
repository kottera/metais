# ADR-009 — 2026-10-20-metais-skill: A5's host access for `metais skill` built

**2026-10-20, outside a train.** `metais skill` (ADR-008 A2) is built
with the host access A5 allows, and no more.

- **`internal/skill.Install` is the one way to `hostfs`.** It opens the
  caller's directory with `hostfs.OpenWriter` and hands the rest of the
  package plain functions over it, so `layers.json`'s new
  `single-entry-point` rule holds: no other function names `hostfs`.
- **`hostfs.Writer.ReadFile`** is new. It is `Dir.ReadFile` on the
  writer's own root, after checking every component above the file as a
  write does, so the stamp is read through the same root the write uses.
  Its comment names metais skill as its one caller. A render still never
  opens an existing file (D8 item 4).
- **The `never-depends` rules** name `internal/loader`, `internal/zipfmt`,
  `internal/manifest`, `internal/inputs`, `internal/template` and
  `internal/browse`. `internal/pkgtree` is in the closure, as the type
  `hostfs`'s source-directory reader returns, and A5 item 4 says so.
- **No symbol is allowed anywhere new.** The package uses `hostfs`,
  `crypto/sha256`, `embed`, `regexp` and `bytes`.
  `TestModuleAllowListIsR2143` is unchanged.
- **Hostile cases.** `internal/skill`'s `TestHostile` covers each case
  item 5 lists and checks that nothing outside the directory changes and no
  temporary file is left behind:
  - `.claude` as a link pointing out and in;
  - a link and a file at the skill's directory;
  - a link, a directory and a FIFO at `SKILL.md`;
  - a directory that cannot be written.

  `TestHandEdits` covers a file over the 64 KiB bound, one at the bound
  with a forged stamp, and a stamp holding a control character. That
  stamp is never echoed. `TestHostile` and `internal/cli`'s
  `TestSkillRefusals` cover a `--dir` that names a file or nothing.
- **The SC7 harness** runs the command in `internal/conformance`'s
  `TestSkillHostIsolation`, as item 5 requires:
  - 24 command lines: each harness in human and `--json` form, and
    `--print` in both, each in both path modes;
  - each run in environments A and B, B started two seconds after A;
  - a decoy `--dir` whose name holds the sandbox-path sentinel;
  - A's decoys, sentinel `SKILL.md` copies among them, in the home,
    working and temporary directories.

  It makes checks 1 to 4 with the runner's own helpers. Check 4 allows
  only the `SKILL.md` the decision names and its directories, and nothing
  at all with `--print`. The test also asserts that every run exits 0 and
  that the file written is what `--print` prints. Check 2's vocabulary
  adds the skill's text, which is the tool's own words. A self-check
  shows the decoy directory's absolute path in a stream is caught.

# ADR-007 — W3-NB1: the sample generator settles the input constraints, so `--optional mixed` previews the ACR

**2026-10-24, PLAN-2026-10-metais-values-and-rules W3-NB1.** A
non-blocking finding of Wave 3's verification: `metais render
examples/accessibility-conformance-report --sample --optional mixed --seed
7` failed with `schema.sample_invalid` about `section508`, `input.missing`
under the condition at `/allOf/0/if` of the package's
`inputs/edition.schema.json`. That is its input constraints (ADR-003 D7),
not a row schema's `if`.

- **The mechanism, in the generator.** The generator made each input
  alone, from its own schema. It decided each optional input by that
  input's `include` stream and never read `input_constraints`. Under
  `mixed`, the report's optional `edition` and each optional table were
  decided by independent draws, so seed 7 paired an edition with tables
  it refuses. `all` passed only because the edition's `examples` value is
  `international`, which holds every table. `none` passed only because an
  omitted edition is `wcag`, which holds none. With the edition supplied
  by `--set /report/edition=<edition>` at seed 7, `wcag`, `section508`
  and `en301549` failed under `all`, and `section508`, `en301549` and
  `international` under `none`. The
  package's constraint is satisfiable, and a generator that decides the
  tables after the edition satisfies it. So the defect is the generator's,
  and the package is unchanged. The DORA incident report failed the same
  way under `none` and `mixed` at seeds 1 and 7: its stage's branch
  requires fields of `incident` that the generator did not know of.
- **The fix, within D9.** `sample.Generator.Inputs` makes the inputs
  object the way item 15 makes an object, the modes applying to inputs
  as ADR-003 D1's `required` lets them, and settles its conditionals as
  item 9 does:
  - each input is made from its own schema and the schemas the
    constraints give its name;
  - an optional input is decided by its `include` stream, unless the
    constraints require it, when it is made, or give it the false schema,
    when it is left out;
  - each `if` is evaluated against the inputs object, and the inputs
    whose schemas or requirement changed are decided again, at most
    `maxPasses` times.

  An input the constraints both require and refuse is
  `schema.sample_unsatisfiable` with `reason` `type`. Conditionals that
  do not settle are `reason` `conditionals`. Both are about the input,
  located at the constraints' schema. A package without
  `input_constraints` is generated exactly as before. Where the old
  generation validated, its data is unchanged.
- **The specification.** Unchanged. MTS-SMP-002 requires data that
  validates, and §13.6 leaves how it is made to the tool. No `smp/`
  vector is added, because a conforming generator may still refuse with
  `schema.sample_invalid`, so no vector can require a sample to succeed.
- **Proof.** `internal/sample`'s `TestInputsConstraints` (every mode, 32
  seeds, and mixed reaching both branches), `TestInputsSupplied`,
  `TestInputsConstrainValue`, `TestInputsUnsatisfiable` and
  `TestInputsWithoutConstraints`. `internal/cli`'s
  `TestACRSampleEditionTables` renders the ACR with no finding under
  `all`, `none` and `mixed` at seeds 1, 7 and 42, and with each edition
  supplied, and checks that acr.json holds exactly the edition's tables.
  Its mixed and edition-supplied runs are red on `main`.

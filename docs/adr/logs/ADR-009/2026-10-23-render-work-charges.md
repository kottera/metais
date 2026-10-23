# ADR-009 — 2026-10-23-render-work-charges: Amendment A10, D11's charges and `assign-bytes`

**2026-10-23, outside a train.** This item carries out Ed's rulings at
PLAN-2026-10-metais-v1's T-A4 on its item M-6, where the engine's work and
size charges parted from the specification. Amendment A10 is proposed with
them.

- **The specification follows the engine** for a range made a value
  (16 + 2 + n), an array written (a unit per element, nested arrays'
  included) and a string's `size` shorthand (a unit per byte). §7.11 now
  defines an *operator*, and MTS-TPL-063 and MTS-SEC-008 state the three
  charges. `tpl/charge-range-value`, `tpl/charge-array-write` and
  `tpl/charge-size-shorthand` render at their exact bound, 135, 20 and 38
  units, and are refused one unit below it. All three passed on main
  before the change, since the engine already made these charges.
  `render_meter_test.go` already pinned each of them.
- **The engine follows the specification** for `assign-bytes`. The
  standard catalog's `build`, `internal/template/filters`' `text` and
  `json_script`, and `internal/dates`' `date` no longer charge held bytes
  for the strings they build. `ifchanged` no longer charges its body
  either. An assigned filter result is charged once, at the `assign`.
  `sec/assign-bytes-filter-built` renders
  `{% assign s = "abc" | append: "d" %}`, then writes `s`, `s | append:
  "e"` and an `ifchanged` body, at `assign-bytes=4`, and refuses it at 3.
  Its `at-the-limit` case failed on main, refused by `assign-bytes`, and
  passes after the change. `sec/assign-bytes` and
  `sec/hostile-template-work`'s `doubling-assign-bytes` pass before and
  after it. `TestRenderCharges` pins the held bytes of the same template,
  4, where main counted 17. `TestGrowthRefusedBeforeWork` holds each
  growing filter to `render-work` in place of `assign-bytes`: the meter
  holds the operands' units and 16 KiB more, and each filter is refused,
  having charged its operands, before it builds its result.
- **`WorkPartials`** stays as the meter's own count, which tests read.
  Its comments say it is bookkeeping that never refuses and is no charge
  of D11.
- **The calibration cases still hold**, measured at the defaults on
  darwin/arm64 against main at f8929b83 and this change. The
  `assign-bytes` case is still refused by `assign-bytes`. It used 0.15 to
  0.32 s of CPU and peaked at 145 MiB before the change, and 0.31 s and
  145 MiB after it, since it now assigns twice as many copies before it
  is refused. W7-C9 recorded 0.45 s and 142 MiB. The
  `assign-bytes-capture` case is still refused by `assign-bytes`. It
  peaked at 529 MiB before the change and at 658 to 785 MiB after it,
  within 1 GiB. The string its captures write is built by doubling
  assigns, whose built strings were charged twice, so before the change
  they spent half of `assign-bytes` and left the captures less of it.
  Captures fed by an input string spend none of it on assigns. Nine
  nested captures of 64 MiB each, written from a 32 MiB input, peaked at
  974 MiB on main and 1005 MiB with this change, and both were refused by
  `assign-bytes`. That template applies no filter and assigns nothing,
  so this change does not reach it. Its margin is for the work on what a
  capture holds, and this item changes nothing there. Amendment A6's
  joint render cases assign nothing. They peaked at 998 MiB (`json`) and
  982 MiB (`yaml`) after the change, within 1 GiB.
  `TestCalibrationLowered` still has every case reach its limit.

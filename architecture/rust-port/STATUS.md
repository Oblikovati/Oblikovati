# Rust port status

Last checked: 2026-11-18. Recheck the branch, issues, and scenario source before
using this status; the live scenario directory can change.

## Start here

- Read `ACTION-PLAN.md`, `REVIEW-2026-11-17.md`, and the applicable `AGENTS.md`.
- Preserve the user's modified `README.md` and the untracked review file.
- Integrate completed work into `rust-work` and push; remove merged task branches,
  worktrees, and task-owned caches after checking them.
- Use the Go application as the reference and BDD scenarios as acceptance.
  Do not change Martensite; report kernel defects to its agent.

## Current evidence

- The ledger reports **0 verified of 2,760** declarations at local snapshot
  `b576445b98cffd686dcc05aa8c017de2a591c8c8`. This is not an upstream scenario
  commit. Run `python3 -B tools/bdd/behavior_cli.py report` for current counts.
- Review P0 F1–F4 are merged. Workspace all-target Clippy and `make docs-lint`
  passed. The post-merge docking diagnostic passed 8 scenarios and 65 steps;
  two capture-only cases were excluded. Input was mixed X11 and scripted events,
  and the unversioned source prevents verified acceptance. See
  `receipts/2026-11-17-p0-dock-lifecycle-4073/diagnostic.json`.
- The GUI library passed 587 tests during the P0 work. A full all-target run
  reported 22 integration failures across four targets; triage is #4078. That
  run preceded the final dock lifecycle change and is not a current green suite.
- The head part-scene suite passed 134 of 135 tests; a modified body click
  selection test fails on both this work and clean parent `7ebc06bdb` (#4091).
- The scripted native-window cube journey passed on `1ac04f5f3`, including
  undo, redo, save/reopen, and named-face checks. Its images show a front
  projection, and input is injected into the GUI loop; OS-level acceptance and
  solve-precision qualification remain open (#3857, #3928). See
  `receipts/2026-11-18-cube-gui-diagnostic-3857/diagnostic.json`.
- Fillet drawing, suppression-health, and extrusion-preview regressions pass
  using the complete named-solid evaluator (#4074). The evaluator requires
  explicit blend and named-face operations.
- Fillet and Chamfer creation reuse a current matching prefix. Tests count zero
  full body lookup, validation, and publication replays with a warm prefix;
  without it they count one of each. This does not measure frame rate (#4079).
- Feature-boundary checkpoints retain bodies, lineage, datums, and health.
  Session guards reject stale or foreign histories and restore exact names only
  for the accepted edit. Real-kernel tests compare named face areas with a full
  replay, including a Fillet consumed by a later Chamfer (#4079).
- Each document session owns up to four policy-scoped part results and their
  checkpoints. Hosts share a matching result; publishing a fifth policy retires
  the least recently published one. An edit invalidates all old generations,
  and closing the document drops all results (#4079).
- A cold checkpoint request for a part without associative sketch recompute
  publishes its result and boundaries from one replay instead of two. Ownership
  tests cover repeat requests and isolation between documents (#4079).
- A cold checkpoint request with a settled face sketch also publishes from one
  ordered replay. When sketch state changes, a second replay publishes the
  updated state; the native-owner test checks both paths (#4079).
- Certified Extrude creation publishes its validated document replay before
  display when a current prefix is available. Repeated previews reuse that
  prefix and cancel without an edit. Tests check two-extrusion names, policy
  isolation, and rejection by injected validators (#4079).
- Parameter Set and Update reuse a current document checkpoint for a proven
  clean prefix. A typed API test edits a Fillet parameter twice, checks named
  faces against a full replay, and exercises undo, redo, save, and reopen (#4079).
- Drawing creation, live display, previews, and saved views obtain a current
  document-owned part, including when the injected owner uses another policy
  or is absent. Tests cover source edits, undo, retained results, and refusing
  to save an unhealthy source (#4079).
- On isolated Xvfb, 8 of 9 native drawing-opening tests passed. The remaining
  test could not select the clipped `layout` tab; it failed the same way on
  clean parent `01c398155` (related clipped-tab investigation #4082).
- The desktop replay owner creates checkpoints on demand for blend edit
  previews. Successive previews and accepted edits reuse matching checkpoints.
  Associative sketches use ordered recompute. The per-document session does not
  yet own the complete evaluation service, and other edits and parameter changes
  still use conservative replay (#4079).
- A newly appended feature may add parameters without dirtying an unchanged
  prefix when no earlier feature reads them. Other parameter changes remain
  conservative (#4079).
- The dock lifecycle inventory and representative first-frame native tests pass.
  Projects pointer docking, modal-child saved-layout restart, and File Properties
  native docking remain open in #4073.
- The 60 fps sketch target remains unproven. The recorded 1440 × 900 probe
  measured 20.738 ms median and 24.861 ms p95 against a 16.67 ms frame budget.
  See `receipts/2026-11-16-sketch-frame-baseline-4031/` and #4032.
- The full gate is not green. The unreleased API contract is #3965; the native
  build-provenance and receipt-reuse gaps are #3978 and #3974. Do not count an
  excluded or unexecuted scenario as a pass.

## Work queue

1. Complete the evaluation owner and safe dirty-prefix replay in #4079, with
   generation, policy, invalidation, naming, and replay-count evidence.
2. Repair and verify the GUI integration targets in #4078. The separate
   first-interaction row movement is #4077.
3. Complete the dock lifecycle checks still listed in #4073.
4. Continue ledger baseline and milestone exit work in #3865, #3861, and #3860.
5. Recheck scenario changes through the scheduled sync; the source may be edited
   while tests run. Capture source hashes before and after every diagnostic.

## Evidence and history

`behavior-ledger.json` and its generated report are the counting source.
`receipts/` contains recorded runs and their limitations. The prior status is
preserved in [STATUS-history-2026-11-17.md](STATUS-history-2026-11-17.md).

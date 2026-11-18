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

- The live feature-run inventory reports application closing green: 3 of 3
  scenarios at SHA-256 `d87eaf041fc93701e0de5c911dd3b4c0622386d184f05954a8409400e412c0d0`.
  Run `python3 -B tools/bdd/feature_progress.py /mnt/vininas-projects/MCAD-Scenarios`
  for current feature hashes and counts. This result is a full-feature diagnostic
  at its recorded application commit; the strict acceptance ledger is separate.
- Viewport pan has a full-feature diagnostic: all four scenarios and 32 steps
  passed at SHA-256 `4303c8a64b9262841dba3b8b48a74e51581a6a0ef9d98798286d514c53155647`.
  Two separate X11 mouse tests passed for pan dragging and context-menu actions.
- Viewport zoom has a full-feature diagnostic: all seven scenarios and 74
  steps passed at SHA-256 `cb07444a13797a6d98fac5b8f26463343a1cbff5ed4781c359290bd6e4856e9c`.
  The wheel steps used X11 input; other actions used scripted window input.
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
- A focused X11 mouse and keyboard test creates a parameter-driven, fully
  constrained cube; edits its size; checks undo, redo, save/reopen, and named
  topology. Its [isometric capture](receipts/2026-11-18-os-cube-3857/cube.png)
  shows the resulting solid. This is a diagnostic, not a BDD scenario receipt;
  solve-precision qualification remains open (#3857, #3928).
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
- The document session runs and publishes normal full and ordered-sketch
  recomputes and creates feature checkpoints. The file workspace supplies the
  document identity and validates the policy; tests cover separate documents,
  stale generations, and checkpoint reuse (#4079).
- Checkpoint candidate replay and restoration now use a session-issued
  capability. The workspace rejects a foreign, wrong-policy, or stale prefix;
  tests also reject stale candidate histories (#4079).
- Certified Fillet and Chamfer edits use that capability for checkpoint replay.
  They require a current descendant and preserve injected validator refusals;
  tests cover foreign identities and a Fillet consumed by a Chamfer (#4079).
- Fillet and Chamfer creation now use a session-issued whole-part capability
  for append replay. Tests retain the replay counts and reject foreign or stale
  histories and validator refusals (#4079).
- Extrude creation also uses that capability, including the first empty-part
  replay. Tests cover policy and identity mismatches, an injected refusal, and
  named faces after a second extrusion (#4079).
- Captured append results are sealed to their document and policy. The file
  workspace restores and publishes them only for the accepted transition;
  tests reject foreign documents, wrong policies, and stale history (#4079).
- Feature-edit previews now replay through the document session, under its
  identity and policy. A private preview does not publish a result or change
  the document generation; the Fillet test compares full and checkpoint
  previews and checks cancellation and history (#4079).
- Extrude creation previews also use the session. A current append prefix is
  inspected under its generation guard without copying all bodies; tests compare
  named face areas with a full replay and cover policy isolation (#4079).
- Face-based datum previews use the same session preview service and current
  checkpoints. Tests check the offset geometry, unchanged document generation,
  and the face-plane command through history and reopening (#4079).
- Sketch inspection reads a retained part result when identity, resolution,
  solver options, and generation match. It recomputes for a stale or unmatched
  result; tests check face-sketch placement after a depth edit (#4079).
- Sketch continuity resolves all live watches in one batch per history refresh.
  Tests count one resolver call, preserve individual failures, and reject a
  malformed result count; the 60 fps target is not yet established (#4079).
- Fixed and origin-plane sketch continuity avoids a part replay when orientation
  has no feature-axis dependency. A mixed or feature-oriented batch still uses
  ordered replay; real-kernel tests count both paths (#4079).
- The scripted native Extrude preview test now opens Parameters from its
  Quick Access icon. Its former text lookup clicked the browser node; the test
  checks preview updates, stale requests, and cancellation (#4078).
- The native Fillet edit test now targets the radius input using its visible
  label. The previous click hit the edge count; the corrected scripted test
  checks preview, cancel, commit, undo, redo, and save/reopen (#4078).
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
- End-of-Part edits use matching retained checkpoints for a proven clean prefix.
  Tests check rollback, restore, names, undo/redo, save/reopen, and full replay
  for transactions and mismatched policies (#4079).
- Feature deletion uses the same owner-checked replay path. Tests check later
  and first-feature deletion, replay counts, names, undo/redo, and reopen (#4079).
- Feature reorder reuses the unchanged prefix before the first moved feature.
  Tests cover replay counts, names, rejected moves, undo/redo, and reopen (#4079).
- Prism distance edits use the same checkpoint path. Tests compare every named
  face area with full replay and cover invalid inputs, history, and reopen (#4079).
- Drawing creation, live display, previews, and saved views obtain a current
  document-owned part, including when the injected owner uses another policy
  or is absent. Tests cover source edits, undo, retained results, and refusing
  to save an unhealthy source (#4079).
- On isolated Xvfb, 8 of 9 native drawing-opening tests passed. The remaining
  test could not select the clipped `layout` tab; it failed the same way on
  clean parent `01c398155` (related clipped-tab investigation #4082).
- The desktop replay owner creates checkpoints on demand for blend edit
  previews. Successive previews and accepted edits reuse matching checkpoints.
  Associative sketches use ordered recompute. Some preview and feature-specific
  orchestration still lives outside the session, so the complete evaluation
  service remains unfinished (#4079).
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

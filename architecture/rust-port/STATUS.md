# Rust port status

Last checked: 2026-11-20. Recheck the branch, issues, and scenario source before
using this status; the live scenario directory can change.

## Start here

- Read `ACTION-PLAN.md`, `REVIEW-2026-11-17.md`, and the applicable `AGENTS.md`.
- Preserve the user's modified `README.md` and the untracked review file.
- Integrate completed work into `rust-work` and push; remove merged task branches,
  worktrees, and task-owned caches after checking them.
- Use the Go application as the reference and BDD scenarios as acceptance.
  Do not change Martensite; report kernel defects to its agent.

## Current evidence

- Ribbon Appearance (#3957): the 2026-11-20 diagnostic passed 14 of 18
  scenarios. Three failed at missing Tools panels; one used an unpinned
  expansion where the fixture notes require a pinned expansion. The shared
  part setup also lacks the visible solid required by the fixture notes.
  `BLOCKERS.md` changed during the run, so the result is not inventory
  evidence. See `receipts/2026-11-20-ribbon-appearance-diagnostic/`. A native
  mouse test passes the pinned expansion through Text Off and Normal; see
  `receipts/2026-11-20-ribbon-pinned-3957/`.
- Midpoint Line (#4178): the scenario source changed and now requires an
  Inventor fixture for all six scenarios. Inventor interop is deferred. The
  unfinished feature worktree is retained, and a native YAML fixture is
  requested in `BLOCKERS.md`. Earlier runs do not verify the current source.
  Selected-origin identity is tracked separately in #4179.
- Constraint Settings (#4158): a full diagnostic on the uncommitted feature
  worktree passed five of eight scenarios with the feature and support hashes
  unchanged during the run. Three scenarios stopped at undefined setup steps;
  fixture questions are in the feature's `BLOCKERS.md`. This worktree result is
  not inventory evidence. The same result was reproduced after the checkbox
  diagnostic fix (#4175), with feature and support hashes unchanged. Point
  inference persistence (#4160), Relax Mode (#4164), and settings acknowledgement
  (#4172) remain in progress. Overlapping glyph selection (#4169) is merged.
  The merged branch has separate passing persistence, typed API, and X11 glyph
  checks in `receipts/2026-11-19-merged-glyph-0b3efd7/`.
- New Sketch (#4156): the full run at `60c48f25f` passed nine of ten
  scenarios. The accumulated document fixture remains unresolved (#4011) and
  requires circular sketch patterns (#3905). See the feature's `BLOCKERS.md`
  and the full-run receipt.
- Explode Block (#4153): the full run at `91e69a377` passed two of five
  scenarios. The other three need the portable fixtures requested in the
  feature's `BLOCKERS.md`. Native checks also cover canvas and browser
  explosion, color inheritance, selection of another instance, and analytic
  Measure before and after explosion. The latter fix is merged at `1d727f246`.
  The API contract is on `work/explode-block-310` at `0bf27c8`; its full API
  gate passed, but the contract is not released (#310, #311).
- Constraint inference scope (#4152): the full native run at `f7186111b`
  passed two of four scenarios. The recorded selection fixtures still need
  portable geometry; the scenario owner has been asked in `BLOCKERS.md`.
- Sketch preferences (#4148): the complete native run at `6fdb0dc6c` passed
  all three scenarios and 26 steps. Sketch edits are staged, Axes can be
  applied off and on, and HUD input controls retain their values on Cancel.
  Line scenario `line-30` now keeps the Background sketch, but its measurement
  fixture needs independent source coordinates (#4150).
- Sketch visibility (#4151): the complete native run at `e82b8006b` passed
  both scenarios and 17 steps. Isolation preserves earlier body visibility;
  a native regression also checks restoration when the sketch is finished.
- Model browser: the full native run at `0890f84e0` passed 97 of 100
  scenarios against feature SHA-256
  `603382ec4009d73a74e960c5981d255c4ca9ac5489b94fcf57193c7b09458b2e`.
  Two failed test checks were corrected and passed focused reruns; the full
  feature has not been rerun after those fixes. The remaining extrusion-owned
  angle needs tapered extrusion (#4127, Martensite #463). The feature is not
  done. The scenario source is unversioned, so strict verified acceptance is
  still blocked by #3978.
- The full workspace-layout run at `c0060be36` passed 50 of 52 executed
  cases, including both outline examples. Its inventory counts 49 of 51
  declarations. The spacing and User Interface menu scenarios remain
  incomplete; their source questions are in the feature's `BLOCKERS.md`.
- Source edits have invalidated earlier runs, including runs previously
  reported as green. Counts are tied to source hashes and must be regenerated
  with `tools/bdd/feature_progress.py` before reporting them.
- Sketch constraint-glyph hover passed its complete one-scenario native run at
  `e33bac9dd`. Application closing previously passed 3 of 3 scenarios, but its
  source has changed and now includes assembly scenarios, which are deferred.
  Run `python3 -B tools/bdd/feature_progress.py /mnt/vininas-projects/MCAD-Scenarios`
  for current feature hashes and counts. This result is a full-feature diagnostic
  at its recorded application commit; the strict acceptance ledger is separate.
- Viewport pan has a full-feature diagnostic: all four scenarios and 32 steps
  passed at SHA-256 `4303c8a64b9262841dba3b8b48a74e51581a6a0ef9d98798286d514c53155647`.
  Two separate X11 mouse tests passed for pan dragging and context-menu actions.
- Viewport zoom has a full-feature diagnostic: all seven scenarios and 74
  steps passed at SHA-256 `cb07444a13797a6d98fac5b8f26463343a1cbff5ed4781c359290bd6e4856e9c`.
  The wheel steps used X11 input; other actions used scripted window input.
- Viewport navigation wheel has a full-feature diagnostic: all 12 scenarios
  and 143 steps passed at SHA-256 `9793d6a9ad9655ef4412295075dfbf9f6f5a5f97566e5564f2ca9f5a9c68f62a`.
  This run used scripted window input.
- Main-window-frame passed 20 executed cases and 149 steps using X11,
  Openbox, tint2, and AT-SPI. Functional and visual notes were reviewed;
  minimize and maximize use a saved-part fixture. One prior full run failed
  intermittently at the application-bar double-click; its report is retained
  beside the passing run (#4063).
- Viewport orbit has a full-feature diagnostic: all 10 executed cases,
  including both examples of one outline, passed at SHA-256
  `d0e82e1671dc51da3c8eafa7d25101dc66713552639adf9dec041fac6d05eeb6`.
  This run used scripted window input.
- Viewport entity selection now has executable scenarios and an Inventor part
  fixture. The fixture step reaches the document host, which rejects the binary
  file as invalid YAML; native Inventor and 3D-sketch support remain open
  (#4092). Separate X11 face-selection tests do not count as scenario passes.
- Corliss 0.1.0 now reads point and line records from the Inventor fixture's
  two populated 3D sketches, with names and endpoint identities. This is a
  read-only prerequisite, not native document opening or viewport support;
  see `implementation-inventor-spatial.md` (#4092). A validated model now
  retains the recognized points and lines with shared endpoint identities.
  Corliss solid conversion is requested in [Martensite #545](https://github.com/Oblikovati/Martensite/issues/545).
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

1. Diagnose Ribbon Appearance (#3957) against its current source. Its fixture
   notes require a visible solid, but the shared part setup creates an empty
   part; clarification was requested. Midpoint Line (#4178)
   now requires a deferred Inventor fixture; retain its unfinished worktree.
   Constraint Settings (#4158) is paused with uncommitted work while its
   fixture questions and Relax Mode (#4164) remain open. Revisit `new-sketch`
   when fixture #4011 and circular patterns #3905 are available. Defer
   Inventor-file, assembly, and 3D-sketch scenarios.
2. Complete the evaluation owner and safe dirty-prefix replay in #4079, with
   generation, policy, invalidation, naming, and replay-count evidence.
3. Repair and verify the GUI integration targets in #4078. The separate
   first-interaction row movement is #4077.
4. Complete the dock lifecycle checks still listed in #4073.
5. Continue ledger baseline and milestone exit work in #3865, #3861, and #3860.
6. Recheck scenario changes through the scheduled sync; the source may be edited
   while tests run. Capture source hashes before and after every diagnostic.

## Evidence and history

`feature-run-results.json` and `tools/bdd/feature_progress.py` report full-feature
diagnostics against current source hashes. The separate behavior ledger reports
strictly verified acceptance; unversioned scenario source prevents that proof.
`receipts/` contains recorded runs and their limitations. The prior status is
preserved in [STATUS-history-2026-11-17.md](STATUS-history-2026-11-17.md).

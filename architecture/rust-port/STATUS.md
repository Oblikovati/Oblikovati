# Rust port status

Last checked: 2026-11-15. Refresh from current evidence at each checkpoint.

## Start here

1. Read this file, `ACTION-PLAN.md`, the pending handoff, and `git status`.
2. Preserve the user's modified `README.md` and existing stashes.
3. Verify the pinned scenario checkout and receipt before relying on a result.
4. Use the Go application as a reference and the BDD scenarios as acceptance.
5. Merge finished work into local `rust-work`; clean merged task worktrees and caches.

## Current state

- **R0 remains open.** The full gate, baseline fields and milestone exit audits
  are incomplete. Run `python3 -B tools/bdd/behavior_cli.py report` for current
  counts; source and application changes can make earlier receipts stale.
- At `e7d0230d17d606a0b70728a745d79345a57f3928`, the ledger reported zero
  current verified scenarios, 155 stale receipts, 34 blocked and 38 excluded
  across 2,720 declarations. The gate also requires missing baseline fields.
- `rust-work` is local and has not passed the full pre-push gate. All 20 former
  application PR heads were merged locally and their PRs closed. Do not remove
  their remaining remote branches until the consolidated work is safely pushed.
- The 2026-11-15 `make gate` stopped at the API pin check, before workspace
  tests: Express requires an unreleased API contract. Release and pin work is
  tracked in #3965. API branch `agent/express-release-prep` passed its gate and
  its issue-scope duplication scan at 1.0269%. The two example packages now
  report 95.7% and 85.7% coverage; aggregate Go coverage is 84.6%. The branch
  is local and no release was published. The ledger also rejects stale receipts.
- Scenario pin: `8ec8d2c1e569fbc5c8cd2ec85e9b9380c62c147f`, a local snapshot
  of the unversioned source. It is not an upstream commit. The live feature
  inventory matched the pin at the source sync. The source tags 38 observations
  as capture-only and the remaining 34 document-creation cases as capture-host
  only; these are excluded or blocked, not passing. Follow `bdd-workflow.md`
  when source files change; use a unique Cargo target for each active worktree.
- The native harness injects input into the window frame loop. It does not prove
  OS-level key and mouse delivery. Close test windows and retain receipts.
- `make docs-lint` passed after the source-size and worktree-discovery fixes
  (#3970, #3971). The full `make gate` remains open.
- Martensite work is owned separately. Report validated kernel gaps in
  `Oblikovati/Martensite`; do not modify the kernel in this lane.
- GUI unit tests isolate the shared ImGui context under parallel execution
  (#3878). The final eight-thread run passed 495 tests; the production context
  ownership check still rejects a second active owner. Native windows and
  headless diagnostics now share a test-only lease (#3878). Two Clear tests
  passed twice with two threads, and a mixed group of 15 passed with eight;
  the full native head suite remains unverified. A partial serial run
  recorded two failures; see `receipts/2026-11-15-head-partial/`. The native
  composition test passes with the sized Xvfb setup (#3973); use
  `make gate-xvfb` to run the actual gate on a suitable virtual display.

## Queue

| Order | Action | State | Tracking |
| --- | --- | --- | --- |
| 1 | Complete the behavior ledger and R0 baseline fields | In progress | #3865 |
| 2 | Review milestone mappings and Go baselines | In progress | #3861 |
| 3 | Replace old prose claims with current durable receipts | In progress | `behavior-receipts.md` |
| 4 | Audit R0–R3 exits and the full gate | In progress | #3860 |
| 5 | Split over-limit Rust files | Done | Guard remains enforced (#3860) |

## Active gaps

- Extrusion undo and redo: the native diagnostic passed three of four scenario
  steps. Named edge lengths are available through the typed API; selecting and
  measuring two edges in the viewport remains open (#3967). See
  `receipts/2026-11-15-extrusion-history-diagnostic/`. General analytic curve
  picking is blocked by missing interior extrema in
  [Martensite #462](https://github.com/Oblikovati/Martensite/issues/462).
- Extrusion entry: the new metric part and origin-plane selection scenario
  and the following preview, cancel/reopen and zero distance recovery scenarios
  passed before later source and test changes (#3927, #3931, #3929); their
  receipts are stale. The solve precision certificate remains open (#3928); see
  `solver-precision-investigation.md`.
- Three-point arc: the collinear-input boundary scenario passed; see
  `receipts/2026-11-15-arc-boundary-refresh/`. Its receipt is now stale.
- Extrusion editing: browser distance editing, candidate preview, cancel,
  history and reopen passed focused native tests (#3968); see
  `receipts/2026-11-15-extrusion-distance-edit/`. A live XTest and MCPBridge run
  also checked the edit and preserved shared parameters; see
  `receipts/2026-11-15-live-extrusion-edit/`. The pinned profile-extrusion
  contract has no taper input; kernel work is tracked in
  [Martensite #463](https://github.com/Oblikovati/Martensite/issues/463).
- A manual XTest and MCPBridge run created a part and free rectangle, extruded,
  undid, redid, saved and reopened it. A second run compared pre-save and
  reopened topology keys and measured a pre-save face key after reopening.
  The runs did not create the parameter-driven cube. See
  `receipts/2026-11-15-live-os-extrusion/` and
  `receipts/2026-11-15-live-face-replay/`.
- A later live XTest run created the zero-DOF, parameter-driven cube, edited
  `size`, undid, redid, saved, reopened, and resolved its original face keys.
  The per-face measurements and captures are in
  `receipts/2026-11-15-live-parameter-cube/`. This does not close the solver
  precision or full milestone gates (#3928, #3857).
- The native cube test now checks the projected rectangle edges and proves the
  check fails when the sketch is hidden (#3969); see
  `receipts/2026-11-15-rectangle-visibility/`.

- Viewport orientation: all 20 pinned scenarios passed on the merged revision
  (#3961). See `receipts/2026-11-15-orientation-verified/`. The run uses
  frame-loop input; OS-level input and the full Go baseline remain open.

- Document opening: the native scenario that activates an already open part
  passed all nine steps at `8aa822fad` (#3889); see
  `receipts/2026-11-15-open-existing-refresh/`. The receipt is now stale after
  the Clear change. Input is frame-loop scripted.
- Document opening: Express open and Load Full are in `rust-work` (#3949).
  The pinned journey passes 13 of 14 steps; Finish and Simplify need the
  scenario clarification recorded in #3963. Saved views require matching source,
  mesh quality and producer identity. Cache creation is unavailable on macOS
  and Windows until the running executable can be verified (#3964). See
  `receipts/2026-11-15-express-contract/` for the diagnostic run.
- Document creation: the earlier diagnostic passed 28 of 39 scenarios; its
  receipt is historical. The updated feature is capture-host only, and five
  cases are also capture-only. The portable gate cannot claim these as passes;
  see #3899, #3951, and #3952.
- Document tabs: the diagnostic passed 9 of 38 scenarios (#3900). Two context
  menu scenarios passed on the merged revision; see
  `receipts/2026-11-15-tab-context-verified/`. Two Home menu scenarios need
  source clarification in their `BLOCKERS.md`. The keyboard scenario receipts
  predate later source changes (#3950).
- Quick access: the diagnostic passed 11 of 12 scenarios. Clear can open
  before selection and can clear multiple browser-selected bodies in one edit
  (#3962). Viewport multi-selection, faces, and occurrences remain open; see
  `receipts/2026-11-15-clear-multiple-bodies/`. Clear has confirmation, undo and
  redo checks in `receipts/2026-11-15-clear-appearance/`; its wider selection
  workflow remains open (#3962). The recorded profile remains incomplete
  (#3955).
- Body properties: all six pinned scenarios passed with checked setup, list
  scrolling and exact physical-value rows (#3954). Receipts become stale after
  relevant source changes; see `receipts/2026-11-15-body-properties-verified/`.
- Sketch ribbon: inactive-sketch setup was unbound (#3956). The source's
  outside-edit Create availability conflicts with the Go gate; the scenario
  owner has an actionable blocker.
- Ribbon appearance: 16 of 18 pinned scenarios passed; the Team Web fixture
  remains missing (#3957). See `receipts/2026-11-15-ribbon-appearance/`.
- Two-point rectangle: nine of ten scenarios passed in a configured diagnostic
  run. Scenario 10 needs the recorded fixture to assess its transient count
  (#3953). See the scenario owner's `BLOCKERS.md`.
- Martensite closest-point behavior is tracked separately in
  [Martensite #406](https://github.com/Oblikovati/Martensite/issues/406).

## Evidence and history

`behavior-ledger.json` and its generated report are the current counting source.
`receipts/` contains native reports, captures and diagnostic runs. A failed or
instrumented run is not acceptance evidence. The previous detailed status is
preserved in [STATUS-history-2026-11-15.md](STATUS-history-2026-11-15.md); its
old pass claims are historical and can be stale.

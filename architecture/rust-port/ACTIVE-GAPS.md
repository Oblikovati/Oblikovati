<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Rust port: detailed gaps and historical observations

The reports below describe individual runs. They are not current verified
acceptance evidence. The build-provenance gap in #3978 applies to every native
BDD receipt. Check [STATUS.md](STATUS.md) and the ledger before using counts.


- Point cancellation: point-11 requires a recorded sketch that was not supplied.
  The scenario owner's `features/sketch/point/BLOCKERS.md` requests the sketch
  or the properties of an acceptable substitute (#3887). The sizing report is
  in `receipts/2026-11-15-point11-sizing/` and is not acceptance evidence.
- Extrusion undo and redo: the native diagnostic passed three of four scenario
  steps. Named edge lengths are available through the typed API; selecting and
  measuring two edges in the viewport remains open (#3967). See
  `receipts/2026-11-15-extrusion-history-diagnostic/`. General analytic curve
  picking is blocked by missing interior extrema in
  [Martensite #462](https://github.com/Oblikovati/Martensite/issues/462).
- Extrusion entry: the new metric part and origin-plane selection scenario
  and the following preview, cancel/reopen and zero distance recovery scenarios
  passed on the current pin (#3927, #3931, #3929); their receipts are now
  stale after the offset-plane change. See
  `receipts/2026-11-15-extrusion-r4-current-1/` through `-4/`. The solve
  precision certificate remains open (#3928); see
  `solver-precision-investigation.md`.
- Rectangle zero-first-size: the corrected portable scenario passed eight
  native steps after the status fix; its receipt is now stale. See
  `receipts/2026-11-15-rectangle-status-current/`. OS input and intermediate
  captures are in `receipts/2026-11-15-rectangle-zero-corrected/`. The raw
  status message was fixed and tested in #3975; see
  `receipts/2026-11-15-zero-size-status/`.
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
- Projection fixture: the native rectangular source scenario passed on the
  current pin (#3976); its receipt is now stale. The offset-plane scenario
  passed in an earlier run (#3977); see `receipts/2026-11-15-offset-plane-scenario/`.
  The chosen 10 mm square is a fixture; the source did not record its size.
  OS input and MCPBridge evidence is in `receipts/2026-11-15-live-offset-plane/`.
- The native cube test now checks the projected rectangle edges and proves the
  check fails when the sketch is hidden (#3969); see
  `receipts/2026-11-15-rectangle-visibility/`.

- Viewport orientation: all 20 pinned scenarios passed on the merged revision
  (#3961). See `receipts/2026-11-15-orientation-verified/`. The run uses
  frame-loop input; OS-level input and the full Go baseline remain open.

- Document opening: the native scenario that activates an already open part
  passed all nine steps on the current source pin (#3889); its receipt is now
  stale. See
  `receipts/2026-11-15-open-existing-current-pin/`. Input is frame-loop scripted.
- Document saving: seven basic save, reopen, cancel, and overwrite scenarios
  passed on the current source pin (#3888); their receipts are now stale. See
  `receipts/2026-11-15-save-r4-current-1/` through `-7/`. Input is frame-loop scripted.
- Document opening: Express open and Load Full are in `rust-work` (#3949).
  The pinned journey passes 13 of 14 steps; Finish and Simplify still need
  implementation (#3963). The scenario owner clarified that
  this step checks visibility and enabled state only; it does not invoke the
  commands. Saved views require matching source,
  mesh quality and producer identity. Cache creation is unavailable on macOS
  and Windows until the running executable can be verified (#3964). See
  `receipts/2026-11-15-express-contract/` for the diagnostic run.
- Document creation: the earlier diagnostic passed 28 of 39 scenarios; its
  receipt is historical. The updated feature is capture-host only, and five
  cases are also capture-only. The portable gate cannot claim these as passes;
  see #3899, #3951, and #3952.
- Document tabs: eight R4 tab and close scenarios passed on the current source
  pin (#3900); their receipts are now stale. See
  `receipts/2026-11-15-tabs-r4-current-1/` through `-8/`. The earlier full-file
  diagnostic passed 9 of 38 scenarios. The scenario owner clarified that the
  Home menu cases inspect labels and dismiss the menu; keyboard receipts are
  stale (#3950). Three clarified Home-tab and window-list scenarios passed
  a later diagnostic; see `receipts/2026-11-15-tabs-clarified/`.
- Quick access: the diagnostic passed 11 of 12 scenarios. Clear can open
  before selection and can clear multiple browser-selected bodies in one edit
  (#3962). Viewport multi-selection, faces, and occurrences remain open; see
  `receipts/2026-11-15-clear-multiple-bodies/`. Clear has confirmation, undo and
  redo checks in `receipts/2026-11-15-clear-appearance/`; its wider selection
  workflow remains open (#3962). The recorded profile remains incomplete
  (#3955). The scenario owner clarified that the menu-inspection case does
  not require a particular toolbar profile.
- Body properties: all six pinned scenarios passed with checked setup, list
  scrolling and exact physical-value rows (#3954). Receipts become stale after
  relevant source changes; see `receipts/2026-11-15-body-properties-verified/`.
- Sketch ribbon: inactive-sketch setup was unbound (#3956). The scenario owner
  clarified that outside-edit cases check enabled states for one part and one
  sketch; compare that observation with the Go gate before changing behavior.
- Ribbon appearance: 16 of 18 pinned scenarios passed in an earlier run (#3957).
  The scenario owner clarified the Team Web and reset starting states and that
  the test checks panel visibility, not command destinations. Reassess the two
  remaining cases against the clarified fixture notes. See
  `receipts/2026-11-15-ribbon-appearance/`.
- Two-point rectangle: the scenario owner clarified that scenario 10 uses an
  empty metric sketch and does not require recorded whole-sketch counts. Its
  corrected eight-step native run passed, but its receipt is stale (#3953).
- Martensite closest-point behavior is tracked separately in
  [Martensite #406](https://github.com/Oblikovati/Martensite/issues/406).


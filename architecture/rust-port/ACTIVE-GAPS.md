<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Rust port: detailed gaps and historical observations

The reports below describe individual runs. They are not current verified
acceptance evidence. The build-provenance gap in #3978 applies to every native
BDD receipt. Check [STATUS.md](STATUS.md) and the ledger before using counts.

- Point cancellation: point-11 passed a focused native run and the full Point
  feature run (#3887). The latest full run passed 8 of 9 scenarios; point-10
  needs the recorded sketch fixture. See `receipts/2026-11-15-point-fixture-full/`.
  Captures remain historical (#3978).
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
  A later live OS-input run shows the plane overlay and undo/redo (#3981); see
  `receipts/2026-11-15-live-workplane-overlay/`.
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
  stale (#3950). A later XTest and MCPBridge check covered neighboring tabs
  and Home wrap; it is manual evidence, not a scenario receipt. See
  `receipts/2026-11-15-live-tab-keyboard/`. Three clarified Home-tab and window-list scenarios passed
  a later diagnostic; see `receipts/2026-11-15-tabs-clarified/`.
  Hide/restore and the hidden-tab window list also passed after the list fixture
  was corrected to create its documents before opening the dialog (#3900); see
  `receipts/2026-11-15-tabs-visibility-corrected/`.
  A full diagnostic before those corrections passed 18 of 38 scenarios; 20
  stopped at undefined steps. See `receipts/2026-11-15-tabs-full-current/`.
- Quick access: the diagnostic passed 11 of 12 scenarios. Clear can open
  before selection and can clear multiple browser-selected bodies in one edit
  (#3962). Viewport body multi-selection now passes frame-loop tests; named
  faces, occurrences, and OS-level input remain open. See
  `implementation-viewport-body-toggle.md` and
  `receipts/2026-11-15-clear-multiple-bodies/`. Clear has confirmation, undo and
  redo checks in `receipts/2026-11-15-clear-appearance/`; its wider selection
  workflow remains open (#3962). The recorded profile remains incomplete
  (#3955). The scenario owner clarified that the menu-inspection case does
  not require a particular toolbar profile.
- Body properties: all six pinned scenarios passed with checked setup, list
  scrolling and exact physical-value rows (#3954). Receipts become stale after
  relevant source changes; see `receipts/2026-11-15-body-properties-verified/`.
- Sketch ribbon: the clarified feature ran with one of three scenarios passing
  (#3956). Both outside-edit cases fail because Line is disabled; their later
  command checks did not run. The existing-sketch edit, Line prompt and
  tooltip, Finish and workspace restoration passed. See
  `receipts/2026-11-15-sketch-availability-clarified/`. This is diagnostic
  evidence, not verified acceptance (#3978).
- Ribbon appearance: the clarified pinned feature ran with 15 of 18 scenarios
  passing (#3957). Three stop at the required Team Web visibility setup; the
  Tools panel list has no Team Web or Tutorials entry. The setup now operates
  the real visibility controls and does not assume the captured state. See
  `receipts/2026-11-15-ribbon-clarified/`. This is diagnostic evidence, not a
  verified acceptance result (#3978).
- Two-point rectangle: the scenario owner clarified that scenario 10 uses an
  empty metric sketch and does not require recorded whole-sketch counts. Its
  corrected eight-step native run passed, but its receipt is stale (#3953).
- Martensite closest-point behavior is tracked separately in
  [Martensite #406](https://github.com/Oblikovati/Martensite/issues/406).

- Clear Appearance: multiple exact face overrides can now be selected and
  cleared in one edit. Owner and native-window tests cover selection, Cancel,
  undo, redo and reopen. An OS-input and MCPBridge check also covered two
  selected faces; see `receipts/2026-11-15-live-clear-faces/`. Mixed body and
  face selections now clear together; see
  `receipts/2026-11-15-live-clear-mixed/`. Assembly occurrences remain open (#3962).

- iProperties: the desktop editor now creates and edits supported typed values,
  rejects invalid input and type changes, and keeps parameter-owned properties
  read-only (#3958). A native test covers the types and save/reopen; an OS-input
  integer check is in `receipts/2026-11-15-live-typed-iproperties/`. The full
  gate and current BDD acceptance remain open.

- Assembly occurrence appearance overrides need durable, document-qualified
  occurrence identities (#3986). The Go reference drops unknown assembly
  fields and cannot safely round-trip the Rust schema-3 document shape. A Go
  fail-closed guard is isolated at `work/occurrence-identity-go-3986`; durable
  identity and cross-host migration remain unimplemented.

# Part Modify Slot Implementation

## Overview

Part Modify **Slot** is implemented end-to-end across `oblikovati-model`,
`oblikovati-martensite`, `oblikovati-automation`, `oblikovati-gui`, and
`oblikovati-head` in Part program version `53`, supporting rounded and
rectangular slots, rotation, distance/through-all/to-surface terminations,
counterbore/spotface/countersink seats, fastener clearance fits, end-mill
options, center marks, `iMate`, `Extend Start`, and browser `Edit Feature`.

## Architecture

- **Model (`crates/model/src/feature/slot.rs`, `crates/model/src/history/hole_create.rs`)**:
  - Defines `SlotShape` (`Rounded`, `Rectangular`), `SlotSeat` (`None`,
    `Counterbore`, `Spotface`, `Countersink`), `ResolvedSlotSeat`,
    `SlotEndMill` (`Flat`, `Point { angle_deg }`), `SlotCenterMarks` (`None`,
    `Center`, `EndCenters`, `All`), `SlotRecipe`, `SlotDefinition`, and
    `ResolvedSlot`.
  - Enforces positive finite dimensions, `length >= width`,
    `seat_width > width`, and open `(0, 180)` angle bounds.
  - `DocumentSnapshot::create_slot` and `DocumentSnapshot::edit_slot` provide
    transactional creation and in-place editing.
- **Martensite Adapter (`crates/martensite/src/prism/named/blend/slot.rs`, `solid/slot_evaluation.rs`)**:
  - Builds exact rounded stadium (`BRepBuilderAPIMakeEdge::from_circ_points` +
    lines) and rectangular (`BRepBuilderAPIMakePolygon`) cut prisms via
    `BRepPrimAPIMakePrism::new` rotated about the placement face normal.
  - Supports stepped `Counterbore`/`Spotface` prisms and lofted `Countersink`
    frustums via `BRepOffsetAPIThruSections::new(true, true, tol)`.
  - `NamedSolid::slot_preview_triangles` tessellates the cut tool solid for the
    live red Cut preview.
- **Automation & Desktop Owner (`crates/automation/src/slot.rs`, `crates/head/src/owner_slot*.rs`)**:
  - `prepare_slot_definition`, `build_slot_definition`, and `replace_slot`
    share validation between preview and commit.
  - `owner_slot.rs` provides planar face picking, live red cut preview,
    `browser.slot_edits` hydration, and transactional create/edit commits.
- **GUI (`crates/gui/src/slot_dialog/`, `crates/gui/src/ribbon_panel_additional/commands.rs`)**:
  - Exposed from the `3D Model > Modify` additional command area (`Slot`),
    available even on an empty Part (`blank-metric-part.opd`) while keeping
    `OK` disabled until a valid face and target solid exist.
  - v1.1 `280px` dockable Properties panel with `Input Geometry`, `Shape`,
    `Hole Type` (Simple / Clearance with fastener fit lookup), `Seat`,
    `Termination` (`Distance`, `Through All`, `To Surface` consuming Origin
    planes), `End Mill` (`Flat` / `Point` defaulting to `90°`), `Center Marks`,
    `iMate`, `Extend Start`, live plan + cross-section diagram, and inline red
    validation indicators.

## Verification

- `oblikovati-model`: `part_features` (96/96).
- `oblikovati-martensite`: `sketch_prism slot_replay` exact analytic volume
  verification across Rounded, Rectangular 45° rotated, `ToSurface(XY)`,
  Counterbore, and Countersink slots.
- `oblikovati-gui`: full library test suite.
- `oblikovati-head`: all **8/8 Wayland BDD scenarios (66/66 steps)** in
  `features/solid-model/slot/slot.feature`.

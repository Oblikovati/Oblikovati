# Part Modify Hole Implementation (#4093)

## Overview

Part Modify **Hole** is implemented end-to-end across `oblikovati-model`,
`oblikovati-martensite`, `oblikovati-automation`, `oblikovati-gui`, and
`oblikovati-head`, preserving byte-compatibility with existing v1 Hole fixtures
while supporting full v2 Hole authoring and editing in Part program version `52`.

## Architecture

- **Model (`crates/model/src/feature/hole/`)**:
  - `types.rs` defines `HoleKind` (`Simple`, `Clearance`, `Tapped`,
    `TaperTapped`), `HoleOriginPlane` (`Xy`, `Xz`, `Yz`),
    `HoleTerminationSurface`, `HoleSeat` (`None`, `Counterbore`, `Spotface`,
    `Countersink`), `ResolvedHoleSeat`, `HoleDrillPoint` (`Flat`, `Angle`),
    `HoleFastenerFit`, `HoleFastenerSpec`, `HoleThreadSpec`, and `HoleOptions`.
  - Validation enforces positive finite diameters/depths, `seat_diameter >
    bore_diameter` for stepped/conical seats, `(0, 180)` angle bounds, and
    forbids seats on `TaperTapped` holes.
  - `DocumentSnapshot::edit_hole` enables in-place transactional feature edits.
- **Martensite Adapter (`crates/martensite/src/prism/named/`)**:
  - Plain flat bores continue using `BRepPrimAPIMakeCylinder`.
  - Seated (`Counterbore`, `Spotface`, `Countersink`) and angled-drill-point
    holes construct a single revolved solid via `BRepPrimAPIMakeRevol::full` and
    `BRepSweepRevol::full` from the closed meridian polygon, avoiding internal
    tool-tool coplanar seams while naming the conical apex degenerate edge
    (`hole-tip-apex`) alongside lateral and step faces.
  - `NamedSolid::hole_preview_triangles` tessellates the exact cut tool solid
    for the viewport red Cut preview, including flipped-direction and
    `Extend Start` clearance previews.
- **Automation & Desktop Owner (`crates/automation/src/hole.rs`,
  `crates/head/src/owner_hole*.rs`)**:
  - `prepare_hole_definition`, `build_hole_definition`, and `replace_hole` share
    validation between preview and commit.
  - `owner_hole_pick.rs` snaps clicks near visible unconsumed sketch circles to
    their 3D center on the host planar face and otherwise selects planar faces.
  - `browser.hole_edits` hydrates `Edit Feature` from saved v1/v2 recipes.
- **GUI (`crates/gui/src/hole_dialog/`)**:
  - v1.1 dockable Properties shell (`280px` upper-left at `14px` inset, Teal
    selection pills, `40/60` `1px`-bordered `JetBrains Mono` numeric inputs,
    pinned `44px` footer with `OK`, `Cancel`, and `+`).
  - Shared `SessionPresets` bar with `Sort Order` (`Recently Used`,
    `Date Created`, `Alphanumeric`, `Last Modified`) and `New Hole Default`
    (`Last Used`, `Use Current Preset`, `No Preset`).
  - Live vector cross-section diagram (`diagram.rs`) reflecting seat type,
    conical angle, and flat/angled drill point.

## Verification

- `oblikovati-model`: `hole_definition` (6/6) and `part_features` (96/96).
- `oblikovati-martensite`: `sketch_prism hole` (25/25), including `hole_v2`
  exact analytic volume checks for `ToSurface`, `Counterbore`, `Countersink`,
  `Spotface + Extend Start`, and `118°` angled drill point.
- `oblikovati-persistence`: `package_hole` (1/1).
- `oblikovati-automation`: `hole` unit tests (4/4).
- `oblikovati-gui`: full library suite (850/850).
- `oblikovati-head`: `hole_scene` (4/4) and all **14/14 Wayland BDD scenarios
  (119/119 steps)** in `features/solid-model/hole/hole.feature`.

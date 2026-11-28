# Part Modify Chamfer Implementation

## Overview

Part Modify **Chamfer** v2 is implemented end-to-end across `oblikovati-model`,
`oblikovati-martensite`, `oblikovati-automation`, `oblikovati-gui`, and
`oblikovati-head` in Part program version `55`, upgrading equal-distance
named-edge chamfers to support `Distance & Angle`, `Two Distances` with
direction reversal, partial-edge chamfers with `Set Driven Dimension` (`To
Start`, `Chamfer`, `To End`), `Edge Chain` and `Setback` options, `Apply`
continuation, inline zero-distance red validation, `"Create Chamfer"` failure
recovery with `Edit`, and upper-left viewport-clamped Properties layout while
keeping v1 recipes byte-identical.

## Architecture

- **Model (`crates/model/src/feature/chamfer_options.rs`, `chamfer.rs`, `history/fillet_create.rs`)**:
  - Adds `ChamferMode` (`EqualDistance`, `DistanceAndAngle`, `TwoDistances`),
    `ChamferDrivenDimension` (`ToStart`, `Chamfer`, `ToEnd`),
    `ChamferPartialSpec`, `ChamferOptions`, and `ResolvedChamfer`.
  - Encodes `ChamferRecipe` v1 (exact legacy format when
    `options.is_default()`) and v2 in Part program version `55`, plus
    `DocumentSnapshot::edit_chamfer` for transactional in-place edits.
- **Martensite Adapter (`crates/martensite/src/prism/named/blend/chamfer_v2.rs`, `solid/blend_evaluation.rs`)**:
  - Executes `EqualDistance` (`add_with_dist`), `DistanceAndAngle`
    (`add_dist_angle` with `add_with_dists` fallback), and `TwoDistances`
    (`add_with_dists` on the reference face) via `BRepFilletAPIMakeChamfer`.
  - Executes partial-edge chamfers via exact analytical triangular cut wedges
    (`BRepBuilderAPIMakePolygon` + `BRepPrimAPIMakePrism` + `PrismBoolean::Cut`)
    preserving exact `0.5 * d1 * d2 * length` bevel removal and full topology
    lineage, plus `NamedSolid::edge_span` for endpoint and length queries.
- **Automation & Desktop Owner (`crates/automation/src/chamfer_v2.rs`, `crates/head/src/owner_chamfer_v2*.rs`)**:
  - `build_chamfer_v2_definition`, `prepare_chamfer_v2`, and `replace_chamfer`
    share validation between preview and commit.
  - `owner_chamfer_v2.rs` handles reference-face and partial-edge viewport
    picking, live replacement preview, `Apply` keep-open continuation,
    `browser.chamfer_v2_edits` hydration, and transactional create/edit commits.
- **GUI (`crates/gui/src/chamfer_panel/`, `blend_dialog.rs`, `blend_panel.rs`)**:
  - Dockable `Chamfer` Properties panel anchored upper-left (`14px` inset) clear
    of the ViewCube and Navigation Bar, tightly content-fitted to the active
    `Chamfer` or `Partial` tab, clamped on window resize, with red zero-distance
    validation disabling `OK`/`Apply`, `Apply` continuation, and `"Create
    Chamfer"` failure recovery (`Edit`).

## Verification

- `oblikovati-model`: `part_features` (96/96).
- `oblikovati-martensite`: `sketch_prism chamfer_v2` exact analytic volume
  verification across `Distance & Angle`, `Two Distances`, and `Partial` edges.
- `oblikovati-gui`: full library test suite including `blend_dialog` (6/6).
- `oblikovati-head`: all **10/10 Wayland BDD scenarios (93/93 steps)** in
  `features/solid-model/chamfer/chamfer.feature`.

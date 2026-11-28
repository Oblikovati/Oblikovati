# Part Modify Fillet Implementation

## Overview

Part Modify **Fillet** v2 is implemented end-to-end across `oblikovati-model`,
`oblikovati-martensite`, `oblikovati-automation`, `oblikovati-gui`, and
`oblikovati-head` in Part program version `54`, upgrading constant-radius
named-edge fillets to support variable-radius stations, `Smooth Radius
Transition`, `Tangent` and `Smooth G2` continuity, shared-vertex corner
setbacks (`Minimal` and per-edge distances), `Edge` / `Loop` / `Feature` /
`Solid` selection modes, collapsible `Advanced Properties`, and partial-blend
recovery via the `Create Fillet` modal while keeping v1 recipes byte-identical.

## Architecture

- **Model (`crates/model/src/feature/fillet_options.rs`, `fillet.rs`, `history/fillet_create.rs`)**:
  - Adds `FilletMode` (`Constant`, `Variable`, `Setbacks`),
    `FilletContinuity` (`Tangent`, `SmoothG2`), `FilletEdgeSet`,
    `FilletVariablePoint`, `FilletCornerSetback`, `FilletAdvancedOptions`,
    `FilletOptions`, and `ResolvedFillet`.
  - Encodes `FilletRecipe` v1 (exact legacy format when `options.is_default()`)
    and v2 in Part program version `54`, plus `DocumentSnapshot::edit_fillet`
    for transactional in-place edits.
- **Martensite Adapter (`crates/martensite/src/prism/named/blend/fillet_v2.rs`, `fillet_topology.rs`)**:
  - Executes multi-edge-set constant radii, variable-radius station laws
    (`add_with_pairs` with closed-seam fallback), and `Smooth G2`
    (`FilletShape::Polynomial`, `Continuity::G2`) via `BRepFilletAPIMakeFillet`.
  - Exposes `edge_reference_keys`, `shared_fillet_vertices`, `face_loop_edges`,
    `solid_fillet_edge_groups`, `probe_edge_fillet`, and
    `partition_blendable_sets` on `NamedSolid` for topology-driven selection
    modes and partial-blend partitioning.
- **Automation & Desktop Owner (`crates/automation/src/fillet_v2.rs`, `crates/head/src/owner_fillet_v2*.rs`)**:
  - `build_fillet_v2_definition`, `prepare_fillet_v2`, and `replace_fillet`
    share validation across preview and commit.
  - `owner_fillet_v2.rs` and `owner_fillet_v2_pick.rs` handle `Loop`,
    `Feature`, `Solid`, and shared-vertex viewport picking, partial-blend
    probing, live replacement preview, `browser.fillet_v2_edits` hydration,
    and transactional create/edit commits.
- **GUI (`crates/gui/src/fillet_panel/`, `blend_dialog.rs`, `engine/draw_controls.rs`)**:
  - Upgrades the dockable `Fillet` Properties panel with the `Constant` /
    `Variable` / `Setbacks` palette, `Edge` / `Loop` / `Feature` / `Solid`
    selection controls, variable-radius position rows with `Smooth Radius
    Transition` and `Continuity` menu, `Corner Setbacks` with status-bar
    `"No vertex available"` reporting, `Minimal`, per-edge setback fields,
    collapsible `Advanced Properties`, inline red zero-radius validation, and
    the `"Create Fillet"` partial-blend resolution modal.

## Verification

- `oblikovati-model`: `fillet_definition` (6/6) and `part_features` (96/96).
- `oblikovati-martensite`: `sketch_prism fillet_v2` topology and blend tests.
- `oblikovati-gui`: full library test suite including `blend_dialog` (6/6).
- `oblikovati-head`: all **15/15 Wayland BDD scenarios (104/104 steps)** in
  `features/solid-model/fillet/fillet.feature`.

## Historical v1 Notes

The typed feature API exposes a Fillet radius as scalar zero. Radius edits
accept a positive length or a named length parameter. A private candidate is
replayed by the certified Fillet service before the edit enters history.
Invalid or unsupported input is refused without committing the candidate.
The desktop opens the same Fillet panel from a selected browser feature. It
previews an isolated candidate, cancels without an edit, and commits through
the typed API. The draft is tied to the document generation and is refused
after another edit. See `receipts/2026-11-17-fillet-ui-edit-4045/` for the
native input test and its limits.

The creation panel can open before selection when a visible solid is present.
It waits for a picked edge before preview or commit. The selected input edge
remains visible as a display overlay during the replacement preview. See
`receipts/2026-11-17-fillet-selected-preview-4045/`.

The live viewport feature scenarios exercise native edge selection, a radius
change, and creation, plus extrusion edit and cancel. See
`receipts/2026-11-17-viewport-fillet-bdd-4045/`.

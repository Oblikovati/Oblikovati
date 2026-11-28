# Surface Boundary Patch Implementation Evidence (`#3918`)

## Scope

End-to-end Rust port of **3D Model > Surface > Boundary Patch** covering all six
scenarios (`boundary-patch-01` through `boundary-patch-06`) in
`solid-model/boundary-patch/boundary-patch.feature`.

## Architecture & Version Slots

- **Program format**: `v70` (`MAX_PROGRAM_VERSION = 70`)
- **Source recipe format**: `v60` (`WorkPlaneDefinitionRecipe::BoundaryPatch`)
- **Model** (`crates/model/src/feature/boundary_patch.rs`,
  `crates/model/src/feature/part/boundary_patch_evaluation.rs`,
  `crates/model/src/history/boundary_patch_create.rs`):
  - `BoundaryCondition` (`Free` / G0, `Tangent` / G1, `Smooth` / G2) with
    non-negative finite `weight`.
  - `BoundaryCurveRef::SketchCurve { sketch, entity }`, `BoundaryEdgeSet`,
    `BoundaryPatchRecipe`, `BoundaryPatchDefinition`, `ResolvedBoundaryPatch`.
  - `DocumentSnapshot::{create_boundary_patch, edit_boundary_patch}` with
    automatic sketch consumption visibility hiding.
- **Martensite** (`crates/martensite/src/boundary_patch.rs`,
  `crates/martensite/src/boundary_patch_chain.rs`):
  - Exact planar B-rep face/shell construction via `BRepBuilderAPIMakeFace` +
    `BRepBuilderAPIMakePolygon` / `BRepBuilderAPIMakeEdge` wrapped in a
    `TopoDS_Shell` (`BlendNamedTopology::from_boundary_patch_shell`) so
    `is_surface() == true` and bodies appear under `Surface Bodies` as
    `Srf1`..`Srf5`.
  - Automatic closed-loop sketch edge chaining (`chain_sketch_boundary_loop`)
    and interior diagonal/chord rejection (`is_interior_diagonal_curve`).
- **Automation** (`crates/automation/src/boundary_patch.rs`):
  - `BoundaryPatchInputs`, `PreparedBoundaryPatch`,
    `prepare_boundary_patch_definition`, `build_boundary_patch_definition`,
    `replace_boundary_patch` (`Boundary Patch1`, `Boundary Patch2`, ...).
- **GUI** (`crates/gui/src/boundary_patch_dialog/`, `3D Model > Surface > Boundary Patch`):
  - `IconGlyph::BoundaryPatch` icon assets and `RibbonPanel::PartSurface`
    command button.
  - Dockable `Boundary Patch` Properties panel with `Automatic Edge Chain`,
    `Edges1`/`Edges2` edge-set list, 3-icon Condition selector (`G0`/`G1`/`G2`)
    and `Weight` field, `Guide Rails` (`1. Edge` indicator), `OK`/`Apply`/`Cancel`.
- **Desktop Owner & Preview** (`crates/head/src/owner_boundary_patch.rs`,
  `crates/head/src/owner_boundary_patch_seed.rs`):
  - Visible sketch curve ray picking with automatic loop chaining, single-edge
    selection, and interior diagonal rejection.
  - Translucent Surface Warm-Amber live preview (`[0.96, 0.62, 0.04, 0.42]`),
    `Edit Feature` cancel rollback, and Undo/Redo checkpoint integration.

## Verification

- `cargo test --locked -p oblikovati-model boundary_patch`
- `cargo test --locked -p oblikovati-martensite --test sketch_prism boundary_patch`
- `cargo clippy --locked --workspace --all-targets --all-features -- -D warnings`
- `cargo xtask docs && cargo xtask script-docs-check`
- Live native Wayland BDD (`OBK_BDD_AREA=solid-model OBK_BDD_FEATURE=boundary-patch`):
  all 6/6 scenarios (`boundary-patch-01`..`boundary-patch-06`) verified against
  the authentic fixtures `fixtures/cad/revolve/revolve-profile.opd` and
  `fixtures/cad/boundary-patch/six-solids-five-loops-candidate.opd`.

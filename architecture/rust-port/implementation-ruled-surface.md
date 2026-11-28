# Part Surface Ruled Surface Implementation Evidence (#3918)

## Scope

End-to-end Rust implementation of **3D Model > Surface > Ruled Surface** (`solid-model/ruled-surface/ruled-surface.feature`, scenarios `ruled-surface-01` through `ruled-surface-10`).

## Architecture Summary

- **Model (`crates/model`)**:
  - Added `RuledSurfaceType` (`Normal`, `Tangent`, `Sweep`), `RuledSurfaceVector` (`OriginX`, `OriginY`, `OriginZ`, `WorkAxis(FeatureId)`, `Edge { body, edge }`), `RuledSurfaceRecipe`, `RuledSurfaceDefinition`, and `ResolvedRuledSurface` in `crates/model/src/feature/ruled_surface.rs`.
  - Assigned disjoint program format slot **`v76`** (`MAX_PROGRAM_VERSION = 76`) and source format slot **`v66`** in `crates/model/src/feature/work_plane/collection/version.rs` and `crates/model/src/feature/work_plane/recipe.rs`.
  - Enforced strict recipe validation: strictly positive finite `distance > 0.0`, finite `angle_deg` in `[-180.0, 180.0]`, distinct kind-2 edges on the same document as `target_body`, and mandatory `vector` in `Sweep` mode.
  - Added non-destructive surface body evaluation (`crates/model/src/feature/part/ruled_surface_evaluation.rs`) and `DocumentSnapshot::{create_ruled_surface, edit_ruled_surface}` (`crates/model/src/history/ruled_surface_create.rs`).
- **Martensite (`crates/martensite`)**:
  - Implemented `NamedSolid::evaluate_ruled_surface_result`, `NamedSolid::boundary_edge_chain`, and `NamedSolid::ruled_surface_preview_arrow` in `crates/martensite/src/prism/named/blend/ruled_surface.rs`.
  - Constructed real B-rep ruled surface shells via `BRepPrimAPIMakePrism` on detached boundary edges for all three modes (`Normal`, `Tangent`, `Sweep`), with Rodrigues draft-angle rotation about the edge tangent and direction flip, wrapped in `BlendNamedTopology::from_ruled_surface_shell`.
  - Added replay tests in `crates/martensite/tests/sketch_prism/named/rebuild/ruled_surface_replay.rs` verifying exact analytic face areas, 4-edge boundary chaining, all three modes, `Angle` (`0`, `15`, `180` deg), `Flip` arrow reversal, and `v76`/`v66` recipe round-trip.
- **Automation (`crates/automation`)**:
  - Added `RuledSurfaceInputs`, `PreparedRuledSurface` (`Ruled Surface1` naming), `build_ruled_surface_definition`, `prepare_ruled_surface_definition`, and `replace_ruled_surface` in `crates/automation/src/ruled_surface.rs`.
- **GUI (`crates/gui`)**:
  - Registered `IconGlyph::RuledSurface` (`ruled-surface.svg`) and exposed `3D Model > Surface > Ruled Surface` (`RibbonPanel::PartSurface`).
  - Built the dockable `Ruled Surface` task dialog (`crates/gui/src/ruled_surface_dialog/{types.rs,mod.rs,controls.rs}`) with 3 type modes (`Normal`, `Tangent`, `Sweep`), `Edges` selector and `Edge1`..`Edge4` list, `Automatic Edge Chain`, `Vector` selector (accepting Origin axes from the Model Browser or linear edges), `Flip` toggle, 3D-projected direction arrow overlay, `Distance` with red text/border validation when `<= 0`, `Angle` (`-180..=180` deg), and `OK` / `Apply` / `Cancel` footer controls.
- **Desktop Owner (`crates/head`)**:
  - Added `crates/head/src/owner_ruled_surface.rs` and `owner_ruled_surface_seed.rs` with surface-preferred edge picking, boundary edge chaining, live additive surface preview, 3D direction arrow projection (`frames/gui_model_click.rs`), transactional `OK`/`Apply` commits, and `Edit Feature` hydration.

## Verification

- `cargo test --locked -p oblikovati-model ruled_surface`
- `cargo test --locked -p oblikovati-martensite ruled_surface`
- Live native Wayland BDD suite (`OBK_BDD_AREA=solid-model OBK_BDD_FEATURE=ruled-surface`): 10/10 scenarios (`ruled-surface-01`..`ruled-surface-10`).
- `cargo fmt --all -- --check`, `cargo clippy --locked --workspace --all-targets --all-features -- -D warnings`, `cargo xtask docs && cargo xtask script-docs-check`.

# Extend Surface Implementation Evidence (`#3918`)

## Scope

Full end-to-end delivery of **Part Surface Extend** (`solid-model/extend-surface/extend-surface.feature`, scenarios `extend-surface-01` through `extend-surface-06`, zero `@needs_fixture`).

## Architecture

### Model (`crates/model`)

- Added `crates/model/src/feature/extend_surface.rs` defining `ExtendSurfaceMethod` (`Extend` / `Stretch`), `ExtendSurfaceOriginPlane`, `ExtendSurfaceTermination` (`Face` / `WorkPlane` / `OriginPlane`), `ExtendSurfaceExtent` (`Distance(f64)` / `To(ExtendSurfaceTermination)`), `ExtendSurfaceRecipe`, `ExtendSurfaceDefinition`, `ResolvedExtendSurfaceExtent`, and `ResolvedExtendSurface`.
- Assigned disjoint Part program format version `71` (`MAX_PROGRAM_VERSION = 71`) and source recipe version `61`.
- Added `PartFeatureDefinition::ExtendSurface`, `crates/model/src/feature/part/extend_surface_evaluation.rs` (in-place surface body replacement preserving body lineage), and transactional `DocumentSnapshot::{create_extend_surface, edit_extend_surface}`.

### Martensite (`crates/martensite`)

- Added `crates/martensite/src/prism/named/blend/extend_surface.rs`, `extend_surface_geom.rs`, and `crates/martensite/src/prism/named/solid/extend_surface_evaluation.rs`.
- Implemented free boundary edge chaining (`NamedSolid::surface_boundary_edge_chain`) connecting open-shell boundary edges across shared endpoints.
- Implemented exact planar face half-plane boundary extension in both `Stretch` mode (single enlarged planar B-rep face) and `Extend` mode (original planar face plus contiguous extension strip face(s) in the shell), supporting `Distance` and `To` plane intersection, plus `BlendNamedTopology::from_extended_shell_preserving_body` and analytic `NamedSolid::surface_area_mm2`.

### Automation (`crates/automation`)

- Added `crates/automation/src/extend_surface.rs` (`ExtendSurfaceInputs`, `PreparedExtendSurface`, `build_extend_surface_definition`, `prepare_extend_surface_definition`, `replace_extend_surface`) generating sequential `Extend1` names.

### GUI (`crates/gui`)

- Registered `IconGlyph::ExtendSurface` backed by `cmd-model-extend-surface-32.svg` and `extend-surface.svg`.
- Exposed `3D Model > Surface > Extend` (`RibbonPanel::PartSurface`) on top, side, and floating ribbons, using per-panel collapsed caption widths on `3D Model` so narrow 1024 px ribbons keep `Sketch` expanded.
- Added the dockable `Extend Surface` Properties panel (`crates/gui/src/extend_surface_dialog/{mod,controls,types}.rs`) with `Edges` selector, `Edge Chain` checkbox (default checked), `Extents` (`Distance` default `5 mm` vs `To` termination selector replacing the distance field and disabling `OK` until supplied), `More` (`Method`: `Extend` vs `Stretch`), and preview-gated `OK` / `Cancel`.

### Desktop Head (`crates/head`)

- Added `crates/head/src/owner_extend_surface.rs`, `owner_extend_surface_pick.rs`, and `owner_extend_surface_seed.rs` for surface boundary edge ray picking, Edge Chain perimeter highlighting via `selected_edge_outlines`, translucent live surface extension preview (`PreviewKind::BlendInput`), transactional commit, and Model Browser `Edit Feature` cancel restoration.

## Verification

- `oblikovati-model` unit tests: validation, positive-distance enforcement, duplicate-edge rejection, and v61/v71 recipe round-trip.
- `oblikovati-martensite` replay tests (`extend_surface_replay.rs`): exact area `1000 -> 1200 mm²` at 5 mm (`Extend`, 2 faces), `1400 mm²` at 10 mm (`Stretch`, 1 face), `1750 mm²` for 4-edge `Edge Chain` at 5 mm, and `To` plane refusal on parallel plane.
- `oblikovati-gui` unit tests: dialog defaults and distance parsing.
- Live Wayland native BDD suite (`crates/head/tests/solid-model/extend-surface/{setup,steps}.rs`): all 6/6 scenarios (`extend-surface-01`..`06`) executed and passed on the native Wayland window.

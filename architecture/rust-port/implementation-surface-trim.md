# Surface Trim Implementation Evidence (#3918)

## Scope

Implements the `3D Model > Surface > Trim` (`Trim Surface`) feature end-to-end across
model persistence, Martensite B-rep surface splitting, automation, GUI, desktop owner
picking/preview, and live Wayland BDD scenarios (`solid-model/surface-trim/surface-trim.feature`).

## Version Slot

- Part program format: `v75` (`MAX_PROGRAM_VERSION = 75`)
- Persisted recipe source format: `v65` (`WorkPlaneDefinitionRecipe::SurfaceTrim`)

## Architecture Summary

- **Model (`crates/model`)**:
  - `SurfaceTrimTool` (`OriginPlane`, `WorkPlane`, `SurfaceFace`), `SurfaceTrimSide` (`Positive`, `Negative`), `SurfaceTrimRecipe`, `SurfaceTrimDefinition` in `crates/model/src/feature/surface_trim.rs`.
  - Replay evaluation in `crates/model/src/feature/part/surface_trim_evaluation.rs` replacing the target surface body slot and preserving its primary body reference key.
  - Snapshot mutation helpers `create_surface_trim` / `edit_surface_trim` in `crates/model/src/history/surface_trim_create.rs`.
- **Martensite (`crates/martensite`)**:
  - Real B-rep surface splitting in `crates/martensite/src/prism/named/blend/surface_trim.rs` using `BRepAlgoAPISplitter`, analytic `BRepGProp::surface_properties` face area + centroid classification, `BRepCheckAnalyzer` validation, area conservation check, and strict area reduction check on the kept shell.
  - `NamedSolid::preview_surface_trim_regions` exposes both candidate halves and their analytic areas for interactive viewport preview and region switching.
  - `BlendNamedTopology::from_trimmed_shell_preserving_body` names the trimmed open shell while preserving primary surface body identity and lineage.
- **Automation (`crates/automation`)**:
  - `SurfaceTrimFeatureInputs`, `PreparedSurfaceTrim`, `build_surface_trim_definition`, `prepare_surface_trim_definition` (`Trim1`, `Trim2`, ...), and `replace_surface_trim` in `crates/automation/src/surface_trim.rs`.
- **GUI (`crates/gui`)**:
  - `3D Model > Surface > Trim` command button with `IconGlyph::TrimSurface` (`assets/icons/trim-surface.svg`).
  - Dockable `Trim Surface` dialog (`crates/gui/src/surface_trim_dialog/`) with `Cutting Tool` selector, `Remove` selector + adjacent `Flip` region-switch control, `OK` / `Apply` / `Cancel`, and Model Browser `Edit Feature` hydration.
- **Desktop Owner (`crates/head`)**:
  - `owner_surface_trim.rs` and `owner_surface_trim_pick.rs` handle origin/work plane & surface face tool picking, surface region ray-hit classification, translucent red removal-region preview highlighting (including pre-trim suppression evaluation during `Edit Feature`), transactional `OK`/`Apply`, and cancel rollback.

## Verification

- Unit test: `oblikovati_model::feature::surface_trim::tests::surface_trim_validates_tools_reads_and_program_v75_source_v65_round_trip`
- Martensite replay test: `surface_trim_splits_surface_reduces_area_and_supports_region_flip`
- Live Wayland BDD suite (`OBK_BDD_AREA=solid-model OBK_BDD_FEATURE=surface-trim`):
  - `surface-trim-01`: Select a cutting plane and removal region
  - `surface-trim-02`: Switch the removal region
  - `surface-trim-03`: Accept the selected trim
  - `surface-trim-04`: Apply a trim and cancel the next operation
  - `surface-trim-05`: Cancel a change to an existing trim

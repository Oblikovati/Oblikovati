# Part Modify Finish implementation (`#3918`)

## Scope

Implements the Part Modify **Finish** feature across model, Martensite,
automation, GUI, desktop owner, and native Wayland BDD scenarios
(`finish-01`..`finish-05`).

## Model (`crates/model`)

- Added `FinishType` (5 categories: `Appearance`, `MaterialCoating`,
  `HeatTreatment`, `SurfaceTexture`, `Paint`), `FinishRecipe`,
  `FinishDefinition`, and `ResolvedFinish` in `crates/model/src/feature/finish.rs`.
- Allocated Part program version `65` (`MAX_PROGRAM_VERSION = 65`) and
  source recipe version `55` for `WorkPlaneDefinitionRecipe::Finish`.
- Added `PartFeatureDefinition::Finish`, `PartFeatureFailure::FinishInput`,
  dedicated `finish_evaluation.rs`, `PartFeatureEvaluator::evaluate_finish`,
  and transactional `DocumentSnapshot::{create_finish, edit_finish}`.
- Attributed upstream body and face reads so suppressing an upstream body
  marks `Finish1` as `Blocked` (warning state) while preserving the authored
  settings and keeping the feature editable in the browser.

## Martensite (`crates/martensite`)

- Added exact analytic face area integration (`BRepGProp::surface_properties`)
  in `crates/martensite/src/prism/named/blend/finish.rs` (`compute_finish_area_mm2`
  and `NamedSolid::finish_area_mm2`), converting internal `cm²` to `mm²`
  (`40 × 25 mm` top planar face evaluates to exact `1000 mm²`; empty evaluates
  to `0.000 mm²`).
- Added non-destructive `evaluate_named_finish` in
  `crates/martensite/src/prism/named/solid/finish_evaluation.rs`, preserving
  solid volume and topology lineage while recording per-face linear RGBA
  appearance overrides (`finish_face_colors`) on `BlendNamedTopology`.
- Added unit and replay tests in
  `crates/martensite/tests/sketch_prism/named/rebuild/finish_replay.rs`
  verifying `1000 mm²` area, exclusion subtraction, volume preservation,
  per-face appearance attachment, recipe version round-trip (`v65`/`v55`),
  and `Blocked` health when the upstream body is suppressed.

## Automation, GUI, and Desktop Owner

- Added typed `FinishFeatureInputs`, `PreparedFinish`,
  `build_finish_definition`, `prepare_finish_definition` (`Finish1` naming),
  and `replace_finish` in `crates/automation/src/finish.rs`.
- Registered `IconGlyph::Finish` backed by `cmd-model-finish-32.svg` in
  `3D Model > Modify`.
- Built the `280px` dockable `Finish` Properties panel
  (`crates/gui/src/finish_dialog/`) with `Input Geometry` (`Include`,
  `Exclude`, live `Area`), `Behavior` (5 finish type buttons, `Process`
  catalog including `Zn12/Cn/T2` -> `Turquoise` / `12 micron`, `Hardness`
  and `Depth` on Type 3, `Barrel Finishing` on Type 4, and
  `Prime/Paint RAL 3018` / `Red` / `50 micron` on Type 5),
  `Advanced Properties` (`Export all Finish Parameters`), and gated `OK`.
- Added the collapsible `Finishes` Model Browser folder
  (`crates/gui/src/browser_finish.rs`) with warning badges for unavailable
  geometry and double-click `Edit Feature` hydration.
- Wired `owner_finish.rs`, viewport face picking, and per-face live preview
  and committed finish color rendering in `part_scene/face_display.rs`.

## Verification

- `oblikovati-model` unit tests (`finish_recipe_validates_faces_exclusions_and_round_trip`).
- `oblikovati-martensite` replay tests (`finish_replay`).
- Native Wayland BDD (`OBK_BDD_AREA=solid-model OBK_BDD_FEATURE=finish`):
  5/5 scenarios (`finish-01`..`finish-05`).
- Workspace formatting (`cargo fmt --all -- --check`), clippy (`-D warnings`),
  and docs checks (`cargo xtask docs`, `cargo xtask script-docs-check`).

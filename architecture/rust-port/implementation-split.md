# Part Modify — Split Implementation (`#3918`)

## Summary

End-to-end native Rust implementation of Part Modify **Split** covering both
**Split Face** (`All Faces` and explicit face subsets along a sketch curve or
plane) and **Split Solid** (`Keep Both`, `Keep Side A`, `Keep Side B` along a
work plane or origin plane) across model, Martensite geometry kernel,
automation, GUI, desktop owner, and native Wayland BDD tests.

## Architecture

### Model (`crates/model`)

- `crates/model/src/feature/split.rs`: defines `SplitMode` (`Faces`, `Solid`),
  `SplitKeepSide` (`Both`, `SideA`, `SideB`), `SplitTool` (`SketchCurve`,
  `OriginPlane`, `WorkPlane`), `SplitTarget` (`AllFaces`, `Faces`, `Solid`),
  `SplitRecipe`, `SplitDefinition`, and resolved evaluation inputs.
- `crates/model/src/feature/work_plane/recipe.rs` & `version.rs`: persists
  `WorkPlaneDefinitionRecipe::Split` at `source_version = 51` with
  `minimum_program_version = 61` (`MAX_PROGRAM_VERSION = 61`).
- `crates/model/src/feature/part/split_evaluation.rs`: resolves sketch curve
  tools (including face-placed sketches), origin/work plane tools, target body
  lineage, and multi-output body replacement via `body_lineage::replace_with_outputs`.
- `crates/model/src/history/hole_create.rs`: exposes `DocumentSnapshot::create_split`
  and `DocumentSnapshot::edit_split`.

### Geometry Kernel (`crates/martensite`)

- `crates/martensite/src/prism/named/blend/split_tool.rs`: constructs oversized
  planar or ruled cutting surfaces from work planes, origin planes, and sketch
  curves (`Line`, `Arc`, `Circle`), plus 1D section edges via `BRepAlgoAPISection`
  for explicit face-subset splits.
- `crates/martensite/src/prism/named/blend/split.rs`: runs `BRepAlgoAPISplitter`
  on the solid outer shell for `SplitMode::Faces` (`AllFaces` and face subsets)
  and on the solid body for `SplitMode::Solid` (`Both`, `SideA`, `SideB`),
  classifying halves along the cutting plane normal and validating solid health
  with `BRepCheckAnalyzer`.
- `crates/martensite/src/prism/named/solid/split_evaluation.rs`: preserves
  parent body reference lineage for single-body splits and assigns deterministic
  `"split-a"` / `"split-b"` derived body keys when `Keep Both` creates two
  distinct solid bodies.

### Automation, GUI & Desktop Owner

- `crates/automation/src/split.rs`: typed `SplitFeatureInputs`,
  `prepare_split_definition`, `build_split_definition`, and `replace_split`
  with default feature naming prefix `Split` (`Split1`, `Split2`, ...).
- `crates/gui/src/split_dialog/`: dockable `280px` Properties panel with
  `Input Geometry`, `Tool` selector pill (`Select Tool` / `1 Curve` / `1 Work Plane`)
  and `Clear`, `All Faces` checkbox, mode-switch button between `Faces` and `Solid`,
  three `Keep Side` options (`Keep Both`, `Keep Side A`, `Keep Side B`), and
  `OK` / `Cancel` / `+` footer.
- `crates/gui/src/icons/` & `ribbon_panel_additional/`: adds `IconGlyph::Split`
  (`cmd-model-split-32.svg`) and wires `Split` under `3D Model > Modify`.
- `crates/head/src/owner_split.rs`, `owner_split_pick.rs`, `owner_split_seed.rs`:
  handles viewport curve/plane/face/solid picking, browser plane/body picking,
  live replacement preview, pre-split face count recovery and descendant face
  remapping during edit, and atomic multi-body naming (`Solid4`, `Solid5`, ...).

## Verification

- `crates/model`: `feature::split` unit tests.
- `crates/martensite`: `sketch_prism::named::rebuild::split_replay` replay tests
  verifying exact volumes, face counts, and 1-body vs 2-body outputs.
- `crates/gui`: `split_dialog::tests` unit tests for mode switching, `All Faces`
  clearing, and browser plane reference parsing.
- `crates/head`: native Wayland BDD suite `crates/head/tests/solid-model/split/`
  covering all 8 scenarios (`split-01`..`split-07` and `split-05a`).

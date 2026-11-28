# Part Modify Direct Edit Implementation

## Overview

Part Modify **Direct Edit** (`#3918`) is implemented end-to-end across
`oblikovati-model`, `oblikovati-martensite`, `oblikovati-automation`,
`oblikovati-gui`, and `oblikovati-head` in Part program version `62` (recipe
source version `52`), delivering the floating in-canvas Direct Edit toolbar
(`Move`, `Size`, `Scale`, `Rotate`, `Delete`), `Faces`/`Solids` selection
filtering, viewport highlight outlines and mode-dependent manipulator gizmos,
`Uniform` and `Non Uniform` B-rep scaling, `Local`/`World` rotation coordinates,
`Snap Parallel` status bar prompting, green `+` apply with child operation rows
(`Direct Edit1` -> `Scale1`) in the Model Browser, and two-stage red `X`
selection-clear / toolbar-close cancellation.

## Architecture

- **Model (`crates/model/src/feature/direct_edit.rs`, `feature/part/direct_edit_evaluation.rs`, `history/hole_create.rs`)**:
  - Defines `DirectEditMode` (`Move`, `Size`, `Scale`, `Rotate`, `Delete`),
    `DirectEditSelectionFilter` (`Faces`, `Solids`), `DirectEditCoordinateMode`
    (`Local`, `World`), `DirectEditScaleMode` (`Uniform`, `NonUniform`),
    `DirectEditOperationKind`, `DirectEditOperation` (ordered named child
    operations such as `Scale1`, `Move1`), `DirectEditRecipe`,
    `DirectEditDefinition`, and `ResolvedDirectEdit`.
  - Uses disjoint Part program version `62` (`MAX_PROGRAM_VERSION = 62`) and
    `WorkPlaneDefinitionRecipe::DirectEdit` source version `52`.
  - `PartFeatureEvaluation::evaluate_direct_edit` resolves the target solid
    lineage slot, invokes `PartFeatureEvaluator::evaluate_direct_edit`,
    verifies body identity preservation, and replaces the solid slot in order.
  - Extracted `PrefixBackend` into `crates/model/src/feature/part/prefix_backend.rs`
    to keep both `prefix_replay.rs` and `evaluate.rs` strictly under 500 lines.
  - `DocumentSnapshot::create_direct_edit` and `edit_direct_edit` support
    transactional creation and in-place sub-operation updates.
- **Martensite Adapter (`crates/martensite/src/prism/named/blend/direct_edit.rs`, `blend/naming/direct_edit.rs`, `solid/direct_edit_evaluation.rs`)**:
  - Applies ordered B-rep transformations on detached copies while tracking
    every named sub-entity through `BlendHistoryRecord` and `BlendNamedTopology`:
    - `Scale` (`Uniform` or equal-axis `NonUniform`): analytic scaling about the
      body centroid via `Transform3::set_scale` and `BRepBuilderAPITransform`
      (preserving analytic cylindrical/planar surfaces; factor `2.0` scales
      volume by exact `8x`).
    - `Scale` (distinct-axis `NonUniform`): general affine 3D scaling via
      `GeneralTransform3` and `BRepBuilderAPIGTransform`.
    - `Move` and `Rotate`: rigid `Transform3::translation` and
      `Transform3::rotation` via `BRepBuilderAPITransform`.
  - Validates resulting geometry and analytic properties before naming.
- **Automation (`crates/automation/src/direct_edit.rs`)**:
  - `DirectEditFeatureInputs`, `PreparedDirectEdit`,
    `build_direct_edit_definition`, `prepare_direct_edit_definition` (default
    prefix `"Direct Edit"` -> `Direct Edit1`), and `replace_direct_edit`.
- **GUI (`crates/gui/src/direct_edit_dialog/`, `crates/gui/src/browser_direct_edit.rs`)**:
  - Exposed in `3D Model > Modify` (`Direct Edit`) with `IconGlyph::DirectEdit`
    using the `feature-icon.svg` artwork.
  - Floating in-canvas `"Direct Edit"` toolbar with five mode buttons,
    `Faces`/`Solids` filter popup (clearing active selection on switch),
    `Uniform`/`Non Uniform` scale fields (initializing `X`, `Y`, `Z` from the
    current uniform factor), `Local`/`World` coordinate menu, `Snap Parallel`
    status prompt, green `+` apply, and two-stage red `X` cancel.
  - Viewport manipulator overlay (`##direct-edit-manipulator`) rendering Move
    triad, Size arrows, and Rotate rings when geometry is selected.
  - Model Browser renders named child operation rows (`Scale1`, etc.) under
    `Direct Edit1`, allowing direct inspection and in-place editing.
- **Desktop Owner (`crates/head/src/owner_direct_edit*.rs`)**:
  - Wires `GuiDirectEditAction` through `work_hole.rs`, `owner_blend.rs`,
    `owner_browser.rs`, `owner_command_state.rs`, and `owner_viewport_selection.rs`.
  - Routes viewport picks according to the active `Faces`/`Solids` filter,
    highlights selected faces/solids via `selected_browser_outline`, evaluates
    live replacement previews for Scale operations, and persists sequential
    apply/edit operations into `Direct Edit1`.

## Verification

- `oblikovati-martensite`: `sketch_prism direct_edit_replay` (3/3 passed)
  verifying exact `8x` volume on uniform factor `2.0`, exact `3x` volume on
  non-uniform `[2.0, 1.5, 1.0]`, exact identity `[1.0, 1.0, 1.0]` edit, Move +
  Rotate centroid/volume invariants, and recipe `v52` / program `v62` round-trip.
- `oblikovati-gui`: `direct_edit_dialog::tests` (2/2 passed) and full `icons`
  rasterization suite (18/18 passed, 1 ignored).
- `oblikovati-head` native Wayland BDD (`features/solid-model/direct-edit/direct-edit.feature`):
  - **8/8 scenarios passed (57/57 steps)**: `direct-edit-01` through
    `direct-edit-08`.

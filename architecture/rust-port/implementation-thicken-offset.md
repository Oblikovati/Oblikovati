# Part Modify Thicken/Offset Implementation

## Overview

Part Modify **Thicken/Offset** (`#3918`) is implemented end-to-end across
`oblikovati-model`, `oblikovati-martensite`, `oblikovati-automation`,
`oblikovati-gui`, and `oblikovati-head` in Part program version `60` (recipe
source version `50`), supporting solid **Thicken** (`Thicken1`, with `Join`,
`Cut`, `Intersect`, and `New Solid` Booleans) and surface **Offset**
(`OffsetSrf1`, registering open surface shells in `Surface Bodies`), three
direction modes (inward `Dir 1`, outward `Dir 2`, and `Symmetric`),
`Automatic Face Chain`, `Automatic Blending`, `Advanced Properties`
approximation options (`Allow Approximation`, `Type`, `Tolerance`, and
specified percentage validation), browser `Edit Feature` hydration, live
preview, and `"Edit Thicken"` modal error recovery.

## Architecture

- **Model (`crates/model/src/feature/thicken_offset.rs`, `feature/part/thicken_offset_evaluation.rs`, `history/hole_create.rs`)**:
  - Defines `ThickenOutputMode` (`Solid`, `Surface`), `ThickenDirection`
    (`Negative`, `Positive`, `Symmetric`), `ThickenBoolean` (`Join`, `Cut`,
    `Intersect`, `NewSolid`), `ThickenApproximationType` (`Mean`,
    `NeverTooThin`, `NeverTooThick`), `ThickenApproximationTolerance`
    (`Optimized`, `Specified`), `ThickenOffsetRecipe`,
    `ThickenOffsetDefinition`, and `ResolvedThickenOffset`.
  - Validates same-document body and face references, non-empty unique face
    lists, strictly positive distances (`> 0.0`) for Solid Thicken, non-negative
    distances (`>= 0.0`, including `0 mm`) for Offset Surface, and positive
    percentage bounds when `Specified` tolerance is active.
  - `PartFeatureEvaluation::evaluate_thicken_offset` registers a new body for
    `Solid + NewSolid` and for `Surface` (`OffsetSrf`), and replaces the source
    solid slot in-place for `Solid + Join/Cut/Intersect` and `Automatic Blending`.
  - Uses disjoint Part program version `60` (`MAX_PROGRAM_VERSION = 60`) and
    `WorkPlaneDefinitionRecipe::ThickenOffset` source version `50`.
  - `DocumentSnapshot::create_thicken_offset` and
    `DocumentSnapshot::edit_thicken_offset` provide transactional creation and
    in-place editing.
- **Martensite Adapter (`crates/martensite/src/prism/named/blend/thicken_offset.rs`, `thicken_offset_chain.rs`, `solid/thicken_offset_evaluation.rs`)**:
  - `NamedSolid::tangent_face_chain` expands a clicked face along edge-adjacent
    G1 tangent-continuous curved faces (`BRepAdaptorCurve` + analytic surface
    normals), chaining cylindrical side faces to adjacent toroidal fillets
    (`2 Faces`).
  - Planar solid thicken uses `BRepPrimAPIMakePrism` along the oriented face
    normal (`+d`, `-d`, or `[-d/2, +d/2]`); curved faces use
    `BRepOffsetAPIMakeThickSolid::make_thick_solid_by_simple` (with exact
    coaxial rounded-cylinder analytic offset for `Automatic Blending` on
    tangent cylinder+torus pairs, avoiding BOP sliver meshing defects).
  - `Join`, `Cut`, and `Intersect` run through `NamedSolid::boolean_tools`
    (`BRepAlgoAPIBooleanOperation`), so outward `Cut` with zero interior overlap
    authentically declines with `EvaluationFailureKind::UnchangedCut`.
  - Offset Surface (`OffsetSrf`) constructs planar translated or
    `BRepOffsetAPIMakeOffsetShape::perform_by_simple` curved offset shells
    (including exact `0 mm` face copies) named via
    `BlendNamedTopology::from_offset_shell` (`is_surface() == true`).
- **Automation & Desktop Owner (`crates/automation/src/thicken_offset.rs`, `crates/head/src/owner_thicken_offset*.rs`)**:
  - `ThickenOffsetInputs`, `build_thicken_offset_definition`,
    `prepare_thicken_offset_definition` (`Thicken` / `OffsetSrf` naming), and
    `replace_thicken_offset` share validation between preview and commit.
  - `owner_thicken_offset.rs` handles `SelectionKind::Face` viewport picking
    with optional `Automatic Face Chain`, live additive/replacement preview,
    `browser.thicken_offset_edits` hydration, and transactional create/edit
    commits.
- **GUI (`crates/gui/src/thicken_offset_dialog/`, `crates/gui/src/ribbon_panel_additional/commands.rs`)**:
  - Exposed in `3D Model > Modify` (`Thicken/Offset`) and browser `Edit Feature`.
  - Dockable `280px` Properties panel that switches header to `OffsetSrf` and
    shows `Body Name` (`Srf1`) in Surface mode, hides Boolean controls and the
    third direction choice when `Automatic Blending` is checked, displays red
    validation indicators with disabled `OK` for invalid distances or
    locale-formatted specified tolerance percentages (`1,0 %`), and surfaces
    the `"Edit Thicken"` error modal (`Details`, `Edit`, `Cancel`) on failed
    Boolean edits.

## Verification

- `oblikovati-model`: `feature::thicken_offset::tests::thicken_offset_validates_distance_rules_per_output_mode` (pass).
- `oblikovati-martensite`: `sketch_prism thicken_offset_replay` (3/3 pass):
  - Exact analytic volumes for New Solid thicken (`6.0 cm³`, 2 solids), Join
    thicken (`30.0 cm³`, 1 solid), and authentic outward Cut failure.
  - `0 mm` (`z = 2.0 cm`) and `1 mm` (`z = 2.1 cm`) Offset Surface bodies with
    `is_surface() == true`.
  - Coaxial rounded-cylinder (`Cylinder + Torus`) 2-face tangent chain and
    `Automatic Blending` thicken with clean display meshing.
- `oblikovati-gui`: full `--lib` suite including `thicken_offset_dialog::tests` (pass).
- `oblikovati-head` Wayland BDD (`features/solid-model/thicken-offset/thicken-offset.feature`):
  - **9/9 scenarios pass (`thicken-offset-01` through `thicken-offset-09`)**,
    including authentic `"Edit Thicken"` Cut failure recovery in
    `thicken-offset-05` and 2-face curved tangent chaining + `Automatic Blending`
    in `thicken-offset-09`.

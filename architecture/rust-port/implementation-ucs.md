# Work Features — User Coordinate System (`#3918`)

Implements Part Work Features **User Coordinate System (UCS)** end-to-end across
`crates/model`, `crates/automation`, `crates/gui`, and `crates/head`.

## Architecture and Persistence

- **Model (`crates/model/src/feature/ucs.rs`, `crates/model/src/history/ucs_create.rs`)**:
  - `UcsRecipe` (`origin`, `x_dir`, `y_dir`, optional `vertex_ref`, `visible`),
    `UcsDefinition` (validating finite coordinates, Gram-Schmidt orthonormalization,
    and right-handed `z_dir = x_dir x y_dir`), and `ResolvedUcs` deriving the
    seven child work features (`YZ Plane`, `XZ Plane`, `XY Plane`, `X Axis`,
    `Y Axis`, `Z Axis`, and `Center Point`).
  - Disjoint persistence slot in `WorkPlaneDefinitionRecipe::Ucs`:
    `source_version = 59`, `minimum_program_version = 69`,
    `MAX_PROGRAM_VERSION = 69`.
  - Integrated into `PartFeatureDefinition::Ucs`, `WorkGeometryResult::Ucs`,
    and `WorkGeometryEvaluation::ucs` / `ucs_frames`.
  - `DocumentSnapshot::create_ucs`, `DocumentSnapshot::edit_ucs`,
    `DocumentSnapshot::set_ucs_visible`, and `DocumentSnapshot::next_ucs_name`
    with default prefix `UCS` (`UCS1`, `UCS2`, ...).

- **GUI and Viewport Triad (`crates/gui/src/ucs_tool/`, `crates/gui/src/browser_ucs.rs`)**:
  - Wired `3D Model > Work Features > UCS` ribbon button (`IconGlyph::WorkCoordinateSystem`)
    to `GuiShellAction::Ucs(GuiUcsAction::Start)`.
  - Added `GuiUcsStage` (`PickOrigin`, `PickXDirection`, `PickYDirection`),
    `GuiUcsAction` (`Start`, `Redefine`, `Cancel`, `SetVisibility`),
    `GuiUcsTriad`, `GuiUcsPlacementView`, and `GuiUcsState`.
  - 3D-projected RGB triad renderer (`draw_ucs_triads`) showing origin handle,
    X/Y/Z axes, arrowheads, labels, and active stage handle rings for both
    interactive placement and visible committed UCS features.
  - Model Browser support (`browser_ucs.rs`):
    - Expandable `UCS1` feature node listing its 7 child work features
      (`YZ Plane`, `XZ Plane`, `XY Plane`, `X Axis`, `Y Axis`, `Z Axis`,
      `Center Point`).
    - Context menu offering `Redefine Feature` and checked/unchecked
      `Visibility` toggle persisted to document history.

- **Desktop Owner (`crates/head/src/owner_ucs.rs`, `crates/head/src/frames/gui_model_click.rs`)**:
  - Interactive 3-stage UCS placement: solid vertex picking for origin,
    X-axis handle/direction picking, and Y-axis handle/direction picking with
    transactional commit to document history.
  - `Escape` cancellation in both viewport input handling (`process_ucs_inputs`)
    and GUI frame handling returns cleanly to `Ready` without mutating history.
  - `Redefine Feature` re-enters interactive placement for an existing UCS;
    pressing `Escape` preserves the existing UCS and returns to `Ready`.
  - Screen projection of preview and committed UCS triads via
    `window.model_pixel`.

## Verification

- **Model Unit Tests (`crates/model`)**:
  - `feature::ucs::tests::orthonormalizes_right_handed_basis_and_derives_children`
  - `feature::ucs::tests::rejects_degenerate_or_collinear_axes`
  - `feature::ucs::tests::recipe_round_trips_with_visibility_and_vertex_ref`
  - `history::hole_create::ucs_create::tests::ucs_create_redefine_visibility_and_recipe_round_trip`
- **Native Wayland BDD (`user-coordinate-system.feature`)**:
  - `user-coordinate-system-01 Begin defining a coordinate system`
  - `user-coordinate-system-02 Advance to the Y direction and cancels`
  - `user-coordinate-system-03 Complete a UCS and inspect its browser contents`
  - `user-coordinate-system-04 Toggle visibility of a UCS`
  - `user-coordinate-system-05 Cancel redefining an existing coordinate system`

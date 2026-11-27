# Rib implementation

Part Create **Rib** (`SketchRibDefinition`, Part program v48) authors an open
sketch profile web/fin with configurable orientation (`Parallel to Sketch Plane`
and `Normal to Sketch Plane`), thickness, thicken side (`Symmetric` / `Side 1` /
`Side 2`), `To Next` / `Finite` extent, `Extend Profile`, and `Draft` (`At Top`
/ `At Root`) through the certified Martensite solid Boolean pipeline.

## Architecture and modules

- **Model & persistence (`crates/model`):**
  - `crates/model/src/feature/sketch_rib.rs` defines `RibDirection`,
    `RibThickenSide`, `RibThicknessPlane`, `SketchRibRecipe`,
    `SketchRibDefinition`, and `ResolvedSketchRib`.
  - `crates/model/src/history/rib_create.rs` adds
    `DocumentSnapshot::create_sketch_rib`, consuming and hiding the source
    sketch on commit.
  - Program version bumped to `48` (`source_version = 42`).
- **Kernel adapter (`crates/martensite`):**
  - `crates/martensite/src/rib.rs` validates profile containment within the
    target solid's bounds and open-region clearance via
    `BRepExtremaDistShapeShape`, builds the thickened web prism with
    `BRepBuilderAPIMakePolygon` / `BRepBuilderAPIMakeFace` /
    `BRepPrimAPIMakePrism`, and fuses it via `NamedSolid::boolean_tools` with
    verified positive volume gain and topology lineage.
  - Fixed `crates/martensite/src/prism/named/blend/shell.rs` offset tolerance to
    `sew.min(maximum_tolerance * 0.5)` and `tolerances::check_budget` so inward
    shells on rectangular blocks avoid 1-ULP rounding overshoots.
- **Automation (`crates/automation`):**
  - `crates/automation/src/rib.rs` exposes `PreparedRib`, `RibInputs`,
    `build_rib_definition`, `prepare_rib_definition`, `replace_rib`,
    `open_profile_lines`, and `features.add "rib"` (`featureargs::KIND_RIB`).
  - Added `Create.Rib` to the Part ribbon inventory.
- **GUI & Desktop (`crates/gui`, `crates/head`):**
  - `crates/gui/src/rib_dialog/{types.rs,modals.rs,tabs.rs,mod.rs}` implements
    the v1.1 dockable `Rib` panel (`Shape`, `Draft` with live `At Top` / `At
    Root` cross-section diagram, and `Boss` tabs), red validation styling for
    non-positive thickness, and transient modals (`Rib - Sketch Required`,
    `Create rib feature failed` with `Edit` / `Cancel`, and `Rib Warning`).
  - Bundled `CreateFeatureDialogs` in `crates/gui/src/engine/create_dialogs.rs`
    and `commands/misc.rs` to keep engine and command files well under 500
    lines.
  - `crates/head/src/owner_rib.rs` owns preview, open-line sketch discovery,
    edit seeds, and checkpointed commit; `owner_shell_feature.rs` clears stale
    removed-face selections after committing a Shell.

## Verification

- Native Wayland BDD (`rib-01`..`rib-05`): 5/5 passed (37/37 steps) —
  [2026-11-27-rib-desktop](receipts/2026-11-27-rib-desktop/README.md).
- Kernel & v48 reopen: `cargo test --locked -p oblikovati-martensite --test rib`.

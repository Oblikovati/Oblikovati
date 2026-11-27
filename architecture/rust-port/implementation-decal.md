# Decal implementation

Part Create **Decal** (`SketchDecalDefinition`, Part program v49, Sketch recipe
v18) applies an inserted 2D sketch raster image onto a live named solid face
through the certified Martensite solid lineage pipeline.

## Architecture and modules

- **Model & persistence (`crates/model`):**
  - `crates/model/src/document_sketch/images.rs` adds `SketchImage` (anchor,
    width, height, rotation, opacity) and `DocumentSketches::add_image` on
    Sketch recipe version `18`.
  - `crates/model/src/feature/sketch_decal.rs` defines `SketchDecalRecipe`,
    `SketchDecalDefinition`, and `ResolvedSketchDecal`.
  - `crates/model/src/history/decal_create.rs` adds
    `DocumentSnapshot::create_sketch_decal`.
  - Program version bumped to `49` (`source_version = 43`).
- **Kernel adapter (`crates/martensite`):**
  - `crates/martensite/src/decal.rs` validates the live named face on the target
    solid, rejects `Wrap to Face` on planar faces, and preserves exact body,
    face, edge, and vertex lineage through `NamedSolid::rigid_candidate`.
- **Automation (`crates/automation`):**
  - `crates/automation/src/sketches/image.rs` implements
    `METHOD_SKETCH_ADD_IMAGE` (`AddSketchImageArgs` -> `AddSketchImageResult`).
  - `crates/automation/src/decal.rs` exposes `PreparedDecal`, `DecalInputs`,
    `build_decal_definition`, `prepare_decal_definition`, `replace_decal`, and
    `features.add "decal"`.
  - Added `Create.Decal` to the Part ribbon inventory.
- **GUI & Desktop (`crates/gui`, `crates/head`):**
  - `crates/gui/src/decal_dialog/{types.rs,modals.rs,mod.rs}` implements the
    v1.1 dockable `Decal` Properties panel (`Input Geometry` with `Image` and
    `Face` selectors, `Behavior` with `Automatic Face Chain` and planar-gated
    `Wrap to Face`, and transient `"Decal - Image Sketch Required"` modal).
  - Registered `IconGlyph::Decal` (`cmd-model-decal-32.svg` + `decal.svg`) on
    the Part `Create` ribbon panel at cell 7.
  - Bundled `CreateFeatureState` in `crates/head/src/owner_blend.rs` and
    `commands/create_enabled.rs` in `crates/gui` to keep `owner.rs` and
    `commands_enabled.rs` under 500 lines.
  - `crates/head/src/owner_decal.rs` owns image-sketch discovery, planar face
    detection (`resolve_planar_face`), preview, edit seeds, and checkpointed
    commit.

## Verification

- Native Wayland BDD (`decal-01`..`decal-04`): 4/4 passed (30/30 steps) —
  [2026-11-27-decal-desktop](receipts/2026-11-27-decal-desktop/README.md).
- Kernel & v49 reopen: `cargo test --locked -p oblikovati-martensite --test decal`.

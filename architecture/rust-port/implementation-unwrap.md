# Unwrap implementation

Part **Unwrap** (`UnwrapDefinition`, Part program v51) unfolds selected solid
faces into an oriented planar surface body (`ShapeEnum::Shell` inside
`NamedSolid::Derived`) with live alignment orientation preview, face chaining,
alignment vertex discovery, and status-bar edge-selector guidance.

## Architecture and modules

- **Model & persistence (`crates/model`):**
  - `crates/model/src/feature/unwrap.rs` defines `UnwrapAlignment` (`OriginXY`,
    `OriginXZ`, `OriginYZ`, `FlippedXY`), `UnwrapRecipe`, `UnwrapDefinition`,
    and `ResolvedUnwrap`.
  - `crates/model/src/history/unwrap_create.rs` adds
    `DocumentSnapshot::create_unwrap`.
  - Program version bumped to `51` (`source_version = 45`).
- **Kernel adapter (`crates/martensite`):**
  - `crates/martensite/src/unwrap.rs` implements `evaluate_named_unwrap`,
    validating live selected faces and any alignment vertex, computing unfolded
    planar extents from `BRepBndLib::add_optimal`, building an oriented planar
    polygon face wrapped in a `Shell`, and binding deterministic face/edge/vertex
    identities via `BlendNamedTopology::from_unwrap_shell` (`is_surface() == true`).
  - `crates/martensite/tests/unwrap.rs` verifies Unwrap surface body creation
    and Part program v51 roundtrip.
- **Automation (`crates/automation`):**
  - `crates/automation/src/unwrap.rs` exposes `PreparedUnwrap`, `UnwrapInputs`,
    `build_unwrap_definition`, `prepare_unwrap_definition`, `replace_unwrap`,
    and `features.add "unwrap"`.
- **GUI & Desktop (`crates/gui`, `crates/head`):**
  - Added `IconGlyph::Unwrap` (`cmd-model-unwrap-32.svg` + `unwrap.svg`) to the
    3D Model `Create` ribbon panel (`cell(8)`).
  - `crates/gui/src/unwrap_dialog/{types.rs,mod.rs}` implements the v1.1
    dockable `Unwrap` Properties panel (`Faces` + `Clear`, `Alignment Vertex`,
    `Linear Result`, `Rigid Result`, `Seam Edges`, `Auto Face Chain`,
    `Merge Result Body`, and 4 `Alignment` buttons `XY`/`XZ`/`YZ`/`Flip`),
    driving the bottom status bar prompt via `CreateFeatureDialogs::status_prompt`.
  - `crates/head/src/owner_unwrap.rs` owns face chaining, alignment vertex
    resolution, live surface preview, `Edit Feature` / browser double-click
    seeding, and checkpointed commit.

## Verification

- Native Wayland BDD (`unwrap-01`..`unwrap-06`): 6/6 passed (40/40 steps) —
  [2026-11-27-unwrap-desktop](receipts/2026-11-27-unwrap-desktop/README.md).
- Kernel & v51 reopen: `cargo test --locked -p oblikovati-martensite --test unwrap`.

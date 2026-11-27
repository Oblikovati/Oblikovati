# Import implementation

Part **Import** (`ImportedModelDefinition`, Part program v50) brings external
STL meshes and STEP BRep solids into a Part document through Martensite's real
`martensite-tkdestl` (`RwStl`, `StlApiReader`) and `martensite-tkdestep`
(`StepControlReader`) exchange crates.

## Architecture and modules

- **Model & persistence (`crates/model`):**
  - `crates/model/src/feature/imported_model.rs` defines `ImportedFileKind`
    (`Stl`, `Step`), `ImportMode` (`ConvertModel`, `ReferenceModel`),
    `StlImportUnits` (`Template Units`, `inch`, `foot`, `centimeter`,
    `millimeter`, `meter`, `micron`), `StlImportOptions`, `StepImportOptions`,
    `ImportedModelRecipe`, `ImportedModelDefinition`, and
    `ResolvedImportedModel`.
  - `crates/model/src/history/import_create.rs` adds
    `DocumentSnapshot::create_imported_model`.
  - Program version bumped to `50` (`source_version = 44`).
- **Kernel adapter (`crates/martensite`):**
  - Linked `martensite-tkdestep`, `martensite-tkdestl`, and `martensite-tkxsbase`
    at the pinned Martensite commit (`db01546b`).
  - `crates/martensite/src/import.rs` exposes `inspect_stl` (`RwStl`),
    `inspect_step` (`StepControlReader` + `MANIFOLD_SOLID_BREP` solid names),
    and `evaluate_named_imported_model` (`StlApiReader` / `StepControlReader` +
    unit scaling + `BlendNamedTopology::from_derived`).
- **Automation (`crates/automation`):**
  - `crates/automation/src/import.rs` exposes `PreparedImport`, `ImportInputs`,
    `build_import_definition`, `prepare_import_definition`,
    `replace_imported_model`, and `features.add "import"`.
- **GUI & Desktop (`crates/gui`, `crates/head`):**
  - Enabled `Import` on `Manage > Insert` (`crates/gui/src/ribbon_panel_contents/muted.rs`).
  - `crates/gui/src/import_dialog/{types.rs,picker.rs,stl_options.rs,mod.rs}`
    implements the `Import File` picker with exchange file-type filters, the
    `STL Import Options` modal (Import Units, Template Units, Bounding Box,
    Translation Report, and Color / gated RGB controls), and the v1.1 dockable
    `Import` STEP dialog (`Options` tab with `Convert Model` / `Reference Model`,
    Object Filters, Part Options with `Surfaces`; `Select` tab with `Load Model`
    tree and viewport preview).
  - `crates/gui/src/browser_import.rs` adds the `3rd Party` Model Browser folder
    for imported STL meshes, the crossed-link visual badge for suppressed
    reference links, and the reference context menu (`Edit Import`, `Suppress
    Link` / `Unsuppress Link`, `Break Link`).
  - `crates/head/src/owner_import.rs` owns file inspection, live STEP preview,
    edit seeds, link suppression/restoration, and checkpointed commit.

## Verification

- Native Wayland BDD (`import-01`..`import-12`): 12/12 passed (66/66 steps) —
  [2026-11-27-import-desktop](receipts/2026-11-27-import-desktop/README.md).
- Kernel & v50 reopen: `cargo test --locked -p oblikovati-martensite --test import`.

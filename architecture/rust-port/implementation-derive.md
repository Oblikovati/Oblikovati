# Derive Part Create implementation and verification

Derive imports solid bodies from a native source `.opd` component into a
destination part with single-solid, keep-seams, separate-solid, and composite
styles, associative source synchronization, uniform scaling, and origin-plane
mirroring across `model`, `martensite`, `session`, `automation`, `gui`, and
`head`.

## Architecture and schema

- **Model (`crates/model`)**: Added `DerivedPartDefinition`,
  `DerivedPartRecipe`, `DeriveStyle` (`SingleSolid`, `SolidWithSeams`,
  `MaintainSolids`, `SingleComposite`), `DeriveMirrorPlane` (`XY`, `YZ`, `XZ`),
  and `ResolvedDerivedPart` at Part program version 47, wired into
  `PartFeatureDefinition::DerivedPart`, replay resolution,
  `DocumentSnapshot::create_derived_part`, and Undo/Redo history.
- **Martensite adapter (`crates/martensite`)**: Added `evaluate_derived_part` in
  `crates/martensite/src/derive.rs` and named topology propagation in
  `crates/martensite/src/prism/named/blend/naming/derived.rs`, evaluating
  uniform scale and planar mirror transforms (`YZ`, `XY`, `XZ`) over the source
  part's evaluated solids with single-solid Boolean union or separate-solid
  multi-body output.
- **Session source loading (`crates/session`)**: Added
  `DocumentSession::derived_sources`, `FileWorkspace::load_derived_source_snapshot`,
  and `FileWorkspace::sync_derived_sources` so derived source snapshots stay
  synchronized across preview, commit, replay, and document reopen.
- **Automation (`crates/automation`)**: Added `PreparedDerivedPart`,
  `DeriveInputs`, `build_derived_part_definition`,
  `prepare_derived_part_definition`, `replace_derived_part`, and
  `features.add` `"derive"` with positive scale validation and unique
  source-named browser identities.
- **Desktop GUI and owner (`crates/gui`, `crates/head`)**: Enabled `Derive` on
  the Part `Manage > Insert` ribbon, added the `Select Component` file picker
  modal (`crates/gui/src/derive_dialog/picker.rs`) and dockable `Derived Part`
  dialog (`crates/gui/src/derive_dialog/mod.rs`) built to UI Design System v1.1
  with `Objects`, `Options`, and `Representation` tabs, live 3D scale/mirror
  preview, Model Browser double-click opening of the native source document, and
  `Edit Derived Part` context-menu editing with `Cancel` preview discard.

## Verification

- **Kernel and persistence tests**: `cargo test --locked -p oblikovati-martensite --test derive`
  verifies source import, scale, YZ mirroring, separate-solid output, and Part
  program v47 `DocumentPackage` encode/decode/reopen.
- **Native desktop BDD**: All 6 Derive scenarios (`derive-01`..`derive-06`)
  passed on native Wayland. See the
  [native Derive desktop receipt](receipts/2026-11-27-derive-desktop/README.md).

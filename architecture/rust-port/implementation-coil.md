# Coil Part Create implementation and verification

Coil creates certified helical and spiral solid features across `model`,
`martensite`, `automation`, `gui`, and `head`, supporting all four sizing
methods (`Pitch and Revolution`, `Revolution and Height`, `Pitch and Height`,
and `Spiral`), taper angle, left-hand and right-hand rotation, axis reversal,
closed start/end flat and transition angles, and `Join`/`Cut`/`Intersect`/`New Solid`
output modes.

## Architecture and schema

- **Model (`crates/model`)**: Added `SketchCoilDefinition`, `SketchCoilRecipe`,
  `CoilMethod`, `CoilHandedness`, `CoilEndTreatment`, and `CoilAxisSource`
  (`WorkAxis(WorkAxisReference)` or `SketchLine`) at Part program version 45,
  wired into `PartFeatureDefinition::SketchCoil`, replay resolution,
  `DocumentSnapshot::create_sketch_coil`, and Undo/Redo history.
- **Martensite adapter (`crates/martensite`)**: Added `build_coil` in
  `crates/martensite/src/coil.rs` using `BRepOffsetAPIMakePipeShell`
  (`set_mode_frenet(false)`) over a 3D `GeomAPI_Interpolate` BSpline spine with
  analytic start/end closure transitions, exact one-to-one named face lineage
  (`crates/martensite/src/prism/named/blend/naming/coil.rs`), and Boolean
  evaluation (`evaluate_coil` / `evaluate_coil_boolean`).
- **Automation (`crates/automation`)**: Added `PreparedCoil`,
  `build_coil_definition`, `prepare_coil_definition`, `features.add` `"coil"`,
  and enabled `Create.Coil` on the Part ribbon.
- **Desktop GUI and owner (`crates/gui`, `crates/head`)**: Built the dockable
  `Coil` Properties panel (`crates/gui/src/coil_dialog/`) on the shared
  `FeaturePropertiesShell` with a scrollable body and pinned `OK`/`Cancel`
  footer, Origin `X`/`Y`/`Z Axis` browser selection, no-profile sketch-on-plane
  auto-reopen, zero-pitch and zero-revolution validation, `ClearProfile`,
  `FlipAxis`, and `Edit Feature` reopening.

## Verification

- **Kernel and persistence tests**: `cargo test --locked -p oblikovati-martensite --test coil`
  verifies all 6 Coil geometry variants (left/right helix, outward spiral,
  5° taper, reversed axis, closed start/end 90°/90°) through
  `CertifiedNamedSolidEvaluator` and Part program v45 `DocumentPackage`
  encode/decode/reopen.
- **Native desktop BDD**: All 7 Coil scenarios (`coil-01`..`coil-07`) passed on
  native Wayland. See the
  [native Coil desktop receipt](receipts/2026-11-27-coil-desktop/README.md).

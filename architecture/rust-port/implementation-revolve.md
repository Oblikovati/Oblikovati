# Revolve Part Create implementation and verification

Revolve supports full-turn (`360 deg`), partial (`Default`, `Flipped`, `Symmetric`,
and `Asymmetric` with `Angle A` / `Angle B`), `Surface` output, and single-body
`Join`, `Cut`, `Intersect`, and `New Solid` Booleans across `model`, `martensite`,
`automation`, `gui`, and `head`.

## Architecture and schema

- **Model (`crates/model`)**: Extended `SketchRevolveDefinition` and
  `SketchRevolveRecipe` (recipe version 2, backward-compatible with v1) with
  explicit straight-line sketch axis selection, full-turn flag, four revolve
  directions, secondary asymmetric angle, surface output, and Boolean target
  recipes. Wired `PartFeatureDefinition::SketchRevolve` at Part program version 44
  with replay resolution and Undo/Redo history preservation.
- **Martensite adapter (`crates/martensite`)**: Added `build_revolve` and
  `build_surface_revolve` in `crates/martensite/src/revolve/solid.rs`. Full-turn
  revolve validates axis crossing with `BRepPrimAPIMakeRevol::full` and extracts
  exact one-to-one face lineage with `BRepSweepRevol::full` and
  `brep_lib::update_inner_tolerances`. Partial and asymmetric directions rotate
  the source profile before sweeping through the total span so all four directions
  share one certified naming pipeline.
- **Automation (`crates/automation`)**: Added `PreparedRevolve`, unconsumed-sketch
  profile auto-selection, comma/dot decimal angle parsing enforcing `(0, 360 deg]`,
  `features.add` `"revolve"`, and enabled `Create.Revolve` on the Part ribbon.
- **Desktop GUI and owner (`crates/gui`, `crates/head`)**: Extracted the reusable
  `FeaturePropertiesShell` (`crates/gui/src/feature_properties/`) and built the
  dockable `Revolution` Properties panel (`crates/gui/src/revolve_dialog/`),
  default `R` keyboard shortcut, empty-part sketch-on-plane auto-reopen flow,
  viewport straight-edge axis picking, owned `d0` angle model parameter with
  Angle A `Parameters` inspection, preview visibility toggle, session presets,
  and `Edit Feature` reopening.

## Verification

- **Certified kernel probes**: `crates/martensite/tests/release_probe/revolve_sources.rs`
  verifies analytic volumes, named face topology, Undo/Redo, package reopen, and
  analytical smooth radial vertex normals across full, partial, asymmetric,
  surface, and Boolean (`Join`, `Cut`, `Intersect`) revolutions.
- **Native desktop BDD**: All 25 Revolve scenarios (`revolve-01`..`18`,
  `revolve-21`..`27`, 189 steps) passed on native Wayland. See the
  [native Revolve and smooth shading receipt](receipts/2026-11-27-revolve-desktop-smooth-shading/README.md).

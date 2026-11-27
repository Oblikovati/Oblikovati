# Coil desktop workflow verification

Verified Part Create Coil (all 7 scenarios, 53 steps) on native Wayland
against the bundled `docking-visible-solid.opd` part fixture, together with
certified `BRepOffsetAPIMakePipeShell` geometry, named face lineage, and
Part program version 45 persistence.

## Native desktop BDD run

- Wayland (RADV Vulkan): 7 passed, 0 failed (53/53 steps, 978.58s) —
  [wayland-native.txt](wayland-native.txt)

Scenarios covered: `coil-01`..`coil-07` (no-profile sketch-on-plane auto-reopen,
Origin `Y Axis` browser pick, default `Pitch and Revolution` 15 mm / 2.5 rev
helical preview, `New Solid` creation to 2 bodies, all four methods
`Pitch and Revolution` / `Revolution and Height` / `Pitch and Height` / `Spiral`
with outward spiral preview and `Edit Feature` persistence, right-hand rotation
and `Close Start` / `Close End` flat and transition angle fields, 5° taper and
`FlipAxis` reversal, `ClearProfile` requirement, and zero-pitch / zero-revolution
validation with preview hide and Cancel recovery).

## Kernel and persistence verification

- `cargo test --locked -p oblikovati-martensite --test coil`: verified left-hand
  and right-hand helices, outward spiral, 5° taper, reversed axis, and closed
  start/end 90°/90° through `CertifiedNamedSolidEvaluator` and Part program v45
  `DocumentPackage` encode/decode/reopen.
- `cargo test --locked -p oblikovati-model --lib --test work_planes`,
  `oblikovati-persistence --test package_work_planes`,
  `oblikovati-automation --lib`, `oblikovati-gui --lib --test dockable_window_guard --test file_dialog`,
  `cargo fmt --all -- --check`, `oblikovati-build docs`, `markdownlint-cli2`,
  and `cargo clippy --locked --workspace --all-targets --all-features -- -D warnings`.

# Emboss desktop workflow verification

Verified Part Create Emboss (all 7 scenarios, 31 steps) on native Wayland
against the bundled `docking-visible-solid.opd` part fixture, together with
certified named sketch-prism Boolean evaluation and Part program version 46
persistence.

## Native desktop BDD run

- Wayland (RADV Vulkan): 7 passed, 0 failed (31/31 steps, 109.62s) —
  [wayland-native.txt](wayland-native.txt)

Scenarios covered: `emboss-01`..`emboss-07` (`Sketch Doctor` open-loop
diagnosis with Cancel returning to the model, raised 1 mm `Emboss1` creation on
a planar solid face with Model Browser entry, double-clicking `Emboss1` to edit
into a 1 mm circular recess, third-operation `Taper` field initially displaying
`0,0` and first-operation direction marker reversal, `"Edit Emboss Feature
Failed"` modal on flush-plane third-operation edit with `Edit` returning to the
settings, `Wrap to Face` Face selector gating `OK` until the planar face is
selected and accepting a raised wrapped feature, and zero-depth out-of-range
warning dismissal with Cancel closing the edit dialog).

## Kernel and persistence verification

- `cargo test --locked -p oblikovati-martensite --test emboss`: verified raised
  emboss, engraved recess, wrap-to-planar-face, clean refusal of flush-plane
  `EmbossEngraveFromPlane`, and Part program v46 `DocumentPackage`
  encode/decode/reopen.
- `cargo test --locked -p oblikovati-model --lib --test work_planes`,
  `oblikovati-persistence --test package_work_planes`,
  `oblikovati-automation --lib`,
  `oblikovati-gui --test dockable_window_guard --test file_dialog`,
  `cargo fmt --all -- --check`, `oblikovati-build docs`, `markdownlint-cli2`,
  and `cargo clippy --locked --workspace --all-targets --all-features -- -D warnings`.

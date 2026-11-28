# Rib desktop workflow verification

Verified Part Create Rib (all 5 scenarios, 37 steps) on native Wayland against
the native `body-properties-start.opd`, `emboss/open-profile.opd`,
`rib/origin-profile-candidate.opd`, and
`rib/side-face-profile-unique-candidate.opd` fixtures, together with certified
Martensite open-profile rib solid construction/fusing and Part program version
48 persistence.

## Native desktop BDD run

- Wayland (RADV Vulkan): 5 passed, 0 failed (37/37 steps, 37.50s) —
  [wayland-native.txt](wayland-native.txt)

Scenarios covered: `rib-01`..`rib-05` (visible unadaptive sketch required tip,
shelling `origin-profile-candidate.opd` top face and reporting `Create rib
feature failed` with `Edit` / `Cancel` in both `Parallel to Sketch Plane` and
`Normal to Sketch Plane` orientations, `Boss` tab controls `Centers` / `Select
All` / `Diameter` / `Offset` / `Draft Angle`, `Draft` tab `Hold Thickness` `At
Top` / `At Root` switching the live cross-section diagram while keeping `OK`
disabled without a profile, `side-face-profile-unique-candidate.opd` two-solid
selection failing cleanly with `Create rib feature failed` and preserving
`Sketch3` visibility on cancel, and zero-thickness red validation styling with
out-of-range warning and recovery).

## Kernel and persistence verification

- `cargo test --locked -p oblikovati-martensite --test rib`: verified valid
  internal rib web construction and Boolean fuse, out-of-bounds profile refusal,
  and Part program v48 `DocumentPackage` encode/decode/reopen.
- `cargo test --locked -p oblikovati-model --lib --test work_planes`,
  `oblikovati-persistence --test package_work_planes`,
  `oblikovati-automation --lib`,
  `oblikovati-gui --lib --test dockable_window_guard`,
  `cargo fmt --all -- --check`, `oblikovati-build docs`, and
  `cargo clippy --locked --workspace --all-targets -- -D warnings`.

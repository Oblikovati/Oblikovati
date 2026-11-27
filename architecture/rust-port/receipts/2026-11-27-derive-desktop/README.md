# Derive desktop workflow verification

Verified Part Create Derive (all 6 scenarios, 42 steps) on native Wayland
against the native `blank-metric-part.opd` and `body-properties-start.opd`
fixtures, together with certified Martensite scale/mirror evaluation and Part
program version 47 persistence.

## Native desktop BDD run

- Wayland (RADV Vulkan): 6 passed, 0 failed (42/42 steps, 49.16s) —
  [wayland-native.txt](wayland-native.txt)

Scenarios covered: `derive-01`..`derive-06` (`Manage > Insert > Derive` opening
the `Select Component` file picker modal and launching the dockable `Derived
Part` dialog with object tree and live preview, `Objects` tab groups `Solid
Bodies`/`Sketches`/`Work Geometry`/`Parameters` and derive styles, `Options`
link controls / `Scale Factor` / `Mirror part`, `Representation` tab showing
`[Primary]` for `Model State` and `Design View`, `Mirror part` revealing `XY
Plane`/`YZ Plane`/`XZ Plane` and selecting `YZ Plane`, separate-solid mirrored
derive creating two solid bodies with the native source component listed in the
Model Browser, double-clicking the browser entry to open the native source
document, returning to the destination to choose `Edit Derived Part` from the
context menu, scaling `1` to `2` to enlarge the live preview, and clicking
`Cancel` to close without modifying the destination).

## Kernel and persistence verification

- `cargo test --locked -p oblikovati-martensite --test derive`: verified source
  import, uniform scale, YZ mirroring, separate-solid output, and Part program
  v47 `DocumentPackage` encode/decode/reopen.
- `cargo test --locked -p oblikovati-model --lib --test work_planes`,
  `oblikovati-persistence --test package_work_planes`,
  `oblikovati-automation --lib`,
  `oblikovati-gui --test dockable_window_guard --test file_dialog`,
  `cargo fmt --all -- --check`, `oblikovati-build docs`, `markdownlint-cli2`,
  and `cargo clippy --locked --workspace --all-targets -- -D warnings`.

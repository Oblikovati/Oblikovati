# Revolve desktop workflow and smooth Phong shading

Verified Part Create Revolve (23 scenarios, 174 steps) on both native Wayland
and X11 (Xvfb + Lavapipe), together with analytical per-vertex smooth shading
and batched `Graphic3dStructure` triangle arrays.

## Native desktop BDD runs

- Wayland (RADV Vulkan): 23 passed, 0 failed (174/174 steps, 164.60s) —
  [wayland-native.txt](wayland-native.txt)
- X11 (Xvfb + Lavapipe): 23 passed, 0 failed (174/174 steps, 176.31s) —
  [x11-native.txt](x11-native.txt)
- Scenario and fixture hashes: [current-inputs.sha256](current-inputs.sha256)

Scenarios covered: `revolve-02`..`revolve-15`, `revolve-17`, `revolve-18`,
`revolve-21`..`revolve-27` (straight-edge axis picking, zero/negative/upper/lower
angle validation, `Default`/`Flipped`/`Symmetric`/`Asymmetric` directions,
`Escape` cancel and `R` shortcut reopen, full and partial Undo/Redo, `Edit Feature`,
`To` reference extent validation, `Surface` output with 4 curves and profile clear/repick,
direction and Boolean icon/dropdown selector styles, `Join`/`Cut`/`Intersect`/`New Solid`
Booleans on `revolve-boolean-base.opd`).

## Smooth shading and kernel verification

- `cargo test -p oblikovati-martensite --test release_probe revolve_sources`:
  5 passed, 0 failed — includes `revolved_cylinder_display_mesh_produces_smooth_outward_radial_normals`
  verifying analytical outward unit radial normals on curved B-rep faces.
- `cargo test -p oblikovati-viewport --test native lighting`:
  5 passed, 0 failed — includes `per_vertex_normals_produce_smooth_phong_gradient_across_one_triangle`
  confirming interpolated per-vertex Phong shading in Vulkan.
- `cargo test -p oblikovati-viewport --lib`: 191 passed, 0 failed (including rotated
  smooth normal cofactor transformation in `scene_placement`).
- `cargo test -p oblikovati-model --lib` (203 passed), `oblikovati-drawing` (all passed),
  `oblikovati-automation --lib` (923 passed), `oblikovati-gui --lib` (850 passed),
  and `cargo clippy --all-targets -- -D warnings` across all 7 affected crates (0 warnings).
